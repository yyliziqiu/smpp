package smpp

type ReceiptTo struct {
	MessageId string
	SystemId  string
	ExpiredAt int64
	hasTook   bool
}

type ReceiptHeap []*ReceiptTo

func (h *ReceiptHeap) Len() int {
	return len(*h)
}

func (h *ReceiptHeap) Less(i int, j int) bool {
	a := *h
	return a[i].ExpiredAt < a[j].ExpiredAt
}

func (h *ReceiptHeap) Swap(i int, j int) {
	a := *h
	a[i], a[j] = a[j], a[i]
}

func (h *ReceiptHeap) Push(v any) {
	*h = append(*h, v.(*ReceiptTo))
}

func (h *ReceiptHeap) Pop() any {
	a := *h
	n := len(a)
	v := a[n-1]
	a[n-1] = nil
	*h = a[0 : n-1]
	return v
}
