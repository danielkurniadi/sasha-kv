# Sasha KV: In Memory Key-Value Store/Cache

sasha-kv is an in-memory cache that uses simple key-value store similar to [MemCached](memcached.org).
sasha-kv is also persistent and durable.

Any object can be stored for a given duration or forever. The cache will cleanup and purge any expired items automatically for you.
The cache is thread-safe and supports concurrent reads and write access at the same time.


