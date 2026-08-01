package report

// ChartPanel is one canvas in the HTML report.
type ChartPanel struct {
	ID    string
	Title string
	Tall  bool
}

// ChartGroup is a titled section of charts.
type ChartGroup struct {
	Title     string
	ID        string
	Panels    []ChartPanel
	Single    bool
	Collapsed bool // default-closed in report (dense sections)
}

type chartDataset struct {
	Label   string    `json:"label"`
	Data    []float64 `json:"data"`
	Dashed  bool      `json:"dashed,omitempty"`
	Fill    bool      `json:"fill,omitempty"`
	YAxisID string    `json:"yAxisID,omitempty"`
}

type chartSpec struct {
	Kind       string         `json:"kind"`
	Labels     []string       `json:"labels"`
	Datasets   []chartDataset `json:"datasets"`
	HideLegend bool           `json:"hideLegend,omitempty"`
	Stacked    bool           `json:"stacked,omitempty"`
	DualAxis   bool           `json:"dualAxis,omitempty"`
	Area       bool           `json:"area,omitempty"`
}

type builtCharts struct {
	Groups []ChartGroup
	Specs  map[string]chartSpec
	Count  int
}
