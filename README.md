# ABAC HTTP Proxy

HTTP proxy with traffic interception & payload modification.

## Features

- **Request Interception**: Inspect/modify requests before forwarding
- **Response Modification**: Transform response payloads before returning to client
- **Pluggable Interceptors**: Custom logic via `Interceptor` interface
- **TLS Support**: Optional HTTPS proxy
- **Logging**: Built-in request/response logging

## Usage

```bash
bazel run //cmd/proxy -- --port 8080 --target http://api.example.com
```

### Flags

- `--port`: Proxy listen port (default: 8080)
- `--target`: Target base URL to proxy to (required)
- `--tls`: Enable TLS
- `--cert`: TLS certificate file
- `--key`: TLS private key file

### Environment Variables

- `PROXY_PORT`
- `PROXY_TARGET`
- `PROXY_TLS`
- `PROXY_CERT`
- `PROXY_KEY`

## Custom Interceptors

Implement the `Interceptor` interface:

```go
type Interceptor interface {
    InterceptRequest(req *http.Request) *http.Request
    InterceptResponse(resp *http.Response) error
}
```

### Built-in Interceptors

**PassthroughInterceptor**: No modifications (default)

**ExampleInterceptor**: Add headers & modify JSON
```go
interceptor := &proxy.ExampleInterceptor{
    AddHeaders: map[string]string{
        "X-Proxy": "abac-proxy",
    },
    ModifyJSON: true,
    JSONTransform: func(data map[string]any) map[string]any {
        data["intercepted"] = true
        return data
    },
}
```

**LoggingInterceptor**: Log all traffic
```go
interceptor := &proxy.LoggingInterceptor{
    LogBodies: true,
}
```

**ChainInterceptor**: Combine multiple interceptors
```go
interceptor := &proxy.ChainInterceptor{
    Interceptors: []proxy.Interceptor{
        &proxy.LoggingInterceptor{},
        &proxy.ExampleInterceptor{ModifyJSON: true},
    },
}
```

## Helper Functions

**ReadAndReplaceBody**: Modify response body
```go
func (i *MyInterceptor) InterceptResponse(resp *http.Response) error {
    return proxy.ReadAndReplaceBody(resp, func(body []byte) ([]byte, error) {
        // Transform body
        return modifiedBody, nil
    })
}
```

## Build

```bash
# Build all
bazel build //...

# Run tests
bazel test //...

# Update BUILD files
bazel run //:gazelle
```
