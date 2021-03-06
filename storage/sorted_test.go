package storage

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lucaslsl/faktory/util"
	"github.com/stretchr/testify/assert"
)

func TestBasicSortedSet(t *testing.T) {
	t.Parallel()

	defer os.RemoveAll("/tmp/sorted.db")
	db, err := Open("rocksdb", "/tmp/sorted.db")
	assert.NoError(t, err)
	defer db.Close()

	jid, j1 := fakeJob()
	past := time.Now()

	r := db.Retries()
	assert.Equal(t, "retries", r.Name())
	assert.EqualValues(t, 0, r.Size())
	err = r.AddElement(util.Thens(past), fmt.Sprintf("0%s", jid), j1)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, r.Size())

	jid, j2 := fakeJob()
	err = r.AddElement(util.Thens(past), fmt.Sprintf("1%s", jid), j2)
	assert.NoError(t, err)
	assert.EqualValues(t, 2, r.Size())

	current := time.Now()
	jid, j3 := fakeJob()
	err = r.AddElement(util.Thens(current.Add(10*time.Second)), jid, j3)
	assert.NoError(t, err)
	assert.EqualValues(t, 3, r.Size())

	results, err := r.RemoveBefore(util.Thens(current.Add(1 * time.Second)))
	assert.NoError(t, err)
	assert.EqualValues(t, 1, r.Size())
	assert.Equal(t, 2, len(results))
	values := [][]byte{j1, j2}
	assert.Equal(t, values, results)

	var key []byte
	r.Each(func(idx int, k, v []byte) error {
		cp := make([]byte, len(k))
		copy(cp, k)
		key = cp
		return nil
	})

	assert.NotNil(t, key)
	err = r.Remove(key)
	assert.NoError(t, err)
	assert.EqualValues(t, 0, r.Size())
}

func TestRocksSortedSet(b *testing.T) {
	b.Parallel()
	defer os.RemoveAll("/tmp/rocks.db")

	db, err := Open("rocksdb", "/tmp/rocks.db")
	assert.NoError(b, err)
	defer db.Close()

	count := 1000
	retries := db.Retries()
	start := time.Now()
	for i := 0; i < count; i++ {
		jid, job := fakeJob()
		ts := util.Thens(start.Add(time.Duration(10*i) * time.Second))
		err = retries.AddElement(ts, jid, job)
		assert.NoError(b, err)
	}

	pageSize := 12
	given := 0
	err = retries.Page(10, 12, func(idx int, key []byte, elm []byte) error {
		given++
		return nil
	})
	assert.NoError(b, err)
	assert.Equal(b, pageSize, given)

	amt := 0
	akey := []byte{}
	err = retries.Each(func(idx int, key []byte, elm []byte) error {
		akey = make([]byte, len(key))
		copy(akey, key)

		assert.True(b, len(key) > 40, key)
		assert.NotNil(b, elm)
		amt += 1
		return nil
	})
	assert.NoError(b, err)
	assert.EqualValues(b, count, amt)

	strs := strings.Split(string(akey), "|")
	assert.EqualValues(b, 0, db.Working().Size())
	err = retries.MoveTo(db.Working(), strs[0], strs[1], func(payload []byte) (string, []byte, error) {
		return util.Nows(), payload, nil
	})
	assert.NoError(b, err)
	assert.EqualValues(b, 1, db.Working().Size())
	assert.EqualValues(b, count-1, retries.Size())
	count -= 1

	err = retries.MoveTo(db.Working(), "1231", strs[1], func(payload []byte) (string, []byte, error) {
		return util.Nows(), payload, nil
	})
	assert.Error(b, err)

	remd := 0
	start = time.Now()
	for i := 0; i < count; i++ {
		ts := util.Thens(start.Add(time.Duration(5*i) * time.Second))
		elms, err := retries.RemoveBefore(ts)
		assert.NoError(b, err)
		remd += len(elms)
		assert.EqualValues(b, count-remd, retries.Size())
		assert.True(b, len(elms) == 0 || len(elms) == 1 || len(elms) == 2)
	}
	assert.EqualValues(b, 499, retries.Size())
	retries.Clear()
	assert.EqualValues(b, 0, retries.Size())
}

func fakeJob() (string, []byte) {
	return fakeJobWithPriority(5)
}

func fakeJobWithPriority(priority uint64) (string, []byte) {
	jid := util.RandomJid()
	nows := util.Nows()
	return jid, []byte(fmt.Sprintf(`{"jid":"%s","created_at":"%s","priority":%d,"queue":"default","args":[1,2,3],"class":"SomeWorker"}`, jid, nows, priority))
}
