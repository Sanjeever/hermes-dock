export namespace main {
	
	export class WebStatus {
	    enabled: boolean;
	    running: boolean;
	    host: string;
	    port: string;
	    localUrl: string;
	    lanUrls: string[];
	    primaryUrl: string;
	    usingDefaultPassword: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new WebStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.running = source["running"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.localUrl = source["localUrl"];
	        this.lanUrls = source["lanUrls"];
	        this.primaryUrl = source["primaryUrl"];
	        this.usingDefaultPassword = source["usingDefaultPassword"];
	        this.error = source["error"];
	    }
	}
	export class ChannelFile {
	    updated_at: string;
	    platforms: Record<string, Array<ChannelSummary>>;
	
	    static createFrom(source: any = {}) {
	        return new ChannelFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.updated_at = source["updated_at"];
	        this.platforms = this.convertValues(source["platforms"], Array<ChannelSummary>, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProviderConfigEntry {
	    label: string;
	    provider: string;
	    baseUrl: string;
	    apiMode: string;
	    apiKey: string;
	    modelListUrl: string;
	    defaultModel: string;
	    builtin: boolean;
	    disabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProviderConfigEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.provider = source["provider"];
	        this.baseUrl = source["baseUrl"];
	        this.apiMode = source["apiMode"];
	        this.apiKey = source["apiKey"];
	        this.modelListUrl = source["modelListUrl"];
	        this.defaultModel = source["defaultModel"];
	        this.builtin = source["builtin"];
	        this.disabled = source["disabled"];
	    }
	}
	export class ProviderConfig {
	    providers: Record<string, ProviderConfigEntry>;
	
	    static createFrom(source: any = {}) {
	        return new ProviderConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providers = this.convertValues(source["providers"], ProviderConfigEntry, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AuxModel {
	    provider: string;
	    model: string;
	    baseUrl: string;
	    apiKey: string;
	    timeout: number;
	    extraBody: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new AuxModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.baseUrl = source["baseUrl"];
	        this.apiKey = source["apiKey"];
	        this.timeout = source["timeout"];
	        this.extraBody = source["extraBody"];
	    }
	}
	export class ModelConfig {
	    provider: string;
	    default: string;
	    baseUrl: string;
	    apiMode: string;
	    apiKey: string;
	    auxiliaryMode: string;
	    auxiliary: Record<string, AuxModel>;
	    fallbacks: string[];
	    rawProviders: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new ModelConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.default = source["default"];
	        this.baseUrl = source["baseUrl"];
	        this.apiMode = source["apiMode"];
	        this.apiKey = source["apiKey"];
	        this.auxiliaryMode = source["auxiliaryMode"];
	        this.auxiliary = this.convertValues(source["auxiliary"], AuxModel, true);
	        this.fallbacks = source["fallbacks"];
	        this.rawProviders = source["rawProviders"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EnvVar {
	    key: string;
	    value: string;
	    secret: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EnvVar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.secret = source["secret"];
	    }
	}
	export class RuntimeProfileStatus {
	    enabled: boolean;
	    state: string;
	    pid: number;
	    startedAt: string;
	    lastExitCode: number;
	    restartCount: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeProfileStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.state = source["state"];
	        this.pid = source["pid"];
	        this.startedAt = source["startedAt"];
	        this.lastExitCode = source["lastExitCode"];
	        this.restartCount = source["restartCount"];
	        this.message = source["message"];
	    }
	}
	export class RuntimeStatus {
	    updatedAt: string;
	    profiles: Record<string, RuntimeProfileStatus>;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.updatedAt = source["updatedAt"];
	        this.profiles = this.convertValues(source["profiles"], RuntimeProfileStatus, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProfileEntry {
	    id: string;
	    name: string;
	    enabled: boolean;
	    createdAt: string;
	    updatedAt: string;
	    modelAuxiliaryMode: string;
	    setupCompletedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProfileEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.modelAuxiliaryMode = source["modelAuxiliaryMode"];
	        this.setupCompletedAt = source["setupCompletedAt"];
	    }
	}
	export class ProfileRegistry {
	    schemaVersion: number;
	    profiles: ProfileEntry[];
	
	    static createFrom(source: any = {}) {
	        return new ProfileRegistry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.profiles = this.convertValues(source["profiles"], ProfileEntry);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UIState {
	    lastPage: string;
	    lastProfile: string;
	
	    static createFrom(source: any = {}) {
	        return new UIState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lastPage = source["lastPage"];
	        this.lastProfile = source["lastProfile"];
	    }
	}
	export class BackupRecord {
	    id: string;
	    reason: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.reason = source["reason"];
	        this.path = source["path"];
	    }
	}
	export class MigrationRecord {
	    id: string;
	    appliedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new MigrationRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.appliedAt = source["appliedAt"];
	    }
	}
	export class ComposeSettings {
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
	    memoryLimit: string;
	    cpuLimit: string;
	    shmSize: string;
	
	    static createFrom(source: any = {}) {
	        return new ComposeSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image = source["image"];
	        this.containerName = source["containerName"];
	        this.gatewayHost = source["gatewayHost"];
	        this.gatewayPort = source["gatewayPort"];
	        this.dashboardHost = source["dashboardHost"];
	        this.dashboardPort = source["dashboardPort"];
	        this.dashboardEnabled = source["dashboardEnabled"];
	        this.dashboardUsername = source["dashboardUsername"];
	        this.dashboardPassword = source["dashboardPassword"];
	        this.gatewayBusyInputMode = source["gatewayBusyInputMode"];
	        this.gatewayBusyAckEnabled = source["gatewayBusyAckEnabled"];
	        this.backgroundNotifications = source["backgroundNotifications"];
	        this.memoryLimit = source["memoryLimit"];
	        this.cpuLimit = source["cpuLimit"];
	        this.shmSize = source["shmSize"];
	    }
	}
	export class LauncherState {
	    schemaVersion: number;
	    appVersion: string;
	    instanceId: string;
	    managedCompose: boolean;
	    composeHash: string;
	    templateVersion: string;
	    skillsSnapshotImage: string;
	    hermesImage: string;
	    composeSettings: ComposeSettings;
	    previousHermesImage: string;
	    lastSuccessfulHermesImage: string;
	    initializedAt: string;
	    updatedAt: string;
	    migrations: MigrationRecord[];
	    backups: BackupRecord[];
	    ui: UIState;
	    modelAuxiliaryMode: string;
	
	    static createFrom(source: any = {}) {
	        return new LauncherState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemaVersion = source["schemaVersion"];
	        this.appVersion = source["appVersion"];
	        this.instanceId = source["instanceId"];
	        this.managedCompose = source["managedCompose"];
	        this.composeHash = source["composeHash"];
	        this.templateVersion = source["templateVersion"];
	        this.skillsSnapshotImage = source["skillsSnapshotImage"];
	        this.hermesImage = source["hermesImage"];
	        this.composeSettings = this.convertValues(source["composeSettings"], ComposeSettings);
	        this.previousHermesImage = source["previousHermesImage"];
	        this.lastSuccessfulHermesImage = source["lastSuccessfulHermesImage"];
	        this.initializedAt = source["initializedAt"];
	        this.updatedAt = source["updatedAt"];
	        this.migrations = this.convertValues(source["migrations"], MigrationRecord);
	        this.backups = this.convertValues(source["backups"], BackupRecord);
	        this.ui = this.convertValues(source["ui"], UIState);
	        this.modelAuxiliaryMode = source["modelAuxiliaryMode"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppState {
	    appVersion: string;
	    instanceRoot: string;
	    state: LauncherState;
	    profiles: ProfileRegistry;
	    activeProfile: string;
	    profileStatus: RuntimeStatus;
	    compose: ComposeSettings;
	    environment: EnvVar[];
	    model: ModelConfig;
	    providers: ProviderConfig;
	    channels: ChannelFile;
	    dockerAvailable: boolean;
	    composeAvailable: boolean;
	    containerStatus: string;
	    web: WebStatus;
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appVersion = source["appVersion"];
	        this.instanceRoot = source["instanceRoot"];
	        this.state = this.convertValues(source["state"], LauncherState);
	        this.profiles = this.convertValues(source["profiles"], ProfileRegistry);
	        this.activeProfile = source["activeProfile"];
	        this.profileStatus = this.convertValues(source["profileStatus"], RuntimeStatus);
	        this.compose = this.convertValues(source["compose"], ComposeSettings);
	        this.environment = this.convertValues(source["environment"], EnvVar);
	        this.model = this.convertValues(source["model"], ModelConfig);
	        this.providers = this.convertValues(source["providers"], ProviderConfig);
	        this.channels = this.convertValues(source["channels"], ChannelFile);
	        this.dockerAvailable = source["dockerAvailable"];
	        this.composeAvailable = source["composeAvailable"];
	        this.containerStatus = source["containerStatus"];
	        this.web = this.convertValues(source["web"], WebStatus);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class ChannelSummary {
	    id: string;
	    name: string;
	    type: string;
	    thread_id: string;
	
	    static createFrom(source: any = {}) {
	        return new ChannelSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.thread_id = source["thread_id"];
	    }
	}
	
	export class CreateProfileRequest {
	    id: string;
	    name: string;
	    enabled: boolean;
	    copyFrom: string;
	    copyMode: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateProfileRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	        this.copyFrom = source["copyFrom"];
	        this.copyMode = source["copyMode"];
	    }
	}
	
	export class FeishuConfig {
	    appId: string;
	    appSecret: string;
	    domain: string;
	    allowedUsers: string;
	    groupPolicy: string;
	
	    static createFrom(source: any = {}) {
	        return new FeishuConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appId = source["appId"];
	        this.appSecret = source["appSecret"];
	        this.domain = source["domain"];
	        this.allowedUsers = source["allowedUsers"];
	        this.groupPolicy = source["groupPolicy"];
	    }
	}
	
	
	
	export class ModelListRequest {
	    providerId: string;
	    providerKey: string;
	    apiKey: string;
	    baseUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelListRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerId = source["providerId"];
	        this.providerKey = source["providerKey"];
	        this.apiKey = source["apiKey"];
	        this.baseUrl = source["baseUrl"];
	    }
	}
	export class ModelOption {
	    id: string;
	    ownedBy: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ownedBy = source["ownedBy"];
	    }
	}
	export class ModelProviderPreset {
	    key: string;
	    label: string;
	    provider: string;
	    baseUrl: string;
	    apiMode: string;
	    defaultModel: string;
	    modelListUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelProviderPreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.provider = source["provider"];
	        this.baseUrl = source["baseUrl"];
	        this.apiMode = source["apiMode"];
	        this.defaultModel = source["defaultModel"];
	        this.modelListUrl = source["modelListUrl"];
	    }
	}
	
	
	
	
	
	
	export class SkillFileInfo {
	    path: string;
	    sizeBytes: number;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillFileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.sizeBytes = source["sizeBytes"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class SkillDetail {
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
	    preview: string;
	    previewTruncated: boolean;
	    files: SkillFileInfo[];
	    fileCount: number;
	    filesTruncated: boolean;
	    conflictPaths: string[];
	
	    static createFrom(source: any = {}) {
	        return new SkillDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.version = source["version"];
	        this.author = source["author"];
	        this.platforms = source["platforms"];
	        this.tags = source["tags"];
	        this.path = source["path"];
	        this.category = source["category"];
	        this.builtin = source["builtin"];
	        this.conflict = source["conflict"];
	        this.error = source["error"];
	        this.sizeBytes = source["sizeBytes"];
	        this.updatedAt = source["updatedAt"];
	        this.preview = source["preview"];
	        this.previewTruncated = source["previewTruncated"];
	        this.files = this.convertValues(source["files"], SkillFileInfo);
	        this.fileCount = source["fileCount"];
	        this.filesTruncated = source["filesTruncated"];
	        this.conflictPaths = source["conflictPaths"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SkillHubCategory {
	    key: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillHubCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.name = source["name"];
	    }
	}
	export class SkillHubSignature {
	    signed: boolean;
	    keyId: string;
	    contentHash: string;
	    packageMd5: string;
	    payload: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillHubSignature(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.signed = source["signed"];
	        this.keyId = source["keyId"];
	        this.contentHash = source["contentHash"];
	        this.packageMd5 = source["packageMd5"];
	        this.payload = source["payload"];
	    }
	}
	export class SkillHubFile {
	    path: string;
	    sha256: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new SkillHubFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.sha256 = source["sha256"];
	        this.size = source["size"];
	    }
	}
	export class SkillHubSecurity {
	    provider: string;
	    status: string;
	    text: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillHubSecurity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.status = source["status"];
	        this.text = source["text"];
	        this.url = source["url"];
	    }
	}
	export class SkillHubDetail {
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
	    ownerName: string;
	    homepage: string;
	    securityReports: SkillHubSecurity[];
	    files: SkillHubFile[];
	    fileCount: number;
	    signature: SkillHubSignature;
	
	    static createFrom(source: any = {}) {
	        return new SkillHubDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slug = source["slug"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.categoryName = source["categoryName"];
	        this.source = source["source"];
	        this.version = source["version"];
	        this.downloads = source["downloads"];
	        this.stars = source["stars"];
	        this.installs = source["installs"];
	        this.requiresApiKey = source["requiresApiKey"];
	        this.verified = source["verified"];
	        this.installed = source["installed"];
	        this.installedPath = source["installedPath"];
	        this.tags = source["tags"];
	        this.ownerName = source["ownerName"];
	        this.homepage = source["homepage"];
	        this.securityReports = this.convertValues(source["securityReports"], SkillHubSecurity);
	        this.files = this.convertValues(source["files"], SkillHubFile);
	        this.fileCount = source["fileCount"];
	        this.signature = this.convertValues(source["signature"], SkillHubSignature);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SkillHubQuery {
	    keyword: string;
	    category: string;
	    page: number;
	    pageSize: number;
	    sortBy: string;
	    order: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillHubQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.keyword = source["keyword"];
	        this.category = source["category"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.sortBy = source["sortBy"];
	        this.order = source["order"];
	    }
	}
	
	
	export class SkillHubSkill {
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
	
	    static createFrom(source: any = {}) {
	        return new SkillHubSkill(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slug = source["slug"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.categoryName = source["categoryName"];
	        this.source = source["source"];
	        this.version = source["version"];
	        this.downloads = source["downloads"];
	        this.stars = source["stars"];
	        this.installs = source["installs"];
	        this.requiresApiKey = source["requiresApiKey"];
	        this.verified = source["verified"];
	        this.installed = source["installed"];
	        this.installedPath = source["installedPath"];
	        this.tags = source["tags"];
	    }
	}
	export class SkillHubState {
	    skills: SkillHubSkill[];
	    categories: SkillHubCategory[];
	    total: number;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new SkillHubState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.skills = this.convertValues(source["skills"], SkillHubSkill);
	        this.categories = this.convertValues(source["categories"], SkillHubCategory);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SkillSummary {
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
	
	    static createFrom(source: any = {}) {
	        return new SkillSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.version = source["version"];
	        this.author = source["author"];
	        this.platforms = source["platforms"];
	        this.tags = source["tags"];
	        this.path = source["path"];
	        this.category = source["category"];
	        this.builtin = source["builtin"];
	        this.conflict = source["conflict"];
	        this.error = source["error"];
	        this.sizeBytes = source["sizeBytes"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class SkillsState {
	    activeProfile: string;
	    skills: SkillSummary[];
	    total: number;
	    builtinCount: number;
	    customCount: number;
	    conflictCount: number;
	
	    static createFrom(source: any = {}) {
	        return new SkillsState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activeProfile = source["activeProfile"];
	        this.skills = this.convertValues(source["skills"], SkillSummary);
	        this.total = source["total"];
	        this.builtinCount = source["builtinCount"];
	        this.customCount = source["customCount"];
	        this.conflictCount = source["conflictCount"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SyncBundledSkillsResult {
	    activeProfile: string;
	    syncedSkills: string[];
	    syncedFiles: number;

	    static createFrom(source: any = {}) {
	        return new SyncBundledSkillsResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activeProfile = source["activeProfile"];
	        this.syncedSkills = source["syncedSkills"];
	        this.syncedFiles = source["syncedFiles"];
	    }
	}
	export class TextFileRequest {
	    path: string;
	    content: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new TextFileRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.content = source["content"];
	        this.reason = source["reason"];
	    }
	}
	
	export class UpdateMirrorLink {
	    label: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateMirrorLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.url = source["url"];
	    }
	}
	export class UpdateInfo {
	    currentVersion: string;
	    latestVersion: string;
	    available: boolean;
	    dismissed: boolean;
	    releaseUrl: string;
	    assetUrl: string;
	    assetName: string;
	    mirrors: UpdateMirrorLink[];
	    checkedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.available = source["available"];
	        this.dismissed = source["dismissed"];
	        this.releaseUrl = source["releaseUrl"];
	        this.assetUrl = source["assetUrl"];
	        this.assetName = source["assetName"];
	        this.mirrors = this.convertValues(source["mirrors"], UpdateMirrorLink);
	        this.checkedAt = source["checkedAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class WeComConfig {
	    botId: string;
	    secret: string;
	    websocketUrl: string;
	    dmPolicy: string;
	    allowedUsers: string;
	    groupPolicy: string;
	    groupAllowUsers: string;
	
	    static createFrom(source: any = {}) {
	        return new WeComConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.botId = source["botId"];
	        this.secret = source["secret"];
	        this.websocketUrl = source["websocketUrl"];
	        this.dmPolicy = source["dmPolicy"];
	        this.allowedUsers = source["allowedUsers"];
	        this.groupPolicy = source["groupPolicy"];
	        this.groupAllowUsers = source["groupAllowUsers"];
	    }
	}
	export class WebSettingsRequest {
	    enabled: boolean;
	    host: string;
	    port: string;
	
	    static createFrom(source: any = {}) {
	        return new WebSettingsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.host = source["host"];
	        this.port = source["port"];
	    }
	}

}
