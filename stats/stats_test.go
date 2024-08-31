package stats

import "testing"

var stats = &Stats{extensions: map[string]uint64{}}

func TestStats_IncBootstrap(t *testing.T) {
	t.Parallel()

	stats.IncBootstrap()
	if stats.bootstrap != 1 {
		t.Errorf("Expected bootstrap count to be 1, but got %d", stats.bootstrap)
	}
}

func TestStats_IncUDPError(t *testing.T) {
	t.Parallel()

	stats.IncUDPError(true)
	if stats.writeError != 1 {
		t.Errorf("Expected write error count to be 1, but got %d", stats.writeError)
	}

	stats.IncUDPError(false)
	if stats.readError != 1 {
		t.Errorf("Expected read error count to be 1, but got %d", stats.readError)
	}
}

func TestStats_IncRtClearing(t *testing.T) {
	t.Parallel()

	stats.IncRtClearing()
	if stats.rtClearing != 1 {
		t.Errorf("Expected rtClearing count to be 1, but got %d", stats.rtClearing)
	}
}

func TestStats_IncNonUTF8(t *testing.T) {
	t.Parallel()

	stats.IncNonUTF8()
	if stats.nonUTF8 != 1 {
		t.Errorf("Expected nonUTF8 count to be 1, but got %d", stats.nonUTF8)
	}
}
func TestStats_IncDBError_Add(t *testing.T) {
	t.Parallel()

	stats.IncDBError(true)
	if stats.addError != 1 {
		t.Errorf("Expected add error count to be 1, but got %d", stats.addError)
	}

	stats.IncDBError(true)
	if stats.addError != 2 {
		t.Errorf("Expected add error count to be 2, but got %d", stats.addError)
	}
}

func TestStats_IncDBError_Check(t *testing.T) {
	t.Parallel()

	stats.IncDBError(false)
	if stats.checkError != 1 {
		t.Errorf("Expected check error count to be 1, but got %d", stats.checkError)
	}

	stats.IncDBError(false)
	if stats.checkError != 2 {
		t.Errorf("Expected check error count to be 2, but got %d", stats.checkError)
	}
}

func TestStats_IncLeech(t *testing.T) {
	t.Parallel()

	stats.IncLeech(true, [8]byte{1, 2, 3, 4, 5, 6, 7, 8})
	if stats.mseEncryption != 1 {
		t.Errorf("Expected mseEncryption count to be 1, but got %d", stats.mseEncryption)
	}
	if stats.extensions["[1 2 3 4 5 6 7 8]"] != 1 {
		t.Errorf("Expected extensions count to be 1, but got %d", stats.extensions["[1 2 3 4 5 6 7 8]"])
	}

	stats.IncLeech(false, [8]byte{1, 2, 3, 4, 5, 6, 7, 8})
	if stats.plaintext != 1 {
		t.Errorf("Expected plaintext count to be 1, but got %d", stats.plaintext)
	}
	if stats.extensions["[1 2 3 4 5 6 7 8]"] != 2 {
		t.Errorf("Expected extensions count to be 2, but got %d", stats.extensions["[1 2 3 4 5 6 7 8]"])
	}
}
