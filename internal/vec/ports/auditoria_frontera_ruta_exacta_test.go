package ports

import (
	"errors"
	"strings"
	"testing"
)

func TestOrdenAuditoriaFronteraRutaExactaEsMinimaYCerrada(t *testing.T) {
	orden := OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_0123456789abcdef0123456789abcdef",
		Motivo:         MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
		Superficie:     SuperficieAuditoriaFronteraRutaExactaContratacionTemporal,
		Ruta:           "/api/vec/contratacion-temporal/solicitudes",
		ActorRef:       "actor:verificado:001",
	}
	if err := orden.Validar(); err != nil {
		t.Fatalf("orden valida: %v", err)
	}
	for nombre, mutar := range map[string]func(*OrdenAuditoriaFronteraRutaExacta){
		"correlacion libre": func(o *OrdenAuditoriaFronteraRutaExacta) { o.CorrelacionRef = "correlacion libre" },
		"motivo libre":      func(o *OrdenAuditoriaFronteraRutaExacta) { o.Motivo = "detalle privado" },
		"superficie ajena":  func(o *OrdenAuditoriaFronteraRutaExacta) { o.Superficie = "api.ajena" },
		"query en ruta":     func(o *OrdenAuditoriaFronteraRutaExacta) { o.Ruta += "?dato=privado" },
		"cabecera en actor": func(o *OrdenAuditoriaFronteraRutaExacta) { o.ActorRef = "actor\nlibre" },
		"error en ruta": func(o *OrdenAuditoriaFronteraRutaExacta) {
			o.Ruta = "/api/vec/contratacion-temporal/error-libre" + strings.Repeat("/", 1)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := orden
			mutar(&alterada)
			if !errors.Is(alterada.Validar(), ErrOrdenAuditoriaFronteraRutaExactaInvalida) {
				t.Fatalf("orden invalida aceptada: %#v", alterada)
			}
		})
	}
}
