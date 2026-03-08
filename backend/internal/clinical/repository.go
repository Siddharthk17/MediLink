package clinical

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// DrugCheckerRepository provides data access for drug interaction caching.
type DrugCheckerRepository interface {
	GetCachedInteraction(ctx context.Context, rxNormA, rxNormB string) (*DrugInteraction, error)
	CacheInteraction(ctx context.Context, interaction *DrugInteraction) error
	GetAllInteractionsForDrug(ctx context.Context, rxNormCode string) ([]*DrugInteraction, error)
	GetDrugClass(ctx context.Context, rxNormCode string) (string, error)
	StoreDrugClass(ctx context.Context, rxNormCode, drugName, drugClass string) error
}

// PostgresDrugCheckerRepository implements DrugCheckerRepository using PostgreSQL.
type PostgresDrugCheckerRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewDrugCheckerRepository creates a new PostgresDrugCheckerRepository.
func NewDrugCheckerRepository(db *sqlx.DB, logger zerolog.Logger) *PostgresDrugCheckerRepository {
	return &PostgresDrugCheckerRepository{db: db, logger: logger}
}

// GetCachedInteraction checks the local cache for a drug interaction.
// Returns nil, nil on cache miss.
func (r *PostgresDrugCheckerRepository) GetCachedInteraction(ctx context.Context, rxNormA, rxNormB string) (*DrugInteraction, error) {
	a, b := NormalizeDrugPair(rxNormA, rxNormB)

	var row struct {
		DrugARxNorm  string         `db:"drug_a_rxnorm"`
		DrugBRxNorm  string         `db:"drug_b_rxnorm"`
		DrugAName    string         `db:"drug_a_name"`
		DrugBName    string         `db:"drug_b_name"`
		Severity     string         `db:"severity"`
		Description  string         `db:"description"`
		Mechanism    sql.NullString `db:"mechanism"`
		ClinicalEff  sql.NullString `db:"clinical_effect"`
		Management   sql.NullString `db:"management"`
		Source       string         `db:"source"`
	}

	err := r.db.GetContext(ctx, &row,
		`SELECT drug_a_rxnorm, drug_b_rxnorm, drug_a_name, drug_b_name, severity, description, mechanism, clinical_effect, management, source
		 FROM drug_interactions WHERE drug_a_rxnorm = $1 AND drug_b_rxnorm = $2`, a, b)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // cache miss
		}
		return nil, fmt.Errorf("query cached interaction: %w", err)
	}

	return &DrugInteraction{
		DrugA:          MedicationInfo{RxNormCode: row.DrugARxNorm, Name: row.DrugAName},
		DrugB:          MedicationInfo{RxNormCode: row.DrugBRxNorm, Name: row.DrugBName},
		Severity:       InteractionSeverity(row.Severity),
		Description:    row.Description,
		Mechanism:      row.Mechanism.String,
		ClinicalEffect: row.ClinicalEff.String,
		Management:     row.Management.String,
		Source:         row.Source,
		Cached:         true,
	}, nil
}

// CacheInteraction stores a drug interaction result permanently.
func (r *PostgresDrugCheckerRepository) CacheInteraction(ctx context.Context, interaction *DrugInteraction) error {
	a, b := NormalizeDrugPair(interaction.DrugA.RxNormCode, interaction.DrugB.RxNormCode)
	nameA, nameB := interaction.DrugA.Name, interaction.DrugB.Name
	if a != interaction.DrugA.RxNormCode {
		nameA, nameB = interaction.DrugB.Name, interaction.DrugA.Name
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO drug_interactions (id, drug_a_rxnorm, drug_b_rxnorm, drug_a_name, drug_b_name, severity, description, mechanism, clinical_effect, management, source, last_verified)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 ON CONFLICT (drug_a_rxnorm, drug_b_rxnorm) DO UPDATE SET
		   severity = EXCLUDED.severity,
		   description = EXCLUDED.description,
		   mechanism = EXCLUDED.mechanism,
		   clinical_effect = EXCLUDED.clinical_effect,
		   management = EXCLUDED.management,
		   last_verified = EXCLUDED.last_verified`,
		uuid.New(), a, b, nameA, nameB,
		string(interaction.Severity), interaction.Description,
		nullStr(interaction.Mechanism), nullStr(interaction.ClinicalEffect),
		nullStr(interaction.Management), interaction.Source, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("cache interaction: %w", err)
	}

	r.logger.Info().Str("drugA", a).Str("drugB", b).Str("severity", string(interaction.Severity)).Msg("interaction cached")
	return nil
}

// GetAllInteractionsForDrug returns all cached interactions involving a drug.
func (r *PostgresDrugCheckerRepository) GetAllInteractionsForDrug(ctx context.Context, rxNormCode string) ([]*DrugInteraction, error) {
	var rows []struct {
		DrugARxNorm  string         `db:"drug_a_rxnorm"`
		DrugBRxNorm  string         `db:"drug_b_rxnorm"`
		DrugAName    string         `db:"drug_a_name"`
		DrugBName    string         `db:"drug_b_name"`
		Severity     string         `db:"severity"`
		Description  string         `db:"description"`
		Mechanism    sql.NullString `db:"mechanism"`
		ClinicalEff  sql.NullString `db:"clinical_effect"`
		Management   sql.NullString `db:"management"`
		Source       string         `db:"source"`
	}

	err := r.db.SelectContext(ctx, &rows,
		`SELECT drug_a_rxnorm, drug_b_rxnorm, drug_a_name, drug_b_name, severity, description, mechanism, clinical_effect, management, source
		 FROM drug_interactions WHERE drug_a_rxnorm = $1 OR drug_b_rxnorm = $1`, rxNormCode)
	if err != nil {
		return nil, fmt.Errorf("query all interactions: %w", err)
	}

	results := make([]*DrugInteraction, 0, len(rows))
	for _, row := range rows {
		results = append(results, &DrugInteraction{
			DrugA:          MedicationInfo{RxNormCode: row.DrugARxNorm, Name: row.DrugAName},
			DrugB:          MedicationInfo{RxNormCode: row.DrugBRxNorm, Name: row.DrugBName},
			Severity:       InteractionSeverity(row.Severity),
			Description:    row.Description,
			Mechanism:      row.Mechanism.String,
			ClinicalEffect: row.ClinicalEff.String,
			Management:     row.Management.String,
			Source:         row.Source,
			Cached:         true,
		})
	}
	return results, nil
}

// GetDrugClass returns the drug class for a given RxNorm code.
func (r *PostgresDrugCheckerRepository) GetDrugClass(ctx context.Context, rxNormCode string) (string, error) {
	var drugClass string
	err := r.db.GetContext(ctx, &drugClass,
		`SELECT drug_class FROM drug_classes WHERE rxnorm_code = $1`, rxNormCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("query drug class: %w", err)
	}
	return drugClass, nil
}

// StoreDrugClass caches a drug class mapping.
func (r *PostgresDrugCheckerRepository) StoreDrugClass(ctx context.Context, rxNormCode, drugName, drugClass string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO drug_classes (id, rxnorm_code, drug_name, drug_class) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (rxnorm_code) DO UPDATE SET drug_class = EXCLUDED.drug_class`,
		uuid.New(), rxNormCode, drugName, drugClass)
	return err
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
