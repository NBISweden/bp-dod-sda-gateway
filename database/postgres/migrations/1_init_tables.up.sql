CREATE TABLE IF NOT EXISTS origin_dataset
(
  accession            TEXT PRIMARY KEY,
  rems_workflow_id     INT                      NOT NULL,
  rems_organisation_id TEXT                     NOT NULL,
  dataset_xml          XML                      NOT NULL,
  image_xml            XML                      NOT NULL,
  annotation_xml       XML,
  observation_xml      XML                      NOT NULL,
  observer_xml         XML,
  policy_xml           XML                      NOT NULL,
  sample_xml           XML                      NOT NULL,
  staining_xml         XML                      NOT NULL,
  registered_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT clock_timestamp()
);

CREATE TABLE IF NOT EXISTS dataset_image
(
  id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  dataset_accession TEXT REFERENCES origin_dataset (accession),
  alias             TEXT NOT NULL,
  CONSTRAINT dataset_image_unique_idx UNIQUE (dataset_accession, alias)
);
CREATE TABLE IF NOT EXISTS image_file
(
  id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  dataset_accession TEXT REFERENCES origin_dataset (accession),
  image_alias       TEXT        NOT NULL,

  accession         TEXT UNIQUE NOT NULL,
  FOREIGN KEY (dataset_accession, image_alias) REFERENCES dataset_image (dataset_accession, alias)
);

CREATE TABLE IF NOT EXISTS dod_dataset
(
  accession         TEXT PRIMARY KEY,
  requested_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT clock_timestamp(),
  requested_by_user TEXT                     NOT NULL,
  released_at       TIMESTAMP WITH TIME ZONE
);

CREATE TYPE METADATA_TYPE AS ENUM ('annotation','dataset','image','landing_page','observation','observer','organisation','policy','rems','sample','staining');

CREATE TABLE IF NOT EXISTS dod_dataset_metadata_file
(
  id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  dataset_accession TEXT REFERENCES dod_dataset (accession),
  type              METADATA_TYPE NOT NULL,
  accession         TEXT          NOT NULL,

  xml_content       XML           NOT NULL,

  CONSTRAINT dod_dataset_metadata_file_unique_idx UNIQUE (dataset_accession, type)
);

CREATE TABLE IF NOT EXISTS dod_dataset_created_from_origin
(
  id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  dod_accession    TEXT REFERENCES dod_dataset (accession),
  origin_accession TEXT REFERENCES origin_dataset (accession),
  CONSTRAINT dod_dataset_created_from_origin_unique_idx UNIQUE (dod_accession, origin_accession)
);

CREATE TABLE IF NOT EXISTS dod_image
(
  id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  dod_accession    TEXT REFERENCES dod_dataset (accession),
  origin_accession TEXT REFERENCES origin_dataset (accession),
  image_alias      TEXT NOT NULL,

  FOREIGN KEY (origin_accession, image_alias) REFERENCES dataset_image (dataset_accession, alias),
  CONSTRAINT dod_image_unique_idx UNIQUE (dod_accession, origin_accession, image_alias)
)
