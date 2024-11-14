package persistence

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitMQ struct {
	// todo: This is just a simple use now, you need to be able to customize it later.
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchange string
	key      string

	dataQueue amqp.Queue

	ReconnectDelay time.Duration //todo: In the future, the reconnection interval of the rabbitmq queue will be controlled here.

	globalCtx        context.Context
	globalCtxCanFunc context.CancelFunc
}

func makeRabbitMQ(url_ *url.URL) (Database, error) {
	var err error
	// url_.Scheme = "amqp"
	rmq := new(rabbitMQ)
	rmq.globalCtx, rmq.globalCtxCanFunc = context.WithCancel(context.Background())
	publishCtx, publishCtxCanFunc := context.WithTimeout(rmq.globalCtx, 5*time.Second)
	defer publishCtxCanFunc()

	rmq.conn, err = amqp.Dial(url_.String())
	if err != nil {
		return nil, err
	}
	rmq.ch, err = rmq.conn.Channel()
	if err != nil {
		return nil, err
	}

	statusNotifyQueue, err := rmq.ch.QueueDeclare(
		"hello",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return nil, err
	}

	if err = rmq.ch.PublishWithContext(publishCtx,
		"",
		statusNotifyQueue.Name,
		false,
		false,
		amqp.Publishing(amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("Hello"),
		})); err != nil {
		return nil, err
	}

	rmq.dataQueue, err = rmq.ch.QueueDeclare(
		"magnet_data",
		true,
		false,
		false,
		false,
		nil)
	if err != nil {
		return nil, err
	}

	return rmq, nil
}

func (r *rabbitMQ) Engine() databaseEngine {
	return RabbitMQ
}

func (r *rabbitMQ) DoesTorrentExist(infoHash []byte) (bool, error) {
	return false, nil
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

	dataSize := len(data)
	if dataSize > 4708106 {
		return errors.New("encode data exceeds maximum 4708106")
	}

	publishCtx, canCtx := context.WithTimeout(r.globalCtx, 5*time.Second)
	defer canCtx()

	if err = r.ch.PublishWithContext(publishCtx,
		"",
		r.dataQueue.Name,
		false,
		false,
		amqp.Publishing(amqp.Publishing{
			Body: data,
		})); err != nil {
		return err
	}

	return nil
}

func (r *rabbitMQ) Close() error {
	var err error
	err = r.ch.Close()
	if err != nil {
		return err
	}
	err = r.conn.Close()
	if err != nil {
		return err
	}
	log.Println("successfully closed rabbitmq")
	return nil
}

func (r *rabbitMQ) GetNumberOfTorrents() (uint, error) {
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
