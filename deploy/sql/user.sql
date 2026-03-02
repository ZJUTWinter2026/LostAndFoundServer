CREATE TABLE `user`
(
    `id`             bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `username`       varchar(50)     NOT NULL COMMENT '用户名(学号/工号)',
    `name`           varchar(50)     NOT NULL COMMENT '姓名',
    `id_card`        varchar(18)     NOT NULL COMMENT '身份证号',
    `password`       varchar(255)    NOT NULL COMMENT '密码',
    `usertype`       varchar(32)     NOT NULL DEFAULT 'STUDENT' COMMENT '用户类型: STUDENT, ADMIN, SYSTEM_ADMIN',
    `campus`         varchar(32)     NOT NULL DEFAULT '' COMMENT '所属校区: ZHAO_HUI, PING_FENG, MO_GAN_SHAN, 仅管理员有效',
    `first_login`    BOOLEAN         NOT NULL DEFAULT TRUE COMMENT '首次登陆',
    `disabled_until` timestamp(3)    NULL COMMENT '禁用截止时间',
    `created_at`     timestamp(3)    NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at`     timestamp(3)    NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_username` (`username`),
    KEY `idx_usertype` (`usertype`),
    KEY `idx_campus` (`campus`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='用户表';
