package mibolsa

import (
	"errors"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Las preferencias admiten una autoridad propia de cada superficie. Las
// demás acciones de Mi Bolsa conservan su contrato de vínculo exterior.
func validarOrdenPreferencias(o Orden, ahora time.Time, superficie dominiovec.SuperficieAutenticacionActorV1) (dominiovec.ResultadoContextoActorRegistradoV2, string, error) {
	r, ea := o.ResultadoContexto.Clonar()
	d, ev := o.Vinculo.Datos()
	if ahora.IsZero() || ea != nil || ev != nil ||
		(superficie != dominiovec.SuperficieAutenticacionExternaPersonalV1 && superficie != dominiovec.SuperficieAutenticacionInternaCorporativaV1) ||
		r.Contexto.Principal.AuthMethod == dominiovec.AuthMethodDemo || d.MetodoObservado == dominiovec.AuthMethodDemo ||
		d.Superficie != superficie || d.CuentaPrivilegiada || d.GarantiaObservada != dominiovec.AuthAssuranceHigh ||
		o.Vinculo.ValidarPara(r) != nil || !o.Vinculo.VigenteEn(ahora, r) ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(o.Motivo) || o.Correlacion.Validar() != nil {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, "", errors.Join(dominiovec.ErrAutorizacionDenegada, bolsa.ErrConsultaMiBolsaInvalida)
	}
	var candidato string
	for _, v := range r.Contexto.Instantanea.Vinculos {
		if v.Tipo == dominiovec.TipoReferenciaContextoActorCandidato && v.VigenteEn(ahora) {
			if candidato != "" {
				return dominiovec.ResultadoContextoActorRegistradoV2{}, "", errors.Join(dominiovec.ErrAutorizacionDenegada, bolsa.ErrConsultaMiBolsaInvalida)
			}
			candidato = v.Referencia
		}
	}
	if candidato == "" {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, "", errors.Join(dominiovec.ErrAutorizacionDenegada, bolsa.ErrConsultaMiBolsaInvalida)
	}
	return r, candidato, nil
}
