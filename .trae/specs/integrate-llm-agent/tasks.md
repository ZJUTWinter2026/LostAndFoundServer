# Tasks

## Phase 1: 基础设施配置

- [x] Task 1: 添加大模型和向量数据库配置
  - [x] SubTask 1.1: 在`comm/config.go`中添加LLMConfig结构体（API地址、API密钥、模型名称等）
  - [x] SubTask 1.2: 在`comm/config.go`中添加EmbeddingConfig结构体（API地址、API密钥、模型名称、向量维度等）
  - [x] SubTask 1.3: 在`comm/config.go`中添加MilvusConfig结构体（地址、Collection名称等）
  - [x] SubTask 1.4: 更新`conf/config.example.yaml`添加配置示例

- [x] Task 2: 添加eino框架依赖
  - [x] SubTask 2.1: 在go.mod中添加eino相关依赖
  - [x] SubTask 2.2: 添加milvus-sdk-go依赖
  - [x] SubTask 2.3: 执行go mod tidy更新依赖

## Phase 2: 数据模型扩展

- [x] Task 3: 扩展Post模型
  - [x] SubTask 3.1: 修改`deploy/sql/post.sql`添加summary字段（TEXT类型）
  - [ ] SubTask 3.2: **[用户手动执行]** 运行gorm gen重新生成`dao/model/post.gen.go`
  - [x] SubTask 3.3: 更新PostRepo支持Summary字段的读写

- [x] Task 4: 创建聊天记录数据模型SQL
  - [x] SubTask 4.1: 创建`deploy/sql/chat_session.sql`定义ChatSession表（会话ID、用户ID、创建时间、标题等）
  - [x] SubTask 4.2: 创建`deploy/sql/chat_message.sql`定义ChatMessage表（消息ID、会话ID、角色、内容、创建时间等）
  - [ ] SubTask 4.3: **[用户手动执行]** 运行gorm gen生成`dao/model/chat_session.gen.go`和`dao/model/chat_message.gen.go`
  - [ ] SubTask 4.4: **[用户手动执行]** 运行gorm gen生成`dao/query/`相关代码

- [ ] Task 5: 创建聊天记录仓库
  - [ ] SubTask 5.1: 创建`dao/repo/chat_session_repo.go`实现会话CRUD
  - [ ] SubTask 5.2: 创建`dao/repo/chat_message_repo.go`实现消息CRUD

## Phase 3: 向量存储集成

- [x] Task 6: 实现Milvus客户端封装
  - [x] SubTask 6.1: 创建`pkg/milvus/client.go`封装Milvus连接和Collection管理
  - [x] SubTask 6.2: 实现Collection自动创建（包含post_id字段和向量字段）
  - [x] SubTask 6.3: 实现向量插入、更新、删除方法
  - [x] SubTask 6.4: 实现向量相似度搜索方法

- [x] Task 7: 实现向量存储仓库
  - [x] SubTask 7.1: 创建`dao/repo/vector_repo.go`封装向量存储操作
  - [x] SubTask 7.2: 实现InsertPostVector方法（插入Post向量）
  - [x] SubTask 7.3: 实现UpdatePostVector方法（更新Post向量）
  - [x] SubTask 7.4: 实现DeletePostVector方法（删除Post向量）
  - [x] SubTask 7.5: 实现SearchSimilarPosts方法（向量相似度搜索）

## Phase 4: 大模型服务集成

- [x] Task 8: 实现LLM服务封装
  - [x] SubTask 8.1: 创建`pkg/llm/chat_model.go`封装OpenAI兼容的ChatModel
  - [x] SubTask 8.2: 创建`pkg/llm/embedding.go`封装嵌入模型调用
  - [x] SubTask 8.3: 实现多模态消息构建（支持文本和图片URL）
  - [x] SubTask 8.4: 实现总结文本生成的Prompt模板

- [x] Task 9: 实现Post总结生成服务
  - [x] SubTask 9.1: 创建`service/summary_service.go`
  - [x] SubTask 9.2: 实现GeneratePostSummary方法（调用多模态大模型生成总结）
  - [x] SubTask 9.3: 定义总结生成的Prompt模板（提取时间、地点、物品特征等关键信息）

## Phase 5: Agent工具实现

- [x] Task 10: 创建Agent工具基础结构
  - [x] SubTask 10.1: 创建`agent/tools/`目录结构
  - [x] SubTask 10.2: 创建工具基础接口定义
  - [x] SubTask 10.3: 创建工具上下文结构（包含用户ID、权限等）

- [x] Task 11: 实现get_post_detail工具
  - [x] SubTask 11.1: 创建`agent/tools/get_post_detail.go`
  - [x] SubTask 11.2: 定义工具的Input/Output结构
  - [x] SubTask 11.3: 实现工具逻辑（调用PostRepo获取详情）
  - [x] SubTask 11.4: 添加工具描述信息供LLM理解

- [x] Task 12: 实现search_posts工具
  - [x] SubTask 12.1: 创建`agent/tools/search_posts.go`
  - [x] SubTask 12.2: 定义工具的Input/Output结构（支持publish_type、campus筛选）
  - [x] SubTask 12.3: 实现向量搜索逻辑（调用VectorRepo）
  - [x] SubTask 12.4: 添加工具描述信息

- [x] Task 13: 实现get_my_posts工具
  - [x] SubTask 13.1: 创建`agent/tools/get_my_posts.go`
  - [x] SubTask 13.2: 实现获取当前用户发布列表逻辑

