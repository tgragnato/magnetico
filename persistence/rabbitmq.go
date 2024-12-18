package persistence

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitMQ struct {
	url       string
	conn      *amqp.Connection
	ch        *amqp.Channel
	dataQueue *amqp.Queue

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

	r.dataQueue = new(amqp.Queue)
	*r.dataQueue, err = r.ch.QueueDeclare(
		"magnetico",
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

	err = r.ch.Publish(
		"",
		r.dataQueue.Name,
		false,
		false,
		amqp.Publishing(amqp.Publishing{
			Body: data,
		}),
	)
	if err == nil {
		r.cache[string(infoHash)] = time.Now().Add(10 * time.Minute)
	}

	return err
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
