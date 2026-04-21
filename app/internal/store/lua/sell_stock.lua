-- sell_stock.lua
-- KEYS[1] = bank:stocks
-- KEYS[2] = wallet:{id}:stocks
-- KEYS[3] = bank:stock_names
-- ARGV[1] = stock_name

local stock_name = ARGV[1]

-- Check if stock exists in the bank's registry
local exists = redis.call('SISMEMBER', KEYS[3], stock_name)
if exists == 0 then
    return 'STOCK_NOT_FOUND'
end

-- Check if wallet has stock to sell
local qty = tonumber(redis.call('HGET', KEYS[2], stock_name) or '0')
if qty <= 0 then
    return 'WALLET_OUT_OF_STOCK'
end

-- Atomically transfer: wallet -1, bank +1
redis.call('HINCRBY', KEYS[2], stock_name, -1)
redis.call('HINCRBY', KEYS[1], stock_name, 1)

return 'OK'
