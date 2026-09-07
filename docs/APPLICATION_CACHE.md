# Application cache analysis

This report records the codebase analysis completed before application caching
was added. The review covered HTTP controllers and middleware, services, models,
the GORM database setup, cache adapters, authentication and token lifecycle,
user/device ownership, heartbeats, audit connection tracking, and process
monitoring. This project has no separate repository package; database access is
performed in services and a small number of controllers.

## Accepted cache locations

| Data | Key | TTL | Invalidation | Risk |
| --- | --- | --- | --- | --- |
| Latest public app release | `rustdesk-api:v1:app-release:latest:{all,windows,linux,macos,android}` | 5 minutes | Delete every fixed alias key after successful create, update, delete, or enable-status change | Low |
| Public enabled client downloads | `rustdesk-api:v1:client-downloads:active` | 5 minutes | Delete after successful create, update, delete, or enable-status change | Low |
| Public active announcements | `rustdesk-api:v1:announcements:active` | 2 minutes | Delete after a successful create transaction, update, or delete | Low |

These values are public, globally identical, mostly static, and have centralized
write paths. Empty latest-release results are not cached. Arbitrary platform
strings retain their existing exact database lookup and are not cached, which
also prevents unbounded key creation.

All three integrations use cache-aside reads. Cache misses, timeouts, invalid
JSON, and other adapter failures fall back to GORM. Cache set and delete errors
are logged and never become API errors. Database mutations happen before cache
invalidation.

## Rejected cache locations

| Data or flow | Reason | Risk |
| --- | --- | --- |
| Users, passwords, MFA, tokens, JWT decisions, subscriptions, roles, permissions | Disablement, expiry, revocation, fingerprint checks, and payment activation must be authoritative immediately | Critical |
| Peers, device ownership, address books, tags, user groups, device groups | Authorization and ownership depend on several independently changing tables | High |
| Heartbeat strategy delivery | A result depends on peer ownership, device group, address-book tags, and strategy priority; stale policy changes device behavior | High |
| Process-monitor rules and status | Device-specific configuration and live reports have multiple mutation paths | High |
| Online state, audit heartbeats, relay/rendezvous or socket state | Real-time and process-local by design | Critical |
| OAuth provider/configuration and login options | Security-sensitive configuration used during authentication | High |
| Orders, invite codes, login logs, share records | User-scoped, mutable, or one-time state | Critical |
| Dashboard aggregates | Expensive, but peer online counts and current-day audit counts change continuously | High |
| Server probes | Results are live network state | High |
| Server version and subscription plan definitions | Already process-memory/file configuration; Redis provides no database benefit | None |

No user-scoped object is cached, so there is no user or tenant cache-key boundary
to get wrong. If such caching is introduced later, the key must include the
owner/tenant identifier and every ownership and permission mutation must be
part of its invalidation design.
