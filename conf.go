package main

import (
	"github.com/kylelemons/go-gypsy/yaml"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Configuration
type Conf struct {
	Seed string
	SecureToken string
	IsServer bool
	tagsMux sync.RWMutex
	tags map[string]bool
}

// Get tags
func (c *Conf) Tags() []string {
	c.tagsMux.RLock()
	defer c.tagsMux.RUnlock()
	keys := make([]string, 0, len(c.tags))
	for k := range c.tags {
		keys = append(keys, k)
	}
	return keys
}

// Auto tag
func (c *Conf) autoTag() {
	c.tagsMux.Lock()
	defer c.tagsMux.Unlock()
	tokens := strings.FieldsFunc(hostname, func (r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})
	numbersOnlyRegexp, _ := regexp.Compile("^[[:digit:]]+$")
	for _, token := range tokens {
		cleanTag := c.cleanTag(token)
		// Min 2 characters && not just numbers && not only numbers
		if len(cleanTag) >= 2 && !numbersOnlyRegexp.MatchString(cleanTag) {
			// Count numbers
			numberCount := float64(len(numbersOnlyRegexp.FindAllStringSubmatch(cleanTag, -1)))
			strLen := float64(len(cleanTag))
			if numberCount >= strLen * 0.5 {
				// More than half is numbers, ignore
				continue
			}
			c.tags[cleanTag] = true
		}
	}
}

// Clean tag
func (c *Conf) cleanTag(in string) string {
	tagRegexp, _ := regexp.Compile("^[[:alnum:]-]+$")
	cleanTag := strings.ToLower(strings.TrimSpace(in))
	// Must be alphanumeric
	if !tagRegexp.MatchString(cleanTag) {
		return ""
	}
	return cleanTag
}

// Reload config every once in a while
func (co *Conf) startAutoReload() {
	go func() {
		c := time.Tick(time.Duration(60) * time.Second)
		for _ = range c {
			co.load()
		}
	}()
}

// Load config files
func (c *Conf) load() {
	c.tagsMux.Lock()
	defer c.tagsMux.Unlock()
	mainConf := "/etc/indispenso/indispenso.conf"
	additionalFilesPath := "/etc/indispenso/conf.d/*"
	files, _ := filepath.Glob(additionalFilesPath)
	files = append([]string{mainConf}, files...) // Prepend item
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			// Not existing
			continue
		}

		// Read
		conf, confErr := yaml.ReadFile(file)
		if confErr != nil {
			log.Printf("Failed reading %s: %v", file, confErr)
			continue
		}

		// Skip empty
		if conf == nil {
			continue
		}

		// Root map
		if conf.Root == nil {
			continue
		}
		rootMap := conf.Root.(yaml.Map)

		// Read base conf
		if file == mainConf {
			// Seed
			if rootMap.Key("seed") != nil {
				seed := rootMap.Key("seed").(yaml.Scalar).String()
				if len(seed) > 0 {
					c.Seed = seed
				}
			}

			// Secure token
			if rootMap.Key("secure_token") != nil {
				secureToken := rootMap.Key("secure_token").(yaml.Scalar).String()
				if len(secureToken) > 0 {
					c.SecureToken = secureToken
				}
			}

			// Server
			if rootMap.Key("server_enabled") != nil {
				serverEnabled := rootMap.Key("server_enabled").(yaml.Scalar).String()
				if len(serverEnabled) > 0 && (serverEnabled == "1" || serverEnabled == "true") {
					c.IsServer = true
				} else {
					c.IsServer = false
				}
			}
		}

		// Tags
		if rootMap.Key("tags") != nil {
			tags := rootMap.Key("tags").(yaml.List)
			if tags != nil {
				for _, tag := range tags {
					cleanTag := c.cleanTag(tag.(yaml.Scalar).String())
					if len(cleanTag) > 0 {
						c.tags[cleanTag] = true
					} else {
						log.Printf("Invalid tag %s, must be alphanumeric", tag)
					}
				}
			}
		}
	}
}

func newConf() *Conf {
	return &Conf{
		tags: make(map[string]bool),
	}
}
