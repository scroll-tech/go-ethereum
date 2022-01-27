package leveldb

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/iden3/go-merkletree/db"
	"github.com/iden3/go-merkletree/db/test"
	"github.com/stretchr/testify/require"
)

var rmDirs []string

func levelDbStorage(t *testing.T) db.Storage {
	dir, err := ioutil.TempDir("", "db")
	rmDirs = append(rmDirs, dir)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	sto, err := NewLevelDbStorage(dir, false)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	return sto
}

func TestLevelDb(t *testing.T) {
	test.TestReturnKnownErrIfNotExists(t, levelDbStorage(t))
	test.TestStorageInsertGet(t, levelDbStorage(t))
	test.TestStorageWithPrefix(t, levelDbStorage(t))
	test.TestConcatTx(t, levelDbStorage(t))
	test.TestList(t, levelDbStorage(t))
	test.TestIterate(t, levelDbStorage(t))
}

func TestLevelDbInterface(t *testing.T) {
	var db db.Storage //nolint:gosimple

	dir, err := ioutil.TempDir("", "db")
	require.Nil(t, err)
	rmDirs = append(rmDirs, dir)
	sto, err := NewLevelDbStorage(dir, false)
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
