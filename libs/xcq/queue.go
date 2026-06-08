package xcq

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyQueue      = errors.New("empty queue")
	ErrIndexOutOfRange = errors.New("index out of range")
)

type Queue struct {
	step  int
	path  string
	list  []any
	head  int
	tail  int
	debug bool
}

func New(n int) *Queue {
	return New2(n, "")
}

func New2(n int, path string) *Queue {
	return &Queue{
		step: n,
		path: path,
		list: make([]any, n+1),
	}
}

// 获取队列容量
func (q *Queue) cap() int {
	return len(q.list)
}

// 获取队列长度
func (q *Queue) len() int {
	if q.tail > q.head {
		return q.tail - q.head
	}
	if q.tail < q.head {
		return q.tail + q.cap() - q.head
	}
	return 0
}

// 获取指定下标的前一个下标
func (q *Queue) prev(i int) int {
	return (i - 1 + q.cap()) % q.cap()
}

// 获取指定下标的后一个下标
func (q *Queue) next(i int) int {
	return (i + 1) % q.cap()
}

// 获取头下标的前一个下标
func (q *Queue) headprev() int {
	return q.prev(q.head)
}

// 获取头下标的后一个下标
func (q *Queue) headnext() int {
	return q.next(q.head)
}

// 获取尾下标的前一个下标
func (q *Queue) tailprev() int {
	return q.prev(q.tail)
}

// 获取尾下标的后一个下标
func (q *Queue) tailnext() int {
	return q.next(q.tail)
}

// 从队列尾向队列中添加一个元素
func (q *Queue) push(item any) {
	// 若队列已满，则扩容
	if q.tailnext() == q.head {
		q.grow()
	}

	// 添加元素
	q.list[q.tail] = item
	q.tail = q.tailnext()

	// 打印 debug 信息
	if q.debug {
		q.print(fmt.Sprintf("push %+v", item))
	}
}

func (q *Queue) print(tag string) {
	if q.debug {
		fmt.Printf("[%s] %s\n", q.status(), tag)
	}
}

func (q *Queue) status() string {
	return fmt.Sprintf("%2d - %-2d, %2d / %-2d", q.head, q.tail, q.len(), q.cap())
}

func (q *Queue) grow() {
	dst := make([]any, q.cap()+q.step)

	i, j := q.head, 0
	for ; i != q.tail; i, j = q.next(i), j+1 {
		dst[j] = q.list[i]
	}

	q.list = dst
	q.head = 0
	q.tail = j

	if q.debug {
		q.print("grow")
	}
}

// 从队列头弹出一个元素
func (q *Queue) pop() (any, bool) {
	// 判断是否为空
	if q.empty() {
		return nil, false
	}

	// 弹出元素
	item := q.list[q.head]
	q.list[q.head] = nil
	q.head = q.headnext()

	// 打印 debug 信息
	if q.debug {
		q.print(fmt.Sprintf("pop  %+v", item))
	}

	return item, true
}

func (q *Queue) empty() bool {
	return q.head == q.tail
}

// 获取指定下标元素
func (q *Queue) get(i int) (any, error) {
	if q.empty() {
		return nil, ErrEmptyQueue
	}
	if !q.valid(i) {
		return nil, ErrIndexOutOfRange
	}
	return q.list[i], nil
}

func (q *Queue) valid(i int) bool {
	if q.head < q.tail {
		return i >= q.head && i < q.tail
	}
	if q.head > q.tail {
		return i >= q.head || i < q.tail
	}
	return false
}

// 重置队列
func (q *Queue) reset(data []any) {
	initCap := (len(data)/q.step+1)*q.step + 1

	list := make([]any, initCap)
	for i, item := range data {
		list[i] = item
	}

	q.list = list
	q.head = 0
	q.tail = len(data)
}

// 复制列表
func (q *Queue) copyList() []any {
	cpy := make([]any, 0, q.len())
	for i := q.head; i != q.tail; i = q.next(i) {
		cpy = append(cpy, q.list[i])
	}
	return cpy
}
