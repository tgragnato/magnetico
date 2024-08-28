package persistence

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"time"

	"gopkg.in/patrickmn/go-cache.v2"
	zmq "gopkg.in/zeromq/goczmq.v4"
)

type zeromq struct {
	context *zmq.Sock
	cache   *cache.Cache
}

func makeZeroMQ(url_ *url.URL) (Database, error) {
	url_.Scheme = "tcp"
	instance := new(zeromq)
	context, err := zmq.NewPub(url_.String())
	if err != nil {
		return nil, err
	}
	instance.context = context
	instance.cache = cache.New(5*time.Minute, 10*time.Minute)
	return instance, nil
}

func (instance *zeromq) Engine() databaseEngine {
	return ZeroMQ
}

func (instance *zeromq) DoesTorrentExist(infoHash []byte) (bool, error) {
	_, found := instance.cache.Get(string(infoHash))
	return found, nil
}

func (instance *zeromq) AddNewTorrent(infoHash []byte, name string, files []File) error {
	data, err := json.Marshal(SimpleTorrentSummary{
		InfoHash: hex.EncodeToString(infoHash),
		Name:     name,
		Files:    files,
	})
	if err != nil {
		return errors.New("Failed to encode metadata " + err.Error())
	}
	err = instance.context.SendMessage([][]byte{data})
	if err != nil {
		return errors.New("Failed to transmit " + err.Error())
	}
	instance.cache.Set(string(infoHash), data, cache.DefaultExpiration)
	return nil
}

func (instance *zeromq) Close() error {
	instance.context.Destroy()
	return nil
}

func (instance *zeromq) GetNumberOfTorrents() (uint, error) {
	return 0, nil
}

func (instance *zeromq) QueryTorrents(
	query string,
	epoch int64,
	orderBy OrderingCriteria,
	ascending bool,
	limit uint,
	lastOrderedValue *float64,
	lastID *uint64,
) ([]TorrentMetadata, error) {
	return nil, errors.New("query not supported")
}

func (instance *zeromq) GetTorrent(infoHash []byte) (*TorrentMetadata, error) {
	return nil, errors.New("fetch not supported")
}

func (instance *zeromq) GetFiles(infoHash []byte) ([]File, error) {
	return nil, errors.New("file fetch not supported")
}

func (instance *zeromq) GetStatistics(from string, n uint) (*Statistics, error) {
	return nil, errors.New("statistics not supported")
}
