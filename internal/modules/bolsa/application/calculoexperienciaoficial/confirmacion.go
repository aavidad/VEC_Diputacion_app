package calculoexperienciaoficial

import (
	"bytes"
	"sort"
	"time"

	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func nuevaSolicitudConfirmacion(
	perfil perfilServicio,
	orden DatosOrdenConfiable,
	fuente puertosbolsa.FuenteExactaCalculoReglasBaremo,
	preparado calculoPreparado,
	autLectura puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	autEscritura puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	lecturaNoAntesDe, fuenteSolicitadaEn, escrituraNoAntesDe, solicitadaEn time.Time,
) (SolicitudConfirmacionDuradera, error) {
	referenciaIntento, err := orden.CorrelacionEscritura.ValorCanonico()
	if err != nil {
		return SolicitudConfirmacionDuradera{}, ErrConfirmacionInvalida
	}
	datos := &DatosConfirmacionDuradera{
		Perfil: perfilDuradero(perfil), ReferenciaIntento: referenciaIntento,
		Selector: orden.Selector, Fuente: fuente, Clave: preparado.clave,
		Intencion: preparado.intencion, Resultado: preparado.resultado,
		ResultadoCanonico:     append([]byte(nil), preparado.canonico...),
		HuellaResultadoSHA256: preparado.huella,
		AutorizacionLectura:   autLectura, AutorizacionEscritura: autEscritura,
		CorrelacionLectura:   orden.CorrelacionLectura,
		CorrelacionEscritura: orden.CorrelacionEscritura,
		Motivo:               orden.Motivo, LecturaNoAntesDe: lecturaNoAntesDe,
		FuenteSolicitadaEn: fuenteSolicitadaEn, EscrituraNoAntesDe: escrituraNoAntesDe,
		SolicitadaEn: solicitadaEn,
	}
	solicitud := SolicitudConfirmacionDuradera{datos: datos}
	if solicitud.validar() != nil {
		return SolicitudConfirmacionDuradera{}, ErrConfirmacionInvalida
	}
	return solicitud, nil
}

func perfilDuradero(
	perfil perfilServicio,
) PerfilConfirmacionDuradera {
	if perfil == perfilExternoOrdinario {
		return PerfilConfirmacionExternoOrdinario
	}
	if perfil == perfilInternoAlto {
		return PerfilConfirmacionInternoAlto
	}
	return ""
}

func (s SolicitudConfirmacionDuradera) validar() error {
	if s.datos == nil {
		return ErrConfirmacionInvalida
	}
	datos := s.datos
	if !datos.Perfil.valido() || !referenciaIntentoValida(datos.ReferenciaIntento) ||
		!instanteFuenteValido(datos.LecturaNoAntesDe) ||
		!instanteFuenteValido(datos.FuenteSolicitadaEn) ||
		!instanteFuenteValido(datos.EscrituraNoAntesDe) ||
		!instanteFuenteValido(datos.SolicitadaEn) ||
		datos.FuenteSolicitadaEn.Before(datos.LecturaNoAntesDe) ||
		datos.EscrituraNoAntesDe.Before(datos.FuenteSolicitadaEn) ||
		datos.SolicitadaEn.Before(datos.EscrituraNoAntesDe) || !selectorValido(datos.Selector) ||
		validarFuenteExacta(
			datos.Fuente, datos.Selector, datos.AutorizacionLectura,
			datos.FuenteSolicitadaEn, datos.SolicitadaEn,
		) != nil ||
		datos.Clave.Validar() != nil || datos.Intencion.Validar() != nil ||
		datos.Resultado.Validar() != nil ||
		!huellaSHA256Valida(datos.HuellaResultadoSHA256) {
		return ErrConfirmacionInvalida
	}
	canonico, err := datos.Resultado.RepresentacionCanonica()
	huella, errHuella := datos.Resultado.HuellaSHA256()
	estado, fase, errEstado := estadoYFaseOficial(datos.Resultado)
	if err != nil || errHuella != nil || errEstado != nil ||
		!bytes.Equal(canonico, datos.ResultadoCanonico) ||
		huella != datos.HuellaResultadoSHA256 ||
		datos.Intencion.ValidarPara(datos.Clave, huella, estado, fase) != nil {
		return ErrConfirmacionInvalida
	}
	referencia, _ := datos.CorrelacionEscritura.ValorCanonico()
	if referencia != datos.ReferenciaIntento ||
		!correlacionesDistintas(datos.CorrelacionLectura, datos.CorrelacionEscritura) ||
		validarAutorizacionesConfirmacion(*datos) != nil {
		return ErrConfirmacionInvalida
	}
	return nil
}

func validarAutorizacionesConfirmacion(datos DatosConfirmacionDuradera) error {
	recursoLectura, errLectura := recursoLectura(datos.Selector)
	recursoEscritura, errEscritura := recursoEscritura(datos.Intencion)
	accionEscritura, finalidadEscritura :=
		accionConfirmarCalculo, finalidadConfirmarCalculo
	if datos.Clave.Tipo() == oficial.EfectoRectificacion {
		accionEscritura, finalidadEscritura =
			accionRectificarCalculo, finalidadRectificarCalculo
	}
	if errLectura != nil || errEscritura != nil ||
		validarEvidenciaExacta(datos.AutorizacionLectura, datos.CorrelacionLectura,
			datos.Motivo, recursoLectura, accionLeerFuenteCalculo, finalidadLeerFuente,
			[]string{"fuente_reglas", "instantanea_experiencia", "prueba_procedencia"},
			datos.LecturaNoAntesDe, datos.FuenteSolicitadaEn) != nil ||
		validarEvidenciaExacta(datos.AutorizacionEscritura, datos.CorrelacionEscritura,
			datos.Motivo, recursoEscritura, accionEscritura, finalidadEscritura,
			[]string{"auditoria", "resultado_canonico", "salida_eventos"},
			datos.EscrituraNoAntesDe, datos.SolicitadaEn) != nil {
		return ErrConfirmacionInvalida
	}
	lectura, _ := datos.AutorizacionLectura.Datos()
	escritura, _ := datos.AutorizacionEscritura.Datos()
	if lectura.HuellaDecisionSHA256 == escritura.HuellaDecisionSHA256 ||
		lectura.Decision.DecisionRef == escritura.Decision.DecisionRef ||
		lectura.Decision.PrincipalID != escritura.Decision.PrincipalID ||
		lectura.Decision.PerfilActivoRef != escritura.Decision.PerfilActivoRef ||
		!lectura.Decision.VinculoAutenticacionActor.CoincideExactamenteCon(
			escritura.Decision.VinculoAutenticacionActor,
		) || !perfilEvidenciaValido(datos.Perfil, lectura.Decision) ||
		!perfilEvidenciaValido(datos.Perfil, escritura.Decision) {
		return ErrConfirmacionInvalida
	}
	return nil
}

func validarEvidenciaExacta(
	evidencia puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	recurso dominiovec.RecursoAutorizable,
	accion, finalidad string,
	campos []string,
	desde, instante time.Time,
) error {
	datos, err := evidencia.Datos()
	valorCorrelacion, errCorrelacion := correlacion.ValorCanonico()
	huellaRecurso, errRecurso := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || errCorrelacion != nil || errRecurso != nil ||
		desde.IsZero() || instante.Before(desde) || evidencia.ValidarEn(instante) != nil ||
		evidencia.ValidarMotivo(motivo) != nil || datos.VerificadaEn.Before(desde) ||
		datos.VerificadaEn.After(instante) || datos.Decision.EmitidaEn.After(datos.VerificadaEn) {
		return ErrConfirmacionInvalida
	}
	decision := datos.Decision
	if decision.Accion != accion || decision.ModuloID != recurso.ModuloID ||
		decision.TipoRecurso != recurso.Tipo || decision.RecursoRef != recurso.Referencia ||
		decision.ContextoRecursoHuellaSHA256 != huellaRecurso ||
		decision.Finalidad != finalidad || decision.CorrelacionRef != valorCorrelacion ||
		!listasIguales(decision.CamposPermitidos, campos) {
		return ErrConfirmacionInvalida
	}
	return nil
}

func listasIguales(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	primera, segunda := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(primera)
	sort.Strings(segunda)
	for indice := range primera {
		if primera[indice] != segunda[indice] {
			return false
		}
	}
	return true
}
