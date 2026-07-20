package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/GehirnInc/crypt/sha512_crypt"
	"gopkg.in/yaml.v3"
)

const (
	defaultDufsImage      = "sigoden/dufs:v0.46.0"
	defaultDufsPort       = "9878"
	defaultDufsUsername   = "qizhihe"
	defaultDufsPassword   = "123456"
	dufsContainerPort     = 5000
	defaultHostBridgePort = "9877"
)

var dufsUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type dufsConfig struct {
	ServePath    string   `yaml:"serve-path"`
	Bind         string   `yaml:"bind"`
	Port         int      `yaml:"port"`
	Hidden       []string `yaml:"hidden"`
	Auth         []string `yaml:"auth"`
	AllowUpload  bool     `yaml:"allow-upload"`
	AllowDelete  bool     `yaml:"allow-delete"`
	AllowSearch  bool     `yaml:"allow-search"`
	AllowSymlink bool     `yaml:"allow-symlink"`
	AllowArchive bool     `yaml:"allow-archive"`
	AllowHash    bool     `yaml:"allow-hash"`
	EnableCORS   bool     `yaml:"enable-cors"`
	LogFormat    string   `yaml:"log-format"`
}

func (a *App) validateDufsSettings(settings ComposeSettings) error {
	if err := validateDufsConfigSettings(settings); err != nil {
		return err
	}
	if !settings.DufsEnabled {
		return nil
	}
	conflicts := map[string]string{}
	if settings.HostControlEnabled != "false" {
		conflicts[defaultHostBridgePort] = "Host Bridge"
	}
	if cfg, err := a.readWebConfig(); err == nil {
		if cfg.Enabled {
			conflicts[cfg.Port] = "Web 管理"
		}
	} else {
		conflicts[defaultWebPort] = "Web 管理"
	}
	if name := conflicts[settings.DufsPort]; name != "" {
		return fmt.Errorf("Dufs 端口与%s端口冲突", name)
	}
	return nil
}

func validateDufsConfigSettings(settings ComposeSettings) error {
	port, err := strconv.Atoi(strings.TrimSpace(settings.DufsPort))
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("Dufs 端口必须是 1-65535 的数字")
	}
	if !dufsUsernamePattern.MatchString(settings.DufsUsername) {
		return fmt.Errorf("Dufs 用户名只能包含字母、数字、点、下划线和连字符")
	}
	return nil
}

func (a *App) ensureDufsConfig(settings ComposeSettings, password, reason string) (ComposeSettings, error) {
	settings = withComposeDefaults(settings)
	if err := validateDufsConfigSettings(settings); err != nil {
		return settings, err
	}

	passwordHash := ""
	if password != "" {
		var err error
		passwordHash, err = sha512_crypt.New().Generate([]byte(password), nil)
		if err != nil {
			return settings, fmt.Errorf("生成 Dufs 密码哈希失败：%w", err)
		}
		settings.DufsUsingDefaultPassword = password == defaultDufsPassword
	} else if fileExists(a.dufsConfigPath()) {
		var err error
		passwordHash, err = readDufsPasswordHash(a.dufsConfigPath())
		if err != nil {
			return settings, err
		}
		settings.DufsUsingDefaultPassword = sha512_crypt.New().Verify(passwordHash, []byte(defaultDufsPassword)) == nil
	} else {
		var err error
		passwordHash, err = sha512_crypt.New().Generate([]byte(defaultDufsPassword), nil)
		if err != nil {
			return settings, fmt.Errorf("生成 Dufs 默认密码哈希失败：%w", err)
		}
		settings.DufsUsingDefaultPassword = true
	}

	cfg := dufsConfig{
		ServePath: "/data",
		Bind:      "0.0.0.0",
		Port:      dufsContainerPort,
		Hidden: []string{
			".DS_Store",
			"._*",
			"Thumbs.db",
			"desktop.ini",
			".Spotlight-V100",
			".Trashes",
		},
		Auth:         []string{settings.DufsUsername + ":" + passwordHash + "@/:rw"},
		AllowUpload:  true,
		AllowDelete:  true,
		AllowSearch:  true,
		AllowSymlink: false,
		AllowArchive: true,
		AllowHash:    false,
		EnableCORS:   false,
		LogFormat:    `$remote_addr $remote_user "$request" $status`,
	}
	content, err := yaml.Marshal(cfg)
	if err != nil {
		return settings, fmt.Errorf("生成 Dufs 配置失败：%w", err)
	}
	current, err := os.ReadFile(a.dufsConfigPath())
	if err == nil && string(current) == string(content) {
		settings.DufsPassword = ""
		return settings, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return settings, fmt.Errorf("读取 Dufs 配置失败：%w", err)
	}
	if err == nil {
		if err := a.backupFile(a.dufsConfigPath(), reason); err != nil {
			return settings, err
		}
	}
	if err := ensureDir(filepath.Dir(a.dufsConfigPath())); err != nil {
		return settings, err
	}
	if err := atomicWriteFile(a.dufsConfigPath(), content, 0600); err != nil {
		return settings, fmt.Errorf("写入 Dufs 配置失败：%w", err)
	}
	settings.DufsPassword = ""
	return settings, nil
}

func readDufsPasswordHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取 Dufs 配置失败：%w", err)
	}
	var cfg dufsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("Dufs 配置无效：%w", err)
	}
	if len(cfg.Auth) != 1 {
		return "", fmt.Errorf("Dufs 配置必须包含一个共享账号")
	}
	_, credential, ok := strings.Cut(cfg.Auth[0], ":")
	if !ok {
		return "", fmt.Errorf("Dufs 账号配置无效")
	}
	passwordHash, _, ok := strings.Cut(credential, "@")
	if !ok || !strings.HasPrefix(passwordHash, "$6$") {
		return "", fmt.Errorf("Dufs 密码必须使用 SHA-512 crypt 哈希")
	}
	return passwordHash, nil
}

func (a *App) dufsStatus() DufsStatus {
	settings := a.readComposeSettings()
	localURL := "http://127.0.0.1:" + settings.DufsPort
	lanURLs := lanWebURLs(settings.DufsPort)
	primaryURL := localURL
	if len(lanURLs) > 0 {
		primaryURL = lanURLs[0]
	}
	return DufsStatus{
		Enabled:              settings.DufsEnabled,
		Port:                 settings.DufsPort,
		Username:             settings.DufsUsername,
		LocalURL:             localURL,
		LanURLs:              lanURLs,
		PrimaryURL:           primaryURL,
		UsingDefaultPassword: settings.DufsUsingDefaultPassword,
	}
}

func (a *App) dufsRuntimeHash() (string, error) {
	hash := sha256.New()
	settings := a.readComposeSettings()
	service := []byte(renderDufsService(settings))
	_, _ = fmt.Fprintf(hash, "%d:", len(service))
	_, _ = hash.Write(service)
	for _, path := range []string{a.dufsConfigPath(), a.overridePath()} {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("读取 Dufs 运行配置失败：%w", err)
		}
		_, _ = fmt.Fprintf(hash, "%d:", len(data))
		_, _ = hash.Write(data)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
