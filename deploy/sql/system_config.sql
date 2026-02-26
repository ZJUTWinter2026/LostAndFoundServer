CREATE TABLE `system_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增ID',
  `config_key` varchar(64) NOT NULL COMMENT '配置键名',
  `config_value` text NOT NULL COMMENT '配置值(JSON格式)',
  `description` varchar(255) DEFAULT '' COMMENT '描述',
  `created_at` timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at` timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_config_key` (`config_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统配置表';

INSERT INTO `system_config` (`config_key`, `config_value`, `description`) VALUES
('feedback_types', '["恶意发布","信息不全","不实消息","恶心血腥","涉黄信息","其它类型"]', '用户投诉与反馈类型'),
('item_types', '["电子","饭卡","文体","证件","衣包","饰品","其它类型"]', '物品类型分类'),
('claim_validity_days', '30', '认领时效(天)');
