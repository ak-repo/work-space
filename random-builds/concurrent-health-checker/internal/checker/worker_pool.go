package checker

import (
	"context"
	"sync"
)

type indexedJob[I any] struct {
	index int
	value I
}

type indexedResult[O any] struct {
	index int
	value O
}

// RunPool processes jobs with a fixed number of workers and preserves input order.
func RunPool[I any, O any](ctx context.Context, workers int, inputs []I, fn func(context.Context, I) O) ([]O, error) {
	if workers < 1 {
		workers = 1
	}
	if workers > len(inputs) && len(inputs) > 0 {
		workers = len(inputs)
	}

	outputs := make([]O, len(inputs))
	jobs := make(chan indexedJob[I])
	results := make(chan indexedResult[O])

	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func(in <-chan indexedJob[I], out chan<- indexedResult[O]) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-in:
					if !ok {
						return
					}
					value := fn(ctx, job.value)
					select {
					case out <- indexedResult[O]{index: job.index, value: value}:
					case <-ctx.Done():
						return
					}
				}
			}
		}(jobs, results)
	}

	go func(out chan<- indexedJob[I]) {
		defer close(out)
		for i, input := range inputs {
			select {
			case out <- indexedJob[I]{index: i, value: input}:
			case <-ctx.Done():
				return
			}
		}
	}(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		outputs[result.index] = result.value
	}
	return outputs, ctx.Err()
}
