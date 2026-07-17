import type {EnvVar, PlatformKey, SyncBundledSkillsResult} from './types';
import {envValue} from './utils';

export function closedPolicyValue(value: string) {
    return value === 'open' || value === '' ? 'open' : 'closed';
}

export function disabledPolicyValue(value: string) {
    return value === 'open' || value === '' ? 'open' : 'disabled';
}

export function syncBundledSkillsMessage(result: SyncBundledSkillsResult) {
	if (result.syncedFiles === 0 && result.skippedFiles > 0) return `已保留 ${result.skippedFiles} 个用户修改文件`;
	if (result.syncedFiles === 0) return '内置技能已是最新';
	const skillCount = result.syncedSkills.length;
	const skipped = result.skippedFiles > 0 ? `，保留 ${result.skippedFiles} 个用户修改文件` : '';
	if (skillCount > 0) return `已同步 ${skillCount} 个内置技能，写入 ${result.syncedFiles} 个文件${skipped}`;
	return `已写入内置技能文件 ${result.syncedFiles} 个${skipped}`;
}

export function restoreDefaultSkillsMessage(result: SyncBundledSkillsResult) {
    if (result.syncedFiles === 0) return '没有可恢复的默认技能文件';
    const skillCount = result.syncedSkills.length;
    if (skillCount > 0) return `已恢复 ${skillCount} 个默认技能，写入 ${result.syncedFiles} 个文件`;
    return `已恢复默认技能文件 ${result.syncedFiles} 个`;
}

export function platformLabel(platform: PlatformKey) {
    const labels: Record<PlatformKey, string> = {
        weixin: '个人微信',
        wecom: '企业微信',
        feishu: '飞书 / Lark',
    };
    return labels[platform];
}

export function firstBoundPlatform(env: EnvVar[]): PlatformKey {
    if (envValue(env, 'WEIXIN_ACCOUNT_ID') && envValue(env, 'WEIXIN_TOKEN')) return 'weixin';
    if (envValue(env, 'WECOM_BOT_ID') && envValue(env, 'WECOM_SECRET')) return 'wecom';
    if (envValue(env, 'FEISHU_APP_ID') && envValue(env, 'FEISHU_APP_SECRET')) return 'feishu';
    return 'weixin';
}

export function channelStatusKey(platform: string, id: string, action: string) {
    return `${platform}:${id}:${action}`;
}

export function shouldPollRuntimeStatus(applyActive: boolean, busy: boolean, containerStatus: string, profileStates: string[]) {
    if (applyActive) return true;
    return !busy && containerStatus === 'running' && profileStates.includes('starting');
}
