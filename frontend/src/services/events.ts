import {EventsOn as WailsEventsOn} from '../../wailsjs/runtime/runtime';
import {isWebRuntime, webClientID} from './api';

type Handler = (payload: unknown) => void;

const handlers: Record<string, Set<Handler>> = {};
let socket: WebSocket | null = null;
let reconnectTimer = 0;

export function EventsOn<T>(event: string, callback: (payload: T) => void) {
    if (!isWebRuntime()) {
        return WailsEventsOn(event, callback);
    }
    if (!handlers[event]) handlers[event] = new Set();
    handlers[event].add(callback as Handler);
    ensureSocket();
    let active = true;
    return () => {
        if (!active) return;
        active = false;
        const eventHandlers = handlers[event];
        eventHandlers?.delete(callback as Handler);
        if (eventHandlers?.size === 0) delete handlers[event];
        if (!hasSubscriptions()) disconnectEvents();
    };
}

export function disconnectEvents() {
    window.clearTimeout(reconnectTimer);
    reconnectTimer = 0;
    const current = socket;
    socket = null;
    if (!current) return;
    current.onmessage = null;
    current.onclose = null;
    try {
        current.close();
    } catch {
        // CONNECTING 状态的浏览器实现可能拒绝 close；回调已解除，不会再重连。
    }
}

function hasSubscriptions() {
    return Object.values(handlers).some((eventHandlers) => eventHandlers.size > 0);
}

function ensureSocket() {
    if (!hasSubscriptions()) return;
    if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) return;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const nextSocket = new WebSocket(`${protocol}//${window.location.host}/ws/events?client_id=${encodeURIComponent(webClientID)}`);
    socket = nextSocket;
    nextSocket.onmessage = (message) => {
        let event: unknown;
        try {
            event = JSON.parse(String(message.data));
        } catch {
            console.error('收到无法解析的 Web 事件');
            return;
        }
        if (!event || typeof event !== 'object' || typeof (event as {event?: unknown}).event !== 'string') {
            console.error('收到格式错误的 Web 事件');
            return;
        }
        const parsed = event as {event: string; payload: unknown};
        handlers[parsed.event]?.forEach((handler) => {
            try {
                handler(parsed.payload);
            } catch {
                console.error('处理 Web 事件失败');
            }
        });
    };
    nextSocket.onclose = () => {
        if (socket === nextSocket) socket = null;
        window.clearTimeout(reconnectTimer);
        reconnectTimer = 0;
        if (hasSubscriptions()) reconnectTimer = window.setTimeout(ensureSocket, 1500);
    };
}
