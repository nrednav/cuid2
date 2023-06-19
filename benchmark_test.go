package cuid2

import (
	"log"
	"testing"
)

var result string

func benchmarkGenerate(b *testing.B, length int) {
	var id string

	generate, err := Init(WithLength(length))
	if err != nil {
		log.Fatalln("Error: Could not initialise Cuid2 generator")
	}

	for n := 0; n < b.N; n++ {
		id = generate()
	}

	result = id
}

func BenchmarkGenerate8(b *testing.B)  { benchmarkGenerate(b, 8) }
func BenchmarkGenerate16(b *testing.B) { benchmarkGenerate(b, 16) }
func BenchmarkGenerate24(b *testing.B) { benchmarkGenerate(b, 24) }
func BenchmarkGenerate32(b *testing.B) { benchmarkGenerate(b, 32) }