- [x] Task 14: 实现get_my_claims工具
  - [x] SubTask 14.1: 创建`agent/tools/get_my_claims.go`
  - [x] SubTask 14.2: 实现获取当前用户认领申请列表逻辑

- [x] Task 15: 实现get_my_feedbacks工具
  - [x] SubTask 15.1: 创建`agent/tools/get_my_feedbacks.go`
  - [x] SubTask 15.2: 实现获取当前用户投诉反馈列表逻辑

- [x] Task 16: 实现publish_post工具
  - [x] SubTask 16.1: 创建`agent/tools/publish_post.go`
  - [x] SubTask 16.2: 定义工具的Input结构（包含所有发布必要字段）
  - [x] SubTask 16.3: 实现发布逻辑（创建Post、生成总结、向量化存储）
  - [x] SubTask 16.4: 添加字段校验逻辑

- [x] Task 17: 实现apply_claim工具
  - [x] SubTask 17.1: 创建`agent/tools/apply_claim.go`
  - [x] SubTask 17.2: 实现认领申请逻辑

- [x] Task 18: 实现cancel_claim工具
  - [x] SubTask 18.1: 创建`agent/tools/cancel_claim.go`
  - [x] SubTask 18.2: 实现取消认领申请逻辑

- [x] Task 19: 实现review_claim工具
  - [x] SubTask 19.1: 创建`agent/tools/review_claim.go`
  - [x] SubTask 19.2: 实现审核认领申请逻辑（包含权限校验）

- [x] Task 20: 实现submit_feedback工具
  - [x] SubTask 20.1: 创建`agent/tools/submit_feedback.go`
  - [x] SubTask 20.2: 实现提交投诉反馈逻辑

- [x] Task 21: 实现cancel_post工具
  - [x] SubTask 21.1: 创建`agent/tools/cancel_post.go`
  - [x] SubTask 21.2: 实现取消发布逻辑

## Phase 6: Agent核心实现

- [x] Task 22: 创建Agent服务
  - [x] SubTask 22.1: 创建`agent/agent.go`定义Agent结构
  - [x] SubTask 22.2: 使用eino框架创建ChatModelAgent
  - [x] SubTask 22.3: 注册所有工具到Agent
  - [x] SubTask 22.4: 实现对话处理方法（支持流式响应）

- [x] Task 23: 创建Agent服务层
  - [x] SubTask 23.1: 创建`service/agent_service.go`
  - [x] SubTask 23.2: 实现Chat方法（处理用户消息、调用Agent、保存聊天记录）
  - [x] SubTask 23.3: 实现CreateSession方法
  - [x] SubTask 23.4: 实现GetChatHistory方法

## Phase 7: API接口实现

- [x] Task 24: 创建Agent API接口
  - [x] SubTask 24.1: 创建`api/agent/chat.go`实现对话接口
  - [x] SubTask 24.2: 创建`api/agent/session.go`实现会话管理接口
  - [x] SubTask 24.3: 创建`api/agent/history.go`实现历史记录接口

- [x] Task 25: 注册路由
  - [x] SubTask 25.1: 在`register/route.go`中注册Agent相关路由

## Phase 8: 业务集成

- [x] Task 26: 集成总结生成到发布流程
  - [x] SubTask 26.1: 修改`api/post/publish.go`在创建Post后调用总结生成服务
  - [ ] SubTask 26.2: 修改`api/post/update.go`在更新Post后重新生成总结

- [x] Task 27: 添加Milvus初始化引导
  - [x] SubTask 27.1: 在`register/boot.go`中添加Milvus连接引导器
  - [x] SubTask 27.2: 实现Collection自动创建逻辑

## Phase 9: 错误处理与优化

- [x] Task 28: 添加错误码
  - [x] SubTask 28.1: 在`comm/code.go`中添加Agent相关错误码
  - [x] SubTask 28.2: 添加向量存储相关错误码

- [x] Task 29: 添加日志和监控
  - [x] SubTask 29.1: 在关键路径添加日志记录
  - [x] SubTask 29.2: 添加LLM调用耗时统计

# Task Dependencies
- [Task 2] depends on [Task 1]
- [Task 3] depends on [Task 1] (SQL修改后需用户手动生成代码)
- [Task 4] depends on [Task 2] (SQL创建后需用户手动生成代码)
- [Task 5] depends on [Task 4] (等待用户生成.gen.go文件后)
- [Task 6] depends on [Task 1]
- [Task 7] depends on [Task 6]
- [Task 8] depends on [Task 1, Task 2]
- [Task 9] depends on [Task 8]
- [Task 10] depends on [Task 2]
- [Task 11-21] depend on [Task 10]
- [Task 22] depends on [Task 8, Task 10]
- [Task 23] depends on [Task 22]
- [Task 24] depends on [Task 23]
- [Task 25] depends on [Task 24]
- [Task 26] depends on [Task 9, Task 7]
- [Task 27] depends on [Task 6]

# Important Notes
- 所有 `.gen.go` 后缀的文件由 gorm gen 根据 SQL 自动生成，不要直接修改
- 需要修改数据模型时，先修改 `deploy/sql/` 目录下的 SQL 文件
- 修改 SQL 后，需要用户手动运行 gorm gen 命令重新生成代码
- **Task 5 需要用户先运行 gorm gen 生成聊天记录相关的 model 文件后才能实现**
