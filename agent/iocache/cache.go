// Copyright © 2017 Joyent, Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package iocache

import (
	"context"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/bschofield/pg_prefaulter/agent/fhcache"
	"github.com/bschofield/pg_prefaulter/agent/structs"
	"github.com/bschofield/pg_prefaulter/config"
	"github.com/bschofield/pg_prefaulter/lib"
	log "github.com/rs/zerolog/log"
)

// IOCache is a read-through cache to:
//
// a) provide a reentrant interface
// b) deduplicate page pread(2) requests (i.e. no thundering-herd for the same
//    page file)
// c) prevent tainting of the filesystem cache (i.e. ZFS ARC) by artificially
//    promoting pages from the MRU to the MFU.
// d) sized sufficiently large so that we can spend our time faulting in pages
//    vs performing cache hits.
type IOCache struct {
	ctx context.Context
	wg  sync.WaitGroup
	cfg *config.IOCacheConfig

	purgeLock sync.Mutex
	c         gcache.Cache
	fhCache   *fhcache.FileHandleCache
}

// New creates a new IOCache.
func New(ctx context.Context, cfg *config.Config, fhc *fhcache.FileHandleCache) (*IOCache, error) {
	ioc := &IOCache{
		ctx:     ctx,
		cfg:     &cfg.IOCacheConfig,
		fhCache: fhc,
	}

	ioWorkQueue := make(chan structs.IOCacheKey)
	for ioWorker := uint(0); ioWorker < ioc.cfg.MaxConcurrentIOs; ioWorker++ {
		ioc.wg.Add(1)
		go func(threadID uint) {
			defer func() {
				ioc.wg.Done()
			}()

			for {
				select {
				case <-ioc.ctx.Done():
					return
				case ioReq, ok := <-ioWorkQueue:
					if !ok {
						return
					}

					if err := ioc.fhCache.PrefaultPage(ioReq); err != nil {
						// If we had a problem prefaulting in the WAL file, for whatever
						// reason, attempt to remove it from the cache.
						ioc.c.Remove(ioReq)

						log.Warn().Uint("io-worker-thread-id", threadID).Err(err).
							Uint64("database", uint64(ioReq.Database)).
							Uint64("relation", uint64(ioReq.Relation)).
							Uint64("block", uint64(ioReq.Block)).Msg("unable to prefault page")
					}
				}
			}
		}(ioWorker)
	}
	log.Info().Uint("io-worker-threads", ioc.cfg.MaxConcurrentIOs).Msg("started IO worker threads")

	ioc.c = gcache.New(int(ioc.cfg.Size)).
		ARC().
		LoaderExpireFunc(func(key interface{}) (interface{}, *time.Duration, error) {
			select {
			case <-ioc.ctx.Done():
			case ioWorkQueue <- key.(structs.IOCacheKey):
			}

			return struct{}{}, &ioc.cfg.TTL, nil

		}).
		Build()

	go lib.LogCacheStats(ioc.ctx, ioc.c, "iocache-stats")

	return ioc, nil
}

// GetIFPresent forwards to gcache.Cache's GetIFPresent().
func (ioc *IOCache) GetIFPresent(k interface{}) (interface{}, error) {
	return ioc.c.GetIFPresent(k)
}

// Purge purges the IOCache of its cache (and all downstream caches)
func (ioc *IOCache) Purge() {
	ioc.purgeLock.Lock()
	defer ioc.purgeLock.Unlock()

	ioc.c.Purge()
	ioc.fhCache.Purge()
}

// Wait blocks until the IOCache finishes shutting down its workers.
func (ioc *IOCache) Wait() {
	ioc.wg.Wait()
}
