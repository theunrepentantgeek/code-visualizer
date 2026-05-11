# Position-Weighted Triage Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix squad-triage routing so issues are distributed across team members by position-weighting keyword matches — title matches score highest, early sections score moderately, and boilerplate/late content is excluded.

**Architecture:** Replace the flat `issueText.includes(kw)` scoring block in `squad-triage.yml` (lines 118–216) with a zone-based scorer that splits the issue into title / section 1 / section 2 / excluded zones, applies position multipliers (4× / 2× / 1× / 0), and uses whole-word regex matching. Also expand the routing table keywords with singular/plural variants and remove the generic "review" keyword from Ripley.

**Tech Stack:** GitHub Actions workflow YAML, inline JavaScript (actions/github-script@v9), Node.js (for offline verification script)

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `.squad/routing.md` | Modify (lines 56–66) | Expanded keyword variants in Work Type → Agent table |
| `.squad/templates/routing.md` | Modify | Template copy — add same Work Type → Agent table |
| `.github/workflows/squad-triage.yml` | Modify (lines 118–216) | Zone-based scoring with whole-word matching |
| `.squad/templates/workflows/squad-triage.yml` | Modify (lines 118–216) | Template copy — same scoring changes |

No new files are created. No files are deleted.

---

### Task 1: Update routing table keywords

**Files:**
- Modify: `.squad/routing.md:56-66`

- [ ] **Step 1: Replace the Work Type → Agent table in `.squad/routing.md`**

Replace lines 56–66 with:

```markdown
## Work Type → Agent

| Work Type | Primary | Secondary |
|-----------|---------|----------|
| Architecture, decisions, architectural | Ripley | — |
| Core logic, metric, metrics, provider, providers | Dallas | — |
| Command, commands, config, configuration, integration, CLI, flag, flags | Kane | — |
| Test, tests, testing, golden file, golden files, edge case, edge cases | Lambert | — |
| Technical debt, code quality, maintainability, refactor, refactoring, duplication | Parker | — |
| Abstraction, abstractions, interface, interfaces, type, types, code smell, code smells, design pattern, design patterns, API design | Bishop | — |
```

Key changes from the current table:
- **Ripley:** removed "review" (too generic — matches boilerplate footer), added "architectural"
- **Dallas:** added singular "metric", "provider"
- **Kane:** added singular "command", plus "configuration", "CLI", "flag", "flags"
- **Lambert:** added singular "test", "testing", singular "golden file", singular "edge case"
- **Parker:** added "refactor", "duplication", split "technical debt" stays as 2-word phrase
- **Bishop:** added singular forms of all plural keywords

- [ ] **Step 2: Verify the file looks correct**

Run: `cat .squad/routing.md | head -70 | tail -15`

Expected: the new table with expanded keywords, no "review" in Ripley's row.

- [ ] **Step 3: Commit**

```bash
git add .squad/routing.md
git commit -m "fix: expand routing keywords and remove generic 'review'

Add singular/plural variants for all routing keywords. Remove 'review'
from Ripley's entry — it matches boilerplate footer text in every issue,
causing all issues to route to the Lead.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 2: Update template routing table

**Files:**
- Modify: `.squad/templates/routing.md`

- [ ] **Step 1: Add the Work Type → Agent table to the template routing.md**

The template file at `.squad/templates/routing.md` does NOT currently have a
"Work Type → Agent" section. It ends after the Rules section (line 55). Append
the same table that was added to `.squad/routing.md` in Task 1:

```markdown

## Work Type → Agent

