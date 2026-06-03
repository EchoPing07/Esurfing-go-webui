#!/usr/bin/env python3
"""Build an OpenWrt .ipk package (opkg compatible).

Usage: python3 make_ipk.py <output.ipk> <file1> <file2> ...

An ipk is a GNU ar archive containing (in order):
  1. debian-binary
  2. data.tar.gz
  3. control.tar.gz
"""

import os
import sys


def ar_entry(name: str, data: bytes) -> bytes:
    """Create one ar archive entry: 60-byte header + data + alignment padding."""
    size = len(data)

    # ar header layout (exactly 60 bytes):
    #   Offset  Len  Field
    #   0       16   Name (space padded)
    #   16      12   Mtime (decimal, space padded)
    #   28      6    UID  (decimal, space padded, "0")
    #   34      6    GID  (decimal, space padded, "0")
    #   40      8    Mode (octal, space padded, "100644 ")
    #   48      10   Size (decimal, space padded)
    #   58      2    Magic "`\n"
    fields = bytearray(60)
    fields[0:16]   = name.encode().ljust(16, b' ')[0:16]
    fields[16:28]  = b'0'.ljust(12, b' ')
    fields[28:34]  = b'0'.ljust(6, b' ')
    fields[34:40]  = b'0'.ljust(6, b' ')
    fields[40:48]  = b'100644 '.ljust(8, b' ')[0:8]
    fields[48:58]  = str(size).encode().ljust(10, b' ')[0:10]
    fields[58:60]  = b'`\n'

    result = bytes(fields) + data
    # ar spec: file data must be 2-byte aligned
    if size % 2 != 0:
        result += b'\n'
    return result


def main():
    if len(sys.argv) < 3:
        print(f"Usage: {sys.argv[0]} <output.ipk> <file1> [file2 ...]")
        sys.exit(1)

    output = sys.argv[1]
    files = sys.argv[2:]

    # Collect all entry data first for total size
    entries_data = []
    total = len(b'!<arch>\n')
    for f in files:
        with open(f, 'rb') as fh:
            content = fh.read()
        entry = ar_entry(os.path.basename(f), content)
        entries_data.append(entry)
        total += len(entry)
        print(f"  {f}: content={len(content)}B, entry={len(entry)}B")

    with open(output, 'wb') as out:
        out.write(b'!<arch>\n')
        for entry in entries_data:
            out.write(entry)

    print(f"Built: {output} ({total} bytes)")


if __name__ == '__main__':
    main()
