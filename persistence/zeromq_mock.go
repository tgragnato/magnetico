//go:build !cgo

package persistence

import (
	"errors"
	"net/url"
)

type zeromq struct{}

func makeZeroMQ(url_ *url.URL) (Database, error) {
	return &zeromq{}, nil
}

func (instance *zeromq) Engine() databaseEngine {
	return ZeroMQ
}

func (instance *zeromq) DoesTorrentExist(infoHash []byte) (bool, error) {
	return false, nil
}

func (instance *zeromq) AddNewTorrent(infoHash []byte, name string, files []File) error {
	return errors.New("add not supported")
}

func (instance *zeromq) Close() error {
	return nil
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
