import { fetchApi, getAuthHeaders } from './api';

const DEFAULT_TIMEOUT_MS = 10 * 60 * 1000;

export function waitForRewriteTask(taskId, { timeoutMs = DEFAULT_TIMEOUT_MS } = {}) {
  if (!taskId) {
    return Promise.reject(new Error('缺少任务ID'));
  }

  return new Promise((resolve, reject) => {
    const controller = new AbortController();
    const timeout = window.setTimeout(() => {
      controller.abort();
      reject(new Error('图像生成超时，请稍后在历史记录中查看结果'));
    }, timeoutMs);

    const cleanup = () => {
      window.clearTimeout(timeout);
      controller.abort();
    };

    subscribeRewriteTask(taskId, { signal: controller.signal })
      .then((task) => {
        cleanup();
        resolveTaskResult(task, resolve, reject);
      })
      .catch((err) => {
        cleanup();
        if (err.name === 'AbortError') return;
        pollRewriteTask(taskId, { timeoutMs }).then(resolve, reject);
      });
  });
}

async function subscribeRewriteTask(taskId, { signal }) {
  const authHeaders = await getAuthHeaders();
  const response = await fetch(`/api/v1/images/rewrite/tasks/${encodeURIComponent(taskId)}/events`, {
    headers: {
      Accept: 'text/event-stream',
      ...authHeaders,
    },
    credentials: 'include',
    signal,
  });

  if (!response.ok || !response.body) {
    throw new Error('订阅生成任务失败');
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });

    let boundary = buffer.indexOf('\n\n');
    while (boundary >= 0) {
      const rawEvent = buffer.slice(0, boundary);
      buffer = buffer.slice(boundary + 2);
      const event = parseSSEEvent(rawEvent);
      if (event.event === 'complete' && event.data) {
        return JSON.parse(event.data);
      }
      boundary = buffer.indexOf('\n\n');
    }
  }

  throw new Error('订阅连接已关闭');
}

function parseSSEEvent(rawEvent) {
  return rawEvent.split('\n').reduce((event, line) => {
    if (line.startsWith('event:')) {
      event.event = line.slice(6).trim();
    } else if (line.startsWith('data:')) {
      event.data = `${event.data || ''}${line.slice(5).trim()}`;
    }
    return event;
  }, {});
}

async function pollRewriteTask(taskId, { timeoutMs }) {
  const startedAt = Date.now();
  while (Date.now() - startedAt < timeoutMs) {
    const task = await fetchApi(`/api/v1/images/rewrite/tasks/${encodeURIComponent(taskId)}`);
    if (task.status === 'succeeded' || task.status === 'failed') {
      return new Promise((resolve, reject) => resolveTaskResult(task, resolve, reject));
    }
    await delay(2000);
  }
  throw new Error('图像生成超时，请稍后在历史记录中查看结果');
}

function resolveTaskResult(task, resolve, reject) {
  if (task.status === 'failed') {
    reject(new Error(task.error || '图像生成失败'));
    return;
  }
  if (task.response?.image_url) {
    resolve(task.response);
    return;
  }
  reject(new Error('图像生成未返回结果'));
}

function delay(ms) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

export async function resolveRewriteResponse(data) {
  if (data?.image_url) {
    return data;
  }
  if (data?.task_id) {
    return waitForRewriteTask(data.task_id);
  }
  return data;
}
