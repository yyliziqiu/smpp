package xcq

// Queue recycle queue
type Queue struct {
	step int
	list []any
	head int
	tail int
}

func New(n int) *Queue {
	return &Queue{
		step: n,
		list: make([]any, n+1),
	}
}

func (q *Queue) cap() int {
	return len(q.list)
}

func (q *Queue) len() int {
	if q.tail > q.head {
		return q.tail - q.head
	}
	if q.tail < q.head {
		return q.tail + q.cap() - q.head
	}
	return 0
}

func (q *Queue) prev(i int) int {
	return (i - 1 + q.cap()) % q.cap()
}

func (q *Queue) next(i int) int {
	return (i + 1) % q.cap()
}

func (q *Queue) headprev() int {
	return q.prev(q.head)
}

func (q *Queue) headnext() int {
	return q.next(q.head)
}

func (q *Queue) tailprev() int {
	return q.prev(q.tail)
}

func (q *Queue) tailnext() int {
	return q.next(q.tail)
}

func (q *Queue) push(item any) {
	if q.tailnext() == q.head {
		q.grow()
	}

	q.list[q.tail] = item
	q.tail = q.tailnext()
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
}

func (q *Queue) pop() (any, bool) {
	if q.empty() {
		return nil, false
	}

	item := q.list[q.head]
	q.list[q.head] = nil
	q.head = q.headnext()

	return item, true
}

func (q *Queue) empty() bool {
	return q.head == q.tail
}

func (q *Queue) Push(item any) {
	q.push(item)
}

func (q *Queue) Pop() (any, bool) {
	return q.pop()
}

type Check func(item any) bool

func (q *Queue) Pops(check Check) {
	for q.head != q.tail {
		item := q.list[q.head]
		ok := check(item)
		if !ok {
			break
		}
		q.list[q.head] = nil
		q.head = q.headnext()
	}
}
