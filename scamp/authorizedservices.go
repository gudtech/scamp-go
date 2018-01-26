package scamp

// import "fmt"
import "errors"
import "bufio"
import "bytes"

// AuthorizedServiceSpec contains service's fingerprint and registered actions
type AuthorizedServiceSpec struct {
	Fingerprint []byte
	Actions     []serviceProxyClass
}

// AuthorizedServicesCache contains array of service's AuthorizedServiceSpec
type AuthorizedServicesCache struct {
	services []AuthorizedServiceSpec
}

// NewAuthorizedServicesCache Initializes amd returns a pointesr to a new AuthorizedServicesCache
func NewAuthorizedServicesCache() (cache *AuthorizedServicesCache) {
	cache = new(AuthorizedServicesCache)
	cache.services = make([]AuthorizedServiceSpec, 100)

	return
}

// LoadAuthorizedServices calls NewAuthorizedServicesCache() if *bufio.Scanner bytes are > 0
func (cache *AuthorizedServicesCache) LoadAuthorizedServices(s *bufio.Scanner) (err error) {
	var read bool
	count := 1

	for {
		read = s.Scan()

		if !read {
			break
			// Skip empty lines
		} else if len(s.Bytes()) == 0 {
			continue
		}

		_, err = NewAuthorizedServicesSpec(s.Bytes())
		if err != nil {
			// Trace.Printf("Error creating AuthorizedServicesCache: %s", err)
		}

		count = count + 1
	}

	return
}

// NewAuthorizedServicesSpec returns a pointer to an AuthorizedServiceSpec which contains the service's fingerprint and svailable actions
func NewAuthorizedServicesSpec(line []byte) (spec *AuthorizedServiceSpec, err error) {
	s := bufio.NewScanner(bytes.NewReader(line))
	s.Split(bufio.ScanWords)

	s.Scan()
	if len(s.Bytes()) == 0 || s.Bytes()[0] == '#' {
		err = errors.New("invalid")
		return
	}

	spec = new(AuthorizedServiceSpec)
	spec.Fingerprint = make([]byte, len(s.Bytes()))
	copy(spec.Fingerprint, s.Bytes())
	spec.Actions = make([]serviceProxyClass, 100) //TODO: do a better job allocating here

	index := 0
	var read bool
	for {
		read = s.Scan()
		if !read {
			break
		}

		// spec.Actions[index].className = make([]byte, len(s.Bytes()))
		spec.Actions[index].className = string(s.Bytes())
		index = index + 1
	}

	return
}
