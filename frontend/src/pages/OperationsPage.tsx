import type {RefObject} from 'react';
import {useEffect, useState} from 'react';
import {Clipboard, ExternalLink, Loader2, Play, RefreshCcw, RotateCcw, Square, TerminalSquare, Trash2} from 'lucide-react';
import {QRCodeSVG} from 'qrcode.react';
import {AdvancedPage} from './AdvancedPage';
import {ChannelsPage} from './ChannelsPage';
import {DeployPage} from './DeployPage';
import {Health} from '../components/primitives';
import type {AppState, ComposeSettings, OperationsTab, ProxySettings, WebSettingsRequest, WebStatus} from '../types';
import {containerStatusText, endpointURL, isPortValue, profileStatusText, statusClassName} from '../utils';

export function OperationsPage(props: {
    tab: OperationsTab;
    setTab: (value: OperationsTab) => void;
    state: AppState;
    compose: ComposeSettings;
    proxy: ProxySettings;
    setCompose: (value: ComposeSettings) => void;
    setProxy: (value: ProxySettings) => void;
    deployDirty: boolean;
    needsRebuild: boolean;
    busy: string;
    logs: string[];
    activeProfileName: string;
    weixinBound: boolean;
    wecomBound: boolean;
    feishuBound: boolean;
    weixinHomeChannel: string;
    advancedOptions: Array<{ value: string; label: string }>;
    advancedPath: string;
    setAdvancedPath: (value: string) => void;
    advancedOpen: boolean;
    setAdvancedOpen: (value: boolean) => void;
    advancedContent: string;
    setAdvancedContent: (value: string) => void;
    advancedStatus: string;
    advancedDirty: boolean;
    autoScrollLogs: boolean;
    setAutoScrollLogs: (value: boolean) => void;
    logRef: RefObject<HTMLPreElement>;
    logsFollowing: boolean;
    lastOperationError: string;
    onStart: () => void;
    onStop: () => void;
    onRestart: () => void;
    onRebuild: () => void;
    onLogs: () => void;
    onClearLogs: () => void;
    onCopyLogs: () => void;
    onOpenEndpoint: (endpoint: 'dashboard' | 'gateway') => void;
    onOpenAssistantPlatforms: () => void;
    onSaveDeploy: () => void;
    onDiscardDeploy: () => void;
    onRefreshChannels: () => void;
    channelActionStatus: Record<string, string>;
    onHomeChannel: (platform: string, id: string) => void;
    onTestChannel: (platform: string, id: string) => void;
    onSaveAdvanced: () => void;
    onFactoryReset: () => Promise<void>;
    resetConfirmPhrase: string;
    webStatus: WebStatus;
    onSaveWebSettings: (settings: WebSettingsRequest) => Promise<boolean>;
    onChangeWebPassword: (oldPassword: string, newPassword: string) => Promise<boolean>;
    onResetWebPassword: () => Promise<boolean>;
}) {
    const allTabs: Array<{ id: OperationsTab; label: string }> = [
        {id: 'status', label: '运行控制'},
        {id: 'deploy', label: '部署参数'},
        {id: 'channels', label: '通道诊断'},
        {id: 'advanced', label: '高级编辑'},
    ];
    const tabs = allTabs;
    return (
        <section className="operations-stack">
            <div className="ops-tabs">
                {tabs.map((item) => <button key={item.id} className={props.tab === item.id ? 'selected' : ''} onClick={() => props.setTab(item.id)}>{item.label}</button>)}
            </div>
            {props.tab === 'status' && (
                <StatusAndLogs
                    state={props.state}
                    compose={props.compose}
                    needsRebuild={props.needsRebuild}
                    busy={props.busy}
                    logs={props.logs}
                    weixinBound={props.weixinBound}
                    wecomBound={props.wecomBound}
                    feishuBound={props.feishuBound}
                    autoScrollLogs={props.autoScrollLogs}
                    setAutoScrollLogs={props.setAutoScrollLogs}
                    logRef={props.logRef}
                    logsFollowing={props.logsFollowing}
                    lastOperationError={props.lastOperationError}
                    onStart={props.onStart}
                    onStop={props.onStop}
                    onRestart={props.onRestart}
                    onRebuild={props.onRebuild}
                    onLogs={props.onLogs}
                    onClearLogs={props.onClearLogs}
                    onCopyLogs={props.onCopyLogs}
                    onOpenEndpoint={props.onOpenEndpoint}
                    webStatus={props.webStatus}
                    onSaveWebSettings={props.onSaveWebSettings}
                    onChangeWebPassword={props.onChangeWebPassword}
                    onResetWebPassword={props.onResetWebPassword}
                />
            )}
            {props.tab === 'deploy' && <DeployPage compose={props.compose} proxy={props.proxy} setCompose={props.setCompose} setProxy={props.setProxy} dirty={props.deployDirty} busy={!!props.busy} onSave={props.onSaveDeploy} onDiscard={props.onDiscardDeploy}/>}
            {props.tab === 'channels' && (
                <div className="operations-context">
                    <ChannelsPage
                        channels={props.state.channels}
                        activeProfileName={props.activeProfileName}
                        hasPlatformBinding={props.weixinBound || props.wecomBound || props.feishuBound}
                        weixinHomeChannel={props.weixinHomeChannel}
                        busy={!!props.busy}
                        actionStatus={props.channelActionStatus}
                        onRefresh={props.onRefreshChannels}
                        onOpenAssistantPlatforms={props.onOpenAssistantPlatforms}
                        onHome={props.onHomeChannel}
                        onTest={props.onTestChannel}
                    />
                </div>
            )}
            {props.tab === 'advanced' && (
                <div className="operations-context">
                    <div className="setting-note">高级编辑是专家逃生口。默认编辑当前助手：{props.activeProfileName}；模型、部署和平台等结构化页面保存时，可能覆盖这里的部分字段。</div>
                    <AdvancedPage
                        options={props.advancedOptions}
                        path={props.advancedPath}
                        setPath={props.setAdvancedPath}
                        open={props.advancedOpen}
                        setOpen={props.setAdvancedOpen}
                        content={props.advancedContent}
                        setContent={props.setAdvancedContent}
                        status={props.advancedStatus}
                        dirty={props.advancedDirty}
                        busy={!!props.busy}
                        onSave={props.onSaveAdvanced}
                        onFactoryReset={props.onFactoryReset}
                        resetConfirmPhrase={props.resetConfirmPhrase}
                    />
                </div>
            )}
        </section>
    );
}

