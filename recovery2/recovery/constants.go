package recovery

type State int8

const (
	// Entry default entry status for a recovery
	Entry State = iota
	// Queue recovery in queue to initilize recovery worker
	Queued
	// Start Recovery running currently
	Running
	// Pause Recovery temporarily stopped
	Paused
	// Done Recovery finished
	Done
	// Cancel Recovery to be removed
	Canceled
)

type Prty int8

const (
	// VeryLowPr just a priority
	VeryLowPr Prty = iota
	// LowPr just a priority
	LowPr
	// MediumPr just a priority
	MediumPr
	// HighPr just a priority
	HighPr
	// VeryHighPr just a priority
	VeryHighPr
	// UrgentPr just a priority
	UrgentPr
)

func ParsePrty(p Prty) string {
	switch p {
	case VeryLowPr:
		return "Very Low"
	case LowPr:
		return "Low"
	case MediumPr:
		return "Medium"
	case HighPr:
		return "High"
	case VeryHighPr:
		return "Very High"
	default:
		return "Priority out of bounds"
	}
}

func ParseState(p State) string {
	switch p {
	case Entry:
		return "Entry"
	case Queued:
		return "Queued"
	case Running:
		return "Running"
	case Paused:
		return "Paused"
	case Done:
		return "Done"
	case Canceled:
		return "Canceled"
	default:
		return "State out of bounds"
	}
}
