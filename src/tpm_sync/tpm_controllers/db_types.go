package tpm_controllers

// IterationGroup defines two columns that will be used to GROUP BY the results and get the averages, min and max
type IterationGroup struct {
	X string
	Y string
}

// IterationStatsResponse holds the result for multiple graphs/groups
type IterationStatsResponse struct {
	Graphs map[string][][]interface{} `json:"graphs"`
}

// FinishedCountData holds the count of finished and total sessions, with filter
type FinishedCountData struct {
	LearnRule     string `json:"learn_rule"`
	TPMType       string `json:"tpm_type"`
	HLGroup       string `json:"h_l_group"`
	FinishedCount int    `json:"finished_count"`
	TotalCount    int    `json:"total_count"`
}

// SessionAvgsAndCounts holds relevant data for a specific query, like all sessions with a specific K
type SessionAvgsAndCounts struct {
	AvgLearnIterations     float64 `json:"avg_learn_iterations"`
	AvgStimulateIterations float64 `json:"avg_stimulate_iterations"`
	TotalCount             int     `json:"total_count"`
	FinishedCount          int     `json:"finished_count"`
	UnfinishedCount        int     `json:"unfinished_count"`
}

type HistogramEntry struct {
	RangeLabel    string
	FinishedCount int
	TotalCount    int
	AvgLearn      float64
	AvgStim       float64
	AvgDataSize   float64
}

// type SuccessIterationCorrelationData struct {
// 	Histogram []HistogramEntry
// }