function StatusAndLogs(props: {
    state: AppState;
    compose: ComposeSettings;
    needsRebuild: boolean;
    busy: string;
    logs: string[];
    weixinBound: boolean;
    wecomBound: boolean;
    feishuBound: boolean;
    autoScrollLogs: boolean;
    setAutoScrollLogs: (value: boolean) => void;
    logRef: RefObject<HTMLPreElement>;
    logsFollowing: boolean;
    lastOperationError: string;
    onStart: () => void;
    onStop: () => void;
    onRestart: () => void;
    onRebuild: () => void;
    onLogs: () => void;
    onClearLogs: () => void;
    onCopyLogs: () => void;
    onOpenEndpoint: (endpoint: 'dashboard' | 'gateway') => void;
    webStatus: WebStatus;
    onSaveWebSettings: (settings: WebSettingsRequest) => Promise<boolean>;
    onChangeWebPassword: (oldPassword: string, newPassword: string) => Promise<boolean>;
    onResetWebPassword: () => Promise<boolean>;
}) {
    const actionBusy = props.busy !== '';
    const dashboardURL = endpointURL(props.compose.dashboardHost, props.compose.dashboardPort);
    const gatewayURL = endpointURL(props.compose.gatewayHost, props.compose.gatewayPort);
    const endpointsReady = props.state.containerStatus === 'running';
    const dashboardReady = endpointsReady && isPortValue(props.compose.dashboardPort);
    const gatewayReady = endpointsReady && isPortValue(props.compose.gatewayPort);
    const profiles = props.state.profiles?.profiles || [];
    const profileStatuses = props.state.profileStatus?.profiles || {};
    const runningProfiles = profiles.filter((profile) => profileStatuses[profile.id]?.state === 'running').length;
    const startingProfiles = profiles.filter((profile) => profileStatuses[profile.id]?.state === 'starting').length;
    const failedProfiles = profiles.filter((profile) => profileStatuses[profile.id]?.state === 'failed').length;
    const notConfiguredProfiles = profiles.filter((profile) => profileStatuses[profile.id]?.state === 'not_configured').length;
    const latestLogs = props.logs.slice(-4);
    const summary = operationSummary(props.state.containerStatus, props.needsRebuild, runningProfiles, startingProfiles, failedProfiles, notConfiguredProfiles);
    return (
        <section className="operations-control">
            <div className="panel operations-hero">
                <div>
                    <p className="eyebrow">运行控制</p>
                    <h2>{summary.title}</h2>
                    <p>{summary.detail}</p>
                </div>
                <div className="operation-primary-actions">
                    {props.needsRebuild ? (
                        <button className="primary no-margin" onClick={props.onRebuild} disabled={actionBusy}><RotateCcw size={16}/>应用并重建</button>
                    ) : props.state.containerStatus === 'running' ? (
                        <button className="ghost" onClick={props.onRestart} disabled={actionBusy}><RefreshCcw size={16}/>重启容器</button>
                    ) : (
                        <button className="primary no-margin" onClick={props.onStart} disabled={actionBusy}><Play size={16}/>启动容器</button>
                    )}
                    {!props.needsRebuild && <button className="ghost" onClick={props.onRebuild} disabled={actionBusy}><RotateCcw size={16}/>应用并重建</button>}
                    <button className="ghost" onClick={props.onStop} disabled={actionBusy || props.state.containerStatus !== 'running'}><Square size={16}/>停止</button>
                </div>
                {props.busy && <div className="busy"><Loader2 size={16} className="spin"/>{props.busy}</div>}
                {props.lastOperationError && <div className="operation-error">最近错误：{props.lastOperationError}</div>}
            </div>

            <div className="operation-strip">
                <div className="operation-mini-card">
                    <span>容器</span>
                    <strong>{containerStatusText(props.state.containerStatus)}</strong>
                </div>
                <div className="operation-mini-card">
                    <span>助手</span>
                    <strong>{runningProfiles}/{profiles.length} 运行中</strong>
                </div>
                <button className="operation-mini-card clickable" onClick={() => props.onOpenEndpoint('dashboard')} disabled={!dashboardReady} title={dashboardReady ? '打开控制台' : '容器运行且端口有效后可打开'}>
                    <span>控制台</span>
                    <strong>{dashboardURL}</strong>
                    <ExternalLink size={15}/>
                </button>
                <button className="operation-mini-card clickable" onClick={() => props.onOpenEndpoint('gateway')} disabled={!gatewayReady} title={gatewayReady ? '打开网关' : '容器运行且端口有效后可打开'}>
                    <span>网关</span>
                    <strong>{gatewayURL}</strong>
                    <ExternalLink size={15}/>
                </button>
            </div>

            <WebManagementCard
                status={props.webStatus}
                busy={actionBusy}
                onSave={props.onSaveWebSettings}
                onChangePassword={props.onChangeWebPassword}
                onResetPassword={props.onResetWebPassword}
            />

            <div className="operation-diagnostics">
                <Health label="Docker" ok={props.state.dockerAvailable}/>
                <Health label="Compose" ok={props.state.composeAvailable}/>
                <Health label="个人微信" ok={props.weixinBound}/>
                <Health label="企业微信 AI Bot" ok={props.wecomBound}/>
                <Health label="飞书 / Lark" ok={props.feishuBound}/>
            </div>

            <div className="panel operations-compact-panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">助手运行状态</p>
                        <h2>运行概览</h2>
                    </div>
                </div>
                <div className="profile-list compact-list">
                    {profiles.map((profile) => {
                        const status = props.state.profileStatus?.profiles?.[profile.id];
                        return (
                            <div key={profile.id} className={`profile-row static-row ${props.state.activeProfile === profile.id ? 'selected' : ''}`}>
                                <div>
                                    <strong>{profile.name || profile.id}</strong>
                                    <code>{profile.id}</code>
                                </div>
                                <div className="profile-state">
                                    <span className={`profile-status ${statusClassName(status?.state, profile.enabled)}`}>{profileStatusText(status?.state, profile.enabled)}</span>
                                    <small>{profileStatusHint(status?.state, profile.enabled)}</small>
                                </div>
                            </div>
                        );
                    })}
                </div>
            </div>
            <div className="panel operations-compact-panel">
                <div className="log-head">
                    <div>
                        <p className="eyebrow">最近操作</p>
                        {latestLogs.length === 0 && <p className="muted">暂无命令输出。</p>}
                    </div>
                    <div className="actions compact">
                        <button className="ghost" onClick={props.onLogs}><TerminalSquare size={16}/>{props.logsFollowing ? '停止跟随' : '跟随日志'}</button>
                    </div>
                </div>
                {latestLogs.length > 0 && (
                    <div className="recent-logs">
                        {latestLogs.map((line, index) => <code key={`${index}-${line}`}>{line}</code>)}
                    </div>
                )}
                <details className="log-details">
                    <summary>展开完整日志</summary>
                    <div className="log-tools">
                        <button className="ghost icon-only" onClick={props.onCopyLogs} disabled={props.logs.length === 0} title="复制日志"><Clipboard size={16}/></button>
                        <button className="ghost icon-only" onClick={props.onClearLogs} disabled={props.logs.length === 0} title="清空日志"><Trash2 size={16}/></button>
                        <label className="mini-toggle"><input type="checkbox" checked={props.autoScrollLogs} onChange={(event) => props.setAutoScrollLogs(event.target.checked)}/>自动滚动</label>
                    </div>
                    <pre ref={props.logRef} className="logbox">{props.logs.length ? props.logs.join('\n') : '暂无命令输出。'}</pre>
                </details>
            </div>
        </section>
    );
}

