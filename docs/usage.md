# codeviz Usage

## Synopsis

```
codeviz <target-path> [flags]
```

## Required Flags

| Flag       | Short | Description                        |
|------------|-------|------------------------------------|
| `--output` | `-o`  | Output PNG file path               |
| `--size`   | `-s`  | Metric for rectangle area          |

### Size metric values

`file-size`, `file-lines`, `file-age`, `file-freshness`, `author-count`

## Optional Flags

| Flag               | Short | Default          | Description                  |
|--------------------|-------|------------------|------------------------------|
| `--fill`           | `-f`  | same as `--size` | Metric for fill colour       |
| `--fill-palette`   |       | metric default   | Palette for fill colour      |
| `--border`         | `-b`  | none             | Metric for border colour     |
| `--border-palette` |       | metric default   | Palette for border colour    |
| `--width`          |       | 1920             | Image width in pixels        |
| `--height`         |       | 1080             | Image height in pixels       |
| `--verbose`        | `-v`  | false            | Enable debug-level logging   |
| `--format`         |       | text             | Output format (text or json) |

### Metric values

`file-size`, `file-lines`, `file-type`, `file-age`, `file-freshness`, `author-count`

Note: `file-type` is only valid for `--fill` and `--border`, not `--size`.

### Palette values

`categorization`, `temperature`, `good-bad`, `neutral`

## Examples

### Basic treemap by file size

```sh
codeviz ./src -o treemap.png -s file-size
```

### Colour by file type

```sh
codeviz ./src -o treemap.png -s file-size -f file-type
```

### File lines with temperature palette

```sh
codeviz ./src -o treemap.png -s file-lines -f file-lines --fill-palette temperature
```

### Git freshness with border showing author count

```sh
codeviz ./src -o treemap.png -s file-lines -f file-freshness -b author-count
```

### Custom dimensions and JSON output

```sh
codeviz ./src -o treemap.png -s file-size --width 3840 --height 2160 --format json
```

### Verbose mode for debugging

```sh
codeviz ./src -o treemap.png -s file-size -v
```

## Exit Codes

| Code | Meaning                                            |
|------|----------------------------------------------------|
| 0    | Success — PNG written to output path               |
| 1    | Invalid arguments or validation failure            |
| 2    | Target path does not exist or is not a directory   |
| 3    | Git-required metric used on non-git directory      |
| 4    | Output path error (parent missing, permission)     |
| 5    | Internal error during scan/render                  |
| 6    | No files available after filtering (e.g. all binary) |
