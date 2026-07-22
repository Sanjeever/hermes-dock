import {fireEvent, render, screen} from '@testing-library/react';
import {describe, expect, it, vi} from 'vitest';
import {AdvancedPage} from './AdvancedPage';

vi.mock('../components/CodeEditor', () => ({CodeEditor: () => <div data-testid="code-editor"/>}));

function renderAdvanced(overrides: Partial<Parameters<typeof AdvancedPage>[0]> = {}) {
    const props: Parameters<typeof AdvancedPage>[0] = {
        options: [
            {value: 'data/config.yaml', label: 'Hermes 配置'},
            {value: 'data/.env', label: '环境变量'},
            {value: 'docker-compose.override.yaml', label: 'Compose 覆盖'},
        ],
        path: 'data/config.yaml',
        setPath: vi.fn(),
        open: true,
        setOpen: vi.fn(),
        content: 'value',
        setContent: vi.fn(),
        status: '已加载',
        dirty: true,
        busy: false,
        webRuntime: false,
        backupStatus: '',
        backupManifest: null,
        onExportBackup: vi.fn().mockResolvedValue(undefined),
        onInspectBackup: vi.fn().mockResolvedValue(undefined),
        onImportBackup: vi.fn().mockResolvedValue(undefined),
        onClearBackupManifest: vi.fn(),
        onSave: vi.fn(),
        onFactoryReset: vi.fn().mockResolvedValue(undefined),
        resetConfirmPhrase: '恢复出厂设置',
        ...overrides,
    };
    render(<AdvancedPage {...props}/>);
    return props;
}

describe('AdvancedPage confirmations', () => {
    it('keeps the current file until discarding unsaved changes is confirmed', () => {
        const props = renderAdvanced();

        fireEvent.change(screen.getByRole('combobox'), {target: {value: 'data/.env'}});
        expect(props.setPath).not.toHaveBeenCalled();
        expect(screen.getByRole('alertdialog', {name: '放弃未保存修改？'})).toBeInTheDocument();
        fireEvent.click(screen.getByRole('button', {name: '放弃修改并切换'}));
        expect(props.setPath).toHaveBeenCalledWith('data/.env');
    });

    it('requires typed confirmation before saving a Compose override in Web management', () => {
        const onSave = vi.fn();
        renderAdvanced({path: 'docker-compose.override.yaml', webRuntime: true, onSave});

        fireEvent.click(screen.getByRole('button', {name: '保存'}));
        const confirmButton = screen.getByRole('button', {name: '确认保存'});
        expect(confirmButton).toBeDisabled();
        expect(onSave).not.toHaveBeenCalled();
        fireEvent.change(screen.getByLabelText('输入“确认”'), {target: {value: '确认'}});
        fireEvent.click(confirmButton);
        expect(onSave).toHaveBeenCalledWith('确认');
    });
});
