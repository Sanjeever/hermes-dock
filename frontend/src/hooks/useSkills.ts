import {useState} from 'react';
import {DeleteSkill, GetSkillDetail, GetSkillHubDetail, InstallSkillHubSkill, ListProfileSkills, ListSkillHubSkills, OpenSkillDirectory, RestoreDefaultSkills, SyncBundledSkills} from '../services/api';
import type {Notice, SkillDetail, SkillHubDetail, SkillHubQuery, SkillHubState, SkillsState} from '../types';
import {restoreDefaultSkillsMessage, syncBundledSkillsMessage} from '../appPolicies';

type Run = (label: string, action: () => Promise<unknown>, options?: {rebuildRequired?: boolean}) => Promise<boolean>;

export function useSkills(options: {
    run: Run;
    refresh: () => Promise<string>;
    appendLog: (line: string) => void;
    setBusy: (value: string) => void;
    setNotice: (value: Notice | null) => void;
    setLastOperationError: (value: string) => void;
    setNeedsRebuild: (value: boolean) => void;
}) {
    const [skillsState, setSkillsState] = useState<SkillsState | null>(null);
    const [skillDetail, setSkillDetail] = useState<SkillDetail | null>(null);
    const [skillsStatus, setSkillsStatus] = useState('');
    const [skillHubState, setSkillHubState] = useState<SkillHubState | null>(null);
    const [skillHubDetail, setSkillHubDetail] = useState<SkillHubDetail | null>(null);
    const [skillHubStatus, setSkillHubStatus] = useState('');

    async function loadSkills() {
        setSkillsStatus('正在读取技能');
        try {
            setSkillsState(await ListProfileSkills() as SkillsState);
            setSkillsStatus('');
        } catch (error) {
            const message = String(error);
            setSkillsStatus(message);
            options.appendLog(message);
        }
    }

    async function loadSkillDetail(path: string) {
        setSkillsStatus('正在读取技能详情');
        try {
            setSkillDetail(await GetSkillDetail(path) as SkillDetail);
            setSkillsStatus('');
        } catch (error) {
            const message = String(error);
            setSkillDetail(null);
            setSkillsStatus(message);
            options.appendLog(message);
        }
    }

    async function deleteSkill(path: string) {
        const ok = await options.run('正在删除技能', () => DeleteSkill(path), {rebuildRequired: true});
        if (!ok) return false;
        setSkillDetail(null);
        await loadSkills();
        options.setNotice({type: 'ok', message: '已删除技能并创建备份，重建后生效'});
        return true;
    }

    async function updateBundledSkills(restore: boolean) {
        const label = restore ? '正在恢复默认技能' : '正在同步内置技能';
        options.setBusy(label);
        options.setNotice({type: 'info', message: label});
        setSkillsStatus(label);
        options.setLastOperationError('');
        try {
            const result = restore ? await RestoreDefaultSkills() : await SyncBundledSkills();
            await options.refresh();
            await loadSkills();
            if (restore) setSkillDetail(null);
            const summary = restore ? restoreDefaultSkillsMessage(result) : syncBundledSkillsMessage(result);
            if (result.syncedFiles > 0) options.setNeedsRebuild(true);
            setSkillsStatus(summary);
            options.setNotice({type: 'ok', message: summary});
            return true;
        } catch (error) {
            const message = String(error);
            options.appendLog(message);
            setSkillsStatus(message);
            options.setNotice({type: 'error', message});
            options.setLastOperationError(message);
            return false;
        } finally {
            options.setBusy('');
        }
    }

    async function openSkillDirectory(path: string) {
        try {
            await OpenSkillDirectory(path);
        } catch (error) {
            const message = String(error);
            options.appendLog(message);
            setSkillsStatus(message);
            options.setNotice({type: 'error', message});
        }
    }

    async function loadSkillHubSkills(query: SkillHubQuery) {
        setSkillHubStatus('正在读取技能中心');
        try {
            setSkillHubState(await ListSkillHubSkills(query) as SkillHubState);
            setSkillHubStatus('');
        } catch (error) {
            const message = String(error);
            setSkillHubStatus(message);
            options.appendLog(message);
        }
    }

    async function loadSkillHubDetail(slug: string) {
        setSkillHubStatus('正在读取技能详情');
        try {
            setSkillHubDetail(await GetSkillHubDetail(slug) as SkillHubDetail);
            setSkillHubStatus('');
        } catch (error) {
            const message = String(error);
            setSkillHubDetail(null);
            setSkillHubStatus(message);
            options.appendLog(message);
        }
    }

    async function installSkillHubSkill(slug: string) {
        const ok = await options.run('正在安装技能', () => InstallSkillHubSkill(slug), {rebuildRequired: true});
        if (!ok) return false;
        await loadSkills();
        await loadSkillHubDetail(slug);
        return true;
    }

    function resetForProfile() {
        setSkillDetail(null);
        setSkillHubDetail(null);
        setSkillHubState(null);
        setSkillHubStatus('');
    }

    return {
        skillsState, skillDetail, skillsStatus, skillHubState, skillHubDetail, skillHubStatus,
        loadSkills, loadSkillDetail, deleteSkill,
        syncBundledSkills: () => updateBundledSkills(false),
        restoreDefaultSkills: () => updateBundledSkills(true),
        openSkillDirectory, loadSkillHubSkills, loadSkillHubDetail, installSkillHubSkill, resetForProfile,
    };
}