function WebManagementCard(props: {
    status: WebStatus;
    busy: boolean;
    onSave: (settings: WebSettingsRequest) => Promise<boolean>;
    onChangePassword: (oldPassword: string, newPassword: string) => Promise<boolean>;
    onResetPassword: () => Promise<boolean>;
}) {
    const [enabled, setEnabled] = useState(props.status.enabled);
    const [scope, setScope] = useState(props.status.host === '127.0.0.1' ? 'local' : 'lan');
    const [port, setPort] = useState(props.status.port || '9876');
    const [oldPassword, setOldPassword] = useState('');
    const [newPassword, setNewPassword] = useState('');
    const [copied, setCopied] = useState('');
    const host = scope === 'local' ? '127.0.0.1' : '0.0.0.0';
    const primaryURL = props.status.primaryUrl || props.status.localUrl;

    useEffect(() => {
        setEnabled(props.status.enabled);
        setScope(props.status.host === '127.0.0.1' ? 'local' : 'lan');
        setPort(props.status.port || '9876');
    }, [props.status.enabled, props.status.host, props.status.port]);

    async function copy(value: string) {
        await navigator.clipboard.writeText(value);
        setCopied(value);
        window.setTimeout(() => setCopied(''), 1200);
    }

    async function saveSettings() {
        await props.onSave({enabled, host, port});
    }

    async function changePassword() {
        const ok = await props.onChangePassword(oldPassword, newPassword);
        if (ok) {
            setOldPassword('');
            setNewPassword('');
        }
    }

    return (
        <div className="panel web-management-card">
            <div className="web-management-header">
                <div>
                    <p className="eyebrow">Web 管理</p>
                    <h3>{props.status.running ? '局域网管理入口运行中' : 'Web 管理未运行'}</h3>
                    <p>局域网设备可通过下方地址访问 Web 管理。默认访问密码是 123456，建议首次登录后修改。</p>
                </div>
                <div className={`status-pill ${props.status.running ? 'ok' : 'warn'}`}>{props.status.running ? '运行中' : '未运行'}</div>
            </div>
            {props.status.usingDefaultPassword && <div className="web-password-warning">当前仍在使用默认访问密码，建议修改。</div>}
            {props.status.error && <div className="operation-error">Web 管理启动失败：{props.status.error}</div>}
            <div className="web-management-grid">
                <div className="web-addresses">
                    <AddressRow label="本机地址" value={props.status.localUrl} copied={copied} onCopy={copy}/>
                    {(props.status.lanUrls || []).map((url) => <AddressRow key={url} label="局域网地址" value={url} copied={copied} onCopy={copy}/>)}
                </div>
                {primaryURL && (
                    <div className="web-qr">
                        <QRCodeSVG value={primaryURL} size={128}/>
                        <span>扫码打开 Web 管理</span>
                    </div>
                )}
            </div>
            <div className="web-settings-row">
                <label className="switch-line">
                    <input type="checkbox" checked={enabled} onChange={(event) => setEnabled(event.target.checked)}/>
                    <span>开启 Web 管理</span>
                </label>
                <label>
                    <span>访问范围</span>
                    <select value={scope} onChange={(event) => setScope(event.target.value)}>
                        <option value="lan">局域网</option>
                        <option value="local">仅本机</option>
                    </select>
                </label>
                <label>
                    <span>端口</span>
                    <input value={port} onChange={(event) => setPort(event.target.value)} inputMode="numeric"/>
                </label>
                <button className="ghost" onClick={saveSettings} disabled={props.busy}>保存 Web 设置</button>
            </div>
            <div className="web-password-row">
                <input type="password" placeholder="旧访问密码" value={oldPassword} onChange={(event) => setOldPassword(event.target.value)}/>
                <input type="password" placeholder="新访问密码" value={newPassword} onChange={(event) => setNewPassword(event.target.value)}/>
                <button className="ghost" onClick={changePassword} disabled={props.busy || !oldPassword || !newPassword}>修改密码</button>
                <button className="ghost danger-text" onClick={props.onResetPassword} disabled={props.busy}>重置为 123456</button>
            </div>
        </div>
    );
}

