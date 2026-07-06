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
    return () => {
        handlers[event]?.delete(callback as Handler);
    };
}

function ensureSocket() {
    if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) return;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    socket = new WebSocket(`${protocol}//${window.location.host}/ws/events?client_id=${encodeURIComponent(webClientID)}`);
    socket.onmessage = (message) => {
        const event = JSON.parse(message.data) as { event: string; payload: unknown };
        handlers[event.event]?.forEach((handler) => handler(event.payload));
    };
    socket.onclose = () => {
        socket = null;
        window.clearTimeout(reconnectTimer);
        reconnectTimer = window.setTimeout(ensureSocket, 1500);
    };
}