| Work Type | Primary | Secondary |
|-----------|---------|----------|
| Architecture, decisions, architectural | Ripley | — |
| Core logic, metric, metrics, provider, providers | Dallas | — |
| Command, commands, config, configuration, integration, CLI, flag, flags | Kane | — |
| Test, tests, testing, golden file, golden files, edge case, edge cases | Lambert | — |
| Technical debt, code quality, maintainability, refactor, refactoring, duplication | Parker | — |
| Abstraction, abstractions, interface, interfaces, type, types, code smell, code smells, design pattern, design patterns, API design | Bishop | — |
```

Note: The template uses placeholder names (`{Name}`) throughout. The Work Type
table also uses placeholder names, which is correct for a template. However, the
current `.squad/routing.md` has real names — so the template should keep
placeholder `{Name}` values for consistency with the rest of the template file.
Update the member names to `{Name}` in the template version:

```markdown

## Work Type → Agent

| Work Type | Primary | Secondary |
|-----------|---------|----------|
| Architecture, decisions, architectural | {Name} | — |
| Core logic, metric, metrics, provider, providers | {Name} | — |
| Command, commands, config, configuration, integration, CLI, flag, flags | {Name} | — |
| Test, tests, testing, golden file, golden files, edge case, edge cases | {Name} | — |
| Technical debt, code quality, maintainability, refactor, refactoring, duplication | {Name} | — |
| Abstraction, abstractions, interface, interfaces, type, types, code smell, code smells, design pattern, design patterns, API design | {Name} | — |
```

- [ ] **Step 2: Commit**

```bash
git add .squad/templates/routing.md
git commit -m "fix: add expanded Work Type table to routing template

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 3: Replace scoring logic in live workflow

**Files:**
- Modify: `.github/workflows/squad-triage.yml:118-216`

This is the main change. Replace lines 118–216 (from the `issueText` construction
through the end of the scoring block) with zone-based scoring using whole-word
regex matching.

- [ ] **Step 1: Replace the issueText construction and scoring block**

Replace lines 118–216 of `.github/workflows/squad-triage.yml` with the following.
The surrounding code (copilot evaluation above, Lead fallback below) is unchanged.

Find this block (line 118–216):
```javascript
            // Determine best assignee based on issue content and routing
            const issueText = `${issue.title}\n${issue.body || ''}`.toLowerCase();

            let assignedMember = null;
            let triageReason = '';
            let copilotTier = null;

            // First, evaluate @copilot fit if enabled
            if (hasCopilot) {
              const isNotSuitable = notSuitableKeywords.some(kw => issueText.includes(kw));
              const isGoodFit = !isNotSuitable && goodFitKeywords.some(kw => issueText.includes(kw));
              const isNeedsReview = !isNotSuitable && !isGoodFit && needsReviewKeywords.some(kw => issueText.includes(kw));
```

And replace the full block from line 118 through line 216 with:

