/*
 * init.c - SOBER Infrastructure Initialization Layer
 *
 * A single entry point for building and deploying any host in the fleet.
 *
 * Usage:
 *   ./init <host> [command] [options]
 *
 * Hosts:
 *   otus            - Local NixOS workstation (nixos-rebuild switch)
 *   strix           - Fly.io pastebin container
 *   styx            - Fly.io Nix remote builder container
 *   athene          - Fly.io Matrix homeserver container
 *   bubo            - Fly.io Git forge container
 *   glaucidium      - Fly.io misc container
 *
 * Commands:
 *   build           - Build only, do not apply/deploy
 *   switch          - Build and apply (default for otus)
 *   deploy          - Build, push, and deploy to Fly.io (default for servers)
 *
 * Options:
 *   --dry-run       - Print commands without executing them
 *   --help          - Show this help text
 *
 * Examples:
 *   ./init otus
 *   ./init otus build
 *   ./init strix deploy
 *   ./init strix --dry-run
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* ------------------------------------------------------------------ */
/* Constants                                                           */
/* ------------------------------------------------------------------ */

#define MAX_CMD 1024

/* Known hosts */
static const char *WORKSTATIONS[] = { "otus", NULL };
static const char *SERVERS[]      = { "strix", "styx", "athene", "bubo", "glaucidium", NULL };

/* ------------------------------------------------------------------ */
/* Helpers                                                             */
/* ------------------------------------------------------------------ */

/* Returns 1 if needle is in a NULL-terminated list of strings. */
static int
in_list(const char *needle, const char *list[])
{
    int i;
    for (i = 0; list[i] != NULL; i++) {
        if (strcmp(needle, list[i]) == 0)
            return 1;
    }
    return 0;
}

/* Run a shell command, or just print it if dry_run is set. */
static int
run(const char *cmd, int dry_run)
{
    printf("  $ %s\n", cmd);
    if (dry_run)
        return 0;
    return system(cmd);
}

/* Build a string into buf, checking it fits within size. */
static void
build_cmd(char *buf, size_t size, const char *fmt, const char *a, const char *b)
{
    int n = snprintf(buf, size, fmt, a, b);
    if (n < 0 || (size_t)n >= size) {
        fprintf(stderr, "error: command string too long\n");
        exit(1);
    }
}

/* ------------------------------------------------------------------ */
/* Actions                                                             */
/* ------------------------------------------------------------------ */

static void
usage(void)
{
    puts("Usage: ./init <host> [build|switch|deploy] [--dry-run] [--help]");
    puts("");
    puts("Hosts (workstation): otus");
    puts("Hosts (servers):     strix  styx  athene  bubo  glaucidium");
    puts("");
    puts("Commands:");
    puts("  build    - Build only, do not apply");
    puts("  switch   - Build and apply  (default for otus)");
    puts("  deploy   - Build, push, and deploy to Fly.io  (default for servers)");
    puts("");
    puts("Options:");
    puts("  --dry-run  Print commands without running them");
    puts("  --help     Show this message");
}

/*
 * Build + switch the local NixOS workstation.
 */
static int
action_workstation(const char *host, const char *cmd, int dry_run)
{
    char buf[MAX_CMD];

    if (strcmp(cmd, "build") == 0) {
        printf("\n[init] Building %s (build only)...\n", host);
        build_cmd(buf, sizeof(buf), "nixos-rebuild build --flake .#%s", host, "");
        return run(buf, dry_run);
    }

    /* Default: switch */
    printf("\n[init] Switching %s...\n", host);
    build_cmd(buf, sizeof(buf), "sudo nixos-rebuild switch --flake .#%s", host, "");
    return run(buf, dry_run);
}

/*
 * Build, push, and deploy a Fly.io server container.
 * Delegates to the per-host deploy.sh which handles skopeo + fly deploy.
 */
static int
action_server(const char *host, const char *cmd, int dry_run)
{
    char buf[MAX_CMD];

    if (strcmp(cmd, "build") == 0) {
        printf("\n[init] Building %s-image (build only)...\n", host);
        build_cmd(buf, sizeof(buf), "nix build .#%s-image", host, "");
        return run(buf, dry_run);
    }

    /* Default: full deploy via the host's deploy.sh */
    printf("\n[init] Deploying %s to Fly.io...\n", host);
    build_cmd(buf, sizeof(buf), "bash hosts/server/%s/deploy.sh", host, "");
    return run(buf, dry_run);
}

/* ------------------------------------------------------------------ */
/* Entry point                                                         */
/* ------------------------------------------------------------------ */

int
main(int argc, char *argv[])
{
    const char *host    = NULL;
    const char *cmd     = NULL;
    int         dry_run = 0;
    int         i;

    /* Parse arguments */
    for (i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--help") == 0 || strcmp(argv[i], "-h") == 0) {
            usage();
            return 0;
        } else if (strcmp(argv[i], "--dry-run") == 0) {
            dry_run = 1;
        } else if (host == NULL) {
            host = argv[i];
        } else if (cmd == NULL) {
            cmd = argv[i];
        } else {
            fprintf(stderr, "error: unexpected argument '%s'\n", argv[i]);
            usage();
            return 1;
        }
    }

    if (host == NULL) {
        fprintf(stderr, "error: no host specified\n\n");
        usage();
        return 1;
    }

    /* Validate host and set default command */
    if (in_list(host, WORKSTATIONS)) {
        if (cmd == NULL)
            cmd = "switch";
        return action_workstation(host, cmd, dry_run);
    } else if (in_list(host, SERVERS)) {
        if (cmd == NULL)
            cmd = "deploy";
        return action_server(host, cmd, dry_run);
    } else {
        fprintf(stderr, "error: unknown host '%s'\n\n", host);
        usage();
        return 1;
    }
}
