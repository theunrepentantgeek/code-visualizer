package main

type RenderCmd struct {
	Treemap TreemapCmd `cmd:"" help:"Generate a treemap visualization."`
}
