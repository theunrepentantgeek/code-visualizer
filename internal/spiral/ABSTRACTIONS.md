# Abstractions — `internal/spiral`

## Resolution

**Purpose.**

- `Resolution` is the spiral's time granularity: it fixes both the duration of one time bucket and the nominal number of buckets per revolution ([timebucket.go#L9](timebucket.go#L9), [timebucket.go#L10](timebucket.go#L10)).

**Boundary and invariants.**

- Hourly means one-hour buckets at 24 spots per lap (a lap is a day); Daily means one-day buckets at 28 spots per lap (a lap is four weeks) ([timebucket.go#L13](timebucket.go#L13), [timebucket.go#L15](timebucket.go#L15), [timebucket.go#L30](timebucket.go#L30)).
- Any unrecognised value behaves as Hourly; both switches default rather than erroring ([timebucket.go#L24](timebucket.go#L24), [timebucket.go#L34](timebucket.go#L34)).
- The nominal cadence is only a default: at Daily resolution the actual spots-per-lap is chosen from a fixed candidate list to even out spacing ([cadence.go#L5](cadence.go#L5), [cadence.go#L8](cadence.go#L8)).

**Related operations.**

- `BuildTimeBuckets` derives bucket boundaries from it; `SpotsPerLap` derives the layout cadence ([timebucket.go#L62](timebucket.go#L62), [cadence.go#L8](cadence.go#L8)).

**Proper-use patterns.**

- Get the layout cadence from the package-level `SpotsPerLap`, not `Resolution.SpotsPerLap`, so daily runs get the tuned cadence ([stages.go#L171](stages.go#L171), [cadence.go#L17](cadence.go#L17)).

**Anti-patterns.**

- Do not hard-code bucket durations or lap sizes at call sites; both live on the resolution ([timebucket.go#L20](timebucket.go#L20), [timebucket.go#L30](timebucket.go#L30)).

**Source locations.**

- [timebucket.go#L10](timebucket.go#L10) — `Resolution`, `SpotsPerLap`, `bucketDuration`.
- [cadence.go#L8](cadence.go#L8) — cadence selection.

## TimeBucket

**Purpose.**

- `TimeBucket` is one interval of project history: its half-open time span, the files active in it, and the metric values aggregated over those files ([timebucket.go#L39](timebucket.go#L39), [timebucket.go#L40](timebucket.go#L40)).
- It is the unit of data the spiral is built from — one bucket becomes one disc ([stages.go#L88](stages.go#L88), [layout.go#L57](layout.go#L57)).

**Boundary and invariants.**

- The span is half-open — `Start` inclusive, `End` exclusive — and assignment places each commit in exactly one bucket ([timebucket.go#L41](timebucket.go#L41), [timebucket.go#L42](timebucket.go#L42), [bucketing.go#L41](bucketing.go#L41)).
- A file appears in every bucket it has a commit in, so `Files` may contain duplicates across buckets ([bucketing.go#L10](bucketing.go#L10), [bucketing.go#L27](bucketing.go#L27)).
- Aggregated values are populated only after assignment, and each has an `…Available` companion flag that distinguishes "absent" from "zero" ([timebucket.go#L45](timebucket.go#L45), [aggregation.go#L17](aggregation.go#L17)).

**Related operations.**

- `BuildTimeBuckets` creates the consecutive spans, `AssignFilesToBuckets` fills `Files`, and `AggregateBucketMetrics` fills the metric values ([timebucket.go#L62](timebucket.go#L62), [bucketing.go#L15](bucketing.go#L15), [aggregation.go#L17](aggregation.go#L17)).

**Proper-use patterns.**

- Build, assign, then aggregate in that order; the pipeline stage does exactly this before anything reads a bucket's values ([stages.go#L88](stages.go#L88), [stages.go#L93](stages.go#L93), [stages.go#L101](stages.go#L101)).

**Anti-patterns.**

- Do not read `SizeValue` and friends as meaningful without checking their availability flag ([timebucket.go#L47](timebucket.go#L47), [aggregation.go#L82](aggregation.go#L82)).

**Source locations.**

- [timebucket.go#L40](timebucket.go#L40) — `TimeBucket`, `BuildTimeBuckets`.
- [bucketing.go#L15](bucketing.go#L15) — assignment and the half-open containment rule.

## SpiralNode

**Purpose.**

- `SpiralNode` is a bucket's placed disc: its absolute circle on the canvas, its spiral polar coordinates, and the time span it stands for ([node.go#L10](node.go#L10), [node.go#L12](node.go#L12)).

**Boundary and invariants.**

- Node `i` corresponds to bucket `i`; the layout allocates one node per bucket and every later pass indexes the two slices in lockstep ([layout.go#L57](layout.go#L57), [discsize.go#L22](discsize.go#L22), [render.go#L214](render.go#L214)).
- `Geometry.Center` is already absolute pixels; `Angle` runs clockwise from north and `SpiralRadius` measures out from the canvas centre ([node.go#L11](node.go#L11), [node.go#L14](node.go#L14)).
- Disc radius is not set by layout: `ApplyDiscSizes` fills it afterwards, and a radius of zero marks an empty bucket that must not be drawn ([layout.go#L35](layout.go#L35), [discsize.go#L22](discsize.go#L22), [render.go#L215](render.go#L215)).

**Related operations.**

- `LayoutWithCadence` positions the nodes, `ApplyDiscSizes` sizes them, and the render pass draws discs and labels from them ([layout.go#L47](layout.go#L47), [discsize.go#L11](discsize.go#L11), [render.go#L191](render.go#L191)).

**Proper-use patterns.**

- Translate whole layouts by moving node centres and the layout's `CY` together, so the guide track and discs stay aligned ([stages.go#L179](stages.go#L179), [stages.go#L182](stages.go#L182)).

**Anti-patterns.**

- Do not treat a zero-radius node as a drawable disc; it means the bucket had no activity ([discsize.go#L22](discsize.go#L22), [render.go#L215](render.go#L215)).

**Source locations.**

- [node.go#L12](node.go#L12) — `SpiralNode`.
- [discsize.go#L11](discsize.go#L11) — disc sizing, including the empty-bucket rule.

## SpiralLayout

**Purpose.**

- `SpiralLayout` bundles the placed nodes with the Archimedean parameters that generated them (`r = A + B*theta`, plus the centre and final angle) ([layout.go#L20](layout.go#L20), [layout.go#L23](layout.go#L23)).

**Boundary and invariants.**

- Exposing the parameters is deliberate: the guide track is re-generated from `A`, `B`, and `MaxTheta` rather than interpolated from node positions ([layout.go#L21](layout.go#L21), [render.go#L175](render.go#L175)).
- An empty bucket list yields the zero layout, so consumers must tolerate no nodes ([layout.go#L53](layout.go#L53), [render.go#L163](render.go#L163)).

**Related operations.**

- `Layout`/`LayoutWithCadence` produce it; the surface builder and track renderer consume its parameters ([layout.go#L36](layout.go#L36), [surface.go#L28](surface.go#L28), [render.go#L162](render.go#L162)).

**Proper-use patterns.**

- Derive anything spiral-shaped — the track, the surface annulus, the noise seed — from the layout parameters so it stays consistent with the discs ([render.go#L175](render.go#L175), [surface.go#L28](surface.go#L28), [surface.go#L39](surface.go#L39)).

**Anti-patterns.**

- Do not reconstruct spiral geometry by fitting a curve through node centres; the parameters are carried precisely to avoid that ([layout.go#L21](layout.go#L21)).

**Source locations.**

- [layout.go#L23](layout.go#L23) — `SpiralLayout`.
- [layout.go#L47](layout.go#L47) — parameterised construction.
