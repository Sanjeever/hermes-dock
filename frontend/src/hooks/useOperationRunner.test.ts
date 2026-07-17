import {act, renderHook} from '@testing-library/react';
import {describe, expect, it, vi} from 'vitest';
import {useOperationRunner} from './useOperationRunner';

describe('useOperationRunner', () => {
    it('runs refresh hooks and marks rebuild after a successful action', async () => {
        const calls: string[] = [];
        const setBusy = vi.fn();
        const setNotice = vi.fn();
        const setLastOperationError = vi.fn();
        const setNeedsRebuild = vi.fn();
        const {result} = renderHook(() => useOperationRunner({
            refresh: async () => '',
            appendLog: vi.fn(),
            setBusy,
            setNotice,
            setLastOperationError,
            setNeedsRebuild,
        }));

        let ok = false;
        await act(async () => {
            ok = await result.current('正在保存', async () => { calls.push('action'); }, {
                rebuildRequired: true,
                beforeRefresh: () => calls.push('before-refresh'),
                afterSuccess: () => calls.push('after-success'),
            });
        });

        expect(ok).toBe(true);
        expect(calls).toEqual(['action', 'before-refresh', 'after-success']);
        expect(setNeedsRebuild).toHaveBeenCalledWith(true);
        expect(setBusy).toHaveBeenLastCalledWith('');
        expect(setNotice).toHaveBeenLastCalledWith({type: 'ok', message: '已保存'});
    });

	it('reports action failures and refreshes real state', async () => {
        const refresh = vi.fn(async () => '');
        const appendLog = vi.fn();
        const setNotice = vi.fn();
        const {result} = renderHook(() => useOperationRunner({
            refresh,
            appendLog,
            setBusy: vi.fn(),
            setNotice,
            setLastOperationError: vi.fn(),
            setNeedsRebuild: vi.fn(),
        }));

        await act(async () => {
            expect(await result.current('正在保存', async () => { throw new Error('failed'); })).toBe(false);
        });
		expect(refresh).toHaveBeenCalledTimes(1);
        expect(appendLog).toHaveBeenCalledWith('Error: failed');
        expect(setNotice).toHaveBeenLastCalledWith({type: 'error', message: 'Error: failed'});
    });
});
