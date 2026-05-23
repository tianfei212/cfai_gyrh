package kiosk

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// Run loads configuration, launches Chrome in kiosk mode, and optionally
// restarts it if the browser process exits.
func Run(ctx context.Context, configPath string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	chromePath, err := FindChrome(cfg.ChromePath, nil)
	if err != nil {
		return err
	}
	log.Printf("使用 Chrome: %s", chromePath)
	log.Printf("打开页面: %s", cfg.URL)

	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	return runLoop(runCtx, cfg, chromePath)
}

func runLoop(ctx context.Context, cfg Config, chromePath string) error {
	for {
		cmd := exec.CommandContext(ctx, chromePath, BuildChromeArgs(cfg)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("启动 Chrome 失败: %w", err)
		}
		log.Printf("Chrome 已启动，PID=%d", cmd.Process.Pid)

		err := cmd.Wait()
		if ctx.Err() != nil {
			if cfg.CloseChromeOnExit && cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			return nil
		}
		if err != nil {
			log.Printf("Chrome 已退出: %v", err)
		} else {
			log.Printf("Chrome 已退出")
		}

		if !cfg.AutoRestart {
			return err
		}

		delay := cfg.RestartDelay
		if delay <= 0 {
			delay = DefaultRestartDelaySeconds * time.Second
		}
		log.Printf("%s 后重新打开 Chrome", delay)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
	}
}
