#!/usr/bin/env python3
import os
import re
import sys
import urllib.parse


def main() -> int:
    if len(sys.argv) != 3:
        return 2
    origin, output_path = sys.argv[1:]
    expected = urllib.parse.urlparse(origin)
    data = sys.stdin.read()
    for raw in re.findall(r"https?://[^\s<>\"']+", data):
        raw = raw.rstrip(").,;]}")
        parsed = urllib.parse.urlparse(raw)
        if (
            parsed.scheme == "http"
            and parsed.hostname == expected.hostname
            and parsed.port == expected.port
            and parsed.path == "/authorize"
        ):
            if not os.path.exists(output_path):
                fd = os.open(output_path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
                with os.fdopen(fd, "w") as handle:
                    handle.write(raw)
            return 0
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
