import {useEffect, useState} from 'react';
import {CheckForUpdates, DismissUpdate, InstallUpdate, OpenUpdateURL, SetAutoUpdateEnabled} from '../services/api';
import {EventsOn} from '../services/events';
import type {Notice, UpdateInfo, UpdateStatus} from '../types';

export function useUpdates(options: {
    appVersion?: string;
    appendLog: (line: string) => void;
    setNotice: (value: Notice | null) => void;
    onStatusChanged: (status: UpdateStatus) => void;
}) {
    const [info, setInfo] = useState<UpdateInfo | null>(null);
    const [busy, setBusy] = useState(false);
    const [progress, setProgress] = useState('');

    useEffect(() => EventsOn<{message: string; percent: number}>('update:progress', (event) => {
        const suffix = event.percent > 0 && event.percent < 100 ? ` ${event.percent}%` : '';
        setProgress(`${event.message}${suffix}`);
    }), []);

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

    async function install() {
        if (!info?.available || !info.assetUrl) return;
        setBusy(true);
        setProgress('正在准备更新');
        try {
            await InstallUpdate(info.latestVersion);
            setProgress('正在重启企智盒');
            options.setNotice({type: 'ok', message: '更新已下载并验证，正在重启企智盒'});
        } catch (error) {
            const message = String(error);
            options.appendLog(message);
            options.setNotice({type: 'error', message});
            setProgress('');
            setBusy(false);
        }
    }

    async function setAutoUpdate(enabled: boolean) {
        setBusy(true);
        try {
            const status = await SetAutoUpdateEnabled(enabled);
            options.onStatusChanged(status);
            options.setNotice({type: 'ok', message: enabled ? '已开启静默自动升级' : '已关闭静默自动升级'});
        } catch (error) {
            const message = String(error);
            options.appendLog(message);
            options.setNotice({type: 'error', message});
        } finally {
            setBusy(false);
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

    return {info, busy, progress, check, dismiss, install, open, setAutoUpdate};
}
