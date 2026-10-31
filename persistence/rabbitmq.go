package persistence

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitMQ struct {
	url               string
	conn              *amqp.Connection
	ch                *amqp.Channel
	publisherConfirms <-chan amqp.Confirmation
	dataQueue         *amqp.Queue
	dhtQueue          *amqp.Queue
	dhtOutboundQueue  *amqp.Queue
	dhtMaxNeighbors   uint
	dhtFilterNodes    []net.IPNet
	dhtConfigured     bool
	dhtPending        map[string]amqp.Delivery

	cache map[string]time.Time
	sync.Mutex
}

func makeRabbitMQ(url_ *url.URL) (Database, error) {
	r := new(rabbitMQ)
	r.url = url_.String()
	if err := r.connect(); err != nil {
		return nil, err
	}

	r.cache = map[string]time.Time{}
	go func() {
		for range time.NewTicker(10 * time.Minute).C {
			go r.cleanup()
		}
	}()

	return r, nil
}

func (r *rabbitMQ) cleanup() {
	r.Lock()
	defer r.Unlock()

	for key, value := range r.cache {
		if time.Now().After(value) {
			delete(r.cache, key)
		}
	}
}

func (r *rabbitMQ) connect() (err error) {
	r.conn, err = amqp.Dial(r.url)
	if err != nil {
		return
	}
	r.ch, err = r.conn.Channel()
	if err != nil {
		return
	}
	err = r.ch.Confirm(false)
	if err != nil {
		return
	}
	r.publisherConfirms = r.ch.NotifyPublish(make(chan amqp.Confirmation, 1))

	r.dataQueue = new(amqp.Queue)
	*r.dataQueue, err = r.ch.QueueDeclare(
		"magnetico",
		true,
		false,
		false,
		false,
		amqp.Table{},
	)
	if err != nil {
		return err
	}
	r.dhtPending = make(map[string]amqp.Delivery)
	r.dhtQueue = new(amqp.Queue)
	*r.dhtQueue, err = r.ch.QueueDeclare(
		"magnetico_dht_routing",
		true,
		false,
		false,
		false,
		amqp.Table{},
	)
	if err != nil {
		return err
	}
	r.dhtOutboundQueue = new(amqp.Queue)
	*r.dhtOutboundQueue, err = r.ch.QueueDeclare(
		"magnetico_dht_outbound",
		true,
		false,
		false,
		false,
		amqp.Table{},
	)
	return
}

func (r *rabbitMQ) Engine() databaseEngine {
	return RabbitMQ
}

func (r *rabbitMQ) DoesTorrentExist(infoHash []byte) (bool, error) {
	r.Lock()
	defer r.Unlock()
	_, found := r.cache[string(infoHash)]
	return found, nil
}

func (r *rabbitMQ) AddNewTorrent(infoHash []byte, name string, files []File) error {
	data, err := json.Marshal(SimpleTorrentSummary{
		InfoHash: hex.EncodeToString(infoHash),
		Name:     name,
		Files:    files,
	})
	if err != nil {
		return errors.New("failed to encode metadata " + err.Error())
	}

	r.Lock()
	defer r.Unlock()

	if r.ch.IsClosed() || r.conn.IsClosed() {
		if err := r.connect(); err != nil {
			return err
		}
	}

	if _, found := r.cache[string(infoHash)]; found {
		return errors.New("torrent already exists")
	}

	err = r.publishConfirmedLocked(context.Background(), r.dataQueue.Name, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         data,
	})
	if err == nil {
		r.cache[string(infoHash)] = time.Now().Add(10 * time.Minute)
	}

	return err
}

func (r *rabbitMQ) publishConfirmedLocked(ctx context.Context, queue string, message amqp.Publishing) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.publisherConfirms == nil {
		return errors.New("rabbitmq: publisher confirmations are not configured")
	}
	if err := r.ch.Publish("", queue, false, false, message); err != nil {
		return fmt.Errorf("rabbitmq: publish: %w", err)
	}
	return awaitRabbitPublisherConfirmation(ctx, r.publisherConfirms)
}

func awaitRabbitPublisherConfirmation(ctx context.Context, confirmations <-chan amqp.Confirmation) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case confirmation, ok := <-confirmations:
		if !ok {
			return errors.New("rabbitmq: publisher confirmation channel closed")
		}
		if !confirmation.Ack {
			return fmt.Errorf("rabbitmq: broker rejected publish with delivery tag %d", confirmation.DeliveryTag)
		}
		return nil
	}
}

