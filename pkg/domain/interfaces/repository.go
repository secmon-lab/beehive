package interfaces

// Repository defines the interface for data persistence
type Repository interface {
	IoCRepository
	SourceStateRepository
	HistoryRepository
}
