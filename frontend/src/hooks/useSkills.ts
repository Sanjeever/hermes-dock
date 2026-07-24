import {useEffect, useRef, useState} from 'react';
import {BatchDeleteSkills, DeleteSkill, GetSkillDetail, GetSkillHubDetail, InstallSkillHubSkill, ListProfileSkills, ListSkillHubSkills, OpenSkillDirectory, RestoreDefaultSkills, SyncBundledSkills} from '../services/api';
import type {LoadState, Notice, SkillDetail, SkillHubDetail, SkillHubQuery, SkillHubState, SkillsState} from '../types';
import {restoreDefaultSkillsMessage, syncBundledSkillsMessage} from '../appPolicies';

type Run = (label: string, action: () => Promise<unknown>, options?: {rebuildRequired?: boolean}) => Promise<boolean>;

export function useSkills(options: {
	getProfileID: () => string;
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
    const [skillsLoadState, setSkillsLoadState] = useState<LoadState>('idle');
    const [skillsLoadError, setSkillsLoadError] = useState('');
    const [skillHubState, setSkillHubState] = useState<SkillHubState | null>(null);
    const [skillHubDetail, setSkillHubDetail] = useState<SkillHubDetail | null>(null);
    const [skillHubStatus, setSkillHubStatus] = useState('');
    const [skillHubLoadState, setSkillHubLoadState] = useState<LoadState>('idle');
    const [skillHubLoadError, setSkillHubLoadError] = useState('');
    const mounted = useRef(true);
    const skillsGeneration = useRef(0);
    const skillDetailGeneration = useRef(0);
    const skillHubGeneration = useRef(0);
    const skillHubDetailGeneration = useRef(0);

    useEffect(() => {
        mounted.current = true;
        return () => {
            mounted.current = false;
            skillsGeneration.current++;
            skillDetailGeneration.current++;
            skillHubGeneration.current++;
            skillHubDetailGeneration.current++;
        };
    }, []);

    async function loadSkills() {
		const profileID = options.getProfileID();
        const generation = ++skillsGeneration.current;
        setSkillsStatus('正在读取技能');
        setSkillsLoadState('loading');
        setSkillsLoadError('');
        try {
            const next = await ListProfileSkills(profileID) as SkillsState;
            if (!mounted.current || generation !== skillsGeneration.current || profileID !== options.getProfileID()) return;
			setSkillsState(next);
            setSkillsStatus('');
            setSkillsLoadState('success');
        } catch (error) {
            if (!mounted.current || generation !== skillsGeneration.current || profileID !== options.getProfileID()) return;
            const message = String(error);
            setSkillsStatus(message);
            setSkillsLoadState('error');
            setSkillsLoadError(message);
            options.appendLog(message);
        }
    }

    async function loadSkillDetail(path: string) {
		const profileID = options.getProfileID();
        const generation = ++skillDetailGeneration.current;
        setSkillsStatus('正在读取技能详情');
        try {
            const next = await GetSkillDetail(profileID, path) as SkillDetail;
            if (!mounted.current || generation !== skillDetailGeneration.current || profileID !== options.getProfileID()) return;
			setSkillDetail(next);
            setSkillsStatus('');
        } catch (error) {
            if (!mounted.current || generation !== skillDetailGeneration.current || profileID !== options.getProfileID()) return;
            const message = String(error);
            setSkillDetail(null);
            setSkillsStatus(message);
            options.appendLog(message);
        }
    }

    async function deleteSkill(path: string) {
		const profileID = options.getProfileID();
		const ok = await options.run('正在删除技能', () => DeleteSkill(profileID, path), {rebuildRequired: true});
        if (!mounted.current || !ok) return false;
        setSkillDetail(null);
        await loadSkills();
        if (!mounted.current) return false;
        options.setNotice({type: 'ok', message: '已删除技能并创建备份，重建后生效'});
        return true;
    }

    async function batchDeleteSkills(paths: string[]) {
		const profileID = options.getProfileID();
		const ok = await options.run(`正在删除 ${paths.length} 个技能`, () => BatchDeleteSkills(profileID, paths), {rebuildRequired: true});
        if (!mounted.current) return false;
        if (!ok) {
            await loadSkills();
            return false;
        }
        setSkillDetail(null);
        await loadSkills();
        if (!mounted.current) return false;
        options.setNotice({type: 'ok', message: `已删除 ${paths.length} 个技能并创建备份，重建后生效`});
        return true;
    }

    async function updateBundledSkills(restore: boolean) {
		const profileID = options.getProfileID();
        const label = restore ? '正在恢复默认技能' : '正在同步内置技能';
        options.setBusy(label);
        options.setNotice({type: 'info', message: label});
        setSkillsStatus(label);
        options.setLastOperationError('');
		try {
			const result = restore ? await RestoreDefaultSkills(profileID) : await SyncBundledSkills(profileID);
            if (!mounted.current) return false;
            await options.refresh();
            if (!mounted.current) return false;
            await loadSkills();
            if (!mounted.current) return false;
            if (restore) setSkillDetail(null);
            const summary = restore ? restoreDefaultSkillsMessage(result) : syncBundledSkillsMessage(result);
            if (result.syncedFiles > 0) options.setNeedsRebuild(true);
            setSkillsStatus(summary);
            options.setNotice({type: 'ok', message: summary});
            return true;
        } catch (error) {
            if (!mounted.current) return false;
            const message = String(error);
            options.appendLog(message);
            setSkillsStatus(message);
            options.setNotice({type: 'error', message});
            options.setLastOperationError(message);
            return false;
        } finally {
            if (mounted.current) options.setBusy('');
        }
    }

    async function openSkillDirectory(path: string) {
        try {
			await OpenSkillDirectory(options.getProfileID(), path);
        } catch (error) {
            if (!mounted.current) return;
            const message = String(error);
            options.appendLog(message);
            setSkillsStatus(message);
            options.setNotice({type: 'error', message});
        }
    }

    async function loadSkillHubSkills(query: SkillHubQuery) {
		const profileID = options.getProfileID();
        const generation = ++skillHubGeneration.current;
        setSkillHubStatus('正在读取技能中心');
        setSkillHubLoadState('loading');
        setSkillHubLoadError('');
        try {
            const next = await ListSkillHubSkills(profileID, query) as SkillHubState;
            if (!mounted.current || generation !== skillHubGeneration.current || profileID !== options.getProfileID()) return;
			setSkillHubState(next);
            setSkillHubStatus('');
            setSkillHubLoadState('success');
        } catch (error) {
            if (!mounted.current || generation !== skillHubGeneration.current || profileID !== options.getProfileID()) return;
            const message = String(error);
            setSkillHubStatus(message);
            setSkillHubLoadState('error');
            setSkillHubLoadError(message);
            options.appendLog(message);
        }
    }

    async function loadSkillHubDetail(slug: string) {
		const profileID = options.getProfileID();
        const generation = ++skillHubDetailGeneration.current;
        setSkillHubStatus('正在读取技能详情');
        try {
            const next = await GetSkillHubDetail(profileID, slug) as SkillHubDetail;
            if (!mounted.current || generation !== skillHubDetailGeneration.current || profileID !== options.getProfileID()) return;
			setSkillHubDetail(next);
            setSkillHubStatus('');
        } catch (error) {
            if (!mounted.current || generation !== skillHubDetailGeneration.current || profileID !== options.getProfileID()) return;
            const message = String(error);
            setSkillHubDetail(null);
            setSkillHubStatus(message);
            options.appendLog(message);
        }
    }

    async function installSkillHubSkill(slug: string) {
		const profileID = options.getProfileID();
		const ok = await options.run('正在安装技能', () => InstallSkillHubSkill(profileID, slug), {rebuildRequired: true});
        if (!mounted.current || !ok) return false;
        await loadSkills();
        if (!mounted.current) return false;
        await loadSkillHubDetail(slug);
        return mounted.current;
    }

    function resetForProfile() {
        skillsGeneration.current++;
        skillDetailGeneration.current++;
        skillHubGeneration.current++;
        skillHubDetailGeneration.current++;
        setSkillsState(null);
        setSkillDetail(null);
        setSkillsStatus('');
        setSkillsLoadState('idle');
        setSkillsLoadError('');
        setSkillHubDetail(null);
        setSkillHubState(null);
        setSkillHubStatus('');
        setSkillHubLoadState('idle');
        setSkillHubLoadError('');
    }

    return {
        skillsState, skillDetail, skillsStatus, skillsLoadState, skillsLoadError,
        skillHubState, skillHubDetail, skillHubStatus, skillHubLoadState, skillHubLoadError,
        loadSkills, loadSkillDetail, deleteSkill, batchDeleteSkills,
        syncBundledSkills: () => updateBundledSkills(false),
        restoreDefaultSkills: () => updateBundledSkills(true),
        openSkillDirectory, loadSkillHubSkills, loadSkillHubDetail, installSkillHubSkill, resetForProfile,
    };
}
