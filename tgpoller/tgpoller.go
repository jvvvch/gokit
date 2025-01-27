package tgpoller

import (
	"context"
	"log"
	"sync"

	"github.com/NicoNex/echotron/v3"
)

type handler interface {
	HandleUpdate(ctx context.Context, update *echotron.Update)
}

type tgpoller struct {
	handler handler
	api     *echotron.API
	updates chan *echotron.Update
}

func New(api *echotron.API, handler handler) *tgpoller {
	return &tgpoller{
		api:     api,
		handler: handler,
		updates: make(chan *echotron.Update),
	}
}

func (p *tgpoller) StartPolling(ctx context.Context, skipPending bool) {
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go p.poll(ctx, wg, skipPending)
	go p.listen(ctx, wg)
	wg.Wait()
}

func (p *tgpoller) poll(ctx context.Context, wg *sync.WaitGroup, skipPending bool) {
	defer func() {
		log.Println("poller: stopped polling updates")
		wg.Done()
	}()

	log.Println("poller: started polling updates")

	var (
		firstRun = true
		offset   = 0
		timeout  = 0
	)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		response, err := p.api.GetUpdates(&echotron.UpdateOptions{Offset: offset, Timeout: timeout})
		if err != nil {
			log.Printf("poller: error while getting updates: %v\n", err)
			continue
		}

		updates := response.Result

		if l := len(updates); l > 0 {
			offset = updates[l-1].ID + 1
		}

		if skipPending && firstRun {
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

func (p *tgpoller) listen(ctx context.Context, wg *sync.WaitGroup) {
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
