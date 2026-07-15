package ports

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	// ErrAutorizacionEjecucionDocumentalV4Invalida representa una denegacion
	// cerrada. Ausencia, ambiguedad, caducidad o cualquier discrepancia producen
	// el mismo error y nunca una concesion parcial.
	ErrAutorizacionEjecucionDocumentalV4Invalida = errors.New("vec: autorizacion de ejecucion documental v4 invalida")
	// ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida impide usar una
	// solicitud estructural o su proyeccion transaccional como credencial de red.
	ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida = errors.New("vec: serializacion de autorizacion de ejecucion documental v4 prohibida")
)

const (
	EsquemaSolicitudVinculadaAutorizacionEjecucionDocumentalV4  = "vec.documentos.autorizacion-ejecucion.solicitud-vinculada.v4"
	EsquemaSolicitudAplicacionAutorizacionEjecucionDocumentalV4 = "vec.documentos.autorizacion-ejecucion.solicitud-aplicacion.v4"

	// AccionEjecutarPlanDocumentalV4 es deliberadamente una unica accion
	// positiva. No se acepta una accion aportada libremente ni un comodin.
	AccionEjecutarPlanDocumentalV4 = "vec.documentos.ejecucion.ejecutar_plan_v4"

	AtributoAutorizacionDocumentalEfectoRef        = "ejecucion_documental_efecto_ref"
	AtributoAutorizacionDocumentalHuellaPlanSHA256 = "ejecucion_documental_huella_plan_sha256"
)

// ExpectativaAutorizacionEjecucionDocumentalV4 contiene los valores resueltos
// por el servidor que deben coincidir exactamente con la decision. No concede
// autoridad y tampoco acredita la procedencia criptografica del PDP.
//
// Recurso debe contener al menos un ambito explicito y exactamente los dos
// atributos de efecto y plan definidos arriba. Los valores sensibles de ambito
// deben llegar tokenizados o protegidos con HMAC antes de construirlo.
type ExpectativaAutorizacionEjecucionDocumentalV4 struct {
	// DecisionEsperada debe proceder del registro/resolucion confiable del
	// servidor, no de la peticion. Su huella completa se coteja con la evidencia
	// para comprometer tambien asignacion, rol, controles, catalogo, politicas y
	// garantia; no solo los campos funcionales repetidos debajo.
	DecisionEsperada                domain.DecisionAutorizacion
	PrincipalID                     string
	PerfilActivoRef                 string
	AutenticacionRef                string
	SesionRef                       string
	ControlSesionRef                string
	ControlSesionRevision           uint64
	ControlSesionHuellaSHA256       string
	ContextoActorRef                string
	ContextoActorVersion            uint64
	ContextoActorHuellaSHA256       string
	Recurso                         domain.RecursoAutorizable
	Finalidad                       string
	CorrelacionRef                  string
	EfectoRef                       string
	HuellaPlanSHA256                string
	CamposPermitidosEsperados       []string
	ObligacionesEsperadas           []string
	CumplimientosObligacionesPorRef map[string]string
}

type datosSolicitudVinculadaAutorizacionEjecucionDocumentalV4 struct {
	esquema                         string
	principalID                     string
	perfilActivoRef                 string
	autenticacionRef                string
	sesionRef                       string
	controlSesionRef                string
	controlSesionRevision           uint64
	controlSesionHuellaSHA256       string
	contextoActorRef                string
	contextoActorVersion            uint64
	contextoActorHuellaSHA256       string
	accion                          string
	recurso                         domain.RecursoAutorizable
	huellaRecursoSHA256             string
	huellaAmbitosSHA256             string
	finalidad                       string
	correlacionRef                  string
	efectoRef                       string
	huellaPlanSHA256                string
	camposPermitidos                []string
	obligaciones                    []string
	cumplimientosObligacionesPorRef map[string]string
	huellaCamposPermitidosSHA256    string
	huellaObligacionesSHA256        string
	huellaCumplimientosSHA256       string
	esquemaHuellaDecision           string
	huellaDecisionSHA256            string
	decisionRef                     string
	verificadaEn                    time.Time
	vinculadaEn                     time.Time
	validaHasta                     time.Time
	evidencia                       EvidenciaUsoDecisionAutorizacion
	huellaSolicitudVinculadaSHA256  string
}

// SolicitudVinculadaAutorizacionEjecucionDocumentalV4 es una comprobacion
// estructural, opaca e inmutable. No es una capacidad ni concede autoridad: la
// evidencia de la que parte puede construirse desde un DTO publico. Su valor
// cero se deniega y no puede reconstruirse desde una proyeccion persistida.
//
// La autoridad de composicion final debe nacer dentro de un conector
// homologado que implemente ConectorEjecucionDocumentalAtestadaV4. El nucleo
// nunca convierte esta comprobacion estructural en una concesion por si solo.
type SolicitudVinculadaAutorizacionEjecucionDocumentalV4 struct {
	datos *datosSolicitudVinculadaAutorizacionEjecucionDocumentalV4
}

// NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4 estrecha una
// evidencia estructural ya concedida. Nunca amplia su accion, actor, recurso,
// finalidad, ambitos, campos u obligaciones. vinculadaEn debe proceder del
// reloj confiable del servidor. El resultado sigue sin ser autoridad.
//
// EvidenciaUsoDecisionAutorizacion actualmente deniega toda obligacion no
// vacia porque aun no existe una prueba tipada de cumplimiento. Este puente
// conserva esa garantia: una obligacion esperada o un cumplimiento espurio se
// deniegan hasta que el contrato raiz pueda acreditarlos positivamente.
func NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4(
	evidencia EvidenciaUsoDecisionAutorizacion,
	expectativa ExpectativaAutorizacionEjecucionDocumentalV4,
	vinculadaEn time.Time,
) (SolicitudVinculadaAutorizacionEjecucionDocumentalV4, error) {
	expectativa = clonarExpectativaAutorizacionEjecucionDocumentalV4(expectativa)
	datosEvidencia, err := evidencia.Datos()
	if err != nil || expectativa.validar() != nil ||
		!instanteEjecucionDocumentalV3Valido(vinculadaEn) ||
		evidencia.ValidarEn(vinculadaEn) != nil ||
		vinculadaEn.Before(datosEvidencia.VerificadaEn) {
		return SolicitudVinculadaAutorizacionEjecucionDocumentalV4{}, errorAutorizacionEjecucionDocumentalV4()
	}

	decision := datosEvidencia.Decision
	vinculo, err := decision.VinculoAutenticacionActor.Datos()
	huellaDecisionEsperada, errDecisionEsperada := huellaDecisionAutorizacionReforzadaV1(
		expectativa.DecisionEsperada,
	)
	huellaRecurso, errRecurso := expectativa.Recurso.HuellaContextoAutorizacionSHA256()
	huellaAmbitos := huellaMapaAutorizacionEjecucionDocumentalV4(
		"vec.documentos.autorizacion-ejecucion.ambitos.v4",
		expectativa.Recurso.Ambitos,
	)
	if err != nil || errDecisionEsperada != nil || errRecurso != nil ||
		!esSHA256Hexadecimal(huellaAmbitos) ||
		datosEvidencia.EsquemaHuella != EsquemaHuellaDecisionAutorizacionReforzadaV1 ||
		huellaDecisionEsperada != datosEvidencia.HuellaDecisionSHA256 ||
		!decision.VinculoAutenticacionActor.CoincideExactamenteCon(
			expectativa.DecisionEsperada.VinculoAutenticacionActor,
		) ||
		decision.ValidarEvidenciaInstantanea() != nil || !decision.Concedida || decision.Codigo != "concedida" ||
		decision.PrincipalID != expectativa.PrincipalID ||
		decision.PerfilActivoRef != expectativa.PerfilActivoRef ||
		decision.Accion != AccionEjecutarPlanDocumentalV4 ||
		decision.RecursoRef != expectativa.Recurso.Referencia ||
		decision.ModuloID != expectativa.Recurso.ModuloID ||
		decision.TipoRecurso != expectativa.Recurso.Tipo ||
		decision.ContextoRecursoHuellaSHA256 != huellaRecurso ||
		decision.Finalidad != expectativa.Finalidad ||
		decision.CorrelacionRef != expectativa.CorrelacionRef ||
		vinculo.PrincipalID != expectativa.PrincipalID ||
		vinculo.PerfilActivoRef != expectativa.PerfilActivoRef ||
		vinculo.AutenticacionRef != expectativa.AutenticacionRef ||
		vinculo.SesionRef != expectativa.SesionRef ||
		vinculo.ControlSesionRef != expectativa.ControlSesionRef ||
		vinculo.ControlSesionRevision != expectativa.ControlSesionRevision ||
		vinculo.ControlSesionHuellaSHA256 != expectativa.ControlSesionHuellaSHA256 ||
		vinculo.ContextoActorRef != expectativa.ContextoActorRef ||
		vinculo.ContextoActorVersion != expectativa.ContextoActorVersion ||
		vinculo.ContextoActorHuellaSHA256 != expectativa.ContextoActorHuellaSHA256 ||
		!listaExactaAutorizacionEjecucionDocumentalV4(
			decision.CamposPermitidos,
			expectativa.CamposPermitidosEsperados,
		) ||
		!listaExactaAutorizacionEjecucionDocumentalV4(
			decision.Obligaciones,
			expectativa.ObligacionesEsperadas,
		) ||
		!cumplimientosObligacionesEjecucionDocumentalV4Validos(
			expectativa.ObligacionesEsperadas,
			expectativa.CumplimientosObligacionesPorRef,
		) {
		return SolicitudVinculadaAutorizacionEjecucionDocumentalV4{}, errorAutorizacionEjecucionDocumentalV4()
	}

	campos := clonarOrdenarListaAutorizacionEjecucionDocumentalV4(
		expectativa.CamposPermitidosEsperados,
	)
	obligaciones := clonarOrdenarListaAutorizacionEjecucionDocumentalV4(
		expectativa.ObligacionesEsperadas,
	)
	cumplimientos := clonarMapaAutorizacionEjecucionDocumentalV4(
		expectativa.CumplimientosObligacionesPorRef,
	)
	datos := &datosSolicitudVinculadaAutorizacionEjecucionDocumentalV4{
		esquema:                         EsquemaSolicitudVinculadaAutorizacionEjecucionDocumentalV4,
		principalID:                     expectativa.PrincipalID,
		perfilActivoRef:                 expectativa.PerfilActivoRef,
		autenticacionRef:                expectativa.AutenticacionRef,
		sesionRef:                       expectativa.SesionRef,
		controlSesionRef:                expectativa.ControlSesionRef,
		controlSesionRevision:           expectativa.ControlSesionRevision,
		controlSesionHuellaSHA256:       expectativa.ControlSesionHuellaSHA256,
		contextoActorRef:                expectativa.ContextoActorRef,
		contextoActorVersion:            expectativa.ContextoActorVersion,
		contextoActorHuellaSHA256:       expectativa.ContextoActorHuellaSHA256,
		accion:                          AccionEjecutarPlanDocumentalV4,
		recurso:                         clonarRecursoAutorizacionEjecucionDocumentalV4(expectativa.Recurso),
		huellaRecursoSHA256:             huellaRecurso,
		huellaAmbitosSHA256:             huellaAmbitos,
		finalidad:                       expectativa.Finalidad,
		correlacionRef:                  expectativa.CorrelacionRef,
		efectoRef:                       expectativa.EfectoRef,
		huellaPlanSHA256:                expectativa.HuellaPlanSHA256,
		camposPermitidos:                campos,
		obligaciones:                    obligaciones,
		cumplimientosObligacionesPorRef: cumplimientos,
		huellaCamposPermitidosSHA256: huellaListaAutorizacionEjecucionDocumentalV4(
			"vec.documentos.autorizacion-ejecucion.campos.v4", campos,
		),
		huellaObligacionesSHA256: huellaListaAutorizacionEjecucionDocumentalV4(
			"vec.documentos.autorizacion-ejecucion.obligaciones.v4", obligaciones,
		),
		huellaCumplimientosSHA256: huellaMapaAutorizacionEjecucionDocumentalV4(
			"vec.documentos.autorizacion-ejecucion.cumplimientos.v4", cumplimientos,
		),
		esquemaHuellaDecision: datosEvidencia.EsquemaHuella,
		huellaDecisionSHA256:  datosEvidencia.HuellaDecisionSHA256,
		decisionRef:           decision.DecisionRef,
		verificadaEn:          datosEvidencia.VerificadaEn,
		vinculadaEn:           vinculadaEn,
		validaHasta:           decision.ValidaHasta,
		evidencia:             evidencia,
	}
	datos.huellaSolicitudVinculadaSHA256 = datos.calcularHuella()
	solicitud := SolicitudVinculadaAutorizacionEjecucionDocumentalV4{datos: datos}
	if solicitud.validarEstructura() != nil {
		return SolicitudVinculadaAutorizacionEjecucionDocumentalV4{}, errorAutorizacionEjecucionDocumentalV4()
	}
	return solicitud, nil
}

