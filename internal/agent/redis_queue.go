package agent

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type RedisQueue struct {
	address, password string
	tlsConfig         *tls.Config
	database          int
	prefix            string
	timeout           time.Duration
	leaseDuration     time.Duration
	errors            chan error
	errorsDropped     atomic.Uint64
}

const redisLeaseDuration = 15 * time.Minute

const (
	maxRedisQueueJobs          = 100_000
	maxRedisQueueJobsPerTenant = 10_000 //nolint:unused // per-tenant quota needs a per-tenant index; global admission cap is enforced today
)

// redisMaxJobsCap is the global admission ceiling for active jobs (pending +
// delayed + running + dead). It is enforced atomically inside the enqueue
// script to bound unbounded backlog/noisy-neighbor load. OLLAMA_AGENT_REDIS_MAX_JOBS
// overrides the default.
func redisMaxJobsCap() int {
	raw := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REDIS_MAX_JOBS"))
	if raw == "" {
		return maxRedisQueueJobs
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return maxRedisQueueJobs
	}
	return value
}

func redisLeaseDurationMillis(duration time.Duration) (string, error) {
	if duration <= 0 {
		duration = redisLeaseDuration
	}
	millis := duration / time.Millisecond
	if duration%time.Millisecond != 0 {
		millis++
	}
	if millis < 3 {
		millis = 3
	}
	if millis > 24*60*60*1000 {
		return "", errors.New("Redis queue lease duration exceeds 24 hours")
	}
	return strconv.FormatInt(int64(millis), 10), nil
}

const redisKeyTypeHelpers = `
local function keyType(key)
  local reply = redis.call('TYPE', key)
  if type(reply) == 'table' then return reply.ok end
  return reply
end
local function typeError(key, expected)
  local actual = keyType(key)
  if actual ~= 'none' and actual ~= expected then
    return redis.error_reply('WRONGTYPE queue key ' .. key .. ': expected ' .. expected .. ', got ' .. actual)
  end
  return nil
end
local function integerValue(value, minimum, maximum)
  if type(value) ~= 'number' or value ~= value or value == math.huge or value == -math.huge then return false end
  return value >= minimum and value <= maximum and value == math.floor(value)
end
local function integerArgument(value, minimum, maximum)
  if type(value) ~= 'string' or value == '' then return nil end
  local number = tonumber(value)
  if not integerValue(number, minimum, maximum) then return nil end
  return number
end
local function nonemptyArgument(value, maximumLength)
  return type(value) == 'string' and value ~= '' and #value <= maximumLength
end
local function distinctKeys(keys)
  local seen = {}
  for _, key in ipairs(keys) do
    if seen[key] then return false end
    seen[key] = true
  end
  return true
end
local function listPositions(listKey, member)
  return redis.call('LPOS', listKey, member, 'COUNT', 0) or {}
end
local function decodeJob(key, raw, expectedID, allowUnstamped)
  if not raw then return nil, redis.error_reply('missing queue job JSON at ' .. key) end
  local ok, job = pcall(cjson.decode, raw)
  if not ok or type(job) ~= 'table' then
    return nil, redis.error_reply('invalid queue job JSON at ' .. key)
  end
  if not nonemptyArgument(job.id, 256) or (expectedID and job.id ~= expectedID)
    or not nonemptyArgument(job.mission_id, 256)
    or (job.status ~= 'pending' and job.status ~= 'running' and job.status ~= 'succeeded' and job.status ~= 'failed' and job.status ~= 'dead_letter')
    or not integerValue(job.attempts, 0, 20)
    or not integerValue(job.max_attempts, 1, 20) then
    return nil, redis.error_reply('invalid queue job fields at ' .. key)
  end
  if not allowUnstamped and (not integerValue(job.created_at_ms, 1, 9007199254740991)
    or not integerValue(job.available_at_ms, 1, 9007199254740991)
    or not integerValue(job.updated_at_ms, 1, 9007199254740991)) then
    return nil, redis.error_reply('legacy queue job requires inspection/migration at ' .. key)
  end
  if job.status == 'pending' and (job.attempts >= job.max_attempts
    or (not allowUnstamped and not integerValue(job.available_seq, 1, 9007199254740991))) then
    return nil, redis.error_reply('pending queue job reached max attempts or has no sequence at ' .. key)
  end
  if job.status == 'running' and (job.attempts < 1 or job.attempts > job.max_attempts
    or not nonemptyArgument(job.worker_id, 256)
    or not nonemptyArgument(job.lease_token, 256)
    or not integerValue(job.locked_at_ms, 1, 9007199254740991)
    or not integerValue(job.lease_until_ms, 1, 9007199254740991)) then
    return nil, redis.error_reply('running queue job has no authoritative lease at ' .. key)
  end
  return job, nil
end
`

