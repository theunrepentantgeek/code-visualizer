# Hugo + Hextra Documentation Site Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a landing + docs + gallery + blog site for `codeviz` using Hugo (extended) with the Hextra theme, deployed to GitHub Pages via GitHub Actions.

**Architecture:** A self-contained Hugo site lives in the existing `docs/` directory. Hugo publishes only `docs/content/`, so the planning artifacts in `docs/superpowers/` and `docs/writing-style.md` stay unpublished. Hextra is pulled via Hugo Modules. Gallery images are generated fresh from the `codeviz` binary (reusing `task samples`) so they always reflect current features. New Taskfile tasks drive local preview and build; a `pages.yml` workflow builds and deploys.

**Tech Stack:** Hugo extended v0.164.0, Hextra v0.12.3 (Hugo Module), Hugo Modules (Go 1.26.4), go-task, GitHub Actions, GitHub Pages.

---

## Prerequisites (already done)

- Hugo extended v0.164.0 install added to `.devcontainer/install-dependencies.sh`; `tools/hugo` added to `.gitignore`.
- For local work outside the devcontainer, `tools/hugo` must exist. Install it with:

```bash
os=$(go env GOOS); arch=$(go env GOARCH)
curl -sL "https://github.com/gohugoio/hugo/releases/download/v0.164.0/hugo_extended_0.164.0_${os}-${arch}.tar.gz" | tar xz -C tools hugo
```

All `hugo` invocations below use `./tools/hugo` (or plain `hugo` inside the devcontainer where `tools/` is on `PATH`).

## File Structure

```
docs/
  hugo.yaml                       # Task 1  — site config
  go.mod / go.sum                 # Task 1  — Hugo Modules manifest (Hextra)
  content/
    _index.md                     # Task 5  — landing page
    hero.png                      # generated (gitignored) — landing showcase
    docs/
      _index.md                   # Task 3  — docs section index
      usage.md                    # Task 3  — migrated
      palettes/
        index.md                  # Task 3  — migrated (leaf bundle)
        palette-*.png             # Task 3  — 6 swatch PNGs co-located
      configuration.md            # Task 3  — stub
    gallery/
      _index.md                   # Task 4  — gallery grid
      <viz>.png                   # generated (gitignored) — 5 renders
    blog/
      _index.md                   # Task 6  — blog index
      hello-codeviz.md            # Task 6  — first post
  static/
    .gitkeep                      # Task 1
  superpowers/                    # unchanged — NOT published
  writing-style.md                # unchanged — NOT published
Taskfile.yml                      # Task 2  — docs:gallery / docs:serve / docs:build
.github/workflows/pages.yml       # Task 7  — CI/CD
.gitignore                        # Tasks 1,4,5 — ignore generated artifacts
```

The palette PNGs use a Hugo **leaf bundle** (`palettes/index.md` + co-located images) so existing `![](palette-x.png)` links resolve correctly under the project-page sub-path with no rewriting. Gallery and hero images are placed as **page resources** next to their `_index.md`/`_index.md` bundle for the same reason.

---

## Task 1: Scaffold the Hugo site with the Hextra module

**Files:**
- Create: `docs/hugo.yaml`
- Create: `docs/go.mod`
- Create: `docs/static/.gitkeep`
- Modify: `.gitignore`

- [ ] **Step 1: Create the Hugo Modules manifest**

Create `docs/go.mod`:

```
module github.com/theunrepentantgeek/code-visualizer/docs

go 1.26
```

- [ ] **Step 2: Create the site configuration**

Create `docs/hugo.yaml`:

```yaml
baseURL: "https://theunrepentantgeek.github.io/code-visualizer/"
title: "codeviz"
languageCode: "en-gb"

module:
  imports:
    - path: github.com/imfing/hextra

enableInlineShortcodes: true

markup:
  highlight:
    noClasses: false
  goldmark:
    renderer:
      unsafe: true

menu:
  main:
    - name: Documentation
      pageRef: /docs
      weight: 1
    - name: Gallery
      pageRef: /gallery
      weight: 2
    - name: Blog
      pageRef: /blog
      weight: 3
    - name: Search
      weight: 4
      params:
        type: search
    - name: GitHub
      weight: 5
      url: "https://github.com/theunrepentantgeek/code-visualizer"
      params:
        icon: github

params:
  description: "Visualise the shape of your codebase."
  navbar:
    displayTitle: true
    displayLogo: false
  theme:
    default: system
    displayToggle: true
  editURL:
    enable: true
    base: "https://github.com/theunrepentantgeek/code-visualizer/edit/main/docs/content"
  search:
    enable: true
    type: flexsearch
    flexsearch:
      index: content
```

