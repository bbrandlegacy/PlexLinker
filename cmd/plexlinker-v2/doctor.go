package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"plexlinker/internal/aria2manager"
	"plexlinker/internal/browser"
	"plexlinker/internal/buildinfo"
	"plexlinker/internal/client"
	"plexlinker/internal/config"
	"plexlinker/internal/storage"
)

func runDoctor(arguments []string) error {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	destination := flags.String("destination", "", "destination directory to verify as writable")
	timeout := flags.Duration("timeout", 20*time.Second, "maximum time for each network or service check")
	if err := flags.Parse(arguments); err != nil {
		return err
	}

	failures := 0
	pass := func(format string, values ...any) { fmt.Printf("PASS  "+format+"\n", values...) }
	warn := func(format string, values ...any) { fmt.Printf("WARN  "+format+"\n", values...) }
	fail := func(format string, values ...any) {
		failures++
		fmt.Printf("FAIL  "+format+"\n", values...)
	}

	fmt.Printf("PlexLinker doctor %s (%s/%s)\n", buildinfo.Version, runtime.GOOS, runtime.GOARCH)
	configPath, err := config.GetV2ConfigPath()
	if err != nil {
		fail("config path: %v", err)
	} else {
		pass("config path: %s", configPath)
	}

	cfg, err := config.LoadV2Config()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	cfg.Normalize()
	if err := cfg.ValidateV2(); err != nil {
		fail("configuration: %v", err)
	} else {
		pass("configuration is valid; remote host is %s", cfg.BaseURL)
	}
	if cfg.Aria2Secret == "" {
		cfg.Aria2Secret, err = aria2manager.GenerateSecret()
		if err != nil {
			fail("generate aria2 RPC secret: %v", err)
		} else if err := cfg.SaveV2(); err != nil {
			fail("save generated aria2 RPC secret: %v", err)
		} else {
			pass("generated and saved an aria2 RPC secret")
		}
	}

	if path, err := exec.LookPath("aria2c"); err != nil {
		fail("aria2c is not available on PATH: %v", err)
	} else {
		pass("aria2 executable: %s", path)
	}

	manager, managerErr := aria2manager.New(aria2manager.Config{
		Endpoint: cfg.Aria2Endpoint, Secret: cfg.Aria2Secret, AutoStart: cfg.Aria2AutoStart,
		MaxConcurrent: cfg.MaxConcurrentDownloads,
	})
	if managerErr != nil {
		fail("aria2 manager: %v", managerErr)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		status, ensureErr := manager.Ensure(ctx)
		cancel()
		if ensureErr != nil {
			fail("persistent aria2 RPC: %v", ensureErr)
		} else {
			pass("aria2 RPC %s at %s", status.Version, cfg.Aria2Endpoint)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	persistence, persistenceErr := persistenceProbe(ctx)
	cancel()
	if persistenceErr != nil {
		fail("persistence: %v", persistenceErr)
	} else {
		pass("persistence: %s", persistence)
	}

	ctx, cancel = context.WithTimeout(context.Background(), *timeout)
	volumes, volumeErr := storage.NewVolumeProvider().ListVolumes(ctx)
	cancel()
	if volumeErr != nil {
		fail("enumerate destination volumes: %v", volumeErr)
	} else if len(volumes) == 0 {
		warn("no destination volumes were discovered")
	} else {
		pass("discovered %d destination volume(s)", len(volumes))
		for _, volume := range volumes {
			availability := "available"
			if !volume.Available {
				availability = "unavailable: " + volume.Error
			}
			fmt.Printf("      %s [%s] %s\n", volume.Path, volume.Kind, availability)
		}
	}

	if *destination == "" {
		warn("destination write check skipped; pass --destination PATH")
	} else if err := storage.NewBrowser().IsWritable(*destination); err != nil {
		fail("destination write check: %v", err)
	} else {
		pass("destination is writable: %s", *destination)
	}

	if cfg.ValidateV2() == nil {
		ctx, cancel = context.WithTimeout(context.Background(), *timeout)
		html, fetchErr := client.NewClient(cfg).FetchDirectoryContext(ctx, "")
		cancel()
		if fetchErr != nil {
			fail("remote HTTPS/authentication check: %v", fetchErr)
		} else if _, parseErr := browser.ParseHTMLDirectory(html, ""); parseErr != nil {
			fail("remote directory parser check: %v", parseErr)
		} else {
			pass("remote directory authentication and parsing")
		}
	}

	if failures > 0 {
		return fmt.Errorf("%d check(s) failed", failures)
	}
	if runtime.GOOS == "windows" {
		if _, err := fmt.Fprintln(os.Stdout, "NEXT  Queue a small download, disconnect SSH, reconnect, and confirm progress with the Downloads tab."); err != nil {
			return fmt.Errorf("write acceptance instructions: %w", err)
		}
	}
	return nil
}
