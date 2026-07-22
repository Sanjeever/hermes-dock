import type {ReactNode} from 'react';
import {useEffect, useId, useRef} from 'react';
import {CircleAlert} from 'lucide-react';

export function ConfirmDialog(props: {
    open: boolean;
    title: string;
    description: string;
    confirmLabel: string;
    tone?: 'primary' | 'danger';
    busy?: boolean;
    confirmDisabled?: boolean;
    children?: ReactNode;
    onConfirm: () => void;
    onCancel: () => void;
}) {
    const dialogRef = useRef<HTMLElement | null>(null);
    const busyRef = useRef(!!props.busy);
    const cancelRef = useRef(props.onCancel);
    const titleID = useId();
    const descriptionID = useId();

    useEffect(() => {
        busyRef.current = !!props.busy;
        cancelRef.current = props.onCancel;
    }, [props.busy, props.onCancel]);

    useEffect(() => {
        if (!props.open) return;
        const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
        const focusable = () => Array.from(dialogRef.current?.querySelectorAll<HTMLElement>('button:not([disabled]), input:not([disabled])') || []);
        focusable()[0]?.focus();
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape' && !busyRef.current) {
                event.preventDefault();
                cancelRef.current();
                return;
            }
            if (event.key !== 'Tab') return;
            const items = focusable();
            if (items.length === 0) return;
            const first = items[0];
            const last = items[items.length - 1];
            if (event.shiftKey && document.activeElement === first) {
                event.preventDefault();
                last.focus();
            } else if (!event.shiftKey && document.activeElement === last) {
                event.preventDefault();
                first.focus();
            }
        };
        document.addEventListener('keydown', handleKeyDown);
        return () => {
            document.removeEventListener('keydown', handleKeyDown);
            previousFocus?.focus();
        };
    }, [props.open]);

    if (!props.open) return null;
    return (
        <div className="confirm-dialog-overlay">
            <section ref={dialogRef} className={`confirm-dialog ${props.tone === 'danger' ? 'danger' : ''}`} role="alertdialog" aria-modal="true" aria-labelledby={titleID} aria-describedby={descriptionID}>
                <div className="confirm-dialog-heading">
                    <span className="confirm-dialog-icon" aria-hidden="true"><CircleAlert size={20}/></span>
                    <div>
                        <h2 id={titleID}>{props.title}</h2>
                        <p id={descriptionID}>{props.description}</p>
                    </div>
                </div>
                {props.children && <div className="confirm-dialog-content">{props.children}</div>}
                <div className="confirm-dialog-actions">
                    <button className="ghost" type="button" onClick={props.onCancel} disabled={props.busy}>取消</button>
                    <button className={props.tone === 'danger' ? 'danger-button compact' : 'primary no-margin'} type="button" onClick={props.onConfirm} disabled={props.busy || props.confirmDisabled}>{props.confirmLabel}</button>
                </div>
            </section>
        </div>
    );
}
