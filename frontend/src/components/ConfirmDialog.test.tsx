import {fireEvent, render, screen} from '@testing-library/react';
import {describe, expect, it, vi} from 'vitest';
import {ConfirmDialog} from './ConfirmDialog';

describe('ConfirmDialog', () => {
    it('requires an explicit confirmation', () => {
        const onConfirm = vi.fn();
        const onCancel = vi.fn();
        render(<ConfirmDialog open title="强制重建" description="服务会短暂中断。" confirmLabel="确认强制重建" tone="danger" onConfirm={onConfirm} onCancel={onCancel}/>);

        expect(screen.getByRole('alertdialog', {name: '强制重建'})).toBeInTheDocument();
        expect(screen.getByRole('button', {name: '取消'})).toHaveFocus();
        expect(onConfirm).not.toHaveBeenCalled();
        fireEvent.click(screen.getByRole('button', {name: '确认强制重建'}));
        expect(onConfirm).toHaveBeenCalledTimes(1);
    });

    it('cancels with Escape and restores focus', () => {
        const trigger = document.createElement('button');
        document.body.appendChild(trigger);
        trigger.focus();
        const onCancel = vi.fn();
        const view = render(<ConfirmDialog open title="确认操作" description="确认说明" confirmLabel="继续" onConfirm={vi.fn()} onCancel={onCancel}/>);

        fireEvent.keyDown(document, {key: 'Escape'});
        expect(onCancel).toHaveBeenCalledTimes(1);
        view.unmount();
        expect(trigger).toHaveFocus();
        trigger.remove();
    });

    it('disables actions while busy', () => {
        render(<ConfirmDialog open title="确认操作" description="确认说明" confirmLabel="继续" busy onConfirm={vi.fn()} onCancel={vi.fn()}/>);

        expect(screen.getByRole('button', {name: '取消'})).toBeDisabled();
        expect(screen.getByRole('button', {name: '继续'})).toBeDisabled();
    });
});
