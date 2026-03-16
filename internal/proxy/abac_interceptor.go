package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/abac/proxy/internal/log"
	"github.com/abac/proxy/internal/policy"
	"github.com/abac/proxy/internal/storage"
)

type contextKey string

const (
	contextKeyTokenValid    contextKey = "abac_token_valid"
	contextKeyRequestPath   contextKey = "abac_request_path"
	contextKeyRequestMethod contextKey = "abac_request_method"
)

type ABACInterceptor struct {
	// For file-based policy (static)
	engine *policy.PolicyEngine

	// For database-based policy (cached with TTL)
	cache *policy.PolicyCache
}

func NewABACInterceptor(policyPath string) (*ABACInterceptor, error) {
	engine, err := policy.NewPolicyEngine(policyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy engine: %w", err)
	}

	return &ABACInterceptor{
		engine: engine,
	}, nil
}

// NewABACInterceptorFromDB creates an ABAC interceptor from database storage with caching
func NewABACInterceptorFromDB(ctx context.Context, databaseURL string) (*ABACInterceptor, error) {
	pool, err := storage.NewPool(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	store := storage.NewStore(pool)

	// Create policy cache with 15 second TTL
	cache := policy.NewPolicyCache(store, 15*time.Second)

	return &ABACInterceptor{
		cache: cache,
	}, nil
}

func (a *ABACInterceptor) InterceptRequest(req *http.Request) *http.Request {
	logger := log.From(req.Context())

	token := extractToken(req)

	_, err := a.getEngine(req.Context(), token)
	if err != nil {
		logger.Errorw("failed to load policy or invalid token",
			"error", err,
		)
		ctx := context.WithValue(req.Context(), contextKeyTokenValid, false)
		ctx = context.WithValue(ctx, contextKeyRequestPath, req.URL.Path)
		ctx = context.WithValue(ctx, contextKeyRequestMethod, req.Method)
		return req.WithContext(ctx)
	}

	// Token is valid if we successfully loaded the engine
	logger.Infow("token validated",
		"path", req.URL.Path,
		"method", req.Method,
	)

	ctx := context.WithValue(req.Context(), contextKeyTokenValid, true)
	ctx = context.WithValue(ctx, contextKeyRequestPath, req.URL.Path)
	ctx = context.WithValue(ctx, contextKeyRequestMethod, req.Method)

	return req.WithContext(ctx)
}

func (a *ABACInterceptor) InterceptResponse(resp *http.Response) error {
	logger := log.From(resp.Request.Context())

	tokenValid, _ := resp.Request.Context().Value(contextKeyTokenValid).(bool)
	path, _ := resp.Request.Context().Value(contextKeyRequestPath).(string)
	method, _ := resp.Request.Context().Value(contextKeyRequestMethod).(string)

	if !tokenValid {
		logger.Warnw("access denied: invalid token",
			"path", path,
			"method", method,
		)
		replaceWithError(resp, http.StatusForbidden, "invalid or missing token")
		return nil
	}

	token := extractToken(resp.Request)

	engine, err := a.getEngine(resp.Request.Context(), token)
	if err != nil {
		logger.Errorw("failed to load policy",
			"path", path,
			"error", err,
		)
		replaceWithError(resp, http.StatusInternalServerError, "policy engine error")
		return nil
	}

	rule, found := engine.FindMatchingRule(path, method)
	action := ""
	if found {
		action = rule.Action
		logger.Infow("policy rule matched",
			"path", path,
			"method", method,
			"route", rule.Route,
			"action", action,
		)
	} else {
		action = engine.GetDefaultAction()
		logger.Infow("no matching rule, using default action",
			"path", path,
			"method", method,
			"action", action,
		)
	}

	if action == "deny" {
		logger.Warnw("access denied by policy",
			"path", path,
			"method", method,
		)
		replaceWithError(resp, http.StatusForbidden, "access denied by policy")
		return nil
	}

	if !found || rule.ResponseFilter == nil {
		logger.Infow("access allowed, no filter applied",
			"path", path,
			"method", method,
		)
		return nil
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		logger.Errorw("non-JSON response cannot be filtered",
			"path", path,
			"method", method,
			"content_type", contentType,
		)
		replaceWithError(resp, http.StatusForbidden, "non-JSON response cannot be filtered")
		return nil
	}

	filterErr := ReadAndReplaceBody(resp, func(body []byte) ([]byte, error) {
		var data interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			logger.Errorw("failed to parse JSON response",
				"path", path,
				"error", err,
			)
			return nil, fmt.Errorf("invalid JSON response")
		}

		filterer := engine.GetFilterer()
		filtered, err := filterer.Apply(data, *rule.ResponseFilter)
		if err != nil {
			logger.Errorw("failed to apply response filter",
				"path", path,
				"error", err,
			)
			return nil, fmt.Errorf("failed to filter response")
		}

		filteredBody, err := json.Marshal(filtered)
		if err != nil {
			logger.Errorw("failed to marshal filtered response",
				"path", path,
				"error", err,
			)
			return nil, fmt.Errorf("failed to encode response")
		}

		logger.Infow("response filtered successfully",
			"path", path,
			"method", method,
			"filter_type", string(rule.ResponseFilter.Type),
			"original_size", len(body),
			"filtered_size", len(filteredBody),
		)

		return filteredBody, nil
	})

	if filterErr != nil {
		replaceWithError(resp, http.StatusInternalServerError, filterErr.Error())
	}

	return nil
}

// getEngine returns the policy engine, either from static file or cached from DB
func (a *ABACInterceptor) getEngine(ctx context.Context, token string) (*policy.PolicyEngine, error) {
	// File-based policy (static)
	if a.engine != nil {
		return a.engine, nil
	}

	// Database-based policy (cached with TTL)
	if a.cache != nil {
		if token == "" {
			return nil, fmt.Errorf("token required for database-backed policy")
		}
		return a.cache.GetByToken(ctx, token)
	}

	return nil, fmt.Errorf("no policy engine configured")
}

func extractToken(req *http.Request) string {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

func replaceWithError(resp *http.Response, statusCode int, message string) {
	errorBody := map[string]string{"error": message}
	body, _ := json.Marshal(errorBody)

	resp.StatusCode = statusCode
	resp.Status = http.StatusText(statusCode)
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Type", "application/json")
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
}
