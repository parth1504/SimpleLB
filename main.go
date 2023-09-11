package main

import (
	"container/heap"
	"fmt"
	"math/rand"
	"time"
)

const nRequester = 100
const nWorker = 10

func op() int{
	n:=rand.Int63n(int64(time.Second))
	time.Sleep(time.Duration(nWorker*n))
	return int(n)
}

type Request struct{
	fn func() int
	c chan int
}

func requester(work chan Request){
	c:=make(chan int)
	for{
		time.Sleep(time.Duration(rand.Int63n(int64(nWorker * 2 * time.Second))))
		work <- Request{op,c}
		<-c
	}
}
type Worker struct{
	i int
	requests chan Request
	pending int
}

func (w *Worker) work(done chan *Worker){
	for{
		req:= <-w.requests
		req.c <-req.fn()
		done <-w
	}
}

type Pool []*Worker

func ( p Pool) Len() int { return len(p)}

func (p Pool) Less(i,j int) bool{
	return p[i].pending<p[j].pending
}

func (p *Pool) Swap(i, j int ){
	a:= *p
	a[i],a[j]= a[j],a[i]
	a[i].i=i
	a[j].i=j
}

func (p *Pool) Push(x interface{}){
	a:=*p
	n:=len(a)
    a=a[0: n+1]
	w:= x.(*Worker)
	a[n]=w
	w.i=n
	*p=a
}

func (p *Pool) Pop() interface{}{
	a:= *p
	*p= a[0:len(a)-1]
	w:=a[len(a)-1]
	w.i=-1
	return w
}

type Balancer struct{
	pool Pool
	done chan *Worker
	i int
}

func NewBalancer() *Balancer {
	done:= make(chan *Worker, nWorker)
	b:= &Balancer{make(Pool, 0, nWorker),done,0}
	for i:=0;i<nWorker;i++{
		w:= &Worker{requests: make(chan Request,nRequester)}
        heap.Push(&b.pool,w)
		go w.work(done)
	}
	return b
}

func (b *Balancer) balance(work chan Request){
	for{
		select{

			case req := <-work:
				b.dispatch(req)
			case w:= <- b.done:
				b.completed(w)
		}
		b.print()
	}
}

func (b *Balancer) print(){
	sum:=0
	sumsq:=0
	for _,w:= range b.pool{
		fmt.Printf("%d ", w.pending)
		sum+=w.pending
		sumsq+=w.pending*w.pending
	}
	average:=float64(sum)/float64(len(b.pool))
	variance:= float64(sumsq)/float64(len(b.pool)) - average*average
	fmt.Printf(" %.2f %.2f\n",average,variance)
}

func (b *Balancer) dispatch(req Request){
	w:=heap.Pop(&b.pool).(*Worker)
	w.requests <-req
	w.pending++
	heap.Push(&b.pool,w)
}

func (b *Balancer) completed( w *Worker){
	w.pending--;
	heap.Remove(&b.pool,w.i)
	heap.Push(&b.pool,w)
}

func main(){
	work:=make(chan Request)
	for i:=0;i<nRequester;i++{
		go requester(work)
	}
	NewBalancer().balance(work)
}