import {CheckCircle2, MessageSquare, QrCode, Save, Square} from 'lucide-react';
import {QRCodeSVG} from 'qrcode.react';
import {FeishuGroupPolicySelect, Field, PolicySelect} from '../components/fields';
import {IconButton} from '../components/primitives';
import type {EnvVar, PlatformKey} from '../types';
import {enumValue, envValue, setEnvValue} from '../utils';

export function PlatformsPage(props: {
    env: EnvVar[];
    setEnv: (value: EnvVar[]) => void;
    qrData: string;
    qrStatus: string;
    selected: PlatformKey;
    setSelected: (value: PlatformKey) => void;
    busy: boolean;
    onWeixinLogin: () => void;
    onCancelWeixin: () => void;
    onSaveWeCom: () => Promise<boolean>;
    onSaveFeishu: () => Promise<boolean>;
}) {
    const weixinBound = !!envValue(props.env, 'WEIXIN_ACCOUNT_ID') && !!envValue(props.env, 'WEIXIN_TOKEN');
    const wecomBound = !!envValue(props.env, 'WECOM_BOT_ID') && !!envValue(props.env, 'WECOM_SECRET');
    const feishuBound = !!envValue(props.env, 'FEISHU_APP_ID') && !!envValue(props.env, 'FEISHU_APP_SECRET');
    const set = (key: string, value: string) => props.setEnv(setEnvValue(props.env, key, value));

    return (
        <section className="platform-stack">
            <div className="platform-cards">
                <PlatformCard id="weixin" selected={props.selected} bound={weixinBound} title="个人微信" note="适合个人测试，扫码登录" onSelect={props.setSelected}/>
                <PlatformCard id="wecom" selected={props.selected} bound={wecomBound} title="企业微信" note="适合企业微信 AI Bot" onSelect={props.setSelected}/>
                <PlatformCard id="feishu" selected={props.selected} bound={feishuBound} title="飞书 / Lark" note="适合飞书机器人" onSelect={props.setSelected}/>
            </div>
            {props.selected === 'weixin' && <WeixinPanel env={props.env} qrData={props.qrData} qrStatus={props.qrStatus} busy={props.busy} onWeixinLogin={props.onWeixinLogin} onCancelWeixin={props.onCancelWeixin}/>}
            {props.selected === 'wecom' && <WeComPanel env={props.env} set={set} busy={props.busy} onSave={props.onSaveWeCom}/>}
            {props.selected === 'feishu' && <FeishuPanel env={props.env} set={set} busy={props.busy} onSave={props.onSaveFeishu}/>}
        </section>
    );
}

function PlatformCard(props: { id: PlatformKey; selected: PlatformKey; bound: boolean; title: string; note: string; onSelect: (value: PlatformKey) => void }) {
    return (
        <button className={`platform-card ${props.selected === props.id ? 'selected' : ''}`} onClick={() => props.onSelect(props.id)}>
            {props.bound ? <CheckCircle2 size={18}/> : <MessageSquare size={18}/>}
            <strong>{props.title}</strong>
            <span>{props.bound ? '已绑定' : props.note}</span>
        </button>
    );
}

function WeixinPanel(props: { env: EnvVar[]; qrData: string; qrStatus: string; busy: boolean; onWeixinLogin: () => void; onCancelWeixin: () => void }) {
    const accountID = envValue(props.env, 'WEIXIN_ACCOUNT_ID');
    const homeChannel = envValue(props.env, 'WEIXIN_HOME_CHANNEL');
    const bound = !!accountID && !!envValue(props.env, 'WEIXIN_TOKEN');
    return (
        <div className="panel">
            <p className="eyebrow">个人微信</p>
            <div className="qr-stage">
                {props.qrData ? <QRCodeSVG value={props.qrData} size={184}/> : bound ? <CheckCircle2 className="bound-icon" size={112}/> : <QrCode size={120}/>}
                <span>{props.qrStatus || (bound ? `已绑定个人微信 ${maskID(accountID)}` : '点击扫码登录绑定个人微信。')}</span>
                {bound && homeChannel && !props.qrStatus && <small>默认通道 {maskID(homeChannel)}</small>}
            </div>
            <div className="actions">
                <IconButton icon={QrCode} label="扫码登录" onClick={props.onWeixinLogin} disabled={props.busy}/>
                <IconButton icon={Square} label="取消" onClick={props.onCancelWeixin} disabled={props.busy}/>
            </div>
        </div>
    );
}

