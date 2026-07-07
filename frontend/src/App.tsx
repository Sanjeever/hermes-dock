import {useEffect, useRef, useState} from 'react';
import {CheckCircle2, CircleAlert, Clipboard, ExternalLink, RefreshCcw, RotateCcw} from 'lucide-react';
import './App.css';
import logoUniversal from './assets/images/logo-universal.png';
import {
    CancelWeixinLogin,
    CheckForUpdates,
    CompleteProfileSetup,
    CreateProfile,
    DeleteProfile,
    DismissUpdate,
    DeleteSkill,
    FactoryResetInstance,
    FetchProviderConfigModelList,
    GetAppState,
    GetSkillDetail,
    GetSkillHubDetail,
    InstallSkillHubSkill,
    ListProfileSkills,
    ListSkillHubSkills,
    OpenEndpoint,
    OpenUpdateURL,
    OpenSkillDirectory,
    ReadTextFile,
    RebuildHermes,
    RestartHermes,
    SaveComposeSettings,
    SaveFeishuConfig,
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
    StopHermes,
    StopTailLogs,
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
import {factoryResetPhrase, fallbackProviderConfig, nav} from './constants';
import type {AppState, ComposeSettings, EnvVar, ModelConfig, ModelOption, Notice, OperationsTab, Page, PlatformKey, ProviderConfig, RunOptions, SkillDetail, SkillHubDetail, SkillHubQuery, SkillHubState, SkillsState, UpdateInfo, WizardStep} from './types';
import {advancedFileOptions, containerStatusText, defaultAdvancedPath, doneLabel, envValue, firstProviderID, modelOptionKey, profileFilePath, titleFor, toPlainModelConfig, toPlainProviderConfig, webAdvancedFileOptions} from './utils';

function App() {
    const webRuntime = isWebRuntime();
    const [page, setPage] = useState<Page>('assistants');
    const [operationsTab, setOperationsTab] = useState<OperationsTab>('status');
    const [wizardStep, setWizardStep] = useState<WizardStep | null>(null);
    const [state, setState] = useState<AppState | null>(null);
    const stateRef = useRef<AppState | null>(null);
    const activeProfileRef = useRef('default');
    const [env, setEnv] = useState<EnvVar[]>([]);
    const [compose, setCompose] = useState<ComposeSettings | null>(null);
    const [model, setModel] = useState<ModelConfig | null>(null);
    const [logs, setLogs] = useState<string[]>([]);
    const logRef = useRef<HTMLPreElement>(null!);
    const [logsFollowing, setLogsFollowing] = useState(false);
    const [busy, setBusy] = useState('');
    const [notice, setNotice] = useState<Notice | null>(null);
    const [refreshError, setRefreshError] = useState('');
    const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null);
    const [updateBusy, setUpdateBusy] = useState(false);
    const updateCheckedRef = useRef(false);
    const [needsRebuild, setNeedsRebuild] = useState(false);
    const [qrData, setQrData] = useState('');
    const [qrStatus, setQrStatus] = useState('');
    const [advancedPath, setAdvancedPath] = useState('data/config.yaml');
    const [advancedContent, setAdvancedContent] = useState('');
    const [advancedStatus, setAdvancedStatus] = useState('');
    const [advancedDirty, setAdvancedDirty] = useState(false);
    const [advancedOpen, setAdvancedOpen] = useState(false);
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
    const [weixinLoginProfile, setWeixinLoginProfile] = useState('');
    const [channelActionStatus, setChannelActionStatus] = useState<Record<string, string>>({});
    const [lastOperationError, setLastOperationError] = useState('');
    const [skillsState, setSkillsState] = useState<SkillsState | null>(null);
    const [skillDetail, setSkillDetail] = useState<SkillDetail | null>(null);
    const [skillsStatus, setSkillsStatus] = useState('');
    const [skillHubState, setSkillHubState] = useState<SkillHubState | null>(null);
    const [skillHubDetail, setSkillHubDetail] = useState<SkillHubDetail | null>(null);
    const [skillHubStatus, setSkillHubStatus] = useState('');
    const [assistantSkillsMode, setAssistantSkillsMode] = useState(false);
    const dirtyMessage = '当前有未保存修改，请先保存或放弃修改后再切换';

    useEffect(() => {
        refresh();
        const offDocker = EventsOn('docker:progress', (event: { line?: string; done?: boolean; code?: number }) => {
            if (event.line) appendLog(event.line);
            if (event.done) {
                appendLog(`命令退出，代码 ${event.code}`);
                setBusy('');
                refresh();
            }
        });
        const offLogs = EventsOn('logs:line', (event: { line?: string }) => event.line && appendLog(event.line));
        const isCurrentProfileEvent = (event: { profile_id?: string }) => !event.profile_id || event.profile_id === activeProfileRef.current;
        const offQR = EventsOn('weixin-login:qr', (event: { scan_data: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) return;
            setWeixinLoginProfile(event.profile_id || activeProfileRef.current);
            setQrData(event.scan_data);
            setQrStatus('等待微信扫码');
        });
        const offQRStatus = EventsOn('weixin-login:status', (event: { status?: string; message?: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) return;
            setQrStatus(event.message || event.status || '');
        });
        const offQRDone = EventsOn('weixin-login:confirmed', (event: { account_id: string; user_id: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) {
                setWeixinLoginProfile((current) => current === event.profile_id ? '' : current);
                setNeedsRebuild(true);
                const profile = stateRef.current?.profiles?.profiles?.find((item) => item.id === event.profile_id);
                setNotice({type: 'ok', message: `${profile?.name || event.profile_id || '其他助手'} 的微信已绑定成功，重建后生效`});
                refresh();
                return;
            }
            setQrStatus(`绑定成功 ${event.user_id || event.account_id}`);
            setQrData('');
            setWeixinLoginProfile('');
            setNeedsRebuild(true);
            refresh();
        });
        const offQRError = EventsOn('weixin-login:error', (event: { message: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) return;
            setQrStatus(event.message);
            setWeixinLoginProfile('');
        });
        return () => {
            offDocker();
            offLogs();
            offQR();
            offQRStatus();
            offQRDone();
            offQRError();
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
        if (page !== 'operations' || operationsTab !== 'advanced') return;
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
        if (!webRuntime || advancedDirty) return;
        const options = webAdvancedFileOptions(state?.activeProfile || 'default');
        if (!options.some((option) => option.value === advancedPath)) {
            setAdvancedPath(options[0].value);
            setAdvancedOpen(false);
        }
    }, [webRuntime, state?.activeProfile, advancedPath, advancedDirty]);

    useEffect(() => {
        if (!state?.activeProfile) return;
        setSkillDetail(null);
        setSkillHubDetail(null);
        setSkillHubState(null);
        setSkillHubStatus('');
        loadSkills();
    }, [state?.activeProfile]);

    useEffect(() => {
        activeProfileRef.current = state?.activeProfile || 'default';
        setQrData('');
        setQrStatus('');
    }, [state?.activeProfile]);

    useEffect(() => {
        const profile = state?.profiles?.profiles?.find((item) => item.id === state.activeProfile);
        if (!profile) return;
        if (!profile.setupCompletedAt && !wizardStep) {
            setWizardStep('model');
        }
    }, [state?.activeProfile, state?.profiles?.profiles, wizardStep]);

    useEffect(() => {
        if (!state || busy || state.containerStatus !== 'running') return;
        const hasStartingProfile = Object.values(state.profileStatus?.profiles || {}).some((status) => status.state === 'starting');
        if (!hasStartingProfile) return;
        const timer = window.setTimeout(() => {
            refresh();
        }, 1500);
        return () => window.clearTimeout(timer);
    }, [state, busy]);

    useEffect(() => {
        if (!state?.appVersion || updateCheckedRef.current) return;
        updateCheckedRef.current = true;
        checkForUpdates(false);
    }, [state?.appVersion]);

    async function refresh() {
        let next: unknown;
        try {
            next = await GetAppState();
        } catch (error) {
            const message = String(error);
            setRefreshError(message);
            appendLog(message);
            return message;
        }
        const nextState = next as AppState;
        const firstRefresh = !stateRef.current;
        const previousProfile = stateRef.current?.activeProfile;
        const profileChanged = !!previousProfile && previousProfile !== nextState.activeProfile;
        stateRef.current = nextState;
        activeProfileRef.current = nextState.activeProfile || 'default';
        setState(nextState);
        const nextEnv = nextState.environment || [];
        if (!platformDirtyRef.current) {
            setEnv(nextEnv);
        }
        if (!deployDirtyRef.current) {
            setCompose(nextState.compose);
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
        setRefreshError('');
        return '';
    }

    async function selectProfile(id: string) {
        if (id === state?.activeProfile) return true;
        if (weixinLoginProfile && weixinLoginProfile !== id) {
            setNotice({type: 'error', message: '微信扫码登录进行中，请先完成或取消后再切换助手'});
            return false;
        }
        if (hasUnsavedChanges()) {
            setNotice({type: 'error', message: dirtyMessage});
            return false;
        }
        const target = state?.profiles?.profiles?.find((profile) => profile.id === id);
        return await run('正在切换 Profile', () => SelectProfile(id), {
            beforeRefresh: () => {
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
        return await run('正在创建 Profile', () => CreateProfile({
            id: newProfileID,
            name: newProfileName,
            enabled: newProfileEnabled,
            copyFrom: state?.activeProfile || 'default',
            copyMode: newProfileCopyMode,
        }), {
            beforeRefresh: () => {
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
        return await run('正在删除 Profile', () => DeleteProfile(id, id), {rebuildRequired: true});
    }

    async function loadSkills() {
        setSkillsStatus('正在读取技能');
        try {
            const next = await ListProfileSkills();
            setSkillsState(next as SkillsState);
            setSkillsStatus('');
        } catch (error) {
            const message = String(error);
            setSkillsStatus(message);
            appendLog(message);
        }
    }

    async function loadSkillDetail(path: string) {
        setSkillsStatus('正在读取技能详情');
        try {
            const next = await GetSkillDetail(path);
            setSkillDetail(next as SkillDetail);
            setSkillsStatus('');
        } catch (error) {
            const message = String(error);
            setSkillDetail(null);
            setSkillsStatus(message);
            appendLog(message);
        }
    }

    async function deleteSkill(path: string) {
        const ok = await run('正在删除技能', () => DeleteSkill(path), {rebuildRequired: true});
        if (!ok) return false;
        setSkillDetail(null);
        await loadSkills();
        setNotice({type: 'ok', message: '已删除技能并创建备份，重建后生效'});
        return true;
    }

    async function openSkillDirectory(path: string) {
        try {
            await OpenSkillDirectory(path);
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
        }
    }

    async function loadSkillHubSkills(query: SkillHubQuery) {
        setSkillHubStatus('正在读取技能中心');
        try {
            const next = await ListSkillHubSkills(query);
            setSkillHubState(next as SkillHubState);
            setSkillHubStatus('');
        } catch (error) {
            const message = String(error);
            setSkillHubStatus(message);
            appendLog(message);
        }
    }

    async function loadSkillHubDetail(slug: string) {
        setSkillHubStatus('正在读取技能详情');
        try {
            const next = await GetSkillHubDetail(slug);
            setSkillHubDetail(next as SkillHubDetail);
            setSkillHubStatus('');
        } catch (error) {
            const message = String(error);
            setSkillHubDetail(null);
            setSkillHubStatus(message);
            appendLog(message);
        }
    }

    async function installSkillHubSkill(slug: string) {
        const ok = await run('正在安装技能', () => InstallSkillHubSkill(slug), {rebuildRequired: true});
        if (!ok) return false;
        await loadSkills();
        await loadSkillHubDetail(slug);
        setNotice({type: 'ok', message: '已安装技能，重建后生效'});
        return true;
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
            const items = await FetchProviderConfigModelList(provider);
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
            const items = await FetchProviderConfigModelList(provider);
            setAuxModelOptions((current) => ({...current, [optionsKey]: items as ModelOption[]}));
            setAuxModelListStatus(`已拉取 ${(items as ModelOption[]).length} 个模型`);
        } catch (error) {
            setAuxModelOptions((current) => ({...current, [optionsKey]: []}));
            setAuxModelListStatus(String(error));
            appendLog(String(error));
        }
    }

    async function run(label: string, action: () => Promise<unknown>, options: RunOptions = {}) {
        setBusy(label);
        setNotice({type: 'info', message: label});
        setLastOperationError('');
        try {
            await action();
            options.beforeRefresh?.();
            const refreshMessage = await refresh();
            if (options.rebuildRequired) {
                setNeedsRebuild(true);
            }
            options.afterSuccess?.();
            if (refreshMessage) {
                const message = `${doneLabel(label)}，但刷新状态失败：${refreshMessage}`;
                setNotice({type: 'error', message});
                setLastOperationError(message);
                return true;
            }
            setNotice({type: 'ok', message: doneLabel(label)});
            return true;
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
            setLastOperationError(message);
            return false;
        } finally {
            setBusy('');
        }
    }

    async function checkForUpdates(force: boolean) {
        setUpdateBusy(true);
        try {
            const info = await CheckForUpdates(force);
            setUpdateInfo(info);
            if (force) {
                if (info.available && !info.dismissed) {
                    setNotice({type: 'ok', message: `发现新版本 v${info.latestVersion}`});
                } else if (info.available && info.dismissed) {
                    setNotice({type: 'info', message: `v${info.latestVersion} 已忽略`});
                } else {
                    setNotice({type: 'ok', message: '当前已是最新版本'});
                }
            }
        } catch (error) {
            const message = String(error);
            if (force) {
                appendLog(message);
                setNotice({type: 'error', message});
            }
        } finally {
            setUpdateBusy(false);
        }
    }

    async function dismissUpdate() {
        if (!updateInfo?.latestVersion) return;
        try {
            await DismissUpdate(updateInfo.latestVersion);
            setUpdateInfo({...updateInfo, dismissed: true});
            setNotice({type: 'ok', message: `已忽略 v${updateInfo.latestVersion}`});
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
        }
    }

    async function openUpdateURL(url: string) {
        if (!url) return;
        try {
            await OpenUpdateURL(url);
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
        }
    }

    async function copyUpdateCommand(url: string) {
        if (!updateInfo?.assetName || !url) return;
        const command = `curl -L -o ${updateInfo.assetName} ${url}\nsudo apt install -y ./${updateInfo.assetName}`;
        try {
            await navigator.clipboard.writeText(command);
            setNotice({type: 'ok', message: '已复制安装命令'});
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
        }
    }

    async function loadAdvancedFile(path: string) {
        setAdvancedStatus('正在读取文件');
        setAdvancedContent('');
        try {
            setAdvancedContent(await ReadTextFile(path));
            setAdvancedStatus(`已加载 ${path}`);
            setAdvancedDirty(false);
        } catch (error) {
            setAdvancedContent('');
            setAdvancedStatus(String(error));
            setAdvancedDirty(false);
            appendLog(String(error));
        }
    }

    async function saveAdvancedFile() {
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
            await SaveTextFile({path: advancedPath, content: advancedContent, reason: 'before-advanced-save', confirm});
            setAdvancedDirty(false);
            setNeedsRebuild(true);
            const refreshMessage = await refresh();
            if (refreshMessage) {
                setAdvancedStatus(`已保存 ${advancedPath}，但刷新状态失败`);
                setNotice({type: 'error', message: `已保存文件，但刷新状态失败：${refreshMessage}`});
                return;
            }
            setAdvancedStatus(`已保存 ${advancedPath}，应用并重建后生效`);
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
        const path = profileFilePath(profileID, 'SOUL.md');
        setSoulStatus('正在读取人格文件');
        try {
            setSoulContent(await ReadTextFile(path));
            setSoulStatus(`已加载 ${path}`);
            setSoulDirty(false);
        } catch (error) {
            setSoulContent('');
            setSoulStatus(String(error));
            setSoulDirty(false);
            appendLog(String(error));
        }
    }

    async function saveSoulFile() {
        const path = profileFilePath(state?.activeProfile || 'default', 'SOUL.md');
        setBusy('正在保存人格文件');
        setNotice({type: 'info', message: '正在保存人格文件'});
        setSoulStatus('正在保存人格文件');
        try {
            await SaveTextFile({path, content: soulContent, reason: 'before-soul-save'});
            setSoulDirty(false);
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

    async function saveModelService() {
        if (!model) return false;
        return await run('正在保存模型服务', async () => {
            await SaveProviderConfig(toPlainProviderConfig(providers));
            await SaveModelConfig(toPlainModelConfig(model));
        }, {rebuildRequired: true, beforeRefresh: () => markModelDirty(false)});
    }

    async function saveCurrentPlatform() {
        if (!platformDirty) return true;
        if (selectedPlatform === 'wecom') {
            return await saveWeComConfig();
        }
        if (selectedPlatform === 'feishu') {
            return await saveFeishuConfig();
        }
        markPlatformDirty(false);
        return true;
    }

    async function finishProfileSetup(apply: boolean) {
        const profileID = state?.activeProfile || 'default';
        return await run(apply ? '正在完成并重建' : '正在完成配置', async () => {
            await CompleteProfileSetup(profileID);
            if (apply) {
                await RebuildHermes();
            }
        }, {
            afterSuccess: () => {
                if (apply) setNeedsRebuild(false);
                setWizardStep(null);
            },
        });
    }

    function hasUnsavedChanges() {
        return advancedDirty || soulDirty || modelDirty || platformDirty || deployDirty;
    }

    function markModelDirty(value: boolean) {
        modelDirtyRef.current = value;
        setModelDirty(value);
    }

    function markPlatformDirty(value: boolean) {
        platformDirtyRef.current = value;
        setPlatformDirty(value);
    }

    function markDeployDirty(value: boolean) {
        deployDirtyRef.current = value;
        setDeployDirty(value);
    }

    async function startWeixinLogin() {
        const profileID = state?.activeProfile || 'default';
        setWeixinLoginProfile(profileID);
        const ok = await run('正在启动微信扫码登录', StartWeixinLogin);
        if (!ok) {
            setWeixinLoginProfile('');
        }
    }

    async function cancelWeixinLogin() {
        const ok = await run('正在取消微信扫码登录', CancelWeixinLogin);
        if (!ok) return;
        setQrData('');
        setQrStatus('');
        setWeixinLoginProfile('');
    }

    async function saveWeComConfig() {
        if (envValue(env, 'WECOM_BOT_ID').trim() === '' || envValue(env, 'WECOM_SECRET').trim() === '') {
            const message = '请填写企业微信 Bot ID 和 Secret 后再保存';
            setNotice({type: 'error', message});
            setLastOperationError(message);
            return false;
        }
        return await run('正在保存企业微信配置', () => SaveWeComConfig({
            botId: envValue(env, 'WECOM_BOT_ID'),
            secret: envValue(env, 'WECOM_SECRET'),
            websocketUrl: envValue(env, 'WECOM_WEBSOCKET_URL'),
            dmPolicy: closedPolicyValue(envValue(env, 'WECOM_DM_POLICY')),
            allowedUsers: '',
            groupPolicy: closedPolicyValue(envValue(env, 'WECOM_GROUP_POLICY')),
            groupAllowUsers: '',
        }), {rebuildRequired: true, beforeRefresh: () => markPlatformDirty(false)});
    }

    async function saveFeishuConfig() {
        if (envValue(env, 'FEISHU_APP_ID').trim() === '' || envValue(env, 'FEISHU_APP_SECRET').trim() === '') {
            const message = '请填写飞书 App ID 和 App Secret 后再保存';
            setNotice({type: 'error', message});
            setLastOperationError(message);
            return false;
        }
        return await run('正在保存飞书配置', () => SaveFeishuConfig({
            appId: envValue(env, 'FEISHU_APP_ID'),
            appSecret: envValue(env, 'FEISHU_APP_SECRET'),
            domain: envValue(env, 'FEISHU_DOMAIN') || 'feishu',
            allowedUsers: '',
            groupPolicy: disabledPolicyValue(envValue(env, 'FEISHU_GROUP_POLICY')),
        }), {rebuildRequired: true, beforeRefresh: () => markPlatformDirty(false)});
    }

    async function unbindPlatform(platform: PlatformKey) {
        const label = platformLabel(platform);
        if (!window.confirm(`确定取消绑定${label}？这会清空当前助手的绑定密钥，保存后需要应用并重建才会影响运行中的容器。`)) return;
        await run(`正在取消绑定${label}`, () => UnbindPlatform(platform), {
            rebuildRequired: true,
            beforeRefresh: () => markPlatformDirty(false),
            afterSuccess: () => {
                if (platform === 'weixin') {
                    setQrData('');
                    setQrStatus('');
                    setWeixinLoginProfile('');
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
        markDeployDirty(false);
        setNotice({type: 'ok', message: '已放弃部署参数修改'});
    }

    function openOperations(tab: OperationsTab) {
        setOperationsTab(tab);
        setPage('operations');
    }

    async function factoryReset() {
        if (webRuntime) {
            setNotice({type: 'error', message: 'Web 管理不提供恢复出厂设置'});
            return;
        }
        await run('正在恢复出厂设置', FactoryResetInstance, {
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

    function changeAdvancedPath(path: string) {
        if (path === advancedPath) return;
        if (advancedDirty) {
            setNotice({type: 'error', message: '当前文件有未保存修改，请先保存后再切换文件'});
            return;
        }
        setAdvancedDirty(false);
        setAdvancedPath(path);
    }

    async function openEndpoint(endpoint: 'dashboard' | 'gateway') {
        try {
            await OpenEndpoint(endpoint);
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
                await SaveProviderConfig(toPlainProviderConfig(providers));
                await SaveModelConfig(toPlainModelConfig(model));
                markModelDirty(false);
                setNeedsRebuild(true);
            }
            await TestModel();
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
        const ok = await run('正在设置默认通道', () => SetHomeChannel(platform, id), {rebuildRequired: true});
        setChannelActionStatus((current) => ({...current, [key]: ok ? '已设为默认，应用并重建后生效' : '设置默认通道失败'}));
    }

    async function testChannel(platform: string, id: string) {
        const key = channelStatusKey(platform, id, 'test');
        const homeKey = channelStatusKey(platform, id, 'home');
        setChannelActionStatus((current) => {
            const next = {...current, [key]: '正在发送测试消息'};
            delete next[homeKey];
            return next;
        });
        const ok = await run('正在发送测试消息', () => SendTestMessage(platform, id, '企智盒测试消息'));
        setChannelActionStatus((current) => ({...current, [key]: ok ? '测试消息已发送' : '测试消息发送失败'}));
    }

    const statusClass = state?.containerStatus === 'running' ? 'ok' : 'warn';
    const weixinBound = envValue(env, 'WEIXIN_ACCOUNT_ID') && envValue(env, 'WEIXIN_TOKEN');
    const wecomBound = envValue(env, 'WECOM_BOT_ID') && envValue(env, 'WECOM_SECRET');
    const feishuBound = envValue(env, 'FEISHU_APP_ID') && envValue(env, 'FEISHU_APP_SECRET');
    const weixinHomeChannel = envValue(env, 'WEIXIN_HOME_CHANNEL');
    const currentProviderKey = model ? model.provider : '';
    const currentModelOptionsKey = model ? modelOptionKey(currentProviderKey) : '';
    const visibleModelOptions = currentModelOptionsKey === modelOptionsKey ? modelOptions : [];
    const activeProfile = state?.profiles?.profiles?.find((profile) => profile.id === state.activeProfile);
    const activeSetupDone = !!activeProfile?.setupCompletedAt;
    const showContainerStatus = page !== 'assistants' || activeSetupDone;
    const showRebuildBanner = needsRebuild && (page !== 'assistants' || activeSetupDone);
    const unsavedChanges = hasUnsavedChanges();
    const showUpdateBanner = !!updateInfo?.available && !updateInfo.dismissed;
    const updateAssetIsDeb = !!updateInfo?.assetName?.endsWith('.deb');
    const updateBannerDetail = updateAssetIsDeb ? 'Debian 13 可下载新的 deb 包后安装。' : '可在发布页下载适合当前系统的安装包。';

    return (
        <div className="shell">
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
                <nav>
                    {nav.map((item) => {
                        const Icon = item.icon;
                        return (
                            <button key={item.id} className={page === item.id ? 'active' : ''} onClick={() => setPage(item.id)}>
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

            <main className={`workspace ${page === 'assistants' && assistantSkillsMode ? 'skills-workspace-mode' : ''}`}>
                <header className="topbar">
                    <div>
                        <h1>{titleFor(page)}</h1>
                    </div>
                    <div className="topbar-actions">
                        {page === 'operations' && state?.profiles?.profiles && (
                            <label className="profile-picker">
                                <span>当前 Profile</span>
                                <select value={state.activeProfile || 'default'} onChange={(event) => selectProfile(event.target.value)} disabled={!!busy}>
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
                        <button className="ghost topbar-update-button" onClick={() => checkForUpdates(true)} disabled={updateBusy}>
                            <RefreshCcw size={15}/>{updateBusy ? '检查中' : '检查更新'}
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
                            {updateInfo.releaseUrl && <button onClick={() => openUpdateURL(updateInfo.releaseUrl)}><ExternalLink size={15}/>发布页</button>}
                            {updateInfo.assetUrl && updateAssetIsDeb && <button onClick={() => copyUpdateCommand(updateInfo.assetUrl)}><Clipboard size={15}/>复制安装命令</button>}
                            {updateInfo.assetUrl && !updateAssetIsDeb && <button onClick={() => openUpdateURL(updateInfo.assetUrl)}><ExternalLink size={15}/>下载安装包</button>}
                            {updateAssetIsDeb && (updateInfo.mirrors || []).map((mirror) => (
                                <button key={mirror.label} onClick={() => copyUpdateCommand(mirror.url)}><Clipboard size={15}/>{mirror.label}</button>
                            ))}
                            {!updateAssetIsDeb && (updateInfo.mirrors || []).map((mirror) => (
                                <button key={mirror.label} onClick={() => openUpdateURL(mirror.url)}><ExternalLink size={15}/>{mirror.label}</button>
                            ))}
                            <button onClick={dismissUpdate}>忽略</button>
                        </div>
                    </div>
                )}
                {unsavedChanges && (
                    <div className="dirty-banner">
                        <span>当前有未保存修改，请回到对应页面保存或放弃修改后再切换助手。</span>
                    </div>
                )}
                {!state && refreshError && (
                    <div className="panel startup-error">
                        <p className="eyebrow">启动器状态读取失败</p>
                        <h2>暂时无法加载 Hermes Dock</h2>
                        <p>{refreshError}</p>
                        <button className="primary no-margin" onClick={() => refresh()} disabled={!!busy}>重试加载</button>
                    </div>
                )}
                {showRebuildBanner && (
                    <div className="rebuild-banner">
                        <span>配置已保存，重建后生效。</span>
                        <button onClick={() => run('正在应用并重建', RebuildHermes, {afterSuccess: () => setNeedsRebuild(false)})} disabled={!!busy}>
                            <RotateCcw size={16}/>应用并重建
                        </button>
                    </div>
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
                        busy={!!busy}
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
                        setSoulDirty={setSoulDirty}
                        qrData={qrData}
                        qrStatus={qrStatus}
                        modelDirty={modelDirty}
                        platformDirty={platformDirty}
                        selectedPlatform={selectedPlatform}
                        setSelectedPlatform={selectPlatform}
                        needsRebuild={needsRebuild}
                        hasPlatformBinding={!!weixinBound || !!wecomBound || !!feishuBound}
                        skillsState={skillsState}
                        skillDetail={skillDetail}
                        skillsStatus={skillsStatus}
                        skillHubState={skillHubState}
                        skillHubDetail={skillHubDetail}
                        skillHubStatus={skillHubStatus}
                        onSelect={selectProfile}
                        onCreate={createProfile}
                        onRename={(id, name) => run('正在更新助手', () => UpdateProfileName(id, name))}
                        onEnabled={(id, enabled) => run(enabled ? '正在启用助手' : '正在停用助手', () => SetProfileEnabled(id, enabled), {rebuildRequired: true})}
                        onMove={(id, direction) => run('正在调整顺序', () => MoveProfile(id, direction))}
                        onDelete={deleteProfile}
                        onSaveModelService={saveModelService}
                        onFetchModels={fetchModels}
                        onFetchAuxModels={fetchAuxModels}
                        onTestModel={testCurrentModel}
                        onSaveSoul={saveSoulFile}
                        onDiscardSoul={() => loadSoulFile(state?.activeProfile || 'default')}
                        onWeixinLogin={startWeixinLogin}
                        onCancelWeixin={cancelWeixinLogin}
                        onSaveWeCom={saveWeComConfig}
                        onSaveFeishu={saveFeishuConfig}
                        onUnbindPlatform={unbindPlatform}
                        onSaveCurrentPlatform={saveCurrentPlatform}
                        onFinishSetup={finishProfileSetup}
                        onRebuild={() => run('正在应用并重建', RebuildHermes, {afterSuccess: () => setNeedsRebuild(false)})}
                        onOpenOperations={openOperations}
                        onRefreshSkills={loadSkills}
                        onSkillDetail={loadSkillDetail}
                        onDeleteSkill={deleteSkill}
                        onOpenSkillDirectory={webRuntime ? async () => undefined : openSkillDirectory}
                        onSearchSkillHub={loadSkillHubSkills}
                        onSkillHubDetail={loadSkillHubDetail}
                        onInstallSkillHubSkill={installSkillHubSkill}
                        onSkillsModeChange={setAssistantSkillsMode}
                    />
                )}

                {page === 'operations' && state && compose && (
                    <OperationsPage
                        tab={operationsTab}
                        setTab={setOperationsTab}
                        state={state}
                        compose={compose}
                        setCompose={(value) => {
                            setCompose(value);
                            markDeployDirty(true);
                        }}
                        deployDirty={deployDirty}
                        needsRebuild={needsRebuild}
                        busy={busy}
                        logs={logs}
                        activeProfileName={state.profiles?.profiles?.find((profile) => profile.id === state.activeProfile)?.name || state.activeProfile || 'default'}
                        weixinBound={!!weixinBound}
                        wecomBound={!!wecomBound}
                        feishuBound={!!feishuBound}
                        weixinHomeChannel={weixinHomeChannel}
                        advancedOptions={webRuntime ? webAdvancedFileOptions(state.activeProfile || 'default') : advancedFileOptions(state.activeProfile || 'default')}
                        advancedPath={advancedPath}
                        setAdvancedPath={changeAdvancedPath}
                        advancedOpen={advancedOpen}
                        setAdvancedOpen={setAdvancedOpen}
                        advancedContent={advancedContent}
                        setAdvancedContent={(value) => {
                            setAdvancedContent(value);
                            setAdvancedDirty(true);
                        }}
                        advancedStatus={advancedStatus}
                        advancedDirty={advancedDirty}
                        autoScrollLogs={autoScrollLogs}
                        setAutoScrollLogs={setAutoScrollLogs}
                        logRef={logRef}
                        logsFollowing={logsFollowing}
                        lastOperationError={lastOperationError}
                        onStart={() => run('正在启动', StartHermes)}
                        onStop={() => run('正在停止', StopHermes)}
                        onRestart={() => run('正在重启', RestartHermes)}
                        onRebuild={() => run('正在应用并重建', RebuildHermes, {afterSuccess: () => setNeedsRebuild(false)})}
                        onLogs={tailLogs}
                        onClearLogs={() => setLogs([])}
                        onCopyLogs={copyLogs}
                        onOpenEndpoint={openEndpoint}
                        onOpenAssistantPlatforms={() => {
                            setPage('assistants');
                            setWizardStep('platforms');
                        }}
                        onSaveDeploy={() => run('正在保存部署配置', () => SaveComposeSettings({...compose, dashboardEnabled: true}), {rebuildRequired: true, beforeRefresh: () => markDeployDirty(false)})}
                        onDiscardDeploy={discardDeployChanges}
                        onRefreshChannels={() => run('正在刷新通道', refresh)}
                        channelActionStatus={channelActionStatus}
                        onHomeChannel={setHomeChannel}
                        onTestChannel={testChannel}
                        onSaveAdvanced={saveAdvancedFile}
                        onFactoryReset={factoryReset}
                        resetConfirmPhrase={factoryResetPhrase}
                        webRuntime={webRuntime}
                        webStatus={state.web}
                        onSaveWebSettings={(settings) => run('正在保存 Web 管理设置', () => SaveWebSettings(settings))}
                        onChangeWebPassword={(oldPassword, newPassword) => run('正在修改 Web 访问密码', () => ChangeWebPassword(oldPassword, newPassword))}
                        onResetWebPassword={() => run('正在重置 Web 访问密码', ResetWebPassword)}
                    />
                )}
            </main>
        </div>
    );
}

function closedPolicyValue(value: string) {
    return value === 'open' || value === '' ? 'open' : 'closed';
}

function disabledPolicyValue(value: string) {
    return value === 'open' || value === '' ? 'open' : 'disabled';
}

function platformLabel(platform: PlatformKey) {
    switch (platform) {
        case 'weixin':
            return '个人微信';
        case 'wecom':
            return '企业微信';
        case 'feishu':
            return '飞书 / Lark';
        default:
            return platform;
    }
}

function firstBoundPlatform(env: EnvVar[]): PlatformKey {
    if (envValue(env, 'WEIXIN_ACCOUNT_ID') && envValue(env, 'WEIXIN_TOKEN')) return 'weixin';
    if (envValue(env, 'WECOM_BOT_ID') && envValue(env, 'WECOM_SECRET')) return 'wecom';
    if (envValue(env, 'FEISHU_APP_ID') && envValue(env, 'FEISHU_APP_SECRET')) return 'feishu';
    return 'weixin';
}

function channelStatusKey(platform: string, id: string, action: string) {
    return `${platform}:${id}:${action}`;
}

export default App;
