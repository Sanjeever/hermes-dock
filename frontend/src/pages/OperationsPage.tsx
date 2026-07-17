import type {RefObject} from 'react';
import {useEffect, useState} from 'react';
import {CheckCircle2, CircleAlert, Clipboard, Download, ExternalLink, Loader2, Play, RefreshCcw, RotateCcw, Square, TerminalSquare, Trash2} from 'lucide-react';
import {QRCodeSVG} from 'qrcode.react';
import {AdvancedPage} from './AdvancedPage';
import {ChannelsPage} from './ChannelsPage';
import {DeployPage} from './DeployPage';
import type {AppState, ComposeSettings, InstanceBackupManifest, OperationsTab, ProxySettings, UpdateInfo, UpdateStatus, WebSettingsRequest, WebStatus} from '../types';
import {containerStatusText, endpointURL, isPortValue, profileStatusText, statusClassName} from '../utils';

export function OperationsPage(props: {
    scope?: 'runtime' | 'settings';
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
    webRuntime: boolean;
    backupStatus: string;
    backupManifest: InstanceBackupManifest | null;
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
    onOpenEndpoint: (endpoint: 'dashboard' | 'gateway' | 'dufs') => void;
    onOpenAssistantPlatforms: () => void;
    onSaveDeploy: () => void;
    onDiscardDeploy: () => void;
    onRefreshChannels: () => void;
    channelActionStatus: Record<string, string>;
    onHomeChannel: (platform: string, id: string) => void;
    onTestChannel: (platform: string, id: string) => void;
    onSaveAdvanced: () => void;
    onExportBackup: (targetPath: string) => Promise<void>;
    onInspectBackup: (path: string) => Promise<void>;
    onImportBackup: (path: string, confirm: string) => Promise<void>;
    onFactoryReset: () => Promise<void>;
    resetConfirmPhrase: string;
    webStatus: WebStatus;
    onSaveWebSettings: (settings: WebSettingsRequest) => Promise<boolean>;
    onChangeWebPassword: (oldPassword: string, newPassword: string) => Promise<boolean>;
    onResetWebPassword: () => Promise<boolean>;
    updateInfo: UpdateInfo | null;
    updateStatus: UpdateStatus;
    updateBusy: boolean;
    updateProgress: string;
    onCheckUpdate: () => void;
    onInstallUpdate: () => void;
    onSetAutoUpdate: (enabled: boolean) => Promise<void>;
}) {
    const tabs: Array<{ id: OperationsTab; label: string }> = props.scope === 'settings' ? [
        {id: 'basic', label: '基础设置'},
        {id: 'network', label: '网络设置'},
        {id: 'update', label: '软件更新'},
        {id: 'advanced', label: '高级设置'},
        {id: 'remote', label: 'Web 管理'},
    ] : [
        {id: 'runtime', label: '服务状态'},
        {id: 'diagnostics', label: '日志诊断'},
    ];
    const activeTab = tabs.some((item) => item.id === props.tab) ? props.tab : tabs[0].id;
    return (
        <section className="operations-stack">
            <div className="ops-tabs">
                {tabs.map((item) => <button key={item.id} className={activeTab === item.id ? 'selected' : ''} onClick={() => props.setTab(item.id)}>{item.label}</button>)}
            </div>
            {activeTab === 'runtime' && (
                <RuntimePage
                    state={props.state}
                    compose={props.compose}
                    needsRebuild={props.needsRebuild}
                    busy={props.busy}
                    weixinBound={props.weixinBound}
                    wecomBound={props.wecomBound}
                    feishuBound={props.feishuBound}
                    lastOperationError={props.lastOperationError}
                    onStart={props.onStart}
                    onStop={props.onStop}
                    onRestart={props.onRestart}
                    onRebuild={props.onRebuild}
                    onOpenEndpoint={props.onOpenEndpoint}
                    onOpenDiagnostics={() => props.setTab('diagnostics')}
                />
            )}
            {activeTab === 'remote' && (
                <section className="operations-context">
                    <WebManagementCard
                        status={props.webStatus}
                        busy={!!props.busy}
                        onSave={props.onSaveWebSettings}
                        onChangePassword={props.onChangeWebPassword}
                        onResetPassword={props.onResetWebPassword}
                    />
                </section>
            )}
            {activeTab === 'diagnostics' && (
                <section className="diagnostics-stack">
                    <div className="panel ops-page-intro">
                        <div>
                            <p className="eyebrow">诊断</p>
                            <h2>通道和运行日志</h2>
                            <p>当前助手：{props.activeProfileName}。消息没有按预期到达时，先刷新通道并发送测试消息，再查看日志。</p>
                        </div>
                    </div>
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
                    <OperationLogPanel
                        logs={props.logs}
                        autoScrollLogs={props.autoScrollLogs}
                        setAutoScrollLogs={props.setAutoScrollLogs}
                        logRef={props.logRef}
                        logsFollowing={props.logsFollowing}
                        lastOperationError={props.lastOperationError}
                        onLogs={props.onLogs}
                        onClearLogs={props.onClearLogs}
                        onCopyLogs={props.onCopyLogs}
                    />
                </section>
            )}
            {activeTab === 'update' && (
                <section className="operations-context">
                    <UpdateSettingsCard
                        currentVersion={props.state.appVersion}
                        info={props.updateInfo}
                        status={props.updateStatus}
                        busy={props.updateBusy}
                        progress={props.updateProgress}
                        onCheck={props.onCheckUpdate}
                        onInstall={props.onInstallUpdate}
                        onSetAutoUpdate={props.onSetAutoUpdate}
                    />
                </section>
            )}
            {activeTab === 'basic' && (
                <section className="advanced-ops-stack">
                    <DeployPage section="basic" compose={props.compose} proxy={props.proxy} hostBridge={props.state.hostBridge} dufs={props.state.dufs} setCompose={props.setCompose} setProxy={props.setProxy} dirty={props.deployDirty} busy={!!props.busy} onOpenEndpoint={props.onOpenEndpoint} onSave={props.onSaveDeploy} onDiscard={props.onDiscardDeploy}/>
                </section>
            )}
            {activeTab === 'network' && (
                <section className="advanced-ops-stack">
                    <DeployPage section="network" compose={props.compose} proxy={props.proxy} hostBridge={props.state.hostBridge} dufs={props.state.dufs} setCompose={props.setCompose} setProxy={props.setProxy} dirty={props.deployDirty} busy={!!props.busy} onOpenEndpoint={props.onOpenEndpoint} onSave={props.onSaveDeploy} onDiscard={props.onDiscardDeploy}/>
                </section>
            )}
            {activeTab === 'advanced' && (
                <section className="advanced-ops-stack">
                    <div className="operations-context">
                        <div className="setting-note">仅在你知道要修改什么时使用。当前助手：{props.activeProfileName}</div>
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
                            webRuntime={props.webRuntime}
                            backupStatus={props.backupStatus}
                            backupManifest={props.backupManifest}
                            onExportBackup={props.onExportBackup}
                            onInspectBackup={props.onInspectBackup}
                            onImportBackup={props.onImportBackup}
                            onSave={props.onSaveAdvanced}
                            onFactoryReset={props.onFactoryReset}
                            resetConfirmPhrase={props.resetConfirmPhrase}
                        />
                    </div>
                </section>
            )}
        </section>
    );
}

