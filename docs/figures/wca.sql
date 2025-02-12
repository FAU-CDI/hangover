-- A WissKI Column Archive (wca) is a single sqlite file that holds a set of triples representing values for each path.
-- "Column" here refers to a tabular view of a WissKI Dataset, with each Field and it's values forming a column.
-- A WCA is represented as a single SQLite database (typically ending in ".wca.sqlite"), and SHOULD NOT be updated once created. 

CREATE TABLE `wca_manifest`
-- WissKI Column Archive Manifest
(	
    Description TEXT,                                               -- Human-readable description of this archive
    Created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,                    -- Time when index creation started
    Pathbuilder BLOB NOT NULL,                                      -- XML (utf8-bytes) of Pathbuilder(s) used in this archive.

    Version TEXT PRIMARY KEY                                        -- Manifest Version
    CHECK (Version = "1.0")                                         -- Should be "1.0"
);


CREATE TABLE `wca_columns`
-- Primary columns data
(
    `Path` TEXT,                                                    -- Machine name of the path that this column belongs to 
    `URI` TEXT,                                                     -- Canonical Entity URI
    `Value` BLOB                                                    -- JSON-encoded value for this column. See go 'value.go' for exact struct.
);

CREATE INDEX `wca_path` ON `wca_columns` (`Path`);                  -- lookup all entities for a given path
CREATE INDEX `wca_uri` ON `wca_columns` (`Path`, `URI`);            -- find value for specific uri


CREATE TABLE `wca_sameas`
-- Equivalences of underlying URIs
(
    `Alias` TEXT UNIQUE NOT NULL,                                   -- Non-canonical URI that has an alias
    `Canonical` TEXT NOT NULL                                       -- Matching canonical URI
);

CREATE UNIQUE INDEX `wca_canonical` ON `wca_sameas` (`Alias`);      -- Lookup a canonical URI 
CREATE UNIQUE INDEX `wca_aliasof` ON `wca_sameas` (`Canonical`);    -- Find aliases of a specific URI