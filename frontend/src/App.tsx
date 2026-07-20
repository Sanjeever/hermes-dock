import {useEffect, useRef, useState} from 'react';
import {CheckCircle2, CircleAlert, Download, RefreshCcw, RotateCcw} from 'lucide-react';
import './styles/tokens.css';
import './App.css';
import './styles/refresh.css';
import './styles/responsive.css';
import logoUniversal from './assets/images/logo-universal.png';
import {
    CancelWeixinLogin,
    CancelFeishuLogin,
	CancelDingTalkLogin,
    BatchCopyProfileConfig,
    CompleteProfileSetup,
    CreateProfile,
    DeleteProfile,
    DeleteSkill,
    ExportInstanceBackup,
    FactoryResetInstance,
    FetchProviderConfigModelList,
    GetAppState,
    GetSkillDetail,
    GetSkillHubDetail,
    ImportInstanceBackup,
    InspectInstanceBackup,
    InstallSkillHubSkill,
    ListProfileSkills,
    ListSkillHubSkills,
    OpenFileManagement,
    OpenWebManagement,
    OpenSkillDirectory,
    ReadTextFile,
    RebuildHermes,
    RestartHermes,
    RestoreDefaultSkills,
    RestoreDefaultSoul,
    SaveComposeSettings,
    SaveProxySettings,
    SaveFeishuConfig,
	SaveDingTalkConfig,
    SaveModelConfig,
    SaveProviderConfig,
    SaveTextFile,
    SaveWeComConfig,
    SendTestMessage,
    SelectProfile,
    SetHomeChannel,
    SetProfileEnabled,
    StartHermes,
    StartWeixinLogin,
    StartFeishuLogin,
	StartDingTalkLogin,
    StopHermes,
    StopTailLogs,
    SyncBundledSkills,
    SyncBundledContent,
    TailLogs,
    TestModel,
    UnbindPlatform,
    MoveProfile,
    UpdateProfileName,
    SaveWebSettings,
    ChangeWebPassword,
    ResetWebPassword,
    isWebRuntime,
} from './services/api';
import {EventsOn} from './services/events';
import {AssistantsPage} from './pages/AssistantsPage';
import {OperationsPage} from './pages/OperationsPage';
import {OverviewPage} from './pages/OverviewPage';
import {factoryResetPhrase, fallbackProviderConfig, nav} from './constants';
import type {ApplyConfigStatus, AppState, BatchProfileConfigRequest, BatchProfileConfigResult, BundledContentSyncRequest, BundledContentSyncResult, ComposeSettings, EnvVar, InstanceBackupManifest, ModelConfig, ModelOption, Notice, OperationsTab, Page, PlatformKey, ProviderConfig, ProviderEntry, ProxySettings, RunOptions, SkillDetail, SkillHubDetail, SkillHubQuery, SkillHubState, SkillsState, WebSettingsRequest, WizardStep} from './types';
import {advancedFileOptions, containerStatusText, defaultAdvancedPath, doneLabel, envValue, firstProviderID, modelOptionKey, profileFilePath, titleFor, toPlainModelConfig, toPlainProviderConfig} from './utils';
import {channelStatusKey, closedPolicyValue, disabledPolicyValue, firstBoundPlatform, platformLabel, restoreDefaultSkillsMessage, shouldPollRuntimeStatus, syncBundledSkillsMessage} from './appPolicies';
import {useOperationRunner} from './hooks/useOperationRunner';
import {useSkills} from './hooks/useSkills';
import {useUpdates} from './hooks/useUpdates';

