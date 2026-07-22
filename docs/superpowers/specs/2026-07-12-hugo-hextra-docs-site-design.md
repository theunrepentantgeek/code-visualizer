# Design: Documentation Site with Hugo + Hextra

**Date:** 2026-07-12
**Status:** Approved (design)
**Author:** Bevan Arps (with Copilot)

## Summary

Build a documentation and marketing site for `codeviz` using **Hugo** (extended)
with the **Hextra** theme, hosted on **GitHub Pages** and deployed automatically
via **GitHub Actions**. The site covers a landing page, reference documentation,
a gallery of rendered sample visualisations, and a blog/changelog.

Hugo + Hextra was chosen because it is the only option that satisfies every hard
constraint simultaneously: a single Go binary (no Node toolchain), best-in-class
build performance, native Markdown authoring, and a modern all-in-one theme that
already provides landing, docs, gallery, and blog layouts out of the box. Astro +
Starlight was the strongest alternative but was rejected because it introduces a
Node toolchain both locally and in CI.

## Goals

- A cohesive, performant, modern documentation site anyone can navigate.
- Landing page, reference docs, gallery, and blog/changelog.
- Zero Node toolchain — Hugo extended (single Go binary) only.
- Markdown-native authoring.
- Automated build and deploy to GitHub Pages via GitHub Actions.
- Gallery images generated fresh from the `codeviz` binary so they always reflect
  current features.

## Non-Goals (YAGNI)

- No versioned documentation.
- No internationalisation (i18n).
- No external search service (Algolia); Hextra's built-in FlexSearch suffices.
- No custom domain (project-page URL for now).
- No redesign of the tool's rendered output — the gallery displays existing renders.

## Architecture & Directory Layout

The Hugo site lives in the existing top-level `docs/` directory. Hugo publishes
only the `content/` subtree, so the existing planning artifacts in
`docs/superpowers/` and the meta file `docs/writing-style.md` remain unpublished.

```
docs/
  hugo.yaml                # site config (baseURL, Hextra module, menus, search)
  go.mod / go.sum          # Hugo Modules manifest (fetches Hextra)
  content/
    _index.md              # landing page (hero + feature cards + sample showcase)
    docs/
      _index.md
      usage.md             # migrated from docs/usage.md
      palettes.md          # migrated from docs/palettes.md
      configuration.md     # stub: config-file / flags reference (room to grow)
    gallery/
      _index.md            # image grid of each visualisation type
    blog/
      _index.md
      <first-post>.md      # changelog / announcement seed
  assets/
    palettes/              # 6 palette swatch PNGs (moved from docs/*.png)
    gallery/               # generated sample renders (gitignored)
  static/                  # favicon, etc.
  public/                  # Hugo build output (gitignored)
  superpowers/             # existing plans & specs — NOT published
  writing-style.md         # meta file — NOT published
```

### Key Decisions

- **Hextra installed via Hugo Modules**, not a git submodule. The repository
  already has Go 1.26.4, so `hugo mod get` pulls the theme and `hugo mod get -u`
  keeps it updated (Dependabot-friendly). The `docs/go.mod` for Hugo Modules is
  independent of the repository-root `go.mod` for the Go application; they live in
  different directories and do not conflict.
- **`docs/` is the single source of truth** for published content. The existing
  `docs/usage.md` and `docs/palettes.md` are *moved* into `docs/content/docs/` so
  there is no duplication.
- The tool's rendered sample images feed both the gallery and the landing
  showcase, generated fresh at build time (see Build Pipeline).

## Content Mapping & Migration

| Source today | Destination | Notes |
|---|---|---|
| `docs/usage.md` | `docs/content/docs/usage.md` | Add Hextra front-matter (`title`, `weight`); fix relative image links |
| `docs/palettes.md` | `docs/content/docs/palettes.md` | Same; palette PNGs referenced from `assets/palettes/` |
| `docs/palette-*.png` (6) | `docs/assets/palettes/` | Swatch images |
| `samples/<viz>/code-visualizer.png` | `docs/assets/gallery/` (generated) | Fed from `task samples`, gitignored |
| — new — | `docs/content/_index.md` | Landing: hero, feature cards, showcase images |
| — new — | `docs/content/gallery/_index.md` | Image grid of each visualisation type |
| — new — | `docs/content/blog/_index.md` + first post | Changelog / announcement seed |
| — new — | `docs/content/docs/configuration.md` | Stub for config-file / flags reference |

