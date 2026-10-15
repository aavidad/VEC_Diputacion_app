package bootstrap

import (
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestPoliticaS6AdmiteRolSoloLectorYExigePublicacionExacta(t *testing.T) {
	consulta := dominiovec.ConcesionRol{Accion: puertosbolsa.AccionConsultarTransicionesSituacion,
		ModuloID: "bolsa", TipoRecurso: puertosbolsa.TipoRecursoGobiernoTransicionesSituacion,
		Finalidades: []string{puertosbolsa.FinalidadConsultaTransicionesSituacion},
		CamposPermitidos: []string{puertosbolsa.CampoConsultaTransicionesSituacion},
		GarantiaMinima: dominiovec.AuthAssuranceHigh}
	if !concesionesTransicionesS6Exactas([]dominiovec.ConcesionRol{consulta}) {
		t.Fatal("el rol lector quedó bloqueado por no tener publicación")
	}
	publicar := dominiovec.ConcesionRol{Accion: puertosbolsa.AccionPublicarTransicionesSituacion,
		ModuloID: "bolsa", TipoRecurso: puertosbolsa.TipoRecursoGobiernoTransicionesSituacion,
		Finalidades: []string{puertosbolsa.FinalidadGobiernoTransicionesSituacion},
		GarantiaMinima: dominiovec.AuthAssuranceHigh}
	if !concesionesTransicionesS6Exactas([]dominiovec.ConcesionRol{consulta, publicar}) {
		t.Fatal("la publicación exacta no quedó disponible")
	}
	publicar.Obligaciones = []string{"aprobar_sin_firma"}
	if concesionesTransicionesS6Exactas([]dominiovec.ConcesionRol{consulta, publicar}) {
		t.Fatal("la publicación con obligación ajena quedó aceptada")
	}
}
