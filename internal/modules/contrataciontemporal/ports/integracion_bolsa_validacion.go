package ports

import (
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func huellaIntegracionBolsaValida(valor string) bool {
	return patronHuellaSHA256.MatchString(valor) && valor != strings.Repeat("0", 64)
}

func validarVinculoRespuestaBolsa(
	operacionRef string,
	organizacionRef string,
	expedienteRef string,
	versionExpediente uint64,
	correlacionRef string,
	necesidad ReferenciaVersionadaIntegracionBolsa,
	resultado ReferenciaVersionadaIntegracionBolsa,
	procedencia ProcedenciaIntegracionBolsa,
	contexto ContextoPeticionIntegracionBolsa,
	necesidadEsperada ReferenciaVersionadaIntegracionBolsa,
) error {
	if contexto.ValidarEn(procedencia.EmitidaEn) != nil ||
		!domain.ReferenciaOpacaValida(operacionRef) ||
		!domain.ReferenciaOpacaValida(organizacionRef) ||
		!domain.ReferenciaOpacaValida(expedienteRef) ||
		versionExpediente == 0 ||
		!domain.ReferenciaOpacaValida(correlacionRef) ||
		necesidad.Validar() != nil || resultado.Validar() != nil ||
		procedencia.Validar() != nil ||
		operacionRef != contexto.OperacionRef ||
		organizacionRef != contexto.OrganizacionRef ||
		expedienteRef != contexto.ExpedienteRef ||
		versionExpediente != contexto.VersionExpediente ||
		correlacionRef != contexto.CorrelacionRef ||
		necesidad != necesidadEsperada ||
		procedencia.EmitidaEn.Before(contexto.SolicitadaEn) ||
		!procedencia.EmitidaEn.Before(contexto.ValidaHasta) {
		return ErrRespuestaBolsaNoConfiable
	}
	return nil
}