- [ ] **Step 3: Create the static placeholder**

Create `docs/static/.gitkeep` (empty file) so the directory is tracked.

- [ ] **Step 4: Ignore generated Hugo artifacts**

Add these lines to the end of `.gitignore`:

```
# Hugo documentation site
docs/public/
docs/resources/
docs/.hugo_build.lock
docs/content/gallery/*.png
docs/content/hero.png
```

- [ ] **Step 5: Fetch the Hextra module and lock versions**

Run:

```bash
cd docs && ../tools/hugo mod get github.com/imfing/hextra@v0.12.3 && cd ..
```

Expected: `docs/go.mod` gains a `require github.com/imfing/hextra v0.12.3` line and `docs/go.sum` is created.

- [ ] **Step 6: Verify the site builds with a temporary home page**

Create a throwaway home page so the build has content, then build:

```bash
printf -- '---\ntitle: codeviz\n---\n' > docs/content/_index.md
cd docs && ../tools/hugo --gc && cd ..
```

Expected: build succeeds, `Pages` count ≥ 1, exit code 0. Remove the throwaway file afterwards: `rm docs/content/_index.md`.

- [ ] **Step 7: Commit**

```bash
git add docs/hugo.yaml docs/go.mod docs/go.sum docs/static/.gitkeep .gitignore
git commit -m "feat(docs): scaffold Hugo site with Hextra theme"
```

---

## Task 2: Add Taskfile tasks for gallery generation, serve, and build

**Files:**
- Modify: `Taskfile.yml`

- [ ] **Step 1: Add the docs tasks**

Add the following three tasks to `Taskfile.yml` (place them after the `samples:` task). `HUGO` points at the locally installed binary; `CODEVIZ`/`FOOTER` mirror the `samples` task.

```yaml
  docs:gallery:
    desc: Generate gallery + hero images for the docs site from fresh samples
    deps:
      - samples
    cmds:
      - mkdir -p docs/content/gallery
      - for:
          matrix:
            VIZ: [tree-map, bubble-tree, radial-tree, spiral, scatter]
        cmd: cp samples/{{.ITEM.VIZ}}/code-visualizer.png docs/content/gallery/{{.ITEM.VIZ}}.png
      - cp samples/tree-map/code-visualizer.png docs/content/hero.png

  docs:serve:
    desc: Serve the documentation site locally with live reload
    deps:
      - docs:gallery
    dir: docs
    cmds:
      - "{{.HUGO}} mod get"
      - "{{.HUGO}} server --buildDrafts"
    vars:
      HUGO: '{{joinPath .ROOT_DIR "tools" "hugo"}}'

  docs:build:
    desc: Build the documentation site into docs/public
    deps:
      - docs:gallery
    dir: docs
    cmds:
      - "{{.HUGO}} mod get"
      - "{{.HUGO}} --minify --gc"
    vars:
      HUGO: '{{joinPath .ROOT_DIR "tools" "hugo"}}'
```

- [ ] **Step 2: Verify the gallery task generates images**

Run:

```bash
./tools/task docs:gallery
ls docs/content/gallery/ docs/content/hero.png
```

Expected: `docs/content/gallery/` contains `tree-map.png`, `bubble-tree.png`, `radial-tree.png`, `spiral.png`, `scatter.png`; `docs/content/hero.png` exists. All are gitignored (Task 1, Step 4).

- [ ] **Step 3: Commit**

```bash
git add Taskfile.yml
git commit -m "feat(docs): add docs:gallery, docs:serve, docs:build tasks"
```

---

## Task 3: Migrate reference documentation

**Files:**
- Create: `docs/content/docs/_index.md`
- Create: `docs/content/docs/usage.md` (from `docs/usage.md`)
- Create: `docs/content/docs/palettes/index.md` (from `docs/palettes.md`)
- Move: `docs/palette-*.png` → `docs/content/docs/palettes/`
- Create: `docs/content/docs/configuration.md`
- Delete: `docs/usage.md`, `docs/palettes.md`

- [ ] **Step 1: Create the docs section index**

Create `docs/content/docs/_index.md`:

