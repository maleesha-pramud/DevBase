# DevBase Installation Script for Windows (PowerShell)
# Usage: iwr -useb https://raw.githubusercontent.com/maleesha-pramud/devbase/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$REPO = "maleesha-pramud/devbase"
$BINARY_NAME = "devbase.exe"
$INSTALL_DIR = "$env:LOCALAPPDATA\Programs\DevBase"

# Function to write colored output
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        "x86" { return "386" }
        default {
            Write-ColorOutput "Error: Unsupported architecture: $arch" "Red"
            exit 1
        }
    }
}

# Get latest release version
function Get-LatestVersion {
    Write-ColorOutput "Fetching latest version..." "Yellow"
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest"
        $version = $release.tag_name
        Write-ColorOutput "Latest version: $version" "Green"
        return $version
    }
    catch {
        Write-ColorOutput "Error: Could not fetch latest version" "Red"
        Write-ColorOutput $_.Exception.Message "Red"
        exit 1
    }
}

# Download and install binary
function Install-DevBase {
    param(
        [string]$Version,
        [string]$Arch
    )
    
    $downloadUrl = "https://github.com/$REPO/releases/download/$Version/devbase-windows-$Arch.zip"
    $tempDir = New-Item -ItemType Directory -Path "$env:TEMP\devbase-install-$(Get-Random)" -Force
    $zipFile = "$tempDir\devbase.zip"
    
    Write-ColorOutput "Downloading DevBase from $downloadUrl..." "Yellow"
    
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipFile -UseBasicParsing
    }
    catch {
        Write-ColorOutput "Error: Failed to download binary" "Red"
        Write-ColorOutput "URL: $downloadUrl" "Yellow"
        Remove-Item -Path $tempDir -Recurse -Force
        exit 1
    }
    
    Write-ColorOutput "Extracting archive..." "Yellow"
    Expand-Archive -Path $zipFile -DestinationPath $tempDir -Force
    
    # Create install directory if it doesn't exist
    if (-not (Test-Path $INSTALL_DIR)) {
        New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
    }
    
    Write-ColorOutput "Installing to $INSTALL_DIR..." "Yellow"
    Copy-Item -Path "$tempDir\$BINARY_NAME" -Destination "$INSTALL_DIR\$BINARY_NAME" -Force
    
    # Clean up
    Remove-Item -Path $tempDir -Recurse -Force
    
    Write-ColorOutput "✓ DevBase installed successfully!" "Green"
}

# Add to PATH
function Add-ToPath {
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    
    if ($currentPath -notlike "*$INSTALL_DIR*") {
        Write-ColorOutput "Adding DevBase to PATH..." "Yellow"
        [Environment]::SetEnvironmentVariable(
            "Path",
            "$currentPath;$INSTALL_DIR",
            "User"
        )
        Write-ColorOutput "✓ Added to PATH (restart terminal to use 'devbase' command)" "Green"
        
        # Update current session PATH
        $env:Path = "$env:Path;$INSTALL_DIR"
    }
    else {
        Write-ColorOutput "✓ DevBase already in PATH" "Green"
    }
}

# Verify installation
function Test-Installation {
    if (Test-Path "$INSTALL_DIR\$BINARY_NAME") {
        Write-ColorOutput "✓ Installation verified" "Green"
        Write-Host ""
        Write-ColorOutput "Run 'devbase' to get started!" "Green"
        Write-ColorOutput "(You may need to restart your terminal)" "Yellow"
    }
    else {
        Write-ColorOutput "Warning: Installation may have failed" "Yellow"
    }
}

# Main installation process
function Main {
    Write-ColorOutput "╔════════════════════════════════════════╗" "Green"
    Write-ColorOutput "║   DevBase Installation Script          ║" "Green"
    Write-ColorOutput "╚════════════════════════════════════════╝" "Green"
    Write-Host ""
    
    $arch = Get-Architecture
    Write-ColorOutput "Detected architecture: windows-$arch" "Green"
    
    $version = Get-LatestVersion
    Install-DevBase -Version $version -Arch $arch
    Add-ToPath
    Test-Installation
    
    Write-Host ""
    Write-ColorOutput "Installation complete! 🎉" "Green"
    Write-Host ""
    Write-ColorOutput "To uninstall, run:" "White"
    Write-ColorOutput "  Remove-Item -Path '$INSTALL_DIR' -Recurse -Force" "Yellow"
    Write-ColorOutput "  Remove-Item -Path '$env:USERPROFILE\devbase.db' (optional - removes database)" "Yellow"
    Write-ColorOutput "  Then manually remove '$INSTALL_DIR' from your PATH" "Yellow"
}

# Run main function
Main
