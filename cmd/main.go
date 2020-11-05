package main

import (
	"fmt"
	"github.com/frankegoesdown/easy_lru_cache"
)

func main() {
	c, _ := easy_lru_cache.New2Q(2, 0.0, 0.0)
	err := c.Put(2, 4)
	fmt.Println(err)
	fmt.Println(c.Get(2))
	//fmt.Println(c.Get(4))
}
