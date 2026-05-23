#!/usr/bin/env bash
# Script to create a new machine on Fly.io using a fresh deploy token.

APP="sober-services"
TOKEN=$(fly tokens create deploy --app $APP)

curl -i -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  "https://api.machines.dev/v1/apps/$APP/machines" \
  -d '{
    "config": {
      "init": {
        "exec": [
          "/run/current-system/systemd/lib/systemd/systemd"
        ]
      },
      "containers": [
        {
          "name": "ubuntu",
          "image": "jrei/systemd-ubuntu",
          "cmd": [
            "/usr/bin/systemd"
          ]
        }
      ],
      "guest": {
        "cpu_kind": "shared",
        "cpus": 1,
        "memory_mb": 2048
      }
    }
  }'
