import {type RefObject, useEffect, useMemo, useRef, useState} from 'react';
import {basicSetup} from 'codemirror';
import {yaml} from '@codemirror/lang-yaml';
import {HighlightStyle, StreamLanguage, syntaxHighlighting} from '@codemirror/language';
import {EditorState} from '@codemirror/state';
import {EditorView, keymap} from '@codemirror/view';
import {indentWithTab} from '@codemirror/commands';
import {gotoLine, openSearchPanel, search} from '@codemirror/search';
import {tags} from '@lezer/highlight';
import {
    Activity,
    Boxes,
    CheckCircle2,
    CircleAlert,
    Clipboard,
    CornerDownRight,
    ExternalLink,
    Eye,
    EyeOff,
    FileCog,
    Gauge,
    Loader2,
    MessageSquare,
    Play,
    QrCode,
    RefreshCcw,
    RotateCcw,
    Save,
    Search,
    Settings,
    Square,
    TerminalSquare,
    Trash2,
} from 'lucide-react';
import {QRCodeSVG} from 'qrcode.react';
import './App.css';
import {
    CancelWeixinLogin,
    FetchModelList,
    FetchProviderConfigModelList,
    GetAppState,
    OpenEndpoint,
    ReadTextFile,
    RebuildHermes,
    RestartHermes,
    SaveComposeSettings,
    SaveEnvironment,
    SaveModelConfig,
    SaveProviderConfig,
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

type Page = 'dashboard' | 'deploy' | 'providers' | 'models' | 'platforms' | 'channels' | 'advanced';

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
type ProviderEntry = {
    label: string;
    provider: string;
    baseUrl: string;
    apiMode: string;
    apiKey: string;
    modelListUrl: string;
    defaultModel: string;
    builtin: boolean;
    disabled: boolean;
};
type ProviderConfig = { providers: Record<string, ProviderEntry> };
type ModelListRequest = { providerId: string; providerKey: string; apiKey: string; baseUrl: string };
type ModelOption = { id: string; ownedBy: string };
type ChannelSummary = { id: string; name: string; type: string; thread_id?: string };
type ChannelFile = { updated_at: string; platforms: Record<string, ChannelSummary[]> };
type Notice = { type: 'ok' | 'error' | 'info'; message: string };
type RunOptions = { rebuildRequired?: boolean; afterSuccess?: () => void };
type AppState = {
    appVersion: string;
    instanceRoot: string;
    compose: ComposeSettings;
    environment: EnvVar[];
    model: ModelConfig;
    providers: ProviderConfig;
    channels: ChannelFile;
    dockerAvailable: boolean;
    composeAvailable: boolean;
    containerStatus: string;
};

const nav: Array<{ id: Page; label: string; icon: typeof Gauge }> = [
    {id: 'dashboard', label: '总览', icon: Gauge},
    {id: 'deploy', label: '部署', icon: Settings},
    {id: 'providers', label: '供应商', icon: Boxes},
    {id: 'models', label: '模型', icon: Boxes},
    {id: 'platforms', label: '平台绑定', icon: MessageSquare},
    {id: 'channels', label: '通道', icon: Activity},
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

const editorHighlight = HighlightStyle.define([
    {tag: tags.comment, color: '#75806f', fontStyle: 'italic'},
    {tag: tags.propertyName, color: '#2c6455', fontWeight: '600'},
    {tag: tags.variableName, color: '#2c6455', fontWeight: '600'},
    {tag: tags.string, color: '#7a4d12'},
    {tag: tags.number, color: '#7156a0'},
    {tag: tags.bool, color: '#7156a0'},
    {tag: tags.keyword, color: '#1c6a7a', fontWeight: '600'},
    {tag: tags.operator, color: '#68715f'},
    {tag: tags.punctuation, color: '#68715f'},
]);

const codeEditorTheme = EditorView.theme({
    '&': {
        height: '100%',
        color: '#20251f',
        backgroundColor: '#f7f2e8',
    },
    '&.cm-focused': {
        outline: 'none',
    },
    '.cm-scroller': {
        fontFamily: '"JetBrains Mono", "SF Mono", Menlo, Consolas, monospace',
        fontSize: '13px',
        lineHeight: '1.55',
    },
    '.cm-content': {
        minHeight: '560px',
        padding: '14px 0',
        caretColor: '#20251f',
    },
    '.cm-line': {
        padding: '0 16px',
    },
    '.cm-gutters': {
        backgroundColor: '#eee8da',
        color: '#7a7568',
        borderRight: '0',
    },
    '.cm-lineNumbers .cm-gutterElement': {
        minWidth: '38px',
        padding: '0 12px 0 8px',
    },
    '.cm-activeLine': {
        backgroundColor: '#ece5d6',
    },
    '.cm-activeLineGutter': {
        backgroundColor: '#e3dccd',
        color: '#20251f',
    },
    '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
        backgroundColor: '#cfe57f80',
    },
    '.cm-searchMatch': {
        backgroundColor: '#d8f26399',
        outline: '1px solid #98b84b',
    },
    '.cm-searchMatch-selected': {
        backgroundColor: '#c5ee44',
    },
    '.cm-panels': {
        backgroundColor: '#eee8da',
        color: '#20251f',
        borderTop: '0',
        borderBottom: '1px solid #ddd4c4',
    },
    '.cm-panel.cm-search': {
        display: 'flex',
        flexWrap: 'wrap',
        alignItems: 'center',
        gap: '8px',
        padding: '10px',
    },
    '.cm-panel.cm-search input': {
        width: '220px',
        minHeight: '32px',
        backgroundColor: '#fffaf0',
    },
    '.cm-panel.cm-search button': {
        minHeight: '30px',
        padding: '0 10px',
        borderRadius: '6px',
    },
});

const envLanguage = StreamLanguage.define<null>({
    startState: () => null,
    token(stream) {
        if (stream.sol()) {
            stream.eatSpace();
            if (stream.peek() === '#') {
                stream.skipToEnd();
                return 'comment';
            }
            if (stream.match('export')) {
                return 'keyword';
            }
            if (stream.match(/[A-Za-z_][A-Za-z0-9_]*/)) {
                return 'variableName';
            }
        }
        if (stream.peek() === '#') {
            stream.skipToEnd();
            return 'comment';
        }
        if (stream.peek() === '=') {
            stream.next();
            return 'operator';
        }
        if (stream.peek() === '"' || stream.peek() === "'") {
            const quote = stream.next();
            let escaped = false;
            while (!stream.eol()) {
                const next = stream.next();
                if (next === quote && !escaped) break;
                escaped = next === '\\' && !escaped;
                if (next !== '\\') escaped = false;
            }
            return 'string';
        }
        if (stream.match(/[^\s#]+/)) {
            return 'string';
        }
        stream.next();
        return null;
    },
});

const fallbackModelProviderPresets: ModelProviderPreset[] = [
    {
        key: 'dashscope-payg',
        label: 'DashScope 按量计费',
        provider: 'custom',
        baseUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
        apiMode: 'chat_completions',
        defaultModel: 'qwen3.7-max',
        modelListUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1/models',
    },
    {
        key: 'opencode-go',
        label: 'OpenCode Go',
        provider: 'custom',
        baseUrl: 'https://opencode.ai/zen/go/v1',
        apiMode: 'chat_completions',
        defaultModel: 'deepseek-v4-flash',
        modelListUrl: 'https://opencode.ai/zen/go/v1/models',
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

const fallbackProviderConfig: ProviderConfig = {
    providers: fallbackModelProviderPresets.reduce<Record<string, ProviderEntry>>((providers, preset) => {
        providers[preset.key] = {
            label: preset.label,
            provider: preset.provider,
            baseUrl: preset.baseUrl,
            apiMode: preset.apiMode,
            apiKey: '',
            modelListUrl: preset.modelListUrl,
            defaultModel: preset.defaultModel,
            builtin: true,
            disabled: false,
        };
        return providers;
    }, {}),
};

function App() {
    const [page, setPage] = useState<Page>('dashboard');
    const [state, setState] = useState<AppState | null>(null);
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

    useEffect(() => {
        if (page !== 'advanced') return;
        if (advancedDirty) return;
        loadAdvancedFile(advancedPath);
    }, [page, advancedPath]);

    useEffect(() => {
        if (!autoScrollLogs || !logRef.current) return;
        logRef.current.scrollTop = logRef.current.scrollHeight;
    }, [logs, autoScrollLogs]);

    async function refresh() {
        const next = await GetAppState();
        setState(next as AppState);
        setEnv((next as AppState).environment || []);
        setCompose((next as AppState).compose);
        setModel((next as AppState).model);
        const nextProviders = (next as AppState).providers || fallbackProviderConfig;
        setProviders(nextProviders);
        setSelectedProvider((current) => nextProviders.providers[current] ? current : firstProviderID(nextProviders));
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

    function changeAdvancedPath(path: string) {
        if (path === advancedPath) return;
        if (advancedDirty && !window.confirm('当前文件有未保存修改，切换后会丢失这些修改。是否继续？')) {
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
                    <div className={`status-pill ${statusClass}`}>
                        {state?.containerStatus === 'running' ? <CheckCircle2 size={16}/> : <CircleAlert size={16}/>}
                        {containerStatusText(state?.containerStatus)}
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
                    />
                )}

                {page === 'channels' && state && (
                    <ChannelsPage channels={state.channels} weixinHomeChannel={weixinHomeChannel} busy={!!busy} onRefresh={() => run('正在刷新通道', refresh)}
                                  onHome={(platform, id) => run('正在设置默认通道', () => SetHomeChannel(platform, id), {rebuildRequired: true})}
                                  onTest={(platform, id) => run('正在发送测试消息', () => SendTestMessage(platform, id, 'Hermes Dock 测试消息'))}/>
                )}

                {page === 'advanced' && (
                    <AdvancedPage
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
    onStart: () => void;
    onStop: () => void;
    onRestart: () => void;
    onRebuild: () => void;
    onLogs: () => void;
    onClearLogs: () => void;
    onCopyLogs: () => void;
    autoScrollLogs: boolean;
    setAutoScrollLogs: (value: boolean) => void;
    logRef: RefObject<HTMLPreElement>;
    onOpenEndpoint: (endpoint: 'dashboard' | 'gateway') => void;
    onOpenDeploy: () => void;
    onOpenPlatforms: () => void;
}) {
    const actionBusy = props.busy !== '';
    const dashboardURL = endpointURL(props.compose.dashboardHost, props.compose.dashboardPort);
    const gatewayURL = endpointURL(props.compose.gatewayHost, props.compose.gatewayPort);
    return (
        <section className="grid two">
            <div className="panel hero-panel">
                <div>
                    <p className="eyebrow">当前镜像</p>
                    <h2>{props.compose.image}</h2>
                    <div className="endpoint-grid">
                        <button className="endpoint-card" onClick={() => props.onOpenEndpoint('dashboard')}>
                            <span>控制台</span>
                            <strong>{dashboardURL}</strong>
                            <ExternalLink size={16}/>
                        </button>
                        <button className="endpoint-card" onClick={() => props.onOpenEndpoint('gateway')}>
                            <span>网关</span>
                            <strong>{gatewayURL}</strong>
                            <ExternalLink size={16}/>
                        </button>
                    </div>
                </div>
                <div className="actions">
                    <IconButton icon={Play} label="启动" onClick={props.onStart} disabled={actionBusy}/>
                    <IconButton icon={Square} label="停止" onClick={props.onStop} disabled={actionBusy}/>
                    <IconButton icon={RefreshCcw} label="重启" onClick={props.onRestart} disabled={actionBusy}/>
                    <IconButton icon={RotateCcw} label="重建" onClick={props.onRebuild} disabled={actionBusy}/>
                </div>
                {props.busy && <div className="busy"><Loader2 size={16} className="spin"/>{props.busy}</div>}
            </div>
            <div className="panel">
                <p className="eyebrow">就绪状态</p>
                <div className="health-list">
                    <Health label="Docker" ok={props.state.dockerAvailable} onClick={props.onOpenDeploy}/>
                    <Health label="Compose" ok={props.state.composeAvailable} onClick={props.onOpenDeploy}/>
                    <Health label="个人微信" ok={props.weixinBound} onClick={props.onOpenPlatforms}/>
                    <Health label="企业微信 AI Bot" ok={props.wecomBound} onClick={props.onOpenPlatforms}/>
                </div>
            </div>
            <div className="panel wide">
                <div className="log-head">
                    <p className="eyebrow">实时输出</p>
                    <div className="actions compact">
                        <button className="ghost" onClick={props.onLogs}><TerminalSquare size={16}/>刷新日志</button>
                        <button className="ghost icon-only" onClick={props.onCopyLogs} disabled={props.logs.length === 0} title="复制日志"><Clipboard size={16}/></button>
                        <button className="ghost icon-only" onClick={props.onClearLogs} disabled={props.logs.length === 0} title="清空日志"><Trash2 size={16}/></button>
                        <label className="mini-toggle"><input type="checkbox" checked={props.autoScrollLogs} onChange={(event) => props.setAutoScrollLogs(event.target.checked)}/>自动滚动</label>
                    </div>
                </div>
                <pre ref={props.logRef} className="logbox">{props.logs.length ? props.logs.join('\n') : '暂无命令输出。'}</pre>
            </div>
        </section>
    );
}

function DeployPage({compose, setCompose, busy, onSave}: { compose: ComposeSettings; setCompose: (value: ComposeSettings) => void; busy: boolean; onSave: () => void }) {
    const update = (key: keyof Omit<ComposeSettings, 'dashboardEnabled'>, value: string) => setCompose({...compose, dashboardEnabled: true, [key]: value});
    const portsValid = isPortValue(compose.gatewayPort) && isPortValue(compose.dashboardPort);
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
                <div className="setting-note">控制台默认启用</div>
                {!portsValid && <div className="form-warning">端口必须是 1-65535 的数字</div>}
                <button className="primary" onClick={onSave} disabled={busy || !portsValid}><Save size={16}/>保存部署配置</button>
            </div>
        </section>
    );
}

function ProvidersPage(props: {
    providers: ProviderConfig;
    setProviders: (value: ProviderConfig) => void;
    selectedProvider: string;
    setSelectedProvider: (value: string) => void;
    model: ModelConfig | null;
    busy: boolean;
    showApiKey: boolean;
    setShowApiKey: (value: boolean) => void;
    modelOptions: ModelOption[];
    modelListStatus: string;
    onFetchModels: (provider: ProviderEntry) => void;
    onSave: () => void;
}) {
    const ids = providerIDs(props.providers);
    const selectedID = props.providers.providers[props.selectedProvider] ? props.selectedProvider : ids[0];
    const selected = props.providers.providers[selectedID];
    const refs = selectedID ? providerReferenceLabels(props.model, selectedID) : [];
    const updateSelected = (next: ProviderEntry) => {
        props.setProviders({providers: {...props.providers.providers, [selectedID]: next}});
    };
    const addProvider = () => {
        const id = nextProviderID(props.providers, '自定义供应商');
        props.setProviders({
            providers: {
                ...props.providers.providers,
                [id]: {
                    label: '自定义供应商',
                    provider: 'custom',
                    baseUrl: '',
                    apiMode: 'chat_completions',
                    apiKey: '',
                    modelListUrl: '',
                    defaultModel: '',
                    builtin: false,
                    disabled: false,
                },
            },
        });
        props.setSelectedProvider(id);
    };
    const deleteSelected = () => {
        if (!selected || selected.builtin || refs.length > 0) return;
        const next = {...props.providers.providers};
        delete next[selectedID];
        props.setProviders({providers: next});
        props.setSelectedProvider(firstProviderID({providers: next}));
    };
    if (!selected) {
        return (
            <section className="panel">
                <p className="eyebrow">供应商</p>
                <button className="primary" onClick={addProvider}>新增供应商</button>
            </section>
        );
    }
    return (
        <section className="grid two provider-layout">
            <div className="panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">供应商</p>
                        <h2>连接配置</h2>
                    </div>
                    <button className="ghost" onClick={addProvider}>新增</button>
                </div>
                <div className="provider-list">
                    {ids.map((id) => {
                        const item = props.providers.providers[id];
                        const configured = item.apiKey.trim() !== '';
                        return (
                            <button key={id} className={`provider-list-item ${id === selectedID ? 'selected' : ''}`} onClick={() => props.setSelectedProvider(id)}>
                                <strong>{item.label || id}</strong>
                                <span>{configured ? '已配置密钥' : '未配置密钥'} · {item.disabled ? '已禁用' : '启用中'}</span>
                            </button>
                        );
                    })}
                </div>
            </div>
            <div className="panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">{selected.builtin ? '内置供应商' : '自定义供应商'}</p>
                        <h2>{selected.label || selectedID}</h2>
                    </div>
                    <label className="mini-toggle"><input type="checkbox" checked={!selected.disabled} onChange={(event) => updateSelected({...selected, disabled: !event.target.checked})}/>启用</label>
                </div>
                <div className="field-grid">
                    <Field label="显示名称" value={selected.label} onChange={(value) => updateSelected({...selected, label: value})}/>
                    <Field label="推荐默认模型" value={selected.defaultModel} onChange={(value) => updateSelected({...selected, defaultModel: value})}/>
                </div>
                <Field label="接口地址" value={selected.baseUrl} onChange={(value) => updateSelected({...selected, baseUrl: value})}/>
                <label className="field">
                    <span>API 模式</span>
                    <select value={selected.apiMode || 'chat_completions'} onChange={(event) => updateSelected({...selected, apiMode: event.target.value})}>
                        <option value="chat_completions">OpenAI Chat Completions</option>
                        <option value="anthropic_messages">Anthropic Messages</option>
                    </select>
                </label>
                <SecretField label="API 密钥" value={selected.apiKey} visible={props.showApiKey} setVisible={props.setShowApiKey} onChange={(value) => updateSelected({...selected, apiKey: value})}/>
                <Field label="模型列表地址" value={selected.modelListUrl} onChange={(value) => updateSelected({...selected, modelListUrl: value})}/>
                {props.modelOptions.length > 0 && (
                    <label className="field">
                        <span>从已拉取模型中选择推荐默认模型</span>
                        <select value={selected.defaultModel} onChange={(event) => updateSelected({...selected, defaultModel: event.target.value})}>
                            {ensureCurrentModelOption(props.modelOptions, selected.defaultModel).map((item) => <option key={item.id} value={item.id}>{item.ownedBy ? `${item.id} · ${item.ownedBy}` : item.id}</option>)}
                        </select>
                    </label>
                )}
                {refs.length > 0 && <div className="form-warning">正在被使用：{refs.join('、')}</div>}
                <div className="actions">
                    <button className="ghost" onClick={() => props.onFetchModels(selected)} disabled={props.busy || selected.apiKey.trim() === '' || selected.baseUrl.trim() === ''}><RefreshCcw size={16}/>拉取模型列表</button>
                    <button className="ghost" onClick={deleteSelected} disabled={props.busy || selected.builtin || refs.length > 0}><Trash2 size={16}/>删除</button>
                    <button className="primary" onClick={props.onSave} disabled={props.busy}><Save size={16}/>保存供应商配置</button>
                </div>
                {props.modelListStatus && <span className="inline-status">{props.modelListStatus}</span>}
            </div>
        </section>
    );
}

function ModelsPage(props: {
    model: ModelConfig;
    setModel: (value: ModelConfig) => void;
    selectedAux: string;
    setSelectedAux: (value: string) => void;
    providers: ProviderConfig;
    modelOptions: ModelOption[];
    modelListStatus: string;
    auxModelOptions: Record<string, ModelOption[]>;
    auxModelListStatus: string;
    busy: boolean;
    onFetchModels: () => void;
    onFetchAuxModels: (providerID: string) => void;
    onSave: () => void;
    onTest: () => void;
}) {
    const {model, setModel, selectedAux, setSelectedAux} = props;
    const aux = model.auxiliary?.[selectedAux] || {provider: 'auto', model: '', baseUrl: '', apiKey: '', timeout: 30, extraBody: {}};
    const setAux = (next: AuxModel) => setModel({...model, auxiliary: {...model.auxiliary, [selectedAux]: next}});
    const enabledProviders = providerIDs(props.providers).filter((id) => !props.providers.providers[id].disabled);
    const selectedProviderID = props.providers.providers[model.provider] ? model.provider : firstProviderID(props.providers);
    const selectedProvider = props.providers.providers[selectedProviderID];
    const selectedProviderOptionsKey = modelOptionKey(selectedProviderID);
    const modelChoices = ensureCurrentModelOption(props.modelOptions, model.default);
    const showModelSelect = props.modelOptions.length > 0;
    const customAuxiliary = model.auxiliaryMode === 'custom';
    const modelReady = !!selectedProvider && model.default.trim() !== '';
    const modelCanTest = !!selectedProvider && modelReady && !selectedProvider.disabled && selectedProvider.apiKey.trim() !== '';
    const selectedAuxProviderID = aux.provider && aux.provider !== 'auto' && props.providers.providers[aux.provider] ? aux.provider : selectedProviderID;
    const selectedAuxProvider = props.providers.providers[selectedAuxProviderID];
    const auxProviderOptionsKey = modelOptionKey(selectedAuxProviderID);
    const auxUsesMainProvider = auxProviderOptionsKey === selectedProviderOptionsKey;
    const auxProviderOptions = props.auxModelOptions[auxProviderOptionsKey] || (auxUsesMainProvider ? props.modelOptions : []);
    const auxCurrentModel = aux.model || selectedAuxProvider?.defaultModel || model.default;
    const auxModelChoices = ensureCurrentModelOption(auxProviderOptions, auxCurrentModel);
    const showAuxModelSelect = auxProviderOptions.length > 0;
    const auxProviderReady = !!selectedAuxProvider && !selectedAuxProvider.disabled && selectedAuxProvider.apiKey.trim() !== '';
    const applyProvider = (id: string) => {
        const provider = props.providers.providers[id];
        if (!provider) return;
        setModel({
            ...model,
            provider: id,
            default: model.provider === id ? model.default : provider.defaultModel,
        });
    };
    const setAuxiliaryMode = (mode: string) => {
        if (mode !== 'custom') {
            setModel({...model, auxiliaryMode: mode});
            return;
        }
        const initialized = {...model.auxiliary};
        for (const key of Object.keys(auxLabels)) {
            const current = initialized[key] || {provider: '', model: '', baseUrl: '', apiKey: '', timeout: 30, extraBody: {}};
            const useCurrentProvider = current.provider && current.provider !== 'auto';
            const currentProviderID = useCurrentProvider && props.providers.providers[current.provider] ? current.provider : selectedProviderID;
            initialized[key] = {
                ...current,
                provider: currentProviderID,
                model: current.model || model.default,
                baseUrl: '',
                apiKey: '',
                timeout: current.timeout || 30,
                extraBody: current.extraBody || {},
            };
        }
        setModel({...model, auxiliaryMode: mode, auxiliary: initialized});
    };
    const applyAuxProvider = (id: string) => {
        const provider = props.providers.providers[id];
        if (!provider) return;
        setAux({
            ...aux,
            provider: id,
            model: provider.defaultModel,
            baseUrl: '',
            apiKey: '',
            timeout: aux.timeout || 30,
            extraBody: aux.extraBody || {},
        });
    };
    const setAuxModel = (value: string) => {
        setAux({
            ...aux,
            provider: selectedAuxProviderID,
            model: value,
            baseUrl: '',
            apiKey: '',
            timeout: aux.timeout || 30,
            extraBody: aux.extraBody || {},
        });
    };
    return (
        <section className="grid two">
            <div className="panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">主模型</p>
                        <h2>选择供应商和模型</h2>
                    </div>
                </div>
                <div className="provider-grid">
                    {enabledProviders.map((id) => {
                        const provider = props.providers.providers[id];
                        return (
                        <button key={id} className={`provider-card ${selectedProviderID === id ? 'selected' : ''}`} onClick={() => applyProvider(id)}>
                            <strong>{provider.label}</strong>
                            <span>{provider.apiKey.trim() === '' ? '未配置密钥' : provider.defaultModel || '手动填写模型'}</span>
                        </button>
                        );
                    })}
                </div>
                {selectedProvider?.disabled && <div className="form-warning">当前供应商已禁用，请重新选择或在供应商页启用。</div>}
                {selectedProvider && selectedProvider.apiKey.trim() === '' && <div className="form-warning">当前供应商未配置 API 密钥，可以保存模型选择，但不能测试或调用。</div>}
                <label className="field">
                    <span>模型</span>
                    {showModelSelect ? (
                        <select value={model.default} onChange={(event) => setModel({...model, default: event.target.value})}>
                            {model.default.trim() === '' && <option value="">请选择模型</option>}
                            {modelChoices.map((item) => <option key={item.id} value={item.id}>{item.ownedBy ? `${item.id} · ${item.ownedBy}` : item.id}</option>)}
                        </select>
                    ) : (
                        <input value={model.default || ''} onChange={(event) => setModel({...model, default: event.target.value})}/>
                    )}
                </label>
                <div className="actions model-actions">
                    <button className="ghost" onClick={props.onFetchModels} disabled={props.busy || !selectedProvider || selectedProvider.disabled || selectedProvider.apiKey.trim() === ''}><RefreshCcw size={16}/>拉取模型列表</button>
                    {props.modelListStatus && <span className="inline-status">{props.modelListStatus}</span>}
                </div>
                <div className="actions">
                    <button className="primary" onClick={props.onSave} disabled={props.busy || !modelReady}><Save size={16}/>保存模型配置</button>
                    <button className="ghost test-button" onClick={props.onTest} disabled={props.busy || !modelCanTest}><Activity size={16}/>测试模型</button>
                </div>
            </div>
            <div className="panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">辅助模型</p>
                        <h2>选择策略</h2>
                    </div>
                </div>
                <div className="segmented">
                    {[
                        ['auto', '自动'],
                        ['follow-main', '跟随主模型'],
                        ['custom', '分别配置'],
                    ].map(([mode, label]) => (
                        <button key={mode} className={model.auxiliaryMode === mode ? 'selected' : ''} onClick={() => setAuxiliaryMode(mode)}>{label}</button>
                    ))}
                </div>
                {!customAuxiliary && (
                    <div className="mode-summary">
                        <strong>{model.auxiliaryMode === 'follow-main' ? '使用主模型' : '由 Hermes 自动选择'}</strong>
                        <span>{model.auxiliaryMode === 'follow-main' ? model.default : '适合大多数新手场景'}</span>
                    </div>
                )}
                {customAuxiliary && (
                    <>
                        <label className="field">
                            <span>用途</span>
                            <select value={selectedAux} onChange={(event) => setSelectedAux(event.target.value)}>
                                {Object.keys(auxLabels).map((key) => <option key={key} value={key}>{auxLabels[key]}</option>)}
                            </select>
                        </label>
                        <div className="provider-grid compact">
                            {enabledProviders.map((id) => {
                                const provider = props.providers.providers[id];
                                return (
                                <button key={id} className={`provider-card ${selectedAuxProviderID === id ? 'selected' : ''}`} onClick={() => applyAuxProvider(id)}>
                                    <strong>{provider.label}</strong>
                                    <span>{provider.apiKey.trim() === '' ? '未配置密钥' : provider.defaultModel || '手动填写模型'}</span>
                                </button>
                                );
                            })}
                        </div>
                        {selectedAuxProvider?.disabled && <div className="form-warning">该辅助模型供应商已禁用。</div>}
                        {selectedAuxProvider && selectedAuxProvider.apiKey.trim() === '' && <div className="form-warning">该供应商未配置 API 密钥。</div>}
                        <label className="field">
                            <span>模型</span>
                            {showAuxModelSelect ? (
                                <select value={auxCurrentModel} onChange={(event) => setAuxModel(event.target.value)}>
                                    {auxCurrentModel.trim() === '' && <option value="">请选择模型</option>}
                                    {auxModelChoices.map((item) => <option key={item.id} value={item.id}>{item.ownedBy ? `${item.id} · ${item.ownedBy}` : item.id}</option>)}
                                </select>
                            ) : (
                                <input value={aux.model || ''} onChange={(event) => setAuxModel(event.target.value)}/>
                            )}
                        </label>
                        <div className="actions model-actions">
                            <button className="ghost" onClick={() => props.onFetchAuxModels(selectedAuxProviderID)} disabled={props.busy || !auxProviderReady}><RefreshCcw size={16}/>拉取模型列表</button>
                            {props.auxModelListStatus && <span className="inline-status">{props.auxModelListStatus}</span>}
                        </div>
                    </>
                )}
            </div>
        </section>
    );
}

function PlatformsPage(props: {
    env: EnvVar[];
    setEnv: (value: EnvVar[]) => void;
    qrData: string;
    qrStatus: string;
    busy: boolean;
    onSaveEnv: () => void;
    onWeixinLogin: () => void;
    onCancelWeixin: () => void;
    onSaveWeCom: () => void;
}) {
    const set = (key: string, value: string) => props.setEnv(setEnvValue(props.env, key, value));
    const setWeixinPolicy = (key: string, value: string) => {
        let next = setEnvValue(props.env, key, value);
        if (key === 'WEIXIN_DM_POLICY') {
            next = setEnvValue(next, 'WEIXIN_ALLOW_ALL_USERS', value === 'open' ? 'true' : 'false');
        }
        props.setEnv(next);
    };
    const weixinDMPolicy = envValue(props.env, 'WEIXIN_DM_POLICY') || 'open';
    const weixinGroupPolicy = envValue(props.env, 'WEIXIN_GROUP_POLICY') || 'open';
    const wecomDMPolicy = envValue(props.env, 'WECOM_DM_POLICY') || 'open';
    const wecomGroupPolicy = envValue(props.env, 'WECOM_GROUP_POLICY') || 'open';
    return (
        <section className="grid two">
            <div className="panel">
                <p className="eyebrow">个人微信</p>
                <div className="qr-stage">
                    {props.qrData ? <QRCodeSVG value={props.qrData} size={184}/> : <QrCode size={120}/>}
                    <span>{props.qrStatus || '点击扫码登录绑定个人微信。'}</span>
                </div>
                <div className="actions">
                    <IconButton icon={QrCode} label="扫码登录" onClick={props.onWeixinLogin} disabled={props.busy}/>
                    <IconButton icon={Square} label="取消" onClick={props.onCancelWeixin} disabled={props.busy}/>
                </div>
                <PolicySelect label="私聊策略" value={weixinDMPolicy} onChange={(value) => setWeixinPolicy('WEIXIN_DM_POLICY', value)}/>
                {weixinDMPolicy === 'allowlist' && <Field label="允许用户" value={envValue(props.env, 'WEIXIN_ALLOWED_USERS')} onChange={(value) => set('WEIXIN_ALLOWED_USERS', value)}/>}
                <PolicySelect label="群聊策略" value={weixinGroupPolicy} onChange={(value) => set('WEIXIN_GROUP_POLICY', value)}/>
                {weixinGroupPolicy === 'allowlist' && <Field label="允许群用户" value={envValue(props.env, 'WEIXIN_GROUP_ALLOWED_USERS')} onChange={(value) => set('WEIXIN_GROUP_ALLOWED_USERS', value)}/>}
                <button className="primary" onClick={props.onSaveEnv} disabled={props.busy}><Save size={16}/>保存微信策略</button>
            </div>
            <div className="panel">
                <p className="eyebrow">企业微信 AI Bot WebSocket</p>
                <Field label="机器人 ID" value={envValue(props.env, 'WECOM_BOT_ID')} onChange={(value) => set('WECOM_BOT_ID', value)}/>
                <Field label="密钥" value={envValue(props.env, 'WECOM_SECRET')} secret onChange={(value) => set('WECOM_SECRET', value)}/>
                <Field label="WebSocket 地址" value={envValue(props.env, 'WECOM_WEBSOCKET_URL') || 'wss://openws.work.weixin.qq.com'} onChange={(value) => set('WECOM_WEBSOCKET_URL', value)}/>
                <div className="field-grid">
                    <PolicySelect label="私聊策略" value={wecomDMPolicy} onChange={(value) => set('WECOM_DM_POLICY', value)}/>
                    <PolicySelect label="群聊策略" value={wecomGroupPolicy} onChange={(value) => set('WECOM_GROUP_POLICY', value)}/>
                </div>
                {wecomDMPolicy === 'allowlist' && <Field label="允许用户" value={envValue(props.env, 'WECOM_ALLOWED_USERS')} onChange={(value) => set('WECOM_ALLOWED_USERS', value)}/>}
                {wecomGroupPolicy === 'allowlist' && <Field label="允许群用户" value={envValue(props.env, 'WECOM_GROUP_ALLOWED_USERS')} onChange={(value) => set('WECOM_GROUP_ALLOWED_USERS', value)}/>}
                <button className="primary" onClick={props.onSaveWeCom} disabled={props.busy}><Save size={16}/>保存企业微信配置</button>
            </div>
        </section>
    );
}

function ChannelsPage({channels, weixinHomeChannel, busy, onRefresh, onHome, onTest}: {
    channels: ChannelFile;
    weixinHomeChannel: string;
    busy: boolean;
    onRefresh: () => void;
    onHome: (platform: string, id: string) => void;
    onTest: (platform: string, id: string) => void;
}) {
    const rows = useMemo(() => Object.entries(channels.platforms || {}).flatMap(([platform, items]) => items.map((item) => ({platform, ...item}))), [channels]);
    return (
        <section className="panel">
            <div className="section-head">
                <div>
                    <p className="eyebrow">通道目录</p>
                    <h2>可用会话</h2>
                </div>
                <button className="ghost" onClick={onRefresh} disabled={busy}><RefreshCcw size={16}/>刷新</button>
            </div>
            <div className="table">
                {rows.length === 0 && <p className="muted">还没有发现通道。请先启动 Hermes，并从微信或企业微信发送一条消息。</p>}
                {rows.map((row) => (
                    <div className="table-row" key={`${row.platform}-${row.id}`}>
                        <code>{row.platform}</code>
                        <span>{row.name || row.id}{row.platform === 'weixin' && row.id === weixinHomeChannel && <b className="home-badge">默认</b>}</span>
                        <span>{row.type}</span>
                        {row.platform === 'weixin' ? (
                            <button onClick={() => onHome(row.platform, row.id)} disabled={busy || row.id === weixinHomeChannel}>{row.id === weixinHomeChannel ? '已默认' : '设为默认'}</button>
                        ) : <span className="muted">-</span>}
                        <button onClick={() => onTest(row.platform, row.id)} disabled={busy}>测试</button>
                    </div>
                ))}
            </div>
        </section>
    );
}

function AdvancedPage(props: { path: string; setPath: (value: string) => void; content: string; setContent: (value: string) => void; status: string; dirty: boolean; busy: boolean; onSave: () => void }) {
    const [editorView, setEditorView] = useState<EditorView | null>(null);
    const languageLabel = props.path.endsWith('.env') ? '.env' : 'YAML';

    return (
        <section className="panel">
            <div className="section-head">
                <div>
                    <p className="eyebrow">原始文件编辑器</p>
                    <h2>{props.path}</h2>
                </div>
                <span className={`inline-status ${props.dirty ? 'dirty' : ''}`}>{props.dirty ? '有未保存修改' : props.status}</span>
            </div>
            <div className="advanced-toolbar">
                <select value={props.path} onChange={(event) => props.setPath(event.target.value)}>
                    <option value="data/config.yaml">data/config.yaml</option>
                    <option value="data/.env">data/.env</option>
                    <option value="docker-compose.override.yaml">docker-compose.override.yaml</option>
                </select>
                <div className="editor-actions">
                    <span className="language-badge">{languageLabel}</span>
                    <button type="button" className="ghost" onClick={() => editorView && openSearchPanel(editorView)} disabled={!editorView} title="搜索">
                        <Search size={16}/>搜索
                    </button>
                    <button type="button" className="ghost" onClick={() => editorView && gotoLine(editorView)} disabled={!editorView} title="跳转到行">
                        <CornerDownRight size={16}/>跳行
                    </button>
                    <button className="primary" onClick={props.onSave} disabled={props.busy || !props.dirty}><Save size={16}/>保存</button>
                </div>
            </div>
            <CodeEditor path={props.path} value={props.content} onChange={props.setContent} onReady={setEditorView}/>
        </section>
    );
}

function CodeEditor(props: { path: string; value: string; onChange: (value: string) => void; onReady: (view: EditorView | null) => void }) {
    const hostRef = useRef<HTMLDivElement | null>(null);
    const viewRef = useRef<EditorView | null>(null);
    const onChangeRef = useRef(props.onChange);
    const syncingRef = useRef(false);

    useEffect(() => {
        onChangeRef.current = props.onChange;
    }, [props.onChange]);

    useEffect(() => {
        if (!hostRef.current) return;
        const language = props.path.endsWith('.env') ? envLanguage : yaml();
        const view = new EditorView({
            parent: hostRef.current,
            state: EditorState.create({
                doc: props.value,
                extensions: [
                    basicSetup,
                    keymap.of([indentWithTab]),
                    search({top: true}),
                    language,
                    syntaxHighlighting(editorHighlight),
                    codeEditorTheme,
                    EditorView.lineWrapping,
                    EditorView.updateListener.of((update) => {
                        if (update.docChanged && !syncingRef.current) {
                            onChangeRef.current(update.state.doc.toString());
                        }
                    }),
                ],
            }),
        });
        viewRef.current = view;
        props.onReady(view);
        return () => {
            props.onReady(null);
            view.destroy();
            viewRef.current = null;
        };
    }, [props.path]);

    useEffect(() => {
        const view = viewRef.current;
        if (!view) return;
        const current = view.state.doc.toString();
        if (props.value === current) return;
        syncingRef.current = true;
        view.dispatch({
            changes: {from: 0, to: current.length, insert: props.value},
        });
        syncingRef.current = false;
    }, [props.value]);

    return <div className="code-editor" ref={hostRef}/>;
}

function PolicySelect({label, value, onChange}: { label: string; value: string; onChange: (value: string) => void }) {
    return (
        <label className="field">
            <span>{label}</span>
            <select value={value || 'open'} onChange={(event) => onChange(event.target.value)}>
                <option value="open">开放</option>
                <option value="allowlist">指定名单</option>
                <option value="closed">关闭</option>
            </select>
        </label>
    );
}

function SecretField(props: { label: string; value: string; visible: boolean; setVisible: (value: boolean) => void; onChange: (value: string) => void }) {
    return (
        <label className="field">
            <span>{props.label}</span>
            <div className="secret-input">
                <input type={props.visible ? 'text' : 'password'} value={props.value || ''} onChange={(event) => props.onChange(event.target.value)}/>
                <button type="button" onClick={() => props.setVisible(!props.visible)} title={props.visible ? '隐藏密钥' : '显示密钥'}>
                    {props.visible ? <EyeOff size={16}/> : <Eye size={16}/>}
                </button>
            </div>
        </label>
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

function Health({label, ok, onClick}: { label: string; ok: boolean; onClick?: () => void }) {
    return <button className={`health ${ok ? 'ok' : 'warn'}`} onClick={onClick}>{ok ? <CheckCircle2 size={18}/> : <CircleAlert size={18}/>}<span>{label}</span></button>;
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

function endpointURL(host: string, port: string) {
    const localHost = !host || host === '0.0.0.0' || host === '::' ? '127.0.0.1' : host;
    return `http://${localHost}:${port}`;
}

function isPortValue(value: string) {
    if (!/^\d+$/.test(value)) return false;
    const port = Number(value);
    return port >= 1 && port <= 65535;
}

function doneLabel(label: string) {
    return label.replace(/^正在/, '已');
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

function providerIDs(config: ProviderConfig) {
    return Object.keys(config.providers || {}).sort((left, right) => {
        const a = config.providers[left];
        const b = config.providers[right];
        if (a.builtin !== b.builtin) return a.builtin ? -1 : 1;
        return (a.label || left).localeCompare(b.label || right, 'zh-Hans-CN');
    });
}

function firstProviderID(config: ProviderConfig) {
    return providerIDs(config)[0] || 'dashscope-payg';
}

function providerReferenceLabels(model: ModelConfig | null, providerID: string) {
    const refs: string[] = [];
    if (!model) return refs;
    if (model.provider === providerID) refs.push('主模型');
    for (const [key, aux] of Object.entries(model.auxiliary || {})) {
        if (aux.provider === providerID) refs.push(`辅助模型：${auxLabels[key] || key}`);
    }
    return refs;
}

function nextProviderID(config: ProviderConfig, label: string) {
    const base = slugProviderID(label) || 'custom-provider';
    let id = base;
    let index = 2;
    while (config.providers[id]) {
        id = `${base}-${index}`;
        index += 1;
    }
    return id;
}

function slugProviderID(label: string) {
    const ascii = label.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
    return ascii ? `custom-${ascii}` : 'custom-provider';
}

function ensureCurrentModelOption(options: ModelOption[], current: string) {
    if (!current) return options;
    if (options.some((item) => item.id === current)) return options;
    return [{id: current, ownedBy: ''}, ...options];
}

function modelOptionKey(providerID: string) {
    return providerID || 'dashscope-payg';
}

function toPlainModelConfig(model: ModelConfig): any {
    const next = JSON.parse(JSON.stringify(model)) as ModelConfig;
    next.baseUrl = '';
    next.apiMode = '';
    next.apiKey = '';
    for (const aux of Object.values(next.auxiliary || {})) {
        aux.baseUrl = '';
        aux.apiKey = '';
    }
    return next;
}

function toPlainProviderConfig(providers: ProviderConfig): any {
    return JSON.parse(JSON.stringify(providers));
}

export default App;
