package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

var (
	urlFlag      = flag.String("u", "", "目标网站地址")
	cookieFlag   = flag.String("c", "", "指定目标网站的Cookie")
	fileFlag     = flag.String("f", "", "包含URL或JS文件的路径")
	outputFlag   = flag.String("ou", "url.txt", "提取的URL输出文件名")
	allFlag      = flag.Bool("a", false, "提取并处理后的URL保存到url.txt文件中")
	pathDepth    = flag.Int("t", -1, "指定保留的路径层级数")
	client       *http.Client
	urlRegex     *regexp.Regexp
	processedURL = sync.Map{}
)

func init() {
	// 初始化忽略SSL验证的HTTP客户端
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: transport}

	// 编译正则表达式（优化版）
	pattern := `(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']*|(?:/|\./|\.\./)[^"'><,;| *()(%%$^/\\\[\]][^"'><,;|()]*|(?:[a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|/][^"|']*)?)|(?:[a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:\?[^"|']*)?)))(?:"|')`
	urlRegex = regexp.MustCompile(pattern)
}

func main() {

	red := "\033[31m"
	reset := "\033[0m"

	fmt.Println(red + "       _  _____ ______ _           _           ")
	fmt.Println("      | |/ ____|  ____(_)         | |          ")
	fmt.Println("      | | (___ | |__   _ _ __   __| | ___ _ __ ")
	fmt.Println("  _   | |\\___ \\|  __| | | '_ \\ / _` |/ _ \\ '__|")
	fmt.Println(" | |__| |____) | |    | | | | | (_| |  __/ |   ")
	fmt.Println("  \\____/|_____/|_|    |_|_| |_|\\__,_|\\___|_|   ")
	fmt.Println("                                               ")
	fmt.Println("JSFinder-go是一款用作快速在网站的js文件中提取URL的工具  https://github.com/kk12-30/JSFinder-go" + reset)
	fmt.Println("                                               ")
	flag.Parse()

	var allUrls []string
	var baseDomains []string

	// 收集基准域名
	if *urlFlag != "" {
		if u, err := url.Parse(*urlFlag); err == nil {
			baseDomains = append(baseDomains, u.Hostname())
			allUrls = append(allUrls, *urlFlag)
		}
	}

	if *fileFlag != "" {
		fileUrls, fileDoms := processInputFile(*fileFlag)
		allUrls = append(allUrls, fileUrls...)
		baseDomains = append(baseDomains, fileDoms...)
	}

	// 处理所有URL
	var processedUrls []string
	for _, u := range allUrls {
		if urls := processSingleURL(u); urls != nil {
			processedUrls = append(processedUrls, urls...)
		}
	}

	// 过滤和去重
	finalUrls := filterUrls(
		unique(append(allUrls, processedUrls...)),
		unique(baseDomains),
	)

	// 保存结果
	if *allFlag {
		fmt.Println("结果保存至url.txt文件中")
		saveUrls(finalUrls, *outputFlag)
	}

	// 输出结果
	fmt.Println("提取结果:")
	for _, u := range finalUrls {
		size := getContentLength(u)
		fmt.Printf("\033[32m%s\033[0m    [Size:%s]\n", u, formatSize(size))
	}
}

