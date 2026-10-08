package domain

import (
	"sort"
	"time"
)

// ConsultaCapacidadInformativaV3 describe un recurso ya resuelto por su módulo.
// No representa una solicitud ejecutable ni admite referencias del navegador
// como autoridad sobre ámbitos o atributos.
type ConsultaCapacidadInformativaV3 struct {
	Accion    string
	Recurso   RecursoAutorizable
	Finalidad string
}

type ResultadoCapacidadInformativaV3 struct {
	Concedida        bool
	CamposPermitidos []string
	Obligaciones     []string
}

const maximoCapacidadesInformativasV3 = 128

// EvaluarCapacidadesInformativasV3 reutiliza exactamente la evaluación pura
// que usa la decisión V3. No crea DecisionRef, concesiones, denegaciones ni
// material consumible. Todos los recursos pertenecen al mismo actor y perfil.
func EvaluarCapacidadesInformativasV3(
	vinculo VinculoAutenticacionActorV2,
	resultado ResultadoContextoActorRegistradoV2,
	instantanea InstantaneaAutorizacion,
	consultas []ConsultaCapacidadInformativaV3,
	instante time.Time,
) ([]ResultadoCapacidadInformativaV3, time.Time, error) {
	if len(consultas) == 0 || len(consultas) > maximoCapacidadesInformativasV3 ||
		resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil ||
		instantanea.Validar() != nil || !instanteAutorizacionCanonico(instante) {
		return nil, time.Time{}, ErrConfiguracionAccesoInvalida
	}
	datos, err := vinculo.Datos()
	if err != nil {
		return nil, time.Time{}, ErrConfiguracionAccesoInvalida
	}
	if instantanea.AsignacionPerfil.PrincipalID != datos.PrincipalID ||
		instantanea.AsignacionPerfil.PerfilActivoRef != datos.PerfilActivoRef ||
		!vinculo.VigenteEn(instante, resultado) ||
		instantanea.ControlVigenciaVersionRol.ActualizadoEn.After(instante) {
		return nil, time.Time{}, ErrAutorizacionDenegada
	}
	politicas := append([]PoliticaRestrictiva(nil), instantanea.Politicas...)
	sort.Slice(politicas, func(i, j int) bool {
		return politicas[i].Referencia() < politicas[j].Referencia()
	})
	limite := limitarVentanaEvaluacionAutorizacionV3(instante, instante.Add(5*time.Minute), datos, instantanea)
	limite = limiteContextoCapacidadInformativaV3(resultado.Contexto.Instantanea, limite)
	if !limite.After(instante) {
		return nil, time.Time{}, ErrAutorizacionDenegada
	}
	salidas := make([]ResultadoCapacidadInformativaV3, 0, len(consultas))
	for _, consulta := range consultas {
		if !textoAutorizacionSinComodinSeguro(consulta.Accion, 256, false) ||
			!textoAutorizacionSinComodinSeguro(consulta.Finalidad, 512, false) ||
			consulta.Recurso.Validar() != nil {
			return nil, time.Time{}, ErrSolicitudAutorizacionInvalida
		}
		solicitud := SolicitudAutorizacion{Accion: consulta.Accion, Recurso: consulta.Recurso, Finalidad: consulta.Finalidad}
		concedida, _, _, campos, obligaciones, err := evaluarResultadoAutorizacionV3(
			solicitud, datos.GarantiaObservada, instantanea, politicas, instante,
		)
		if err != nil {
			return nil, time.Time{}, ErrConfiguracionAccesoInvalida
		}
		salidas = append(salidas, ResultadoCapacidadInformativaV3{
			Concedida: concedida, CamposPermitidos: append([]string{}, campos...),
			Obligaciones: append([]string{}, obligaciones...),
		})
	}
	return salidas, limite, nil
}

func limiteContextoCapacidadInformativaV3(contexto InstantaneaContextoActor, limite time.Time) time.Time {
	if contexto.VigenteHasta.Before(limite) {
		limite = contexto.VigenteHasta
	}
	for _, vinculo := range contexto.Vinculos {
		if vinculo.VigenteHasta.Before(limite) {
			limite = vinculo.VigenteHasta
		}
	}
	return limite
}
