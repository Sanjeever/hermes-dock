export namespace main {
	
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

}

