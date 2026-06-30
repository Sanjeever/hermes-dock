import type {RefObject} from 'react';
import {Clipboard, ExternalLink, Loader2, Play, RefreshCcw, RotateCcw, Square, TerminalSquare, Trash2} from 'lucide-react';
import {Health, IconButton} from '../components/primitives';
import type {AppState, ComposeSettings} from '../types';
import {endpointURL, profileStatusText, statusClassName} from '../utils';

export function Dashboard(props: {
    state: AppState;
    compose: ComposeSettings;
    busy: string;
    logs: string[];
    weixinBound: boolean;
    wecomBound: boolean;
    feishuBound: boolean;
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
    onOpenProfiles: () => void;
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
                    <Health label="飞书 / Lark" ok={props.feishuBound} onClick={props.onOpenPlatforms}/>
                </div>
            </div>
            <div className="panel wide">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">Profiles</p>
                        <h2>运行概览</h2>
                    </div>
                    <button className="ghost" onClick={props.onOpenProfiles}>管理 Profiles</button>
                </div>
                <div className="profile-list compact-list">
                    {(props.state.profiles?.profiles || []).map((profile) => {
                        const status = props.state.profileStatus?.profiles?.[profile.id];
                        return (
                            <button key={profile.id} className={`profile-row ${props.state.activeProfile === profile.id ? 'selected' : ''}`} onClick={props.onOpenProfiles}>
                                <div>
                                    <strong>{profile.name || profile.id}</strong>
                                    <code>{profile.id}</code>
                                </div>
                                <span className={`profile-status ${statusClassName(status?.state, profile.enabled)}`}>{profileStatusText(status?.state, profile.enabled)}</span>
                            </button>
                        );
                    })}
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
