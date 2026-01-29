# DevBase Installation Guide

This guide covers all available methods to install and uninstall DevBase on Windows, Linux, and macOS.

---

## 📥 Installation Methods

### Windows

#### ✅ **Method 1: Windows Installer (RECOMMENDED)**

**Download:** [devbase-windows-installer.exe](https://github.com/maleesha-pramud/devbase/releases/latest)

**Installation Steps:**
1. Download `devbase-windows-installer.exe` from the latest release
2. Double-click the installer
3. Follow the installation wizard
4. ✅ Check "Add to PATH" option (enabled by default)
5. Click Install
6. Restart your terminal/PowerShell
7. Run `devbase` command from anywhere

**Benefits:**
- ✅ Automatic PATH configuration
- ✅ Easy uninstallation via Windows Settings
- ✅ No manual configuration needed
- ✅ Professional installation experience
- ✅ Desktop shortcut option

---

#### Method 2: PowerShell Installation Script

**Command:**
```powershell
iwr -useb https://raw.githubusercontent.com/maleesha-pramud/devbase/main/install.ps1 | iex
```

**Installation Steps:**
1. Open PowerShell (as Administrator recommended)
2. Run the command above
3. Script will automatically:
   - Detect your system architecture
   - Download the latest version
   - Install to `%LOCALAPPDATA%\Programs\DevBase`
   - Add to PATH
4. Restart your terminal
5. Run `devbase` command

**Benefits:**
- ✅ One-line installation
- ✅ Automatic updates available
- ✅ Cross-version compatible

---

#### Method 3: Manual Installation from ZIP

**Download:** `devbase-windows-amd64.zip` (or `devbase-windows-arm64.zip`)

**Installation Steps:**
1. Download the appropriate ZIP file from [releases](https://github.com/maleesha-pramud/devbase/releases/latest)
2. Extract `devbase.exe` to a folder (e.g., `C:\Program Files\DevBase\`)
3. Add the folder to System PATH:
   - Press `Win + X` → System → Advanced system settings
   - Click "Environment Variables"
   - Under "User variables", select "Path" → Edit
   - Click "New" and add your folder path
   - Click OK on all dialogs
4. Restart terminal
5. Run `devbase` command

**When to use:**
- Corporate environments with restricted script execution
- Need full control over installation location
- Air-gapped systems

---

#### Method 4: Install with Go

**Prerequisites:** Go 1.21 or later

**Command:**
```bash
go install github.com/maleesha-pramud/devbase/cmd/devbase@latest
```

**Installation Steps:**
1. Ensure Go is installed and `%GOPATH%\bin` is in PATH
2. Run the command above
3. Binary will be installed to `%GOPATH%\bin\devbase.exe`
4. Run `devbase` command

**When to use:**
- You're a Go developer
- Want to build from source
- Need the absolute latest code

---

#### Method 5: Build from Source

**Prerequisites:** Git and Go 1.21 or later

**Commands:**
```bash
git clone https://github.com/maleesha-pramud/devbase.git
cd devbase
go build -o devbase.exe ./cmd/devbase
# Move devbase.exe to a folder in your PATH
```

**When to use:**
- Development purposes
- Contributing to the project
- Custom modifications needed

---

### Linux / macOS

#### ✅ **Method 1: Installation Script (RECOMMENDED)**

**Command:**
```bash
curl -fsSL https://raw.githubusercontent.com/maleesha-pramud/devbase/main/install.sh | bash
```

**Installation Steps:**
1. Open Terminal
2. Run the command above
3. Script will automatically:
   - Detect your OS and architecture
   - Download the latest version
   - Install to `/usr/local/bin/devbase`
   - Request sudo password if needed
4. Run `devbase` command immediately

**Benefits:**
- ✅ One-line installation
- ✅ Automatic platform detection
- ✅ No manual PATH configuration
- ✅ Handles permissions automatically

---

#### Method 2: Manual Installation from TAR.GZ

**Download:**
- Linux: `devbase-linux-amd64.tar.gz` (or arm64/armv7)
- macOS: `devbase-darwin-amd64.tar.gz` (Intel) or `devbase-darwin-arm64.tar.gz` (Apple Silicon)

**Installation Steps:**
```bash
# Download the appropriate file
wget https://github.com/maleesha-pramud/devbase/releases/latest/download/devbase-linux-amd64.tar.gz

# Extract
tar -xzf devbase-*.tar.gz

# Move to PATH
sudo mv devbase /usr/local/bin/

# Make executable (if needed)
sudo chmod +x /usr/local/bin/devbase

# Verify installation
devbase --version
```

**When to use:**
- Need specific version
- Air-gapped systems
- Custom installation location

---

#### Method 3: Install with Go

**Prerequisites:** Go 1.21 or later

**Command:**
```bash
go install github.com/maleesha-pramud/devbase/cmd/devbase@latest
```

**Installation Steps:**
1. Ensure `$GOPATH/bin` is in your PATH
2. Run the command above
3. Binary installed to `$GOPATH/bin/devbase`
4. Run `devbase` command

---

#### Method 4: Build from Source

**Prerequisites:** Git and Go 1.21 or later

**Commands:**
```bash
git clone https://github.com/maleesha-pramud/devbase.git
cd devbase
go build -o devbase ./cmd/devbase
sudo mv devbase /usr/local/bin/
```

---

## 🗑️ Uninstallation Methods

### Windows

#### ✅ **Method 1: Uninstall via Windows Settings (If installed with installer)**

**Steps:**
1. Open Windows Settings (`Win + I`)
2. Go to **Apps** → **Installed apps** (Windows 11) or **Apps & features** (Windows 10)
3. Search for "DevBase"
4. Click the three dots (...) → **Uninstall**
5. Follow the uninstallation wizard

**What gets removed:**
- ✅ Binary files
- ✅ PATH entries (automatic)
- ✅ Registry entries
- ✅ Start Menu shortcuts
- ❌ Database file (optional - preserved by default)

---

#### Method 2: Uninstall via Control Panel

**Steps:**
1. Open Control Panel
2. **Programs and Features**
3. Find "DevBase" in the list
4. Right-click → **Uninstall**
5. Follow the wizard

---

#### Method 3: Run Installer Again (If installed with installer)

**Steps:**
1. Download and run `devbase-windows-installer.exe` again
2. Installer will detect existing installation
3. Select "Uninstall" or "Modify" option
4. Follow prompts

---

#### Method 4: Manual Uninstall (If installed via script or manually)

**Commands:**
```powershell
# Remove binary
Remove-Item -Path "$env:LOCALAPPDATA\Programs\DevBase" -Recurse -Force -ErrorAction SilentlyContinue

# Or if in different location
Remove-Item -Path "C:\Program Files\DevBase" -Recurse -Force -ErrorAction SilentlyContinue

# Manually remove from PATH:
# Settings → System → About → Advanced system settings → Environment Variables
# Find PATH variable, remove DevBase folder path
```

**Remove database (optional):**
```powershell
Remove-Item -Path "$env:USERPROFILE\devbase.db" -ErrorAction SilentlyContinue
```

---

#### Method 5: Uninstall Go Installation

**Commands:**
```powershell
# Windows
Remove-Item -Path "$env:GOPATH\bin\devbase.exe"

# Remove database (optional)
Remove-Item -Path "$env:USERPROFILE\devbase.db"
```

---

### Linux / macOS

#### ✅ **Method 1: Simple Remove (RECOMMENDED)**

**Commands:**
```bash
# Remove binary
sudo rm /usr/local/bin/devbase

# Remove database (optional)
rm ~/devbase.db
```

**That's it!** No PATH cleanup needed.

---

#### Method 2: Uninstall Go Installation

**Commands:**
```bash
# Remove binary
rm $(which devbase)

# Or if not in PATH
rm $GOPATH/bin/devbase

# Remove database (optional)
rm ~/devbase.db
```

---

#### Method 3: Verify Complete Removal

**Commands:**
```bash
# Check if binary still exists
which devbase

# Check if database exists
ls -la ~/devbase.db

# If nothing found, uninstallation complete!
```

---

## 📊 Installation Method Comparison

| Method | Windows | Linux/macOS | Auto-PATH | Easy Uninstall | Recommended |
|--------|---------|-------------|-----------|----------------|-------------|
| **Installer/Script** | ✅ | ✅ | ✅ | ✅ | **⭐ YES** |
| PowerShell Script | ✅ | ❌ | ✅ | ⚠️ Manual | Good |
| Bash Script | ❌ | ✅ | ✅ | ⚠️ Manual | Good |
| Manual ZIP/TAR | ✅ | ✅ | ❌ | ❌ | When needed |
| Go Install | ✅ | ✅ | ⚠️ If configured | ⚠️ Manual | Developers |
| Build from Source | ✅ | ✅ | ❌ | ❌ | Contributors |

---

## 🔍 Troubleshooting

### `devbase: command not found` (Linux/macOS)

**Solution:**
```bash
# Check if installed
which devbase

# If not found, ensure /usr/local/bin is in PATH
echo $PATH | grep /usr/local/bin

# Add to PATH if missing (add to ~/.bashrc or ~/.zshrc)
export PATH="/usr/local/bin:$PATH"
source ~/.bashrc  # or ~/.zshrc
```

---

### `devbase: The term 'devbase' is not recognized` (Windows)

**Solution:**
```powershell
# Check if PATH is set
$env:Path -split ';' | Select-String -Pattern 'DevBase'

# Restart terminal to apply PATH changes

# Or add manually to current session
$env:Path += ";$env:LOCALAPPDATA\Programs\DevBase"
```

---

### Database File Location

**Default locations:**
- **Windows:** `%USERPROFILE%\devbase.db` (e.g., `C:\Users\YourName\devbase.db`)
- **Linux/macOS:** `~/devbase.db` (e.g., `/home/username/devbase.db`)

**To backup:**
```bash
# Linux/macOS
cp ~/devbase.db ~/devbase.db.backup

# Windows
Copy-Item "$env:USERPROFILE\devbase.db" "$env:USERPROFILE\devbase.db.backup"
```

---

## 📝 Notes

- **Recommended installations** are highlighted with ✅
- Database file (`devbase.db`) is **never automatically deleted** during uninstallation to prevent data loss
- After installation, you may need to **restart your terminal** for PATH changes to take effect
- For updates, install over the existing installation (installer will handle upgrades)
- All installation methods are compatible with DevBase v1.7.15+

---

## 🆘 Need Help?

- **Issues:** [GitHub Issues](https://github.com/maleesha-pramud/devbase/issues)
- **Documentation:** [README.md](https://github.com/maleesha-pramud/devbase/blob/main/README.md)
- **Releases:** [GitHub Releases](https://github.com/maleesha-pramud/devbase/releases)
