import {useEffect, useMemo, useState} from 'react';
import {
    Activity,
    Boxes,
    CheckCircle2,
    CircleAlert,
    Database,
    FileCog,
    Gauge,
    KeyRound,
    Loader2,
    MessageSquare,
    Play,
    QrCode,
    RefreshCcw,
    RotateCcw,
    Save,
    Settings,
    Square,
    TerminalSquare,
    Wrench,
} from 'lucide-react';
import {QRCodeSVG} from 'qrcode.react';
import './App.css';
import {
    CancelWeixinLogin,
    FetchModelList,
    GetAppState,
    GetModelProviderPresets,
    InitializeInstance,
    ReadTextFile,
    RebuildHermes,
    RestartHermes,
    RunDiagnostics,
    SaveComposeSettings,
    SaveEnvironment,
    SaveModelConfig,
    SaveTextFile,
    SaveWeComConfig,
    SendTestMessage,
    SetHomeChannel,
    StartHermes,
    StartWeixinLogin,
    StopHermes,
    TailLogs,
    TestModel,
} from '../wailsjs/go/main/App';
import {EventsOn} from '../wailsjs/runtime/runtime';

type Page = 'dashboard' | 'environment' | 'deploy' | 'models' | 'platforms' | 'channels' | 'diagnostics' | 'advanced';

type EnvVar = { key: string; value: string; secret: boolean };
type ComposeSettings = {
    image: string;
    containerName: string;
    gatewayHost: string;
    gatewayPort: string;
    dashboardHost: string;
    dashboardPort: string;
    dashboardEnabled: boolean;
    dashboardUsername: string;
    dashboardPassword: string;
    memoryLimit: string;
    cpuLimit: string;
    shmSize: string;
};
type AuxModel = { provider: string; model: string; baseUrl: string; apiKey: string; timeout: number; extraBody: Record<string, unknown> };
type ModelConfig = {
    provider: string;
    default: string;
    baseUrl: string;
    apiMode: string;
    apiKey: string;
    auxiliaryMode: string;
    auxiliary: Record<string, AuxModel>;
};
type ModelProviderPreset = {
    key: string;
    label: string;
    provider: string;
    baseUrl: string;
    apiMode: string;
    defaultModel: string;
    modelListUrl: string;
};
type ModelOption = { id: string; ownedBy: string };
type Diagnostic = { id: string; label: string; status: string; message: string; severity: string; fixable: boolean };
type ChannelSummary = { id: string; name: string; type: string; thread_id?: string };
type ChannelFile = { updated_at: string; platforms: Record<string, ChannelSummary[]> };
type AppState = {
    appVersion: string;
    instanceRoot: string;
    compose: ComposeSettings;
    environment: EnvVar[];
    model: ModelConfig;
    channels: ChannelFile;
    diagnostics: Diagnostic[];
    dockerAvailable: boolean;
    composeAvailable: boolean;
    containerStatus: string;
};

const nav: Array<{ id: Page; label: string; icon: typeof Gauge }> = [
    {id: 'dashboard', label: '总览', icon: Gauge},
    {id: 'environment', label: '环境变量', icon: KeyRound},
    {id: 'deploy', label: '部署', icon: Settings},
    {id: 'models', label: '模型', icon: Boxes},
    {id: 'platforms', label: '平台绑定', icon: MessageSquare},
    {id: 'channels', label: '通道', icon: Activity},
    {id: 'diagnostics', label: '诊断', icon: Wrench},
    {id: 'advanced', label: '高级编辑', icon: FileCog},
];

const auxLabels: Record<string, string> = {
    vision: '视觉理解',
    web_extract: '网页提取',
    compression: '上下文压缩',
    skills_hub: '技能中心',
    approval: '审批',
    mcp: 'MCP 配置',
    title_generation: '标题生成',
    tts_audio_tags: 'TTS 音频标签',
    triage_specifier: '任务分流',
    kanban_decomposer: '看板拆解',
    profile_describer: '档案描述',
    curator: '技能维护',
    monitor: '监控',
};

