-- +goose Up
-- Un RISK vigente por REQ. Una reevaluación actualiza la fila y queda en auditoría.
CREATE TABLE risk_assessment (
    id             uuid PRIMARY KEY,
    code           text        NOT NULL UNIQUE,
    requirement_id uuid        NOT NULL UNIQUE REFERENCES requirement (id) ON DELETE RESTRICT,
    level          text        NOT NULL CHECK (level IN ('high', 'medium', 'low')),
    justification  text        NOT NULL,
    assessed_by    text        NOT NULL REFERENCES app_user (id) ON DELETE RESTRICT,
    created_at     timestamptz NOT NULL,
    updated_at     timestamptz NOT NULL
);

-- Características de calidad de ISO/IEC 25010 en juego.
CREATE TABLE risk_quality_characteristic (
    risk_assessment_id uuid NOT NULL REFERENCES risk_assessment (id) ON DELETE CASCADE,
    characteristic     text NOT NULL CHECK (characteristic IN (
        'functional_suitability', 'performance_efficiency', 'compatibility', 'usability',
        'reliability', 'security', 'maintainability', 'portability')),
    PRIMARY KEY (risk_assessment_id, characteristic)
);

-- +goose Down
DROP TABLE risk_quality_characteristic;
DROP TABLE risk_assessment;
