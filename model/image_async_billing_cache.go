package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
)

// A cache operation is tagged on the hash that actually participated. A cold
// hash populated after the database commit already includes the committed
// change and must never receive the same credit a second time. Losing a hash
// before commit can conservatively delay a credit until its next hydration;
// it must not turn into an extra credit on a replacement hash.
type imageAsyncCacheOperation struct {
	key, marker string
	user        bool
	amount      int
}

type imageAsyncCacheCheckpoint struct {
	marker     string
	operations []imageAsyncCacheOperation
}

func newImageAsyncCacheCheckpoint() *imageAsyncCacheCheckpoint {
	return &imageAsyncCacheCheckpoint{marker: "ImageAsyncCheckpoint:" + common.GetRandomString(24)}
}

// Hydrate outside the database transaction: SQLite may have only one database
// connection. The Lua operation inside the transaction still requires a live
// hash and fails closed rather than authorizing against a stale batch balance.
func prepareImageAsyncReservationCache(task *Task, target int) error {
	if !common.RedisEnabled || target <= task.Quota {
		return nil
	}
	if task.PrivateData.BillingSource == "" || task.PrivateData.BillingSource == "wallet" {
		if _, err := GetUserCache(task.UserId); err != nil {
			return err
		}
	}
	token, err := GetTokenByIds(task.PrivateData.TokenId, task.UserId)
	if err != nil {
		return err
	}
	_, err = GetTokenByKey(token.Key, false)
	return err
}

func (checkpoint *imageAsyncCacheCheckpoint) apply(id int, key string, user bool, amount int, reserve, unlimited bool) error {
	if !common.RedisEnabled {
		return nil
	}
	const script = `
if tonumber(redis.call('HGET', KEYS[1], 'Id') or '0') ~= tonumber(ARGV[1]) then return -1 end
local field = 'RemainQuota'
if ARGV[3] == '1' then
  field = 'Quota'
  if tonumber(redis.call('HGET', KEYS[1], 'CacheSchema') or '0') ~= tonumber(ARGV[7]) then return -1 end
elseif redis.call('HEXISTS', KEYS[1], 'UsedQuota') == 0 then return -1 end
local balance = tonumber(redis.call('HGET', KEYS[1], field))
if balance == nil then return -1 end
if ARGV[5] == '1' then
  if ARGV[6] ~= '1' and balance < tonumber(ARGV[4]) then return 0 end
  redis.call('HINCRBY', KEYS[1], field, -tonumber(ARGV[4]))
  if ARGV[3] ~= '1' then redis.call('HINCRBY', KEYS[1], 'UsedQuota', tonumber(ARGV[4])) end
end
redis.call('HSET', KEYS[1], ARGV[2], '1')
return 1`
	userArg, reserveArg, unlimitedArg := 0, 0, 0
	if user {
		userArg = 1
	}
	if reserve {
		reserveArg = 1
	}
	if unlimited {
		unlimitedArg = 1
	}
	result, err := common.RDB.Eval(context.Background(), script, []string{key}, id, checkpoint.marker, userArg, amount, reserveArg, unlimitedArg, userCacheSchemaVersion).Int()
	if err != nil {
		return fmt.Errorf("image async quota cache unavailable: %w", err)
	}
	if result != 1 {
		if !reserve {
			// A cold refund hash will hydrate from the committed DB.
			return nil
		}
		if result == 0 {
			return errors.New("image async cached quota insufficient")
		}
		return errors.New("image async quota cache unavailable")
	}
	delta := amount
	if reserve {
		delta = -amount
	}
	checkpoint.operations = append(checkpoint.operations, imageAsyncCacheOperation{key: key, marker: checkpoint.marker, user: user, amount: delta})
	return nil
}

func (checkpoint *imageAsyncCacheCheckpoint) finish(committed bool) {
	const script = `
if redis.call('HEXISTS', KEYS[1], ARGV[1]) == 0 then return 0 end
redis.call('HDEL', KEYS[1], ARGV[1])
local delta = tonumber(ARGV[2])
if delta == 0 then return 1 end
local field = 'RemainQuota'
if ARGV[3] == '1' then field = 'Quota' end
redis.call('HINCRBY', KEYS[1], field, delta)
if ARGV[3] ~= '1' then redis.call('HINCRBY', KEYS[1], 'UsedQuota', -delta) end
return 1`
	for _, operation := range checkpoint.operations {
		delta := 0
		if committed && operation.amount > 0 {
			delta = operation.amount
		}
		if !committed && operation.amount < 0 {
			delta = -operation.amount
		}
		userArg := 0
		if operation.user {
			userArg = 1
		}
		if err := common.RDB.Eval(context.Background(), script, []string{operation.key}, operation.marker, delta, userArg).Err(); err != nil {
			common.SysError(fmt.Sprintf("image async quota cache checkpoint completion failed: %v", err))
		}
	}
}
