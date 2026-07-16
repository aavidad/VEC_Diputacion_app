package ports

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// VentanaMaximaUsoAutorizacionBaremacion limita el tiempo durante el que una
// decision ya evaluada puede viajar hasta el punto de aplicacion.
const VentanaMaximaUsoAutorizacionBaremacion = 30 * time.Second

// AccionOperacionBaremacion es una accion cerrada. Una decision concedida para
// una accion nunca habilita otra, aunque ambas actuen sobre el mismo recurso.
type AccionOperacionBaremacion string

const (
	AccionReservarAltaBaremacion                  AccionOperacionBaremacion = "bolsa.baremacion.alta.reservar"
	AccionConfirmarAltaBaremacion                 AccionOperacionBaremacion = "bolsa.baremacion.alta.confirmar"
	AccionAbandonarAltaBaremacion                 AccionOperacionBaremacion = "bolsa.baremacion.alta.abandonar"
	AccionReservarDecisionBaremacion              AccionOperacionBaremacion = "bolsa.baremacion.decision.reservar"
	AccionPrevalidarArchivoProbatorioBaremacion   AccionOperacionBaremacion = "bolsa.baremacion.archivo.prevalidar"
	AccionConfirmarDecisionBaremacion             AccionOperacionBaremacion = "bolsa.baremacion.decision.confirmar"
	AccionAdoptarDecisionInicialBaremacion        AccionOperacionBaremacion = "bolsa.baremacion.decision.inicial.adoptar"
	AccionRectificarDecisionBaremacion            AccionOperacionBaremacion = "bolsa.baremacion.decision.rectificar"
	AccionRevocarDecisionBaremacion               AccionOperacionBaremacion = "bolsa.baremacion.decision.revocar"
	AccionRehabilitarDecisionBaremacion           AccionOperacionBaremacion = "bolsa.baremacion.decision.rehabilitar"
	AccionAbandonarDecisionBaremacion             AccionOperacionBaremacion = "bolsa.baremacion.decision.abandonar"
	AccionConsultarBaremacionVigente              AccionOperacionBaremacion = "bolsa.baremacion.vigente.consultar"
	AccionConsultarVersionBaremacion              AccionOperacionBaremacion = "bolsa.baremacion.version.consultar"
	AccionConsultarCriterioBaremacion             AccionOperacionBaremacion = "bolsa.criterio.consultar"
	AccionConsultarEvidenciaBaremacion            AccionOperacionBaremacion = "bolsa.evidencia.consultar"
	AccionConsultarRepresentacionBaremacion       AccionOperacionBaremacion = "bolsa.representacion.consultar"
	AccionCalcularPuntuacionBaremacion            AccionOperacionBaremacion = "bolsa.puntuacion.calcular"
	AccionRecuperarCalculoBaremacion              AccionOperacionBaremacion = "bolsa.puntuacion.calculo.recuperar"
	AccionConsultarPoliticaFirmaBaremacion        AccionOperacionBaremacion = "bolsa.firma.politica.consultar"
	AccionCodificarDecisionBaremacion             AccionOperacionBaremacion = "bolsa.decision.codificar"
	AccionCustodiarDecisionBaremacion             AccionOperacionBaremacion = "bolsa.decision.custodiar"
	AccionPrepararFirmaDecisionBaremacion         AccionOperacionBaremacion = "bolsa.decision.firma.preparar"
	AccionConsultarFirmaDecisionBaremacion        AccionOperacionBaremacion = "bolsa.decision.firma.consultar"
	AccionValidarFirmaDecisionBaremacion          AccionOperacionBaremacion = "bolsa.decision.firma.validar"
	AccionSellarTiempoDecisionBaremacion          AccionOperacionBaremacion = "bolsa.decision.firma.sellar_tiempo"
	AccionAumentarFirmaDecisionBaremacion         AccionOperacionBaremacion = "bolsa.decision.firma.aumentar"
	AccionRecuperarBinarioFirmadoBaremacion       AccionOperacionBaremacion = "bolsa.decision.firma.binario.recuperar"
	AccionCustodiarDocumentoFirmadoBaremacion     AccionOperacionBaremacion = "bolsa.decision.firma.documento.custodiar"
	AccionRetenerDocumentoFirmadoBaremacion       AccionOperacionBaremacion = "bolsa.decision.firma.documento.retener"
	AccionRecuperarArtefactoFirmaBaremacion       AccionOperacionBaremacion = "bolsa.decision.firma.artefacto.recuperar"
	AccionRecuperarValidacionFirmaBaremacion      AccionOperacionBaremacion = "bolsa.decision.firma.validacion.recuperar"
	AccionRecuperarSelloTiempoFirmaBaremacion     AccionOperacionBaremacion = "bolsa.decision.firma.sello_tiempo.recuperar"
	AccionRecuperarAumentoFirmaBaremacion         AccionOperacionBaremacion = "bolsa.decision.firma.aumento.recuperar"
	AccionConsultarEvidenciaTransaccionBaremacion AccionOperacionBaremacion = "bolsa.baremacion.transaccion.consultar"
)

// ClaseRecursoOperacionBaremacion impide reutilizar accidentalmente una
// capacidad entre clases con identificadores de aspecto similar.
type ClaseRecursoOperacionBaremacion string

