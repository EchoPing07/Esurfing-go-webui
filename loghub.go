package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string `json:"ts"`
	Level     string `json:"lv"`
	Message   string `json:"m"`
}

// 日志等级定义
const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
	LevelOK    = "ok"
)

// levelPriority 返回日志等级的优先级数值，数值越大越严重
func levelPriority(level string) int {
	switch level {
	case LevelDebug:
		return 0
	case LevelInfo, LevelOK:
		return 1
	case LevelWarn:
		return 2
	case LevelError:
		return 3
	default:
		return 1
	}
}

// LogHub 日志中心，支持订阅/发布模式和文件持久化
type LogHub struct {
	mu          sync.RWMutex
	fileMu      sync.Mutex
	closed      bool // Close() 已执行，禁止文件写入
	entries     []LogEntry
	subs        map[chan LogEntry]struct{}
	maxSize     int
	logDir      string
	logFile     *os.File
	logSize     int64
	maxFileSize int64
	minLevel    int
}

// NewLogHub 创建日志中心，logDir 为空时禁用文件持久化
func NewLogHub(maxSize int, logDir string) *LogHub {
	h := &LogHub{
		entries:     make([]LogEntry, 0, maxSize),
		subs:        make(map[chan LogEntry]struct{}),
		maxSize:     maxSize,
		logDir:      logDir,
		maxFileSize: 1 * 1024 * 1024, // 1MB
	}
	if logDir != "" {
		h.initLogFile()
	}
	return h
}

// initLogFile 初始化日志文件：创建目录、清理旧日志、打开新文件
func (h *LogHub) initLogFile() {
	if err := os.MkdirAll(h.logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[loghub] failed to create log dir: %v\n", err)
		return
	}
	h.cleanupOldLogs()
	h.openNewFile()
}

// openNewFile 以当前时间命名打开新日志文件，成功返回 true
func (h *LogHub) openNewFile() bool {
	name := time.Now().Format("2006-01-02_15-04-05.000") + ".json"
	path := filepath.Join(h.logDir, name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[loghub] failed to open log file: %v\n", err)
		return false
	}
	h.logFile = f
	info, _ := f.Stat()
	if info != nil {
		h.logSize = info.Size()
	}
	return true
}

// cleanupOldLogs 只保留最近 3 个日志文件
func (h *LogHub) cleanupOldLogs() {
	entries, err := os.ReadDir(h.logDir)
	if err != nil {
		return
	}
	var files []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			files = append(files, e)
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() > files[j].Name()
	})
	if len(files) > 3 {
		for _, f := range files[3:] {
			if err := os.Remove(filepath.Join(h.logDir, f.Name())); err != nil {
				fmt.Fprintf(os.Stderr, "[loghub] failed to remove old log %s: %v\n", f.Name(), err)
			}
		}
	}
}

// Add 添加日志并通知所有订阅者，同时持久化到文件
func (h *LogHub) Add(level, msg string) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
	}

	h.mu.Lock()
	if levelPriority(level) < h.minLevel {
		h.mu.Unlock()
		return
	}
	h.entries = append(h.entries, entry)
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[len(h.entries)-h.maxSize:]
	}
	for ch := range h.subs {
		select {
		case ch <- entry:
		default:
		}
	}
	h.mu.Unlock()

	h.fileMu.Lock()
	h.writeToFile(entry)
	h.fileMu.Unlock()
}

// SetLevel 设置最低日志等级
func (h *LogHub) SetLevel(level string) {
	h.mu.Lock()
	h.minLevel = levelPriority(level)
	h.mu.Unlock()
}

// writeToFile 将日志条目写入文件，超过大小限制时轮转
func (h *LogHub) writeToFile(entry LogEntry) {
	if h.closed || h.logFile == nil {
		return
	}
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[loghub] failed to marshal log: %v\n", err)
		return
	}
	data = append(data, '\n')
	n, err := h.logFile.Write(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[loghub] failed to write log, reopening: %v\n", err)
		oldFile := h.logFile
		h.logFile = nil
		if h.openNewFile() {
			if oldFile != nil {
				oldFile.Close()
			}
			n, err = h.logFile.Write(data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[loghub] retry write also failed: %v\n", err)
				return
			}
		} else {
			h.logFile = oldFile
			return
		}
	}
	h.logSize += int64(n)
	if h.logSize >= h.maxFileSize {
		oldFile := h.logFile
		h.logFile = nil
		h.cleanupOldLogs()
		if !h.openNewFile() {
			// 轮转失败，继续使用旧文件
			h.logFile = oldFile
		} else if oldFile != nil {
			oldFile.Close()
		}
	}
}

// Close 关闭日志文件并清理所有订阅者
func (h *LogHub) Close() {
	h.mu.Lock()
	for ch := range h.subs {
		close(ch)
	}
	h.subs = make(map[chan LogEntry]struct{})
	h.mu.Unlock()

	h.fileMu.Lock()
	defer h.fileMu.Unlock()
	h.closed = true
	if h.logFile != nil {
		_ = h.logFile.Sync()
		h.logFile.Close()
		h.logFile = nil
	}
}

// GetAll 获取全部日志
func (h *LogHub) GetAll() []LogEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]LogEntry, len(h.entries))
	copy(result, h.entries)
	return result
}

// Subscribe 订阅日志流
func (h *LogHub) Subscribe() chan LogEntry {
	ch := make(chan LogEntry, 64)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe 取消订阅
func (h *LogHub) Unsubscribe(ch chan LogEntry) {
	h.mu.Lock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
	h.mu.Unlock()
}

// FormatSSE 将日志条目格式化为 SSE 消息
func FormatSSE(entry LogEntry) []byte {
	data, _ := json.Marshal(entry)
	return append(append([]byte("data: "), data...), '\n', '\n')
}
