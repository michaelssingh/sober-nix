{
  config,
  lib,
  pkgs,
  ...
}:

{
  options = {
    sober.core.perf.lowend.enable = lib.mkEnableOption "Low-end performance optimizations";
  };

  config = lib.mkIf config.sober.core.perf.lowend.enable {
    # Speed up system with alternative kernel/scheduler
    boot.kernelPackages = pkgs.linuxPackages_xanmod; # xanmod LTS (6.18) — the xanmodKernels attrset was restructured in 26.05

    # Efficient swap management for limited RAM
    swapDevices = [ ];
    zramSwap = {
      enable = true;
      algorithm = "zstd";
      memoryPercent = 75;
      priority = 100;
    };

    # System services for resource management
    services.ananicy = {
      enable = true;
      package = pkgs.ananicy-cpp;
      rulesProvider = pkgs.ananicy-rules-cachyos;
    };
    services.irqbalance.enable = true;
    services.earlyoom = {
      enable = true;
      freeMemThreshold = 5;
      freeSwapThreshold = 5;
    };
    services.fstrim.enable = lib.mkDefault true;

    # Performance governance
    powerManagement.cpuFreqGovernor = "performance";

    # --- Systemd Optimizations ---
    systemd.settings.Manager = {
      DefaultTimeoutStopSec = "15s";
      DefaultStartLimitIntervalSec = "10s";
    };

    services.journald.extraConfig = ''
      SystemMaxUse=50M
      MaxLevelStore=notice
      MaxLevelSyslog=notice
      MaxLevelConsole=notice
    '';

    # Disable core dumps to save disk I/O
    systemd.coredump.enable = false;

    # Enable systemd OOM killer
    systemd.oomd.enable = true;
  };
}
