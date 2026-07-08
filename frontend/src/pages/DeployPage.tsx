import {useState, type ReactNode} from 'react';
import {Network, RotateCcw, Save} from 'lucide-react';
import {Field, SecretField} from '../components/fields';
import type {ComposeSettings, ProxySettings} from '../types';
import {isPortValue} from '../utils';

export function DeployPage({section = 'basic', compose, proxy, setCompose, setProxy, dirty, busy, onSave, onDiscard}: {
    section?: 'basic' | 'network';
    compose: ComposeSettings;
    proxy: ProxySettings;
    setCompose: (value: ComposeSettings) => void;
    setProxy: (value: ProxySettings) => void;
    dirty: boolean;
    busy: boolean;
    onSave: () => void;
    onDiscard: () => void;
}) {
    const [passwordVisible, setPasswordVisible] = useState(false);
    const update = (key: keyof Omit<ComposeSettings, 'dashboardEnabled'>, value: string) => setCompose({...compose, dashboardEnabled: true, [key]: value});
    const updateProxyText = (key: keyof Omit<ProxySettings, 'enabled'>, value: string) => setProxy({...proxy, [key]: value});
    const portsValid = isPortValue(compose.gatewayPort) && isPortValue(compose.dashboardPort);
    const proxyReady = !proxy.enabled || !!(proxy.httpProxy.trim() || proxy.httpsProxy.trim() || proxy.allProxy.trim());
    const isBasic = section === 'basic';

    return (
        <section className="deploy-stack settings-stack">
            <div className="panel deploy-summary">
                <div>
                    <p className="eyebrow">{isBasic ? '基础设置' : '网络设置'}</p>
                    <h2>{isBasic ? '日常会用到的设置' : '端口、资源和代理'}</h2>
                    <p className="setup-subtitle">{isBasic ? '设置版本、管理页登录和消息处理方式。' : '一般不需要修改。端口冲突或网络不通时再调整。'}</p>
                </div>
                <div className="actions compact">
                    <button className="ghost" onClick={onDiscard} disabled={busy || !dirty}><RotateCcw size={16}/>放弃修改</button>
                    <button className="primary no-margin" onClick={onSave} disabled={busy || !portsValid || !proxyReady}><Save size={16}/>保存设置</button>
                </div>
            </div>

            {isBasic ? (
                <>
                    <div className="panel settings-list">
                        <SettingRow title="Hermes 版本" description="一般保持默认。需要指定版本时再修改。">
                            <Field label="镜像" value={compose.image} onChange={(value) => update('image', value)}/>
                        </SettingRow>
                        <SettingRow title="管理页登录" description="用于打开 Hermes 管理页。">
                            <div className="setting-control-stack">
                                <Field label="用户名" value={compose.dashboardUsername} onChange={(value) => update('dashboardUsername', value)}/>
                                <SecretField label="密码" value={compose.dashboardPassword} visible={passwordVisible} setVisible={setPasswordVisible} onChange={(value) => update('dashboardPassword', value)}/>
                                {compose.dashboardPassword === '123456' && <div className="form-warning">仍在使用默认密码，建议修改。</div>}
                            </div>
                        </SettingRow>
                    </div>
                    <div className="panel settings-list">
                        <SettingRow title="消息处理" description="助手忙碌时，新消息如何处理。">
                            <GatewaySelect
                                label="忙碌时"
                                value={compose.gatewayBusyInputMode || 'steer'}
                                onChange={(value) => update('gatewayBusyInputMode', value)}
                                options={[
                                    {value: 'queue', label: '排队处理'},
                                    {value: 'steer', label: '引导当前任务'},
                                    {value: 'interrupt', label: '中断当前任务'},
                                ]}
                            />
                        </SettingRow>
                        <SettingRow title="忙碌回复" description="是否自动回复“正在处理”。">
                            <GatewaySelect
                                label="自动回复"
                                value={compose.gatewayBusyAckEnabled || 'false'}
                                onChange={(value) => update('gatewayBusyAckEnabled', value)}
                                options={[
                                    {value: 'true', label: '启用'},
                                    {value: 'false', label: '关闭'},
                                ]}
                            />
                        </SettingRow>
                        <SettingRow title="后台通知" description="控制后台消息通知频率。">
                            <GatewaySelect
                                label="通知"
                                value={compose.backgroundNotifications || 'result'}
                                onChange={(value) => update('backgroundNotifications', value)}
                                options={[
                                    {value: 'all', label: '运行更新和最终结果'},
                                    {value: 'result', label: '仅最终结果'},
                                    {value: 'error', label: '仅失败结果'},
                                    {value: 'off', label: '关闭'},
                                ]}
                            />
                        </SettingRow>
                    </div>
                </>
            ) : (
                <>
                    <div className="panel settings-list">
                        <SettingRow title="管理页地址" description="用于在浏览器打开管理页面。">
                            <div className="setting-control-grid">
                                <Field label="监听地址" value={compose.dashboardHost} onChange={(value) => update('dashboardHost', value)}/>
                                <Field label="端口" value={compose.dashboardPort} onChange={(value) => update('dashboardPort', value)}/>
                            </div>
                        </SettingRow>
                        <SettingRow title="消息入口地址" description="平台消息进入 Hermes 的本机入口。">
                            <div className="setting-control-grid">
                                <Field label="监听地址" value={compose.gatewayHost} onChange={(value) => update('gatewayHost', value)}/>
                                <Field label="端口" value={compose.gatewayPort} onChange={(value) => update('gatewayPort', value)}/>
                            </div>
                        </SettingRow>
                        {!portsValid && <div className="form-warning settings-warning">端口必须是 1-65535 的数字。</div>}
                    </div>
                    <div className="panel settings-list">
                        <SettingRow title="资源配额" description="设备资源紧张时再调整。">
                            <div className="setting-control-grid three">
                                <Field label="内存限制" value={compose.memoryLimit} onChange={(value) => update('memoryLimit', value)}/>
                                <Field label="CPU 限制" value={compose.cpuLimit} onChange={(value) => update('cpuLimit', value)}/>
                                <Field label="共享内存" value={compose.shmSize} onChange={(value) => update('shmSize', value)}/>
                            </div>
                        </SettingRow>
                    </div>
                    <div className="panel settings-list">
                        <SettingRow title="宿主机代理" description="网络访问不通时再开启。">
                            <div className="setting-control-stack">
                                <label className="toggle">
                                    <input type="checkbox" checked={proxy.enabled} onChange={(event) => setProxy({...proxy, enabled: event.target.checked})}/>
                                    启用代理
                                </label>
                                <button className="ghost no-margin" type="button" onClick={() => setProxy({
                                    ...proxy,
                                    enabled: true,
                                    httpProxy: 'http://host.docker.internal:7890',
                                    httpsProxy: 'http://host.docker.internal:7890',
                                    noProxy: proxy.noProxy || 'localhost,127.0.0.1,::1,host.docker.internal',
                                })}>
                                    <Network size={16}/>使用常见本机代理
                                </button>
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
                </>
            )}
        </section>
    );
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
