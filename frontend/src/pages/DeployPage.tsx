import {useState} from 'react';
import {Network, RotateCcw, Save} from 'lucide-react';
import {Field, SecretField} from '../components/fields';
import type {ComposeSettings, ProxySettings} from '../types';
import {isPortValue} from '../utils';

export function DeployPage({compose, proxy, setCompose, setProxy, dirty, busy, onSave, onDiscard}: { compose: ComposeSettings; proxy: ProxySettings; setCompose: (value: ComposeSettings) => void; setProxy: (value: ProxySettings) => void; dirty: boolean; busy: boolean; onSave: () => void; onDiscard: () => void }) {
    const [passwordVisible, setPasswordVisible] = useState(false);
    const update = (key: keyof Omit<ComposeSettings, 'dashboardEnabled'>, value: string) => setCompose({...compose, dashboardEnabled: true, [key]: value});
    const updateProxyText = (key: keyof Omit<ProxySettings, 'enabled'>, value: string) => setProxy({...proxy, [key]: value});
    const portsValid = isPortValue(compose.gatewayPort) && isPortValue(compose.dashboardPort);
    const proxyReady = !proxy.enabled || !!(proxy.httpProxy.trim() || proxy.httpsProxy.trim() || proxy.allProxy.trim());
    return (
        <section className="deploy-stack">
            <div className="panel deploy-summary">
                <div>
                    <p className="eyebrow">部署参数</p>
                    <h2>容器启动参数</h2>
                    <p className="setup-subtitle">这些设置保存后，需要应用并重建容器才会生效。</p>
                </div>
                <div className="actions compact">
                    <button className="ghost" onClick={onDiscard} disabled={busy || !dirty}><RotateCcw size={16}/>放弃修改</button>
                    <button className="primary no-margin" onClick={onSave} disabled={busy || !portsValid || !proxyReady}><Save size={16}/>保存部署参数</button>
                </div>
            </div>
            <div className="deploy-grid">
                <div className="panel">
                    <p className="eyebrow">镜像</p>
                    <h2>Hermes 版本</h2>
                    <Field label="镜像" value={compose.image} onChange={(value) => update('image', value)}/>
                </div>
                <div className="panel">
                    <p className="eyebrow">访问端口</p>
                    <h2>本机入口</h2>
                    <div className="field-grid">
                        <Field label="网关监听地址" value={compose.gatewayHost} onChange={(value) => update('gatewayHost', value)}/>
                        <Field label="网关端口" value={compose.gatewayPort} onChange={(value) => update('gatewayPort', value)}/>
                        <Field label="控制台监听地址" value={compose.dashboardHost} onChange={(value) => update('dashboardHost', value)}/>
                        <Field label="控制台端口" value={compose.dashboardPort} onChange={(value) => update('dashboardPort', value)}/>
                    </div>
                    {!portsValid && <div className="form-warning">端口必须是 1-65535 的数字</div>}
                </div>
                <div className="panel">
                    <p className="eyebrow">资源限制</p>
                    <h2>容器配额</h2>
                    <div className="field-grid">
                        <Field label="内存限制" value={compose.memoryLimit} onChange={(value) => update('memoryLimit', value)}/>
                        <Field label="CPU 限制" value={compose.cpuLimit} onChange={(value) => update('cpuLimit', value)}/>
                        <Field label="共享内存" value={compose.shmSize} onChange={(value) => update('shmSize', value)}/>
                    </div>
                </div>
                <div className="panel">
                    <p className="eyebrow">控制台</p>
                    <h2>登录信息</h2>
                    <Field label="控制台用户名" value={compose.dashboardUsername} onChange={(value) => update('dashboardUsername', value)}/>
                    <SecretField label="控制台密码" value={compose.dashboardPassword} visible={passwordVisible} setVisible={setPasswordVisible} onChange={(value) => update('dashboardPassword', value)}/>
                    {compose.dashboardPassword === '123456' && <div className="form-warning">当前仍使用默认控制台密码，建议修改后保存并应用。</div>}
                    <div className="setting-note">控制台固定启用。</div>
                </div>
            </div>
            <div className="panel">
                <p className="eyebrow">网关行为</p>
                <h2>消息处理策略</h2>
                <div className="field-grid">
                    <GatewaySelect
                        label="忙碌输入模式"
                        value={compose.gatewayBusyInputMode || 'steer'}
                        onChange={(value) => update('gatewayBusyInputMode', value)}
                        options={[
                            {value: 'queue', label: '排队处理'},
                            {value: 'steer', label: '引导当前任务'},
                            {value: 'interrupt', label: '中断当前任务'},
                        ]}
                    />
                    <GatewaySelect
                        label="忙碌确认回复"
                        value={compose.gatewayBusyAckEnabled || 'false'}
                        onChange={(value) => update('gatewayBusyAckEnabled', value)}
                        options={[
                            {value: 'true', label: '启用'},
                            {value: 'false', label: '关闭'},
                        ]}
                    />
                    <GatewaySelect
                        label="后台通知"
                        value={compose.backgroundNotifications || 'result'}
                        onChange={(value) => update('backgroundNotifications', value)}
                        options={[
                            {value: 'all', label: '运行更新和最终结果'},
                            {value: 'result', label: '仅最终结果'},
                            {value: 'error', label: '仅失败结果'},
                            {value: 'off', label: '关闭'},
                        ]}
                    />
                </div>
                <div className="setting-note">保存后需要回到运行控制页应用并重建。</div>
            </div>
            <div className="panel">
                <div className="deploy-panel-head">
                    <div>
                        <p className="eyebrow">容器代理</p>
                        <h2>使用宿主机 HTTP 代理</h2>
                    </div>
                    <label className="toggle">
                        <input type="checkbox" checked={proxy.enabled} onChange={(event) => setProxy({...proxy, enabled: event.target.checked})}/>
                        启用
                    </label>
                </div>
                <div className="proxy-actions">
                    <button className="ghost" type="button" onClick={() => setProxy({
                        ...proxy,
                        enabled: true,
                        httpProxy: 'http://host.docker.internal:7890',
                        httpsProxy: 'http://host.docker.internal:7890',
                        noProxy: proxy.noProxy || 'localhost,127.0.0.1,::1,host.docker.internal',
                    })}>
                        <Network size={16}/>常见本机代理端口
                    </button>
                </div>
                <div className="field-grid">
                    <Field label="HTTP_PROXY" value={proxy.httpProxy} onChange={(value) => updateProxyText('httpProxy', value)}/>
                    <Field label="HTTPS_PROXY" value={proxy.httpsProxy} onChange={(value) => updateProxyText('httpsProxy', value)}/>
                    <Field label="ALL_PROXY" value={proxy.allProxy} onChange={(value) => updateProxyText('allProxy', value)}/>
                    <Field label="NO_PROXY" value={proxy.noProxy} onChange={(value) => updateProxyText('noProxy', value)}/>
                </div>
                {!proxyReady && <div className="form-warning">启用代理时，请至少填写一个代理地址。</div>}
                <div className="setting-note">容器里的 <code>127.0.0.1</code> 不是宿主机；宿主机代理通常使用 <code>host.docker.internal</code>。</div>
            </div>
        </section>
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