// 获取URL的Content-Length
func getContentLength(url string) int64 {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.108 Safari/537.36")
	if *cookieFlag != "" {
		req.Header.Set("Cookie", *cookieFlag)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	return resp.ContentLength
}

// 格式化文件大小
func formatSize(size int64) string {
	if size == 0 {
		return "Unknown"
	}
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// 核心过滤逻辑
func filterUrls(urls, baseDomains []string) []string {
	var filtered []string
domainLoop:
	for _, rawUrl := range urls {
		parsed, err := url.Parse(rawUrl)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}

		// 精确域名匹配检查
		currentHost := parsed.Hostname()
		for _, d := range baseDomains {
			if currentHost == d {
				// 处理路径（替换:id为1）
				currentPath := replaceIDInPath(parsed.Path)

				// 处理路径（仅对js、css、svg、xml扩展名进行处理）
				currentPath = cleanPath(currentPath)

				// 限制路径层级
				if *pathDepth > 0 {
					currentPath = limitPathDepth(currentPath, *pathDepth)
				}

				// 重建URL
				parsed.Path = currentPath
				parsed.RawQuery = ""
				parsed.Fragment = ""

				finalUrl := strings.TrimSuffix(parsed.String(), "/")
				filtered = append(filtered, finalUrl)
				continue domainLoop
			}
		}
	}
	return unique(filtered)
}

// 替换路径中的:id为1
func replaceIDInPath(path string) string {
	return strings.ReplaceAll(path, ":id", "1")
}

// 智能路径清理
func cleanPath(originalPath string) string {
	currentPath := path.Clean(originalPath)

	// 检查路径最后一段的扩展名
	base := path.Base(currentPath)
	ext := path.Ext(base)

	// 仅对js、css、svg、xml扩展名进行处理
	switch ext {
	case ".js", ".css", ".svg", ".xml", ".vue", ".ts":
		return path.Dir(currentPath)
	default:
		return currentPath
	}
}

// 限制路径层级
func limitPathDepth(path string, depth int) string {
	parts := strings.Split(path, "/")
	if len(parts) <= depth {
		return path
	}
	return strings.Join(parts[:depth+1], "/")
}

// 从文件读取初始URL并提取域名
func processInputFile(path string) ([]string, []string) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer file.Close()

	var (
		urls    []string
		domains []string
	)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		rawUrl := strings.TrimSpace(scanner.Text())
		if u, err := url.Parse(rawUrl); err == nil && u.Host != "" {
			urls = append(urls, rawUrl)
			domains = append(domains, u.Hostname())
		}
	}
	return urls, domains
}

// 处理单个URL
func processSingleURL(target string) []string {
	content := fetchContent(target)
	if content == "" {
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil
	}

	var (
		wg      sync.WaitGroup
		results = make(chan []string)
	)

	// 处理内联脚本
	wg.Add(1)
	go func() {
		defer wg.Done()
		scripts := extractInlineScripts(doc)
		results <- parseScripts(scripts, target)
	}()

	// 处理外部脚本
	externalScripts := extractExternalScripts(doc, target)
	for _, scriptURL := range externalScripts {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			content := fetchContent(u)
			results <- parseScripts([]string{content}, u)
		}(scriptURL)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var collected []string
	for res := range results {
		collected = append(collected, res...)
	}
	return collected
}

// 提取内联脚本
func extractInlineScripts(doc *goquery.Document) []string {
	var scripts []string
	doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		if _, exists := s.Attr("src"); !exists {
			scripts = append(scripts, s.Text())
		}
	})
	return scripts
}

// 提取外部脚本
func extractExternalScripts(doc *goquery.Document, baseURL string) []string {
	var scripts []string
	doc.Find("script[src]").Each(func(_ int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			scripts = append(scripts, resolveURL(baseURL, src))
		}
	})
	return scripts
}

// 解析脚本内容并提取URL
func parseScripts(scripts []string, baseURL string) []string {
	var urls []string
	for _, script := range scripts {
		matches := urlRegex.FindAllStringSubmatch(script, -1)
		for _, match := range matches {
			if len(match) > 1 {
				rawURL := strings.Trim(match[1], `"'`)
				absoluteURL := resolveURL(baseURL, rawURL)
				urls = append(urls, absoluteURL)
			}
		}
	}
	return urls
}

// 获取网页内容
func fetchContent(url string) string {
	if _, loaded := processedURL.LoadOrStore(url, true); loaded {
		return ""
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.108 Safari/537.36")
	if *cookieFlag != "" {
		req.Header.Set("Cookie", *cookieFlag)
	}

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

// 解析相对URL
func resolveURL(base, relative string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return relative
	}

	relURL, err := url.Parse(relative)
	if err != nil {
		return relative
	}

	return baseURL.ResolveReference(relURL).String()
}

// 去重函数
func unique(input []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, val := range input {
		if !seen[val] {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}

// 保存URL到文件
func saveUrls(urls []string, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("创建文件失败: %v\n", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, u := range urls {
		writer.WriteString(u + "\n")
	}
	writer.Flush()
}