function UpdateSettingsCard(props: {
    currentVersion: string;
    info: UpdateInfo | null;
    status: UpdateStatus;
    busy: boolean;
    progress: string;
    onCheck: () => void;
    onInstall: () => void;
    onSetAutoUpdate: (enabled: boolean) => Promise<void>;
}) {
    const available = !!props.info?.available;
    return (
        <div className="panel update-settings-card">
            <div className="web-management-header">
                <div>
                    <p className="eyebrow">软件更新</p>
                    <h3>{available ? `发现新版本 v${props.info?.latestVersion}` : '企智盒版本管理'}</h3>
                    <p>更新只会重启启动器，正在运行的 Hermes Agent Docker 容器不会停止。</p>
                </div>
                <div className={`status-pill ${available ? 'warn' : 'ok'}`}>{available ? '可更新' : `v${props.currentVersion}`}</div>
            </div>
            <div className="update-version-grid">
                <div><span>当前版本</span><strong>v{props.currentVersion}</strong></div>
                <div><span>最新版本</span><strong>{props.info?.latestVersion ? `v${props.info.latestVersion}` : '尚未检查'}</strong></div>
            </div>
            <div className="update-settings-actions">
                <button className="ghost" onClick={props.onCheck} disabled={props.busy}><RefreshCcw size={16} className={props.busy && !props.progress ? 'spin' : undefined}/>{props.busy && !props.progress ? '检查中' : '检查更新'}</button>
                {available && <button className="primary no-margin" onClick={props.onInstall} disabled={props.busy || !props.info?.assetUrl}><Download size={16}/>{props.busy ? (props.progress || '正在更新') : '立即更新'}</button>}
            </div>
            <label className="update-auto-row">
                <span>
                    <strong>自动下载安装更新</strong>
                    <small>开启后注册系统定时任务，每天凌晨 2 点检查并安装稳定版本。默认关闭。</small>
                </span>
                <input type="checkbox" checked={props.status.autoUpdateEnabled} onChange={(event) => props.onSetAutoUpdate(event.target.checked)} disabled={props.busy}/>
            </label>
            {props.status.autoUpdateEnabled && !props.status.taskRegistered && <div className="form-warning">自动更新已开启，但系统定时任务未注册。</div>}
            {props.status.lastError && <div className="operation-error">自动更新：{props.status.lastError}</div>}
        </div>
    );
}

