package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	TimeOutSeconds = 10 // the specified timeout
)

type CrawConfig struct {
	Max_depth        int      `json:"max_depth"`
	Min_sleep        int      `json:"min_sleep"`
	Max_sleep        int      `json:"max_sleep"`
	Timeout          bool     `json:"timeout"`
	Root_urls        []string `json:"root_urls"`
	Blacklisted_urls []string `json:"blacklisted_urls"`
	User_agents      []string `json:"user_agents"`
}

type Crawler struct {
	crawCfg    *CrawConfig
	links      map[string]bool // all links
	start_time time.Time
}

func (p *Crawler) loadConfig() {
	p.crawCfg = new(CrawConfig)
	p.links = make(map[string]bool)

	// open json file
	jsonfile, err := os.Open("config.json")

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("[%s][INFO] Successfully opened config.json.\n", getCurTime())

	defer jsonfile.Close()

	byteVal, err1 := ioutil.ReadAll(jsonfile)
	if err1 != nil {
		fmt.Printf("WARN: Cannot parse json file: %v\n", err1)
		return
	}

	json.Unmarshal([]byte(byteVal), p.crawCfg)
}

// return: response
func (p *Crawler) request(url1 string) string {
	res, err := http.Get(url1)

	if err != nil {
		return ""
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("WARN: Cannot parse content in: [%s]\n", url1)
		return ""
	}

	return string(body)
}

// return: response
func (p *Crawler) requestBody(url1 string) io.Reader {
	// https://stackoverflow.com/questions/16895294/how-to-set-timeout-for-http-get-requests-in-golang
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Get(url1)
	//res, err := http.Get(url1)
	if err != nil {
		fmt.Printf("[%s][DBG] Cannot get response from %s.\n",
			getCurTime(), url1)
		return nil
	}

	return res.Body
}

// making url absolute
func Normalize_link_old(url, rootUrl string) string {
	if strings.HasPrefix(url, "/") && !strings.HasPrefix(url, "//") {
		return fmt.Sprint(rootUrl, url)
	}
	return url
}

// if the URL contains some keywords configured in cfg file, then return ture
func (p *Crawler) Is_Blacklisted(url1 string) bool {
	for _, blurl := range p.crawCfg.Blacklisted_urls {
		if strings.Contains(url1, blurl) {
			//fmt.Printf("[%s][DBG] URL: [%s] contains blacklisted [%s].\n",
			//	getCurTime(), url1, blurl)
			return true
		}
	}

	return false
}

// filters url if it is blacklisted or not valid, we put filtering logic here
func (p *Crawler) Should_accept_url(url1 string) bool {
	if len(url1) == 0 {
		return false
	}

	if !IsValidUrl(url1) {
		//fmt.Printf("[%s][DBG] Invalid URL: %s.\n",
		//	getCurTime(), url1)
		return false
	}

	if p.Is_Blacklisted(url1) {
		//fmt.Printf("[%s][DBG] Blacklist URL: %s.\n",
		//	getCurTime(), url1)
		return false
	}
	return true
}

// gathers links to be visited in the future from a web page's body.
func (p *Crawler) Extract_Urls(body io.Reader, rooturl string) map[string]bool {
	urls := make(map[string]bool)

	if body == nil {
		return urls
	}

	urlarr := All(body)

	for _, aurl := range urlarr {
		absoluteUrl := normalize_link(aurl, rooturl)

		if p.Should_accept_url(absoluteUrl) {
			urls[absoluteUrl] = true
		}
	}

	return urls
}

/*
Removes a link from our current links list
and blacklists it so we don't visit it in the future
:param link: link to remove and blacklist
*/
func (p *Crawler) Remove_and_blacklist(url string) {
	p.crawCfg.Blacklisted_urls = append(p.crawCfg.Blacklisted_urls, url)

	p.removeVisitedLink(url)
}

/*
Selects a random link out of the available link list and visits it.
Blacklists any link that is not responsive or that contains no other links.
Please note that this function is recursive and will keep calling itself until
a dead end has reached or when we ran out of links
:param depth: our current link depth
*/
func (p *Crawler) Browse_from_links(depth int) {
	if len(p.links) == 0 {
		fmt.Printf("[%s]: Hit a dead end, moving to the next root URL.\n",
			getCurTime())
		return
	}

	is_dead_reached := depth > p.crawCfg.Max_depth
	if is_dead_reached {
		fmt.Printf("[%s]: Hit a dead end for depth, moving to the next root URL.\n",
			getCurTime())
		return
	}

	// timeout
	if p.is_Timeout_reached() {
		fmt.Printf("[%s]: Timeout has exceeded, exiting...\n", getCurTime())
		return
	}

	random_link := p.choiseRandomLink()

	fmt.Printf("[%s]: Visiting %s\n", getCurTime(), random_link)

	subpage := p.requestBody(random_link)

	sub_links := p.Extract_Urls(subpage, random_link)

	// sleep for a random amount of time
	p.SleepRandom()

	// make sure we have more than 1 link to pick from
	if len(sub_links) > 1 {
		// extract links from the new page
		p.links = p.Extract_Urls(subpage, random_link)
	} else {
		p.Remove_and_blacklist(random_link)
	}

	p.Browse_from_links(depth + 1)
}

