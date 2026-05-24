package main

type RenderCmd struct {
	Treemap    TreemapCmd    `cmd:"" help:"Generate a treemap visualization."`
	Radial     RadialCmd     `cmd:"" help:"Generate a radial tree visualization."`
	Bubbletree BubbletreeCmd `cmd:"" help:"Generate a bubble tree visualization."`
	Spiral     SpiralCmd     `cmd:"" help:"Generate a spiral timeline visualization."`
	Scatter    ScatterCmd    `cmd:"" help:"Generate a scatter plot visualization."`
}
