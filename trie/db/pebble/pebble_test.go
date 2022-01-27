package pebble

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/iden3/go-merkletree/db"
	"github.com/iden3/go-merkletree/db/test"
	"github.com/stretchr/testify/require"
)

var rmDirs []string

func pebbleStorage(t *testing.T) db.Storage {
	dir, err := ioutil.TempDir("", "db")
	rmDirs = append(rmDirs, dir)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	sto, err := NewPebbleStorage(dir, false)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	return sto
}

func TestPebble(t *testing.T) {
	test.TestReturnKnownErrIfNotExists(t, pebbleStorage(t))
	test.TestStorageInsertGet(t, pebbleStorage(t))
	test.TestStorageWithPrefix(t, pebbleStorage(t))
	test.TestConcatTx(t, pebbleStorage(t))
	test.TestIterate(t, pebbleStorage(t))
	test.TestList(t, pebbleStorage(t))
}

func TestPebbleInterface(t *testing.T) {
	var db db.Storage //nolint:gosimple

	dir, err := ioutil.TempDir("", "db")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir)
	sto, err := NewPebbleStorage(dir, false)
	require.Nil(t, err)
	db = sto
	require.NotNil(t, db)
}

func TestMain(m *testing.M) {
	result := m.Run()
	for _, dir := range rmDirs {
		os.RemoveAll(dir) //nolint:errcheck,gosec
	}
	os.Exit(result)
}