const fallbackModelProviderPresets: ModelProviderPreset[] = [
    {
        key: 'dashscope-payg',
        label: 'DashScope 按量计费',
        provider: 'custom',
        baseUrl: 'https://dashscope.aliyuncs.com/apps/anthropic',
        apiMode: 'anthropic_messages',
        defaultModel: 'qwen3.7-max',
        modelListUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1/models',
    },
    {
        key: 'deepseek',
        label: 'DeepSeek',
        provider: 'deepseek',
        baseUrl: 'https://api.deepseek.com',
        apiMode: 'chat_completions',
        defaultModel: 'deepseek-v4-flash',
        modelListUrl: 'https://api.deepseek.com/models',
    },
];

function App() {
    const [page, setPage] = useState<Page>('dashboard');
    const [state, setState] = useState<AppState | null>(null);
    const [env, setEnv] = useState<EnvVar[]>([]);
    const [compose, setCompose] = useState<ComposeSettings | null>(null);
    const [model, setModel] = useState<ModelConfig | null>(null);
    const [logs, setLogs] = useState<string[]>([]);
    const [busy, setBusy] = useState('');
    const [qrData, setQrData] = useState('');
    const [qrStatus, setQrStatus] = useState('');
    const [advancedPath, setAdvancedPath] = useState('data/config.yaml');
    const [advancedContent, setAdvancedContent] = useState('');
    const [selectedAux, setSelectedAux] = useState('vision');
    const [providerPresets, setProviderPresets] = useState<ModelProviderPreset[]>(fallbackModelProviderPresets);
    const [modelOptions, setModelOptions] = useState<ModelOption[]>([]);
    const [modelListStatus, setModelListStatus] = useState('');

    useEffect(() => {
        refresh();
        GetModelProviderPresets().then((items) => setProviderPresets(items as ModelProviderPreset[])).catch((error) => appendLog(String(error)));
        const offDocker = EventsOn('docker:progress', (event: { line?: string; done?: boolean; code?: number }) => {
            if (event.line) appendLog(event.line);
            if (event.done) {
                appendLog(`命令退出，代码 ${event.code}`);
                setBusy('');
                refresh();
            }
        });
        const offLogs = EventsOn('logs:line', (event: { line?: string }) => event.line && appendLog(event.line));
        const offQR = EventsOn('weixin-login:qr', (event: { scan_data: string }) => {
            setQrData(event.scan_data);
            setQrStatus('等待微信扫码');
        });
        const offQRStatus = EventsOn('weixin-login:status', (event: { status?: string; message?: string }) => {
            setQrStatus(event.message || event.status || '');
        });
        const offQRDone = EventsOn('weixin-login:confirmed', (event: { account_id: string; user_id: string }) => {
            setQrStatus(`绑定成功 ${event.user_id || event.account_id}`);
            setQrData('');
            refresh();
        });
        const offQRError = EventsOn('weixin-login:error', (event: { message: string }) => setQrStatus(event.message));
        return () => {
            offDocker();
            offLogs();
            offQR();
            offQRStatus();
            offQRDone();
            offQRError();
        };
    }, []);

    async function refresh() {
        const next = await GetAppState();
        setState(next as AppState);
        setEnv((next as AppState).environment || []);
        setCompose((next as AppState).compose);
        setModel((next as AppState).model);
    }

    function appendLog(line: string) {
        setLogs((current) => [...current.slice(-300), line]);
    }

    async function fetchModels() {
        if (!model) return;
        const providerKey = providerKeyFor(model, providerPresets);
        setModelListStatus('正在拉取模型列表');
        try {
            const items = await FetchModelList({providerKey, apiKey: model.apiKey});
            setModelOptions(items as ModelOption[]);
            setModelListStatus(`已拉取 ${(items as ModelOption[]).length} 个模型`);
        } catch (error) {
            setModelOptions([]);
            setModelListStatus(String(error));
            appendLog(String(error));
        }
    }

    async function run(label: string, action: () => Promise<unknown>) {
        setBusy(label);
        try {
            await action();
            await refresh();
        } catch (error) {
            appendLog(String(error));
        } finally {
            setBusy('');
        }
    }

    const statusClass = state?.containerStatus === 'running' ? 'ok' : 'warn';
    const weixinBound = envValue(env, 'WEIXIN_ACCOUNT_ID') && envValue(env, 'WEIXIN_TOKEN');
    const wecomBound = envValue(env, 'WECOM_BOT_ID') && envValue(env, 'WECOM_SECRET');

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
                    <div className={`status-pill ${statusClass}`}>
                        {state?.containerStatus === 'running' ? <CheckCircle2 size={16}/> : <CircleAlert size={16}/>}
                        {containerStatusText(state?.containerStatus)}
                    </div>
                </header>

                {page === 'dashboard' && state && compose && (
                    <Dashboard
                        state={state}
                        compose={compose}
                        busy={busy}
                        logs={logs}
                        weixinBound={!!weixinBound}
                        wecomBound={!!wecomBound}
                        onInit={() => run('正在重新释放', () => InitializeInstance(compose))}
                        onStart={() => run('正在启动', StartHermes)}
                        onStop={() => run('正在停止', StopHermes)}
                        onRestart={() => run('正在重启', RestartHermes)}
                        onRebuild={() => run('正在重建', RebuildHermes)}
                        onLogs={() => TailLogs()}
                    />
                )}

                {page === 'environment' && (
                    <EnvironmentPage env={env} setEnv={setEnv} onSave={() => run('正在保存环境变量', () => SaveEnvironment(env))}/>
                )}

                {page === 'deploy' && compose && (
                    <DeployPage compose={compose} setCompose={setCompose} onSave={() => run('正在保存部署配置', () => SaveComposeSettings(compose))}/>
                )}

                {page === 'models' && model && (
                    <ModelsPage
                        model={model}
                        setModel={setModel}
                        selectedAux={selectedAux}
                        setSelectedAux={setSelectedAux}
                        providerPresets={providerPresets}
                        modelOptions={modelOptions}
                        modelListStatus={modelListStatus}
                        onFetchModels={fetchModels}
                        onSave={() => run('正在保存模型配置', () => SaveModelConfig(toPlainModelConfig(model)))}
                        onTest={() => run('正在测试模型', TestModel)}
                    />
                )}

                {page === 'platforms' && (
                    <PlatformsPage
                        env={env}
                        setEnv={setEnv}
                        qrData={qrData}
                        qrStatus={qrStatus}
                        onSaveEnv={() => run('正在保存平台配置', () => SaveEnvironment(env))}
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
                        }))}
                    />
                )}

                {page === 'channels' && state && (
                    <ChannelsPage channels={state.channels} onHome={(platform, id) => run('正在设置默认通道', () => SetHomeChannel(platform, id))}
                                  onTest={(platform, id) => run('正在发送测试消息', () => SendTestMessage(platform, id, 'Hermes Dock 测试消息'))}/>
                )}

                {page === 'diagnostics' && state && (
                    <DiagnosticsPage diagnostics={state.diagnostics} onRefresh={() => run('正在运行诊断', async () => {
                        const diagnostics = await RunDiagnostics();
                        setState({...state, diagnostics: diagnostics as Diagnostic[]});
                    })}/>
                )}

                {page === 'advanced' && (
                    <AdvancedPage
                        path={advancedPath}
                        setPath={setAdvancedPath}
                        content={advancedContent}
                        setContent={setAdvancedContent}
                        onLoad={() => run('正在读取文件', async () => setAdvancedContent(await ReadTextFile(advancedPath)))}
                        onSave={() => run('正在保存文件', () => SaveTextFile({path: advancedPath, content: advancedContent, reason: 'before-advanced-save'}))}
                    />
                )}
            </main>
        </div>
    );
}

