export type Page = 'overview' | 'assistants' | 'operations' | 'settings';
export type WizardStep = 'model' | 'soul' | 'platforms' | 'finish';
export type OperationsTab = 'runtime' | 'remote' | 'diagnostics' | 'basic' | 'network' | 'advanced';
export type PlatformKey = 'weixin' | 'wecom' | 'feishu';

export type EnvVar = { key: string; value: string; secret: boolean };
export type ComposeSettings = {
    image: string;
    containerName: string;
    gatewayHost: string;
    gatewayPort: string;
    dashboardHost: string;
    dashboardPort: string;
    dashboardEnabled: boolean;
    dashboardUsername: string;
    dashboardPassword: string;
    gatewayBusyInputMode: string;
    gatewayBusyAckEnabled: string;
    backgroundNotifications: string;
    hostControlEnabled: string;
    memoryLimit: string;
    cpuLimit: string;
    shmSize: string;
};
export type ResourceLimitsRecommendation = {
    memoryLimit: string;
    cpuLimit: string;
    dockerMemoryGB: number;
    dockerCPU: number;
};
export type ProxySettings = {
    enabled: boolean;
    httpProxy: string;
    httpsProxy: string;
    allProxy: string;
    noProxy: string;
};
export type AuxModel = { provider: string; model: string; baseUrl: string; apiKey: string; timeout: number; extraBody: Record<string, unknown> };
export type ModelConfig = {
    provider: string;
    default: string;
    baseUrl: string;
    apiMode: string;
    apiKey: string;
    auxiliaryMode: string;
    auxiliary: Record<string, AuxModel>;
};
export type ModelProviderPreset = {
    key: string;
    label: string;
    provider: string;
    baseUrl: string;
    apiMode: string;
    defaultModel: string;
    modelListUrl: string;
};
export type ProviderEntry = {
    label: string;
    provider: string;
    baseUrl: string;
    apiMode: string;
    apiKey: string;
    modelListUrl: string;
    defaultModel: string;
    builtin: boolean;
    disabled: boolean;
};
export type ProviderConfig = { providers: Record<string, ProviderEntry> };
export type ModelListRequest = { providerId: string; providerKey: string; apiKey: string; baseUrl: string };
export type ModelOption = { id: string; ownedBy: string };
export type ChannelSummary = { id: string; name: string; type: string; thread_id?: string };
export type ChannelFile = { updated_at: string; platforms: Record<string, ChannelSummary[]> };
export type ProfileEntry = { id: string; name: string; enabled: boolean; createdAt: string; updatedAt: string; modelAuxiliaryMode: string; setupCompletedAt?: string };
export type ProfileRegistry = { schemaVersion: number; profiles: ProfileEntry[] };
export type RuntimeProfileStatus = { enabled: boolean; state: string; pid: number; startedAt: string; lastExitCode: number; restartCount: number; message: string };
export type RuntimeStatus = { generation: string; updatedAt: string; profiles: Record<string, RuntimeProfileStatus> };
export type SkillSummary = {
    name: string;
    description: string;
    version: string;
    author: string;
    platforms: string[];
    tags: string[];
    path: string;
    category: string;
    builtin: boolean;
    conflict: boolean;
    error: string;
    sizeBytes: number;
    updatedAt: string;
};
export type SkillsState = {
    activeProfile: string;
    skills: SkillSummary[];
    total: number;
    builtinCount: number;
    customCount: number;
    conflictCount: number;
};
export type SyncBundledSkillsResult = { activeProfile: string; syncedSkills: string[]; syncedFiles: number };
export type SkillFileInfo = { path: string; sizeBytes: number; updatedAt: string };
export type SkillDetail = SkillSummary & {
    preview: string;
    previewTruncated: boolean;
    files: SkillFileInfo[];
    fileCount: number;
    filesTruncated: boolean;
    conflictPaths: string[];
};
export type SkillHubQuery = { keyword: string; category: string; page: number; pageSize: number; sortBy: string; order: string };
export type SkillHubCategory = { key: string; name: string };
export type SkillHubSkill = {
    slug: string;
    name: string;
    description: string;
    category: string;
    categoryName: string;
    source: string;
    version: string;
    downloads: number;
    stars: number;
    installs: number;
    requiresApiKey: boolean;
    verified: boolean;
    installed: boolean;
    installedPath: string;
    tags: string[];
};
export type SkillHubState = {
    skills: SkillHubSkill[];
    categories: SkillHubCategory[];
    total: number;
    page: number;
    pageSize: number;
};
export type SkillHubSecurity = { provider: string; status: string; text: string; url: string };
export type SkillHubFile = { path: string; sha256: string; size: number };
export type SkillHubSignature = { signed: boolean; keyId: string; contentHash: string; packageMd5: string; payload: string };
export type SkillHubDetail = SkillHubSkill & {
    ownerName: string;
    homepage: string;
    securityReports: SkillHubSecurity[];
    files: SkillHubFile[];
    fileCount: number;
    signature: SkillHubSignature;
};
export type Notice = { type: 'ok' | 'error' | 'info'; message: string };
export type RunOptions = { rebuildRequired?: boolean; beforeRefresh?: () => void; afterSuccess?: () => void };
export type UpdateMirrorLink = { label: string; url: string };
export type UpdateInfo = {
    currentVersion: string;
    latestVersion: string;
    available: boolean;
    dismissed: boolean;
    releaseUrl: string;
    assetUrl: string;
    assetName: string;
    mirrors: UpdateMirrorLink[];
    checkedAt: string;
};
export type InstanceBackupProfile = {
    id: string;
    name: string;
    enabled: boolean;
    isDefault: boolean;
};
export type InstanceBackupManifest = {
    format: string;
    schemaVersion: number;
    appVersion: string;
    templateVersion: string;
    createdAt: string;
    sourceInstanceRoot: string;
    includesSecrets: boolean;
    includesWebSettings: boolean;
    profiles: InstanceBackupProfile[];
    fileCount: number;
    totalBytes: number;
    excludedPaths: string[];
    path?: string;
};
export type InstanceBackupImportResult = {
    manifest: InstanceBackupManifest;
    preImportBackupPath: string;
};
export type AppState = {
    appVersion: string;
    instanceRoot: string;
    profiles: ProfileRegistry;
    activeProfile: string;
    profileStatus: RuntimeStatus;
    compose: ComposeSettings;
    proxy: ProxySettings;
    environment: EnvVar[];
    model: ModelConfig;
    providers: ProviderConfig;
    channels: ChannelFile;
    dockerAvailable: boolean;
    composeAvailable: boolean;
    containerStatus: string;
    web: WebStatus;
    hostBridge: HostBridgeStatus;
};

export type HostBridgeStatus = {
    enabled: boolean;
    running: boolean;
    address: string;
    error: string;
};

export type WebStatus = {
    enabled: boolean;
    running: boolean;
    host: string;
    port: string;
    localUrl: string;
    lanUrls: string[];
    primaryUrl: string;
    usingDefaultPassword: boolean;
    error: string;
};

export type WebSettingsRequest = {
    enabled: boolean;
    host: string;
    port: string;
};

export type WebTextFileKind = 'profile_config' | 'profile_env' | 'profile_soul' | 'compose_override';
