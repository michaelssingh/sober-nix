#!/nix/store/60m4rxhg2fldqaak400c0lry96ijrzqn-python3-3.13.13/bin/python3

# SPDX-FileCopyrightText: Chris Braun (cryzed) <cryzed@googlemail.com>
#
# SPDX-License-Identifier: GPL-3.0-or-later

"""
Insert login information using Bitwarden CLI and a dmenu-compatible application
(e.g. dmenu, rofi -dmenu, ...).
"""

import sys;import site;import functools;sys.argv[0] = '/nix/store/m1p3mlf2qgr376l7bwjzp0bl9bh074n9-qutebrowser-3.7.0/share/qutebrowser/userscripts/qute-bitwarden';functools.reduce(lambda k, p: site.addsitedir(p, k), ['/nix/store/m1p3mlf2qgr376l7bwjzp0bl9bh074n9-qutebrowser-3.7.0/lib/python3.13/site-packages','/nix/store/d7f1iz87hgm7a0v0m6kh9pz4akzbs4yb-python3.13-colorama-0.4.6/lib/python3.13/site-packages','/nix/store/8y32jrqnknxj6hakyg8x64y75gbl8jry-python3.13-pyyaml-6.0.3/lib/python3.13/site-packages','/nix/store/83qykbzagjhqykl6m08qbk2nay0vnk1l-python3.13-pyqt6-sip-13.10.2/lib/python3.13/site-packages','/nix/store/h921vvlwx0zfwwi9w1d3dm3g071hs9hd-python3.13-dbus-python-1.4.0/lib/python3.13/site-packages','/nix/store/vja3raaq07p9hmcqzr3dyian0n58d18n-python3.13-pyqt6-6.11.0/lib/python3.13/site-packages','/nix/store/g8gmizmn14zv6s9r75jdkkh5d0b1y8ks-python3.13-pyqt6-webengine-6.11.0/lib/python3.13/site-packages','/nix/store/1zac3f2hkdf2wwh4zh34knqi160xc9fc-python3.13-jinja2-3.1.6/lib/python3.13/site-packages','/nix/store/7kzhhwp7zqj3gz7g4an960pdq1c98an5-python3.13-markupsafe-3.0.3/lib/python3.13/site-packages','/nix/store/pg8jlnqaww41s7iwcqgc3bsxrv7lpmjb-python3.13-pygments-2.20.0/lib/python3.13/site-packages','/nix/store/cpm6g7n432h8v2cxwpcj2jz905a5c3y3-python3.13-tldextract-5.3.1/lib/python3.13/site-packages','/nix/store/w0ifzb6nrg9izlxk018dva2770g32yba-python3.13-filelock-3.20.3/lib/python3.13/site-packages','/nix/store/yclf543g13ngziva0j30prwr0rx3kc4h-python3.13-idna-3.13/lib/python3.13/site-packages','/nix/store/rks4c012hxa7z01sfc7w4p7mn74zfp2x-python3.13-requests-2.33.1/lib/python3.13/site-packages','/nix/store/dddj23r1wqpa6f9csjr37661hv5aq4gy-python3.13-certifi-2026.01.04/lib/python3.13/site-packages','/nix/store/4zv7hxkwpv4r83fdh9jmwn1649z6wazl-python3.13-charset-normalizer-3.4.7/lib/python3.13/site-packages','/nix/store/hd9vsw1ajhl9031aq4chw08zfjamv754-python3.13-urllib3-2.6.3/lib/python3.13/site-packages','/nix/store/slgvwgd9knma7909h5psbp0xsff8kp2c-python3.13-requests-file-2.1.0/lib/python3.13/site-packages','/nix/store/188iydzv1jqal6k3rq0d7g5kbvyanhyl-python3.13-beautifulsoup4-4.14.3/lib/python3.13/site-packages','/nix/store/xpcamnjys1iq0vrn8f41s1fqs58vip36-python3.13-soupsieve-2.8.3/lib/python3.13/site-packages','/nix/store/kqdr0pijf558v8p6bm0i246jnhhh4ckq-python3.13-typing-extensions-4.15.0/lib/python3.13/site-packages','/nix/store/vx8xfj8i5sdvbbifs2lmgv73pcg4h22k-python3.13-readability-lxml-0.8.4.1/lib/python3.13/site-packages','/nix/store/vm4j40l88jshvx6sz4dcas99m71f04jz-python3.13-chardet-5.2.0/lib/python3.13/site-packages','/nix/store/gwpjfw95ij5ip43ygwhfjyj4c110av3c-python3.13-cssselect-1.3.0/lib/python3.13/site-packages','/nix/store/xkbzjgq6pvlnlsh9if1y3pdj5yx8nw54-python3.13-lxml-6.0.2/lib/python3.13/site-packages','/nix/store/7wqzki36ykxqaf4nza0268vkncds0s8w-python3.13-lxml-html-clean-0.4.4/lib/python3.13/site-packages','/nix/store/ikhx1g8iiiqqv6l8n9hnhi75jk280sh7-python3.13-pykeepass-4.1.1.post1/lib/python3.13/site-packages','/nix/store/n3fxq6622243458xbqv08nyp9lvps3qv-python3.13-argon2-cffi-25.1.0/lib/python3.13/site-packages','/nix/store/jlwl707jssakj0plrwmzk8yjzx8k9ka5-python3.13-argon2-cffi-bindings-25.1.0/lib/python3.13/site-packages','/nix/store/34mandgnlpgpipkdlrm6v988kj72y089-python3.13-cffi-2.0.0/lib/python3.13/site-packages','/nix/store/vzfg067i45rzc9fjq0v9670jj8gphmz9-python3.13-pycparser-3.00/lib/python3.13/site-packages','/nix/store/2ydxid0m3j5x70r6v0lskrf9wbfhyvsq-python3.13-construct-2.10.70/lib/python3.13/site-packages','/nix/store/8bsmivrpsbahajfki474d4wvqyma4vfg-python3.13-lz4-4.4.5/lib/python3.13/site-packages','/nix/store/32f0qzypz0zyb6gy2za1skssa2m5lc2d-python3.13-pycryptodomex-3.23.0/lib/python3.13/site-packages','/nix/store/bjcy8w8dmhl4qdvlr1834zdlv1wpd75y-python3.13-stem-1.8.3/lib/python3.13/site-packages','/nix/store/3hc2ws7glh43qf3mkm8kvdaxd4l8phl9-python3.13-pynacl-1.6.2/lib/python3.13/site-packages','/nix/store/jw06b34vxzxjs7h9rmg7i309v7f2bi27-python3.13-adblock-0.6.0/lib/python3.13/site-packages','/nix/store/9vpy6ikx3nyndy15jmk6gq3wx3v2fqfp-python3.13-pyperclip-1.11.0/lib/python3.13/site-packages'], site._init_pathinfo());
USAGE = """The domain of the site has to be in the name of the Bitwarden entry, for example: "github.com/cryzed" or
"websites/github.com".  The login information is inserted by emulating key events using qutebrowser's fake-key command in this manner:
[USERNAME]<Tab>[PASSWORD], which is compatible with almost all login forms.

If enabled, with the `--totp` flag, it will also move the TOTP code to the
clipboard, much like the Firefox add-on.

You must log into Bitwarden CLI using `bw login` prior to use of this script.
The session key will be stored using keyctl for the number of seconds passed to
the --auto-lock option.

To use in qutebrowser, run: `spawn --userscript qute-bitwarden`
"""

