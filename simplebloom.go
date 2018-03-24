package simplebloom

import (
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"log"
	"os"

	"encoding/gob"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/spaolacci/murmur3"
)

type BloomFilter interface {
	Put([]byte)
	PutString(string)

	Has([]byte) bool
	HasString(string) bool

	Close()
}

type FileBloomFilter struct {
	*MemoryBloomFilter
	target string
}

type MemoryBloomFilter struct {
	k  uint
	bs BitSets
}

type RedisBloomFilter struct {
	cli redis.Conn
	n   uint
	k   uint
}

func HashData(data []byte, seed uint) uint {
	sha_data := sha256.Sum256(data)
	data = sha_data[:]
	m := murmur3.New64WithSeed(uint32(seed))
	m.Write(data)
	return uint(m.Sum64())
}

// NewMemoryBloomFilter 创建一个内存的bloom filter
func NewMemoryBloomFilter(n uint, k uint) *MemoryBloomFilter {
	return &MemoryBloomFilter{
		k:  k,
		bs: NewBitSets(n),
	}
}

// Put 添加一条记录
func (filter *MemoryBloomFilter) Put(data []byte) {
	l := uint(len(filter.bs))
	for i := uint(0); i < filter.k; i++ {
		filter.bs.Set(HashData(data, i) % l)
	}
}

// Put 添加一条string记录
func (filter *MemoryBloomFilter) PutString(data string) {
	filter.Put([]byte(data))
}

// Has 推测记录是否已存在
func (filter *MemoryBloomFilter) Has(data []byte) bool {
	l := uint(len(filter.bs))

	for i := uint(0); i < filter.k; i++ {
		if !filter.bs.IsSet(HashData(data, i) % l) {
			return false
		}
	}

	return true
}

// Has 推测记录是否已存在
func (filter *MemoryBloomFilter) HasString(data string) bool {
	return filter.Has([]byte(data))
}

// Close 关闭bloom filter
func (filter *MemoryBloomFilter) Close() {
	filter.bs = nil
}

// NewFileBloomFilter 创建一个以文件为存储介质的bloom filter
// target 文件保存处
// 本质上就是增加了MemoryBloomFilter, 在创建时打开文件, 在Close时保存文件
func NewFileBloomFilter(target string, n uint, k uint) *FileBloomFilter {
	memory_filter := NewMemoryBloomFilter(n, k)
	filter := &FileBloomFilter{
		memory_filter, target,
	}
	filter.reStore()

	return filter
}

func (filter *FileBloomFilter) Close() {
	filter.store()
	filter.bs = nil
}

func (filter *FileBloomFilter) store() {
	f, err := os.Create(filter.target)
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "Open file"))
	}
	defer f.Close()

	gzip_writer := gzip.NewWriter(f)
	defer gzip_writer.Close()

	encoder := gob.NewEncoder(gzip_writer)
	err = encoder.Encode(filter.bs)
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "gzip"))
	}
}

func (filter *FileBloomFilter) reStore() {
	f, err := os.Open(filter.target)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Fatalf("%+v", errors.Wrap(err, "Open file"))
	}
	defer f.Close()

	gzip_reader, err := gzip.NewReader(f)
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "Ungzip"))
	}

	decoder := gob.NewDecoder(gzip_reader)
	err = decoder.Decode(&filter.bs)
	if err != nil {
		log.Fatalf("%+v", errors.Wrap(err, "gob decode"))
	}
}

func NewRedisBloomFilter(cli redis.Conn, n, k uint) *RedisBloomFilter {
	filter := &RedisBloomFilter{
		cli: cli,
		n:   n,
		k:   k,
	}
	length, _ := redis.Int64(cli.Do("LLEN", filter.redisKey()))
	if uint(length) != n {
		bs := make([]interface{}, n)
		push_args := []interface{}{filter.redisKey()}
		push_args = append(push_args, bs...)
		cli.Do("DEL", filter.redisKey())
		cli.Do("LPUSH", push_args...)
	}

	return filter
}

func (filter *RedisBloomFilter) Put(data []byte) {
	for i := uint(0); i < filter.k; i++ {
		_, err := filter.cli.Do("LSET", filter.redisKey(), HashData(data, i)%filter.n, "1")
		if err != nil {
			log.Fatalf("%+v", errors.Wrap(err, "LSET"))
		}
	}
}

func (filter *RedisBloomFilter) PutString(data string) {
	filter.Put([]byte(data))
}

func (filter *RedisBloomFilter) Has(data []byte) bool {
	for i := uint(0); i < filter.k; i++ {
		index := HashData(data, i) % filter.n
		value, err := redis.String(filter.cli.Do("LINDEX", filter.redisKey(), index))
		if err != nil {
			log.Fatalf("%+v", errors.Wrap(err, "LINDEX"))
		}
		if value != "1" {
			return false
		}
	}

	return true
}

func (filter *RedisBloomFilter) HasString(data string) bool {
	return filter.Has([]byte(data))
}

// Close 只将cli设置为nil, 关闭redis连接的操作放在调用处
func (filter *RedisBloomFilter) Close() {
	filter.cli = nil
}

// redisKey 根据filter的n和k来生成一个独立的redis key
func (filter *RedisBloomFilter) redisKey() string {
	return fmt.Sprintf("_bloomfilter:n%d:k%d", filter.n, filter.k)
}
