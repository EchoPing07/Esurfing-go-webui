// esurfing-go-webui 程序入口，启动 Web 服务和客户端管理器
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

//go:embed web
var webFS embed.FS

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	var (
		configPath = flag.String("c", "esurfing_data/config.json", "config file path")
		port       = flag.Int("p", 0, "web server port (overrides config)")
	)
	flag.Parse()

	log.SetFlags(log.LstdFlags)
	log.Println("esurfing-go-webui starting")

	logHub := NewLogHub(1000)
	manager := NewManager(*configPath, logHub)

	if err := manager.Load(); err != nil {
		log.Printf("warning: failed to load config: %v", err)
	}

	settings := manager.GetSettings()
	webPort := settings.WebPort
	if *port > 0 {
		webPort = *port
	}
	if webPort == 0 {
		webPort = 8080
	}

	bindAddr := "0.0.0.0"
	switch settings.AccessMode {
	case "localhost":
		bindAddr = "127.0.0.1"
	case "lan":
		bindAddr = "0.0.0.0"
	default:
		bindAddr = "0.0.0.0"
	}

	api := NewAPI(manager, logHub)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	webSub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("failed to access embedded web files: %v", err)
	}

	fileServer := http.FileServer(http.FS(webSub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/favicon.ico" && r.URL.Path != "/icon.svg" {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", bindAddr, webPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("web server listening on %s:%d", bindAddr, webPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	if dataDir := filepath.Dir(*configPath); dataDir != "." {
		_ = os.MkdirAll(dataDir, 0755)
	}

	manager.StartEnabled()
	log.Println("enabled clients started")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-sigCh
	log.Printf("received signal %v, shutting down", sig)

	manager.StopAll()
	log.Println("all clients stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	log.Println("esurfing-go-webui stopped")
}
