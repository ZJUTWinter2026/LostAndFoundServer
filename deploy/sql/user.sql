CREATE TABLE `user` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增ID',
  `uid` bigint(20) NOT NULL COMMENT '学号/工号',
  `name` varchar(50) DEFAULT '' COMMENT '姓名',
  `id_card` varchar(18) DEFAULT '' COMMENT '身份证号',
  `password` varchar(255) NOT NULL COMMENT '密码',
  `usertype` varchar(32) NOT NULL DEFAULT 'STUDENT' COMMENT '用户类型: STUDENT, ADMIN, SYSTEM_ADMIN',
  `first_login` BOOLEAN NOT NULL DEFAULT TRUE COMMENT '首次登陆',
  `disabled_until` datetime DEFAULT NULL COMMENT '禁用截止时间',
  `created_at` timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at` timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_uid` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';