function Dashboard(props: {
    state: AppState;
    compose: ComposeSettings;
    busy: string;
    logs: string[];
    weixinBound: boolean;
    wecomBound: boolean;
    onInit: () => void;
    onStart: () => void;
    onStop: () => void;
    onRestart: () => void;
    onRebuild: () => void;
    onLogs: () => void;
}) {
    const actionBusy = props.busy !== '';
    return (
        <section className="grid two">
            <div className="panel hero-panel">
                <div>
                    <p className="eyebrow">当前镜像</p>
                    <h2>{props.compose.image}</h2>
                    <p className="muted">控制台 {props.compose.dashboardHost}:{props.compose.dashboardPort} · 网关 {props.compose.gatewayHost}:{props.compose.gatewayPort}</p>
                </div>
                <div className="actions">
                    <IconButton icon={Database} label="重新释放" onClick={props.onInit} disabled={actionBusy}/>
                    <IconButton icon={Play} label="启动" onClick={props.onStart} disabled={actionBusy}/>
                    <IconButton icon={Square} label="停止" onClick={props.onStop} disabled={actionBusy}/>
                    <IconButton icon={RefreshCcw} label="重启" onClick={props.onRestart} disabled={actionBusy}/>
                    <IconButton icon={RotateCcw} label="重建" onClick={props.onRebuild} disabled={actionBusy}/>
                    <IconButton icon={TerminalSquare} label="日志" onClick={props.onLogs} disabled={false}/>
                </div>
                {props.busy && <div className="busy"><Loader2 size={16} className="spin"/>{props.busy}</div>}
            </div>
            <div className="panel">
                <p className="eyebrow">就绪状态</p>
                <div className="health-list">
                    <Health label="Docker" ok={props.state.dockerAvailable}/>
                    <Health label="Compose" ok={props.state.composeAvailable}/>
                    <Health label="个人微信" ok={props.weixinBound}/>
                    <Health label="企业微信 AI Bot" ok={props.wecomBound}/>
                </div>
            </div>
            <div className="panel wide">
                <p className="eyebrow">实时输出</p>
                <pre className="logbox">{props.logs.length ? props.logs.join('\n') : '暂无命令输出。'}</pre>
            </div>
        </section>
    );
}

