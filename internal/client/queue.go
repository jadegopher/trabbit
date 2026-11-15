package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	rmqc "github.com/michaelklishin/rabbit-hole/v3"

	"github.com/jadegopher/trabbit/internal/domain"
)

// QueueInfo represents a queue, its properties and key metrics.
// This is a custom struct to capture fields not present in rabbit-hole/v3.
type QueueInfo struct {
	Name                      string                 `json:"name"`
	Type                      string                 `json:"type,omitempty"`
	Vhost                     string                 `json:"vhost,omitempty"`
	Durable                   bool                   `json:"durable"`
	AutoDelete                bool                   `json:"auto_delete"`
	Exclusive                 bool                   `json:"exclusive,omitempty"`
	Arguments                 map[string]interface{} `json:"arguments"`
	Node                      string                 `json:"node,omitempty"`
	Status                    string                 `json:"state,omitempty"`
	Consumers                 int                    `json:"consumers,omitempty"`
	ConsumerCapacity          float64                `json:"consumer_capacity,omitempty"`
	ConsumerUtilisation       float64                `json:"consumer_utilisation,omitempty"`
	Policy                    string                 `json:"policy,omitempty"`
	OperatorPolicy            string                 `json:"operator_policy,omitempty"`
	EffectivePolicyDefinition map[string]interface{} `json:"effective_policy_definition,omitempty"`
	StorageVersion            int                    `json:"storage_version,omitempty"`
	Internal                  bool                   `json:"internal,omitempty"`
	InternalOwner             bool                   `json:"internal_owner,omitempty"`
	SingleActiveConsumerTag   string                 `json:"single_active_consumer_tag,omitempty"`
	ExclusiveConsumerTag      string                 `json:"exclusive_consumer_tag,omitempty"`
	IdleSince                 time.Time              `json:"idle_since,omitempty"`

	Messages                      int               `json:"messages,omitempty"`
	MessagesDetails               *rmqc.RateDetails `json:"messages_details,omitempty"`
	MessagesReady                 int               `json:"messages_ready,omitempty"`
	MessagesReadyDetails          *rmqc.RateDetails `json:"messages_ready_details,omitempty"`
	MessagesUnacknowledged        int               `json:"messages_unacknowledged,omitempty"`
	MessagesUnacknowledgedDetails *rmqc.RateDetails `json:"messages_unacknowledged_details,omitempty"`

	MessageBytes               int64 `json:"message_bytes,omitempty"`
	MessageBytesReady          int64 `json:"message_bytes_ready,omitempty"`
	MessageBytesUnacknowledged int64 `json:"message_bytes_unacknowledged,omitempty"`
	MessageBytesRAM            int64 `json:"message_bytes_ram,omitempty"`
	MessageBytesPersistent     int64 `json:"message_bytes_persistent,omitempty"`
	MessagesRAM                int   `json:"messages_ram,omitempty"`
	MessagesPersistent         int   `json:"messages_persistent,omitempty"`
	Memory                     int64 `json:"memory,omitempty"`

	MessageStats      *rmqc.MessageStats `json:"message_stats,omitempty"`
	ConsumerDetails   []consumerInfo     `json:"consumer_details"`
	ReductionsDetails *rmqc.RateDetails  `json:"reductions_details"`
	GarbageCollection struct {
		MinBinVHeapSize *int64 `json:"min_bin_vheap_size"`
		MinHeapSize     *int64 `json:"min_heap_size"`
		FullSweepAfter  *int64 `json:"fullsweep_after"`
		MinorGCs        *int64 `json:"minor_gcs"`
	} `json:"garbage_collection"`
}

type consumerInfo struct {
	ChannelDetails struct {
		Name string `json:"name"`
	} `json:"channel_details"`
	Tag            string                 `json:"consumer_tag"`
	AckRequired    bool                   `json:"ack_required"`
	Exclusive      bool                   `json:"exclusive"`
	PrefetchCount  int                    `json:"prefetch_count"`
	Active         *bool                  `json:"active"`
	ActivityStatus string                 `json:"activity_status"`
	TimeoutMillis  *int64                 `json:"consumer_timeout"`
	Arguments      map[string]interface{} `json:"arguments"`
}

func (s Client) ListQueues(ctx context.Context, params ListParameters) (domain.PagedQueues, error) {
	path := "/api/queues"
	if params.VHost != "" {
		path += "/" + url.PathEscape(params.VHost)
	}
	var res rmqc.PagedQueueInfo
	err := s.request(ctx, http.MethodGet, path+"?"+applyListParameters(params).Encode(), nil, &res)
	if err != nil {
		return domain.PagedQueues{}, err
	}
	return domain.PagedQueues{Items: fromQueues(res.Items), Pagination: domain.Pagination{
		Page: res.Page, PageCount: res.PageCount, PageSize: res.PageSize, TotalCount: res.TotalCount, FilteredCount: res.FilteredCount,
	}}, nil
}

