package worker

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// TransformFunc defines how an input raw record is transformed into a domain entity.
type TransformFunc[In any, Out any] func(item In) (*Out, error)

// Pool processes input items concurrently using N workers and flushes batches of M items.
type Pool[In any, Out any] struct {
	numWorkers  int
	jobQueue    chan In
	batchQueue  chan []Out
	batchSize   int
	flushPeriod time.Duration
	transform   TransformFunc[In, Out]
}

// NewPool creates an initialized concurrent worker pool.
func NewPool[In any, Out any](
	numWorkers int,
	queueCap int,
	batchSize int,
	flushPeriod time.Duration,
	transform TransformFunc[In, Out],
) *Pool[In, Out] {
	return &Pool[In, Out]{
		numWorkers:  numWorkers,
		jobQueue:    make(chan In, queueCap),
		batchQueue:  make(chan []Out, (queueCap/batchSize)+16),
		batchSize:   batchSize,
		flushPeriod: flushPeriod,
		transform:   transform,
	}
}

// Start launches worker goroutines and the batch aggregator.
func (p *Pool[In, Out]) Start(ctx context.Context) <-chan []Out {
	intermediate := make(chan Out, p.batchSize*p.numWorkers)

	var wgWorkers sync.WaitGroup
	for i := 0; i < p.numWorkers; i++ {
		wgWorkers.Add(1)
		go func(workerID int) {
			defer wgWorkers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-p.jobQueue:
					if !ok {
						return
					}
					out, err := p.transform(item)
					if err != nil {
						log.Debug().Err(err).Int("worker", workerID).Msg("transform error")
						continue
					}
					if out != nil {
						select {
						case intermediate <- *out:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}(i)
	}

	// Close intermediate channel once all workers finish
	go func() {
		wgWorkers.Wait()
		close(intermediate)
	}()

	// Batch aggregator
	go func() {
		defer close(p.batchQueue)
		batch := make([]Out, 0, p.batchSize)
		ticker := time.NewTicker(p.flushPeriod)
		defer ticker.Stop()

		flush := func() {
			if len(batch) > 0 {
				cpy := make([]Out, len(batch))
				copy(cpy, batch)
				select {
				case p.batchQueue <- cpy:
				case <-ctx.Done():
					return
				}
				batch = batch[:0]
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				flush()
			case out, ok := <-intermediate:
				if !ok {
					flush()
					return
				}
				batch = append(batch, out)
				if len(batch) >= p.batchSize {
					flush()
				}
			}
		}
	}()

	return p.batchQueue
}

// Submit enqueues an item for processing. Blocks when buffer is full (backpressure).
func (p *Pool[In, Out]) Submit(item In) {
	p.jobQueue <- item
}

// Close closes the input queue to signal workers to drain.
func (p *Pool[In, Out]) Close() {
	close(p.jobQueue)
}
