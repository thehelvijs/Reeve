// Command agent runs on each Linux host and pushes telemetry (systemd, Docker,
// cron, host metrics, log errors) to the Reeve server on an interval.
package main

import (
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"
)

var version = "dev"

// config is the agent's runtime configuration, read from the environment.
type config struct {
	ServerURL      string
	Token          string
	Interval       time.Duration
	AutoUpdate     bool
	UpdateInterval time.Duration
}

func loadConfig() (config, error) {
	c := config{
		ServerURL:      os.Getenv("REEVE_SERVER_URL"),
		Token:          os.Getenv("REEVE_AGENT_TOKEN"),
		Interval:       15 * time.Second,
		AutoUpdate:     os.Getenv("REEVE_AUTO_UPDATE") != "false",
		UpdateInterval: time.Hour,
	}
	if v := os.Getenv("REEVE_PUSH_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return c, err
		}
		c.Interval = d
	}
	if v := os.Getenv("REEVE_UPDATE_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return c, err
		}
		c.UpdateInterval = d
	}
	return c, nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("agent config: %v", err)
	}
	if cfg.ServerURL == "" || cfg.Token == "" {
		log.Fatal("agent: REEVE_SERVER_URL and REEVE_AGENT_TOKEN are required")
	}
	log.Printf("Reeve agent %s pushing to %s every %s", version, cfg.ServerURL, cfg.Interval)

	p := newPusher(cfg)
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	updateTicker := time.NewTicker(cfg.UpdateInterval)
	defer updateTicker.Stop()

	var updating atomic.Bool

	run := func() {
		p.flushBuffer()
		if err := p.send(gather(version, cfg)); err != nil {
			log.Printf("push failed (buffered): %v", err)
		}
	}
	run()
	for {
		select {
		case <-ticker.C:
			run()
		case <-updateTicker.C:
			if cfg.AutoUpdate {
				go runSelfUpdate(cfg, &updating)
			}
		}
	}
}

// runSelfUpdate runs checkAndUpdate off the push loop so a slow download or
// hash never blocks a push tick; the guard drops a tick if one is still in
// flight instead of stacking overlapping runs.
func runSelfUpdate(cfg config, updating *atomic.Bool) {
	if !updating.CompareAndSwap(false, true) {
		return
	}
	defer updating.Store(false)
	if err := cfg.checkAndUpdate(); err != nil {
		log.Printf("self-update: %v", err)
	}
}
