package persistence

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	mrand "math/rand"
	"testing"
	"text/template"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresDatabase_ExecuteTemplate(t *testing.T) {
	t.Parallel()

	db := &postgresDatabase{}

	text := "Hello, {{.Name}}!"
	data := struct {
		Name string
	}{
		Name: "World",
	}

	expected := "Hello, World!"

	result := db.executeTemplate(text, data, template.FuncMap{})
	if result != expected {
		t.Errorf("Expected result to be %q, but got %q", expected, result)
	}
}

func TestPostgresDatabase_OrderOn(t *testing.T) {
	t.Parallel()

	db := &postgresDatabase{}

	testCases := []struct {
		orderBy  OrderingCriteria
		expected string
	}{
		{ByRelevance, "discovered_on"},
		{ByTotalSize, "total_size"},
		{ByDiscoveredOn, "discovered_on"},
		{ByNFiles, "n_files"},
	}

	for _, tc := range testCases {
		result := db.orderOn(tc.orderBy)
		if result != tc.expected {
			t.Errorf("Expected orderOn(%v) to return %q, but got %q", tc.orderBy, tc.expected, result)
		}
	}
}

func TestDoesTorrentExist(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	var random [100]byte
	_, err = rand.Read(random[:])
	if err != nil {
		for i := 0; i < 100; i++ {
			random[i] = byte(mrand.Intn(256))
		}
	}
	infohash := sha1.Sum(random[:])

	rows := sqlmock.NewRows([]string{"1"}).AddRow("1")
	mock.ExpectQuery("SELECT 1 FROM torrents WHERE info_hash = \\$1;").WithArgs(infohash[:]).WillReturnRows(rows)

	db := &postgresDatabase{conn: conn}
	found, err := db.DoesTorrentExist(infohash[:])
	if err != nil {
		t.Error(err)
	}
	if !found {
		t.Error("row returned but no result found")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}

	rows = sqlmock.NewRows([]string{"1"})
	mock.ExpectQuery("SELECT 1 FROM torrents WHERE info_hash = \\$1;").WithArgs(infohash[:]).WillReturnRows(rows)
	found, err = db.DoesTorrentExist(infohash[:])
	if err != nil {
		t.Error(err)
	}
	if found {
		t.Error("no row returned but result found")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPostgresDatabase_GetNumberOfTorrents(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	rows := sqlmock.NewRows([]string{"exact_count"}).AddRow(10)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)::BIGINT AS exact_count FROM torrents;").WillReturnRows(rows)

	db := &postgresDatabase{conn: conn}
	count, err := db.GetNumberOfTorrents()
	if err != nil {
		t.Error(err)
	}
	if count != 10 {
		t.Errorf("Expected count to be 10, but got %d", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}

	rows = sqlmock.NewRows([]string{"exact_count"})
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)::BIGINT AS exact_count FROM torrents;").WillReturnRows(rows)

	count, err = db.GetNumberOfTorrents()
	if err == nil {
		t.Error("no rows returned for query without corresponding error")
	}
	if count != 0 {
		t.Errorf("Expected count to be 0, but got %d", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}

	rows = sqlmock.NewRows([]string{"exact_count"}).AddRow(nil)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)::BIGINT AS exact_count FROM torrents;").WillReturnRows(rows)

	count, err = db.GetNumberOfTorrents()
	if err != nil {
		t.Error(err)
	}
	if count != 0 {
		t.Errorf("Expected count to be 0, but got %d", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPostgresDatabase_Close(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	db := &postgresDatabase{conn: conn}

	mock.ExpectClose()

	err = db.Close()
	if err != nil {
		t.Error(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPostgresDatabase_GetTorrent(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var random [100]byte
	_, err = rand.Read(random[:])
	if err != nil {
		for i := 0; i < 100; i++ {
			random[i] = byte(mrand.Intn(256))
		}
	}
	infohash := sha1.Sum(random[:])
	name := "Test Torrent"
	size := uint64(1024)
	discoveredOn := time.Now().Unix()

	rows := sqlmock.NewRows([]string{"info_hash", "name", "total_size", "discovered_on", "n_files"}).
		AddRow(infohash[:], name, size, discoveredOn, 5)
	mock.ExpectQuery("SELECT t.info_hash, t.name, t.total_size, t.discovered_on, \\(SELECT COUNT\\(\\*\\) FROM files f WHERE f.torrent_id = t.id\\) AS n_files FROM torrents t WHERE t.info_hash = \\$1;").
		WithArgs(infohash[:]).
		WillReturnRows(rows)

	db := &postgresDatabase{conn: conn}
	torrent, err := db.GetTorrent(infohash[:])
	if err != nil {
		t.Error(err)
	}
	if torrent == nil {
		t.Fatal("Expected torrent to be found, but got nil")
	}

	if !bytes.Equal(torrent.InfoHash, infohash[:]) {
		t.Errorf("Expected InfoHash to be %v, but got %v", infohash, torrent.InfoHash)
	}
	if torrent.Name != name {
		t.Errorf("Expected Name to be %q, but got %q", name, torrent.Name)
	}
	if torrent.Size != size {
		t.Errorf("Expected Size to be %d, but got %d", size, torrent.Size)
	}
	if torrent.DiscoveredOn != discoveredOn {
		t.Errorf("Expected DiscoveredOn to be %v, but got %v", discoveredOn, torrent.DiscoveredOn)
	}
	if torrent.NFiles != 5 {
		t.Errorf("Expected NFiles to be 5, but got %d", torrent.NFiles)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}

	rows = sqlmock.NewRows([]string{"info_hash", "name", "total_size", "discovered_on", "n_files"})
	mock.ExpectQuery("SELECT t.info_hash, t.name, t.total_size, t.discovered_on, \\(SELECT COUNT\\(\\*\\) FROM files f WHERE f.torrent_id = t.id\\) AS n_files FROM torrents t WHERE t.info_hash = \\$1;").
		WithArgs(infohash[:]).
		WillReturnRows(rows)

	torrent, err = db.GetTorrent(infohash[:])
	if err != nil {
		t.Error(err)
	}
	if torrent != nil {
		t.Error("Expected torrent to be not found, but got a result")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