EPILOG = """Dependencies: tldextract (Python 3 module), pyperclip (optional
Python module, used for TOTP codes), Bitwarden CLI (1.7.4 is known to work
but older versions may well also work)

WARNING: The login details are viewable as plaintext in qutebrowser's debug log
(qute://log) and might be shared if you decide to submit a crash report!"""

import argparse
import enum
import functools
import os
import shlex
import subprocess
import sys
import json
import tldextract

argument_parser = argparse.ArgumentParser(
    description=__doc__,
    usage=USAGE,
    epilog=EPILOG,
)
argument_parser.add_argument('url', nargs='?', default=os.getenv('QUTE_URL'))
argument_parser.add_argument('--dmenu-invocation', '-d', default='rofi -dmenu -i -p Bitwarden',
                             help='Invocation used to execute a dmenu-provider')
argument_parser.add_argument('--password-prompt-invocation', '-p', default='rofi -dmenu -p "Master Password" -password -lines 0',
                             help='Invocation used to prompt the user for their Bitwarden password')
argument_parser.add_argument('--no-insert-mode', '-n', dest='insert_mode', action='store_false',
                             help="Don't automatically enter insert mode")
argument_parser.add_argument('--totp', '-t', action='store_true',
                             help="Copy TOTP key to clipboard")
argument_parser.add_argument('--io-encoding', '-i', default='UTF-8',
                             help='Encoding used to communicate with subprocesses')
argument_parser.add_argument('--merge-candidates', '-m', action='store_true',
                             help='Merge pass candidates for fully-qualified and registered domain name')
argument_parser.add_argument('--auto-lock', type=int, default=900,
                             help='Automatically lock the vault after this many seconds')
group = argument_parser.add_mutually_exclusive_group()
group.add_argument('--username-only', '-e',
                   action='store_true', help='Only insert username')
group.add_argument('--password-only', '-w',
                   action='store_true', help='Only insert password')
group.add_argument('--totp-only', '-T',
                   action='store_true', help='Only insert totp code')

stderr = functools.partial(print, file=sys.stderr)


class ExitCodes(enum.IntEnum):
    SUCCESS = 0
    FAILURE = 1
    # 1 is automatically used if Python throws an exception
    NO_PASS_CANDIDATES = 2
    COULD_NOT_MATCH_USERNAME = 3
    COULD_NOT_MATCH_PASSWORD = 4


def qute_command(command):
    with open(os.environ['QUTE_FIFO'], 'w') as fifo:
        fifo.write(command + '\n')
        fifo.flush()


def ask_password(password_prompt_invocation):
    pass

