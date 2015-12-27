package main

import (
	"fmt"
	"github.com/spf13/cast"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"regexp"
	"strings"
	"gopkg.in/fsnotify.v1"
)

type Conf struct {
	Token             string // Pre-shared token in configuration, never via the wire
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
	Home              string //home directory
	confFlags         *pflag.FlagSet
}

const defaultHomePath = "/etc/indispenso/"

func newConfig() *Conf {
	c := new(Conf)
	viper.SetConfigName("indispenso")
	viper.SetEnvPrefix("ind")

	// Defaults
	viper.SetDefault("Token", "")
	viper.SetDefault("Hostname", getDefaultHostName())
	viper.SetDefault("UseAutoTag", true)
	viper.SetDefault("ServerEnabled", false)
	viper.SetDefault("Home", defaultHomePath)
	viper.SetDefault("Debug", false)
	viper.SetDefault("ServerPort", 1897)
	viper.SetDefault("EndpointURI", "")
	viper.SetDefault("SslCertFile", "cert.pem")
	viper.SetDefault("SslPrivateKeyFile", "key.pem")
	viper.SetDefault("AutoGenerateCert", true)
	viper.SetDefault("ClientPort", 1898)

	//Flags
	c.confFlags = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	configFile := c.confFlags.StringP("Config", "c", "", "Config file location default is /etc/indispenso/indispenso.{json,toml,yaml,yml,properties,props,prop}")
	c.confFlags.BoolP("serverEnabled", "s", false, "Deine if server module shoud be started or not")
	c.confFlags.BoolP("debug", "d", false, "Enable debug mode")
	c.confFlags.StringP("home", "p", defaultHomePath, "Home direcotry where all config files are located")
	c.confFlags.StringP("endpointUri", "e", "", "URI of server interface, used by client")
	c.confFlags.StringP("token", "t", "", "Secret token")
	c.confFlags.StringP("hostname", "i", getDefaultHostName(), "Hostname that is use to identify itself")
	c.confFlags.BoolP("help", "h", false, "Print help message")

	c.confFlags.Parse(os.Args[1:])
	if len(*configFile) > 2 {
		viper.SetConfigFile(*configFile)
	}
	viper.BindPFlags(c.confFlags)
	viper.AutomaticEnv()

	homePath := viper.GetString("Home")
	if len(homePath ) > 0 {
		viper.AddConfigPath(homePath)
	}else{
		viper.AddConfigPath("config")
	}

	viper.ReadInConfig()
	c.Update()

	viper.OnConfigChange(func(in fsnotify.Event) { c.Update() } )
	viper.WatchConfig()

	return c
}

func (c *Conf )Update() {
	log.Println("Updating config")
	viper.Unmarshal(c)
	c.AutoRepair()
	if c.Debug {
		log.Printf("Configuration: %+v", c)
	}
}

func (c *Conf) IsHelp() bool {
	return viper.GetBool("help")
}

func (c *Conf) AutoRepair() {
	fullUriPattern, _ := regexp.Compile("(http[s]{0,1})://([^/:]+):?([0-9]{0,}).*")
	if !fullUriPattern.MatchString(c.EndpointURI) {
		protocol := "https"
		host := getDefaultHostName()
		port := cast.ToString(c.ServerPort)
		hostWithPortPattern, _ := regexp.Compile("([^:/]+):?([0-9]{0,})")
		repaired := false

		if hostWithPortPattern.MatchString(c.EndpointURI) {
			matches := hostWithPortPattern.FindAllStringSubmatch(c.EndpointURI, -1)
			if val := matches[0][1]; val != "" {
				host = val
			}
			if val := matches[0][2]; val != "" {
				port = val
			}
			repaired = true
		}

		if repaired {
			c.EndpointURI = fmt.Sprintf("%s://%s:%s/", protocol, host, port)
			log.Printf("EndpointURI successfully reparied to: %s", c.EndpointURI)
		}
	}
}

func (c *Conf) PrintHelp() {
	c.confFlags.Usage()
	os.Exit(0)
}

func (c *Conf) GetSslPrivateKeyFile() string {
	return c.HomeFile(c.SslPrivateKeyFile)
}

func (c *Conf) GetSslCertFile() string {
	return c.HomeFile(c.SslCertFile)
}

func (c *Conf) ConfFile() string {
	return viper.ConfigFileUsed()
}

func getDefaultHostName() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	return "localhost"
}

func (c *Conf) GetHome() string {
	if( c.Home == "/" ){
		return c.Home
	}
	return strings.TrimRight(c.Home, "/")
}

func (c *Conf) HomeFile(fileName string) string {
	return fmt.Sprintf("%s/%s", c.GetHome(), fileName)
}

func (c *Conf) ServerRequest(path string) string {
	return fmt.Sprintf("%s/%s", strings.TrimRight(conf.EndpointURI, "/"), strings.TrimLeft(path, "/"))
}

func (c *Conf) Validate() {
	// Must have token
	minLen := 32
	if len(strings.TrimSpace(c.Token)) < minLen {
		log.Fatal(fmt.Sprintf("Must have secure token with minimum length of %d", minLen))
	}

	if _, err := os.Stat(c.GetHome()); os.IsNotExist(err) {
		log.Fatal(fmt.Sprintf("Home directory doesn't exists: %s", c.GetHome()))
	}
}

func (c *Conf) isClientEnabled() bool {
	return len(conf.EndpointURI) > 0
}

func (c *Conf) GetTags() []string {
	tagsList := viper.GetStringSlice("tagslist")
	if viper.GetBool("useautotag") {
		autoTags := c.hostTagDiscovery()
		tagsList = append(tagsList, autoTags...)
	}

	return tagsList
}

// Auto tag
func (c *Conf) hostTagDiscovery() []string {

	tokens := strings.FieldsFunc(c.Hostname, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})
	ret := make([]string, len(tokens))

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
