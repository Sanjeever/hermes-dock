import * as wailsApp from '../../wailsjs/go/main/App';
import type {
    AppState,
    ComposeSettings,
    InstanceBackupImportResult,
    InstanceBackupManifest,
    ModelConfig,
    ProxySettings,
    ProviderConfig,
    ProviderEntry,
    ResourceLimitsRecommendation,
    SkillHubQuery,
    SyncBundledSkillsResult,
    UpdateInfo,
    WebSettingsRequest,
    WebTextFileKind,
} from '../types';

type RPCResponse<T> = { ok: true; result: T } | { ok: false; error: string };

export const isWebRuntime = () => typeof window !== 'undefined' && !(window as any).go?.main?.App;
export const webClientID = globalThis.crypto?.randomUUID ? globalThis.crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`;

async function rpc<T>(method: string, params: unknown[] = []): Promise<T> {
    const response = await fetch('/api/rpc', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        credentials: 'same-origin',
        body: JSON.stringify({method, params}),
    });
    if (response.status === 401) {
        window.dispatchEvent(new CustomEvent('web-session-expired'));
        throw new Error('登录已失效，请重新登录');
    }
    const body = await response.json() as RPCResponse<T>;
    if (!body.ok) throw new Error(body.error || '操作失败');
    return body.result as T;
}

function wailsOrRPC<T>(name: string, params: unknown[] = []): Promise<T> {
    if (!isWebRuntime()) {
        const fn = (wailsApp as unknown as Record<string, (...args: unknown[]) => Promise<unknown>>)[name];
        return fn(...params) as Promise<T>;
    }
    return rpc<T>(name, params);
}

export function loginWeb(password: string) {
    return fetch('/api/login', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        credentials: 'same-origin',
        body: JSON.stringify({password}),
    }).then(async (response) => {
        const body = await response.json().catch(() => ({}));
        if (!response.ok || body.ok === false) throw new Error(body.error || '登录失败');
        return body;
    });
}

export function logoutWeb() {
    return fetch('/api/logout', {method: 'POST', credentials: 'same-origin'});
}

export function getWebSession(): Promise<{ authenticated: boolean; usingDefaultPassword: boolean }> {
    return fetch('/api/session', {credentials: 'same-origin'}).then((response) => response.json());
}

export const GetAppState = () => wailsOrRPC<AppState>('GetAppState');
export const StartHermes = () => wailsOrRPC<void>('StartHermes');
export const StopHermes = () => wailsOrRPC<void>('StopHermes');
export const RestartHermes = () => wailsOrRPC<void>('RestartHermes');
export const RebuildHermes = () => wailsOrRPC<void>('RebuildHermes');
export const TailLogs = () => isWebRuntime() ? rpc<void>('TailLogs', [webClientID]) : wailsOrRPC<void>('TailLogs');
export const StopTailLogs = () => isWebRuntime() ? rpc<void>('StopTailLogs', [webClientID]) : wailsOrRPC<void>('StopTailLogs');
export const StartWeixinLogin = () => wailsOrRPC<void>('StartWeixinLogin');
export const CancelWeixinLogin = () => wailsOrRPC<void>('CancelWeixinLogin');
export const TestModel = () => wailsOrRPC<void>('TestModel');
export const CompleteProfileSetup = (id: string) => wailsOrRPC<void>('CompleteProfileSetup', [id]);
export const CreateProfile = (req: unknown) => wailsOrRPC<void>('CreateProfile', [req]);
export const DeleteProfile = (id: string, confirm = '') => isWebRuntime() ? rpc<void>('DeleteProfile', [{id, confirm}]) : wailsOrRPC<void>('DeleteProfile', [id]);
export const MoveProfile = (id: string, direction: string) => wailsOrRPC<void>('MoveProfile', [id, direction]);
export const UpdateProfileName = (id: string, name: string) => wailsOrRPC<void>('UpdateProfileName', [id, name]);
export const SetProfileEnabled = (id: string, enabled: boolean) => wailsOrRPC<void>('SetProfileEnabled', [id, enabled]);
export const SelectProfile = (id: string) => wailsOrRPC<void>('SelectProfile', [id]);
export const GetRecommendedResourceLimits = () => wailsOrRPC<ResourceLimitsRecommendation>('GetRecommendedResourceLimits');
export const SaveComposeSettings = (settings: ComposeSettings) => wailsOrRPC<void>('SaveComposeSettings', [settings]);
export const SaveProxySettings = (settings: ProxySettings) => wailsOrRPC<void>('SaveProxySettings', [settings]);
export const SaveModelConfig = (model: ModelConfig) => wailsOrRPC<void>('SaveModelConfig', [model]);
export const SaveProviderConfig = (providers: ProviderConfig) => wailsOrRPC<void>('SaveProviderConfig', [providers]);
export const SaveWeComConfig = (config: unknown) => wailsOrRPC<void>('SaveWeComConfig', [config]);
export const SaveFeishuConfig = (config: unknown) => wailsOrRPC<void>('SaveFeishuConfig', [config]);
export const UnbindPlatform = (platform: string) => wailsOrRPC<void>('UnbindPlatform', [platform]);
export const FetchProviderConfigModelList = (provider: ProviderEntry) => wailsOrRPC<unknown[]>('FetchProviderConfigModelList', [provider]);
export const SetHomeChannel = (platform: string, id: string) => wailsOrRPC<void>('SetHomeChannel', [platform, id]);
export const SendTestMessage = (platform: string, id: string, message: string) => wailsOrRPC<void>('SendTestMessage', [platform, id, message]);
export const ListProfileSkills = () => wailsOrRPC<unknown>('ListProfileSkills');
export const GetSkillDetail = (path: string) => wailsOrRPC<unknown>('GetSkillDetail', [path]);
export const DeleteSkill = (path: string) => isWebRuntime() ? rpc<void>('DeleteSkill', [{path, confirm: true}]) : wailsOrRPC<void>('DeleteSkill', [path]);
export const SyncBundledSkills = () => wailsOrRPC<SyncBundledSkillsResult>('SyncBundledSkills');
export const RestoreDefaultSkills = () => isWebRuntime() ? rpc<SyncBundledSkillsResult>('RestoreDefaultSkills', [{confirm: true}]) : wailsOrRPC<SyncBundledSkillsResult>('RestoreDefaultSkills');
export const RestoreDefaultSoul = () => isWebRuntime() ? rpc<void>('RestoreDefaultSoul', [{confirm: true}]) : wailsOrRPC<void>('RestoreDefaultSoul');
export const ListSkillHubSkills = (query: SkillHubQuery) => wailsOrRPC<unknown>('ListSkillHubSkills', [query]);
export const GetSkillHubDetail = (slug: string) => wailsOrRPC<unknown>('GetSkillHubDetail', [slug]);
export const InstallSkillHubSkill = (slug: string) => wailsOrRPC<void>('InstallSkillHubSkill', [slug]);
export const ReadWebTextFile = (kind: WebTextFileKind) => rpc<string>('ReadWebTextFile', [kind]);
export const SaveWebTextFile = (kind: WebTextFileKind, content: string, confirm = '') => rpc<void>('SaveWebTextFile', [{kind, content, confirm}]);
export const ReadTextFile = (path: string) => isWebRuntime() ? ReadWebTextFile(webTextFileKind(path)) : wailsOrRPC<string>('ReadTextFile', [path]);
export const SaveTextFile = (req: { path: string; content: string; reason?: string; confirm?: string }) => {
    if (isWebRuntime()) return SaveWebTextFile(webTextFileKind(req.path), req.content, req.confirm || '');
    return wailsOrRPC<void>('SaveTextFile', [req]);
};
export const FactoryResetInstance = () => wailsOrRPC<void>('FactoryResetInstance');
export const ExportInstanceBackup = (targetPath = '') => wailsOrRPC<InstanceBackupManifest>('ExportInstanceBackup', [targetPath]);
export const InspectInstanceBackup = (path = '') => wailsOrRPC<InstanceBackupManifest>('InspectInstanceBackup', [path]);
export const ImportInstanceBackup = (path: string, confirm: string) => wailsOrRPC<InstanceBackupImportResult>('ImportInstanceBackup', [{path, confirm}]);
export const OpenSkillDirectory = (path: string) => wailsOrRPC<void>('OpenSkillDirectory', [path]);
export const OpenEndpoint = async (endpoint: string) => {
    if (!isWebRuntime()) return wailsApp.OpenEndpoint(endpoint);
    const url = await rpc<string>('OpenEndpoint', [endpoint]);
    window.open(url, '_blank', 'noopener,noreferrer');
};
export const SaveWebSettings = (settings: WebSettingsRequest) => wailsOrRPC<void>('SaveWebSettings', [settings]);
export const ChangeWebPassword = (oldPassword: string, newPassword: string) => wailsOrRPC<void>('ChangeWebPassword', [oldPassword, newPassword]);
export const ResetWebPassword = () => wailsOrRPC<void>('ResetWebPassword');
export const OpenWebManagement = () => {
    if (isWebRuntime()) {
        window.open(window.location.href, '_blank', 'noopener,noreferrer');
        return Promise.resolve();
    }
    return wailsOrRPC<void>('OpenWebManagement');
};
export const CheckForUpdates = (force: boolean) => wailsOrRPC<UpdateInfo>('CheckForUpdates', [force]);
export const DismissUpdate = (version: string) => wailsOrRPC<void>('DismissUpdate', [version]);
export const OpenUpdateURL = (url: string) => {
    if (isWebRuntime()) {
        window.open(url, '_blank', 'noopener,noreferrer');
        return Promise.resolve();
    }
    return wailsOrRPC<void>('OpenUpdateURL', [url]);
};

function webTextFileKind(path: string): WebTextFileKind {
    if (path === 'docker-compose.override.yaml') return 'compose_override';
    if (path.endsWith('/SOUL.md') || path === 'data/SOUL.md') return 'profile_soul';
    if (path.endsWith('/.env') || path === 'data/.env') return 'profile_env';
    if (path.endsWith('/config.yaml') || path === 'data/config.yaml') return 'profile_config';
    throw new Error('Web 管理不开放该文件');
}
