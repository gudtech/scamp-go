package scamp

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
)

// Configer interface for Config methods
type Configer interface {
	Load(string) error
	ServiceKeyPath(string, string) []byte
	ServiceCertPath(string, string) []byte
	Get(string) (string, bool)
	Set(string, string)
}

// Config represents scamp config
type Config struct {
	// string key for easy equals, byte return for easy nil
	values            map[string][]byte
	testMultiCastIP   net.IP
	testMultiCastPort int
}

// TODO: Will I regret using such a common name as a global variable?
var defaultConfig *Config

var defaultAnnounceInterval = 5

// DefaultConfigPath is the path at which the library will, by default, look for its configuration.
var DefaultConfigPath = "/etc/SCAMP/soa.conf"

var configLine = regexp.MustCompile(`^\s*([\S^=]+)\s*=\s*([\S]+)`)
var globalConfig *Config

var defaultGroupIP = net.IPv4(239, 63, 248, 106)
var defaultGroupPort = 5555

func initConfig(configPath string) (err error) {
	defaultConfig = NewConfig()
	err = DefaultConfig().Load(configPath)
	if err != nil {
		err = fmt.Errorf("could not load config: %s", err)
		return
	}

	randomDebuggerString = scampDebuggerRandomString()

	return
}

// NewConfig creates a new configuration struct with default values initialized.
func NewConfig() (conf *Config) {
	conf = new(Config)
	conf.values = make(map[string][]byte)

	return
}

// SetDefaultConfig sets the global configuration manually if need be.
func SetDefaultConfig(conf *Config) {
	defaultConfig = conf
}

// DefaultConfig fetches the global configuration struct for use.
// This function panics if the global configuration is not initialized (with `Initialize()`).
func DefaultConfig() (conf *Config) {
	if defaultConfig == nil {
		panic("defaultConfig is not initialized! initialize config before using package functionality.")
	}
	return defaultConfig
}

// Load loads configuration k/v pairs from the file at the given path.
func (c *Config) Load(configPath string) (err error) {
	file, err := os.Open(configPath)
	if err != nil {
		err = fmt.Errorf("couldn't read config from `%s`: %v", configPath, err)
		return
	}
	scanner := bufio.NewScanner(file)
	c.doLoad(scanner)

	return
}

func (c *Config) doLoad(scanner *bufio.Scanner) (err error) {
	var read bool
	for {
		read = scanner.Scan()
		if !read {
			break
		}

		re := configLine.FindSubmatch(scanner.Bytes())
		if re != nil {
			c.values[string(re[1])] = re[2]
		}
	}

	return
}

// ServiceKeyPath uses the configuration to generate a path at which the key for the given service name should be found.
func (c *Config) ServiceKeyPath(serviceName, optPath string) []byte {
	if len(optPath) != 0 {
		return []byte(fmt.Sprintf("%s/%s.key", optPath, serviceName))
	}
	path := c.values[serviceName+".soa_key"]
	if path == nil {
		path = []byte(fmt.Sprintf("/etc/GT_private/services/%s.key", serviceName))
	}
	return path
}

// ServiceCertPath uses the configuration to generate a path at which the certificate for the given service name should be found.
func (c *Config) ServiceCertPath(serviceName, optPath string) []byte {
	if len(optPath) != 0 {
		return []byte(fmt.Sprintf("%s/%s.crt", optPath, serviceName))
	}
	path := c.values[serviceName+".soa_cert"]
	if path == nil {
		path = []byte(fmt.Sprintf("/etc/GT_private/services/%s.crt", serviceName))
	}
	return path
}

// discoveryMulticastIP returns the configured discovery address, or the default one
// if there is no configured address (discovery.multicast_address)
func (c *Config) discoveryMulticastIP() (ip net.IP) {
	if c.testMultiCastIP != nil {
		return c.testMultiCastIP
	}
	rawAddr := c.values["discovery.multicast_address"]
	if rawAddr != nil {
		return net.ParseIP(string(rawAddr))
	}
	return defaultGroupIP
}

// DiscoveryMulticastPort returns the configured discovery port, or the default one
// if there is no configured port (discovery.port)
func (c *Config) discoveryMulticastPort() (port int) {
	if c.testMultiCastPort > 0 {
		return c.testMultiCastPort
	}
	portBytes := c.values["discovery.port"]
	if portBytes != nil {
		port64, err := strconv.ParseInt(string(portBytes), 10, 0)
		if err != nil {
			Error.Printf("could not parse discovery.port `%s`. falling back to default", err)
			port = int(defaultGroupPort)
		} else {
			port = int(port64)
		}

		return
	}
	port = defaultGroupPort
	return
}

func (c *Config) localDiscoveryMulticast() bool {
	_, ok := c.values["discovery.local_multicast"]
	return ok
}

// Get returns the value of a given config option as a string, or false if it is not set.
func (c *Config) Get(key string) (value string, ok bool) {
	valueBytes, ok := c.values[key]
	value = string(valueBytes)
	return
}

// Set sets the given key to the given value in the configuration
func (c *Config) Set(key string, value string) {
	valueBytes := []byte(value)
	c.values[key] = valueBytes
}
