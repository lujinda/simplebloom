package simplebloom

import (
	"fmt"
	"testing"

	"os"

	"github.com/gomodule/redigo/redis"
)

func RandTest(t *testing.T, filter BloomFilter, n int) {
	for i := 0; i < n; i++ {
		filter.PutString(fmt.Sprintf("r%d", i))
	}

	var miss_numbers int

	for i := 0; i < n; i++ {
		exists_record := fmt.Sprintf("r%d", i)
		not_exists_record := fmt.Sprintf("rr%d", i)
		if !filter.HasString(exists_record) {
			miss_numbers++
		}

		if filter.HasString(not_exists_record) {
			miss_numbers++
		}
	}
	hit_rate := float64(n-miss_numbers) / float64(n)
	fmt.Printf("hit rate: %f\n", hit_rate)

	if hit_rate < 0.9 {
		t.Fatalf("Oh, fuck. hit rate is %f, too low", hit_rate)
	}
}

func TestMemoryBloomFilter(t *testing.T) {
	var filter BloomFilter = NewMemoryBloomFilter(64<<20, 5)
	RandTest(t, filter, 50000)

}

func TestFileBloomFilter(t *testing.T) {
	target := "bloom.tmp"
	defer os.Remove(target)
	var filter BloomFilter = NewFileBloomFilter(target, 64<<20, 5)
	RandTest(t, filter, 50000)
}

func TestRedisBloomFilter(t *testing.T) {
	cli, err := redis.DialURL("redis://10.1.10.4")
	if err != nil {
		t.Fatal(err)
	}
	var filter BloomFilter = NewRedisBloomFilter(cli, 2000, 5)
	RandTest(t, filter, 50)
}