```markdown
---
title: Documentation
weight: 1
---

Reference documentation for `codeviz` — the command-line tool that renders the
shape of a codebase as tree-maps, radial trees, bubble trees, spirals, and
scatter plots.

{{< cards >}}
  {{< card link="usage" title="Usage" subtitle="Commands, flags, and configuration files." >}}
  {{< card link="palettes" title="Palettes" subtitle="The built-in colour palettes and when to use each." >}}
  {{< card link="configuration" title="Configuration" subtitle="Configuration-file reference." >}}
{{< /cards >}}
```

- [ ] **Step 2: Migrate the usage page**

Move the content of `docs/usage.md` into a new file `docs/content/docs/usage.md`, prepending Hextra front-matter. The body is copied verbatim except that the existing top-level `# codeviz Usage` heading is removed (Hextra renders the title from front-matter).

```bash
{ printf -- '---\ntitle: Usage\nweight: 1\n---\n\n'; tail -n +2 docs/usage.md; } > docs/content/docs/usage.md
```

Then open `docs/content/docs/usage.md` and confirm the first content line is the `## Synopsis` heading (not a duplicate H1). `usage.md` contains no images, so no link rewriting is required.

- [ ] **Step 3: Migrate the palettes page as a leaf bundle**

Create the bundle directory, move the images, and create `index.md`:

```bash
mkdir -p docs/content/docs/palettes
git mv docs/palette-categorization.png docs/palette-temperature.png docs/palette-good-bad.png \
       docs/palette-neutral.png docs/palette-foliage.png docs/palette-terrain.png \
       docs/content/docs/palettes/
{ printf -- '---\ntitle: Palettes\nweight: 2\n---\n\n'; tail -n +2 docs/palettes.md; } > docs/content/docs/palettes/index.md
```

The existing image links in `palettes.md` are of the form `![...](palette-categorization.png)`. Because the images are now co-located in the leaf bundle, these relative links resolve unchanged — no rewriting needed.

- [ ] **Step 4: Create the configuration stub**

Create `docs/content/docs/configuration.md`:

```markdown
---
title: Configuration
weight: 3
---

`codeviz` reads an optional configuration file (`.yaml`, `.yml`, or `.json`)
supplied with the `--config` flag. This page documents the available keys.

{{< callout type="info" >}}
This reference is being expanded. For the authoritative list of flags, run
`codeviz <visualization> --help`, and see the [Usage](../usage) page.
{{< /callout >}}
```

- [ ] **Step 5: Remove the originals**

```bash
git rm docs/usage.md docs/palettes.md
```

- [ ] **Step 6: Verify the docs build and links resolve**

```bash
cd docs && ../tools/hugo --gc --printPathWarnings 2>&1 | tee /tmp/hugo-docs.log; cd ..
grep -Ei "ERROR|WARN|REF_NOT_FOUND" /tmp/hugo-docs.log || echo "no errors/warnings"
```

Expected: build succeeds, exit code 0, no `ERROR`/`WARN`/`REF_NOT_FOUND` lines. Confirm the palette images copied into the output:

```bash
ls docs/public/docs/palettes/ | grep palette-categorization.png
```

Expected: the image appears in the built output.

- [ ] **Step 7: Commit**

```bash
git add docs/content/docs Taskfile.yml
git rm --cached docs/usage.md docs/palettes.md 2>/dev/null || true
git commit -m "docs: migrate usage and palettes into Hugo content"
```

---

## Task 4: Build the gallery page

**Files:**
- Create: `docs/content/gallery/_index.md`

- [ ] **Step 1: Ensure gallery images exist**

```bash
./tools/task docs:gallery
```

Expected: `docs/content/gallery/{tree-map,bubble-tree,radial-tree,spiral,scatter}.png` exist (gitignored).

- [ ] **Step 2: Create the gallery page**

Create `docs/content/gallery/_index.md`. Each entry references its co-located generated image (page resource), so links resolve under the project sub-path. Prose follows `docs/writing-style.md` (British English, em dashes, no contractions).

```markdown
---
title: Gallery
weight: 2
---

Every image below is rendered from this very repository — regenerated on each
build, so what you see reflects the current feature set rather than a
hand-picked snapshot.

## Tree-map

A space-filling map where each rectangle is a file, sized by a metric of your
choosing. Best when you want density at a glance.

![Tree-map visualisation](tree-map.png)

## Bubble-tree

Nested circles that trade some space efficiency for a softer, more organic read
of the hierarchy.

![Bubble-tree visualisation](bubble-tree.png)

## Radial-tree

The directory tree unrolled around a centre point — structure and depth become
immediately legible.

![Radial-tree visualisation](radial-tree.png)

## Spiral

Files laid along a spiral, rewarding a scan from the centre outwards when
ordering matters more than grouping.

![Spiral visualisation](spiral.png)

## Scatter

Two metrics plotted against one another, with an optional logarithmic axis —
the view to reach for when you are hunting correlations.

![Scatter visualisation](scatter.png)
```

