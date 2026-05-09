# SOBER Systems Infrastructure

This repository contains the declarative NixOS system configurations for the **SOBER** (Systems Oriented Business Engineering & Research) infrastructure. It uses **Nix Flakes** to manage multiple hosts and architectures from a single source of truth.

## 🦉 Naming Convention: The Owl Schema
Hosts are named after Owl genera to reflect the system's identity (Observation, Wisdom, Adaptability).

| Hostname | Hardware | Architecture | Role | Vibe |
| :--- | :--- | :--- | :--- | :--- |
| **Athene** | Lenovo Ideapad 130-15AST | `x86_64-linux` | Workstation | *The Little Owl* - Compact, grounded, wise observer. |
| **Tyto** | Lenovo Yoga (Future) | `x86_64-linux` | Workstation | *The Barn Owl* - Widespread, highly adaptable daily driver. |
| **Strix** | Oracle Ampere A1 (Future) | `aarch64-linux` | Server | *The Earless Owl* - Silent, powerful, operates in the dark (Cloud). |

## 📂 Directory Layout

The project follows a **System-Centric** layout, separating hardware (Hosts) from software behavior (Modules/Users).

```text
~/git/sober-nix/
├── hosts/              # 🖥️ Hardware-specific entry points
│   └── athene/         # Lenovo Ideapad 130-15AST
├── modules/            # 🧱 Reusable logic "Bricks"
│   ├── core/           # System-wide foundational settings
│   ├── home/           # User-space configs (Firefox, Nvim, Sway, Shell)
│   ├── roles/          # High-level abstractions (e.g., Workstation role)
│   └── services/       # System services (Greetd, Kanata keyboard mapper)
├── users/              # 👤 User profile composition
│   └── michael/        # Profile variants (Core, Server, Workstation)
├── flake.nix           # The orchestration brain
└── flake.lock          # Reproducible dependency pin
```
