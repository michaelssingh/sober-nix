# SOBER Systems Infrastructure

This repository contains the declarative NixOS system configurations for the **SOBER** (Systems Oriented Business Engineering & Research) infrastructure. It uses **Nix Flakes** to manage multiple hosts and architectures from a single source of truth.

## 🦉 Naming Convention: Owl Schema & Anime Characters
Hosts are primarily named after Owl genera (to reflect the system's identity: Observation, Wisdom, Adaptability) alongside notable anime characters.

| Hostname | Hardware | Architecture | Role | Vibe |
| :--- | :--- | :--- | :--- | :--- |
| **Otus** | Lenovo IdeaPad Slim 1-14AST-05 | `x86_64-linux` | Workstation | *The Scops Owl* - Small, adaptable. |
| **Athene** | Fly.io MicroVM (256MB) | `x86_64-linux` | IRC Bouncer | *The Little Owl* - Small, vigilant. |
| **Bubo** | Fly.io MicroVM (512MB) | `x86_64-linux` | Git Forge | *The Eagle-Owl* - Large, sovereign, powerful. |
| **Glaucidium** | Fly.io MicroVM (256MB) | `x86_64-linux` | VPN Gateway | *The Pygmy Owl* - Tiny but fierce. |
| **Strix** | Fly.io MicroVM (256MB) | `x86_64-linux` | Pastebin | *The Wood Owl* - Silent, record-keeping. |
| **Styx** | Fly.io MicroVM (2048MB) | `x86_64-linux` | Nix Builder | *The Stygian Owl* - Dark, constructing. |

## 📂 Directory Layout

The project follows a **System-Centric** layout, separating hardware (Hosts) from software behavior (Modules/Users).

```text
~/git/sober-nix/
├── cmd/                # 🛠️ Command line scripts and repository utility tools
├── docs/               # 📖 Documentation and system cheatsheets
├── hosts/              # 🖥️ Hardware and host-specific entry points
│   ├── otus/           # Lenovo IdeaPad Slim 1-14AST-05 Workstation
│   ├── athene/         # Fly.io MicroVM - Soju IRC Bouncer
│   ├── bubo/           # Fly.io MicroVM - Forgejo Git Forge
│   ├── glaucidium/     # Fly.io MicroVM - WireGuard VPN Gateway
│   ├── strix/          # Fly.io MicroVM - rustypaste Pastebin
│   └── styx/           # Fly.io MicroVM - Nix Remote Builder
├── modules/            # 🧱 Reusable NixOS and Home Manager logic modules
│   ├── core/           # System-wide foundational settings (e.g., SSH keys)
│   ├── home/           # User-space configs (Firefox, Nvim, Shell, features)
│   ├── overlays/       # Nixpkgs additions and modifications
│   ├── roles/          # High-level abstractions (e.g., Workstation role)
│   └── services/       # System services and background daemons
├── packages/           # 📦 Custom Nix packages (e.g., antigravity, typos, etc.)
├── secrets/            # 🔑 Age-encrypted credentials managed by sops-nix
├── users/              # 👤 User profile compositions
│   └── michael/        # Profile variants (Core, Server, Workstation)
├── flake.nix           # The orchestration brain and flake output definitions
├── flake.lock          # Reproducible dependency pin
└── GEMINI.md           # Shared configuration and memory rules for Gemini CLI
```