const (
	ClaseRecursoBaremacion       ClaseRecursoOperacionBaremacion = "baremacion"
	ClaseRecursoProceso          ClaseRecursoOperacionBaremacion = "proceso"
	ClaseRecursoEvidencia        ClaseRecursoOperacionBaremacion = "evidencia"
	ClaseRecursoRepresentacion   ClaseRecursoOperacionBaremacion = "representacion"
	ClaseRecursoCalculo          ClaseRecursoOperacionBaremacion = "calculo"
	ClaseRecursoPoliticaFirma    ClaseRecursoOperacionBaremacion = "politica_firma"
	ClaseRecursoDecision         ClaseRecursoOperacionBaremacion = "decision"
	ClaseRecursoSesionFirma      ClaseRecursoOperacionBaremacion = "sesion_firma"
	ClaseRecursoArtefactoFirma   ClaseRecursoOperacionBaremacion = "artefacto_firma"
	ClaseRecursoValidacionFirma  ClaseRecursoOperacionBaremacion = "validacion_firma"
	ClaseRecursoSelloTiempo      ClaseRecursoOperacionBaremacion = "sello_tiempo"
	ClaseRecursoAumentoFirma     ClaseRecursoOperacionBaremacion = "aumento_firma"
	ClaseRecursoDocumentoFirmado ClaseRecursoOperacionBaremacion = "documento_firmado"
	ClaseRecursoTransaccion      ClaseRecursoOperacionBaremacion = "transaccion"
)

type especificacionAccionBaremacion struct {
	clase  ClaseRecursoOperacionBaremacion
	campos []string
}

var especificacionesAccionBaremacion = map[AccionOperacionBaremacion]especificacionAccionBaremacion{
	AccionReservarAltaBaremacion:                  {ClaseRecursoBaremacion, []string{"reserva.alta"}},
	AccionConfirmarAltaBaremacion:                 {ClaseRecursoBaremacion, []string{"baremacion", "evidencia_transaccion"}},
	AccionAbandonarAltaBaremacion:                 {ClaseRecursoBaremacion, []string{"reserva.alta"}},
	AccionReservarDecisionBaremacion:              {ClaseRecursoBaremacion, []string{"reserva.decision"}},
	AccionPrevalidarArchivoProbatorioBaremacion:   {ClaseRecursoBaremacion, []string{"archivo_probatorio"}},
	AccionConfirmarDecisionBaremacion:             {ClaseRecursoBaremacion, []string{"baremacion", "decision", "evidencia_transaccion"}},
	AccionAdoptarDecisionInicialBaremacion:        {ClaseRecursoBaremacion, []string{"decision.inicial.contenido"}},
	AccionRectificarDecisionBaremacion:            {ClaseRecursoBaremacion, []string{"decision.rectificacion.contenido"}},
	AccionRevocarDecisionBaremacion:               {ClaseRecursoBaremacion, []string{"decision.revocacion.contenido"}},
	AccionRehabilitarDecisionBaremacion:           {ClaseRecursoBaremacion, []string{"decision.rehabilitacion.contenido"}},
	AccionAbandonarDecisionBaremacion:             {ClaseRecursoBaremacion, []string{"reserva.decision"}},
	AccionConsultarBaremacionVigente:              {ClaseRecursoBaremacion, []string{"baremacion"}},
	AccionConsultarVersionBaremacion:              {ClaseRecursoBaremacion, []string{"baremacion"}},
	AccionConsultarCriterioBaremacion:             {ClaseRecursoProceso, []string{"criterio", "evidencia_consulta"}},
	AccionConsultarEvidenciaBaremacion:            {ClaseRecursoEvidencia, []string{"documento", "evidencia_consulta"}},
	AccionConsultarRepresentacionBaremacion:       {ClaseRecursoRepresentacion, []string{"representacion", "evidencia_consulta"}},
	AccionCalcularPuntuacionBaremacion:            {ClaseRecursoBaremacion, []string{"calculo", "evidencia_gobierno"}},
	AccionRecuperarCalculoBaremacion:              {ClaseRecursoCalculo, []string{"calculo", "evidencia_gobierno"}},
	AccionConsultarPoliticaFirmaBaremacion:        {ClaseRecursoPoliticaFirma, []string{"politica_firma"}},
	AccionCodificarDecisionBaremacion:             {ClaseRecursoDecision, []string{"documento_canonico"}},
	AccionCustodiarDecisionBaremacion:             {ClaseRecursoDecision, []string{"documento_custodiado", "evidencia_custodia"}},
	AccionPrepararFirmaDecisionBaremacion:         {ClaseRecursoDecision, []string{"sesion_firma", "evidencia_preparacion"}},
	AccionConsultarFirmaDecisionBaremacion:        {ClaseRecursoSesionFirma, []string{"estado_firma", "artefacto_firma", "evidencia_consulta"}},
	AccionValidarFirmaDecisionBaremacion:          {ClaseRecursoArtefactoFirma, []string{"validacion_firma", "evidencia_validacion"}},
	AccionSellarTiempoDecisionBaremacion:          {ClaseRecursoArtefactoFirma, []string{"sello_tiempo", "evidencia_sello"}},
	AccionAumentarFirmaDecisionBaremacion:         {ClaseRecursoArtefactoFirma, []string{"firma_longeva", "evidencia_aumento"}},
	AccionRecuperarBinarioFirmadoBaremacion:       {ClaseRecursoDocumentoFirmado, []string{"documento_firmado.binario", "evidencia_recuperacion"}},
	AccionCustodiarDocumentoFirmadoBaremacion:     {ClaseRecursoDocumentoFirmado, []string{"documento_firmado.custodia", "evidencia_custodia"}},
	AccionRetenerDocumentoFirmadoBaremacion:       {ClaseRecursoDocumentoFirmado, []string{"documento_firmado.retencion", "evidencia_retencion"}},
	AccionRecuperarArtefactoFirmaBaremacion:       {ClaseRecursoArtefactoFirma, []string{"artefacto_firma"}},
	AccionRecuperarValidacionFirmaBaremacion:      {ClaseRecursoValidacionFirma, []string{"validacion_firma"}},
	AccionRecuperarSelloTiempoFirmaBaremacion:     {ClaseRecursoSelloTiempo, []string{"sello_tiempo"}},
	AccionRecuperarAumentoFirmaBaremacion:         {ClaseRecursoAumentoFirma, []string{"firma_longeva", "evidencia_aumento"}},
	AccionConsultarEvidenciaTransaccionBaremacion: {ClaseRecursoTransaccion, []string{"auditoria", "evento_outbox", "evidencia_transaccion"}},
}

