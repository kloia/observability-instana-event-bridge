# systemd Service

Example service file:

```text
systemd/instana-event-bridge.service
```

## Install

```bash
sudo mkdir -p /opt/instana-event-bridge

sudo cp instana-event-bridge /opt/instana-event-bridge/
sudo cp config.json /opt/instana-event-bridge/

sudo cp systemd/instana-event-bridge.service /etc/systemd/system/

sudo systemctl daemon-reload

sudo systemctl enable instana-event-bridge

sudo systemctl start instana-event-bridge
```

## Status

```bash
sudo systemctl status instana-event-bridge
```

## Logs

```bash
journalctl -u instana-event-bridge -f
```