```javascript
            // Determine best assignee based on issue content and routing
            //
            // Zone-based scoring: keywords found early in the issue (title,
            // first section) score higher than keywords buried in details or
            // boilerplate. Content below a standalone --- rule is excluded.
            const issueTitle = issue.title || '';
            const issueBody = issue.body || '';

            // Strip content at or below first standalone --- horizontal rule.
            // Table separator rows (|---|---|) are NOT standalone rules.
            let strippedBody = issueBody;
            const hrIndex = strippedBody.search(/^---\s*$/m);
            if (hrIndex !== -1) {
              strippedBody = strippedBody.substring(0, hrIndex);
            }

            // Split into sections by ## headings, fall back to paragraphs.
            let sections;
            if (/^## /m.test(strippedBody)) {
              sections = strippedBody.split(/^## /m);
            } else {
              sections = strippedBody.split(/\n\n+/);
            }

            // Build scored zones: title (4×), section 1 (2×), section 2 (1×).
            // Filter empty sections (e.g. text before first ## heading).
            const nonEmptySections = sections.filter(s => s.trim().length > 0);
            const zones = [
              { text: issueTitle, multiplier: 4 },
            ];
            if (nonEmptySections.length > 0) {
              zones.push({ text: nonEmptySections[0], multiplier: 2 });
            }
            if (nonEmptySections.length > 1) {
              zones.push({ text: nonEmptySections[1], multiplier: 1 });
            }

            // Flat issueText still needed for @copilot evaluation (unchanged).
            const issueText = `${issueTitle}\n${issueBody}`.toLowerCase();

            let assignedMember = null;
            let triageReason = '';
            let copilotTier = null;

            // First, evaluate @copilot fit if enabled
            if (hasCopilot) {
              const isNotSuitable = notSuitableKeywords.some(kw => issueText.includes(kw));
              const isGoodFit = !isNotSuitable && goodFitKeywords.some(kw => issueText.includes(kw));
              const isNeedsReview = !isNotSuitable && !isGoodFit && needsReviewKeywords.some(kw => issueText.includes(kw));

              if (isGoodFit) {
                copilotTier = 'good-fit';
                assignedMember = { name: '@copilot', role: 'Coding Agent' };
                triageReason = '🟢 Good fit for @copilot — matches capability profile';
              } else if (isNeedsReview) {
                copilotTier = 'needs-review';
                assignedMember = { name: '@copilot', role: 'Coding Agent' };
                triageReason = '🟡 Routing to @copilot (needs review) — a squad member should review the PR';
              } else if (isNotSuitable) {
                copilotTier = 'not-suitable';
                // Fall through to normal routing
              }
            }

            // If not routed to @copilot, use routing.md keyword scoring
            if (!assignedMember && routingContent) {
              // Parse "Work Type → Agent" table from routing.md
              const routingRules = [];
              const routingLines = routingContent.split('\n');
              let inRoutingTable = false;
              let pastSeparator = false;

              for (const rline of routingLines) {
                if (rline.match(/^#+.*Work Type/i)) {
                  inRoutingTable = true;
                  pastSeparator = false;
                  continue;
                }
                if (inRoutingTable && /^#+\s/.test(rline) && !rline.match(/Work Type/i)) {
                  break;
                }
                if (inRoutingTable && rline.includes('---')) {
                  pastSeparator = true;
                  continue;
                }
                if (inRoutingTable && pastSeparator && rline.startsWith('|')) {
                  const cells = rline.split('|').map(c => c.trim()).filter(Boolean);
                  if (cells.length >= 2) {
                    const keywords = cells[0].toLowerCase().split(',').map(s => s.trim()).filter(Boolean);
                    const memberName = cells[1].trim();
                    // Skip placeholder/empty rows
                    if (memberName.startsWith('{') || memberName === '—' || memberName === '-') continue;
                    const member = members.find(m => m.name.toLowerCase() === memberName.toLowerCase());
                    if (member) {
                      routingRules.push({ member, keywords });
                    }
                  }
                }
              }

              // Escape regex metacharacters so keywords match literally.
              function escapeRegex(s) {
                return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
              }

              // Keyword weight: multi-word phrases score higher (words²).
              function keywordWeight(kw) {
                const words = kw.split(/\s+/).length;
                return words * words; // 1-word=1, 2-word=4, 3-word=9
              }

              // Test whether a keyword appears as a whole word in text.
              function matchesKeyword(text, kw) {
                const re = new RegExp('\\b' + escapeRegex(kw) + '\\b', 'i');
                return re.test(text);
              }

              // Score each member across all zones. For each keyword, only
              // the highest-scoring zone counts (no double-counting).
              let bestScore = 0;
              let bestMember = null;
              let bestKeywords = [];

              for (const rule of routingRules) {
                let score = 0;
                const matched = [];
                for (const kw of rule.keywords) {
                  let kwBestZoneScore = 0;
                  for (const zone of zones) {
                    if (matchesKeyword(zone.text, kw)) {
                      const zoneScore = keywordWeight(kw) * zone.multiplier;
                      if (zoneScore > kwBestZoneScore) {
                        kwBestZoneScore = zoneScore;
                      }
                    }
                  }
                  if (kwBestZoneScore > 0) {
                    score += kwBestZoneScore;
                    matched.push(kw);
                  }
                }
                if (score > bestScore) {
                  bestScore = score;
                  bestMember = rule.member;
                  bestKeywords = matched;
                }
              }

              if (bestMember) {
                assignedMember = bestMember;
                triageReason = `Matched routing keywords: ${bestKeywords.join(', ')} (score: ${bestScore})`;
              }
            }
```

