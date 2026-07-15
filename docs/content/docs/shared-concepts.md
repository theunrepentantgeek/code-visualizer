---
title: Shared concepts
weight: 8
---

Every visualisation draws on the same vocabulary of metrics, palettes, and
filters. This page collects the concepts that apply across all of the commands.

## Metric values

| Metric                 | Valid for `--size`/`--disc-size` | Valid for `--fill`/`--border` | Description                                         |
| ---------------------- | :------------------------------: | :---------------------------: | --------------------------------------------------- |
| `file-size`            |                ✓                 |               ✓               | File size in bytes                                  |
| `file-lines`           |                ✓                 |               ✓               | Number of non-binary lines                          |
| `file-age`             |            ✓ *(git)*             |           ✓ *(git)*           | Time since first commit (days)                      |
| `file-freshness`       |            ✓ *(git)*             |           ✓ *(git)*           | Time since last commit (days)                       |
| `author-count`         |            ✓ *(git)*             |           ✓ *(git)*           | Number of distinct commit authors                   |
| `commit-count`         |            ✓ *(git)*             |           ✓ *(git)*           | Total number of commits touching the file           |
| `total-lines-added`    |            ✓ *(git)*             |           ✓ *(git)*           | Accumulated lines added across all commits          |
| `total-lines-removed`  |            ✓ *(git)*             |           ✓ *(git)*           | Accumulated lines removed across all commits        |
| `commit-density`       |            ✓ *(git)*             |           ✓ *(git)*           | Commits per month of file lifetime                  |
| `file-type`            |                —                 |               ✓               | File extension category                             |

Metrics marked *(git)* require the target directory to be inside a git repository.
The table above lists the most commonly-used metrics; run `codeviz help metrics`
for the complete set, including source-code metrics and aggregation expressions.

## Palette values

`categorization`, `temperature`, `good-bad`, `neutral`, `foliage`, `terrain`

A palette is chosen by appending it to a colour metric, as in
`--fill file-freshness,temperature`. When you omit the palette, each metric falls
back to its own default. See [Palettes](/docs/palettes) for detailed descriptions
and colour samples.

## Filter rules

Two repeatable flags control which files appear in a visualisation. The
`--include` flag takes a glob that selects matching files, and the `--exclude`
flag takes a glob that removes them. Every file is included by default, so an
`--exclude` glob on its own trims a subset, whereas an `--include` paired with a
catch-all `--exclude '*'` narrows the view to only the files you name.

Include and exclude rules are evaluated together in the order they appear on the
command line, and the first rule that matches a file wins — much like a
`.gitignore`.

```sh
# Include only Go files
codeviz tree-map ./src -o out.png -s file-size --include '*.go' --exclude '*'

# Exclude generated Go files
codeviz tree-map ./src -o out.png -s file-size --exclude '*_gen.go' --exclude '*_gen_test.go'
```

## Selection metrics (user-defined classification)

You can define your own **classification metrics** in the config file. Each
metric assigns a category string to files by matching their relative path against
an ordered list of glob rules — the first match wins.

```yaml
# codeviz.yaml
selectionMetrics:
  code-purpose:
    - category: test
      filename: "*_test.go"
    - category: source
      filename: "*"
  code-source:
    - category: generated
      filename: "*_gen.go"
    - category: authored
      filename: "*"
```

Use the metric name (for example `code-purpose`) anywhere a classification metric
is accepted:

```sh
# Colour tree-map cells by whether each file is a test or source file
codeviz --config codeviz.yaml tree-map ./src -o out.png -s file-size -f code-purpose
```

Selection metrics use the `categorization` palette by default, showing each
distinct category in a unique colour. Files that match no rule receive no colour,
and are rendered in the palette's neutral fallback colour.
