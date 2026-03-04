# Agent API 接口文档

本文档面向前端开发，目标是让你不看后端代码也能完整接入当前 Agent 对话能力。

## 基础约定

- 基础路径：`/api/agent`
- 鉴权方式：登录态（依赖服务端 Session/Cookie）
- 非流式接口响应格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

- `code=0` 表示成功，其他 code 视为失败
- Agent 功能受配置开关控制，关闭时返回 `30108`

## 目录

1. 创建会话 `POST /session`
2. 会话列表 `GET /sessions`
3. 流式对话 `POST /stream` (SSE)
4. 对话历史 `GET /history`
5. 前端完整参考代码（TypeScript）

---

## 1. 创建会话

### 请求

- 方法：`POST`
- 路径：`/api/agent/session`
- `Content-Type: application/json`

```json
{
  "title": "会话标题（可选）"
}
```

### 字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `title` | `string` | 否 | 会话标题。可不传，后端会在首轮对话后自动生成 |

### 成功响应

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

### 请求

- 方法：`GET`
- 路径：`/api/agent/sessions`

### 成功响应

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

### 前端建议

- 按 `updated_at` 做列表排序展示（后端已降序返回）
- 首次进入页面先拉会话列表，若为空可自动创建会话

---

## 3. 发送消息（流式）

该接口是 SSE 响应，不是标准 JSON 一次性返回。

### 请求

- 方法：`POST`
- 路径：`/api/agent/stream`
- `Content-Type: application/json`

```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "帮我搜索一下有没有人捡到黑色雨伞",
  "images": ["https://example.com/image1.jpg"]
}
```

### 字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `session_id` | `string` | 是 | 会话ID |
| `message` | `string` | 是 | 用户文本输入 |
| `images` | `string[]` | 否 | 图片 URL 列表 |

### SSE 事件格式

服务端按 `data: ...\n\n` 推送事件，最终以 `data: [DONE]` 结束。

#### 3.1 `content`

```json
data: {"event_id":"550e8400-e29b-41d4-a716-446655440000-1","seq":1,"ts":1741248000123,"type":"content","content":"我帮您搜索一下"}
```

#### 3.2 `tool_call`

```json
data: {"event_id":"550e8400-e29b-41d4-a716-446655440000-2","seq":2,"ts":1741248000456,"type":"tool_call","data":{"id":"call_123","name":"search_posts","arguments":"{\"query\":\"黑色雨伞\"}"}}
```

#### 3.3 `tool_result`

```json
data: {"event_id":"550e8400-e29b-41d4-a716-446655440000-3","seq":3,"ts":1741248000789,"type":"tool_result","data":{"tool_call_id":"call_123","tool_name":"search_posts","result":"找到2条相关记录..."}}
```

#### 3.4 结束标记

```text
data: [DONE]
```

### 事件字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `event_id` | `string` | 事件唯一标识，当前格式为 `session_id-seq` |
| `seq` | `number` | 会话内递增序号（从 1 开始） |
| `ts` | `number` | 服务端事件时间戳（Unix 毫秒） |
| `type` | `"content" \| "tool_call" \| "tool_result"` | 事件类型 |
| `content` | `string` | 当 `type=content` 时存在 |
| `data` | `object` | 当 `type=tool_call/tool_result` 时存在 |

### 流式接口错误语义

- 如果请求在“进入 SSE 前”失败（如未登录、参数错误、会话处理中），返回普通 JSON 错误，不会进入流
- 如果已进入 SSE 过程中发生异常，连接可能提前结束，前端应处理“未收到 `[DONE]`”的场景

---

## 4. 获取对话历史

### 请求

- 方法：`GET`
- 路径：`/api/agent/history?session_id={session_id}`

### 成功响应

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
        "content": "",
        "tool_calls": [
          {
            "id": "call_123",
            "name": "search_posts",
            "arguments": "{\"query\":\"黑色雨伞\"}"
          }
        ],
        "created_at": "2024-01-15 10:30:05"
      },
      {
        "role": "tool",
        "content": "找到2条相关记录...",
        "tool_result": {
          "tool_call_id": "call_123",
          "tool_name": "search_posts",
          "result": "找到2条相关记录..."
        },
        "created_at": "2024-01-15 10:30:05"
      }
    ]
  }
}
```

### 历史消息字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `messages[].role` | `"user" \| "assistant" \| "tool"` | 消息角色 |
| `messages[].content` | `string` | 原始消息内容 |
| `messages[].images` | `string[]` | 用户消息中的图片 URL（仅 user 常见） |
| `messages[].tool_calls` | `{id,name,arguments}[]` | assistant 的工具调用计划 |
| `messages[].tool_result` | `{tool_call_id,tool_name,result}` | tool 消息的执行结果 |
| `messages[].created_at` | `string` | 时间字符串，格式 `yyyy-MM-dd HH:mm:ss` |

### 前端渲染建议

- `role=user`：按普通用户气泡渲染
- `role=assistant` 且 `tool_calls` 非空：渲染“工具调用卡片”，可展开 `arguments`
- `role=assistant` 且 `content` 非空：渲染助手文本
- `role=tool`：渲染工具结果卡片，优先显示 `tool_result.tool_name` 和 `tool_result.result`

---

## 错误码

| 错误码 | 说明 |
|--------|------|
| `0` | 成功 |
| `10001` | 参数无效 |
| `10002` | 未登录 |
| `30108` | AI 助手功能已禁用 |
| `30109` | 会话正在处理中，请稍后再试 |
| `50001` | 服务端内部错误 |

---

## 5. 前端完整参考代码（TypeScript）

下面示例可直接作为前端 SDK 雏形，包含：

- 基础请求封装
- 会话管理
- 流式 SSE 解析
- 历史与流式统一消息模型

```ts
// agent-api.ts

