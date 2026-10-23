-- +goose Up
CREATE TABLE IF NOT EXISTS urls (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code         varchar(10) NOT NULL,
    original_url text        NOT NULL,
    created_at   timestamptz,
    clicks       bigint      DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_code ON urls (code);
CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls (created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS urls;
