//go:build integration

package cuid2

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	// The number of histogram buckets to use for distribution analysis.
	HistogramBuckets = 20

	// The tolerance for how much a histogram bin's size can deviate
	// from the expected average size.
	DistributionTolerance = 0.05
)

type workerResult struct {
	collisions int
	histogram  []int
}

func TestCollisions(t *testing.T) {
	// The original formula: 7^8 * 2 = 11,529,602
	totalIdsToGenerate := int64(math.Pow(7, 8) * 2)
	numWorkers := 7
	idsPerWorker := int(totalIdsToGenerate / int64(numWorkers))

	log.Printf("Testing %d unique Cuids across %d workers...", totalIdsToGenerate, numWorkers)

	resultsChan := make(chan workerResult, numWorkers)
	var wg sync.WaitGroup
	var totalGenerated atomic.Int64
	doneChan := make(chan struct{})

	// Start the progress reporter goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Second)

		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				progress := float64(totalGenerated.Load()) / float64(totalIdsToGenerate) * 100
				fmt.Printf("\rProgress: %.2f%%", progress)
			case <-doneChan:
				fmt.Printf("\rProgress: 100.00%%\n")
				return
			}
		}
	}()

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go runCollisionWorker(&wg, resultsChan, idsPerWorker, &totalGenerated)
	}

	// Wait for all workers to finish
	wg.Wait()
	close(resultsChan)
	close(doneChan)

	// Aggregate results
	totalCollisions := 0
	finalHistogram := make([]int, HistogramBuckets)

	for result := range resultsChan {
		totalCollisions += result.collisions

		for i, count := range result.histogram {
			finalHistogram[i] += count
		}
	}

	log.Println("All workers finished.")
	log.Printf("Final Histogram: %v", finalHistogram)

	// --- Validation ---

	if totalCollisions > 0 {
		t.Fatalf("%d collisions detected", totalCollisions)
	}

	log.Println("No collisions detected.")

	// Validate histogram distribution
	expectedBinSize := float64(totalIdsToGenerate) / float64(HistogramBuckets)
	minBinSize := math.Floor(expectedBinSize * (1 - DistributionTolerance))
	maxBinSize := math.Ceil(expectedBinSize * (1 + DistributionTolerance))

	log.Printf("Expected bin size: ~%.2f (Min: %.2f, Max: %.2f)", expectedBinSize, minBinSize, maxBinSize)

	for i, binSize := range finalHistogram {
		if float64(binSize) < minBinSize || float64(binSize) > maxBinSize {
			t.Errorf("Histogram distribution is outside the %.2f%% tolerance.", DistributionTolerance*100)
			t.Fatalf("Bin %d size %d is outside the expected range [%.2f, %.2f]", i, binSize, minBinSize, maxBinSize)
		}
	}

	log.Println("Histogram distribution is within tolerance.")
}

// runCollisionWorker generates IDs, checks for collisions, and builds a histogram in a single pass.
func runCollisionWorker(wg *sync.WaitGroup, results chan<- workerResult, numIds int, totalCounter *atomic.Int64) {
	defer wg.Done()

	idSet := make(map[string]struct{}, numIds)
	result := workerResult{
		histogram: make([]int, HistogramBuckets),
	}

	// Pre-calculate histogram constants
	numPermutations, _ := new(big.Float).SetInt(
		new(big.Int).Exp(big.NewInt(Base36), big.NewInt(int64(DefaultIdLength-1)), nil),
	).Int(nil)
	bucketLength := new(big.Int).Div(numPermutations, big.NewInt(HistogramBuckets))

	for i := 0; i < numIds; i++ {
		id := Generate()
		totalCounter.Add(1)

		// 1. Check for collisions
		if _, exists := idSet[id]; exists {
			result.collisions++
		}

		idSet[id] = struct{}{}

		// 2. Calculate histogram bucket
		bigIntVal, ok := new(big.Int).SetString(id[1:], Base36)

		if !ok {
			continue
		}

		bucket := new(big.Int).Div(bigIntVal, bucketLength)

		result.histogram[bucket.Int64()]++
	}

	results <- result
}
