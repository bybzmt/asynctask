package main

import (
	"container/list"
)

type lruKv struct {
	key interface{}
	val interface{}
}

type Lru struct {
	all  map[interface{}]*list.Element
	list list.List
	Max  int
}

func (l *Lru) Init(max int) *Lru {
	l.all = make(map[interface{}]*list.Element, max)
	l.list.Init()
	l.Max = max
	return l
}

func (l *Lru) Get(key interface{}) (val interface{}, ok bool) {
	e, ok := l.all[key]
	if !ok {
		return nil, false
	}

	l.list.MoveToFront(e)

	return e.Value.(*lruKv).val, true
}

func (l *Lru) Add(key, val interface{}) {
	e, ok := l.all[key]
	if ok {
		e.Value.(*lruKv).val = val
		return
	}

	kv := &lruKv{
		key: key,
		val: val,
	}
	e = l.list.PushFront(kv)
	l.all[key] = e

	if l.list.Len() > l.Max {
		e = l.list.Back()
		l.list.Remove(e)

		delete(l.all, e.Value.(*lruKv).key)
	}
}

func (l *Lru) Each(fn func(k, v interface{})) {
	ele := l.list.Front()
	for ele != nil {
		kv, ok := ele.Value.(*lruKv)
		if !ok {
			panic("lru data err")
		}

		fn(kv.key, kv.val)

		ele = ele.Next()
	}
}

func (l *Lru) Len() int {
	return l.list.Len()
}
