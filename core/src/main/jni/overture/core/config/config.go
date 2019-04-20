// Copyright (c) 2016 shawn1m. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package config

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/shadowsocks/overture/core/cache"
	"github.com/shadowsocks/overture/core/common"
	"github.com/shadowsocks/overture/core/hosts"
)

type Config struct {
	BindAddress           string `json:"BindAddress"`
	HTTPAddress           string `json:"HTTPAddress"`
	PrimaryDNS            []*common.DNSUpstream
	AlternativeDNS        []*common.DNSUpstream
	OnlyPrimaryDNS        bool
	IPv6UseAlternativeDNS bool
	IPNetworkFile         struct {
		Primary     string
		Alternative string
	}
	AclFile     string
	HostsFile   string
	MinimumTTL  int
	CacheSize   int
	RejectQtype []uint16

	DomainPrimaryList        []string
	DomainAlternativeList    []string
	IPNetworkPrimaryList     []*net.IPNet
	IPNetworkAlternativeList []*net.IPNet
	Hosts                    *hosts.Hosts
	Cache                    *cache.Cache
}

// New config with json file and do some other initiate works
func NewConfig(configFile string) *Config {

	config := parseJson(configFile)

	config.getAclList()
	config.IPNetworkPrimaryList = getIPNetworkList(config.IPNetworkFile.Primary)
	config.IPNetworkAlternativeList = getIPNetworkList(config.IPNetworkFile.Alternative)

	if config.MinimumTTL > 0 {
		log.Info("Minimum TTL is " + strconv.Itoa(config.MinimumTTL))
	} else {
		log.Info("Minimum TTL is disabled")
	}

	config.Cache = cache.New(config.CacheSize)
	if config.CacheSize > 0 {
		log.Info("CacheSize is " + strconv.Itoa(config.CacheSize))
	} else {
		log.Info("Cache is disabled")
	}

	h, err := hosts.New(config.HostsFile)
	if err != nil {
		log.Info("Load hosts file failed: ", err)
	} else {
		config.Hosts = h
		log.Info("Load hosts file successful")
	}

	return config
}

func parseJson(path string) *Config {

	f, err := os.Open(path)
	if err != nil {
		log.Fatal("Open config file failed: ", err)
		os.Exit(1)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal("Read config file failed: ", err)
		os.Exit(1)
	}

	j := new(Config)
	err = json.Unmarshal(b, j)
	if err != nil {
		log.Fatal("Json syntex error: ", err)
		os.Exit(1)
	}

	return j
}

func (c *Config) getAclList() {
	f, err := os.Open(c.AclFile)
	if err != nil {
		log.Error("Open ACL file failed: ", err)
		return
	}
	defer f.Close()

	// Based on: https://stackoverflow.com/a/17871737/2245107
	subnetTester := regexp.MustCompile(`^(((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])\.){3,3}(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])(/(3[0-2]|[12]?[0-9]))?|(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))(/(12[0-8]|1[01][0-9]|[1-9]?[0-9]))?)$`)

	var inProxyList = true

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		switch line {
		case "[outbound_block_list]":
			panic("outbound_block_list unsupported")
		case "[black_list]", "[bypass_list]":
			inProxyList = false
		case "[white_list]", "[proxy_list]":
			inProxyList = true
		case "[reject_all]", "[bypass_all]", "[accept_all]", "[proxy_all]":
		default:
			if len(line) > 0 && !strings.HasPrefix(line, "#") && !subnetTester.MatchString(line) {
				_, err := regexp.Compile(line)
				if err == nil {
					if inProxyList {
						c.DomainAlternativeList = append(c.DomainAlternativeList, line)
					} else {
						c.DomainPrimaryList = append(c.DomainPrimaryList, line)
					}
				}
			}
		}
	}
}

func getIPNetworkList(file string) []*net.IPNet {

	ipnl := make([]*net.IPNet, 0)
	f, err := os.Open(file)
	if err != nil {
		log.Error("Open IP network file failed: ", err)
		return nil
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		_, ip_net, err := net.ParseCIDR(s.Text())
		if err != nil {
			break
		}
		ipnl = append(ipnl, ip_net)
	}

	if len(ipnl) > 0 {
		log.Info("Load " + file + " successful")
	} else {
		log.Warn("There is no element in " + file)
	}

	return ipnl
}