// Determines whether the specified timeout has reached, if no timeout
// is specified then return false
// return: indicating whether the timeout has reached
func (p *Crawler) is_Timeout_reached() bool {
	if !p.crawCfg.Timeout {
		return false
	}

	curTime := time.Now()
	endTime := p.start_time.Add(TimeOutSeconds * time.Second)
	is_timed_out := curTime.After(endTime)
	return is_timed_out
}

//  Collects links from our root urls, stores them and then calls
//  `_browse_from_links` to browse them
func (p *Crawler) Crawl() {
	p.start_time = time.Now()

	for {
		i := CryptRandom(0, len(p.crawCfg.Root_urls))
		url := p.crawCfg.Root_urls[i]

		body := p.requestBody(url)
		p.links = p.Extract_Urls(body, url)
		fmt.Printf("[%s]: found %d links\n", getCurTime(), len(p.links))

		p.Browse_from_links(0)
	}
}

// www.dotnetperls.com/rand-go
func CryptRandom(minRange, maxRange int) int {
	delta := int64(maxRange - minRange)
	i, _ := rand.Int(rand.Reader, big.NewInt(delta))

	return int(i.Int64())
}

func (p *Crawler) SleepRandom() {
	if p.crawCfg.Max_sleep == 0 {
		return
	}
	i := CryptRandom(p.crawCfg.Min_sleep, p.crawCfg.Max_sleep)

	sleepduration := time.Duration(i) * time.Millisecond
	time.Sleep(sleepduration)
}

func (p *Crawler) choiseRandomLink() string {
	i := CryptRandom(0, len(p.links))

	// stackoverflow: get an arbitrary key item from a map
	for k := range p.links {
		if i == 0 {
			return k
		}
		i--
	}

	panic("never")
}

func (p *Crawler) removeVisitedLink(url string) {
	delete(p.links, url)
}

/*
Normalizes links extracted from the DOM by making them all absolute, so
we can request them, for example, turns a "/images" link extracted from https://imgur.com
to "https://imgur.com/images"
:param link: link found in the DOM
:param root_url: the URL the DOM was loaded from
:return: absolute link
*/
func normalize_link(link, root_url string) string {
	linknew := removeWhiteSpaceNewLine(link)
	u1, err := url.Parse(linknew)
	if err != nil {
		fmt.Printf("[%s][DBG] Cannot parse URL: [%s]!\n",
			getCurTime(), linknew)
		return linknew
	}

	base, err := url.Parse(root_url)
	if err != nil {
		fmt.Printf("[%s][DBG] Cannot parse Base URL: %s!\n",
			getCurTime(), root_url)
		return linknew
	}

	// https://stackoverflow.com/questions/34668012/combine-url-paths-with-path-join
	fullUrl := base.ResolveReference(u1).String()
	//fmt.Printf("[%s][DBG] Joint URL: [%s].\n", getCurTime(), fullUrl)

	return removeWhiteSpaceNewLine(fullUrl)
}

func removeWhiteSpace(s string) string {
	if strings.Contains(s, " ") {
		snew := strings.Replace(s, " ", "", -1)
		return snew
	} else {
		return s
	}
}

func removeWhiteSpaceNewLine(s string) string {
	snew := s
	if strings.Contains(snew, " ") {
		snew = strings.Replace(snew, " ", "", -1)
	}
	if strings.Contains(snew, "\t") {
		snew = strings.Replace(snew, "\t", "", -1)
	}
	if strings.Contains(snew, "\n") {
		snew = strings.Replace(snew, "\n", "", -1)
	}
	return snew
}

func IsValidUrl(str1 string) bool {
	return GetUrlValidator().regex.MatchString(str1)
}

type UrlValidator struct {
	regex *regexp.Regexp
}

func (p *UrlValidator) Init() {
	regexstr := `(https?|ftp|file)://[-A-Za-z0-9+&@#/%?=~_|!:,.;]+[-A-Za-z0-9+&@#/%=~_|]`
	p.regex = regexp.MustCompile(regexstr)
}

var (
	once     sync.Once
	instance *UrlValidator
)

func GetUrlValidator() *UrlValidator {
	once.Do(func() {
		instance = &UrlValidator{}
		instance.Init()
	})

	return instance
}

func getCurTime() string {
	//return time.Now().Format("2006-02-02 15:05:05")
	return time.Now().Format(time.RFC1123)
}
