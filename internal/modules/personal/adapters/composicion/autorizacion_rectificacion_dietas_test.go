package composicion

import (
	"testing"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

func TestRectificacionDietasConservaAudienciasSeparadas(t *testing.T) {
	casos := []struct {
		operacion                    personaldomain.OperacionRectificacionDietas
		accion, finalidad, audiencia string
	}{
		{personaldomain.SolicitarRectificacionDietas, personalports.AccionSolicitarRectificacionDietas, "solicitar_rectificacion_dietas", personalports.AudienciaSolicitarRectificacionDietas},
		{personaldomain.ConsultarRectificacionDietas, personalports.AccionConsultarRectificacionDietas, "consultar_rectificacion_dietas_propia", personalports.AudienciaConsultarRectificacionDietas},
		{personaldomain.ConfirmarRectificacionDietas, personalports.AccionResolverRectificacionDietas, "resolver_rectificacion_dietas", personalports.AudienciaResolverRectificacionDietas},
		{personaldomain.RechazarRectificacionDietas, personalports.AccionResolverRectificacionDietas, "resolver_rectificacion_dietas", personalports.AudienciaResolverRectificacionDietas},
		{"otra", "", "", ""},
	}
	for _, caso := range casos {
		accion, finalidad, audiencia := contratoAutorizacionRectificacionDietas(caso.operacion)
		if accion != caso.accion || finalidad != caso.finalidad || audiencia != caso.audiencia {
			t.Fatalf("operación %q: %q/%q/%q", caso.operacion, accion, finalidad, audiencia)
		}
	}
}
