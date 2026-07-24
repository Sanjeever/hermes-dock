import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';

vi.mock('./api', () => ({
    isWebRuntime: () => true,
    webClientID: 'test-client',
}));

class FakeWebSocket {
    static readonly CONNECTING = 0;
    static readonly OPEN = 1;
    static readonly CLOSED = 3;
    static instances: FakeWebSocket[] = [];

    readyState = FakeWebSocket.CONNECTING;
    onmessage: ((event: MessageEvent) => void) | null = null;
    onclose: (() => void) | null = null;
    close = vi.fn(() => { this.readyState = FakeWebSocket.CLOSED; });

    constructor(readonly url: string) {
        FakeWebSocket.instances.push(this);
    }

    receive(data: string) {
        this.onmessage?.({data} as MessageEvent);
    }
}

describe('Web events', () => {
    beforeEach(() => {
        FakeWebSocket.instances = [];
        vi.stubGlobal('WebSocket', FakeWebSocket);
    });

    afterEach(async () => {
        const {disconnectEvents} = await import('./events');
        disconnectEvents();
        vi.restoreAllMocks();
        vi.unstubAllGlobals();
    });

    it('isolates malformed messages and handler failures', async () => {
        const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
        const {EventsOn} = await import('./events');
        const healthyHandler = vi.fn();
        const offBroken = EventsOn('status', () => { throw new Error('handler failed'); });
        const offHealthy = EventsOn('status', healthyHandler);
        const current = FakeWebSocket.instances[0];

        current.receive('{invalid');
        current.receive(JSON.stringify({event: 'status', payload: {state: 'running'}}));

        expect(healthyHandler).toHaveBeenCalledWith({state: 'running'});
        expect(consoleError).toHaveBeenCalledTimes(2);
        offBroken();
        offHealthy();
    });

    it('closes the socket after the final subscription is removed', async () => {
        const {EventsOn} = await import('./events');
        const offFirst = EventsOn('first', vi.fn());
        const offSecond = EventsOn('second', vi.fn());
        const current = FakeWebSocket.instances[0];

        offFirst();
        expect(current.close).not.toHaveBeenCalled();
        offSecond();
        expect(current.close).toHaveBeenCalledOnce();
    });
});
