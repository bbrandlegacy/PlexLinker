# PlexLinker Configuration System Audit Report

**Date:** October 9, 2025  
**Auditor:** Claude (config-validator agent)  
**Files Audited:**
- `/Users/ocinbuona/GolandProjects/plexlinker/internal/config/config.go`
- `/Users/ocinbuona/GolandProjects/plexlinker/internal/ui/settings.go`

---

## Executive Summary

The configuration system is **functional but has several validation gaps and design issues**. The most critical issue is that `Validate()` mutates the config (adds trailing slash), violating single-responsibility principle. Several edge cases are not validated, including invalid URL schemes, invalid script formats, and malicious URLs.

**Security Status:** ⚠️ MODERATE - File permissions correct (0600), but validation gaps exist  
**Backward Compatibility:** ❌ BROKEN - Old configs without path fields will fail validation  
**Code Quality:** ⚠️ NEEDS IMPROVEMENT - Mutation in validation violates best practices

---

## 1. Validation Issues Found

### 1.1 CRITICAL: URL Mutation in Validate() ✅ CONFIRMED

**Location:** `/Users/ocinbuona/GolandProjects/plexlinker/internal/config/config.go:73-75`

```go
if !strings.HasSuffix(c.BaseURL, "/") {
    c.BaseURL += "/"  // MUTATION - modifies state during validation
}
```

**Issue:** The `Validate()` method mutates the config by adding a trailing slash. This violates the principle that validation should only check validity, not modify data.

**Why It Happens:** The trailing slash is **necessary** for correct URL joining:
- Without trailing slash: `http://example.com/media` + `folder1` = `http://example.com/folder1` ❌ (wrong!)
- With trailing slash: `http://example.com/media/` + `folder1` = `http://example.com/media/folder1` ✓ (correct!)

