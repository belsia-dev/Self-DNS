$ErrorActionPreference = "Stop"

$RepoUrl = "https://github.com/belsia-dev/Self-DNS.git"
$RepoDir = "Self-DNS"
$AppDir = "ui-windows"
$WailsVersion = "v2.12.0"
$ScriptRoot = $PSScriptRoot
$WorkDir = (Get-Location).Path
$TargetDir = $null

function Write-Banner {
    $line = "=" * 78
    Write-Host ""
    Write-Host $line -ForegroundColor Cyan
    Write-Host "   SELF DNS WINDOWS INSTALLER :: CLONE + SETUP + BUILD" -ForegroundColor White
    Write-Host "   repo   : $RepoUrl" -ForegroundColor DarkGray
    Write-Host "   start  : $WorkDir" -ForegroundColor DarkGray
    Write-Host $line -ForegroundColor Cyan
    Write-Host ""
}

function Write-Section {
    param(
        [string]$Number,
        [string]$Title
    )

    Write-Host ""
    Write-Host "[$Number] $Title" -ForegroundColor Blue
}

function Write-Info {
    param([string]$Message)
    Write-Host "[*] $Message" -ForegroundColor Cyan
}

function Write-Ok {
    param([string]$Message)
    Write-Host "[ok] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[!] $Message" -ForegroundColor Yellow
}

function Fail-Step {
    param([string]$Message)
    Write-Host "[x] $Message" -ForegroundColor Red
    exit 1
}

function Ask-YesNo {
    param(
        [string]$Prompt,
        [bool]$DefaultYes = $true
    )

    $suffix = if ($DefaultYes) { "[Y/n]" } else { "[y/N]" }

    while ($true) {
        $answer = Read-Host "? $Prompt $suffix"

        if ([string]::IsNullOrWhiteSpace($answer)) {
            return $DefaultYes
        }

        switch ($answer.Trim().ToLowerInvariant()) {
            "y" { return $true }
            "yes" { return $true }
            "n" { return $false }
            "no" { return $false }
            default { Write-Warn "Please answer y or n." }
        }
    }
}

