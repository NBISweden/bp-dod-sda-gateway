# BigPicture Dataset On Demand Sensitive Data Archive Gateway (bp-dod-sda-gateway)

The bp-dod-sda-gateway creates **On Demand Datasets** in the BigPicture project: a user picks
individual images from one or more published BigPicture datasets, and the gateway makes them
available as a new dataset in the Sensitive Data Archive (SDA), with its own accession, metadata
and REMS catalogue item. The user can then apply for access to the dataset through REMS.

## Design

### The idea

A dataset in SDA is a list of archived files. The image files of a published dataset are already
archived, so a new dataset can reuse them. To create an On Demand Dataset the gateway does not copy
any image data: it writes a new, smaller set of metadata XML files that describe only the selected
images, uploads and ingests those, and asks SDA to create a dataset of the new metadata files plus
the existing image files.

### Overview

```
                        ┌────────────────── bp-dod-sda-gateway ──────────────────┐
 metadata-submitter     │                                                        │
 sync API ─────────────▶│ 1. Origin dataset sync ───────▶ Postgres database      │
                        │    (timer, or admin API)        (origin metadata,      │
                        │                                  images, file accessions,
 client with a user's   │                                  On Demand Datasets)   │
 bearer token ─────────▶│ 2. Dataset On Demand API ──┐                           │
                        │                            ▼                           │
                        │ 3. On Demand Dataset creation (background) ────────────┼──▶ SDA s3inbox
                        │                                                        ├──▶ SDA API
                        │                                                        ├──▶ REMS
                        │                                                        └──▶ SMTP server
                        └────────────────────────────────────────────────────────┘
```

### 1. Origin dataset sync

Code: [origin_dataset_sync](origin_dataset_sync).

