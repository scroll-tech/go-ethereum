package zktrie

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatabase(t *testing.T) {
	db := NewZkTrieMemoryDb()
	db.UpdatePreimage(nil, nil)
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key_%d", i))
		value := []byte(fmt.Sprintf("value_%d", i))
		db.Init(key, value)
	}

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			key := []byte(fmt.Sprintf("key_%d", i))
			value := []byte(fmt.Sprintf("value_%d", i))
			err := db.Put(key, value)
			assert.NoError(t, err)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			key := []byte(fmt.Sprintf("key_%d", i))
			value := []byte(fmt.Sprintf("value_%d", i))
			gotValue, err := db.Get(key)
			assert.NoError(t, err)
			assert.Equal(t, value, gotValue)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 100; i < 200; i++ {
			key := []byte(fmt.Sprintf("key_%d", i))
			value, err := db.Get(key)
			assert.Equal(t, ErrKeyNotFound, err)
			assert.Nil(t, value)
		}
	}()

	wg.Wait()
}
