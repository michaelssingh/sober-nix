#!/usr/bin/env python3
import os
import sys
import subprocess
import urllib.parse

# qutebrowser passes the current URL via an environment variable
url = os.environ.get('QUTE_URL')

if url and "inv.nadeko.net" in url:
    parsed = urllib.parse.urlparse(url)
    params = urllib.parse.parse_qs(parsed.query)
    if 'v' in params:
        url = f"https://www.youtube.com/watch?v={params['v'][0]}"

subprocess.Popen(['mpv', url])