func applyListParameters(params ListParameters) url.Values {
	q := url.Values{}
	if params.Page > 0 {
		q.Set("page", strconv.Itoa(params.Page))
	}
	if params.PageSize > 0 {
		q.Set("page_size", strconv.Itoa(params.PageSize))
	}
	if params.Name != "" {
		q.Set("name", params.Name)
	}
	if params.UseRegex {
		q.Set("use_regex", "true")
	}
	return q
}

func queuePath(vhost, name string) string {
	return "/api/queues/" + url.PathEscape(vhost) + "/" + url.PathEscape(name)
}
func (s Client) CreateQueue(ctx context.Context, params CreateQueueParams) error {
	return s.request(ctx, http.MethodPut, queuePath(params.VHost, params.Name), rmqc.QueueSettings{
		Durable: params.Durable, AutoDelete: params.AutoDelete, Arguments: params.Arguments,
	}, nil)
}
func (s Client) DeleteQueue(ctx context.Context, vhost, name string) error {
	return s.request(ctx, http.MethodDelete, queuePath(vhost, name), nil, nil)
}
func (s Client) PurgeQueue(ctx context.Context, vhost, name string) error {
	return s.request(ctx, http.MethodDelete, queuePath(vhost, name)+"/contents", nil, nil)
}

func (s Client) GetQueueDetailedStats(ctx context.Context, vhost, name string, history time.Duration) (domain.QueueDetailedStats, error) {
	// RabbitMQ rejects ranges with more than 500 samples. Use its hour-view
	// resolution when five-second samples would exceed that limit.
	sampleInterval := queueHistorySampleInterval
	if history/sampleInterval > queueHistoryMaxSamples {
		sampleInterval = queueHistoryLongSampleInterval
	}
	ageSeconds := strconv.Itoa(int(history / time.Second))
	intervalSeconds := strconv.Itoa(int(sampleInterval / time.Second))
	qs := url.Values{}
	for _, prefix := range []string{"lengths", "msg_rates", "data_rates"} {
		qs.Set(prefix+"_age", ageSeconds)
		qs.Set(prefix+"_incr", intervalSeconds)
	}

	var res QueueInfo
	if err := s.request(ctx, http.MethodGet, queuePath(vhost, name)+"?"+qs.Encode(), nil, &res); err != nil {
		return domain.QueueDetailedStats{}, err
	}

	return domain.QueueDetailedStats{
		Name:  res.Name,
		VHost: res.Vhost,
		Type:  res.Type,
		State: res.Status,
		Messages: domain.Messages{
			Ready:           res.MessagesReady,
			ReadyBytes:      res.MessageBytesReady,
			Unacked:         res.MessagesUnacknowledged,
			UnackedBytes:    res.MessageBytesUnacknowledged,
			Total:           res.Messages,
			TotalBytes:      res.MessageBytes,
			RAM:             res.MessagesRAM,
			RAMBytes:        res.MessageBytesRAM,
			Persistent:      res.MessagesPersistent,
			PersistentBytes: res.MessageBytesPersistent,
		},
		MessagesReady:             optionalSamples(res.MessagesReadyDetails),
		MessagesUnacked:           optionalSamples(res.MessagesUnacknowledgedDetails),
		MessagesTotal:             optionalSamples(res.MessagesDetails),
		MessageStats:              fromMessageStats(res.MessageStats),
		StorageVersion:            res.StorageVersion,
		Durable:                   res.Durable,
		Exclusive:                 res.Exclusive,
		Internal:                  res.Internal,
		InternalOwner:             res.InternalOwner,
		SingleActiveConsumerTag:   res.SingleActiveConsumerTag,
		ExclusiveConsumerTag:      res.ExclusiveConsumerTag,
		IdleSince:                 res.IdleSince,
		Arguments:                 res.Arguments,
		Policy:                    res.Policy,
		OperatorPolicy:            res.OperatorPolicy,
		EffectivePolicyDefinition: res.EffectivePolicyDefinition,
		Consumers:                 res.Consumers,
		ConsumerCapacity:          res.ConsumerCapacity,
		ConsumerUtilisation:       res.ConsumerUtilisation,
		Memory:                    res.Memory,
		ConsumerDetails:           fromConsumers(res.ConsumerDetails),
		Runtime: domain.QueueRuntime{
			Reductions:       optionalRateDetails(res.ReductionsDetails),
			MinBinVHeapWords: res.GarbageCollection.MinBinVHeapSize,
			MinHeapWords:     res.GarbageCollection.MinHeapSize,
			FullSweepAfter:   res.GarbageCollection.FullSweepAfter,
			MinorGCs:         res.GarbageCollection.MinorGCs,
		},
	}, nil
}

