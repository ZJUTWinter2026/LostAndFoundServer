CREATE TABLE `announcement` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增ID',
  `title` varchar(255) NOT NULL COMMENT '标题',
  `content` text NOT NULL COMMENT '内容',
  `type` varchar(32) NOT NULL DEFAULT 'SYSTEM' COMMENT '类型: SYSTEM系统公告, REGION区域公告',
  `status` varchar(32) NOT NULL DEFAULT 'PENDING' COMMENT '状态: PENDING待审核, APPROVED已通过',
  `publisher_id` bigint(20) NOT NULL COMMENT '发布者ID',
  `reviewed_by` bigint(20) DEFAULT NULL COMMENT '审核人ID',
  `reviewed_at` datetime DEFAULT NULL COMMENT '审核时间',
  `created_at` timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at` timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  `deleted_at` bigint(20) unsigned DEFAULT '0' COMMENT '删除时间 (软删除)',
  PRIMARY KEY (`id`),
  KEY `idx_type` (`type`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='公告通知表';
