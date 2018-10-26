package scamp

// Initialize performs package-level setup. This must be called before calling any other package functionality, as it sets up global configuration.
// TODO: move to package init() function
// func Initialize(configPath string) (err error) {
// 	initSCAMPLogger()
// 	err = initConfig(configPath)
// 	if err != nil {
// 		return
// 	}

// 	// TODO: move all of this
// 	cachePath, found := DefaultConfig().Get("discovery.cache_path")
// 	if !found {
// 		err = fmt.Errorf("no such config param `discovery.cache_path`. must be set to use scamp-go")
// 		return
// 	}

// 	DefaultCache, err = NewServiceCache(cachePath)
// 	if err != nil {
// 		return
// 	}

// 	return
// }
