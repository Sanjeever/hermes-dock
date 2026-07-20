import {useState} from 'react';
import {CheckCircle2, MessageSquare, QrCode, Save, Square, Unlink} from 'lucide-react';
import {QRCodeSVG} from 'qrcode.react';
import {FeishuGroupPolicySelect, Field, PolicySelect, SecretField} from '../components/fields';
import {IconButton} from '../components/primitives';
import type {EnvVar, PlatformKey} from '../types';
import {enumValue, envValue, setEnvValue} from '../utils';

export function PlatformsPage(props: {
    env: EnvVar[];
    setEnv: (value: EnvVar[]) => void;
    qrData: string;
    qrStatus: string;
    qrPlatform: PlatformKey | '';
    selected: PlatformKey;
    setSelected: (value: PlatformKey) => void;
    busy: boolean;
    onWeixinLogin: () => void;
    onCancelWeixin: () => void;
    onFeishuLogin: () => void;
    onCancelFeishu: () => void;
    onDingTalkLogin: () => void;
    onCancelDingTalk: () => void;
    onSaveWeCom: () => Promise<boolean>;
    onSaveFeishu: () => Promise<boolean>;
    onSaveDingTalk: () => Promise<boolean>;
    onUnbind: (platform: PlatformKey) => void;
}) {
    const [confirmUnbind, setConfirmUnbind] = useState<PlatformKey | null>(null);
    const weixinBound = !!envValue(props.env, 'WEIXIN_ACCOUNT_ID') && !!envValue(props.env, 'WEIXIN_TOKEN');
    const wecomBound = !!envValue(props.env, 'WECOM_BOT_ID') && !!envValue(props.env, 'WECOM_SECRET');
    const feishuBound = !!envValue(props.env, 'FEISHU_APP_ID') && !!envValue(props.env, 'FEISHU_APP_SECRET');
    const dingtalkBound = !!envValue(props.env, 'DINGTALK_CLIENT_ID') && !!envValue(props.env, 'DINGTALK_CLIENT_SECRET');
    const set = (key: string, value: string) => props.setEnv(setEnvValue(props.env, key, value));
    const selectPlatform = (value: PlatformKey) => {
        setConfirmUnbind(null);
        props.setSelected(value);
    };
    const requestUnbind = (platform: PlatformKey) => setConfirmUnbind(platform);
    const confirmUnbindPlatform = () => {
        if (!confirmUnbind) return;
        props.onUnbind(confirmUnbind);
        setConfirmUnbind(null);
    };

    return (
        <section className="platform-stack">
            <div className="platform-cards">
                <PlatformCard id="weixin" selected={props.selected} bound={weixinBound} title="个人微信" note="适合个人测试，扫码登录" busy={props.busy} onSelect={selectPlatform}/>
                <PlatformCard id="wecom" selected={props.selected} bound={wecomBound} title="企业微信" note="适合企业微信 AI Bot" busy={props.busy} onSelect={selectPlatform}/>
                <PlatformCard id="feishu" selected={props.selected} bound={feishuBound} title="飞书 / Lark" note="适合飞书机器人" busy={props.busy} onSelect={selectPlatform}/>
                <PlatformCard id="dingtalk" selected={props.selected} bound={dingtalkBound} title="钉钉" note="适合钉钉 Stream 机器人" busy={props.busy} onSelect={selectPlatform}/>
            </div>
            {confirmUnbind && (
                <div className="danger-confirm platform-unbind-confirm">
                    <span>确认取消绑定{platformLabel(confirmUnbind)}？这会清空当前助手的绑定密钥，保存后需要应用配置才会影响运行中的服务。</span>
                    <button className="danger-button compact" onClick={confirmUnbindPlatform} disabled={props.busy}><Unlink size={16}/>确认取消绑定</button>
                    <button className="ghost" onClick={() => setConfirmUnbind(null)} disabled={props.busy}>取消</button>
                </div>
            )}
            {props.selected === 'weixin' && <WeixinPanel env={props.env} qrData={props.qrPlatform === 'weixin' ? props.qrData : ''} qrStatus={props.qrPlatform === 'weixin' ? props.qrStatus : ''} busy={props.busy} onWeixinLogin={props.onWeixinLogin} onCancelWeixin={props.onCancelWeixin} onUnbind={() => requestUnbind('weixin')}/>}
            {props.selected === 'wecom' && <WeComPanel env={props.env} set={set} busy={props.busy} onSave={props.onSaveWeCom} onUnbind={() => requestUnbind('wecom')}/>}
            {props.selected === 'feishu' && <FeishuPanel env={props.env} set={set} qrData={props.qrPlatform === 'feishu' ? props.qrData : ''} qrStatus={props.qrPlatform === 'feishu' ? props.qrStatus : ''} busy={props.busy} onLogin={props.onFeishuLogin} onCancel={props.onCancelFeishu} onSave={props.onSaveFeishu} onUnbind={() => requestUnbind('feishu')}/>}
            {props.selected === 'dingtalk' && <DingTalkPanel env={props.env} set={set} qrData={props.qrPlatform === 'dingtalk' ? props.qrData : ''} qrStatus={props.qrPlatform === 'dingtalk' ? props.qrStatus : ''} busy={props.busy} onLogin={props.onDingTalkLogin} onCancel={props.onCancelDingTalk} onSave={props.onSaveDingTalk} onUnbind={() => requestUnbind('dingtalk')}/>}
        </section>
    );
}

