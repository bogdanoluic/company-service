ALTER TABLE companies
    ADD COLUMN version BIGINT NOT NULL DEFAULT 1;

ALTER TABLE companies
    ADD CONSTRAINT companies_version_positive
    CHECK (version > 0);

---- create above / drop below ----

ALTER TABLE companies
    DROP CONSTRAINT companies_version_positive;

ALTER TABLE companies
    DROP COLUMN version;
