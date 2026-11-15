package widgets

import "github.com/jadegopher/trabbit/internal/domain"

type MsgQueues struct {
	Queues     domain.Queues
	Pagination domain.Pagination
	RequestID  int
}

type MsgQueueSelectionChanged struct {
	VHost string
	Name  string
}

type MsgQueueDetailedStats struct {
	VHost   string
	Name    string
	FetchID int
	Stats   domain.QueueDetailedStats
}
