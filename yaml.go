package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	"resty.dev/v3"
)

type SubscriptionInfo struct {
	Url        string
	Name       string
	Upload     int64
	Download   int64
	Total      int64
	Expire     int64
	StatusCode int
	RawBody    string
}

type ProxiesContainer struct {
	Proxies            []map[string]any `yaml:"proxies"`
	TransparentHeaders map[string]string
	SubInfos           []*SubscriptionInfo
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

func parseSubscriptionUserinfo(header string) (upload, download, total, expire int64) {
	pairs := strings.Split(header, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "upload":
			upload = val
		case "download":
			download = val
		case "total":
			total = val
		case "expire":
			expire = val
		}
	}
	return
}

func extractFilename(contentDisposition string) string {
	parts := strings.Split(contentDisposition, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "filename*=") {
			// RFC 2231 encoding
			value := strings.TrimPrefix(part, "filename*=")
			if idx := strings.Index(value, "''"); idx != -1 {
				value = value[idx+2:]
			}
			value = strings.Trim(value, "\"")
			// URL decode
			decoded, err := url.PathUnescape(value)
			if err == nil {
				value = decoded
			}
			return removeExtension(value)
		}
		if strings.HasPrefix(part, "filename=") {
			value := strings.TrimPrefix(part, "filename=")
			value = strings.Trim(value, "\"")
			return removeExtension(value)
		}
	}
	return ""
}

func removeExtension(filename string) string {
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		return filename[:idx]
	}
	return filename
}

func ExtractProxies(url string, name string) (nodes ProxiesContainer, err error) {
	nodes = ProxiesContainer{
		TransparentHeaders: make(map[string]string),
		SubInfos:           make([]*SubscriptionInfo, 0, 1),
	}

	subInfo := &SubscriptionInfo{
		Url:  url,
		Name: name,
	}

	L().Info(fmt.Sprintf("Fetching nodes: %s", url))

	client := resty.New()
	defer func() {
		err = client.Close()
	}()

	req := client.R()
	req.SetHeader("User-Agent", "clash.meta/v1.19.14")

	res, err := req.Get(url)
	if err != nil {
		return
	}

	subInfo.StatusCode = res.StatusCode()
	subInfo.RawBody = res.String()

	if res.StatusCode() != 200 {
		err = fmt.Errorf("%d\n%s", res.StatusCode(), res.String())
		return
	}

	err = yaml.Unmarshal(res.Bytes(), &nodes)
	if err != nil {
		return
	}

	headers := res.Header()

	contentDisposition := headers.Get("Content-Disposition")
	if contentDisposition != "" {
		filename := extractFilename(contentDisposition)
		if filename != "" {
			subInfo.Name = filename
		}
	}

	for header := range ClashHeaders {
		headerValue := headers.Get(header)
		if headerValue != "" {
			nodes.TransparentHeaders[header] = headerValue
		}
	}

	userinfoHeader := headers.Get("Subscription-Userinfo")
	if userinfoHeader != "" {
		upload, download, total, expire := parseSubscriptionUserinfo(userinfoHeader)
		subInfo.Upload = upload
		subInfo.Download = download
		subInfo.Total = total
		subInfo.Expire = expire
	}

	nodes.SubInfos = append(nodes.SubInfos, subInfo)

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
