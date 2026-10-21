-- Token bucket.
-- KEYS[1] = bucket key
-- ARGV: capacity, refill_per_sec, now (секунды, float), cost
-- Возвращает {allowed (0|1), retry_after_ms}
local k    = KEYS[1]
local cap  = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local now  = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])

local b      = redis.call('HMGET', k, 'tokens', 'ts')
local tokens = tonumber(b[1])
local ts     = tonumber(b[2])
if tokens == nil or ts == nil then
  tokens = cap
  ts = now
end

tokens = math.min(cap, tokens + math.max(0, now - ts) * rate)

local allowed = 0
local retry_ms = 0
if tokens >= cost then
  tokens = tokens - cost
  allowed = 1
else
  retry_ms = math.ceil((cost - tokens) / rate * 1000)
end

redis.call('HSET', k, 'tokens', tokens, 'ts', now)
redis.call('PEXPIRE', k, math.ceil(cap / rate * 1000) + 1000)
return {allowed, retry_ms}
