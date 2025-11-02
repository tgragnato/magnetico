//go:build cgo

package persistence

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sync"
	"time"

	zmq "github.com/pebbe/zmq4"
)

type zeromq struct {
	socket *zmq.Socket
	cache  map[string]time.Time
	sync.Mutex
}

func makeZeroMQ(url_ *url.URL) (Database, error) {
	url_.Scheme = "tcp"
	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		return nil, err
	}
	err = socket.Bind(url_.String())
	if err != nil {
		return nil, err
	}
	instance := &zeromq{
		socket: socket,
		cache:  map[string]time.Time{},
	}
	go func() {
		for range time.NewTicker(10 * time.Minute).C {
			go instance.cleanup()
		}
	}()
	return instance, nil
}

func (instance *zeromq) cleanup() {
	instance.Lock()
	defer instance.Unlock()

	for key, value := range instance.cache {
		if time.Now().After(value) {
			delete(instance.cache, key)
		}
	}
}

func (instance *zeromq) Engine() databaseEngine {
	return ZeroMQ
}

func (instance *zeromq) DoesTorrentExist(infoHash []byte) (bool, error) {
	instance.Lock()
	defer instance.Unlock()
	_, found := instance.cache[string(infoHash)]
	return found, nil
}

func (instance *zeromq) AddNewTorrent(infoHash []byte, name string, files []File) error {
	data, err := json.Marshal(SimpleTorrentSummary{
		InfoHash: hex.EncodeToString(infoHash),
		Name:     name,
		Files:    files,
	})
	if err != nil {
		return errors.New("failed to encode metadata " + err.Error())
	}

	instance.Lock()
	defer instance.Unlock()

	if _, found := instance.cache[string(infoHash)]; found {
		return errors.New("torrent already exists")
	}
	instance.cache[string(infoHash)] = time.Now().Add(10 * time.Minute)

	_, err = instance.socket.SendMessage(data)
	return err
}

func (instance *zeromq) Close() error {
	return instance.socket.Close()
}

func (instance *zeromq) GetNumberOfTorrents() (uint, error) {
	return 0, nil
}

func (instance *zeromq) GetNumberOfQueryTorrents(query string, epoch int64) (uint64, error) {
	return 0, nil
}

func (instance *zeromq) QueryTorrents(
	query string,
	epoch int64,
	orderBy OrderingCriteria,
	ascending bool,
	limit uint64,
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

func (instance *zeromq) Export() (chan SimpleTorrentSummary, error) {
	return nil, errors.New("export not supported")
}