function EnvironmentPage({env, setEnv, onSave}: { env: EnvVar[]; setEnv: (value: EnvVar[]) => void; onSave: () => void }) {
    const common = ['WEIXIN_DM_POLICY', 'WEIXIN_GROUP_POLICY', 'WECOM_DM_POLICY', 'WECOM_GROUP_POLICY', 'TERMINAL_LIFETIME_SECONDS'];
    return (
        <section className="grid two">
            <div className="panel">
                <p className="eyebrow">常用环境变量</p>
                {common.map((key) => <Field key={key} label={key} value={envValue(env, key)} secret={key.includes('KEY')} onChange={(value) => setEnv(setEnvValue(env, key, value))}/>)}
                <button className="primary" onClick={onSave}><Save size={16}/>保存环境变量</button>
            </div>
            <div className="panel">
                <p className="eyebrow">全部环境变量</p>
                <div className="kv-list">
                    {env.map((item, index) => (
                        <div className="kv-row" key={`${item.key}-${index}`}>
                            <input value={item.key} onChange={(event) => setEnv(env.map((entry, i) => i === index ? {...entry, key: event.target.value} : entry))}/>
                            <input type={item.secret ? 'password' : 'text'} value={item.value} onChange={(event) => setEnv(env.map((entry, i) => i === index ? {...entry, value: event.target.value} : entry))}/>
                        </div>
                    ))}
                </div>
                <button className="ghost" onClick={() => setEnv([...env, {key: '', value: '', secret: false}])}>添加变量</button>
            </div>
        </section>
    );
}

function DeployPage({compose, setCompose, onSave}: { compose: ComposeSettings; setCompose: (value: ComposeSettings) => void; onSave: () => void }) {
    const update = (key: keyof ComposeSettings, value: string | boolean) => setCompose({...compose, [key]: value});
    return (
        <section className="grid two">
            <div className="panel">
                <p className="eyebrow">镜像与端口</p>
                <Field label="镜像" value={compose.image} onChange={(value) => update('image', value)}/>
                <div className="field-grid">
                    <Field label="网关监听地址" value={compose.gatewayHost} onChange={(value) => update('gatewayHost', value)}/>
                    <Field label="网关端口" value={compose.gatewayPort} onChange={(value) => update('gatewayPort', value)}/>
                    <Field label="控制台监听地址" value={compose.dashboardHost} onChange={(value) => update('dashboardHost', value)}/>
                    <Field label="控制台端口" value={compose.dashboardPort} onChange={(value) => update('dashboardPort', value)}/>
                </div>
            </div>
            <div className="panel">
                <p className="eyebrow">资源限制与控制台</p>
                <div className="field-grid">
                    <Field label="内存限制" value={compose.memoryLimit} onChange={(value) => update('memoryLimit', value)}/>
                    <Field label="CPU 限制" value={compose.cpuLimit} onChange={(value) => update('cpuLimit', value)}/>
                    <Field label="共享内存" value={compose.shmSize} onChange={(value) => update('shmSize', value)}/>
                    <Field label="控制台用户名" value={compose.dashboardUsername} onChange={(value) => update('dashboardUsername', value)}/>
                </div>
                <Field label="控制台密码" value={compose.dashboardPassword} secret onChange={(value) => update('dashboardPassword', value)}/>
                <label className="toggle"><input type="checkbox" checked={compose.dashboardEnabled} onChange={(event) => update('dashboardEnabled', event.target.checked)}/>启用控制台</label>
                <button className="primary" onClick={onSave}><Save size={16}/>保存部署配置</button>
            </div>
        </section>
    );
}

