package database

import (
	"database/sql"
	"sync"
	"time"

	"hacknhbackend.eparker.dev/util"
)

type Operation struct {
	execute func() error
	result  chan error
}

type DBQueue struct {
	operations chan Operation
	shutdown   chan struct{}
	wg         sync.WaitGroup
}

var (
	queue *DBQueue
	once  sync.Once
)

func GetQueue() *DBQueue {
	once.Do(func() {
		queue = &DBQueue{
			operations: make(chan Operation, util.Config.Database.QueueSize),
			shutdown:   make(chan struct{}),
		}
		queue.start()
	})

	return queue
}

func (q *DBQueue) start() {
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		for {
			select {
			case op := <-q.operations:
				err := op.execute()
				op.result <- err
			case <-q.shutdown:
				return
			}
		}
	}()
}

func (q *DBQueue) EnqueueOperation(operation func() error) error {
	resultChan := make(chan error, 1)
	op := Operation{
		execute: operation,
		result:  resultChan,
	}

	select {
	case q.operations <- op:
		return <-resultChan
	case <-time.After(5 * time.Second):
		return ErrorQueueTimeout
	}
}

func (q *DBQueue) Shutdown() {
	close(q.shutdown)
	q.wg.Wait()
	close(q.operations)
}

func QueuedExec(query string, args ...interface{}) error {
	return GetQueue().EnqueueOperation(func() error {
		_, err := db.Exec(query, args...)
		return err
	})
}

func QueuedQuery(query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	err := GetQueue().EnqueueOperation(func() error {
		var err error
		rows, err = db.Query(query, args...)
		return err
	})
	return rows, err
}

func QueuedQueryRow(query string, args ...interface{}) *sql.Row {
	var row *sql.Row
	_ = GetQueue().EnqueueOperation(func() error {
		row = db.QueryRow(query, args...)
		return nil
	})
	return row
}

func QueuedBegin() (*sql.Tx, error) {
	var tx *sql.Tx
	err := GetQueue().EnqueueOperation(func() error {
		var err error
		tx, err = db.Begin()
		return err
	})

	return tx, err
}
