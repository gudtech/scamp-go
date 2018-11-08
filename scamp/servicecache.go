package scamp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"sync"
)

// ServiceCache represents data read from the scamp discovery cache
type ServiceCache struct {
	path          string
	cacheM        sync.Mutex
	identIndex    map[string]*serviceProxy
	actionIndex   map[string][]*serviceProxy
	verifyRecords bool
}

// NewServiceCache creates a new *ServiceCache and initializes its fields
func NewServiceCache(path string) (cache *ServiceCache, err error) {
	cache = new(ServiceCache)
	cache.path = path

	cache.identIndex = make(map[string]*serviceProxy)
	cache.actionIndex = make(map[string][]*serviceProxy)
	cache.verifyRecords = true

	//moving this here for now
	err = cache.Refresh()
	if err != nil {
		return
	}

	return
}

func (c *ServiceCache) disableRecordVerification() {
	c.verifyRecords = true
}

func (c *ServiceCache) enableRecordVerification() {
	c.verifyRecords = false
}

func (c *ServiceCache) Store(instance *serviceProxy) {
	c.cacheM.Lock()
	defer c.cacheM.Unlock()

	c.storeNoLock(instance)

	return
}

func (c *ServiceCache) storeNoLock(instance *serviceProxy) {
	_, ok := c.identIndex[instance.ident]
	if !ok {
		c.identIndex[instance.ident] = instance
	} else {
		// Not sure if this is a hard error yet
		// Error.Printf("tried to store instance that was already tracked")
		// Override existing version. Correct logic?
		c.identIndex[instance.ident] = instance
	}

	for _, class := range instance.classes {
		for _, action := range class.actions {
			for _, protocol := range instance.protocols {
				mungedName := fmt.Sprintf("%s:%s.%s~%d#%s", instance.sector, class.className, action.actionName, action.version, protocol)
				serviceProxies, ok := c.actionIndex[mungedName]
				if ok {
					serviceProxies = append(serviceProxies, instance)
				} else {
					serviceProxies = []*serviceProxy{instance}
				}

				c.actionIndex[mungedName] = serviceProxies
			}
		}
	}

	return
}

func (c *ServiceCache) removeNoLock(instance *serviceProxy) (err error) {
	_, ok := c.identIndex[instance.ident]
	if !ok {
		err = fmt.Errorf("tried removing an ident which was not being tracked: %s", instance.ident)
		return
	}

	delete(c.identIndex, instance.ident)

	return
}

// TODO: in a perfect world we'd do upserts to the cache
// and sweep for stale proxy definitions.
func (c *ServiceCache) clearNoLock() (err error) {
	c.identIndex = make(map[string]*serviceProxy)
	c.actionIndex = make(map[string][]*serviceProxy)

	return
}

func (c *ServiceCache) Retrieve(ident string) (instance *serviceProxy) {
	c.cacheM.Lock()
	defer c.cacheM.Unlock()

	instance, ok := c.identIndex[ident]
	if !ok {
		instance = nil
		return
	}

	return
}

func (c *ServiceCache) SearchByAction(sector, action string, version int, envelope string) (instances []*serviceProxy, err error) {
	mungedName := fmt.Sprintf("%s:%s~%d#%s", sector, action, version, envelope)
	instances = c.actionIndex[mungedName]
	if len(instances) == 0 {
		err = fmt.Errorf("no instances found: %s", mungedName)
		return
	}
	return
}

func (c *ServiceCache) Size() int {
	c.cacheM.Lock()
	defer c.cacheM.Unlock()

	return len(c.identIndex)
}

func (c *ServiceCache) All() (proxies []*serviceProxy) {
	c.cacheM.Lock()
	defer c.cacheM.Unlock()

	size := len(c.identIndex)
	proxies = make([]*serviceProxy, size)

	index := 0
	for _, proxy := range c.identIndex {
		proxies[index] = proxy
		index++
	}

	return
}

var sep = []byte(`%%%`)
var newline = []byte("\n")

func (c *ServiceCache) Refresh() (err error) {
	c.cacheM.Lock()
	defer c.cacheM.Unlock()

	stat, err := os.Stat(c.path)
	if err != nil {
		return
	} else if stat.IsDir() {
		err = fmt.Errorf("cannot use cache path: `%s` is a directory", c.path)
		return
	}

	cacheHandle, err := os.Open(c.path)
	if err != nil {
		return
	}
	defer cacheHandle.Close()

	s := bufio.NewScanner(cacheHandle)
	err = c.DoScan(s)
	if err != nil {
		return
	}

	return
}

