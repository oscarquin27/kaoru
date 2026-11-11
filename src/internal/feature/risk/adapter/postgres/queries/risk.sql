-- name: NextCode :one
INSERT INTO code_sequence (prefix, year, last_value)
VALUES ($1, $2, 1)
ON CONFLICT (prefix, year) DO UPDATE SET last_value = code_sequence.last_value + 1
RETURNING last_value;

-- name: GetRiskByRequirement :one
SELECT r.id, r.code, r.requirement_id, r.level, r.justification, r.assessed_by,
       u.display_name AS assessed_by_name, r.created_at, r.updated_at
FROM risk_assessment r
         JOIN app_user u ON u.id = r.assessed_by
WHERE r.requirement_id = $1;

-- name: ListCharacteristics :many
SELECT characteristic
FROM risk_quality_characteristic
WHERE risk_assessment_id = $1
ORDER BY characteristic;

-- name: InsertRisk :exec
INSERT INTO risk_assessment (id, code, requirement_id, level, justification, assessed_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateRisk :execrows
UPDATE risk_assessment
SET level         = $2,
    justification = $3,
    assessed_by   = $4,
    updated_at    = $5
WHERE id = $1;

-- name: DeleteCharacteristics :exec
DELETE
FROM risk_quality_characteristic
WHERE risk_assessment_id = $1;

-- name: InsertCharacteristic :exec
INSERT INTO risk_quality_characteristic (risk_assessment_id, characteristic)
VALUES ($1, $2);
