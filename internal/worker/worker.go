package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"math/rand"

	"github.com/emirpasic/gods/queues/priorityqueue"
	"github.com/emirpasic/gods/utils"
	"golang.org/x/time/rate"
)

type Job struct {
	Name         string
	Priority     int
	Id           int
	attemtNumber int
}

type JobWorker struct {
	goroutines_count, retries, timeout          int
	Queue                                       priorityqueue.Queue
	Mu                                          sync.Mutex
	StatusMap                                   map[int]string
	stopChan, notifyChan, pauseChan, resumeChan chan struct{}
	limiter                                     *rate.Limiter
}

func NewJobWorker(goroutines_count, retries, timeout int) *JobWorker {
	return &JobWorker{
		Queue: *priorityqueue.NewWith(func(a, b interface{}) int { // Компаратор
			priorityA := a.(Job).Priority
			priorityB := b.(Job).Priority
			return -utils.IntComparator(priorityA, priorityB) // "-" означает, что сортировка по убыванию
		}),
		goroutines_count: goroutines_count,
		retries:          retries,
		timeout:          timeout,
		Mu:               sync.Mutex{},
		StatusMap:        map[int]string{},
		notifyChan:       make(chan struct{}, 100),
		stopChan:         make(chan struct{}),
		pauseChan:        make(chan struct{}, goroutines_count),
		resumeChan:       make(chan struct{}, goroutines_count),
		limiter:          rate.NewLimiter(rate.Every(time.Minute/100), 100), // Всего 100 токенов, каждый токен пополняется раз в 0,6 секунд
	}
}

func (w *JobWorker) Start() {
	for i := 0; i < w.goroutines_count; i++ {
		go w.Work(i)
		log.Printf("Воркер %d запущен\n", i+1)
	}
}

func (w *JobWorker) Stop() {
	close(w.stopChan)

	// Если воркеры на паузе, то возобновим их
	for i := 0; i < w.goroutines_count; i++ {
		select {
		case <-w.pauseChan:
			w.resumeChan <- struct{}{}
		default:
		}
	}
}

func (w *JobWorker) Pause() {
	for i := 0; i < w.goroutines_count; i++ {
		w.pauseChan <- struct{}{}
	}
}

func (w *JobWorker) Resume() {
	for i := 0; i < w.goroutines_count; i++ {
		w.resumeChan <- struct{}{}
	}
}

func (w *JobWorker) Work(workerID int) {
	defer func() { // Обработка паники в воркере с возможностью перезапуска
		if r := recover(); r != nil {
			log.Printf("Воркер %d восстановлен после паники: %v", workerID, r)
			go w.Work(workerID)
		}
	}()

	for {
		// Select для приоритета канала остановки воркера
		select {
		case <-w.stopChan:
			log.Printf("Воркер %d остановлен\n", workerID)
			return
		default:
			select {
			case <-w.notifyChan: // Использование канала уведомлений важно, чтобы приоритетные задачи выполнясь раньше
				w.Mu.Lock()
				element, _ := w.Queue.Dequeue()
				job := element.(Job)

				w.StatusMap[job.Id] = "Выполняется"
				w.Mu.Unlock()
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(w.timeout)*time.Second)
				defer cancel()

				err := w.PerformJobWithTimeout(ctx, job)
				w.Mu.Lock()
				if err != nil {
					if job.attemtNumber < w.retries {
						job.attemtNumber++
						if err == context.DeadlineExceeded {
							w.StatusMap[job.Id] = "Истекло время ожидания задачи! Ожидает повторного выполнения в очереди"
							log.Printf("Истекло время ожидания %s! Попытка: %d. Ожидает повторного выполнения в очереди", job.Name, job.attemtNumber)
						} else {
							w.StatusMap[job.Id] = "Ошибка при выполнении задачи! Ожидает повторного выполнения в очереди"
							log.Printf("Ошибка при выполнении %s! Попытка: %d. Ожидает повторного выполнения в очереди", job.Name, job.attemtNumber)
						}
						w.Queue.Enqueue(job)
						w.notifyChan <- struct{}{}
					} else {
						w.StatusMap[job.Id] = "Задача не выполнена после максимального количества попыток"
						log.Printf("%s не выполнена после максимального количества попыток", job.Name)
					}
				} else {
					w.StatusMap[job.Id] = "Выполнена"
				}
				w.Mu.Unlock()
			case <-w.stopChan:
				log.Printf("Воркер %d остановлен\n", workerID)
				return
			case <-w.pauseChan:
				log.Printf("Воркер %d приостановлен", workerID)
				<-w.resumeChan
				log.Printf("Воркер %d возобновлен", workerID)
			}
		}
	}
}

func (w *JobWorker) PerformJobWithTimeout(ctx context.Context, job Job) error {
	done := make(chan error)
	go func() {
		done <- w.PerformJob(job)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *JobWorker) PerformJob(job Job) error {
	log.Printf("Работа %s начата в %s\n", job.Name, time.Now().Format("2006/01/02 15:04:05"))

	// Имитация выполнения работы; max = 3s
	<-time.After(time.Duration(rand.Int63n(int64(3 * time.Second))))

	// Имитация ошибки
	if rand.Int()%5 == 0 {
		return errors.New(fmt.Sprintf("Ошибка в выполнении %s", job.Name))
	}

	log.Printf("Выполнена работа %s в %s\n", job.Name, time.Now().Format("2006/01/02 15:04:05"))
	return nil
}

func (w *JobWorker) AddJob(name string, priority int) (id int, err error) {
	if !w.limiter.Allow() { // Проверка ограничения на 100 задач в минуту
		return 0, errors.New("Подано более 100 задач в минуту!")
	}

	id = func() int { // Генерация уникального значения id
		for {
			i := rand.Int()
			_, ok := w.StatusMap[i]
			if !ok {
				return i
			}
		}
	}()

	w.Mu.Lock()
	w.Queue.Enqueue(Job{
		Name:     name,
		Priority: priority,
		Id:       id,
	})
	w.Mu.Unlock()
	w.notifyChan <- struct{}{}

	w.StatusMap[id] = "Ожидает в очереди"

	return id, nil
}

func (w *JobWorker) AddOrUpdateJob(id int, name string, priority int) (status string, err error) {
	w.Mu.Lock()
	defer w.Mu.Unlock()

	newQueue := priorityqueue.NewWith(w.Queue.Comparator)
	flag := false

	// Можно полностью создать новую очередь, так как есть ограничение по ТЗ на количество задач в минуту
	for _, item := range w.Queue.Values() {
		job := item.(Job)
		if job.Id != id {
			newQueue.Enqueue(job)
		} else {
			flag = true
		}
	}

	newJob := Job{
		Name:     name,
		Priority: priority,
		Id:       id,
	}
	newQueue.Enqueue(newJob)

	w.Queue = *newQueue
	w.StatusMap[id] = "Ожидает в очереди"
	w.notifyChan <- struct{}{}

	if flag {
		status = "Параметры задачи обновлены"
	} else {
		status = "Задача добавлена"
	}

	return status, nil
}
