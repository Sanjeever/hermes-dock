import {CheckCircle2, QrCode, Save, Square} from 'lucide-react';
import {QRCodeSVG} from 'qrcode.react';
import {FeishuGroupPolicySelect, Field, PolicySelect} from '../components/fields';
import {IconButton} from '../components/primitives';
import type {EnvVar} from '../types';
import {enumValue, envValue, setEnvValue} from '../utils';

export function PlatformsPage(props: {
    env: EnvVar[];
    setEnv: (value: EnvVar[]) => void;
    qrData: string;
    qrStatus: string;
    busy: boolean;
    onSaveEnv: () => void;
    onWeixinLogin: () => void;
    onCancelWeixin: () => void;
    onSaveWeCom: () => void;
    onSaveFeishu: () => void;
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
    const weixinAccountID = envValue(props.env, 'WEIXIN_ACCOUNT_ID');
    const weixinHomeChannel = envValue(props.env, 'WEIXIN_HOME_CHANNEL');
    const weixinBound = !!weixinAccountID && !!envValue(props.env, 'WEIXIN_TOKEN');
    const wecomDMPolicy = envValue(props.env, 'WECOM_DM_POLICY') || 'open';
    const wecomGroupPolicy = envValue(props.env, 'WECOM_GROUP_POLICY') || 'open';
    const feishuDomain = enumValue(envValue(props.env, 'FEISHU_DOMAIN'), ['feishu', 'lark'], 'feishu');
    const feishuGroupPolicy = enumValue(envValue(props.env, 'FEISHU_GROUP_POLICY'), ['open', 'allowlist', 'disabled'], 'allowlist');
    return (
        <section className="grid two">
            <div className="panel">
                <p className="eyebrow">个人微信</p>
                <div className="qr-stage">
                    {props.qrData ? <QRCodeSVG value={props.qrData} size={184}/> : weixinBound ? <CheckCircle2 className="bound-icon" size={112}/> : <QrCode size={120}/>}
                    <span>{props.qrStatus || (weixinBound ? `已绑定个人微信 ${maskID(weixinAccountID)}` : '点击扫码登录绑定个人微信。')}</span>
                    {weixinBound && weixinHomeChannel && !props.qrStatus && <small>默认通道 {maskID(weixinHomeChannel)}</small>}
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
            <div className="panel wide">
                <p className="eyebrow">飞书 / Lark WebSocket</p>
                <div className="field-grid">
                    <label className="field">
                        <span>平台区域</span>
                        <select value={feishuDomain} onChange={(event) => set('FEISHU_DOMAIN', event.target.value)}>
                            <option value="feishu">飞书（中国大陆）</option>
                            <option value="lark">Lark（海外）</option>
                        </select>
                    </label>
                    <Field label="App ID" value={envValue(props.env, 'FEISHU_APP_ID')} onChange={(value) => set('FEISHU_APP_ID', value)}/>
                    <Field label="App Secret" value={envValue(props.env, 'FEISHU_APP_SECRET')} secret onChange={(value) => set('FEISHU_APP_SECRET', value)}/>
                    <FeishuGroupPolicySelect label="群聊策略" value={feishuGroupPolicy} onChange={(value) => set('FEISHU_GROUP_POLICY', value)}/>
                </div>
                <Field label="允许用户" value={envValue(props.env, 'FEISHU_ALLOWED_USERS')} onChange={(value) => set('FEISHU_ALLOWED_USERS', value)}/>
                <div className="setting-note">使用 WebSocket 模式连接飞书开放平台；允许用户填写 Open ID，多个用逗号分隔。</div>
                <button className="primary" onClick={props.onSaveFeishu} disabled={props.busy}><Save size={16}/>保存飞书配置</button>
            </div>
        </section>
    );
}

function maskID(value: string) {
    if (value.length <= 10) return value;
    return `${value.slice(0, 6)}...${value.slice(-4)}`;
}
