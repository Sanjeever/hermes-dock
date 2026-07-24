import {useEffect, useRef, useState} from 'react';
import {AlertTriangle, CheckCircle2, Clipboard, FileText, FolderOpen, Globe2, Play, Plus, RotateCcw, Settings, Square} from 'lucide-react';
import type {AppState} from '../types';
import {containerStatusText, profileStatusText, statusClassName} from '../utils';

export function OverviewPage(props: {
    state: AppState;
    needsRebuild: boolean;
    busy: string;
    lastOperationError: string;
    logs: string[];
    onStart: () => void;
    onStop: () => void;
    onRebuild: () => void;
    onOpenAssistants: () => void;
    onOpenLogs: () => void;
    onOpenSettings: () => void;
    onOpenAccessSettings: () => void;
    onOpenWebManagement: () => void;
    onOpenFileManagement: () => void;
}) {
    const profiles = props.state.profiles?.profiles || [];
    const profileStatuses = props.state.profileStatus?.profiles || {};
    const activeProfile = profiles.find((profile) => profile.id === props.state.activeProfile);
    const setupIncomplete = !!activeProfile && !activeProfile.setupCompletedAt;
    const runningProfiles = profiles.filter((profile) => profileStatuses[profile.id]?.state === 'running').length;
    const failedProfiles = profiles.filter((profile) => profileStatuses[profile.id]?.state === 'failed').length;
    const notConfiguredProfiles = profiles.filter((profile) => profileStatuses[profile.id]?.state === 'not_configured').length;
    const latestLogs = props.logs.slice(-3);
    const summary = serviceSummary(props.state.containerStatus, props.needsRebuild, setupIncomplete, runningProfiles, failedProfiles, notConfiguredProfiles);
    const actionBusy = props.busy !== '';

    return (
        <section className="overview-stack">
            <div className={`service-hero ${summary.tone}`}>
                <div className="service-hero-main">
                    <span className="service-dot" aria-hidden="true"/>
                    <div>
                        <p className="eyebrow">当前状态</p>
                        <div className="service-title-row">
                            <h2>{summary.title}</h2>
                            <span className={`service-status-badge ${summary.tone}`}>{summary.badge}</span>
                        </div>
                        <p>{summary.detail}</p>
                    </div>
                </div>
                <div className="service-actions">
                    {setupIncomplete ? (
                        <button className="primary no-margin" onClick={props.onOpenAssistants} disabled={actionBusy}>继续配置</button>
                    ) : props.needsRebuild ? (
                        <button className="primary no-margin" onClick={props.onRebuild} disabled={actionBusy}><RotateCcw size={16}/>应用配置</button>
                    ) : props.state.containerStatus === 'running' ? (
                        <button className="ghost" onClick={props.onStop} disabled={actionBusy}><Square size={16}/>停止服务</button>
                    ) : (
                        <button className="primary no-margin" onClick={props.onStart} disabled={actionBusy}><Play size={16}/>启动服务</button>
                    )}
                    <button className="ghost" onClick={props.onOpenLogs} disabled={actionBusy}><FileText size={16}/>查看日志</button>
                </div>
            </div>

            {props.lastOperationError && (
                <div className="next-error">
                    <AlertTriangle size={18}/>
                    <div>
                        <strong>最近操作失败</strong>
                        <p>{props.lastOperationError}</p>
                        <span>下一步：打开日志诊断，查看失败原因。</span>
                    </div>
                    <button className="ghost" onClick={props.onOpenLogs}>查看日志</button>
                </div>
            )}

            <div className="overview-grid">
                <section className="panel overview-section">
                    <div className="section-head">
                        <div>
                            <p className="eyebrow">服务</p>
                            <h2>运行检查</h2>
                        </div>
                    </div>
                    <div className="status-list">
                        <StatusLine label="Docker" ok={props.state.dockerAvailable} detail={props.state.dockerAvailable ? '已可用' : '请先启动 Docker Desktop'}/>
                        <StatusLine label="运行组件" ok={props.state.composeAvailable} detail={props.state.composeAvailable ? '已可用' : '请检查 Docker Compose'}/>
                        <StatusLine label="助手配置" ok={!setupIncomplete} detail={setupIncomplete ? '请完成四步配置' : '已完成'}/>
                        <StatusLine label="服务" ok={props.state.containerStatus === 'running'} detail={containerStatusText(props.state.containerStatus)}/>
                        <StatusLine label="配置" ok={!props.needsRebuild} detail={props.needsRebuild ? '需要应用配置' : '已应用'}/>
                    </div>
                </section>

                <section className="panel overview-section">
                    <div className="section-head">
                        <div>
                            <p className="eyebrow">助手</p>
                            <h2>{runningProfiles}/{profiles.length} 运行中</h2>
                        </div>
                        <button className="ghost" onClick={props.onOpenAssistants}>管理助手</button>
                    </div>
                    {profiles.length === 0 ? (
                        <div className="overview-empty">
                            <strong>还没有助手</strong>
                            <span>下一步：创建助手并选择模型服务。</span>
                            <button className="primary no-margin" onClick={props.onOpenAssistants}><Plus size={16}/>创建助手</button>
                        </div>
                    ) : (
                        <div className="overview-profile-list">
                            {profiles.slice(0, 5).map((profile) => {
                            const status = profileStatuses[profile.id];
                            return (
                                <div key={profile.id} className="overview-profile-row">
                                    <div>
                                        <strong>{profile.name || profile.id}</strong>
                                        <code>{profile.id}</code>
                                    </div>
                                    <span className={`profile-status ${statusClassName(status?.state, profile.enabled)}`}>{profileStatusText(status?.state, profile.enabled)}</span>
                                </div>
                            );
                            })}
                        </div>
                    )}
                </section>
            </div>

            <AccessEntrances
                state={props.state}
                busy={actionBusy}
                onOpenSettings={props.onOpenAccessSettings}
                onOpenWebManagement={props.onOpenWebManagement}
                onOpenFiles={props.onOpenFileManagement}
            />

            <section className="panel overview-section">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">下一步</p>
                        <h2>{summary.nextTitle}</h2>
                    </div>
                </div>
                <div className="next-actions">
                    <button className={setupIncomplete ? 'primary no-margin' : 'ghost'} onClick={props.onOpenAssistants}>{setupIncomplete ? '继续配置' : '配置助手'}</button>
                    <button className="ghost" onClick={props.onOpenSettings}><Settings size={16}/>部署设置</button>
                    <button className="ghost" onClick={props.onOpenLogs}><FileText size={16}/>日志诊断</button>
                </div>
                {latestLogs.length > 0 && (
                    <div className="overview-log-tail">
                        {latestLogs.map((line, index) => <code key={`${index}-${line}`}>{line}</code>)}
                    </div>
                )}
            </section>
        </section>
    );
}