func (e ExpectativaAutorizacionEjecucionDocumentalV4) validar() error {
	huellaDecisionEsperada, errDecisionEsperada := huellaDecisionAutorizacionReforzadaV1(
		e.DecisionEsperada,
	)
	if errDecisionEsperada != nil || !esSHA256Hexadecimal(huellaDecisionEsperada) ||
		!e.DecisionEsperada.Concedida || e.DecisionEsperada.Codigo != "concedida" ||
		e.DecisionEsperada.PrincipalID != e.PrincipalID ||
		e.DecisionEsperada.PerfilActivoRef != e.PerfilActivoRef ||
		e.DecisionEsperada.Accion != AccionEjecutarPlanDocumentalV4 ||
		e.DecisionEsperada.RecursoRef != e.Recurso.Referencia ||
		e.DecisionEsperada.ModuloID != e.Recurso.ModuloID ||
		e.DecisionEsperada.TipoRecurso != e.Recurso.Tipo ||
		e.DecisionEsperada.Finalidad != e.Finalidad ||
		e.DecisionEsperada.CorrelacionRef != e.CorrelacionRef ||
		!listaExactaAutorizacionEjecucionDocumentalV4(
			e.DecisionEsperada.CamposPermitidos,
			e.CamposPermitidosEsperados,
		) ||
		!listaExactaAutorizacionEjecucionDocumentalV4(
			e.DecisionEsperada.Obligaciones,
			e.ObligacionesEsperadas,
		) ||
		!referenciaOpacaAlmacenValida(e.PrincipalID, 512) ||
		!referenciaOpacaAlmacenValida(e.PerfilActivoRef, 512) ||
		!referenciaOpacaAlmacenValida(e.AutenticacionRef, 512) ||
		!referenciaOpacaAlmacenValida(e.SesionRef, 512) ||
		!referenciaOpacaAlmacenValida(e.ControlSesionRef, 512) ||
		e.ControlSesionRevision == 0 || !esSHA256Hexadecimal(e.ControlSesionHuellaSHA256) ||
		!referenciaOpacaAlmacenValida(e.ContextoActorRef, 512) ||
		e.ContextoActorVersion == 0 || !esSHA256Hexadecimal(e.ContextoActorHuellaSHA256) ||
		e.Recurso.Validar() != nil || contieneComodinRecursoAlmacen(e.Recurso) ||
		len(e.Recurso.Ambitos) == 0 || len(e.Recurso.Atributos) != 2 ||
		!referenciaOpacaAlmacenValida(e.Finalidad, 512) ||
		!referenciaOpacaAlmacenValida(e.CorrelacionRef, 512) ||
		!referenciaEjecucionDocumentalV3Valida(e.EfectoRef) ||
		!esSHA256Hexadecimal(e.HuellaPlanSHA256) ||
		e.Recurso.Atributos[AtributoAutorizacionDocumentalEfectoRef] != e.EfectoRef ||
		e.Recurso.Atributos[AtributoAutorizacionDocumentalHuellaPlanSHA256] != e.HuellaPlanSHA256 ||
		!listaExactaAutorizacionEjecucionDocumentalV4(
			e.CamposPermitidosEsperados,
			e.CamposPermitidosEsperados,
		) ||
		!listaExactaAutorizacionEjecucionDocumentalV4(
			e.ObligacionesEsperadas,
			e.ObligacionesEsperadas,
		) ||
		!cumplimientosObligacionesEjecucionDocumentalV4Validos(
			e.ObligacionesEsperadas,
			e.CumplimientosObligacionesPorRef,
		) {
		return ErrAutorizacionEjecucionDocumentalV4Invalida
	}
	return nil
}

