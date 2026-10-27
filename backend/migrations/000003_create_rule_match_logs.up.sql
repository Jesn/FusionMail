-- 规则命中记录表：单封邮件维度的规则命中审计
CREATE TABLE IF NOT EXISTS rule_match_logs (
    id BIGSERIAL PRIMARY KEY,
    email_id BIGINT NOT NULL,
    rule_id BIGINT NOT NULL,
    rule_name VARCHAR(255) NOT NULL,
    actions TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_rule_match_logs_email_id ON rule_match_logs(email_id);
CREATE INDEX IF NOT EXISTS idx_rule_match_logs_rule_id ON rule_match_logs(rule_id);