function AddressRow(props: { label: string; value: string; copied: string; onCopy: (value: string) => void }) {
    if (!props.value) return null;
    return (
        <div className="web-address-row">
            <span>{props.label}</span>
            <code>{props.value}</code>
            <button className="icon-button" onClick={() => props.onCopy(props.value)} title="复制地址">
                <Clipboard size={15}/>
            </button>
            {props.copied === props.value && <em>已复制</em>}
        </div>
    );
}

function operationSummary(containerStatus: string, needsRebuild: boolean, runningProfiles: number, startingProfiles: number, failedProfiles: number, notConfiguredProfiles: number) {
    if (needsRebuild) {
        return {title: '配置已保存，等待应用', detail: '点击“应用并重建”后，新配置才会进入运行态。'};
    }
    if (containerStatus !== 'running') {
        return {title: 'Hermes 容器未运行', detail: '启动容器后，已绑定平台的助手才会开始接收消息。'};
    }
    if (failedProfiles > 0) {
        return {title: '有助手启动失败', detail: '查看最近操作或展开完整日志，通常能看到失败原因。'};
    }
    if (startingProfiles > 0) {
        return {title: '助手正在启动', detail: '通常需要 5-15 秒，完成后会自动刷新为运行中。'};
    }
    if (runningProfiles > 0) {
        return {title: `${runningProfiles} 个助手运行中`, detail: notConfiguredProfiles > 0 ? '部分助手尚未绑定平台，不会参与运行。' : '容器和助手都已进入运行态。'};
    }
    return {title: '暂无可运行助手', detail: '请先在助手页绑定微信、企业微信或飞书。'};
}

function profileStatusHint(state?: string, enabled = true) {
    if (!enabled || state === 'disabled') return '不参与运行';
    switch (state) {
        case 'running':
            return '正在接收平台消息';
        case 'starting':
            return '等待 runner 上报';
        case 'not_configured':
            return '去助手页绑定平台';
        case 'failed':
            return '查看日志';
        case 'stopped':
        case 'exited':
            return '容器或进程未运行';
        default:
            return '等待状态同步';
    }
}
