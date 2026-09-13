-- 外设租借与损坏赔付模块表结构（GORM AutoMigrate 会自动执行，此脚本供参考与初始化）
-- 会员欠款字段：外设损坏赔偿押金不足部分计入 users.debt
-- 注意：MySQL 8.0 不支持 ADD COLUMN IF NOT EXISTS，重复执行请先确认列是否存在
ALTER TABLE users ADD COLUMN debt DECIMAL(12,2) DEFAULT 0 AFTER balance;

CREATE TABLE IF NOT EXISTS peripherals (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    device_no VARCHAR(32) NOT NULL UNIQUE,
    device_type VARCHAR(16) NOT NULL,
    name VARCHAR(64) NOT NULL,
    status VARCHAR(16) DEFAULT 'available',
    created_at DATETIME(3),
    updated_at DATETIME(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS peripheral_rentals (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    rental_no VARCHAR(40) NOT NULL UNIQUE,
    user_id BIGINT UNSIGNED NOT NULL,
    peripheral_id BIGINT UNSIGNED NOT NULL,
    device_no VARCHAR(32) NOT NULL,
    device_type VARCHAR(16) NOT NULL,
    deposit DECIMAL(12,2) DEFAULT 0,
    expected_return_at DATETIME(3) NOT NULL,
    returned_at DATETIME(3) NULL,
    status VARCHAR(16) DEFAULT 'renting',
    damage_desc VARCHAR(255) DEFAULT '',
    compensation DECIMAL(12,2) DEFAULT 0,
    handler_id BIGINT UNSIGNED DEFAULT 0,
    handler_name VARCHAR(64) DEFAULT '',
    debt_amount DECIMAL(12,2) DEFAULT 0,
    refund_amount DECIMAL(12,2) DEFAULT 0,
    remark VARCHAR(255) DEFAULT '',
    created_at DATETIME(3),
    updated_at DATETIME(3),
    KEY idx_peripheral_rentals_user (user_id),
    KEY idx_peripheral_rentals_peripheral (peripheral_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