func (s SolicitudVinculadaAutorizacionEjecucionDocumentalV4) ValidarEn(instante time.Time) error {
	if s.validarEstructura() != nil || !instanteEjecucionDocumentalV3Valido(instante) ||
		instante.Before(s.datos.vinculadaEn) || !instante.Before(s.datos.validaHasta) ||
		s.datos.evidencia.ValidarEn(instante) != nil {
		return errorAutorizacionEjecucionDocumentalV4()
	}
	return nil
}

func (s SolicitudVinculadaAutorizacionEjecucionDocumentalV4) HuellaSHA256() (string, error) {
	if s.validarEstructura() != nil {
		return "", errorAutorizacionEjecucionDocumentalV4()
	}
	return s.datos.huellaSolicitudVinculadaSHA256, nil
}

func (s SolicitudVinculadaAutorizacionEjecucionDocumentalV4) validarEstructura() error {
	if s.datos == nil {
		return ErrAutorizacionEjecucionDocumentalV4Invalida
	}
	d := s.datos
	datosEvidencia, errEvidencia := d.evidencia.Datos()
	expectativa := ExpectativaAutorizacionEjecucionDocumentalV4{
		DecisionEsperada: datosEvidencia.Decision,
		PrincipalID:      d.principalID, PerfilActivoRef: d.perfilActivoRef,
		AutenticacionRef: d.autenticacionRef, SesionRef: d.sesionRef,
		ControlSesionRef: d.controlSesionRef, ControlSesionRevision: d.controlSesionRevision,
		ControlSesionHuellaSHA256: d.controlSesionHuellaSHA256,
		ContextoActorRef:          d.contextoActorRef, ContextoActorVersion: d.contextoActorVersion,
		ContextoActorHuellaSHA256: d.contextoActorHuellaSHA256,
		Recurso:                   d.recurso, Finalidad: d.finalidad, CorrelacionRef: d.correlacionRef,
		EfectoRef: d.efectoRef, HuellaPlanSHA256: d.huellaPlanSHA256,
		CamposPermitidosEsperados:       d.camposPermitidos,
		ObligacionesEsperadas:           d.obligaciones,
		CumplimientosObligacionesPorRef: d.cumplimientosObligacionesPorRef,
	}
	huellaRecurso, errRecurso := d.recurso.HuellaContextoAutorizacionSHA256()
	if d.esquema != EsquemaSolicitudVinculadaAutorizacionEjecucionDocumentalV4 ||
		d.accion != AccionEjecutarPlanDocumentalV4 || expectativa.validar() != nil ||
		errRecurso != nil || huellaRecurso != d.huellaRecursoSHA256 ||
		huellaMapaAutorizacionEjecucionDocumentalV4(
			"vec.documentos.autorizacion-ejecucion.ambitos.v4", d.recurso.Ambitos,
		) != d.huellaAmbitosSHA256 ||
		huellaListaAutorizacionEjecucionDocumentalV4(
			"vec.documentos.autorizacion-ejecucion.campos.v4", d.camposPermitidos,
		) != d.huellaCamposPermitidosSHA256 ||
		huellaListaAutorizacionEjecucionDocumentalV4(
			"vec.documentos.autorizacion-ejecucion.obligaciones.v4", d.obligaciones,
		) != d.huellaObligacionesSHA256 ||
		huellaMapaAutorizacionEjecucionDocumentalV4(
			"vec.documentos.autorizacion-ejecucion.cumplimientos.v4",
			d.cumplimientosObligacionesPorRef,
		) != d.huellaCumplimientosSHA256 ||
		!instanteEjecucionDocumentalV3Valido(d.verificadaEn) ||
		!instanteEjecucionDocumentalV3Valido(d.vinculadaEn) ||
		!instanteEjecucionDocumentalV3Valido(d.validaHasta) ||
		d.vinculadaEn.Before(d.verificadaEn) || !d.vinculadaEn.Before(d.validaHasta) ||
		d.evidencia.ValidarEn(d.vinculadaEn) != nil || errEvidencia != nil ||
		datosEvidencia.EsquemaHuella != d.esquemaHuellaDecision ||
		datosEvidencia.HuellaDecisionSHA256 != d.huellaDecisionSHA256 ||
		!datosEvidencia.VerificadaEn.Equal(d.verificadaEn) ||
		datosEvidencia.Decision.DecisionRef != d.decisionRef ||
		!datosEvidencia.Decision.ValidaHasta.Equal(d.validaHasta) ||
		!d.coincideDecision(datosEvidencia.Decision) ||
		!esSHA256Hexadecimal(d.huellaSolicitudVinculadaSHA256) ||
		d.calcularHuella() != d.huellaSolicitudVinculadaSHA256 {
		return ErrAutorizacionEjecucionDocumentalV4Invalida
	}
	return nil
}

