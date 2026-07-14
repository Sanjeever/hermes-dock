package main

type AppState struct {
	AppVersion       string           `json:"appVersion"`
	InstanceRoot     string           `json:"instanceRoot"`
	State            LauncherState    `json:"state"`
	Profiles         ProfileRegistry  `json:"profiles"`
	ActiveProfile    string           `json:"activeProfile"`
	ProfileStatus    RuntimeStatus    `json:"profileStatus"`
	Compose          ComposeSettings  `json:"compose"`
	Proxy            ProxySettings    `json:"proxy"`
	Environment      []EnvVar         `json:"environment"`
	Model            ModelConfig      `json:"model"`
	Providers        ProviderConfig   `json:"providers"`
	Channels         ChannelFile      `json:"channels"`
	DockerAvailable  bool             `json:"dockerAvailable"`
	ComposeAvailable bool             `json:"composeAvailable"`
	ContainerStatus  string           `json:"containerStatus"`
	Web              WebStatus        `json:"web"`
	HostBridge       HostBridgeStatus `json:"hostBridge"`
}

type HostBridgeStatus struct {
	Enabled bool   `json:"enabled"`
	Running bool   `json:"running"`
	Address string `json:"address"`
	Error   string `json:"error"`
}

type WebStatus struct {
	Enabled              bool     `json:"enabled"`
	Running              bool     `json:"running"`
	Host                 string   `json:"host"`
	Port                 string   `json:"port"`
	LocalURL             string   `json:"localUrl"`
	LanURLs              []string `json:"lanUrls"`
	PrimaryURL           string   `json:"primaryUrl"`
	UsingDefaultPassword bool     `json:"usingDefaultPassword"`
	Error                string   `json:"error"`
}

type WebSettingsRequest struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
	Port    string `json:"port"`
}

type WebTextFileRequest struct {
	Kind    string `json:"kind"`
	Content string `json:"content"`
	Confirm string `json:"confirm"`
}

type UpdateInfo struct {
	CurrentVersion string             `json:"currentVersion"`
	LatestVersion  string             `json:"latestVersion"`
	Available      bool               `json:"available"`
	Dismissed      bool               `json:"dismissed"`
	ReleaseURL     string             `json:"releaseUrl"`
	AssetURL       string             `json:"assetUrl"`
	AssetName      string             `json:"assetName"`
	Mirrors        []UpdateMirrorLink `json:"mirrors"`
	CheckedAt      string             `json:"checkedAt"`
}

type UpdateMirrorLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type DeleteProfileRequest struct {
	ID      string `json:"id"`
	Confirm string `json:"confirm"`
}

type DeleteSkillRequest struct {
	Path    string `json:"path"`
	Confirm bool   `json:"confirm"`
}

type RestoreDefaultRequest struct {
	Confirm bool `json:"confirm"`
}

type InstanceBackupManifest struct {
	Format              string                  `json:"format"`
	SchemaVersion       int                     `json:"schemaVersion"`
	AppVersion          string                  `json:"appVersion"`
	TemplateVersion     string                  `json:"templateVersion"`
	CreatedAt           string                  `json:"createdAt"`
	SourceInstanceRoot  string                  `json:"sourceInstanceRoot"`
	IncludesSecrets     bool                    `json:"includesSecrets"`
	IncludesWebSettings bool                    `json:"includesWebSettings"`
	Profiles            []InstanceBackupProfile `json:"profiles"`
	FileCount           int                     `json:"fileCount"`
	TotalBytes          int64                   `json:"totalBytes"`
	ExcludedPaths       []string                `json:"excludedPaths"`
	Path                string                  `json:"path,omitempty"`
}

type InstanceBackupProfile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	IsDefault bool   `json:"isDefault"`
}

type InstanceBackupImportRequest struct {
	Path    string `json:"path"`
	Confirm string `json:"confirm"`
}

type InstanceBackupImportResult struct {
	Manifest            InstanceBackupManifest `json:"manifest"`
	PreImportBackupPath string                 `json:"preImportBackupPath"`
}

type LauncherState struct {
	SchemaVersion             int               `json:"schemaVersion"`
	AppVersion                string            `json:"appVersion"`
	InstanceID                string            `json:"instanceId"`
	ManagedCompose            bool              `json:"managedCompose"`
	ComposeHash               string            `json:"composeHash"`
	TemplateVersion           string            `json:"templateVersion"`
	SkillsSnapshotImage       string            `json:"skillsSnapshotImage"`
	HermesImage               string            `json:"hermesImage"`
	ComposeSettings           ComposeSettings   `json:"composeSettings"`
	PreviousHermesImage       string            `json:"previousHermesImage"`
	LastSuccessfulHermesImage string            `json:"lastSuccessfulHermesImage"`
	InitializedAt             string            `json:"initializedAt"`
	UpdatedAt                 string            `json:"updatedAt"`
	Migrations                []MigrationRecord `json:"migrations"`
	Backups                   []BackupRecord    `json:"backups"`
	UI                        UIState           `json:"ui"`
	ModelAuxiliaryMode        string            `json:"modelAuxiliaryMode"`
}

