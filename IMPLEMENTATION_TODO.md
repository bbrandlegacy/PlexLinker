# PlexLinker - Task Scheduler + Username Implementation TODO

## Overview
Implement Windows Task Scheduler execution for SSH persistence + Username selection system.

## Phase 1: Username Selection System

### 1.1 Config Structure Changes
- [ ] **File**: `internal/config/config.go`
  - [ ] Add `Usernames []string` field to Config struct
  - [ ] Add `LastUser string` field (optional, for auto-select)
  - [ ] Update JSON tags appropriately
  - [ ] Add validation in `Validate()` method (ensure Usernames exists)

### 1.2 User Selector UI Component
- [ ] **New File**: `internal/ui/user_selector.go`
  - [ ] Create `UserSelectorUI` struct with:
    - `screen *tcell.Screen`
    - `usernames []string`
    - `selectedIndex int`
    - `mode` (selection vs new user input)
  - [ ] Implement `NewUserSelectorUI(screen, usernames, lastUser)` constructor
  - [ ] Implement `Draw()` method:
    - Show title: "Select User"
    - List usernames with arrow indicator `>`
    - Show "+ Add New User" option at bottom
    - Show instructions: "[↑↓]Navigate [Enter]Select [Esc]Quit"
  - [ ] Implement `HandleKeyEvent(ev)` method:
    - Up/Down arrows: Navigate list
    - Enter: Return selected username
    - If "Add New User" selected: Switch to input mode
  - [ ] Implement `ShowNewUserInput()` method:
    - Text input field for new username
    - Validation: non-empty, alphanumeric + underscore/hyphen only
    - Return new username string

### 1.3 First-Time User Creation
- [ ] **New File**: `internal/ui/user_creator.go` (or add to user_selector.go)
  - [ ] Create `UserCreatorUI` struct
  - [ ] Implement `Draw()` method:
    - Welcome message
    - Prompt: "Create your username:"
    - Input field
    - Instructions
  - [ ] Implement `HandleKeyEvent()` for text input
  - [ ] Return validated username

### 1.4 Main Application Integration
- [ ] **File**: `cmd/plexlinker/main.go`
  - [ ] Add `currentUser string` field to `App` struct
  - [ ] In `run()` function, BEFORE screen initialization:
    - [ ] Load config
    - [ ] Check if `cfg.Usernames` is empty or nil
    - [ ] If empty: Show UserCreatorUI, get username, save to config
    - [ ] If has usernames: Show UserSelectorUI, get selected username
    - [ ] Save selected username to `cfg.LastUser`, save config
    - [ ] Set `app.currentUser = selectedUsername`
  - [ ] Pass `currentUser` to components that need it (BrowserUI, ScriptGenerator)

### 1.5 Update UI Components
- [ ] **File**: `internal/ui/menu.go`
  - [ ] Update header to show: "PlexLinker - User: {currentUser}"
  - [ ] Optional: Add "Switch User" menu option (can be Phase 2)

- [ ] **File**: `internal/ui/browser.go`
  - [ ] Add `currentUser string` field to BrowserUI struct
  - [ ] Accept currentUser in constructor
  - [ ] Pass to ScriptGeneratorUI when opening

- [ ] **File**: `internal/ui/script_generator.go`
  - [ ] Add `currentUser string` field to ScriptGeneratorState
  - [ ] Pass to downloader.ScriptGenerator when generating scripts

---

## Phase 2: Task Scheduler Implementation

### 2.1 Task Scheduler Execution (Windows)
- [ ] **File**: `internal/downloader/exec_windows.go`
  - [ ] Add `executeViaTaskScheduler(scriptPath, username, scriptName)` function:
    - [ ] Generate unique task name: `PlexLinker_{username}_{scriptName}_{timestamp}_{random4}`
    - [ ] Build `schtasks /create` command:
      - Task name: generated name
      - Action: `powershell -ExecutionPolicy Bypass -File "{scriptPath}"`
      - Schedule: `/sc once /st 00:00 /sd 01/01/2000` (run immediately)
      - Permissions: `/rl LIMITED` (no admin needed)
      - Force: `/f` flag
    - [ ] Execute `schtasks /create` via `exec.Command()`
    - [ ] Check for errors
    - [ ] Execute `schtasks /run /tn "{taskName}"` to start immediately
    - [ ] Return ScriptExecution with task name tracked

  - [ ] Add `executeDirect(scriptPath)` function (fallback):
    - [ ] Use existing PowerShell Start-Process method
    - [ ] Return ScriptExecution with warning flag

  - [ ] Modify `executeScriptPlatform(scriptPath, fileCount)`:
    - [ ] Extract username from context (pass as parameter)
    - [ ] Extract script name from path
    - [ ] Try `executeViaTaskScheduler()` first
    - [ ] If success: Return with success message
    - [ ] If failure: Log warning, call `executeDirect()`
    - [ ] Return ScriptExecution with execution method flag

