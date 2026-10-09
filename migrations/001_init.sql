CREATE TABLE IF NOT EXISTS history (
    id TEXT PRIMARY KEY,
    template_name TEXT NOT NULL,
    variables_json TEXT NOT NULL DEFAULT '{}',
    expanded_prompt TEXT NOT NULL,
    completion TEXT NOT NULL,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    pinned INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_history_created_at ON history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_history_pinned ON history(pinned);
