import {useEffect, useRef, useState, type ReactNode} from 'react';
import {Clipboard, Cpu, ExternalLink, FolderOpen, Network, RotateCcw, Save} from 'lucide-react';
import {Field, SecretField} from '../components/fields';
import {ChooseSharedDirectory, GetRecommendedResourceLimits, isWebRuntime} from '../services/api';
import type {ComposeSettings, DufsStatus, HostBridgeStatus, ProxySettings} from '../types';
import {isPortValue} from '../utils';

export function DeployPage({section = 'basic', compose, proxy, hostBridge, dufs, setCompose, setProxy, dirty, busy, onOpenEndpoint, onSave, onDiscard}: {
    section?: 'basic' | 'access' | 'advanced';
    compose: ComposeSettings;
    proxy: ProxySettings;
    hostBridge: HostBridgeStatus;
    dufs: DufsStatus;
    setCompose: (value: ComposeSettings) => void;
    setProxy: (value: ProxySettings) => void;
    dirty: boolean;
    busy: boolean;
    onOpenEndpoint: (endpoint: 'dashboard' | 'gateway' | 'dufs') => void;
    onSave: () => void;
    onDiscard: () => void;
}) {
    const [passwordVisible, setPasswordVisible] = useState(false);
    const [dufsPasswordVisible, setDufsPasswordVisible] = useState(false);
    const [copiedDufsURL, setCopiedDufsURL] = useState(false);
    const [resourceRecommendation, setResourceRecommendation] = useState<{ dockerMemoryGB: number; dockerCPU: number } | null>(null);
    const [resourceRecommendationBusy, setResourceRecommendationBusy] = useState(false);
    const [resourceRecommendationError, setResourceRecommendationError] = useState('');
    const [directoryPickerError, setDirectoryPickerError] = useState('');
    const composeRef = useRef(compose);
    const update = (key: Exclude<keyof ComposeSettings, 'dashboardEnabled' | 'dufsEnabled' | 'dufsUsingDefaultPassword'>, value: string) => setCompose({...compose, dashboardEnabled: true, [key]: value});
    const updateProxyText = (key: keyof Omit<ProxySettings, 'enabled'>, value: string) => setProxy({...proxy, [key]: value});
    const dufsAccountValid = /^[A-Za-z0-9._-]+$/.test(compose.dufsUsername);
    const dufsPortValid = !compose.dufsEnabled || isPortValue(compose.dufsPort);
    const servicePortsValid = isPortValue(compose.gatewayPort) && isPortValue(compose.dashboardPort);
    const proxyReady = !proxy.enabled || !!(proxy.httpProxy.trim() || proxy.httpsProxy.trim() || proxy.allProxy.trim());
    const dockerSettingsHint = !dufsAccountValid
        ? '文件管理用户名格式无效，请先修正。'
        : !dufsPortValid
            ? '文件管理端口无效，请先修正。'
            : !servicePortsValid
                ? '服务端口无效，请先修正。'
                : !proxyReady
                    ? '启用代理时，请至少填写一个代理地址。'
                    : dirty ? 'Docker 设置有未保存修改。' : '没有未保存修改。';
    const dockerSettingsValid = dufsAccountValid && dufsPortValid && servicePortsValid && proxyReady;
    const isBasic = section === 'basic';
    const isAdvanced = section === 'advanced';
    const resourceStatus = resourceRecommendation
        ? `Docker 可用资源：${resourceRecommendation.dockerMemoryGB}G 内存 / ${resourceRecommendation.dockerCPU} CPU`
        : resourceRecommendationError || (resourceRecommendationBusy ? '正在读取 Docker 可用资源...' : '推荐值按 Docker 可用资源计算。');

    useEffect(() => {
        composeRef.current = compose;
    }, [compose]);

    useEffect(() => {
        if (!isAdvanced) return;
        let cancelled = false;
        setResourceRecommendationBusy(true);
        setResourceRecommendationError('');
        GetRecommendedResourceLimits()
            .then((recommendation) => {
                if (cancelled) return;
                setResourceRecommendation({
                    dockerMemoryGB: recommendation.dockerMemoryGB,
                    dockerCPU: recommendation.dockerCPU,
                });
            })
            .catch((error) => {
                if (cancelled) return;
                setResourceRecommendation(null);
                setResourceRecommendationError(errorMessage(error) || 'Docker 未运行，无法计算推荐值');
            })
            .finally(() => {
                if (!cancelled) setResourceRecommendationBusy(false);
            });
        return () => {
            cancelled = true;
        };
    }, [isAdvanced]);

    async function applyRecommendedResourceLimits() {
        setResourceRecommendationBusy(true);
        setResourceRecommendationError('');
        try {
            const recommendation = await GetRecommendedResourceLimits();
            setResourceRecommendation({
                dockerMemoryGB: recommendation.dockerMemoryGB,
                dockerCPU: recommendation.dockerCPU,
            });
            const current = composeRef.current;
            setCompose({
                ...current,
                dashboardEnabled: true,
                memoryLimit: recommendation.memoryLimit,
                cpuLimit: recommendation.cpuLimit,
            });
        } catch (error) {
            setResourceRecommendation(null);
            setResourceRecommendationError(errorMessage(error) || 'Docker 未运行，无法计算推荐值');
        } finally {
            setResourceRecommendationBusy(false);
        }
    }

    async function copyDufsURL() {
        if (!dufs.primaryUrl) return;
        await navigator.clipboard.writeText(dufs.primaryUrl);
        setCopiedDufsURL(true);
        window.setTimeout(() => setCopiedDufsURL(false), 1200);
    }

    async function chooseSharedDirectory() {
        setDirectoryPickerError('');
        try {
            const selected = await ChooseSharedDirectory(compose.sharedDirectory);
            if (selected) update('sharedDirectory', selected);
        } catch (error) {
            setDirectoryPickerError(errorMessage(error) || '无法打开目录选择器');
        }
    }

    return (
        <section className={`deploy-stack settings-stack ${isBasic ? '' : 'access-settings-stack'}`}>
            {isBasic && <>
                <div className="panel settings-list">
                    <SettingRow title="宿主机控制" description="允许助手操作当前电脑。">
                        <div className="setting-control-stack">
                            <label className="toggle">
                                <input
                                    type="checkbox"
                                    checked={compose.hostControlEnabled !== 'false'}
                                    onChange={(event) => setCompose({...compose, dashboardEnabled: true, hostControlEnabled: event.target.checked ? 'true' : 'false'})}
                                />
                                允许宿主机操作
                            </label>
                            {compose.hostControlEnabled !== 'false' && <div className="form-warning">可操作文件、剪贴板、屏幕和应用，不逐次确认。</div>}
                            {hostBridge.error && <div className="form-warning">启动失败：{hostBridge.error}</div>}
                            {!hostBridge.error && hostBridge.enabled && <div className="setting-note">{hostBridge.running ? '服务运行中。' : '保存后启动服务。'}</div>}
                        </div>
                    </SettingRow>
                </div>
                <div className="panel settings-list">
                    <SettingRow title="消息处理" description="助手忙碌时的处理方式。">
                        <div className="setting-control-grid three">
                            <GatewaySelect label="忙碌时" value={compose.gatewayBusyInputMode || 'steer'} onChange={(value) => update('gatewayBusyInputMode', value)} options={[
                                {value: 'queue', label: '排队处理'},
                                {value: 'steer', label: '引导当前任务'},
                                {value: 'interrupt', label: '中断当前任务'},
                            ]}/>
                            <GatewaySelect label="自动回复" value={compose.gatewayBusyAckEnabled || 'false'} onChange={(value) => update('gatewayBusyAckEnabled', value)} options={[
                                {value: 'true', label: '启用'},
                                {value: 'false', label: '关闭'},
                            ]}/>
                            <GatewaySelect label="后台通知" value={compose.backgroundNotifications || 'result'} onChange={(value) => update('backgroundNotifications', value)} options={[
                                {value: 'all', label: '全部'},
                                {value: 'result', label: '仅结果'},
                                {value: 'error', label: '仅失败'},
                                {value: 'off', label: '关闭'},
                            ]}/>
                        </div>
                    </SettingRow>
                </div>
                <DeploySaveActions hint={dockerSettingsHint} busy={busy} dirty={dirty} valid={dockerSettingsValid} onSave={onSave} onDiscard={onDiscard}/>
            </>}

            {section === 'access' && <>
                <div className="panel settings-list access-files-card">
                        <SettingRow title="共享文件" description="所有助手共同读写，用于批量输入和交付文件。">
                            <div className="setting-control-stack">
                                <label className="field directory-field">
                                    <span>宿主机目录</span>
                                    <div className="directory-field-control">
                                        <input value={compose.sharedDirectory} onChange={(event) => update('sharedDirectory', event.target.value)}/>
                                        {!isWebRuntime() && <button className="ghost" type="button" onClick={chooseSharedDirectory} disabled={busy}><FolderOpen size={16}/>选择目录</button>}
                                    </div>
                                    <div className="field-hint">容器内固定为 /opt/data/.dock/shared，不包含在实例备份中。</div>
                                </label>
                                {directoryPickerError && <div className="operation-error">{directoryPickerError}</div>}
                                <label className="toggle">
                                    <input
                                        type="checkbox"
                                        checked={compose.dufsEnabled}
                                        onChange={(event) => setCompose({...compose, dashboardEnabled: true, dufsEnabled: event.target.checked})}
                                    />
                                    开启局域网文件管理
                                </label>
                                {compose.dufsEnabled && (
                                    <>
                                        <div className="setting-control-grid">
                                            <Field label="用户名" value={compose.dufsUsername} onChange={(value) => update('dufsUsername', value)}/>
                                            <Field label="端口" value={compose.dufsPort} onChange={(value) => update('dufsPort', value)}/>
                                        </div>
                                        <SecretField
                                            label="新密码"
                                            value={compose.dufsPassword || ''}
                                            visible={dufsPasswordVisible}
                                            setVisible={setDufsPasswordVisible}
                                            onChange={(value) => update('dufsPassword', value)}
                                            hint="留空表示不修改当前密码。"
                                        />
                                        {!dufsAccountValid && <div className="form-warning">用户名只能包含字母、数字、点、下划线和连字符。</div>}
                                        {compose.dufsUsingDefaultPassword && !compose.dufsPassword && <div className="form-warning">文件管理仍使用默认密码 123456，建议修改后再供局域网使用。</div>}
                                        <div className="form-warning">仅限可信局域网使用。</div>
                                        {dufs.enabled && dufs.primaryUrl && (
                                            <div className="actions compact">
                                                <button className="ghost no-margin" type="button" onClick={() => onOpenEndpoint('dufs')} disabled={busy}><ExternalLink size={16}/>打开文件管理</button>
                                                <button className="ghost no-margin" type="button" onClick={copyDufsURL} disabled={busy}><Clipboard size={16}/>{copiedDufsURL ? '已复制' : '复制访问地址'}</button>
                                            </div>
                                        )}
                                    </>
                                )}
                            </div>
                        </SettingRow>
                        {!dufsPortValid && <div className="form-warning settings-warning">文件管理端口必须是 1-65535 的数字。</div>}
                </div>
                <DeploySaveActions hint={dockerSettingsHint} busy={busy} dirty={dirty} valid={dockerSettingsValid} onSave={onSave} onDiscard={onDiscard}/>
            </>}

            {isAdvanced && <>
                <div className="panel settings-list">
                    <SettingRow title="运行参数" description="仅在排障或端口冲突时修改。">
                        <div className="setting-control-stack">
                            <div className="setting-control-grid">
                                <Field label="控制台用户名" value={compose.dashboardUsername} onChange={(value) => update('dashboardUsername', value)}/>
                                <SecretField label="控制台密码" value={compose.dashboardPassword} visible={passwordVisible} setVisible={setPasswordVisible} onChange={(value) => update('dashboardPassword', value)}/>
                            </div>
                            {compose.dashboardPassword === '123456' && <div className="form-warning">控制台仍在使用默认密码。</div>}
                            <div className="setting-control-grid">
                                <Field label="控制台端口" value={compose.dashboardPort} onChange={(value) => update('dashboardPort', value)}/>
                                <Field label="消息服务端口" value={compose.gatewayPort} onChange={(value) => update('gatewayPort', value)}/>
                            </div>
                            {!servicePortsValid && <div className="form-warning">端口必须是 1-65535 的数字。</div>}
                        </div>
                    </SettingRow>
                </div>
                <div className="panel settings-list">
                    <SettingRow title="资源配额" description="默认按 Docker 可用资源计算。">
                        <div className="setting-control-stack">
                            <div className="resource-recommendation"><span>{resourceStatus}</span></div>
                            <button className="ghost no-margin" type="button" onClick={applyRecommendedResourceLimits} disabled={busy || resourceRecommendationBusy}><Cpu size={16}/>使用推荐值</button>
                            <div className="setting-control-grid three">
                                <Field label="内存限制" value={compose.memoryLimit} onChange={(value) => update('memoryLimit', value)}/>
                                <Field label="CPU 限制" value={compose.cpuLimit} onChange={(value) => update('cpuLimit', value)}/>
                                <Field label="共享内存" value={compose.shmSize} onChange={(value) => update('shmSize', value)}/>
                            </div>
                        </div>
                    </SettingRow>
                </div>
                <div className="panel settings-list">
                    <SettingRow title="宿主机代理" description="网络访问不通时再开启。">
                        <div className="setting-control-stack">
                            <label className="toggle"><input type="checkbox" checked={proxy.enabled} onChange={(event) => setProxy({...proxy, enabled: event.target.checked})}/>启用代理</label>
                            <button className="ghost no-margin" type="button" onClick={() => setProxy({...proxy, enabled: true, httpProxy: 'http://host.docker.internal:7890', httpsProxy: 'http://host.docker.internal:7890', noProxy: proxy.noProxy || 'localhost,127.0.0.1,::1,host.docker.internal'})}><Network size={16}/>使用常见本机代理</button>
                            <div className="setting-control-grid">
                                <Field label="HTTP_PROXY" value={proxy.httpProxy} onChange={(value) => updateProxyText('httpProxy', value)}/>
                                <Field label="HTTPS_PROXY" value={proxy.httpsProxy} onChange={(value) => updateProxyText('httpsProxy', value)}/>
                                <Field label="ALL_PROXY" value={proxy.allProxy} onChange={(value) => updateProxyText('allProxy', value)}/>
                                <Field label="NO_PROXY" value={proxy.noProxy} onChange={(value) => updateProxyText('noProxy', value)}/>
                            </div>
                            {!proxyReady && <div className="form-warning">启用代理时，请至少填写一个代理地址。</div>}
                        </div>
                    </SettingRow>
                </div>
                <DeploySaveActions hint={dockerSettingsHint} busy={busy} dirty={dirty} valid={dockerSettingsValid} onSave={onSave} onDiscard={onDiscard}/>
            </>}
        </section>
    );
}

