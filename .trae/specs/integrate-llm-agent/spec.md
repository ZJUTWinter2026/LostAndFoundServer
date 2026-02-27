# 大模型Agent助手集成 Spec

## Why
当前校园失物招领系统仅支持传统的表单操作方式，用户需要手动填写各种信息并逐步操作。为了提升用户体验，需要集成一个大模型Agent助手，让用户可以通过自然语言对话的方式完成系统操作，同时利用向量搜索实现智能化的失物招领匹配。

## What Changes
- 在业务配置中添加大模型相关配置（OpenAI格式）和嵌入模型配置
- 在业务配置中添加Milvus向量数据库配置
- Post表添加summary字段用于存储AI生成的总结文本
- 新增向量存储相关代码，实现总结文本的向量化和Milvus存储
- 新增Agent对话相关API接口
- 新增聊天记录存储功能
- 实现多个Tool为Agent提供系统能力

## Impact
- Affected specs: 新增AI助手能力、向量搜索能力
- Affected code: 
  - `comm/config.go` - 添加新配置结构
  - `deploy/sql/post.sql` - 添加summary字段
  - `deploy/sql/chat_session.sql` - 新增聊天会话表
  - `deploy/sql/chat_message.sql` - 新增聊天消息表
  - `dao/repo/` - 新增向量存储仓库
  - `api/agent/` - 新增Agent相关API
  - `register/route.go` - 注册新路由

## Important Notes
- **所有 `.gen.go` 后缀的文件由 gorm gen 根据 SQL 自动生成，不要直接修改**
- 需要修改数据模型时，先修改 `deploy/sql/` 目录下的 SQL 文件
- 修改 SQL 后，需要用户手动运行 gorm gen 命令重新生成代码

## ADDED Requirements

### Requirement: 大模型配置管理
系统SHALL支持通过配置文件配置大模型服务，包括：
- API地址（支持OpenAI兼容格式）
- API密钥
- 模型名称
- 嵌入模型配置

#### Scenario: 配置加载成功
- **WHEN** 系统启动时
- **THEN** 系统正确加载大模型和嵌入模型配置

### Requirement: Milvus向量数据库集成
系统SHALL支持Milvus向量数据库的配置和连接，用于存储和检索Post的向量表示。

#### Scenario: Milvus连接成功
- **WHEN** 系统启动时
- **THEN** 系统成功连接Milvus并创建必要的Collection

### Requirement: Post总结生成与向量化
系统SHALL在Post创建或更新时自动生成总结文本并进行向量化存储。

#### Scenario: 创建Post时生成总结
- **WHEN** 用户创建新的失物/招领信息
- **THEN** 系统调用大模型生成包含时间、地点、物品特征的总结文本
- **AND** 系统将总结文本存入Post表的summary字段
- **AND** 系统调用嵌入模型将总结文本向量化
- **AND** 系统将向量存入Milvus并关联post_id

#### Scenario: 更新Post时更新总结
- **WHEN** 用户更新失物/招领信息
- **THEN** 系统重新生成总结文本并更新向量化数据

#### Scenario: 多模态输入处理
- **WHEN** Post包含图片时
- **THEN** 系统将图片一并传给多模态大模型进行分析，生成更准确的总结

### Requirement: Agent对话接口
系统SHALL提供Agent对话相关API接口，支持用户与AI助手进行交互。

#### Scenario: 发起对话
- **WHEN** 用户发送消息给Agent
- **THEN** 系统创建或继续对话会话
- **AND** 系统将用户消息存入聊天记录
- **AND** 系统调用Agent处理消息
- **AND** 系统返回Agent响应
- **AND** 系统将Agent响应存入聊天记录

#### Scenario: 获取对话历史
- **WHEN** 用户请求获取对话历史
- **THEN** 系统返回该会话的所有聊天记录

#### Scenario: 创建新会话
- **WHEN** 用户请求创建新会话
- **THEN** 系统创建新的对话会话并返回会话ID

### Requirement: 聊天记录存储
系统SHALL持久化存储用户的聊天记录，支持多会话管理。

#### Scenario: 存储聊天记录
- **WHEN** 用户与Agent进行对话
- **THEN** 系统将每条消息（用户消息和Agent响应）存入数据库

### Requirement: Agent Tool - get_post_detail
系统SHALL提供get_post_detail工具，允许Agent根据post_id获取失物/招领信息的详细内容。

#### Scenario: 获取Post详情成功
- **WHEN** Agent调用get_post_detail工具并传入有效的post_id
- **THEN** 系统返回该Post的完整详情信息

#### Scenario: Post不存在
- **WHEN** Agent调用get_post_detail工具但post_id不存在
- **THEN** 系统返回错误信息

### Requirement: Agent Tool - search_posts
系统SHALL提供search_posts工具，允许Agent通过自然语言query进行向量搜索，并支持筛选条件。

#### Scenario: 向量搜索成功
- **WHEN** Agent调用search_posts工具并传入自然语言query
- **THEN** 系统将query向量化
- **AND** 系统在Milvus中进行相似度搜索
- **AND** 系统返回相关的Post列表

