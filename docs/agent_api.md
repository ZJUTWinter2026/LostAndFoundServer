# Agent API 快速接入文档

本文档是前端快速接入版，字段与示例已对齐当前后端真实行为。

## 基础约定

- 基础路径：`/api/agent`
- 鉴权：登录态 Cookie/Session
- 非流式接口统一返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

- `code=0` 表示成功，其他表示失败
- Agent 功能开关关闭时返回 `30108`

## 一次完整对话最少流程

1. `POST /api/agent/session` 创建会话，拿到 `session_id`
2. `POST /api/agent/stream` 发送消息，流式接收文本片段
3. 收到 `data: [DONE]` 后结束本轮
4. `GET /api/agent/history` 拉取完整历史用于回显

## 1. 创建会话

- 方法：`POST`
- 路径：`/api/agent/session`
- Body：

```json
{
  "title": "可选标题"
}
```

- 成功响应：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

## 2. 会话列表

- 方法：`GET`
- 路径：`/api/agent/sessions`
- 成功响应：

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

## 3. 流式对话（SSE）

- 方法：`POST`
- 路径：`/api/agent/stream`
- `Content-Type: application/json`
- 请求体：

```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "帮我搜索一下有没有人捡到黑色雨伞",
  "images": ["https://example.com/image1.jpg"]
}
```

### SSE 输出格式

服务端按 `data: ...\n\n` 推送，结束时发送：

```text
data: [DONE]
```

文本事件示例：

```json
data: {"event_id":"550e8400-e29b-41d4-a716-446655440000-1","seq":1,"ts":1741248000123,"content":"我帮您搜索一下"}
```

### 事件字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `event_id` | `string` | 事件标识，格式 `session_id-seq` |
| `seq` | `number` | 当前会话内递增序号（从 1 开始） |
| `ts` | `number` | 服务端时间戳（毫秒） |
| `content` | `string` | 助手文本增量 |

说明：
- 现在不会返回 `type` 字段
- 不会返回工具调用相关字段

### 错误语义

- 如果连接建立前失败（如未登录、参数错误、会话处理中），返回普通 JSON 错误
- 如果连接建立后中断，可能收不到 `[DONE]`，前端应将断流视为异常结束

## 4. 对话历史

- 方法：`GET`
- 路径：`/api/agent/history?session_id={session_id}`
- 成功响应：

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
        "content": "我帮您检索到了2条相关记录，请提供物品特征我再帮您进一步筛选。",
        "created_at": "2024-01-15 10:30:05"
      }
    ]
  }
}
```

### 历史消息字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `messages[].role` | `"user" \| "assistant"` | 角色 |
| `messages[].content` | `string` | 文本内容 |
| `messages[].images` | `string[]` | 仅用户消息可能带图 |
| `messages[].created_at` | `string` | 格式 `yyyy-MM-dd HH:mm:ss` |

说明：
- 历史只返回用户可见文本
- 不返回 `role=tool` 消息
- 不返回工具调用明细

## 5. 最小 TypeScript 接入示例

```ts
type ApiResp<T> = { code: number; message: string; data: T };

export type AgentSession = {
  session_id: string;
  title: string;
  created_at: string;
  updated_at: string;
};

export type AgentHistoryMessage = {
  role: "user" | "assistant";
  content: string;
  images?: string[];
  created_at: string;
};

export type AgentStreamEvent = {
  event_id: string;
  seq: number;
  ts: number;
  content?: string;
};

async function requestJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(url, {
    credentials: "include",
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {})
    }
  });

  const body = (await resp.json()) as ApiResp<T>;
  if (!resp.ok || body.code !== 0) {
    throw new Error(`Agent API error: code=${body.code}, message=${body.message}`);
  }
  return body.data;
}

export function createSession(title = "") {
  return requestJSON<{ session_id: string }>("/api/agent/session", {
    method: "POST",
    body: JSON.stringify({ title })
  });
}

export function getHistory(sessionId: string) {
  const q = new URLSearchParams({ session_id: sessionId });
  return requestJSON<{ messages: AgentHistoryMessage[] }>(`/api/agent/history?${q.toString()}`);
}

export async function streamChat(
  payload: { session_id: string; message: string; images?: string[] },
  onChunk: (text: string, event: AgentStreamEvent) => void,
  onDone: () => void,
  onError: (err: Error) => void
) {
  try {
    const resp = await fetch("/api/agent/stream", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload)
    });

    const ct = resp.headers.get("content-type") || "";
    if (!resp.ok || !ct.includes("text/event-stream")) {
      throw new Error(`Stream init failed: ${resp.status} ${await resp.text()}`);
    }
    if (!resp.body) throw new Error("Stream body is empty");

    const reader = resp.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const frames = buffer.split("\n\n");
      buffer = frames.pop() || "";

      for (const frame of frames) {
        const lines = frame.split("\n");
        for (const line of lines) {
          if (!line.startsWith("data: ")) continue;
          const raw = line.slice(6).trim();

          if (raw === "[DONE]") {
            onDone();
            return;
          }

          try {
            const event = JSON.parse(raw) as AgentStreamEvent;
            onChunk(event.content || "", event);
          } catch {
            // Ignore malformed SSE chunk
          }
        }
      }
    }

    onDone();
  } catch (err) {
    onError(err as Error);
  }
}
```

## 错误码

| 错误码 | 含义 |
|--------|------|
| `0` | 成功 |
| `10001` | 参数无效 |
| `10002` | 未登录 |
| `30108` | AI 助手功能已禁用 |
| `30109` | 会话正在处理中 |
| `50001` | 服务端内部错误 |