func (d *datosSolicitudVinculadaAutorizacionEjecucionDocumentalV4) coincideDecision(
	decision domain.DecisionAutorizacion,
) bool {
	vinculo, err := decision.VinculoAutenticacionActor.Datos()
	return err == nil && decision.ValidarEvidenciaInstantanea() == nil && decision.Concedida &&
		decision.Codigo == "concedida" && decision.PrincipalID == d.principalID &&
		decision.PerfilActivoRef == d.perfilActivoRef && decision.Accion == d.accion &&
		decision.RecursoRef == d.recurso.Referencia && decision.ModuloID == d.recurso.ModuloID &&
		decision.TipoRecurso == d.recurso.Tipo &&
		decision.ContextoRecursoHuellaSHA256 == d.huellaRecursoSHA256 &&
		decision.Finalidad == d.finalidad && decision.CorrelacionRef == d.correlacionRef &&
		listaExactaAutorizacionEjecucionDocumentalV4(decision.CamposPermitidos, d.camposPermitidos) &&
		listaExactaAutorizacionEjecucionDocumentalV4(decision.Obligaciones, d.obligaciones) &&
		vinculo.PrincipalID == d.principalID && vinculo.PerfilActivoRef == d.perfilActivoRef &&
		vinculo.AutenticacionRef == d.autenticacionRef && vinculo.SesionRef == d.sesionRef &&
		vinculo.ControlSesionRef == d.controlSesionRef &&
		vinculo.ControlSesionRevision == d.controlSesionRevision &&
		vinculo.ControlSesionHuellaSHA256 == d.controlSesionHuellaSHA256 &&
		vinculo.ContextoActorRef == d.contextoActorRef &&
		vinculo.ContextoActorVersion == d.contextoActorVersion &&
		vinculo.ContextoActorHuellaSHA256 == d.contextoActorHuellaSHA256
}

func (d *datosSolicitudVinculadaAutorizacionEjecucionDocumentalV4) calcularHuella() string {
	return huellaCanonicaFormatoDocumental([]string{
		EsquemaSolicitudVinculadaAutorizacionEjecucionDocumentalV4,
		d.principalID, d.perfilActivoRef, d.autenticacionRef, d.sesionRef,
		d.controlSesionRef, strconv.FormatUint(d.controlSesionRevision, 10),
		d.controlSesionHuellaSHA256, d.contextoActorRef,
		strconv.FormatUint(d.contextoActorVersion, 10), d.contextoActorHuellaSHA256,
		d.accion, d.recurso.Referencia, d.recurso.ModuloID, d.recurso.Tipo,
		d.huellaRecursoSHA256, d.huellaAmbitosSHA256, d.finalidad, d.correlacionRef,
		d.efectoRef, d.huellaPlanSHA256, d.huellaCamposPermitidosSHA256,
		d.huellaObligacionesSHA256, d.huellaCumplimientosSHA256,
		d.esquemaHuellaDecision, d.huellaDecisionSHA256, d.decisionRef,
		d.verificadaEn.Format(time.RFC3339Nano), d.vinculadaEn.Format(time.RFC3339Nano),
		d.validaHasta.Format(time.RFC3339Nano),
	})
}

// ClaveAplicacionAutorizacionEjecucionDocumentalV4 liga de forma exacta la
// decision, el plan y el efecto. No es autoridad. El adaptador duradero debera
// comprobar la terna y reclamar UNIQUE(DecisionRef) dentro del mismo COMMIT que
// aplique el efecto de negocio.
type ClaveAplicacionAutorizacionEjecucionDocumentalV4 struct {
	DecisionRef      string
	HuellaPlanSHA256 string
	EfectoRef        string
}

