# Image Async API

new-api can wrap the existing synchronous OpenAI-compatible image endpoints in a local asynchronous task flow. Upstream providers are still called synchronously by a background worker; clients get a local `task_id` immediately and poll new-api for the final result.

## Endpoints

### Create an async image generation task

```bash
curl -X POST "$BASE/v1/images/generations?async=true" \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: demo-001" \
  -d '{
    "model": "sora_image",
    "prompt": "cat",
    "size": "1024x1024",
    "n": 1
  }'
```

Response:

```json
{"code":"success","message":"","data":"task_xxx"}
```

### Create an async image edit task

The edit endpoint preserves the raw `multipart/form-data` body, including the original boundary, so multi-image uploads and `mask` fields can be replayed by the worker.

```bash
curl -X POST "$BASE/v1/images/edits?async=true&webhook=https://example.com/notify" \
  -H "Authorization: Bearer $KEY" \
  -H "Idempotency-Key: edit-001" \
  -F "model=gpt-image-1" \
  -F "prompt=add glasses" \
  -F "image=@./input.png"
```

### Query a task

```bash
curl "$BASE/v1/images/tasks/task_xxx" \
  -H "Authorization: Bearer $KEY"
```

The query endpoint only returns tasks owned by the current token user and only for `platform=sync-task`, `action=image-sync`.

Visible statuses are:

- `IN_PROGRESS` for `NOT_START`, `SUBMITTED`, `QUEUED`, and `IN_PROGRESS`
- `SUCCESS`
- `FAILURE`

Successful `data` is the upstream OpenAI-compatible image response body.

## Idempotency

`Idempotency-Key` or `X-Idempotency-Key` can be sent on submit requests. For the same user:

- same key and same request hash returns the original `task_id`
- same key and different request hash returns `409`

The request hash includes user id, method, path without `async`, behavior-affecting query parameters such as `webhook`, content type, and body SHA256. Authorization and API keys are not included.

## Webhook

Pass `webhook=https://example.com/notify` in the submit query string. new-api sends terminal `SUCCESS` or `FAILURE` payloads matching the query response shape.

Headers:

- `Content-Type: application/json`
- `X-NewAPI-Task-ID`
- `X-NewAPI-Timestamp`
- `X-NewAPI-Signature: sha256=<hmac>` when `IMAGE_ASYNC_WEBHOOK_SECRET` is set

Webhook delivery uses the existing worker/SSRF-protected HTTP path and retries with backoff.

## Billing

Submit performs a real pre-consume and stores the pre-consumed quota on the task. On failure, the worker refunds the task quota. On success, the worker finalizes the task and records usage; repeated compensation checks are guarded by `BillingFinalized` and `BillingRefunded` in private task data.

## Environment Variables

- `IMAGE_ASYNC_STORAGE_DIR`: request body storage root, default `/data/image-async`
- `IMAGE_ASYNC_ENABLED`: set to `false` to disable async submit routing, default enabled
- `IMAGE_ASYNC_WORKER_ENABLED`: set to `false` to disable the background worker, default enabled
- `IMAGE_ASYNC_MAX_BODY_MB`: maximum persisted request body size, default `64`
- `IMAGE_ASYNC_BODY_TTL_HOURS`: body file cleanup TTL, default `72`
- `IMAGE_ASYNC_TASK_TIMEOUT_SECONDS`: unfinished task timeout, default `600`
- `IMAGE_ASYNC_STALE_IN_PROGRESS_SECONDS`: stale `IN_PROGRESS` recovery window, default `900`
- `IMAGE_ASYNC_MAX_ATTEMPTS`: reserved for retry policy compatibility, default `3`
- `IMAGE_ASYNC_WORKER_INTERVAL_SECONDS`: worker loop interval, default `2`
- `IMAGE_ASYNC_WORKER_BATCH_SIZE`: worker scan batch size, default `10`
- `IMAGE_ASYNC_WORKER_CONCURRENCY`: global worker concurrency, default equals `IMAGE_ASYNC_WORKER_BATCH_SIZE`; the production compose example sets it to `100`
- `IMAGE_ASYNC_WORKER_QUEUE_SCAN_SIZE`: queued task candidate scan size before per-user concurrency filtering, default `max(100, batch_size * worker_concurrency * 10)`
- `IMAGE_ASYNC_USER_GROUP_CONCURRENCY`: per-user running task limit by user group, default `default:2,vip:5,svip:10`. Entries accept `group:limit` or `group=limit`, separated by commas or semicolons.
- `IMAGE_ASYNC_WEBHOOK_RETRY`: max webhook attempts, default `3`
- `IMAGE_ASYNC_WEBHOOK_SECRET`: optional HMAC secret for webhook signatures
- `IMAGE_ASYNC_WEBHOOK_TIMEOUT_SECONDS`: webhook HTTP timeout, default `10`
- `IMAGE_ASYNC_RESULT_STORAGE`: final image result storage mode, `passthrough` or `s3`, default `passthrough`
- `IMAGE_ASYNC_S3_PROVIDER`: optional provider label stored in private task metadata, for example `tencent-cos`
- `IMAGE_ASYNC_S3_ENDPOINT`: S3-compatible endpoint, for Tencent COS Hong Kong use `https://cos.ap-hongkong.myqcloud.com`
- `IMAGE_ASYNC_S3_REGION`: S3 signing region, for Tencent COS Hong Kong use `ap-hongkong`
- `IMAGE_ASYNC_S3_BUCKET`: bucket name, including Tencent APPID, for example `lhcos-8330f-1430163992`
- `IMAGE_ASYNC_S3_ACCESS_KEY_ID`: S3 access key id; for Tencent COS this is `SecretId`
- `IMAGE_ASYNC_S3_SECRET_ACCESS_KEY`: S3 secret access key; for Tencent COS this is `SecretKey`
- `IMAGE_ASYNC_S3_PUBLIC_BASE_URL`: public read base URL used in task results
- `IMAGE_ASYNC_S3_FORCE_PATH_STYLE`: set to `false` for modern Tencent COS buckets, default `false`
- `IMAGE_ASYNC_S3_OBJECT_PREFIX`: object key prefix, for example `newapi/images/`
- `IMAGE_ASYNC_CONVERT_B64_TO_URL`: convert upstream `b64_json` image results to object URLs, default `true`
- `IMAGE_ASYNC_CACHE_UPSTREAM_URL`: download upstream image URLs and cache them to S3/COS, default `true`
- `IMAGE_ASYNC_RESULT_MAX_MB`: maximum final image download/decode/upload size, default `64`