const redisClaimScript = redisKeyTypeHelpers + `
if not distinctKeys(KEYS) then return redis.error_reply('queue script requires distinct Redis keys') end
if #ARGV ~= 3 then return redis.error_reply('invalid queue script argument count') end
if not nonemptyArgument(ARGV[1], 256) or not nonemptyArgument(ARGV[2], 256) then
  return redis.error_reply('invalid queue worker or lease token')
end
local leaseDurationMs = integerArgument(ARGV[3], 1, 86400000)
if not leaseDurationMs then return redis.error_reply('invalid queue lease duration') end
local err = typeError(KEYS[1], 'list')
if err then return err end
err = typeError(KEYS[2], 'zset')
if err then return err end
err = typeError(KEYS[3], 'zset')
if err then return err end
err = typeError(KEYS[5], 'list')
if err then return err end
local candidateID = redis.call('LINDEX', KEYS[1], -1)
if not candidateID then return nil end
local candidateKey = KEYS[4] .. candidateID
err = typeError(candidateKey, 'string')
if err then return err end
local candidateRaw = redis.call('GET', candidateKey)
if not candidateRaw then
  -- Orphan pending index entry: the job payload expired (7-day TTL) or was
  -- removed while its id lingered in the pending list. Drop the dangling entry
  -- so the queue does not get stuck repeatedly claiming a missing job; the next
  -- claim proceeds to the following candidate.
  redis.call('LREM', KEYS[1], 0, candidateID)
  return nil
end
local candidateJob, decodeErr = decodeJob(candidateKey, candidateRaw, candidateID)
if decodeErr then return decodeErr end
if candidateJob.status ~= 'pending' then
  return redis.error_reply('pending index points to non-pending job at ' .. candidateKey)
end
if candidateJob.attempts >= candidateJob.max_attempts then
  return redis.error_reply('pending job reached max attempts at ' .. candidateKey)
end
if #listPositions(KEYS[1], candidateID) ~= 1
	  or redis.call('ZSCORE', KEYS[2], candidateID)
	  or redis.call('ZSCORE', KEYS[3], candidateID)
  or #listPositions(KEYS[5], candidateID) ~= 0 then
  return redis.error_reply('pending job has duplicate/conflicting queue indexes at ' .. candidateKey)
end
local id = redis.call('RPOP', KEYS[1])
if not id then return nil end
local job = candidateJob
job.status = 'running'
job.attempts = job.attempts + 1
job.worker_id = ARGV[1]
job.lease_token = ARGV[2]
local serverTime = redis.call('TIME')
local nowMs = tonumber(serverTime[1]) * 1000 + math.floor(tonumber(serverTime[2]) / 1000)
local leaseUntilMs = nowMs + leaseDurationMs
job.locked_at_ms = nowMs
job.updated_at_ms = nowMs
job.lease_until_ms = leaseUntilMs
job.locked_at = cjson.null
job.updated_at = cjson.null
redis.call('SET', KEYS[4] .. id, cjson.encode(job), 'EX', '604800')
redis.call('ZADD', KEYS[3], leaseUntilMs, id)
return cjson.encode(job)
`

const redisMoveDueScript = redisKeyTypeHelpers + `
if not distinctKeys(KEYS) then return redis.error_reply('queue script requires distinct Redis keys') end
if #ARGV ~= 0 then return redis.error_reply('invalid queue script argument count') end
local serverTime = redis.call('TIME')
local nowMs = tonumber(serverTime[1]) * 1000 + math.floor(tonumber(serverTime[2]) / 1000)
local err = typeError(KEYS[1], 'zset')
if err then return err end
err = typeError(KEYS[3], 'list')
if err then return err end
err = typeError(KEYS[4], 'zset')
if err then return err end
err = typeError(KEYS[5], 'list')
if err then return err end
local ids = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', nowMs)
local entries = {}
for _, id in ipairs(ids) do
  local jobKey = KEYS[2] .. id
  err = typeError(jobKey, 'string')
  if err then return err end
  local raw = redis.call('GET', jobKey)
  local job, decodeErr = decodeJob(jobKey, raw, id)
  if decodeErr then return decodeErr end
  if job.status ~= 'pending' then
    return redis.error_reply('delayed index points to non-pending job at ' .. jobKey)
  end
  if #listPositions(KEYS[3], id) ~= 0 or redis.call('ZSCORE', KEYS[4], id)
    or #listPositions(KEYS[5], id) ~= 0 then
    return redis.error_reply('delayed job has duplicate/conflicting queue indexes at ' .. jobKey)
  end
  table.insert(entries, { id = id, job = job })
end
table.sort(entries, function(a, b)
  local aSeq = tonumber(a.job.available_seq)
  local bSeq = tonumber(b.job.available_seq)
  if aSeq ~= bSeq then return aSeq < bSeq end
  if a.job.created_at_ms ~= b.job.created_at_ms then return a.job.created_at_ms < b.job.created_at_ms end
  return a.id < b.id
end)
local moved = 0
for _, entry in ipairs(entries) do
  if redis.call('ZSCORE', KEYS[1], entry.id) then
    redis.call('LPUSH', KEYS[3], entry.id)
    redis.call('ZREM', KEYS[1], entry.id)
    moved = moved + 1
  end
end
return moved
`

const redisHeartbeatScript = redisKeyTypeHelpers + `
if not distinctKeys(KEYS) then return redis.error_reply('queue script requires distinct Redis keys') end
if #ARGV ~= 4 then return redis.error_reply('invalid queue script argument count') end
if not nonemptyArgument(ARGV[1], 256) or not nonemptyArgument(ARGV[2], 256)
  or not nonemptyArgument(ARGV[4], 256) then
  return redis.error_reply('invalid queue heartbeat identity')
end
local leaseDurationMs = integerArgument(ARGV[3], 1, 86400000)
if not leaseDurationMs then return redis.error_reply('invalid queue lease duration') end
local err = typeError(KEYS[1], 'string')
if err then return err end
err = typeError(KEYS[2], 'zset')
if err then return err end
err = typeError(KEYS[3], 'list')
if err then return err end
err = typeError(KEYS[4], 'zset')
if err then return err end
err = typeError(KEYS[5], 'list')
if err then return err end
local raw = redis.call('GET', KEYS[1])
if not raw then return nil end
local job, decodeErr = decodeJob(KEYS[1], raw, ARGV[4])
if decodeErr then return decodeErr end
if job.status ~= 'running' or job.worker_id ~= ARGV[1] or job.lease_token ~= ARGV[2] then return nil end
local expiresAt = redis.call('ZSCORE', KEYS[2], job.id)
local serverTime = redis.call('TIME')
local nowMs = tonumber(serverTime[1]) * 1000 + math.floor(tonumber(serverTime[2]) / 1000)
if not expiresAt or tonumber(expiresAt) ~= job.lease_until_ms or tonumber(expiresAt) <= nowMs then return nil end
if #listPositions(KEYS[3], job.id) ~= 0 or redis.call('ZSCORE', KEYS[4], job.id)
  or #listPositions(KEYS[5], job.id) ~= 0 then
  return redis.error_reply('running job has duplicate/conflicting queue indexes at ' .. KEYS[1])
end
local leaseUntilMs = nowMs + leaseDurationMs
job.lease_until_ms = leaseUntilMs
job.updated_at_ms = nowMs
job.locked_at = cjson.null
job.updated_at = cjson.null
redis.call('SET', KEYS[1], cjson.encode(job), 'EX', '604800')
redis.call('ZADD', KEYS[2], leaseUntilMs, job.id)
return 'ok'
`

