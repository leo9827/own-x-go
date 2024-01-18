package monitor

type Partition struct {
	NodeName      string
	StartIndex    int
	EndIndex      int
	SortBy        string
	Round         int64
	RoundDone     bool
	BatchCount    int
	FinBatchCount int
	BatchSize     int
}