func (c ClaveAplicacionAutorizacionEjecucionDocumentalV4) validar() error {
	if !referenciaEjecucionDocumentalV3Valida(c.DecisionRef) ||
		!esSHA256Hexadecimal(c.HuellaPlanSHA256) ||
		!referenciaEjecucionDocumentalV3Valida(c.EfectoRef) {
		return ErrAutorizacionEjecucionDocumentalV4Invalida
	}
	return nil
}

type datosSolicitudAplicacionAutorizacionEjecucionDocumentalV4 struct {
	vinculo      SolicitudVinculadaAutorizacionEjecucionDocumentalV4
	solicitadaEn time.Time
	huella       string
}

// SolicitudAplicacionAutorizacionEjecucionDocumentalV4 es una peticion opaca
// y no autoritativa preparada para el registro atomico posterior. Un adaptador
// no debe aceptarla como prueba autosuficiente: la autoridad interna que la
// presenta y la evidencia deben revalidarse en la misma operacion duradera.
type SolicitudAplicacionAutorizacionEjecucionDocumentalV4 struct {
	datos *datosSolicitudAplicacionAutorizacionEjecucionDocumentalV4
}

// ProyeccionAplicacionAutorizacionEjecucionDocumentalV4 es una copia defensiva
// y no autoritativa para mapear columnas dentro de la transaccion. No permite
// reconstruir la solicitud opaca ni demuestra procedencia del PDP.
type ProyeccionAplicacionAutorizacionEjecucionDocumentalV4 struct {
	Esquema                         string
	Clave                           ClaveAplicacionAutorizacionEjecucionDocumentalV4
	EsquemaHuellaDecision           string
	HuellaDecisionSHA256            string
	PerfilActivoRef                 string
	ContextoActorHuellaSHA256       string
	Accion                          string
	RecursoRef                      string
	ModuloID                        string
	TipoRecurso                     string
	HuellaRecursoSHA256             string
	HuellaAmbitosSHA256             string
	Finalidad                       string
	CorrelacionRef                  string
	HuellaCamposPermitidosSHA256    string
	HuellaObligacionesSHA256        string
	HuellaCumplimientosSHA256       string
	VerificadaEn                    time.Time
	VinculadaEn                     time.Time
	SolicitadaEn                    time.Time
	ValidaHasta                     time.Time
	HuellaSolicitudVinculadaSHA256  string
	HuellaSolicitudAplicacionSHA256 string
}

func (s SolicitudVinculadaAutorizacionEjecucionDocumentalV4) PrepararSolicitudAplicacionEn(
	instante time.Time,
) (SolicitudAplicacionAutorizacionEjecucionDocumentalV4, error) {
	if s.ValidarEn(instante) != nil {
		return SolicitudAplicacionAutorizacionEjecucionDocumentalV4{}, errorAutorizacionEjecucionDocumentalV4()
	}
	datos := &datosSolicitudAplicacionAutorizacionEjecucionDocumentalV4{
		vinculo: s, solicitadaEn: instante,
	}
	datos.huella = datos.calcularHuella()
	solicitud := SolicitudAplicacionAutorizacionEjecucionDocumentalV4{datos: datos}
	if solicitud.validarEstructura() != nil {
		return SolicitudAplicacionAutorizacionEjecucionDocumentalV4{}, errorAutorizacionEjecucionDocumentalV4()
	}
	return solicitud, nil
}

func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) ValidarContraEn(
	decisionRef, huellaPlanSHA256, efectoRef string,
	instante time.Time,
) error {
	if s.validarEstructura() != nil || !instanteEjecucionDocumentalV3Valido(instante) ||
		instante.Before(s.datos.solicitadaEn) || s.datos.vinculo.ValidarEn(instante) != nil {
		return errorAutorizacionEjecucionDocumentalV4()
	}
	clave := s.clave()
	if clave.DecisionRef != decisionRef || clave.HuellaPlanSHA256 != huellaPlanSHA256 ||
		clave.EfectoRef != efectoRef {
		return errorAutorizacionEjecucionDocumentalV4()
	}
	return nil
}