- [ ] **Step 3: Verify the gallery renders**

```bash
cd docs && ../tools/hugo --gc && cd ..
ls docs/public/gallery/ | grep -E "tree-map.png|scatter.png"
```

Expected: build succeeds; both images appear in `docs/public/gallery/`.

- [ ] **Step 4: Commit**

```bash
git add docs/content/gallery/_index.md
git commit -m "feat(docs): add gallery page"
```

---

## Task 5: Build the landing page

**Files:**
- Create: `docs/content/_index.md`

- [ ] **Step 1: Ensure the hero image exists**

```bash
./tools/task docs:gallery
ls docs/content/hero.png
```

Expected: `docs/content/hero.png` exists (gitignored).

- [ ] **Step 2: Create the landing page**

Create `docs/content/_index.md` using Hextra's home layout. Prose follows
`docs/writing-style.md`.

```markdown
---
title: codeviz
layout: hextra-home
---

{{< hextra/hero-headline >}}
  See the shape of your codebase
{{< /hextra/hero-headline >}}

{{< hextra/hero-subtitle >}}
  codeviz turns a directory tree into a picture — tree-maps, radial trees,
  bubble trees, spirals, and scatter plots — so structure, size, and hot spots
  become obvious at a glance.
{{< /hextra/hero-subtitle >}}

{{< hextra/hero-button text="Get started" link="docs/usage" >}}

<div class="hx-mt-6"></div>

![A tree-map of this repository](hero.png)

{{< hextra/feature-grid >}}
  {{< hextra/feature-card title="Five visualisations" subtitle="Tree-map, radial tree, bubble tree, spiral, and scatter — one binary, one command." >}}
  {{< hextra/feature-card title="Thoughtful palettes" subtitle="Built-in colour palettes for categorisation, temperature, terrain, and more." >}}
  {{< hextra/feature-card title="Git-aware" subtitle="Reads repository metadata to weight and colour what actually matters." >}}
  {{< hextra/feature-card title="Fast and scriptable" subtitle="A single Go binary with sensible defaults and configuration files when you need them." >}}
{{< /hextra/feature-grid >}}
```

- [ ] **Step 3: Verify the landing page builds**

```bash
cd docs && ../tools/hugo --gc && cd ..
test -f docs/public/index.html && grep -q "See the shape of your codebase" docs/public/index.html && echo OK
```

Expected: prints `OK`.

- [ ] **Step 4: Commit**

```bash
git add docs/content/_index.md
git commit -m "feat(docs): add landing page"
```

---

## Task 6: Build the blog / changelog section

**Files:**
- Create: `docs/content/blog/_index.md`
- Create: `docs/content/blog/hello-codeviz.md`

- [ ] **Step 1: Create the blog index**

Create `docs/content/blog/_index.md`:

```markdown
---
title: Blog
weight: 3
---
```

- [ ] **Step 2: Create the first post**

Create `docs/content/blog/hello-codeviz.md`. Prose follows `docs/writing-style.md`.

```markdown
---
title: "A home for codeviz"
date: 2026-07-12
authors:
  - name: Bevan Arps
    link: https://github.com/theunrepentantgeek
---

Documentation deserves better than a single sprawling README. This site is the
new home for `codeviz` — a place where the usage reference, the palette guide,
and a gallery of live renders can each breathe.

The gallery is the part I am most pleased with. Every image is regenerated from
this repository on each build, so it can never drift out of step with the code.
When a new visualisation lands, its picture appears here automatically — no
stale screenshots, no manual bookkeeping.

More to come as the tool grows. For now, browse the [gallery](../gallery) and
skim the [usage guide](../docs/usage).
```

- [ ] **Step 3: Verify the blog builds**

```bash
cd docs && ../tools/hugo --gc && cd ..
test -f docs/public/blog/hello-codeviz/index.html && echo OK
```

Expected: prints `OK`.

- [ ] **Step 4: Commit**

```bash
git add docs/content/blog
git commit -m "feat(docs): add blog section with first post"
```

---

## Task 7: Add the GitHub Pages deployment workflow

**Files:**
- Create: `.github/workflows/pages.yml`

- [ ] **Step 1: Create the workflow**

Create `.github/workflows/pages.yml`. Note the step order: `Setup Pages` (id
`pages`) runs *before* `Build site` so the resolved `base_url` is available when
Hugo bakes in the `baseURL`.

