import type {InstanceBackupManifest} from './types';

export function inspectedBackupMatchesInput(manifest: InstanceBackupManifest | null, inputPath: string, webRuntime: boolean) {
    if (!manifest) return false;
    if (!webRuntime) return true;
    const inspectedPath = manifest.path?.trim() || '';
    return inspectedPath !== '' && inspectedPath === inputPath.trim();
}
