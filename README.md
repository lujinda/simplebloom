一个实现简单的bloom filter, 可以使用内存， 文件， redis三种工作方式 `MemoryBloomFilter`, `FileBloomFilter`, `RedisBloomFilter`



__test__

```
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

	for i := 0; i < n; i++ {
		exists_record := fmt.Sprintf("r%d", i)
		not_exists_record := fmt.Sprintf("rr%d", i)
		if !filter.HasString(exists_record) {
			t.Fatalf("%s 应该存在", exists_record)
		}

		if filter.HasString(not_exists_record) {
			t.Fatalf("%s 应该不存在", exists_record)
		}
	}
}

func TestMemoryBloomFilter(t *testing.T) {
	var filter BloomFilter = NewMemoryBloomFilter(64<<20, 5)
	RandTest(t, filter, 50000)

}

func TestFileBloomFilter(t *testing.T) {
	target := "bloom.tmp"
	defer os.Remove(target)
	var filter BloomFilter = NewFileBloomFilter(target, 2000, 5)
	filter.PutString("aaaa")
	filter.PutString("bbbb")
	filter.PutString("cccc")
	filter.Close()

	filter = NewFileBloomFilter(target, 2000, 5)
	if !filter.HasString("aaaa") {
		t.Fatal("aaaa 应该存在")
	}
	if filter.HasString("dddd") {
		t.Fatal("ddd 应该不存在")
	}
	RandTest(t, filter, 50)
}

func TestRedisBloomFilter(t *testing.T) {
	cli, err := redis.DialURL("redis://10.1.10.4")
	if err != nil {
		t.Fatal(err)
	}
	var filter BloomFilter = NewRedisBloomFilter(cli, 2000, 5)
	RandTest(t, filter, 50)
}

```


