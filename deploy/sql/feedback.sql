CREATE TABLE `feedback` (
  `id` BIGINT UNSIGNED AUTO_INCREMENT COMMENT '自增ID',
  `post_id` BIGINT NOT NULL COMMENT '物品ID',
  `reporter_id` BIGINT NOT NULL COMMENT '投诉者ID',
  `type` VARCHAR(50) NOT NULL COMMENT '投诉类型',
  `type_other` VARCHAR(15) DEFAULT NULL COMMENT '其它类型说明',
  `description` VARCHAR(500) DEFAULT NULL COMMENT '详细说明',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态 0未处理 1已处理',
  `processed_by` BIGINT DEFAULT NULL COMMENT '处理人ID',
  `processed_at` TIMESTAMP(3) DEFAULT NULL COMMENT '处理时间',
  `created_at` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  `deleted_at` BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间 (软删除)',
  PRIMARY KEY (`id`),
  KEY `idx_post_id` (`post_id`),
  KEY `idx_reporter_id` (`reporter_id`),
  KEY `idx_status` (`status`),
  KEY `idx_type` (`type`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='投诉反馈表';

