package xcq

// EnableDebug 开启 debug 模式，该模式下会输出队列的操作日志
func (q *Queue) EnableDebug() *Queue {
	q.debug = true
	return q
}

// Get 获取指定下标的元素
func (q *Queue) Get(i int) (any, error) {
	return q.get(i)
}

// HeadItem 获取头元素
func (q *Queue) HeadItem() (any, error) {
	return q.get(q.head)
}

// TailItem 获取尾元素
func (q *Queue) TailItem() (any, error) {
	return q.get(q.tailprev())
}

// Status 获取队列状态
func (q *Queue) Status() string {
	return q.status()
}

// Empty 判断队列是否为空
func (q *Queue) Empty() bool {
	return q.empty()
}

// Cap 获取队列容量
func (q *Queue) Cap() int {
	return q.cap() - 1
}

// Len 获取队列长度
func (q *Queue) Len() int {
	return q.len()
}

// Push 从队列尾向队列中添加一个元素
func (q *Queue) Push(item any) {
	q.push(item)
}

// Pop 从队列头弹出一个元素
func (q *Queue) Pop() (any, bool) {
	return q.pop()
}

// Filter 元素符合条件返回 true，否则返回 false
type Filter func(item any) bool

// Pops 从队列头开始弹出所有符合条件的元素，直到遇到第一个不符合条件的元素停止
func (q *Queue) Pops(filter Filter) []any {
	ret := make([]any, 0)

	for q.head != q.tail {
		item := q.list[q.head]
		ok := filter(item)
		if !ok {
			break
		}
		ret = append(ret, item)
		q.list[q.head] = nil
		q.head = q.headnext()
	}

	return ret
}

// Pops2 从队列头开始弹出所有符合条件的元素，直到遇到第一个不符合条件的元素停止
func (q *Queue) Pops2(filter Filter) {
	for q.head != q.tail {
		item := q.list[q.head]
		ok := filter(item)
		if !ok {
			break
		}
		q.list[q.head] = nil
		q.head = q.headnext()
	}
}

// Remove 需要删除的元素返回 true，否则返回 false
type Remove func(item any) bool

// Slide  类似于滑动窗口，在队列尾添加一个元素，并从队列头开始直到第一个不需要删除的元素出现，该元素前面的元素全部删除
// 第一个返回值表示被删除的元素
// 第二个返回值表示被删除的元素个数
func (q *Queue) Slide(item any, rmf Remove) (rmd []any) {
	q.push(item)

	for !q.empty() && rmf(q.list[q.head]) {
		if rm, ok := q.pop(); ok {
			rmd = append(rmd, rm)
		}
	}

	if len(rmd) > 0 && q.debug {
		q.print("slide")
	}

	return rmd
}

// SlideN 类似于滑动窗口，在队列尾添加一个元素，如果添加完元素队列长度大于 n，则删除前面的元素，最后只保留队列后 n 个元素
// 第一个返回值表示被删除的元素
// 第二个返回值表示是窗口否发生了滑动
func (q *Queue) SlideN(item any, n int) (rmd []any) {
	q.push(item)

	for q.len() > n {
		if rm, ok := q.pop(); ok {
			rmd = append(rmd, rm)
		}
	}

	if len(rmd) > 0 && q.debug {
		q.print("slide")
	}

	return rmd
}

// Walk 遍历队列
// reverse false：从头到尾遍历，true：从尾到头遍历
func (q *Queue) Walk(f func(item any), reverse bool) {
	if reverse {
		for i := q.tailprev(); i != q.headprev(); i = q.prev(i) {
			f(q.list[i])
		}
	} else {
		for i := q.head; i != q.tail; i = q.next(i) {
			f(q.list[i])
		}
	}
}

// Find 遍历队列，返回第一个符合条件的元素
// reverse false：从头到尾遍历，true：从尾到头遍历
func (q *Queue) Find(filter Filter, reverse bool) (ret any, idx int) {
	if reverse {
		for i := q.tailprev(); i != q.headprev(); i = q.prev(i) {
			if item := q.list[i]; filter(item) {
				ret, idx = item, i
				break
			}
		}
	} else {
		for i := q.head; i != q.tail; i = q.next(i) {
			if item := q.list[i]; filter(item) {
				ret, idx = item, i
				break
			}
		}
	}
	return
}

// FindAll 遍历队列，返回全部符合条件的元素
func (q *Queue) FindAll(f Filter) []any {
	ret := make([]any, 0)
	for i := q.head; i != q.tail; i = q.next(i) {
		if item := q.list[i]; f(item) {
			ret = append(ret, item)
		}
	}
	return ret
}

// TerminalN 获取队列前/后 n 个 item
func (q *Queue) TerminalN(n int, reverse bool) []any {
	ret := make([]any, 0, n)

	if n > q.len() {
		n = q.len()
	}

	if reverse {
		for i, j := 0, q.tailprev(); i < n && j != q.headprev(); i, j = i+1, q.prev(j) {
			ret = append(ret, q.list[j])
		}
	} else {
		for i, j := 0, q.head; i < n && j != q.tail; i, j = i+1, q.next(j) {
			ret = append(ret, q.list[j])
		}
	}

	return ret
}

// Terminal 获取队列前/后多个符合条件的 item，遇到第一个不符合条件的 item 停止遍历
func (q *Queue) Terminal(filter Filter, reverse bool) []any {
	ret := make([]any, 0)

	if reverse {
		for i := q.tailprev(); i != q.headprev(); i = q.prev(i) {
			item := q.list[i]
			if !filter(item) {
				break
			}
			ret = append(ret, item)
		}
	} else {
		for i := q.head; i != q.tail; i = q.next(i) {
			item := q.list[i]
			if !filter(item) {
				break
			}
			ret = append(ret, item)
		}
	}

	return ret
}

// Window
// 返回结果包含 bgn item，不包含 end item
func (q *Queue) Window(bgn Filter, end Filter) []any {
	run := false
	ret := make([]any, 0)

	for i := q.head; i != q.tail; i = q.next(i) {
		item := q.list[i]
		if !run && bgn(item) {
			run = true
		}
		if run && end(item) {
			break
		}
		if run {
			ret = append(ret, item)
		}
	}

	return ret
}

// Reset 重置队列
func (q *Queue) Reset(data []any) {
	q.reset(data)
}
