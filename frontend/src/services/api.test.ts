import {afterEach, describe, expect, it, vi} from 'vitest';
import {cancelWebRequests, getWebSession, loginWeb} from './api';

describe('Web API requests', () => {
    afterEach(() => {
        vi.useRealTimers();
        vi.restoreAllMocks();
        vi.unstubAllGlobals();
    });

    it('reports a timeout when the server does not respond', async () => {
        vi.useFakeTimers();
        vi.stubGlobal('fetch', vi.fn((_input: RequestInfo | URL, init?: RequestInit) => new Promise<Response>((_resolve, reject) => {
            init?.signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')));
        })));

        const request = getWebSession();
        const rejection = expect(request).rejects.toThrow('请求超时，请稍后重试');
        await vi.advanceTimersByTimeAsync(15_000);

        await rejection;
    });

    it('distinguishes an invalid JSON response', async () => {
        vi.stubGlobal('fetch', vi.fn(async () => new Response('<html>bad gateway</html>', {
            status: 200,
            headers: {'Content-Type': 'text/html'},
        })));

        await expect(loginWeb('secret')).rejects.toThrow('服务器返回的数据格式错误');
    });

    it('cancels pending requests when the Web UI unmounts', async () => {
        vi.stubGlobal('fetch', vi.fn((_input: RequestInfo | URL, init?: RequestInit) => new Promise<Response>((_resolve, reject) => {
            init?.signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')));
        })));

        const request = getWebSession();
        const rejection = expect(request).rejects.toThrow('请求已取消');
        cancelWebRequests();

        await rejection;
    });
});