function DeploySaveActions(props: { hint: string; busy: boolean; dirty: boolean; valid: boolean; onSave: () => void; onDiscard: () => void }) {
    return (
        <div className="settings-save-actions">
            <span>{props.hint}</span>
            <div className="actions compact">
                <button className="ghost" onClick={props.onDiscard} disabled={props.busy || !props.dirty}><RotateCcw size={16}/>放弃 Docker 设置</button>
                <button className="primary no-margin" onClick={props.onSave} disabled={props.busy || !props.dirty || !props.valid}><Save size={16}/>保存 Docker 设置</button>
            </div>
        </div>
    );
}

function errorMessage(error: unknown) {
    return error instanceof Error ? error.message : '';
}

function SettingRow(props: { title: string; description: string; children: ReactNode }) {
    return (
        <div className="setting-row">
            <div className="setting-row-copy">
                <strong>{props.title}</strong>
                <span>{props.description}</span>
            </div>
            <div className="setting-row-control">
                {props.children}
            </div>
        </div>
    );
}

function GatewaySelect(props: { label: string; value: string; options: Array<{ value: string; label: string }>; onChange: (value: string) => void }) {
    return (
        <label className="field">
            <span>{props.label}</span>
            <select value={props.value} onChange={(event) => props.onChange(event.target.value)}>
                {props.options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
            </select>
        </label>
    );
}