// VinculoAutenticacionBaremacion contiene hechos ya verificados de la sesion.
// No concede acceso: la unica fuente de concesion aceptada por el constructor
// es DecisionAutorizacion.
type VinculoAutenticacionBaremacion struct {
	SujetoRef                 string
	Metodo                    dominiovec.AuthMethod
	Garantia                  dominiovec.AuthAssurance
	AutenticacionRef          string
	SesionRef                 string
	SesionEmitidaEn           time.Time
	SesionValidaHasta         time.Time
	VinculoAutenticacionActor dominiovec.VinculoAutenticacionActorV1
}

func (v VinculoAutenticacionBaremacion) validar() (dominiovec.DatosVinculoAutenticacionActorV1, error) {
	datos, err := v.VinculoAutenticacionActor.Datos()
	if !referenciaValida(v.SujetoRef, 512) || !v.Metodo.Valido() || v.Metodo == dominiovec.AuthMethodDemo ||
		!v.Garantia.Valida() || !referenciaValida(v.AutenticacionRef, 512) ||
		!referenciaValida(v.SesionRef, 512) || err != nil ||
		v.Metodo != datos.MetodoObservado || v.Garantia != datos.GarantiaObservada ||
		v.AutenticacionRef != datos.AutenticacionRef || v.SesionRef != datos.SesionRef ||
		!v.SesionEmitidaEn.Equal(datos.SesionEmitidaEn) || !v.SesionValidaHasta.Equal(datos.SesionValidaHasta) ||
		v.SesionEmitidaEn.IsZero() || v.SesionValidaHasta.IsZero() ||
		!v.SesionValidaHasta.After(v.SesionEmitidaEn) {
		return dominiovec.DatosVinculoAutenticacionActorV1{}, ErrAutorizacionBaremacionInvalida
	}
	return datos, nil
}

type datosAutorizacionOperacionBaremacion struct {
	principalRef      string
	sujetoRef         string
	perfilActorRef    string
	metodo            dominiovec.AuthMethod
	garantia          dominiovec.AuthAssurance
	garantiaMinima    dominiovec.AuthAssurance
	autenticacionRef  string
	sesionRef         string
	sesionEmitidaEn   time.Time
	sesionValidaHasta time.Time
	vinculo           dominiovec.VinculoAutenticacionActorV1
	autorizacionRef   string
	accion            AccionOperacionBaremacion
	clase             ClaseRecursoOperacionBaremacion
	recursoRef        string
	finalidad         string
	correlacionRef    string
	campos            []string
	emitidaEn         time.Time
	validaHasta       time.Time
	evidencia         puertosvec.EvidenciaUsoDecisionAutorizacion
	recursoAlmacen    *dominiovec.RecursoAutorizable
}

// ContextoOperacionBaremacion es una capacidad opaca e inmutable. Su valor
// cero deniega y no existe un literal publico que pueda rellenar sus datos.
type ContextoOperacionBaremacion struct {
	datos *datosAutorizacionOperacionBaremacion
}

// EsNulo distingue la ausencia contractual de una capacidad invalida o ya no
// vigente sin exponer ninguno de sus datos internos. Es la unica comprobacion
// admisible para campos que deben estar exactamente ausentes.
func (c ContextoOperacionBaremacion) EsNulo() bool {
	return c.datos == nil
}

// ContextoConsultaBaremacion mantiene el nombre semantico sin crear una via de
// acceso menos exigente.
type ContextoConsultaBaremacion = ContextoOperacionBaremacion

// ProyeccionAutorizacionBaremacion es una copia solo de lectura para construir
// trazabilidad. Modificarla nunca modifica ni concede la capacidad original.
type ProyeccionAutorizacionBaremacion struct {
	PrincipalRef        string
	SujetoRef           string
	PerfilActorClave    string
	MetodoAutenticacion dominiovec.AuthMethod
	NivelAutenticacion  dominiovec.AuthAssurance
	GarantiaMinima      dominiovec.AuthAssurance
	AutenticacionRef    string
	SesionRef           string
	SesionEmitidaEn     time.Time
	SesionValidaHasta   time.Time
	AutorizacionRef     string
	Accion              AccionOperacionBaremacion
	ClaseRecurso        ClaseRecursoOperacionBaremacion
	RecursoRef          string
	FinalidadClave      string
	CorrelacionRef      string
	CamposPermitidos    []string
	EmitidaEn           time.Time
	ValidaHasta         time.Time
}

// NuevaAutorizacionOperacionBaremacion deriva la capacidad exclusivamente de
// una decision positiva y vigente. Las obligaciones se rechazan porque este
// contrato no implementa ninguna de forma implicita. La lista de campos debe
// coincidir exactamente con la definida para la accion.
func NuevaAutorizacionOperacionBaremacion(
	decision dominiovec.DecisionAutorizacion,
	vinculo VinculoAutenticacionBaremacion,
	instante time.Time,
) (ContextoOperacionBaremacion, error) {
	accion := AccionOperacionBaremacion(decision.Accion)
	especificacion, conocida := especificacionesAccionBaremacion[accion]
	if !conocida || accionRequiereRecursoAlmacenBaremacion(accion) {
		return ContextoOperacionBaremacion{}, ErrAutorizacionBaremacionInvalida
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: decision.RecursoRef,
		ModuloID:   "bolsa",
		Tipo:       string(especificacion.clase),
		Ambitos:    map[string]string{"sujeto_ref": vinculo.SujetoRef},
	}
	return nuevaAutorizacionOperacionBaremacion(decision, recurso, vinculo, instante, false)
}

