# Sync Scheduler

Sync Scheduler uses `gocron` and `go-redis` library to provide job scheduling and distributed locking under one library while abstracting out all the inner details.

## Code
### Creation
```
s7l := sync_scheduler.NewSyncScheduler(name, expiry, 
    sync_scheduler.RedisOptions{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
```
- `name` is the unique name of the job being performed.
- `expiry` can be used to set the expiry for the job (can be an int or time.Duration).
- `Addr` is the address of the Redis server.
- `Password` is the password of the Redis server (Optional).
- `DB` is the database number of the Redis server.

### Configuration
```
err := s7l.Every(1).Seconds().KeepLocked().Do(func, ...params)
```
- `Every(1)` creates a new periodic job with a given interval.
- `Seconds()` sets the interval to seconds (available options: `Minute()`, `Hour()`).
- `KeepLocked()` keeps the job locked until the job' expiry (even if the job is completed).
- `Do(func, ...params)` is the function reference with the parameters to be executed.
- Returns `error` if the job could not be scheduled. 

### In Action
```
s7l.StartAsync()
```
- `StartAsync()` starts the scheduler in a separate goroutine.

##### Go through tests in-case of any issue 

## Running locally
- Start the container with `docker compose up`
- Run the tests with `go test -v`