- [ ] **File**: `internal/downloader/generator.go`
  - [ ] Modify `ExecuteScript(scriptPath, fileCount, username)`:
    - [ ] Add `username string` parameter
    - [ ] Pass to `executeScriptPlatform()`

### 2.2 PowerShell Script Self-Cleanup
- [ ] **File**: `internal/downloader/generator.go`
  - [ ] Modify `buildCategoryPowerShellScript()`:
    - [ ] Add at END of script (before ReadKey):
      ```powershell
      # Self-delete the scheduled task
      if ($env:PLEXLINKER_TASK_NAME) {
          Start-Sleep -Seconds 2
          schtasks /delete /tn $env:PLEXLINKER_TASK_NAME /f 2>$null
      }
      ```
  - [ ] Modify `buildBatchScript()` (same addition)

### 2.3 Task Name Passing
- [ ] **File**: `internal/downloader/exec_windows.go`
  - [ ] In `executeViaTaskScheduler()`:
    - [ ] Set environment variable when creating task:
      - Modify task action to: `cmd /c "set PLEXLINKER_TASK_NAME={taskName} && powershell ..."`
      - OR: Pass as PowerShell parameter: `powershell -ExecutionPolicy Bypass -Command "$env:PLEXLINKER_TASK_NAME='{taskName}'; & '{scriptPath}'"`

### 2.4 Orphan Task Cleanup (Startup)
- [ ] **File**: `internal/downloader/generator.go`
  - [ ] Add `CleanupOrphanedTasks(username)` function:
    - [ ] Query all tasks: `schtasks /query /fo CSV /v`
    - [ ] Filter tasks matching: `PlexLinker_{username}_*`
    - [ ] Parse creation date/time
    - [ ] If older than 24 hours: Delete task
    - [ ] Return count of cleaned tasks

- [ ] **File**: `cmd/plexlinker/main.go`
  - [ ] In `run()`, AFTER user selection:
    - [ ] Call `downloader.CleanupOrphanedTasks(currentUser)`
    - [ ] Log count of cleaned tasks (debug mode only)

### 2.5 User Feedback for Execution Method
- [ ] **File**: `internal/ui/script_generator.go`
  - [ ] Modify `generateAndRun()`:
    - [ ] Track execution method for each script (from ScriptExecution)
    - [ ] Count successes per method (taskScheduler vs direct)
    - [ ] Update status message:
      - If all via Task Scheduler: "✓ 2 scripts launched via Task Scheduler (SSH-safe)"
      - If any fallback: "⚠️ 1 script in FALLBACK MODE - Keep SSH open!"
      - If all fallback: "⚠️ ALL SCRIPTS IN FALLBACK MODE - DO NOT CLOSE SSH!"

### 2.6 ScriptExecution Struct Enhancement
- [ ] **File**: `internal/downloader/generator.go`
  - [ ] Add fields to `ScriptExecution`:
    - [ ] `ExecutionMethod string` ("task_scheduler" or "direct")
    - [ ] `TaskName string` (for task scheduler method)
    - [ ] `Warning bool` (true if fallback used)

---

## Phase 3: Integration & Testing

### 3.1 End-to-End Flow Testing
- [ ] Test: First-time user creation
  - [ ] Empty config → Prompts for username → Saves to config
- [ ] Test: Username selection
  - [ ] Shows list of usernames
  - [ ] Arrow navigation works
  - [ ] Enter selects username
  - [ ] LastUser auto-selects (if implemented)
- [ ] Test: Add new user from selector
  - [ ] "+ Add New User" option works
  - [ ] Input validation works
  - [ ] New user added to config
- [ ] Test: Task Scheduler execution (Windows only)
  - [ ] Task created with correct name format
  - [ ] PowerShell window opens
  - [ ] Downloads start
  - [ ] Task self-deletes after completion
- [ ] Test: Multiple concurrent scripts
  - [ ] Each script gets unique task
  - [ ] All PowerShell windows open
  - [ ] No task name collisions
