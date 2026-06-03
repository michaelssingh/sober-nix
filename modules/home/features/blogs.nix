{
  config,
  pkgs,
  lib,
  ...
}:

{
  programs.newsboat = {
    urls = [
      # --- Research / Whitepapers / Systems Papers ---
      {
        url = "http://export.arxiv.org/rss/cs.DC";
        tags = [
          "tech"
          "papers"
          "distributed-systems"
        ];
      }
      {
        url = "http://export.arxiv.org/rss/cs.NI";
        tags = [
          "tech"
          "papers"
          "networking"
        ];
      }
      {
        url = "https://www.usenix.org/conferences/rss";
        tags = [
          "tech"
          "papers"
          "osdi"
          "nsdi"
        ];
      }
      {
        url = "https://research.google/blog/rss/";
        tags = [
          "tech"
          "research"
          "cloud"
        ];
      }
      {
        url = "https://www.microsoft.com/en-us/research/blog/feed/";
        tags = [
          "tech"
          "research"
        ];
      }
      {
        url = "http://muratbuffalo.blogspot.com/feeds/posts/default";
        tags = [
          "tech"
          "distributed-systems"
          "papers"
        ];
      }

      # --- Community ---
      {
        url = "https://news.ycombinator.com/rss";
        tags = [
          "tech"
          "community"
          "news"
        ];
      }
      {
        url = "https://lobste.rs/rss";
        tags = [
          "tech"
          "distributed-systems"
          "community"
        ];
      }

      # --- Systems / Engineering / Nix ---
      {
        url = "https://nixos.org/blog/announcements-rss.xml";
        tags = [
          "tech"
          "nix"
        ];
      }
      {
        url = "https://lwn.net/headlines/rss";
        tags = [
          "tech"
          "linux"
          "kernel"
        ];
      }
      {
        url = "https://blog.rust-lang.org/feed.xml";
        tags = [
          "tech"
          "rust"
        ];
      }

      # --- Fedora / Red Hat Ecosystem ---
      {
        url = "https://fedoramagazine.org/feed/";
        tags = [
          "tech"
          "fedora"
          "linux"
        ];
      }
      {
        url = "https://communityblog.fedoraproject.org/feed/";
        tags = [
          "tech"
          "fedora"
        ];
      }

      # --- Distributed Systems / Cloud / Engineering ---
      {
        url = "https://netflixtechblog.com/feed";
        tags = [
          "tech"
          "distributed-systems"
          "cloud"
        ];
      }
      {
        url = "https://www.uber.com/blog/engineering/rss/";
        tags = [
          "tech"
          "distributed-systems"
          "engineering"
        ];
      }
      {
        url = "https://blog.cloudflare.com/rss/";
        tags = [
          "tech"
          "cloud"
          "security"
        ];
      }
      {
        url = "https://www.allthingsdistributed.com/atom.xml";
        tags = [
          "tech"
          "distributed-systems"
        ];
      }
      {
        url = "https://cloud.google.com/blog/products/devops-sre/feed";
        tags = [
          "tech"
          "sre"
          "cloud"
        ];
      }
      {
        url = "https://aws.amazon.com/blogs/architecture/feed/";
        tags = [
          "tech"
          "aws"
          "cloud"
          "engineering"
        ];
      }
      {
        url = "https://blog.bytebytego.com/feed";
        tags = [
          "tech"
          "engineering"
        ];
      }
      {
        url = "https://platformengineering.org/blog/rss.xml";
        tags = [
          "tech"
          "platform-engineering"
        ];
      }
      {
        url = "https://thenewstack.io/feed/";
        tags = [
          "tech"
          "cloud-native"
        ];
      }
      {
        url = "https://www.cncf.io/feed/";
        tags = [
          "tech"
          "cncf"
          "cloud-native"
        ];
      }

      # --- Golang ---
      {
        url = "https://go.dev/blog/feed.atom";
        tags = [
          "tech"
          "golang"
        ];
      }
      {
        url = "https://dave.cheney.net/feed";
        tags = [
          "tech"
          "golang"
        ];
      }
      {
        url = "https://www.ardanlabs.com/blog/index.xml";
        tags = [
          "tech"
          "golang"
        ];
      }

      # --- Philosophy / Psychology / Spirituality ---
      {
        url = "https://aeon.co/feed.rss";
        tags = [
          "philosophy"
          "psychology"
        ];
      }
      {
        url = "https://www.psychologytoday.com/blog/feed";
        tags = [ "psychology" ];
      }
      {
        url = "https://samharris.org/feed/";
        tags = [
          "philosophy"
          "spirituality"
        ];
      }
    ];
  };
}