// NuevaAutorizacionOperacionAlmacenBaremacion es la unica variante que admite
// las tres acciones de almacen del modulo. El recurso debe ser exactamente el
// que el servidor entrego al PDP, incluidos todos los vinculos tecnicos. Se
// conserva una copia defensiva para impedir que esos atributos se reconstruyan
// o amplien despues de emitir la decision.
func NuevaAutorizacionOperacionAlmacenBaremacion(
	decision dominiovec.DecisionAutorizacion,
	recurso dominiovec.RecursoAutorizable,
	vinculo VinculoAutenticacionBaremacion,
	instante time.Time,
) (ContextoOperacionBaremacion, error) {
	accion := AccionOperacionBaremacion(decision.Accion)
	if !accionRequiereRecursoAlmacenBaremacion(accion) {
		return ContextoOperacionBaremacion{}, ErrAutorizacionBaremacionInvalida
	}
	return nuevaAutorizacionOperacionBaremacion(decision, recurso, vinculo, instante, true)
}

func nuevaAutorizacionOperacionBaremacion(
	decision dominiovec.DecisionAutorizacion,
	recurso dominiovec.RecursoAutorizable,
	vinculo VinculoAutenticacionBaremacion,
	instante time.Time,
	recursoAlmacen bool,
) (ContextoOperacionBaremacion, error) {
	accion := AccionOperacionBaremacion(decision.Accion)
	especificacion, conocida := especificacionesAccionBaremacion[accion]
	datosVinculo, errVinculo := vinculo.validar()
	huellaContextoEsperada, errContexto := recurso.HuellaContextoAutorizacionSHA256()
	if decision.ValidarEvidenciaInstantanea() != nil || !decision.Concedida || !decision.VigenteEn(instante) || !conocida ||
		recursoAlmacen != accionRequiereRecursoAlmacenBaremacion(accion) ||
		decision.ModuloID != "bolsa" || decision.TipoRecurso != string(especificacion.clase) ||
		!recursoAutorizableBaremacionValido(recurso, decision, vinculo.SujetoRef, recursoAlmacen) ||
		errContexto != nil || decision.ContextoRecursoHuellaSHA256 != huellaContextoEsperada ||
		errVinculo != nil || !decision.VinculoAutenticacionActor.CoincideExactamenteCon(vinculo.VinculoAutenticacionActor) ||
		decision.PrincipalID != datosVinculo.PrincipalID || decision.PerfilActivoRef != datosVinculo.PerfilActivoRef ||
		!dominiovec.CumpleGarantiaAutenticacion(vinculo.Garantia, decision.GarantiaMinima) ||
		!referenciaValida(decision.PrincipalID, 512) || !referenciaValida(decision.PerfilActivoRef, 512) ||
		!referenciaValida(decision.DecisionRef, 512) || !referenciaValida(decision.RecursoRef, 512) ||
		!claveValida(decision.Finalidad) || !referenciaValida(decision.CorrelacionRef, 512) ||
		len(decision.Obligaciones) != 0 || !mismosCamposExactos(decision.CamposPermitidos, especificacion.campos) ||
		decision.EmitidaEn.UTC().Before(datosVinculo.SesionRevalidadaEn.UTC()) {
		return ContextoOperacionBaremacion{}, ErrAutorizacionBaremacionInvalida
	}
	validaHasta := decision.ValidaHasta.UTC()
	if vinculo.SesionValidaHasta.UTC().Before(validaHasta) {
		validaHasta = vinculo.SesionValidaHasta.UTC()
	}
	limiteUso := instante.UTC().Add(VentanaMaximaUsoAutorizacionBaremacion)
	if limiteUso.Before(validaHasta) {
		validaHasta = limiteUso
	}
	if !validaHasta.After(instante.UTC()) {
		return ContextoOperacionBaremacion{}, ErrAutorizacionBaremacionInvalida
	}
	// El contexto de almacen conserva la decision original completa. Si el
	// recorte de Bolsa fuese menor, el puente ampliaria autoridad al cruzar de
	// puerto; por ello esa decision no es apta y se deniega en origen.
	if recursoAlmacen && !validaHasta.Equal(decision.ValidaHasta.UTC()) {
		return ContextoOperacionBaremacion{}, ErrAutorizacionBaremacionInvalida
	}
	evidencia, err := puertosvec.NuevaEvidenciaUsoDecisionAutorizacion(decision, instante.UTC())
	if err != nil {
		return ContextoOperacionBaremacion{}, ErrAutorizacionBaremacionInvalida
	}
	campos := append([]string(nil), especificacion.campos...)
	sort.Strings(campos)
	datos := &datosAutorizacionOperacionBaremacion{
		principalRef: decision.PrincipalID, sujetoRef: vinculo.SujetoRef, perfilActorRef: decision.PerfilActivoRef,
		metodo: vinculo.Metodo, garantia: vinculo.Garantia, garantiaMinima: decision.GarantiaMinima,
		autenticacionRef: vinculo.AutenticacionRef, sesionRef: vinculo.SesionRef,
		sesionEmitidaEn:   vinculo.SesionEmitidaEn.UTC(),
		sesionValidaHasta: vinculo.SesionValidaHasta.UTC(),
		vinculo:           vinculo.VinculoAutenticacionActor,
		autorizacionRef:   decision.DecisionRef, accion: accion, clase: especificacion.clase,
		recursoRef: decision.RecursoRef, finalidad: decision.Finalidad, correlacionRef: decision.CorrelacionRef,
		campos: campos, emitidaEn: decision.EmitidaEn.UTC(), validaHasta: validaHasta,
		evidencia: evidencia,
	}
	if recursoAlmacen {
		clon := clonarRecursoAutorizableBaremacion(recurso)
		datos.recursoAlmacen = &clon
	}
	return ContextoOperacionBaremacion{datos: datos}, nil
}

