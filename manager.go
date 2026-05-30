package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ManagedClient 受管理的客户端实例，包含配置和运行状态
type ManagedClient struct {
	Name          string `json:"interface"`
	Enabled       bool   `json:"enabled"`
	Status        string `json:"status"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	CheckInterval int    `json:"check_interval"`
	RetryInterval int    `json:"retry_interval"`
	DnsAddress    string `json:"dns_address"`
	BindInterface string `json:"bind_interface"`
	MaxRetries    int    `json:"max_retries"`
	UserIP        string `json:"user_ip"`
	LastHeartbeat string `json:"last_heartbeat"`
	NextHeartbeat string `json:"next_heartbeat"`
	LastLogin     string `json:"last_login"`
	ErrorCount    int    `json:"error_count"`

	client *Client
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// ManagerConfig 持久化配置结构
type ManagerConfig struct {
	Interfaces []ManagedClient `json:"interfaces"`
	Settings   Settings        `json:"settings"`
}

// Settings 全局设置
type Settings struct {
	WebPort           int    `json:"web_port"`
	DefaultMaxRetries int    `json:"default_max_retries"`
	AccessMode        string `json:"access_mode"`
}

// Manager 客户端管理器，负责配置持久化和多客户端生命周期管理
type Manager struct {
	mu       sync.RWMutex
	clients  map[string]*ManagedClient
	settings Settings
	configPath string
	logHub   *LogHub
}

// NewManager 创建管理器实例
func NewManager(configPath string, logHub *LogHub) *Manager {
	return &Manager{
		clients:    make(map[string]*ManagedClient),
		settings:   Settings{WebPort: 8080, DefaultMaxRetries: 5},
		configPath: configPath,
		logHub:     logHub,
	}
}

// Load 从配置文件加载客户端列表和全局设置
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var mc ManagerConfig
	if err := json.Unmarshal(data, &mc); err != nil {
		return err
	}

	m.settings = mc.Settings
	if m.settings.WebPort == 0 {
		m.settings.WebPort = 8080
	}

	for i := range mc.Interfaces {
		iface := &mc.Interfaces[i]
		iface.Status = "disabled"
		iface.UserIP = ""
		iface.LastHeartbeat = ""
		iface.LastLogin = ""
		iface.ErrorCount = 0
		if iface.BindInterface == "" {
			iface.BindInterface = iface.Name
		}
		m.clients[iface.Name] = iface
	}

	return nil
}

// Save 保存当前配置到文件
func (m *Manager) Save() error {
	m.mu.RLock()
	mc := ManagerConfig{
		Settings: m.settings,
	}
	for _, c := range m.clients {
		c.mu.RLock()
		mc.Interfaces = append(mc.Interfaces, ManagedClient{
			Name:          c.Name,
			Enabled:       c.Enabled,
			Username:      c.Username,
			Password:      c.Password,
			CheckInterval: c.CheckInterval,
			RetryInterval: c.RetryInterval,
			DnsAddress:    c.DnsAddress,
			BindInterface: c.BindInterface,
			MaxRetries:    c.MaxRetries,
		})
		c.mu.RUnlock()
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(mc, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0644)
}

// GetAll 获取全部客户端列表
func (m *Manager) GetAll() []ManagedClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]ManagedClient, 0, len(m.clients))
	for _, c := range m.clients {
		c.mu.RLock()
		mc := ManagedClient{
			Name:          c.Name,
			Enabled:       c.Enabled,
			Status:        c.Status,
			Username:      c.Username,
			Password:      c.Password,
			CheckInterval: c.CheckInterval,
			RetryInterval: c.RetryInterval,
			DnsAddress:    c.DnsAddress,
			BindInterface: c.BindInterface,
			MaxRetries:    c.MaxRetries,
			UserIP:        c.UserIP,
			LastHeartbeat: c.LastHeartbeat,
			NextHeartbeat: c.NextHeartbeat,
			LastLogin:     c.LastLogin,
			ErrorCount:    c.ErrorCount,
		}
		c.mu.RUnlock()
		result = append(result, mc)
	}
	return result
}

// Get 根据名称获取客户端
func (m *Manager) Get(name string) *ManagedClient {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clients[name]
}

// Add 添加新客户端配置
func (m *Manager) Add(mc *ManagedClient) error {
	m.mu.Lock()
	if _, exists := m.clients[mc.Name]; exists {
		m.mu.Unlock()
		return fmt.Errorf("interface %s already exists", mc.Name)
	}

	mc.Status = "disabled"
	if mc.BindInterface == "" {
		mc.BindInterface = mc.Name
	}
	m.clients[mc.Name] = mc
	m.mu.Unlock()

	return m.Save()
}

// Update 更新客户端配置，运行中的客户端会自动重启
func (m *Manager) Update(name string, mc *ManagedClient) error {
	m.mu.Lock()
	existing, exists := m.clients[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("interface %s not found", name)
	}

	existing.mu.Lock()
	wasRunning := existing.client != nil
	if wasRunning {
		existing.cancel()
		existing.client = nil
	}

	existing.Username = mc.Username
	existing.Password = mc.Password
	existing.CheckInterval = mc.CheckInterval
	existing.RetryInterval = mc.RetryInterval
	existing.DnsAddress = mc.DnsAddress
	existing.MaxRetries = mc.MaxRetries
	existing.BindInterface = mc.BindInterface
	existing.mu.Unlock()
	m.mu.Unlock()

	if err := m.Save(); err != nil {
		return err
	}

	if wasRunning {
		existing.mu.RLock()
		en := existing.Enabled
		existing.mu.RUnlock()
		if en {
			return m.startClient(existing)
		}
	}
	return nil
}

// Delete 删除客户端配置并停止运行
func (m *Manager) Delete(name string) error {
	m.mu.Lock()
	mc, exists := m.clients[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("interface %s not found", name)
	}

	mc.mu.Lock()
	if mc.client != nil {
		mc.cancel()
		mc.client = nil
	}
	mc.mu.Unlock()

	delete(m.clients, name)
	m.mu.Unlock()

	return m.Save()
}

// Enable 启用客户端并开始认证
func (m *Manager) Enable(name string) error {
	m.mu.RLock()
	mc, exists := m.clients[name]
	m.mu.RUnlock()
	if !exists {
		return fmt.Errorf("interface %s not found", name)
	}

	mc.mu.Lock()
	if mc.Username == "" || mc.Password == "" {
		mc.mu.Unlock()
		return fmt.Errorf("username or password is empty")
	}
	mc.Enabled = true
	mc.Status = "offline"
	mc.ErrorCount = 0
	mc.mu.Unlock()

	if err := m.Save(); err != nil {
		return err
	}

	m.logHub.Add("info", fmt.Sprintf("[%s] enabled", name))
	return m.startClient(mc)
}

// Disable 禁用客户端并停止运行
func (m *Manager) Disable(name string) error {
	m.mu.RLock()
	mc, exists := m.clients[name]
	m.mu.RUnlock()
	if !exists {
		return fmt.Errorf("interface %s not found", name)
	}

	mc.mu.Lock()
	mc.Enabled = false
	mc.Status = "disabled"
	mc.UserIP = ""
	mc.LastHeartbeat = ""
	mc.NextHeartbeat = ""
	mc.ErrorCount = 0
	if mc.client != nil {
		mc.cancel()
		mc.client = nil
	}
	mc.mu.Unlock()

	if err := m.Save(); err != nil {
		return err
	}

	m.logHub.Add("info", fmt.Sprintf("[%s] disabled", name))
	return nil
}

// Login 手动触发客户端登录
func (m *Manager) Login(name string) error {
	m.mu.RLock()
	mc, exists := m.clients[name]
	m.mu.RUnlock()
	if !exists {
		return fmt.Errorf("interface %s not found", name)
	}

	mc.mu.RLock()
	if !mc.Enabled {
		mc.mu.RUnlock()
		return fmt.Errorf("interface %s is not enabled", name)
	}
	mc.mu.RUnlock()

	return m.startClient(mc)
}

// Logout 注销指定客户端
func (m *Manager) Logout(name string) error {
	m.mu.RLock()
	mc, exists := m.clients[name]
	m.mu.RUnlock()
	if !exists {
		return fmt.Errorf("interface %s not found", name)
	}

	mc.mu.Lock()
	if mc.client != nil {
		mc.cancel()
		mc.client = nil
	}
	mc.Status = "offline"
	mc.UserIP = ""
	mc.LastHeartbeat = ""
	mc.mu.Unlock()

	m.logHub.Add("info", fmt.Sprintf("[%s] disconnected", name))
	return nil
}

// startClient 启动客户端协程，注册状态变更、认证成功、心跳回调
func (m *Manager) startClient(mc *ManagedClient) error {
	mc.mu.Lock()
	if mc.client != nil {
		mc.cancel()
		mc.client = nil
	}

	cfg := &Config{
		Username:      mc.Username,
		Password:      mc.Password,
		CheckInterval: mc.CheckInterval,
		RetryInterval: mc.RetryInterval,
		BindInterface: mc.BindInterface,
		DnsAddress:    mc.DnsAddress,
	}

	client, err := NewClient(cfg)
	if err != nil {
		mc.mu.Unlock()
		return err
	}

	name := mc.Name
	mc.client = client
	mc.ctx, mc.cancel = context.WithCancel(context.Background())

	var lastStatus string

	client.OnStatusChange = func(status string) {
		mc.mu.Lock()
		mc.Status = status
		if status == "offline" {
			mc.UserIP = ""
			mc.LastHeartbeat = ""
		}
		mr := mc.MaxRetries
		mc.mu.Unlock()

		if status != lastStatus {
			m.logHub.Add("info", fmt.Sprintf("[%s] status: %s", name, status))
			lastStatus = status
		}

		if status == "offline" && mc.Enabled {
			mc.mu.Lock()
			mc.ErrorCount++
			ec := mc.ErrorCount
			mc.mu.Unlock()

			if mr > 0 && ec >= mr {
				mc.mu.Lock()
				mc.Enabled = false
				mc.Status = "disabled"
				mc.mu.Unlock()
				m.logHub.Add("er", fmt.Sprintf("[%s] max retries %d reached, auto disabled", name, mr))
			}
		}
	}

	client.OnAuthSuccess = func(userIP string) {
		mc.mu.Lock()
		mc.UserIP = userIP
		mc.LastLogin = time.Now().Format(time.RFC3339)
		mc.ErrorCount = 0
		mc.NextHeartbeat = ""
		mc.mu.Unlock()

		m.logHub.Add("ok", fmt.Sprintf("[%s] auth finished, ip: %s", name, userIP))
	}

	client.OnHeartbeat = func(interval int) {
		mc.mu.Lock()
		mc.LastHeartbeat = time.Now().Format(time.RFC3339)
		mc.ErrorCount = 0
		if interval > 0 {
			mc.NextHeartbeat = time.Now().Add(time.Duration(interval) * time.Second).Format(time.RFC3339)
		}
		mc.mu.Unlock()
	}

	mc.Status = "offline"
	mc.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.logHub.Add("er", fmt.Sprintf("[%s] panic: %v", name, r))
				mc.mu.Lock()
				mc.Status = "offline"
				mc.client = nil
				mc.mu.Unlock()
			}
		}()
		client.Start()
	}()

	return nil
}

// GetSettings 获取全局设置
func (m *Manager) GetSettings() Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.settings
}

// UpdateSettings 更新全局设置
func (m *Manager) UpdateSettings(s Settings) error {
	m.mu.Lock()
	m.settings = s
	m.mu.Unlock()
	return m.Save()
}

// StartEnabled 启动所有已启用的客户端
func (m *Manager) StartEnabled() {
	m.mu.RLock()
	clients := make([]*ManagedClient, 0)
	for _, mc := range m.clients {
		if mc.Enabled {
			clients = append(clients, mc)
		}
	}
	m.mu.RUnlock()

	for _, mc := range clients {
		if err := m.startClient(mc); err != nil {
			m.logHub.Add("er", fmt.Sprintf("[%s] auto-start failed: %v", mc.Name, err))
		} else {
			m.logHub.Add("info", fmt.Sprintf("[%s] auto-started", mc.Name))
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// StopAll 停止所有运行中的客户端
func (m *Manager) StopAll() {
	m.mu.RLock()
	clients := make([]*ManagedClient, 0, len(m.clients))
	for _, mc := range m.clients {
		clients = append(clients, mc)
	}
	m.mu.RUnlock()

	for _, mc := range clients {
		mc.mu.Lock()
		if mc.client != nil {
			mc.client.Cancel()
			mc.client = nil
			mc.Status = "offline"
			mc.UserIP = ""
			mc.LastHeartbeat = ""
		}
		mc.mu.Unlock()
	}

	time.Sleep(200 * time.Millisecond)
}
