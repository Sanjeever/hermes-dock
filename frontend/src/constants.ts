import type {ComponentType} from 'react';
import {Activity, Boxes, LayoutDashboard, Settings} from 'lucide-react';
import type {ModelProviderPreset, Page, ProviderEntry, ProviderConfig} from './types';

export const factoryResetPhrase = '删除 ~/.hermes-dock';

export const nav: Array<{ id: Page; label: string; icon: ComponentType<{ size?: string | number }> }> = [
    {id: 'overview', label: '总览', icon: LayoutDashboard},
    {id: 'assistants', label: '助手', icon: Boxes},
    {id: 'operations', label: '运行', icon: Activity},
    {id: 'settings', label: '设置', icon: Settings},
];

export const auxLabels: Record<string, string> = {
    vision: '视觉理解',
    web_extract: '网页提取',
    compression: '上下文压缩',
    skills_hub: '技能中心',
    approval: '审批',
    mcp: 'MCP 配置',
    title_generation: '标题生成',
    tts_audio_tags: 'TTS 音频标签',
    triage_specifier: '任务分流',
    kanban_decomposer: '看板拆解',
    profile_describer: '档案描述',
    curator: '技能维护',
    monitor: '监控',
};

export const fallbackModelProviderPresets: ModelProviderPreset[] = [
    {
        key: 'dashscope-payg',
        label: '百炼按量计费',
        provider: 'custom',
        baseUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
        apiMode: 'chat_completions',
        defaultModel: 'qwen3.7-max',
        modelListUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1/models',
    },
    {
        key: 'bailian-coding-plan',
        label: '百炼 Coding Plan',
        provider: 'custom',
        baseUrl: 'https://coding.dashscope.aliyuncs.com/v1',
        apiMode: 'chat_completions',
        defaultModel: 'qwen3.7-max',
        modelListUrl: 'https://coding.dashscope.aliyuncs.com/v1/models',
    },
    {
        key: 'bailian-token-plan-team',
        label: '百炼 Token Plan 团队版',
        provider: 'custom',
        baseUrl: 'https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1',
        apiMode: 'chat_completions',
        defaultModel: 'qwen3.7-max',
        modelListUrl: 'https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1/models',
    },
    {
        key: 'zhipu-payg',
        label: '智谱按量计费',
        provider: 'custom',
        baseUrl: 'https://open.bigmodel.cn/api/paas/v4',
        apiMode: 'chat_completions',
        defaultModel: 'glm-5.2',
        modelListUrl: 'https://open.bigmodel.cn/api/paas/v4/models',
    },
    {
        key: 'zhipu-coding-plan',
        label: '智谱 Coding Plan',
        provider: 'custom',
        baseUrl: 'https://open.bigmodel.cn/api/coding/paas/v4',
        apiMode: 'chat_completions',
        defaultModel: 'glm-5.2',
        modelListUrl: 'https://open.bigmodel.cn/api/coding/paas/v4/models',
    },
    {
        key: 'opencode-go',
        label: 'OpenCode Go',
        provider: 'custom',
        baseUrl: 'https://opencode.ai/zen/go/v1',
        apiMode: 'chat_completions',
        defaultModel: 'deepseek-v4-flash',
        modelListUrl: 'https://opencode.ai/zen/go/v1/models',
    },
    {
        key: 'deepseek',
        label: 'DeepSeek',
        provider: 'deepseek',
        baseUrl: 'https://api.deepseek.com',
        apiMode: 'chat_completions',
        defaultModel: 'deepseek-v4-flash',
        modelListUrl: 'https://api.deepseek.com/models',
    },
    {
        key: 'agnes',
        label: 'Agnes AI',
        provider: 'custom',
        baseUrl: 'https://apihub.agnes-ai.com/v1',
        apiMode: 'chat_completions',
        defaultModel: 'agnes-2.0-flash',
        modelListUrl: 'https://apihub.agnes-ai.com/v1/models',
    },
];

export const fallbackProviderConfig: ProviderConfig = {
    providers: fallbackModelProviderPresets.reduce<Record<string, ProviderEntry>>((providers, preset) => {
        providers[preset.key] = {
            label: preset.label,
            provider: preset.provider,
            baseUrl: preset.baseUrl,
            apiMode: preset.apiMode,
            apiKey: '',
            modelListUrl: preset.modelListUrl,
            defaultModel: preset.defaultModel,
            builtin: true,
            disabled: false,
        };
        return providers;
    }, {}),
};
