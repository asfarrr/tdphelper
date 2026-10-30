CREATE TABLE materials (
    id                   INTEGER PRIMARY KEY,
    name                 TEXT    NOT NULL UNIQUE,
    machine_kop_per_hour INTEGER NOT NULL CHECK (machine_kop_per_hour >= 0),
    failure_bp           INTEGER NOT NULL CHECK (failure_bp >= 0 AND failure_bp < 10000),
    hidden               INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE filaments (
    id               INTEGER PRIMARY KEY,
    material_id      INTEGER NOT NULL REFERENCES materials (id),
    color_name       TEXT    NOT NULL,
    color_hex        TEXT    NOT NULL,
    brand            TEXT    NOT NULL,
    price_kop_per_kg INTEGER NOT NULL CHECK (price_kop_per_kg >= 0),
    low_stock_grams  INTEGER NOT NULL CHECK (low_stock_grams >= 0),
    hidden           INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE spools (
    id                INTEGER PRIMARY KEY,
    filament_id       INTEGER NOT NULL REFERENCES filaments (id),
    initial_grams     INTEGER NOT NULL CHECK (initial_grams > 0),
    empty_spool_grams INTEGER NOT NULL CHECK (empty_spool_grams >= 0),
    remaining_grams   INTEGER NOT NULL CHECK (remaining_grams >= 0),
    status            TEXT    NOT NULL CHECK (status IN ('active', 'empty'))
);

CREATE INDEX spools_filament_id ON spools (filament_id);

-- order_id gets a foreign key once orders exist (stage 2).
CREATE TABLE stock_movements (
    id          INTEGER PRIMARY KEY,
    spool_id    INTEGER NOT NULL REFERENCES spools (id),
    delta_grams INTEGER NOT NULL,
    kind        TEXT    NOT NULL CHECK (kind IN ('purchase', 'consume', 'waste', 'personal', 'adjust')),
    order_id    INTEGER,
    note        TEXT    NOT NULL,
    created_at  TEXT    NOT NULL
);

CREATE INDEX stock_movements_spool_id ON stock_movements (spool_id);

CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
