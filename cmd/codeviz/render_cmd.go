package main

type RenderCmd struct {
	Treemap    TreemapCmd    `cmd:"" name:"tree-map"    help:"Generate a treemap visualization."`
	Radial     RadialCmd     `cmd:"" help:"Generate a radial tree visualization."`
	Bubbletree BubbletreeCmd `cmd:"" name:"bubble-tree" help:"Generate a bubble tree visualization."`
	Spiral     SpiralCmd     `cmd:"" help:"Generate a spiral timeline visualization."`
	Scatter    ScatterCmd    `cmd:"" help:"Generate a scatter plot visualization."`
}
