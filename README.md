# DevBase v1.0.0

**DevBase** is a high-performance CLI project manager built with Go, featuring optimistic UI updates, concurrent directory scanning, and seamless VS Code integration.

## 🚀 Features

- **⚡ Optimistic UI Updates** - Instant visual feedback with automatic rollback on errors
- **🔍 Intelligent Project Discovery** - Automatically finds Go, Node.js, and Git repositories
- **📊 SQLite Database** - WAL mode enabled for maximum performance
- **🔄 Git Integration** - Shallow cloning for fast project restoration
- **⚙️ Concurrent Scanning** - Worker pool pattern for lightning-fast directory traversal
- **💻 VS Code Integration** - One-key project opening
- **🎨 Beautiful TUI** - Built with Bubble Tea for a modern terminal experience

## 📦 Installation

### Option 1: Use Pre-built Binary
Download `DevBase.exe` and place it in your PATH.

### Option 2: Build from Source
```bash
git clone https://github.com/example/devbase
cd devbase
go build -ldflags="-s -w" -o DevBase.exe .\cmd\devbase\main.go
```

## 🎮 Usage

### Interactive Mode (Default)
```bash
DevBase
```

### Commands
```bash
DevBase --help      # Show help information
DevBase --version   # Show version
DevBase scan        # Scan directories (interactive mode)
```

## ⌨️ Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Open project in VS Code |
| `s` | Scan for new projects |
| `c` | Clear all projects (requires confirmation) |
| `d` | Archive project (deletes directory) |
| `r` | Restore archived project (clones from repo) |
| `/` | Filter/search projects |
| `ESC` | Cancel confirmation dialogs |
| `q` or `Ctrl+C` | Quit |

## 🏗️ Architecture

### Modules

- **`models/`** - Project data structures with GORM tags
- **`db/`** - Database layer with optimized SQLite configuration
- **`engine/`** - File system operations, Git integration, and scanning
- **`ui/`** - Bubble Tea UI with optimistic updates
- **`cmd/devbase/`** - Main application entry point

### Key Technologies

- **GORM** - ORM with SQLite driver (modernc.org/sqlite - pure Go, no CGO)
- **Bubble Tea** - Terminal UI framework
- **go-git** - Git operations in Go
- **SQLite WAL Mode** - Write-Ahead Logging for better concurrency

## 🔧 Performance Optimizations

1. **SQLite Configuration**
   - WAL (Write-Ahead Logging) mode enabled
   - `PRAGMA synchronous = NORMAL`
   - Prepared statement caching
   - Max 1 open connection (prevents SQLite locking)

2. **Directory Scanning**
   - 10 concurrent worker goroutines
   - Ignores heavy directories: `node_modules`, `dist`, `build`, `vendor`
   - Buffered channels for throughput

3. **Git Operations**
   - Shallow cloning with `Depth: 1`
   - Only downloads latest commit (saves bandwidth)

4. **UI Updates**
   - Optimistic updates for instant feedback
   - Background operations with automatic rollback
   - Non-blocking VS Code launching

## 📋 Requirements

- **VS Code** - Must be installed with `code` command in PATH
- **Git** - Required for restore functionality (cloning repositories)

### Installing VS Code CLI
If `code` command is not available:
1. Open VS Code
2. Press `Ctrl+Shift+P` (Command Palette)
3. Type: "Shell Command: Install 'code' command in PATH"
4. Select and run

## 📁 Database

DevBase stores all project data in `devbase.db` (SQLite) in the current directory.

### Project Schema
- **ID** - Unique identifier
- **Name** - Project name (derived from directory)
- **Path** - Full file system path
- **RepoURL** - Git repository URL
- **Status** - `active` or `archived`
- **LastOpened** - Timestamp (used for sorting)
- **Tags** - String array for categorization
- **CreatedAt** / **UpdatedAt** - Automatic timestamps

## 🎯 How It Works

### Optimistic UI Pattern

**Archive Operation (Press 'd'):**
1. UI immediately shows `[Archived]` status
2. Background: `engine.ArchiveProject()` deletes directory and updates DB
3. Success: No change needed (already displayed)
4. Failure: UI reverts to original state, error displayed

**Restore Operation (Press 'r'):**
1. UI immediately shows `[Processing...]`
2. Background: `engine.RestoreProject()` clones repo and updates DB
3. Success: Status changes to `[Active]`
4. Failure: UI reverts to original state, error displayed

### Scanning Process

1. Press `s` to initiate scan
2. Worker pool (10 goroutines) activated
3. Main thread walks directory tree, sends paths to workers
4. Workers check for project markers: `package.json`, `go.mod`, `.git`
5. Results collected and deduplicated
6. New projects added to database
7. UI automatically reloads with updated list

## 🐛 Troubleshooting

**VS Code won't open:**
- Ensure VS Code is installed
- Verify `code` command is in PATH: `code --version`

**Scan is slow:**
- Large directories with many subdirectories take time
- Heavy folders are automatically skipped

**Database locked:**
- Only one DevBase instance should run at a time
- WAL mode minimizes locking issues

## 📊 Project Structure

```
DevBase/
├── cmd/
│   └── devbase/
│       └── main.go          # Application entry point
├── db/
│   └── db.go                # Database operations
├── engine/
│   ├── ops.go               # Archive/restore operations
│   └── scanner.go           # Concurrent directory scanner
├── models/
│   └── project.go           # Project data model
├── ui/
│   └── main_view.go         # Bubble Tea UI with optimistic updates
├── devbase.db               # SQLite database (created on first run)
├── DevBase.exe              # Production executable
└── README.md
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

MIT License - See LICENSE file for details

## 🙏 Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [GORM](https://gorm.io/) - ORM library
- [go-git](https://github.com/go-git/go-git) - Git implementation in Go
- [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) - Pure Go SQLite driver

---

**DevBase v1.0.0** - Built with ❤️ in Go
