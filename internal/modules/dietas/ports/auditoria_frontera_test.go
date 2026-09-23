package ports

import (
	"strings"
	"testing"
)

func TestOrdenAuditoriaFronteraComisionMinimizaRutaYActor(t *testing.T) {
	base := OrdenAuditoriaFronteraComision{
		CorrelacionRef: "corr_" + strings.Repeat("a", 32),
		Motivo:         MotivoFronteraAccesoDenegado,
		Ruta:           RutaAuditoriaFronteraDetalle,
		Accion:         AccionFronteraConsultarDetalle,
		ActorRef:       "desarrollo:actor_sintetico",
	}
	if err := base.Validar(); err != nil {
		t.Fatal(err)
	}
	for _, cambio := range []func(*OrdenAuditoriaFronteraComision){
		func(o *OrdenAuditoriaFronteraComision) { o.Ruta += "/dco_persona" },
		func(o *OrdenAuditoriaFronteraComision) { o.ActorRef = "actor con datos" },
		func(o *OrdenAuditoriaFronteraComision) { o.Motivo = "error SQL" },
		func(o *OrdenAuditoriaFronteraComision) { o.Accion = "enviar" },
		func(o *OrdenAuditoriaFronteraComision) { o.CorrelacionRef = "corr_" + strings.Repeat("A", 32) },
	} {
		orden := base
		cambio(&orden)
		if orden.Validar() == nil {
			t.Fatalf("orden aceptó dato no minimizado: %+v", orden)
		}
	}
}