```yaml
name: Deploy documentation site

on:
  push:
    branches: [main]
    paths:
      - "docs/**"
      - "samples/**"
      - "cmd/**"
      - "internal/**"
      - "go.mod"
      - "go.sum"
      - ".github/workflows/pages.yml"
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: false

env:
  HUGO_VERSION: 0.164.0

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version-file: .go-version

      - name: Install Hugo (extended)
        run: |
          curl -sL "https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_linux-amd64.tar.gz" \
            | sudo tar xz -C /usr/local/bin hugo

      - name: Install Task
        uses: arduino/setup-task@v2
        with:
          version: 3.x
          repo-token: ${{ secrets.GITHUB_TOKEN }}

      - name: Setup Pages
        id: pages
        uses: actions/configure-pages@v5

      - name: Generate gallery images
        run: |
          mkdir -p tools
          cp "$(command -v hugo)" tools/hugo
          task docs:gallery

      - name: Build site
        working-directory: docs
        run: |
          hugo mod get
          hugo --minify --gc --baseURL "${{ steps.pages.outputs.base_url }}/"

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: docs/public

  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

- [ ] **Step 2: Validate the workflow YAML**

Run:

```bash
python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/pages.yml')); print('valid yaml')"
```

Expected: prints `valid yaml`.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/pages.yml
git commit -m "ci(docs): deploy documentation site to GitHub Pages"
```

- [ ] **Step 4: One-time repository setup (manual, cannot be scripted in this repo)**

In the GitHub UI: **Settings → Pages → Build and deployment → Source → GitHub Actions**. Document this in the PR description so the reviewer completes it before/after merge.

---

## Task 8: Final full-site verification

**Files:** none (verification only)

- [ ] **Step 1: Clean build from scratch**

```bash
rm -rf docs/public docs/resources
./tools/task docs:build
```

Expected: exit code 0.

- [ ] **Step 2: Confirm every top-level section rendered**

```bash
for p in index.html docs/usage/index.html docs/palettes/index.html \
         docs/configuration/index.html gallery/index.html blog/index.html \
         blog/hello-codeviz/index.html; do
  test -f "docs/public/$p" && echo "OK  $p" || echo "MISSING  $p"
done
```

Expected: seven `OK` lines, no `MISSING`.

- [ ] **Step 3: Confirm search index and generated images shipped**

```bash
ls docs/public/gallery/*.png | wc -l          # expect 5
find docs/public -name "*.json" | grep -qi flex && echo "search index present" || echo "check search index name"
```

Expected: `5`, and a FlexSearch index JSON is present (name may vary by Hextra version; confirm a search index JSON exists under `docs/public/`).

- [ ] **Step 4: Confirm unpublished material stayed unpublished**

```bash
test ! -e docs/public/superpowers && test ! -e docs/public/writing-style/index.html && echo "OK: planning artifacts not published"
```

Expected: prints the OK line.

- [ ] **Step 5: Run the existing CI to ensure nothing regressed**

Run `task ci` via an Explore subagent (per repository workflow rules) and confirm it passes. The docs site is independent of the Go build, so `task ci` should be unaffected.

- [ ] **Step 6: Final commit (if any verification tweaks were needed)**

```bash
git add -A
git commit -m "test(docs): verify full site build" || echo "nothing to commit"
```

---

## Self-Review Notes

- **Spec coverage:** architecture/layout (Task 1), content migration (Task 3), theme install + config (Task 1), search (Task 1 config), gallery generation reusing `task samples` (Tasks 2, 4), landing (Task 5), blog (Task 6), Taskfile tasks (Task 2), CI/CD to Pages (Task 7), non-goals respected (no versioning/i18n/Algolia/custom domain), writing-style adherence (Tasks 4–6 prose). Devcontainer Hugo install was completed ahead of the plan.
- **Deviation from spec (intentional):** the spec placed palette PNGs under `docs/assets/palettes/` and gallery images under `docs/assets/gallery/`. This plan instead uses Hugo **page bundles** (`content/docs/palettes/` leaf bundle; images co-located with the gallery/home `_index.md`). Page bundles make `![](x.png)` links resolve correctly under the GitHub Pages project sub-path without URL rewriting or `assets` pipeline shortcodes — the correct Hugo mechanism for the stated goal.
- **Verification model:** because this is a docs site rather than library code, each task's "test" is a Hugo build plus concrete output-file/text assertions, which is the honest correctness check here.
