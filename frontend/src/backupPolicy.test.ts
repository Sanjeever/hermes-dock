import {describe, expect, it} from 'vitest';
import type {InstanceBackupManifest} from './types';
import {inspectedBackupMatchesInput} from './backupPolicy';

const manifest = {path: '/tmp/A.hdbackup'} as InstanceBackupManifest;

describe('inspectedBackupMatchesInput', () => {
    it('invalidates a Web inspection when the input path changes', () => {
        expect(inspectedBackupMatchesInput(manifest, '/tmp/A.hdbackup', true)).toBe(true);
        expect(inspectedBackupMatchesInput(manifest, '/tmp/B.hdbackup', true)).toBe(false);
    });

    it('allows the native file picker manifest without a Web path input', () => {
        expect(inspectedBackupMatchesInput(manifest, '', false)).toBe(true);
    });
});
