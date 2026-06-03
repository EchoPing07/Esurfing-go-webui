#!/usr/bin/env python3
"""Build an OpenWrt .ipk package (opkg compatible).

Usage: python3 make_ipk.py <output.ipk> <file1> <file2> ...

An ipk is a gzipped tar archive containing:
  1. debian-binary
  2. data.tar.gz
  3. control.tar.gz

opkg's deb_extract() uses gzip_exec + get_header_tar to walk the outer
archive, so the outer container MUST be tar.gz (not ar).
"""

import os
import sys
import tarfile


def main():
    if len(sys.argv) < 3:
        print(f"Usage: {sys.argv[0]} <output.ipk> <file1> [file2 ...]")
        sys.exit(1)

    output = sys.argv[1]
    files = sys.argv[2:]

    with tarfile.open(output, 'w:gz') as tar:
        for f in files:
            arcname = os.path.basename(f)
            tar.add(f, arcname=arcname)
            print(f"  Added: {arcname} ({os.path.getsize(f)} bytes)")

    print(f"Built: {output} ({os.path.getsize(output)} bytes)")


if __name__ == '__main__':
    main()
