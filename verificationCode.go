package verificationCode

import (
	"context"
	"github.com/go-tron/base-error"
	"github.com/go-tron/random"
	"github.com/go-tron/redis"
	"time"
)

var (
	ErrorVerificationCodeCreateId   = baseError.SystemFactory("2701", "id and code can't be empty")
	ErrorVerificationCodeInterval   = baseError.Factory("2702", "please try again later[{}]")
	ErrorVerificationCodeVerifyId   = baseError.SystemFactory("2703", "id can't be empty")
	ErrorVerificationCodeVerifyCode = baseError.SystemFactory("2704", "code can't be empty")
	ErrorVerificationCodeExpired    = baseError.New("2705", "code verify failed")
	ErrorVerificationCodeVerify     = baseError.New("2706", "code verify failed")
)

type VerificationCode struct {
	Redis        *redis.Redis
	Name         string
	ExpireTime   int
	IntervalTime int
}

func (vc *VerificationCode) Remove(id string) error {
	ctx := context.Background()
	multi := vc.Redis.TxPipeline()
	multi.Del(ctx, vc.Name+":"+id)
	multi.Del(ctx, vc.Name+":i:"+id)
	_, err := multi.Exec(ctx)
	return err
}

func (vc *VerificationCode) Create(id string, code string) error {

	if id == "" || code == "" {
		return ErrorVerificationCodeCreateId()
	}

	remainTime, err := vc.Redis.TTL(context.Background(), vc.Name+":i:"+id).Result()
	if err != nil {
		return err
	}

	if remainTime > 0 {
		return ErrorVerificationCodeInterval(remainTime)
	}

	ctx := context.Background()
	multi := vc.Redis.TxPipeline()
	multi.Set(ctx, vc.Name+":"+id, code, time.Duration(vc.ExpireTime)*time.Second)
	multi.Set(ctx, vc.Name+":i:"+id, time.Now(), time.Duration(vc.IntervalTime)*time.Second)
	_, err = multi.Exec(ctx)

	return err
}

func (vc *VerificationCode) CreateCode(id string) (string, error) {
	code := random.Number(6)
	if err := vc.Create(id, code); err != nil {
		return "", err
	}
	return code, nil
}

func (vc *VerificationCode) Verify(id string, code string) error {

	if id == "" {
		return ErrorVerificationCodeVerifyId()
	}
	if code == "" {
		return ErrorVerificationCodeVerifyCode()
	}

	old, err := vc.Redis.Get(context.Background(), vc.Name+":"+id).Result()

	if err != nil {
		if err == redis.Nil {
			return ErrorVerificationCodeExpired
		}
		return err
	}

	if old != code {
		return ErrorVerificationCodeVerify
	}

	vc.Remove(id)

	return nil
}
