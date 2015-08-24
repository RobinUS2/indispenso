package main

import (
	"github.com/kylelemons/go-gypsy/yaml"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Configuration
type Conf struct {
	tags map[string]bool
}

// Get tags
func (c *Conf) Tags() []string {
	keys := make([]string, 0, len(c.tags))
	for k := range c.tags {
		keys = append(keys, k)
	}
	return keys
}

// Load config files
func (c *Conf) load() {
	mainConf := "/etc/indispenso.conf"
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

		// Read base conf
		if file == mainConf {
			seed := conf.Root.(yaml.Map).Key("seed").(yaml.Scalar).String()
			log.Printf("%v", seed)
		}

		// Tags
		tags := conf.Root.(yaml.Map).Key("tags").(yaml.List)
		tagRegexp, _ := regexp.Compile("[[:alnum:]]")
		if tags != nil {
			for _, tag := range tags {
				cleanTag := strings.ToLower(tag.(yaml.Scalar).String())
				if tagRegexp.MatchString(cleanTag) {
					c.tags[cleanTag] = true
				} else {
					log.Printf("Invalid tag %s, must be alphanumeric", tag)
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