func accionRequiereRecursoAlmacenBaremacion(accion AccionOperacionBaremacion) bool {
	switch accion {
	case AccionCustodiarDecisionBaremacion,
		AccionCustodiarDocumentoFirmadoBaremacion,
		AccionRetenerDocumentoFirmadoBaremacion:
		return true
	default:
		return false
	}
}

func recursoAutorizableBaremacionValido(
	recurso dominiovec.RecursoAutorizable,
	decision dominiovec.DecisionAutorizacion,
	sujetoRef string,
	recursoAlmacen bool,
) bool {
	if recurso.Validar() != nil || recurso.Referencia != decision.RecursoRef ||
		recurso.ModuloID != "bolsa" || recurso.Tipo != decision.TipoRecurso ||
		len(recurso.Ambitos) != 1 || recurso.Ambitos["sujeto_ref"] != sujetoRef {
		return false
	}
	if !recursoAlmacen {
		return len(recurso.Atributos) == 0
	}
	return atributosRecursoAlmacenBaremacionValidos(
		AccionOperacionBaremacion(decision.Accion), recurso.Atributos,
	)
}

func atributosRecursoAlmacenBaremacionValidos(
	accion AccionOperacionBaremacion,
	atributos map[string]string,
) bool {
	requeridos := []string{
		puertosvec.AtributoAlmacenOperacionRef,
		puertosvec.AtributoAlmacenCargaRef,
		puertosvec.AtributoAlmacenClasificacion,
		puertosvec.AtributoAlmacenSujetoSeudonimoHMAC,
		puertosvec.AtributoAlmacenHuellaSolicitudHMAC,
		puertosvec.AtributoAlmacenEfectoRef,
	}
	requiereObjeto := accion == AccionRetenerDocumentoFirmadoBaremacion
	if requiereObjeto {
		requeridos = append(requeridos,
			puertosvec.AtributoAlmacenObjetoRef,
			puertosvec.AtributoAlmacenObjetoVersion,
		)
	}
	if !accionRequiereRecursoAlmacenBaremacion(accion) || len(atributos) != len(requeridos) {
		return false
	}
	for _, clave := range requeridos {
		valor, existe := atributos[clave]
		if !existe || valor == "" || valor != strings.TrimSpace(valor) || strings.ContainsRune(valor, '*') {
			return false
		}
	}
	if !referenciaValida(atributos[puertosvec.AtributoAlmacenOperacionRef], 512) ||
		!referenciaValida(atributos[puertosvec.AtributoAlmacenCargaRef], 512) ||
		!claveValida(atributos[puertosvec.AtributoAlmacenClasificacion]) ||
		!huellaHMACSHA256Valida(atributos[puertosvec.AtributoAlmacenSujetoSeudonimoHMAC]) ||
		!huellaHMACSHA256Valida(atributos[puertosvec.AtributoAlmacenHuellaSolicitudHMAC]) ||
		!referenciaValida(atributos[puertosvec.AtributoAlmacenEfectoRef], 512) {
		return false
	}
	if requiereObjeto {
		objeto := puertosvec.ReferenciaObjetoAlmacen{
			Referencia: atributos[puertosvec.AtributoAlmacenObjetoRef],
			Version:    atributos[puertosvec.AtributoAlmacenObjetoVersion],
		}
		return objeto.Validar() == nil
	}
	_, tieneReferenciaObjeto := atributos[puertosvec.AtributoAlmacenObjetoRef]
	_, tieneVersionObjeto := atributos[puertosvec.AtributoAlmacenObjetoVersion]
	return !tieneReferenciaObjeto && !tieneVersionObjeto
}

func clonarRecursoAutorizableBaremacion(
	recurso dominiovec.RecursoAutorizable,
) dominiovec.RecursoAutorizable {
	clon := recurso
	clon.Ambitos = clonarMapaBaremacion(recurso.Ambitos)
	clon.Atributos = clonarMapaBaremacion(recurso.Atributos)
	return clon
}

func clonarMapaBaremacion(origen map[string]string) map[string]string {
	if origen == nil {
		return nil
	}
	clon := make(map[string]string, len(origen))
	for clave, valor := range origen {
		clon[clave] = valor
	}
	return clon
}

func (c ContextoOperacionBaremacion) Proyeccion() ProyeccionAutorizacionBaremacion {
	if c.datos == nil {
		return ProyeccionAutorizacionBaremacion{}
	}
	d := c.datos
	return ProyeccionAutorizacionBaremacion{
		PrincipalRef: d.principalRef, SujetoRef: d.sujetoRef, PerfilActorClave: d.perfilActorRef,
		MetodoAutenticacion: d.metodo, NivelAutenticacion: d.garantia, GarantiaMinima: d.garantiaMinima,
		AutenticacionRef: d.autenticacionRef, SesionRef: d.sesionRef, SesionEmitidaEn: d.sesionEmitidaEn,
		SesionValidaHasta: d.sesionValidaHasta,
		AutorizacionRef:   d.autorizacionRef, Accion: d.accion, ClaseRecurso: d.clase, RecursoRef: d.recursoRef,
		FinalidadClave: d.finalidad, CorrelacionRef: d.correlacionRef,
		CamposPermitidos: append([]string(nil), d.campos...), EmitidaEn: d.emitidaEn, ValidaHasta: d.validaHasta,
	}
}

