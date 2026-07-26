package application

import "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"

// DatosPropuestaCoberturaParaAdaptador es una copia de la vista cuyo origen
// queda cerrado en application. El adaptador solo puede leerla y traducirla.
type DatosPropuestaCoberturaParaAdaptador struct {
	Estado             domain.EstadoPropuestaDecisionCobertura
	ViaRecomendada     domain.ClaveCatalogo
	Evaluaciones       []domain.EvaluacionViaPropuestaCobertura
	IdentidadSemantica domain.IdentidadSemanticaPropuestaDecisionCobertura
}

// ResultadoPropuestaCoberturaParaAdaptador evita que campos públicos de una
// PresentacionPropuestaCobertura mutable lleguen a una respuesta HTTP.
type ResultadoPropuestaCoberturaParaAdaptador struct {
	datos DatosPropuestaCoberturaParaAdaptador
	sello string
}

func nuevaResultadoPropuestaCoberturaParaAdaptador(
	p PresentacionPropuestaCobertura,
) (ResultadoPropuestaCoberturaParaAdaptador, error) {
	datos := DatosPropuestaCoberturaParaAdaptador{
		Estado: p.Estado, ViaRecomendada: p.ViaRecomendada,
		Evaluaciones:       copiarEvaluacionesCobertura(p.Evaluaciones),
		IdentidadSemantica: p.IdentidadSemantica,
	}
	if !datosPropuestaCoberturaAdaptadorValidos(datos) {
		return ResultadoPropuestaCoberturaParaAdaptador{}, ErrPresentacionPropuestaCoberturaNoConfiable
	}
	return ResultadoPropuestaCoberturaParaAdaptador{datos: datos, sello: datos.IdentidadSemantica.HuellaSHA256}, nil
}

func (r ResultadoPropuestaCoberturaParaAdaptador) DatosParaAdaptador() (DatosPropuestaCoberturaParaAdaptador, bool) {
	if r.sello == "" || r.sello != r.datos.IdentidadSemantica.HuellaSHA256 || !datosPropuestaCoberturaAdaptadorValidos(r.datos) {
		return DatosPropuestaCoberturaParaAdaptador{}, false
	}
	r.datos.Evaluaciones = copiarEvaluacionesCobertura(r.datos.Evaluaciones)
	return r.datos, true
}

func datosPropuestaCoberturaAdaptadorValidos(p DatosPropuestaCoberturaParaAdaptador) bool {
	if p.IdentidadSemantica.Validar() != nil || !p.ViaRecomendada.Valida() ||
		!estadoPropuestaAdaptadorValido(p.Estado) || len(p.Evaluaciones) == 0 || len(p.Evaluaciones) > 64 {
		return false
	}
	vias := make(map[domain.ClaveCatalogo]struct{}, len(p.Evaluaciones))
	prioridades := make(map[uint16]struct{}, len(p.Evaluaciones))
	for _, evaluacion := range p.Evaluaciones {
		if !evaluacion.ViaClave.Valida() || !estadoEvaluacionAdaptadorValido(evaluacion.Estado) || evaluacion.Prioridad == 0 || len(evaluacion.ResultadosOmitidos)+len(evaluacion.AusenciasBloqueantes)+len(evaluacion.AusenciasAdmitidas)+len(evaluacion.NoHabilitantes)+len(evaluacion.Conflictos) > 32 {
			return false
		}
		if _, existe := vias[evaluacion.ViaClave]; existe {
			return false
		}
		if _, existe := prioridades[evaluacion.Prioridad]; existe {
			return false
		}
		vias[evaluacion.ViaClave] = struct{}{}
		prioridades[evaluacion.Prioridad] = struct{}{}
		for _, grupo := range [][]domain.ClaveCatalogo{evaluacion.ResultadosOmitidos, evaluacion.AusenciasBloqueantes, evaluacion.AusenciasAdmitidas, evaluacion.NoHabilitantes, evaluacion.Conflictos} {
			for _, clave := range grupo {
				if !clave.Valida() {
					return false
				}
			}
		}
	}
	return true
}

func copiarEvaluacionesCobertura(entrada []domain.EvaluacionViaPropuestaCobertura) []domain.EvaluacionViaPropuestaCobertura {
	salida := make([]domain.EvaluacionViaPropuestaCobertura, len(entrada))
	for i, evaluacion := range entrada {
		salida[i] = evaluacion
		salida[i].ResultadosOmitidos = append([]domain.ClaveCatalogo(nil), evaluacion.ResultadosOmitidos...)
		salida[i].AusenciasBloqueantes = append([]domain.ClaveCatalogo(nil), evaluacion.AusenciasBloqueantes...)
		salida[i].AusenciasAdmitidas = append([]domain.ClaveCatalogo(nil), evaluacion.AusenciasAdmitidas...)
		salida[i].NoHabilitantes = append([]domain.ClaveCatalogo(nil), evaluacion.NoHabilitantes...)
		salida[i].Conflictos = append([]domain.ClaveCatalogo(nil), evaluacion.Conflictos...)
	}
	return salida
}

func estadoPropuestaAdaptadorValido(estado domain.EstadoPropuestaDecisionCobertura) bool {
	return estado == domain.PropuestaCoberturaViable || estado == domain.PropuestaCoberturaIncompleta || estado == domain.PropuestaCoberturaConflictiva || estado == domain.PropuestaCoberturaSinVia
}

func estadoEvaluacionAdaptadorValido(estado domain.EstadoEvaluacionViaCobertura) bool {
	return estado == domain.EvaluacionViaCoberturaViable || estado == domain.EvaluacionViaCoberturaIncompleta || estado == domain.EvaluacionViaCoberturaConflictiva || estado == domain.EvaluacionViaCoberturaNoViable
}
