package golly

import "sync"

var (
	wg sync.WaitGroup
)

func AddWait(key string) {
	wg.Add(1)
}