func (c ContextoOperacionBaremacion) Validar() error {
	if c.datos == nil {
		return ErrAutorizacionBaremacionInvalida
	}
	d := c.datos
	datosVinculo, errVinculo := d.vinculo.Datos()
	datosEvidencia, errEvidencia := d.evidencia.Datos()
	decision := datosEvidencia.Decision
	especificacion, conocida := especificacionesAccionBaremacion[d.accion]
	if !conocida || d.clase != especificacion.clase || !mismosCamposExactos(d.campos, especificacion.campos) ||
		!recursoAlmacenLigadoBaremacionValido(d, decision) ||
		errEvidencia != nil || d.evidencia.ValidarEn(datosEvidencia.VerificadaEn) != nil ||
		decision.DecisionRef != d.autorizacionRef || decision.PrincipalID != d.principalRef ||
		decision.PerfilActivoRef != d.perfilActorRef || decision.Accion != string(d.accion) ||
		decision.RecursoRef != d.recursoRef || decision.ModuloID != "bolsa" ||
		decision.TipoRecurso != string(d.clase) || decision.Finalidad != d.finalidad ||
		decision.CorrelacionRef != d.correlacionRef ||
		!decision.VinculoAutenticacionActor.CoincideExactamenteCon(d.vinculo) ||
		!mismosCamposExactos(decision.CamposPermitidos, d.campos) || len(decision.Obligaciones) != 0 ||
		!decision.EmitidaEn.UTC().Equal(d.emitidaEn) || d.validaHasta.After(decision.ValidaHasta.UTC()) ||
		datosEvidencia.VerificadaEn.Before(decision.EmitidaEn) ||
		datosEvidencia.VerificadaEn.Before(d.emitidaEn) || !datosEvidencia.VerificadaEn.Before(d.validaHasta) ||
		!referenciaValida(d.principalRef, 512) || !referenciaValida(d.sujetoRef, 512) ||
		!referenciaValida(d.perfilActorRef, 512) || !d.metodo.Valido() || d.metodo == dominiovec.AuthMethodDemo ||
		!d.garantia.Valida() || !d.garantiaMinima.Valida() ||
		!dominiovec.CumpleGarantiaAutenticacion(d.garantia, d.garantiaMinima) ||
		!referenciaValida(d.autenticacionRef, 512) || !referenciaValida(d.sesionRef, 512) || errVinculo != nil ||
		d.principalRef != datosVinculo.PrincipalID || d.perfilActorRef != datosVinculo.PerfilActivoRef ||
		d.metodo != datosVinculo.MetodoObservado || d.garantia != datosVinculo.GarantiaObservada ||
		d.autenticacionRef != datosVinculo.AutenticacionRef || d.sesionRef != datosVinculo.SesionRef ||
		!d.sesionEmitidaEn.Equal(datosVinculo.SesionEmitidaEn) ||
		!d.sesionValidaHasta.Equal(datosVinculo.SesionValidaHasta) || d.sesionEmitidaEn.IsZero() ||
		d.sesionValidaHasta.IsZero() || !d.sesionValidaHasta.After(d.sesionEmitidaEn) ||
		d.emitidaEn.Before(d.sesionEmitidaEn) || d.validaHasta.After(d.sesionValidaHasta) ||
		!referenciaValida(d.autorizacionRef, 512) || !referenciaValida(d.recursoRef, 512) ||
		!claveValida(d.finalidad) || !referenciaValida(d.correlacionRef, 512) || d.emitidaEn.IsZero() ||
		d.validaHasta.IsZero() || !d.validaHasta.After(d.emitidaEn) {
		return ErrAutorizacionBaremacionInvalida
	}
	return nil
}

func recursoAlmacenLigadoBaremacionValido(
	datos *datosAutorizacionOperacionBaremacion,
	decision dominiovec.DecisionAutorizacion,
) bool {
	requerido := accionRequiereRecursoAlmacenBaremacion(datos.accion)
	if !requerido {
		return datos.recursoAlmacen == nil
	}
	if datos.recursoAlmacen == nil || !datos.validaHasta.Equal(decision.ValidaHasta.UTC()) ||
		!recursoAutorizableBaremacionValido(
			*datos.recursoAlmacen, decision, datos.sujetoRef, true,
		) {
		return false
	}
	huella, err := datos.recursoAlmacen.HuellaContextoAutorizacionSHA256()
	return err == nil && huella == decision.ContextoRecursoHuellaSHA256
}

func (c ContextoOperacionBaremacion) ValidarPara(
	accion AccionOperacionBaremacion,
	clase ClaseRecursoOperacionBaremacion,
	recursoRef string,
) error {
	if c.Validar() != nil || !referenciaValida(recursoRef, 512) || c.datos.accion != accion ||
		c.datos.clase != clase || c.datos.recursoRef != recursoRef {
		return ErrAutorizacionBaremacionInvalida
	}
	return nil
}

func (c ContextoOperacionBaremacion) ValidarVigentePara(
	accion AccionOperacionBaremacion,
	clase ClaseRecursoOperacionBaremacion,
	recursoRef string,
	instante time.Time,
) error {
	if c.ValidarPara(accion, clase, recursoRef) != nil || instante.IsZero() ||
		instante.UTC().Before(c.datos.emitidaEn) || !instante.UTC().Before(c.datos.validaHasta) ||
		c.datos.evidencia.ValidarEn(instante.UTC()) != nil {
		return ErrAutorizacionBaremacionInvalida
	}
	return nil
}

// EvidenciaUsoAutorizacion entrega al adaptador duradero la capacidad opaca
// que debe revalidar y consumir en la misma transaccion que el efecto. La
// copia conserva su inmutabilidad: no puede serializarse ni reconstruirse.
func (c ContextoOperacionBaremacion) EvidenciaUsoAutorizacion() (
	puertosvec.EvidenciaUsoDecisionAutorizacion,
	error,
) {
	if c.Validar() != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacion{}, ErrAutorizacionBaremacionInvalida
	}
	return c.datos.evidencia, nil
}

