package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"resty.dev/v3"
	"strings"
)

type ProxiesContainer struct {
	Proxies            []map[string]any `yaml:"proxies"`
	TransparentHeaders map[string]string
}

type Set map[string]bool

func NewSet(e ...string) (result Set) {
	result = make(Set, len(e))
	for _, v := range e {
		result[v] = true
	}
	return
}

func (s Set) Has(e string) bool {
	_, ok := s[e]
	return ok
}

var RuleTypes = NewSet(
	"DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD", "DOMAIN-REGEX", "GEOSITE",
	"IP-CIDR", "IP-CIDR6", "IP-SUFFIX", "IP-ASN", "GEOIP", "SRC-GEOIP", "SRC-IP-ASN",
	"SRC-IP-CIDR", "SRC-IP-SUFFIX", "DST-PORT", "SRC-PORT", "IN-PORT", "IN-TYPE",
	"IN-USER", "IN-NAME", "PROCESS-PATH", "PROCESS-PATH", "PROCESS-PATH-REGEX",
	"PROCESS-PATH-REGEX", "PROCESS-NAME", "PROCESS-NAME", "PROCESS-NAME", "PROCESS-NAME-REGEX",
	"PROCESS-NAME-REGEX", "PROCESS-NAME-REGEX", "UID", "NETWORK", "DSCP", "RULE-SET", "AND",
	"OR", "NOT", "SUB-RULE",
)

var ClashHeaders = NewSet(
	"Content-Disposition", "Profile-Update-Interval",
	"Subscription-Userinfo", "Profile-Web-Page-Url",
)

func ExtractProxies(url string) (nodes ProxiesContainer, err error) {
	nodes = ProxiesContainer{
		TransparentHeaders: make(map[string]string),
	}

	L().Info(fmt.Sprintf("Fetching nodes: %s", url))

	client := resty.New()
	defer func() {
		err = client.Close()
	}()

	req := client.R()
	req.SetHeader("User-Agent", "clash-verge/v2.2.3")

	res, err := req.Get(url)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(res.Bytes(), &nodes)
	if err != nil {
		return
	}

	headers := res.Header()

	for header := range ClashHeaders {
		headerValue := headers.Get(header)
		if headerValue != "" {
			nodes.TransparentHeaders[header] = headerValue
		}
	}

	return
}

func BuildTemplate(
	template string, Proxies ProxiesContainer, ruleLines []*Ruleset,
) (result map[string]any, err error) {
	err = yaml.Unmarshal([]byte(template), &result)
	result["proxies"] = Proxies.Proxies
	rules := make([]string, 0, 4096)

	for _, rule := range ruleLines {
		tag := rule.tag
		for _, r := range strings.Split(rule.content, "\n") {
			r = strings.TrimSpace(r)
			if len(r) == 0 || r[0] == '#' {
				continue
			}

			ruleComponents := strings.Split(r, ",")
			if len(ruleComponents) < 2 {
				err = fmt.Errorf("rules must have at least 2 componets: %s", rule.content)
				return
			}

			if !RuleTypes.Has(ruleComponents[0]) {
				continue
			}

			if len(ruleComponents) == 3 {
				rules = append(rules, fmt.Sprintf(
					"%s,%s,%s,%s",
					ruleComponents[0], ruleComponents[1], tag, ruleComponents[2],
				))
			} else {
				rules = append(rules, r+","+tag)
			}
		}
	}

	result["rules"] = rules

	return
}

func Marshal(y map[string]any) (result string, err error) {
	resultBytes, err := yaml.Marshal(y)
	if err != nil {
		return
	}

	result = string(resultBytes)

	return
}