function ModelsPage(props: {
    model: ModelConfig;
    setModel: (value: ModelConfig) => void;
    selectedAux: string;
    setSelectedAux: (value: string) => void;
    providerPresets: ModelProviderPreset[];
    modelOptions: ModelOption[];
    modelListStatus: string;
    onFetchModels: () => void;
    onSave: () => void;
    onTest: () => void;
}) {
    const {model, setModel, selectedAux, setSelectedAux} = props;
    const aux = model.auxiliary?.[selectedAux] || {provider: 'auto', model: '', baseUrl: '', apiKey: '', timeout: 30, extraBody: {}};
    const setAux = (next: AuxModel) => setModel({...model, auxiliary: {...model.auxiliary, [selectedAux]: next}});
    const selectedProviderKey = providerKeyFor(model, props.providerPresets);
    const modelChoices = ensureCurrentModelOption(props.modelOptions, model.default);
    const applyProvider = (key: string) => {
        const preset = props.providerPresets.find((item) => item.key === key);
        if (!preset) return;
        const currentProviderKey = providerKeyFor(model, props.providerPresets);
        setModel({
            ...model,
            provider: preset.provider,
            baseUrl: preset.baseUrl,
            apiMode: preset.apiMode,
            default: preset.defaultModel,
            apiKey: currentProviderKey === preset.key ? model.apiKey : '',
        });
    };
    return (
        <section className="grid two">
            <div className="panel">
                <p className="eyebrow">主模型</p>
                <label className="field">
                    <span>供应商</span>
                    <select value={selectedProviderKey} onChange={(event) => applyProvider(event.target.value)}>
                        {props.providerPresets.map((preset) => <option key={preset.key} value={preset.key}>{preset.label}</option>)}
                    </select>
                </label>
                <label className="field">
                    <span>模型</span>
                    {modelChoices.length > 0 ? (
                        <select value={model.default} onChange={(event) => setModel({...model, default: event.target.value})}>
                            {modelChoices.map((item) => <option key={item.id} value={item.id}>{item.ownedBy ? `${item.id} · ${item.ownedBy}` : item.id}</option>)}
                        </select>
                    ) : (
                        <input value={model.default || ''} onChange={(event) => setModel({...model, default: event.target.value})}/>
                    )}
                </label>
                <Field label="接口地址" value={model.baseUrl} onChange={(value) => setModel({...model, baseUrl: value})}/>
                <Field label="API 模式" value={model.apiMode} onChange={(value) => setModel({...model, apiMode: value})}/>
                <Field label="API 密钥" value={model.apiKey} secret onChange={(value) => setModel({...model, apiKey: value})}/>
                <div className="actions model-actions">
                    <button className="ghost" onClick={props.onFetchModels}><RefreshCcw size={16}/>拉取模型列表</button>
                    {props.modelListStatus && <span className="inline-status">{props.modelListStatus}</span>}
                </div>
            </div>
            <div className="panel">
                <p className="eyebrow">辅助模型策略</p>
                <div className="segmented">
                    {[
                        ['auto', '自动'],
                        ['follow-main', '跟随主模型'],
                        ['custom', '分别配置'],
                    ].map(([mode, label]) => (
                        <button key={mode} className={model.auxiliaryMode === mode ? 'selected' : ''} onClick={() => setModel({...model, auxiliaryMode: mode})}>{label}</button>
                    ))}
                </div>
                <select value={selectedAux} onChange={(event) => setSelectedAux(event.target.value)}>
                    {Object.keys(auxLabels).map((key) => <option key={key} value={key}>{auxLabels[key]}</option>)}
                </select>
                <Field label="供应商" value={aux.provider} onChange={(value) => setAux({...aux, provider: value})}/>
                <Field label="模型" value={aux.model} onChange={(value) => setAux({...aux, model: value})}/>
                <Field label="接口地址" value={aux.baseUrl} onChange={(value) => setAux({...aux, baseUrl: value})}/>
                <Field label="API 密钥" value={aux.apiKey} secret onChange={(value) => setAux({...aux, apiKey: value})}/>
                <Field label="超时秒数" value={String(aux.timeout || '')} onChange={(value) => setAux({...aux, timeout: Number(value) || 0})}/>
                <div className="actions">
                    <button className="primary" onClick={props.onSave}><Save size={16}/>保存模型配置</button>
                    <button className="ghost test-button" onClick={props.onTest}><Activity size={16}/>测试模型</button>
                </div>
            </div>
        </section>
    );
}

