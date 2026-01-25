package utils

type Pair[U, V any] struct {
    First U
    Second V
}

type Stack[T any] []T

func (s *Stack[T]) Push(v T) Stack[T] {
    *s = append(*s, v)
    return *s
}

func (s *Stack[T]) Pop() (res T, ok bool) {
	l := len(*s)
    ok = l > 0
    if !ok { return }
    res = (*s)[l-1]
    *s  = (*s)[:l-1]
    return
}

type Queue[T any] []T
func (q *Queue[T]) Add(v T) Queue[T] {
    *q = append(*q, v)
    return *q
}
func (q *Queue[T]) Remove() (res T, ok bool) {
    ok = len(*q) > 0
    if !ok { return }
    res = (*q)[0]
    *q  = (*q)[1:]
    return
}
