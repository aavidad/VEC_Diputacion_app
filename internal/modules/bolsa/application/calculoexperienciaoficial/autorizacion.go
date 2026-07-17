package calculoexperienciaoficial

import (
	"context"
	"errors"
	"strconv"
	"time"

	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	moduloBolsaCalculoOficial = "bolsa"

	accionLeerFuenteCalculo  = "bolsa.calculo_experiencia.fuente.leer"
	tipoRecursoFuenteCalculo = "fuente_calculo_experiencia"
	finalidadLeerFuente      = "calculo_oficial_experiencia"

	accionConfirmarCalculo     = "bolsa.calculo_experiencia.oficial.confirmar"
	accionRectificarCalculo    = "bolsa.calculo_experiencia.oficial.rectificar"
	tipoRecursoCalculoOficial  = "calculo_experiencia_oficial"
	tipoRecursoRectificacion   = "rectificacion_calculo_experiencia_oficial"
	finalidadConfirmarCalculo  = "confirmacion_calculo_oficial_experiencia"
	finalidadRectificarCalculo = "rectificacion_calculo_oficial_experiencia"
)

func nuevasPoliticas(
	perfil aplicacionvec.PerfilProteccionUsoAutorizacion,
) (aplicacionvec.PoliticaUsoDecisionAutorizacion,
	aplicacionvec.PoliticaUsoDecisionAutorizacion,
	aplicacionvec.PoliticaUsoDecisionAutorizacion, error,
) {
	lectura, err := aplicacionvec.NuevaPoliticaUsoDecisionAutorizacion(
		accionLeerFuenteCalculo, moduloBolsaCalculoOficial,
		tipoRecursoFuenteCalculo, finalidadLeerFuente,
		[]string{"fuente_reglas", "instantanea_experiencia", "prueba_procedencia"},
		perfil,
	)
	if err != nil {
		return aplicacionvec.PoliticaUsoDecisionAutorizacion{},
			aplicacionvec.PoliticaUsoDecisionAutorizacion{},
			aplicacionvec.PoliticaUsoDecisionAutorizacion{}, err
	}
	escritura, err := aplicacionvec.NuevaPoliticaUsoDecisionAutorizacion(
		accionConfirmarCalculo, moduloBolsaCalculoOficial,
		tipoRecursoCalculoOficial, finalidadConfirmarCalculo,
		[]string{"auditoria", "resultado_canonico", "salida_eventos"}, perfil,
	)
	if err != nil {
		return aplicacionvec.PoliticaUsoDecisionAutorizacion{},
			aplicacionvec.PoliticaUsoDecisionAutorizacion{},
			aplicacionvec.PoliticaUsoDecisionAutorizacion{}, err
	}
	rectificacion, err := aplicacionvec.NuevaPoliticaUsoDecisionAutorizacion(
		accionRectificarCalculo, moduloBolsaCalculoOficial,
		tipoRecursoRectificacion, finalidadRectificarCalculo,
		[]string{"auditoria", "resultado_canonico", "salida_eventos"}, perfil,
	)
	return lectura, escritura, rectificacion, err
}

func (s *Servicio) autorizarLectura(
	ctx context.Context,
	datos DatosOrdenConfiable,
	desde time.Time,
) (puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2, time.Time, error) {
	recurso, err := recursoLectura(datos.Selector)
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, time.Time{},
			errors.Join(dominiovec.ErrAutorizacionDenegada, ErrOrdenInvalida, err)
	}
	evidencia, err := s.exigidor.ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
		ctx, datos.ContextoActor, datos.VinculoAutenticacionActor, recurso,
		datos.CorrelacionLectura, datos.Motivo, s.polLectura,
	)
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, time.Time{},
			errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	ahora := instanteCanonico(s.reloj.Ahora())
	if ahora.IsZero() || ahora.Before(desde) ||
		validarEvidenciaExacta(
			evidencia, datos.CorrelacionLectura, datos.Motivo, recurso,
			accionLeerFuenteCalculo, finalidadLeerFuente,
			[]string{"fuente_reglas", "instantanea_experiencia", "prueba_procedencia"},
			desde, ahora,
		) != nil || !evidenciaPerteneceAOrden(evidencia, datos, s.perfil) {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, time.Time{},
			errors.Join(dominiovec.ErrAutorizacionDenegada, ErrFuenteNoConfiable)
	}
	return evidencia, ahora, nil
}

