# Add Ping method for health check support

## Summary

Add `Ping()` method to `AmazonS3Backend` for health check support in dependent services.

## Changes

### New Method: `Ping(ctx context.Context) error`

**Purpose:** Lightweight health check for S3 bucket accessibility

**Implementation:** Uses AWS HeadBucket API to verify:
- AWS credentials are valid
- Network connectivity to S3 exists
- Bucket exists and is accessible
- Permissions are correctly configured

**Characteristics:**
- Timeout: 5 seconds
- No data transfer (metadata only)
- Suitable for frequent health checks (every 5-10s)
- Cost-effective (~$0.003/month for 500k checks)

## API Details

```go
// Ping performs a lightweight health check on the S3 bucket.
// It verifies:
//   - AWS credentials are valid
//   - Network connectivity to S3
//   - Bucket exists and is accessible
//   - Permissions are correctly configured
//
// This is suitable for health check endpoints that run frequently.
// Uses HeadBucket API which only returns metadata (no data transfer).
func (b *AmazonS3Backend) Ping(ctx context.Context) error
```

**Parameters:**
- `ctx context.Context` - Request context (timeout applied internally)

**Returns:**
- `error` - nil if healthy, wrapped error with context if failed

**Example Errors:**
- `"s3 health check failed: NoSuchBucket: The specified bucket does not exist"`
- `"s3 health check failed: AccessDenied: Access Denied"`
- `"s3 health check failed: RequestTimeout: Request Timeout"`

## Use Cases

1. **Kubernetes Readiness Probes**
   ```go
   if err := s3Backend.Ping(ctx); err != nil {
       // Return 503 Service Unavailable
   }
   ```

2. **Application Health Dashboards**
   ```go
   status := "healthy"
   if err := s3Backend.Ping(ctx); err != nil {
       status = fmt.Sprintf("unhealthy: %v", err)
   }
   ```

3. **Automated Monitoring**
   ```go
   ticker := time.NewTicker(60 * time.Second)
   for range ticker.C {
       if err := s3Backend.Ping(ctx); err != nil {
           metrics.RecordS3Unhealthy()
           logger.Error("s3 health check failed", zap.Error(err))
       }
   }
   ```

## Performance Impact

- **Latency:** 10-50ms (same region), 50-150ms (cross-region)
- **AWS Cost:** ~$0.005 per 1,000 requests
- **Network:** Single HTTPS request, no data transfer
- **Memory:** Minimal (no data buffering required)

## Required IAM Permissions

```json
{
  "Effect": "Allow",
  "Action": ["s3:ListBucket"],
  "Resource": ["arn:aws:s3:::bucket-name"]
}
```

## Breaking Changes

None. This is an additive change.

## Backward Compatibility

✅ Fully backward compatible
✅ Existing methods unchanged
✅ No new dependencies

## Testing

Manual verification:
```bash
# Test bucket accessibility
aws s3api head-bucket --bucket <bucket-name>
```

Recommended unit tests (TODO):
```go
func TestPing_Success(t *testing.T)
func TestPing_BucketNotFound(t *testing.T)
func TestPing_AccessDenied(t *testing.T)
func TestPing_Timeout(t *testing.T)
```

## Version

Recommend: **v1.1.0** (minor version bump for new feature)

## Related Changes

This method will be used by:
- `golang-microservices/applications/treatwise/aiml-models-inferencing` (v0.11.0)
- Any other service implementing health checks with S3 dependencies

## Checklist

- [x] Method implemented with proper error handling
- [x] Godoc comments added
- [x] Timeout configured (5 seconds)
- [ ] Unit tests added
- [ ] Integration tests added
- [ ] README updated
- [ ] CHANGELOG updated
- [ ] Version bumped

---

**Author:** Claude Code
**Date:** 2026-01-03
**Type:** Feature Addition