// CrearContextoAlmacenCustodiarDecision es el unico puente desde la decision
// positiva de custodia canonica a una escritura tecnica. Los vinculos deben
// coincidir con los atributos que ya evaluo el PDP.
func (c ContextoOperacionBaremacion) CrearContextoAlmacenCustodiarDecision(
	vinculos puertosvec.VinculosOperacionAlmacen,
) (puertosvec.ContextoOperacionAlmacen, error) {
	return c.crearContextoAlmacenBaremacion(AccionCustodiarDecisionBaremacion, vinculos)
}

// CrearContextoAlmacenCustodiarDocumentoFirmado deriva exclusivamente la
// escritura de la copia institucional firmada; nunca habilita retencion,
// lectura ni eliminacion.
func (c ContextoOperacionBaremacion) CrearContextoAlmacenCustodiarDocumentoFirmado(
	vinculos puertosvec.VinculosOperacionAlmacen,
) (puertosvec.ContextoOperacionAlmacen, error) {
	return c.crearContextoAlmacenBaremacion(AccionCustodiarDocumentoFirmadoBaremacion, vinculos)
}

// CrearContextoAlmacenRetenerDocumentoFirmado deriva solo la retencion de la
// referencia y version exactas incluidas en el recurso evaluado.
func (c ContextoOperacionBaremacion) CrearContextoAlmacenRetenerDocumentoFirmado(
	vinculos puertosvec.VinculosOperacionAlmacen,
) (puertosvec.ContextoOperacionAlmacen, error) {
	return c.crearContextoAlmacenBaremacion(AccionRetenerDocumentoFirmadoBaremacion, vinculos)
}

func (c ContextoOperacionBaremacion) crearContextoAlmacenBaremacion(
	accion AccionOperacionBaremacion,
	vinculos puertosvec.VinculosOperacionAlmacen,
) (puertosvec.ContextoOperacionAlmacen, error) {
	if c.Validar() != nil || c.datos.accion != accion || c.datos.recursoAlmacen == nil {
		return puertosvec.ContextoOperacionAlmacen{}, ErrAutorizacionBaremacionInvalida
	}
	datosEvidencia, err := c.datos.evidencia.Datos()
	if err != nil || datosEvidencia.Decision.Accion != string(accion) ||
		!datosEvidencia.Decision.ValidaHasta.UTC().Equal(c.datos.validaHasta) {
		return puertosvec.ContextoOperacionAlmacen{}, ErrAutorizacionBaremacionInvalida
	}
	recurso := clonarRecursoAutorizableBaremacion(*c.datos.recursoAlmacen)
	var contexto puertosvec.ContextoOperacionAlmacen
	switch accion {
	case AccionCustodiarDecisionBaremacion:
		contexto, err = puertosvec.NuevoContextoCustodiarDecisionBaremacionAlmacen(
			datosEvidencia.Decision, recurso, vinculos, datosEvidencia.VerificadaEn,
		)
	case AccionCustodiarDocumentoFirmadoBaremacion:
		contexto, err = puertosvec.NuevoContextoCustodiarDocumentoFirmadoAlmacen(
			datosEvidencia.Decision, recurso, vinculos, datosEvidencia.VerificadaEn,
		)
	case AccionRetenerDocumentoFirmadoBaremacion:
		contexto, err = puertosvec.NuevoContextoRetenerDocumentoFirmadoAlmacen(
			datosEvidencia.Decision, recurso, vinculos, datosEvidencia.VerificadaEn,
		)
	default:
		return puertosvec.ContextoOperacionAlmacen{}, ErrAutorizacionBaremacionInvalida
	}
	proyeccion, errProyeccion := contexto.Proyeccion()
	if err != nil || errProyeccion != nil ||
		proyeccion.AutorizacionRef != c.datos.autorizacionRef ||
		proyeccion.RecursoRef != c.datos.recursoRef ||
		proyeccion.AccionNegocio != string(c.datos.accion) ||
		!proyeccion.ValidaHasta.Equal(c.datos.validaHasta) {
		return puertosvec.ContextoOperacionAlmacen{}, ErrAutorizacionBaremacionInvalida
	}
	return contexto, nil
}

// MismoVinculoAutenticacionQue exige la misma sesion, controles y documento
// de actor V1. Las decisiones y acciones pueden ser distintas —por ejemplo,
// reservar y confirmar—, pero nunca se mezclan actores o sesiones.
func (c ContextoOperacionBaremacion) MismoVinculoAutenticacionQue(
	otro ContextoOperacionBaremacion,
) bool {
	if c.Validar() != nil || otro.Validar() != nil || c.datos.sujetoRef != otro.datos.sujetoRef {
		return false
	}
	return c.datos.vinculo.CoincideExactamenteCon(otro.datos.vinculo)
}

// CoincideExactamenteCon distingue un reintento de la reutilizacion de una
// capacidad para otra decision o efecto. La huella reforzada incluye todos los
// campos de la decision y el vinculo V1; el sujeto se cruza aparte porque es
// una relacion de negocio resuelta por el servidor.
func (c ContextoOperacionBaremacion) CoincideExactamenteCon(
	otro ContextoOperacionBaremacion,
) bool {
	if c.Validar() != nil || otro.Validar() != nil {
		return false
	}
	d, o := c.datos, otro.datos
	datos, err := c.datos.evidencia.Datos()
	datosOtro, errOtro := otro.datos.evidencia.Datos()
	return err == nil && errOtro == nil &&
		datos.EsquemaHuella == datosOtro.EsquemaHuella &&
		datos.HuellaDecisionSHA256 == datosOtro.HuellaDecisionSHA256 &&
		datos.VerificadaEn.Equal(datosOtro.VerificadaEn) &&
		d.principalRef == o.principalRef && d.sujetoRef == o.sujetoRef &&
		d.perfilActorRef == o.perfilActorRef && d.metodo == o.metodo && d.garantia == o.garantia &&
		d.garantiaMinima == o.garantiaMinima && d.autenticacionRef == o.autenticacionRef &&
		d.sesionRef == o.sesionRef && d.sesionEmitidaEn.Equal(o.sesionEmitidaEn) &&
		d.sesionValidaHasta.Equal(o.sesionValidaHasta) && d.vinculo.CoincideExactamenteCon(o.vinculo) &&
		d.autorizacionRef == o.autorizacionRef && d.accion == o.accion && d.clase == o.clase &&
		d.recursoRef == o.recursoRef && d.finalidad == o.finalidad && d.correlacionRef == o.correlacionRef &&
		mismosCamposExactos(d.campos, o.campos) && d.emitidaEn.Equal(o.emitidaEn) &&
		d.validaHasta.Equal(o.validaHasta) && mismosRecursosAlmacenBaremacion(d.recursoAlmacen, o.recursoAlmacen)
}