func (c *ServiceCache) DoScan(s *bufio.Scanner) (err error) {
	c.clearNoLock()

	// var entries int = 0
	// Scan through buf by lines according to this basic ABNF
	// (SLOP* SEP CLASSRECORD NL CERT NL SIG NL NL)*
	var classRecordsRaw, certRaw, sigRaw []byte
	for {
		var didScan bool
		for {
			didScan = s.Scan()
			if bytes.Equal(s.Bytes(), sep) || !didScan {
				break
			}
		}
		if !didScan {
			break
		}
		s.Scan() // consume the separator

		if len(s.Bytes()) == 0 {
			// err = errors.New("unexpected newline after separator")
			break
		}
		classRecordsRaw = make([]byte, len(s.Bytes()))
		copy(classRecordsRaw, s.Bytes())
		s.Scan() // consume the classRecords

		if len(s.Bytes()) != 0 {
			err = errors.New("expected newline after class records")
			return
		}

		var certBuffer bytes.Buffer
		for s.Scan() {
			// Error.Printf("%s", s.Bytes())
			if len(s.Bytes()) == 0 {
				break
			}
			certBuffer.Write(s.Bytes())
			certBuffer.Write(newline)
		}
		certRaw = certBuffer.Bytes()[0 : len(certBuffer.Bytes())-1]

		var sigBuffer bytes.Buffer
		for s.Scan() {
			// Error.Printf("%s", s.Bytes())
			if len(s.Bytes()) == 0 {
				break
			} else if bytes.Equal(s.Bytes(), sep) {
				break
			}
			sigBuffer.Write(s.Bytes())
			sigBuffer.Write(newline)
		}
		sigRaw = sigBuffer.Bytes()[0 : len(sigBuffer.Bytes())-1]

		// Error.Printf("`%s`", sigRaw)

		// Use those extracted value to make an instance
		serviceProxy, err := newServiceProxy(classRecordsRaw, certRaw, sigRaw)
		if err != nil {
			return fmt.Errorf("newServiceProxy err: %s", err)
		}

		// Validating is a very expensive operation in the benchmarks
		if c.verifyRecords {
			err = serviceProxy.Validate()
			if err != nil {
				err = c.removeNoLock(serviceProxy)
				if err != nil {
					Error.Printf("could not remove service proxy (benign on first pass, otherwise it means the service has gone to a bad state): `%s`", err)
				}
				continue
			}
		}

		c.storeNoLock(serviceProxy)
	}

	return
}

var startCert = []byte(`-----BEGIN CERTIFICATE-----`)
var endCert = []byte(`-----END CERTIFICATE-----`)

func scanCertficates(data []byte, atEOF bool) (advance int, token []byte, err error) {
	var i int

	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// assert cert start line
	if i = bytes.Index(data, startCert); i == -1 {
		return 0, nil, nil
	}

	// assert end line, consume if present
	if i = bytes.Index(data, endCert); i >= 0 {
		return i + len(endCert), data[0 : i+len(endCert)], nil
	}
	return 0, nil, nil
}

// RETRY LOGIC
// TODO: cannot implement rety until cache.Refresh() is rewritten to support updating the cache rather than clearing and rebuilding
// each time it is called

// MaxRetries is the maximum number of retries before bailing.
var MaxRetries = 10

var errMaxRetriesReached = errors.New("exceeded retry limit")

// // Func represents functions that can be retried.
// type Func func(attempt int) (retry bool, err error)

// // Do keeps trying the function until the second argument
// // returns false, or no error is returned.
// func Do(fn Func) error {
// 	var err error
// 	var cont bool
// 	attempt := 1
// 	for {
// 		cont, err = fn(attempt)
// 		if !cont || err == nil {
// 			break
// 		}
// 		attempt++
// 		if attempt > MaxRetries {
// 			return errMaxRetriesReached
// 		}
// 	}
// 	return err
// }

// IsMaxRetries checks whether the error is due to hitting the
// maximum number of retries or not.
func IsMaxRetries(err error) bool {
	return err == errMaxRetriesReached
}