const (
	queueHistoryMaxSamples         = 500 // RabbitMQ's per-request range limit.
	queueHistorySampleInterval     = 5 * time.Second
	queueHistoryLongSampleInterval = time.Minute
)

func fromConsumers(input []consumerInfo) []domain.QueueConsumer {
	consumers := make([]domain.QueueConsumer, len(input))
	for i, c := range input {
		consumers[i] = domain.QueueConsumer{
			Channel: c.ChannelDetails.Name, Tag: c.Tag, AckRequired: c.AckRequired,
			Exclusive: c.Exclusive, PrefetchCount: c.PrefetchCount, Active: c.Active,
			ActivityStatus: c.ActivityStatus, TimeoutMillis: c.TimeoutMillis, Arguments: c.Arguments,
		}
	}
	return consumers
}

func optionalRateDetails(input *rmqc.RateDetails) *domain.RateDetails {
	if input == nil {
		return nil
	}
	result := fromRateDetails(*input)
	return &result
}

func (s Client) Publish(ctx context.Context, vhost, exchange string, options PublishOptions) (PublishResult, error) {
	properties := make(map[string]interface{}, len(options.Properties.Properties)+1)
	for k, v := range options.Properties.Properties {
		properties[k] = v
	}
	properties["headers"] = options.Properties.Headers
	var result PublishResult
	err := s.request(ctx, http.MethodPost, "/api/exchanges/"+url.PathEscape(vhost)+"/"+url.PathEscape(exchange)+"/publish", rmqc.PublishOptions{
		RoutingKey: options.RoutingKey, Payload: options.Payload, PayloadEncoding: options.PayloadEncoding, Properties: properties,
	}, &result)
	return result, err
}
func optionalSamples(details *rmqc.RateDetails) []domain.Sample {
	if details == nil {
		return nil
	}
	return fromSamples(details.Samples)
}
func fromRateDetails(rd rmqc.RateDetails) domain.RateDetails {
	return domain.RateDetails{
		Rate:    float64(rd.Rate),
		Samples: fromSamples(rd.Samples),
	}
}

func fromMessageStats(ms *rmqc.MessageStats) domain.MessageStats {
	if ms == nil {
		return domain.MessageStats{}
	}
	return domain.MessageStats{
		Publish:                 ms.Publish,
		PublishDetails:          fromRateDetails(ms.PublishDetails),
		Deliver:                 ms.Deliver,
		DeliverDetails:          fromRateDetails(ms.DeliverDetails),
		DeliverNoAck:            ms.DeliverNoAck,
		DeliverNoAckDetails:     fromRateDetails(ms.DeliverNoAckDetails),
		DeliverGet:              ms.DeliverGet,
		DeliverGetDetails:       fromRateDetails(ms.DeliverGetDetails),
		Redeliver:               ms.Redeliver,
		RedeliverDetails:        fromRateDetails(ms.RedeliverDetails),
		Get:                     ms.Get,
		GetDetails:              fromRateDetails(ms.GetDetails),
		GetNoAck:                ms.GetNoAck,
		GetNoAckDetails:         fromRateDetails(ms.GetNoAckDetails),
		Ack:                     ms.Ack,
		AckDetails:              fromRateDetails(ms.AckDetails),
		ReturnUnroutable:        ms.ReturnUnroutable,
		ReturnUnroutableDetails: fromRateDetails(ms.ReturnUnroutableDetails),
		DropUnroutable:          ms.DropUnroutable,
		DropUnroutableDetails:   fromRateDetails(ms.DropUnroutableDetails),
	}
}

func fromSamples(samples []rmqc.RateDetailSample) []domain.Sample {
	result := make([]domain.Sample, 0, len(samples))
	for _, s := range samples {
		result = append(
			result, domain.Sample{
				Timestamp: s.Timestamp,
				Value:     int(s.Sample),
			},
		)
	}
	return result
}

func fromQueues(q []rmqc.QueueInfo) domain.Queues {
	result := make(domain.Queues, 0, len(q))
	for _, elem := range q {
		result = append(result, fromQueue(elem))
	}

	return result
}

func fromQueue(q rmqc.QueueInfo) domain.Queue {
	return domain.Queue{
		VHost: q.Vhost,
		Name:  q.Name,
		Type:  q.Type,
		State: q.Status,
		Messages: domain.Messages{
			Ready:   q.MessagesReady,
			Unacked: q.MessagesUnacknowledged,
			Total:   q.Messages,
		},
	}
}