function AccessEntrances(props: {
    state: AppState;
    busy: boolean;
    onOpenSettings: () => void;
    onOpenWebManagement: () => void;
    onOpenFiles: () => void;
}) {
    const [copied, setCopied] = useState('');
    const copiedTimer = useRef(0);
    const webReady = props.state.web.enabled && props.state.web.running;
    const filesReady = props.state.dufs.enabled && props.state.containerStatus === 'running';
    const webLANURL = webReady && props.state.web.host === '0.0.0.0' ? (props.state.web.lanUrls?.[0] || '') : '';
    const dufsLANURL = filesReady ? (props.state.dufs.lanUrls?.[0] || '') : '';
    const webAccessNote = !props.state.web.enabled ? '未开启' : !props.state.web.running ? '未运行' : '仅本机可访问';
    const filesAccessNote = !props.state.dufs.enabled ? '未开启' : props.state.containerStatus !== 'running' ? '服务未运行' : '仅本机可访问';

    useEffect(() => () => window.clearTimeout(copiedTimer.current), []);

    async function copyAddress(id: string, value: string) {
        window.clearTimeout(copiedTimer.current);
        try {
            await navigator.clipboard.writeText(value);
            setCopied(id);
        } catch {
            setCopied(`${id}-failed`);
        }
        copiedTimer.current = window.setTimeout(() => setCopied(''), 1200);
    }

    return (
        <section className="panel access-entrances">
            <div className="section-head">
                <div>
                    <p className="eyebrow">访问入口</p>
                    <h2>管理与共享</h2>
                </div>
                <button className="ghost" onClick={props.onOpenSettings}><Settings size={16}/>访问与网络</button>
            </div>
            <div className="access-entrance-grid">
                <div className="access-entrance-card">
                    <div className="access-entrance-icon"><Globe2 size={20}/></div>
                    <div className="access-entrance-copy">
                        <strong>Web 管理</strong>
                        <span>在其他设备管理企智盒和所有助手。</span>
                    </div>
                    <div className="access-entrance-actions">
                        <button className="primary no-margin" onClick={props.onOpenWebManagement} disabled={props.busy || !webReady}>打开管理</button>
                        {webLANURL ? <button className="ghost" onClick={() => copyAddress('web', webLANURL)} disabled={props.busy}><Clipboard size={16}/>{copied === 'web' ? '已复制' : copied === 'web-failed' ? '复制失败' : '复制局域网地址'}</button> : <span className="access-entrance-note">{webAccessNote}</span>}
                    </div>
                </div>
                <div className="access-entrance-card">
                    <div className="access-entrance-icon files"><FolderOpen size={20}/></div>
                    <div className="access-entrance-copy">
                        <strong>共享文件</strong>
                        <span>在局域网设备上传、下载和整理文件。</span>
                    </div>
                    <div className="access-entrance-actions">
                        <button className="primary no-margin" onClick={props.onOpenFiles} disabled={props.busy || !filesReady}>打开文件管理</button>
                        {dufsLANURL ? <button className="ghost" onClick={() => copyAddress('dufs', dufsLANURL)} disabled={props.busy}><Clipboard size={16}/>{copied === 'dufs' ? '已复制' : copied === 'dufs-failed' ? '复制失败' : '复制局域网地址'}</button> : <span className="access-entrance-note">{filesAccessNote}</span>}
                    </div>
                </div>
            </div>
        </section>
    );
}

