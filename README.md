# Sasha KV: In Memory Key-Value Store/Cache

sasha-kv is a simple in-memory cache that uses simple key-value store similar to [MemCached](https://memcached.org) or [Redis](https://redis.io)
sasha-kv is also persistent and durable with disk backup mechanism.

Any object will be decoded as string (use case: json or xml data) and can be stored for a given duration or forever.
The cache will cleanup and purge any expired items automatically.
It is thread-safe and supports concurrent reads and write access.