Note: the copilot evaluation block is reproduced in full above because it sits
inside the replaced range. The copilot logic itself is unchanged — only the
`issueText` construction above it and the scoring loop below it change.

- [ ] **Step 2: Verify the YAML is valid**

Run: `node -e "const fs = require('fs'); const y = require('js-yaml'); y.load(fs.readFileSync('.github/workflows/squad-triage.yml', 'utf8')); console.log('YAML valid');" 2>&1 || echo "Install js-yaml: npm install --no-save js-yaml && node -e \"const fs = require('fs'); const y = require('js-yaml'); y.load(fs.readFileSync('.github/workflows/squad-triage.yml', 'utf8')); console.log('YAML valid');\""` 

If js-yaml is not available, alternatively verify with Python:
`python3 -c "import yaml; yaml.safe_load(open('.github/workflows/squad-triage.yml')); print('YAML valid')"`

Expected: "YAML valid"

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/squad-triage.yml
git commit -m "fix: position-weighted keyword scoring in squad triage

Replace flat issueText.includes() scoring with zone-based scoring:
- Title: 4× multiplier
- Section 1: 2× multiplier
- Section 2: 1× multiplier
- Below --- horizontal rule: excluded (0×)

Also switch from substring matching to whole-word regex matching
to prevent false positives (e.g. 'types' inside 'prototypes').

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 4: Copy scoring changes to template workflow

**Files:**
- Modify: `.squad/templates/workflows/squad-triage.yml:118-216`

- [ ] **Step 1: Apply the same replacement from Task 3 to the template**

The template file `.squad/templates/workflows/squad-triage.yml` has identical
content to the live workflow at lines 118–216. Apply the exact same replacement
as Task 3 Step 1.

- [ ] **Step 2: Verify both files have the same scoring block**

Run: `diff .github/workflows/squad-triage.yml .squad/templates/workflows/squad-triage.yml`

Expected: no output (files are identical).

- [ ] **Step 3: Commit**

```bash
git add .squad/templates/workflows/squad-triage.yml
git commit -m "fix: sync template workflow with position-weighted scoring

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 5: Verify scoring against real issues

**Files:**
- No files modified — this is a verification-only task.

- [ ] **Step 1: Run a verification script against real issue data**

Run the following Node.js script to simulate the new scoring against recent
issues. This uses the same zone-splitting and scoring logic from the workflow.

```bash
node -e '
const routingRules = [
  { member: "Ripley",  keywords: ["architecture", "decisions", "architectural"] },
  { member: "Dallas",  keywords: ["core logic", "metric", "metrics", "provider", "providers"] },
  { member: "Kane",    keywords: ["command", "commands", "config", "configuration", "integration", "cli", "flag", "flags"] },
  { member: "Lambert", keywords: ["test", "tests", "testing", "golden file", "golden files", "edge case", "edge cases"] },
  { member: "Parker",  keywords: ["technical debt", "code quality", "maintainability", "refactor", "refactoring", "duplication"] },
  { member: "Bishop",  keywords: ["abstraction", "abstractions", "interface", "interfaces", "type", "types", "code smell", "code smells", "design pattern", "design patterns", "api design"] },
];