function PlatformCard(props: { id: PlatformKey; selected: PlatformKey; bound: boolean; title: string; note: string; busy: boolean; onSelect: (value: PlatformKey) => void }) {
    return (
        <button className={`platform-card ${props.selected === props.id ? 'selected' : ''}`} onClick={() => props.onSelect(props.id)} disabled={props.busy}>
            {props.bound ? <CheckCircle2 size={18}/> : <MessageSquare size={18}/>}
            <strong>{props.title}</strong>
            <span>{props.bound ? '已绑定' : props.note}</span>
        </button>
    );
}

function WeixinPanel(props: { env: EnvVar[]; qrData: string; qrStatus: string; busy: boolean; onWeixinLogin: () => void; onCancelWeixin: () => void; onUnbind: () => void }) {
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
                <button className="ghost danger-text" onClick={props.onUnbind} disabled={props.busy || !bound}><Unlink size={16}/>取消绑定</button>
            </div>
        </div>
    );
}

function WeComPanel(props: { env: EnvVar[]; set: (key: string, value: string) => void; busy: boolean; onSave: () => Promise<boolean>; onUnbind: () => void }) {
    const botID = envValue(props.env, 'WECOM_BOT_ID');
    const secret = envValue(props.env, 'WECOM_SECRET');
    const dmPolicy = closedPolicyValue(envValue(props.env, 'WECOM_DM_POLICY'));
    const groupPolicy = closedPolicyValue(envValue(props.env, 'WECOM_GROUP_POLICY'));
    const canSave = botID.trim() !== '' && secret.trim() !== '';
    const bound = canSave;
    const [secretVisible, setSecretVisible] = useState(false);
    return (
        <div className="panel">
            <p className="eyebrow">企业微信 AI Bot WebSocket</p>
            <Field label="Bot ID" value={botID} onChange={(value) => props.set('WECOM_BOT_ID', value)}/>
            <SecretField label="Secret" value={secret} visible={secretVisible} setVisible={setSecretVisible} onChange={(value) => props.set('WECOM_SECRET', value)}/>
            <Field label="WebSocket 地址" value={envValue(props.env, 'WECOM_WEBSOCKET_URL') || 'wss://openws.work.weixin.qq.com'} onChange={(value) => props.set('WECOM_WEBSOCKET_URL', value)}/>
            <div className="field-grid">
                <PolicySelect label="私聊策略" value={dmPolicy} onChange={(value) => props.set('WECOM_DM_POLICY', value)}/>
                <PolicySelect label="群聊策略" value={groupPolicy} onChange={(value) => props.set('WECOM_GROUP_POLICY', value)}/>
            </div>
            {!canSave && <div className="form-warning">请填写 Bot ID 和 Secret 后再保存。</div>}
            <div className="actions">
                <button className="primary" onClick={props.onSave} disabled={props.busy || !canSave}><Save size={16}/>保存企业微信配置</button>
                <button className="ghost danger-text" onClick={props.onUnbind} disabled={props.busy || !bound}><Unlink size={16}/>取消绑定</button>
            </div>
        </div>
    );
}