#### Scenario: 带筛选条件的搜索
- **WHEN** Agent调用search_posts工具并传入publish_type、campus等筛选参数
- **THEN** 系统在向量搜索的基础上应用筛选条件
- **AND** 系统返回符合条件的Post列表

### Requirement: Agent Tool - get_my_posts
系统SHALL提供get_my_posts工具，允许Agent获取当前用户发布的失物/招领信息列表。

#### Scenario: 获取我的发布列表
- **WHEN** Agent调用get_my_posts工具
- **THEN** 系统返回当前用户发布的所有Post列表

### Requirement: Agent Tool - get_my_claims
系统SHALL提供get_my_claims工具，允许Agent获取当前用户提交的认领申请列表。

#### Scenario: 获取我的认领申请列表
- **WHEN** Agent调用get_my_claims工具
- **THEN** 系统返回当前用户提交的所有认领申请列表

### Requirement: Agent Tool - get_my_feedbacks
系统SHALL提供get_my_feedbacks工具，允许Agent获取当前用户提交的投诉反馈列表。

#### Scenario: 获取我的投诉反馈列表
- **WHEN** Agent调用get_my_feedbacks工具
- **THEN** 系统返回当前用户提交的所有投诉反馈列表

### Requirement: Agent Tool - publish_post
系统SHALL提供publish_post工具，允许Agent根据对话中收集的信息帮用户发布失物或招领信息。

#### Scenario: 发布成功
- **WHEN** Agent调用publish_post工具并提供完整的发布信息
- **THEN** 系统创建新的Post记录
- **AND** 系统生成总结文本并存储
- **AND** 系统进行向量化处理
- **AND** 系统返回新创建的Post ID

#### Scenario: 信息不完整
- **WHEN** Agent调用publish_post工具但缺少必要信息
- **THEN** 系统返回错误信息，提示缺少的字段

### Requirement: Agent Tool - apply_claim
系统SHALL提供apply_claim工具，允许Agent帮用户申请认领物品。

#### Scenario: 认领申请成功
- **WHEN** Agent调用apply_claim工具并提供post_id和认领说明
- **THEN** 系统创建新的认领申请记录
- **AND** 系统返回认领申请ID

### Requirement: Agent Tool - cancel_claim
系统SHALL提供cancel_claim工具，允许Agent帮用户取消认领申请。

#### Scenario: 取消认领成功
- **WHEN** Agent调用cancel_claim工具并提供有效的claim_id
- **THEN** 系统删除该认领申请
- **AND** 系统返回成功信息

### Requirement: Agent Tool - review_claim
系统SHALL提供review_claim工具，允许Agent帮用户（发布者或管理员）审核认领申请。

#### Scenario: 审核认领成功
- **WHEN** Agent调用review_claim工具并提供claim_id和审核结果
- **THEN** 系统更新认领申请状态
- **AND** 如果同意认领，系统更新Post状态为已解决

### Requirement: Agent Tool - submit_feedback
系统SHALL提供submit_feedback工具，允许Agent帮用户提交投诉反馈。

#### Scenario: 提交投诉成功
- **WHEN** Agent调用submit_feedback工具并提供post_id和投诉内容
- **THEN** 系统创建新的投诉反馈记录
- **AND** 系统返回反馈ID

### Requirement: Agent Tool - cancel_post
系统SHALL提供cancel_post工具，允许Agent帮用户取消已通过的发布信息。

#### Scenario: 取消发布成功
- **WHEN** Agent调用cancel_post工具并提供post_id
- **THEN** 系统更新Post状态为已取消

### Requirement: Agent上下文管理
系统SHALL在Agent处理请求时注入用户上下文信息，包括用户ID、用户类型等。

#### Scenario: 上下文注入
- **WHEN** Agent处理用户请求时
- **THEN** 系统自动注入当前用户的身份信息
- **AND** Agent可以根据用户身份执行权限控制

### Requirement: Agent意图识别
系统SHALL让Agent能够识别用户意图并选择合适的工具执行操作。

#### Scenario: 识别搜索意图
- **WHEN** 用户询问"有没有人捡到黑色钱包"
- **THEN** Agent识别为搜索意图
- **AND** Agent调用search_posts工具进行向量搜索

#### Scenario: 识别发布意图
- **WHEN** 用户说"我丢了一个红色的雨伞"
- **THEN** Agent识别为发布意图
- **AND** Agent收集必要信息后调用publish_post工具

## MODIFIED Requirements

### Requirement: Post数据模型扩展
Post模型SHALL新增summary字段用于存储AI生成的总结文本。

#### Scenario: 数据库迁移
- **WHEN** 系统部署时
- **THEN** 数据库Post表新增summary字段（TEXT类型）
- **NOTE** 需要修改 `deploy/sql/post.sql` 后由用户手动运行 gorm gen

## REMOVED Requirements
无移除的需求。
