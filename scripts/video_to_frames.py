#!/usr/bin/env python3
"""Extract sampled PNG frames from a video using ffmpeg."""

import argparse
import os
import shutil
import subprocess
import sys


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Extract sampled frames from a video file"
    )
    parser.add_argument("-i", "--input", required=True, help="Input video file")
    parser.add_argument("-o", "--output", required=True, help="Output directory")
    parser.add_argument(
        "-r",
        "--fps",
        type=float,
        default=8.0,
        help="Sample FPS for extracted frames",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()

    if args.fps <= 0:
        print("fps must be > 0", file=sys.stderr)
        return 2

    if not os.path.isfile(args.input):
        print(f"input video not found: {args.input}", file=sys.stderr)
        return 2

    ffmpeg = shutil.which("ffmpeg")
    if not ffmpeg:
        print("ffmpeg not found in PATH", file=sys.stderr)
        return 2

    os.makedirs(args.output, exist_ok=True)

    output_pattern = os.path.join(args.output, "frame_%06d.png")
    cmd = [
        ffmpeg,
        "-hide_banner",
        "-loglevel",
        "error",
        "-y",
        "-i",
        args.input,
        "-vf",
        f"fps={args.fps:g}",
        output_pattern,
    ]

    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0:
        err = proc.stderr.strip() or "failed to extract frames"
        print(err, file=sys.stderr)
        return proc.returncode

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
