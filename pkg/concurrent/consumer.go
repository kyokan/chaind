package concurrent

import (
	"sync"
)

type ConsumerFunc func(interface{})

func Consume(items []interface{}, fn ConsumerFunc, concurrency int) {
	if len(items) == 0 {
		return
	}

	if len(items) < concurrency {
		concurrency = len(items)
	}

	var wg sync.WaitGroup
	wg.Add(concurrency)
	itemCh := make(chan interface{})
	for i := 0; i < concurrency; i++ {
		go func() {
			for {
				item, more := <-itemCh
				if !more {
					wg.Done()
					return
				}

				fn(item)
			}
		}()
	}

	go func() {
		for len(items) > 0 {
			last := len(items) - 1
			num := items[last]
			items = items[:last]
			itemCh <- num
		}

		close(itemCh)
	}()

	wg.Wait()
}

func ConsumeUint64s(items []uint64, fn func(uint64), concurrency int) {
	buf := make([]interface{}, len(items), len(items))
	for i := range items {
		buf[i] = items[i]
	}

	Consume(buf, func(i interface{}) {
		fn(i.(uint64))
	}, concurrency)
}

func ConsumeStrings(items []string, fn func(string), concurrency int) {
	buf := make([]interface{}, len(items), len(items))
	for i := range items {
		buf[i] = items[i]
	}

	Consume(buf, func(i interface{}) {
		fn(i.(string))
	}, concurrency)
}