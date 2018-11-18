package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
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

	fmt.Println("INFO: Successfully opened config.json.")

	defer jsonfile.Close()

	byteVal, err1 := ioutil.ReadAll(jsonfile)
	if err1 != nil {
		fmt.Printf("WARN: Cannot parse json file: %v\n", err1)
		return
	}

	json.Unmarshal([]byte(byteVal), p.crawCfg)
}

// return: response
func (p *Crawler) request(url string) string {
	res, err := http.Get(url)

	if err != nil {
		return ""
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("WARN: Cannot parse: %s\n", url)
		return ""
	}

	return string(body)
}

// return: response
func (p *Crawler) requestBody(url string) io.Reader {
	res, err := http.Get(url)

	if err != nil {
		return nil
	}

	return res.Body
}

// making url absolute
func Normalize_link(url, rootUrl string) string {
	if strings.HasPrefix(url, "/") && !strings.HasPrefix(url, "//") {
		return fmt.Sprint(rootUrl, url)
	}
	return url
}

// if the URL contains some keywords configured in cfg file, then return ture
func (p *Crawler) Is_Blacklisted(url string) bool {
	for _, blurl := range p.crawCfg.Blacklisted_urls {
		if strings.Contains(url, blurl) {
			return true
		}
	}

	return false
}

// filters url if it is blacklisted or not valid, we put filtering logic here
func (p *Crawler) Should_accept_url(url string) bool {
	return (len(url) > 0) && IsValidUrl(url) && !p.Is_Blacklisted(url)
}

// gathers links to be visited in the future from a web page's body.
func (p *Crawler) Extract_Urls(body io.Reader, rooturl string) map[string]bool {
	urls := make(map[string]bool)

	if body == nil {
		return urls
	}

	urlarr := All(body)

	for _, url := range urlarr {
		urls[url] = true
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
		fmt.Println("Hit a dead end, moving to the next root URL")
		return
	}

	is_dead_reached := depth > p.crawCfg.Max_depth
	if is_dead_reached {
		fmt.Println("Hit a dead end for depth, moving to the next root URL")
		return
	}

	// TODO timeout

	random_link := p.choiseRandomLink()

	fmt.Printf("Visiting %s\n", random_link)

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

func (p *Crawler) is_Timeout_reached() bool {
	// TODO timeout reached

	return false
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
		fmt.Printf("found %d links", len(p.links))

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

	sleepduration := time.Duration(i) * time.Second
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