const redisAckScript = redisKeyTypeHelpers + `
if not distinctKeys(KEYS) then return redis.error_reply('queue script requires distinct Redis keys') end
if #ARGV ~= 3 then return redis.error_reply('invalid queue script argument count') end
if not nonemptyArgument(ARGV[1], 256) or not nonemptyArgument(ARGV[2], 256)
  or not nonemptyArgument(ARGV[3], 256) then return redis.error_reply('invalid queue ack identity') end
local err = typeError(KEYS[1], 'string')
if err then return err end
err = typeError(KEYS[2], 'zset')
if err then return err end
err = typeError(KEYS[3], 'list')
if err then return err end
err = typeError(KEYS[4], 'zset')
if err then return err end
err = typeError(KEYS[5], 'list')
if err then return err end
local raw = redis.call('GET', KEYS[1])
if not raw then return nil end
local job, decodeErr = decodeJob(KEYS[1], raw, ARGV[3])
if decodeErr then return decodeErr end
if job.status ~= 'running' or job.worker_id ~= ARGV[1] or job.lease_token ~= ARGV[2] then return nil end
local expiresAt = redis.call('ZSCORE', KEYS[2], job.id)
local serverTime = redis.call('TIME')
local nowMs = tonumber(serverTime[1]) * 1000 + math.floor(tonumber(serverTime[2]) / 1000)
if not expiresAt or tonumber(expiresAt) ~= job.lease_until_ms or tonumber(expiresAt) <= nowMs then return nil end
if #listPositions(KEYS[3], job.id) ~= 0 or redis.call('ZSCORE', KEYS[4], job.id)
  or #listPositions(KEYS[5], job.id) ~= 0 then
  return redis.error_reply('running job has duplicate/conflicting queue indexes at ' .. KEYS[1])
end
job.status = 'succeeded'
job.worker_id = ''
job.lease_token = ''
job.locked_at = cjson.null
job.locked_at_ms = 0
job.lease_until_ms = 0
job.updated_at_ms = nowMs
job.updated_at = cjson.null
redis.call('SET', KEYS[1], cjson.encode(job), 'EX', '604800')
redis.call('ZREM', KEYS[2], job.id)
return cjson.encode(job)
`

const redisNackScript = redisKeyTypeHelpers + `
if not distinctKeys(KEYS) then return redis.error_reply('queue script requires distinct Redis keys') end
if #ARGV ~= 6 then return redis.error_reply('invalid queue script argument count') end
if not nonemptyArgument(ARGV[1], 256) or not nonemptyArgument(ARGV[2], 256)
  or not nonemptyArgument(ARGV[5], 256) or type(ARGV[3]) ~= 'string' or #ARGV[3] > 2000 then
  return redis.error_reply('invalid queue nack identity or error')
end
if ARGV[4] ~= 'retry' and ARGV[4] ~= 'terminal' then
  return redis.error_reply('invalid queue nack mode')
end
local backoffMs = integerArgument(ARGV[6], 0, 300000)
if not backoffMs then return redis.error_reply('invalid queue retry backoff') end
if ARGV[4] == 'retry' and backoffMs < 1 then return redis.error_reply('retry backoff must be positive') end
local err = typeError(KEYS[1], 'string')
if err then return err end
err = typeError(KEYS[2], 'zset')
if err then return err end
err = typeError(KEYS[3], 'zset')
if err then return err end
err = typeError(KEYS[4], 'string')
if err then return err end
err = typeError(KEYS[5], 'list')
if err then return err end
err = typeError(KEYS[6], 'list')
if err then return err end
local raw = redis.call('GET', KEYS[1])
if not raw then return nil end
local job, decodeErr = decodeJob(KEYS[1], raw, ARGV[5])
if decodeErr then return decodeErr end
if job.status ~= 'running' or job.worker_id ~= ARGV[1] or job.lease_token ~= ARGV[2] then return nil end
if job.attempts < 1 or job.attempts > job.max_attempts then
  return redis.error_reply('invalid running queue attempt count')
end
local expiresAt = redis.call('ZSCORE', KEYS[2], job.id)
local serverTime = redis.call('TIME')
local nowMs = tonumber(serverTime[1]) * 1000 + math.floor(tonumber(serverTime[2]) / 1000)
if not expiresAt or tonumber(expiresAt) ~= job.lease_until_ms or tonumber(expiresAt) <= nowMs then return nil end
if #listPositions(KEYS[5], job.id) ~= 0 or redis.call('ZSCORE', KEYS[3], job.id)
  or #listPositions(KEYS[6], job.id) ~= 0 then
  return redis.error_reply('running job has duplicate/conflicting queue indexes at ' .. KEYS[1])
end
local sequenceValue = redis.call('GET', KEYS[4]) or '0'
local currentSequence = tonumber(sequenceValue)
if not integerValue(currentSequence, 0, 9007199254740990) then
  return redis.error_reply('invalid queue sequence counter')
end
job.last_error = ARGV[3]
job.worker_id = ''
job.lease_token = ''
job.locked_at = cjson.null
job.locked_at_ms = 0
job.lease_until_ms = 0
job.updated_at_ms = nowMs
job.updated_at = cjson.null
if ARGV[4] == 'terminal' then
  job.status = 'failed'
  redis.call('LPUSH', KEYS[6], job.id)
elseif job.attempts >= job.max_attempts then
  job.status = 'dead_letter'
  redis.call('LPUSH', KEYS[6], job.id)
else
  job.available_seq = redis.call('INCR', KEYS[4])
  job.status = 'pending'
  job.available_at_ms = nowMs + backoffMs
  job.available_at = cjson.null
  redis.call('ZADD', KEYS[3], job.available_at_ms, job.id)
end
redis.call('SET', KEYS[1], cjson.encode(job), 'EX', '604800')
redis.call('ZREM', KEYS[2], job.id)
return cjson.encode(job)
`

