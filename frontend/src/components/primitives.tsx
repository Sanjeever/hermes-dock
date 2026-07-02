import type {ComponentType} from 'react';
import {CheckCircle2, CircleAlert} from 'lucide-react';

export function IconButton({icon: Icon, label, onClick, disabled = false}: { icon: ComponentType<{ size?: string | number }>; label: string; onClick: () => void; disabled?: boolean }) {
    return <button className="icon-button" onClick={onClick} disabled={disabled} title={label}><Icon size={17}/><span>{label}</span></button>;
}

export function Health({label, ok, onClick}: { label: string; ok: boolean; onClick?: () => void }) {
    const content = <>{ok ? <CheckCircle2 size={18}/> : <CircleAlert size={18}/>}<span>{label}</span></>;
    if (onClick) {
        return <button className={`health clickable ${ok ? 'ok' : 'warn'}`} onClick={onClick}>{content}</button>;
    }
    return <div className={`health ${ok ? 'ok' : 'warn'}`}>{content}</div>;
}
