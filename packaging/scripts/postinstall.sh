#!/bin/sh
set -e

if ! getent group bitcoind-exporter >/dev/null; then
    groupadd --system bitcoind-exporter
fi

if ! getent passwd bitcoind-exporter >/dev/null; then
    useradd --system \
        --gid bitcoind-exporter \
        --home-dir /var/lib/bitcoind-exporter \
        --create-home \
        --shell /usr/sbin/nologin \
        bitcoind-exporter
fi

mkdir -p /var/lib/bitcoind-exporter
chown bitcoind-exporter:bitcoind-exporter /var/lib/bitcoind-exporter

if [ -d /run/systemd/system ]; then
    systemctl daemon-reload >/dev/null 2>&1 || true
    systemctl enable --now bitcoind-exporter.service >/dev/null 2>&1 || true
fi
