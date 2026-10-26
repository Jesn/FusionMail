-- ============================================================
-- FusionMail schema baseline
--
-- 来源：对「GORM AutoMigrate 在干净 PostgreSQL 上产出的真实 schema」
--       执行 pg_dump --schema-only 导出。
--
-- 为什么需要它：backend/migrations/ 下原有的 39 个 SQL 迁移从未被任何代码
-- 执行过（唯一入口 migrate.sh 只跑 001-003 且硬编码本地容器名），线上 schema
-- 实际由 AutoMigrate 从 GORM 模型推导。实测把 39 个 SQL 迁移按序跑在干净库上，
-- 24 成功 / 15 失败——001 用了 PostgreSQL 保留字 references 作列名导致语法错误，
-- 后续全部级联失败。也就是说这套迁移链根本无法从零建库。
--
-- 本文件是 schema 的唯一真相起点：后续所有结构变更都以新版本迁移追加，
-- AutoMigrate 仅保留给测试环境使用。
--
-- 生成时间：2026-10-27
-- ============================================================

CREATE TABLE public.account_groups (
    id bigint NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    display_order bigint DEFAULT 0,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE SEQUENCE public.account_groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.account_groups_id_seq OWNED BY public.account_groups.id;

CREATE TABLE public.adapters (
    id bigint NOT NULL,
    name character varying(50) NOT NULL,
    display_name character varying(100) NOT NULL,
    auth_type character varying(20) NOT NULL,
    description text,
    is_enabled boolean DEFAULT true,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE public.adapters_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.adapters_id_seq OWNED BY public.adapters.id;

CREATE TABLE public.api_keys (
    id bigint NOT NULL,
    key_hash character varying(64) NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    permissions text,
    rate_limit bigint DEFAULT 100,
    enabled boolean DEFAULT true,
    expires_at timestamp with time zone,
    total_requests bigint DEFAULT 0,
    last_used_at timestamp with time zone,
    last_ip character varying(45),
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE public.api_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.api_keys_id_seq OWNED BY public.api_keys.id;

CREATE TABLE public.bayesian_trainings (
    id bigint NOT NULL,
    user_uid character varying(64) NOT NULL,
    email_id character varying(255),
    is_spam boolean NOT NULL,
    tokens text NOT NULL,
    created_at timestamp with time zone
);

CREATE SEQUENCE public.bayesian_trainings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.bayesian_trainings_id_seq OWNED BY public.bayesian_trainings.id;

CREATE TABLE public.deleted_email_keys (
    id bigint NOT NULL,
    account_uid character varying(64) NOT NULL,
    dedupe_key character varying(128) NOT NULL,
    deleted_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

CREATE SEQUENCE public.deleted_email_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.deleted_email_keys_id_seq OWNED BY public.deleted_email_keys.id;

CREATE TABLE public.email_accounts (
    id bigint NOT NULL,
    uid character varying(64) NOT NULL,
    email character varying(255) NOT NULL,
    provider_id bigint NOT NULL,
    adapter_id bigint NOT NULL,
    encrypted_credentials text NOT NULL,
    smtp_enabled boolean DEFAULT false,
    proxy_enabled boolean DEFAULT false,
    proxy_type character varying(20),
    proxy_host character varying(255),
    proxy_port bigint,
    proxy_username character varying(255),
    encrypted_proxy_password text,
    status character varying(20) DEFAULT 'active'::character varying,
    consecutive_auth_failures bigint DEFAULT 0 NOT NULL,
    auto_disabled_at timestamp with time zone,
    disable_reason character varying(100),
    sync_enabled boolean DEFAULT true,
    sync_interval bigint DEFAULT 2,
    sync_mode character varying(20) DEFAULT 'polling'::character varying,
    last_sync_at timestamp with time zone,
    last_sync_status character varying(20),
    last_sync_error text,
    uid_validity bigint DEFAULT 0,
    last_uid bigint DEFAULT 0,
    first_sync_days bigint DEFAULT 7,
    batch_size bigint DEFAULT 100,
    max_emails_per_sync bigint DEFAULT 5000,
    sync_cursor text,
    sync_progress_json jsonb DEFAULT '{}'::jsonb,
    server_delete_policy character varying(10) DEFAULT 'off'::character varying,
    group_id bigint,
    parent_account_uid character varying(64),
    total_emails bigint DEFAULT 0,
    unread_count bigint DEFAULT 0,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE SEQUENCE public.email_accounts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.email_accounts_id_seq OWNED BY public.email_accounts.id;

CREATE TABLE public.email_attachments (
    id bigint NOT NULL,
    email_id bigint NOT NULL,
    filename character varying(255) NOT NULL,
    content_type character varying(100),
    size_bytes bigint NOT NULL,
    storage_type character varying(20) DEFAULT 'local'::character varying,
    storage_path text NOT NULL,
    is_inline boolean DEFAULT false,
    content_id character varying(255),
    created_at timestamp with time zone
);

CREATE SEQUENCE public.email_attachments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.email_attachments_id_seq OWNED BY public.email_attachments.id;

CREATE TABLE public.email_label_relations (
    email_id bigint NOT NULL,
    label_id bigint NOT NULL,
    created_at timestamp with time zone
);

CREATE TABLE public.email_labels (
    id bigint NOT NULL,
    name character varying(100) NOT NULL,
    color character varying(20),
    description text,
    email_count bigint DEFAULT 0,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE public.email_labels_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.email_labels_id_seq OWNED BY public.email_labels.id;

CREATE TABLE public.email_lists (
    id bigint NOT NULL,
    user_uid character varying(64) NOT NULL,
    type character varying(20) NOT NULL,
    target character varying(255) NOT NULL,
    target_type character varying(20) NOT NULL,
    reason text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_email_lists_target_type CHECK (((target_type)::text = ANY ((ARRAY['email'::character varying, 'domain'::character varying])::text[]))),
    CONSTRAINT chk_email_lists_type CHECK (((type)::text = ANY ((ARRAY['whitelist'::character varying, 'blacklist'::character varying])::text[])))
);

CREATE SEQUENCE public.email_lists_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.email_lists_id_seq OWNED BY public.email_lists.id;

CREATE TABLE public.email_oauth2_tokens (
    id bigint NOT NULL,
    provider_id bigint NOT NULL,
    name character varying(100) NOT NULL,
    client_id character varying(255) NOT NULL,
    client_secret_encrypted text NOT NULL,
    redirect_uri character varying(500) NOT NULL,
    enabled boolean DEFAULT true,
    is_default boolean DEFAULT false,
    usage_count bigint DEFAULT 0,
    quota_daily bigint DEFAULT '-1'::integer,
    quota_monthly bigint DEFAULT '-1'::integer,
    last_used_at timestamp with time zone,
    metadata text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE public.email_oauth2_tokens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.email_oauth2_tokens_id_seq OWNED BY public.email_oauth2_tokens.id;

CREATE TABLE public.email_rules (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    account_uid character varying(64) NOT NULL,
    description text,
    enabled boolean DEFAULT true,
    priority bigint DEFAULT 0,
    match_mode character varying(10) DEFAULT 'all'::character varying,
    stop_processing boolean DEFAULT false,
    conditions text NOT NULL,
    actions text NOT NULL,
    matched_count bigint DEFAULT 0,
    last_matched_at timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE public.email_rules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.email_rules_id_seq OWNED BY public.email_rules.id;

CREATE TABLE public.emails (
    id bigint NOT NULL,
    provider_id character varying(255) NOT NULL,
    account_uid character varying(64) NOT NULL,
    message_id character varying(255),
    dedupe_key character varying(128),
    subject text NOT NULL,
    from_address character varying(255) NOT NULL,
    from_name character varying(255),
    to_address character varying(255),
    to_addresses text,
    cc_addresses text,
    bcc_addresses text,
    reply_to character varying(255),
    text_body text,
    html_body text,
    snippet text,
    is_read boolean DEFAULT false,
    is_starred boolean DEFAULT false,
    is_archived boolean DEFAULT false,
    is_deleted boolean DEFAULT false,
    labels text,
    local_labels text,
    folder character varying(255),
    source_is_read boolean,
    source_labels text,
    source_folder character varying(255),
    has_attachment boolean DEFAULT false,
    has_attachments boolean DEFAULT false,
    attachments_count bigint DEFAULT 0,
    sent_at timestamp with time zone NOT NULL,
    received_at timestamp with time zone NOT NULL,
    synced_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    size_bytes bigint,
    thread_id character varying(255),
    in_reply_to character varying(255),
    "references" text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    is_spam boolean DEFAULT false,
    spam_score numeric DEFAULT 0,
    spam_confidence numeric DEFAULT 0,
    spam_reason text,
    spam_detected_at timestamp with time zone,
    spam_detected_by character varying(50),
    user_marked_spam boolean DEFAULT false,
    user_marked_at timestamp with time zone
);

CREATE SEQUENCE public.emails_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.emails_id_seq OWNED BY public.emails.id;

CREATE TABLE public.provider_adapters (
    provider_id bigint NOT NULL,
    adapter_id bigint NOT NULL,
    priority bigint DEFAULT 0,
    created_at timestamp with time zone
);

CREATE TABLE public.providers (
    id bigint NOT NULL,
    name character varying(50) NOT NULL,
    display_name character varying(100) NOT NULL,
    default_adapter_id bigint,
    email_domains text[],
    supported_protocols text NOT NULL,
    recommended_protocol character varying(20) NOT NULL,
    requires_o_auth boolean DEFAULT false,
    imap_host character varying(255),
    imap_port bigint,
    pop3_host character varying(255),
    pop3_port bigint,
    smtp_host character varying(255),
    smtp_port bigint,
    imap_encryption character varying(20) DEFAULT 'ssl'::character varying,
    pop3_encryption character varying(20) DEFAULT 'ssl'::character varying,
    smtp_encryption character varying(20) DEFAULT 'ssl'::character varying,
    enabled boolean DEFAULT true,
    sort_order bigint DEFAULT 0,
    description text,
    metadata text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE public.providers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.providers_id_seq OWNED BY public.providers.id;

CREATE TABLE public.sender_reputations (
    id bigint NOT NULL,
    email character varying(255) NOT NULL,
    domain character varying(255),
    reputation_score numeric DEFAULT 50,
    trust_level character varying(20),
    total_emails bigint DEFAULT 0,
    spam_count bigint DEFAULT 0,
    ham_count bigint DEFAULT 0,
    rbl_status character varying(20),
    rbl_checked_at timestamp with time zone,
    rbl_lists text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_sender_reputations_rbl_status CHECK (((rbl_status)::text = ANY ((ARRAY['clean'::character varying, 'listed'::character varying, 'unknown'::character varying])::text[]))),
    CONSTRAINT chk_sender_reputations_trust_level CHECK (((trust_level)::text = ANY ((ARRAY['trusted'::character varying, 'neutral'::character varying, 'suspicious'::character varying, 'blocked'::character varying])::text[])))
);

CREATE SEQUENCE public.sender_reputations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.sender_reputations_id_seq OWNED BY public.sender_reputations.id;

CREATE TABLE public.sent_emails (
    id bigint NOT NULL,
    account_uid character varying(64) NOT NULL,
    provider_msg_id character varying(255),
    message_id character varying(255),
    subject text,
    from_address character varying(255),
    from_name character varying(255),
    to_addresses text,
    cc_addresses text,
    bcc_addresses text,
    text_body text,
    html_body text,
    has_attachments boolean DEFAULT false,
    attachment_count bigint DEFAULT 0,
    attachment_info text,
    reply_to_email_id bigint,
    forward_from_id bigint,
    in_reply_to character varying(255),
    "references" text,
    status character varying(20) DEFAULT 'sent'::character varying,
    error_message text,
    sent_at timestamp with time zone NOT NULL,
    retry_count bigint DEFAULT 0,
    last_retry_at timestamp with time zone,
    sender_type character varying(20),
    size_bytes bigint,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

CREATE SEQUENCE public.sent_emails_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.sent_emails_id_seq OWNED BY public.sent_emails.id;

CREATE TABLE public.settings (
    id bigint NOT NULL,
    user_id bigint,
    category character varying(50) NOT NULL,
    key character varying(100) NOT NULL,
    value text NOT NULL,
    value_type character varying(20) DEFAULT 'string'::character varying NOT NULL,
    is_sensitive boolean DEFAULT false,
    is_public boolean DEFAULT false,
    description text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE public.settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.settings_id_seq OWNED BY public.settings.id;

CREATE TABLE public.spam_detection_logs (
    id bigint NOT NULL,
    email_id character varying(255),
    is_spam boolean NOT NULL,
    final_score numeric NOT NULL,
    detection_details text,
    processing_time_ms bigint,
    created_at timestamp with time zone
);

CREATE SEQUENCE public.spam_detection_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.spam_detection_logs_id_seq OWNED BY public.spam_detection_logs.id;

CREATE TABLE public.spam_rules (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    category character varying(50),
    pattern text NOT NULL,
    score bigint DEFAULT 10,
    enabled boolean,
    is_builtin boolean DEFAULT false,
    hit_count bigint DEFAULT 0,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    CONSTRAINT chk_spam_rules_category CHECK (((category)::text = ANY ((ARRAY['keyword'::character varying, 'pattern'::character varying, 'header'::character varying, 'content'::character varying, 'url'::character varying, 'attachment'::character varying])::text[])))
);

CREATE SEQUENCE public.spam_rules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.spam_rules_id_seq OWNED BY public.spam_rules.id;

CREATE TABLE public.sync_logs (
    id bigint NOT NULL,
    account_uid character varying(64) NOT NULL,
    sync_type character varying(20) NOT NULL,
    status character varying(20) NOT NULL,
    emails_fetched bigint DEFAULT 0,
    emails_new bigint DEFAULT 0,
    emails_updated bigint DEFAULT 0,
    total_estimated bigint,
    current_batch bigint,
    total_batches bigint,
    sync_cursor text,
    is_first_sync boolean DEFAULT false,
    started_at timestamp with time zone NOT NULL,
    completed_at timestamp with time zone,
    duration_ms bigint,
    error_message text
);

CREATE SEQUENCE public.sync_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.sync_logs_id_seq OWNED BY public.sync_logs.id;

CREATE TABLE public.users (
    id bigint NOT NULL,
    username character varying(100) NOT NULL,
    email character varying(255) NOT NULL,
    password_hash character varying(255) NOT NULL,
    display_name character varying(255),
    avatar_url text,
    timezone character varying(50) DEFAULT 'UTC'::character varying,
    language character varying(10) DEFAULT 'en'::character varying,
    theme character varying(20) DEFAULT 'light'::character varying,
    role character varying(20) DEFAULT 'user'::character varying,
    is_active boolean DEFAULT true,
    email_verified boolean DEFAULT false,
    last_login_at timestamp with time zone,
    last_login_ip character varying(45),
    failed_login_attempts bigint DEFAULT 0,
    locked_until timestamp with time zone,
    session_version bigint DEFAULT 0 NOT NULL,
    two_factor_enabled boolean DEFAULT false,
    two_factor_secret text,
    two_factor_backup text,
    two_factor_verified boolean DEFAULT false,
    two_factor_enabled_at timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;

CREATE TABLE public.webhook_logs (
    id bigint NOT NULL,
    webhook_id bigint NOT NULL,
    request_url text NOT NULL,
    request_method character varying(10),
    request_headers text,
    request_body text,
    response_status bigint,
    response_body text,
    response_time_ms bigint,
    success boolean,
    error_message text,
    retry_count bigint DEFAULT 0,
    created_at timestamp with time zone
);

CREATE SEQUENCE public.webhook_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.webhook_logs_id_seq OWNED BY public.webhook_logs.id;

CREATE TABLE public.webhooks (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    url text NOT NULL,
    method character varying(10) DEFAULT 'POST'::character varying,
    headers text,
    enabled boolean DEFAULT true,
    events text NOT NULL,
    filters text,
    retry_enabled boolean DEFAULT true,
    max_retries bigint DEFAULT 3,
    retry_intervals text DEFAULT '[10, 30, 60]'::text,
    total_calls bigint DEFAULT 0,
    success_calls bigint DEFAULT 0,
    failed_calls bigint DEFAULT 0,
    last_called_at timestamp with time zone,
    last_status character varying(20),
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE public.webhooks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.webhooks_id_seq OWNED BY public.webhooks.id;

ALTER TABLE ONLY public.account_groups ALTER COLUMN id SET DEFAULT nextval('public.account_groups_id_seq'::regclass);

ALTER TABLE ONLY public.adapters ALTER COLUMN id SET DEFAULT nextval('public.adapters_id_seq'::regclass);

ALTER TABLE ONLY public.api_keys ALTER COLUMN id SET DEFAULT nextval('public.api_keys_id_seq'::regclass);

ALTER TABLE ONLY public.bayesian_trainings ALTER COLUMN id SET DEFAULT nextval('public.bayesian_trainings_id_seq'::regclass);

ALTER TABLE ONLY public.deleted_email_keys ALTER COLUMN id SET DEFAULT nextval('public.deleted_email_keys_id_seq'::regclass);

ALTER TABLE ONLY public.email_accounts ALTER COLUMN id SET DEFAULT nextval('public.email_accounts_id_seq'::regclass);

ALTER TABLE ONLY public.email_attachments ALTER COLUMN id SET DEFAULT nextval('public.email_attachments_id_seq'::regclass);

ALTER TABLE ONLY public.email_labels ALTER COLUMN id SET DEFAULT nextval('public.email_labels_id_seq'::regclass);

ALTER TABLE ONLY public.email_lists ALTER COLUMN id SET DEFAULT nextval('public.email_lists_id_seq'::regclass);

ALTER TABLE ONLY public.email_oauth2_tokens ALTER COLUMN id SET DEFAULT nextval('public.email_oauth2_tokens_id_seq'::regclass);

ALTER TABLE ONLY public.email_rules ALTER COLUMN id SET DEFAULT nextval('public.email_rules_id_seq'::regclass);

ALTER TABLE ONLY public.emails ALTER COLUMN id SET DEFAULT nextval('public.emails_id_seq'::regclass);

ALTER TABLE ONLY public.providers ALTER COLUMN id SET DEFAULT nextval('public.providers_id_seq'::regclass);

ALTER TABLE ONLY public.sender_reputations ALTER COLUMN id SET DEFAULT nextval('public.sender_reputations_id_seq'::regclass);

ALTER TABLE ONLY public.sent_emails ALTER COLUMN id SET DEFAULT nextval('public.sent_emails_id_seq'::regclass);

ALTER TABLE ONLY public.settings ALTER COLUMN id SET DEFAULT nextval('public.settings_id_seq'::regclass);

ALTER TABLE ONLY public.spam_detection_logs ALTER COLUMN id SET DEFAULT nextval('public.spam_detection_logs_id_seq'::regclass);

ALTER TABLE ONLY public.spam_rules ALTER COLUMN id SET DEFAULT nextval('public.spam_rules_id_seq'::regclass);

ALTER TABLE ONLY public.sync_logs ALTER COLUMN id SET DEFAULT nextval('public.sync_logs_id_seq'::regclass);

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);

ALTER TABLE ONLY public.webhook_logs ALTER COLUMN id SET DEFAULT nextval('public.webhook_logs_id_seq'::regclass);

ALTER TABLE ONLY public.webhooks ALTER COLUMN id SET DEFAULT nextval('public.webhooks_id_seq'::regclass);

ALTER TABLE ONLY public.account_groups
    ADD CONSTRAINT account_groups_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.adapters
    ADD CONSTRAINT adapters_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.bayesian_trainings
    ADD CONSTRAINT bayesian_trainings_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.deleted_email_keys
    ADD CONSTRAINT deleted_email_keys_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.email_accounts
    ADD CONSTRAINT email_accounts_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.email_attachments
    ADD CONSTRAINT email_attachments_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.email_label_relations
    ADD CONSTRAINT email_label_relations_pkey PRIMARY KEY (email_id, label_id);

ALTER TABLE ONLY public.email_labels
    ADD CONSTRAINT email_labels_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.email_lists
    ADD CONSTRAINT email_lists_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.email_oauth2_tokens
    ADD CONSTRAINT email_oauth2_tokens_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.email_rules
    ADD CONSTRAINT email_rules_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.emails
    ADD CONSTRAINT emails_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.provider_adapters
    ADD CONSTRAINT provider_adapters_pkey PRIMARY KEY (provider_id, adapter_id);

ALTER TABLE ONLY public.providers
    ADD CONSTRAINT providers_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.sender_reputations
    ADD CONSTRAINT sender_reputations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.sent_emails
    ADD CONSTRAINT sent_emails_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT settings_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.spam_detection_logs
    ADD CONSTRAINT spam_detection_logs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.spam_rules
    ADD CONSTRAINT spam_rules_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.sync_logs
    ADD CONSTRAINT sync_logs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.webhook_logs
    ADD CONSTRAINT webhook_logs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_pkey PRIMARY KEY (id);

CREATE INDEX idx_account_groups_deleted_at ON public.account_groups USING btree (deleted_at);

CREATE UNIQUE INDEX idx_adapters_name ON public.adapters USING btree (name);

CREATE INDEX idx_api_keys_enabled ON public.api_keys USING btree (enabled);

CREATE UNIQUE INDEX idx_api_keys_key_hash ON public.api_keys USING btree (key_hash);

CREATE INDEX idx_bayesian_trainings_email_id ON public.bayesian_trainings USING btree (email_id);

CREATE INDEX idx_bayesian_trainings_is_spam ON public.bayesian_trainings USING btree (is_spam);

CREATE INDEX idx_bayesian_trainings_user_uid ON public.bayesian_trainings USING btree (user_uid);

CREATE INDEX idx_created_at ON public.webhook_logs USING btree (created_at DESC);

CREATE INDEX idx_email_accounts_adapter_id ON public.email_accounts USING btree (adapter_id);

CREATE INDEX idx_email_accounts_deleted_at ON public.email_accounts USING btree (deleted_at);

CREATE INDEX idx_email_accounts_group_id ON public.email_accounts USING btree (group_id);

CREATE INDEX idx_email_accounts_parent_account_uid ON public.email_accounts USING btree (parent_account_uid);

CREATE INDEX idx_email_accounts_provider_id ON public.email_accounts USING btree (provider_id);

CREATE UNIQUE INDEX idx_email_accounts_uid ON public.email_accounts USING btree (uid);

CREATE INDEX idx_email_attachments_email_id ON public.email_attachments USING btree (email_id);

CREATE UNIQUE INDEX idx_email_labels_name ON public.email_labels USING btree (name);

CREATE INDEX idx_email_lists_target ON public.email_lists USING btree (target);

CREATE INDEX idx_email_lists_type ON public.email_lists USING btree (type);

CREATE INDEX idx_email_lists_user_uid ON public.email_lists USING btree (user_uid);

CREATE INDEX idx_email_oauth2_tokens_enabled ON public.email_oauth2_tokens USING btree (enabled);

CREATE INDEX idx_email_oauth2_tokens_is_default ON public.email_oauth2_tokens USING btree (is_default);

CREATE INDEX idx_email_oauth2_tokens_last_used_at ON public.email_oauth2_tokens USING btree (last_used_at);

CREATE INDEX idx_email_oauth2_tokens_provider_id ON public.email_oauth2_tokens USING btree (provider_id);

CREATE INDEX idx_email_rules_account_uid ON public.email_rules USING btree (account_uid);

CREATE INDEX idx_email_rules_enabled ON public.email_rules USING btree (enabled);

CREATE INDEX idx_email_rules_priority ON public.email_rules USING btree (priority);

CREATE INDEX idx_emails_dedupe_key ON public.emails USING btree (dedupe_key);

CREATE INDEX idx_emails_deleted_at ON public.emails USING btree (deleted_at);

CREATE INDEX idx_emails_from_address ON public.emails USING btree (from_address);

CREATE INDEX idx_emails_fulltext_search ON public.emails USING gin (to_tsvector('english'::regconfig, ((((COALESCE(subject, ''::text) || ' '::text) || (COALESCE(from_name, ''::character varying))::text) || ' '::text) || COALESCE(text_body, ''::text))));

CREATE INDEX idx_emails_has_attachment ON public.emails USING btree (has_attachment);

CREATE INDEX idx_emails_is_archived ON public.emails USING btree (is_archived);

CREATE INDEX idx_emails_is_deleted ON public.emails USING btree (is_deleted);

CREATE INDEX idx_emails_is_read ON public.emails USING btree (is_read);

CREATE INDEX idx_emails_is_spam ON public.emails USING btree (is_spam);

CREATE INDEX idx_emails_is_starred ON public.emails USING btree (is_starred);

CREATE INDEX idx_emails_message_id ON public.emails USING btree (message_id);

CREATE INDEX idx_emails_spam_score ON public.emails USING btree (spam_score);

CREATE INDEX idx_emails_to_address ON public.emails USING btree (to_address);

CREATE UNIQUE INDEX idx_provider_account ON public.emails USING btree (provider_id, account_uid);

CREATE INDEX idx_providers_default_adapter_id ON public.providers USING btree (default_adapter_id);

CREATE UNIQUE INDEX idx_providers_name ON public.providers USING btree (name);

CREATE INDEX idx_sender_reputations_domain ON public.sender_reputations USING btree (domain);

CREATE UNIQUE INDEX idx_sender_reputations_email ON public.sender_reputations USING btree (email);

CREATE INDEX idx_sender_reputations_reputation_score ON public.sender_reputations USING btree (reputation_score);

CREATE INDEX idx_sender_reputations_trust_level ON public.sender_reputations USING btree (trust_level);

CREATE INDEX idx_sent_at ON public.emails USING btree (sent_at DESC);

CREATE INDEX idx_sent_emails_account_uid ON public.sent_emails USING btree (account_uid);

CREATE INDEX idx_sent_emails_deleted_at ON public.sent_emails USING btree (deleted_at);

CREATE INDEX idx_sent_emails_forward_from_id ON public.sent_emails USING btree (forward_from_id);

CREATE INDEX idx_sent_emails_message_id ON public.sent_emails USING btree (message_id);

CREATE INDEX idx_sent_emails_reply_to_email_id ON public.sent_emails USING btree (reply_to_email_id);

CREATE INDEX idx_sent_emails_sent_at ON public.sent_emails USING btree (sent_at);

CREATE INDEX idx_sent_emails_status ON public.sent_emails USING btree (status);

CREATE INDEX idx_settings_category ON public.settings USING btree (category);

CREATE INDEX idx_settings_is_sensitive ON public.settings USING btree (is_sensitive);

CREATE INDEX idx_settings_public ON public.settings USING btree (is_public) WHERE (is_public = true);

CREATE INDEX idx_settings_sensitive ON public.settings USING btree (is_sensitive) WHERE (is_sensitive = true);

CREATE INDEX idx_settings_user_category ON public.settings USING btree (user_id, category);

CREATE INDEX idx_settings_user_id ON public.settings USING btree (user_id);

CREATE INDEX idx_spam_detection_logs_email_id ON public.spam_detection_logs USING btree (email_id);

CREATE INDEX idx_spam_detection_logs_is_spam ON public.spam_detection_logs USING btree (is_spam);

CREATE INDEX idx_spam_rules_category ON public.spam_rules USING btree (category);

CREATE INDEX idx_spam_rules_enabled ON public.spam_rules USING btree (enabled);

CREATE INDEX idx_spam_rules_is_builtin ON public.spam_rules USING btree (is_builtin);

CREATE INDEX idx_started_at ON public.sync_logs USING btree (started_at DESC);

CREATE INDEX idx_sync_logs_account_uid ON public.sync_logs USING btree (account_uid);

CREATE INDEX idx_sync_logs_status ON public.sync_logs USING btree (status);

CREATE UNIQUE INDEX idx_users_email ON public.users USING btree (email);

CREATE UNIQUE INDEX idx_users_username ON public.users USING btree (username);

CREATE INDEX idx_webhook_logs_webhook_id ON public.webhook_logs USING btree (webhook_id);

CREATE INDEX idx_webhooks_enabled ON public.webhooks USING btree (enabled);

CREATE UNIQUE INDEX uk_deleted_email_keys ON public.deleted_email_keys USING btree (account_uid, dedupe_key);

CREATE UNIQUE INDEX uk_settings_user_category_key ON public.settings USING btree (user_id, category, key);

ALTER TABLE ONLY public.email_accounts
    ADD CONSTRAINT fk_email_accounts_adapter_ref FOREIGN KEY (adapter_id) REFERENCES public.adapters(id) ON UPDATE CASCADE;

ALTER TABLE ONLY public.email_accounts
    ADD CONSTRAINT fk_email_accounts_provider_ref FOREIGN KEY (provider_id) REFERENCES public.providers(id) ON UPDATE CASCADE;

ALTER TABLE ONLY public.email_oauth2_tokens
    ADD CONSTRAINT fk_email_oauth2_tokens_provider FOREIGN KEY (provider_id) REFERENCES public.providers(id);

ALTER TABLE ONLY public.email_attachments
    ADD CONSTRAINT fk_emails_attachments FOREIGN KEY (email_id) REFERENCES public.emails(id);

ALTER TABLE ONLY public.provider_adapters
    ADD CONSTRAINT fk_provider_adapters_adapter FOREIGN KEY (adapter_id) REFERENCES public.adapters(id);

ALTER TABLE ONLY public.providers
    ADD CONSTRAINT fk_providers_default_adapter FOREIGN KEY (default_adapter_id) REFERENCES public.adapters(id);

ALTER TABLE ONLY public.provider_adapters
    ADD CONSTRAINT fk_providers_supported_adapters FOREIGN KEY (provider_id) REFERENCES public.providers(id);
