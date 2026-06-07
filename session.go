package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SessionData 持久化的认证会话信息，用于意外退出后恢复心跳
type SessionData struct {
	ClientID  string `json:"client_id"`
	Hostname  string `json:"hostname"`
	Mac       string `json:"mac"`
	UserIP    string `json:"user_ip"`
	AcIP      string `json:"ac_ip"`
	AlgoID    string `json:"algo_id"`
	Ticket    string `json:"ticket"`
	KeepUrl   string `json:"keep_url"`
	TermUrl   string `json:"term_url"`
	IndexUrl  string `json:"index_url"`
	SavedAt   string `json:"saved_at"`
}

// sessionPath 返回会话文件路径
func sessionPath(sessionsDir, name string) string {
	return filepath.Join(sessionsDir, name+".json")
}

// SaveSession 保存会话信息到文件
func SaveSession(sessionsDir, name string, sd *SessionData) error {
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return fmt.Errorf("create sessions dir: %w", err)
	}
	sd.SavedAt = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	return os.WriteFile(sessionPath(sessionsDir, name), data, 0644)
}

// LoadSession 从文件加载会话信息
func LoadSession(sessionsDir, name string) (*SessionData, error) {
	data, err := os.ReadFile(sessionPath(sessionsDir, name))
	if err != nil {
		return nil, err
	}
	var sd SessionData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, err
	}
	if sd.ClientID == "" || sd.Ticket == "" || sd.KeepUrl == "" {
		return nil, fmt.Errorf("incomplete session data")
	}
	return &sd, nil
}

// DeleteSession 删除会话文件
func DeleteSession(sessionsDir, name string) {
	_ = os.Remove(sessionPath(sessionsDir, name))
}
