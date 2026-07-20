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
    const promise = new Promise<T>((next) => { resolve = next; });
    return {promise, resolve};
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
});
