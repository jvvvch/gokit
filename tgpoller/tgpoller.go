package tgpoller

import (
	"context"
	"log"
	"sync"
	"time"
)

type updateIDGetter[T any] func(update T) int

type handler[T any] interface {
	GetUpdates(ctx context.Context, offset, timeout int) ([]T, error)
	HandleUpdate(ctx context.Context, update T)
}

type tgpoller[T any] struct {
	handler        handler[T]
	updates        chan T
	updateIDGetter updateIDGetter[T]
}

func New[T any](handler handler[T], updateIDGetter updateIDGetter[T]) *tgpoller[T] {
	return &tgpoller[T]{
		handler:        handler,
		updates:        make(chan T),
		updateIDGetter: updateIDGetter,
	}
}

type Options struct {
	SkipPending bool
	Timeout     time.Duration
}

func (p *tgpoller[T]) StartPolling(ctx context.Context, options Options) {
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go p.poll(ctx, wg, options)
	go p.listen(ctx, wg)
	wg.Wait()
}

func (p *tgpoller[T]) poll(ctx context.Context, wg *sync.WaitGroup, options Options) {
	defer func() {
		log.Println("poller: stopped polling updates")
		wg.Done()
	}()

	log.Println("poller: started polling updates")

	firstRun := true
	offset := 0
	timeout := int(options.Timeout.Seconds())

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		updates, err := p.handler.GetUpdates(ctx, offset, timeout)
		if err != nil {
			log.Printf("poller: error while getting updates: %v\n", err)
			continue
		}

		if l := len(updates); l > 0 {
			offset = p.updateIDGetter(updates[l-1]) + 1
		}

		if options.SkipPending && firstRun {
			firstRun = false
			continue
		}

		for _, u := range updates {
			select {
			case <-ctx.Done():
				return
			case p.updates <- u:
			}
		}
	}
}

func (p *tgpoller[T]) listen(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		log.Println("poller: stopped listening updates")
		wg.Done()
	}()

	log.Println("poller: started listening updates")

	for {
		select {
		case <-ctx.Done():
			return
		case u := <-p.updates:
			go p.handler.HandleUpdate(ctx, u)
		}
	}
}
