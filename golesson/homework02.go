package main

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// addTen 通过指针把整数值增加 10。
func addTen(n *int) {
	if n == nil {
		return
	}
	*n += 10
}

// multiplySliceByTwo 通过切片指针把每个元素乘以 2。
func multiplySliceByTwo(nums *[]int) {
	if nums == nil {
		return
	}
	for i := range *nums {
		(*nums)[i] *= 2
	}
}

// printOdds 打印 1~10 的奇数。
func printOdds(wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 1; i <= 10; i += 2 {
		fmt.Println("odd:", i)
		time.Sleep(30 * time.Millisecond)
	}
}

// printEvens 打印 1~10 的偶数。
func printEvens(wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 2; i <= 10; i += 2 {
		fmt.Println("even:", i)
		time.Sleep(30 * time.Millisecond)
	}
}

type Task func()

type taskResult struct {
	name     string
	duration time.Duration
}

// Shape 定义形状接口。
type Shape interface {
	Area() float64
	Perimeter() float64
}

type Rectangle struct {
	Width  float64
	Height float64
}

func (r Rectangle) Area() float64 {
	return r.Width * r.Height
}

func (r Rectangle) Perimeter() float64 {
	return 2 * (r.Width + r.Height)
}

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

func (c Circle) Perimeter() float64 {
	return 2 * math.Pi * c.Radius
}

type Person struct {
	Name string
	Age  int
}

type Employee struct {
	Person
	EmployeeID string
}

func (e Employee) PrintInfo() {
	fmt.Printf("Employee{Name:%s, Age:%d, EmployeeID:%s}\n", e.Name, e.Age, e.EmployeeID)
}

// runTasksConcurrently 并发执行任务，并统计每个任务耗时。
func runTasksConcurrently(tasks map[string]Task) []taskResult {
	var wg sync.WaitGroup
	results := make(chan taskResult, len(tasks))

	for name, task := range tasks {
		wg.Add(1)
		go func(taskName string, fn Task) {
			defer wg.Done()
			start := time.Now()
			fn()
			results <- taskResult{
				name:     taskName,
				duration: time.Since(start),
			}
		}(name, task)
	}

	wg.Wait()
	close(results)

	out := make([]taskResult, 0, len(tasks))
	for r := range results {
		out = append(out, r)
	}
	return out
}

// produceOneToTen 生产 1~10 的整数并发送到通道。
func produceOneToTen(ch chan<- int) {
	defer close(ch)
	for i := 1; i <= 10; i++ {
		ch <- i
	}
}

// consumeInts 从通道中接收整数并打印。
func consumeInts(ch <-chan int, label string) {
	for n := range ch {
		fmt.Printf("%s receive: %d\n", label, n)
	}
}

// produceHundred 发送 1~100 到带缓冲通道中。
func produceHundred(ch chan<- int) {
	defer close(ch)
	for i := 1; i <= 100; i++ {
		ch <- i
	}
}

// countWithMutex 使用互斥锁保护共享计数器。
func countWithMutex(workers, times int) int {
	var mu sync.Mutex
	var wg sync.WaitGroup
	counter := 0

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < times; j++ {
				mu.Lock()
				counter++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return counter
}

// countWithAtomic 使用原子操作实现无锁计数器。
func countWithAtomic(workers, times int) int64 {
	var wg sync.WaitGroup
	var counter int64

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < times; j++ {
				atomic.AddInt64(&counter, 1)
			}
		}()
	}

	wg.Wait()
	return counter
}

func main() {
	// 题目1：整数指针 +10
	x := 5
	fmt.Println("before addTen:", x)
	addTen(&x)
	fmt.Println("after addTen: ", x)

	// 题目2：切片指针，每个元素 *2
	arr := []int{1, 2, 3, 4}
	fmt.Println("before multiplySliceByTwo:", arr)
	multiplySliceByTwo(&arr)
	fmt.Println("after multiplySliceByTwo: ", arr)

	// 题目3：用两个协程分别打印奇数和偶数
	var wg sync.WaitGroup
	wg.Add(2)
	go printOdds(&wg)
	go printEvens(&wg)
	wg.Wait()

	// 题目4：并发任务调度 + 统计每个任务耗时
	tasks := map[string]Task{
		"task-A": func() { time.Sleep(120 * time.Millisecond) },
		"task-B": func() { time.Sleep(80 * time.Millisecond) },
		"task-C": func() { time.Sleep(150 * time.Millisecond) },
	}
	results := runTasksConcurrently(tasks)
	for _, r := range results {
		fmt.Printf("%s finished in %v\n", r.name, r.duration)
	}

	// 题目5：面向对象 - Shape 接口与实现
	rect := Rectangle{Width: 3, Height: 4}
	circle := Circle{Radius: 2}
	shapes := []Shape{rect, circle}
	for _, s := range shapes {
		fmt.Printf("%T -> area=%.2f, perimeter=%.2f\n", s, s.Area(), s.Perimeter())
	}

	// 题目6：组合 Person + Employee，并输出信息
	emp := Employee{
		Person:     Person{Name: "Alice", Age: 28},
		EmployeeID: "E-1001",
	}
	emp.PrintInfo()

	// 题目7：通道实现两个协程通信（无缓冲）
	unbuffered := make(chan int)
	var channelWG sync.WaitGroup
	channelWG.Add(1)
	go func() {
		defer channelWG.Done()
		consumeInts(unbuffered, "unbuffered")
	}()
	produceOneToTen(unbuffered)
	channelWG.Wait()

	// 题目8：带缓冲通道，发送 100 个整数并消费
	buffered := make(chan int, 16)
	channelWG.Add(1)
	go func() {
		defer channelWG.Done()
		consumeInts(buffered, "buffered")
	}()
	produceHundred(buffered)
	channelWG.Wait()

	// 题目9：sync.Mutex 保护共享计数器
	mutexCount := countWithMutex(10, 1000)
	fmt.Printf("mutex counter = %d\n", mutexCount)

	// 题目10：sync/atomic 无锁计数器
	atomicCount := countWithAtomic(10, 1000)
	fmt.Printf("atomic counter = %d\n", atomicCount)
}
