package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrPayrollProfileInvalid = errors.New("personal payroll profile invalid")
	ErrPayrollDraftInvalid   = errors.New("personal payroll draft invalid")
)

type EmploymentRegime string

const (
	RegimeCareerCivilServant  EmploymentRegime = "funcionario_carrera"
	RegimeInterimCivilServant EmploymentRegime = "funcionario_interino"
	RegimeLabourStaff         EmploymentRegime = "personal_laboral"
	RegimeTemporaryLabour     EmploymentRegime = "personal_laboral_temporal"
	RegimeSeniorManager       EmploymentRegime = "alto_cargo"
)

type PayrollConceptKind string

const (
	ConceptBasicPay        PayrollConceptKind = "retribucion_basica"
	ConceptComplementary   PayrollConceptKind = "retribucion_complementaria"
	ConceptSeniority       PayrollConceptKind = "trienio_antiguedad"
	ConceptExtraPay        PayrollConceptKind = "paga_extraordinaria"
	ConceptVariable        PayrollConceptKind = "productividad_gratificacion"
	ConceptSocialSecurity  PayrollConceptKind = "cotizacion_seguridad_social"
	ConceptTaxWithholding  PayrollConceptKind = "retencion_irpf"
	ConceptAdvanceDeduct   PayrollConceptKind = "anticipo_o_reintegro"
	ConceptCourtAttachment PayrollConceptKind = "embargo"
)

type PayrollProfile struct {
	EmployeeID        string
	Regime            EmploymentRegime
	Group             string
	Level             string
	PositionID        string
	SeniorityTriennia int
	IBAN              string
	ActiveFrom        time.Time
	ActiveTo          time.Time
}

func (p PayrollProfile) Validate() error {
	if strings.TrimSpace(p.EmployeeID) == "" || strings.TrimSpace(string(p.Regime)) == "" {
		return ErrPayrollProfileInvalid
	}
	if p.SeniorityTriennia < 0 {
		return ErrPayrollProfileInvalid
	}
	return nil
}

type PayrollConcept struct {
	Code                  string
	Name                  string
	Kind                  PayrollConceptKind
	AmountCents           int64
	CountsForContribution bool
	CountsForTax          bool
	SourceRef             string
}

type PayrollPeriod struct {
	Year  int
	Month time.Month
	Extra bool
}

type PayrollDraft struct {
	EmployeeID       string
	Period           PayrollPeriod
	Regime           EmploymentRegime
	Concepts         []PayrollConcept
	GrossCents       int64
	DeductionsCents  int64
	NetCents         int64
	ContributionBase int64
	TaxBase          int64
}

func BuildPayrollDraft(profile PayrollProfile, period PayrollPeriod, concepts []PayrollConcept) (PayrollDraft, error) {
	if err := profile.Validate(); err != nil {
		return PayrollDraft{}, err
	}
	if period.Year <= 0 || period.Month < time.January || period.Month > time.December {
		return PayrollDraft{}, ErrPayrollDraftInvalid
	}
	draft := PayrollDraft{
		EmployeeID: profile.EmployeeID,
		Period:     period,
		Regime:     profile.Regime,
		Concepts:   append([]PayrollConcept(nil), concepts...),
	}
	for _, concept := range concepts {
		if strings.TrimSpace(concept.Code) == "" || strings.TrimSpace(string(concept.Kind)) == "" {
			return PayrollDraft{}, ErrPayrollDraftInvalid
		}
		if isDeduction(concept.Kind) {
			draft.DeductionsCents += abs(concept.AmountCents)
			continue
		}
		draft.GrossCents += concept.AmountCents
		if concept.CountsForContribution {
			draft.ContributionBase += concept.AmountCents
		}
		if concept.CountsForTax {
			draft.TaxBase += concept.AmountCents
		}
	}
	draft.NetCents = draft.GrossCents - draft.DeductionsCents
	return draft, nil
}

func isDeduction(kind PayrollConceptKind) bool {
	switch kind {
	case ConceptSocialSecurity, ConceptTaxWithholding, ConceptAdvanceDeduct, ConceptCourtAttachment:
		return true
	default:
		return false
	}
}

func abs(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}