const redisReclaimScript = redisKeyTypeHelpers + `
if not distinctKeys(KEYS) then return redis.error_reply('queue script requires distinct Redis keys') end
if #ARGV ~= 0 then return redis.error_reply('invalid queue script argument count') end
local err = typeError(KEYS[1], 'zset')
if err then return err end
err = typeError(KEYS[3], 'list')
if err then return err end
err = typeError(KEYS[4], 'list')
if err then return err end
err = typeError(KEYS[5], 'string')
if err then return err end
err = typeError(KEYS[6], 'zset')
if err then return err end
local sequenceValue = redis.call('GET', KEYS[5]) or '0'
local sequence = tonumber(sequenceValue)
if not integerValue(sequence, 0, 9007199254740990) then
  return redis.error_reply('invalid queue sequence counter')
end
local serverTime = redis.call('TIME')
local nowMs = tonumber(serverTime[1]) * 1000 + math.floor(tonumber(serverTime[2]) / 1000)
local ids = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', nowMs)
local entries = {}
local retryCount = 0
for _, id in ipairs(ids) do
  local jobKey = KEYS[2] .. id
  err = typeError(jobKey, 'string')
  if err then return err end
  local raw = redis.call('GET', jobKey)
  local job, decodeErr = decodeJob(jobKey, raw, id)
  if decodeErr then return decodeErr end
  if job.status ~= 'running' then
    return redis.error_reply('lease index points to non-running job at ' .. jobKey)
  end
  local leaseScore = redis.call('ZSCORE', KEYS[1], id)
  if not leaseScore or tonumber(leaseScore) ~= job.lease_until_ms or tonumber(leaseScore) > nowMs then
    return redis.error_reply('lease index does not match running job at ' .. jobKey)
  end
  if #listPositions(KEYS[3], id) ~= 0 or redis.call('ZSCORE', KEYS[6], id)
    or #listPositions(KEYS[4], id) ~= 0 then
    return redis.error_reply('running job has duplicate/conflicting queue indexes at ' .. jobKey)
  end
  if job.attempts < 1 or job.attempts > job.max_attempts then
    return redis.error_reply('invalid running queue attempt count at ' .. jobKey)
  end
  if job.attempts < job.max_attempts then retryCount = retryCount + 1 end
  table.insert(entries, { id = id, job = job })
end
if sequence + retryCount > 9007199254740991 then
  return redis.error_reply('queue sequence counter overflow')
end
local reclaimed = 0
for _, entry in ipairs(entries) do
  local id = entry.id
  local job = entry.job
  redis.call('ZREM', KEYS[1], id)
  job.worker_id = ''
  job.lease_token = ''
  job.locked_at = cjson.null
  job.locked_at_ms = 0
  job.lease_until_ms = 0
  job.updated_at_ms = nowMs
  job.updated_at = cjson.null
  if job.attempts >= job.max_attempts then
    job.status = 'dead_letter'
    job.last_error = 'worker lease expired after max attempts'
    redis.call('SET', KEYS[2] .. id, cjson.encode(job), 'EX', '604800')
    redis.call('LPUSH', KEYS[3], id)
  else
    sequence = sequence + 1
    job.available_seq = sequence
    job.status = 'pending'
    job.available_at_ms = nowMs
    job.available_at = cjson.null
    redis.call('SET', KEYS[2] .. id, cjson.encode(job), 'EX', '604800')
    redis.call('LPUSH', KEYS[4], id)
  end
  reclaimed = reclaimed + 1
end
if sequence > tonumber(sequenceValue) then redis.call('SET', KEYS[5], tostring(sequence)) end
return reclaimed
`

const redisEnqueueScript = redisKeyTypeHelpers + `
if not distinctKeys(KEYS) then return redis.error_reply('queue script requires distinct Redis keys') end
if #ARGV < 3 or #ARGV > 4 then return redis.error_reply('invalid queue script argument count') end
if not nonemptyArgument(ARGV[2], 256) then return redis.error_reply('invalid mission id') end
if type(ARGV[3]) ~= 'string' or #ARGV[3] > 256 then return redis.error_reply('invalid organization id') end
local err = typeError(KEYS[1], 'string')
if err then return err end
err = typeError(KEYS[3], 'list')
if err then return err end
err = typeError(KEYS[4], 'zset')
if err then return err end
err = typeError(KEYS[5], 'zset')
if err then return err end
err = typeError(KEYS[6], 'list')
if err then return err end
err = typeError(KEYS[7], 'string')
if err then return err end
local job, decodeErr = decodeJob('enqueue payload', ARGV[1], nil, true)
if decodeErr then return decodeErr end
if not nonemptyArgument(job.id, 256) or type(job.mission_id) ~= 'string' or job.mission_id ~= ARGV[2]
	  or (job.organization_id or '') ~= ARGV[3]
  or job.status ~= 'pending' or job.attempts ~= 0 or job.attempts >= job.max_attempts then
  return redis.error_reply('invalid enqueue payload identity or initial state')
end
local newJobKey = KEYS[2] .. job.id
err = typeError(newJobKey, 'string')
if err then return err end
local existingID = redis.call('GET', KEYS[1])
if existingID then
  if not nonemptyArgument(existingID, 256) then return redis.error_reply('invalid mission queue index') end
  local existingKey = KEYS[2] .. existingID
  err = typeError(existingKey, 'string')
  if err then return err end
  local existingRaw = redis.call('GET', existingKey)
  local existing, existingErr = decodeJob(existingKey, existingRaw, existingID)
  if existingErr then return existingErr end
  if existing.mission_id ~= ARGV[2] then
    return redis.error_reply('mission queue index points to a different mission')
  end
	if existing.status == 'pending' or existing.status == 'running' then
	  local existingOrganization = existing.organization_id or ''
	  local bindOrganization = false
	  if existingOrganization ~= ARGV[3] then
	    if existingOrganization ~= '' or ARGV[3] == '' then
	      return redis.error_reply('mission queue job organization does not match')
	    end
	    bindOrganization = true
	  end
	  local pendingPositions = listPositions(KEYS[3], existingID)
    local delayedScore = redis.call('ZSCORE', KEYS[4], existingID)
    local leaseScore = redis.call('ZSCORE', KEYS[5], existingID)
    local deadPositions = listPositions(KEYS[6], existingID)
    if #deadPositions ~= 0 then return redis.error_reply('active mission queue job appears in dead-letter index') end
	    if existing.status == 'pending' then
      local pendingMemberships = #pendingPositions
      if delayedScore then pendingMemberships = pendingMemberships + 1 end
      if pendingMemberships ~= 1 or leaseScore then
        return redis.error_reply('pending mission queue job has inconsistent queue membership')
      end
    else
      if #pendingPositions ~= 0 or delayedScore or not leaseScore
        or tonumber(leaseScore) ~= existing.lease_until_ms then
        return redis.error_reply('running mission queue job has inconsistent lease membership')
	      end
	    end
	    if bindOrganization then
	      existing.organization_id = ARGV[3]
	      existingRaw = cjson.encode(existing)
	      redis.call('SET', existingKey, existingRaw, 'EX', '604800')
	    end
	    return existingRaw
  end
end
if redis.call('EXISTS', newJobKey) == 1 then
  return redis.error_reply('queue job id already exists')
end
-- Global admission cap: bound total active jobs (pending + delayed + running +
-- dead) so a flood cannot grow the queue without limit. Only applies to a
-- genuinely new job; idempotent re-enqueue of an existing mission returned above.
if #ARGV == 4 then
  local cap = tonumber(ARGV[4])
  if cap and cap > 0 then
    local active = redis.call('LLEN', KEYS[3]) + redis.call('ZCARD', KEYS[4]) + redis.call('ZCARD', KEYS[5]) + redis.call('LLEN', KEYS[6])
    if active >= cap then
      return redis.error_reply('queue admission rejected: global job quota reached')
    end
  end
end
local serverTime = redis.call('TIME')
local nowMs = tonumber(serverTime[1]) * 1000 + math.floor(tonumber(serverTime[2]) / 1000)
job.created_at_ms = nowMs
job.available_at_ms = nowMs
job.updated_at_ms = nowMs
job.available_seq = integerArgument(redis.call('GET', KEYS[7]) or '0', 0, 9007199254740990)
if not job.available_seq then return redis.error_reply('invalid queue sequence counter') end
job.available_seq = redis.call('INCR', KEYS[7])
job.created_at = cjson.null
job.available_at = cjson.null
job.updated_at = cjson.null
local encoded = cjson.encode(job)
redis.call('SET', newJobKey, encoded, 'EX', '604800')
redis.call('SET', KEYS[1], job.id, 'EX', '604800')
redis.call('LPUSH', KEYS[3], job.id)
return encoded
`

