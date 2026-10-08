package firmaemisorv2

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"maps"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

func TestEmisorPlanUsaAutoridadNominalYSuContextoPropio(t *testing.T) {
	for _, via := range []string{ports.ViaFirmaCertificadoVEC, ports.ViaFirmaExternaPortafirmas} {
		a, _, e, m, r, ctx := escenario(t, via)
		sha := r.Atributos["descriptor_firma_sha256"]
		delete(r.Atributos, "descriptor_firma_sha256")
		r.Atributos["plan_firma_sha256"] = sha
		material, err := a.AutorizarMaterialPlanFirmaV2(ctx, m, r)
		if err != nil || material.ValidarEstructura() != nil || e.llamadas != 1 {
			t.Fatalf("emisión: %v", err)
		}
		huella, err := r.HuellaContextoAutorizacionSHA256()
		if err != nil || material.ResumenCapacidad().EfectoHuellaSHA256() != huella {
			t.Fatalf("contexto exterior: %v", err)
		}
		if _, err := a.AutorizarMaterialFirmaVerificadaV2(ctx, m, r); err == nil || e.llamadas != 1 {
			t.Fatal("el contrato interior acepta el contexto exterior")
		}
		delete(r.Atributos, "plan_firma_sha256")
		r.Atributos["descriptor_firma_sha256"] = sha
		if _, err := a.AutorizarMaterialPlanFirmaV2(ctx, m, r); err == nil || e.llamadas != 1 {
			t.Fatal("el plan acepta una autorización interior")
		}
	}
}

// La decisión exterior del plan lleva, como la interior, los ámbitos de la
// asignación vigente: con unidad, un recurso sólo de organización no llega al
// PDP y el que lleva la unidad de la asignación sí.
func TestEmisorPlanLlevaLosAmbitosDeLaAsignacion(t *testing.T) {
	a, _, e, m, r, ctx := escenario(t, ports.ViaFirmaCertificadoVEC)
	a.autorizacion.(*autorizacionPrueba).ambitos = []vd.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{m.OrganizacionRef}},
		{Clave: "unidad_ref", Valores: []string{m.UnidadFirmanteRef}}}
	sha := r.Atributos["descriptor_firma_sha256"]
	delete(r.Atributos, "descriptor_firma_sha256")
	r.Atributos["plan_firma_sha256"] = sha
	if _, err := a.AutorizarMaterialPlanFirmaV2(ctx, m, r); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || e.llamadas != 0 {
		t.Fatal("recurso exterior sin la unidad de la asignación llegó al PDP")
	}
	r.Ambitos["unidad_ref"] = m.UnidadFirmanteRef
	material, err := a.AutorizarMaterialPlanFirmaV2(ctx, m, r)
	if err != nil || material.ValidarEstructura() != nil || e.llamadas != 1 {
		t.Fatalf("recurso exterior con la unidad de la asignación denegado: %v", err)
	}
	if d, _ := e.solicitud.Datos(); !maps.Equal(d.Recurso.Ambitos, r.Ambitos) {
		t.Fatal("el PDP no recibió los ámbitos de la asignación")
	}
}

// La huella de contexto que calcula el PDP en Go para un recurso exterior con
// unidad es la misma que calcula AD209 en SQL: el vector ct185_ad209 fija este
// mismo valor con las mismas entradas.
func TestFirmaV2HuellaPlanConUnidadIgualQueSQL(t *testing.T) {
	sol := `{"Via":"certificado_vec","OrganizacionRef":"org_fija","UnidadFirmanteRef":"unidad:fija"}`
	material, envoltorio := sha256.Sum256([]byte(sol)), sha256.Sum256([]byte(`{"e":1}`))
	r := vd.RecursoAutorizable{Referencia: "documento:fijo", ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoFirmaVec,
		Ambitos:   map[string]string{"organizacion_ref": "org_fija", "unidad_ref": "unidad:fija"},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(material[:]), "plan_firma_sha256": hex.EncodeToString(envoltorio[:])}}
	h, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil || h != "14b23972f887f011031a06ba4ae7a7798548a3532211f8712f2a8d1a448df0db" {
		t.Fatalf("huella distinta de la de AD209: %s %v", h, err)
	}
}
