package cpu

import "errors"

// Stack holds return addresses when the call operation is being
// completed. It can also be used for storing integers.
type Stack struct {
	entries []int
}

func NewStack() *Stack {
	return &Stack{}
}

func (s *Stack) Push(val int) {
	s.entries = append(s.entries, val)
}

func (s *Stack) Pop() (int, error) {
	if s.Empty() {
		return 0, errors.New("pop from an empty stack")
	}

	// get top
	length := len(s.entries)
	top := s.entries[length-1]

	// truncate
	s.entries = s.entries[:length-1]

	return top, nil
}

func (s *Stack) Size() int {
	return len(s.entries)
}

func (s *Stack) Empty() bool {
	return len(s.entries) == 0
}
