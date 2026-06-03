#!/usr/bin/env python3
"""Build an OpenWrt .ipk package.

Usage: python3 make_ipk.py <output.ipk> <file1> <file2> ...

An ipk is a GNU ar archive containing:
  1. debian-binary  (content: "2.0\n")
  2. data.tar.gz    (the package payload)
  3. control.tar.gz (package metadata)
"""

import os
import sys


def ar_entry(filename: str, data: bytes) -> bytes:
    """Create a single ar archive entry (header + data + padding)."""
    name = filename.encode('ascii')
    size = len(data)
    mtime = int(os.path.getmtime(filename)) if os.path.exists(filename) else 0

    # ar header: exactly 60 bytes
    #  Name    (16)  | Mtime (12) | UID (6) | GID (6) | Mode (8) | Size (10) | End (2)
    hdr = bytearray(60)
    hdr[0:16]   = name.ljust(16, b' ')
    hdr[16:28]  = str(mtime).encode().ljust(12, b' ')
    hdr[28:34]  = b'0     '
    hdr[34:40]  = b'0     '
    hdr[40:48]  = b'100644  '
    hdr[48:58]  = str(size).encode().ljust(10, b' ')
    hdr[58:60]  = b'\x60\n'

    result = bytes(hdr) + data
    # ar spec: file data must be 2-byte aligned
    if size % 2 == 1:
        result += b'\n'
    return result


def main():
    if len(sys.argv) < 3:
        print(f"Usage: {sys.argv[0]} <output.ipk> <file1> [file2 ...]")
        sys.exit(1)

    output = sys.argv[1]
    files = sys.argv[2:]

    ar_data = b'!<arch>\n'
    for f in files:
        with open(f, 'rb') as fh:
            content = fh.read()
        entry = ar_entry(f, content)
        print(f"  Entry: {f} -> data {len(content)} bytes, entry {len(entry)} bytes")
        ar_data += entry

    with open(output, 'wb') as out:
        out.write(ar_data)

    print(f"Built: {output} ({len(ar_data)} bytes)")


if __name__ == '__main__':
    main()