function WeComPanel(props: { env: EnvVar[]; set: (key: string, value: string) => void; busy: boolean; onSave: () => Promise<boolean> }) {
    const dmPolicy = closedPolicyValue(envValue(props.env, 'WECOM_DM_POLICY'));
    const groupPolicy = closedPolicyValue(envValue(props.env, 'WECOM_GROUP_POLICY'));
    return (
        <div className="panel">
            <p className="eyebrow">企业微信 AI Bot WebSocket</p>
            <Field label="Bot ID" value={envValue(props.env, 'WECOM_BOT_ID')} onChange={(value) => props.set('WECOM_BOT_ID', value)}/>
            <Field label="Secret" value={envValue(props.env, 'WECOM_SECRET')} secret onChange={(value) => props.set('WECOM_SECRET', value)}/>
            <Field label="WebSocket 地址" value={envValue(props.env, 'WECOM_WEBSOCKET_URL') || 'wss://openws.work.weixin.qq.com'} onChange={(value) => props.set('WECOM_WEBSOCKET_URL', value)}/>
            <div className="field-grid">
                <PolicySelect label="私聊策略" value={dmPolicy} onChange={(value) => props.set('WECOM_DM_POLICY', value)}/>
                <PolicySelect label="群聊策略" value={groupPolicy} onChange={(value) => props.set('WECOM_GROUP_POLICY', value)}/>
            </div>
            <button className="primary" onClick={props.onSave} disabled={props.busy}><Save size={16}/>保存企业微信配置</button>
        </div>
    );
}

function FeishuPanel(props: { env: EnvVar[]; set: (key: string, value: string) => void; busy: boolean; onSave: () => Promise<boolean> }) {
    const domain = enumValue(envValue(props.env, 'FEISHU_DOMAIN'), ['feishu', 'lark'], 'feishu');
    const groupPolicy = disabledPolicyValue(envValue(props.env, 'FEISHU_GROUP_POLICY'));
    return (
        <div className="panel">
            <p className="eyebrow">飞书 / Lark WebSocket</p>
            <div className="field-grid">
                <label className="field">
                    <span>平台区域</span>
                    <select value={domain} onChange={(event) => props.set('FEISHU_DOMAIN', event.target.value)}>
                        <option value="feishu">飞书（中国大陆）</option>
                        <option value="lark">Lark（海外）</option>
                    </select>
                </label>
                <Field label="App ID" value={envValue(props.env, 'FEISHU_APP_ID')} onChange={(value) => props.set('FEISHU_APP_ID', value)}/>
                <Field label="App Secret" value={envValue(props.env, 'FEISHU_APP_SECRET')} secret onChange={(value) => props.set('FEISHU_APP_SECRET', value)}/>
                <FeishuGroupPolicySelect label="群聊策略" value={groupPolicy} onChange={(value) => props.set('FEISHU_GROUP_POLICY', value)}/>
            </div>
            <div className="setting-note">使用 WebSocket 模式连接飞书开放平台；群聊策略只保留开放和关闭。</div>
            <button className="primary" onClick={props.onSave} disabled={props.busy}><Save size={16}/>保存飞书配置</button>
        </div>
    );
}

function closedPolicyValue(value: string) {
    return value === 'open' || value === '' ? 'open' : 'closed';
}

function disabledPolicyValue(value: string) {
    return value === 'open' || value === '' ? 'open' : 'disabled';
}

function maskID(value: string) {
    if (value.length <= 10) return value;
    return `${value.slice(0, 6)}...${value.slice(-4)}`;
}
