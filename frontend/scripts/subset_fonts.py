#!/usr/bin/env python3

from __future__ import annotations

import argparse
import glob
import importlib.util
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path


TEXT_EXTENSIONS = {
    ".css",
    ".csv",
    ".htm",
    ".html",
    ".js",
    ".json",
    ".md",
    ".svg",
    ".txt",
    ".xml",
    "",
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Subset locally hosted fonts in frontend/public during build."
    )
    parser.add_argument(
        "--root",
        default=".",
        help="Project root that config paths are resolved from.",
    )
    parser.add_argument(
        "--config",
        default="frontend/font-subset.config.json",
        help="Path to font subset config JSON.",
    )
    return parser.parse_args()


def load_config(config_path: Path) -> dict:
    if not config_path.exists():
        return {"fonts": []}
    with config_path.open("r", encoding="utf-8") as fh:
        return json.load(fh)


def default_env(root: Path) -> dict[str, str]:
    env = dict(os.environ)
    env.setdefault("PUBLIC_DIR", str((root / "public").resolve()))
    env.setdefault("SNAPSHOT_DIR", str((root / ".font-subset-empty").resolve()))
    return env


def resolve_value(root: Path, value: str, env: dict[str, str]) -> Path:
    expanded = os.path.expandvars(value)
    path = Path(expanded)
    if path.is_absolute():
        return path
    return (root / path).resolve()


def resolve_paths(root: Path, patterns: list[str], env: dict[str, str]) -> list[Path]:
    matches: list[Path] = []
    seen: set[Path] = set()
    for pattern in patterns:
        expanded = os.path.expandvars(pattern)
        glob_pattern = expanded if os.path.isabs(expanded) else str(root / expanded)
        for raw in glob.glob(glob_pattern, recursive=True):
            path = Path(raw)
            if not path.is_file():
                continue
            if path in seen:
                continue
            seen.add(path)
            matches.append(path)
    return matches


def extract_text(paths: list[Path], always_include: str) -> str:
    chunks: list[str] = []
    if always_include:
        chunks.append(always_include)
    for path in paths:
        if path.suffix.lower() not in TEXT_EXTENSIONS:
            continue
        chunks.append(path.read_text(encoding="utf-8", errors="ignore"))

    seen: set[str] = set()
    ordered: list[str] = []
    for chunk in chunks:
        for char in chunk:
            if char in seen:
                continue
            seen.add(char)
            ordered.append(char)
    return "".join(ordered)


def subset_font(root: Path, item: dict) -> None:
    env = default_env(root)
    input_path = resolve_value(root, item["input"], env)
    output_path = resolve_value(root, item["output"], env)
    if not input_path.exists():
        raise FileNotFoundError(f"font input not found: {input_path}")

    text_sources = resolve_paths(root, item.get("text_sources", []), env)
    subset_text = extract_text(text_sources, item.get("always_include", ""))
    if not subset_text:
        raise RuntimeError(f"no subset text collected for {input_path}")

    output_path.parent.mkdir(parents=True, exist_ok=True)

    with tempfile.NamedTemporaryFile("w", encoding="utf-8", delete=False) as fh:
        fh.write(subset_text)
        text_file = fh.name

    try:
        cmd = [
            sys.executable,
            "-m",
            "fontTools.subset",
            str(input_path),
            f"--text-file={text_file}",
            f"--output-file={output_path}",
            f"--flavor={item.get('flavor', 'woff2')}",
            "--passthrough-tables",
        ]

        layout_features = item.get("layout_features", [])
        if layout_features:
            cmd.append("--layout-features=" + ",".join(layout_features))
        if item.get("drop_hinting", False):
            cmd.append("--no-hinting")

        subprocess.run(cmd, check=True)
    finally:
        os.unlink(text_file)

    if item.get("delete_inputs_after_build", False):
        try:
            if input_path.resolve() != output_path.resolve():
                input_path.unlink()
        except FileNotFoundError:
            pass


def main() -> int:
    args = parse_args()
    root = Path(args.root).resolve()
    config_path = Path(args.config)
    if not config_path.is_absolute():
        config_path = (root / config_path).resolve()
    config = load_config(config_path)
    fonts = config.get("fonts", [])
    if not fonts:
        return 0

    if importlib.util.find_spec("fontTools.subset") is None:
        raise RuntimeError("fontTools.subset not found; install fonttools before running this script")

    for item in fonts:
        subset_font(root, item)
    return 0


if __name__ == "__main__":
    sys.exit(main())