func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) ProyeccionParaTransaccion() (
	ProyeccionAplicacionAutorizacionEjecucionDocumentalV4,
	error,
) {
	if s.validarEstructura() != nil {
		return ProyeccionAplicacionAutorizacionEjecucionDocumentalV4{}, errorAutorizacionEjecucionDocumentalV4()
	}
	d := s.datos.vinculo.datos
	return ProyeccionAplicacionAutorizacionEjecucionDocumentalV4{
		Esquema: EsquemaSolicitudAplicacionAutorizacionEjecucionDocumentalV4, Clave: s.clave(),
		EsquemaHuellaDecision: d.esquemaHuellaDecision,
		HuellaDecisionSHA256:  d.huellaDecisionSHA256, PerfilActivoRef: d.perfilActivoRef,
		ContextoActorHuellaSHA256: d.contextoActorHuellaSHA256, Accion: d.accion,
		RecursoRef: d.recurso.Referencia, ModuloID: d.recurso.ModuloID, TipoRecurso: d.recurso.Tipo,
		HuellaRecursoSHA256: d.huellaRecursoSHA256, HuellaAmbitosSHA256: d.huellaAmbitosSHA256,
		Finalidad: d.finalidad, CorrelacionRef: d.correlacionRef,
		HuellaCamposPermitidosSHA256: d.huellaCamposPermitidosSHA256,
		HuellaObligacionesSHA256:     d.huellaObligacionesSHA256,
		HuellaCumplimientosSHA256:    d.huellaCumplimientosSHA256,
		VerificadaEn:                 d.verificadaEn, VinculadaEn: d.vinculadaEn,
		SolicitadaEn: s.datos.solicitadaEn, ValidaHasta: d.validaHasta,
		HuellaSolicitudVinculadaSHA256:  d.huellaSolicitudVinculadaSHA256,
		HuellaSolicitudAplicacionSHA256: s.datos.huella,
	}, nil
}

// EvidenciaEstructural devuelve la evidencia defensiva necesaria para que la
// capa de aplicacion vuelva a comprobarla. No convierte la solicitud en una
// concesion ni demuestra por si sola que la decision proceda del PDP.
func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) EvidenciaEstructural() (
	EvidenciaUsoDecisionAutorizacion,
	error,
) {
	if s.validarEstructura() != nil {
		return EvidenciaUsoDecisionAutorizacion{}, errorAutorizacionEjecucionDocumentalV4()
	}
	return s.datos.vinculo.datos.evidencia, nil
}

func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) validarEstructura() error {
	if s.datos == nil || s.datos.vinculo.validarEstructura() != nil ||
		!instanteEjecucionDocumentalV3Valido(s.datos.solicitadaEn) ||
		s.datos.vinculo.ValidarEn(s.datos.solicitadaEn) != nil ||
		!esSHA256Hexadecimal(s.datos.huella) || s.datos.calcularHuella() != s.datos.huella ||
		s.clave().validar() != nil {
		return ErrAutorizacionEjecucionDocumentalV4Invalida
	}
	return nil
}

func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) clave() ClaveAplicacionAutorizacionEjecucionDocumentalV4 {
	if s.datos == nil || s.datos.vinculo.datos == nil {
		return ClaveAplicacionAutorizacionEjecucionDocumentalV4{}
	}
	d := s.datos.vinculo.datos
	return ClaveAplicacionAutorizacionEjecucionDocumentalV4{
		DecisionRef: d.decisionRef, HuellaPlanSHA256: d.huellaPlanSHA256, EfectoRef: d.efectoRef,
	}
}

func (d *datosSolicitudAplicacionAutorizacionEjecucionDocumentalV4) calcularHuella() string {
	if d == nil || d.vinculo.datos == nil {
		return ""
	}
	clave := ClaveAplicacionAutorizacionEjecucionDocumentalV4{
		DecisionRef:      d.vinculo.datos.decisionRef,
		HuellaPlanSHA256: d.vinculo.datos.huellaPlanSHA256,
		EfectoRef:        d.vinculo.datos.efectoRef,
	}
	return huellaCanonicaFormatoDocumental([]string{
		EsquemaSolicitudAplicacionAutorizacionEjecucionDocumentalV4,
		d.vinculo.datos.huellaSolicitudVinculadaSHA256,
		clave.DecisionRef, clave.HuellaPlanSHA256, clave.EfectoRef,
		d.solicitadaEn.Format(time.RFC3339Nano),
	})
}

func listaExactaAutorizacionEjecucionDocumentalV4(recibida, esperada []string) bool {
	return camposAutorizacionExactos(recibida, esperada)
}

func cumplimientosObligacionesEjecucionDocumentalV4Validos(
	obligaciones []string,
	cumplimientos map[string]string,
) bool {
	if len(obligaciones) != len(cumplimientos) ||
		!listaExactaAutorizacionEjecucionDocumentalV4(obligaciones, obligaciones) {
		return false
	}
	for _, obligacion := range obligaciones {
		referencia, existe := cumplimientos[obligacion]
		if !existe || !referenciaOpacaAlmacenValida(referencia, 512) ||
			contieneComodinContextoAlmacen(obligacion, referencia) {
			return false
		}
	}
	for obligacion := range cumplimientos {
		encontrada := false
		for _, esperada := range obligaciones {
			if obligacion == esperada {
				encontrada = true
				break
			}
		}
		if !encontrada {
			return false
		}
	}
	return true
}

func huellaListaAutorizacionEjecucionDocumentalV4(esquema string, valores []string) string {
	ordenados := clonarOrdenarListaAutorizacionEjecucionDocumentalV4(valores)
	return huellaCanonicaFormatoDocumental(append([]string{esquema}, ordenados...))
}

