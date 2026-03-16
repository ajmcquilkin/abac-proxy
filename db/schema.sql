\restrict m79VdfRrEVLg0LcBXtbC9IZ68FH92hDZwRQmdUeTRX2NRVr8aanAVwN6fLXGpdS

-- Dumped from database version 17.8 (6108b59)
-- Dumped by pg_dump version 18.3 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

-- *not* creating schema, since initdb creates it


--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS '';


--
-- Name: update_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: downstream_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.downstream_tokens (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    policy_id uuid NOT NULL,
    token_hash text NOT NULL,
    name character varying(255) NOT NULL,
    expires_at timestamp with time zone,
    last_used_at timestamp with time zone,
    revoked boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: policies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.policies (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    version text NOT NULL,
    base_url text NOT NULL,
    default_action text DEFAULT 'deny'::text NOT NULL,
    rules jsonb NOT NULL,
    is_active boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    upstream_credential_id uuid NOT NULL
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying NOT NULL
);


--
-- Name: upstream_credentials; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.upstream_credentials (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    token text NOT NULL,
    api_endpoint text,
    token_type character varying(50) DEFAULT 'bearer'::character varying,
    expires_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: downstream_tokens downstream_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.downstream_tokens
    ADD CONSTRAINT downstream_tokens_pkey PRIMARY KEY (id);


--
-- Name: downstream_tokens downstream_tokens_policy_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.downstream_tokens
    ADD CONSTRAINT downstream_tokens_policy_id_name_key UNIQUE (policy_id, name);


--
-- Name: policies policies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.policies
    ADD CONSTRAINT policies_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: upstream_credentials upstream_credentials_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.upstream_credentials
    ADD CONSTRAINT upstream_credentials_pkey PRIMARY KEY (id);


--
-- Name: upstream_credentials upstream_credentials_user_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.upstream_credentials
    ADD CONSTRAINT upstream_credentials_user_id_name_key UNIQUE (user_id, name);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_downstream_tokens_hash_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_downstream_tokens_hash_active ON public.downstream_tokens USING btree (token_hash) WHERE (revoked = false);


--
-- Name: idx_downstream_tokens_policy_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_downstream_tokens_policy_id ON public.downstream_tokens USING btree (policy_id);


--
-- Name: idx_policies_is_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_policies_is_active ON public.policies USING btree (is_active);


--
-- Name: idx_policies_user_active; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_policies_user_active ON public.policies USING btree (user_id) WHERE (is_active = true);


--
-- Name: idx_policies_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_policies_user_id ON public.policies USING btree (user_id);


--
-- Name: idx_upstream_credentials_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_upstream_credentials_user_id ON public.upstream_credentials USING btree (user_id);


--
-- Name: policies policies_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER policies_updated_at BEFORE UPDATE ON public.policies FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();


--
-- Name: users users_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();


--
-- Name: downstream_tokens downstream_tokens_policy_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.downstream_tokens
    ADD CONSTRAINT downstream_tokens_policy_id_fkey FOREIGN KEY (policy_id) REFERENCES public.policies(id) ON DELETE CASCADE;


--
-- Name: policies policies_upstream_credential_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.policies
    ADD CONSTRAINT policies_upstream_credential_id_fkey FOREIGN KEY (upstream_credential_id) REFERENCES public.upstream_credentials(id);


--
-- Name: policies policies_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.policies
    ADD CONSTRAINT policies_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: upstream_credentials upstream_credentials_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.upstream_credentials
    ADD CONSTRAINT upstream_credentials_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict m79VdfRrEVLg0LcBXtbC9IZ68FH92hDZwRQmdUeTRX2NRVr8aanAVwN6fLXGpdS


--
-- Dbmate schema migrations
--

INSERT INTO public.schema_migrations (version) VALUES
    ('20260315000001'),
    ('20260316000001'),
    ('20260316000002');
