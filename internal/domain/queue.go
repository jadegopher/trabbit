package domain

import "time"

type Queues []Queue

type PagedQueues struct {
	Items      Queues
	Pagination Pagination
}

type Pagination struct {
	Page          int
	PageCount     int
	PageSize      int
	TotalCount    int
	FilteredCount int
}

type Queue struct {
	VHost string
	Name  string
	Type  string
	// TODO: Features
	State string

	Messages Messages
}

type Messages struct {
	Ready           int
	ReadyBytes      int64
	Unacked         int
	UnackedBytes    int64
	Total           int
	TotalBytes      int64
	RAM             int
	RAMBytes        int64
	Persistent      int
	PersistentBytes int64
}

type Sample struct {
	Timestamp int64
	Value     int
}

type RateDetails struct {
	Rate    float64
	Samples []Sample
}

type MessageStats struct {
	Publish                 int64
	PublishDetails          RateDetails
	Deliver                 int64
	DeliverDetails          RateDetails
	DeliverNoAck            int64
	DeliverNoAckDetails     RateDetails
	DeliverGet              int64
	DeliverGetDetails       RateDetails
	Redeliver               int64
	RedeliverDetails        RateDetails
	Get                     int64
	GetDetails              RateDetails
	GetNoAck                int64
	GetNoAckDetails         RateDetails
	Ack                     int64
	AckDetails              RateDetails
	ReturnUnroutable        int64
	ReturnUnroutableDetails RateDetails
	DropUnroutable          int64
	DropUnroutableDetails   RateDetails
}

type QueueDetailedStats struct {
	Name            string
	VHost           string
	Type            string
	State           string
	Messages        Messages
	MessagesReady   []Sample
	MessagesUnacked []Sample
	MessagesTotal   []Sample
	MessageStats    MessageStats

	StorageVersion            int
	Durable                   bool
	Exclusive                 bool
	Internal                  bool
	InternalOwner             bool
	SingleActiveConsumerTag   string
	ExclusiveConsumerTag      string
	IdleSince                 time.Time
	Arguments                 map[string]interface{}
	Policy                    string
	OperatorPolicy            string
	EffectivePolicyDefinition map[string]interface{}
	Consumers                 int
	ConsumerCapacity          float64
	ConsumerUtilisation       float64
	Memory                    int64
	ConsumerDetails           []QueueConsumer
	Runtime                   QueueRuntime
}

// QueueConsumer describes a subscription, rather than the channel's aggregate QoS.
type QueueConsumer struct {
	Channel, Tag   string
	AckRequired    bool
	Exclusive      bool
	PrefetchCount  int
	Active         *bool
	ActivityStatus string
	TimeoutMillis  *int64
	Arguments      map[string]interface{}
}

type QueueBinding struct {
	VHost, Queue, Source, RoutingKey, PropertiesKey string
	Arguments                                       map[string]interface{}
}

func (b QueueBinding) IsDefault() bool { return b.Source == "" }

// Pointers distinguish missing broker metrics from valid zero values.
type QueueRuntime struct {
	Reductions       *RateDetails
	MinBinVHeapWords *int64
	MinHeapWords     *int64
	FullSweepAfter   *int64
	MinorGCs         *int64
}