func huellaMapaAutorizacionEjecucionDocumentalV4(esquema string, valores map[string]string) string {
	claves := make([]string, 0, len(valores))
	for clave := range valores {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	canonicos := []string{esquema}
	for _, clave := range claves {
		canonicos = append(canonicos, clave, valores[clave])
	}
	return huellaCanonicaFormatoDocumental(canonicos)
}

// HuellaObligacionesVaciasAutorizacionEjecucionDocumentalV4 fija el unico
// valor admitido mientras el flujo V4 no persista evidencia tipada de
// cumplimiento y revocacion de obligaciones.
func HuellaObligacionesVaciasAutorizacionEjecucionDocumentalV4() string {
	return huellaListaAutorizacionEjecucionDocumentalV4(
		"vec.documentos.autorizacion-ejecucion.obligaciones.v4", nil,
	)
}

// HuellaCumplimientosVaciosAutorizacionEjecucionDocumentalV4 acompana a la
// lista de obligaciones vacia; un mapa no vacio se deniega.
func HuellaCumplimientosVaciosAutorizacionEjecucionDocumentalV4() string {
	return huellaMapaAutorizacionEjecucionDocumentalV4(
		"vec.documentos.autorizacion-ejecucion.cumplimientos.v4", nil,
	)
}

func clonarExpectativaAutorizacionEjecucionDocumentalV4(
	e ExpectativaAutorizacionEjecucionDocumentalV4,
) ExpectativaAutorizacionEjecucionDocumentalV4 {
	e.DecisionEsperada = clonarDecisionAutorizacionCanonica(e.DecisionEsperada)
	e.Recurso = clonarRecursoAutorizacionEjecucionDocumentalV4(e.Recurso)
	e.CamposPermitidosEsperados = append([]string(nil), e.CamposPermitidosEsperados...)
	e.ObligacionesEsperadas = append([]string(nil), e.ObligacionesEsperadas...)
	e.CumplimientosObligacionesPorRef = clonarMapaAutorizacionEjecucionDocumentalV4(
		e.CumplimientosObligacionesPorRef,
	)
	return e
}

func clonarRecursoAutorizacionEjecucionDocumentalV4(
	recurso domain.RecursoAutorizable,
) domain.RecursoAutorizable {
	recurso.Ambitos = clonarMapaAutorizacionEjecucionDocumentalV4(recurso.Ambitos)
	recurso.Atributos = clonarMapaAutorizacionEjecucionDocumentalV4(recurso.Atributos)
	return recurso
}

func clonarMapaAutorizacionEjecucionDocumentalV4(origen map[string]string) map[string]string {
	copia := make(map[string]string, len(origen))
	for clave, valor := range origen {
		copia[clave] = valor
	}
	return copia
}

func clonarOrdenarListaAutorizacionEjecucionDocumentalV4(origen []string) []string {
	copia := append([]string(nil), origen...)
	sort.Strings(copia)
	return copia
}

func errorAutorizacionEjecucionDocumentalV4() error {
	return errors.Join(domain.ErrAutorizacionDenegada, ErrAutorizacionEjecucionDocumentalV4Invalida)
}

func (SolicitudVinculadaAutorizacionEjecucionDocumentalV4) String() string {
	return "[SOLICITUD-VINCULADA-AUTORIZACION-EJECUCION-DOCUMENTAL-V4-OPACA]"
}
func (s SolicitudVinculadaAutorizacionEjecucionDocumentalV4) GoString() string { return s.String() }
func (s SolicitudVinculadaAutorizacionEjecucionDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudVinculadaAutorizacionEjecucionDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SolicitudVinculadaAutorizacionEjecucionDocumentalV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida
}
func (*SolicitudVinculadaAutorizacionEjecucionDocumentalV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida
}
func (SolicitudVinculadaAutorizacionEjecucionDocumentalV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida
}
func (*SolicitudVinculadaAutorizacionEjecucionDocumentalV4) UnmarshalText([]byte) error {
	return ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida
}

func (SolicitudAplicacionAutorizacionEjecucionDocumentalV4) String() string {
	return "[SOLICITUD-APLICACION-AUTORIZACION-EJECUCION-DOCUMENTAL-V4-OPACA]"
}
func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) GoString() string { return s.String() }
func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudAplicacionAutorizacionEjecucionDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SolicitudAplicacionAutorizacionEjecucionDocumentalV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida
}
func (*SolicitudAplicacionAutorizacionEjecucionDocumentalV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida
}
func (SolicitudAplicacionAutorizacionEjecucionDocumentalV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida
}
func (*SolicitudAplicacionAutorizacionEjecucionDocumentalV4) UnmarshalText([]byte) error {
	return ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida
}

func (ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) String() string {
	return "[PROYECCION-APLICACION-AUTORIZACION-EJECUCION-DOCUMENTAL-V4-INTERNA]"
}
func (p ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) GoString() string { return p.String() }
func (p ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (p ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
func (ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida
}
func (*ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida
}
func (ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida
}
func (*ProyeccionAplicacionAutorizacionEjecucionDocumentalV4) UnmarshalText([]byte) error {
	return ErrSerializacionAutorizacionEjecucionDocumentalV4Prohibida
}
