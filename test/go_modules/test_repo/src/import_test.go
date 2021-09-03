package src

import (
	"testing"

	"github.com/DataDog/zstd"
	"github.com/golang/snappy"
	"github.com/mattn/go-sqlite3"
	"github.com/peterebden/go-cli-init/v5/logging"
)

func TestCLIImport(t *testing.T) {
	logging.MustGetLogger()
}

func TestZSTImport(t *testing.T) {
	zstd.NewReader(nil)
}

func TestSQLLite3(t *testing.T) {
	_ = sqlite3.SQLiteStmt{}
}

func TestSnappy(t *testing.T) {
	_ = snappy.MaxEncodedLen(1234)
}
