# Manual Testing Guide - Settings Cache

## Prerequisites
- Server must be running: `go run cmd/server/main.go`
- Database must be initialized with test data
- User must be logged in to access dashboard and settings

## Test Scenarios

### Scenario 1: Initial Cache Miss
**Goal**: Verify cache fetches from database on first request

**Steps**:
1. Restart the server
2. Navigate to http://localhost:8080/ (dashboard)
3. Check server logs

**Expected Result**:
- Dashboard loads successfully with settings data
- Server logs show: `[SettingsCache] CACHE MISS - Initial fetch from database`
- Server logs show: `[SettingsCache] Cache refreshed - ProLabore: X.XX, INSS: X.XX`

### Scenario 2: Cache Hit Within TTL
**Goal**: Verify cache serves data without DB queries when within 5-minute TTL

**Steps**:
1. After Scenario 1, reload the dashboard (F5) within 5 minutes
2. Check server logs

**Expected Result**:
- Dashboard loads successfully with same settings data
- Server logs show: `[SettingsCache] CACHE HIT - Serving cached data (age: Xs, TTL: 5m0s)`
- No database query logs

### Scenario 3: Cache Invalidation on Settings Update
**Goal**: Verify cache is invalidated when settings are updated

**Steps**:
1. Navigate to http://localhost:8080/settings
2. Update any setting (e.g., change Pro-Labore value)
3. Click "Save"
4. Check server logs
5. Reload the dashboard
6. Check server logs again

**Expected Result**:
- Settings update succeeds
- Server logs show: `[SettingsCache] CACHE INVALIDATED - Next request will fetch from database`
- Server logs show: `[SettingsCache] CACHE MISS - Initial fetch from database` (on next dashboard load)
- Dashboard displays updated values

### Scenario 4: TTL Expiration After 5 Minutes
**Goal**: Verify cache automatically refreshes after TTL expires

**Steps**:
1. Load dashboard (cache hit)
2. Wait 5+ minutes without any requests
3. Reload the dashboard
4. Check server logs

**Expected Result**:
- Dashboard loads successfully
- Server logs show: `[SettingsCache] CACHE EXPIRED - Refreshing from database (last fetch: 5mXs ago)`
- Server logs show: `[SettingsCache] Cache refreshed - ProLabore: X.XX, INSS: X.XX`

### Scenario 5: Concurrent Access Thread Safety
**Goal**: Verify cache handles concurrent requests correctly

**Steps**:
1. Open multiple browser tabs
2. Reload all tabs rapidly within 1 second
3. Check server logs

**Expected Result**:
- All tabs load successfully with consistent data
- Server logs show multiple cache hits
- Server logs might show: `[SettingsCache] CACHE HIT (double-check) - Another goroutine refreshed cache`
- Only one database query is made (if cache was expired)

## Success Criteria Checklist

- [ ] Scenario 1: Initial cache miss works correctly
- [ ] Scenario 2: Cache hits work within TTL
- [ ] Scenario 3: Cache invalidates on settings update
- [ ] Scenario 4: Cache expires and refreshes after TTL
- [ ] Scenario 5: Concurrent access is thread-safe
- [ ] No console errors in browser
- [ ] Dashboard displays correct settings data
- [ ] Settings page can update values
- [ ] Performance improvement observable (cache hits are instant)

## Performance Metrics

**Before Caching**:
- 3 database queries per request
- Every dashboard load queries: pro_labore, inss_ceiling, inss_rate

**After Caching**:
- 3 database queries per 5 minutes (or when settings updated)
- Subsequent requests within TTL serve from memory (sub-millisecond)
- **~99% reduction in database queries** for frequently accessed pages

## Troubleshooting

### Cache not working
- Check server logs for cache events
- Verify `settingsCacheService` is initialized in main.go
- Verify handlers receive cache service in constructor

### Settings not updating
- Check cache invalidation happens in Update() handler
- Verify server logs show "CACHE INVALIDATED" message
- Check database was actually updated

### Thread safety issues
- Check for race condition warnings: `go test -race ./...`
- Verify sync.RWMutex is used correctly
- Check server logs for double-checked locking pattern