type MigrationRecord struct {
	ID        string `json:"id"`
	AppliedAt string `json:"appliedAt"`
}

type BackupRecord struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
	Path   string `json:"path"`
}

type UIState struct {
	LastPage    string `json:"lastPage"`
	LastProfile string `json:"lastProfile"`
}

type ProfileRegistry struct {
	SchemaVersion int            `json:"schemaVersion"`
	Profiles      []ProfileEntry `json:"profiles"`
}

type ProfileEntry struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Enabled            bool   `json:"enabled"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
	ModelAuxiliaryMode string `json:"modelAuxiliaryMode"`
	SetupCompletedAt   string `json:"setupCompletedAt,omitempty"`
}

type CreateProfileRequest struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	CopyFrom string `json:"copyFrom"`
	CopyMode string `json:"copyMode"`
}

type RuntimeManifest struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Generation    string                   `json:"generation"`
	GeneratedAt   string                   `json:"generatedAt"`
	Profiles      []RuntimeManifestProfile `json:"profiles"`
}

type RuntimeManifestProfile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	Home      string `json:"home"`
	IsDefault bool   `json:"isDefault"`
	Runnable  bool   `json:"runnable"`
	Reason    string `json:"reason"`
}

type RuntimeStatus struct {
	Generation string                          `json:"generation"`
	UpdatedAt  string                          `json:"updatedAt"`
	Profiles   map[string]RuntimeProfileStatus `json:"profiles"`
}

type RuntimeProfileStatus struct {
	Enabled      bool   `json:"enabled"`
	State        string `json:"state"`
	PID          int    `json:"pid"`
	StartedAt    string `json:"startedAt"`
	LastExitCode int    `json:"lastExitCode"`
	RestartCount int    `json:"restartCount"`
	Message      string `json:"message"`
}

type ComposeSettings struct {
	Image                   string `json:"image"`
	ContainerName           string `json:"containerName"`
	GatewayHost             string `json:"gatewayHost"`
	GatewayPort             string `json:"gatewayPort"`
	DashboardHost           string `json:"dashboardHost"`
	DashboardPort           string `json:"dashboardPort"`
	DashboardEnabled        bool   `json:"dashboardEnabled"`
	DashboardUsername       string `json:"dashboardUsername"`
	DashboardPassword       string `json:"dashboardPassword"`
	GatewayBusyInputMode    string `json:"gatewayBusyInputMode"`
	GatewayBusyAckEnabled   string `json:"gatewayBusyAckEnabled"`
	BackgroundNotifications string `json:"backgroundNotifications"`
	HostControlEnabled      string `json:"hostControlEnabled"`
	MemoryLimit             string `json:"memoryLimit"`
	CPULimit                string `json:"cpuLimit"`
	ShmSize                 string `json:"shmSize"`
}

type ResourceLimitsRecommendation struct {
	MemoryLimit    string `json:"memoryLimit"`
	CPULimit       string `json:"cpuLimit"`
	DockerMemoryGB int    `json:"dockerMemoryGB"`
	DockerCPU      int    `json:"dockerCPU"`
}

type ProxySettings struct {
	Enabled    bool   `json:"enabled"`
	HTTPProxy  string `json:"httpProxy"`
	HTTPSProxy string `json:"httpsProxy"`
	ALLProxy   string `json:"allProxy"`
	NoProxy    string `json:"noProxy"`
}

type EnvVar struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

type ModelConfig struct {
	Provider      string                 `json:"provider"`
	Default       string                 `json:"default"`
	BaseURL       string                 `json:"baseUrl"`
	APIMode       string                 `json:"apiMode"`
	APIKey        string                 `json:"apiKey"`
	AuxiliaryMode string                 `json:"auxiliaryMode"`
	Auxiliary     map[string]AuxModel    `json:"auxiliary"`
	Fallbacks     []string               `json:"fallbacks"`
	RawProviders  map[string]interface{} `json:"rawProviders"`
}

type ProviderConfig struct {
	Providers map[string]ProviderConfigEntry `json:"providers"`
}

type ProviderConfigEntry struct {
	Label        string `json:"label" yaml:"label"`
	Provider     string `json:"provider" yaml:"provider"`
	BaseURL      string `json:"baseUrl" yaml:"base_url"`
	APIMode      string `json:"apiMode" yaml:"api_mode"`
	APIKey       string `json:"apiKey" yaml:"api_key"`
	ModelListURL string `json:"modelListUrl" yaml:"model_list_url"`
	DefaultModel string `json:"defaultModel" yaml:"default_model"`
	Builtin      bool   `json:"builtin" yaml:"builtin"`
	Disabled     bool   `json:"disabled" yaml:"disabled"`
}

type ModelProviderPreset struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	Provider     string `json:"provider"`
	BaseURL      string `json:"baseUrl"`
	APIMode      string `json:"apiMode"`
	DefaultModel string `json:"defaultModel"`
	ModelListURL string `json:"modelListUrl"`
}

type ModelListRequest struct {
	ProviderID  string `json:"providerId"`
	ProviderKey string `json:"providerKey"`
	APIKey      string `json:"apiKey"`
	BaseURL     string `json:"baseUrl"`
}

type ModelOption struct {
	ID      string `json:"id"`
	OwnedBy string `json:"ownedBy"`
}

type AuxModel struct {
	Provider  string                 `json:"provider"`
	Model     string                 `json:"model"`
	BaseURL   string                 `json:"baseUrl"`
	APIKey    string                 `json:"apiKey"`
	Timeout   int                    `json:"timeout"`
	ExtraBody map[string]interface{} `json:"extraBody"`
}

type ChannelFile struct {
	UpdatedAt string                      `json:"updated_at" yaml:"updated_at"`
	Platforms map[string][]ChannelSummary `json:"platforms" yaml:"platforms"`
}

type ChannelSummary struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type" yaml:"type"`
	ThreadID string `json:"thread_id" yaml:"thread_id"`
}

type TextFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Reason  string `json:"reason"`
}

type SkillsState struct {
	ActiveProfile string         `json:"activeProfile"`
	Skills        []SkillSummary `json:"skills"`
	Total         int            `json:"total"`
	BuiltinCount  int            `json:"builtinCount"`
	CustomCount   int            `json:"customCount"`
	ConflictCount int            `json:"conflictCount"`
}

type SyncBundledSkillsResult struct {
	ActiveProfile string   `json:"activeProfile"`
	SyncedSkills  []string `json:"syncedSkills"`
	SyncedFiles   int      `json:"syncedFiles"`
}

type SkillSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Platforms   []string `json:"platforms"`
	Tags        []string `json:"tags"`
	Path        string   `json:"path"`
	Category    string   `json:"category"`
	Builtin     bool     `json:"builtin"`
	Conflict    bool     `json:"conflict"`
	Error       string   `json:"error"`
	SizeBytes   int64    `json:"sizeBytes"`
	UpdatedAt   string   `json:"updatedAt"`
}

type SkillDetail struct {
	SkillSummary
	Preview          string          `json:"preview"`
	PreviewTruncated bool            `json:"previewTruncated"`
	Files            []SkillFileInfo `json:"files"`
	FileCount        int             `json:"fileCount"`
	FilesTruncated   bool            `json:"filesTruncated"`
	ConflictPaths    []string        `json:"conflictPaths"`
}

type SkillFileInfo struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	UpdatedAt string `json:"updatedAt"`
}

type SkillHubQuery struct {
	Keyword  string `json:"keyword"`
	Category string `json:"category"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	SortBy   string `json:"sortBy"`
	Order    string `json:"order"`
}

type SkillHubState struct {
	Skills     []SkillHubSkill    `json:"skills"`
	Categories []SkillHubCategory `json:"categories"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"pageSize"`
}

type SkillHubCategory struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type SkillHubSkill struct {
	Slug           string   `json:"slug"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Category       string   `json:"category"`
	CategoryName   string   `json:"categoryName"`
	Source         string   `json:"source"`
	Version        string   `json:"version"`
	Downloads      int      `json:"downloads"`
	Stars          int      `json:"stars"`
	Installs       int      `json:"installs"`
	RequiresAPIKey bool     `json:"requiresApiKey"`
	Verified       bool     `json:"verified"`
	Installed      bool     `json:"installed"`
	InstalledPath  string   `json:"installedPath"`
	Tags           []string `json:"tags"`
}

type SkillHubDetail struct {
	SkillHubSkill
	OwnerName       string             `json:"ownerName"`
	Homepage        string             `json:"homepage"`
	SecurityReports []SkillHubSecurity `json:"securityReports"`
	Files           []SkillHubFile     `json:"files"`
	FileCount       int                `json:"fileCount"`
	Signature       SkillHubSignature  `json:"signature"`
}

type SkillHubSecurity struct {
	Provider string `json:"provider"`
	Status   string `json:"status"`
	Text     string `json:"text"`
	URL      string `json:"url"`
}

type SkillHubFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type SkillHubSignature struct {
	Signed      bool   `json:"signed"`
	KeyID       string `json:"keyId"`
	ContentHash string `json:"contentHash"`
	PackageMD5  string `json:"packageMd5"`
	Payload     string `json:"payload"`
}

type WeComConfig struct {
	BotID           string `json:"botId"`
	Secret          string `json:"secret"`
	WebSocketURL    string `json:"websocketUrl"`
	DMPolicy        string `json:"dmPolicy"`
	AllowedUsers    string `json:"allowedUsers"`
	GroupPolicy     string `json:"groupPolicy"`
	GroupAllowUsers string `json:"groupAllowUsers"`
}

type FeishuConfig struct {
	AppID        string `json:"appId"`
	AppSecret    string `json:"appSecret"`
	Domain       string `json:"domain"`
	AllowedUsers string `json:"allowedUsers"`
	GroupPolicy  string `json:"groupPolicy"`
}
