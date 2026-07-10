import type {RuntimeProfileStatus, SkillsState, WizardStep} from '../types';
import {profileStatusText, slugProfileID, statusClassName} from '../utils';

export function assistantStatusLabel(setupCompletedAt?: string, status?: RuntimeProfileStatus, enabled = true, needsRebuild = false) {
    if (!setupCompletedAt) return '配置未完成';
    if (!enabled) return '已停用';
    if (needsRebuild) return '有未应用配置';
    return profileStatusText(status?.state, enabled);
}

export function assistantStatusClass(setupCompletedAt?: string, status?: RuntimeProfileStatus, enabled = true, needsRebuild = false) {
    if (!setupCompletedAt || needsRebuild) return 'warn-status';
    return statusClassName(status?.state, enabled);
}

export function createProfileValidationMessage(name: string, id: string, exists: boolean) {
    if (!name.trim()) return '请填写显示名。';
    if (!id.trim()) return '请填写助手 ID。';
    if (exists) return '该助手 ID 已存在，请换一个。';
    if (!/^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])$/.test(id) || id === 'default') {
        return '助手 ID 只能包含小写字母、数字和连字符，且不能使用 default。';
    }
    return '';
}

export function wizardStepHelp(step: WizardStep) {
    switch (step) {
        case 'model': return '填写 API 密钥后，Hermes 才能调用模型。';
        case 'soul': return '可以先用默认内容，之后随时修改。';
        case 'platforms': return '至少绑定一个平台后，助手才能接收消息。';
        case 'finish': return '确认配置结果，再决定是否立即应用。';
        default: return '';
    }
}

export function skillSummaryLabel(state: SkillsState) {
    const base = `已安装 ${state.total} 个`;
    if (state.conflictCount > 0) return `${base}，冲突 ${state.conflictCount} 组`;
    return `${base}，内置 ${state.builtinCount} 个，自定义 ${state.customCount} 个`;
}

export function formatBytes(value: number) {
    if (!value) return '0 B';
    if (value < 1024) return `${value} B`;
    if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
    return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

export function suggestProfileID(profiles: Array<{ id: string }>, name: string) {
    const base = slugProfileID(name).replace(/-+$/, '') || 'assistant';
    const used = new Set(profiles.map((profile) => profile.id));
    let id = base;
    let index = 2;
    while (used.has(id) || id === 'default') {
        const suffix = `-${index}`;
        id = `${base.slice(0, 40 - suffix.length).replace(/-+$/, '')}${suffix}`;
        index += 1;
    }
    return id;
}
