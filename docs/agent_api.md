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

---

## 2. 获取会话列表

获取当前用户的所有对话会话，按最后更新时间倒序排列。

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
  "images": ["https://example.com/image1.jpg"]
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| session_id | string | 是 | 会话ID |
| message | string | 是 | 用户消息内容 |
| images | []string | 否 | 图片URL列表，支持多图 |

### 响应

响应为SSE流，Content-Type为`text/event-stream`。

#### 事件类型

**内容事件 (content)** - AI生成的文本内容片段

```json
data: {"type":"content","content":"我帮您搜索一下"}
```

**工具调用事件 (tool_call)** - AI调用工具

```json
data: {"type":"tool_call","data":{"id":"call_123","name":"search_posts","arguments":"{\"query\":\"黑色雨伞\"}"}}
```

**工具结果事件 (tool_result)** - 工具执行结果

```json
data: {"type":"tool_result","data":{"tool_call_id":"call_123","tool_name":"search_posts","result":"找到2条相关记录..."}}
```

**结束事件**

```
data: [DONE]
```

### 前端处理示例

```javascript
async function sendMessage(sessionId, message, images = []) {
  const response = await fetch('/api/agent/stream', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ session_id: sessionId, message, images })
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
        if (data === '[DONE]') return;

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
        "created_at": "2024-01-15 10:30:05"
      }
    ]
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| messages[].role | string | 角色：user 或 assistant |
| messages[].content | string | 消息内容 |
| messages[].images | []string | 图片URL列表（仅用户消息） |
| messages[].created_at | string | 创建时间 |

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

## 注意事项

1. **图片支持**: 用户消息可以包含多张图片URL，系统会自动识别图片内容
2. **工具调用展示**: 建议前端展示工具调用过程，让用户了解AI正在执行的操作
3. **数据持久化**: 会话数据存储在数据库中，服务重启后历史对话不丢失
