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
	Token       string // Pre-shared token in configuration, never via the wire
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
	Home		      string //home directory
}

const defaultHomePath = "/etc/indispenso/"

func newConfig() *Conf {
	c := new(Conf)
	viper.SetConfigName("indispenso")
	viper.SetEnvPrefix("ind")


	// Defaults
	viper.SetDefault("Token","")
	viper.SetDefault("Hostname",getDefaultHostName())
	viper.SetDefault("UseAutoTag",true)
	viper.SetDefault("ServerEnabled",true)
	viper.SetDefault("Home",defaultHomePath)
	viper.SetDefault("Debug",false)
	viper.SetDefault("ServerPort",897)
	viper.SetDefault("EndpointURI","")
	viper.SetDefault("SslCertFile","cert.pem" )
	viper.SetDefault("SslPrivateKeyFile", "key.pem" )
	viper.SetDefault("AutoGenerateCert", true )
	viper.SetDefault("ClientPort", 898 )

	//Flags
	configFile := pflag.StringP("Config","c","","Config file location default is /etc/indispenso/indispenso.{json,toml,yaml,yml,properties,props,prop}")
	pflag.BoolP("serverEnabled","s",true,"Deine if server module shoud be started or not")
	pflag.BoolP("debug","d", false, "Enable debug mode" )
	pflag.StringP("home","p", defaultHomePath, "Enable debug mode" )
	pflag.StringP("endpointUri","e", "", "URI where server will listen for client requests" )
	pflag.StringP("Token","t", "", "Secret token" )
	pflag.BoolP("help","h", false, "Print help message" )

	pflag.Parse()
	if( len(*configFile) > 2 ) {
		viper.SetConfigFile(*configFile)
	}
	viper.BindPFlags(pflag.CommandLine)
	viper.AutomaticEnv()

	viper.AddConfigPath("config")
	viper.AddConfigPath(viper.GetString("Home"))
	viper.ReadInConfig()

	viper.Unmarshal(c)

	if c.Debug {
		log.Printf("Configuration: %+v", c)
	}
	return c
}

func (c *Conf) IsHelp() bool{
	return viper.GetBool("help")
}

func (c *Conf) PrintHelp() {
	pflag.Usage()
	os.Exit(0)
}

func (c *Conf) GetSslPrivateKeyFile() string{
	return c.HomeFile(c.SslPrivateKeyFile)
}


func (c *Conf) GetSslCertFile() string{
	return c.HomeFile(c.SslCertFile)
}

func (c *Conf) ConfFile() string{
	return viper.ConfigFileUsed()
}

func getDefaultHostName() string {
	if hostname, err := os.Hostname(); err == nil{
		return hostname
	}
	return "localhost"
}

func (c *Conf) GetHome() string{
	return strings.TrimRight(c.Home,"/")
}

func (c *Conf) HomeFile(fileName string) string{
	return fmt.Sprintf("%s/%s",c.GetHome(),fileName)
}

func (c *Conf) Validate() {
	// Must have token
	minLen := 32
	if len(strings.TrimSpace(c.Token)) < minLen {
		log.Fatal(fmt.Sprintf("Must have secure token with minimum length of %d", minLen))
	}

	if _,err := os.Stat(c.GetHome()); os.IsNotExist(err) {
		log.Fatal(fmt.Sprintf("Home directory doesn't exists: %s", c.GetHome()))
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

