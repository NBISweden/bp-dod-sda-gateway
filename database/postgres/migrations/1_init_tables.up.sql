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
  accession         TEXT NOT NULL PRIMARY KEY,
  dataset_accession TEXT NOT NULL REFERENCES origin_dataset (accession)
);
CREATE TABLE IF NOT EXISTS image_file
(
  accession         TEXT NOT NULL PRIMARY KEY,
  dataset_accession TEXT NOT NULL REFERENCES origin_dataset (accession),
  image_accession   TEXT NOT NULL REFERENCES dataset_image (accession),
  base_file_name    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS on_demand_dataset
(
  accession         TEXT PRIMARY KEY,
  requested_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT clock_timestamp(),
  requested_by_user TEXT                     NOT NULL,
  released_at       TIMESTAMP WITH TIME ZONE
);

CREATE TYPE METADATA_TYPE AS ENUM ('annotation','dataset','image','landing_page','observation','observer','organisation','policy','rems','sample','staining');

CREATE TABLE IF NOT EXISTS on_demand_dataset_metadata_file
(
  id                          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  on_demand_dataset_accession TEXT          NOT NULL REFERENCES on_demand_dataset (accession),
  type                        METADATA_TYPE NOT NULL,
  accession                   TEXT          NOT NULL,
  xml_content                 XML           NOT NULL,
  CONSTRAINT on_demand_dataset_metadata_file_unique_idx UNIQUE (on_demand_dataset_accession, type)
);

CREATE TABLE IF NOT EXISTS on_demand_dataset_created_from_origin
(
  id                          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  on_demand_dataset_accession TEXT NOT NULL REFERENCES on_demand_dataset (accession),
  origin_accession            TEXT NOT NULL REFERENCES origin_dataset (accession),
  CONSTRAINT on_demand_dataset_created_from_origin_unique_idx UNIQUE (on_demand_dataset_accession, origin_accession)
);

CREATE TABLE IF NOT EXISTS on_demand_dataset_image
(
  id                          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  on_demand_dataset_accession TEXT NOT NULL REFERENCES on_demand_dataset (accession),
  image_accession             TEXT NOT NULL REFERENCES dataset_image (accession),

  CONSTRAINT on_demand_dataset_image_unique_idx UNIQUE (on_demand_dataset_accession, image_accession)
);