const redisReplayScript = redisKeyTypeHelpers + `
if not distinctKeys(KEYS) then return redis.error_reply('queue script requires distinct Redis keys') end
if #ARGV ~= 2 then return redis.error_reply('invalid queue script argument count') end
if not nonemptyArgument(ARGV[1], 256) then return redis.error_reply('invalid queue replay job id') end
if type(ARGV[2]) ~= 'string' or #ARGV[2] > 256 then return redis.error_reply('invalid queue replay organization id') end
local err = typeError(KEYS[1], 'string')
if err then return err end
err = typeError(KEYS[2], 'zset')
if err then return err end
err = typeError(KEYS[3], 'zset')
if err then return err end
err = typeError(KEYS[4], 'list')
if err then return err end
err = typeError(KEYS[5], 'list')
if err then return err end
err = typeError(KEYS[6], 'string')
if err then return err end
local raw = redis.call('GET', KEYS[1])
if not raw then return nil end
local job, decodeErr = decodeJob(KEYS[1], raw, ARGV[1])
if decodeErr then return decodeErr end
if job.status ~= 'dead_letter' then return nil end
local organizationID = job.organization_id or ''
if organizationID ~= ARGV[2] then
  return redis.error_reply('queue replay organization does not match')
end
if #listPositions(KEYS[4], job.id) ~= 1 or #listPositions(KEYS[5], job.id) ~= 0
  or redis.call('ZSCORE', KEYS[2], job.id) or redis.call('ZSCORE', KEYS[3], job.id) then
  return redis.error_reply('dead-letter job has missing/duplicate/conflicting queue indexes at ' .. KEYS[1])
end
local sequenceValue = redis.call('GET', KEYS[6]) or '0'
local currentSequence = tonumber(sequenceValue)
if not integerValue(currentSequence, 0, 9007199254740990) then
  return redis.error_reply('invalid queue sequence counter')
end
local serverTime = redis.call('TIME')
local nowMs = tonumber(serverTime[1]) * 1000 + math.floor(tonumber(serverTime[2]) / 1000)
job.status = 'pending'
job.attempts = 0
job.last_error = ''
job.worker_id = ''
job.lease_token = ''
job.locked_at = cjson.null
job.locked_at_ms = 0
job.lease_until_ms = 0
job.available_at_ms = nowMs
job.updated_at_ms = nowMs
job.available_seq = redis.call('INCR', KEYS[6])
job.available_at = cjson.null
job.updated_at = cjson.null
redis.call('SET', KEYS[1], cjson.encode(job), 'EX', '604800')
redis.call('LREM', KEYS[4], 0, job.id)
redis.call('LPUSH', KEYS[5], job.id)
return cjson.encode(job)
`

func OpenRedisQueue(ctx context.Context, rawURL, prefix string) (*RedisQueue, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return nil, errors.New("invalid Redis URL")
	}
	if u.Scheme != "redis" && u.Scheme != "rediss" {
		return nil, errors.New("Redis URL must use redis:// or rediss://")
	}
	host := u.Hostname()
	if host == "" {
		return nil, errors.New("invalid Redis host")
	}
	var tlsConfig *tls.Config
	if u.Scheme == "rediss" {
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host}
	} else {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return nil, errors.New("plaintext redis:// is restricted to literal loopback IP addresses; use rediss:// for remote Redis")
		}
	}
	database := 0
	if value := strings.TrimPrefix(u.Path, "/"); value != "" {
		database, err = strconv.Atoi(value)
		if err != nil || database < 0 {
			return nil, errors.New("invalid Redis database")
		}
	}
	password, _ := u.User.Password()
	queuePrefix := strings.TrimSuffix(strings.TrimSpace(prefix), ":")
	if queuePrefix == "" {
		queuePrefix = "ollama:agent"
	}
	if !validRedisQueuePrefix(queuePrefix) {
		return nil, errors.New("invalid Redis queue prefix")
	}
	queue := &RedisQueue{address: u.Host, password: password, tlsConfig: tlsConfig, database: database, prefix: queuePrefix, timeout: 5 * time.Second, leaseDuration: redisLeaseDuration, errors: make(chan error, 32)}
	if err := queue.ping(ctx); err != nil {
		return nil, err
	}
	return queue, nil
}

func validRedisQueuePrefix(prefix string) bool {
	if prefix == "" || len(prefix) > 128 {
		return false
	}
	for _, char := range prefix {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == ':' || char == '_' || char == '-') {
			return false
		}
	}
	return true
}

