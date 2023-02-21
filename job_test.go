package fiberextend_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jrallison/go-workers"
	ext "github.com/h-nosaka/fiberextend"
	"github.com/redis/go-redis/v9"
)

func TestJob(t *testing.T) {
	test := ext.NewTest(t, ext.IFiberExConfig{
		DevMode:      ext.Bool(true),
		UseRedis:     true,
		RedisOptions: &redis.Options{},
		JobDatabase:  0,
	})
	job1 := &ext.IJob{
		Name: "test",
		Proc: func(msg *workers.Msg) {
			test.Ex.Log.Info(fmt.Sprintf("job proc: %+v", msg))
			if err := test.Ex.Redis.Set(context.TODO(), "test_job_1", "fin", time.Duration(1*time.Hour)).Err(); err != nil {
				test.Ex.Log.Error(err.Error())
			}
		},
		Concurrency: 1,
		Class:       "test_class",
		Args:        map[string]interface{}{"foo": "bar"},
	}
	test.Ex.NewJob(job1)
	test.Ex.JobRun()
	test.Run("enqueue_job", func() {
		test.Job("test_job", func() {
			if err := test.Ex.Redis.Set(context.TODO(), "test_job_1", "start", time.Duration(1*time.Hour)).Err(); err != nil {
				t.Error(err)
			}
		}, func() {
			if err := test.Ex.JobEnqueue(job1.Name, job1.Class, job1.Args); err != nil {
				t.Error(err)
			}
		}, &ext.ITestCase{
			Method: ext.TestMethodEqual,
			Want:   "fin",
			Store: func() interface{} {
				time.Sleep(time.Second * 1) // 非同期処理のためsleepを入れる
				value, err := test.Redis.Get("test_job_1")
				if err != nil {
					t.Error(err)
				}
				return value
			},
		})
	})
}

func TestSchedule(t *testing.T) {
	test := ext.NewTest(t, ext.IFiberExConfig{
		DevMode:      ext.Bool(true),
		UseRedis:     true,
		RedisOptions: &redis.Options{},
		JobDatabase:  0,
	})
	job2 := &ext.IJob{
		Name: "test",
		Proc: func(msg *workers.Msg) {
			test.Ex.Log.Info(fmt.Sprintf("job proc: %+v", msg))
			if err := test.Ex.Redis.Set(context.TODO(), "test_job_2", "fin", time.Duration(1*time.Hour)).Err(); err != nil {
				test.Ex.Log.Error(err.Error())
			}
		},
		Concurrency: 1,
		Schedule:    ext.String("@every 1s"),
		Class:       "test_class",
		Args:        map[string]interface{}{"foo": "bar"},
	}
	test.Ex.NewJob(job2)
	test.Ex.JobRun()
	test.Run("scheduled_job", func() {
		test.Job("test_schedule", func() {
			if err := test.Ex.Redis.Set(context.TODO(), "test_job_2", "start", time.Duration(1*time.Hour)).Err(); err != nil {
				t.Error(err)
			}
		}, func() {
			// 手動での実行はしない
		}, &ext.ITestCase{
			Method: ext.TestMethodEqual,
			Want:   "fin",
			Store: func() interface{} {
				time.Sleep(time.Second * 2) // 定期実行のためsleepを入れる
				value, err := test.Redis.Get("test_job_2")
				if err != nil {
					t.Error(err)
				}
				return value
			},
		})
	})
}
