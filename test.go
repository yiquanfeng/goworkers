package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────
// 模拟本地运行队列（固定容量环形数组，仿 runtime.p.runq）
// ─────────────────────────────────────────────
const queueSize = 8 // 为了演示用小一点，runtime 里是 256

type LocalQueue struct {
	head uint32
	tail uint32
	buf  [queueSize]int // 存 goroutine ID
}

// push：P 自己入队（只有自己调用，无竞争）
func (q *LocalQueue) push(gid int) bool {
	h := atomic.LoadUint32(&q.head)
	t := q.tail
	if t-h >= queueSize {
		return false // 满了
	}
	q.buf[t%queueSize] = gid
	atomic.StoreUint32(&q.tail, t+1)
	return true
}

// pop：P 自己出队（从尾部取，LIFO，只有自己调用）
func (q *LocalQueue) pop() (int, bool) {
	for {
		h := atomic.LoadUint32(&q.head)
		t := atomic.LoadUint32(&q.tail)
		if h == t {
			return 0, false // 空
		}
		// 从尾部取（最新的 G，局部性最好）
		t--
		atomic.StoreUint32(&q.tail, t)
		if atomic.LoadUint32(&q.head) <= t {
			return q.buf[t%queueSize], true
		}
		// 和 steal 产生竞争，回滚重试
		atomic.StoreUint32(&q.tail, t+1)
	}
}

// steal：其他 P 来偷，从头部偷走一半（CAS 保证原子性）
func (q *LocalQueue) steal(dst *LocalQueue) int {
	for {
		h := atomic.LoadUint32(&q.head)
		t := atomic.LoadUint32(&q.tail)
		n := (t - h) / 2 // 偷一半
		if n == 0 {
			return 0
		}
		// 把 [h, h+n) 搬到 dst
		for i := uint32(0); i < n; i++ {
			dst.push(q.buf[(h+i)%queueSize])
		}
		// CAS 推进 head：如果这期间 head 被别人动过，重试
		if atomic.CompareAndSwapUint32(&q.head, h, h+n) {
			return int(n)
		}
		// CAS 失败：有其他 P 同时在偷，清掉 dst 重来
		atomic.StoreUint32(&dst.tail, atomic.LoadUint32(&dst.head))
	}
}

func (q *LocalQueue) size() int {
	h := atomic.LoadUint32(&q.head)
	t := atomic.LoadUint32(&q.tail)
	return int(t - h)
}

func (q *LocalQueue) snapshot() []int {
	h := atomic.LoadUint32(&q.head)
	t := atomic.LoadUint32(&q.tail)
	out := make([]int, 0, t-h)
	for i := h; i < t; i++ {
		out = append(out, q.buf[i%queueSize])
	}
	return out
}

// ─────────────────────────────────────────────
// 模拟 P（逻辑处理器）
// ─────────────────────────────────────────────
type P struct {
	id       int
	queue    LocalQueue
	executed []int // 记录执行过哪些 G
	mu       sync.Mutex
}

func (p *P) recordExec(gid int) {
	p.mu.Lock()
	p.executed = append(p.executed, gid)
	p.mu.Unlock()
}

// ─────────────────────────────────────────────
// 演示 1：基本 work stealing
// ─────────────────────────────────────────────
func demo1Basic() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("演示 1：基本 work stealing")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	p0 := &P{id: 0}
	p1 := &P{id: 1}

	// P0 塞满 6 个 goroutine
	for i := 1; i <= 6; i++ {
		p0.queue.push(i * 100)
	}

	fmt.Printf("偷之前 → P0 队列: %v（%d个）\n", p0.queue.snapshot(), p0.queue.size())
	fmt.Printf("偷之前 → P1 队列: %v（%d个）\n", p1.queue.snapshot(), p1.queue.size())

	// P1 队列空了，来偷 P0 的
	stolen := p0.queue.steal(&p1.queue)

	fmt.Printf("\n偷之后 → P0 队列: %v（%d个）\n", p0.queue.snapshot(), p0.queue.size())
	fmt.Printf("偷之后 → P1 队列: %v（%d个，偷了 %d 个）\n",
		p1.queue.snapshot(), p1.queue.size(), stolen)
	fmt.Println()
}