func (q *RedisQueue) key(name string) string  { return q.prefix + ":" + name }
func (q *RedisQueue) jobKey(id string) string { return q.key("job:" + id) }
func (q *RedisQueue) missionKey(id string) string {
	return q.key("mission:" + id)
}
func (q *RedisQueue) pendingKey() string      { return q.key("pending") }
func (q *RedisQueue) delayedKey() string      { return q.key("delayed") }
func (q *RedisQueue) deadKey() string         { return q.key("dead") }
func (q *RedisQueue) leaseKey() string        { return q.key("leases") }
func (q *RedisQueue) allJobsKey() string      { return q.key("jobs") }    //nolint:unused // compatibility/security surface retained for future adapter wiring
func (q *RedisQueue) tenantKeyPrefix() string { return q.key("tenant:") } //nolint:unused // compatibility/security surface retained for future adapter wiring

func (q *RedisQueue) Enqueue(missionID string, maxAttempts int) (QueueJob, error) {
	return q.EnqueueForOrganization("", missionID, maxAttempts)
}

func (q *RedisQueue) EnqueueForOrganization(organizationID, missionID string, maxAttempts int) (QueueJob, error) {
	organizationID = strings.TrimSpace(organizationID)
	if strings.TrimSpace(missionID) == "" {
		return QueueJob{}, errors.New("mission id is required")
	}
	if len(organizationID) > 256 {
		return QueueJob{}, errors.New("organization id is invalid")
	}
	if maxAttempts <= 0 || maxAttempts > 20 {
		maxAttempts = 3
	}
	now := time.Now().UTC()
	job := QueueJob{ID: "job_" + uuid.NewString(), MissionID: missionID, OrganizationID: organizationID, Status: QueuePending, MaxAttempts: maxAttempts, AvailableAt: now, CreatedAt: now, UpdatedAt: now}
	data, err := json.Marshal(job)
	if err != nil {
		return QueueJob{}, err
	}
	value, err := q.do(context.Background(), "EVAL", redisEnqueueScript, "7", q.missionKey(missionID), q.key("job:"), q.pendingKey(), q.delayedKey(), q.leaseKey(), q.deadKey(), q.key("sequence"), string(data), missionID, organizationID, strconv.Itoa(redisMaxJobsCap()))
	if err != nil {
		return QueueJob{}, err
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return QueueJob{}, errors.New("Redis enqueue returned an invalid job")
	}
	if err := unmarshalRedisQueueJob([]byte(text), &job); err != nil {
		return QueueJob{}, err
	}
	return job, nil
}

func (q *RedisQueue) Claim(workerID string, now time.Time) (QueueJob, bool, error) {
	if strings.TrimSpace(workerID) == "" {
		return QueueJob{}, false, errors.New("worker id is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	leaseDuration, err := redisLeaseDurationMillis(q.leaseDuration)
	if err != nil {
		return QueueJob{}, false, err
	}
	if err := q.moveDue(context.Background(), now); err != nil {
		return QueueJob{}, false, err
	}
	if err := q.reclaimExpired(context.Background(), now); err != nil {
		return QueueJob{}, false, err
	}
	leaseToken := uuid.NewString()
	value, err := q.do(context.Background(), "EVAL", redisClaimScript, "5", q.pendingKey(), q.delayedKey(), q.leaseKey(), q.key("job:"), q.deadKey(), workerID, leaseToken, leaseDuration)
	if err != nil {
		return QueueJob{}, false, err
	}
	if value == nil {
		return QueueJob{}, false, nil
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return QueueJob{}, false, errors.New("Redis claim returned an invalid job")
	}
	var job QueueJob
	if err := unmarshalRedisQueueJob([]byte(text), &job); err != nil {
		return QueueJob{}, false, err
	}
	return job, true, nil
}

func (q *RedisQueue) renewLease(claim QueueJob) error {
	if claim.ID == "" || claim.WorkerID == "" || claim.LeaseToken == "" {
		return ErrQueueLeaseLost
	}
	leaseDuration, err := redisLeaseDurationMillis(q.leaseDuration)
	if err != nil {
		return err
	}
	value, err := q.do(context.Background(), "EVAL", redisHeartbeatScript, "5", q.jobKey(claim.ID), q.leaseKey(), q.pendingKey(), q.delayedKey(), q.deadKey(), claim.WorkerID, claim.LeaseToken, leaseDuration, claim.ID)
	if err != nil {
		return err
	}
	if text, ok := value.(string); !ok || text != "ok" {
		return ErrQueueLeaseLost
	}
	return nil
}

func (q *RedisQueue) runWithHeartbeat(ctx context.Context, claim QueueJob, handler func(context.Context, QueueJob) error) error {
	if handler == nil {
		return errors.New("queue handler is required")
	}
	handlerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runQueueHandler(handlerCtx, claim, handler) }()
	leaseDuration := q.leaseDuration
	if leaseDuration <= 0 {
		leaseDuration = redisLeaseDuration
	}
	interval := leaseDuration / 3
	if interval < time.Millisecond {
		interval = time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			return queueHandlerResult(ctx, err)
		case <-ctx.Done():
			return drainQueueHandler(done, cancel, func() error { return q.renewLease(claim) }, interval, ctx.Err())
		case <-ticker.C:
			if err := q.renewLease(claim); err != nil {
				return drainQueueHandler(done, cancel, func() error { return q.renewLease(claim) }, interval, fmt.Errorf("%w: renew Redis queue lease: %v", ErrQueueLeaseLost, err))
			}
		}
	}
}

func (q *RedisQueue) Ack(claim QueueJob) error {
	if claim.ID == "" || claim.WorkerID == "" || claim.LeaseToken == "" {
		return ErrQueueLeaseLost
	}
	value, err := q.do(context.Background(), "EVAL", redisAckScript, "5", q.jobKey(claim.ID), q.leaseKey(), q.pendingKey(), q.delayedKey(), q.deadKey(), claim.WorkerID, claim.LeaseToken, claim.ID)
	if err != nil {
		return err
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return ErrQueueLeaseLost
	}
	return nil
}