function PlatformsPage(props: {
    env: EnvVar[];
    setEnv: (value: EnvVar[]) => void;
    qrData: string;
    qrStatus: string;
    onSaveEnv: () => void;
    onWeixinLogin: () => void;
    onCancelWeixin: () => void;
    onSaveWeCom: () => void;
}) {
    const set = (key: string, value: string) => props.setEnv(setEnvValue(props.env, key, value));
    return (
        <section className="grid two">
            <div className="panel">
                <p className="eyebrow">个人微信</p>
                <div className="qr-stage">
                    {props.qrData ? <QRCodeSVG value={props.qrData} size={184}/> : <QrCode size={120}/>}
                    <span>{props.qrStatus || '点击扫码登录绑定个人微信。'}</span>
                </div>
                <div className="actions">
                    <IconButton icon={QrCode} label="扫码登录" onClick={props.onWeixinLogin}/>
                    <IconButton icon={Square} label="取消" onClick={props.onCancelWeixin}/>
                </div>
                <Field label="私聊策略" value={envValue(props.env, 'WEIXIN_DM_POLICY') || 'open'} onChange={(value) => set('WEIXIN_DM_POLICY', value)}/>
                <Field label="群聊策略" value={envValue(props.env, 'WEIXIN_GROUP_POLICY') || 'open'} onChange={(value) => set('WEIXIN_GROUP_POLICY', value)}/>
                <button className="primary" onClick={props.onSaveEnv}><Save size={16}/>保存微信策略</button>
            </div>
            <div className="panel">
                <p className="eyebrow">企业微信 AI Bot WebSocket</p>
                <Field label="机器人 ID" value={envValue(props.env, 'WECOM_BOT_ID')} onChange={(value) => set('WECOM_BOT_ID', value)}/>
                <Field label="密钥" value={envValue(props.env, 'WECOM_SECRET')} secret onChange={(value) => set('WECOM_SECRET', value)}/>
                <Field label="WebSocket 地址" value={envValue(props.env, 'WECOM_WEBSOCKET_URL') || 'wss://openws.work.weixin.qq.com'} onChange={(value) => set('WECOM_WEBSOCKET_URL', value)}/>
                <div className="field-grid">
                    <Field label="私聊策略" value={envValue(props.env, 'WECOM_DM_POLICY') || 'open'} onChange={(value) => set('WECOM_DM_POLICY', value)}/>
                    <Field label="群聊策略" value={envValue(props.env, 'WECOM_GROUP_POLICY') || 'open'} onChange={(value) => set('WECOM_GROUP_POLICY', value)}/>
                </div>
                <button className="primary" onClick={props.onSaveWeCom}><Save size={16}/>保存企业微信配置</button>
            </div>
        </section>
    );
}

function ChannelsPage({channels, onHome, onTest}: { channels: ChannelFile; onHome: (platform: string, id: string) => void; onTest: (platform: string, id: string) => void }) {
    const rows = useMemo(() => Object.entries(channels.platforms || {}).flatMap(([platform, items]) => items.map((item) => ({platform, ...item}))), [channels]);
    return (
        <section className="panel">
            <p className="eyebrow">通道目录</p>
            <div className="table">
                {rows.length === 0 && <p className="muted">还没有发现通道。请先启动 Hermes，并从微信或企业微信发送一条消息。</p>}
                {rows.map((row) => (
                    <div className="table-row" key={`${row.platform}-${row.id}`}>
                        <code>{row.platform}</code>
                        <span>{row.name || row.id}</span>
                        <span>{row.type}</span>
                        <button onClick={() => onHome(row.platform, row.id)}>设为默认</button>
                        <button onClick={() => onTest(row.platform, row.id)}>测试</button>
                    </div>
                ))}
            </div>
        </section>
    );
}

