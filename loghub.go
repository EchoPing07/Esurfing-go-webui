package main

import (
	"encoding/json"
	"sync"
	"time"
)

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string `json:"ts"`
	Level     string `json:"lv"`
	Message   string `json:"m"`
}

// LogHub 日志中心，支持订阅/发布模式
type LogHub struct {
	mu      sync.RWMutex
	entries []LogEntry
	subs    map[chan LogEntry]struct{}
	maxSize int
}

// NewLogHub 创建日志中心
func NewLogHub(maxSize int) *LogHub {
	return &LogHub{
		entries: make([]LogEntry, 0, maxSize),
		subs:    make(map[chan LogEntry]struct{}),
		maxSize: maxSize,
	}
}

// Add 添加日志并通知所有订阅者
func (h *LogHub) Add(level, msg string) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
	}

	h.mu.Lock()
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
	delete(h.subs, ch)
	h.mu.Unlock()
	close(ch)
}

// Clear 清空日志
func (h *LogHub) Clear() {
	h.mu.Lock()
	h.entries = h.entries[:0]
	h.mu.Unlock()
}

// FormatSSE 将日志条目格式化为 SSE 消息
func FormatSSE(entry LogEntry) []byte {
	data, _ := json.Marshal(entry)
	return append(append([]byte("data: "), data...), '\n', '\n')
}