Every `sync.interval` (default 1 hour), the gateway asks the
[metadata-submitter](https://github.com/CSCfi/metadata-submitter) sync API which datasets have been
published since the last sync. The point in time it last synced up to (the cursor) is stored in the
database. Requests are authenticated with a token that the gateway signs with `sync.private_key`.

For each dataset published since last sync, it downloads a zip from the sync API with the metadata
XML files and a file listing the dataset's files with their SDA file accessions. It stores these as
an origin dataset, and records which image consists of which archived files.

A dataset needs to fulfil the following, otherwise it will be skipped (with a log message):

- its `policy.xml` has exactly one policy, with a `type_of_dataset` that contains `Non-Clinical`;
- it has a `rems.xml` with a REMS workflow and organisation;
- it has `dataset.xml`, `image.xml`, `observation.xml`, `sample.xml` and `staining.xml`.

A sync can also be started through the private `OriginDatasetSyncService` API, for all new datasets
or for a single dataset (see [APIs](#apis)).

### 2. Requesting an On Demand Dataset

Code: [dataset_on_demand_service](dataset_on_demand_service).

A client calls the `RequestDatasetCreation` endpoint with a list of image accessions and an optional
label, using the user's bearer token. For example, with the Connect protocol over HTTP and JSON:

```bash
curl --request POST "$DOD_URL/dataset_on_demand.DatasetOnDemandService/RequestDatasetCreation" \
     --header "Authorization: Bearer $TOKEN" \
     --header "Content-Type: application/json" \
     --data '{"imageAccessions": ["<image accession>", "<image accession>"], "label": "my selection"}'
```

```json
{"onDemandDatasetAccession": "<on demand dataset accession>"}
```

The gateway:

1. **Validates the request.** Every image must belong to a synced origin dataset, and an image may
   only be listed once. Images from several origin datasets can be combined only if the datasets have
   the same REMS workflow (the same data access committee) and the same terms of use (`policy.xml`).
2. **Reuses existing datasets.** The set of image accessions is hashed. If an On Demand Dataset with
   exactly these images already exists, the user is added as a requester of that dataset, and gets
   the notification email directly if it is already released.
3. **Builds the metadata.** It writes new metadata for the selected images and the objects they
   refer to (samples, observations, observers, staining), taken from the origin datasets, under a new
   dataset accession. Aliases (names used by the original dataset submitter) are replaced with
   accessions, so they stay unique when datasets are combined.
4. **Stores the dataset and the request**, including the user's email address from the
   `dod_service.userinfo_endpoint`, and triggers the creation in SDA.

The response contains the On Demand Dataset accession. The endpoint `GetOnDemandDatasetStatus`
returns whether a dataset is still being created or has been released, and `ListMyOnDemandDatasets`
lists the user's own requests (see [Dataset On Demand endpoints](#dataset-on-demand-endpoints)).

### 3. Creating the dataset in SDA

Code: [on_demand_dataset_creation_handler](on_demand_dataset_creation_handler).

This runs in the background, one goroutine per dataset. The gateway uses one service identity
towards SDA: the subject of `inbox.token`.

1. **Upload.** The metadata XML files (`dataset`, `image`, `observation`, `observer`, `policy`,
   `sample`, `staining`) are encrypted with the crypt4gh key in `inbox.c4gh_public_key_file_path` and
   uploaded to the s3inbox under `<dataset accession>/METADATA/`.
2. **Ingest and accession.** The gateway triggers ingestion of each file through the SDA API, polls
   until the file is verified, and assigns it an accession.
3. **Create the dataset.** It calls `dataset/create` with the new metadata files and the existing
   image files, each with its path in the dataset (`METADATA/<type>.xml.c4gh`,
   `IMAGES/IMAGE_<image accession>/<file>.c4gh`), in batches of `sda_api.dataset_create_batch_size`.
4. **Release.** It polls until all files are in the dataset, releases it, and polls until SDA reports
   it as released.
5. **REMS.** It creates a REMS resource and catalogue item for the dataset, using the workflow and
   organisation from the origin dataset's `rems.xml`. Existing ones are reused.
6. **Notification.** If `email.rems_notification` is set, it emails everyone who requested the
   dataset that it can now be applied for in REMS.

All SDA steps poll every `sda_api.poll_interval` (default 5 minutes). At startup, the gateway resumes
monitoring every On Demand Dataset that has not been released yet.

### Current limitations

- Only metadata standard version 2.0 is supported.
- Only non-clinical origin datasets are synced.
- On Demand Datasets have no annotation or landing page metadata, and no DOI.
- Changes to an origin dataset after an On Demand Dataset has been created from it do not affect that
  On Demand Dataset.

## Integrations

The gateway needs these services, which it reaches through the configured URLs:

| Service | Used for |
|---|---|
| [metadata-submitter](https://github.com/CSCfi/metadata-submitter) sync API | Origin dataset metadata and SDA file accessions |
| [SDA s3inbox](https://github.com/neicnordic/sensitive-data-archive/blob/main/sda/cmd/s3inbox/s3inbox.md) | Uploading the generated metadata files |
| [SDA API](https://github.com/neicnordic/sensitive-data-archive/blob/main/sda/cmd/api/api.md) | Ingesting and accessioning the metadata files, and creating and releasing the dataset |
| [REMS](https://github.com/CSCfi/rems/) | Resources and catalogue items for the On Demand Datasets. The workflows and organisations must already exist. |
| OIDC provider | The JWK set for verifying bearer tokens, and the userinfo endpoint for the user's email address |
| SMTP server | Notification emails (optional) |
| Postgres | The gateway's own database. The gateway creates and migrates its tables at startup ([migrations](database/postgres/migrations)). |

## APIs

The gateway serves two [connect-rpc](https://connectrpc.com/) services:

| Service | Definition | Port flag (default) | Authentication |
|---|---|---|---|
| `DatasetOnDemandService` (public) | [dataset_on_demand_service.proto](proto/public/v1/dataset_on_demand_service.proto) | `dod_service.port` (8080) | Bearer token, verified against `auth.jwt_pub_key_url`, with an audience in `auth.allowed_audiences` |
| `OriginDatasetSyncService` (admin) | [origin_dataset_sync_service.proto](proto/private/v1/origin_dataset_sync_service.proto) | `sync.service_port` (8081) | None: do not expose this port outside the deployment |

### Dataset On Demand endpoints

Each endpoint is served at `POST /dataset_on_demand.DatasetOnDemandService/<endpoint>`, over the
Connect, gRPC and gRPC-Web protocols, and needs a bearer token. A request without a valid token
gets `unauthenticated`. The field names below are the JSON names.

| Endpoint | Request | Response | Errors |
|---|---|---|---|
| `RequestDatasetCreation` | `imageAccessions` (list of image accessions), `label` (optional text for the user's own reference) | `onDemandDatasetAccession` | `invalid_argument`: no images, an image listed twice, or images from datasets with different REMS workflows or terms of use. `not_found`: an image is not in any synced origin dataset. `already_exists`: the user has already requested a dataset with exactly these images. |
| `GetOnDemandDatasetStatus` | `onDemandDatasetAccession` | `status`: `STATUS_CREATING` or `STATUS_RELEASED` | `invalid_argument`: no accession. `not_found`: no such dataset. |
| `ListMyOnDemandDatasets` | (empty) | `onDemandDatasets`: list of `accession`, `status`, `requestedAt`, `label` for the datasets the user has requested | |

All endpoints can also return `internal`.

The generated code is in [grpc_gen](grpc_gen). To regenerate it with [buf](https://buf.build/):

```bash
buf generate
```

## Configuration

Each option can be set as a flag, as an environment variable, or in a YAML config file. The
environment variable is the option name in upper case, with `.` and `-` replaced by `_`: for example
`sync.interval` is `SYNC_INTERVAL`. The config file is `config.yaml` in the directory given by
`--config-path` (default `.`), or the file given by `--config-file`.

Durations use Go's [time.ParseDuration](https://pkg.go.dev/time#ParseDuration) format, such as `5m`
or `1h`.

| Option | Type | Required | Default | Description |
|---|---|---|---|---|
| `config-file` | string | | | Direct path to the config file |
| `config-path` | string | | `.` | Directory to look for `config.yaml` in |
| `log.level` | string | | `INFO` | Log level: `PANIC`, `FATAL`, `ERROR`, `WARN`, `INFO`, `DEBUG` or `TRACE` |
| **Public API and authentication** | | | | |
| `dod_service.port` | int | | `8080` | Port for the public `DatasetOnDemandService` |
| `dod_service.userinfo_endpoint` | string | yes | | OIDC userinfo endpoint, used to get the user's email address |
| `auth.jwt_pub_key_url` | string | yes | | URL of the JWK set used to verify bearer tokens |
| `auth.jwt_key_set_cache_ttl` | duration | | `1h` | How long to cache the JWK set |
| `auth.allowed_audiences` | string list | yes | | Comma-separated list of accepted audiences; a token must contain at least one |
| **Origin dataset sync** | | | | |
| `sync.metadata_submitter_sync_url` | string | yes | | URL of the metadata-submitter sync API |
| `sync.private_key` | string | yes | | Private key used to sign tokens for the sync API (ES256) |
| `sync.interval` | duration | | `1h` | How often to check for newly published datasets |
| `sync.service_port` | int | | `8081` | Port for the admin `OriginDatasetSyncService` |
| **SDA** | | | | |
| `inbox.endpoint_url` | string | yes | | URL of the s3inbox |
| `inbox.token` | string | yes | | Token for the s3inbox. Its subject is the user that SDA files are created as. |
| `inbox.c4gh_public_key_file_path` | string | yes | | Path to the crypt4gh public key used to encrypt uploaded files |
| `inbox.disable_https` | bool | | `false` | Set if `inbox.endpoint_url` does not use HTTPS |
| `sda_api.url` | string | yes | | URL of the SDA API |
| `sda_api.poll_interval` | duration | | `5m` | How often to poll the SDA API while creating a dataset |
| `sda_api.dataset_create_batch_size` | int | | `1000` | Number of files in each `dataset/create` request |
| **REMS** | | | | |
| `rems.url` | string | yes | | URL of REMS |
| `rems.user` | string | yes | | User sent in the `x-rems-user-id` header |
| `rems.key` | string | yes | | API key sent in the `x-rems-api-key` header |
| `rems.category_id` | int | | `0` | REMS category for new catalogue items; `0` means no category |
| `rems.demo.workflow_id` | int | | `-1` | For demo instances: workflow ID to use instead of the origin dataset's |
| `rems.demo.organisation_id` | string | | | For demo instances: organisation ID to use instead of the origin dataset's |
| **Email** | | | | |
| `email.rems_notification` | bool | | `false` | Send notification emails when a dataset is released and registered in REMS |
| `email.stmp_host` | string | | | SMTP host |
| `email.stmp_port` | int | | `587` | SMTP port |
| `email.address` | string | | | Address that notification emails are sent from, also used as the SMTP username |
| `email.password` | string | | | Password for `email.address` |
| **Database** | | | | |
| `database.host` | string | yes | | Postgres host |
| `database.port` | int | | `5432` | Postgres port |
| `database.user` | string | yes | | Database user |
| `database.password` | string | yes | | Database password |
| `database.name` | string | | `bp_dod` | Database name |
| `database.schema` | string | | | Schema to use as search path |
| `database.ssl_mode` | string | | `prefer` | SSL mode |
| `database.ca_cert` | string | | | CA certificate |
| `database.client_cert` | string | | | Client certificate |
| `database.client_key` | string | | | Key for the client certificate |
| `database.max_idle_connections` | int | | `2` | Maximum idle connections; `<= 0` keeps none |
| `database.max_open_connections` | int | | `20` | Maximum open connections; `<= 0` means unlimited |
| `database.connection_max_idle_time` | duration | | `5m` | Maximum idle time for a connection; `<= 0` means unlimited |
| `database.connection_max_life_time` | duration | | `0s` | Maximum lifetime of a connection; `<= 0` means unlimited |

Tracing and logging are set up with OpenTelemetry; see
[pkg/observability](pkg/observability/README.md).

## Development

Run the tests:

```bash
go test ./...
```

Build the container image:

```bash
docker build -t bp-dod-sda-gateway .
```
