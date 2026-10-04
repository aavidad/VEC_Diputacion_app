package firmaemisorv2

import (
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
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
