# Spec

Design for a new Canvas abstraction, to allow us to reduce code duplication across different types of visualization.

Rather than simple drawing primitives, this should be focused on creation of data visualizations, with a set of primitives that make sense for this use.

One element should be an `Ink` abstraction that references a pallete, allowing the "colour" of a border, fill, or line to be defined by a metric instead of an RGB value.

Other elements probably include `Rectangle` and `Disc`, both of which would have support for specifying both border and fill inks.

To aid with consistency as visualizations are drawn, we'll need some kind of spec or templating or class system (think CSS for web pages or Styles in MS Word). 

For example, a Treemap might create a spec for rectangles that defines the inks used for border and fill, and then use that to create all the rectangles needed. Or, the Radial visualization might create a spec for file discs, defining the inks used for (again) border and fill, so that all files are represented consistently.

We need two implementations of this abstraction, one that renders to PNG or JPG files, and one that renders to SVG files.
