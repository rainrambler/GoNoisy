package main

import (
	"testing"
)

func TestIsValidUrl1(t *testing.T) {
	str1 := "https://www.163.com"

	res := IsValidUrl(str1)
	wanted := true

	if res != wanted {
		t.Errorf("want: %v, but: %v", wanted, res)
	}
}

func TestIsValidUrl2(t *testing.T) {
	str1 := "https://blog.csdn.net/jeanphorn/article/details/78002490"

	res := IsValidUrl(str1)
	wanted := true

	if res != wanted {
		t.Errorf("want: %v, but: %v", wanted, res)
	}
}

func TestIsValidUrl3(t *testing.T) {
	str1 := "https://www.sogou.com/web?ie=UTF-8&query=url"

	res := IsValidUrl(str1)
	wanted := true

	if res != wanted {
		t.Errorf("want: %v, but: %v", wanted, res)
	}
}

func TestIs_Blacklisted1(t *testing.T) {
	str1 := "https://auto.ifeng.com/c/7uGgIeT1r3g"

	var craw Crawler
	craw.loadConfig()

	res := craw.Is_Blacklisted(str1)
	wanted := false

	if res != wanted {
		t.Errorf("want: %v, but: %v", wanted, res)
	}
}

func TestNormalize_link1(t *testing.T) {
	url1 := "http://ka.sina.com.cn/"
	baseUrl := "http://www.sina.com.cn"

	res := normalize_link(url1, baseUrl)
	wanted := url1

	if res != wanted {
		t.Errorf("want: %v, but: %v", wanted, res)
	}
}

func TestRemoveWhiteSpace1(t *testing.T) {
	url1 := ` http://vip.book.sina.com.cn/ weibobook/`
	res := removeWhiteSpace(url1)
	wanted := `http://vip.book.sina.com.cn/weibobook/`

	if res != wanted {
		t.Errorf("want: %v, but: %v", wanted, res)
	}
}
