# SaaS Engine Development Team

This directory contains the definitions and configurations for the autonomous agent team tasked with building our infrastructure-as-code SaaS platform.

## Agent Roles

### 1. Architect Agent
- **Responsibility:** High-level system design, cross-service communication patterns, and ensuring architectural consistency with core platform requirements.
- **Tools:** System modeling, dependency analysis, roadmap planning.

### 2. Infra Agent
- **Responsibility:** Generation, validation, and maintenance of IaC (Nix/Terraform/Kubernetes manifests).
- **Tools:** IaC linting, structural validation, deployment simulation.

### 3. Service Agent
- **Responsibility:** Application logic, API development, and service-specific implementation.
- **Tools:** Code generation, framework-specific scaffolding, unit testing.

### 4. Quality Agent
- **Responsibility:** Exhaustive testing, security auditing, and compliance verification.
- **Tools:** Integration testing, vulnerability scanning, performance monitoring.

## Workflow Integration
These agents operate under the established `Research -> Strategy -> Execution` lifecycle defined in the root `GEMINI.md`.
