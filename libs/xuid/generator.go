package xuid

import (
	"sync"
	"time"
)

type Generator struct {
	node string
	seed Seed
	mu   sync.Mutex
}

type Seed struct {
	A int64  // 当前时间戳
	B string // 当前时间戳16进制表示
	C int64  // 递增序号
}

func New(node int) *Generator {
	return &Generator{
		node: hex(int64(node), 2),
	}
}

// Get 返回十六位唯一 ID
func (t *Generator) Get() string {
	id, _ := t.GetOrFail()
	return id
}

// GetOrFail 返回十六位唯一 ID
func (t *Generator) GetOrFail() (string, error) {
	t.mu.Lock()

	nano := time.Now().UnixNano()
	curr := nano / 1e9
	if curr < t.seed.A {
		t.mu.Unlock()
		return "", ErrTimeBackForward
	}

	if curr > t.seed.A {
		t.seed.A = curr
		t.seed.B = hex(curr, 8)
		t.seed.C = (nano % 1e7) + 1048576 // 1048576 = 0x100000, 确保C转化为16进制后为6位数，且有一定的增长空间
	}
	t.seed.C++

	id := t.seed.B + t.node + hex(t.seed.C, 6)

	t.mu.Unlock()

	return id, nil
}
