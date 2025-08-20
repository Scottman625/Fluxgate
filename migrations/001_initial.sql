-- 初始化數據庫結構

-- 活動表
CREATE TABLE activities (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(50) NOT NULL,
    name VARCHAR(200) NOT NULL,
    sku VARCHAR(100) NOT NULL,
    initial_stock INTEGER NOT NULL CHECK (initial_stock > 0),
    start_at TIMESTAMP NOT NULL,
    end_at TIMESTAMP NOT NULL,
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'paused', 'ended')),
    config_json JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT valid_time_range CHECK (end_at > start_at),
    CONSTRAINT unique_tenant_sku UNIQUE (tenant_id, sku)
);

-- 建立索引
CREATE INDEX idx_activities_tenant_status ON activities (tenant_id, status);
CREATE INDEX idx_activities_time_range ON activities (start_at, end_at);

-- 隊列進入記錄表（用於統計和審計）
CREATE TABLE queue_entries (
    id BIGSERIAL PRIMARY KEY,
    activity_id BIGINT REFERENCES activities(id),
    user_hash VARCHAR(64) NOT NULL,
    session_id VARCHAR(100) NOT NULL,
    seq_number BIGINT NOT NULL,
    fingerprint JSONB,
    ip_hash VARCHAR(64),
    created_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT unique_session_activity UNIQUE (activity_id, session_id)
);

CREATE INDEX idx_queue_entries_activity_seq ON queue_entries (activity_id, seq_number);
CREATE INDEX idx_queue_entries_user_hash ON queue_entries (activity_id, user_hash);

-- 系統配置表
CREATE TABLE system_config (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 插入預設配置
INSERT INTO system_config (key, value, description) VALUES
('default_queue_ttl', '"3600"', '預設隊列 TTL（秒）'),
('max_release_rate', '"1000"', '最大釋放速率（每秒）'),
('default_poll_interval', '"2000"', '預設輪詢間隔（毫秒）');