function escapeRegex(s) { return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"); }
function keywordWeight(kw) { const w = kw.split(/\s+/).length; return w * w; }
function matchesKeyword(text, kw) { return new RegExp("\\b" + escapeRegex(kw) + "\\b", "i").test(text); }

function scoreIssue(title, body) {
  let strippedBody = body;
  const hrIndex = strippedBody.search(/^---\s*$/m);
  if (hrIndex !== -1) strippedBody = strippedBody.substring(0, hrIndex);
  let sections;
  if (/^## /m.test(strippedBody)) { sections = strippedBody.split(/^## /m); }
  else { sections = strippedBody.split(/\n\n+/); }
  const zones = [{ text: title, multiplier: 4 }];
  const nonEmpty = sections.filter(s => s.trim().length > 0);
  if (nonEmpty.length > 0) zones.push({ text: nonEmpty[0], multiplier: 2 });
  if (nonEmpty.length > 1) zones.push({ text: nonEmpty[1], multiplier: 1 });
  let bestScore = 0, bestMember = null, bestKws = [];
  for (const rule of routingRules) {
    let score = 0; const matched = [];
    for (const kw of rule.keywords) {
      let kwBest = 0;
      for (const zone of zones) {
        if (matchesKeyword(zone.text, kw)) {
          const zs = keywordWeight(kw) * zone.multiplier;
          if (zs > kwBest) kwBest = zs;
        }
      }
      if (kwBest > 0) { score += kwBest; matched.push(kw); }
    }
    if (score > bestScore) { bestScore = score; bestMember = rule.member; bestKws = matched; }
  }
  return bestMember
    ? { member: bestMember, score: bestScore, keywords: bestKws }
    : { member: "Ripley (fallback)", score: 0, keywords: [] };
}

const issues = [
  { n: 205, t: "Treemap size metric incorrectly restricted to Quantity; should also accept Measure",
    b: "## Summary\n\nTreemapCmd.validateConfig rejects any size metric that is not metric.Quantity, but the bubbletree and radial commands both accept Quantity | Measure.\n\n## Details\n\nThis restriction prevents commit-density from being used as a treemap size metric.\n\n---\n*Post-refactoring review — Squad team code review*" },
  { n: 202, t: "Unify parallel *Inks struct hierarchy and move buildMetricInk to shared file",
    b: "## Summary\n\nTwo related structural issues left from the canvas refactoring: four structurally identical two-field *Inks structs exist one per viz type.\n\n## Details\n\nShotgun Surgery: adding a new ink field requires editing 4 files.\n\n## Impact\n\nbuildMetricInk and its siblings are invisible to developers.\n\n---\n*Post-refactoring review — Squad team code review*" },
  { n: 200, t: "Backend.DrawLegend is at the wrong abstraction level",
    b: "## Summary\n\nThe Backend interface is a low-level geometric primitive interface. But DrawLegend takes a high-level LegendData struct.\n\n## Details\n\nThe two backends contain parallel logic.\n\n---\n*Post-refactoring review — Squad team code review*" },
  { n: 198, t: "LegendData uses raw strings instead of typed constants causing type safety loss at model boundary",
    b: "## Summary\n\nLegendPosition and LegendOrientation are typed string aliases providing compile-time safety. But when toLegendData converts LegendConfig to model.LegendData both values are cast to raw string.\n\n## Details\n\nThis loses type safety at the model boundary.\n\n---\n*Post-refactoring review — Squad team code review*" },
  { n: 197, t: "Non-deterministic spiral rendering when categories have equal counts",
    b: "## Summary\n\nThe modeCategory function determines the dominant file classification by iterating over a Go map. When two categories have equal counts the winner is non-deterministic.\n\n## Details\n\nThere is also no unit test for modeCategory.\n\n---\n*Post-refactoring review — Squad team code review*" },
  { n: 196, t: "Directory nodes always render with wrong border metric color in bubbletree and radial",
    b: "## Summary\n\nWhen a border metric is configured directory discs in both bubbletree and radial tree always receive an empty canvas.MetricValue for their border.\n\n## Details\n\nThe metric value lookup skips directory nodes.\n\n---\n*Post-refactoring review — Squad team code review*" },
  { n: 195, t: "Remove dead canvas API fields and unimplemented stubs",
    b: "## Summary\n\nSeveral exported fields and types in internal/canvas represent capabilities that do not exist. They were carried over from the pre-Canvas design or added speculatively but never implemented.\n\n## Details\n\nDead code that misleads developers.\n\n---\n*Post-refactoring review — Squad team code review*" },
  { n: 194, t: "Introduce shared vizCmd pipeline to eliminate command lifecycle duplication",
    b: "## Summary\n\nAll four visualization commands follow the same 15-step lifecycle. Steps 1-9 and 13-15 are structurally identical across all four.\n\n## Details\n\nThis duplication means every lifecycle change must be applied four times.\n\n---\n*Post-refactoring review — Squad team code review*" },
  { n: 193, t: "Replace layeredShape C-style tagged union with Go interface",
    b: "## Summary\n\nlayeredShape is implemented as a C-style discriminated union: a shapeKind integer tag alongside six nullable pointer fields.\n\n## Details\n\nThis pattern is error-prone and un-idiomatic Go. A Go interface with concrete types would be safer.\n\n---\n*Post-refactoring review — Squad team code review*" },
  { n: 192, t: "Extract duplicated command helper methods into shared package-level functions",
    b: "## Summary\n\nFour methods — buildFilterRules validatePaths checkGitRequirement and filterBinaryFiles — are copied verbatim across all four command structs.\n\n## Details\n\nDuplication means bugs must be fixed in four places.\n\n---\n*Post-refactoring review — Squad team code review*" },
];

console.log("=== Position-Weighted Scoring Results ===\n");
const counts = {};
for (const iss of issues) {
  const r = scoreIssue(iss.t, iss.b);
  counts[r.member] = (counts[r.member] || 0) + 1;
  console.log("#" + iss.n + ": " + r.member.padEnd(20) + " score=" + String(r.score).padEnd(4) + " keywords=[" + r.keywords.join(", ") + "]");
}
console.log("\n=== Distribution ===");
for (const [m, c] of Object.entries(counts).sort((a,b) => b[1]-a[1])) {
  console.log("  " + m.padEnd(20) + ": " + c + " issue(s)");
}
const unique = Object.keys(counts).length;
console.log("\nMembers used: " + unique + "/6");
console.log(unique >= 3 ? "PASS: distributed across 3+ members" : "FAIL: too concentrated");
'
```

Expected: Issues distributed across at least 3 different members. Specifically:
- "review" should NOT appear as a matched keyword for any issue
- #205 should route to Dallas or Kane (mentions "metric" and "commands")
- #192 should route to Kane or Parker (mentions "command" and "duplication")
- #193 should route to Bishop (mentions "interface" and "type")
- No single member should get more than 40% of issues

- [ ] **Step 2: Verify "review" in footer is excluded**

Run: `node -e "const body = 'Summary here.\n\n---\n*Post-refactoring review*'; const hr = body.search(/^---\\s*$/m); console.log('HR at:', hr); console.log('Stripped:', JSON.stringify(body.substring(0, hr))); console.log('Contains review:', body.substring(0, hr).includes('review'));"`

Expected:
```
HR at: 14
Stripped: "Summary here.\n"
Contains review: false
```

This confirms the `---` stripping removes the boilerplate footer.

- [ ] **Step 3: Verify word-boundary matching**

Run: `node -e "const re = new RegExp('\\\\b' + 'types' + '\\\\b', 'i'); console.log('types in prototypes:', re.test('prototypes')); console.log('types alone:', re.test('extract types from')); console.log('metric matches metric:', new RegExp('\\\\bmetric\\\\b', 'i').test('the metric is wrong')); console.log('metrics matches metrics:', new RegExp('\\\\bmetrics\\\\b', 'i').test('load metrics from'));"`

Expected:
```
types in prototypes: false
types alone: true
metric matches metric: true
metrics matches metrics: true
```

- [ ] **Step 4: Record verification results**

No commit needed. If all checks pass, the implementation is complete. If any
check fails, review the scoring logic and keyword table, fix, and re-run.
