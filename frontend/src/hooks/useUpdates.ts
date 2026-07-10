import {useEffect, useState} from 'react';
import {CheckForUpdates, DismissUpdate, OpenUpdateURL} from '../services/api';
import type {Notice, UpdateInfo} from '../types';

export function useUpdates(options: {
    appVersion?: string;
    appendLog: (line: string) => void;
    setNotice: (value: Notice | null) => void;
}) {
    const [info, setInfo] = useState<UpdateInfo | null>(null);
    const [busy, setBusy] = useState(false);

    useEffect(() => {
        if (!options.appVersion) return;
        check(false);
    }, [options.appVersion]);

    async function check(force: boolean) {
        setBusy(true);
        try {
            const next = await CheckForUpdates(force);
            setInfo(next);
            if (!force) return;
            if (next.available && !next.dismissed) options.setNotice({type: 'ok', message: `发现新版本 v${next.latestVersion}`});
            else if (next.available) options.setNotice({type: 'info', message: `v${next.latestVersion} 已忽略`});
            else options.setNotice({type: 'ok', message: '当前已是最新版本'});
        } catch (error) {
            if (!force) return;
            const message = String(error);
            options.appendLog(message);
            options.setNotice({type: 'error', message});
        } finally {
            setBusy(false);
        }
    }

    async function dismiss() {
        if (!info?.latestVersion) return;
        try {
            await DismissUpdate(info.latestVersion);
            setInfo({...info, dismissed: true});
            options.setNotice({type: 'ok', message: `已忽略 v${info.latestVersion}`});
        } catch (error) {
            const message = String(error);
            options.appendLog(message);
            options.setNotice({type: 'error', message});
        }
    }

    async function open(url: string) {
        if (!url) return;
        try {
            await OpenUpdateURL(url);
        } catch (error) {
            const message = String(error);
            options.appendLog(message);
            options.setNotice({type: 'error', message});
        }
    }

    async function copyInstallCommand(url: string) {
        if (!info?.assetName || !url) return;
        const command = `curl -L -o ${info.assetName} ${url}\nsudo apt install -y ./${info.assetName}`;
        try {
            await navigator.clipboard.writeText(command);
            options.setNotice({type: 'ok', message: '已复制安装命令'});
        } catch (error) {
            const message = String(error);
            options.appendLog(message);
            options.setNotice({type: 'error', message});
        }
    }

    return {info, busy, check, dismiss, open, copyInstallCommand};
}
