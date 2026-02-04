package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aretw0/lifecycle"
)

// Config represents the application configuration.
type Config struct {
	AppName string `json:"app_name"`
	Debug   bool   `json:"debug"`
	Port    int    `json:"port"`
	Message string `json:"message"`
}

var (
	config     Config
	configMu   sync.RWMutex
	configPath string // Detected path to config.json
)

func loadConfig() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var newCfg Config
	if err := json.Unmarshal(data, &newCfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	configMu.Lock()
	config = newCfg
	configMu.Unlock()

	fmt.Printf("\n✅ Configuration loaded: %+v\n", newCfg)
	return nil
}

func main() {
	fmt.Println("=== Hot Reload Example (Cross-Platform) ===")
	fmt.Println()

	// Detect if running from root or examples directory
	configPath = "config.json"
	if _, err := os.Stat("examples/reload"); err == nil {
		configPath = "examples/reload/config.json"
	}

	// 1. Load initial config
	if err := loadConfig(); err != nil {
		fmt.Printf("❌ Failed to load initial config: %v\n", err)
		fmt.Println("   Creating default config.json...")
		createDefaultConfig()
		if err := loadConfig(); err != nil {
			fmt.Printf("❌ Still failed: %v\n", err)
			os.Exit(1)
		}
	}

	// 2. Setup file watcher (cross-platform!)
	router := lifecycle.NewRouter()
	router.AddSource(lifecycle.NewFileWatchSource(configPath))

	// 3. Handle reload events
	router.Handle("file/*", lifecycle.NewReloadHandler(func(ctx context.Context) error {
		fmt.Println("\n🔄 Config file changed!")
		time.Sleep(100 * time.Millisecond) // Small delay to ensure file write completed
		return loadConfig()
	}))

	// 4. Add periodic status ticker
	router.AddSource(lifecycle.NewTickerSource(3 * time.Second))
	router.HandleFunc("source/tick", func(ctx context.Context, _ lifecycle.Event) error {
		configMu.RLock()
		fmt.Printf("📊 %s running on port %d (debug=%v): %s\n",
			config.AppName, config.Port, config.Debug, config.Message)
		configMu.RUnlock()
		return nil
	})

	fmt.Println()
	fmt.Println("✨ App started with hot reload enabled!")
	fmt.Println("   Edit config.json to see configuration reload in real-time")
	fmt.Println("   Press Ctrl+C to exit")
	fmt.Println()

	lifecycle.Run(router)
	fmt.Println("\n👋 Shutdown complete")
}

func createDefaultConfig() {
	defaultCfg := Config{
		AppName: "LifecycleDemo",
		Debug:   true,
		Port:    8080,
		Message: "Hello from lifecycle hot reload!",
	}

	data, _ := json.MarshalIndent(defaultCfg, "", "  ")
	os.WriteFile(configPath, data, 0644)
}
