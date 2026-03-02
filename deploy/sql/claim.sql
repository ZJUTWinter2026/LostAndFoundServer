CREATE TABLE `claim`
(
    `id`           BIGINT UNSIGNED AUTO_INCREMENT COMMENT '自增ID',
    `post_id`      BIGINT       NOT NULL COMMENT '发布记录ID',
    `claimant_id`  BIGINT       NOT NULL COMMENT '认领者ID',
    `description`  VARCHAR(500) NOT NULL COMMENT '补充说明',
    `proof_images` TEXT         NOT NULL COMMENT '证明图片',
    `status`       VARCHAR(32)  NOT NULL DEFAULT 'PENDING' COMMENT '状态 PENDING待确认 MATCHED已匹配 REJECTED已拒绝',
    `reviewed_by`  BIGINT       NOT NULL COMMENT '审核人ID',
    `reviewed_at`  TIMESTAMP(3) NULL COMMENT '审核时间',
    `created_at`   TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at`   TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    `deleted_at`   BIGINT       NOT NULL DEFAULT 0 COMMENT '删除时间 (软删除)',
    PRIMARY KEY (`id`),
    KEY `idx_post_id` (`post_id`),
    KEY `idx_claimant_id` (`claimant_id`),
    KEY `idx_status` (`status`),
    KEY `idx_deleted_at` (`deleted_at`),
    KEY `idx_post_status` (`post_id`, `status`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT ='认领申请表';

