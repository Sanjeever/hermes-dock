import * as wailsApp from '../../wailsjs/go/main/App';
import type {
    AppState,
    BatchProfileConfigRequest,
    BatchProfileConfigResult,
    BundledContentSyncRequest,
    BundledContentSyncResult,
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
    UpdateStatus,
    WebSettingsRequest,
    WebTextFileKind,
} from '../types';

type RPCResponse<T> = { ok: true; result: T } | { ok: false; error: string };

const webRequestTimeout = 15_000;
const activeWebRequests = new Set<AbortController>();

export const isWebRuntime = () => typeof window !== 'undefined' && !(window as any).go?.main?.App;
export const webClientID = globalThis.crypto?.randomUUID ? globalThis.crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`;

async function rpc<T>(method: string, params: unknown[] = []): Promise<T> {
    const response = await fetchWithTimeout('/api/rpc', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        credentials: 'same-origin',
        body: JSON.stringify({method, params}),
    });
    if (response.status === 401) {
        window.dispatchEvent(new CustomEvent('web-session-expired'));
        throw new Error('登录已失效，请重新登录');
    }
    const body = await readJSON<RPCResponse<T>>(response);
    if (!response.ok) throw new Error(responseError(body, `请求失败（HTTP ${response.status}）`));
    if (!body.ok) throw new Error(body.error || '操作失败');
    return body.result as T;
}

async function fetchWithTimeout(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
    const controller = new AbortController();
    let timedOut = false;
    activeWebRequests.add(controller);
    const timer = window.setTimeout(() => {
        timedOut = true;
        controller.abort();
    }, webRequestTimeout);
    try {
        return await fetch(input, {...init, signal: controller.signal});
    } catch (error) {
        if (controller.signal.aborted) {
            throw new Error(timedOut ? '请求超时，请稍后重试' : '请求已取消');
        }
        const message = error instanceof Error ? error.message : String(error);
        throw new Error(`网络请求失败：${message}`);
    } finally {
        window.clearTimeout(timer);
        activeWebRequests.delete(controller);
    }
}

export function cancelWebRequests() {
    activeWebRequests.forEach((controller) => controller.abort());
    activeWebRequests.clear();
}

async function readJSON<T>(response: Response): Promise<T> {
    try {
        return await response.json() as T;
    } catch {
        if (!response.ok) throw new Error(`请求失败（HTTP ${response.status}，响应不是有效 JSON）`);
        throw new Error('服务器返回的数据格式错误');
    }
}

function responseError(body: unknown, fallback: string) {
    if (body && typeof body === 'object' && typeof (body as {error?: unknown}).error === 'string') {
        return (body as {error: string}).error || fallback;
    }
    return fallback;
}

function wailsOrRPC<T>(name: string, params: unknown[] = []): Promise<T> {
    if (!isWebRuntime()) {
        const fn = (wailsApp as unknown as Record<string, (...args: unknown[]) => Promise<unknown>>)[name];
        return fn(...params) as Promise<T>;
    }
    return rpc<T>(name, params);
}

export async function loginWeb(password: string) {
    const response = await fetchWithTimeout('/api/login', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        credentials: 'same-origin',
        body: JSON.stringify({password}),
    });
    const body = await readJSON<{ok?: boolean; error?: string}>(response);
    if (!response.ok || body.ok === false) throw new Error(responseError(body, `登录失败（HTTP ${response.status}）`));
    return body;
}

export async function logoutWeb() {
    const response = await fetchWithTimeout('/api/logout', {method: 'POST', credentials: 'same-origin'});
    if (response.status === 401) {
        window.dispatchEvent(new CustomEvent('web-session-expired'));
        throw new Error('登录已失效，请重新登录');
    }
    const body = await readJSON<{ok?: boolean; error?: string}>(response);
    if (!response.ok || body.ok === false) throw new Error(responseError(body, `退出登录失败（HTTP ${response.status}）`));
    return body;
}

export async function getWebSession(): Promise<{ authenticated: boolean; usingDefaultPassword: boolean }> {
    const response = await fetchWithTimeout('/api/session', {credentials: 'same-origin'});
    const body = await readJSON<{authenticated?: unknown; usingDefaultPassword?: unknown}>(response);
    if (!response.ok) throw new Error(responseError(body, `检查登录状态失败（HTTP ${response.status}）`));
    if (typeof body.authenticated !== 'boolean') throw new Error('服务器返回的数据格式错误');
    return {
        authenticated: body.authenticated,
        usingDefaultPassword: body.usingDefaultPassword === true,
    };
}

export const GetAppState = (profileID = '') => wailsOrRPC<AppState>('GetAppStateForProfile', [profileID]);
export const StartHermes = () => wailsOrRPC<void>('StartHermes');
export const StopHermes = () => wailsOrRPC<void>('StopHermes');
export const RestartHermes = () => wailsOrRPC<void>('RestartHermes');
export const RebuildHermes = () => wailsOrRPC<void>('RebuildHermes');
export const ForceRebuildHermes = () => wailsOrRPC<void>('ForceRebuildHermes');
export const TailLogs = () => isWebRuntime() ? rpc<void>('TailLogs', [webClientID]) : wailsOrRPC<void>('TailLogs');
export const StopTailLogs = () => isWebRuntime() ? rpc<void>('StopTailLogs', [webClientID]) : wailsOrRPC<void>('StopTailLogs');
export const StartWeixinLogin = (profileID: string) => wailsOrRPC<void>('StartWeixinLoginForProfile', [profileID]);
export const CancelWeixinLogin = () => wailsOrRPC<void>('CancelWeixinLogin');
export const StartFeishuLogin = (profileID: string) => wailsOrRPC<void>('StartFeishuLoginForProfile', [profileID]);
export const CancelFeishuLogin = () => wailsOrRPC<void>('CancelFeishuLogin');
export const StartDingTalkLogin = (profileID: string) => wailsOrRPC<void>('StartDingTalkLoginForProfile', [profileID]);
export const CancelDingTalkLogin = () => wailsOrRPC<void>('CancelDingTalkLogin');
export const TestModel = (profileID: string) => wailsOrRPC<void>('TestModelForProfile', [profileID]);
export const CompleteProfileSetup = (id: string) => wailsOrRPC<void>('CompleteProfileSetup', [id]);
export const CreateProfile = (req: unknown) => wailsOrRPC<void>('CreateProfile', [req]);
export const DeleteProfile = (id: string, confirm = '') => isWebRuntime() ? rpc<void>('DeleteProfile', [{id, confirm}]) : wailsOrRPC<void>('DeleteProfile', [id]);
export const MoveProfile = (id: string, direction: string) => wailsOrRPC<void>('MoveProfile', [id, direction]);
export const UpdateProfileName = (id: string, name: string) => wailsOrRPC<void>('UpdateProfileName', [id, name]);
export const SetProfileEnabled = (id: string, enabled: boolean) => wailsOrRPC<void>('SetProfileEnabled', [id, enabled]);
export const SelectProfile = (id: string) => wailsOrRPC<void>('SelectProfile', [id]);
export const BatchCopyProfileConfig = (req: BatchProfileConfigRequest) => wailsOrRPC<BatchProfileConfigResult>('BatchCopyProfileConfig', [req]);
export const SyncBundledContent = (req: BundledContentSyncRequest) => wailsOrRPC<BundledContentSyncResult>('SyncBundledContent', [req]);
export const RetryPostUpdate = () => wailsOrRPC<UpdateStatus>('RetryPostUpdate');
export const GetRecommendedResourceLimits = () => wailsOrRPC<ResourceLimitsRecommendation>('GetRecommendedResourceLimits');
export const ChooseSharedDirectory = (currentPath: string) => {
    if (isWebRuntime()) return Promise.reject(new Error('请在桌面端选择目录'));
    return wailsApp.ChooseSharedDirectory(currentPath);
};
export const SaveComposeSettings = (settings: ComposeSettings) => wailsOrRPC<void>('SaveComposeSettings', [settings]);
export const SaveProxySettings = (settings: ProxySettings) => wailsOrRPC<void>('SaveProxySettings', [settings]);
export const SaveModelConfig = (profileID: string, model: ModelConfig) => wailsOrRPC<void>('SaveModelConfigForProfile', [profileID, model]);
export const SaveProviderConfig = (profileID: string, providers: ProviderConfig) => wailsOrRPC<void>('SaveProviderConfigForProfile', [profileID, providers]);
export const SaveWeComConfig = (profileID: string, config: unknown) => wailsOrRPC<void>('SaveWeComConfigForProfile', [profileID, config]);
export const SaveFeishuConfig = (profileID: string, config: unknown) => wailsOrRPC<void>('SaveFeishuConfigForProfile', [profileID, config]);
export const SaveDingTalkConfig = (profileID: string, config: unknown) => wailsOrRPC<void>('SaveDingTalkConfigForProfile', [profileID, config]);
export const ApplyRecommendedDingTalkSettings = (profileID: string) => wailsOrRPC<void>('ApplyRecommendedDingTalkSettingsForProfile', [profileID]);
export const UnbindPlatform = (profileID: string, platform: string) => wailsOrRPC<void>('UnbindPlatformForProfile', [profileID, platform]);
export const FetchProviderConfigModelList = (profileID: string, provider: ProviderEntry) => wailsOrRPC<unknown[]>('FetchProviderConfigModelListForProfile', [profileID, provider]);
export const SetHomeChannel = (profileID: string, platform: string, id: string) => wailsOrRPC<void>('SetHomeChannelForProfile', [profileID, platform, id]);
export const SendTestMessage = (profileID: string, platform: string, id: string, message: string) => wailsOrRPC<void>('SendTestMessageForProfile', [profileID, platform, id, message]);
export const ListProfileSkills = (profileID: string) => wailsOrRPC<unknown>('ListProfileSkillsForProfile', [profileID]);
export const GetSkillDetail = (profileID: string, path: string) => wailsOrRPC<unknown>('GetSkillDetailForProfile', [profileID, path]);
export const DeleteSkill = (profileID: string, path: string) => isWebRuntime() ? rpc<void>('DeleteSkillForProfile', [{profileId: profileID, path, confirm: true}]) : wailsOrRPC<void>('DeleteSkillForProfile', [profileID, path]);
export const BatchDeleteSkills = (profileID: string, paths: string[]) => isWebRuntime() ? rpc<void>('BatchDeleteSkillsForProfile', [{profileId: profileID, paths, confirm: true}]) : wailsOrRPC<void>('BatchDeleteSkillsForProfile', [profileID, paths]);
export const SyncBundledSkills = (profileID: string) => wailsOrRPC<SyncBundledSkillsResult>('SyncBundledSkillsForProfile', [profileID]);
export const RestoreDefaultSkills = (profileID: string) => isWebRuntime() ? rpc<SyncBundledSkillsResult>('RestoreDefaultSkillsForProfile', [{profileId: profileID, confirm: true}]) : wailsOrRPC<SyncBundledSkillsResult>('RestoreDefaultSkillsForProfile', [profileID]);
export const RestoreDefaultSoul = (profileID: string) => isWebRuntime() ? rpc<void>('RestoreDefaultSoulForProfile', [{profileId: profileID, confirm: true}]) : wailsOrRPC<void>('RestoreDefaultSoulForProfile', [profileID]);
export const ListSkillHubSkills = (profileID: string, query: SkillHubQuery) => wailsOrRPC<unknown>('ListSkillHubSkillsForProfile', [profileID, query]);
export const GetSkillHubDetail = (profileID: string, slug: string) => wailsOrRPC<unknown>('GetSkillHubDetailForProfile', [profileID, slug]);
export const InstallSkillHubSkill = (profileID: string, slug: string) => wailsOrRPC<void>('InstallSkillHubSkillForProfile', [profileID, slug]);
export const ReadWebTextFile = (profileID: string, kind: WebTextFileKind) => rpc<string>('ReadWebTextFile', [profileID, kind]);
export const SaveWebTextFile = (profileID: string, kind: WebTextFileKind, content: string, confirm = '') => rpc<void>('SaveWebTextFile', [{profileId: profileID, kind, content, confirm}]);
export const ReadTextFile = (profileID: string, path: string) => isWebRuntime() ? ReadWebTextFile(profileID, webTextFileKind(path)) : wailsOrRPC<string>('ReadTextFile', [path]);
export const SaveTextFile = (profileID: string, req: { path: string; content: string; reason?: string; confirm?: string }) => {
    if (isWebRuntime()) return SaveWebTextFile(profileID, webTextFileKind(req.path), req.content, req.confirm || '');
    return wailsOrRPC<void>('SaveTextFile', [req]);
};
export const FactoryResetInstance = () => wailsOrRPC<void>('FactoryResetInstance');
export const ExportInstanceBackup = (targetPath = '') => wailsOrRPC<InstanceBackupManifest>('ExportInstanceBackup', [targetPath]);
export const InspectInstanceBackup = (path = '') => wailsOrRPC<InstanceBackupManifest>('InspectInstanceBackup', [path]);
export const ImportInstanceBackup = (path: string, confirm: string) => wailsOrRPC<InstanceBackupImportResult>('ImportInstanceBackup', [{path, confirm}]);
export const OpenSkillDirectory = (profileID: string, path: string) => wailsOrRPC<void>('OpenSkillDirectoryForProfile', [profileID, path]);
export const OpenFileManagement = async () => {
    if (!isWebRuntime()) return wailsApp.OpenFileManagement();
    const url = await rpc<string>('OpenFileManagement');
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
export const InstallUpdate = (version: string) => wailsOrRPC<void>('InstallUpdate', [version]);
export const SetAutoUpdateEnabled = (enabled: boolean) => wailsOrRPC<UpdateStatus>('SetAutoUpdateEnabled', [enabled]);
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
