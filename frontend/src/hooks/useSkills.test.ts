import {act, renderHook} from '@testing-library/react';
import {beforeEach, describe, expect, it, vi} from 'vitest';
import {ListProfileSkills} from '../services/api';
import type {SkillsState} from '../types';
import {useSkills} from './useSkills';

vi.mock('../services/api', () => ({
    BatchDeleteSkills: vi.fn(),
    DeleteSkill: vi.fn(),
    GetSkillDetail: vi.fn(),
    GetSkillHubDetail: vi.fn(),
    InstallSkillHubSkill: vi.fn(),
    ListProfileSkills: vi.fn(),
    ListSkillHubSkills: vi.fn(),
    OpenSkillDirectory: vi.fn(),
    RestoreDefaultSkills: vi.fn(),
    SyncBundledSkills: vi.fn(),
}));

function deferred<T>() {
    let resolve!: (value: T) => void;
    let reject!: (reason?: unknown) => void;
    const promise = new Promise<T>((next, fail) => {
        resolve = next;
        reject = fail;
    });
    return {promise, resolve, reject};
}

function skillsState(profileID: string): SkillsState {
    return {activeProfile: profileID, skills: [], total: 0, builtinCount: 0, customCount: 0, conflictCount: 0};
}

describe('useSkills', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('ignores a slower response from the previous profile', async () => {
        const oldRequest = deferred<SkillsState>();
        const newRequest = deferred<SkillsState>();
        vi.mocked(ListProfileSkills).mockImplementation((profileID) => (
            profileID === 'old' ? oldRequest.promise : newRequest.promise
        ));
        let profileID = 'old';
        const {result} = renderHook(() => useSkills({
            getProfileID: () => profileID,
            run: vi.fn(),
            refresh: vi.fn(async () => ''),
            appendLog: vi.fn(),
            setBusy: vi.fn(),
            setNotice: vi.fn(),
            setLastOperationError: vi.fn(),
            setNeedsRebuild: vi.fn(),
        }));

        let oldLoad!: Promise<void>;
        act(() => { oldLoad = result.current.loadSkills(); });
        profileID = 'new';
        let newLoad!: Promise<void>;
        act(() => {
            result.current.resetForProfile();
            newLoad = result.current.loadSkills();
        });
        await act(async () => {
            newRequest.resolve(skillsState('new'));
            await newLoad;
        });
        expect(result.current.skillsState?.activeProfile).toBe('new');

        await act(async () => {
            oldRequest.resolve(skillsState('old'));
            await oldLoad;
        });
        expect(result.current.skillsState?.activeProfile).toBe('new');
    });

    it('does not report an async failure after unmount', async () => {
        const pending = deferred<SkillsState>();
        vi.mocked(ListProfileSkills).mockReturnValue(pending.promise);
        const appendLog = vi.fn();
        const {result, unmount} = renderHook(() => useSkills({
            getProfileID: () => 'default',
            run: vi.fn(),
            refresh: vi.fn(async () => ''),
            appendLog,
            setBusy: vi.fn(),
            setNotice: vi.fn(),
            setLastOperationError: vi.fn(),
            setNeedsRebuild: vi.fn(),
        }));

        let load!: Promise<void>;
        act(() => { load = result.current.loadSkills(); });
        unmount();
        await act(async () => {
            pending.reject(new Error('late failure'));
            await load;
        });

        expect(appendLog).not.toHaveBeenCalled();
    });
});
