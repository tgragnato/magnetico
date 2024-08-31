package persistence

import (
	"crypto/rand"
	"crypto/sha1"
	mrand "math/rand"
	"testing"
	"text/template"

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
