# Checklist

## 配置验证
- [ ] 大模型配置正确加载（API地址、密钥、模型名称）
- [ ] 嵌入模型配置正确加载
- [ ] Milvus配置正确加载

## 数据模型验证
- [ ] Post表summary字段已添加
- [ ] ChatSession表已创建
- [ ] ChatMessage表已创建
- [ ] gorm gen代码已生成

## Milvus集成验证
- [ ] Milvus连接成功
- [ ] Collection自动创建成功
- [ ] 向量插入功能正常
- [ ] 向量更新功能正常
- [ ] 向量删除功能正常
- [ ] 向量相似度搜索功能正常

## 总结生成验证
- [ ] 创建Post时自动生成总结文本
- [ ] 更新Post时自动更新总结文本
- [ ] 总结文本包含时间、地点、物品特征等关键信息
- [ ] 多模态输入（含图片）时总结正确生成
- [ ] 总结文本正确存入数据库

## 向量化验证
- [ ] 总结文本正确向量化
- [ ] 向量正确存入Milvus
- [ ] 向量与post_id正确关联

## Agent工具验证
- [ ] get_post_detail工具正确返回Post详情
- [ ] get_post_detail工具正确处理不存在的Post
- [ ] search_posts工具正确进行向量搜索
- [ ] search_posts工具正确应用筛选条件
- [ ] get_my_posts工具正确返回用户发布列表
- [ ] get_my_claims工具正确返回用户认领申请列表
- [ ] get_my_feedbacks工具正确返回用户投诉反馈列表
- [ ] publish_post工具正确创建Post
- [ ] publish_post工具正确处理信息不完整情况
- [ ] apply_claim工具正确创建认领申请
- [ ] cancel_claim工具正确取消认领申请
- [ ] review_claim工具正确审核认领申请
- [ ] submit_feedback工具正确提交投诉反馈
- [ ] cancel_post工具正确取消发布

## Agent对话验证
- [ ] 创建会话接口正常工作
- [ ] 发送消息接口正常工作
- [ ] 获取历史记录接口正常工作
- [ ] 聊天记录正确存储
- [ ] Agent正确识别用户意图
- [ ] Agent正确调用对应工具
- [ ] Agent正确返回响应

## 权限验证
- [ ] Agent工具正确校验用户权限
- [ ] 非发布者无法审核认领申请
- [ ] 非发布者无法取消他人发布
- [ ] 用户只能查看自己的数据

## API接口验证
- [ ] POST /api/agent/session 创建会话接口正常
- [ ] POST /api/agent/chat 发送消息接口正常
- [ ] GET /api/agent/history 获取历史接口正常
- [ ] GET /api/agent/sessions 获取会话列表接口正常

## 错误处理验证
- [ ] LLM调用失败时有正确的错误处理
- [ ] Milvus连接失败时有正确的错误处理
- [ ] 工具执行失败时Agent正确处理异常

## 日志验证
- [ ] LLM调用有日志记录
- [ ] 向量操作有日志记录
- [ ] Agent对话有日志记录