func mismosRecursosAlmacenBaremacion(
	primero, segundo *dominiovec.RecursoAutorizable,
) bool {
	if primero == nil || segundo == nil {
		return primero == nil && segundo == nil
	}
	if primero.Referencia != segundo.Referencia || primero.ModuloID != segundo.ModuloID ||
		primero.Tipo != segundo.Tipo || !mismosMapasBaremacion(primero.Ambitos, segundo.Ambitos) ||
		!mismosMapasBaremacion(primero.Atributos, segundo.Atributos) {
		return false
	}
	return true
}

func mismosMapasBaremacion(primero, segundo map[string]string) bool {
	if len(primero) != len(segundo) {
		return false
	}
	for clave, valor := range primero {
		if segundo[clave] != valor {
			return false
		}
	}
	return true
}

// CamposRequeridosOperacionBaremacion permite configurar politicas sin
// duplicar cadenas. Devuelve una copia y nunca una concesion.
func CamposRequeridosOperacionBaremacion(accion AccionOperacionBaremacion) ([]string, bool) {
	especificacion, existe := especificacionesAccionBaremacion[accion]
	if !existe {
		return nil, false
	}
	campos := append([]string(nil), especificacion.campos...)
	sort.Strings(campos)
	return campos, true
}

// ClaseRecursoRequeridaOperacionBaremacion expone la clase exacta ligada a la
// accion para que el PEP y los adaptadores construyan el mismo recurso cerrado.
func ClaseRecursoRequeridaOperacionBaremacion(
	accion AccionOperacionBaremacion,
) (ClaseRecursoOperacionBaremacion, bool) {
	especificacion, existe := especificacionesAccionBaremacion[accion]
	if !existe {
		return "", false
	}
	return especificacion.clase, true
}

// AccionAdopcionParaClase devuelve la unica accion positiva que puede adoptar
// cada transicion del historial. No existe una accion generica que herede o
// amplie permisos entre una decision ordinaria y una actuacion inspectora.
func AccionAdopcionParaClase(clase dominiobolsa.ClaseDecisionTecnica) (AccionOperacionBaremacion, bool) {
	switch clase {
	case dominiobolsa.ClaseDecisionInicial:
		return AccionAdoptarDecisionInicialBaremacion, true
	case dominiobolsa.ClaseDecisionRectificacion:
		return AccionRectificarDecisionBaremacion, true
	case dominiobolsa.ClaseDecisionRevocacion:
		return AccionRevocarDecisionBaremacion, true
	case dominiobolsa.ClaseDecisionRehabilitacion:
		return AccionRehabilitarDecisionBaremacion, true
	default:
		return "", false
	}
}

func (ContextoOperacionBaremacion) String() string { return "[AUTORIZACION-BAREMACION-OPACA]" }
func (ContextoOperacionBaremacion) GoString() string {
	return "ports.ContextoOperacionBaremacion{[OPACA]}"
}
func (c ContextoOperacionBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (ContextoOperacionBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionProhibida
}
func (ContextoOperacionBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionProhibida
}
func (ContextoOperacionBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionProhibida
}

func mismosCamposExactos(recibidos, esperados []string) bool {
	if len(recibidos) == 0 || len(recibidos) != len(esperados) {
		return false
	}
	a := append([]string(nil), recibidos...)
	b := append([]string(nil), esperados...)
	for indice := range a {
		a[indice] = strings.TrimSpace(a[indice])
	}
	for indice := range b {
		b[indice] = strings.TrimSpace(b[indice])
	}
	sort.Strings(a)
	sort.Strings(b)
	for indice := range a {
		if a[indice] == "" || a[indice] == "*" || a[indice] != b[indice] ||
			(indice > 0 && a[indice] == a[indice-1]) {
			return false
		}
	}
	return true
}

func accionReservaCambio(clase ClaseCambioBaremacion) (AccionOperacionBaremacion, bool) {
	switch clase {
	case ClaseCambioAltaBaremacion:
		return AccionReservarAltaBaremacion, true
	case ClaseCambioIncorporarDecision:
		return AccionReservarDecisionBaremacion, true
	default:
		return "", false
	}
}

func accionConfirmacionCambio(clase ClaseCambioBaremacion) (AccionOperacionBaremacion, bool) {
	switch clase {
	case ClaseCambioAltaBaremacion:
		return AccionConfirmarAltaBaremacion, true
	case ClaseCambioIncorporarDecision:
		return AccionConfirmarDecisionBaremacion, true
	default:
		return "", false
	}
}

func accionAbandonoCambio(clase ClaseCambioBaremacion) (AccionOperacionBaremacion, bool) {
	switch clase {
	case ClaseCambioAltaBaremacion:
		return AccionAbandonarAltaBaremacion, true
	case ClaseCambioIncorporarDecision:
		return AccionAbandonarDecisionBaremacion, true
	default:
		return "", false
	}
}
