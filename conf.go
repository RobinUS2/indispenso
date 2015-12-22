package main

import (
	"github.com/spf13/viper"
	"strings"
	"regexp"
	"os"
	"fmt"
	"github.com/spf13/pflag"
)


type Conf struct {
	SecureToken       string // Pre-shared token in configuration, never via the wire
	Hostname          string
	TagsList          []string
	UseAutoTag        bool
	ServerEnabled     bool
	EndpointURI       string
	ServerPort        int
	SslCertFile       string // TLS certificate file
	SslPrivateKeyFile string // Private key file
	AutoGenerateCert  bool
	ClientPort        int
	Debug             bool
}


func newConfigV() *Conf {
	c := new(Conf)

	viper.AddConfigPath("config")
	viper.AddConfigPath("/etc/indispenso/")
	viper.SetConfigName("indispenso")
	viper.SetEnvPrefix("ind")


	// Defaults
	viper.SetDefault("SecureToken","")
	viper.SetDefault("Hostname",getDefaultHostName())
	viper.SetDefault("UseAutoTag",true)
	viper.SetDefault("ServerEnabled",true)
	viper.SetDefault("Debug",false)

	configFile := pflag.StringP("Config","c","","Config file location default is /etc/indispenso/indispenso.{json,toml,yaml,yml,properties,props,prop}")
	pflag.BoolP("serverEnabled","s",true,"Deine if server module shoud be started or not")
	pflag.BoolP("Debug","d", false, "Enable debug mode" )

	viper.SetDefault("ServerPort",897)
	viper.SetDefault("EndpointURI","")
	viper.SetDefault("SslCertFile","./cert.pem" )
	viper.SetDefault("SslPrivateKeyFile", "./key.pem" )
	viper.SetDefault("AutoGenerateCert", true )
	viper.SetDefault("ClientPort", 898 )

	pflag.Parse()
	if( len(*configFile) > 2 ) {
		viper.SetConfigFile(*configFile)
	}
	viper.BindPFlags(pflag.CommandLine)
	viper.AutomaticEnv()
	viper.ReadInConfig()

	viper.Unmarshal(c)

	if c.Debug {
		log.Printf("Configuration: %+v", c)
	}
	return c
}

func getDefaultHostName() string {
	if hostname, err := os.Hostname(); err == nil{
		return hostname
	}
	return "localhost"
}

func (c *Conf) Validate() {
	// Must have token
	minLen := 32
	if len(strings.TrimSpace(c.SecureToken)) < minLen {
		log.Fatal(fmt.Sprintf("Must have secure token with minimum length of %d", minLen))
	}
}

func (c *Conf) isClientEnabled()bool{
	return len(conf.EndpointURI) > 0
}

func (c *Conf)getTags() []string {
	tagsList := viper.GetStringSlice("tagslist")
	if viper.GetBool("useautotag") {
		autoTags := c.hostTagDiscovery()
		tagsList = append(tagsList,autoTags...)
	}

	return tagsList
}

// Auto tag
func (c *Conf) hostTagDiscovery() []string {

	tokens := strings.FieldsFunc(c.Hostname, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})
	ret := make([]string,len(tokens))

	numbersOnlyRegexp, _ := regexp.Compile("^[[:digit:]]+$")
	numbersRegexp, _ := regexp.Compile("[[:digit:]]")
	for _, token := range tokens {
		cleanTag := cleanTag(token)
		// Min 2 characters && not just numbers && not only numbers
		if len(cleanTag) >= 2 && !numbersOnlyRegexp.MatchString(cleanTag) {
			// Count numbers
			numberCount := float64(len(numbersRegexp.FindAllStringSubmatch(cleanTag, -1)))
			strLen := float64(len(cleanTag))
			if numberCount >= strLen*0.5 {
				// More than half is numbers, ignore
				continue
			}
			ret = append(ret, cleanTag)
		}
	}

	return ret
}

// Clean tag
func cleanTag(in string) string {
	tagRegexp, _ := regexp.Compile("^[[:alnum:]-]+$")
	cleanTag := strings.ToLower(strings.TrimSpace(in))
	// Must be alphanumeric
	if !tagRegexp.MatchString(cleanTag) {
		return ""
	}
	return cleanTag
}