function FeishuPanel(props: { env: EnvVar[]; set: (key: string, value: string) => void; qrData: string; qrStatus: string; busy: boolean; onLogin: () => void; onCancel: () => void; onSave: () => Promise<boolean>; onUnbind: () => void }) {
    const appID = envValue(props.env, 'FEISHU_APP_ID');
    const appSecret = envValue(props.env, 'FEISHU_APP_SECRET');
    const domain = enumValue(envValue(props.env, 'FEISHU_DOMAIN'), ['feishu', 'lark'], 'feishu');
    const groupPolicy = disabledPolicyValue(envValue(props.env, 'FEISHU_GROUP_POLICY'));
    const canSave = appID.trim() !== '' && appSecret.trim() !== '';
    const bound = canSave;
    const [showAdvanced, setShowAdvanced] = useState(false);
    const [secretVisible, setSecretVisible] = useState(false);
    return (
        <div className="panel">
            <p className="eyebrow">飞书 / Lark WebSocket</p>
            <div className="qr-stage">
                {props.qrData ? <QRCodeSVG value={props.qrData} size={184}/> : bound ? <CheckCircle2 className="bound-icon" size={112}/> : <QrCode size={120}/>}
                <span>{props.qrStatus || (bound ? `已绑定${domain === 'lark' ? ' Lark' : '飞书'} ${maskID(appID)}` : '扫码创建飞书 / Lark 机器人并自动绑定。')}</span>
            </div>
            <div className="actions">
                <IconButton icon={QrCode} label={bound ? '重新扫码创建' : '扫码创建并绑定'} onClick={props.onLogin} disabled={props.busy}/>
                <IconButton icon={Square} label="取消" onClick={props.onCancel} disabled={props.busy || !props.qrStatus}/>
                <button className="ghost" onClick={() => setShowAdvanced((value) => !value)} disabled={props.busy}>{showAdvanced ? '收起已有应用配置' : '使用已有应用（高级）'}</button>
                <button className="ghost danger-text" onClick={props.onUnbind} disabled={props.busy || !bound}><Unlink size={16}/>取消绑定</button>
            </div>
            {showAdvanced && <>
            <div className="field-grid">
                <label className="field">
                    <span>平台区域</span>
                    <select value={domain} onChange={(event) => props.set('FEISHU_DOMAIN', event.target.value)}>
                        <option value="feishu">飞书（中国大陆）</option>
                        <option value="lark">Lark（海外）</option>
                    </select>
                </label>
                <Field label="App ID" value={appID} onChange={(value) => props.set('FEISHU_APP_ID', value)}/>
                <SecretField label="App Secret" value={appSecret} visible={secretVisible} setVisible={setSecretVisible} onChange={(value) => props.set('FEISHU_APP_SECRET', value)}/>
                <FeishuGroupPolicySelect label="群聊策略" value={groupPolicy} onChange={(value) => props.set('FEISHU_GROUP_POLICY', value)}/>
            </div>
            <div className="setting-note">已有应用使用 WebSocket 模式连接；群聊策略只保留开放和关闭。</div>
            {!canSave && <div className="form-warning">请填写 App ID 和 App Secret 后再保存。</div>}
            <div className="actions">
                <button className="primary" onClick={props.onSave} disabled={props.busy || !canSave}><Save size={16}/>保存飞书配置</button>
            </div>
            </>}
        </div>
    );
}

function DingTalkPanel(props: { env: EnvVar[]; set: (key: string, value: string) => void; qrData: string; qrStatus: string; busy: boolean; onLogin: () => void; onCancel: () => void; onSave: () => Promise<boolean>; onUnbind: () => void }) {
    const clientID = envValue(props.env, 'DINGTALK_CLIENT_ID');
    const clientSecret = envValue(props.env, 'DINGTALK_CLIENT_SECRET');
    const requireMention = envValue(props.env, 'DINGTALK_REQUIRE_MENTION') !== 'false';
    const canSave = clientID.trim() !== '' && clientSecret.trim() !== '';
    const bound = canSave;
    const [showAdvanced, setShowAdvanced] = useState(false);
    const [secretVisible, setSecretVisible] = useState(false);
    return (
        <div className="panel">
            <p className="eyebrow">钉钉 Stream 机器人</p>
            <div className="qr-stage">
                {props.qrData ? <QRCodeSVG value={props.qrData} size={184}/> : bound ? <CheckCircle2 className="bound-icon" size={112}/> : <QrCode size={120}/>}
                <span>{props.qrStatus || (bound ? `已绑定钉钉 ${maskID(clientID)}` : '扫码创建钉钉机器人并自动绑定。')}</span>
            </div>
            <div className="actions">
                <IconButton icon={QrCode} label={bound ? '重新扫码绑定' : '扫码创建并绑定'} onClick={props.onLogin} disabled={props.busy}/>
                <IconButton icon={Square} label="取消" onClick={props.onCancel} disabled={props.busy || !props.qrStatus}/>
                <button className="ghost" onClick={() => setShowAdvanced((value) => !value)} disabled={props.busy}>{showAdvanced ? '收起已有应用配置' : '使用已有应用（高级）'}</button>
                <button className="ghost danger-text" onClick={props.onUnbind} disabled={props.busy || !bound}><Unlink size={16}/>取消绑定</button>
            </div>
            {showAdvanced && <>
                <div className="field-grid">
                    <Field label="AppKey" value={clientID} onChange={(value) => props.set('DINGTALK_CLIENT_ID', value)}/>
                    <SecretField label="AppSecret" value={clientSecret} visible={secretVisible} setVisible={setSecretVisible} onChange={(value) => props.set('DINGTALK_CLIENT_SECRET', value)}/>
                    <label className="mini-toggle"><input type="checkbox" checked={requireMention} onChange={(event) => props.set('DINGTALK_REQUIRE_MENTION', event.target.checked ? 'true' : 'false')}/>群聊仅在 @机器人时回复</label>
                </div>
                <div className="setting-note">默认允许所有钉钉用户访问；扫码页可能显示 openClaw，这是钉钉授权页的来源标识。</div>
                {!canSave && <div className="form-warning">请填写 AppKey 和 AppSecret 后再保存。</div>}
                <div className="actions">
                    <button className="primary" onClick={props.onSave} disabled={props.busy || !canSave}><Save size={16}/>保存钉钉配置</button>
                </div>
            </>}
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

function platformLabel(platform: PlatformKey) {
    switch (platform) {
        case 'weixin':
            return '个人微信';
        case 'wecom':
            return '企业微信';
        case 'feishu':
            return '飞书 / Lark';
        case 'dingtalk':
            return '钉钉';
        default:
            return '平台';
    }
}
