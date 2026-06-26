package main

type AppState struct {
	AppVersion       string          `json:"appVersion"`
	InstanceRoot     string          `json:"instanceRoot"`
	State            LauncherState   `json:"state"`
	Compose          ComposeSettings `json:"compose"`
	Environment      []EnvVar        `json:"environment"`
	Model            ModelConfig     `json:"model"`
	Channels         ChannelFile     `json:"channels"`
	DockerAvailable  bool            `json:"dockerAvailable"`
	ComposeAvailable bool            `json:"composeAvailable"`
	ContainerStatus  string          `json:"containerStatus"`
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
	LastPage string `json:"lastPage"`
}

type ComposeSettings struct {
	Image             string `json:"image"`
	ContainerName     string `json:"containerName"`
	GatewayHost       string `json:"gatewayHost"`
	GatewayPort       string `json:"gatewayPort"`
	DashboardHost     string `json:"dashboardHost"`
	DashboardPort     string `json:"dashboardPort"`
	DashboardEnabled  bool   `json:"dashboardEnabled"`
	DashboardUsername string `json:"dashboardUsername"`
	DashboardPassword string `json:"dashboardPassword"`
	MemoryLimit       string `json:"memoryLimit"`
	CPULimit          string `json:"cpuLimit"`
	ShmSize           string `json:"shmSize"`
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

type WeComConfig struct {
	BotID           string `json:"botId"`
	Secret          string `json:"secret"`
	WebSocketURL    string `json:"websocketUrl"`
	DMPolicy        string `json:"dmPolicy"`
	AllowedUsers    string `json:"allowedUsers"`
	GroupPolicy     string `json:"groupPolicy"`
	GroupAllowUsers string `json:"groupAllowUsers"`
}
