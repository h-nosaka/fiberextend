package fiberextend

import (
	"context"
	"fmt"
	"time"

	"github.com/bamzi/jobrunner"
	"github.com/jrallison/go-workers"
	"go.uber.org/zap"
)

type IJob struct {
	Name        string                 // ジョブ名
	Proc        func(msg *workers.Msg) // 処理内容
	Concurrency int                    // 優先度
	Schedule    *string                // cron形式 https://github.com/bamzi/jobrunner
	Class       string                 // スケジュール実行時のクラス名
	Args        func() interface{}     // スケジュール実行時のパラメータ
	Middlewares []workers.Action       // ジョブ特有のアクション
}

type jobInfo struct{}

func (p jobInfo) Call(queue string, msg *workers.Msg, next func() bool) bool {
	// 初期化
	Log.Info(fmt.Sprintf("job start: %s", queue), zap.Any("msg", msg))
	// 処理
	ok := next()
	// 終了処理
	Log.Info(fmt.Sprintf("job finish: %s", queue), zap.Any("msg", msg))
	return ok
}

func (p IJob) Run() {
	if Ex.checkCronNode() { // cronはシングルノードで動作するようにチェックする
		Log.Info("scheduled job start", zap.Any("job", p))
		if _, err := workers.Enqueue(p.Name, p.Class, p.Args()); err != nil {
			Log.Error(err.Error(), zap.Any("job", p))
		}
	}
}

const cronActiveNodeKey = "active_node:cron"

func (p *IFiberEx) NewJob(jobs ...*IJob) {
	workers.Configure(map[string]string{
		"server":   p.Config.RedisOptions.Addr,
		"database": fmt.Sprintf("%d", p.Config.JobDatabase),
		"pool":     fmt.Sprintf("%d", p.Config.JobPool),
		"process":  fmt.Sprintf("%d", p.Config.JobProcess),
	})
	workers.Middleware.Append(&jobInfo{})

	// cron実行のためのnode登録
	if err := Redis.Set(context.Background(), cronActiveNodeKey, p.NodeId, time.Duration(0)).Err(); err != nil {
		p.Log.Error(err.Error())
	}

	jobrunner.Start()
	for _, job := range jobs {
		workers.Process(job.Name, job.Proc, job.Concurrency, job.Middlewares...)
		if job.Schedule != nil {
			if err := jobrunner.Schedule(*job.Schedule, *job); err != nil {
				p.Log.Error(err.Error(), zap.Any("job", *job))
			}
		}
	}
}

func (p *IFiberEx) JobRun(jobs ...IJob) {
	go workers.Run()
}

func (p *IFiberEx) JobEnqueue(queue string, class string, args interface{}) error {
	if _, err := workers.Enqueue(queue, class, args); err != nil {
		p.Log.Error(err.Error(), zap.String("name", queue), zap.String("class", class), zap.Any("args", args))
		return err
	}
	return nil
}

func (p *IFiberEx) JobEnqueueIn(queue string, class string, in float64, args interface{}) error {
	if _, err := workers.EnqueueIn(queue, class, in, args); err != nil {
		p.Log.Error(err.Error(), zap.String("name", queue), zap.String("class", class), zap.Float64("in", in), zap.Any("args", args))
		return err
	}
	return nil
}

func (p *IFiberEx) JobEnqueueAt(queue string, class string, at time.Time, args interface{}) error {
	if _, err := workers.EnqueueAt(queue, class, at, args); err != nil {
		p.Log.Error(err.Error(), zap.String("name", queue), zap.String("class", class), zap.String("at", at.String()), zap.Any("args", args))
		return err
	}
	return nil
}

func (p *IFiberEx) checkCronNode() bool {
	rs, err := p.Redis.Get(context.Background(), cronActiveNodeKey).Result()
	if err != nil {
		return false
	}
	return rs == p.NodeId
}
