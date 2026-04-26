#!/usr/bin/env bash

set -euo pipefail

REPO_URL="https://github.com/belsia-dev/Self-DNS.git"
REPO_DIR="Self-DNS"
APP_DIR="ui"
APP_BUNDLE_NAME="SelfDNS Control Center.app"
MACOS_INSTALL_DIR="/Applications"
WAILS_VERSION="v2.12.0"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORK_DIR="$(pwd)"
TARGET_DIR=""
OS_NAME="$(uname -s)"
PKG_MANAGER=""
APT_UPDATED=0
OUTPUT_PATH=""

RESET=""
BOLD=""
DIM=""
RED=""
GREEN=""
YELLOW=""
BLUE=""
MAGENTA=""
CYAN=""
WHITE=""

supports_color() {
    if [[ ! -t 1 ]]; then
        return 1
    fi

    if ! command -v tput >/dev/null 2>&1; then
        return 1
    fi

    if [[ "$(tput colors 2>/dev/null || printf '0')" -lt 8 ]]; then
        return 1
    fi

    return 0
}

if supports_color; then
    RESET="$(printf '\033[0m')"
    BOLD="$(printf '\033[1m')"
    DIM="$(printf '\033[2m')"
    RED="$(printf '\033[31m')"
    GREEN="$(printf '\033[32m')"
    YELLOW="$(printf '\033[33m')"
    BLUE="$(printf '\033[34m')"
    MAGENTA="$(printf '\033[35m')"
    CYAN="$(printf '\033[36m')"
    WHITE="$(printf '\033[37m')"
fi

repeat_char() {
    local char="$1"
    local count="$2"
    local out=""
    local i=0

    while [[ "$i" -lt "$count" ]]; do
        out="${out}${char}"
        i=$((i + 1))
    done

    printf '%s' "$out"
}

print_banner() {
    local line
    line="$(repeat_char "=" 78)"

    printf '\n%s%s%s\n' "$BOLD$CYAN" "$line" "$RESET"
    printf '%s   SELF DNS INSTALLER :: CLONE + SETUP + BUILD%s\n' "$BOLD$WHITE" "$RESET"
    printf '%s   repo   : %s%s\n' "$DIM" "$REPO_URL" "$RESET"
    printf '%s   start  : %s%s\n' "$DIM" "$WORK_DIR" "$RESET"
    printf '%s%s%s\n\n' "$BOLD$CYAN" "$line" "$RESET"
}

section() {
    printf '\n%s[%s]%s %s\n' "$BOLD$BLUE" "$1" "$RESET" "$2"
}

info() {
    printf '%s[*]%s %s\n' "$CYAN" "$RESET" "$1"
}

success() {
    printf '%s[ok]%s %s\n' "$GREEN" "$RESET" "$1"
}

warn() {
    printf '%s[!]%s %s\n' "$YELLOW" "$RESET" "$1"
}

fail() {
    printf '%s[x]%s %s\n' "$RED" "$RESET" "$1" >&2
    exit 1
}

lower_text() {
    printf '%s' "$1" | tr '[:upper:]' '[:lower:]'
}

ask_yes_no() {
    local prompt="$1"
    local default="${2:-Y}"
    local suffix="[Y/n]"
    local answer=""
    local normalized=""

    if [[ "$(lower_text "$default")" == "n" ]]; then
        suffix="[y/N]"
    fi

    while true; do
        printf '%s?%s %s %s ' "$BOLD$MAGENTA" "$RESET" "$prompt" "$suffix"
        IFS= read -r answer
        answer="${answer:-$default}"
        normalized="$(lower_text "$answer")"

        case "$normalized" in
            y|yes)
                return 0
                ;;
            n|no)
                return 1
                ;;
            *)
                warn "Please answer y or n."
                ;;
        esac
    done
}

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

run_elevated() {
    if [[ "$(id -u)" -eq 0 ]]; then
        "$@"
    else
        sudo "$@"
    fi
}

add_go_bin_to_path() {
    export PATH="/usr/local/go/bin:/opt/homebrew/bin:$HOME/go/bin:$PATH"

    if command_exists go; then
        local gopath
        gopath="$(go env GOPATH 2>/dev/null || printf '%s/go' "$HOME")"
        export PATH="$gopath/bin:$PATH"
    fi

    hash -r
}

