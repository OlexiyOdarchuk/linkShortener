CREATE TABLE IF NOT EXISTS clicks (
    user_id Int64,
    short_code String,
    country String,
    city String,
    user_agent String,
    referer String,
    clicked_at DateTime64(3, 'UTC') DEFAULT now64()
)
ENGINE = MergeTree()
ORDER BY (user_id, short_code, clicked_at)