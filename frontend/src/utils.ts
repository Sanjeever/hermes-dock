import {auxLabels, nav} from './constants';
import type {EnvVar, ModelConfig, ModelOption, Page, ProviderConfig, ProviderEntry} from './types';

export function titleFor(page: Page) {
    return nav.find((item) => item.id === page)?.label || '企智盒';
}

export function containerStatusText(status?: string) {
    switch (status) {
        case 'running':
            return '运行中';
        case 'stopped':
            return '已停止';
        case 'missing':
            return '未创建';
        case 'unknown':
            return '未知';
        default:
            return '未知';
    }
}

export function profileStatusText(state?: string, enabled = true) {
    if (!enabled) return '已停用';
    switch (state) {
        case 'running':
            return '运行中';
        case 'failed':
            return '失败';
        case 'not_configured':
            return '未绑定平台';
        case 'exited':
            return '已退出';
        case 'stopped':
            return '已停止';
        case 'disabled':
            return '已停用';
        case 'starting':
            return '启动中';
        default:
            return '未知';
    }
}

export function statusClassName(state?: string, enabled = true) {
    if (!enabled || state === 'disabled') return 'muted-status';
    if (state === 'running') return 'ok-status';
    if (state === 'failed') return 'bad-status';
    if (state === 'stopped') return 'muted-status';
    return 'warn-status';
}

export function slugProfileID(value: string) {
    return value.toLowerCase().replace(/[^a-z0-9-]+/g, '-').replace(/^-+/, '').slice(0, 40);
}

export function defaultAdvancedPath(profileID: string) {
    if (!profileID || profileID === 'default') return 'data/config.yaml';
    return `data/profiles/${profileID}/config.yaml`;
}

export function profileFilePath(profileID: string, name: string) {
    if (!profileID || profileID === 'default') return `data/${name}`;
    return `data/profiles/${profileID}/${name}`;
}

export function advancedFileOptions(profileID: string) {
    return [
        {value: profileFilePath(profileID, 'config.yaml'), label: `${profileID}/config.yaml`},
        {value: profileFilePath(profileID, '.env'), label: `${profileID}/.env`},
        {value: 'docker-compose.override.yaml', label: 'docker-compose.override.yaml'},
    ];
}

export function isPortValue(value: string) {
    if (!/^\d+$/.test(value)) return false;
    const port = Number(value);
    return port >= 1 && port <= 65535;
}

export function doneLabel(label: string) {
    return label.replace(/^正在/, '已');
}

export function envValue(env: EnvVar[], key: string) {
    return env.find((item) => item.key === key)?.value || '';
}

export function enumValue(value: string, allowed: string[], fallback: string) {
    return allowed.includes(value) ? value : fallback;
}

export function setEnvValue(env: EnvVar[], key: string, value: string) {
    const next = [...env];
    const index = next.findIndex((item) => item.key === key);
    if (index >= 0) {
        next[index] = {...next[index], value};
    } else {
        next.push({key, value, secret: /KEY|TOKEN|SECRET|PASSWORD|PASS|AUTH/i.test(key)});
    }
    return next;
}

export function providerIDs(config: ProviderConfig) {
    return Object.keys(config.providers || {}).sort((left, right) => {
        const a = config.providers[left];
        const b = config.providers[right];
        if (a.builtin !== b.builtin) return a.builtin ? -1 : 1;
        return (a.label || left).localeCompare(b.label || right, 'zh-Hans-CN');
    });
}

export function firstProviderID(config: ProviderConfig) {
    return providerIDs(config)[0] || 'dashscope-payg';
}

export function providerReferenceLabels(model: ModelConfig | null, providerID: string) {
    const refs: string[] = [];
    if (!model) return refs;
    if (model.provider === providerID) refs.push('主模型');
    for (const [key, aux] of Object.entries(model.auxiliary || {})) {
        if (aux.provider === providerID) refs.push(`辅助模型：${auxLabels[key] || key}`);
    }
    return refs;
}

export function nextProviderID(config: ProviderConfig, label: string) {
    const base = slugProviderID(label) || 'custom-provider';
    let id = base;
    let index = 2;
    while (config.providers[id]) {
        id = `${base}-${index}`;
        index += 1;
    }
    return id;
}

export function slugProviderID(label: string) {
    const ascii = label.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
    return ascii ? `custom-${ascii}` : 'custom-provider';
}

export function ensureCurrentModelOption(options: ModelOption[], current: string) {
    if (!current) return options;
    if (options.some((item) => item.id === current)) return options;
    return [{id: current, ownedBy: ''}, ...options];
}

export function modelOptionKey(providerID: string) {
    return providerID || 'dashscope-payg';
}

export function isVolcengineArkAgentPlanProvider(provider?: ProviderEntry) {
    if (!provider || provider.provider.trim().toLowerCase() !== 'custom') return false;
    try {
        const url = new URL(provider.baseUrl);
        return url.hostname.toLowerCase() === 'ark.cn-beijing.volces.com'
            && url.pathname.replace(/\/+$/, '') === '/api/plan/v3';
    } catch {
        return false;
    }
}

export function toPlainModelConfig(model: ModelConfig): any {
    const next = JSON.parse(JSON.stringify(model)) as ModelConfig;
    next.baseUrl = '';
    next.apiMode = '';
    next.apiKey = '';
    for (const aux of Object.values(next.auxiliary || {})) {
        aux.baseUrl = '';
        aux.apiKey = '';
    }
    return next;
}

export function toPlainProviderConfig(providers: ProviderConfig): any {
    return JSON.parse(JSON.stringify(providers));
}
