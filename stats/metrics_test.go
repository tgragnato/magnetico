package stats

import (
	"bytes"
	"log"
	"testing"
)

func TestGetInstance(t *testing.T) {
	t.Parallel()

	instance1 := GetInstance()
	instance2 := GetInstance()
	if instance1 != instance2 {
		t.Error("Expected GetInstance to return the same instance")
	}

	if instance1.extensions == nil {
		t.Error("Expected GetInstance to initialize the extensions map")
	}
}

func TestFlush(t *testing.T) {
	t.Parallel()

	stats := &Stats{extensions: map[string]uint64{}}
	stats.bootstrap = 3
	stats.rtClearing = 2
	stats.writeError = 1
	stats.readError = 4
	stats.nonUTF8 = 5
	stats.checkError = 2
	stats.addError = 1
	stats.mseEncryption = 10
	stats.plaintext = 20
	stats.extensions = map[string]uint64{
		"ext1": 10,
		"ext2": 5,
		"ext3": 3,
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)

	stats.Flush()
	if len(buf.String()) == 0 {
		t.Error("Expected log message, but got an empty buffer")
	}

	if stats.bootstrap != 0 {
		t.Error("bootstrap was not reset")
	}
	if stats.rtClearing != 0 {
		t.Error("rtClearing was not reset")
	}
	if stats.writeError != 0 {
		t.Error("writeError was not reset")
	}
	if stats.readError != 0 {
		t.Error("readError was not reset")
	}
	if stats.nonUTF8 != 0 {
		t.Error("nonUTF8 was not reset")
	}
	if stats.checkError != 0 {
		t.Error("checkError was not reset")
	}
	if stats.addError != 0 {
		t.Error("addError was not reset")
	}
	if stats.mseEncryption != 0 {
		t.Error("mseEncryption was not reset")
	}
	if stats.plaintext != 0 {
		t.Error("plaintext was not reset")
	}
	if len(stats.extensions) != 0 {
		t.Error("Expected len(extensions) == 0, but it's not empty")
	}
}
