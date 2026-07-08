import {Eye, EyeOff} from 'lucide-react';

export function PolicySelect({label, value, onChange}: { label: string; value: string; onChange: (value: string) => void }) {
    return (
        <label className="field">
            <span>{label}</span>
            <select value={value || 'open'} onChange={(event) => onChange(event.target.value)}>
                <option value="open">开放</option>
                <option value="closed">关闭</option>
            </select>
        </label>
    );
}

export function FeishuGroupPolicySelect({label, value, onChange}: { label: string; value: string; onChange: (value: string) => void }) {
    return (
        <label className="field">
            <span>{label}</span>
            <select value={value || 'open'} onChange={(event) => onChange(event.target.value)}>
                <option value="open">开放</option>
                <option value="disabled">关闭</option>
            </select>
        </label>
    );
}

export function SecretField(props: { label: string; value: string; visible: boolean; setVisible: (value: boolean) => void; onChange: (value: string) => void }) {
    return (
        <label className="field">
            <span>{props.label}</span>
            <div className="secret-input">
                <input type={props.visible ? 'text' : 'password'} value={props.value || ''} onChange={(event) => props.onChange(event.target.value)} autoComplete="off"/>
                <button type="button" onClick={() => props.setVisible(!props.visible)} title={props.visible ? '隐藏密钥' : '显示密钥'} aria-label={props.visible ? '隐藏密钥' : '显示密钥'}>
                    {props.visible ? <EyeOff size={16}/> : <Eye size={16}/>}
                </button>
            </div>
        </label>
    );
}

export function Field({label, value, onChange, secret = false}: { label: string; value: string; onChange: (value: string) => void; secret?: boolean }) {
    return (
        <label className="field">
            <span>{label}</span>
            <input type={secret ? 'password' : 'text'} value={value || ''} onChange={(event) => onChange(event.target.value)} autoComplete={secret ? 'off' : undefined}/>
        </label>
    );
}
