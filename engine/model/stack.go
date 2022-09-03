package model

import "fmt"

type Stack struct {
	top  *Element
	size int
}

type Element struct {
	value interface{} // All types satisfy the empty interface, so we can store anything here.
	next  *Element
}

func (s *Stack) Len() int {
	return s.size
}

func (s *Stack) Push(value interface{}) {
	s.top = &Element{value, s.top}
	s.size++
}

func (s *Stack) Pop() (value interface{}) {
	if s.size > 0 {
		value, s.top = s.top.value, s.top.next
		s.size--
		return
	}
	return nil
}

func (s *Stack) Print() string {
	v := s.top
	var buffer []byte
	for v != nil {
		buffer = append(buffer, []byte(fmt.Sprintf("%v ", v.value))...)
		v = v.next
	}
	return string(buffer)
}