# Metrics

The current implementation of metrics isn't going to scale to the number of metrics we expect to want to support in the longer term. Plus, performance is already becoming a problem, so we need a design that will support better performing implementations.

## Requirements

Having every possible metric captured as a member on FileNode and DirectoryNode won't scale to the hundreds of planned metrics. They instead need to capture just the metrics of interest for this visualization.

We should be applying the open/closed principle - adding a new metric shouldn't require modification of any other code. This requires metrics to be self-describing and automtically registered.

Only file-system metrics are cheap to acquire, because they're instantly available during the file scan. Other metrics can be expensive and/or slow. We need a design that allows apprpriate batching and parallel execution when retrieving metrics.

There are three kinds of metrics, and we should represent them in our design.

- Quantities - exact int values, e.g. file sizes, file counts
- Measures - double values, e.g. calculated percentages, rates, fractions
- Classifications - string values, e.g. file type, t-shirt size

## Design

### Metrics

Use the existing metric package to define a new metric framework, with the following 

- Name - string name of a metric
- Kind - enum for the type of a metric (Quantity/Measure/Classification)
- Provider - an interface defining how a metric works
  - Kind() Kind - returns the kind of metric
  - Load(root DirectoryNode) - loads the metric for the given tree
  - Dependencies() []Name - returns a list of other metrics that are required for this one to be computed (e.g. folder-size requires file-size)
- Register(Name, Provider) - registration function

### Providers

Providers should be segregated into their own packages; they link into the core framework by calling metric.Register(). Each provider should include a RegisterMetrics() function that can be called at app entry to activate the provider e.g. filesystem.RegisterMetrics() and git.RegisterMetrics()

When multiple providers are in use, we can run each Load() in parallel.

For filesystem metrics, simple values like file-size are captured during our directory walk. Others (such as file-count for directories) are calculated by the filesystem provider.

Git metrics are retrieved by a git centric provider.

### Model

Move FileNode and Directory node to a new package, `model`, and get renamed to `File` and `Directory`

`File` has reader and writer methods for each kind of metric
  - Quantity(name) int
  - SetQuantity(name, int)
  - Measure(name) double
  - SetMeasure(name) double
  - Classification(name) string
  - SetClassification(name, string)

`Directory` has the same methods. Both are safe for concurrent use, should parallel providers try to update a given file or directory at the same time.

