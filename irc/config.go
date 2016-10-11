// Copyright (c) 2012-2014 Jeremy Latt
// Copyright (c) 2014-2015 Edmund Huber
// Copyright (c) 2016- Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package irc

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type PassConfig struct {
	Password string
}

// TLSListenConfig defines configuration options for listening on TLS
type TLSListenConfig struct {
	Cert string
	Key  string
}

// Certificate returns the TLS certificate assicated with this TLSListenConfig
func (conf *TLSListenConfig) Config() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(conf.Cert, conf.Key)
	if err != nil {
		return nil, errors.New("tls cert+key: invalid pair")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, err
}

func (conf *PassConfig) PasswordBytes() []byte {
	bytes, err := DecodePassword(conf.Password)
	if err != nil {
		log.Fatal("decode password error: ", err)
	}
	return bytes
}

type AccountRegistrationConfig struct {
	Enabled          bool
	EnabledCallbacks []string `yaml:"enabled-callbacks"`
	Callbacks        struct {
		Mailto struct {
			Server string
			Port   int
			TLS    struct {
				Enabled            bool
				InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
				ServerName         string `yaml:"servername"`
			}
			Username             string
			Password             string
			Sender               string
			VerifyMessageSubject string `yaml:"verify-message-subject"`
			VerifyMessage        string `yaml:"verify-message"`
		}
	}
}

type Config struct {
	Network struct {
		Name string
	}

	Server struct {
		PassConfig
		Password         string
		Name             string
		Listen           []string
		Wslisten         string                      `yaml:"ws-listen"`
		TLSListeners     map[string]*TLSListenConfig `yaml:"tls-listeners"`
		CheckIdent       bool                        `yaml:"check-ident"`
		Log              string
		MOTD             string
		ProxyAllowedFrom []string `yaml:"proxy-allowed-from"`
	}

	Datastore struct {
		Path string
	}

	Registration struct {
		Accounts AccountRegistrationConfig
	}

	Operator map[string]*PassConfig

	Limits struct {
		NickLen       int  `yaml:"nicklen"`
		ChannelLen    int  `yaml:"channellen"`
		AwayLen       int  `yaml:"awaylen"`
		KickLen       int  `yaml:"kicklen"`
		TopicLen      int  `yaml:"topiclen"`
		WhowasEntries uint `yaml:"whowas-entries"`
	}
}

func (conf *Config) Operators() map[string][]byte {
	operators := make(map[string][]byte)
	for name, opConf := range conf.Operator {
		name, err := CasefoldName(name)
		if err == nil {
			operators[name] = opConf.PasswordBytes()
		} else {
			log.Println("Could not casefold oper name:", err.Error())
		}
	}
	return operators
}

func (conf *Config) TLSListeners() map[string]*tls.Config {
	tlsListeners := make(map[string]*tls.Config)
	for s, tlsListenersConf := range conf.Server.TLSListeners {
		config, err := tlsListenersConf.Config()
		if err != nil {
			log.Fatal(err)
		}
		name, err := CasefoldName(s)
		if err == nil {
			tlsListeners[name] = config
		} else {
			log.Println("Could not casefold TLS listener:", err.Error())
		}
	}
	return tlsListeners
}

func LoadConfig(filename string) (config *Config, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	// we need this so PasswordBytes returns the correct info
	if config.Server.Password != "" {
		config.Server.PassConfig.Password = config.Server.Password
	}

	if config.Network.Name == "" {
		return nil, errors.New("Network name missing")
	}
	if config.Server.Name == "" {
		return nil, errors.New("Server name missing")
	}
	if !IsHostname(config.Server.Name) {
		return nil, errors.New("Server name must match the format of a hostname")
	}
	if config.Datastore.Path == "" {
		return nil, errors.New("Datastore path missing")
	}
	if len(config.Server.Listen) == 0 {
		return nil, errors.New("Server listening addresses missing")
	}
	if config.Limits.NickLen < 1 || config.Limits.ChannelLen < 2 || config.Limits.AwayLen < 1 || config.Limits.TopicLen < 1 || config.Limits.TopicLen < 1 {
		return nil, errors.New("Limits aren't setup properly, check them and make them sane")
	}
	return config, nil
}