func (s *Servicio) autorizarEscritura(
	ctx context.Context,
	datos DatosOrdenConfiable,
	intencion oficial.IntencionResultadoV1,
	desde time.Time,
) (puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2, time.Time, error) {
	recurso, err := recursoEscritura(intencion)
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, time.Time{},
			errors.Join(dominiovec.ErrAutorizacionDenegada, ErrResultadoNoConfiable, err)
	}
	politica := s.polAlta
	accion, finalidad := accionConfirmarCalculo, finalidadConfirmarCalculo
	if datos.TipoEfecto == oficial.EfectoRectificacion {
		politica = s.polRectifica
		accion, finalidad = accionRectificarCalculo, finalidadRectificarCalculo
	}
	evidencia, err := s.exigidor.ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
		ctx, datos.ContextoActor, datos.VinculoAutenticacionActor, recurso,
		datos.CorrelacionEscritura, datos.Motivo, politica,
	)
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, time.Time{},
			errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	ahora := instanteCanonico(s.reloj.Ahora())
	if ahora.IsZero() || ahora.Before(desde) ||
		validarEvidenciaExacta(
			evidencia, datos.CorrelacionEscritura, datos.Motivo, recurso,
			accion, finalidad,
			[]string{"auditoria", "resultado_canonico", "salida_eventos"}, desde, ahora,
		) != nil || !evidenciaPerteneceAOrden(evidencia, datos, s.perfil) {
		return puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, time.Time{},
			errors.Join(dominiovec.ErrAutorizacionDenegada, ErrConfirmacionInvalida)
	}
	return evidencia, ahora, nil
}

func evidenciaPerteneceAOrden(
	evidencia puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	datos DatosOrdenConfiable,
	perfil perfilServicio,
) bool {
	proyeccion, err := evidencia.Datos()
	if err != nil {
		return false
	}
	decision := proyeccion.Decision
	return decision.PrincipalID == datos.ContextoActor.Principal.ID &&
		decision.PerfilActivoRef == datos.ContextoActor.PerfilActivoRef &&
		decision.VinculoAutenticacionActor.CoincideExactamenteCon(
			datos.VinculoAutenticacionActor,
		) && perfilEvidenciaValido(perfilDuradero(perfil), decision)
}

func recursoLectura(
	selector puertosbolsa.SelectorFuenteExactaCalculoReglasBaremo,
) (dominiovec.RecursoAutorizable, error) {
	estado := selector.EstadoReglas
	huellaSelector, err := selector.HuellaSHA256V1()
	if err != nil {
		return dominiovec.RecursoAutorizable{}, err
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: "fuente:" + huellaSelector,
		ModuloID:   moduloBolsaCalculoOficial, Tipo: tipoRecursoFuenteCalculo,
		Ambitos: map[string]string{
			"convocatoria_ref": selector.Convocatoria.Referencia(),
			"sujeto_ref":       selector.SujetoPseudonimo.Referencia(),
		},
		Atributos: map[string]string{
			"selector_sha256":     huellaSelector,
			"reglas_revision":     strconv.FormatUint(estado.Revision(), 10),
			"reglas_huella":       estado.HuellaEstadoSHA256(),
			"entrada_huella":      selector.InstantaneaEntrada.HuellaSHA256(),
			"sujeto_huella":       selector.SujetoPseudonimo.HuellaSHA256(),
			"convocatoria_huella": selector.Convocatoria.HuellaSHA256(),
		},
	}
	return recurso, recurso.Validar()
}

func recursoEscritura(
	intencion oficial.IntencionResultadoV1,
) (dominiovec.RecursoAutorizable, error) {
	huella, err := intencion.HuellaSHA256()
	if err != nil {
		return dominiovec.RecursoAutorizable{}, err
	}
	clave := intencion.Clave()
	if clave.Validar() != nil {
		return dominiovec.RecursoAutorizable{}, ErrResultadoNoConfiable
	}
	sujeto := clave.SujetoPseudonimizado()
	convocatoria := clave.Convocatoria()
	causa := clave.Causa()
	motor := clave.Motor()
	tipoRecurso := tipoRecursoCalculoOficial
	prefijo := "calculo-oficial:"
	if clave.Tipo() == oficial.EfectoRectificacion {
		tipoRecurso = tipoRecursoRectificacion
		prefijo = "rectificacion-calculo-oficial:"
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: prefijo + huella,
		ModuloID:   moduloBolsaCalculoOficial, Tipo: tipoRecurso,
		Ambitos: map[string]string{
			"sujeto_ref":       sujeto.Referencia,
			"convocatoria_ref": convocatoria.Referencia,
		},
		Atributos: map[string]string{
			"intencion_sha256": huella,
			"causa_clave":      causa.Clave, "motor_contrato": motor.Contrato,
			"motor_version": strconv.FormatUint(motor.Version, 10),
			"motor_huella":  motor.HuellaContratoSHA256,
			"tipo_efecto":   string(clave.Tipo()),
		},
	}
	if predecesor, presente := clave.Predecesor(); presente {
		recurso.Atributos["predecesor_ref"] = predecesor.ReferenciaRecibo
		recurso.Atributos["predecesor_huella"] = predecesor.HuellaReciboSHA256
	}
	return recurso, recurso.Validar()
}