func (r *rabbitMQ) Close() error {
	if err := r.ch.Close(); err != nil {
		return err
	}
	return r.conn.Close()
}

func (r *rabbitMQ) GetNumberOfTorrents() (uint, error) {
	return 0, nil
}

func (r *rabbitMQ) GetNumberOfQueryTorrents(query string, epoch int64) (uint64, error) {
	return 0, nil
}

func (r *rabbitMQ) QueryTorrents(query string, epoch int64, orderBy OrderingCriteria, ascending bool, limit uint64, lastOrderedValue *float64, lastID *uint64) ([]TorrentMetadata, error) {
	return nil, errors.New("query not supported")
}

func (r *rabbitMQ) GetTorrent(infoHash []byte) (*TorrentMetadata, error) {
	return nil, errors.New("fetch not supported")
}

func (r *rabbitMQ) GetFiles(infoHash []byte) ([]File, error) {
	return nil, errors.New("file fetch not supported")
}

func (r *rabbitMQ) GetStatistics(from string, n uint) (*Statistics, error) {
	return nil, errors.New("statistics not supported")
}

func (r *rabbitMQ) Export() (chan SimpleTorrentSummary, error) {
	return nil, errors.New("export not supported")
}

type rabbitDHTEvent struct {
	Sequence uint64
	Added    []string
	Consumed []string
	Hashes   [][20]byte
}

type rabbitDHTState struct {
	nodes  map[string]uint
	hashes [10][20]byte
}

func (r *rabbitMQ) ConfigureDHTRoutingTable(maxNeighbors uint, filterNodes []net.IPNet) error {
	r.Lock()
	defer r.Unlock()
	if r.dhtQueue == nil {
		return fmt.Errorf("persistence: RabbitMQ DHT queue is not initialized")
	}
	r.dhtMaxNeighbors = maxNeighbors
	r.dhtFilterNodes = append([]net.IPNet(nil), filterNodes...)
	r.dhtConfigured = true
	return nil
}

func (r *rabbitMQ) dhtEventsLocked() ([]rabbitDHTEvent, error) {
	if !r.dhtConfigured || r.dhtQueue == nil {
		return nil, fmt.Errorf("persistence: RabbitMQ DHT routing is not configured")
	}
	queue, err := r.ch.QueueDeclarePassive(r.dhtQueue.Name, true, false, false, false, amqp.Table{})
	if err != nil {
		return nil, fmt.Errorf("persistence: inspect RabbitMQ DHT queue: %w", err)
	}
	deliveries := make([]amqp.Delivery, 0, queue.Messages)
	for i := 0; i < queue.Messages; i++ {
		delivery, ok, err := r.ch.Get(r.dhtQueue.Name, false)
		if err != nil {
			for _, pending := range deliveries {
				_ = pending.Nack(false, true)
			}
			return nil, fmt.Errorf("persistence: read RabbitMQ DHT queue: %w", err)
		}
		if !ok {
			break
		}
		deliveries = append(deliveries, delivery)
	}

	events := make([]rabbitDHTEvent, 0, len(deliveries))
	var eventErr error
	for _, delivery := range deliveries {
		var event rabbitDHTEvent
		if err := json.Unmarshal(delivery.Body, &event); err != nil {
			if eventErr == nil {
				eventErr = fmt.Errorf("persistence: decode RabbitMQ DHT event: %w", err)
			}
		} else {
			events = append(events, event)
		}
		if err := delivery.Nack(false, true); err != nil && eventErr == nil {
			eventErr = fmt.Errorf("persistence: requeue RabbitMQ DHT event: %w", err)
		}
	}
	if eventErr != nil {
		return nil, eventErr
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Sequence < events[j].Sequence })
	return events, nil
}

func (r *rabbitMQ) dhtStateLocked() (rabbitDHTState, uint64, error) {
	events, err := r.dhtEventsLocked()
	if err != nil {
		return rabbitDHTState{}, 0, err
	}
	state := rabbitDHTState{nodes: make(map[string]uint)}
	var sequence uint64
	for _, event := range events {
		if event.Sequence > sequence {
			sequence = event.Sequence
		}
		if len(event.Added) > 0 {
			if uint64(len(state.nodes))+uint64(len(event.Added)) > uint64(r.dhtMaxNeighbors)*uint64(r.dhtMaxNeighbors) {
				clear(state.nodes)
			}
			for _, node := range event.Added {
				state.nodes[node]++
			}
		}
		for _, node := range event.Consumed {
			if state.nodes[node] > 1 {
				state.nodes[node]--
			} else {
				delete(state.nodes, node)
			}
		}
		for i := 0; i < len(event.Hashes) && i < len(state.hashes); i++ {
			state.hashes[i] = event.Hashes[i]
		}
	}
	return state, sequence, nil
}