func (q *RedisQueue) Nack(claim QueueJob, runErr error) (QueueJob, error) {
	if claim.ID == "" || claim.WorkerID == "" || claim.LeaseToken == "" {
		return QueueJob{}, ErrQueueLeaseLost
	}
	job, err := q.get(claim.ID)
	if err != nil {
		return QueueJob{}, err
	}
	if job.Status != QueueRunning || job.WorkerID != claim.WorkerID || job.LeaseToken != claim.LeaseToken {
		return QueueJob{}, ErrQueueLeaseLost
	}
	if job.Attempts < 1 {
		return QueueJob{}, errors.New("invalid Redis queue attempt count")
	}
	lastError := ""
	if runErr != nil {
		lastError = limitError(runErr.Error(), 2000)
	}
	backoff := time.Duration(1<<(job.Attempts-1)) * time.Second
	if backoff > 5*time.Minute {
		backoff = 5 * time.Minute
	}
	mode := "retry"
	if errors.Is(runErr, ErrQueueNonRetryable) {
		mode = "terminal"
	}
	value, err := q.do(context.Background(), "EVAL", redisNackScript, "6", q.jobKey(job.ID), q.leaseKey(), q.delayedKey(), q.key("sequence"), q.pendingKey(), q.deadKey(), claim.WorkerID, claim.LeaseToken, lastError, mode, job.ID, strconv.FormatInt(backoff.Milliseconds(), 10))
	if err != nil {
		return QueueJob{}, err
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return QueueJob{}, ErrQueueLeaseLost
	}
	if err := unmarshalRedisQueueJob([]byte(text), &job); err != nil {
		return QueueJob{}, err
	}
	return job, nil
}

func (q *RedisQueue) Replay(jobID string) (QueueJob, error) {
	return q.ReplayForOrganization("", jobID)
}

func (q *RedisQueue) ReplayForOrganization(organizationID, jobID string) (QueueJob, error) {
	organizationID = strings.TrimSpace(organizationID)
	if len(organizationID) > 256 {
		return QueueJob{}, errors.New("organization id is invalid")
	}
	value, err := q.do(context.Background(), "EVAL", redisReplayScript, "6", q.jobKey(jobID), q.delayedKey(), q.leaseKey(), q.deadKey(), q.pendingKey(), q.key("sequence"), jobID, organizationID)
	if err != nil {
		return QueueJob{}, err
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return QueueJob{}, fmt.Errorf("job %s is not replayable", jobID)
	}
	var job QueueJob
	if err := unmarshalRedisQueueJob([]byte(text), &job); err != nil {
		return QueueJob{}, err
	}
	return job, nil
}

// List returns the queue jobs for a status, or an error when the Redis
// dependency is unavailable or returns an unexpected response. Callers that
// derive health MUST distinguish an error (dependency down) from an empty
// slice (genuinely no jobs); returning nil for both would mask outages as a
// healthy empty queue.
func (q *RedisQueue) List(status QueueStatus) ([]QueueJob, error) {
	const maxScans = 10000
	const maxJobs = 100000
	const scanCount = 256
	keys := make(map[string]struct{})
	cursor := "0"
	for scans := 0; scans < maxScans; scans++ { //nolint:intrange // scans is used by the exhaustion guard below
		value, err := q.do(context.Background(), "SCAN", cursor, "MATCH", q.key("job:*"), "COUNT", strconv.Itoa(scanCount))
		if err != nil {
			return nil, fmt.Errorf("redis queue list scan: %w", err)
		}
		response, ok := value.([]any)
		if !ok || len(response) != 2 {
			return nil, errors.New("redis queue list: unexpected SCAN response")
		}
		cursor, ok = response[0].(string)
		if !ok {
			return nil, errors.New("redis queue list: invalid SCAN cursor")
		}
		page, ok := response[1].([]any)
		if !ok {
			return nil, errors.New("redis queue list: invalid SCAN page")
		}
		for _, raw := range page {
			key, ok := raw.(string)
			if !ok || !strings.HasPrefix(key, q.key("job:")) {
				continue
			}
			keys[key] = struct{}{}
			if len(keys) > maxJobs {
				return nil, fmt.Errorf("redis queue list: exceeded %d keys", maxJobs)
			}
		}
		if cursor == "0" {
			break
		}
		if scans == maxScans-1 {
			return nil, errors.New("redis queue list: scan did not converge")
		}
	}
	keyList := make([]string, 0, len(keys))
	for key := range keys {
		keyList = append(keyList, key)
	}
	sort.Strings(keyList)
	jobs := []QueueJob{}
	for offset := 0; offset < len(keyList); offset += 128 {
		end := min(offset+128, len(keyList))
		args := make([]string, 0, end-offset+1)
		args = append(args, "MGET")
		args = append(args, keyList[offset:end]...)
		value, err := q.do(context.Background(), args...)
		if err != nil {
			return nil, fmt.Errorf("redis queue list mget: %w", err)
		}
		values, ok := value.([]any)
		if !ok || len(values) != end-offset {
			return nil, errors.New("redis queue list: unexpected MGET response")
		}
		for _, raw := range values {
			text, ok := raw.(string)
			if !ok {
				continue
			}
			var job QueueJob
			if err := unmarshalRedisQueueJob([]byte(text), &job); err == nil && (status == "" || job.Status == status) {
				jobs = append(jobs, job)
			}
		}
	}
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].CreatedAt.Equal(jobs[j].CreatedAt) {
			return jobs[i].ID < jobs[j].ID
		}
		return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
	})
	return jobs, nil
}

