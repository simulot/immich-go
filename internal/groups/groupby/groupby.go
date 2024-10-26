package groupby

type GroupBy int

const (
	GroupByNone    GroupBy = iota
	GroupByBurst           // Group by burst
	GroupByRawJpg          // Group by raw/jpg
	GroupByHeicJpg         // Group by heic/jpg
	GroupByOther           // Group by other (same radical, not previous cases)
)
