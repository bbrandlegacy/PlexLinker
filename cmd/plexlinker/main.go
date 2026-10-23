package main

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"plexlinker/internal/config"
	"plexlinker/internal/downloader"
	"plexlinker/internal/logger"
	tvui "plexlinker/internal/tvrenamer/ui"
	"plexlinker/internal/ui"
)

var (
	log *logger.Logger
)

type AppState int

const (
	StateMenu AppState = iota
	StateBrowser
	StateTVRenamer
	StateSettings
	StateQuit
)

type App struct {
	screen        tcell.Screen
	config        *config.Config
	state         AppState
	currentUser   string
	menuUI        *ui.MainMenuUI
	browserUI     *ui.BrowserUI
	tvRenamerUI   *tvui.TVBrowserUI
	settingsUI    *ui.SettingsUI
	generator     *downloader.ScriptGenerator
	signalHandler *ui.SignalHandler
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load config first to check debug mode
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Initialize centralized debug logger
	logCfg := logger.Config{
		Enabled:  cfg.DebugMode || os.Getenv("PLEX_DEBUG") != "",
		FilePath: "plexlinker_debug.log",
		ToStderr: true, // Log to both file and stderr
	}

	if err := logger.Initialize(logCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Close() }()

	log = logger.Get()
	log.Main().Println("=== PlexLinker Debug Mode Started ===")
	log.Main().Printf("Config loaded: BaseURL=%s, Username=%s, DebugMode=%v",
		cfg.BaseURL, cfg.Username, cfg.DebugMode)

	log.Main().Println("Starting run()")

	log.Main().Println("Creating tcell screen...")
	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("failed to create screen: %v", err)
	}

	log.Main().Println("Initializing screen...")
	if err := screen.Init(); err != nil {
		return fmt.Errorf("failed to initialize screen: %v", err)
	}
	defer screen.Fini()

	// Username selection/creation
	log.Main().Println("Handling username selection...")
	var currentUser string
	if len(cfg.Usernames) == 0 {
		// First time - create username
		log.Main().Println("No usernames found, prompting for creation...")
		currentUser = promptCreateUsername(&screen)
		if currentUser == "QUIT" {
			return fmt.Errorf("user cancelled setup")
		}
		cfg.Usernames = append(cfg.Usernames, currentUser)
		cfg.LastUser = currentUser
		if err := cfg.Save(); err != nil {
			log.Main().Printf("Failed to save config: %v", err)
		}
	} else {
		// Show selection UI
		log.Main().Println("Showing username selector...")
		currentUser = showUsernameSelector(&screen, cfg.Usernames, cfg.LastUser)
		if currentUser == "QUIT" {
			return fmt.Errorf("user cancelled")
		}
		// Check if it's a new user (not in the list)
		isNewUser := true
		for _, existing := range cfg.Usernames {
			if existing == currentUser {
				isNewUser = false
				break
			}
		}
		if isNewUser {
			cfg.Usernames = append(cfg.Usernames, currentUser)
		}
		cfg.LastUser = currentUser
		if err := cfg.Save(); err != nil {
			log.Main().Printf("Failed to save config: %v", err)
		}
	}
	log.Main().Printf("Selected user: %s", currentUser)

	log.Main().Println("Creating app components...")
	menuUI := ui.NewMainMenuUI(&screen, cfg)
	menuUI.SetCurrentUser(currentUser)

	app := &App{
		screen:        screen,
		config:        cfg,
		state:         StateMenu,
		currentUser:   currentUser,
		menuUI:        menuUI,
		browserUI:     ui.NewBrowserUI(&screen, cfg, currentUser),
		tvRenamerUI:   tvui.NewTVBrowserUI(&screen),
		settingsUI:    ui.NewSettingsUI(&screen, cfg),
		generator:     downloader.NewScriptGenerator(cfg),
		signalHandler: ui.NewSignalHandler(&screen),
	}

	log.Main().Println("Starting signal handler...")
	// Start signal monitoring
	app.signalHandler.Start()
	defer app.signalHandler.Stop()

	// Clean up old scripts (older than 2 days)
	log.Main().Println("Cleaning up old scripts...")
	if err := downloader.CleanupOldScripts(); err != nil {
		log.Main().Printf("Script cleanup failed (non-fatal): %v", err)
		// Non-fatal error, continue anyway
	}

	// Clean up orphaned tasks for current user
	log.Main().Printf("Cleaning up orphaned tasks for user: %s", currentUser)
	if count, err := downloader.CleanupOrphanedTasks(currentUser); err != nil {
		log.Main().Printf("Task cleanup failed (non-fatal): %v", err)
	} else if count > 0 {
		log.Main().Printf("Cleaned up %d orphaned task(s)", count)
	}

	log.Main().Println("Entering main app loop...")
	return app.run()
}

func (app *App) run() error {
	defer func() {
		if r := recover(); r != nil {
			app.screen.Fini()
			log.Main().Printf("Application panic: %v", r)
			fmt.Fprintf(os.Stderr, "Application panic: %v\n", r)
			os.Exit(1)
		}
	}()

	log.Main().Println("Starting main state machine loop")
	for app.state != StateQuit {
		// Check for disconnect signals
		if app.signalHandler.ShouldExit() {
			log.Main().Println("Signal handler requested exit")
			break
		}

		log.Main().Printf("Current state: %v", app.state)
		switch app.state {
		case StateMenu:
			log.Main().Println("Entering menu state")
			app.runMenu()
		case StateBrowser:
			log.Main().Println("Entering browser state")
			app.runBrowser()
		case StateTVRenamer:
			log.Main().Println("Entering TV renamer state")
			app.runTVRenamer()
		case StateSettings:
			log.Main().Println("Entering settings state")
			app.runSettings()
		}
		log.Main().Printf("State changed to: %v", app.state)
	}

	log.Main().Println("Exiting main loop - app.state == StateQuit")
	return nil
}

