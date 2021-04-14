package util

import (
	"sync"

	"github.com/0chain/gorocksdb"
)

/*PNodeDB - a node db that is persisted */
type PNodeDB struct {
	*MemoryNodeDB
	dataDir  string
	db       *gorocksdb.DB
	versions []int64
	mutex    sync.Mutex
}

/*NewPNodeDB - create a new PNodeDB */
func NewPNodeDB(dataDir string, logDir string) (*PNodeDB, error) {
	pnodedb := &PNodeDB{}
	pnodedb.dataDir = dataDir
	pnodedb.MemoryNodeDB = NewMemoryNodeDB()
	return pnodedb, nil
}

/*Flush - flush the db */
func (pndb *PNodeDB) Flush() {}

// Close close the rocksdb
func (pndb *PNodeDB) Close() {}

// GetDBVersions retusn all tracked db versions
func (pndb *PNodeDB) GetDBVersions() []int64 {
	pndb.mutex.Lock()
	defer pndb.mutex.Unlock()
	vs := make([]int64, len(pndb.versions))
	for i, v := range pndb.versions {
		vs[i] = v
	}
	return vs
}

// TrackDBVersion appends the db version to tracked records
func (pndb *PNodeDB) TrackDBVersion(v int64) {
	pndb.mutex.Lock()
	defer pndb.mutex.Unlock()
	pndb.versions = append(pndb.versions, v)
}
