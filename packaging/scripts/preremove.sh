#!/bin/sh
set -e

if [ -d /run/systemd/system ]; then
    systemctl disable --now bitcoind-exporter.service >/dev/null 2>&1 || true
fi
