---
hide:
  - toc
---

# Get Started with DDEV

[DDEV](https://github.com/ddev/ddev) is an open source tool for launching local PHP development environments in minutes.

These environments can be extended, version controlled, and shared, so you can take advantage of a Docker workflow without Docker experience or bespoke configuration. Projects can be changed, powered down, or removed just as easily as they’re started.

## System Requirements

=== "macOS"

    ### macOS
    
    Runs natively on ARM64 (Apple Silicon) and AMD64 machines.

    * RAM: 8GB
    * Storage: 256GB
    * [Colima](https://github.com/abiosoft/colima) or [Docker Desktop](https://www.docker.com/products/docker-desktop/)
    * Colima and Docker Desktop both require macOS Big Sur (11) or higher.
    * On macOS with either Colima or Docker Desktop most users will want to enable Mutagen for best performance, `ddev config global --mutagen-enabled`

=== "Windows WSL2"

    ### Windows WSL2

    * RAM: 8GB
    * Storage: 256GB
    * [Docker Desktop](https://www.docker.com/products/docker-desktop/) on the Windows side or [Docker CE](https://docs.docker.com/engine/install/ubuntu/) inside WSL2
    * Ubuntu or an Ubuntu-derived distro is recommended, though others may work fine

=== "Traditional Windows"

    ### Traditional Windows

    * Any recent edition of Windows Home or Windows Pro.
    * RAM: 8GB
    * Storage: 256GB
    * [Docker Desktop](https://www.docker.com/products/docker-desktop/) using the WSL2 backend

=== "Linux"

    ### Linux

    Most distros and most versions work fine, on both AMD64 and ARM64 architectures.

    * RAM: 8GB
    * Storage: 256GB

=== "Gitpod and GitHub Codespaces"

    ### Gitpod and GitHub Codespaces

    With [Gitpod](https://www.gitpod.io) and [GitHub Codespaces](https://github.com/features/codespaces) you don’t install anything; you only need a browser and an internet connection.
