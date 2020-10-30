package main

import (
	"fmt"
	"github.com/frankegoesdown/easy_lru_cache/internal/lru"
)

func main() {
	cache, _ := lru.NewLRU(2)
	cache.Put(10, 20)
	cache.Put(30, 40)
	cache.Put("50", 20)
	fmt.Println(cache.Get(10))
}
