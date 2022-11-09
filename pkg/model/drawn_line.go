package model

type DrawnLine struct {
	Path  []Point `json:"path"`
	Color int     `json:"color"`
	Width float32 `json:"width"`
}