function RuntimePage(props: {
    state: AppState;
    compose: ComposeSettings;
    needsRebuild: boolean;
    busy: string;
    weixinBound: boolean;
    wecomBound: boolean;
    feishuBound: boolean;
    lastOperationError: string;
    onStart: () => void;
    onStop: () => void;
    onRestart: () => void;
    onRebuild: () => void;
    onOpenEndpoint: (endpoint: 'dashboard' | 'gateway' | 'dufs') => void;
    onOpenDiagnostics: () => void;
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
    const sortedProfiles = [...profiles].sort((a, b) => profilePriority(profileStatuses[b.id]?.state) - profilePriority(profileStatuses[a.id]?.state));
    const summary = operationSummary(props.state.containerStatus, props.needsRebuild, runningProfiles, startingProfiles, failedProfiles, notConfiguredProfiles);
    return (
        <section className="runtime-stack">
            <div className="panel operations-hero">
                <div>
                    <p className="eyebrow">运行</p>
                    <h2>{summary.title}</h2>
                    <p>{summary.detail}</p>
                </div>
                <div className="operation-primary-actions">
                    {props.needsRebuild ? (
                        <button className="primary no-margin" onClick={props.onRebuild} disabled={actionBusy}><RotateCcw size={16}/>应用配置</button>
                    ) : props.state.containerStatus === 'running' ? (
                        <button className="primary no-margin" onClick={props.onOpenDiagnostics} disabled={actionBusy}><TerminalSquare size={16}/>查看日志</button>
                    ) : (
                        <button className="primary no-margin" onClick={props.onStart} disabled={actionBusy}><Play size={16}/>启动服务</button>
                    )}
                    {!props.needsRebuild && props.state.containerStatus !== 'running' && <button className="ghost" onClick={props.onOpenDiagnostics} disabled={actionBusy}><TerminalSquare size={16}/>查看日志</button>}
                    {!props.needsRebuild && props.state.containerStatus === 'running' && <button className="ghost" onClick={props.onRestart} disabled={actionBusy}><RefreshCcw size={16}/>重启服务</button>}
                    {props.needsRebuild && <button className="ghost" onClick={props.onOpenDiagnostics} disabled={actionBusy}><TerminalSquare size={16}/>查看日志</button>}
                    <button className="ghost danger-text" onClick={props.onStop} disabled={actionBusy || props.state.containerStatus !== 'running'}><Square size={16}/>停止服务</button>
                </div>
                {props.busy && <div className="busy"><Loader2 size={16} className="spin"/>{props.busy}</div>}
                {props.lastOperationError && (
                    <div className="operation-error operation-error-action">
                        <span>最近错误：{props.lastOperationError}<br/>下一步：查看日志诊断，定位失败原因。</span>
                        <button className="ghost" onClick={props.onOpenDiagnostics}>查看日志</button>
                    </div>
                )}
            </div>

            <div className="panel runtime-workbench">
                <div className="runtime-workbench-head">
                    <div>
                        <p className="eyebrow">助手运行状态</p>
                        <h2>运行概览</h2>
                    </div>
                    <div className="runtime-workbench-counts">
                        <RuntimeCounter label="容器" value={containerStatusText(props.state.containerStatus)}/>
                        <RuntimeCounter label="助手" value={`${runningProfiles}/${profiles.length} 运行中`}/>
                    </div>
                </div>

                <div className="runtime-table">
                    <div className="runtime-table-row runtime-table-head">
                        <span>助手</span>
                        <span>状态</span>
                        <span>说明</span>
                    </div>
                    {sortedProfiles.map((profile) => {
                        const status = props.state.profileStatus?.profiles?.[profile.id];
                        return (
                            <div key={profile.id} className={`runtime-table-row ${props.state.activeProfile === profile.id ? 'selected' : ''}`}>
                                <div className="runtime-profile-name">
                                    <strong>{profile.name || profile.id}</strong>
                                    <code>{profile.id}</code>
                                </div>
                                <span className={`profile-status ${statusClassName(status?.state, profile.enabled)}`}>{profileStatusText(status?.state, profile.enabled)}</span>
                                <span className="runtime-profile-note">{status?.message || profileStatusHint(status?.state, profile.enabled)}</span>
                            </div>
                        );
                    })}
                </div>

                <div className="runtime-workbench-footer">
                    <RuntimeEndpoints
                        dashboardURL={dashboardURL}
                        gatewayURL={gatewayURL}
                        dashboardReady={dashboardReady}
                        gatewayReady={gatewayReady}
                        onOpenEndpoint={props.onOpenEndpoint}
                    />
                    <div className="runtime-check-strip">
                        <RuntimeCheck label="Docker" ok={props.state.dockerAvailable}/>
                        <RuntimeCheck label="Compose" ok={props.state.composeAvailable}/>
                        <RuntimeCheck label="个人微信" ok={props.weixinBound}/>
                        <RuntimeCheck label="企业微信" ok={props.wecomBound}/>
                        <RuntimeCheck label="飞书 / Lark" ok={props.feishuBound}/>
                    </div>
                </div>
            </div>
        </section>
    );
}

function RuntimeCounter(props: { label: string; value: string }) {
    return (
        <div className="runtime-counter">
            <span>{props.label}</span>
            <strong>{props.value}</strong>
        </div>
    );
}

function RuntimeCheck(props: { label: string; ok: boolean }) {
    return (
        <div className={`runtime-check ${props.ok ? 'ok' : 'warn'}`}>
            {props.ok ? <CheckCircle2 size={15}/> : <CircleAlert size={15}/>}
            <span>{props.label}</span>
        </div>
    );
}

function RuntimeEndpoints(props: {
    dashboardURL: string;
    gatewayURL: string;
    dashboardReady: boolean;
    gatewayReady: boolean;
    onOpenEndpoint: (endpoint: 'dashboard' | 'gateway') => void;
}) {
    return (
        <div className="runtime-endpoints">
            <EndpointRow label="控制台" value={props.dashboardURL} ready={props.dashboardReady} disabledTitle="容器运行且端口有效后可打开" onOpen={() => props.onOpenEndpoint('dashboard')}/>
            <EndpointRow label="网关" value={props.gatewayURL} ready={props.gatewayReady} disabledTitle="容器运行且端口有效后可打开" onOpen={() => props.onOpenEndpoint('gateway')}/>
        </div>
    );
}

function EndpointRow(props: { label: string; value: string; ready: boolean; disabledTitle: string; onOpen: () => void }) {
    return (
        <button className="runtime-endpoint-row" onClick={props.onOpen} disabled={!props.ready} title={props.ready ? `打开${props.label}` : props.disabledTitle}>
            <span>{props.label}</span>
            <code>{props.value}</code>
            <ExternalLink size={16}/>
        </button>
    );
}

function OperationLogPanel(props: {
    logs: string[];
    autoScrollLogs: boolean;
    setAutoScrollLogs: (value: boolean) => void;
    logRef: RefObject<HTMLPreElement>;
    logsFollowing: boolean;
    lastOperationError: string;
    onLogs: () => void;
    onClearLogs: () => void;
    onCopyLogs: () => void;
}) {
    const latestLogs = props.logs.slice(-4);
    return (
        <div className="panel operations-compact-panel operation-log-panel">
            <div className="log-head">
                <div>
                    <p className="eyebrow">运行日志</p>
                    {latestLogs.length === 0 && <p className="muted">暂无命令输出。</p>}
                </div>
                <div className="actions compact">
                    <button className="ghost" onClick={props.onLogs}><TerminalSquare size={16}/>{props.logsFollowing ? '停止跟随' : '实时日志'}</button>
                </div>
            </div>
            {props.lastOperationError && <div className="operation-error">最近错误：{props.lastOperationError}</div>}
            {latestLogs.length > 0 && (
                <div className="recent-logs">
                    {latestLogs.map((line, index) => <code key={`${index}-${line}`}>{line}</code>)}
                </div>
            )}
            <details className="log-details" open={!!props.lastOperationError}>
                <summary>展开完整日志</summary>
                <div className="log-tools">
                    <button className="ghost icon-only" onClick={props.onCopyLogs} disabled={props.logs.length === 0} title="复制日志" aria-label="复制日志"><Clipboard size={16}/></button>
                    <button className="ghost icon-only" onClick={props.onClearLogs} disabled={props.logs.length === 0} title="清空日志" aria-label="清空日志"><Trash2 size={16}/></button>
                    <label className="mini-toggle"><input type="checkbox" checked={props.autoScrollLogs} onChange={(event) => props.setAutoScrollLogs(event.target.checked)}/>自动滚动</label>
                </div>
                <pre ref={props.logRef} className="logbox">{props.logs.length ? props.logs.join('\n') : '暂无命令输出。'}</pre>
            </details>
        </div>
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
    const portValid = isPortValue(port);

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
                <button className="ghost" onClick={saveSettings} disabled={props.busy || !portValid}>保存 Web 设置</button>
            </div>
            {!portValid && <div className="form-warning">端口必须是 1 到 65535 之间的数字。</div>}
            <div className="web-password-row">
                <input type="password" placeholder="旧访问密码" value={oldPassword} onChange={(event) => setOldPassword(event.target.value)} autoComplete="current-password"/>
                <input type="password" placeholder="新访问密码" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} autoComplete="new-password"/>
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
            <button className="icon-button" onClick={() => props.onCopy(props.value)} title="复制地址" aria-label="复制地址">
                <Clipboard size={15}/>
            </button>
            {props.copied === props.value && <em>已复制</em>}
        </div>
    );
}

