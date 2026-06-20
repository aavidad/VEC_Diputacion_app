package domain

import (
	"testing"
	"time"
)

func TestBuildPayrollDraftSeparatesPublicPayrollConcepts(t *testing.T) {
	profile := PayrollProfile{
		EmployeeID:        "EMP-0031",
		Regime:            RegimeCareerCivilServant,
		Group:             "A2",
		Level:             "22",
		PositionID:        "PUESTO-OBRAS-24",
		SeniorityTriennia: 4,
	}
	draft, err := BuildPayrollDraft(profile, PayrollPeriod{Year: 2026, Month: time.June}, []PayrollConcept{
		{Code: "SUELDO_A2", Name: "Sueldo grupo A2", Kind: ConceptBasicPay, AmountCents: 120000, CountsForContribution: true, CountsForTax: true},
		{Code: "TRIENIOS_A2", Name: "Trienios A2", Kind: ConceptSeniority, AmountCents: 18000, CountsForContribution: true, CountsForTax: true},
		{Code: "CD_22", Name: "Complemento destino 22", Kind: ConceptComplementary, AmountCents: 65000, CountsForContribution: true, CountsForTax: true},
		{Code: "SS_CC", Name: "Seguridad Social", Kind: ConceptSocialSecurity, AmountCents: -12800},
		{Code: "IRPF", Name: "Retencion IRPF", Kind: ConceptTaxWithholding, AmountCents: -38500},
	})
	if err != nil {
		t.Fatalf("BuildPayrollDraft() error = %v", err)
	}
	if draft.GrossCents != 203000 {
		t.Fatalf("gross = %d, want 203000", draft.GrossCents)
	}
	if draft.DeductionsCents != 51300 {
		t.Fatalf("deductions = %d, want 51300", draft.DeductionsCents)
	}
	if draft.NetCents != 151700 {
		t.Fatalf("net = %d, want 151700", draft.NetCents)
	}
	if draft.ContributionBase != 203000 || draft.TaxBase != 203000 {
		t.Fatalf("bases = contribution %d tax %d, want 203000/203000", draft.ContributionBase, draft.TaxBase)
	}
}
