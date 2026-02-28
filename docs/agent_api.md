# Agent API 接口文档

本文档描述了校园失物招领系统的AI助手相关接口，供前端工程师对接使用。

## 基础信息

- 所有接口需要用户登录认证
- 基础路径: `/api/agent`
- 响应格式: JSON

**注意：** Agent功能需要后端配置启用。如果功能禁用，所有agent接口将返回错误码`30108`。

---

## 1. 创建会话

创建一个新的对话会话。

### 请求

- **方法**: POST
- **路径**: `/session`
- **Content-Type**: application/json

### 请求参数

```json
{
  "title": "会话标题（可选）"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 否 | 会话标题，如果不提供会自动使用第一条消息作为标题 |

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| session_id | string | 会话唯一标识，UUID格式 |

---

## 2. 获取会话列表

获取当前用户的所有对话会话。

### 请求

- **方法**: GET
- **路径**: `/sessions`

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "sessions": [
      {
        "session_id": "550e8400-e29b-41d4-a716-446655440000",
        "title": "寻找丢失的雨伞",
        "created_at": "2024-01-15 10:30:00",
        "updated_at": "2024-01-15 11:45:00"
      }
    ]
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| sessions | array | 会话列表 |
| sessions[].session_id | string | 会话ID |
| sessions[].title | string | 会话标题 |
| sessions[].created_at | string | 创建时间 |
| sessions[].updated_at | string | 最后更新时间 |

---

## 3. 发送消息（流式）

使用SSE（Server-Sent Events）进行流式对话，实时显示AI回复和工具调用过程。

### 请求

- **方法**: POST
- **路径**: `/stream`
- **Content-Type**: application/json

### 请求参数

```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "帮我搜索一下有没有人捡到黑色雨伞",
  "images": [
    "https://example.com/image1.jpg"
  ]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| session_id | string | 是 | 会话ID |
| message | string | 是 | 用户消息内容 |
| images | []string | 否 | 图片URL列表，支持多图 |

### 图片处理机制

当用户上传图片时，系统会：
1. 使用VisionLLM对每张图片进行内容描述
2. 将图片描述和URL注入到对话上下文中
3. 主LLM可以理解图片内容并正确使用图片URL

这样确保了：
- LLM能理解图片内容（如"一把黑色雨伞"）
- LLM知道图片对应的URL，可在工具调用中正确使用

### 响应

响应为SSE流，Content-Type为`text/event-stream`。

#### 事件类型

##### 3.1 内容事件 (content)

AI生成的文本内容，可能多次发送。

```json
data: {"type":"content","content":"我帮您搜索一下"}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | 事件类型，固定为"content" |
| content | string | 文本内容片段 |

##### 3.2 工具调用事件 (tool_call)

AI决定调用某个工具。

```json
data: {"type":"tool_call","data":{"id":"call_123","name":"search_posts","arguments":"{\"query\":\"黑色雨伞\",\"limit\":5}"}}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | 事件类型，固定为"tool_call" |
| data.id | string | 工具调用ID |
| data.name | string | 工具名称 |
| data.arguments | string | 工具参数（JSON字符串） |

##### 3.3 工具结果事件 (tool_result)

工具执行完成并返回结果。

```json
data: {"type":"tool_result","data":{"tool_call_id":"call_123","tool_name":"search_posts","result":"找到2条相关记录..."}}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | 事件类型，固定为"tool_result" |
| data.tool_call_id | string | 对应的工具调用ID |
| data.tool_name | string | 工具名称 |
| data.result | string | 工具执行结果 |

##### 3.4 结束事件

流式输出结束。

```
data: [DONE]
```

### 前端处理示例

```javascript
async function sendMessage(sessionId, message, images = []) {
  const response = await fetch('/api/agent/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      session_id: sessionId,
      message: message,
      images: images
    })
  });

  const reader = response.body.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    const text = decoder.decode(value);
    const lines = text.split('\n');

    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = line.slice(6);
        
        if (data === '[DONE]') {
          console.log('Stream finished');
          return;
        }

        const event = JSON.parse(data);
        
        switch (event.type) {
          case 'content':
            appendContent(event.content);
            break;
          case 'tool_call':
            showToolCall(event.data.name, event.data.arguments);
            break;
          case 'tool_result':
            showToolResult(event.data.tool_name, event.data.result);
            break;
        }
      }
    }
  }
}
```

---

## 4. 获取对话历史

获取指定会话的聊天记录。

### 请求

- **方法**: GET
- **路径**: `/history?session_id={session_id}`

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| session_id | string | 是 | 会话ID（Query参数） |

### 响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "messages": [
      {
        "role": "user",
        "content": "我丢了一把黑色雨伞",
        "images": ["https://example.com/umbrella.jpg"],
        "created_at": "2024-01-15 10:30:00"
      },
      {
        "role": "assistant",
        "content": "我理解您丢失了一把黑色雨伞。请问您是在哪个校区丢失的？",
        "images": null,
        "created_at": "2024-01-15 10:30:05"
      }
    ]
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| messages | array | 消息列表 |
| messages[].role | string | 角色：user(用户) 或 assistant(AI) |
| messages[].content | string | 消息内容 |
| messages[].images | []string | 图片URL列表（仅用户消息可能有） |
| messages[].created_at | string | 创建时间 |

---

## 工具列表

AI助手可调用的工具：

| 工具名称 | 说明 |
|---------|------|
| get_post_detail | 获取发布详情 |
| search_posts | 搜索失物/招领信息 |
| get_my_posts | 获取我的发布列表 |
| get_my_claims | 获取我的认领申请 |
| get_my_feedbacks | 获取我的投诉反馈 |
| publish_post | 发布失物/招领信息 |
| apply_claim | 申请认领物品 |
| cancel_claim | 取消认领申请 |
| review_claim | 审核认领申请 |
| submit_feedback | 提交投诉反馈 |
| cancel_post | 取消发布 |

---

## 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 10001 | 参数无效 |
| 10002 | 未登录 |
| 30108 | AI助手功能已禁用 |
| 50001 | 服务器内部错误 |

---

## 配置说明

后端需要配置以下内容才能使用Agent功能：

```yaml
agent:
  enable: true
  llm:
    base_url: "https://api.openai.com/v1"
    api_key: "your-api-key"
    model: "gpt-4"
  vision_llm:
    base_url: "https://api.openai.com/v1"
    api_key: "your-api-key"
    model: "gpt-4-vision-preview"
  embedding:
    base_url: "https://api.openai.com/v1"
    api_key: "your-api-key"
    model: "text-embedding-3-small"
    dimension: 1536
  milvus:
    address: "localhost:19530"
    collection: "lost_and_found"
```

| 配置项 | 说明 |
|--------|------|
| enable | 是否启用Agent功能 |
| llm | 主LLM配置，用于对话和工具调用 |
| vision_llm | 视觉LLM配置，用于图片内容识别 |
| embedding | 向量化模型配置，用于语义搜索 |
| milvus | 向量数据库配置 |

---

## 注意事项

1. **图片支持**: 用户消息可以包含多张图片URL，系统会使用VisionLLM识别图片内容
2. **工具调用**: 在流式输出中，前端应展示工具调用过程，让用户了解AI正在执行的操作
3. **会话管理**: 会话数据存储在内存中，服务重启后会丢失
4. **功能开关**: 如果后端禁用Agent功能，所有agent接口将返回错误码30108
