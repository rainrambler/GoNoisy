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
