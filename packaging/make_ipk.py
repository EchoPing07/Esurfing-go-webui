#!/usr/bin/env python3
"""Build an OpenWrt .ipk package (ar archive with debian-binary, data.tar.gz, control.tar.gz)."""

import os
import sys
import struct


def ar_header(name: bytes, size: int, mtime: int) -> bytes:
    """Generate a 60-byte ar file header."""
    return (
        name.ljust(16, b' ')
        + str(mtime).encode().ljust(12, b' ')
        + b'0     0     '         # uid + gid (6+6)
        + b'100644 '              # mode (8, includes trailing space)
        + str(size).encode().ljust(10, b' ')
        + b'`\n'                  # magic: 0x60 + 0x0A
    )


def build_ipk(ipk_path: str, files: list[str]) -> None:
    with open(ipk_path, 'wb') as out:
        out.write(b'!<arch>\n')
        for f in files:
            size = os.path.getsize(f)
            mtime = int(os.path.getmtime(f))
            name = os.path.basename(f).encode() + b'/'
            out.write(ar_header(name, size, mtime))
            with open(f, 'rb') as inp:
                out.write(inp.read())
            if size % 2:
                out.write(b'\n')


if __name__ == '__main__':
    ipk_path = sys.argv[1]
    files = sys.argv[2:]
    build_ipk(ipk_path, files)
    print(f"Built: {ipk_path}")
