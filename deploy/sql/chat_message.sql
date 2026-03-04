CREATE TABLE `chat_message`
(
    `id`                 bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `session_id`         varchar(64)     NOT NULL COMMENT '会话ID',
    `role`               varchar(32)     NOT NULL COMMENT '角色: user, assistant, system, tool',
    `content`            text            NOT NULL COMMENT '消息内容',
    `images`             text            NOT NULL COMMENT '图片URL列表(JSON数组)',
    `image_descriptions` text            NOT NULL COMMENT '图片描述列表(JSON数组)',
    `tool_data`          text            NOT NULL COMMENT '工具调用元数据(JSON)',
    `created_at`         timestamp(3)    NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `deleted_at`         bigint unsigned          DEFAULT '0' COMMENT '删除时间 (软删除)',
    PRIMARY KEY (`id`),
    KEY `idx_session_id` (`session_id`),
    KEY `idx_created_at` (`created_at`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='聊天消息表';
