import type {Dispatch, SetStateAction} from 'react';
import type {Notice, RunOptions} from '../types';
import {doneLabel} from '../utils';

export function useOperationRunner(options: {
    refresh: () => Promise<string>;
    appendLog: (line: string) => void;
    setBusy: Dispatch<SetStateAction<string>>;
    setNotice: Dispatch<SetStateAction<Notice | null>>;
    setLastOperationError: Dispatch<SetStateAction<string>>;
    setNeedsRebuild: Dispatch<SetStateAction<boolean>>;
}) {
    return async function run(label: string, action: () => Promise<unknown>, runOptions: RunOptions = {}) {
        options.setBusy(label);
        options.setNotice({type: 'info', message: label});
        options.setLastOperationError('');
        try {
            await action();
            runOptions.beforeRefresh?.();
            const refreshMessage = await options.refresh();
            if (runOptions.rebuildRequired) options.setNeedsRebuild(true);
            runOptions.afterSuccess?.();
            if (refreshMessage) {
                const message = `${doneLabel(label)}，但刷新状态失败：${refreshMessage}`;
                options.setNotice({type: 'error', message});
                options.setLastOperationError(message);
                return true;
            }
            options.setNotice({type: 'ok', message: doneLabel(label)});
            return true;
        } catch (error) {
            const message = String(error);
            options.appendLog(message);
            options.setNotice({type: 'error', message});
            options.setLastOperationError(message);
			const refreshMessage = await options.refresh();
			if (refreshMessage) options.appendLog(`操作失败后刷新状态失败：${refreshMessage}`);
            return false;
        } finally {
            options.setBusy('');
        }
    };
}
