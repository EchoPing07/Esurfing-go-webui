package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// API Web API 服务，提供接口管理和日志查询
type API struct {
	manager *Manager
	logHub  *LogHub
}

// NewAPI 创建 API 实例
func NewAPI(manager *Manager, logHub *LogHub) *API {
	return &API{manager: manager, logHub: logHub}
}

// RegisterRoutes 注册 API 路由
func (a *API) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/interfaces", a.handleInterfaces)
	mux.HandleFunc("/api/interfaces/", a.handleInterfaceAction)
	mux.HandleFunc("/api/settings", a.handleSettings)
	mux.HandleFunc("/api/logs", a.handleLogs)
	mux.HandleFunc("/api/logs/stream", a.handleLogStream)
	mux.HandleFunc("/api/system/interfaces", a.handleSystemInterfaces)
}

// handleInterfaces 处理 /api/interfaces，GET 查询全部/POST 新增
func (a *API) handleInterfaces(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.writeJSON(w, http.StatusOK, a.manager.GetAll())

	case http.MethodPost:
		var mc ManagedClient
		if err := json.NewDecoder(r.Body).Decode(&mc); err != nil {
			a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if mc.Name == "" {
			mc.Name = mc.BindInterface
		}
		if mc.Name == "" {
			a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "interface name is required"})
			return
		}
		if mc.BindInterface == "" {
			mc.BindInterface = mc.Name
		}
		if err := a.manager.Add(&mc); err != nil {
			a.writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		a.logHub.Add("info", fmt.Sprintf("[%s] added", mc.Name))
		a.writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleInterfaceAction 处理 /api/interfaces/{name}/{action}，支持更新/删除/启用/禁用/登录/注销
func (a *API) handleInterfaceAction(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/interfaces/")
	parts := strings.SplitN(path, "/", 2)
	name := parts[0]
	if name == "" {
		a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "interface name is required"})
		return
	}

	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	if action == "" {
		switch r.Method {
		case http.MethodPut:
			var mc ManagedClient
			if err := json.NewDecoder(r.Body).Decode(&mc); err != nil {
				a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			if err := a.manager.Update(name, &mc); err != nil {
				a.writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			a.logHub.Add("info", fmt.Sprintf("[%s] updated", name))
			a.writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})

		case http.MethodDelete:
			if err := a.manager.Delete(name); err != nil {
				a.writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			a.logHub.Add("info", fmt.Sprintf("[%s] deleted", name))
			a.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch action {
	case "enable":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := a.manager.Enable(name); err != nil {
			a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		a.writeJSON(w, http.StatusOK, map[string]string{"status": "enabled"})

	case "disable":
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := a.manager.Disable(name); err != nil {
			a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		a.writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})

	case "login":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := a.manager.Login(name); err != nil {
			a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		a.writeJSON(w, http.StatusOK, map[string]string{"status": "login started"})

	case "logout":
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := a.manager.Logout(name); err != nil {
			a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		a.writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// handleSettings 处理 /api/settings，GET 查询/PUT 更新全局设置
func (a *API) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.writeJSON(w, http.StatusOK, a.manager.GetSettings())

	case http.MethodPut:
		var s Settings
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := a.manager.UpdateSettings(s); err != nil {
			a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		a.writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleLogs 查询全部日志
func (a *API) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	a.writeJSON(w, http.StatusOK, a.logHub.GetAll())
}

// handleLogStream SSE 实时日志推送
func (a *API) handleLogStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := a.logHub.Subscribe()
	defer a.logHub.Unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			_, _ = w.Write(FormatSSE(entry))
			flusher.Flush()
		}
	}
}

func (a *API) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

// handleSystemInterfaces 查询系统网络接口列表（排除回环，标记已使用）
func (a *API) handleSystemInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	managed := a.manager.GetAll()
	existing := make(map[string]bool)
	for _, m := range managed {
		existing[m.Name] = true
	}

	ifaces, err := ListInterfaces()
	if err != nil {
		a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	type SysIface struct {
		Name       string `json:"name"`
		IP         string `json:"ip"`
		IsUp       bool   `json:"is_up"`
		IsLoopback bool   `json:"is_loopback"`
		Hardware   string `json:"hardware"`
		Used       bool   `json:"used"`
	}

	var result []SysIface
	for _, iface := range ifaces {
		if iface.IsLoopback {
			continue
		}
		result = append(result, SysIface{
			Name:       iface.Name,
			IP:         iface.IP,
			IsUp:       iface.IsUp,
			IsLoopback: iface.IsLoopback,
			Hardware:   iface.Hardware,
			Used:       existing[iface.Name],
		})
	}

	a.writeJSON(w, http.StatusOK, result)
}