- **Front-matter**: Hextra uses `title` + `weight` for sidebar ordering. The
  landing page uses Hextra's `hextra-home` layout with hero/feature shortcodes.
- **Link hygiene**: internal document links are rewritten to Hugo `ref`/`relref`
  so the CI build fails loudly on any broken link or missing image.

## Theme Installation & Site Configuration

- **Hextra via Hugo Modules**: `docs/go.mod` requires `github.com/imfing/hextra`;
  `hugo.yaml` sets `module.imports`. Updated with `hugo mod get -u ./...`.
- **`baseURL`**: `https://theunrepentantgeek.github.io/code-visualizer/` — the
  project-page subpath. A future custom domain swaps `baseURL` and adds a `CNAME`.
- **Search**: Hextra's built-in FlexSearch (client-side, no external service).
- **Enabled Hextra features**: dark/light toggle, top navigation
  (Docs · Gallery · Blog · GitHub link), left sidebar for docs, syntax
  highlighting, and edit-on-GitHub links.
- **Menus**: top-nav and docs sidebar driven by front-matter `weight`.
- **Production build**: `hugo --minify --gc`.

## Build Pipeline, CI/CD & Local Workflow

### Gallery Generation

Gallery images are generated fresh so they always reflect current features,
reusing the existing `task samples` machinery (which builds `codeviz` and renders
PNG + SVG for all five visualisations against this repository into
`samples/<viz>/`).

Flow: `task samples` → copy fresh `samples/*/code-visualizer.png` into
`docs/assets/gallery/` → Hugo build.

### New Taskfile Tasks

Namespaced to follow the existing `fmt:check` / `mod:check` convention:

- `docs:gallery` — `deps: [samples]`; copies fresh sample PNGs into
  `docs/assets/gallery/`.
- `docs:serve` — `deps: [docs:gallery]`; runs `hugo server` from `docs/` for
  local preview.
- `docs:build` — `deps: [docs:gallery]`; runs `hugo --minify --gc` producing
  `docs/public/`.

`docs/public/` and `docs/assets/gallery/` are gitignored (generated artifacts).

### CI/CD — new `.github/workflows/pages.yml`

- **Triggers**: push to `main` touching `docs/**`, `samples/**`, or the Go source
  (so feature changes refresh the gallery); plus `workflow_dispatch`.
- **Build job**: checkout → setup Go (from `.go-version`) → setup Hugo extended
  (pinned version) → `task docs:gallery` (builds `codeviz` and renders) →
  `hugo mod get` → `hugo --minify --baseURL <pages-url>` → upload Pages artifact.
- **Deploy job**: `actions/deploy-pages` with `pages: write` and
  `id-token: write` permissions.
- **One-time setup**: repository **Settings → Pages → Source = GitHub Actions**.

### Correctness Check

The Hugo build itself is the test: broken `ref`/`relref` links or missing images
fail the build (`--panicOnWarning` for link issues). No separate documentation
test framework is introduced.

## Writing Style

All authored prose — landing copy, blog posts, and documentation introductions —
must follow `docs/writing-style.md` (the "Bevan Arps voice"). The load-bearing
conventions are:

- **British English spelling** (colour, behaviour, favour, realise, centre).
- **Em dashes** for parenthetical thoughts; avoid contractions.
- **Descriptive subheadings** that make a point rather than merely label.
- **Code blocks with language tags** (`bash`, `go`, `yaml`, etc.).
- **Explain why, not just what** — provide reasoning and trade-offs.

Migrated reference material (usage tables, palette listings) stays terse and
tabular but adopts British spelling and the heading conventions above.

## Risks & Mitigations

- **Hugo version drift** between local and CI → pin the Hugo extended version in
  both the workflow and documented setup.
- **Gallery generation slows CI** → generation reuses the fast `task samples`
  path; only PNGs are copied into the gallery to keep the artifact small.
- **Broken links after migration** → `--panicOnWarning` turns link/image issues
  into build failures.

## Open Questions

None outstanding. Custom domain and versioned docs are deferred as non-goals.
