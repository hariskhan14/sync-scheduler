package sync_scheduler

import (
	"context"
	"github.com/go-co-op/gocron"
	"github.com/go-redis/redis/v7"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v7"
	"reflect"
	"time"
)

type SyncScheduler struct {
	name       string
	job        *gocron.Job
	lockExpiry time.Duration
	keepLocked bool

	scheduler *gocron.Scheduler
	rsync     *redsync.Redsync
}

type RedisOptions struct {
	Addr     string
	Password string
	DB       int
}

func NewSyncScheduler(name string, lockExpiry time.Duration, redisOpts RedisOptions) *SyncScheduler {
	return &SyncScheduler{
		name:       name,
		lockExpiry: lockExpiry,
		rsync:      makeRedisConnection(redisOpts),
		scheduler:  gocron.NewScheduler(time.UTC),
	}
}

func NewSyncSchedulerWithClient(name string, lockExpiry time.Duration, client *redis.Client) *SyncScheduler {
	redisPool := goredis.NewPool(client)

	return &SyncScheduler{
		name:       name,
		lockExpiry: lockExpiry,
		rsync:      redsync.New(redisPool),
		scheduler:  gocron.NewScheduler(time.UTC),
	}
}

func makeRedisConnection(redisOpts RedisOptions) *redsync.Redsync {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisOpts.Addr,
		Password: redisOpts.Password,
		DB:       redisOpts.DB,
	})
	redisPool := goredis.NewPool(redisClient)
	return redsync.New(redisPool)
}

func (s *SyncScheduler) StartAsync() {
	s.scheduler.StartAsync()
}

func (s *SyncScheduler) KeepLocked() *SyncScheduler {
	s.keepLocked = true
	return s
}

func (s *SyncScheduler) Every(interval int) *SyncScheduler {
	s.scheduler = s.scheduler.Every(interval)
	return s
}

func (s *SyncScheduler) Seconds() *SyncScheduler {
	s.scheduler = s.scheduler.Seconds()
	return s
}

func (s *SyncScheduler) Minutes() *SyncScheduler {
	s.scheduler = s.scheduler.Minutes()
	return s
}

func (s *SyncScheduler) Hours() *SyncScheduler {
	s.scheduler = s.scheduler.Hours()
	return s
}

func (s *SyncScheduler) Do(jobFun interface{}, params ...interface{}) error {
	jobParams := s.addJobParams(params)
	lockedJobFun := s.lockJobFunc(jobFun)

	job, err := s.scheduler.Do(lockedJobFun, jobParams)
	if err != nil {
		return err
	}
	s.job = job
	return nil
}

func (s *SyncScheduler) lockJobFunc(jobFun interface{}) func(params ...interface{}) error {
	return func(params ...interface{}) error {
		ctx := context.Background()
		name, expiry, keepLocked, p := s.separateJobAndFuncParams(params)

		m, err := s.lock(ctx, name, expiry)
		if err != nil {
			return err
		}

		callJobFuncWithParams(jobFun, p)

		if !keepLocked {
			defer s.unlock(ctx, m)
		}
		return nil
	}
}

func (s *SyncScheduler) lock(ctx context.Context, name string, expiry time.Duration) (*redsync.Mutex, error) {
	m := s.rsync.NewMutex(name, redsync.WithExpiry(expiry))

	if err := m.LockContext(ctx); err != nil {
		return nil, err
	}

	return m, nil
}

func (s *SyncScheduler) unlock(ctx context.Context, m *redsync.Mutex) error {
	_, err := m.UnlockContext(ctx)
	return err
}

func callJobFuncWithParams(jobFunc interface{}, params []interface{}) {
	f := reflect.ValueOf(jobFunc)
	if len(params) != f.Type().NumIn() {
		return
	}
	in := make([]reflect.Value, len(params))
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}
	f.Call(in)
}

func (s *SyncScheduler) addJobParams(params []interface{}) []interface{} {
	return insertItemsAt(params, 0, []interface{}{s.name, s.lockExpiry, s.keepLocked})
}

func (s *SyncScheduler) separateJobAndFuncParams(params []interface{}) (string, time.Duration, bool, []interface{}) {
	p := params[0].([]interface{})
	name, _ := p[0].(string)
	expiry, _ := p[1].(time.Duration)
	keepLocked, _ := p[2].(bool)

	return name, expiry, keepLocked, removeItemsAt(p, 0, 3)
}

func removeItemsAt(array []interface{}, index int, skip int) []interface{} {
	ret := make([]interface{}, 0)
	ret = append(ret, array[:index]...)
	return append(ret, array[index+skip:]...)
}

func insertItemsAt(array []interface{}, index int, value []interface{}) []interface{} {
	return append(array[:index], append(value, array[index:]...)...)
}