ensure_homebrew() {
    if command_exists brew; then
        return 0
    fi

    warn "Homebrew is required to install missing packages on macOS."

    if ! ask_yes_no "Install Homebrew now?" "Y"; then
        fail "Cannot continue on macOS without Homebrew for dependency installation."
    fi

    if ! command_exists curl; then
        fail "curl is required to install Homebrew automatically."
    fi

    NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

    if [[ -x /opt/homebrew/bin/brew ]]; then
        eval "$(/opt/homebrew/bin/brew shellenv)"
    elif [[ -x /usr/local/bin/brew ]]; then
        eval "$(/usr/local/bin/brew shellenv)"
    fi

    command_exists brew || fail "Homebrew installation did not expose the brew command."
}

detect_package_manager() {
    if [[ -n "$PKG_MANAGER" ]]; then
        return 0
    fi

    case "$OS_NAME" in
        Darwin)
            ensure_homebrew
            PKG_MANAGER="brew"
            ;;
        Linux)
            if command_exists apt-get; then
                PKG_MANAGER="apt-get"
            elif command_exists dnf; then
                PKG_MANAGER="dnf"
            elif command_exists yum; then
                PKG_MANAGER="yum"
            elif command_exists pacman; then
                PKG_MANAGER="pacman"
            elif command_exists zypper; then
                PKG_MANAGER="zypper"
            elif command_exists apk; then
                PKG_MANAGER="apk"
            else
                fail "No supported package manager was found. Install dependencies manually and rerun."
            fi
            ;;
        *)
            fail "Unsupported OS: $OS_NAME"
            ;;
    esac
}

install_system_packages() {
    detect_package_manager

    case "$PKG_MANAGER" in
        brew)
            brew install "$@"
            ;;
        apt-get)
            if [[ "$APT_UPDATED" -eq 0 ]]; then
                run_elevated apt-get update
                APT_UPDATED=1
            fi
            run_elevated apt-get install -y "$@"
            ;;
        dnf)
            run_elevated dnf install -y "$@"
            ;;
        yum)
            run_elevated yum install -y "$@"
            ;;
        pacman)
            run_elevated pacman -Sy --noconfirm "$@"
            ;;
        zypper)
            run_elevated zypper install -y "$@"
            ;;
        apk)
            run_elevated apk add "$@"
            ;;
        *)
            fail "Unsupported package manager: $PKG_MANAGER"
            ;;
    esac
}

ensure_git() {
    section "02" "Checking git"

    if command_exists git; then
        success "git is ready: $(git --version)"
        return 0
    fi

    warn "git was not found."

    if ! ask_yes_no "Install git now?" "Y"; then
        fail "git is required to clone the repository."
    fi

    case "$OS_NAME" in
        Darwin)
            install_system_packages git
            ;;
        Linux)
            install_system_packages git
            ;;
    esac

    command_exists git || fail "git installation finished, but git is still not available."
    success "git installed successfully."
}

ensure_go() {
    section "03" "Checking Go"
    add_go_bin_to_path

    if command_exists go; then
        success "Go is ready: $(go version)"
        return 0
    fi

    warn "Go was not found."

    if ! ask_yes_no "Install Go now?" "Y"; then
        fail "Go is required to install Wails and build the app."
    fi

    case "$PKG_MANAGER" in
        brew)
            install_system_packages go
            ;;
        apt-get)
            install_system_packages golang-go
            ;;
        dnf|yum|pacman|zypper|apk)
            install_system_packages golang
            ;;
        *)
            detect_package_manager
            ensure_go
            return 0
            ;;
    esac

    add_go_bin_to_path
    command_exists go || fail "Go installation finished, but go is still not available."
    success "Go installed successfully."
}

ensure_node() {
    section "04" "Checking Node.js and npm"

    if command_exists node && command_exists npm; then
        success "Node.js is ready: $(node --version), npm $(npm --version)"
        return 0
    fi

    warn "Node.js or npm was not found. Wails builds require both."

    if ! ask_yes_no "Install Node.js now?" "Y"; then
        fail "Node.js and npm are required for the frontend build."
    fi

    case "$PKG_MANAGER" in
        brew)
            install_system_packages node
            ;;
        apt-get)
            install_system_packages nodejs npm
            ;;
        dnf|yum)
            install_system_packages nodejs npm
            ;;
        pacman)
            install_system_packages nodejs npm
            ;;
        zypper)
            install_system_packages nodejs npm
            ;;
        apk)
            install_system_packages nodejs npm
            ;;
        *)
            detect_package_manager
            ensure_node
            return 0
            ;;
    esac

    command_exists node || fail "Node.js installation finished, but node is still not available."
    command_exists npm || fail "npm installation finished, but npm is still not available."
    success "Node.js installed successfully."
}

