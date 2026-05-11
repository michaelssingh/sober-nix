{ config, lib, pkgs, ... }:

let
  cfg = config.sober.core.time;
in
{
  options = {
    sober.core.time = {
      enable = lib.mkEnableOption "Secure time synchronization and RTC reset protection" // {
        default = true;
      };
      ntpServers = lib.mkOption {
        type = lib.types.listOf lib.types.str;
        default = [
          "time.cloudflare.com"
          "nts.netnod.se"
          "ptbtime1.ptb.de"
        ];
        description = "List of NTS-capable NTP servers.";
      };
    };
  };

  config = lib.mkIf cfg.enable {
    # Time and Locale Settings
    time.timeZone = "America/Barbados";
    i18n.defaultLocale = "en_US.UTF-8";

    # Chrony with NTS for secure time sync
    services.chrony = {
      enable = true;
      enableNTS = true;
      servers = cfg.ntpServers;
      extraConfig = ''
        # Allow stepping the clock if the adjustment is larger than 1 second
        # for the first 3 updates.
        makestep 1.0 3

        # NTS bootstrap: ignore certificate expiration if the clock is not yet synced.
        # This allows NTS to work even if the clock is far in the past.
        nocerttimecheck 1

        # Save drift information
        driftfile /var/lib/chrony/chrony.drift

        # Enable RTC sync (updates the kernel's 11-minute sync)
        rtcsync
      '';
    };

    # fake-hwclock: Save/restore time to handle battery resets (1999)
    # Restores early in boot so logs/services have a semi-sane time.
    systemd.services.fake-hwclock = {
      description = "Restore / save the current clock";
      wantedBy = [ "sysinit.target" ];
      before = [ "sysinit.target" "systemd-fsck-root.service" "chronyd.service" ];
      conflicts = [ "shutdown.target" ];
      serviceConfig = {
        Type = "oneshot";
        RemainAfterExit = true;
        ExecStart = "${pkgs.fake-hwclock}/bin/fake-hwclock load";
        ExecStop = "${pkgs.fake-hwclock}/bin/fake-hwclock save";
      };
    };

    # Periodically save the clock to minimize time loss on power failure
    systemd.timers.fake-hwclock-save = {
      description = "Periodically save the current clock";
      wantedBy = [ "timers.target" ];
      timerConfig.OnCalendar = "hourly";
    };

    systemd.services.fake-hwclock-save = {
      description = "Save the current clock";
      serviceConfig = {
        Type = "oneshot";
        ExecStart = "${pkgs.fake-hwclock}/bin/fake-hwclock save";
      };
    };

    # Ensure chrony starts after fake-hwclock has restored the time
    systemd.services.chronyd.after = [ "fake-hwclock.service" ];

    environment.systemPackages = [ pkgs.fake-hwclock ];
  };
}