func (r *rabbitMQ) appendDHTEventLocked(ctx context.Context, event rabbitDHTEvent, sequence uint64) error {
	event.Sequence = sequence + 1
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("persistence: encode RabbitMQ DHT event: %w", err)
	}
	if err := r.publishConfirmedLocked(ctx, r.dhtQueue.Name, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
	}); err != nil {
		return fmt.Errorf("persistence: publish RabbitMQ DHT event: %w", err)
	}
	return nil
}

func (r *rabbitMQ) AddDHTNodes(ctx context.Context, nodes []net.UDPAddr) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.Lock()
	defer r.Unlock()
	_, sequence, err := r.dhtStateLocked()
	if err != nil {
		return err
	}
	added := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if rabbitDHTNodeAllowed(node, r.dhtFilterNodes) {
			added = append(added, node.String())
		}
	}
	if len(added) == 0 {
		return nil
	}
	return r.appendDHTEventLocked(ctx, rabbitDHTEvent{Added: added}, sequence)
}

func (r *rabbitMQ) GetDHTNodes(ctx context.Context) ([]net.UDPAddr, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.Lock()
	defer r.Unlock()
	state, sequence, err := r.dhtStateLocked()
	if err != nil {
		return nil, err
	}
	result := make([]net.UDPAddr, 0, r.dhtMaxNeighbors)
	consumed := make([]string, 0, r.dhtMaxNeighbors)
	for node := range state.nodes {
		if uint(len(result)) >= r.dhtMaxNeighbors {
			break
		}
		addr, err := rabbitDHTAddrFromString(node)
		if err != nil {
			continue
		}
		result = append(result, addr)
		consumed = append(consumed, node)
	}
	if len(consumed) > 0 {
		if err := r.appendDHTEventLocked(ctx, rabbitDHTEvent{Consumed: consumed}, sequence); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *rabbitMQ) IsDHTRoutingEmpty(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	r.Lock()
	defer r.Unlock()
	state, _, err := r.dhtStateLocked()
	if err != nil {
		return false, err
	}
	return len(state.nodes) == 0, nil
}

func (r *rabbitMQ) DumpDHTNodes(ctx context.Context, ipv4 bool) ([]net.UDPAddr, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.Lock()
	defer r.Unlock()
	state, _, err := r.dhtStateLocked()
	if err != nil {
		return nil, err
	}
	result := make([]net.UDPAddr, 0, 100)
	for node := range state.nodes {
		addr, err := rabbitDHTAddrFromString(node)
		if err != nil || ipv4 != (addr.IP.To4() != nil) {
			continue
		}
		result = append(result, addr)
		if len(result) == 100 {
			break
		}
	}
	return result, nil
}

func (r *rabbitMQ) AddDHTHashes(ctx context.Context, hashes [][20]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.Lock()
	defer r.Unlock()
	_, sequence, err := r.dhtStateLocked()
	if err != nil {
		return err
	}
	if len(hashes) > 10 {
		hashes = hashes[:10]
	}
	return r.appendDHTEventLocked(ctx, rabbitDHTEvent{Hashes: hashes}, sequence)
}

func (r *rabbitMQ) GetDHTHashes(ctx context.Context) ([10][20]byte, error) {
	if err := ctx.Err(); err != nil {
		return [10][20]byte{}, err
	}
	r.Lock()
	defer r.Unlock()
	state, _, err := r.dhtStateLocked()
	return state.hashes, err
}

func (r *rabbitMQ) EnqueueDHTMessage(ctx context.Context, message DHTOutboundMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	message, err := prepareDHTOutboundMessage(message)
	if err != nil {
		return err
	}
	r.Lock()
	defer r.Unlock()
	if r.ch.IsClosed() || r.conn.IsClosed() {
		if err := r.connect(); err != nil {
			return err
		}
	}
	queue, err := r.dhtOutboundQueueLocked()
	if err != nil {
		return err
	}
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("rabbitmq: encode DHT message: %w", err)
	}
	if err := r.publishConfirmedLocked(ctx, queue.Name, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		MessageId:    message.ID,
		Body:         body,
	}); err != nil {
		return fmt.Errorf("rabbitmq: enqueue DHT message: %w", err)
	}
	return nil
}

