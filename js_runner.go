package main

import (
	"fmt"
	"github.com/dop251/goja"
	"golang.org/x/sync/errgroup"
	"runtime/debug"
)

type Ruleset struct {
	tag     string
	url     string
	content string
}

func downloadRulesets(vm *goja.Runtime) (resultLines []*Ruleset, err error) {
	rulesetsFunc := func(func(string, string)) {}
	jsRulesetsFunc := vm.Get("rulesets")

	if jsRulesetsFunc == nil {
		return
	}

	err = vm.ExportTo(jsRulesetsFunc, &rulesetsFunc)

	if err != nil {
		return
	}

	errGroup := new(errgroup.Group)
	limiter := make(chan bool, 8)
	urlList := make([]string, 0, 8)
	resultCh := make(chan *Ruleset, 8)

	rulesetsFunc(func(tag string, url string) {
		urlList = append(urlList, url)
		errGroup.Go(func() error {
			limiter <- true
			defer func() {
				<-limiter
			}()

			content, e := GetOrPut(url, FetchString)
			if e != nil {
				return e
			}

			resultCh <- &Ruleset{
				tag:     tag,
				url:     url,
				content: content,
			}
			return nil
		})
	})

	resultMap := make(map[string]*Ruleset)
	collected := make(chan bool, 1)
	go func() {
		for line := range resultCh {
			resultMap[line.url] = line
		}
		collected <- true
	}()

	err = errGroup.Wait()
	close(resultCh)
	if err != nil {
		return nil, err
	}

	<-collected
	resultLines = make([]*Ruleset, len(urlList))
	for i, url := range urlList {
		resultLines[i] = resultMap[url]
	}

	return
}

func ExecJs(script string, template string, proxies ProxiesContainer) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("[panic] %v\n%s", r, string(debug.Stack()))
		}
	}()

	vm := goja.New()
	err = vm.Set("log", func(v any) {
		L().Info(fmt.Sprintf("[JS] %v", v))
	})
	if err != nil {
		return
	}

	_, err = vm.RunString(script)
	if err != nil {
		return
	}

	ruleLines, err := downloadRulesets(vm)
	if err != nil {
		return
	}

	conf, err := BuildTemplate(template, proxies, ruleLines)
	if err != nil {
		return
	}

	buildConfigFunc := func(map[string]any) {}
	jsBuildConfigFunc := vm.Get("buildConfig")
	if jsBuildConfigFunc != nil {
		err = vm.ExportTo(jsBuildConfigFunc, &buildConfigFunc)
		if err != nil {
			return
		}
	}

	buildConfigFunc(conf)

	result, err = Marshal(conf)

	return
}
