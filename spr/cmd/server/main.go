package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"

	"github.com/acheong08/hackeurope-spr/internal/server"
)

// Config holds all environment configuration
type Config struct {
	// Server
	Port string

	// Registry
	RegistryURL   string
	RegistryToken string
	RegistryOwner string

	// GitHub
	GitHubToken string
	RepoOwner   string
	RepoName    string

	// Mongo (for aggregation)
	MongoURI string

	// Baseline for diff generation
	BaselinePath string

	// OpenAI API key for AI analysis
	OpenAIAPIKey string
}

func loadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Port:          getEnv("PORT", "8080"),
		RegistryURL:   getEnv("REGISTRY_URL", "https://git.duti.dev"),
		RegistryToken: getEnv("REGISTRY_TOKEN", ""),
		RegistryOwner: getEnv("REGISTRY_OWNER", "acheong08"),
		GitHubToken:   getEnv("GITHUB_TOKEN", ""),
		RepoOwner:     getEnv("REPO_OWNER", "acheong08"),
		RepoName:      getEnv("REPO_NAME", "hackeurope-spr"),
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		BaselinePath:  getEnv("BASELINE_PATH", "safe-sample.json"),
		OpenAIAPIKey:  getEnv("OPENAI_API_KEY", ""),
	}

	// Validate required fields
	if config.RegistryToken == "" {
		return nil, fmt.Errorf("REGISTRY_TOKEN is required")
	}
	if config.GitHubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is required")
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for demo
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client represents a connected WebSocket client
type Client struct {
	conn   *websocket.Conn
	config *Config
	send   chan server.Message
	// Track if analysis is running (one at a time)
	analysisCtx    context.Context
	analysisCancel context.CancelFunc
}

func newClient(conn *websocket.Conn, config *Config) *Client {
	return &Client{
		conn:   conn,
		config: config,
		send:   make(chan server.Message, 256),
	}
}

func (c *Client) SendMessage(msg server.Message) {
	select {
	case c.send <- msg:
	default:
		// Channel full, drop message
		log.Println("Warning: message channel full, dropping message")
	}
}

func (c *Client) SendLog(message, level string) {
	c.SendMessage(server.NewLogMessage(message, level))
}

func (c *Client) SendProgress(percent int, stage, message string) {
	c.SendMessage(server.NewProgressMessage(percent, stage, message))
}

func (c *Client) SendError(message string, err error) {
	c.SendMessage(server.NewErrorMessage(message, err))
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(msg); err != nil {
				log.Printf("Error writing message: %v", err)
				return
			}

		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		// Cancel any running analysis
		if c.analysisCancel != nil {
			c.analysisCancel()
		}
		c.conn.Close()
	}()

	for {
		var msg server.Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return
		}

		switch msg.Type {
		case server.TypeAnalyze:
			c.handleAnalyze(msg)
		case server.TypePing:
			// Respond with pong
			c.SendMessage(server.Message{Type: "pong"})
		default:
			c.SendError(fmt.Sprintf("Unknown message type: %s", msg.Type), nil)
		}
	}
}

func (c *Client) handleAnalyze(msg server.Message) {
	// Check if already analyzing
	if c.analysisCtx != nil && c.analysisCtx.Err() == nil {
		c.SendError("Analysis already in progress", nil)
		return
	}

	// Parse payload
	payload, err := server.ParseAnalyzePayload(msg)
	if err != nil {
		c.SendError("Failed to parse analyze request", err)
		return
	}

	// Create cancellable context for this analysis
	c.analysisCtx, c.analysisCancel = context.WithCancel(context.Background())
	defer func() {
		c.analysisCtx = nil
		c.analysisCancel = nil
	}()

	// Run analysis pipeline
	pipeline := server.NewPipeline(c.config.RegistryURL, c.config.RegistryToken, c.config.RegistryOwner,
		c.config.GitHubToken, c.config.RepoOwner, c.config.RepoName, c, c.config.BaselinePath, c.config.OpenAIAPIKey)

	if err := pipeline.Run(c.analysisCtx, payload.PackageJSON); err != nil {
		if c.analysisCtx.Err() == context.Canceled {
			c.SendLog("Analysis cancelled", "warning")
		} else {
			c.SendError("Analysis failed", err)
		}
		return
	}

	c.SendMessage(server.NewCompleteMessage(true, "Analysis complete"))
}

func serveWs(config *Config, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := newClient(conn, config)

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// WebSocket endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(config, w, r)
	})

	port := config.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
