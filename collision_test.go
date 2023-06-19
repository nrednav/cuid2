package cuid2

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"strconv"
	"sync"
	"testing"
)

func TestCollisions(t *testing.T) {
	n := int64(math.Pow(float64(7), float64(8)) * 2)
	log.Printf("Testing %v unique Cuids...", n)

	numPools := int64(7)
	pools := createIdPools(numPools, n/numPools)

	ids := []string{}
	for _, pool := range pools {
		ids = append(ids, pool.ids...)
	}

	sampleIds := ids[:10]
	set := map[string]struct{}{}
	for _, id := range ids {
		set[id] = struct{}{}
	}

	histogram := pools[0].histogram

	log.Println("Sample Cuids:", sampleIds)
	log.Println("Histogram:", histogram)

	expectedBinSize := math.Ceil(float64(n / numPools / int64(len(histogram))))
	tolerance := 0.05
	minBinSize := math.Round(expectedBinSize * (1 - tolerance))
	maxBinSize := math.Round(expectedBinSize * (1 + tolerance))
	log.Println("Expected bin size:", expectedBinSize)
	log.Println("Min bin size:", minBinSize)
	log.Println("Maximum bin size:", maxBinSize)

	collisionsDetected := int64(len(set)) - n
	if collisionsDetected > 0 {
		t.Fatalf("%v collisions detected", int64(len(set))-n)
	}

	for _, binSize := range histogram {
		withinDistributionTolerance := binSize > minBinSize && binSize < maxBinSize
		if !withinDistributionTolerance {
			t.Errorf("Histogram of generated Cuids is not within the distribution tolerance of %v", tolerance)
			t.Fatalf("Expected bin size: %v, min: %v, max: %v, actual: %v", expectedBinSize, minBinSize, maxBinSize, binSize)
		}
	}

	validateCuids(t, ids)
}

type IdPool struct {
	ids       []string
	numbers   []big.Int
	histogram []float64
}

func NewIdPool(max int) func() *IdPool {
	return func() *IdPool {

		set := map[string]struct{}{}

		for i := 0; i < max; i++ {
			set[Generate()] = struct{}{}
			if i%100000 == 0 {
				progress := float64(i) / float64(max)
				log.Printf("%d%%", int64(progress*100))
			}
			if len(set) < i {
				log.Printf("Collision at: %v", i)
				break
			}
		}

		log.Println("No collisions detected")

		ids := []string{}
		numbers := []big.Int{}

		for element := range set {
			ids = append(ids, element)
			numbers = append(numbers, *idToBigInt(element[1:]))
		}

		return &IdPool{
			ids:       ids,
			numbers:   numbers,
			histogram: buildHistogram(numbers, 20),
		}
	}
}

func idToBigInt(id string) *big.Int {
	bigInt := new(big.Int)
	for _, char := range id {
		base36Rune, _ := strconv.ParseInt(string(char), 36, 64)
		bigInt.Add(big.NewInt(base36Rune), bigInt.Mul(bigInt, big.NewInt(36)))
	}
	return bigInt
}

func buildHistogram(numbers []big.Int, bucketCount int) []float64 {
	log.Println("Building histogram...")

	buckets := make([]float64, bucketCount)
	counter := 1

	numPermutations, _ := big.NewFloat(math.Pow(float64(36), float64(DefaultIdLength-1))).Int(nil)
	bucketLength := new(big.Int).Div(
		numPermutations,
		big.NewInt(int64(bucketCount)),
	)

	for _, number := range numbers {

		if new(big.Int).Mod(big.NewInt(int64(counter)), bucketLength).Int64() == 0 {
			log.Println(number)
		}

		bucket := new(big.Int).Div(
			&number,
			bucketLength,
		)

		if new(big.Int).Mod(big.NewInt(int64(counter)), bucketLength).Int64() == 0 {
			log.Println(bucket)
		}

		buckets[bucket.Int64()]++
		counter++
	}

	return buckets
}

func worker(id int, jobs <-chan func() *IdPool, results chan<- *IdPool) {
	for job := range jobs {
		log.Println("worker", id, "started job")
		results <- job()
	}
}

func createIdPools(numPools int64, maxIdsPerPool int64) []*IdPool {

	jobsList := []func() *IdPool{}
	for i := 0; i < int(numPools); i++ {
		jobsList = append(jobsList, NewIdPool(int(maxIdsPerPool)))
	}

	jobs := make(chan func() *IdPool, numPools)
	results := make(chan *IdPool, numPools)

	for w := 1; w <= int(numPools); w++ {
		go worker(w, jobs, results)
	}

	for _, job := range jobsList {
		jobs <- job
	}
	close(jobs)

	pools := []*IdPool{}
	for a := 1; a <= int(numPools); a++ {
		pool := <-results
		pools = append(pools, pool)
	}

	return pools
}

func validateCuids(t *testing.T, ids []string) {
	log.Printf("Validating all %v Cuids...", len(ids))

	wg := new(sync.WaitGroup)
	validationErrors := make(chan error)

	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if !IsCuid(id) {
				validationErrors <- fmt.Errorf("Cuid (%v) is not valid", id)
			}
		}(id)
	}

	go func() {
		wg.Wait()
		close(validationErrors)
	}()

	numInvalidIds := 0
	for err := range validationErrors {
		log.Println(err.Error())
		numInvalidIds++
	}

	if numInvalidIds > 0 {
		t.Fatalf("%v Cuids were invalid", numInvalidIds)
	}
}