ensure_wails() {
    section "05" "Checking Wails CLI"
    add_go_bin_to_path

    if command_exists wails; then
        success "Wails is ready: $(wails version 2>/dev/null | head -n 1 || printf 'installed')"
        return 0
    fi

    warn "Wails CLI was not found."

    if ! ask_yes_no "Install Wails CLI $WAILS_VERSION now?" "Y"; then
        fail "Wails CLI is required to build the desktop app."
    fi

    go install "github.com/wailsapp/wails/v2/cmd/wails@${WAILS_VERSION}"
    add_go_bin_to_path

    command_exists wails || fail "Wails installation finished, but wails is still not available."
    success "Wails installed successfully."
}

current_checkout_available() {
    [[ -f "$SCRIPT_DIR/ui/wails.json" && -f "$SCRIPT_DIR/server/go.mod" ]]
}

choose_target_dir() {
    section "01" "Choosing project directory"

    if current_checkout_available && ask_yes_no "A Self-DNS checkout is already open at $SCRIPT_DIR. Build this checkout instead of cloning again?" "Y"; then
        TARGET_DIR="$SCRIPT_DIR"
        success "Using the current checkout."
        return 0
    fi

    TARGET_DIR="$WORK_DIR/$REPO_DIR"

    if [[ -d "$TARGET_DIR/.git" ]]; then
        warn "An existing Self-DNS clone was found at $TARGET_DIR."
        if ask_yes_no "Reuse it and pull the latest changes?" "Y"; then
            ensure_git
            git -C "$TARGET_DIR" pull --ff-only
            success "Repository updated."
            return 0
        fi
        fail "Aborted to avoid overwriting the existing directory."
    fi

    if [[ -e "$TARGET_DIR" ]]; then
        fail "Path already exists and is not a clean git checkout: $TARGET_DIR"
    fi

    info "Cloning into $TARGET_DIR"
    ensure_git
    git clone "$REPO_URL" "$TARGET_DIR"
    success "Repository cloned."
}

build_project() {
    section "06" "Building Self DNS"

    if [[ "$OS_NAME" == "Linux" ]]; then
        warn "Linux builds may require extra WebKit/GTK system libraries beyond the core tooling installed here."
    fi

    [[ -d "$TARGET_DIR/$APP_DIR" ]] || fail "Build directory was not found: $TARGET_DIR/$APP_DIR"

    (
        cd "$TARGET_DIR/$APP_DIR"
        wails build -clean
    )

    success "Build completed."
}

install_macos_app() {
    local built_app
    local installed_app

    if [[ "$OS_NAME" != "Darwin" ]]; then
        OUTPUT_PATH="$TARGET_DIR/$APP_DIR/build/bin"
        return 0
    fi

    section "07" "Installing app bundle"

    built_app="$TARGET_DIR/$APP_DIR/build/bin/$APP_BUNDLE_NAME"
    installed_app="$MACOS_INSTALL_DIR/$APP_BUNDLE_NAME"

    [[ -d "$built_app" ]] || fail "Built app bundle was not found: $built_app"

    if [[ -e "$installed_app" ]]; then
        warn "An existing app bundle was found at $installed_app."
        if ! ask_yes_no "Replace the existing app in $MACOS_INSTALL_DIR?" "Y"; then
            warn "Skipping move to $MACOS_INSTALL_DIR. The built app remains at $built_app"
            OUTPUT_PATH="$built_app"
            return 0
        fi

        run_elevated rm -rf "$installed_app"
    fi

    run_elevated mv "$built_app" "$MACOS_INSTALL_DIR/"
    OUTPUT_PATH="$installed_app"
    success "App installed to $installed_app"
}

print_summary() {
    local line
    line="$(repeat_char "-" 78)"

    printf '\n%s%s%s\n' "$BOLD$GREEN" "$line" "$RESET"
    printf '%sBuild finished successfully.%s\n' "$BOLD$WHITE" "$RESET"
    printf '%sProject : %s%s\n' "$DIM" "$TARGET_DIR" "$RESET"
    printf '%sOutput  : %s%s\n' "$DIM" "$OUTPUT_PATH" "$RESET"
    printf '%s%s%s\n\n' "$BOLD$GREEN" "$line" "$RESET"
}

main() {
    print_banner
    choose_target_dir
    ensure_go
    ensure_node
    ensure_wails
    build_project
    install_macos_app
    print_summary
}

main "$@"