function StatusLine(props: { label: string; ok: boolean; detail: string }) {
    return (
        <div className={`status-line ${props.ok ? 'ok' : 'warn'}`}>
            {props.ok ? <CheckCircle2 size={16}/> : <AlertTriangle size={16}/>}
            <strong>{props.label}</strong>
            <span>{props.detail}</span>
        </div>
    );
}

function serviceSummary(containerStatus: string, needsRebuild: boolean, setupIncomplete: boolean, runningProfiles: number, failedProfiles: number, notConfiguredProfiles: number) {
    if (setupIncomplete) {
        return {
            tone: 'warn',
            badge: '待配置',
            title: '助手未配置',
            detail: '先完成四步配置，助手才能接收消息。',
            nextTitle: '继续配置助手',
        };
    }
    if (needsRebuild) {
        return {
            tone: 'warn',
            badge: '待应用',
            title: '配置待应用',
            detail: '已保存的修改还没有生效。',
            nextTitle: '点击“应用配置”让修改生效',
        };
    }
    if (containerStatus !== 'running') {
        return {
            tone: 'idle',
            badge: '未运行',
            title: '服务未运行',
            detail: '启动后，已绑定平台的助手才会接收消息。',
            nextTitle: '先启动服务',
        };
    }
    if (failedProfiles > 0) {
        return {
            tone: 'bad',
            badge: '需处理',
            title: '有助手启动失败',
            detail: '日志里通常会显示失败原因。',
            nextTitle: '打开日志诊断',
        };
    }
    if (runningProfiles > 0) {
        return {
            tone: 'ok',
            badge: '正常',
            title: `${runningProfiles} 个助手运行中`,
            detail: notConfiguredProfiles > 0 ? '部分助手未绑定平台，不会接收消息。' : '服务和助手都已就绪。',
            nextTitle: '可以查看日志或继续配置助手',
        };
    }
    return {
        tone: 'warn',
        badge: '待配置',
        title: '暂无可运行助手',
        detail: '请先给助手绑定微信、企业微信或飞书。',
        nextTitle: '先配置助手平台',
    };
}
