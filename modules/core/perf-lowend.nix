{ config, lib, pkgs, ... }:

{
  options = {
    sober.core.perf.lowend.enable = lib.mkEnableOption "Low-end performance optimizations";
  };

  config = lib.mkIf config.sober.core.perf.lowend.enable {
    # Speed up system with alternative kernel/scheduler
    boot.kernelPackages = pkgs.linuxPackages_xanmod;
    
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
  };
}