function App() {
    const webRuntime = isWebRuntime();
    const [page, setPage] = useState<Page>('overview');
    const [operationsTab, setOperationsTab] = useState<OperationsTab>('runtime');
    const [wizardStep, setWizardStep] = useState<WizardStep | null>(null);
    const [state, setState] = useState<AppState | null>(null);
    const stateRef = useRef<AppState | null>(null);
    const refreshSequenceRef = useRef(0);
	const activeProfileRef = useRef('');
	const advancedPathRef = useRef('data/config.yaml');
	const advancedLoadSequenceRef = useRef(0);
	const soulLoadSequenceRef = useRef(0);
	const advancedEditRevisionRef = useRef(0);
	const soulEditRevisionRef = useRef(0);
	const modelEditRevisionRef = useRef(0);
	const platformEditRevisionRef = useRef(0);
	const deployEditRevisionRef = useRef(0);
    const firstRunWizardCheckedRef = useRef(false);
    const [env, setEnv] = useState<EnvVar[]>([]);
    const [compose, setCompose] = useState<ComposeSettings | null>(null);
    const [proxy, setProxy] = useState<ProxySettings | null>(null);
    const [model, setModel] = useState<ModelConfig | null>(null);
    const [logs, setLogs] = useState<string[]>([]);
    const logRef = useRef<HTMLPreElement>(null!);
    const [logsFollowing, setLogsFollowing] = useState(false);
    const [busy, setBusy] = useState('');
    const [statusRefreshing, setStatusRefreshing] = useState(false);
    const [notice, setNotice] = useState<Notice | null>(null);
    const [refreshError, setRefreshError] = useState('');
    const [needsRebuild, setNeedsRebuild] = useState(false);
    const [qrData, setQrData] = useState('');
    const [qrStatus, setQrStatus] = useState('');
	const [qrPlatform, setQrPlatform] = useState<PlatformKey | ''>('');
    const [advancedPath, setAdvancedPath] = useState('data/config.yaml');
    const [advancedContent, setAdvancedContent] = useState('');
    const [advancedStatus, setAdvancedStatus] = useState('');
    const [advancedDirty, setAdvancedDirty] = useState(false);
    const [advancedOpen, setAdvancedOpen] = useState(false);
    const [backupStatus, setBackupStatus] = useState('');
    const [backupManifest, setBackupManifest] = useState<InstanceBackupManifest | null>(null);
    const [soulContent, setSoulContent] = useState('');
    const [soulStatus, setSoulStatus] = useState('');
    const [soulDirty, setSoulDirty] = useState(false);
    const [showApiKey, setShowApiKey] = useState(false);
    const [autoScrollLogs, setAutoScrollLogs] = useState(true);
    const [providers, setProviders] = useState<ProviderConfig>(fallbackProviderConfig);
    const [modelDirty, setModelDirty] = useState(false);
    const modelDirtyRef = useRef(false);
    const [selectedProvider, setSelectedProvider] = useState('dashscope-payg');
    const [modelOptions, setModelOptions] = useState<ModelOption[]>([]);
    const [modelOptionsKey, setModelOptionsKey] = useState('');
    const [modelListStatus, setModelListStatus] = useState('');
    const [modelTestStatus, setModelTestStatus] = useState('');
    const [selectedAux, setSelectedAux] = useState('vision');
    const [auxModelOptions, setAuxModelOptions] = useState<Record<string, ModelOption[]>>({});
    const [auxModelListStatus, setAuxModelListStatus] = useState('');
    const [newProfileID, setNewProfileID] = useState('');
    const [newProfileName, setNewProfileName] = useState('');
    const [newProfileCopyMode, setNewProfileCopyMode] = useState('clean');
    const [newProfileEnabled, setNewProfileEnabled] = useState(true);
    const [platformDirty, setPlatformDirty] = useState(false);
    const platformDirtyRef = useRef(false);
    const [selectedPlatform, setSelectedPlatform] = useState<PlatformKey>('weixin');
    const [deployDirty, setDeployDirty] = useState(false);
    const deployDirtyRef = useRef(false);
    const [platformLoginProfile, setPlatformLoginProfile] = useState('');
    const [channelActionStatus, setChannelActionStatus] = useState<Record<string, string>>({});
    const [lastOperationError, setLastOperationError] = useState('');
    const [assistantSkillsMode, setAssistantSkillsMode] = useState(false);
    const dirtyMessage = '当前有未保存修改，请先保存或放弃修改后再切换';
    const run = useOperationRunner({refresh, appendLog, setBusy, setNotice, setLastOperationError, setNeedsRebuild});
	const skills = useSkills({getProfileID: () => activeProfileRef.current || 'default', run, refresh, appendLog, setBusy, setNotice, setLastOperationError, setNeedsRebuild});
    const updates = useUpdates({
        appVersion: state?.appVersion,
        appendLog,
        setNotice,
        onStatusChanged: (update) => setState((current) => current ? {...current, update} : current),
    });

    useEffect(() => {
        refresh();
        const offDocker = EventsOn('docker:progress', (event: { line?: string; done?: boolean; code?: number }) => {
            if (event.line) appendLog(event.line);
            if (event.done) {
                appendLog(`命令退出，代码 ${event.code}`);
            }
        });
        const offApply = EventsOn('apply:status', (event: ApplyConfigStatus) => {
            setState((current) => current ? {...current, applyConfig: event} : current);
            if (event.state === 'succeeded') {
                setNotice({type: 'ok', message: event.message || '配置已应用'});
                refresh();
            } else if (event.state === 'failed') {
                setNotice({type: 'error', message: event.error || event.message || '应用配置失败'});
                setLastOperationError(event.error || event.message || '应用配置失败');
                refresh();
            }
        });
        const offBackup = EventsOn('backup:progress', (event: { line?: string }) => {
            if (!event.line) return;
            setBackupStatus(event.line);
            appendLog(event.line);
        });
        const offLogs = EventsOn('logs:line', (event: { line?: string }) => event.line && appendLog(event.line));
        const isCurrentProfileEvent = (event: { profile_id?: string }) => !event.profile_id || event.profile_id === activeProfileRef.current;
        const offQR = EventsOn('weixin-login:qr', (event: { scan_data: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) return;
            setPlatformLoginProfile(event.profile_id || activeProfileRef.current);
            setQrPlatform('weixin');
            setQrData(event.scan_data);
            setQrStatus('等待微信扫码');
        });
        const offQRStatus = EventsOn('weixin-login:status', (event: { status?: string; message?: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) return;
            setQrStatus(event.message || event.status || '');
        });
        const offQRDone = EventsOn('weixin-login:confirmed', (event: { account_id: string; user_id: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) {
                setPlatformLoginProfile((current) => current === event.profile_id ? '' : current);
                setNeedsRebuild(true);
                const profile = stateRef.current?.profiles?.profiles?.find((item) => item.id === event.profile_id);
                setNotice({type: 'ok', message: `${profile?.name || event.profile_id || '其他助手'} 的微信已绑定成功，重建后生效`});
                refresh();
                return;
            }
            setQrStatus(`绑定成功 ${event.user_id || event.account_id}`);
            setQrData('');
            setPlatformLoginProfile('');
			markPlatformDirty(false);
            setNeedsRebuild(true);
            refresh();
        });
        const offQRError = EventsOn('weixin-login:error', (event: { message: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) return;
            setQrStatus(event.message);
            setPlatformLoginProfile('');
        });
        const offFeishuQR = EventsOn('feishu-login:qr', (event: { scan_data: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) return;
            setPlatformLoginProfile(event.profile_id || activeProfileRef.current);
            setQrPlatform('feishu');
            setQrData(event.scan_data);
            setQrStatus('等待飞书扫码');
        });
        const offFeishuDone = EventsOn('feishu-login:confirmed', (event: { status?: string; bot_name?: string; domain?: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) {
                setPlatformLoginProfile((current) => current === event.profile_id ? '' : current);
                setNeedsRebuild(true);
                const profile = stateRef.current?.profiles?.profiles?.find((item) => item.id === event.profile_id);
                setNotice({type: 'ok', message: `${profile?.name || event.profile_id || '其他助手'} 的飞书 / Lark 已绑定成功，重建后生效`});
                refresh();
                return;
            }
            const platform = event.domain === 'lark' ? 'Lark' : '飞书';
            setQrStatus(event.status || `${platform} 已绑定成功${event.bot_name ? `：${event.bot_name}` : ''}`);
            setQrData('');
            setPlatformLoginProfile('');
			markPlatformDirty(false);
            setNeedsRebuild(true);
            refresh();
        });
        const offFeishuError = EventsOn('feishu-login:error', (event: { message: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) return;
            setQrStatus(event.message);
            setPlatformLoginProfile('');
        });
        const offDingTalkQR = EventsOn('dingtalk-login:qr', (event: { scan_data: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) return;
            setPlatformLoginProfile(event.profile_id || activeProfileRef.current);
            setQrPlatform('dingtalk');
            setQrData(event.scan_data);
            setQrStatus('等待钉钉扫码');
        });
        const offDingTalkDone = EventsOn('dingtalk-login:confirmed', (event: { status?: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) {
                setPlatformLoginProfile((current) => current === event.profile_id ? '' : current);
                setNeedsRebuild(true);
                const profile = stateRef.current?.profiles?.profiles?.find((item) => item.id === event.profile_id);
                setNotice({type: 'ok', message: `${profile?.name || event.profile_id || '其他助手'} 的钉钉已绑定成功，应用配置后生效`});
                refresh();
                return;
            }
            setQrStatus(event.status || '钉钉已绑定成功');
            setQrData('');
            setPlatformLoginProfile('');
			markPlatformDirty(false);
            setNeedsRebuild(true);
            refresh();
        });
        const offDingTalkError = EventsOn('dingtalk-login:error', (event: { message: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) return;
            setQrStatus(event.message);
            setPlatformLoginProfile('');
        });
        return () => {
            offDocker();
            offApply();
            offBackup();
            offLogs();
            offQR();
            offQRStatus();
            offQRDone();
            offQRError();
            offFeishuQR();
            offFeishuDone();
            offFeishuError();
            offDingTalkQR();
            offDingTalkDone();
            offDingTalkError();
        };
    }, []);

    useEffect(() => {
        modelDirtyRef.current = modelDirty;
    }, [modelDirty]);

    useEffect(() => {
        platformDirtyRef.current = platformDirty;
    }, [platformDirty]);

    useEffect(() => {
        deployDirtyRef.current = deployDirty;
    }, [deployDirty]);

	useEffect(() => {
		advancedPathRef.current = advancedPath;
	}, [advancedPath]);

    useEffect(() => {
        if (page !== 'settings' || operationsTab !== 'advanced') return;
        if (!advancedOpen) return;
        if (advancedDirty) return;
        loadAdvancedFile(advancedPath);
    }, [page, operationsTab, advancedPath, advancedOpen]);

    useEffect(() => {
        if (page !== 'assistants' || wizardStep !== 'soul') return;
        if (soulDirty) return;
        loadSoulFile(state?.activeProfile || 'default');
    }, [page, wizardStep, state?.activeProfile]);

    useEffect(() => {
        if (!autoScrollLogs || !logRef.current) return;
        logRef.current.scrollTop = logRef.current.scrollHeight;
    }, [logs, autoScrollLogs]);

    useEffect(() => {
        if (!state?.activeProfile || advancedDirty) return;
        setAdvancedPath(defaultAdvancedPath(state.activeProfile));
    }, [state?.activeProfile]);

    useEffect(() => {
        activeProfileRef.current = state?.activeProfile || 'default';
        setQrData('');
        setQrStatus('');
    }, [state?.activeProfile]);

    useEffect(() => {
        if (!state?.activeProfile) return;
        skills.resetForProfile();
        skills.loadSkills();
    }, [state?.activeProfile]);

    useEffect(() => {
        const profile = state?.profiles?.profiles?.find((item) => item.id === state.activeProfile);
        if (!profile) return;
        if (!profile.setupCompletedAt && !wizardStep) {
            setWizardStep('model');
        }
    }, [state?.activeProfile, state?.profiles?.profiles, wizardStep]);

    useEffect(() => {
        if (!state) return;
        const applyActive = !!state.applyConfig?.active;
        const profileStates = Object.values(state.profileStatus?.profiles || {}).map((status) => status.state);
        if (!shouldPollRuntimeStatus(applyActive, !!busy, state.containerStatus, profileStates)) return;
        const timer = window.setTimeout(() => {
            refresh();
        }, 1500);
        return () => window.clearTimeout(timer);
    }, [state, busy]);

    async function refresh() {
        const sequence = ++refreshSequenceRef.current;
        let next: unknown;
        try {
			next = await GetAppState(activeProfileRef.current);
        } catch (error) {
            if (sequence !== refreshSequenceRef.current) return '';
            const message = String(error);
            setRefreshError(message);
            appendLog(message);
            return message;
        }
        if (sequence !== refreshSequenceRef.current) return '';
        const nextState = next as AppState;
        const firstRefresh = !stateRef.current;
        const previousProfile = stateRef.current?.activeProfile;
        const profileChanged = !!previousProfile && previousProfile !== nextState.activeProfile;
        setNeedsRebuild(nextState.needsRebuild);
        stateRef.current = nextState;
        activeProfileRef.current = nextState.activeProfile || 'default';
        setState(nextState);
        const nextEnv = nextState.environment || [];
        if (!platformDirtyRef.current) {
            setEnv(nextEnv);
        }
        if (!deployDirtyRef.current) {
            setCompose(nextState.compose);
            setProxy(nextState.proxy);
        }
        if (!modelDirtyRef.current) {
            setModel(nextState.model);
        }
        const nextProviders = nextState.providers || fallbackProviderConfig;
        if (!modelDirtyRef.current) {
            setProviders(nextProviders);
            setSelectedProvider(nextState.model?.provider && nextProviders.providers[nextState.model.provider] ? nextState.model.provider : firstProviderID(nextProviders));
        }
        if (firstRefresh || profileChanged) {
            setSelectedPlatform(firstBoundPlatform(nextEnv));
        }
        if (!firstRunWizardCheckedRef.current) {
            firstRunWizardCheckedRef.current = true;
            const active = nextState.profiles?.profiles?.find((profile) => profile.id === nextState.activeProfile);
            if (active?.id === 'default' && !active.setupCompletedAt) {
                setPage('assistants');
                setWizardStep('model');
            }
        }
        setRefreshError('');
        return '';
    }

    async function refreshStatus() {
        setStatusRefreshing(true);
        try {
            const message = await refresh();
            setNotice(message
                ? {type: 'error', message: `刷新状态失败：${message}`}
                : {type: 'ok', message: '总览、助手和运行状态已刷新'});
        } finally {
            setStatusRefreshing(false);
        }
    }

    async function selectProfile(id: string) {
        if (id === state?.activeProfile) return true;
        if (platformLoginProfile && platformLoginProfile !== id) {
            setNotice({type: 'error', message: '平台扫码绑定进行中，请先完成或取消后再切换助手'});
            return false;
        }
        if (hasUnsavedChanges()) {
            setNotice({type: 'error', message: dirtyMessage});
            return false;
        }
        const target = state?.profiles?.profiles?.find((profile) => profile.id === id);
        return await run('正在切换助手', () => SelectProfile(id), {
            beforeRefresh: () => {
				activeProfileRef.current = id;
                markModelDirty(false);
                markPlatformDirty(false);
                markDeployDirty(false);
            },
            afterSuccess: () => {
                setAdvancedDirty(false);
                setAdvancedOpen(false);
                setAdvancedPath(defaultAdvancedPath(id));
                setSoulDirty(false);
                setSoulContent('');
                setWizardStep(target?.setupCompletedAt ? null : 'model');
            },
        });
    }

    async function createProfile() {
        return await run('正在创建助手', () => CreateProfile({
            id: newProfileID,
            name: newProfileName,
            enabled: newProfileEnabled,
            copyFrom: state?.activeProfile || 'default',
            copyMode: newProfileCopyMode,
        }), {
            beforeRefresh: () => {
				activeProfileRef.current = newProfileID;
                soulLoadSequenceRef.current++;
                setSoulContent('');
                setSoulDirty(false);
                markModelDirty(false);
                markPlatformDirty(false);
                markDeployDirty(false);
            },
            afterSuccess: () => {
                setNewProfileID('');
                setNewProfileName('');
                setNewProfileCopyMode('clean');
                setNewProfileEnabled(true);
                setPage('assistants');
                setWizardStep('model');
            },
        });
    }

    async function deleteProfile(id: string) {
        return await run('正在删除助手', () => DeleteProfile(id, id), {
            rebuildRequired: true,
            beforeRefresh: () => {
                if (activeProfileRef.current === id) {
                    activeProfileRef.current = 'default';
                    soulLoadSequenceRef.current++;
                    setSoulContent('');
                    setSoulDirty(false);
                }
            },
        });
    }


    function appendLog(line: string) {
        setLogs((current) => [...current.slice(-300), line]);
    }

    async function fetchModels() {
        if (!model) return;
        const providerID = model.provider || selectedProvider || firstProviderID(providers);
        const provider = providers.providers[providerID];
        if (!provider) return;
        const optionsKey = modelOptionKey(providerID);
        setModelListStatus('正在拉取模型列表');
        try {
			const items = await FetchProviderConfigModelList(activeProfileRef.current, provider);
            setModelOptionsKey(optionsKey);
            setModelOptions(items as ModelOption[]);
            setModelListStatus(`已拉取 ${(items as ModelOption[]).length} 个模型`);
        } catch (error) {
            setModelOptionsKey(optionsKey);
            setModelOptions([]);
            setModelListStatus(String(error));
            appendLog(String(error));
        }
    }

    async function fetchProviderModels(providerID: string, provider: ProviderEntry) {
        const optionsKey = modelOptionKey(providerID);
        setModelListStatus('正在拉取模型列表');
        try {
			const items = await FetchProviderConfigModelList(activeProfileRef.current, provider);
            setModelOptionsKey(optionsKey);
            setModelOptions(items as ModelOption[]);
            setModelListStatus(`已拉取 ${(items as ModelOption[]).length} 个模型`);
        } catch (error) {
            setModelOptionsKey(optionsKey);
            setModelOptions([]);
            setModelListStatus(String(error));
            appendLog(String(error));
        }
    }

    async function fetchAuxModels(providerID: string) {
        const provider = providers.providers[providerID];
        if (!provider || provider.disabled) {
            setAuxModelListStatus('供应商不可用');
            return;
        }
        if (provider.apiKey.trim() === '') {
            setAuxModelListStatus('请先在基础模型服务里填写 API 密钥');
            return;
        }
        const optionsKey = modelOptionKey(providerID);
        setAuxModelListStatus('正在拉取模型列表');
        try {
			const items = await FetchProviderConfigModelList(activeProfileRef.current, provider);
            setAuxModelOptions((current) => ({...current, [optionsKey]: items as ModelOption[]}));
            setAuxModelListStatus(`已拉取 ${(items as ModelOption[]).length} 个模型`);
        } catch (error) {
            setAuxModelOptions((current) => ({...current, [optionsKey]: []}));
            setAuxModelListStatus(String(error));
            appendLog(String(error));
        }
    }



    async function loadAdvancedFile(path: string) {
		const sequence = ++advancedLoadSequenceRef.current;
		const profileID = activeProfileRef.current || 'default';
        const revision = advancedEditRevisionRef.current;
        setAdvancedStatus('正在读取文件');
        setAdvancedContent('');
        try {
			const content = await ReadTextFile(profileID, path);
			if (sequence !== advancedLoadSequenceRef.current || path !== advancedPathRef.current || profileID !== activeProfileRef.current || revision !== advancedEditRevisionRef.current) return;
			setAdvancedContent(content);
            setAdvancedStatus(`已加载 ${path}`);
            setAdvancedDirty(false);
        } catch (error) {
			if (sequence !== advancedLoadSequenceRef.current || path !== advancedPathRef.current || profileID !== activeProfileRef.current || revision !== advancedEditRevisionRef.current) return;
            setAdvancedContent('');
            setAdvancedStatus(String(error));
            setAdvancedDirty(false);
            appendLog(String(error));
        }
    }

    async function saveAdvancedFile() {
		const profileID = activeProfileRef.current || 'default';
		const path = advancedPath;
		const revision = advancedEditRevisionRef.current;
        setBusy('正在保存文件');
        setNotice({type: 'info', message: '正在保存文件'});
        setAdvancedStatus('正在保存文件');
        try {
            let confirm = '';
            if (webRuntime && advancedPath === 'docker-compose.override.yaml') {
                confirm = window.prompt('输入“确认”保存 Docker Compose 覆盖文件') || '';
                if (confirm !== '确认') {
                    setAdvancedStatus('已取消保存');
                    setNotice({type: 'error', message: '已取消保存'});
                    return;
                }
            }
			await SaveTextFile(profileID, {path, content: advancedContent, reason: 'before-advanced-save', confirm});
			if (revision === advancedEditRevisionRef.current && path === advancedPathRef.current) setAdvancedDirty(false);
            setNeedsRebuild(true);
            const refreshMessage = await refresh();
            if (refreshMessage) {
                setAdvancedStatus(`已保存 ${advancedPath}，但刷新状态失败`);
                setNotice({type: 'error', message: `已保存文件，但刷新状态失败：${refreshMessage}`});
                return;
            }
            setAdvancedStatus(`已保存 ${advancedPath}，应用配置后生效`);
            setNotice({type: 'ok', message: '已保存文件'});
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setAdvancedStatus(message);
            setNotice({type: 'error', message});
        } finally {
            setBusy('');
        }
    }

    async function loadSoulFile(profileID: string) {
		const sequence = ++soulLoadSequenceRef.current;
        const revision = soulEditRevisionRef.current;
        const path = profileFilePath(profileID, 'SOUL.md');
        setSoulStatus('正在读取人格文件');
        setSoulContent('');
        try {
			const content = await ReadTextFile(profileID, path);
			if (sequence !== soulLoadSequenceRef.current || profileID !== activeProfileRef.current || revision !== soulEditRevisionRef.current) return;
			setSoulContent(content);
            setSoulStatus(`已加载 ${path}`);
            setSoulDirty(false);
        } catch (error) {
			if (sequence !== soulLoadSequenceRef.current || profileID !== activeProfileRef.current || revision !== soulEditRevisionRef.current) return;
            setSoulContent('');
            setSoulStatus(String(error));
            setSoulDirty(false);
            appendLog(String(error));
        }
    }

    async function saveSoulFile() {
		const profileID = activeProfileRef.current || 'default';
		const path = profileFilePath(profileID, 'SOUL.md');
		const revision = soulEditRevisionRef.current;
        setBusy('正在保存人格文件');
        setNotice({type: 'info', message: '正在保存人格文件'});
        setSoulStatus('正在保存人格文件');
        try {
			await SaveTextFile(profileID, {path, content: soulContent, reason: 'before-soul-save'});
			if (revision === soulEditRevisionRef.current && profileID === activeProfileRef.current) setSoulDirty(false);
            setNeedsRebuild(true);
            setSoulStatus(`已保存 ${path}`);
            setNotice({type: 'ok', message: '已保存人格文件'});
            return true;
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setSoulStatus(message);
            setNotice({type: 'error', message});
            return false;
        } finally {
            setBusy('');
        }
    }

    async function restoreDefaultSoul() {
		const profileID = activeProfileRef.current || 'default';
        setBusy('正在恢复默认人格');
        setNotice({type: 'info', message: '正在恢复默认人格'});
        setSoulStatus('正在恢复默认人格');
        try {
			await RestoreDefaultSoul(profileID);
            await loadSoulFile(profileID);
            setSoulDirty(false);
            setNeedsRebuild(true);
            setSoulStatus('已恢复默认人格，应用配置后生效');
            setNotice({type: 'ok', message: '已恢复默认人格'});
            return true;
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setSoulStatus(message);
            setNotice({type: 'error', message});
            return false;
        } finally {
            setBusy('');
        }
    }

    async function saveModelService() {
        if (!model) return false;
		const profileID = activeProfileRef.current || 'default';
		const revision = modelEditRevisionRef.current;
        return await run('正在保存模型服务', async () => {
			await SaveProviderConfig(profileID, toPlainProviderConfig(providers));
			await SaveModelConfig(profileID, toPlainModelConfig(model));
		}, {rebuildRequired: true, beforeRefresh: () => {
			if (revision === modelEditRevisionRef.current) markModelDirty(false);
		}});
    }

    async function saveCurrentPlatform() {
        if (!platformDirty) return true;
        if (selectedPlatform === 'wecom') {
            return await saveWeComConfig();
        }
        if (selectedPlatform === 'feishu') {
            return await saveFeishuConfig();
        }
		if (selectedPlatform === 'dingtalk') {
			return await saveDingTalkConfig();
		}
        markPlatformDirty(false);
        return true;
    }

    async function finishProfileSetup(apply: boolean) {
        const profileID = state?.activeProfile || 'default';
        const completed = await run('正在完成配置', () => CompleteProfileSetup(profileID), {
            afterSuccess: () => {
                setWizardStep(null);
            },
        });
        if (!completed || !apply) return completed;
        return startApplyConfiguration();
    }

    async function startApplyConfiguration() {
        setBusy('正在启动应用任务');
        setNotice({type: 'info', message: '正在启动应用配置任务'});
        setLastOperationError('');
        try {
            await RebuildHermes();
            await refresh();
            return true;
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
            setLastOperationError(message);
            await refresh();
            return false;
        } finally {
            setBusy('');
        }
    }

    async function batchCopyProfiles(request: BatchProfileConfigRequest): Promise<BatchProfileConfigResult | null> {
        setBusy('正在批量应用配置');
        setNotice({type: 'info', message: '正在批量应用配置'});
        try {
            const result = await BatchCopyProfileConfig(request);
            await refresh();
            setNotice({type: result.failed ? 'error' : 'ok', message: `已完成 ${result.succeeded} 个助手${result.failed ? `，${result.failed} 个失败` : ''}`});
            return result;
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
            return null;
        } finally {
            setBusy('');
        }
    }

    async function syncBundledContent(request: BundledContentSyncRequest): Promise<BundledContentSyncResult | null> {
        setBusy('正在同步内置内容');
        setNotice({type: 'info', message: '正在同步内置内容'});
        try {
            const result = await SyncBundledContent(request);
            await refresh();
            setNotice({
                type: result.failed ? 'error' : 'ok',
                message: `新增 ${result.added} 项，更新 ${result.updated} 项，保留 ${result.skipped} 项用户修改${result.failed ? `；${result.failed} 个助手失败` : ''}`,
            });
            return result;
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
            return null;
        } finally {
            setBusy('');
        }
    }

    function hasUnsavedChanges() {
        return advancedDirty || soulDirty || modelDirty || platformDirty || deployDirty;
    }

    function markModelDirty(value: boolean) {
		if (value) modelEditRevisionRef.current++;
        modelDirtyRef.current = value;
        setModelDirty(value);
    }

    function markPlatformDirty(value: boolean) {
		if (value) platformEditRevisionRef.current++;
        platformDirtyRef.current = value;
        setPlatformDirty(value);
    }

    function markDeployDirty(value: boolean) {
		if (value) deployEditRevisionRef.current++;
        deployDirtyRef.current = value;
        setDeployDirty(value);
    }

    async function startWeixinLogin() {
		const profileID = activeProfileRef.current || 'default';
		if (platformDirty) {
			setNotice({type: 'error', message: '当前平台配置有未保存修改，请先保存或放弃后再扫码'});
			return;
		}
        setPlatformLoginProfile(profileID);
        setQrPlatform('weixin');
		const ok = await run('正在启动微信扫码登录', () => StartWeixinLogin(profileID));
        if (!ok) {
            setPlatformLoginProfile('');
        }
    }

    async function cancelWeixinLogin() {
        const ok = await run('正在取消微信扫码登录', CancelWeixinLogin);
        if (!ok) return;
        setQrData('');
        setQrStatus('');
        setQrPlatform('');
        setPlatformLoginProfile('');
    }

    async function startFeishuLogin() {
		const profileID = activeProfileRef.current || 'default';
		if (platformDirty) {
			setNotice({type: 'error', message: '当前平台配置有未保存修改，请先保存或放弃后再扫码'});
			return;
		}
        const replacing = !!envValue(env, 'FEISHU_APP_ID') && !!envValue(env, 'FEISHU_APP_SECRET');
        if (replacing && !window.confirm('扫码创建的新机器人会替换当前飞书 / Lark 绑定；旧飞书应用不会被自动删除。是否继续？')) return;
        setPlatformLoginProfile(profileID);
        setQrPlatform('feishu');
		setQrData('');
		setQrStatus('正在生成飞书二维码');
		const ok = await run('正在启动飞书扫码绑定', () => StartFeishuLogin(profileID));
        if (!ok) setPlatformLoginProfile('');
    }

    async function cancelFeishuLogin() {
        const ok = await run('正在取消飞书扫码绑定', CancelFeishuLogin);
        if (!ok) return;
        setQrData('');
        setQrStatus('');
        setQrPlatform('');
        setPlatformLoginProfile('');
    }

    async function startDingTalkLogin() {
		const profileID = activeProfileRef.current || 'default';
		if (platformDirty) {
			setNotice({type: 'error', message: '当前平台配置有未保存修改，请先保存或放弃后再扫码'});
			return;
		}
		const replacing = !!envValue(env, 'DINGTALK_CLIENT_ID') && !!envValue(env, 'DINGTALK_CLIENT_SECRET');
		if (replacing && !window.confirm('扫码创建的新钉钉机器人会替换当前钉钉绑定。是否继续？')) return;
		setPlatformLoginProfile(profileID);
		setQrPlatform('dingtalk');
		setQrData('');
		setQrStatus('正在生成钉钉二维码');
		const ok = await run('正在启动钉钉扫码绑定', () => StartDingTalkLogin(profileID));
		if (!ok) setPlatformLoginProfile('');
	}

    async function cancelDingTalkLogin() {
		const ok = await run('正在取消钉钉扫码绑定', CancelDingTalkLogin);
		if (!ok) return;
		setQrData('');
		setQrStatus('');
		setQrPlatform('');
		setPlatformLoginProfile('');
	}

    async function saveWeComConfig() {
		if (platformLoginProfile) {
			setNotice({type: 'error', message: '平台扫码绑定进行中，请先完成或取消'});
			return false;
		}
        if (envValue(env, 'WECOM_BOT_ID').trim() === '' || envValue(env, 'WECOM_SECRET').trim() === '') {
            const message = '请填写企业微信 Bot ID 和 Secret 后再保存';
            setNotice({type: 'error', message});
            setLastOperationError(message);
            return false;
        }
		const profileID = activeProfileRef.current || 'default';
		const revision = platformEditRevisionRef.current;
		return await run('正在保存企业微信配置', () => SaveWeComConfig(profileID, {
            botId: envValue(env, 'WECOM_BOT_ID'),
            secret: envValue(env, 'WECOM_SECRET'),
            websocketUrl: envValue(env, 'WECOM_WEBSOCKET_URL'),
            dmPolicy: closedPolicyValue(envValue(env, 'WECOM_DM_POLICY')),
            allowedUsers: '',
            groupPolicy: closedPolicyValue(envValue(env, 'WECOM_GROUP_POLICY')),
            groupAllowUsers: '',
		}), {rebuildRequired: true, beforeRefresh: () => {
			if (revision === platformEditRevisionRef.current) markPlatformDirty(false);
		}});
    }

    async function saveFeishuConfig() {
		if (platformLoginProfile) {
			setNotice({type: 'error', message: '平台扫码绑定进行中，请先完成或取消'});
			return false;
		}
        if (envValue(env, 'FEISHU_APP_ID').trim() === '' || envValue(env, 'FEISHU_APP_SECRET').trim() === '') {
            const message = '请填写飞书 App ID 和 App Secret 后再保存';
            setNotice({type: 'error', message});
            setLastOperationError(message);
            return false;
        }
		const profileID = activeProfileRef.current || 'default';
		const revision = platformEditRevisionRef.current;
		return await run('正在保存飞书配置', () => SaveFeishuConfig(profileID, {
            appId: envValue(env, 'FEISHU_APP_ID'),
            appSecret: envValue(env, 'FEISHU_APP_SECRET'),
            domain: envValue(env, 'FEISHU_DOMAIN') || 'feishu',
            allowedUsers: '',
            groupPolicy: disabledPolicyValue(envValue(env, 'FEISHU_GROUP_POLICY')),
		}), {rebuildRequired: true, beforeRefresh: () => {
			if (revision === platformEditRevisionRef.current) markPlatformDirty(false);
		}});
    }

    async function saveDingTalkConfig() {
		if (platformLoginProfile) {
			setNotice({type: 'error', message: '平台扫码绑定进行中，请先完成或取消'});
			return false;
		}
		if (envValue(env, 'DINGTALK_CLIENT_ID').trim() === '' || envValue(env, 'DINGTALK_CLIENT_SECRET').trim() === '') {
			const message = '请填写钉钉 AppKey 和 AppSecret 后再保存';
			setNotice({type: 'error', message});
			setLastOperationError(message);
			return false;
		}
		const profileID = activeProfileRef.current || 'default';
		const revision = platformEditRevisionRef.current;
		return await run('正在保存钉钉配置', () => SaveDingTalkConfig(profileID, {
			clientId: envValue(env, 'DINGTALK_CLIENT_ID'),
			clientSecret: envValue(env, 'DINGTALK_CLIENT_SECRET'),
			requireMention: envValue(env, 'DINGTALK_REQUIRE_MENTION') !== 'false',
		}), {rebuildRequired: true, beforeRefresh: () => {
			if (revision === platformEditRevisionRef.current) markPlatformDirty(false);
		}});
	}

    async function unbindPlatform(platform: PlatformKey) {
        const label = platformLabel(platform);
		const profileID = activeProfileRef.current || 'default';
		await run(`正在取消绑定${label}`, () => UnbindPlatform(profileID, platform), {
            rebuildRequired: true,
            beforeRefresh: () => markPlatformDirty(false),
            afterSuccess: () => {
                if (platform === 'weixin') {
                    setQrData('');
                    setQrStatus('');
                    setPlatformLoginProfile('');
                }
            },
        });
    }

    function selectPlatform(value: PlatformKey) {
        if (platformDirty && value !== selectedPlatform) {
            setNotice({type: 'error', message: '当前平台配置有未保存修改，请先保存后再切换平台'});
            return;
        }
        setSelectedPlatform(value);
    }

    function discardDeployChanges() {
        if (!state) return;
        setCompose(state.compose);
        setProxy(state.proxy);
        markDeployDirty(false);
        setNotice({type: 'ok', message: '已放弃部署参数修改'});
    }

    async function saveDeploySettings() {
        if (!compose || !proxy) return false;
		const revision = deployEditRevisionRef.current;
        return await run('正在保存部署配置', async () => {
            await SaveComposeSettings(compose);
            await SaveProxySettings(proxy);
		}, {rebuildRequired: true, beforeRefresh: () => {
			if (revision === deployEditRevisionRef.current) markDeployDirty(false);
		}});
    }

	async function saveWebManagementSettings(settings: WebSettingsRequest) {
		const currentHost = window.location.hostname;
		if (webRuntime && settings.host === '127.0.0.1' && currentHost !== '127.0.0.1' && currentHost !== 'localhost' && currentHost !== '::1') {
			setNotice({type: 'error', message: '远程 Web 页面不能把访问范围改为“仅本机”，请在桌面端操作'});
			return false;
		}
		const ok = await run('正在保存 Web 管理设置', () => SaveWebSettings(settings));
		if (!ok || !webRuntime || !settings.enabled) return ok;
		const host = settings.host === '0.0.0.0' ? window.location.hostname : settings.host;
		const target = `http://${host}:${settings.port}`;
		if (target === window.location.origin) return true;
		for (let attempt = 0; attempt < 20; attempt++) {
			const controller = new AbortController();
			const timeout = window.setTimeout(() => controller.abort(), 500);
			try {
				await fetch(`${target}/healthz`, {mode: 'no-cors', cache: 'no-store', signal: controller.signal});
				window.location.replace(target);
				return true;
			} catch {
				await new Promise((resolve) => window.setTimeout(resolve, 250));
			} finally {
				window.clearTimeout(timeout);
			}
		}
		setNotice({type: 'error', message: '新的 Web 管理地址未能启动，已保留当前页面，请检查桌面端状态'});
		return false;
	}

    function openOperations(tab: OperationsTab) {
        setOperationsTab(tab);
        setPage('operations');
    }

    function navigatePage(next: Page) {
        if (next === 'operations') {
            setOperationsTab('runtime');
        } else if (next === 'settings') {
            setOperationsTab('basic');
        }
        setPage(next);
    }

    async function factoryReset() {
        await run('正在恢复出厂设置', FactoryResetInstance, {
            beforeRefresh: () => {
                activeProfileRef.current = 'default';
                soulLoadSequenceRef.current++;
                setSoulContent('');
                setSoulDirty(false);
            },
            afterSuccess: () => {
                setLogs([]);
                setNeedsRebuild(false);
                setAdvancedDirty(false);
                setAdvancedOpen(false);
                setWizardStep('model');
                setPage('assistants');
            },
        });
    }

    async function exportInstanceBackup(targetPath: string) {
		setBackupManifest(null);
        setBusy('正在导出备份');
        setNotice({type: 'info', message: '正在导出备份'});
        setBackupStatus('正在导出备份');
        try {
            const manifest = await ExportInstanceBackup(targetPath);
            setBackupStatus(`已导出 ${manifest.fileCount} 个文件`);
            setNotice({type: 'ok', message: `已导出备份：${manifest.path || '已保存'}`});
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setBackupStatus(message);
            setNotice({type: 'error', message});
        } finally {
            setBusy('');
        }
    }

    async function inspectInstanceBackup(path: string) {
        setBusy('正在检查备份');
        setNotice({type: 'info', message: '正在检查备份'});
        setBackupStatus('正在检查备份');
        try {
            const manifest = await InspectInstanceBackup(path);
            setBackupManifest(manifest);
            setBackupStatus(`已检查 ${manifest.profiles?.length || 0} 个 Profile`);
            setNotice({type: 'ok', message: '备份检查完成'});
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setBackupManifest(null);
            setBackupStatus(message);
            setNotice({type: 'error', message});
        } finally {
            setBusy('');
        }
    }

    async function importInstanceBackup(path: string, confirm: string) {
        setBusy('正在导入备份');
        setNotice({type: 'info', message: '正在导入备份'});
        setBackupStatus('正在导入备份');
        try {
            const result = await ImportInstanceBackup(path, confirm);
            setBackupManifest(result.manifest);
            setLogs([]);
            setNeedsRebuild(true);
            setAdvancedDirty(false);
            setAdvancedOpen(false);
            soulLoadSequenceRef.current++;
            setSoulContent('');
            setSoulDirty(false);
            setBackupStatus('导入完成，应用配置后生效');
            activeProfileRef.current = 'default';
            const refreshMessage = await refresh();
            if (refreshMessage) {
                setNotice({type: 'error', message: `已导入备份，但刷新状态失败：${refreshMessage}`});
                return;
            }
            setNotice({type: 'ok', message: `已导入备份，导入前备份保存在 ${result.preImportBackupPath}`});
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setBackupStatus(message);
            setNotice({type: 'error', message});
        } finally {
            setBusy('');
        }
    }

    function changeAdvancedPath(path: string) {
        if (path === advancedPath) return;
        if (advancedDirty) {
            setNotice({type: 'error', message: '当前文件有未保存修改，请先保存后再切换文件'});
            return;
        }
        setAdvancedDirty(false);
		advancedPathRef.current = path;
		setAdvancedPath(path);
    }

    async function openFileManagement() {
        try {
            await OpenFileManagement();
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
        }
    }

    async function openWebManagement() {
        try {
            await OpenWebManagement();
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
        }
    }

    async function tailLogs() {
        if (logsFollowing) {
            await StopTailLogs();
            setLogsFollowing(false);
            setNotice({type: 'ok', message: '已停止跟随日志'});
            return;
        }
        setNotice({type: 'info', message: '正在跟随日志'});
        try {
            await TailLogs();
            setLogsFollowing(true);
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
        }
    }

    async function copyLogs() {
        try {
            await navigator.clipboard.writeText(logs.join('\n'));
            setNotice({type: 'ok', message: '已复制日志'});
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
        }
    }

    async function testCurrentModel() {
        if (!model) return;
        const provider = model?.provider ? providers.providers[model.provider] : undefined;
        const label = provider?.label || model?.provider || '当前供应商';
        const modelName = model?.default || '当前模型';
        const hadUnsavedModel = modelDirtyRef.current;
        const busyLabel = hadUnsavedModel ? '正在保存并测试模型' : '正在测试模型';
        setBusy(busyLabel);
        setNotice({type: 'info', message: busyLabel});
        setLastOperationError('');
        setModelTestStatus(`${hadUnsavedModel ? '正在保存并测试' : '正在测试'}：${label} / ${modelName}`);
        try {
            if (hadUnsavedModel) {
				const profileID = activeProfileRef.current || 'default';
				await SaveProviderConfig(profileID, toPlainProviderConfig(providers));
				await SaveModelConfig(profileID, toPlainModelConfig(model));
                markModelDirty(false);
                setNeedsRebuild(true);
            }
			await TestModel(activeProfileRef.current || 'default');
            await refresh();
            setModelTestStatus(`已测试：${label} / ${modelName}${hadUnsavedModel ? '（已先保存当前配置）' : ''}`);
            setNotice({type: 'ok', message: '已测试模型'});
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setModelTestStatus('模型测试失败，请查看错误信息');
            setNotice({type: 'error', message});
            setLastOperationError(message);
        } finally {
            setBusy('');
        }
    }

    async function setHomeChannel(platform: string, id: string) {
        const key = channelStatusKey(platform, id, 'home');
        const testKey = channelStatusKey(platform, id, 'test');
        setChannelActionStatus((current) => {
            const next = {...current, [key]: '正在设置默认通道'};
            delete next[testKey];
            return next;
        });
		const profileID = activeProfileRef.current || 'default';
		const ok = await run('正在设置默认通道', () => SetHomeChannel(profileID, platform, id), {rebuildRequired: true});
        setChannelActionStatus((current) => ({...current, [key]: ok ? '已设为默认，应用配置后生效' : '设置默认通道失败'}));
    }

    async function testChannel(platform: string, id: string) {
        const key = channelStatusKey(platform, id, 'test');
        const homeKey = channelStatusKey(platform, id, 'home');
        setChannelActionStatus((current) => {
            const next = {...current, [key]: '正在发送测试消息'};
            delete next[homeKey];
            return next;
        });
		const profileID = activeProfileRef.current || 'default';
		const ok = await run('正在发送测试消息', () => SendTestMessage(profileID, platform, id, '企智盒测试消息'));
        setChannelActionStatus((current) => ({...current, [key]: ok ? '测试消息已发送' : '测试消息发送失败'}));
    }

    const statusClass = state?.containerStatus === 'running' ? 'ok' : 'warn';
    const weixinBound = envValue(env, 'WEIXIN_ACCOUNT_ID') && envValue(env, 'WEIXIN_TOKEN');
    const wecomBound = envValue(env, 'WECOM_BOT_ID') && envValue(env, 'WECOM_SECRET');
    const feishuBound = envValue(env, 'FEISHU_APP_ID') && envValue(env, 'FEISHU_APP_SECRET');
	const dingtalkBound = envValue(env, 'DINGTALK_CLIENT_ID') && envValue(env, 'DINGTALK_CLIENT_SECRET');
	const homeChannels = {
		weixin: envValue(env, 'WEIXIN_HOME_CHANNEL'),
		dingtalk: envValue(env, 'DINGTALK_HOME_CHANNEL'),
	};
    const currentProviderKey = model ? model.provider : '';
    const currentModelOptionsKey = model ? modelOptionKey(currentProviderKey) : '';
    const visibleModelOptions = currentModelOptionsKey === modelOptionsKey ? modelOptions : [];
    const activeProfile = state?.profiles?.profiles?.find((profile) => profile.id === state.activeProfile);
    const activeSetupDone = !!activeProfile?.setupCompletedAt;
    const applyActive = !!state?.applyConfig?.active;
    const blockingBusy = busy || (applyActive ? '正在应用配置' : '');
    const showContainerStatus = page !== 'assistants' || activeSetupDone;
    const showRebuildBanner = needsRebuild && !applyActive && (page !== 'assistants' || activeSetupDone);
    const unsavedChanges = hasUnsavedChanges();
    const updateInfo = updates.info;
    const showUpdateBanner = !!updateInfo?.available && !updateInfo.dismissed;
    const updateBannerDetail = updates.progress || '点击“立即更新”后将自动下载、安装并重启启动器，Hermes Agent 容器不会停止。';

    return (
        <div className="shell">
            <a className="skip-link" href="#main-content">跳到主内容</a>
            <aside className="rail">
                <div className="brand">
                    <div className="brand-mark">
                        <img src={logoUniversal} alt="" aria-hidden="true"/>
                    </div>
                    <div>
                        <strong>企智盒</strong>
                        <span>Docker 启动器</span>
                    </div>
                </div>
                <nav aria-label="主导航">
                    {nav.map((item) => {
                        const Icon = item.icon;
                        return (
                            <button key={item.id} className={page === item.id ? 'active' : ''} aria-current={page === item.id ? 'page' : undefined} onClick={() => navigatePage(item.id)}>
                                <Icon size={18}/>
                                {item.label}
                            </button>
                        );
                    })}
                </nav>
                <div className="root-path">
                    <span>实例目录</span>
                    <code>{state?.instanceRoot || '...'}</code>
                </div>
            </aside>

            <main id="main-content" className={`workspace ${page === 'assistants' && assistantSkillsMode ? 'skills-workspace-mode' : ''}`}>
                <header className="topbar">
                    <div>
                        <h1>{titleFor(page)}</h1>
                    </div>
                    <div className="topbar-actions">
                        {(page === 'operations' || page === 'settings') && state?.profiles?.profiles && (
                            <label className="profile-picker">
                                <span>当前助手</span>
                                <select value={state.activeProfile || 'default'} onChange={(event) => selectProfile(event.target.value)} disabled={!!blockingBusy}>
                                    {state.profiles.profiles.map((profile) => <option key={profile.id} value={profile.id}>{profile.name || profile.id}</option>)}
                                </select>
                            </label>
                        )}
                        {showContainerStatus && (
                            <div className={`status-pill ${statusClass}`}>
                                {state?.containerStatus === 'running' ? <CheckCircle2 size={16}/> : <CircleAlert size={16}/>}
                                {containerStatusText(state?.containerStatus)}
                            </div>
                        )}
                        <button className="ghost topbar-action-button" onClick={refreshStatus} disabled={statusRefreshing} aria-busy={statusRefreshing}>
                            <RefreshCcw size={15} className={statusRefreshing ? 'spin' : undefined}/>{statusRefreshing ? '刷新中' : '刷新状态'}
                        </button>
                        <button className="ghost topbar-action-button" onClick={() => updates.check(true)} disabled={updates.busy}>
                            <RefreshCcw size={15} className={updates.busy ? 'spin' : undefined}/>{updates.busy ? (updates.progress || '处理中') : '检查更新'}
                        </button>
                    </div>
                </header>
                {notice && <div className={`notice ${notice.type}`}>{notice.message}</div>}
                {showUpdateBanner && updateInfo && (
                    <div className="update-banner">
                        <div>
                            <strong>发现新版本 v{updateInfo.latestVersion}</strong>
                            <span>当前版本 v{updateInfo.currentVersion}，{updateBannerDetail}</span>
                        </div>
                        <div className="update-actions">
                            <button className="primary" onClick={() => {
                                if (window.confirm(`即将升级到 v${updateInfo.latestVersion}。升级期间启动器和 Web 管理会暂时不可用，Hermes Agent 容器不会停止。`)) updates.install();
                            }} disabled={updates.busy || !!blockingBusy || !updateInfo.assetUrl}><Download size={15}/>{updates.busy ? (updates.progress || '正在更新') : '立即更新'}</button>
                            <button onClick={updates.dismiss}>忽略</button>
                        </div>
                    </div>
                )}
                {unsavedChanges && (
                    <div className="dirty-banner">
                        <span>当前有未保存修改，请回到对应页面保存或放弃修改后再切换助手。</span>
                    </div>
                )}
                {state?.applyConfig?.active && (
                    <div className={`apply-config-banner ${state.applyConfig.state === 'slow' ? 'slow' : ''}`}>
                        <div>
                            <strong>{state.applyConfig.message || '正在应用配置'}</strong>
                            <span>{state.applyConfig.strategy === 'recreate' ? '正在重建容器' : state.applyConfig.strategy === 'dufs-only' ? '正在更新文件管理' : '正在更新运行配置'}</span>
                        </div>
                        <button className="ghost" onClick={() => openOperations('diagnostics')}>查看日志</button>
                    </div>
                )}
                {!state && refreshError && (
                    <div className="panel startup-error">
                        <p className="eyebrow">启动器状态读取失败</p>
                        <h2>暂时无法加载 Hermes Dock</h2>
                        <p>{refreshError}</p>
                        <p>下一步：请从桌面应用打开，或确认 Web 管理服务正在运行。</p>
                        <button className="primary no-margin" onClick={() => refresh()} disabled={!!busy}>重试加载</button>
                    </div>
                )}
                {showRebuildBanner && (
                    <div className="rebuild-banner">
                        <span>配置已保存，应用后生效。</span>
                        <button onClick={startApplyConfiguration} disabled={!!blockingBusy}>
                            <RotateCcw size={16}/>应用配置
                        </button>
                    </div>
                )}

                {page === 'overview' && state && (
                    <OverviewPage
                        state={state}
                        needsRebuild={needsRebuild}
                        busy={blockingBusy}
                        lastOperationError={lastOperationError}
                        logs={logs}
                        onStart={() => run('正在启动', StartHermes)}
                        onStop={() => run('正在停止', StopHermes)}
                        onRebuild={startApplyConfiguration}
                        onOpenAssistants={() => navigatePage('assistants')}
                        onOpenLogs={() => {
                            setOperationsTab('diagnostics');
                            setPage('operations');
                        }}
                        onOpenSettings={() => navigatePage('settings')}
                        onOpenAccessSettings={() => {
                            setOperationsTab('access');
                            setPage('settings');
                        }}
                        onOpenWebManagement={openWebManagement}
                        onOpenFileManagement={openFileManagement}
                    />
                )}

                {page === 'assistants' && state && (
                    <AssistantsPage
                        state={state}
                        env={env}
                        setEnv={(value) => {
                            setEnv(value);
                            markPlatformDirty(true);
                        }}
                        providers={providers}
                        setProviders={(value) => {
                            setProviders(value);
                            markModelDirty(true);
                            setModelTestStatus('');
                        }}
                        selectedProvider={selectedProvider}
                        setSelectedProvider={(value) => {
                            setSelectedProvider(value);
                            setModelListStatus('');
                            setModelTestStatus('');
                        }}
                        model={model}
                        setModel={(value) => {
                            setModel(value);
                            markModelDirty(true);
                            setModelTestStatus('');
                        }}
                        modelOptions={visibleModelOptions}
                        modelListStatus={modelListStatus}
                        modelTestStatus={modelTestStatus}
                        selectedAux={selectedAux}
                        setSelectedAux={setSelectedAux}
                        auxModelOptions={auxModelOptions}
                        auxModelListStatus={auxModelListStatus}
                        busy={!!blockingBusy}
                        showApiKey={showApiKey}
                        setShowApiKey={setShowApiKey}
                        newProfileID={newProfileID}
                        setNewProfileID={setNewProfileID}
                        newProfileName={newProfileName}
                        setNewProfileName={setNewProfileName}
                        newProfileCopyMode={newProfileCopyMode}
                        setNewProfileCopyMode={setNewProfileCopyMode}
                        newProfileEnabled={newProfileEnabled}
                        setNewProfileEnabled={setNewProfileEnabled}
                        wizardStep={wizardStep}
                        setWizardStep={setWizardStep}
                        soulContent={soulContent}
                        setSoulContent={setSoulContent}
                        soulStatus={soulStatus}
                        soulDirty={soulDirty}
						setSoulDirty={(value) => {
							if (value) soulEditRevisionRef.current++;
							setSoulDirty(value);
						}}
                        qrData={qrData}
                        qrStatus={qrStatus}
						qrPlatform={qrPlatform}
                        modelDirty={modelDirty}
                        platformDirty={platformDirty}
                        selectedPlatform={selectedPlatform}
                        setSelectedPlatform={selectPlatform}
                        needsRebuild={needsRebuild}
                        hasPlatformBinding={!!weixinBound || !!wecomBound || !!feishuBound || !!dingtalkBound}
                        skillsState={skills.skillsState}
                        skillDetail={skills.skillDetail}
                        skillsStatus={skills.skillsStatus}
                        skillHubState={skills.skillHubState}
                        skillHubDetail={skills.skillHubDetail}
                        skillHubStatus={skills.skillHubStatus}
                        onSelect={selectProfile}
                        onCreate={createProfile}
                        onRename={(id, name) => run('正在更新助手', () => UpdateProfileName(id, name))}
                        onEnabled={(id, enabled) => run(enabled ? '正在启用助手' : '正在停用助手', () => SetProfileEnabled(id, enabled), {rebuildRequired: true})}
                        onMove={(id, direction) => run('正在调整顺序', () => MoveProfile(id, direction))}
                        onDelete={deleteProfile}
                        onSaveModelService={saveModelService}
                        onFetchModels={fetchModels}
                        onFetchProviderModels={fetchProviderModels}
                        onFetchAuxModels={fetchAuxModels}
                        onTestModel={testCurrentModel}
                        onSaveSoul={saveSoulFile}
                        onDiscardSoul={() => loadSoulFile(state?.activeProfile || 'default')}
                        onRestoreDefaultSoul={restoreDefaultSoul}
                        onWeixinLogin={startWeixinLogin}
                        onCancelWeixin={cancelWeixinLogin}
						onFeishuLogin={startFeishuLogin}
						onCancelFeishu={cancelFeishuLogin}
						onDingTalkLogin={startDingTalkLogin}
						onCancelDingTalk={cancelDingTalkLogin}
                        onSaveWeCom={saveWeComConfig}
                        onSaveFeishu={saveFeishuConfig}
						onSaveDingTalk={saveDingTalkConfig}
                        onUnbindPlatform={unbindPlatform}
                        onSaveCurrentPlatform={saveCurrentPlatform}
                        onFinishSetup={finishProfileSetup}
                        onRebuild={startApplyConfiguration}
                        onOpenOperations={openOperations}
                        onRefreshSkills={skills.loadSkills}
                        onSyncBundledSkills={skills.syncBundledSkills}
                        onRestoreDefaultSkills={skills.restoreDefaultSkills}
                        onSkillDetail={skills.loadSkillDetail}
                        onDeleteSkill={skills.deleteSkill}
                        onBatchDeleteSkills={skills.batchDeleteSkills}
                        onOpenSkillDirectory={skills.openSkillDirectory}
                        onSearchSkillHub={skills.loadSkillHubSkills}
                        onSkillHubDetail={skills.loadSkillHubDetail}
                        onInstallSkillHubSkill={skills.installSkillHubSkill}
                        onSkillsModeChange={setAssistantSkillsMode}
                        onBatchCopyProfiles={batchCopyProfiles}
                        onSyncBundledContent={syncBundledContent}
                    />
                )}

                {(page === 'operations' || page === 'settings') && state && compose && proxy && (
                    <OperationsPage
                        scope={page === 'settings' ? 'settings' : 'runtime'}
                        tab={operationsTab}
                        setTab={setOperationsTab}
                        state={state}
                        compose={compose}
                        proxy={proxy}
                        setCompose={(value) => {
                            setCompose(value);
                            markDeployDirty(true);
                        }}
                        setProxy={(value) => {
                            setProxy(value);
                            markDeployDirty(true);
                        }}
                        deployDirty={deployDirty}
                        needsRebuild={needsRebuild}
                        busy={blockingBusy}
                        logs={logs}
                        activeProfileName={state.profiles?.profiles?.find((profile) => profile.id === state.activeProfile)?.name || state.activeProfile || 'default'}
                        weixinBound={!!weixinBound}
                        wecomBound={!!wecomBound}
                        feishuBound={!!feishuBound}
						dingtalkBound={!!dingtalkBound}
						homeChannels={homeChannels}
                        advancedOptions={advancedFileOptions(state.activeProfile || 'default')}
                        advancedPath={advancedPath}
                        setAdvancedPath={changeAdvancedPath}
                        advancedOpen={advancedOpen}
                        setAdvancedOpen={setAdvancedOpen}
                        advancedContent={advancedContent}
                        setAdvancedContent={(value) => {
							advancedEditRevisionRef.current++;
                            setAdvancedContent(value);
                            setAdvancedDirty(true);
                        }}
                        advancedStatus={advancedStatus}
                        advancedDirty={advancedDirty}
                        webRuntime={webRuntime}
                        backupStatus={backupStatus}
                        backupManifest={backupManifest}
                        autoScrollLogs={autoScrollLogs}
                        setAutoScrollLogs={setAutoScrollLogs}
                        logRef={logRef}
                        logsFollowing={logsFollowing}
                        lastOperationError={lastOperationError}
                        onStart={() => run('正在启动', StartHermes)}
                        onStop={() => run('正在停止', StopHermes)}
                        onRestart={() => run('正在重启', RestartHermes)}
                        onRebuild={startApplyConfiguration}
                        onLogs={tailLogs}
                        onClearLogs={() => setLogs([])}
                        onCopyLogs={copyLogs}
                        onOpenFileManagement={openFileManagement}
                        onOpenAssistantPlatforms={() => {
                            setPage('assistants');
                            setWizardStep('platforms');
                        }}
                        onSaveDeploy={saveDeploySettings}
                        onDiscardDeploy={discardDeployChanges}
                        onRefreshChannels={() => run('正在刷新通道', refresh)}
                        channelActionStatus={channelActionStatus}
                        onHomeChannel={setHomeChannel}
                        onTestChannel={testChannel}
                        onSaveAdvanced={saveAdvancedFile}
                        onExportBackup={exportInstanceBackup}
                        onInspectBackup={inspectInstanceBackup}
                        onImportBackup={importInstanceBackup}
						onClearBackupManifest={() => setBackupManifest(null)}
                        onFactoryReset={factoryReset}
                        resetConfirmPhrase={factoryResetPhrase}
                        webStatus={state.web}
						onSaveWebSettings={saveWebManagementSettings}
                        onChangeWebPassword={(oldPassword, newPassword) => run('正在修改 Web 访问密码', () => ChangeWebPassword(oldPassword, newPassword))}
                        onResetWebPassword={() => run('正在重置 Web 访问密码', ResetWebPassword)}
                        updateInfo={updates.info}
                        updateStatus={state.update}
                        updateBusy={updates.busy || !!blockingBusy}
                        updateProgress={updates.progress}
                        onCheckUpdate={() => updates.check(true)}
                        onInstallUpdate={() => {
                            if (!updates.info) return;
                            if (window.confirm(`即将升级到 v${updates.info.latestVersion}。升级期间启动器和 Web 管理会暂时不可用，Hermes Agent 容器不会停止。`)) updates.install();
                        }}
                        onSetAutoUpdate={updates.setAutoUpdate}
                    />
                )}
            </main>
        </div>
    );
}

export default App;
