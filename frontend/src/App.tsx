import {useEffect, useRef, useState} from 'react';
import {CheckCircle2, CircleAlert, RotateCcw} from 'lucide-react';
import './App.css';
import {
    CancelWeixinLogin,
    CompleteProfileSetup,
    CreateProfile,
    DeleteProfile,
    FactoryResetInstance,
    FetchProviderConfigModelList,
    GetAppState,
    OpenEndpoint,
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
    TailLogs,
    TestModel,
    MoveProfile,
    UpdateProfileName,
} from '../wailsjs/go/main/App';
import {EventsOn} from '../wailsjs/runtime/runtime';
import {AssistantsPage} from './pages/AssistantsPage';
import {OperationsPage} from './pages/OperationsPage';
import {factoryResetPhrase, fallbackProviderConfig, nav} from './constants';
import type {AppState, ComposeSettings, EnvVar, ModelConfig, ModelOption, Notice, OperationsTab, Page, ProviderConfig, RunOptions, WizardStep} from './types';
import {advancedFileOptions, containerStatusText, defaultAdvancedPath, doneLabel, envValue, firstProviderID, modelOptionKey, profileFilePath, titleFor, toPlainModelConfig, toPlainProviderConfig} from './utils';

function App() {
    const [page, setPage] = useState<Page>('assistants');
    const [operationsTab, setOperationsTab] = useState<OperationsTab>('status');
    const [wizardStep, setWizardStep] = useState<WizardStep | null>(null);
    const [state, setState] = useState<AppState | null>(null);
    const activeProfileRef = useRef('default');
    const [env, setEnv] = useState<EnvVar[]>([]);
    const [compose, setCompose] = useState<ComposeSettings | null>(null);
    const [model, setModel] = useState<ModelConfig | null>(null);
    const [logs, setLogs] = useState<string[]>([]);
    const logRef = useRef<HTMLPreElement>(null!);
    const [busy, setBusy] = useState('');
    const [notice, setNotice] = useState<Notice | null>(null);
    const [needsRebuild, setNeedsRebuild] = useState(false);
    const [qrData, setQrData] = useState('');
    const [qrStatus, setQrStatus] = useState('');
    const [advancedPath, setAdvancedPath] = useState('data/config.yaml');
    const [advancedContent, setAdvancedContent] = useState('');
    const [advancedStatus, setAdvancedStatus] = useState('');
    const [advancedDirty, setAdvancedDirty] = useState(false);
    const [soulContent, setSoulContent] = useState('');
    const [soulStatus, setSoulStatus] = useState('');
    const [soulDirty, setSoulDirty] = useState(false);
    const [showApiKey, setShowApiKey] = useState(false);
    const [autoScrollLogs, setAutoScrollLogs] = useState(true);
    const [providers, setProviders] = useState<ProviderConfig>(fallbackProviderConfig);
    const [selectedProvider, setSelectedProvider] = useState('dashscope-payg');
    const [modelOptions, setModelOptions] = useState<ModelOption[]>([]);
    const [modelOptionsKey, setModelOptionsKey] = useState('');
    const [modelListStatus, setModelListStatus] = useState('');
    const [selectedAux, setSelectedAux] = useState('vision');
    const [auxModelOptions, setAuxModelOptions] = useState<Record<string, ModelOption[]>>({});
    const [auxModelListStatus, setAuxModelListStatus] = useState('');
    const [newProfileID, setNewProfileID] = useState('');
    const [newProfileName, setNewProfileName] = useState('');
    const [newProfileCopyMode, setNewProfileCopyMode] = useState('clean');
    const [newProfileEnabled, setNewProfileEnabled] = useState(true);

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
            setQrData(event.scan_data);
            setQrStatus('等待微信扫码');
        });
        const offQRStatus = EventsOn('weixin-login:status', (event: { status?: string; message?: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) return;
            setQrStatus(event.message || event.status || '');
        });
        const offQRDone = EventsOn('weixin-login:confirmed', (event: { account_id: string; user_id: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) {
                refresh();
                return;
            }
            setQrStatus(`绑定成功 ${event.user_id || event.account_id}`);
            setQrData('');
            refresh();
        });
        const offQRError = EventsOn('weixin-login:error', (event: { message: string; profile_id?: string }) => {
            if (!isCurrentProfileEvent(event)) return;
            setQrStatus(event.message);
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
        if (page !== 'operations' || operationsTab !== 'advanced') return;
        if (advancedDirty) return;
        loadAdvancedFile(advancedPath);
    }, [page, operationsTab, advancedPath]);

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

    async function refresh() {
        const next = await GetAppState();
        const nextState = next as AppState;
        activeProfileRef.current = nextState.activeProfile || 'default';
        setState(nextState);
        setEnv(nextState.environment || []);
        setCompose(nextState.compose);
        setModel(nextState.model);
        const nextProviders = nextState.providers || fallbackProviderConfig;
        setProviders(nextProviders);
        setSelectedProvider(nextState.model?.provider && nextProviders.providers[nextState.model.provider] ? nextState.model.provider : firstProviderID(nextProviders));
    }

    async function selectProfile(id: string) {
        if (id === state?.activeProfile) return;
        if (advancedDirty || soulDirty) {
            setNotice({type: 'error', message: '当前有未保存修改，请先保存或放弃修改后再切换 profile'});
            return;
        }
        const target = state?.profiles?.profiles?.find((profile) => profile.id === id);
        await run('正在切换 Profile', () => SelectProfile(id), {
            afterSuccess: () => {
                setAdvancedDirty(false);
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
        await run('正在删除 Profile', () => DeleteProfile(id), {rebuildRequired: true});
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
        try {
            await action();
            await refresh();
            if (options.rebuildRequired) {
                setNeedsRebuild(true);
            }
            options.afterSuccess?.();
            setNotice({type: 'ok', message: doneLabel(label)});
            return true;
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
            return false;
        } finally {
            setBusy('');
        }
    }

    async function loadAdvancedFile(path: string) {
        setAdvancedStatus('正在读取文件');
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
            await SaveTextFile({path: advancedPath, content: advancedContent, reason: 'before-advanced-save'});
            setAdvancedDirty(false);
            setNeedsRebuild(true);
            setAdvancedStatus(`已保存 ${advancedPath}`);
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
        }, {rebuildRequired: true});
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

    function openOperations(tab: OperationsTab) {
        setOperationsTab(tab);
        setPage('operations');
    }

    async function factoryReset() {
        await run('正在恢复出厂设置', FactoryResetInstance, {
            afterSuccess: () => {
                setLogs([]);
                setNeedsRebuild(false);
                setAdvancedDirty(false);
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
        setNotice({type: 'info', message: '正在读取日志'});
        try {
            await TailLogs();
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

    return (
        <div className="shell">
            <aside className="rail">
                <div className="brand">
                    <div className="brand-mark">HD</div>
                    <div>
                        <strong>Hermes Dock</strong>
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

            <main className="workspace">
                <header className="topbar">
                    <div>
                        <p className="eyebrow">Hermes Agent 容器</p>
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
                    </div>
                </header>
                {notice && <div className={`notice ${notice.type}`}>{notice.message}</div>}
                {showRebuildBanner && (
                    <div className="rebuild-banner">
                        <span>配置已保存，重建后生效。</span>
                        <button onClick={() => run('正在重建', RebuildHermes, {afterSuccess: () => setNeedsRebuild(false)})} disabled={!!busy}>
                            <RotateCcw size={16}/>重建
                        </button>
                    </div>
                )}

                {page === 'assistants' && state && (
                    <AssistantsPage
                        state={state}
                        env={env}
                        setEnv={setEnv}
                        providers={providers}
                        setProviders={setProviders}
                        selectedProvider={selectedProvider}
                        setSelectedProvider={setSelectedProvider}
                        model={model}
                        setModel={setModel}
                        modelOptions={visibleModelOptions}
                        modelListStatus={modelListStatus}
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
                        needsRebuild={needsRebuild}
                        hasPlatformBinding={!!weixinBound || !!wecomBound || !!feishuBound}
                        onSelect={selectProfile}
                        onCreate={createProfile}
                        onRename={(id, name) => run('正在更新助手', () => UpdateProfileName(id, name))}
                        onEnabled={(id, enabled) => run(enabled ? '正在启用助手' : '正在停用助手', () => SetProfileEnabled(id, enabled), {rebuildRequired: true})}
                        onMove={(id, direction) => run('正在调整顺序', () => MoveProfile(id, direction))}
                        onDelete={deleteProfile}
                        onSaveModelService={saveModelService}
                        onFetchModels={fetchModels}
                        onFetchAuxModels={fetchAuxModels}
                        onTestModel={() => run('正在测试模型', TestModel)}
                        onSaveSoul={saveSoulFile}
                        onDiscardSoul={() => loadSoulFile(state?.activeProfile || 'default')}
                        onWeixinLogin={() => run('正在启动微信扫码登录', StartWeixinLogin)}
                        onCancelWeixin={() => CancelWeixinLogin()}
                        onSaveWeCom={() => run('正在保存企业微信配置', () => SaveWeComConfig({
                            botId: envValue(env, 'WECOM_BOT_ID'),
                            secret: envValue(env, 'WECOM_SECRET'),
                            websocketUrl: envValue(env, 'WECOM_WEBSOCKET_URL'),
                            dmPolicy: closedPolicyValue(envValue(env, 'WECOM_DM_POLICY')),
                            allowedUsers: '',
                            groupPolicy: closedPolicyValue(envValue(env, 'WECOM_GROUP_POLICY')),
                            groupAllowUsers: '',
                        }), {rebuildRequired: true})}
                        onSaveFeishu={() => run('正在保存飞书配置', () => SaveFeishuConfig({
                            appId: envValue(env, 'FEISHU_APP_ID'),
                            appSecret: envValue(env, 'FEISHU_APP_SECRET'),
                            domain: envValue(env, 'FEISHU_DOMAIN') || 'feishu',
                            allowedUsers: '',
                            groupPolicy: disabledPolicyValue(envValue(env, 'FEISHU_GROUP_POLICY')),
                        }), {rebuildRequired: true})}
                        onFinishSetup={finishProfileSetup}
                        onRebuild={() => run('正在重建', RebuildHermes, {afterSuccess: () => setNeedsRebuild(false)})}
                        onOpenOperations={openOperations}
                    />
                )}

                {page === 'operations' && state && compose && (
                    <OperationsPage
                        tab={operationsTab}
                        setTab={setOperationsTab}
                        state={state}
                        compose={compose}
                        setCompose={setCompose}
                        needsRebuild={needsRebuild}
                        busy={busy}
                        logs={logs}
                        activeProfileName={state.profiles?.profiles?.find((profile) => profile.id === state.activeProfile)?.name || state.activeProfile || 'default'}
                        weixinBound={!!weixinBound}
                        wecomBound={!!wecomBound}
                        feishuBound={!!feishuBound}
                        weixinHomeChannel={weixinHomeChannel}
                        advancedOptions={advancedFileOptions(state.activeProfile || 'default')}
                        advancedPath={advancedPath}
                        setAdvancedPath={changeAdvancedPath}
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
                        onStart={() => run('正在启动', StartHermes)}
                        onStop={() => run('正在停止', StopHermes)}
                        onRestart={() => run('正在重启', RestartHermes)}
                        onRebuild={() => run('正在重建', RebuildHermes, {afterSuccess: () => setNeedsRebuild(false)})}
                        onLogs={tailLogs}
                        onClearLogs={() => setLogs([])}
                        onCopyLogs={copyLogs}
                        onOpenEndpoint={openEndpoint}
                        onOpenAssistantPlatforms={() => {
                            setPage('assistants');
                            setWizardStep('platforms');
                        }}
                        onSaveDeploy={() => run('正在保存部署配置', () => SaveComposeSettings({...compose, dashboardEnabled: true}), {rebuildRequired: true})}
                        onRefreshChannels={() => run('正在刷新通道', refresh)}
                        onHomeChannel={(platform, id) => run('正在设置默认通道', () => SetHomeChannel(platform, id), {rebuildRequired: true})}
                        onTestChannel={(platform, id) => run('正在发送测试消息', () => SendTestMessage(platform, id, 'Hermes Dock 测试消息'))}
                        onSaveAdvanced={saveAdvancedFile}
                        onFactoryReset={factoryReset}
                        resetConfirmPhrase={factoryResetPhrase}
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

export default App;
