#!/usr/bin/env bash

# Colors
RESET="\033[0m"
GREEN="\033[1;32m"
YELLOW="\033[1;33m"
RED="\033[1;31m"
BLUE="\033[1;34m"

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${RESET} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${RESET} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${RESET} $1"; }
log_error() { echo -e "${RED}[ERROR]${RESET} $1"; }

# RELEASE DOWNLOAD LINKS
CURRENT_VERSION="v0.7.1"
DARWIN_ARM64="https://github.com/yassinebenaid/bunster/releases/download/${CURRENT_VERSION}/bunster_darwin-arm64.tar.gz"
DARWIN_AMD64="https://github.com/yassinebenaid/bunster/releases/download/${CURRENT_VERSION}/bunster_darwin-amd64.tar.gz"
LINUX_386="https://github.com/yassinebenaid/bunster/releases/download/${CURRENT_VERSION}/bunster_linux-386.tar.gz"
LINUX_AMD64="https://github.com/yassinebenaid/bunster/releases/download/${CURRENT_VERSION}/bunster_linux-amd64.tar.gz"
LINUX_ARM64="https://github.com/yassinebenaid/bunster/releases/download/${CURRENT_VERSION}/bunster_linux-arm64.tar.gz"

# Fetch Info
DOWNLOAD_LINK=""
BINARY_NAME=""
fetch_system_info() {
  ARCH=$(uname -m)
  OS=$(uname -s)
  log_info "Finding binary for Arch: ${ARCH}, OS: ${OS}"
  if [[ "$OS" == Darwin && "$ARCH" == "arm64" ]]; then
    DOWNLOAD_LINK="$DARWIN_ARM64"
    BINARY_NAME="bunster/bunster_darwin-arm64"
  elif [[ "$OS" == "Darwin" && "$ARCH" == "x86_64" ]]; then
    DOWNLOAD_LINK="$DARWIN_AMD64"
    BINARY_NAME="bunster/bunster_darwin-amd64"
  elif [[ "$OS" == "Linux" && ("$ARCH" == "i386" || "$ARCH" == "i686") ]]; then
    DOWNLOAD_LINK="$LINUX_386"
    BINARY_NAME="bunster/bunster_linux-386"
  elif [[ "$OS" == "Linux" && "$ARCH" == "x86_64" ]]; then
    DOWNLOAD_LINK="$LINUX_AMD64"
    BINARY_NAME="bunster/bunster_linux-amd64"
  elif [[ "$OS" == "Linux" && "$ARCH" == "arm64" ]]; then
    DOWNLOAD_LINK="$LINUX_ARM64"
    BINARY_NAME="bunster/bunster_linux-arm64"
  else
    log_error "OPERATING SYSTEM AND/OR ARCH NOT SUPPORTED"
    log_info "Attemptinng install with go..."
    if command -v go &>/dev/null; then
      log_success "go is installed, proceeding..."
      go_install
    else
      log_error "go not installed... aborting..."
      exit 1
    fi
  fi
}

# Check dependency and route
check_and_route() {
  log_info "Checking for dependencies..."

  if command -v tar &>/dev/null; then
    log_success "tar is installed"
  else
    log_error "tar is not installed, aborting..."
    exit 1
  fi

  if command -v curl &>/dev/null; then
    log_success "curl is installed, proceeding..."
    curl_install
  elif command -v wget &>/dev/null; then
    log_success "wget is installed, proceeding..."
    wget_install
  elif command -v fetch &>/dev/null; then
    log_success "fetch is installed, proceeding..."
    fetch_install
  elif command -v go &>/dev/null; then
    log_success "go is installed, proceeding..."
    go_install
  else
    log_error "Missing install dependency: \n
Make sure you have atleast one of the following:\n
- wget\n
- curl\n
- fetch\n
- go"
    exit 1
  fi
}

curl_install() {
  log_info "Starting installation..."
  if curl -o bunster.tar.gz -L "$DOWNLOAD_LINK"; then
    log_success "Bunster installed successfully."
    tar_install
  else
    log_error "Failed to download using curl. Exiting..."
    exit 1
  fi
}

wget_install() {
  log_info "Starting installation..."
  if wget -O bunster.tar.gz "$DOWNLOAD_LINK"; then
    log_success "Bunster installed successfully."
    tar_install
  else
    log_error "Failed to download using wget. Exiting..."
    exit 1
  fi
}

fetch_install() {
  log_info "Starting installation..."
  if fetch --location "$DOWNLOAD_LINK" -o bunster.tar.gz; then
    log_success "Bunster installed successfully."
    tar_install
  else
    log_error "Failed to download using wget. Exiting..."
    exit 1
  fi
}

go_install() {
  log_info "Starting installation..."
  if go install github.com/yassinebenaid/bunster/cmd/bunster@latest; then
    log_success "Bunster installed at ~/go/bin/."
    BINARY_NAME="$HOME/go/bin/bunster"
    binary_move
  else
    log_error "Failed to install via go. Exiting..."
    exit 1
  fi
}

tar_install() {
  mkdir bunster/
  log_info "Unzipping tar..."
  if tar -xvzf bunster.tar.gz -C bunster/; then
    log_success "Unzipped tar.gz"
  else
    log_error "Failed to unzip tar.gz"
    exit 1
  fi
  binary_move
}

binary_move() {
  read -p "Move binary to /usr/local/bin? (Y/n): " response
  response=${response:-Y}
  if [[ "$response" =~ ^[Yy]$ ]]; then
    log_info "Proceeding..."
    sudo mv "${BINARY_NAME}" "/usr/local/bin/bunster" || {
      log_error "Failed to install package"
      exit 1
    }
    clean
    log_success "Bunster installed successfully."
  else
    log_error "Operation canceled."
    exit 1
  fi
}

clean() {
  log_info "Cleaning..."
  rm -rf bunster/ bunster.tar.gz
}

main() {
  fetch_system_info
  check_and_route
}

main