function operationSummary(containerStatus: string, needsRebuild: boolean, runningProfiles: number, startingProfiles: number, failedProfiles: number, notConfiguredProfiles: number) {
    if (needsRebuild) {
        return {title: '配置待应用', detail: '点击“应用配置”后，新设置才会生效。'};
    }
    if (containerStatus !== 'running') {
        return {title: '服务未运行', detail: '启动后，已绑定平台的助手才会接收消息。'};
    }
    if (failedProfiles > 0) {
        return {title: '有助手启动失败', detail: '打开日志诊断，通常能看到失败原因。'};
    }
    if (startingProfiles > 0) {
        return {title: '助手正在启动', detail: '通常需要 10-60 秒，完成后会自动刷新为运行中。'};
    }
    if (runningProfiles > 0) {
        return {title: `${runningProfiles} 个助手运行中`, detail: notConfiguredProfiles > 0 ? '部分助手尚未绑定平台，不会参与运行。' : '容器和助手都已进入运行态。'};
    }
    return {title: '暂无可运行助手', detail: '请先在助手页绑定微信、企业微信或飞书。'};
}

function profilePriority(state?: string) {
    switch (state) {
        case 'failed':
            return 5;
        case 'starting':
            return 4;
        case 'running':
            return 3;
        case 'not_configured':
            return 2;
        case 'stopped':
        case 'exited':
            return 1;
        default:
            return 0;
    }
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
