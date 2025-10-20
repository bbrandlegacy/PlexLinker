# PlexLinker Use Cases

This document outlines various use cases for PlexLinker beyond its primary purpose as a media server download tool.

---

## 🎬 Primary Use Case: Media Server Management

### Personal Media Library Organization
**Target Users:** Plex/Jellyfin/Emby server administrators

**Scenario:**
- You have a remote Windows server running Plex
- Media files are hosted on an authenticated web server
- You SSH into the server to queue downloads
- You want downloads to continue after SSH disconnect

**PlexLinker Solution:**
1. SSH into Windows server
2. Browse authenticated directory with PlexLinker
3. Select TV shows and movies
4. Auto-categorization separates content types
5. Generate scripts via Task Scheduler
6. Disconnect SSH - downloads continue
7. Files automatically organized by category

**Benefits:**
- ✅ SSH-safe execution (Task Scheduler)
- ✅ Automatic TV show episode ordering
- ✅ Separate destination folders per media type
- ✅ Resume support via aria2c
- ✅ Plex-ready file organization

---

## 📦 Alternative Use Cases

### 1. **Web Archive & Digital Preservation**

**Target Users:** Digital archivists, researchers, librarians

**Scenario:**
Archive entire authenticated web directories for preservation, including:
- Historical websites before takedown
- Academic paper repositories
- Government data archives
- Corporate knowledge bases

**PlexLinker Features:**
- Browse entire directory trees interactively
- Select specific files or entire directories
- Generate reproducible download scripts
- Preserve directory structure
- Sequential downloads respect rate limits

**Example:**
```bash
# Archive a historical dataset
./plexlinker
# Browse to https://archive.example.org/dataset/
# Select all subdirectories (key 2)
# Generate script (key 6 - no execution)
# Run script later during off-peak hours
```

**Benefits:**
- Non-destructive browsing before downloading
- Audit trail (generated scripts)
- Bandwidth-friendly sequential downloads
- Resume capability for interrupted transfers

---

### 2. **Scientific Data Collection**

**Target Users:** Researchers, data scientists, university labs

**Scenario:**
Download large datasets from authenticated research repositories:
- Genomics data (NCBI, EBI)
- Climate models (NOAA, NASA)
- Astronomical surveys (SDSS, ESO)
- Social science datasets (ICPSR, Dataverse)

**PlexLinker Features:**
- Navigate complex directory hierarchies
- Filter by file extensions (CSV, FITS, HDF5)
- Batch download with progress tracking
- SSH-persistent for long-running transfers
- Cross-platform (Linux servers, macOS workstations)

**Example Workflow:**
```bash
# Configure for research repository
{
  "base_url": "https://data.research.edu/datasets/",
  "username": "researcher_id",
  "movies_path": "/data/raw/experiment_2024/",
  "tv_shows_path": "/data/processed/timeseries/",
  "other_path": "/data/misc/"
}

# Categories become:
# "Movies" → Raw data files
# "TV Shows" → Time-series data (sequential)
# "Other" → Documentation, metadata
```

**Benefits:**
- Organize by experiment/category automatically
- Sequential downloads for quota compliance
- Debug logging for reproducibility
- Script generation for methodology documentation

---

### 3. **Corporate File Distribution**

**Target Users:** IT departments, DevOps teams, system administrators

**Scenario:**
Distribute software packages, updates, or documentation:
- Internal software repositories
- Compliance documentation
- Training materials
- System images and patches

**PlexLinker Features:**
- Browse internal file servers with auth
- Generate download scripts for team distribution
- Version-specific directory navigation
- Multi-user support (username selection)
- Audit trail via generated scripts

**Example:**
```bash
# IT distributes new software package
1. IT admin browses internal repo via PlexLinker
2. Selects correct version and components
3. Generates script (key 6)
4. Shares script with team via Slack/email
5. Team runs script to download packages
6. Consistent deployments across all machines
```

**Benefits:**
- Eliminates "download the wrong version" errors
- Scripts serve as deployment documentation
- Debug mode logs for compliance audits
- Works across Windows, Linux, macOS

---

### 4. **Educational Content Management**

**Target Users:** Professors, students, e-learning platforms

**Scenario:**
Download course materials from authenticated university portals:
- Lecture recordings
- Course readings (PDFs)
- Assignment templates
- Lab datasets

**PlexLinker Features:**
- Navigate course management systems
- Auto-categorization by content type
- Batch download entire courses
- Share download scripts with classmates

**Example:**
```bash
# Student downloads entire semester
{
  "base_url": "https://lms.university.edu/courses/CS101/",
  "movies_path": "~/Desktop/CS101/Lectures/",
  "tv_shows_path": "~/Desktop/CS101/Labs/",  # Sequential lab sessions
  "other_path": "~/Desktop/CS101/Readings/"
}
```

**Benefits:**
- Offline access to all materials
- Sequential lab downloads (Lab 1, Lab 2, etc.)
- One-time setup, semester-long use
- Cross-platform (student laptops, lab computers)

---

### 5. **Backup & Disaster Recovery**

**Target Users:** System administrators, backup engineers

**Scenario:**
Selective backup of web-hosted file systems:
- Cloud storage with HTTP/WebDAV interface
- Network file shares exposed via HTTP
- Remote servers with web-based file browsers

