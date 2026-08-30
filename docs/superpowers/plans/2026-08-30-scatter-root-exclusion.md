# Scatter Root Exclusion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Exclude the scan root from directory-grain scatter plots while retaining every descendant directory.

**Architecture:** Keep the shared root-inclusive `model.WalkDirectories` contract unchanged. Scatter will invoke that walker once for each immediate child, producing a descendant-only dataset and preallocating for exactly `root.AllDirCount` entries.

**Tech Stack:** Go 1.26.1, Gomega, existing model and scatter packages

---

### Task 1: Exclude the root from directory collection

**Files:**
- Modify: `internal/scatter/layout_test.go:58-87`
- Modify: `internal/scatter/data.go:93-126`

- [ ] **Step 1: Change the dataset test to require descendant-only points**

Replace the root-inclusive test with:

```go
func TestCollectDataset_DirectoryGrainExcludesRootAndIncludesDescendants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	const (
		xMetric = metric.Name("file-lines.sum")
		yMetric = metric.Name("file-size.sum")
	)

	root := &model.Directory{Name: "root"}
	child := &model.Directory{Name: "child"}
	grandchild := &model.Directory{Name: "grandchild"}
	child.Dirs = []*model.Directory{grandchild}
	root.Dirs = []*model.Directory{child}

	for i, dir := range []*model.Directory{root, child, grandchild} {
		dir.SetQuantity(xMetric, int64(i+1))
		dir.SetQuantity(yMetric, int64((i+1)*10))
	}

	dataset := CollectDataset(
		root,
		viz.GrainDirectory,
		AxisSpec{Metric: xMetric, Kind: metric.Quantity},
		AxisSpec{Metric: yMetric, Kind: metric.Quantity},
		yMetric,
	)

	g.Expect(dataset.Points).To(HaveLen(2))
	g.Expect([]string{dataset.Points[0].Name(), dataset.Points[1].Name()}).
		To(ConsistOf("child", "grandchild"))
}
```

- [ ] **Step 2: Run the test and verify the root-inclusive implementation fails**

Run:

```bash
go test ./internal/scatter -run TestCollectDataset_DirectoryGrainExcludesRootAndIncludesDescendants
```

Expected: FAIL because the dataset has three points and includes `root`.

- [ ] **Step 3: Traverse each root child and correct capacity**

Change the directory branch in `CollectDataset` to:

```go
if grain == viz.GrainDirectory {
	for _, dir := range root.Dirs {
		model.WalkDirectories(dir, func(current *model.Directory) {
			collectPoint(&dataset, PointDatum{Directory: current}, xAxis, yAxis, sizeMetric)
		})
	}
} else {
	model.WalkFiles(root, func(file *model.File) {
		collectPoint(&dataset, PointDatum{File: file}, xAxis, yAxis, sizeMetric)
	})
}
```

Change directory capacity to:

```go
if grain == viz.GrainDirectory {
	return root.AllDirCount
}
```

- [ ] **Step 4: Run scatter tests**

Run:

```bash
go test ./internal/scatter
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/scatter/data.go internal/scatter/layout_test.go
git commit -m "Exclude root from directory scatter plots" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Verify and publish the PR update

**Files:**
- Modify only if verification exposes a defect in Task 1 files.

- [ ] **Step 1: Format**

Run:

```bash
PATH=/home/bevan/go/bin:$PATH task fmt
```

Expected: exit status 0.

- [ ] **Step 2: Run repository CI**

Run `PATH=/home/bevan/go/bin:$PATH task ci` through an Explore/equivalent
subagent so verbose lint output remains out of the main context.

Expected: build, tests, and lint pass.

- [ ] **Step 3: Smoke-test the plotted node count**

Run:

```bash
./bin/codeviz scatter . -o /tmp/codeviz-scatter-directories.svg \
  --grain directory \
  --x-axis file-lines.sum \
  --y-axis file-size.sum \
  --size file-size.sum
test -s /tmp/codeviz-scatter-directories.svg
rm -f /tmp/codeviz-scatter-directories.svg
```

Expected: the render succeeds and logs one fewer node than the root-inclusive
implementation for the same scan.

- [ ] **Step 4: Push the branch**

Run:

```bash
git push
```

Expected: the open pull request updates successfully.
