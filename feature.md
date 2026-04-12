# Metrics

The current implementation of metrics isn't going to scale to the number of metrics we expect to want to support in the longer term. Plus, performance is already becoming a problem, so we need a design that will support better performing implementations.

## Requirements

Having every possible metric captured as a member on FileNode and DirectoryNode won't scale to the hundreds of planned metrics. They instead need to capture just the metrics of interest for this visualization.

We should be applying the open/closed principle - adding a new metric shouldn't require modification of any other code. This requires metrics to be self-describing and automatically registered.

Only file-system metrics are cheap to acquire, because they're instantly available during the file scan. Other metrics can be expensive and/or slow. We need a design that allows appropriate batching and parallel execution when retrieving metrics.

There are three kinds of metrics, and we should represent them in our design.

- Quantities - exact int values, e.g. file sizes, file counts
- Measures - float64 values, e.g. calculated percentages, rates, fractions
- Classifications - string values, e.g. file type, t-shirt size

## Design

### Metrics

Use the existing metric package to define a new metric framework, with the following 

- Name - string name of a metric
- Kind - enum for the type of a metric (Quantity/Measure/Classification)
- Provider - an interface defining how an individual metric works
  - Name() Name - returns the unique name of this metric
  - Kind() Kind - returns the kind of metric
  - Load(root *model.Directory) error - loads the requested metric for the given tree
  - Dependencies() []Name - returns a list of other metrics that are required for this one to be computed (e.g. folder-size requires file-size)
  - DefaultPalette() string - identifies the normal pallete to use if no other is specified
- Register(Name, Provider) - registration function

Dependencies are used to partially order providers, ensuring that underlying metrics are loaded first.
Providers with no relative dependency are run in parallel.
Other providers are started as soon as all their dependencies are resolved.
Any dependency cycle results in an immediate error.

### Providers

Each metric has a unique provider that knows how to load that value. 

Providers should be segregated into their own packages, based on the data source e.g. filesystem, git

- They link into the core framework by calling metric.Register(). 
- Each provider assembly should include a RegisterMetrics() function that can be called at app entry to activate all the providers in that package e.g. filesystem.RegisterMetrics() and git.RegisterMetrics()

When multiple providers are in use, we can run each Load() in parallel.

For filesystem metrics, simple values like file-size are captured during our directory walk. Others (such as file-count for directories) are calculated by the filesystem provider.

Git metrics are retrieved by a git centric provider.

### Model

Move FileNode and Directory node to a new package, `model`, and get renamed to `File` and `Directory`

`File` has reader and writer methods for each kind of metric
  - Quantity(metric.Provider) (int, bool)
  - SetQuantity(metric.Provider, int)
  - Measure(metric.Provider) (float64, bool)
  - SetMeasure(metric.Provider, float64)
  - Classification(metric.Provider) (string, bool)
  - SetClassification(metric.Provider, string)

`Directory` has the same methods. 

Both `File` and `Directory` are safe for concurrent use, protecting their internals with a lock, and allowing their Set*() methods to be called from any running goroutine.

## Review

Q1) 11. How do consumers read metrics? The current code uses metric.ExtractFileSize(node) to feed values into bucketing and color mapping. The design describes how providers write metrics but not how the treemap layout, bucketing, and color mapping read them. The whole consumption pipeline (TreemapCmd.Run() in the CLI) needs updating — this should be sketched out.

A) Visualizations are configured with the metric.Name to use for each property, and use those names to read values directly from the relevant model.Directory or model.Node. 

Q2)  What happens to the scan package? If FileNode/DirectoryNode move to model, does scan.Scan() now return model.Directory? Or does it return its own type with a conversion step? The boundary between scanning and the model matters for the provider design — the filesystem provider presumably wraps or replaces the current scan logic.

A) We keep the scan package, returning a model.Directory, ready to be populated with metrics. Key filesystem metrics are prepopulated by the scan (those naturally available); all other metrics are obtained by calling provider.Load().

Q3) Default palette mapping is orphaned registry.go maps MetricName → PaletteName. In the new design, where does this live? Metrics are self-describing via Provider, but there's no DefaultPalette() method on Provider, and the existing registry map won't scale to dynamic registration.

A) Each provider has a DefaultPallete() string method that identifies the pallete to use if none is defined.

Q4) 14. No conditional provider registration The design says filesystem.RegisterMetrics() and git.RegisterMetrics() are called at app entry. But what if we're not in a git repo? The current code already handles this conditionally. The design should address whether providers can fail registration gracefully, or whether the framework should query providers for availability before loading.

A) Metric registration is separate from consumption. If a metric is unavailable for any reason (e.g. trying to use a git metric outside of a git repo), Load() will return an appropriate error.

Q5) 15. Kind() on Provider vs on the framework Provider declares Kind() Kind, and metrics have a Kind enum. But File has separate typed setters (SetQuantity, SetMeasure, SetClassification). Nothing enforces that a provider returning Kind() == Quantity actually calls SetQuantity and not SetClassification. Consider whether the framework should enforce this, or if the typed setters are sufficient.

A) All of the Set*() methods should check the provider passed has the Kind expected. This will catch implementation bugs early, at minimal runtime cost.
