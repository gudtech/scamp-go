package scamp

import (
	"bufio"
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type CacheRefresher struct {
	lock    sync.RWMutex
	cache   *ServiceCache
	cancel  context.CancelFunc
	context context.Context
	running int32
	due     *time.Ticker

	initialRefresh bool
	options        RefresherOptions
}

type RefresherOptions struct {
	// Defaults to `5 * time.Second`
	WaitDuration *time.Duration
	// Call cache.Refresh when functions are called
	// Defaults to `true`
	Reactive bool
}

func NewCacheRefresher(cache *ServiceCache, options RefresherOptions) *CacheRefresher {
	waitDuration := 5 * time.Second
	if options.WaitDuration != nil {
		waitDuration = *options.WaitDuration
	}

	return &CacheRefresher{
		lock:    sync.RWMutex{},
		cache:   cache,
		cancel:  nil,
		context: nil,
		running: 0,
		due:     time.NewTicker(waitDuration),

		options: options,
	}
}

func (refresher *CacheRefresher) Reactive() bool {
	return refresher.options.Reactive
}

func (refresher *CacheRefresher) Run(ctx context.Context) {
	if refresher.Reactive() {
		return
	}

	if !atomic.CompareAndSwapInt32(&refresher.running, 0, 1) {
		return
	}

	context, cancel := context.WithCancel(ctx)
	refresher.context = context
	refresher.cancel = cancel

	go func() {
		defer atomic.AddInt32(&refresher.running, -1)

		err := refresher.cache.Refresh()
		if err != nil {
			Error.Printf("refresh cache: %v", err)
		}

		refresher.initialRefresh = true

		for {
			select {
			case <-refresher.context.Done():
				return
			case <-refresher.due.C:
			}

			refresher.markRefresh()
		}
	}()
}

func (refresher *CacheRefresher) markRefresh() {
	err := refresher.cache.Refresh()
	if err != nil {
		Error.Printf("refresh cache: %v", err)
	}

	refresher.initialRefresh = true
}

func (refresher *CacheRefresher) Running() bool {
	return atomic.LoadInt32(&refresher.running) > 0
}

func (refresher *CacheRefresher) Stop() {
	if refresher.cancel != nil {
		refresher.cancel()
	}

	for refresher.Running() {
		time.Sleep(1 * time.Millisecond)
	}

	refresher.context = nil
	refresher.cancel = nil
}

func (refresher *CacheRefresher) Due() bool {
	if refresher.initialRefresh {
		return true
	}

	select {
	case <-refresher.due.C:
		return true
	default:
		return false
	}
}

func (refresher *CacheRefresher) ReactiveRefresh() {
	if refresher.Reactive() && refresher.Due() {
		refresher.markRefresh()
	}
}

// --- Export functions from cache manager ---

func (refresher *CacheRefresher) DisableRecordVerification() {
	refresher.lock.RLock()
	defer refresher.lock.RUnlock()
	refresher.cache.DisableRecordVerification()
}

func (refresher *CacheRefresher) EnableRecordVerification() {
	refresher.lock.RLock()
	defer refresher.lock.RUnlock()
	refresher.cache.EnableRecordVerification()
}

func (refresher *CacheRefresher) Store(instance *serviceProxy) {
	refresher.lock.RLock()
	defer refresher.lock.RUnlock()
	refresher.cache.Store(instance)
}

func (refresher *CacheRefresher) ActionList() []string {
	refresher.lock.RLock()
	defer refresher.lock.RUnlock()

	refresher.ReactiveRefresh()

	return refresher.cache.ActionList()
}

func (refresher *CacheRefresher) Retrieve(ident string) (instance *serviceProxy) {
	refresher.lock.RLock()
	defer refresher.lock.RUnlock()

	refresher.ReactiveRefresh()

	return refresher.cache.Retrieve(ident)
}

func (refresher *CacheRefresher) SearchByAction(sector, action string, version int, envelope string) (instances []*serviceProxy, err error) {
	refresher.lock.RLock()
	defer refresher.lock.RUnlock()

	refresher.ReactiveRefresh()

	return refresher.cache.SearchByAction(sector, action, version, envelope)
}

func (refresher *CacheRefresher) Size() int {
	refresher.lock.RLock()
	defer refresher.lock.RUnlock()

	refresher.ReactiveRefresh()

	return refresher.cache.Size()
}

func (refresher *CacheRefresher) All() (proxies []*serviceProxy) {
	refresher.lock.RLock()
	defer refresher.lock.RUnlock()

	refresher.ReactiveRefresh()

	return refresher.cache.All()
}

func (refresher *CacheRefresher) Refresh() (err error) {
	refresher.lock.RLock()
	defer refresher.lock.RUnlock()
	return refresher.cache.Refresh()
}

func (refresher *CacheRefresher) DoScan(s *bufio.Scanner) (err error) {
	refresher.lock.RLock()
	defer refresher.lock.RUnlock()
	return refresher.cache.DoScan(s)
}