- [ ] Test: SSH persistence (Windows only)
  - [ ] SSH in, generate scripts
  - [ ] Disconnect SSH
  - [ ] Verify PowerShell windows still running
  - [ ] Verify downloads continue
- [ ] Test: Fallback mode
  - [ ] Simulate Task Scheduler failure (stop service or permission issue)
  - [ ] Verify fallback to direct execution
  - [ ] Verify warning message displayed
- [ ] Test: Orphan cleanup
  - [ ] Create old tasks manually
  - [ ] Restart PlexLinker
  - [ ] Verify old tasks deleted

### 3.2 Documentation Updates
- [ ] **File**: `CLAUDE.md`
  - [ ] Document username selection feature
  - [ ] Document Task Scheduler execution
  - [ ] Document SSH persistence
  - [ ] Update "Recent Improvements" section

### 3.3 Build Verification
- [ ] Build for Windows: `GOOS=windows GOARCH=amd64 go build -o plexlinker.exe ./cmd/plexlinker`
- [ ] Build for macOS: `go build -o plexlinker ./cmd/plexlinker`
- [ ] Verify no compilation errors

---

## Implementation Order (Recommended)

1. ✅ Username config structure (1.1)
2. ✅ User selector UI (1.2)
3. ✅ First-time user creation (1.3)
4. ✅ Main app integration (1.4)
5. ✅ UI updates (1.5)
6. ✅ Task Scheduler execution (2.1)
7. ✅ PowerShell self-cleanup (2.2)
8. ✅ Task name passing (2.3)
9. ✅ User feedback (2.5)
10. ✅ ScriptExecution enhancement (2.6)
11. ✅ Orphan cleanup (2.4)
12. ✅ Testing (3.1)
13. ✅ Documentation (3.2)

---

## Key Design Decisions

### Username Format
- **Pattern**: `PlexLinker_{username}_{scriptname}_{timestamp}_{random4}`
- **Example**: `PlexLinker_john_Inception_2010_20251010_143022_a3f9`

### Task Scheduler Command
```bash
schtasks /create /tn "TaskName" /tr "powershell -ExecutionPolicy Bypass -File \"script.ps1\"" /sc once /st 00:00 /sd 01/01/2000 /rl LIMITED /f
schtasks /run /tn "TaskName"
```

### Fallback Logic
```
Try Task Scheduler
  ↓
  Success? → Use Task Scheduler (SSH-safe)
  ↓
  Failed? → Use Direct Execution (SSH-unsafe)
           → Show warning to user
```

### Cleanup Strategy
- **Primary**: Scripts self-delete their tasks on completion
- **Backup**: Startup cleanup removes tasks older than 24h
- **Scope**: Only clean tasks for current user

---

## Critical Edge Cases Addressed

✅ Multiple concurrent scripts (unique task names)
✅ Rapid successive generations (timestamp + random)
✅ Task Scheduler failures (graceful fallback + warning)
✅ SSH disconnection (Task Scheduler survives)
✅ Orphaned tasks (startup cleanup)
✅ Multi-user scenarios (username prefix)
✅ Task name collisions (timestamp + random suffix)
✅ PlexLinker crash (tasks self-delete + 24h expiry)

---

## Files to Create
- `internal/ui/user_selector.go`
- `internal/ui/user_creator.go` (optional, can be in user_selector.go)
- `IMPLEMENTATION_TODO.md` (this file)

## Files to Modify
- `internal/config/config.go`
- `cmd/plexlinker/main.go`
- `internal/ui/menu.go`
- `internal/ui/browser.go`
- `internal/ui/script_generator.go`
- `internal/downloader/exec_windows.go`
- `internal/downloader/generator.go`
- `CLAUDE.md`

---

## Success Criteria
- ✅ Username selected before main menu
- ✅ Username displayed in header
- ✅ Task names include username
- ✅ Scripts execute via Task Scheduler on Windows
- ✅ PowerShell windows survive SSH disconnect
- ✅ Clear warning if fallback mode used
- ✅ Old tasks cleaned up automatically
- ✅ No task name collisions
- ✅ Smooth user experience

---

## Notes for Implementation
- Keep Phase 1 and Phase 2 separate but integrated
- Test each phase independently before moving to next
- Windows-specific code already properly isolated in `*_windows.go` files
- Username system is platform-agnostic (works on all platforms)
- Task Scheduler is Windows-only (has proper stubs for other platforms)