type ApiResp<T> = {
  code: number;
  message: string;
  data: T;
};

export type SessionInfo = {
  session_id: string;
  title: string;
  created_at: string;
  updated_at: string;
};

export type ToolCall = {
  id: string;
  name: string;
  arguments: string;
};

export type ToolResult = {
  tool_call_id: string;
  tool_name: string;
  result: string;
};

export type HistoryMessage = {
  role: "user" | "assistant" | "tool";
  content: string;
  images?: string[];
  tool_calls?: ToolCall[];
  tool_result?: ToolResult;
  created_at: string;
};

export type StreamEvent = {
  event_id: string;
  seq: number;
  ts: number;
  type: "content" | "tool_call" | "tool_result";
  content?: string;
  data?: any;
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

export async function createSession(title = "") {
  return requestJSON<{ session_id: string }>("/api/agent/session", {
    method: "POST",
    body: JSON.stringify({ title })
  });
}

export async function listSessions() {
  return requestJSON<{ sessions: SessionInfo[] }>("/api/agent/sessions");
}

export async function getHistory(sessionId: string) {
  const q = new URLSearchParams({ session_id: sessionId });
  return requestJSON<{ messages: HistoryMessage[] }>(`/api/agent/history?${q.toString()}`);
}

export async function streamChat(
  payload: { session_id: string; message: string; images?: string[] },
  onEvent: (ev: StreamEvent) => void,
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

    // 进入流之前失败时，后端返回 JSON 错误
    const ct = resp.headers.get("content-type") || "";
    if (!resp.ok || !ct.includes("text/event-stream")) {
      const fallback = await resp.text();
      throw new Error(`Stream init failed: ${resp.status} ${fallback}`);
    }

    if (!resp.body) {
      throw new Error("Stream body is empty");
    }

    const reader = resp.body.getReader();
    const decoder = new TextDecoder();

    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      // 按 SSE 空行分帧
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
            const event = JSON.parse(raw) as StreamEvent;
            onEvent(event);
          } catch (e) {
            console.warn("Skip non-JSON SSE line:", raw, e);
          }
        }
      }
    }

    // 正常读完但没收到 DONE，也视为结束（例如网络中断）
    onDone();
  } catch (e) {
    onError(e as Error);
  }
}
```

### 可直接使用的渲染映射示例

```ts
// message-mapper.ts

import type { HistoryMessage, StreamEvent } from "./agent-api";

export type UINode =
  | { kind: "text"; role: "user" | "assistant"; text: string }
  | { kind: "toolCall"; toolName: string; args: string; eventId?: string; seq?: number }
  | { kind: "toolResult"; toolName: string; result: string; eventId?: string; seq?: number };

export function historyToUINodes(msgs: HistoryMessage[]): UINode[] {
  const nodes: UINode[] = [];
  for (const m of msgs) {
    if (m.role === "assistant" && m.tool_calls?.length) {
      for (const tc of m.tool_calls) {
        nodes.push({ kind: "toolCall", toolName: tc.name, args: tc.arguments });
      }
    }

    if (m.role === "tool" && m.tool_result) {
      nodes.push({
        kind: "toolResult",
        toolName: m.tool_result.tool_name,
        result: m.tool_result.result
      });
      continue;
    }

    if (m.content) {
      const role = m.role === "user" ? "user" : "assistant";
      nodes.push({ kind: "text", role, text: m.content });
    }
  }
  return nodes;
}

export function streamEventToUINode(ev: StreamEvent): UINode | null {
  if (ev.type === "content") {
    return { kind: "text", role: "assistant", text: ev.content || "" };
  }

  if (ev.type === "tool_call" && ev.data) {
    return {
      kind: "toolCall",
      toolName: ev.data.name,
      args: ev.data.arguments,
      eventId: ev.event_id,
      seq: ev.seq
    };
  }

  if (ev.type === "tool_result" && ev.data) {
    return {
      kind: "toolResult",
      toolName: ev.data.tool_name,
      result: ev.data.result,
      eventId: ev.event_id,
      seq: ev.seq
    };
  }

  return null;
}
```

---

## 6. 接入建议与排错

- 建议每次发送前禁用输入框，收到 `[DONE]` 或错误后恢复，避免并发触发 `30109`
- 建议在客户端维护 `lastSeq`，如果收到更小 `seq` 可忽略，防止重复渲染
- 图片 URL 建议前端先校验可访问性，减少模型侧空图描述
- 如果流式请求返回非 `text/event-stream`，优先按 JSON 错误处理