function DiagnosticsPage({diagnostics, onRefresh}: { diagnostics: Diagnostic[]; onRefresh: () => void }) {
    return (
        <section className="panel">
            <div className="panel-head">
                <p className="eyebrow">系统检查</p>
                <button className="ghost" onClick={onRefresh}><RefreshCcw size={16}/>刷新</button>
            </div>
            <div className="diagnostics">
                {diagnostics.map((item) => (
                    <div className={`diagnostic ${item.status}`} key={item.id}>
                        {item.status === 'ok' ? <CheckCircle2 size={18}/> : <CircleAlert size={18}/>}
                        <div>
                            <strong>{item.label}</strong>
                            <span>{item.message}</span>
                        </div>
                    </div>
                ))}
            </div>
        </section>
    );
}

function AdvancedPage(props: { path: string; setPath: (value: string) => void; content: string; setContent: (value: string) => void; onLoad: () => void; onSave: () => void }) {
    return (
        <section className="panel">
            <p className="eyebrow">原始文件编辑器</p>
            <div className="advanced-toolbar">
                <select value={props.path} onChange={(event) => props.setPath(event.target.value)}>
                    <option value="data/config.yaml">data/config.yaml</option>
                    <option value="data/.env">data/.env</option>
                    <option value="docker-compose.override.yaml">docker-compose.override.yaml</option>
                </select>
                <button className="ghost" onClick={props.onLoad}>读取</button>
                <button className="primary" onClick={props.onSave}><Save size={16}/>保存</button>
            </div>
            <textarea className="editor" value={props.content} onChange={(event) => props.setContent(event.target.value)} spellCheck={false}/>
        </section>
    );
}

function Field({label, value, onChange, secret = false}: { label: string; value: string; onChange: (value: string) => void; secret?: boolean }) {
    return (
        <label className="field">
            <span>{label}</span>
            <input type={secret ? 'password' : 'text'} value={value || ''} onChange={(event) => onChange(event.target.value)}/>
        </label>
    );
}

function IconButton({icon: Icon, label, onClick, disabled = false}: { icon: typeof Play; label: string; onClick: () => void; disabled?: boolean }) {
    return <button className="icon-button" onClick={onClick} disabled={disabled} title={label}><Icon size={17}/><span>{label}</span></button>;
}

function Health({label, ok}: { label: string; ok: boolean }) {
    return <div className={`health ${ok ? 'ok' : 'warn'}`}>{ok ? <CheckCircle2 size={18}/> : <CircleAlert size={18}/>}<span>{label}</span></div>;
}

function titleFor(page: Page) {
    return nav.find((item) => item.id === page)?.label || 'Hermes Dock';
}

function containerStatusText(status?: string) {
    switch (status) {
        case 'running':
            return '运行中';
        case 'stopped':
            return '已停止';
        case 'missing':
            return '未创建';
        case 'unknown':
            return '未知';
        default:
            return '未知';
    }
}

function envValue(env: EnvVar[], key: string) {
    return env.find((item) => item.key === key)?.value || '';
}

function setEnvValue(env: EnvVar[], key: string, value: string) {
    const next = [...env];
    const index = next.findIndex((item) => item.key === key);
    if (index >= 0) {
        next[index] = {...next[index], value};
    } else {
        next.push({key, value, secret: /KEY|TOKEN|SECRET|PASSWORD|PASS|AUTH/i.test(key)});
    }
    return next;
}

function providerKeyFor(model: ModelConfig, presets: ModelProviderPreset[]) {
    const provider = (model.provider || '').toLowerCase();
    const baseUrl = (model.baseUrl || '').toLowerCase();
    if (provider === 'deepseek' || baseUrl.includes('api.deepseek.com')) {
        return 'deepseek';
    }
    if (provider === 'custom' && baseUrl.includes('dashscope.aliyuncs.com')) {
        return 'dashscope-payg';
    }
    return presets[0]?.key || 'dashscope-payg';
}

function ensureCurrentModelOption(options: ModelOption[], current: string) {
    if (!current) return options;
    if (options.some((item) => item.id === current)) return options;
    return [{id: current, ownedBy: ''}, ...options];
}

function toPlainModelConfig(model: ModelConfig): any {
    return JSON.parse(JSON.stringify(model));
}

export default App;