func (app *App) runMenu() {
	log.Main().Println("runMenu: Starting menu loop")
	for app.state == StateMenu {
		// Check for disconnect
		if app.signalHandler.ShouldExit() {
			log.Main().Println("runMenu: Signal handler exit requested")
			app.state = StateQuit
			return
		}

		log.Main().Println("runMenu: Drawing menu")
		app.menuUI.Draw()

		log.Main().Println("runMenu: Polling for event")
		ev := app.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			log.Main().Printf("runMenu: Key event received: %v", ev.Key())
			option := app.menuUI.HandleKeyEvent(ev)
			log.Main().Printf("runMenu: Menu option selected: %v", option)
			switch option {
			case ui.MenuBrowse:
				log.Main().Println("runMenu: Browse selected, validating config")
				if err := app.config.Validate(); err != nil {
					log.Main().Printf("runMenu: Config validation failed: %v", err)
					app.menuUI.SetStatusMessage(fmt.Sprintf("Please configure settings first: %v", err))
				} else {
					log.Main().Println("runMenu: Config valid, switching to browser")
					app.state = StateBrowser
				}
			case ui.MenuTVRenamer:
				log.Main().Println("runMenu: TV Renamer selected")
				app.state = StateTVRenamer
			case ui.MenuSettings:
				log.Main().Println("runMenu: Settings selected")
				app.state = StateSettings
			case ui.MenuQuit:
				log.Main().Println("runMenu: Quit selected")
				app.state = StateQuit
			}
		case *tcell.EventResize:
			log.Main().Println("runMenu: Resize event")
			app.screen.Sync()
		case *tcell.EventInterrupt:
			log.Main().Println("runMenu: Interrupt received (Ctrl+C)")
			app.state = StateQuit
			return
		}
	}
	log.Main().Println("runMenu: Exiting menu loop")
}

func (app *App) runBrowser() {
	log.Main().Println("runBrowser: Loading initial directory")
	if err := app.browserUI.LoadDirectory(""); err != nil {
		log.Main().Printf("runBrowser: Failed to load directory: %v", err)
		app.menuUI.SetStatusMessage(fmt.Sprintf("Failed to load directory: %v", err))
		app.state = StateMenu
		return
	}

	log.Main().Println("runBrowser: Starting browser loop")
	for app.state == StateBrowser {
		// Check for disconnect
		if app.signalHandler.ShouldExit() {
			log.Main().Println("runBrowser: Signal handler exit requested")
			app.state = StateQuit
			return
		}

		log.Main().Println("runBrowser: Drawing browser UI")
		app.browserUI.Draw()

		log.Main().Println("runBrowser: Polling for event")
		ev := app.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			log.Main().Printf("runBrowser: Key event received: %v", ev.Key())
			shouldExit := app.browserUI.HandleKeyEvent(ev)
			log.Main().Printf("runBrowser: HandleKeyEvent returned shouldExit=%v", shouldExit)
			if shouldExit {
				log.Main().Println("runBrowser: Returning to menu")
				app.state = StateMenu
			}
		case *tcell.EventResize:
			log.Main().Println("runBrowser: Resize event")
			app.screen.Sync()
		case *tcell.EventInterrupt:
			log.Main().Println("runBrowser: Interrupt received (Ctrl+C)")
			app.state = StateQuit
			return
		}
	}
	log.Main().Println("runBrowser: Exiting browser loop")
}

func (app *App) runSettings() {
	for app.state == StateSettings {
		// Check for disconnect
		if app.signalHandler.ShouldExit() {
			app.state = StateQuit
			return
		}

		app.settingsUI.Draw()

		ev := app.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if app.settingsUI.HandleKeyEvent(ev) {
				app.state = StateMenu
			}
		case *tcell.EventResize:
			app.screen.Sync()
		case *tcell.EventInterrupt:
			app.state = StateQuit
			return
		}
	}
}

func (app *App) runTVRenamer() {
	if err := app.tvRenamerUI.LoadCurrentDirectory(); err != nil {
		app.menuUI.SetStatusMessage(fmt.Sprintf("Failed to load directory: %v", err))
		app.state = StateMenu
		return
	}

	for app.state == StateTVRenamer {
		// Check for disconnect
		if app.signalHandler.ShouldExit() {
			app.state = StateQuit
			return
		}

		app.tvRenamerUI.Draw()

		ev := app.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if app.tvRenamerUI.HandleKeyEvent(ev) {
				app.state = StateMenu
			}
		case *tcell.EventResize:
			app.screen.Sync()
		case *tcell.EventInterrupt:
			app.state = StateQuit
			return
		}
	}
}

// promptCreateUsername shows a simple prompt for first-time username creation
func promptCreateUsername(screen *tcell.Screen) string {
	selector := ui.NewUserSelectorUI(screen, []string{}, "")
	// Force new user mode
	selector.Draw()

	for {
		ev := (*screen).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			result := selector.HandleKeyEvent(ev)
			if result != "" {
				return result
			}
		case *tcell.EventResize:
			(*screen).Sync()
		}
		selector.Draw()
	}
}

// showUsernameSelector shows the username selection UI
func showUsernameSelector(screen *tcell.Screen, usernames []string, lastUser string) string {
	selector := ui.NewUserSelectorUI(screen, usernames, lastUser)
	selector.Draw()

	for {
		ev := (*screen).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			result := selector.HandleKeyEvent(ev)
			if result != "" {
				return result
			}
		case *tcell.EventResize:
			(*screen).Sync()
		}
		selector.Draw()
	}
}