// ─────────────────────────────────────────────
// 演示 2：不均衡任务分配时 stealing 的效果
// ─────────────────────────────────────────────
func demo2LoadBalance() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("演示 2：有/无 work stealing 的完成时间对比")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 8 个任务，故意只分给 P0，P1 初始为空
	tasks := make([]int, 8)
	for i := range tasks {
		tasks[i] = i + 1
	}
	taskDuration := 20 * time.Millisecond

	// ── 无 stealing：P0 串行干完所有活，P1 闲着 ──
	start := time.Now()
	var wg sync.WaitGroup
	done0, done1 := 0, 0
	for _, t := range tasks {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(taskDuration)
			done0++
		}(t)
	}
	_ = done1
	wg.Wait()
	noStealTime := time.Since(start)

	// ── 有 stealing：任务均匀分到两个 worker ──
	start = time.Now()
	ch := make(chan int, len(tasks))
	for _, t := range tasks {
		ch <- t
	}
	close(ch)
	var wg2 sync.WaitGroup
	for w := 0; w < 2; w++ {
		wg2.Add(1)
		go func(workerID int) {
			defer wg2.Done()
			for range ch {
				time.Sleep(taskDuration)
			}
		}(w)
	}
	wg2.Wait()
	stealTime := time.Since(start)

	fmt.Printf("无负载均衡（单 worker）: %v\n", noStealTime.Round(time.Millisecond))
	fmt.Printf("有负载均衡（2 workers）: %v\n", stealTime.Round(time.Millisecond))
	fmt.Printf("加速比: %.1fx\n\n", float64(noStealTime)/float64(stealTime))
}

// ─────────────────────────────────────────────
// 演示 3：模拟完整的 GMP work stealing 调度循环
// ─────────────────────────────────────────────
type Scheduler struct {
	ps      []*P
	globalQ []int
	gmu     sync.Mutex
	nextGID int32
}

func newScheduler(numP int) *Scheduler {
	s := &Scheduler{}
	for i := 0; i < numP; i++ {
		s.ps = append(s.ps, &P{id: i})
	}
	return s
}

func (s *Scheduler) spawn(pID int) int {
	gid := int(atomic.AddInt32(&s.nextGID, 1))
	p := s.ps[pID]
	if !p.queue.push(gid) {
		// 本地满了，进全局队列
		s.gmu.Lock()
		s.globalQ = append(s.globalQ, gid)
		s.gmu.Unlock()
	}
	return gid
}

func (s *Scheduler) run(pID int, steps int, log *[]string, mu *sync.Mutex) {
	p := s.ps[pID]
	tick := 0

	for tick < steps {
		tick++
		var gid int
		var got bool

		// 1. 本地队列
		gid, got = p.queue.pop()
		if got {
			p.recordExec(gid)
			mu.Lock()
			*log = append(*log, fmt.Sprintf("  P%d 执行 G%d（本地队列）", pID, gid))
			mu.Unlock()
			time.Sleep(time.Millisecond)
			continue
		}

		// 2. 全局队列（每 3 次 tick 检查一次，模拟运行时的每61次）
		if tick%3 == 0 {
			s.gmu.Lock()
			if len(s.globalQ) > 0 {
				gid = s.globalQ[0]
				s.globalQ = s.globalQ[1:]
				got = true
			}
			s.gmu.Unlock()
			if got {
				p.recordExec(gid)
				mu.Lock()
				*log = append(*log, fmt.Sprintf("  P%d 执行 G%d（全局队列）", pID, gid))
				mu.Unlock()
				time.Sleep(time.Millisecond)
				continue
			}
		}

		// 3. Work steal：随机选一个其他 P 偷
		victims := rand.Perm(len(s.ps))
		for _, vid := range victims {
			if vid == pID {
				continue
			}
			stolen := s.ps[vid].queue.steal(&p.queue)
			if stolen > 0 {
				mu.Lock()
				*log = append(*log, fmt.Sprintf("  P%d 从 P%d 偷了 %d 个 G，队列现在: %v",
					pID, vid, stolen, p.queue.snapshot()))
				mu.Unlock()
				break
			}
		}

		// 执行偷来的
		gid, got = p.queue.pop()
		if got {
			p.recordExec(gid)
			mu.Lock()
			*log = append(*log, fmt.Sprintf("  P%d 执行 G%d（偷来的）", pID, gid))
			mu.Unlock()
			time.Sleep(time.Millisecond)
		} else {
			time.Sleep(time.Millisecond) // 真的没活干，稍等
		}
	}
}

func demo3Simulation() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("演示 3：模拟 GMP work stealing 调度循环")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	s := newScheduler(3)

	// 故意把所有任务都塞给 P0
	fmt.Println("初始状态：把 7 个 G 全部塞给 P0")
	for i := 0; i < 7; i++ {
		s.spawn(0)
	}
	fmt.Printf("P0 队列: %v\nP1 队列: %v\nP2 队列: %v\n\n",
		s.ps[0].queue.snapshot(),
		s.ps[1].queue.snapshot(),
		s.ps[2].queue.snapshot())

	var logs []string
	var logMu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(pid int) {
			defer wg.Done()
			s.run(pid, 12, &logs, &logMu)
		}(i)
	}
	wg.Wait()

	for _, l := range logs {
		fmt.Println(l)
	}

	fmt.Println("\n各 P 执行情况：")
	for _, p := range s.ps {
		fmt.Printf("  P%d 执行了 %d 个 G: %v\n", p.id, len(p.executed), p.executed)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	demo1Basic()
	demo2LoadBalance()
	demo3Simulation()
}