function Command-Exists {
    param([string]$Name)
    return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

function Refresh-Path {
    $machinePath = [System.Environment]::GetEnvironmentVariable("Path", "Machine")
    $userPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
    $env:Path = "$machinePath;$userPath"
    Add-GoBinToPath
}

function Add-GoBinToPath {
    $commonGoBin = "C:\Program Files\Go\bin"
    if ($env:Path -notlike "*$commonGoBin*") {
        $env:Path = "$commonGoBin;$env:Path"
    }

    if (Command-Exists "go") {
        $gopath = (& go env GOPATH).Trim()
        if ([string]::IsNullOrWhiteSpace($gopath)) {
            $gopath = Join-Path $HOME "go"
        }

        $goBin = Join-Path $gopath "bin"
        if ($env:Path -notlike "*$goBin*") {
            $env:Path = "$goBin;$env:Path"
        }
    }
}

function Install-WithPackageManager {
    param(
        [string]$WingetId,
        [string]$ChocoName,
        [string]$ScoopName,
        [string]$Label
    )

    if (Command-Exists "winget") {
        Write-Info "Installing $Label with winget"
        & winget install --id $WingetId -e --accept-package-agreements --accept-source-agreements
        return
    }

    if (Command-Exists "choco") {
        Write-Info "Installing $Label with Chocolatey"
        & choco install $ChocoName -y
        return
    }

    if (Command-Exists "scoop") {
        Write-Info "Installing $Label with Scoop"
        & scoop install $ScoopName
        return
    }

    Fail-Step "winget, Chocolatey, or Scoop is required to install $Label automatically."
}

function Ensure-Git {
    Write-Section "02" "Checking git"

    if (Command-Exists "git") {
        Write-Ok "git is ready: $((& git --version).Trim())"
        return
    }

    Write-Warn "git was not found."

    if (-not (Ask-YesNo "Install git now?")) {
        Fail-Step "git is required to clone the repository."
    }

    Install-WithPackageManager -WingetId "Git.Git" -ChocoName "git" -ScoopName "git" -Label "git"
    Refresh-Path

    if (-not (Command-Exists "git")) {
        Fail-Step "git installation finished, but git is still not available."
    }

    Write-Ok "git installed successfully."
}

function Ensure-Go {
    Write-Section "03" "Checking Go"
    Refresh-Path

    if (Command-Exists "go") {
        Write-Ok "Go is ready: $((& go version).Trim())"
        return
    }

    Write-Warn "Go was not found."

    if (-not (Ask-YesNo "Install Go now?")) {
        Fail-Step "Go is required to install Wails and build the app."
    }

    Install-WithPackageManager -WingetId "GoLang.Go" -ChocoName "golang" -ScoopName "go" -Label "Go"
    Refresh-Path

    if (-not (Command-Exists "go")) {
        Fail-Step "Go installation finished, but go is still not available."
    }

    Write-Ok "Go installed successfully."
}

function Ensure-Node {
    Write-Section "04" "Checking Node.js and npm"

    if ((Command-Exists "node") -and (Command-Exists "npm")) {
        $nodeVersion = (& node --version).Trim()
        $npmVersion = (& npm --version).Trim()
        Write-Ok "Node.js is ready: $nodeVersion, npm $npmVersion"
        return
    }

    Write-Warn "Node.js or npm was not found. Wails builds require both."

    if (-not (Ask-YesNo "Install Node.js now?")) {
        Fail-Step "Node.js and npm are required for the frontend build."
    }

    Install-WithPackageManager -WingetId "OpenJS.NodeJS.LTS" -ChocoName "nodejs-lts" -ScoopName "nodejs-lts" -Label "Node.js"
    Refresh-Path

    if (-not (Command-Exists "node")) {
        Fail-Step "Node.js installation finished, but node is still not available."
    }

    if (-not (Command-Exists "npm")) {
        Fail-Step "npm installation finished, but npm is still not available."
    }

    Write-Ok "Node.js installed successfully."
}

function Ensure-Wails {
    Write-Section "05" "Checking Wails CLI"
    Refresh-Path

    if (Command-Exists "wails") {
        try {
            $wailsVersion = (& wails version 2>$null | Select-Object -First 1).Trim()
        } catch {
            $wailsVersion = "installed"
        }
        Write-Ok "Wails is ready: $wailsVersion"
        return
    }

    Write-Warn "Wails CLI was not found."

    if (-not (Ask-YesNo "Install Wails CLI $WailsVersion now?")) {
        Fail-Step "Wails CLI is required to build the desktop app."
    }

    & go install "github.com/wailsapp/wails/v2/cmd/wails@$WailsVersion"
    Refresh-Path

    if (-not (Command-Exists "wails")) {
        Fail-Step "Wails installation finished, but wails is still not available."
    }

    Write-Ok "Wails installed successfully."
}

function Current-CheckoutAvailable {
    return (Test-Path (Join-Path $ScriptRoot "ui-windows\wails.json")) -and (Test-Path (Join-Path $ScriptRoot "server\go.mod"))
}

function Choose-TargetDir {
    Write-Section "01" "Choosing project directory"

    if ((Current-CheckoutAvailable) -and (Ask-YesNo "A Self-DNS checkout is already open at $ScriptRoot. Build this checkout instead of cloning again?")) {
        $script:TargetDir = $ScriptRoot
        Write-Ok "Using the current checkout."
        return
    }

    $script:TargetDir = Join-Path $WorkDir $RepoDir

    if (Test-Path (Join-Path $script:TargetDir ".git")) {
        Write-Warn "An existing Self-DNS clone was found at $script:TargetDir."
        if (Ask-YesNo "Reuse it and pull the latest changes?") {
            Ensure-Git
            & git -C $script:TargetDir pull --ff-only
            Write-Ok "Repository updated."
            return
        }

        Fail-Step "Aborted to avoid overwriting the existing directory."
    }

    if (Test-Path $script:TargetDir) {
        Fail-Step "Path already exists and is not a clean git checkout: $script:TargetDir"
    }

    Write-Info "Cloning into $script:TargetDir"
    Ensure-Git
    & git clone $RepoUrl $script:TargetDir
    Write-Ok "Repository cloned."
}

function Build-Project {
    Write-Section "06" "Building Self DNS"

    $buildRoot = Join-Path $TargetDir $AppDir
    if (-not (Test-Path $buildRoot)) {
        Fail-Step "Build directory was not found: $buildRoot"
    }

    Push-Location $buildRoot
    try {
        & wails build -clean
    }
    finally {
        Pop-Location
    }

    Write-Ok "Build completed."
}

function Write-Summary {
    $line = "-" * 78
    Write-Host ""
    Write-Host $line -ForegroundColor Green
    Write-Host "Build finished successfully." -ForegroundColor White
    Write-Host "Project : $TargetDir" -ForegroundColor DarkGray
    Write-Host "Output  : $(Join-Path $TargetDir "$AppDir\build\bin")" -ForegroundColor DarkGray
    Write-Host $line -ForegroundColor Green
    Write-Host ""
}

Write-Banner
Choose-TargetDir
Ensure-Go
Ensure-Node
Ensure-Wails
Build-Project
Write-Summary
