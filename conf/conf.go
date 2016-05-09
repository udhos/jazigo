package conf

import (
	//"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

type AppConfig struct {
	MaxConfigFiles int
	Holdtime       time.Duration
	ScanInterval   time.Duration
	MaxConcurrency int
}

type DevAttributes struct {
	NeedLoginChat               bool     // need login chat
	NeedEnabledMode             bool     // need enabled mode
	NeedPagingOff               bool     // need disabled pager
	EnableCommand               string   // enable
	UsernamePromptPattern       string   // Username:
	PasswordPromptPattern       string   // Password:
	EnablePasswordPromptPattern string   // Password:
	DisabledPromptPattern       string   // >
	EnabledPromptPattern        string   // #
	CommandList                 []string // show run
	DisablePagerCommand         string   // term len 0
	SupressAutoLF               bool     // do not send auto LF

	// readTimeout: per-read timeout (protection against inactivity)
	// matchTimeout: full match timeout (protection against slow sender -- think 1 byte per second)
	ReadTimeout         time.Duration // protection against inactivity
	MatchTimeout        time.Duration // protection against slow sender
	SendTimeout         time.Duration // protection against inactivity
	CommandReadTimeout  time.Duration // larger timeout for slow responses (slow show running)
	CommandMatchTimeout time.Duration // larger timeout for slow responses (slow show running)
}

type DevConfig struct {
	Debug          bool
	Deleted        bool
	Model          string
	Id             string
	HostPort       string
	Transports     string
	LoginUser      string
	LoginPassword  string
	EnablePassword string
	Attr           DevAttributes
}

func NewDeviceFromString(str string) (*DevConfig, error) {
	b := []byte(str)
	c := &DevConfig{}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *DevConfig) Dump() ([]byte, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}
	return b, nil
}

type Config struct {
	Options AppConfig
	Devices []DevConfig
}

func New() *Config {
	return &Config{
		Options: AppConfig{
			Holdtime:       300 * time.Second, // FIXME: 12h (do not collect/save new backup before this timeout)
			ScanInterval:   60 * time.Second,  // FIXME: 30m (interval between full table scan)
			MaxConcurrency: 20,
			MaxConfigFiles: 120,
		},
		Devices: []DevConfig{},
	}
}

func Load(path string) (*Config, error) {
	b, readErr := ioutil.ReadFile(path)
	if readErr != nil {
		return nil, readErr
	}
	c := New()
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) Dump() ([]byte, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}
	return b, nil
}