def get_session_key(auto_lock, password_prompt_invocation):
    pass


def pass_(domain, encoding, auto_lock, password_prompt_invocation):
    process = subprocess.run(
        ['rbw', 'ls', '--raw'],
        capture_output=True,
    )

    if process.returncode:
        return '[]'

    entries = json.loads(process.stdout.decode(encoding).strip() or "[]")
    matches = []
    
    for e in entries:
        match = False
        if domain.lower() in (e.get('name') or '').lower():
            match = True
        for uri in (e.get('uris') or []):
            if domain.lower() in uri.lower():
                match = True
                
        if match:
            matches.append({
                'id': e['id'],
                'name': e['name'],
                'login': {
                    'username': e.get('user', '') or '',
                    'password': '', # Fetched on demand
                    'totp': ''
                }
            })

    return json.dumps(matches)


def get_totp_code(selection_id, domain_name, encoding, auto_lock, password_prompt_invocation):
    process = subprocess.run(
        ['rbw', 'code', selection_id],
        capture_output=True,
    )

    if process.returncode:
        return ''

    return process.stdout.decode(encoding).strip()


def dmenu(items, invocation, encoding):
    command = shlex.split(invocation)
    process = subprocess.run(command, input='\n'.join(
        items).encode(encoding), stdout=subprocess.PIPE)
    return process.stdout.decode(encoding).strip()


def fake_key_raw(text):
    for character in text:
        # Escape all characters by default, space requires special handling
        sequence = '" "' if character == ' ' else r'\{}'.format(character)
        qute_command('fake-key {}'.format(sequence))


def main(arguments):
    if not arguments.url:
        argument_parser.print_help()
        return ExitCodes.FAILURE

    extract_result = tldextract.extract(arguments.url)

    # Try to find candidates using targets in the following order: fully-qualified domain name (includes subdomains),
    # the registered domain name and finally: the IPv4 address if that's what
    # the URL represents
    candidates = []
    for target in filter(
        None,
        [
            extract_result.fqdn,
            (
                extract_result.top_domain_under_public_suffix
                if hasattr(extract_result, "top_domain_under_public_suffix")
                else extract_result.registered_domain
            ),
            extract_result.subdomain + "." + extract_result.domain,
            extract_result.domain,
            extract_result.ipv4,
        ],
    ):
        target_candidates = json.loads(
            pass_(
                target,
                arguments.io_encoding,
                arguments.auto_lock,
                arguments.password_prompt_invocation,
            )
        )
        if not target_candidates:
            continue

        candidates = candidates + target_candidates
        if not arguments.merge_candidates:
            break
    else:
        if not candidates:
            stderr('No pass candidates for URL {!r} found!'.format(
                arguments.url))
            return ExitCodes.NO_PASS_CANDIDATES

    if len(candidates) == 1:
        selection = candidates.pop()
    else:
        choices = ['{:s} | {:s}'.format(c['name'], c['login']['username']) for c in candidates]
        choice = dmenu(choices, arguments.dmenu_invocation, arguments.io_encoding)
        choice_tokens = choice.split('|')
        choice_name = choice_tokens[0].strip()
        choice_username = choice_tokens[1].strip()
        selection = next((c for (i, c) in enumerate(candidates)
                          if c['name'] == choice_name
                          and c['login']['username'] == choice_username),
                         None)

    # Nothing was selected, simply return
    if not selection:
        return ExitCodes.SUCCESS

    username = selection['login'].get('username', '')
    
    # Fetch password from rbw
    pw_proc = subprocess.run(['rbw', 'get', selection['id']], capture_output=True, text=True)
    password = pw_proc.stdout.strip()
    
    totp = selection['login'].get('totp', '')

    if arguments.username_only:
        fake_key_raw(username)
    elif arguments.password_only:
        fake_key_raw(password)
    elif arguments.totp_only:
        # No point in moving it to the clipboard in this case
        fake_key_raw(
            get_totp_code(
                selection['id'],
                selection['name'],
                arguments.io_encoding,
                arguments.auto_lock,
                arguments.password_prompt_invocation,
            )
        )
    else:
        # Enter username and password using fake-key and <Tab> (which seems to work almost universally), then switch
        # back into insert-mode, so the form can be directly submitted by
        # hitting enter afterwards
        fake_key_raw(username)
        qute_command('fake-key <Tab>')
        fake_key_raw(password)
        qute_command('fake-key <Return>')

    if arguments.insert_mode:
        qute_command('mode-enter insert')

    # If it finds a TOTP code, it copies it to the clipboard,
    # which is the same behavior as the Firefox add-on.
    if not arguments.totp_only and totp and arguments.totp:
        # The import is done here, to make pyperclip an optional dependency
        import pyperclip
        pyperclip.copy(
            get_totp_code(
                selection['id'],
                selection['name'],
                arguments.io_encoding,
                arguments.auto_lock,
                arguments.password_prompt_invocation,
            )
        )

    return ExitCodes.SUCCESS


if __name__ == '__main__':
    arguments = argument_parser.parse_args()
    sys.exit(main(arguments))
