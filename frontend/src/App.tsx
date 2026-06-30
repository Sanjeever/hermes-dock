import {useEffect, useRef, useState} from 'react';
import {CheckCircle2, CircleAlert, RotateCcw} from 'lucide-react';
import './App.css';
import {
    CancelWeixinLogin,
    CreateProfile,
    DeleteProfile,
    FactoryResetInstance,
    FetchModelList,
    FetchProviderConfigModelList,
    GetAppState,
    OpenEndpoint,
    ReadTextFile,
    RebuildHermes,
    RestartHermes,
    SaveComposeSettings,
    SaveEnvironment,
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
import {AdvancedPage} from './pages/AdvancedPage';
import {ChannelsPage} from './pages/ChannelsPage';
import {Dashboard} from './pages/DashboardPage';
import {DeployPage} from './pages/DeployPage';
import {ModelsPage} from './pages/ModelsPage';
import {PlatformsPage} from './pages/PlatformsPage';
import {ProfilesPage} from './pages/ProfilesPage';
import {ProvidersPage} from './pages/ProvidersPage';
import {SoulPage} from './pages/SoulPage';
import {factoryResetPhrase, fallbackProviderConfig, nav} from './constants';
import type {AppState, ComposeSettings, EnvVar, ModelConfig, ModelListRequest, ModelOption, Notice, Page, ProviderConfig, ProviderEntry, RunOptions} from './types';
import {advancedFileOptions, containerStatusText, defaultAdvancedPath, doneLabel, envValue, firstProviderID, modelOptionKey, profileFilePath, titleFor, toPlainModelConfig, toPlainProviderConfig} from './utils';

function App() {
    const [page, setPage] = useState<Page>('dashboard');
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
    const [selectedAux, setSelectedAux] = useState('vision');
    const [showApiKey, setShowApiKey] = useState(false);
    const [autoScrollLogs, setAutoScrollLogs] = useState(true);
    const [providers, setProviders] = useState<ProviderConfig>(fallbackProviderConfig);
    const [selectedProvider, setSelectedProvider] = useState('dashscope-payg');
    const [modelOptions, setModelOptions] = useState<ModelOption[]>([]);
    const [modelOptionsKey, setModelOptionsKey] = useState('');
    const [modelListStatus, setModelListStatus] = useState('');
    const [auxModelOptions, setAuxModelOptions] = useState<Record<string, ModelOption[]>>({});
    const [auxModelListStatus, setAuxModelListStatus] = useState('');
    const [providerModelOptions, setProviderModelOptions] = useState<ModelOption[]>([]);
    const [providerModelListStatus, setProviderModelListStatus] = useState('');
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
        if (page !== 'advanced') return;
        if (advancedDirty) return;
        loadAdvancedFile(advancedPath);
    }, [page, advancedPath]);

    useEffect(() => {
        if (page !== 'soul') return;
        if (soulDirty) return;
        loadSoulFile(state?.activeProfile || 'default');
    }, [page, state?.activeProfile]);

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
        setSelectedProvider((current) => nextProviders.providers[current] ? current : firstProviderID(nextProviders));
    }

    async function selectProfile(id: string) {
        if (id === state?.activeProfile) return;
        if (advancedDirty || soulDirty) {
            setNotice({type: 'error', message: '当前有未保存修改，请先保存或放弃修改后再切换 profile'});
            return;
        }
        await run('正在切换 Profile', () => SelectProfile(id), {
            afterSuccess: () => {
                setAdvancedDirty(false);
                setAdvancedPath(defaultAdvancedPath(id));
                setSoulDirty(false);
                setSoulContent('');
            },
        });
    }

    async function createProfile() {
        await run('正在创建 Profile', () => CreateProfile({
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
                setPage('profiles');
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
        const providerID = model.provider || firstProviderID(providers);
        const optionsKey = modelOptionKey(providerID);
        setModelListStatus('正在拉取模型列表');
        try {
            const req: ModelListRequest = {providerId: providerID, providerKey: providerID, apiKey: '', baseUrl: ''};
            const items = await FetchModelList(req);
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
        if (!model) return;
        const auxOptionKey = modelOptionKey(providerID);
        const provider = providers.providers[providerID];
        if (!provider || provider.disabled) {
            setAuxModelListStatus('供应商不可用');
            return;
        }
        if (provider.apiKey.trim() === '') {
            setAuxModelListStatus('请先在供应商页填写 API 密钥');
            return;
        }
        setAuxModelListStatus('正在拉取模型列表');
        try {
            const req: ModelListRequest = {providerId: providerID, providerKey: providerID, apiKey: '', baseUrl: ''};
            const items = await FetchModelList(req);
            setAuxModelOptions((current) => ({...current, [auxOptionKey]: items as ModelOption[]}));
            setAuxModelListStatus(`已拉取 ${(items as ModelOption[]).length} 个模型`);
        } catch (error) {
            setAuxModelOptions((current) => ({...current, [auxOptionKey]: []}));
            setAuxModelListStatus(String(error));
            appendLog(String(error));
        }
    }

    async function fetchProviderModels(provider: ProviderEntry) {
        setProviderModelListStatus('正在拉取模型列表');
        try {
            const items = await FetchProviderConfigModelList(provider);
            setProviderModelOptions(items as ModelOption[]);
            setProviderModelListStatus(`已拉取 ${(items as ModelOption[]).length} 个模型`);
        } catch (error) {
            setProviderModelOptions([]);
            setProviderModelListStatus(String(error));
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
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setNotice({type: 'error', message});
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
        } catch (error) {
            const message = String(error);
            appendLog(message);
            setSoulStatus(message);
            setNotice({type: 'error', message});
        } finally {
            setBusy('');
        }
    }

    async function factoryReset() {
        await run('正在恢复出厂设置', FactoryResetInstance, {
            afterSuccess: () => {
                setLogs([]);
                setNeedsRebuild(false);
                setAdvancedDirty(false);
                setPage('providers');
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
                        {state?.profiles?.profiles && (
                            <label className="profile-picker">
                                <span>当前 Profile</span>
                                <select value={state.activeProfile || 'default'} onChange={(event) => selectProfile(event.target.value)} disabled={!!busy}>
                                    {state.profiles.profiles.map((profile) => <option key={profile.id} value={profile.id}>{profile.name || profile.id}</option>)}
                                </select>
                            </label>
                        )}
                        <div className={`status-pill ${statusClass}`}>
                            {state?.containerStatus === 'running' ? <CheckCircle2 size={16}/> : <CircleAlert size={16}/>}
                            {containerStatusText(state?.containerStatus)}
                        </div>
                    </div>
                </header>
                {notice && <div className={`notice ${notice.type}`}>{notice.message}</div>}
                {needsRebuild && (
                    <div className="rebuild-banner">
                        <span>配置已保存，重建后生效。</span>
                        <button onClick={() => run('正在重建', RebuildHermes, {afterSuccess: () => setNeedsRebuild(false)})} disabled={!!busy}>
                            <RotateCcw size={16}/>重建
                        </button>
                    </div>
                )}

                {page === 'dashboard' && state && compose && (
                    <Dashboard
                        state={state}
                        compose={compose}
                        busy={busy}
                        logs={logs}
                        weixinBound={!!weixinBound}
                        wecomBound={!!wecomBound}
                        feishuBound={!!feishuBound}
                        onStart={() => run('正在启动', StartHermes)}
                        onStop={() => run('正在停止', StopHermes)}
                        onRestart={() => run('正在重启', RestartHermes)}
                        onRebuild={() => run('正在重建', RebuildHermes, {afterSuccess: () => setNeedsRebuild(false)})}
                        onLogs={tailLogs}
                        onClearLogs={() => setLogs([])}
                        onCopyLogs={copyLogs}
                        autoScrollLogs={autoScrollLogs}
                        setAutoScrollLogs={setAutoScrollLogs}
                        logRef={logRef}
                        onOpenEndpoint={openEndpoint}
                        onOpenDeploy={() => setPage('deploy')}
                        onOpenPlatforms={() => setPage('platforms')}
                        onOpenProfiles={() => setPage('profiles')}
                    />
                )}

                {page === 'profiles' && state && (
                    <ProfilesPage
                        registry={state.profiles}
                        activeProfile={state.activeProfile}
                        status={state.profileStatus}
                        busy={!!busy}
                        newProfileID={newProfileID}
                        setNewProfileID={setNewProfileID}
                        newProfileName={newProfileName}
                        setNewProfileName={setNewProfileName}
                        newProfileCopyMode={newProfileCopyMode}
                        setNewProfileCopyMode={setNewProfileCopyMode}
                        newProfileEnabled={newProfileEnabled}
                        setNewProfileEnabled={setNewProfileEnabled}
                        onSelect={selectProfile}
                        onCreate={createProfile}
                        onRename={(id, name) => run('正在更新 Profile', () => UpdateProfileName(id, name))}
                        onEnabled={(id, enabled) => run(enabled ? '正在启用 Profile' : '正在停用 Profile', () => SetProfileEnabled(id, enabled), {rebuildRequired: true})}
                        onMove={(id, direction) => run('正在调整顺序', () => MoveProfile(id, direction))}
                        onDelete={deleteProfile}
                    />
                )}

                {page === 'deploy' && compose && (
                    <DeployPage compose={compose} setCompose={setCompose} busy={!!busy} onSave={() => run('正在保存部署配置', () => SaveComposeSettings({...compose, dashboardEnabled: true}), {rebuildRequired: true})}/>
                )}

                {page === 'providers' && (
                    <ProvidersPage
                        providers={providers}
                        setProviders={setProviders}
                        selectedProvider={selectedProvider}
                        setSelectedProvider={setSelectedProvider}
                        model={model}
                        busy={!!busy}
                        showApiKey={showApiKey}
                        setShowApiKey={setShowApiKey}
                        modelOptions={providerModelOptions}
                        modelListStatus={providerModelListStatus}
                        onFetchModels={fetchProviderModels}
                        onSave={() => run('正在保存供应商配置', () => SaveProviderConfig(toPlainProviderConfig(providers)), {rebuildRequired: true})}
                    />
                )}

                {page === 'models' && model && (
                    <ModelsPage
                        model={model}
                        setModel={setModel}
                        selectedAux={selectedAux}
                        setSelectedAux={setSelectedAux}
                        providers={providers}
                        modelOptions={visibleModelOptions}
                        modelListStatus={modelListStatus}
                        auxModelOptions={auxModelOptions}
                        auxModelListStatus={auxModelListStatus}
                        busy={!!busy}
                        onFetchModels={fetchModels}
                        onFetchAuxModels={fetchAuxModels}
                        onSave={() => run('正在保存模型配置', () => SaveModelConfig(toPlainModelConfig(model)), {rebuildRequired: true})}
                        onTest={() => run('正在测试模型', TestModel)}
                    />
                )}

                {page === 'platforms' && (
                    <PlatformsPage
                        env={env}
                        setEnv={setEnv}
                        qrData={qrData}
                        qrStatus={qrStatus}
                        busy={!!busy}
                        onSaveEnv={() => run('正在保存平台配置', () => SaveEnvironment(env), {rebuildRequired: true})}
                        onWeixinLogin={() => run('正在启动微信扫码登录', StartWeixinLogin)}
                        onCancelWeixin={() => CancelWeixinLogin()}
                        onSaveWeCom={() => run('正在保存企业微信配置', () => SaveWeComConfig({
                            botId: envValue(env, 'WECOM_BOT_ID'),
                            secret: envValue(env, 'WECOM_SECRET'),
                            websocketUrl: envValue(env, 'WECOM_WEBSOCKET_URL'),
                            dmPolicy: envValue(env, 'WECOM_DM_POLICY') || 'open',
                            allowedUsers: envValue(env, 'WECOM_ALLOWED_USERS'),
                            groupPolicy: envValue(env, 'WECOM_GROUP_POLICY') || 'open',
                            groupAllowUsers: envValue(env, 'WECOM_GROUP_ALLOWED_USERS'),
                        }), {rebuildRequired: true})}
                        onSaveFeishu={() => run('正在保存飞书配置', () => SaveFeishuConfig({
                            appId: envValue(env, 'FEISHU_APP_ID'),
                            appSecret: envValue(env, 'FEISHU_APP_SECRET'),
                            domain: envValue(env, 'FEISHU_DOMAIN') || 'feishu',
                            allowedUsers: envValue(env, 'FEISHU_ALLOWED_USERS'),
                            groupPolicy: envValue(env, 'FEISHU_GROUP_POLICY') || 'allowlist',
                        }), {rebuildRequired: true})}
                    />
                )}

                {page === 'channels' && state && (
                    <ChannelsPage channels={state.channels} weixinHomeChannel={weixinHomeChannel} busy={!!busy} onRefresh={() => run('正在刷新通道', refresh)}
                                  onHome={(platform, id) => run('正在设置默认通道', () => SetHomeChannel(platform, id), {rebuildRequired: true})}
                                  onTest={(platform, id) => run('正在发送测试消息', () => SendTestMessage(platform, id, 'Hermes Dock 测试消息'))}/>
                )}

                {page === 'soul' && (
                    <SoulPage
                        profileID={state?.activeProfile || 'default'}
                        content={soulContent}
                        setContent={(value) => {
                            setSoulContent(value);
                            setSoulDirty(true);
                        }}
                        status={soulStatus}
                        dirty={soulDirty}
                        busy={!!busy}
                        onSave={saveSoulFile}
                        onDiscard={() => loadSoulFile(state?.activeProfile || 'default')}
                    />
                )}

                {page === 'advanced' && (
                    <AdvancedPage
                        options={advancedFileOptions(state?.activeProfile || 'default')}
                        path={advancedPath}
                        setPath={changeAdvancedPath}
                        content={advancedContent}
                        setContent={(value) => {
                            setAdvancedContent(value);
                            setAdvancedDirty(true);
                        }}
                        status={advancedStatus}
                        dirty={advancedDirty}
                        busy={!!busy}
                        onSave={saveAdvancedFile}
                        onFactoryReset={factoryReset}
                        resetConfirmPhrase={factoryResetPhrase}
                    />
                )}
            </main>
        </div>
    );
}

export default App;
