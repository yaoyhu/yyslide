#!/bin/sh
# 安装 yyslide：curl -fsSL https://raw.githubusercontent.com/yaoyhu/yyslide/main/install.sh | sh
set -eu

REPO="yaoyhu/yyslide"
INSTALL_DIR="${YYSLIDE_INSTALL_DIR:-$HOME/.local/bin}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  darwin|linux) ;;
  *) echo "不支持的系统：$os（Windows 请从 Releases 页面下载 zip）" >&2; exit 1 ;;
esac
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "不支持的架构：$arch" >&2; exit 1 ;;
esac

url="https://github.com/$REPO/releases/latest/download/yyslide_${os}_${arch}.tar.gz"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "下载 $url"
curl -fsSL "$url" | tar -xz -C "$tmp"
mkdir -p "$INSTALL_DIR"
mv "$tmp/yyslide" "$INSTALL_DIR/yyslide"
chmod +x "$INSTALL_DIR/yyslide"

echo "已安装到 $INSTALL_DIR/yyslide"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "注意：$INSTALL_DIR 不在 PATH 中，请把它加入 shell 配置，例如：export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
esac
