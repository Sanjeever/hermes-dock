import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import {beforeEach, describe, expect, it, vi} from 'vitest';
import {ProfileBatchTools} from './ProfileBatchTools';

const listProfileSkills = vi.fn();

vi.mock('../services/api', () => ({
    ListProfileSkills: (...args: unknown[]) => listProfileSkills(...args),
}));

const profiles = [
    {id: 'default', name: '默认助手', enabled: true, createdAt: '', updatedAt: '', modelAuxiliaryMode: 'auto'},
    {id: 'sales', name: '销售助手', enabled: true, createdAt: '', updatedAt: '', modelAuxiliaryMode: 'auto'},
    {id: 'support', name: '支持助手', enabled: true, createdAt: '', updatedAt: '', modelAuxiliaryMode: 'auto'},
];

describe('ProfileBatchTools', () => {
    beforeEach(() => {
        listProfileSkills.mockReset();
        listProfileSkills.mockResolvedValue({skills: []});
    });

    it('synchronizes both bundled content types to every profile by default', async () => {
        const onSync = vi.fn().mockResolvedValue({succeeded: 3, failed: 0, added: 2, updated: 4, unchanged: 6, skipped: 1, results: []});
        render(<ProfileBatchTools
            mode="sync"
            profiles={profiles}
            activeProfile="default"
            busy={false}
            onClose={vi.fn()}
            onCopy={vi.fn()}
            onSync={onSync}
        />);

        fireEvent.click(screen.getByRole('button', {name: '同步到 3 个助手'}));
        await waitFor(() => expect(onSync).toHaveBeenCalledWith({
            targetProfileIds: ['default', 'sales', 'support'],
            syncSoul: true,
            syncSkills: true,
        }));
        expect(await screen.findByText(/保留 1 项用户修改/)).toBeInTheDocument();
    });

    it('excludes the source profile and never includes API keys by default', async () => {
        const onCopy = vi.fn().mockResolvedValue({succeeded: 2, failed: 0, results: []});
        render(<ProfileBatchTools
            mode="copy"
            profiles={profiles}
            activeProfile="default"
            busy={false}
            onClose={vi.fn()}
            onCopy={onCopy}
            onSync={vi.fn()}
        />);

        fireEvent.click(screen.getByRole('button', {name: '应用到 2 个助手'}));
        await waitFor(() => expect(onCopy).toHaveBeenCalledTimes(1));
        expect(onCopy.mock.calls[0][0]).toMatchObject({
            sourceProfileId: 'default',
            targetProfileIds: ['sales', 'support'],
            copyMainModel: true,
            copyAuxiliary: true,
            copyProviders: true,
            includeApiKeys: false,
        });
    });

    it('loads and passes only explicitly selected skills', async () => {
        listProfileSkills.mockResolvedValue({
            skills: [
                {path: 'skills/custom/report', name: 'report', builtin: false},
                {path: 'skills/hermes-dock', name: 'hermes-dock', builtin: true},
            ],
        });
        const onCopy = vi.fn().mockResolvedValue({succeeded: 2, failed: 0, results: []});
        render(<ProfileBatchTools
            mode="copy"
            profiles={profiles}
            activeProfile="default"
            busy={false}
            onClose={vi.fn()}
            onCopy={onCopy}
            onSync={vi.fn()}
        />);

        fireEvent.click(screen.getByLabelText('选择技能'));
        const report = await screen.findByText('report');
        fireEvent.click(report.closest('label')!.querySelector('input')!);
        fireEvent.click(screen.getByRole('button', {name: '应用到 2 个助手'}));
        await waitFor(() => expect(onCopy).toHaveBeenCalledTimes(1));
        expect(onCopy.mock.calls[0][0].skillPaths).toEqual(['skills/custom/report']);
    });

    it('shows target-level failure details', async () => {
        const onCopy = vi.fn().mockResolvedValue({
            succeeded: 1,
            failed: 1,
            results: [
                {profileId: 'sales', success: true, changed: true, error: ''},
                {profileId: 'support', success: false, changed: false, error: '目标缺少供应商'},
            ],
        });
        render(<ProfileBatchTools
            mode="copy"
            profiles={profiles}
            activeProfile="default"
            busy={false}
            onClose={vi.fn()}
            onCopy={onCopy}
            onSync={vi.fn()}
        />);

        fireEvent.click(screen.getByRole('button', {name: '应用到 2 个助手'}));
        expect(await screen.findByText('目标缺少供应商')).toBeInTheDocument();
        expect(screen.getAllByText('support')).toHaveLength(2);
    });

    it('closes with Escape and restores focus', () => {
        const trigger = document.createElement('button');
        document.body.appendChild(trigger);
        trigger.focus();
        const onClose = vi.fn();
        const view = render(<ProfileBatchTools
            mode="sync"
            profiles={profiles}
            activeProfile="default"
            busy={false}
            onClose={onClose}
            onCopy={vi.fn()}
            onSync={vi.fn()}
        />);

        expect(screen.getAllByRole('button', {name: '关闭'})[0]).toHaveFocus();
        fireEvent.keyDown(document, {key: 'Escape'});
        expect(onClose).toHaveBeenCalledTimes(1);
        view.unmount();
        expect(trigger).toHaveFocus();
        trigger.remove();
    });
});