func (r *rabbitMQ) NextDHTMessage(ctx context.Context) (DHTOutboundMessage, bool, error) {
	if err := ctx.Err(); err != nil {
		return DHTOutboundMessage{}, false, err
	}
	r.Lock()
	defer r.Unlock()
	if r.ch.IsClosed() || r.conn.IsClosed() {
		if err := r.connect(); err != nil {
			return DHTOutboundMessage{}, false, err
		}
	}
	for id, delivery := range r.dhtPending {
		var message DHTOutboundMessage
		if err := json.Unmarshal(delivery.Body, &message); err != nil {
			return DHTOutboundMessage{}, false, fmt.Errorf("rabbitmq: decode pending DHT message: %w", err)
		}
		message.ID = id
		return message, true, nil
	}
	queue, err := r.dhtOutboundQueueLocked()
	if err != nil {
		return DHTOutboundMessage{}, false, err
	}
	delivery, ok, err := r.ch.Get(queue.Name, false)
	if err != nil {
		return DHTOutboundMessage{}, false, fmt.Errorf("rabbitmq: dequeue DHT message: %w", err)
	}
	if !ok {
		return DHTOutboundMessage{}, false, nil
	}
	var message DHTOutboundMessage
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		_ = delivery.Nack(false, true)
		return DHTOutboundMessage{}, false, fmt.Errorf("rabbitmq: decode DHT message: %w", err)
	}
	if message.ID == "" {
		message.ID = delivery.MessageId
	}
	if r.dhtPending == nil {
		r.dhtPending = make(map[string]amqp.Delivery)
	}
	r.dhtPending[message.ID] = delivery
	return message, true, nil
}

func (r *rabbitMQ) AckDHTMessage(ctx context.Context, messageID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.Lock()
	defer r.Unlock()
	delivery, ok := r.dhtPending[messageID]
	if !ok {
		return fmt.Errorf("rabbitmq: DHT message %q is not pending", messageID)
	}
	if err := delivery.Ack(false); err != nil {
		return fmt.Errorf("rabbitmq: acknowledge DHT message: %w", err)
	}
	delete(r.dhtPending, messageID)
	return nil
}

func (r *rabbitMQ) ReleaseDHTMessage(ctx context.Context, messageID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.Lock()
	defer r.Unlock()
	delivery, ok := r.dhtPending[messageID]
	if !ok {
		return nil
	}
	var message DHTOutboundMessage
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		return fmt.Errorf("rabbitmq: decode released DHT message: %w", err)
	}
	message.QueuedAt = time.Now().UnixNano()
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("rabbitmq: encode released DHT message: %w", err)
	}
	queue, err := r.dhtOutboundQueueLocked()
	if err != nil {
		return err
	}
	if err := r.publishConfirmedLocked(ctx, queue.Name, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		MessageId:    message.ID,
		Body:         body,
	}); err != nil {
		return fmt.Errorf("rabbitmq: requeue DHT message at tail: %w", err)
	}
	if err := delivery.Ack(false); err != nil {
		return fmt.Errorf("rabbitmq: acknowledge released DHT message: %w", err)
	}
	delete(r.dhtPending, messageID)
	return nil
}

func (r *rabbitMQ) dhtOutboundQueueLocked() (*amqp.Queue, error) {
	if r.dhtOutboundQueue == nil {
		return nil, errors.New("rabbitmq: DHT outbound queue is not initialized")
	}
	return r.dhtOutboundQueue, nil
}

func rabbitDHTNodeAllowed(node net.UDPAddr, filterNodes []net.IPNet) bool {
	if len(filterNodes) > 0 {
		for _, filterNode := range filterNodes {
			if filterNode.Contains(node.IP) {
				return true
			}
		}
		return false
	}
	if !node.IP.IsGlobalUnicast() || node.IP.IsPrivate() {
		return false
	}
	return node.Port == 80 || node.Port == 443 || node.Port >= 1024 && node.Port <= 65535
}

func rabbitDHTAddrFromString(value string) (net.UDPAddr, error) {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return net.UDPAddr{}, err
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return net.UDPAddr{}, err
	}
	return net.UDPAddr{IP: net.ParseIP(host), Port: portNumber}, nil
}
