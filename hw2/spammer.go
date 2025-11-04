package main

import (
	"fmt"
	"sort"
	"sync"
)

func RunPipeline(cmds ...cmd) {
	if len(cmds) == 0 {
		return
	}

	firstIn := make(chan interface{})
	in := firstIn

	for _, c := range cmds {
		out := make(chan interface{})
		go func(in, out chan interface{}, c cmd) {
			c(in, out)
			close(out)
		}(in, out, c)
		in = out
	}

	close(firstIn)
	for range in {
	}
}

func SelectUsers(in, out chan interface{}) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	seen := make(map[uint64]struct{})

	for val := range in {
		email, ok := val.(string)
		if !ok {
			continue
		}

		wg.Add(1)
		go func(e string) {
			defer wg.Done()
			u := GetUser(e)
			mu.Lock()
			if _, exists := seen[u.ID]; !exists {
				seen[u.ID] = struct{}{}
				out <- u
			}
			mu.Unlock()
		}(email)
	}
	wg.Wait()
}

func SelectMessages(in, out chan interface{}) {
	buf := make([]User, 0, GetMessagesMaxUsersBatch)
	var wg sync.WaitGroup

	flush := func(batch []User) {
		if len(batch) == 0 {
			return
		}
		wg.Add(1)
		go func(us []User) {
			defer wg.Done()
			msgs, err := GetMessages(us...)
			if err != nil {
				return
			}
			for _, m := range msgs {
				out <- m
			}
		}(append([]User(nil), batch...))
	}

	for v := range in {
		u, ok := v.(User)
		if !ok {
			continue
		}
		buf = append(buf, u)
		if len(buf) == GetMessagesMaxUsersBatch {
			flush(buf)
			buf = buf[:0]
		}
	}
	flush(buf)
	wg.Wait()
}

func CheckSpam(in, out chan interface{}) {
	sem := NewSemaphore(HasSpamMaxAsyncRequests)
	var wg sync.WaitGroup

	for val := range in {
		id, ok := val.(MsgID)
		if !ok {
			continue
		}

		wg.Add(1)
		sem.Lock()

		go func(mid MsgID) {
			defer wg.Done()
			defer sem.Unlock()

			res, err := HasSpam(mid)
			if err != nil {
				panic(err.Error())
			}
			out <- MsgData{
				ID:      mid,
				HasSpam: res,
			}
		}(id)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	msgs := make([]MsgData, 0, 15)

	for val := range in {
		if md, ok := val.(MsgData); ok {
			msgs = append(msgs, md)
		}
	}

	sort.Slice(msgs, func(i, j int) bool {
		if msgs[i].HasSpam != msgs[j].HasSpam {
			return msgs[i].HasSpam
		}
		return msgs[i].ID < msgs[j].ID
	})

	for _, msg := range msgs {
		out <- fmt.Sprintf("%t %d", msg.HasSpam, msg.ID)
	}
}

type Semaphore struct {
	res chan struct{}
}

func NewSemaphore(cap int) *Semaphore {
	return &Semaphore{
		res: make(chan struct{}, cap),
	}
}

func (s *Semaphore) Lock() {
	s.res <- struct{}{}
}

func (s *Semaphore) Unlock() {
	<-s.res
}
