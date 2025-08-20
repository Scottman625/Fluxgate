-- 插入測試活動數據

-- 測試活動 1
INSERT INTO activities (
    tenant_id, 
    name, 
    sku, 
    initial_stock, 
    start_at, 
    end_at, 
    status, 
    config_json
) VALUES (
    'test-tenant',
    '測試活動 1',
    'test-activity-1',
    1000,
    NOW() - INTERVAL '1 hour',
    NOW() + INTERVAL '24 hours',
    'active',
    '{"release_rate": 10, "max_queue_size": 5000}'
);

-- 測試活動 2
INSERT INTO activities (
    tenant_id, 
    name, 
    sku, 
    initial_stock, 
    start_at, 
    end_at, 
    status, 
    config_json
) VALUES (
    'test-tenant',
    '測試活動 2',
    'test-activity-2',
    500,
    NOW() - INTERVAL '30 minutes',
    NOW() + INTERVAL '12 hours',
    'active',
    '{"release_rate": 5, "max_queue_size": 2000}'
);

-- 插入系統配置
INSERT INTO system_config (key, value, description) VALUES
('default_activity_id', '1', '預設活動 ID'),
('queue_ttl_seconds', '3600', '隊列 TTL（秒）'),
('max_queue_size', '10000', '最大隊列大小'),
('poll_interval_ms', '2000', '輪詢間隔（毫秒）')
ON CONFLICT (key) DO UPDATE SET 
    value = EXCLUDED.value,
    updated_at = NOW();
