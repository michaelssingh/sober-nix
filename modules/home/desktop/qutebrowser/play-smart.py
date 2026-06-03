#!/usr/bin/env python3
import os
import re
import subprocess
import urllib.parse

def main():
    # qutebrowser passes the current URL via an environment variable
    url = os.environ.get('QUTE_URL')
    if not url:
        return

    # 1. Handle Invidious/Alternative instances if we're on a watch page
    if "watch?v=" in url:
        parsed = urllib.parse.urlparse(url)
        params = urllib.parse.parse_qs(parsed.query)
        if 'v' in params:
            url = f"https://www.youtube.com/watch?v={params['v'][0]}"
            play(url)
            return

    # 2. If not a direct watch link, look into the HTML for a "Watch on YouTube" link
    # Qutebrowser provides the page HTML in a temporary file pointed to by QUTE_HTML
    html_path = os.environ.get('QUTE_HTML')
    if html_path and os.path.exists(html_path):
        with open(html_path, 'r', encoding='utf-8') as f:
            html = f.read()
            # Look for the canonical "Watch on YouTube" button or any youtube watch link
            match = re.search(r'https?://(?:www\.)?youtube\.com/watch\?v=[\w-]+', html)
            if match:
                play(match.group(0))
                return

    # 3. Fallback: just play the current URL and let yt-dlp handle it
    play(url)

def play(url):
    # Prefer our custom mpv-queue script if it exists in PATH
    try:
        subprocess.Popen(['mpv-queue', url])
    except FileNotFoundError:
        subprocess.Popen(['mpv', url])

if __name__ == "__main__":
    main()
