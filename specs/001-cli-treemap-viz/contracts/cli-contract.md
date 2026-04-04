# CLI Contract: codeviz

**Branch**: `001-cli-treemap-viz` | **Date**: 2026-04-04

## Command Synopsis

```
codeviz <target-path> [flags]
```

Single command — no subcommands.

## Arguments

| Argument      | Position | Required | Description               |
| ------------- | -------- | -------- | ------------------------- |
| `target-path` | 1        | Yes      | Path to directory to scan |

## Flags

| Flag               | Short | Type     | Required | Default            | Description                                                                                                            |
| ------------------ | ----- | -------- | -------- | ------------------ | ---------------------------------------------------------------------------------------------------------------------- |
| `--output`         | `-o`  | `string` | Yes      | —                  | Output PNG file path                                                                                                   |
| `--size`           | `-s`  | `enum`   | Yes      | —                  | Metric for rectangle area. Values: `file-size`, `file-lines`, `file-age`, `file-freshness`, `author-count`             |
| `--fill`           | `-f`  | `enum`   | No       | (same as `--size`) | Metric for fill colour. Values: `file-size`, `file-lines`, `file-type`, `file-age`, `file-freshness`, `author-count`   |
| `--fill-palette`   | —     | `enum`   | No       | (metric default)   | Palette for fill colour. Values: `categorization`, `temperature`, `good-bad`, `neutral`                                |
| `--border`         | `-b`  | `enum`   | No       | (none)             | Metric for border colour. Values: `file-size`, `file-lines`, `file-type`, `file-age`, `file-freshness`, `author-count` |
| `--border-palette` | —     | `enum`   | No       | (metric default)   | Palette for border colour. Values: `categorization`, `temperature`, `good-bad`, `neutral`                              |
| `--format`         | —     | `enum`   | No       | `text`             | Diagnostic/error output format. Values: `text`, `json`                                                                 |
| `--verbose`        | `-v`  | `bool`   | No       | `false`            | Enable debug-level logging                                                                                             |
| `--width`          | —     | `int`    | No       | `1920`             | Image width in pixels                                                                                                  |
| `--height`         | —     | `int`    | No       | `1080`             | Image height in pixels                                                                                                 |

## Validation Rules

1. `--size` MUST NOT accept `file-type` (categorical metrics disallowed as size metric).
2. `--fill` includes `file-type` in its valid values; `--size` does not.
3. If `--border-palette` is specified, `--border` MUST also be specified.
4. If `--fill` is not specified, it defaults to the value of `--size`.
5. If `--fill-palette` is not specified, it defaults to the metric's default palette (see FR-010a).
6. If `--border` is not specified, no borders are rendered.
7. If `--border` is specified without `--border-palette`, the metric's default palette is used.
8. Git-required metrics (`file-age`, `file-freshness`, `author-count`) MUST error if `target-path` is not a git repository.

## Exit Codes

| Code | Meaning                                                                |
| ---- | ---------------------------------------------------------------------- |
| 0    | Success — PNG written to `--output` path                               |
| 1    | Invalid arguments or validation failure                                |
| 2    | Target path does not exist or is not a directory                       |
| 3    | Git-required metric used on non-git directory                          |
| 4    | Output path error (parent directory does not exist, permission denied) |
| 5    | Internal error (unexpected failure during scan/render)                 |

## Output Behaviour

### Success (exit 0)

- PNG file written to `--output` path.
- Stdout: Summary line (text mode) or JSON object (json mode).

**Text mode**:
```
Rendered treemap: 1,247 files, 83 directories → output.png (1920×1080)
```

**JSON mode**:
```json
{
  "files": 1247,
  "directories": 83,
  "output": "output.png",
  "width": 1920,
  "height": 1080,
  "size_metric": "file-size",
  "fill_metric": "file-size",
  "fill_palette": "neutral",
  "border_metric": null,
  "border_palette": null
}
```

### Error (exit 1–5)

- Stderr: Error message (text mode) or JSON object (json mode).

**Text mode**:
```
error: metric "file-age" requires a git repository, but "/tmp/notagitrepo" is not a git repository
```

**JSON mode**:
```json
{
  "error": "metric \"file-age\" requires a git repository",
  "target": "/tmp/notagitrepo",
  "code": 3
}
```

### Warnings

- Inaccessible files: logged to stderr, do not affect exit code.
- Palette wrap-around (>12 file types with Categorization): logged to stderr as warning.

**Text mode**:
```
warning: 18 distinct file types exceed categorization palette capacity (12); some types will share colours
warning: skipping /path/to/file: permission denied
```

## Usage Examples

```bash
# Basic: treemap sized by file size, fill defaults to file-size with Neutral palette
codeviz ./myproject -o treemap.png --size file-size

# With fill colour: size by lines, colour by file type
codeviz ./myproject -o treemap.png --size file-lines --fill file-type

# With explicit palette
codeviz ./myproject -o treemap.png --size file-size --fill file-age --fill-palette temperature

# Three dimensions: size + fill + border
codeviz ./myrepo -o treemap.png --size file-size --fill file-age --fill-palette temperature --border author-count --border-palette good-bad

# Git metrics
codeviz ./myrepo -o treemap.png --size file-lines --fill file-freshness

# Custom dimensions, verbose, JSON output
codeviz ./myrepo -o treemap.png --size file-size --width 3840 --height 2160 -v --format json
```