func (q *RedisQueue) Start(ctx context.Context, workerID string, handler func(context.Context, QueueJob) error) error {
	if q == nil {
		return errors.New("Redis queue is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(workerID) == "" {
		return errors.New("worker id is required")
	}
	if handler == nil {
		return errors.New("queue handler is required")
	}
	go func() {
		claimFailures := 0
		for {
			if ctx.Err() != nil {
				return
			}
			job, ok, err := q.Claim(workerID, time.Now().UTC())
			if err != nil {
				claimFailures++
				q.reportError(fmt.Errorf("redis queue claim: %w", err))
			} else {
				claimFailures = 0
			}
			if err == nil && ok {
				runErr := queueHandlerResult(ctx, q.runWithHeartbeat(ctx, job, handler))
				if runErr != nil {
					if _, nackErr := q.Nack(job, runErr); nackErr != nil {
						q.reportError(fmt.Errorf("redis queue nack %s: %w", job.ID, nackErr))
					}
				} else {
					if ackErr := q.Ack(job); ackErr != nil {
						q.reportError(fmt.Errorf("redis queue ack %s: %w", job.ID, ackErr))
					}
				}
				continue
			}
			delay := 500 * time.Millisecond
			if err != nil {
				delay = redisQueueClaimRetryDelay(claimFailures)
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()
	return nil
}

func redisQueueClaimRetryDelay(failures int) time.Duration {
	if failures < 1 {
		failures = 1
	}
	shift := min(failures-1, 6)
	backoff := 500 * time.Millisecond * time.Duration(1<<shift)
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}
	half := backoff / 2
	return half + time.Duration(rand.Int63n(int64(backoff-half)))
}

func (q *RedisQueue) Errors() <-chan error {
	if q == nil {
		return nil
	}
	return q.errors
}

func (q *RedisQueue) DroppedErrors() uint64 {
	if q == nil {
		return 0
	}
	return q.errorsDropped.Load()
}

func (q *RedisQueue) reportError(err error) {
	if q == nil || err == nil {
		return
	}
	if q.errors == nil {
		q.errorsDropped.Add(1)
		return
	}
	select {
	case q.errors <- err:
	default:
		q.errorsDropped.Add(1)
	}
}

func (q *RedisQueue) ping(ctx context.Context) error { _, err := q.do(ctx, "PING"); return err }
func (q *RedisQueue) get(id string) (QueueJob, error) {
	value, err := q.do(context.Background(), "GET", q.jobKey(id))
	if err != nil {
		return QueueJob{}, err
	}
	text, ok := value.(string)
	if !ok || text == "" {
		return QueueJob{}, osErrNotExist{}
	}
	var job QueueJob
	if err := unmarshalRedisQueueJob([]byte(text), &job); err != nil {
		return QueueJob{}, err
	}
	if job.ID != id {
		return QueueJob{}, errors.New("Redis queue job ID does not match its key")
	}
	return job, nil
}

func unmarshalRedisQueueJob(data []byte, job *QueueJob) error {
	if err := json.Unmarshal(data, job); err != nil {
		return err
	}
	if job.CreatedAtUnixMilli > 0 {
		job.CreatedAt = time.UnixMilli(job.CreatedAtUnixMilli).UTC()
	}
	if job.AvailableAtUnixMilli > 0 {
		job.AvailableAt = time.UnixMilli(job.AvailableAtUnixMilli).UTC()
	}
	if job.LockedAtUnixMilli > 0 {
		locked := time.UnixMilli(job.LockedAtUnixMilli).UTC()
		job.LockedAt = &locked
	}
	if job.UpdatedAtUnixMilli > 0 {
		job.UpdatedAt = time.UnixMilli(job.UpdatedAtUnixMilli).UTC()
	}
	if job.LeaseUntilUnixMilli <= 0 {
		job.LeaseUntil = time.Time{}
		job.LeaseUntilUnixMilli = 0
		return nil
	}
	job.LeaseUntil = time.UnixMilli(job.LeaseUntilUnixMilli).UTC()
	return nil
}

func (q *RedisQueue) put(job QueueJob) error { //nolint:unused // compatibility/security surface retained for future adapter wiring
	data, _ := json.Marshal(job)
	_, err := q.do(context.Background(), "SET", q.jobKey(job.ID), string(data), "EX", "604800")
	return err
}

func (q *RedisQueue) moveDue(ctx context.Context, now time.Time) error {
	_, err := q.do(ctx, "EVAL", redisMoveDueScript, "5", q.delayedKey(), q.key("job:"), q.pendingKey(), q.leaseKey(), q.deadKey())
	return err
}

func (q *RedisQueue) reclaimExpired(ctx context.Context, now time.Time) error {
	_, err := q.do(ctx, "EVAL", redisReclaimScript, "6", q.leaseKey(), q.key("job:"), q.deadKey(), q.pendingKey(), q.key("sequence"), q.delayedKey())
	return err
}

func (q *RedisQueue) do(ctx context.Context, args ...string) (any, error) {
	timeout := q.timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", q.address)
	if err != nil {
		return nil, err
	}
	if deadline, ok := dialCtx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if q.tlsConfig != nil {
		tlsConn := tls.Client(conn, q.tlsConfig.Clone())
		if err := tlsConn.HandshakeContext(dialCtx); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("Redis TLS handshake failed: %w", err)
		}
		conn = tlsConn
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	if q.password != "" {
		if _, err := redisCommand(conn, reader, "AUTH", q.password); err != nil {
			return nil, err
		}
	}
	if q.database > 0 {
		if _, err := redisCommand(conn, reader, "SELECT", strconv.Itoa(q.database)); err != nil {
			return nil, err
		}
	}
	return redisCommand(conn, reader, args...)
}

func redisCommand(w io.Writer, r *bufio.Reader, args ...string) (any, error) {
	var builder strings.Builder
	builder.WriteString("*" + strconv.Itoa(len(args)) + "\r\n")
	for _, arg := range args {
		builder.WriteString("$" + strconv.Itoa(len(arg)) + "\r\n" + arg + "\r\n")
	}
	if _, err := io.WriteString(w, builder.String()); err != nil {
		return nil, err
	}
	return readRedis(r)
}

func readRedis(r *bufio.Reader) (any, error) {
	kind, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch kind {
	case '+':
		line, err := r.ReadString('\n')
		return strings.TrimSpace(line), err
	case '-':
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		return nil, errors.New(strings.TrimSpace(line))
	case ':':
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		return strconv.ParseInt(strings.TrimSpace(line), 10, 64)
	case '$':
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		size, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			return nil, err
		}
		if size < 0 {
			return nil, nil
		}
		data := make([]byte, size+2)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, err
		}
		return string(data[:size]), nil
	case '*':
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		count, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			return nil, err
		}
		if count < 0 {
			return nil, nil
		}
		items := make([]any, count)
		for index := range items {
			items[index], err = readRedis(r)
			if err != nil {
				return nil, err
			}
		}
		return items, nil
	}
	return nil, fmt.Errorf("unsupported Redis response %q", kind)
}

type osErrNotExist struct{}

func (osErrNotExist) Error() string { return "redis job not found" }