**PlexLinker Features:**
- Browse remote filesystems interactively
- Select critical files/directories only
- Generate backup scripts with timestamps
- Incremental backups (aria2c conditional GET)

**Example:**
```bash
# Weekly backup workflow
1. Browse production file server
2. Select modified directories (sorted by date)
3. Generate timestamped script
4. Script runs via cron/Task Scheduler
5. Downloads only changed files (aria2c feature)
```

**Benefits:**
- Selective backups (not full mirrors)
- Visual confirmation before download
- SSH-safe long-running backups
- Scripts as backup manifests

---

### 6. **Open Source Mirror Management**

**Target Users:** Mirror administrators, distro maintainers

**Scenario:**
Sync specific packages from upstream repositories:
- Linux distribution packages
- Software libraries (PyPI, npm mirrors)
- Container registries
- Documentation sites

**PlexLinker Features:**
- Navigate package hierarchies
- Select specific versions/architectures
- Generate reproducible sync scripts
- Debug logging for sync audits

**Example:**
```bash
# Mirror maintainer syncs new packages
{
  "base_url": "https://upstream.repo.org/packages/",
  "movies_path": "/mirror/stable/amd64/",
  "tv_shows_path": "/mirror/rolling/amd64/",  # Sequential updates
  "other_path": "/mirror/source/"
}
```

**Benefits:**
- Selective mirroring (save bandwidth/storage)
- Version-specific downloads
- Audit trail for mirror policies
- Cross-platform mirror hosts

---

### 7. **Legal Discovery & eDiscovery**

**Target Users:** Legal teams, compliance officers, forensics experts

**Scenario:**
Collect evidence from authenticated document repositories:
- Internal document management systems
- Email archives with web interface
- Compliance document stores

**PlexLinker Features:**
- Browse document hierarchies
- Generate download manifests (scripts)
- Timestamp and audit logging
- Non-destructive collection

**Example:**
```bash
# Legal team collects case documents
1. Browse document repository (read-only)
2. Select relevant date ranges/categories
3. Generate download script (key 6)
4. Script includes embedded timestamps
5. Debug log provides audit trail
6. Chain of custody maintained
```

**Benefits:**
- Non-destructive browsing
- Audit trail (debug logs + scripts)
- Reproducible collections
- Read-only authentication support

---

### 8. **Software Release Engineering**

**Target Users:** Release managers, QA teams, build engineers

**Scenario:**
Download build artifacts from CI/CD systems:
- Jenkins/GitLab artifact servers
- Nightly build repositories
- QA test packages

**PlexLinker Features:**
- Navigate build artifact directories
- Select specific builds/versions
- Generate test deployment scripts
- Multi-user support (track who downloaded what)

**Example:**
```bash
# QA downloads release candidates
{
  "base_url": "https://builds.company.com/releases/",
  "username": "qa_engineer",  # Tracked in scripts
  "movies_path": "/qa/windows_builds/",
  "tv_shows_path": "/qa/incremental_updates/",  # Sequential patches
  "other_path": "/qa/documentation/"
}
```

**Benefits:**
- Version-specific deployments
- Username tracking for accountability
- Scripts as deployment documentation
- Cross-platform QA environments

---

## 🎯 Common Patterns Across Use Cases

### 1. **Authenticated Web Browsing**
- All use cases involve HTTP Basic Auth
- Secure credential management via config
- No browser required

### 2. **Selective Downloads**
- Visual browsing before committing
- Granular file/directory selection
- Preview before download

### 3. **Script Generation**
- Reproducible downloads
- Shareable with teams
- Audit trail / documentation

### 4. **Auto-Categorization**
- Flexible interpretation of "Movies" vs "TV Shows"
- Map to any two-category system
- Sequential ordering for time-series data

### 5. **SSH-Safe Execution**
- Critical for remote servers
- Task Scheduler (Windows) or tmux (Unix)
- Long-running operations

### 6. **Cross-Platform Support**
- Development: macOS/Linux
- Production: Often Windows servers
- User workstations: All platforms

---

## 🚀 Getting Started with Your Use Case

1. **Identify Your Needs:**
   - Authentication method (HTTP Basic Auth)
   - File organization structure
   - Sequential vs parallel downloads
   - Platform (Windows, Linux, macOS)

2. **Map to PlexLinker Concepts:**
   - "Movies" = Your primary file type
   - "TV Shows" = Sequential/ordered files
   - "Other" = Miscellaneous files

3. **Configure Paths:**
   ```json
   {
     "movies_path": "/your/primary/path",
     "tv_shows_path": "/your/sequential/path",
     "other_path": "/your/misc/path"
   }
   ```

4. **Test Workflow:**
   - Browse → Select → Generate (key 6) → Review script → Run manually
   - Once confident → Use key 5 for automatic execution

5. **Automate:**
   - Cron jobs (Linux/macOS)
   - Task Scheduler (Windows)
   - CI/CD pipelines

---

## 📧 Share Your Use Case

Found a creative use for PlexLinker? We'd love to hear about it!

- **GitHub Discussions:** Share your workflow
- **Issues:** Request features for your use case
- **Pull Requests:** Contribute improvements

---

**PlexLinker is flexible by design - if you have authenticated directories to browse and files to download, it can probably help!**
