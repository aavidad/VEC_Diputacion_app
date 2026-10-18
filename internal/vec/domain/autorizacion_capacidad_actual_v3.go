package domain

import (
	"sort"
	"time"
)

// PuedeEjecutarAccionActualV3 consulta el permiso central vigente para una
// acción sin campos. No crea decisión, concesión candidata ni material de
// consumo. El efecto debe volver a autorizarse en su frontera propia.
func PuedeEjecutarAccionActualV3(vinculo VinculoAutenticacionActorV2,
	resultado ResultadoContextoActorRegistradoV2, instantanea InstantaneaAutorizacion,
	accion string, recurso RecursoAutorizable, finalidad string, instante time.Time,
) (bool, error) {
	if resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil || instantanea.Validar() != nil ||
		!instanteAutorizacionCanonico(instante) || !textoAutorizacionSinComodinSeguro(accion, 256, false) ||
		!textoAutorizacionSinComodinSeguro(finalidad, 512, false) || recurso.Validar() != nil {
		return false, ErrAutorizacionDenegada
	}
	datos, err := vinculo.Datos()
	if err != nil || !vinculo.VigenteEn(instante, resultado) ||
		instantanea.AsignacionPerfil.PrincipalID != datos.PrincipalID ||
		instantanea.AsignacionPerfil.PerfilActivoRef != datos.PerfilActivoRef ||
		instantanea.ControlVigenciaVersionRol.ActualizadoEn.After(instante) {
		return false, ErrAutorizacionDenegada
	}
	politicas := append([]PoliticaRestrictiva(nil), instantanea.Politicas...)
	sort.Slice(politicas, func(i, j int) bool { return politicas[i].Referencia() < politicas[j].Referencia() })
	concedida, _, _, campos, obligaciones, err := evaluarResultadoAutorizacionV3(
		SolicitudAutorizacion{Accion: accion, Recurso: recurso, Finalidad: finalidad},
		datos.GarantiaObservada, instantanea, politicas, instante)
	if err != nil {
		return false, err
	}
	return concedida && len(campos) == 0 && len(obligaciones) == 0, nil
}
