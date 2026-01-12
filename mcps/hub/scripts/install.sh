#!/bin/sh
# Hub installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/USER/REPO/main/mcps/hub/scripts/install.sh | sh
#    or: curl -fsSL ... | sh -s -- -v v1.0.0 -d /usr/local/bin
#
# Options:
#   -v VERSION   Install specific version (default: latest)
#   -d DIR       Install directory (default: ~/.local/bin or XDG_BIN_HOME)
#   -h           Show help

set -e

# Configuration
REPO="vaayne/cc-plugins"
BINARY_NAME="hub"
RELEASES_URL="https://github.com/${REPO}/releases"

# Colors (only if terminal supports it)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

info() {
    printf "${BLUE}info${NC}: %s\n" "$1"
}

success() {
    printf "${GREEN}success${NC}: %s\n" "$1"
}

warn() {
    printf "${YELLOW}warn${NC}: %s\n" "$1" >&2
}

error() {
    printf "${RED}error${NC}: %s\n" "$1" >&2
    exit 1
}

usage() {
    cat <<EOF
Hub Installer

Usage: $0 [options]

Options:
    -v VERSION   Install specific version (default: latest)
    -d DIR       Install directory (default: ~/.local/bin or XDG_BIN_HOME)
    -h           Show this help message

Examples:
    # Install latest version
    curl -fsSL URL | sh

    # Install specific version
    curl -fsSL URL | sh -s -- -v v1.0.0

    # Install to custom directory
    curl -fsSL URL | sh -s -- -d /usr/local/bin
EOF
    exit 0
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) error "Unsupported operating system: $(uname -s)" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac
}

# Get latest version from GitHub API
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "https://api.github.com/repos/${REPO}/releases" | \
            grep -o '"tag_name": *"v[^"]*"' | \
            head -1 | \
            sed 's/.*"\(v[^"]*\)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases" | \
            grep -o '"tag_name": *"v[^"]*"' | \
            head -1 | \
            sed 's/.*"\(v[^"]*\)".*/\1/'
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

# Download file
download() {
    url="$1"
    output="$2"
    
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$output"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$url" -O "$output"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

# Verify checksum
verify_checksum() {
    archive="$1"
    checksums="$2"
    filename=$(basename "$archive")
    
    expected=$(grep "$filename" "$checksums" | awk '{print $1}')
    if [ -z "$expected" ]; then
        warn "Could not find checksum for $filename, skipping verification"
        return 0
    fi
    
    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$archive" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$archive" | awk '{print $1}')
    else
        warn "sha256sum/shasum not found, skipping checksum verification"
        return 0
    fi
    
    if [ "$expected" != "$actual" ]; then
        error "Checksum verification failed!\nExpected: $expected\nActual: $actual"
    fi
    
    info "Checksum verified"
}

# Get install directory
get_install_dir() {
    if [ -n "$INSTALL_DIR" ]; then
        echo "$INSTALL_DIR"
    elif [ -n "$XDG_BIN_HOME" ]; then
        echo "$XDG_BIN_HOME"
    else
        echo "$HOME/.local/bin"
    fi
}

# Check if directory is in PATH
check_path() {
    dir="$1"
    case ":$PATH:" in
        *":$dir:"*) return 0 ;;
        *) return 1 ;;
    esac
}

# Main installation
main() {
    VERSION=""
    INSTALL_DIR=""
    
    # Parse arguments
    while [ $# -gt 0 ]; do
        case "$1" in
            -v) VERSION="$2"; shift 2 ;;
            -d) INSTALL_DIR="$2"; shift 2 ;;
            -h|--help) usage ;;
            *) error "Unknown option: $1" ;;
        esac
    done
    
    OS=$(detect_os)
    ARCH=$(detect_arch)
    INSTALL_DIR=$(get_install_dir)
    
    info "Detected OS: $OS"
    info "Detected architecture: $ARCH"
    info "Install directory: $INSTALL_DIR"
    
    # Get version
    if [ -z "$VERSION" ]; then
        info "Fetching latest version..."
        VERSION=$(get_latest_version)
        if [ -z "$VERSION" ]; then
            error "Could not determine latest version"
        fi
    fi
    info "Version: $VERSION"
    
    # Determine archive extension
    if [ "$OS" = "windows" ]; then
        EXT="zip"
    else
        EXT="tar.gz"
    fi
    
    # Build download URLs
    VERSION_NUM="${VERSION#v}"  # Remove 'v' prefix for archive name
    ARCHIVE_NAME="${BINARY_NAME}_${VERSION_NUM}_${OS}_${ARCH}.${EXT}"
    ARCHIVE_URL="${RELEASES_URL}/download/${VERSION}/${ARCHIVE_NAME}"
    CHECKSUMS_URL="${RELEASES_URL}/download/${VERSION}/checksums.txt"
    
    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT
    
    info "Downloading $ARCHIVE_NAME..."
    download "$ARCHIVE_URL" "$TMP_DIR/$ARCHIVE_NAME"
    
    info "Downloading checksums..."
    download "$CHECKSUMS_URL" "$TMP_DIR/checksums.txt"
    
    # Verify checksum
    verify_checksum "$TMP_DIR/$ARCHIVE_NAME" "$TMP_DIR/checksums.txt"
    
    # Extract archive
    info "Extracting..."
    cd "$TMP_DIR"
    if [ "$EXT" = "zip" ]; then
        unzip -q "$ARCHIVE_NAME"
    else
        tar -xzf "$ARCHIVE_NAME"
    fi
    
    # Create install directory if needed
    mkdir -p "$INSTALL_DIR"
    
    # Install binary
    info "Installing to $INSTALL_DIR/$BINARY_NAME..."
    if [ "$OS" = "windows" ]; then
        cp "${BINARY_NAME}.exe" "$INSTALL_DIR/"
    else
        cp "$BINARY_NAME" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    success "Hub $VERSION installed successfully!"
    
    # Check PATH
    if ! check_path "$INSTALL_DIR"; then
        warn "$INSTALL_DIR is not in your PATH"
        echo ""
        echo "Add it to your PATH by adding this to your shell profile:"
        echo ""
        echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
        echo ""
    fi
    
    # Verify installation
    if check_path "$INSTALL_DIR"; then
        echo ""
        info "Verifying installation..."
        "$INSTALL_DIR/$BINARY_NAME" version
    fi
}

main "$@"
