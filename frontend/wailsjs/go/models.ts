export namespace main {
	
	export class Diagnostic {
	    id: string;
	    label: string;
	    status: string;
	    message: string;
	    fixable: boolean;
	    severity: string;
	
	    static createFrom(source: any = {}) {
	        return new Diagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.fixable = source["fixable"];
	        this.severity = source["severity"];
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
	export class UIState {
	    lastPage: string;
	
	    static createFrom(source: any = {}) {
	        return new UIState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lastPage = source["lastPage"];
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
	    compose: ComposeSettings;
	    environment: EnvVar[];
	    model: ModelConfig;
	    channels: ChannelFile;
	    diagnostics: Diagnostic[];
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
	        this.compose = this.convertValues(source["compose"], ComposeSettings);
	        this.environment = this.convertValues(source["environment"], EnvVar);
	        this.model = this.convertValues(source["model"], ModelConfig);
	        this.channels = this.convertValues(source["channels"], ChannelFile);
	        this.diagnostics = this.convertValues(source["diagnostics"], Diagnostic);
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
	
	
	
	
	
	
	export class ModelListRequest {
	    providerKey: string;
	    apiKey: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelListRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerKey = source["providerKey"];
	        this.apiKey = source["apiKey"];
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