## Result Image Storage

`IMAGE_ASYNC_STORAGE_DIR` is only the local request-body storage root used by the async worker to replay the original request. It is not the storage location for final generated images.

With `IMAGE_ASYNC_RESULT_STORAGE=passthrough`, successful task data keeps the upstream response as-is:

- upstream `url` results are stored as upstream URLs
- upstream `b64_json` results are stored as `b64_json`
- no S3/COS configuration is required

With `IMAGE_ASYNC_RESULT_STORAGE=s3`, successful async image results are normalized before they are written to `task.data`:

- upstream `b64_json` is decoded, validated as PNG/JPEG/WebP, uploaded to S3-compatible storage, replaced with `data[].url`, and cleared from `data[].b64_json`
- upstream `url` is downloaded with SSRF protection and cached to S3/COS when `IMAGE_ASYNC_CACHE_UPSTREAM_URL=true`
- when `IMAGE_ASYNC_CACHE_UPSTREAM_URL=false`, upstream URLs are kept without download or upload
- the returned URL is `{IMAGE_ASYNC_S3_PUBLIC_BASE_URL}/{object_key}`
- object keys are generated by the system as `newapi/images/YYYY/MM/DD/{task_id}_{index}.{ext}` by default

Upload failures, unsafe upstream URLs, non-image responses, unsupported image formats, or size-limit violations fail the task and trigger the normal async refund/webhook failure flow. Secrets are not written to `task.data`, task private metadata, or logs.

### Tencent COS Public-Read Private-Write Example

For a Tencent COS bucket configured as public read and private write, use public URLs and keep writes authenticated:

```env
IMAGE_ASYNC_RESULT_STORAGE=s3
IMAGE_ASYNC_S3_PROVIDER=tencent-cos
IMAGE_ASYNC_S3_ENDPOINT=https://cos.ap-hongkong.myqcloud.com
IMAGE_ASYNC_S3_REGION=ap-hongkong
IMAGE_ASYNC_S3_BUCKET=lhcos-8330f-1430163992
IMAGE_ASYNC_S3_ACCESS_KEY_ID=your-secret-id
IMAGE_ASYNC_S3_SECRET_ACCESS_KEY=your-secret-key
IMAGE_ASYNC_S3_FORCE_PATH_STYLE=false
IMAGE_ASYNC_S3_PUBLIC_BASE_URL=https://lhcos-8330f-1430163992.cos.ap-hongkong.myqcloud.com
IMAGE_ASYNC_S3_OBJECT_PREFIX=newapi/images/
IMAGE_ASYNC_CONVERT_B64_TO_URL=true
IMAGE_ASYNC_CACHE_UPSTREAM_URL=true
IMAGE_ASYNC_RESULT_MAX_MB=64
```

Tencent COS maps S3 `AccessKeyId` to Tencent `SecretId`, and S3 `SecretAccessKey` to Tencent `SecretKey`. Buckets created in 2024 or later should generally use virtual-hosted-style access, so `IMAGE_ASYNC_S3_FORCE_PATH_STYLE=false`.

This release does not implement signed result URLs. If the bucket is changed back to private read/write later, signed URL support should be added separately. Public-read images may generate internet egress cost; consider a lifecycle rule for `newapi/images/` such as 7 or 30 days, configure hotlink protection for business domains, monitor public egress traffic, and bind a custom COS domain for production browser preview if Tencent's default bucket domain is not suitable.

## Deployment Notes

Request bodies are stored on local disk by default. In multi-node deployments, run the image async worker only on nodes that can read `IMAGE_ASYNC_STORAGE_DIR`, or configure that directory as shared storage. Stale `IN_PROGRESS` tasks are failed and refunded instead of blindly requeued, which avoids duplicate upstream image generation if the original worker is still running. Use `IMAGE_ASYNC_RESULT_STORAGE=s3` when large final images should be kept out of the database and returned as object-storage URLs.
