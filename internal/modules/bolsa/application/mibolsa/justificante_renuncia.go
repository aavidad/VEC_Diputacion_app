package mibolsa

import (
	"regexp"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var recursoJustificanteRenuncia = regexp.MustCompile(`^justificante-renuncia:[0-9a-f]{64}$`)

// CandidatoDelActo lee el único vínculo candidato activo ya registrado. Dos
// vínculos ambiguos impiden resolver el acto, aunque uno parezca coincidir.
func CandidatoDelActo(a vecdomain.ResultadoContextoActorRegistradoV2,
	v vecdomain.VinculoAutenticacionActorV2) (string, error) {
	if a.Validar() != nil || v.ValidarPara(a) != nil {
		return "", bolsaports.ErrJustificanteRenunciaDenegado
	}
	var candidato string
	for _, vinculo := range a.Contexto.Instantanea.Vinculos {
		if vinculo.Tipo != vecdomain.TipoReferenciaContextoActorCandidato ||
			!vinculo.VigenteEn(a.Contexto.ResueltoEn) {
			continue
		}
		if candidato != "" {
			return "", bolsaports.ErrJustificanteRenunciaDenegado
		}
		candidato = vinculo.Referencia
	}
	if candidato == "" {
		return "", bolsaports.ErrJustificanteRenunciaDenegado
	}
	return candidato, nil
}

func ActoJustificanteValido(a bolsaports.ActoJustificanteRenuncia,
	persona, candidato, bolsa string) bool {
	return recursoJustificanteRenuncia.MatchString(a.RecursoRef) &&
		a.PersonaRef == persona && a.CandidatoRef == candidato && a.BolsaRef == bolsa &&
		a.ParticipacionRef != "" && a.LlamamientoRef != "" && a.VersionDueno == 1
}
