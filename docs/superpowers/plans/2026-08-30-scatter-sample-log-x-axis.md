# Scatter Sample Logarithmic X Axis Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render the checked-in scatter sample with a logarithmic horizontal axis so file-size points are spread more clearly.

**Architecture:** Configure the existing scatter logarithmic scale through the sample YAML rather than changing application code. Keep the sample explanation synchronized with the configuration, then use the repository's existing deterministic sample task to regenerate both image formats.

**Tech Stack:** YAML, Markdown, Task, Go `codeviz` CLI

---

## File Structure

| File | Responsibility |
| --- | --- |
| `samples/scatter/code-visualizer.yml` | Defines metrics and axis scales for the checked-in scatter sample. |
| `samples/scatter/README.md` | Explains what the sample visualizes. |
| `samples/scatter/code-visualizer.png` | Generated raster sample. |
| `samples/scatter/code-visualizer.svg` | Generated vector sample. |

### Task 1: Configure and regenerate the scatter sample

**Files:**
- Modify: `samples/scatter/code-visualizer.yml:12-22`
- Modify: `samples/scatter/README.md:8-20`
- Modify: `samples/scatter/code-visualizer.png`
- Modify: `samples/scatter/code-visualizer.svg`

- [ ] **Step 1: Add the logarithmic X-axis configuration**

Insert `xScale: log` directly after the X-axis metric:

```yaml
scatter:
    xAxis: file-size
    xScale: log
    yAxis: comment-ratio
```

- [ ] **Step 2: Update the sample explanation**

Change the X-axis table row and explanatory paragraph to:

```markdown
| X axis          | `file-size` (log scale) | - |

Each point is one file: its position compares file size on a logarithmic scale
against comment ratio, spreading files according to their order of magnitude.
Point size and colour surface how many declarations (and public declarations)
the file contains.
```

Preserve the table's existing alignment style and em dash character.

- [ ] **Step 3: Regenerate both sample images**

Run:

```bash
task samples-scatter
```

Expected: exit status 0; the command rebuilds `bin/codeviz` and writes both
`samples/scatter/code-visualizer.png` and
`samples/scatter/code-visualizer.svg`.

- [ ] **Step 4: Verify the focused change**

Run:

```bash
git diff --check
git status --short
```

Expected: no whitespace errors; only the sample YAML, README, PNG, and SVG are
modified beyond the already committed design and plan documents.

- [ ] **Step 5: Commit**

```bash
git add samples/scatter/code-visualizer.yml \
  samples/scatter/README.md \
  samples/scatter/code-visualizer.png \
  samples/scatter/code-visualizer.svg
git commit -m "Use logarithmic X axis in scatter sample" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```
