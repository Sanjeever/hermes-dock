import {Save} from 'lucide-react';
import {Field} from '../components/fields';
import type {ComposeSettings} from '../types';
import {isPortValue} from '../utils';

export function DeployPage({compose, setCompose, busy, onSave}: { compose: ComposeSettings; setCompose: (value: ComposeSettings) => void; busy: boolean; onSave: () => void }) {
    const update = (key: keyof Omit<ComposeSettings, 'dashboardEnabled'>, value: string) => setCompose({...compose, dashboardEnabled: true, [key]: value});
    const portsValid = isPortValue(compose.gatewayPort) && isPortValue(compose.dashboardPort);
    return (
        <section className="deploy-stack">
            <div className="panel deploy-summary">
                <div>
                    <p className="eyebrow">部署参数</p>
                    <h2>容器启动参数</h2>
                    <p className="setup-subtitle">这些设置保存后，需要应用并重建容器才会生效。</p>
                </div>
                <button className="primary no-margin" onClick={onSave} disabled={busy || !portsValid}><Save size={16}/>保存部署参数</button>
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
                    <Field label="控制台密码" value={compose.dashboardPassword} secret onChange={(value) => update('dashboardPassword', value)}/>
                    <div className="setting-note">控制台固定启用。</div>
                </div>
            </div>
            <div className="panel">
                <p className="eyebrow">网关行为</p>
                <h2>消息处理策略</h2>
                <div className="field-grid">
                    <GatewaySelect
                        label="忙碌输入模式"
                        value={compose.gatewayBusyInputMode || 'queue'}
                        onChange={(value) => update('gatewayBusyInputMode', value)}
                        options={[
                            {value: 'queue', label: '排队处理'},
                            {value: 'steer', label: '引导当前任务'},
                            {value: 'interrupt', label: '中断当前任务'},
                        ]}
                    />
                    <GatewaySelect
                        label="忙碌确认回复"
                        value={compose.gatewayBusyAckEnabled || 'true'}
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
