package scamp

import (
	"fmt"
)

var DefaultCache *CacheRefresher

//Initialize performs package-level setup. This must be called before calling any other package functionality, as it sets up global configuration.
func Initialize(configPath string, refresherOptions *RefresherOptions) (err error) {
	initSCAMPLogger()
	err = initConfig(configPath)
	if err != nil {
		return
	}

	cachePath, found := DefaultConfig().Get("discovery.cache_path")
	if !found {
		err = fmt.Errorf("no such config param `discovery.cache_path`. must be set to use scamp-go")
		return
	}

	serviceCache, err := NewServiceCache(cachePath)
	if err != nil {
		return
	}

	DefaultCache = NewCacheRefresher(serviceCache, refresherOptions)
	return
}