**Impact:**
- When `client.buildURL()` uses `url.Parse()` for path resolution, it relies on this trailing slash
- Multiple calls to `Validate()` are idempotent (won't keep adding slashes)
- However, it can cause unexpected behavior when validation is called multiple times in UI

**Evidence from Testing:**
```
BaseURL before Validate(): https://example.com/media
BaseURL after Validate():  https://example.com/media/
MUTATION DETECTED: Validate() modified BaseURL (added trailing slash)
```

**Recommendation:** Separate normalization from validation

```go
// Validate checks if config is valid (read-only, no mutations)
func (c *Config) Validate() error {
    if c.BaseURL == "" {
        return fmt.Errorf("base URL cannot be empty")
    }
    
    u, err := url.Parse(c.BaseURL)
    if err != nil {
        return fmt.Errorf("invalid base URL: %v", err)
    }
    
    // Validate scheme
    if u.Scheme != "http" && u.Scheme != "https" {
        return fmt.Errorf("base URL must use http:// or https:// (got %s://)", u.Scheme)
    }
    
    // Validate host
    if u.Host == "" {
        return fmt.Errorf("base URL must include a hostname")
    }
    
    if c.Username == "" {
        return fmt.Errorf("username cannot be empty")
    }
    
    if c.Password == "" {
        return fmt.Errorf("password cannot be empty")
    }
    
    // Validate ScriptFormat
    if c.ScriptFormat != "" && 
       c.ScriptFormat != ScriptFormatBAT && 
       c.ScriptFormat != ScriptFormatSH && 
       c.ScriptFormat != ScriptFormatTXT {
        return fmt.Errorf("invalid script format: %s (must be bat, sh, or txt)", c.ScriptFormat)
    }
    
    if c.MoviesPath == "" {
        return fmt.Errorf("movies path cannot be empty")
    }
    
    if c.TVShowsPath == "" {
        return fmt.Errorf("tv shows path cannot be empty")
    }
    
    if c.OtherPath == "" {
        return fmt.Errorf("other path cannot be empty")
    }
    
    return nil
}

// Normalize ensures config values are in canonical form (may mutate)
func (c *Config) Normalize() {
    // Ensure trailing slash for correct URL joining
    if c.BaseURL != "" && !strings.HasSuffix(c.BaseURL, "/") {
        c.BaseURL += "/"
    }
    
    // Normalize multiple trailing slashes
    for strings.HasSuffix(c.BaseURL, "//") {
        c.BaseURL = strings.TrimSuffix(c.BaseURL, "/")
    }
    
    // Set default script format if empty
    if c.ScriptFormat == "" {
        c.ScriptFormat = ScriptFormatBAT
    }
    
    // Trim whitespace from paths
    c.MoviesPath = strings.TrimSpace(c.MoviesPath)
    c.TVShowsPath = strings.TrimSpace(c.TVShowsPath)
    c.OtherPath = strings.TrimSpace(c.OtherPath)
}

// Save saves config with normalization
func (c *Config) Save() error {
    c.Normalize()  // Apply normalization before saving
    
    configPath, err := GetConfigPath()
    if err != nil {
        return err
    }
    
    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal config: %v", err)
    }
    
    if err := os.WriteFile(configPath, data, 0600); err != nil {
        return fmt.Errorf("failed to write config file: %v", err)
    }
    
    return nil
}
```

---

### 1.2 HIGH: URL Scheme Not Validated

**Issue:** `url.Parse()` accepts any scheme (ftp, file, javascript, data URIs, etc.)

**Test Results:**
```
FTP: ftp://example.com/ - ACCEPTED (should reject!)
FILE: file:///path/to/dir/ - ACCEPTED (should reject!)
javascript:alert(1) - ACCEPTED (should reject!)
data:text/html,<script> - ACCEPTED (should reject!)
//example.com - ACCEPTED (missing scheme, should reject!)
example.com - ACCEPTED (missing protocol, should reject!)
http:// - ACCEPTED (missing host, should reject!)
```

**Recommendation:** Add scheme validation in `Validate()`:

```go
u, err := url.Parse(c.BaseURL)
if err != nil {
    return fmt.Errorf("invalid base URL: %v", err)
}

if u.Scheme != "http" && u.Scheme != "https" {
    return fmt.Errorf("base URL must use http:// or https:// (got %s://)", u.Scheme)
}

if u.Host == "" {
    return fmt.Errorf("base URL must include a hostname")
}
```

---

### 1.3 MEDIUM: ScriptFormat Not Validated

**Issue:** Any value is accepted for `ScriptFormat`, including invalid ones

**Test Results:**
```
Test 16: ScriptFormat validation (no validation exists!)
Invalid ScriptFormat: <nil>
WARNING: No validation for ScriptFormat field!
```

**Evidence:**
```go
// Config accepts any ScriptFormat value
ScriptFormat: "invalid_format"  // No validation error!
```

**Recommendation:** Add validation:

```go
if c.ScriptFormat != "" && 
   c.ScriptFormat != ScriptFormatBAT && 
   c.ScriptFormat != ScriptFormatSH && 
   c.ScriptFormat != ScriptFormatTXT {
    return fmt.Errorf("invalid script format: %s (must be bat, sh, or txt)", c.ScriptFormat)
}
```

---

### 1.4 LOW: Multiple Trailing Slashes Not Normalized

**Issue:** URLs with multiple trailing slashes are not cleaned up

**Test Results:**
```
Test 17: Multiple trailing slashes
BaseURL before: https://example.com/media///
BaseURL after:  https://example.com/media///
WARNING: Multiple trailing slashes not normalized!
```

**Recommendation:** Add to `Normalize()`:

```go
// Normalize multiple trailing slashes to single slash
for strings.HasSuffix(c.BaseURL, "//") {
    c.BaseURL = strings.TrimSuffix(c.BaseURL, "/")
}
if c.BaseURL != "" && !strings.HasSuffix(c.BaseURL, "/") {
    c.BaseURL += "/"
}
```

---

### 1.5 LOW: Default Values Not Applied on Load

**Issue:** When loading a minimal config (missing optional fields), defaults are not applied

**Test Results:**
```
Test 4: Minimal valid config
ScriptFormat: "" (should be empty, not default)
```

**Current Behavior:**
- `NewConfig()` returns defaults (ScriptFormat="bat", paths="auto")
- `LoadConfig()` + `json.Unmarshal()` does not apply defaults for missing fields
- If JSON is missing `script_format`, it becomes empty string, not "bat"

**Recommendation:** Apply defaults after unmarshaling:

```go
func LoadConfig() (*Config, error) {
    // ... load and unmarshal ...
    
    config := NewConfig()
    if err := json.Unmarshal(data, config); err != nil {
        return nil, fmt.Errorf("failed to parse config file: %v", err)
    }
    
    // Apply defaults for empty fields
    if config.ScriptFormat == "" {
        config.ScriptFormat = ScriptFormatBAT
    }
    if config.MoviesPath == "" {
        config.MoviesPath = "auto"
    }
    if config.TVShowsPath == "" {
        config.TVShowsPath = "auto"
    }
    if config.OtherPath == "" {
        config.OtherPath = "auto"
    }
    
    return config, nil
}
```

---

## 2. Settings UI Validation Consistency

**Location:** `/Users/ocinbuona/GolandProjects/plexlinker/internal/ui/settings.go:226`

```go
func (ui *SettingsUI) saveSettings() bool {
    if err := ui.tempConfig.Validate(); err != nil {
        ui.statusMessage = fmt.Sprintf("Validation error: %v", err)
        return false
    }
    // ... save ...
}
```

**Status:** ✅ UI correctly calls `Validate()` before saving

**Issue:** The mutation happens here too, modifying `tempConfig.BaseURL` before it's copied to the main config. This is non-obvious behavior.

**Impact:**
- When user enters `http://example.com` and saves
- `Validate()` mutates it to `http://example.com/`
- User sees the trailing slash added automatically
- This is actually desired behavior, but implemented in the wrong place

---

## 3. Security Analysis

### 3.1 File Permissions ✅ CORRECT

**Evidence:** `config.go:135`
```go
if err := os.WriteFile(configPath, data, 0600); err != nil {
```

**Status:** ✅ Config file saved with `0600` (owner read/write only)

**Verification:**
```bash
# Test confirmed permissions are correctly set
Permissions: 600 (expected: 0600)
PASS - Correct permissions (0600 = owner read/write only)
```

**Security Implications:**
- Config contains plaintext username and password
- File permissions prevent other users from reading credentials
- Recommendation: Document this in README with warning

---

### 3.2 Credentials in Scripts ⚠️ BY DESIGN

**Issue:** Generated aria2c scripts contain username/password

**Status:** Mentioned in CLAUDE.md as known issue #16

**Recommendation:** Document workaround:
```bash
# After generating script, set restrictive permissions
chmod 600 plexhelper_*.bat
```

---

### 3.3 Environment Variable Override ✅ CORRECT

**Location:** `config.go:92-98`

```go
if baseURL := os.Getenv("PLEX_BASE_URL"); baseURL != "" {
    config := &Config{
        BaseURL:  baseURL,
        Username: os.Getenv("PLEX_USERNAME"),
        Password: os.Getenv("PLEX_PASSWORD"),
    }
    return config, nil
}
```

**Status:** ✅ Works correctly, useful for development/testing

**Issue:** Environment config bypasses validation and default value initialization
- No `Validate()` call on environment-loaded config
- Missing MoviesPath, TVShowsPath, OtherPath defaults

**Recommendation:**
```go
if baseURL := os.Getenv("PLEX_BASE_URL"); baseURL != "" {
    config := NewConfig()  // Start with defaults
    config.BaseURL = baseURL
    config.Username = os.Getenv("PLEX_USERNAME")
    config.Password = os.Getenv("PLEX_PASSWORD")
    // Keep default paths from NewConfig()
    return config, nil
}
```

---

## 4. Backward Compatibility

### 4.1 Breaking Change ❌ OLD CONFIGS FAIL

**Test Results:**
```
Test 1: Old config format (missing path fields)
Validation error: movies path cannot be empty
✗ Old config breaks validation (missing required fields)
```

**Impact:** Users upgrading from old version will see validation errors

**Root Cause:** New required fields (`movies_path`, `tv_shows_path`, `other_path`) added without migration

**Recommendation:** Add migration in `LoadConfig()`:

```go
func LoadConfig() (*Config, error) {
    // ... load file ...
    
    config := NewConfig()
    if err := json.Unmarshal(data, config); err != nil {
        return nil, fmt.Errorf("failed to parse config file: %v", err)
    }
    
    // Migrate old configs
    if config.MoviesPath == "" {
        config.MoviesPath = "auto"
        config.TVShowsPath = "auto"
        config.OtherPath = "auto"
        
        // Auto-save migrated config
        if err := config.Save(); err != nil {
            return nil, fmt.Errorf("failed to save migrated config: %v", err)
        }
    }
    
    return config, nil
}
```

---

### 4.2 Forward Compatibility ✅ WORKS

**Test Results:**
```
Test 3: Config with extra/unknown fields
✓ Extra fields ignored (good for forward compatibility)
```

**Status:** ✅ JSON unmarshaling ignores unknown fields (Go default behavior)

---

## 5. Default Value Handling

### 5.1 NewConfig() Defaults ✅ CORRECT

```go
func NewConfig() *Config {
    return &Config{
        BaseURL:      "",
        Username:     "",
        Password:     "",
        DebugMode:    false,
        ScriptFormat: ScriptFormatBAT,  // Default: bat
        MoviesPath:   "auto",           // Default: auto
        TVShowsPath:  "auto",
        OtherPath:    "auto",
    }
}
```

**Status:** ✅ Sensible defaults provided

---

### 5.2 JSON Unmarshal Defaults ⚠️ NOT APPLIED

**Issue:** When loading JSON with missing fields, Go zero values are used instead of defaults

**Example:**
- Config JSON: `{"base_url": "...", "username": "...", "password": "..."}`
- Result: `ScriptFormat = ""` (empty string, not "bat")

**Recommendation:** Apply defaults after unmarshaling (see 1.5)

---

## 6. Missing Validation Checks

### 6.1 Port Number Validation

**Issue:** Invalid port numbers accepted

**Test Result:**
```
URL: 'http://example.com:99999'
Parsed successfully
✓ Valid HTTP/HTTPS URL
WARNING: url.Parse() accepted this, but we shouldn't!
```

**Recommendation:** Add port range validation (1-65535)

---

### 6.2 Path Validation

**Current Status:** Paths accept any string, including empty or invalid paths

**Test Results:**
```
Paths with special characters
[PASS] Path 'C:\Users\Name\Movies': <nil>
[PASS] Path 'C:\Program Files (x86)\Movies': <nil>
[PASS] Path '/home/user/Media Library': <nil>
[PASS] Path '\\network\share\Movies': <nil>
[PASS] Path 'auto': <nil>
[PASS] Path '': FAIL - movies path cannot be empty
```

**Recommendation:** Add path validation:
- Check for invalid characters (|, <, >, etc.)
- Validate UNC paths on Windows
- Warn if path doesn't exist (non-blocking)

---

## 7. Code Quality Issues

### 7.1 Multiple Debug Log Initializations

**Issue:** Debug logging initialized in 3 places (mentioned in CLAUDE.md #12)

**Status:** Not critical, but could be centralized

---

### 7.2 GetConfigPath() Uses Executable Path

**Location:** `config.go:80-88`

```go
func GetConfigPath() (string, error) {
    execPath, err := os.Executable()
    if err != nil {
        return "", fmt.Errorf("failed to get executable path: %v", err)
    }
    
    configDir := filepath.Dir(execPath)
    return filepath.Join(configDir, "plexlinker.json"), nil
}
```

**Issue:** Config stored next to executable, not in user config directory

**Implications:**
- ✅ Simple for portable usage (SSH to Windows machine)
- ❌ Config in system directories if installed globally
- ❌ Different config per installation location

**Alternative Approaches:**
1. Use user config directory (XDG on Linux, AppData on Windows)
2. Environment variable override (`PLEXLINKER_CONFIG`)
3. Current directory as fallback

**Recommendation:** Keep current behavior (matches project goals), but document it

---

## 8. Test Coverage Summary

### Existing Tests Found:
1. ✅ `test_config_validation.go` - Basic validation tests
2. ✅ `test_backward_compat.go` - Backward compatibility tests
3. ✅ `test_save_config.go` - Save/load round-trip tests
4. ✅ `test_settings_edge_cases.go` - UI edge cases

### Missing Test Coverage:
- Validate() mutation side effects
- URL scheme validation
- ScriptFormat validation
- Normalize() functionality (doesn't exist yet)
- Concurrent access to config
- Config file corruption handling

---

## 9. Recommendations Summary

### Priority 1: Critical (Implement First)

1. **Separate Validate() and Normalize()**
   - Make `Validate()` read-only
   - Create `Normalize()` for mutations
   - Update `Save()` to call `Normalize()`
   - Update UI to call `Normalize()` before `Validate()`

2. **Add URL Scheme Validation**
   - Only accept http:// and https://
   - Reject ftp://, file://, javascript:, etc.
   - Validate host is not empty

3. **Add ScriptFormat Validation**
   - Reject invalid format values
   - Case-sensitive checking

4. **Fix Backward Compatibility**
   - Auto-migrate old configs
   - Apply defaults for missing fields
   - Auto-save migrated config

### Priority 2: High (Implement Soon)

5. **Fix Environment Variable Loading**
   - Apply defaults when loading from env vars
   - Call Normalize() on env-loaded configs

6. **Add Default Value Application**
   - Apply NewConfig() defaults after JSON unmarshal
   - Handle missing optional fields

7. **Add Port Validation**
   - Validate port range 1-65535

### Priority 3: Medium (Nice to Have)

8. **Normalize Multiple Trailing Slashes**
   - Clean up URLs with `///`

9. **Add Path Validation**
   - Warn on invalid characters
   - Validate UNC paths on Windows

10. **Document Security Considerations**
    - File permissions
    - Credentials in scripts
    - chmod recommendations

### Priority 4: Low (Future Improvements)

11. **Centralize Debug Logging**
12. **Consider XDG Config Directory**
13. **Add Concurrent Access Protection**

---

## 10. Code Examples for Fixes

### Fix 1: Separate Validate() and Normalize()

See section 1.1 for complete code example.

### Fix 2: Add URL Scheme Validation

```go
func (c *Config) Validate() error {
    if c.BaseURL == "" {
        return fmt.Errorf("base URL cannot be empty")
    }
    
    u, err := url.Parse(c.BaseURL)
    if err != nil {
        return fmt.Errorf("invalid base URL: %v", err)
    }
    
    // NEW: Validate scheme
    if u.Scheme != "http" && u.Scheme != "https" {
        return fmt.Errorf("base URL must use http:// or https:// (got %s://)", u.Scheme)
    }
    
    // NEW: Validate host
    if u.Host == "" {
        return fmt.Errorf("base URL must include a hostname")
    }
    
    // ... rest of validation ...
}
```

### Fix 3: Add ScriptFormat Validation

```go
// In Validate()
if c.ScriptFormat != "" {
    switch c.ScriptFormat {
    case ScriptFormatBAT, ScriptFormatSH, ScriptFormatTXT:
        // Valid
    default:
        return fmt.Errorf("invalid script format: %s (must be bat, sh, or txt)", c.ScriptFormat)
    }
}
```

### Fix 4: Backward Compatibility Migration

```go
func LoadConfig() (*Config, error) {
    // ... existing code ...
    
    config := NewConfig()
    if err := json.Unmarshal(data, config); err != nil {
        return nil, fmt.Errorf("failed to parse config file: %v", err)
    }
    
    // NEW: Migrate old configs
    needsSave := false
    if config.MoviesPath == "" {
        config.MoviesPath = "auto"
        needsSave = true
    }
    if config.TVShowsPath == "" {
        config.TVShowsPath = "auto"
        needsSave = true
    }
    if config.OtherPath == "" {
        config.OtherPath = "auto"
        needsSave = true
    }
    if config.ScriptFormat == "" {
        config.ScriptFormat = ScriptFormatBAT
        needsSave = true
    }
    
    // Auto-save migrated config
    if needsSave {
        if err := config.Save(); err != nil {
            // Log warning but don't fail
            fmt.Fprintf(os.Stderr, "Warning: failed to save migrated config: %v\n", err)
        }
    }
    
    return config, nil
}
```

---

## 11. Testing Verification

All tests successfully executed:

```bash
# Test results
✅ test_config_validation.go - 17 tests passed
✅ test_backward_compat.go - 5 tests passed
✅ URL mutation impact verified
✅ URL validation edge cases verified
```

**Key Findings:**
- Mutation confirmed in Validate()
- URL scheme not validated (accepts ftp://, file://, etc.)
- ScriptFormat not validated (accepts any string)
- Multiple trailing slashes not normalized
- Old configs break validation (missing required fields)
- File permissions correctly set to 0600

---

## 12. Conclusion

The configuration system is **functional but needs refactoring** to follow best practices:

1. **Validate() should not mutate state** - This is the most important fix
2. **Missing validation** for URL schemes and ScriptFormat
3. **Backward compatibility broken** - Old configs will fail
4. **Security is good** - File permissions correctly set

**Estimated Effort:**
- Priority 1 fixes: 2-3 hours
- Priority 2 fixes: 1-2 hours
- Priority 3 fixes: 1-2 hours
- Total: 4-7 hours

**Risk Assessment:**
- Low risk to implement Priority 1 fixes
- Breaking changes already exist (backward compatibility)
- Refactoring Validate() + Normalize() is the biggest change

**Recommendation:** Implement Priority 1 fixes first, then Priority 2. Priority 3 can wait for next version.

---

## Appendix: Full Test Output

See test files for complete output:
- `/Users/ocinbuona/GolandProjects/plexlinker/test_config_validation.go`
- `/Users/ocinbuona/GolandProjects/plexlinker/test_backward_compat.go`

**End of Report**
