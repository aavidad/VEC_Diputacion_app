package plannominal

import (
	"bytes"
	"strings"
	"testing"

	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
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
	r, err := firma.RecursoPlanAutorizadoFirmaV2(m, d.Plan, decision, b, ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef})
	if err != nil {
		t.Fatal(err)
	}
	h, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	// Con la unidad del paso en la asignación, el recurso la lleva y su huella
	// cambia; otra unidad, otra organización o la vía externa con unidad no.
	conUnidad, err := firma.RecursoPlanAutorizadoFirmaV2(m, d.Plan, decision, b, ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef, UnidadRef: m.UnidadFirmanteRef})
	hu, errU := conUnidad.HuellaContextoAutorizacionSHA256()
	if err != nil || errU != nil || conUnidad.Ambitos["unidad_ref"] != m.UnidadFirmanteRef || len(conUnidad.Ambitos) != 2 || hu == h {
		t.Fatalf("recurso exterior con unidad: %v %v", err, errU)
	}
	for caso, a := range map[string]ports.AmbitosOperadorFirmaV2{
		"otra_unidad":       {OrganizacionRef: m.OrganizacionRef, UnidadRef: "unidad:otra"},
		"otra_organizacion": {OrganizacionRef: "org_otra"},
	} {
		if _, err := firma.RecursoPlanAutorizadoFirmaV2(m, d.Plan, decision, b, a); err == nil {
			t.Fatalf("%s aceptado", caso)
		}
	}
	// Vía externa, con su propio envoltorio: sólo organización.
	externa := m
	externa.Via = ports.ViaFirmaExternaPortafirmas
	externa.ReferenciaPortafirmasDeclarada, externa.FechaPortafirmasDeclarada = "portafirmas:prueba", "2026-10-03T11:00:00Z"
	be, err := firma.CanonicoPlanAutorizadoFirmaV2(externa, d, decision)
	if err != nil || externa.Validar() != nil {
		t.Fatalf("material externo de prueba: %v %v", err, externa.Validar())
	}
	if _, err := firma.RecursoPlanAutorizadoFirmaV2(externa, d.Plan, decision, be, ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef}); err != nil {
		t.Fatalf("externa sólo con organización denegada: %v", err)
	}
	if _, err := firma.RecursoPlanAutorizadoFirmaV2(externa, d.Plan, decision, be, ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef, UnidadRef: m.UnidadFirmanteRef}); err == nil {
		t.Fatal("externa con unidad aceptada")
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
		cruzado, err := firma.RecursoPlanAutorizadoFirmaV2(m, otro.Plan, otraDecision, alterado, ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef})
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
