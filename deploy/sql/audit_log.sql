CREATE TABLE `audit_log` (
  `id` BIGINT UNSIGNED AUTO_INCREMENT COMMENT '自增ID',
  `admin_id` BIGINT NOT NULL COMMENT '管理员ID',
  `action_type` VARCHAR(32) NOT NULL COMMENT '操作类型 LOGIN/CREATE/UPDATE/DELETE',
  `reason` VARCHAR(500) DEFAULT NULL COMMENT '理由',
  `post_id` BIGINT NOT NULL COMMENT '发布信息ID',
  `old_status` VARCHAR(32) NOT NULL COMMENT '旧状态',
  `new_status` VARCHAR(32) NOT NULL COMMENT '新状态',
  `created_at` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  `deleted_at` BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间 (软删除)',
  PRIMARY KEY (`id`),
  KEY `idx_admin_id` (`admin_id`),
  KEY `idx_post_id` (`post_id`),
  KEY `idx_action_type` (`action_type`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='审计日志表';

