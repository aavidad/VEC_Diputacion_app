package plannominal

import (
	"bytes"
	"strings"
	"testing"

	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
)

func TestAutorizacionExteriorLigaPlanYDecisionInterior(t *testing.T) {
	f, m, _, _, _ := descriptorPrueba(t, 1)
	d, err := f.DescriptorPlanFijadoFirmaV2(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	decision := strings.Repeat("a", 64)
	b, err := firma.CanonicoPlanAutorizadoFirmaV2(m, d, decision)
	if err != nil || firma.ValidarPlanAutorizadoFirmaV2(m, d.Plan, decision, b) != nil {
		t.Fatalf("envoltorio: %v", err)
	}
	r, err := firma.RecursoPlanAutorizadoFirmaV2(m, d.Plan, decision, b)
	if err != nil {
		t.Fatal(err)
	}
	h, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	for _, cambiarDecision := range []bool{false, true} {
		otro := d
		otraDecision := decision
		if cambiarDecision {
			otraDecision = strings.Repeat("b", 64)
		} else {
			otro.Plan.CatalogoVersion++
		}
		alterado, err := firma.CanonicoPlanAutorizadoFirmaV2(m, otro, otraDecision)
		if err != nil {
			t.Fatal(err)
		}
		if firma.ValidarPlanAutorizadoFirmaV2(m, d.Plan, decision, alterado) == nil {
			t.Fatal("acepta otra publicación o decisión")
		}
		cruzado, err := firma.RecursoPlanAutorizadoFirmaV2(m, otro.Plan, otraDecision, alterado)
		if err != nil {
			t.Fatal(err)
		}
		otraHuella, err := cruzado.HuellaContextoAutorizacionSHA256()
		if err != nil || otraHuella == h {
			t.Fatalf("la autorización no liga el cambio: %v", err)
		}
	}
	for _, alterado := range [][]byte{
		append(bytes.Clone(b), []byte(`{}`)...),
		append([]byte(" "), b...),
		bytes.Replace(b, []byte(`"plan":`), []byte(`"ajeno":null,"plan":`), 1),
		bytes.Replace(b, []byte(`"decision_interior_sha256":`), []byte(`"decision_interior_sha256":"otro","decision_interior_sha256":`), 1),
	} {
		if firma.ValidarPlanAutorizadoFirmaV2(m, d.Plan, decision, alterado) == nil {
			t.Fatal("acepta JSON ambiguo")
		}
	}
}
