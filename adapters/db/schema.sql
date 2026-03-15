-- Registered services table
CREATE TABLE IF NOT EXISTS registered_services (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    installation_id BIGINT NOT NULL,
    owner VARCHAR(255) NOT NULL,
    repository VARCHAR(255) NOT NULL,
    config_paths JSON NOT NULL COMMENT 'Array of file paths to fetch, e.g. ["gateway/httproute.yaml"]',
    branch VARCHAR(255) DEFAULT 'main',
    metadata JSON COMMENT 'Flexible metadata: service_name, team, environment, etc.',
    registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_service (owner, repository),
    INDEX idx_owner (owner)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Config fetch history (audit trail)
CREATE TABLE IF NOT EXISTS config_fetch_history (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    service_id BIGINT NOT NULL,
    owner VARCHAR(255) NOT NULL,
    repository VARCHAR(255) NOT NULL,
    commit_sha VARCHAR(40) NOT NULL,
    branch VARCHAR(255) NOT NULL,
    files_fetched JSON NOT NULL COMMENT 'Array of {path, content, config_type}',
    fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status ENUM('pending', 'success', 'failed') NOT NULL DEFAULT 'pending',
    error_message TEXT,
    INDEX idx_service (service_id),
    INDEX idx_commit (commit_sha),
    INDEX idx_status (status),
    INDEX idx_fetched_at (fetched_at),
    FOREIGN KEY (service_id) REFERENCES registered_services(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE services (
    service_id VARCHAR(50) PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL,
    team_name VARCHAR(100) NOT NULL,
    namespace VARCHAR(100) NOT NULL,
    contact_email VARCHAR(255) NOT NULL,
    config_endpoint TEXT NOT NULL,
    webhook_url TEXT NOT NULL,
    metadata JSON NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_service (service_name, team_name, namespace)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;