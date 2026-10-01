#!/usr/bin/env python3
"""Write a real replay's data.replay gradually using only Python's standard library."""

import argparse
import math
from pathlib import Path
import shlex
import struct
import tempfile
import time
import zipfile

DEFAULT_SOURCE = Path(__file__).resolve().parent.parent / "tests/fixtures/sample.tbreplay"


def positive_int(value):
    number = int(value)
    if number <= 0:
        raise argparse.ArgumentTypeError("must be greater than zero")
    return number


def nonnegative_float(value):
    number = float(value)
    if not math.isfinite(number) or number < 0:
        raise argparse.ArgumentTypeError("must be a finite nonnegative number")
    return number


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", nargs="?", type=Path, default=DEFAULT_SOURCE,
                        help=".tbreplay archive or standalone data.replay (default: bundled real replay)")
    parser.add_argument("--output", type=Path,
                        help="new destination file; default: a random temporary directory/data.replay")
    parser.add_argument("--chunk-size", type=positive_int, default=1024, help="bytes per write (default: 1024)")
    parser.add_argument("--interval-ms", type=nonnegative_float, default=25, help="delay between writes (default: 25 ms)")
    parser.add_argument("--start-delay", type=nonnegative_float, default=3, help="seconds to start a follower before writing (default: 3)")
    args = parser.parse_args()

    try:
        source = args.source.expanduser().resolve()
        if zipfile.is_zipfile(source):
            with zipfile.ZipFile(source) as archive:
                data = archive.read("data.replay")
        else:
            data = source.read_bytes()
        if data[:4] != struct.pack("<I", 0x12345678):
            parser.error("source does not contain a valid data.replay magic")

        output = args.output.expanduser().resolve() if args.output else Path(
            tempfile.mkdtemp(prefix="blitz-live-replay-")
        ) / "data.replay"
        output.parent.mkdir(parents=True, exist_ok=True)
        # Exclusive creation protects both existing destination files and source replays.
        with output.open("xb") as writer:
            print(f"Source: {source}", flush=True)
            print(f"Output: {output}", flush=True)
            print(f"Follow: cargo run -- replay follow {shlex.quote(str(output))}", flush=True)
            print(f"Writing {len(data)} bytes, {args.chunk_size} bytes/chunk, "
                  f"{args.interval_ms:g} ms between writes; starts in {args.start_delay:g}s", flush=True)
            time.sleep(args.start_delay)
            chunks = 0
            for offset in range(0, len(data), args.chunk_size):
                writer.write(data[offset:offset + args.chunk_size])
                writer.flush()
                chunks += 1
                if offset + args.chunk_size < len(data):
                    time.sleep(args.interval_ms / 1000)
            print(f"Done: {len(data)} bytes in {chunks} chunks. File kept at {output}", flush=True)
    except KeyboardInterrupt:
        print("\nStopped; partially written file kept for inspection.", flush=True)
        return 130
    except (OSError, zipfile.BadZipFile, KeyError) as error:
        parser.error(str(error))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
