// Package ports declara contratos hexagonales del modulo de bolsas. Ninguno
// depende de un motor de datos, proveedor de firma o producto de almacenamiento.
package ports

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrSolicitudBaremacionInvalida             = errors.New("bolsa: solicitud de persistencia invalida")
	ErrBaremacionNoEncontrada                  = errors.New("bolsa: baremacion no encontrada")
	ErrVersionBaremacionNoEncontrada           = errors.New("bolsa: version de baremacion no encontrada")
	ErrBaremacionYaExiste                      = errors.New("bolsa: baremacion ya existente")
	ErrVersionBaremacionConflicto              = errors.New("bolsa: version de baremacion en conflicto")
	ErrHistorialBaremacionNoAnexable           = errors.New("bolsa: el historial no puede anexarse")
	ErrClaveIdempotenciaBaremacionInvalida     = errors.New("bolsa: clave de idempotencia invalida")
	ErrClaveIdempotenciaBaremacionReutilizada  = errors.New("bolsa: clave de idempotencia reutilizada")
	ErrCambioBaremacionEnCurso                 = errors.New("bolsa: cambio de baremacion en curso")
	ErrReservaBaremacionNoValida               = errors.New("bolsa: reserva de baremacion no valida")
	ErrFuenteBaremacionNoDisponible            = errors.New("bolsa: fuente fiable no disponible")
	ErrCriterioBaremacionNoEncontrado          = errors.New("bolsa: criterio de baremacion no encontrado")
	ErrCriterioBaremacionNoVigente             = errors.New("bolsa: criterio de baremacion no vigente")
	ErrEvidenciaBaremacionNoEncontrada         = errors.New("bolsa: evidencia de baremacion no encontrada")
	ErrEvidenciaBaremacionNoConfiable          = errors.New("bolsa: evidencia de baremacion no confiable")
	ErrRepresentacionBaremacionNoEncontrada    = errors.New("bolsa: representacion documental no encontrada")
	ErrRepresentacionBaremacionNoConfiable     = errors.New("bolsa: representacion documental no confiable")
	ErrCalculoOficialNoDisponible              = errors.New("bolsa: calculo oficial no disponible")
	ErrCalculoOficialNoReproducible            = errors.New("bolsa: calculo oficial no reproducible")
	ErrPoliticaFirmaNoEncontrada               = errors.New("bolsa: politica de firma no encontrada")
	ErrPoliticaFirmaNoVigente                  = errors.New("bolsa: politica de firma no vigente")
	ErrPoliticaFirmaInsegura                   = errors.New("bolsa: politica de firma no cumple los minimos")
	ErrCodificacionCanonicaNoDisponible        = errors.New("bolsa: codificacion canonica no disponible")
	ErrCustodiaDocumentoFirmableInvalida       = errors.New("bolsa: custodia de documento firmable invalida")
	ErrCargaProtegidaInvalida                  = errors.New("bolsa: carga protegida invalida")
	ErrSerializacionCargaProtegidaProhibida    = errors.New("bolsa: serializacion de carga protegida prohibida")
	ErrTokenReservaBaremacionInvalido          = errors.New("bolsa: token de reserva invalido")
	ErrSerializacionTokenReservaProhibida      = errors.New("bolsa: serializacion de token de reserva prohibida")
	ErrFirmaInteractivaNoDisponible            = errors.New("bolsa: firma interactiva no disponible")
	ErrSesionFirmaNoEncontrada                 = errors.New("bolsa: sesion de firma no encontrada")
	ErrSesionFirmaExpirada                     = errors.New("bolsa: sesion de firma expirada")
	ErrFirmaInteractivaNoCompletada            = errors.New("bolsa: firma interactiva no completada")
	ErrValidacionFirmaNoDisponible             = errors.New("bolsa: validacion de firma no disponible")
	ErrFirmaServidorNoValida                   = errors.New("bolsa: firma no valida")
	ErrValidacionFirmaNoConcluyente            = errors.New("bolsa: validacion de firma no concluyente")
	ErrRevisionPDFFirmaNoConfiable             = errors.New("bolsa: revision PDF de firma no confiable")
	ErrSelloTiempoNoDisponible                 = errors.New("bolsa: sello de tiempo no disponible")
	ErrAumentoFirmaNoDisponible                = errors.New("bolsa: aumento de firma no disponible")
	ErrEvidenciaFirmaNoEncontrada              = errors.New("bolsa: evidencia historica de firma no encontrada")
	ErrGeneracionReferenciaNoDisponible        = errors.New("bolsa: generacion de referencia no disponible")
	ErrAutorizacionBaremacionInvalida          = errors.New("bolsa: autorizacion de operacion invalida")
	ErrAutorizacionBaremacionReutilizada       = errors.New("bolsa: decision de autorizacion reutilizada para otro efecto")
	ErrSerializacionAutorizacionProhibida      = errors.New("bolsa: serializacion de autorizacion prohibida")
	ErrVerificacionSelloBaremacionNoDisponible = errors.New("bolsa: verificacion de sello no disponible")
	ErrSelloBaremacionNoAutentico              = errors.New("bolsa: sello de operacion no autentico")
)

const (
	maximoCargaProtegida                = 64 << 20
	maximoEvidenciasCalculo             = 256
	maximoComprobacionesFirma           = 64
	VentanaMaximaReservaBaremacion      = 10 * time.Minute
	VentanaMaximaSesionFirmaInteractiva = 15 * time.Minute
)

const (
	FormatoFirmaPDFCanonico       = "pdf_canonico"
	PerfilFirmaPAdESBaselineB     = "pades_baseline_b"
	PerfilFirmaPAdESBaselineT     = "pades_baseline_t"
	PerfilFirmaPAdESBaselineLTA   = "pades_baseline_lta"
	AlgoritmoHuellaFirmaSHA256    = "sha256"
	ComprobacionIntegridadFirma   = "integridad_criptografica"
	ComprobacionCadenaConfianza   = "cadena_confianza"
	ComprobacionRevocacionFirma   = "revocacion_instante_firma"
	ComprobacionIdentidadFirmante = "identidad_firmante"
	ComprobacionPoliticaFirma     = "politica_firma"
	ComprobacionDigestDocumento   = "digest_documento"
	ComprobacionAlgoritmosFirma   = "algoritmos_permitidos"
	ComprobacionFormatoPAdES      = "formato_pades"
	ComprobacionPerfilPAdES       = "perfil_pades"
)

var comprobacionesFirmaObligatorias = []string{
	ComprobacionIntegridadFirma,
	ComprobacionCadenaConfianza,
	ComprobacionRevocacionFirma,
	ComprobacionIdentidadFirmante,
	ComprobacionPoliticaFirma,
	ComprobacionDigestDocumento,
	ComprobacionAlgoritmosFirma,
	ComprobacionFormatoPAdES,
	ComprobacionPerfilPAdES,
}

func ComprobacionesFirmaObligatorias() []string {
	return append([]string(nil), comprobacionesFirmaObligatorias...)
}

func perfilFirmaPermitido(perfil string) bool {
	switch perfil {
	case PerfilFirmaPAdESBaselineB, PerfilFirmaPAdESBaselineT, PerfilFirmaPAdESBaselineLTA:
		return true
	default:
		return false
	}
}

func mismoConjuntoClaves(recibidas, esperadas []string) bool {
	if len(recibidas) == 0 || len(recibidas) != len(esperadas) {
		return false
	}
	a := append([]string(nil), recibidas...)
	b := append([]string(nil), esperadas...)
	for _, clave := range a {
		if !claveValida(clave) {
			return false
		}
	}
	sort.Strings(a)
	sort.Strings(b)
	for indice := range a {
		if a[indice] != b[indice] || (indice > 0 && a[indice] == a[indice-1]) {
			return false
		}
	}
	return true
}

var patronClave = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// CargaProtegida copia los bytes y bloquea su serializacion o formateo.
type CargaProtegida struct{ valor []byte }

func NuevaCargaProtegida(valor []byte) (CargaProtegida, error) {
	if len(valor) == 0 || len(valor) > maximoCargaProtegida {
		return CargaProtegida{}, ErrCargaProtegidaInvalida
	}
	return CargaProtegida{valor: append([]byte(nil), valor...)}, nil
}

func (c CargaProtegida) Validar() error {
	if len(c.valor) == 0 || len(c.valor) > maximoCargaProtegida {
		return ErrCargaProtegidaInvalida
	}
	return nil
}

func (c CargaProtegida) Revelar() []byte { return append([]byte(nil), c.valor...) }
func (c CargaProtegida) Tamano() int     { return len(c.valor) }
func (CargaProtegida) String() string    { return "[CARGA-PROTEGIDA]" }
func (CargaProtegida) GoString() string  { return "ports.CargaProtegida{[OCULTA]}" }
func (c CargaProtegida) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (CargaProtegida) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCargaProtegidaProhibida
}
func (CargaProtegida) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionCargaProtegidaProhibida
}

// TokenReservaBaremacion solo admite Base64URL sin relleno, en ASCII y con
// entre 192 y 768 bits. Es una capacidad temporal, nunca un ID de negocio.
type TokenReservaBaremacion struct{ valor string }

func NuevoTokenReservaBaremacion(valor string) (TokenReservaBaremacion, error) {
	if !tokenBase64URLValido(valor) {
		return TokenReservaBaremacion{}, ErrTokenReservaBaremacionInvalido
	}
	return TokenReservaBaremacion{valor: valor}, nil
}
func (t TokenReservaBaremacion) Validar() error {
	if !tokenBase64URLValido(t.valor) {
		return ErrTokenReservaBaremacionInvalido
	}
	return nil
}
func (t TokenReservaBaremacion) Revelar() string { return t.valor }
func (TokenReservaBaremacion) String() string    { return "[TOKEN-RESERVA-OCULTO]" }
func (TokenReservaBaremacion) GoString() string  { return "ports.TokenReservaBaremacion{[OCULTO]}" }
func (t TokenReservaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, t.String())
}
func (TokenReservaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaProhibida
}
func (TokenReservaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionTokenReservaProhibida
}

type ClaseCambioBaremacion string

const (
	ClaseCambioAltaBaremacion     ClaseCambioBaremacion = "alta"
	ClaseCambioIncorporarDecision ClaseCambioBaremacion = "incorporar_decision"
)

func (c ClaseCambioBaremacion) Valida() bool {
	return c == ClaseCambioAltaBaremacion || c == ClaseCambioIncorporarDecision
}

type ReferenciaVersionBaremacion struct {
	BaremacionMeritoRef string
	Numero              uint64
	HuellaEstadoSHA256  string
}

func (r ReferenciaVersionBaremacion) Validar() error {
	if !referenciaValida(r.BaremacionMeritoRef, 512) || r.Numero < 1 || !huellaSHA256Valida(r.HuellaEstadoSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type VersionBaremacion struct {
	Referencia   ReferenciaVersionBaremacion
	Agregado     dominiobolsa.BaremacionMerito
	ConfirmadaEn time.Time
}

func (v VersionBaremacion) Validar() error {
	if v.Referencia.Validar() != nil || v.Agregado.Validar() != nil || v.ConfirmadaEn.IsZero() ||
		v.Referencia.BaremacionMeritoRef != v.Agregado.ID ||
		v.Referencia.Numero != uint64(len(v.Agregado.Decisiones))+1 {
		return ErrSolicitudBaremacionInvalida
	}
	huella, err := v.Agregado.HuellaEstadoSHA256()
	if err != nil || huella != v.Referencia.HuellaEstadoSHA256 {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (v VersionBaremacion) Clonar() (VersionBaremacion, error) {
	agregado, err := v.Agregado.ClonarCanonica()
	if err != nil {
		return VersionBaremacion{}, err
	}
	clon := v
	clon.Agregado = agregado
	clon.ConfirmadaEn = v.ConfirmadaEn.UTC()
	if err := clon.Validar(); err != nil {
		return VersionBaremacion{}, err
	}
	return clon, nil
}

type SolicitudReservarCambioBaremacion struct {
	Contexto            ContextoOperacionBaremacion
	Clase               ClaseCambioBaremacion
	ClaveIdempotencia   string
	BaremacionMeritoRef string
	VersionEsperada     *ReferenciaVersionBaremacion
	HuellaSolicitudHMAC string
	SolicitadaEn        time.Time
	ExpiraEn            time.Time
}

func (s SolicitudReservarCambioBaremacion) Validar() error {
	accion, accionValida := accionReservaCambio(s.Clase)
	if !accionValida || s.Contexto.ValidarPara(accion, ClaseRecursoBaremacion, s.BaremacionMeritoRef) != nil ||
		!s.Clase.Valida() || !referenciaValida(s.ClaveIdempotencia, 512) ||
		!referenciaValida(s.BaremacionMeritoRef, 512) || !huellaHMACSHA256Valida(s.HuellaSolicitudHMAC) ||
		!ventanaValida(s.SolicitadaEn, s.ExpiraEn, VentanaMaximaReservaBaremacion) {
		return ErrSolicitudBaremacionInvalida
	}
	if s.Clase == ClaseCambioAltaBaremacion {
		if s.VersionEsperada != nil {
			return ErrSolicitudBaremacionInvalida
		}
		return nil
	}
	if s.VersionEsperada == nil || s.VersionEsperada.Validar() != nil ||
		s.VersionEsperada.BaremacionMeritoRef != s.BaremacionMeritoRef {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (s SolicitudReservarCambioBaremacion) Clonar() SolicitudReservarCambioBaremacion {
	clon := s
	if s.VersionEsperada != nil {
		version := *s.VersionEsperada
		clon.VersionEsperada = &version
	}
	return clon
}

type ReservaCambioBaremacion struct {
	Token               TokenReservaBaremacion
	Repetida            bool
	VersionConfirmada   *VersionBaremacion
	BaremacionMeritoRef string
	Clase               ClaseCambioBaremacion
	VersionEsperada     *ReferenciaVersionBaremacion
	HuellaSolicitudHMAC string
	ExpiraEn            time.Time
}

func (r ReservaCambioBaremacion) Validar() error {
	if !referenciaValida(r.BaremacionMeritoRef, 512) || !r.Clase.Valida() ||
		!huellaHMACSHA256Valida(r.HuellaSolicitudHMAC) || r.ExpiraEn.IsZero() {
		return ErrReservaBaremacionNoValida
	}
	if r.Clase == ClaseCambioAltaBaremacion {
		if r.VersionEsperada != nil {
			return ErrReservaBaremacionNoValida
		}
	} else if r.VersionEsperada == nil || r.VersionEsperada.Validar() != nil ||
		r.VersionEsperada.BaremacionMeritoRef != r.BaremacionMeritoRef {
		return ErrReservaBaremacionNoValida
	}
	if r.Repetida {
		if r.Token.valor != "" || r.VersionConfirmada == nil || r.VersionConfirmada.Validar() != nil {
			return ErrReservaBaremacionNoValida
		}
		return nil
	}
	if r.Token.Validar() != nil || r.VersionConfirmada != nil {
		return ErrReservaBaremacionNoValida
	}
	return nil
}

// ValidarPara impide aceptar una reserva o una repeticion perteneciente a otra
// solicitud, aun cuando la respuesta sea internamente valida.
func (r ReservaCambioBaremacion) ValidarPara(s SolicitudReservarCambioBaremacion) error {
	if s.Validar() != nil || r.Validar() != nil || r.BaremacionMeritoRef != s.BaremacionMeritoRef ||
		r.Clase != s.Clase || r.HuellaSolicitudHMAC != s.HuellaSolicitudHMAC ||
		!r.ExpiraEn.Equal(s.ExpiraEn.UTC()) || !referenciasVersionIguales(r.VersionEsperada, s.VersionEsperada) {
		return ErrReservaBaremacionNoValida
	}
	if !r.Repetida {
		return nil
	}
	version := r.VersionConfirmada
	if version == nil || version.Referencia.BaremacionMeritoRef != s.BaremacionMeritoRef {
		return ErrReservaBaremacionNoValida
	}
	numeroEsperado := uint64(1)
	if s.VersionEsperada != nil {
		numeroEsperado = s.VersionEsperada.Numero + 1
	}
	if version.Referencia.Numero != numeroEsperado || version.ConfirmadaEn.Before(s.SolicitadaEn.UTC()) {
		return ErrReservaBaremacionNoValida
	}
	return nil
}

func (r ReservaCambioBaremacion) Clonar() (ReservaCambioBaremacion, error) {
	clon := r
	if r.VersionEsperada != nil {
		version := *r.VersionEsperada
		clon.VersionEsperada = &version
	}
	if r.VersionConfirmada != nil {
		version, err := r.VersionConfirmada.Clonar()
		if err != nil {
			return ReservaCambioBaremacion{}, err
		}
		clon.VersionConfirmada = &version
	}
	return clon, clon.Validar()
}

func referenciasVersionIguales(a, b *ReferenciaVersionBaremacion) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// TrazabilidadCambioBaremacion solo admite la motivacion de negocio. Actor,
// accion, modulo, sujeto, versiones y huellas los deriva el repositorio; no se
// aceptan AuditEntry, Event ni mapas libres proporcionados por un cliente.
type TrazabilidadCambioBaremacion struct {
	MotivoClave string
	Motivo      string
}

func (t TrazabilidadCambioBaremacion) Validar() error {
	if !claveValida(t.MotivoClave) || !textoValido(t.Motivo, 8000) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type SolicitudConfirmarCambioBaremacion struct {
	Contexto            ContextoOperacionBaremacion
	Token               TokenReservaBaremacion
	Clase               ClaseCambioBaremacion
	VersionEsperada     *ReferenciaVersionBaremacion
	HuellaSolicitudHMAC string
	Agregado            dominiobolsa.BaremacionMerito
	Manifiesto          *ManifiestoProbatorioBaremacion
	Trazabilidad        TrazabilidadCambioBaremacion
	ConfirmadaEn        time.Time
}

func (s SolicitudConfirmarCambioBaremacion) Validar() error {
	accion, accionValida := accionConfirmacionCambio(s.Clase)
	if !accionValida || s.Contexto.ValidarPara(accion, ClaseRecursoBaremacion, s.Agregado.ID) != nil ||
		s.Token.Validar() != nil || !s.Clase.Valida() ||
		!huellaHMACSHA256Valida(s.HuellaSolicitudHMAC) || s.Agregado.Validar() != nil ||
		s.Contexto.Proyeccion().SujetoRef != s.Agregado.SujetoRef || s.Trazabilidad.Validar() != nil || s.ConfirmadaEn.IsZero() ||
		s.ConfirmadaEn.Before(s.Agregado.CreadaEn) {
		return ErrSolicitudBaremacionInvalida
	}
	if s.Clase == ClaseCambioAltaBaremacion {
		if s.VersionEsperada != nil || len(s.Agregado.Decisiones) != 0 || s.Manifiesto != nil {
			return ErrSolicitudBaremacionInvalida
		}
		return nil
	}
	if s.VersionEsperada == nil || s.VersionEsperada.Validar() != nil ||
		s.VersionEsperada.BaremacionMeritoRef != s.Agregado.ID ||
		uint64(len(s.Agregado.Decisiones)) != s.VersionEsperada.Numero {
		return ErrSolicitudBaremacionInvalida
	}
	ultima, existe := s.Agregado.UltimaDecision()
	if !existe || s.Manifiesto == nil ||
		s.Manifiesto.ValidarCoberturaFirmaPara(*s.VersionEsperada, ultima.Contenido, ultima.Firma) != nil ||
		!s.Manifiesto.autorizacionConfirmacionCoincide(s.Contexto) ||
		ultima.Firma.ManifiestoProbatorioRef != s.Manifiesto.Referencia ||
		ultima.Firma.HuellaManifiestoProbatorioSHA256 != s.Manifiesto.HuellaManifiestoSHA256 ||
		ultima.Firma.SelloManifiestoProbatorioHMACSHA256 != s.Manifiesto.SelloManifiestoHMACSHA256 ||
		s.ConfirmadaEn.Before(ultima.Firma.ValidadaEn) ||
		ultima.Contenido.DecisorRef != s.Contexto.Proyeccion().PrincipalRef ||
		ultima.Contenido.PerfilDecisorClave != s.Contexto.Proyeccion().PerfilActorClave ||
		ultima.Contenido.AutorizacionRef == s.Contexto.Proyeccion().AutorizacionRef ||
		ultima.Contenido.FinalidadClave != s.Contexto.Proyeccion().FinalidadClave ||
		ultima.Contenido.CorrelacionRef != s.Contexto.Proyeccion().CorrelacionRef {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (s SolicitudConfirmarCambioBaremacion) Clonar() (SolicitudConfirmarCambioBaremacion, error) {
	clon := s
	if s.VersionEsperada != nil {
		version := *s.VersionEsperada
		clon.VersionEsperada = &version
	}
	if s.Manifiesto != nil {
		manifiesto := s.Manifiesto.Clonar()
		clon.Manifiesto = &manifiesto
	}
	agregado, err := s.Agregado.ClonarCanonica()
	if err != nil {
		return SolicitudConfirmarCambioBaremacion{}, err
	}
	clon.Agregado = agregado
	return clon, clon.Validar()
}

type EvidenciaTransaccionBaremacion struct {
	AuditoriaRef             string
	HuellaAuditoriaSHA256    string
	EventoOutboxRef          string
	HuellaEventoOutboxSHA256 string
	ConfirmadaEn             time.Time
}

func (e EvidenciaTransaccionBaremacion) Validar() error {
	if !referenciaValida(e.AuditoriaRef, 512) || !huellaSHA256Valida(e.HuellaAuditoriaSHA256) ||
		!referenciaValida(e.EventoOutboxRef, 512) || !huellaSHA256Valida(e.HuellaEventoOutboxSHA256) ||
		e.ConfirmadaEn.IsZero() {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type ResultadoConfirmarCambioBaremacion struct {
	Version   VersionBaremacion
	Evidencia EvidenciaTransaccionBaremacion
}

func (r ResultadoConfirmarCambioBaremacion) Validar() error {
	if r.Version.Validar() != nil || r.Evidencia.Validar() != nil ||
		!r.Version.ConfirmadaEn.Equal(r.Evidencia.ConfirmadaEn) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

// ValidarPara liga el resultado a la mutacion exacta solicitada. Una respuesta
// valida de otra baremacion, version o agregado nunca se acepta por semejanza.
func (r ResultadoConfirmarCambioBaremacion) ValidarPara(s SolicitudConfirmarCambioBaremacion) error {
	if s.Validar() != nil || r.Validar() != nil ||
		r.Version.Referencia.BaremacionMeritoRef != s.Agregado.ID ||
		r.Version.Agregado.ID != s.Agregado.ID || r.Version.ConfirmadaEn.Before(s.ConfirmadaEn.UTC()) {
		return ErrSolicitudBaremacionInvalida
	}
	hash, err := s.Agregado.HuellaEstadoSHA256()
	if err != nil || r.Version.Referencia.HuellaEstadoSHA256 != hash {
		return ErrSolicitudBaremacionInvalida
	}
	numero := uint64(1)
	if s.VersionEsperada != nil {
		numero = s.VersionEsperada.Numero + 1
	}
	if r.Version.Referencia.Numero != numero {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (r ResultadoConfirmarCambioBaremacion) Clonar() (ResultadoConfirmarCambioBaremacion, error) {
	version, err := r.Version.Clonar()
	if err != nil {
		return ResultadoConfirmarCambioBaremacion{}, err
	}
	clon := r
	clon.Version = version
	return clon, clon.Validar()
}

type SolicitudAbandonarReservaBaremacion struct {
	Contexto            ContextoOperacionBaremacion
	Token               TokenReservaBaremacion
	Clase               ClaseCambioBaremacion
	BaremacionMeritoRef string
}

func (s SolicitudAbandonarReservaBaremacion) Validar() error {
	accion, accionValida := accionAbandonoCambio(s.Clase)
	if !accionValida || s.Contexto.ValidarPara(accion, ClaseRecursoBaremacion, s.BaremacionMeritoRef) != nil ||
		s.Token.Validar() != nil || !s.Clase.Valida() || !referenciaValida(s.BaremacionMeritoRef, 512) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type SolicitudObtenerBaremacionVigente struct {
	Contexto            ContextoOperacionBaremacion
	BaremacionMeritoRef string
}

func (s SolicitudObtenerBaremacionVigente) Validar() error {
	if s.Contexto.ValidarPara(AccionConsultarBaremacionVigente, ClaseRecursoBaremacion, s.BaremacionMeritoRef) != nil ||
		!referenciaValida(s.BaremacionMeritoRef, 512) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type SolicitudObtenerVersionBaremacion struct {
	Contexto            ContextoOperacionBaremacion
	BaremacionMeritoRef string
	Numero              uint64
}

func (s SolicitudObtenerVersionBaremacion) Validar() error {
	if s.Contexto.ValidarPara(AccionConsultarVersionBaremacion, ClaseRecursoBaremacion, s.BaremacionMeritoRef) != nil ||
		!referenciaValida(s.BaremacionMeritoRef, 512) || s.Numero < 1 {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

// RepositorioBaremaciones debe derivar y confirmar agregado, auditoria tipada
// y un unico evento outbox en la misma transaccion. Alta crea solo version 1;
// cada incorporacion anexa exactamente una decision con OCC exacto.
//
// Un adaptador duradero no puede fiarse solo de la proyeccion del contexto:
// dentro de esa misma transaccion y con su reloj fiable debe validar la
// EvidenciaUsoAutorizacion, releer la decision registrada por DecisionRef y
// exigir coincidencia exacta de su huella y vinculo V1. Tambien debe comprobar
// que siguen vigentes sesion y contexto de actor, asignacion, rol, control de
// revision y catalogo de politicas. Ausencia, ambiguedad, cambio, revocacion,
// caducidad o error deniegan y revierten reserva/efecto/auditoria/outbox.
//
// Cada mutacion consume de forma unica DecisionRef -> huella del efecto. Solo
// un reintento de la misma decision y del mismo efecto exacto puede recuperar
// el resultado anterior; reutilizarla para otro efecto se deniega. Las lecturas
// sensibles deben realizar la misma revalidacion dentro de su transaccion de
// lectura. Ningun adaptador productivo cumple el puerto si omite estas barreras.
type RepositorioBaremaciones interface {
	ReservarCambio(context.Context, SolicitudReservarCambioBaremacion) (ReservaCambioBaremacion, error)
	ConfirmarCambio(context.Context, SolicitudConfirmarCambioBaremacion) (ResultadoConfirmarCambioBaremacion, error)
	AbandonarReserva(context.Context, SolicitudAbandonarReservaBaremacion) error
	ObtenerVersionVigente(context.Context, SolicitudObtenerBaremacionVigente) (VersionBaremacion, error)
	ObtenerVersion(context.Context, SolicitudObtenerVersionBaremacion) (VersionBaremacion, error)
	ObtenerEvidenciaTransaccion(context.Context, SolicitudObtenerEvidenciaTransaccionBaremacion) (EvidenciaTransaccionBaremacionRecuperada, error)
}

type SolicitudObtenerCriterioBaremacion struct {
	Contexto             ContextoOperacionBaremacion
	ProcesoRef           string
	Clave                string
	Version              int
	HuellaEsperadaSHA256 string
}

func (s SolicitudObtenerCriterioBaremacion) Validar() error {
	if s.Contexto.ValidarPara(AccionConsultarCriterioBaremacion, ClaseRecursoProceso, s.ProcesoRef) != nil ||
		!referenciaValida(s.ProcesoRef, 512) || !claveValida(s.Clave) ||
		s.Version < 1 || !huellaSHA256Valida(s.HuellaEsperadaSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type CriterioBaremacionConfiable struct {
	Referencia              dominiobolsa.ReferenciaCriterio
	PublicacionRef          string
	HuellaPublicacionSHA256 string
	EvidenciaConsultaRef    string
	HuellaEvidenciaSHA256   string
	ConsultadoEn            time.Time
}

func (c CriterioBaremacionConfiable) Validar() error {
	if c.Referencia.Validar() != nil || !referenciaValida(c.PublicacionRef, 512) ||
		!huellaSHA256Valida(c.HuellaPublicacionSHA256) || !referenciaValida(c.EvidenciaConsultaRef, 512) ||
		!huellaSHA256Valida(c.HuellaEvidenciaSHA256) || c.ConsultadoEn.IsZero() {
		return ErrCriterioBaremacionNoVigente
	}
	return nil
}

func (c CriterioBaremacionConfiable) ValidarPara(s SolicitudObtenerCriterioBaremacion) error {
	if s.Validar() != nil || c.Validar() != nil || c.Referencia.ProcesoRef != s.ProcesoRef ||
		c.Referencia.Clave != s.Clave || c.Referencia.Version != s.Version ||
		c.Referencia.HuellaSHA256 != s.HuellaEsperadaSHA256 {
		return ErrCriterioBaremacionNoVigente
	}
	return nil
}

type SolicitudObtenerEvidenciaBaremacion struct {
	Contexto     ContextoOperacionBaremacion
	ProcesoRef   string
	SolicitudRef string
	Evidencia    dominiobolsa.EvidenciaMerito
}

func (s SolicitudObtenerEvidenciaBaremacion) Validar() error {
	if s.Contexto.ValidarPara(AccionConsultarEvidenciaBaremacion, ClaseRecursoEvidencia, s.Evidencia.Referencia.DocumentoRef) != nil ||
		!referenciaValida(s.ProcesoRef, 512) ||
		!referenciaValida(s.SolicitudRef, 512) || s.Evidencia.Validar() != nil {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type EvidenciaBaremacionConfiable struct {
	Evidencia                  dominiobolsa.EvidenciaMerito
	Documento                  dominiovec.DocumentoLogico
	VerificacionPertenenciaRef string
	HuellaVerificacionSHA256   string
	VerificadaEn               time.Time
}

func (e EvidenciaBaremacionConfiable) Validar() error {
	if e.Evidencia.Validar() != nil || e.Documento.Validar() != nil ||
		e.Evidencia.Referencia.DocumentoRef != e.Documento.ID ||
		e.Evidencia.Referencia.VersionDocumento != e.Documento.Version ||
		!referenciaValida(e.VerificacionPertenenciaRef, 512) ||
		!huellaSHA256Valida(e.HuellaVerificacionSHA256) || e.VerificadaEn.IsZero() {
		return ErrEvidenciaBaremacionNoConfiable
	}
	return nil
}

func (e EvidenciaBaremacionConfiable) Clonar() (EvidenciaBaremacionConfiable, error) {
	clon := e
	clon.Evidencia = clonarEvidencia(e.Evidencia)
	documento, err := e.Documento.ClonarCanonico()
	if err != nil {
		return EvidenciaBaremacionConfiable{}, err
	}
	clon.Documento = documento
	return clon, clon.Validar()
}

func (e EvidenciaBaremacionConfiable) ValidarPara(s SolicitudObtenerEvidenciaBaremacion) error {
	if s.Validar() != nil || e.Validar() != nil || !evidenciasIguales(e.Evidencia, s.Evidencia) ||
		e.Documento.ID != s.Evidencia.Referencia.DocumentoRef ||
		e.Documento.Version != s.Evidencia.Referencia.VersionDocumento {
		return ErrEvidenciaBaremacionNoConfiable
	}
	return nil
}

type SolicitudObtenerRepresentacionBaremacion struct {
	Contexto   ContextoOperacionBaremacion
	Referencia dominiobolsa.ReferenciaEvidencia
}

func (s SolicitudObtenerRepresentacionBaremacion) Validar() error {
	if s.Contexto.ValidarPara(AccionConsultarRepresentacionBaremacion, ClaseRecursoRepresentacion, s.Referencia.RepresentacionRef) != nil ||
		s.Referencia.Validar() != nil {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type RepresentacionBaremacionConfiable struct {
	Representacion        dominiovec.RepresentacionDocumento
	EvidenciaConsultaRef  string
	HuellaEvidenciaSHA256 string
	ConsultadaEn          time.Time
}

func (r RepresentacionBaremacionConfiable) Validar() error {
	if r.Representacion.Validar() != nil || !referenciaValida(r.EvidenciaConsultaRef, 512) ||
		r.Representacion.EstadoTecnico != dominiovec.EstadoRepresentacionDisponible ||
		(r.Representacion.EstadoAntivirus != dominiovec.EstadoAntivirusLimpio &&
			r.Representacion.EstadoAntivirus != dominiovec.EstadoAntivirusNoAplica) ||
		!huellaSHA256Valida(r.HuellaEvidenciaSHA256) || r.ConsultadaEn.IsZero() {
		return ErrRepresentacionBaremacionNoConfiable
	}
	return nil
}

func (r RepresentacionBaremacionConfiable) ValidarPara(s SolicitudObtenerRepresentacionBaremacion) error {
	if s.Validar() != nil || r.Validar() != nil || r.Representacion.ID != s.Referencia.RepresentacionRef ||
		r.Representacion.Documento.ID != s.Referencia.DocumentoRef ||
		r.Representacion.Documento.Version != s.Referencia.VersionDocumento ||
		r.Representacion.HuellaContenidoSHA256 != s.Referencia.HuellaSHA256 {
		return ErrRepresentacionBaremacionNoConfiable
	}
	return nil
}

type FuenteDatosBaremacion interface {
	ObtenerCriterio(context.Context, SolicitudObtenerCriterioBaremacion) (CriterioBaremacionConfiable, error)
	ObtenerEvidencia(context.Context, SolicitudObtenerEvidenciaBaremacion) (EvidenciaBaremacionConfiable, error)
	ObtenerRepresentacion(context.Context, SolicitudObtenerRepresentacionBaremacion) (RepresentacionBaremacionConfiable, error)
}

// SolicitudCalcularPuntuacionOficial no contiene PuntosCalculados. Solo admite
// criterio gobernado y evidencias que ya han pasado por FuenteDatosBaremacion.
type SolicitudCalcularPuntuacionOficial struct {
	Contexto            ContextoOperacionBaremacion
	BaremacionMeritoRef string
	ProcesoRef          string
	SolicitudRef        string
	SujetoRef           string
	Criterio            dominiobolsa.ReferenciaCriterio
	Evidencias          []EvidenciaBaremacionConfiable
	PuntosDeclarados    dominiobolsa.Puntos
	SolicitadaEn        time.Time
}

func (s SolicitudCalcularPuntuacionOficial) Validar() error {
	if s.Contexto.ValidarPara(AccionCalcularPuntuacionBaremacion, ClaseRecursoBaremacion, s.BaremacionMeritoRef) != nil ||
		s.Contexto.Proyeccion().SujetoRef != s.SujetoRef ||
		!referenciaValida(s.BaremacionMeritoRef, 512) || !referenciaValida(s.ProcesoRef, 512) ||
		!referenciaValida(s.SolicitudRef, 512) || !referenciaValida(s.SujetoRef, 512) ||
		s.Criterio.Validar() != nil || s.Criterio.ProcesoRef != s.ProcesoRef ||
		!s.PuntosDeclarados.Validos() || s.SolicitadaEn.IsZero() || len(s.Evidencias) == 0 ||
		len(s.Evidencias) > maximoEvidenciasCalculo {
		return ErrSolicitudBaremacionInvalida
	}
	vistas := make(map[string]struct{}, len(s.Evidencias))
	for _, evidencia := range s.Evidencias {
		if evidencia.Validar() != nil {
			return ErrSolicitudBaremacionInvalida
		}
		clave := evidencia.Evidencia.Referencia.DocumentoRef + "\x00" +
			evidencia.Evidencia.Referencia.RepresentacionRef + "\x00" + evidencia.Evidencia.Referencia.HuellaSHA256
		if _, existe := vistas[clave]; existe {
			return ErrSolicitudBaremacionInvalida
		}
		vistas[clave] = struct{}{}
	}
	return nil
}

func (s SolicitudCalcularPuntuacionOficial) Clonar() (SolicitudCalcularPuntuacionOficial, error) {
	clon := s
	clon.Evidencias = make([]EvidenciaBaremacionConfiable, len(s.Evidencias))
	for indice := range s.Evidencias {
		evidencia, err := s.Evidencias[indice].Clonar()
		if err != nil {
			return SolicitudCalcularPuntuacionOficial{}, err
		}
		clon.Evidencias[indice] = evidencia
	}
	return clon, clon.Validar()
}

type ResultadoCalculoOficial struct {
	Calculo               dominiobolsa.CalculoOficialBaremacion
	EvidenciaGobiernoRef  string
	HuellaEvidenciaSHA256 string
}

func (r ResultadoCalculoOficial) Validar() error {
	if r.Calculo.Validar() != nil || !referenciaValida(r.EvidenciaGobiernoRef, 512) ||
		!huellaSHA256Valida(r.HuellaEvidenciaSHA256) {
		return ErrCalculoOficialNoReproducible
	}
	return nil
}

func (r ResultadoCalculoOficial) Clonar() (ResultadoCalculoOficial, error) {
	calculo, err := r.Calculo.ClonarCanonico()
	if err != nil {
		return ResultadoCalculoOficial{}, err
	}
	clon := r
	clon.Calculo = calculo
	return clon, clon.Validar()
}

func (r ResultadoCalculoOficial) ValidarPara(s SolicitudCalcularPuntuacionOficial) error {
	if s.Validar() != nil || r.Validar() != nil || r.Calculo.ProcesoRef != s.ProcesoRef ||
		r.Calculo.SolicitudRef != s.SolicitudRef || r.Calculo.SujetoRef != s.SujetoRef ||
		r.Calculo.BaremacionMeritoRef != s.BaremacionMeritoRef || r.Calculo.Criterio != s.Criterio ||
		r.Calculo.Regla != s.Criterio.ReglaCalculo || r.Calculo.CalculadoEn.Before(s.SolicitadaEn) {
		return ErrCalculoOficialNoReproducible
	}
	evidencias := make([]dominiobolsa.EvidenciaMerito, len(s.Evidencias))
	for indice := range s.Evidencias {
		evidencias[indice] = clonarEvidencia(s.Evidencias[indice].Evidencia)
	}
	esperado := r.Calculo
	esperado.Evidencias = evidencias
	if !r.Calculo.CoincideCon(esperado) {
		return ErrCalculoOficialNoReproducible
	}
	return nil
}

type SolicitudRecuperarCalculoOficial struct {
	Contexto        ContextoOperacionBaremacion
	CalculoRef      string
	HuellaResultado string
}

func (s SolicitudRecuperarCalculoOficial) Validar() error {
	if s.Contexto.ValidarPara(AccionRecuperarCalculoBaremacion, ClaseRecursoCalculo, s.CalculoRef) != nil ||
		!referenciaValida(s.CalculoRef, 512) || !huellaSHA256Valida(s.HuellaResultado) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type CalculadorOficialBaremacion interface {
	CalcularPuntuacionOficial(context.Context, SolicitudCalcularPuntuacionOficial) (ResultadoCalculoOficial, error)
	RecuperarCalculoOficial(context.Context, SolicitudRecuperarCalculoOficial) (ResultadoCalculoOficial, error)
}

type PoliticaFirmaBaremacion struct {
	Referencia                      string
	Version                         int
	HuellaSHA256                    string
	FormatoFirmaClave               string
	PerfilFirmaClave                string
	AlgoritmoHuellaClave            string
	ComprobacionesObligatorias      []string
	RequiereFirmaInteractiva        bool
	RequiereValidacionServidor      bool
	RequiereSelloTiempo             bool
	PoliticaSelloTiempoRef          string
	PoliticaSelloTiempoVersion      int
	HuellaPoliticaSelloTiempoSHA256 string
	RequiereAumentoLongevidad       bool
	NivelAumentoClave               string
	PoliticaLongevidadRef           string
	PoliticaLongevidadVersion       int
	HuellaPoliticaLongevidadSHA256  string
	AprobacionRef                   string
	HuellaAprobacionSHA256          string
	VigenteDesde                    time.Time
	VigenteHasta                    time.Time
}

func (p PoliticaFirmaBaremacion) Validar() error {
	if !referenciaValida(p.Referencia, 512) || p.Version < 1 || !huellaSHA256Valida(p.HuellaSHA256) ||
		p.FormatoFirmaClave != FormatoFirmaPDFCanonico || !perfilFirmaPermitido(p.PerfilFirmaClave) ||
		p.AlgoritmoHuellaClave != AlgoritmoHuellaFirmaSHA256 ||
		!mismoConjuntoClaves(p.ComprobacionesObligatorias, comprobacionesFirmaObligatorias) || !p.RequiereFirmaInteractiva ||
		!p.RequiereValidacionServidor || !referenciaValida(p.AprobacionRef, 512) ||
		!huellaSHA256Valida(p.HuellaAprobacionSHA256) || p.VigenteDesde.IsZero() ||
		p.VigenteHasta.IsZero() || !p.VigenteHasta.After(p.VigenteDesde) {
		return ErrPoliticaFirmaInsegura
	}
	selloPresente := p.PoliticaSelloTiempoRef != "" || p.PoliticaSelloTiempoVersion != 0 ||
		p.HuellaPoliticaSelloTiempoSHA256 != ""
	if p.RequiereSelloTiempo != selloPresente || (p.RequiereSelloTiempo &&
		(!referenciaValida(p.PoliticaSelloTiempoRef, 512) || p.PoliticaSelloTiempoVersion < 1 ||
			!huellaSHA256Valida(p.HuellaPoliticaSelloTiempoSHA256))) {
		return ErrPoliticaFirmaInsegura
	}
	longevidadPresente := p.NivelAumentoClave != "" || p.PoliticaLongevidadRef != "" ||
		p.PoliticaLongevidadVersion != 0 || p.HuellaPoliticaLongevidadSHA256 != ""
	if p.RequiereAumentoLongevidad != longevidadPresente || (p.RequiereAumentoLongevidad &&
		(!claveValida(p.NivelAumentoClave) || !referenciaValida(p.PoliticaLongevidadRef, 512) ||
			p.PoliticaLongevidadVersion < 1 || !huellaSHA256Valida(p.HuellaPoliticaLongevidadSHA256))) {
		return ErrPoliticaFirmaInsegura
	}
	if (p.PerfilFirmaClave == PerfilFirmaPAdESBaselineB && (p.RequiereSelloTiempo || p.RequiereAumentoLongevidad)) ||
		(p.PerfilFirmaClave == PerfilFirmaPAdESBaselineT && (!p.RequiereSelloTiempo || p.RequiereAumentoLongevidad)) ||
		(p.PerfilFirmaClave == PerfilFirmaPAdESBaselineLTA && (!p.RequiereSelloTiempo || !p.RequiereAumentoLongevidad)) {
		return ErrPoliticaFirmaInsegura
	}
	return nil
}

func (p PoliticaFirmaBaremacion) VigenteEn(instante time.Time) bool {
	return p.Validar() == nil && !instante.Before(p.VigenteDesde) && instante.Before(p.VigenteHasta)
}

func (p PoliticaFirmaBaremacion) ValidarPara(s SolicitudObtenerPoliticaFirma) error {
	if s.Validar() != nil || p.Validar() != nil || p.Referencia != s.Referencia || p.Version != s.Version ||
		p.HuellaSHA256 != s.HuellaEsperadaSHA256 || !p.VigenteEn(s.VigenteEn) {
		return ErrPoliticaFirmaNoVigente
	}
	return nil
}

type SolicitudObtenerPoliticaFirma struct {
	Contexto             ContextoOperacionBaremacion
	Referencia           string
	Version              int
	HuellaEsperadaSHA256 string
	VigenteEn            time.Time
}

func (s SolicitudObtenerPoliticaFirma) Validar() error {
	if s.Contexto.ValidarPara(AccionConsultarPoliticaFirmaBaremacion, ClaseRecursoPoliticaFirma, s.Referencia) != nil ||
		!referenciaValida(s.Referencia, 512) || s.Version < 1 ||
		!huellaSHA256Valida(s.HuellaEsperadaSHA256) || s.VigenteEn.IsZero() {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type CatalogoPoliticasFirmaBaremacion interface {
	ObtenerPoliticaFirma(context.Context, SolicitudObtenerPoliticaFirma) (PoliticaFirmaBaremacion, error)
}

type SolicitudCodificarDecisionCanonica struct {
	Contexto             ContextoOperacionBaremacion
	AutorizacionDecision ContextoOperacionBaremacion
	Contenido            dominiobolsa.ContenidoDecisionTecnica
	Politica             PoliticaFirmaBaremacion
}

func (s SolicitudCodificarDecisionCanonica) Validar() error {
	accionAdopcion, conocida := AccionAdopcionParaClase(s.Contenido.Clase)
	if !conocida || s.Contexto.ValidarPara(AccionCodificarDecisionBaremacion, ClaseRecursoDecision, s.Contenido.ID) != nil ||
		s.AutorizacionDecision.ValidarPara(
			accionAdopcion, ClaseRecursoBaremacion, s.Contenido.BaremacionMeritoRef,
		) != nil ||
		s.Contenido.Validar() != nil || s.Politica.Validar() != nil ||
		s.Contexto.Proyeccion().SujetoRef != s.Contenido.SujetoRef || s.Contexto.Proyeccion().PrincipalRef != s.Contenido.DecisorRef ||
		s.Contexto.Proyeccion().PerfilActorClave != s.Contenido.PerfilDecisorClave ||
		s.AutorizacionDecision.Proyeccion().SujetoRef != s.Contenido.SujetoRef ||
		s.AutorizacionDecision.Proyeccion().PrincipalRef != s.Contenido.DecisorRef ||
		s.AutorizacionDecision.Proyeccion().PerfilActorClave != s.Contenido.PerfilDecisorClave ||
		s.AutorizacionDecision.Proyeccion().AutorizacionRef != s.Contenido.AutorizacionRef ||
		s.Contexto.Proyeccion().AutorizacionRef == s.Contenido.AutorizacionRef ||
		s.Contexto.Proyeccion().FinalidadClave != s.Contenido.FinalidadClave ||
		s.AutorizacionDecision.Proyeccion().FinalidadClave != s.Contenido.FinalidadClave ||
		s.Contexto.Proyeccion().CorrelacionRef != s.Contenido.CorrelacionRef ||
		s.AutorizacionDecision.Proyeccion().CorrelacionRef != s.Contenido.CorrelacionRef {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (s SolicitudCodificarDecisionCanonica) Clonar() (SolicitudCodificarDecisionCanonica, error) {
	contenido, err := s.Contenido.ClonarCanonico()
	if err != nil {
		return SolicitudCodificarDecisionCanonica{}, err
	}
	clon := s
	clon.Contenido = contenido
	return clon, clon.Validar()
}

type CodificacionCanonicaDecision struct {
	Carga                       CargaProtegida
	ProcesoRef                  string
	SolicitudRef                string
	SujetoRef                   string
	BaremacionMeritoRef         string
	DecisionRef                 string
	VersionBaremacion           uint64
	PrincipalRef                string
	PerfilActorClave            string
	AutorizacionDecisionRef     string
	AutorizacionCodificacionRef string
	FinalidadClave              string
	CorrelacionRef              string
	FormatoClave                string
	MIME                        string
	HuellaContenidoSHA256       string
	HuellaDocumentoSHA256       string
	VersionCodificador          string
}

func (c CodificacionCanonicaDecision) Validar() error {
	if c.Carga.Validar() != nil || !referenciaValida(c.ProcesoRef, 512) || !referenciaValida(c.SolicitudRef, 512) ||
		!referenciaValida(c.SujetoRef, 512) || !referenciaValida(c.BaremacionMeritoRef, 512) ||
		!referenciaValida(c.DecisionRef, 512) || c.VersionBaremacion < 2 || !claveValida(c.FormatoClave) || !mimeValido(c.MIME) ||
		!referenciaValida(c.PrincipalRef, 512) || !claveValida(c.PerfilActorClave) ||
		!referenciaValida(c.AutorizacionDecisionRef, 512) ||
		!referenciaValida(c.AutorizacionCodificacionRef, 512) ||
		c.AutorizacionDecisionRef == c.AutorizacionCodificacionRef || !claveValida(c.FinalidadClave) ||
		!referenciaValida(c.CorrelacionRef, 512) ||
		!huellaSHA256Valida(c.HuellaContenidoSHA256) || !huellaSHA256Valida(c.HuellaDocumentoSHA256) ||
		!referenciaValida(c.VersionCodificador, 256) {
		return ErrCodificacionCanonicaNoDisponible
	}
	return nil
}

func (c CodificacionCanonicaDecision) ValidarPara(s SolicitudCodificarDecisionCanonica) error {
	huella, err := s.Contenido.HuellaContenidoSHA256()
	if s.Validar() != nil || c.Validar() != nil || err != nil || c.FormatoClave != s.Politica.FormatoFirmaClave ||
		c.ProcesoRef != s.Contenido.ProcesoRef || c.SolicitudRef != s.Contenido.SolicitudRef ||
		c.SujetoRef != s.Contenido.SujetoRef || c.BaremacionMeritoRef != s.Contenido.BaremacionMeritoRef ||
		c.DecisionRef != s.Contenido.ID || c.VersionBaremacion != s.Contenido.VersionBaremacion ||
		c.PrincipalRef != s.Contexto.Proyeccion().PrincipalRef || c.PrincipalRef != s.Contenido.DecisorRef ||
		c.PerfilActorClave != s.Contexto.Proyeccion().PerfilActorClave || c.PerfilActorClave != s.Contenido.PerfilDecisorClave ||
		c.AutorizacionDecisionRef != s.AutorizacionDecision.Proyeccion().AutorizacionRef ||
		c.AutorizacionDecisionRef != s.Contenido.AutorizacionRef ||
		c.AutorizacionCodificacionRef != s.Contexto.Proyeccion().AutorizacionRef ||
		c.FinalidadClave != s.Contexto.Proyeccion().FinalidadClave || c.FinalidadClave != s.Contenido.FinalidadClave ||
		c.CorrelacionRef != s.Contexto.Proyeccion().CorrelacionRef || c.CorrelacionRef != s.Contenido.CorrelacionRef ||
		c.HuellaContenidoSHA256 != huella || c.Carga.Tamano() < 1 {
		return ErrCodificacionCanonicaNoDisponible
	}
	return nil
}

type CodificadorCanonicoDecision interface {
	CodificarDecision(context.Context, SolicitudCodificarDecisionCanonica) (CodificacionCanonicaDecision, error)
}

// AlmacenDocumentosFirmables limita la custodia de decisiones firmadas a la
// lista positiva de operaciones que necesita este flujo. En particular, no
// concede lectura, promocion, inmovilizacion ni eliminacion de objetos.
type AlmacenDocumentosFirmables interface {
	Capacidades(context.Context) (puertosvec.CapacidadesAlmacenObjetos, error)
	Escribir(context.Context, puertosvec.SolicitudEscribirObjeto) (puertosvec.ResultadoOperacionObjeto, error)
	AplicarRetencion(context.Context, puertosvec.SolicitudRetenerObjeto) (puertosvec.ResultadoOperacionObjeto, error)
}

type SolicitudCustodiarDocumentoFirmable struct {
	Contexto            ContextoOperacionBaremacion
	OperacionRef        string
	ClaveIdempotencia   string
	CargaRef            string
	SujetoSeudonimoHMAC string
	HuellaAlmacenHMAC   string
	EfectoRef           string
	ProcesoRef          string
	SolicitudRef        string
	BaremacionMeritoRef string
	DecisionRef         string
	ClasificacionClave  string
	Codificacion        CodificacionCanonicaDecision
}

func (s SolicitudCustodiarDocumentoFirmable) Validar() error {
	if s.Contexto.ValidarPara(AccionCustodiarDecisionBaremacion, ClaseRecursoDecision, s.DecisionRef) != nil ||
		!referenciaValida(s.OperacionRef, 512) ||
		!referenciaValida(s.ClaveIdempotencia, 512) || !referenciaValida(s.CargaRef, 512) ||
		!huellaHMACSHA256Valida(s.SujetoSeudonimoHMAC) || !huellaHMACSHA256Valida(s.HuellaAlmacenHMAC) ||
		!referenciaValida(s.EfectoRef, 512) ||
		!referenciaValida(s.ProcesoRef, 512) ||
		!referenciaValida(s.SolicitudRef, 512) || !referenciaValida(s.BaremacionMeritoRef, 512) ||
		!referenciaValida(s.DecisionRef, 512) || !claveValida(s.ClasificacionClave) ||
		s.Codificacion.Validar() != nil || s.Contexto.Proyeccion().SujetoRef != s.Codificacion.SujetoRef ||
		s.Contexto.Proyeccion().PrincipalRef != s.Codificacion.PrincipalRef ||
		s.Contexto.Proyeccion().PerfilActorClave != s.Codificacion.PerfilActorClave ||
		s.Contexto.Proyeccion().AutorizacionRef == s.Codificacion.AutorizacionDecisionRef ||
		s.Contexto.Proyeccion().AutorizacionRef == s.Codificacion.AutorizacionCodificacionRef ||
		s.Contexto.Proyeccion().FinalidadClave != s.Codificacion.FinalidadClave ||
		s.Contexto.Proyeccion().CorrelacionRef != s.Codificacion.CorrelacionRef ||
		s.ProcesoRef != s.Codificacion.ProcesoRef || s.SolicitudRef != s.Codificacion.SolicitudRef ||
		s.BaremacionMeritoRef != s.Codificacion.BaremacionMeritoRef || s.DecisionRef != s.Codificacion.DecisionRef {
		return ErrCustodiaDocumentoFirmableInvalida
	}
	return nil
}

// PrepararEscritura crea el puente exacto hacia el almacen VEC. La carga se
// vuelve a copiar y la zona admitida evita que un artefacto no confiable llegue
// al firmador. El conector VEC verifica tamano y SHA-256 al escribir.
func (s SolicitudCustodiarDocumentoFirmable) PrepararEscritura() (puertosvec.SolicitudEscribirObjeto, error) {
	if err := s.Validar(); err != nil {
		return puertosvec.SolicitudEscribirObjeto{}, err
	}
	contenido := s.Codificacion.Carga.Revelar()
	contextoAlmacen, err := s.Contexto.CrearContextoAlmacenCustodiarDecision(
		puertosvec.VinculosOperacionAlmacen{
			OperacionRef: s.OperacionRef, CargaRef: s.CargaRef,
			Clasificacion: s.ClasificacionClave, SujetoSeudonimoHMAC: s.SujetoSeudonimoHMAC,
			HuellaSolicitudHMAC: s.HuellaAlmacenHMAC, EfectoRef: s.EfectoRef,
		},
	)
	if err != nil {
		return puertosvec.SolicitudEscribirObjeto{}, ErrCustodiaDocumentoFirmableInvalida
	}
	solicitud := puertosvec.SolicitudEscribirObjeto{
		Contexto:          contextoAlmacen,
		ClaveIdempotencia: s.ClaveIdempotencia, Zona: puertosvec.ZonaAlmacenAdmitida,
		MIME: s.Codificacion.MIME, Tamano: int64(len(contenido)),
		HuellaSHA256: s.Codificacion.HuellaDocumentoSHA256, Contenido: bytes.NewReader(contenido),
	}
	if err := solicitud.Validar(); err != nil {
		return puertosvec.SolicitudEscribirObjeto{}, ErrCustodiaDocumentoFirmableInvalida
	}
	return solicitud, nil
}

type DocumentoFirmableCustodiado struct {
	ProcesoRef                  string
	SolicitudRef                string
	SujetoRef                   string
	BaremacionMeritoRef         string
	DecisionRef                 string
	VersionBaremacion           uint64
	PrincipalRef                string
	PerfilActorClave            string
	AutorizacionDecisionRef     string
	AutorizacionCodificacionRef string
	AutorizacionCustodiaRef     string
	Objeto                      puertosvec.ObjetoAlmacenado
	EvidenciaCustodia           puertosvec.EvidenciaOperacionAlmacen
	FormatoClave                string
	MIME                        string
	Tamano                      int64
	HuellaContenidoSHA256       string
	HuellaDocumentoSHA256       string
	VersionCodificador          string
}

func NuevoDocumentoFirmableCustodiado(s SolicitudCustodiarDocumentoFirmable, r puertosvec.ResultadoOperacionObjeto) (DocumentoFirmableCustodiado, error) {
	d := DocumentoFirmableCustodiado{
		ProcesoRef: s.ProcesoRef, SolicitudRef: s.SolicitudRef, SujetoRef: s.Contexto.Proyeccion().SujetoRef,
		BaremacionMeritoRef: s.BaremacionMeritoRef, DecisionRef: s.DecisionRef,
		VersionBaremacion: s.Codificacion.VersionBaremacion,
		PrincipalRef:      s.Codificacion.PrincipalRef, PerfilActorClave: s.Codificacion.PerfilActorClave,
		AutorizacionDecisionRef:     s.Codificacion.AutorizacionDecisionRef,
		AutorizacionCodificacionRef: s.Codificacion.AutorizacionCodificacionRef,
		AutorizacionCustodiaRef:     s.Contexto.Proyeccion().AutorizacionRef,
		Objeto:                      r.Objeto, EvidenciaCustodia: r.Evidencia, FormatoClave: s.Codificacion.FormatoClave,
		MIME: s.Codificacion.MIME, Tamano: int64(s.Codificacion.Carga.Tamano()),
		HuellaContenidoSHA256: s.Codificacion.HuellaContenidoSHA256,
		HuellaDocumentoSHA256: s.Codificacion.HuellaDocumentoSHA256,
		VersionCodificador:    s.Codificacion.VersionCodificador,
	}
	if s.Validar() != nil || r.Validar() != nil || d.Validar() != nil ||
		r.Evidencia.OperacionRef != s.OperacionRef || r.Evidencia.CorrelacionRef != s.Contexto.Proyeccion().CorrelacionRef ||
		r.Evidencia.AutorizacionRef != s.Contexto.Proyeccion().AutorizacionRef || r.Evidencia.Finalidad != s.Contexto.Proyeccion().FinalidadClave ||
		r.Evidencia.Clasificacion != s.ClasificacionClave || r.Evidencia.Accion != puertosvec.AccionAlmacenEscribir ||
		r.Evidencia.CargaRef != s.CargaRef || r.Evidencia.SujetoSeudonimoHMAC != s.SujetoSeudonimoHMAC ||
		r.Evidencia.RecursoRef != s.DecisionRef || r.Evidencia.ModuloID != "bolsa" ||
		r.Evidencia.HuellaSolicitudHMAC != s.HuellaAlmacenHMAC {
		return DocumentoFirmableCustodiado{}, ErrCustodiaDocumentoFirmableInvalida
	}
	return d, nil
}

func (d DocumentoFirmableCustodiado) Validar() error {
	if !referenciaValida(d.ProcesoRef, 512) || !referenciaValida(d.SolicitudRef, 512) ||
		!referenciaValida(d.SujetoRef, 512) || !referenciaValida(d.BaremacionMeritoRef, 512) ||
		!referenciaValida(d.DecisionRef, 512) || d.VersionBaremacion < 2 ||
		!referenciaValida(d.PrincipalRef, 512) || !claveValida(d.PerfilActorClave) ||
		!referenciaValida(d.AutorizacionDecisionRef, 512) ||
		!referenciaValida(d.AutorizacionCodificacionRef, 512) ||
		!referenciaValida(d.AutorizacionCustodiaRef, 512) ||
		d.AutorizacionDecisionRef == d.AutorizacionCodificacionRef ||
		d.AutorizacionDecisionRef == d.AutorizacionCustodiaRef ||
		d.AutorizacionCodificacionRef == d.AutorizacionCustodiaRef ||
		d.Objeto.Validar() != nil || d.EvidenciaCustodia.Validar() != nil ||
		d.EvidenciaCustodia.AutorizacionRef != d.AutorizacionCustodiaRef ||
		d.Objeto.Objeto != d.EvidenciaCustodia.Objeto || d.Objeto.ConectorID != d.EvidenciaCustodia.ConectorID ||
		d.Objeto.Zona != puertosvec.ZonaAlmacenAdmitida || !claveValida(d.FormatoClave) || !mimeValido(d.MIME) ||
		d.Tamano < 1 || d.Objeto.Tamano != d.Tamano || d.Objeto.MIME != d.MIME ||
		!huellaSHA256Valida(d.HuellaContenidoSHA256) || !huellaSHA256Valida(d.HuellaDocumentoSHA256) ||
		d.Objeto.HuellaSHA256 != d.HuellaDocumentoSHA256 || !referenciaValida(d.VersionCodificador, 256) {
		return ErrCustodiaDocumentoFirmableInvalida
	}
	return nil
}

func (d DocumentoFirmableCustodiado) ValidarPara(s SolicitudCustodiarDocumentoFirmable) error {
	c := s.Codificacion
	if s.Validar() != nil || d.Validar() != nil || d.ProcesoRef != s.ProcesoRef || d.SolicitudRef != s.SolicitudRef ||
		d.SujetoRef != s.Contexto.Proyeccion().SujetoRef || d.BaremacionMeritoRef != s.BaremacionMeritoRef ||
		d.DecisionRef != s.DecisionRef || d.VersionBaremacion != c.VersionBaremacion ||
		d.PrincipalRef != s.Contexto.Proyeccion().PrincipalRef || d.PerfilActorClave != s.Contexto.Proyeccion().PerfilActorClave ||
		d.AutorizacionDecisionRef != c.AutorizacionDecisionRef ||
		d.AutorizacionCodificacionRef != c.AutorizacionCodificacionRef ||
		d.AutorizacionCustodiaRef != s.Contexto.Proyeccion().AutorizacionRef ||
		c.Validar() != nil || d.FormatoClave != c.FormatoClave || d.MIME != c.MIME ||
		d.Tamano != int64(c.Carga.Tamano()) || d.HuellaContenidoSHA256 != c.HuellaContenidoSHA256 ||
		d.HuellaDocumentoSHA256 != c.HuellaDocumentoSHA256 || d.VersionCodificador != c.VersionCodificador {
		return ErrCustodiaDocumentoFirmableInvalida
	}
	return nil
}

type ContextoOperacionFirma struct {
	ContextoOperacionBaremacion
	OperacionRef string
}

func (c ContextoOperacionFirma) Validar() error {
	if c.ContextoOperacionBaremacion.Validar() != nil || !referenciaValida(c.OperacionRef, 512) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type EstadoSesionFirmaInteractiva string

const (
	EstadoSesionFirmaPreparada  EstadoSesionFirmaInteractiva = "preparada"
	EstadoSesionFirmaPendiente  EstadoSesionFirmaInteractiva = "pendiente"
	EstadoSesionFirmaCompletada EstadoSesionFirmaInteractiva = "completada"
	EstadoSesionFirmaRechazada  EstadoSesionFirmaInteractiva = "rechazada"
	EstadoSesionFirmaCancelada  EstadoSesionFirmaInteractiva = "cancelada"
	EstadoSesionFirmaExpirada   EstadoSesionFirmaInteractiva = "expirada"
	EstadoSesionFirmaFallida    EstadoSesionFirmaInteractiva = "fallida"
)

func (e EstadoSesionFirmaInteractiva) Valido() bool {
	switch e {
	case EstadoSesionFirmaPreparada, EstadoSesionFirmaPendiente, EstadoSesionFirmaCompletada,
		EstadoSesionFirmaRechazada, EstadoSesionFirmaCancelada, EstadoSesionFirmaExpirada, EstadoSesionFirmaFallida:
		return true
	default:
		return false
	}
}

type ArtefactoFirma struct {
	ProcesoRef                       string
	SolicitudRef                     string
	SujetoRef                        string
	BaremacionMeritoRef              string
	DecisionRef                      string
	VersionBaremacion                uint64
	SesionFirmaRef                   string
	EvidenciaFirmaInteractivaRef     string
	HuellaEvidenciaInteractivaSHA256 string
	DocumentoFirmable                puertosvec.ReferenciaObjetoAlmacen
	HuellaDocumentoFirmableSHA256    string
	EvidenciaCustodiaRef             string
	FirmaRef                         string
	HuellaFirmaSHA256                string
	DocumentoFirmadoRef              string
	HuellaDocumentoSHA256            string
	HuellaContenidoSHA256            string
	PoliticaFirmaRef                 string
	PoliticaFirmaVersion             int
	HuellaPoliticaFirmaSHA256        string
	FirmanteRef                      string
	PerfilFirmanteClave              string
	FirmadaEn                        time.Time
}

func (a ArtefactoFirma) Validar() error {
	if !referenciaValida(a.ProcesoRef, 512) || !referenciaValida(a.SolicitudRef, 512) ||
		!referenciaValida(a.SujetoRef, 512) || !referenciaValida(a.BaremacionMeritoRef, 512) ||
		!referenciaValida(a.DecisionRef, 512) || a.VersionBaremacion < 2 ||
		!referenciaValida(a.SesionFirmaRef, 512) || !referenciaValida(a.EvidenciaFirmaInteractivaRef, 512) ||
		!huellaSHA256Valida(a.HuellaEvidenciaInteractivaSHA256) || !referenciaValida(a.FirmaRef, 512) ||
		a.DocumentoFirmable.Validar() != nil || !huellaSHA256Valida(a.HuellaDocumentoFirmableSHA256) ||
		!referenciaValida(a.EvidenciaCustodiaRef, 512) ||
		!huellaSHA256Valida(a.HuellaFirmaSHA256) || !referenciaValida(a.DocumentoFirmadoRef, 512) ||
		!huellaSHA256Valida(a.HuellaDocumentoSHA256) || !huellaSHA256Valida(a.HuellaContenidoSHA256) ||
		!referenciaValida(a.PoliticaFirmaRef, 512) || a.PoliticaFirmaVersion < 1 ||
		!huellaSHA256Valida(a.HuellaPoliticaFirmaSHA256) || !referenciaValida(a.FirmanteRef, 512) ||
		!claveValida(a.PerfilFirmanteClave) || a.FirmadaEn.IsZero() {
		return ErrFirmaInteractivaNoCompletada
	}
	return nil
}

type SolicitudPrepararFirmaInteractiva struct {
	Contexto            ContextoOperacionFirma
	ClaveIdempotencia   string
	ProcesoRef          string
	SolicitudRef        string
	BaremacionMeritoRef string
	DecisionRef         string
	Documento           DocumentoFirmableCustodiado
	FirmanteRef         string
	PerfilFirmanteClave string
	Politica            PoliticaFirmaBaremacion
	SolicitadaEn        time.Time
	ExpiraEn            time.Time
}

func (s SolicitudPrepararFirmaInteractiva) Validar() error {
	if s.Contexto.ContextoOperacionBaremacion.ValidarPara(AccionPrepararFirmaDecisionBaremacion, ClaseRecursoDecision, s.DecisionRef) != nil ||
		s.Contexto.Validar() != nil || !referenciaValida(s.ClaveIdempotencia, 512) ||
		!referenciaValida(s.ProcesoRef, 512) || !referenciaValida(s.SolicitudRef, 512) ||
		!referenciaValida(s.BaremacionMeritoRef, 512) || !referenciaValida(s.DecisionRef, 512) ||
		s.Documento.Validar() != nil || s.Documento.ProcesoRef != s.ProcesoRef ||
		s.Documento.SolicitudRef != s.SolicitudRef || s.Documento.SujetoRef != s.Contexto.Proyeccion().SujetoRef ||
		s.Documento.BaremacionMeritoRef != s.BaremacionMeritoRef || s.Documento.DecisionRef != s.DecisionRef ||
		s.Documento.PrincipalRef != s.Contexto.Proyeccion().PrincipalRef || s.Documento.PerfilActorClave != s.Contexto.Proyeccion().PerfilActorClave ||
		s.Documento.EvidenciaCustodia.AutorizacionRef != s.Documento.AutorizacionCustodiaRef ||
		s.Contexto.Proyeccion().AutorizacionRef == s.Documento.AutorizacionDecisionRef ||
		s.Contexto.Proyeccion().AutorizacionRef == s.Documento.AutorizacionCodificacionRef ||
		s.Contexto.Proyeccion().AutorizacionRef == s.Documento.AutorizacionCustodiaRef ||
		s.Documento.EvidenciaCustodia.Finalidad != s.Contexto.Proyeccion().FinalidadClave ||
		s.Documento.EvidenciaCustodia.CorrelacionRef != s.Contexto.Proyeccion().CorrelacionRef ||
		!referenciaValida(s.FirmanteRef, 512) || !claveValida(s.PerfilFirmanteClave) ||
		s.FirmanteRef != s.Contexto.Proyeccion().PrincipalRef || s.PerfilFirmanteClave != s.Contexto.Proyeccion().PerfilActorClave ||
		s.Politica.Validar() != nil || !s.Politica.VigenteEn(s.SolicitadaEn.UTC()) ||
		!ventanaValida(s.SolicitadaEn, s.ExpiraEn, VentanaMaximaSesionFirmaInteractiva) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (a ArtefactoFirma) ValidarPara(s SolicitudPrepararFirmaInteractiva, sesion SesionFirmaInteractiva) error {
	if s.Validar() != nil || sesion.ValidarPara(s) != nil || a.Validar() != nil ||
		a.ProcesoRef != s.ProcesoRef || a.SolicitudRef != s.SolicitudRef || a.SujetoRef != s.Contexto.Proyeccion().SujetoRef ||
		a.BaremacionMeritoRef != s.BaremacionMeritoRef || a.DecisionRef != s.DecisionRef ||
		a.VersionBaremacion != s.Documento.VersionBaremacion ||
		a.SesionFirmaRef != sesion.SesionRef || a.DocumentoFirmable != s.Documento.Objeto.Objeto ||
		a.HuellaDocumentoFirmableSHA256 != s.Documento.HuellaDocumentoSHA256 ||
		a.EvidenciaCustodiaRef != s.Documento.EvidenciaCustodia.Referencia ||
		a.HuellaContenidoSHA256 != s.Documento.HuellaContenidoSHA256 ||
		a.PoliticaFirmaRef != s.Politica.Referencia || a.PoliticaFirmaVersion != s.Politica.Version ||
		a.HuellaPoliticaFirmaSHA256 != s.Politica.HuellaSHA256 || a.FirmanteRef != s.FirmanteRef ||
		a.PerfilFirmanteClave != s.PerfilFirmanteClave || !s.Politica.VigenteEn(a.FirmadaEn.UTC()) ||
		a.FirmadaEn.Before(s.SolicitadaEn) ||
		!a.FirmadaEn.Before(s.ExpiraEn) {
		return ErrFirmaInteractivaNoCompletada
	}
	return nil
}

type SesionFirmaInteractiva struct {
	SesionRef               string
	Estado                  EstadoSesionFirmaInteractiva
	CargaLanzamiento        CargaProtegida
	MIMELanzamiento         string
	Documento               DocumentoFirmableCustodiado
	PoliticaFirmaRef        string
	PoliticaFirmaVersion    int
	HuellaPoliticaSHA256    string
	EvidenciaPreparacionRef string
	HuellaEvidenciaSHA256   string
	PreparadaEn             time.Time
	ExpiraEn                time.Time
}

func (s SesionFirmaInteractiva) Validar() error {
	if !referenciaValida(s.SesionRef, 512) || !s.Estado.Valido() || s.CargaLanzamiento.Validar() != nil ||
		!mimeValido(s.MIMELanzamiento) || s.Documento.Validar() != nil ||
		!referenciaValida(s.PoliticaFirmaRef, 512) || s.PoliticaFirmaVersion < 1 ||
		!huellaSHA256Valida(s.HuellaPoliticaSHA256) || !referenciaValida(s.EvidenciaPreparacionRef, 512) ||
		!huellaSHA256Valida(s.HuellaEvidenciaSHA256) ||
		!ventanaValida(s.PreparadaEn, s.ExpiraEn, VentanaMaximaSesionFirmaInteractiva) {
		return ErrFirmaInteractivaNoDisponible
	}
	return nil
}

func (s SesionFirmaInteractiva) ValidarPara(solicitud SolicitudPrepararFirmaInteractiva) error {
	if solicitud.Validar() != nil || s.Validar() != nil || s.Documento != solicitud.Documento ||
		s.PoliticaFirmaRef != solicitud.Politica.Referencia || s.PoliticaFirmaVersion != solicitud.Politica.Version ||
		s.HuellaPoliticaSHA256 != solicitud.Politica.HuellaSHA256 || s.PreparadaEn.Before(solicitud.SolicitadaEn) ||
		s.ExpiraEn.After(solicitud.ExpiraEn) {
		return ErrFirmaInteractivaNoDisponible
	}
	return nil
}

type SolicitudConsultarFirmaInteractiva struct {
	Contexto              ContextoOperacionFirma
	SesionRef             string
	Documento             DocumentoFirmableCustodiado
	HuellaContenidoSHA256 string
	PoliticaFirmaRef      string
	PoliticaFirmaVersion  int
	HuellaPoliticaSHA256  string
	FirmanteRef           string
	PerfilFirmanteClave   string
}

func (s SolicitudConsultarFirmaInteractiva) Validar() error {
	if s.Contexto.ContextoOperacionBaremacion.ValidarPara(AccionConsultarFirmaDecisionBaremacion, ClaseRecursoSesionFirma, s.SesionRef) != nil ||
		s.Contexto.Validar() != nil || !referenciaValida(s.SesionRef, 512) || s.Documento.Validar() != nil ||
		s.Contexto.Proyeccion().SujetoRef != s.Documento.SujetoRef || !huellaSHA256Valida(s.HuellaContenidoSHA256) ||
		s.HuellaContenidoSHA256 != s.Documento.HuellaContenidoSHA256 || !referenciaValida(s.PoliticaFirmaRef, 512) ||
		s.PoliticaFirmaVersion < 1 || !huellaSHA256Valida(s.HuellaPoliticaSHA256) ||
		!referenciaValida(s.FirmanteRef, 512) || !claveValida(s.PerfilFirmanteClave) ||
		s.FirmanteRef != s.Contexto.Proyeccion().PrincipalRef || s.PerfilFirmanteClave != s.Contexto.Proyeccion().PerfilActorClave {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (c ConsultaFirmaInteractiva) ValidarPara(s SolicitudConsultarFirmaInteractiva) error {
	if s.Validar() != nil || c.Validar() != nil || c.SesionRef != s.SesionRef {
		return ErrFirmaInteractivaNoDisponible
	}
	if c.Estado != EstadoSesionFirmaCompletada {
		return nil
	}
	a := c.Artefacto
	if a == nil || a.ProcesoRef != s.Documento.ProcesoRef || a.SolicitudRef != s.Documento.SolicitudRef ||
		a.SujetoRef != s.Documento.SujetoRef || a.BaremacionMeritoRef != s.Documento.BaremacionMeritoRef ||
		a.DecisionRef != s.Documento.DecisionRef || a.VersionBaremacion != s.Documento.VersionBaremacion ||
		a.DocumentoFirmable != s.Documento.Objeto.Objeto ||
		a.HuellaDocumentoFirmableSHA256 != s.Documento.HuellaDocumentoSHA256 ||
		a.EvidenciaCustodiaRef != s.Documento.EvidenciaCustodia.Referencia ||
		a.HuellaContenidoSHA256 != s.HuellaContenidoSHA256 || a.PoliticaFirmaRef != s.PoliticaFirmaRef ||
		a.PoliticaFirmaVersion != s.PoliticaFirmaVersion || a.HuellaPoliticaFirmaSHA256 != s.HuellaPoliticaSHA256 ||
		a.FirmanteRef != s.FirmanteRef || a.PerfilFirmanteClave != s.PerfilFirmanteClave {
		return ErrFirmaInteractivaNoCompletada
	}
	return nil
}

type ConsultaFirmaInteractiva struct {
	SesionRef             string
	Estado                EstadoSesionFirmaInteractiva
	Artefacto             *ArtefactoFirma
	EvidenciaConsultaRef  string
	HuellaEvidenciaSHA256 string
	ConsultadaEn          time.Time
}

func (c ConsultaFirmaInteractiva) Validar() error {
	if !referenciaValida(c.SesionRef, 512) || !c.Estado.Valido() || !referenciaValida(c.EvidenciaConsultaRef, 512) ||
		!huellaSHA256Valida(c.HuellaEvidenciaSHA256) || c.ConsultadaEn.IsZero() {
		return ErrFirmaInteractivaNoDisponible
	}
	if c.Estado == EstadoSesionFirmaCompletada {
		if c.Artefacto == nil || c.Artefacto.Validar() != nil || c.Artefacto.SesionFirmaRef != c.SesionRef {
			return ErrFirmaInteractivaNoCompletada
		}
	} else if c.Artefacto != nil {
		return ErrFirmaInteractivaNoCompletada
	}
	return nil
}

func (c ConsultaFirmaInteractiva) Clonar() (ConsultaFirmaInteractiva, error) {
	clon := c
	if c.Artefacto != nil {
		artefacto := *c.Artefacto
		clon.Artefacto = &artefacto
	}
	return clon, clon.Validar()
}

type FirmadorInteractivo interface {
	PrepararFirmaInteractiva(context.Context, SolicitudPrepararFirmaInteractiva) (SesionFirmaInteractiva, error)
	ConsultarFirmaInteractiva(context.Context, SolicitudConsultarFirmaInteractiva) (ConsultaFirmaInteractiva, error)
}

type SolicitudRecuperarBinarioFirmado struct {
	Contexto              ContextoOperacionFirma
	DocumentoFirmadoRef   string
	HuellaDocumentoSHA256 string
	LimiteBytes           int64
}

func (s SolicitudRecuperarBinarioFirmado) Validar() error {
	if s.Contexto.ContextoOperacionBaremacion.ValidarPara(
		AccionRecuperarBinarioFirmadoBaremacion, ClaseRecursoDocumentoFirmado, s.DocumentoFirmadoRef,
	) != nil || s.Contexto.Validar() != nil || !referenciaValida(s.DocumentoFirmadoRef, 512) ||
		!huellaSHA256Valida(s.HuellaDocumentoSHA256) || s.LimiteBytes < 1 || s.LimiteBytes > maximoCargaProtegida {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

// BinarioFirmadoRecuperado transporta el PDF final como flujo. El consumidor
// debe comprobar cantidad y SHA-256 al agotarlo y cerrarlo siempre.
type BinarioFirmadoRecuperado struct {
	DocumentoFirmadoRef      string
	HuellaDocumentoSHA256    string
	MIME                     string
	Tamano                   int64
	Contenido                io.ReadCloser
	EvidenciaRecuperacionRef string
	HuellaEvidenciaSHA256    string
	RecuperadoEn             time.Time
}

func (b BinarioFirmadoRecuperado) ValidarPara(s SolicitudRecuperarBinarioFirmado) error {
	if s.Validar() != nil || b.DocumentoFirmadoRef != s.DocumentoFirmadoRef ||
		b.HuellaDocumentoSHA256 != s.HuellaDocumentoSHA256 || b.MIME != "application/pdf" ||
		b.Tamano < 1 || b.Tamano > s.LimiteBytes || lectorCierreNulo(b.Contenido) ||
		!referenciaValida(b.EvidenciaRecuperacionRef, 512) ||
		!huellaSHA256Valida(b.HuellaEvidenciaSHA256) || b.RecuperadoEn.IsZero() {
		return ErrEvidenciaFirmaNoEncontrada
	}
	return nil
}

func lectorCierreNulo(lector io.ReadCloser) bool {
	if lector == nil {
		return true
	}
	valor := reflect.ValueOf(lector)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

type RecuperadorBinarioFirmado interface {
	RecuperarBinarioFirmado(context.Context, SolicitudRecuperarBinarioFirmado) (BinarioFirmadoRecuperado, error)
}

// DocumentoFirmadoCustodiado acredita la copia institucional del PDF final y
// la retencion aplicada sobre exactamente la misma version y huella.
type DocumentoFirmadoCustodiado struct {
	DocumentoFirmadoRef               string
	FirmaRef                          string
	HuellaDocumentoSHA256             string
	Objeto                            puertosvec.ObjetoAlmacenado
	EvidenciaEscritura                puertosvec.EvidenciaOperacionAlmacen
	EvidenciaRetencion                puertosvec.EvidenciaOperacionAlmacen
	EvidenciaRecuperacionRef          string
	HuellaEvidenciaRecuperacionSHA256 string
	PoliticaRetencionRef              string
	RetenidoHasta                     time.Time
}

func (d DocumentoFirmadoCustodiado) ValidarPara(
	artefacto ArtefactoFirma,
	escritura puertosvec.ResultadoOperacionObjeto,
	retencion puertosvec.ResultadoOperacionObjeto,
) error {
	if artefacto.Validar() != nil || escritura.Validar() != nil || retencion.Validar() != nil ||
		!referenciaValida(d.DocumentoFirmadoRef, 512) || !referenciaValida(d.FirmaRef, 512) ||
		!huellaSHA256Valida(d.HuellaDocumentoSHA256) ||
		!referenciaValida(d.EvidenciaRecuperacionRef, 512) ||
		!huellaSHA256Valida(d.HuellaEvidenciaRecuperacionSHA256) ||
		!referenciaValida(d.PoliticaRetencionRef, 512) || d.RetenidoHasta.IsZero() ||
		d.DocumentoFirmadoRef != artefacto.DocumentoFirmadoRef || d.FirmaRef != artefacto.FirmaRef ||
		d.HuellaDocumentoSHA256 != artefacto.HuellaDocumentoSHA256 ||
		d.Objeto != retencion.Objeto || d.EvidenciaEscritura != escritura.Evidencia ||
		d.EvidenciaRetencion != retencion.Evidencia || escritura.Objeto.Objeto != retencion.Objeto.Objeto ||
		escritura.Objeto.HuellaSHA256 != artefacto.HuellaDocumentoSHA256 ||
		retencion.Objeto.HuellaSHA256 != artefacto.HuellaDocumentoSHA256 ||
		!retencion.Objeto.RetenidoHasta.Equal(d.RetenidoHasta) ||
		retencion.Evidencia.Accion != puertosvec.AccionAlmacenAplicarRetencion ||
		retencion.Evidencia.FundamentoRef != d.PoliticaRetencionRef {
		return ErrCustodiaDocumentoFirmableInvalida
	}
	return nil
}

type EstadoValidacionFirma string

const (
	EstadoValidacionFirmaValida        EstadoValidacionFirma = "valida"
	EstadoValidacionFirmaInvalida      EstadoValidacionFirma = "invalida"
	EstadoValidacionFirmaIndeterminada EstadoValidacionFirma = "indeterminada"
)

func (e EstadoValidacionFirma) Valido() bool {
	return e == EstadoValidacionFirmaValida || e == EstadoValidacionFirmaInvalida || e == EstadoValidacionFirmaIndeterminada
}

type EstadoComprobacionFirma string

const (
	EstadoComprobacionSuperada      EstadoComprobacionFirma = "superada"
	EstadoComprobacionNoSuperada    EstadoComprobacionFirma = "no_superada"
	EstadoComprobacionIndeterminada EstadoComprobacionFirma = "indeterminada"
)

func (e EstadoComprobacionFirma) Valido() bool {
	return e == EstadoComprobacionSuperada || e == EstadoComprobacionNoSuperada || e == EstadoComprobacionIndeterminada
}

type ComprobacionFirma struct {
	Clave                 string
	Estado                EstadoComprobacionFirma
	EvidenciaRef          string
	HuellaEvidenciaSHA256 string
}

func (c ComprobacionFirma) Validar() error {
	if !claveValida(c.Clave) || !c.Estado.Valido() || !referenciaValida(c.EvidenciaRef, 512) ||
		!huellaSHA256Valida(c.HuellaEvidenciaSHA256) {
		return ErrValidacionFirmaNoConcluyente
	}
	return nil
}

type SolicitudAumentarFirma struct {
	Contexto          ContextoOperacionFirma
	ClaveIdempotencia string
	Artefacto         ArtefactoFirma
	Validacion        ValidacionFirmaServidor
	SelloTiempo       *SelloTiempoFirma
	Politica          PoliticaFirmaBaremacion
	SolicitadaEn      time.Time
}

func (s SolicitudAumentarFirma) Validar() error {
	if s.Contexto.ContextoOperacionBaremacion.ValidarPara(AccionAumentarFirmaDecisionBaremacion, ClaseRecursoArtefactoFirma, s.Artefacto.FirmaRef) != nil ||
		s.Contexto.Validar() != nil || !referenciaValida(s.ClaveIdempotencia, 512) || s.Artefacto.Validar() != nil ||
		s.Validacion.Validar() != nil || !s.Validacion.AptaParaPerfil(s.Politica, PerfilFirmaPAdESBaselineT) ||
		s.Validacion.Artefacto != s.Artefacto ||
		s.Politica.Validar() != nil || !s.Politica.RequiereAumentoLongevidad || s.SolicitadaEn.IsZero() ||
		s.SolicitadaEn.Before(s.Validacion.ValidadaEn) {
		return ErrSolicitudBaremacionInvalida
	}
	if s.Politica.RequiereSelloTiempo {
		if s.SelloTiempo == nil || s.SelloTiempo.Validar() != nil ||
			s.SelloTiempo.ArtefactoSellado != s.Artefacto ||
			s.Validacion.SelloTiempoVerificadoRef != s.SelloTiempo.SelloTiempoRef ||
			s.Validacion.HuellaSelloTiempoVerificadaSHA256 != s.SelloTiempo.HuellaSelloTiempoSHA256 ||
			s.SelloTiempo.PoliticaSelloTiempoRef != s.Politica.PoliticaSelloTiempoRef ||
			s.SelloTiempo.PoliticaSelloTiempoVersion != s.Politica.PoliticaSelloTiempoVersion ||
			s.SelloTiempo.HuellaPoliticaSelloTiempoSHA256 != s.Politica.HuellaPoliticaSelloTiempoSHA256 {
			return ErrSolicitudBaremacionInvalida
		}
	} else if s.SelloTiempo != nil {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (s SolicitudAumentarFirma) Clonar() (SolicitudAumentarFirma, error) {
	validacion, err := s.Validacion.Clonar()
	if err != nil {
		return SolicitudAumentarFirma{}, err
	}
	clon := s
	clon.Validacion = validacion
	if s.SelloTiempo != nil {
		sello := *s.SelloTiempo
		clon.SelloTiempo = &sello
	}
	return clon, clon.Validar()
}

type ResultadoAumentoFirma struct {
	ArtefactoOrigen                ArtefactoFirma
	Artefacto                      ArtefactoFirma
	NivelAlcanzadoClave            string
	PoliticaLongevidadRef          string
	PoliticaLongevidadVersion      int
	HuellaPoliticaLongevidadSHA256 string
	EvidenciaAumentoRef            string
	HuellaEvidenciaSHA256          string
	AumentadaEn                    time.Time
}

func (r ResultadoAumentoFirma) Validar() error {
	if r.Artefacto.ValidarRevisionPAdESDe(r.ArtefactoOrigen) != nil || !claveValida(r.NivelAlcanzadoClave) ||
		!referenciaValida(r.PoliticaLongevidadRef, 512) || r.PoliticaLongevidadVersion < 1 ||
		!huellaSHA256Valida(r.HuellaPoliticaLongevidadSHA256) || !referenciaValida(r.EvidenciaAumentoRef, 512) ||
		!huellaSHA256Valida(r.HuellaEvidenciaSHA256) || r.AumentadaEn.IsZero() || r.AumentadaEn.Before(r.Artefacto.FirmadaEn) {
		return ErrAumentoFirmaNoDisponible
	}
	return nil
}

func (r ResultadoAumentoFirma) ValidarPara(s SolicitudAumentarFirma) error {
	if s.Validar() != nil || r.Validar() != nil || r.ArtefactoOrigen != s.Artefacto ||
		r.Artefacto.HuellaContenidoSHA256 != s.Artefacto.HuellaContenidoSHA256 ||
		r.Artefacto.FirmanteRef != s.Artefacto.FirmanteRef || r.Artefacto.PerfilFirmanteClave != s.Artefacto.PerfilFirmanteClave ||
		r.NivelAlcanzadoClave != s.Politica.NivelAumentoClave ||
		r.PoliticaLongevidadRef != s.Politica.PoliticaLongevidadRef ||
		r.PoliticaLongevidadVersion != s.Politica.PoliticaLongevidadVersion ||
		r.HuellaPoliticaLongevidadSHA256 != s.Politica.HuellaPoliticaLongevidadSHA256 ||
		r.AumentadaEn.Before(s.SolicitadaEn) {
		return ErrAumentoFirmaNoDisponible
	}
	return nil
}

type AumentadorFirmaLongeva interface {
	AumentarFirma(context.Context, SolicitudAumentarFirma) (ResultadoAumentoFirma, error)
}

type SolicitudRecuperarArtefactoFirma struct {
	Contexto              ContextoOperacionFirma
	FirmaRef              string
	HuellaFirmaSHA256     string
	DocumentoFirmadoRef   string
	HuellaDocumentoSHA256 string
}
type SolicitudRecuperarValidacionFirma struct {
	Contexto               ContextoOperacionFirma
	ValidacionRef          string
	HuellaValidacionSHA256 string
}
type SolicitudRecuperarSelloTiempo struct {
	Contexto          ContextoOperacionFirma
	SelloTiempoRef    string
	HuellaSelloSHA256 string
}
type SolicitudRecuperarAumentoFirma struct {
	Contexto            ContextoOperacionFirma
	EvidenciaAumentoRef string
	HuellaAumentoSHA256 string
}

func (s SolicitudRecuperarArtefactoFirma) Validar() error {
	if s.Contexto.ContextoOperacionBaremacion.ValidarPara(AccionRecuperarArtefactoFirmaBaremacion, ClaseRecursoArtefactoFirma, s.FirmaRef) != nil ||
		s.Contexto.Validar() != nil || !referenciaValida(s.FirmaRef, 512) || !huellaSHA256Valida(s.HuellaFirmaSHA256) ||
		!referenciaValida(s.DocumentoFirmadoRef, 512) || !huellaSHA256Valida(s.HuellaDocumentoSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}
func (s SolicitudRecuperarValidacionFirma) Validar() error {
	if s.Contexto.ContextoOperacionBaremacion.ValidarPara(AccionRecuperarValidacionFirmaBaremacion, ClaseRecursoValidacionFirma, s.ValidacionRef) != nil ||
		s.Contexto.Validar() != nil || !referenciaValida(s.ValidacionRef, 512) || !huellaSHA256Valida(s.HuellaValidacionSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}
func (s SolicitudRecuperarSelloTiempo) Validar() error {
	if s.Contexto.ContextoOperacionBaremacion.ValidarPara(AccionRecuperarSelloTiempoFirmaBaremacion, ClaseRecursoSelloTiempo, s.SelloTiempoRef) != nil ||
		s.Contexto.Validar() != nil || !referenciaValida(s.SelloTiempoRef, 512) || !huellaSHA256Valida(s.HuellaSelloSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}
func (s SolicitudRecuperarAumentoFirma) Validar() error {
	if s.Contexto.ContextoOperacionBaremacion.ValidarPara(AccionRecuperarAumentoFirmaBaremacion, ClaseRecursoAumentoFirma, s.EvidenciaAumentoRef) != nil ||
		s.Contexto.Validar() != nil || !referenciaValida(s.EvidenciaAumentoRef, 512) || !huellaSHA256Valida(s.HuellaAumentoSHA256) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (a ArtefactoFirma) ValidarRecuperacion(s SolicitudRecuperarArtefactoFirma) error {
	if s.Validar() != nil || a.Validar() != nil || a.FirmaRef != s.FirmaRef || a.HuellaFirmaSHA256 != s.HuellaFirmaSHA256 ||
		a.DocumentoFirmadoRef != s.DocumentoFirmadoRef || a.HuellaDocumentoSHA256 != s.HuellaDocumentoSHA256 {
		return ErrEvidenciaFirmaNoEncontrada
	}
	return nil
}

func (v ValidacionFirmaServidor) ValidarRecuperacion(s SolicitudRecuperarValidacionFirma) error {
	if s.Validar() != nil || v.Validar() != nil || v.ValidacionRef != s.ValidacionRef ||
		v.HuellaValidacionSHA256 != s.HuellaValidacionSHA256 {
		return ErrEvidenciaFirmaNoEncontrada
	}
	return nil
}

func (sello SelloTiempoFirma) ValidarRecuperacion(s SolicitudRecuperarSelloTiempo) error {
	if s.Validar() != nil || sello.Validar() != nil || sello.SelloTiempoRef != s.SelloTiempoRef ||
		sello.HuellaSelloTiempoSHA256 != s.HuellaSelloSHA256 {
		return ErrEvidenciaFirmaNoEncontrada
	}
	return nil
}

func (r ResultadoAumentoFirma) ValidarRecuperacion(s SolicitudRecuperarAumentoFirma) error {
	if s.Validar() != nil || r.Validar() != nil || r.EvidenciaAumentoRef != s.EvidenciaAumentoRef ||
		r.HuellaEvidenciaSHA256 != s.HuellaAumentoSHA256 {
		return ErrEvidenciaFirmaNoEncontrada
	}
	return nil
}

// ArchivoEvidenciasFirmaBaremacion permite revalidar historicamente cada capa
// por referencia y huella exactas, sin depender del estado actual del proveedor.
type ArchivoEvidenciasFirmaBaremacion interface {
	RecuperarArtefactoFirma(context.Context, SolicitudRecuperarArtefactoFirma) (ArtefactoFirma, error)
	RecuperarValidacionFirma(context.Context, SolicitudRecuperarValidacionFirma) (ValidacionFirmaServidor, error)
	RecuperarSelloTiempo(context.Context, SolicitudRecuperarSelloTiempo) (SelloTiempoFirma, error)
	RecuperarAumentoFirma(context.Context, SolicitudRecuperarAumentoFirma) (ResultadoAumentoFirma, error)
}

// ConstituirFirmaDecisionConfiable es el unico ensamblador recomendado. Exige
// firma interactiva, validacion servidor concluyente y las capas marcadas por
// la politica exacta. Si hay aumento, exige validacion posterior de sus bytes.
func ConstituirFirmaDecisionConfiable(
	contenido dominiobolsa.ContenidoDecisionTecnica,
	politica PoliticaFirmaBaremacion,
	artefacto ArtefactoFirma,
	validacionInicial ValidacionFirmaServidor,
	sello *SelloTiempoFirma,
	validacionTrasSello *ValidacionFirmaServidor,
	aumento *ResultadoAumentoFirma,
	validacionFinal ValidacionFirmaServidor,
	documentoCustodiado DocumentoFirmadoCustodiado,
	manifiesto ManifiestoProbatorioBaremacion,
) (dominiobolsa.FirmaDecisionTecnica, error) {
	huellaContenido, err := contenido.HuellaContenidoSHA256()
	if err != nil || politica.Validar() != nil || artefacto.Validar() != nil || !politica.VigenteEn(artefacto.FirmadaEn.UTC()) ||
		manifiesto.Validar() != nil || manifiesto.ProcesoRef != contenido.ProcesoRef ||
		manifiesto.SolicitudRef != contenido.SolicitudRef || manifiesto.SujetoRef != contenido.SujetoRef ||
		manifiesto.BaremacionMeritoRef != contenido.BaremacionMeritoRef || manifiesto.DecisionRef != contenido.ID ||
		manifiesto.VersionBase+1 != contenido.VersionBaremacion ||
		!validacionInicial.AptaParaPerfil(politica, PerfilFirmaPAdESBaselineB) ||
		validacionInicial.Artefacto != artefacto || artefacto.HuellaContenidoSHA256 != huellaContenido ||
		artefacto.ProcesoRef != contenido.ProcesoRef || artefacto.SolicitudRef != contenido.SolicitudRef ||
		artefacto.SujetoRef != contenido.SujetoRef || artefacto.BaremacionMeritoRef != contenido.BaremacionMeritoRef ||
		artefacto.DecisionRef != contenido.ID || artefacto.VersionBaremacion != contenido.VersionBaremacion ||
		artefacto.PoliticaFirmaRef != politica.Referencia || artefacto.PoliticaFirmaVersion != politica.Version ||
		artefacto.HuellaPoliticaFirmaSHA256 != politica.HuellaSHA256 || artefacto.FirmanteRef != contenido.DecisorRef ||
		artefacto.PerfilFirmanteClave != contenido.PerfilDecisorClave {
		return dominiobolsa.FirmaDecisionTecnica{}, ErrFirmaServidorNoValida
	}
	if politica.RequiereSelloTiempo {
		if sello == nil || sello.Validar() != nil || sello.ArtefactoOrigen != artefacto ||
			sello.PoliticaSelloTiempoRef != politica.PoliticaSelloTiempoRef ||
			sello.PoliticaSelloTiempoVersion != politica.PoliticaSelloTiempoVersion ||
			sello.HuellaPoliticaSelloTiempoSHA256 != politica.HuellaPoliticaSelloTiempoSHA256 ||
			sello.SelladoEn.Before(validacionInicial.ValidadaEn) || validacionTrasSello == nil ||
			!validacionTrasSello.AptaParaPerfil(politica, PerfilFirmaPAdESBaselineT) ||
			validacionTrasSello.Artefacto != sello.ArtefactoSellado ||
			validacionTrasSello.SelloTiempoVerificadoRef != sello.SelloTiempoRef ||
			validacionTrasSello.HuellaSelloTiempoVerificadaSHA256 != sello.HuellaSelloTiempoSHA256 ||
			validacionTrasSello.ValidadaEn.Before(sello.SelladoEn) {
			return dominiobolsa.FirmaDecisionTecnica{}, ErrSelloTiempoNoDisponible
		}
	} else if sello != nil || validacionTrasSello != nil {
		return dominiobolsa.FirmaDecisionTecnica{}, ErrPoliticaFirmaInsegura
	}
	artefactoFinal := artefacto
	if sello != nil {
		artefactoFinal = sello.ArtefactoSellado
	}
	if politica.RequiereAumentoLongevidad {
		if aumento == nil || aumento.Validar() != nil || aumento.NivelAlcanzadoClave != politica.NivelAumentoClave ||
			aumento.ArtefactoOrigen != artefactoFinal ||
			aumento.PoliticaLongevidadRef != politica.PoliticaLongevidadRef ||
			aumento.PoliticaLongevidadVersion != politica.PoliticaLongevidadVersion ||
			aumento.HuellaPoliticaLongevidadSHA256 != politica.HuellaPoliticaLongevidadSHA256 {
			return dominiobolsa.FirmaDecisionTecnica{}, ErrAumentoFirmaNoDisponible
		}
		artefactoFinal = aumento.Artefacto
	} else if aumento != nil {
		return dominiobolsa.FirmaDecisionTecnica{}, ErrPoliticaFirmaInsegura
	}
	if !validacionFinal.AptaParaPolitica(politica) || validacionFinal.Artefacto != artefactoFinal ||
		validacionFinal.ValidadaEn.Before(validacionInicial.ValidadaEn) ||
		(aumento != nil && (aumento.AumentadaEn.Before(validacionTrasSello.ValidadaEn) ||
			validacionFinal.ValidadaEn.Before(aumento.AumentadaEn))) {
		return dominiobolsa.FirmaDecisionTecnica{}, ErrFirmaServidorNoValida
	}
	if aumento != nil && (validacionFinal.SelloTiempoVerificadoRef != sello.SelloTiempoRef ||
		validacionFinal.HuellaSelloTiempoVerificadaSHA256 != sello.HuellaSelloTiempoSHA256 ||
		validacionFinal.AumentoLongevidadVerificadoRef != aumento.EvidenciaAumentoRef ||
		validacionFinal.HuellaAumentoLongevidadVerificadaSHA256 != aumento.HuellaEvidenciaSHA256) {
		return dominiobolsa.FirmaDecisionTecnica{}, ErrFirmaServidorNoValida
	}
	if sello == nil && !mismaEvidenciaValidacionFirma(validacionInicial, validacionFinal) ||
		sello != nil && aumento == nil && !mismaEvidenciaValidacionFirma(*validacionTrasSello, validacionFinal) {
		return dominiobolsa.FirmaDecisionTecnica{}, ErrFirmaServidorNoValida
	}
	if documentoCustodiado.DocumentoFirmadoRef != artefactoFinal.DocumentoFirmadoRef ||
		documentoCustodiado.FirmaRef != artefactoFinal.FirmaRef ||
		documentoCustodiado.HuellaDocumentoSHA256 != artefactoFinal.HuellaDocumentoSHA256 ||
		documentoCustodiado.Objeto.HuellaSHA256 != artefactoFinal.HuellaDocumentoSHA256 ||
		documentoCustodiado.Objeto.RetenidoHasta.IsZero() ||
		!documentoCustodiado.Objeto.RetenidoHasta.Equal(documentoCustodiado.RetenidoHasta) {
		return dominiobolsa.FirmaDecisionTecnica{}, ErrCustodiaDocumentoFirmableInvalida
	}
	if err := manifiesto.validarCoberturaArtefactosFirmaPara(
		politica, artefacto, validacionInicial, sello, validacionTrasSello, aumento, validacionFinal, documentoCustodiado,
	); err != nil {
		return dominiobolsa.FirmaDecisionTecnica{}, ErrFirmaServidorNoValida
	}
	firma := dominiobolsa.FirmaDecisionTecnica{
		FirmanteRef: artefacto.FirmanteRef, PerfilFirmanteClave: artefacto.PerfilFirmanteClave,
		PoliticaFirmaRef: politica.Referencia, PoliticaFirmaVersion: politica.Version,
		HuellaPoliticaFirmaSHA256: politica.HuellaSHA256, RequiereFirmaInteractiva: true,
		PerfilFirmaAlcanzadoClave:  politica.PerfilFirmaClave,
		RequiereValidacionServidor: true, RequiereSelloTiempo: politica.RequiereSelloTiempo,
		RequiereAumentoLongevidad:             politica.RequiereAumentoLongevidad,
		SesionFirmaInteractivaRef:             artefacto.SesionFirmaRef,
		HuellaEvidenciaFirmaInteractivaSHA256: artefacto.HuellaEvidenciaInteractivaSHA256,
		DocumentoFirmableRef:                  artefacto.DocumentoFirmable.Referencia,
		VersionDocumentoFirmable:              artefacto.DocumentoFirmable.Version,
		HuellaDocumentoFirmableSHA256:         artefacto.HuellaDocumentoFirmableSHA256,
		EvidenciaCustodiaRef:                  artefacto.EvidenciaCustodiaRef,
		FirmaRef:                              artefacto.FirmaRef, HuellaFirmaSHA256: artefacto.HuellaFirmaSHA256,
		DocumentoFirmadoRef:                   artefactoFinal.DocumentoFirmadoRef,
		HuellaDocumentoSHA256:                 artefactoFinal.HuellaDocumentoSHA256,
		DocumentoFirmadoCustodiadoRef:         documentoCustodiado.Objeto.Objeto.Referencia,
		VersionDocumentoFirmadoCustodiado:     documentoCustodiado.Objeto.Objeto.Version,
		EvidenciaRecuperacionFirmadoRef:       documentoCustodiado.EvidenciaRecuperacionRef,
		HuellaEvidenciaRecuperacionSHA256:     documentoCustodiado.HuellaEvidenciaRecuperacionSHA256,
		EvidenciaCustodiaDocumentoFirmadoRef:  documentoCustodiado.EvidenciaEscritura.Referencia,
		EvidenciaRetencionDocumentoFirmadoRef: documentoCustodiado.EvidenciaRetencion.Referencia,
		PoliticaRetencionDocumentoFirmadoRef:  documentoCustodiado.PoliticaRetencionRef,
		DocumentoFirmadoRetenidoHasta:         documentoCustodiado.RetenidoHasta,
		ManifiestoProbatorioRef:               manifiesto.Referencia,
		HuellaManifiestoProbatorioSHA256:      manifiesto.HuellaManifiestoSHA256,
		SelloManifiestoProbatorioHMACSHA256:   manifiesto.SelloManifiestoHMACSHA256,
		HuellaContenidoSHA256:                 huellaContenido,
		ValidacionInicialFirmaRef:             validacionInicial.ValidacionRef,
		HuellaValidacionInicialSHA256:         validacionInicial.HuellaValidacionSHA256,
		ValidadaInicialEn:                     validacionInicial.ValidadaEn,
		ValidacionFirmaRef:                    validacionFinal.ValidacionRef,
		HuellaValidacionSHA256:                validacionFinal.HuellaValidacionSHA256,
		ValidadaEn:                            validacionFinal.ValidadaEn, FirmadaEn: artefacto.FirmadaEn,
	}
	if sello != nil {
		vinculo, err := NuevoVinculoRevisionSelladaPAdES(*sello, *validacionTrasSello)
		if err != nil {
			return dominiobolsa.FirmaDecisionTecnica{}, ErrFirmaServidorNoValida
		}
		firma.SelloTiempoRef = sello.SelloTiempoRef
		firma.HuellaSelloTiempoSHA256 = sello.HuellaSelloTiempoSHA256
		firma.VinculoRevisionSelladaRef = vinculo.Referencia
		firma.HuellaVinculoRevisionSelladaSHA256 = vinculo.HuellaSHA256
		firma.PoliticaSelloTiempoRef = sello.PoliticaSelloTiempoRef
		firma.PoliticaSelloTiempoVersion = sello.PoliticaSelloTiempoVersion
		firma.HuellaPoliticaSelloTiempoSHA256 = sello.HuellaPoliticaSelloTiempoSHA256
		firma.ValidacionSelloTiempoRef = sello.ValidacionSelloRef
		firma.HuellaValidacionSelloTiempoSHA256 = sello.HuellaValidacionSHA256
		firma.SelladaEn = sello.SelladoEn
		firma.ValidacionDocumentoSelladoRef = validacionTrasSello.ValidacionRef
		firma.HuellaValidacionDocumentoSelladoSHA256 = validacionTrasSello.HuellaValidacionSHA256
		firma.ValidadoDocumentoSelladoEn = validacionTrasSello.ValidadaEn
	}
	if aumento != nil {
		vinculo, err := NuevoVinculoRevisionLongevaPAdES(*sello, *validacionTrasSello, *aumento, validacionFinal)
		if err != nil {
			return dominiobolsa.FirmaDecisionTecnica{}, ErrFirmaServidorNoValida
		}
		firma.NivelLongevidadClave = aumento.NivelAlcanzadoClave
		firma.AumentoLongevidadRef = aumento.EvidenciaAumentoRef
		firma.HuellaAumentoLongevidadSHA256 = aumento.HuellaEvidenciaSHA256
		firma.VinculoRevisionLongevaRef = vinculo.Referencia
		firma.HuellaVinculoRevisionLongevaSHA256 = vinculo.HuellaSHA256
		firma.PoliticaLongevidadRef = aumento.PoliticaLongevidadRef
		firma.PoliticaLongevidadVersion = aumento.PoliticaLongevidadVersion
		firma.HuellaPoliticaLongevidadSHA256 = aumento.HuellaPoliticaLongevidadSHA256
		firma.ValidacionLongevidadRef = validacionFinal.ValidacionRef
		firma.HuellaValidacionLongevidadSHA256 = validacionFinal.HuellaValidacionSHA256
		firma.AumentadaEn = aumento.AumentadaEn
	}
	versionBase := ReferenciaVersionBaremacion{
		BaremacionMeritoRef: contenido.BaremacionMeritoRef,
		Numero:              contenido.VersionAnteriorBaremacion,
		HuellaEstadoSHA256:  manifiesto.HuellaVersionBaseSHA256,
	}
	if err := manifiesto.ValidarCoberturaFirmaPara(versionBase, contenido, firma); err != nil {
		return dominiobolsa.FirmaDecisionTecnica{}, ErrFirmaServidorNoValida
	}
	return firma, nil
}

func mismaEvidenciaValidacionFirma(a, b ValidacionFirmaServidor) bool {
	return a.ValidacionRef == b.ValidacionRef &&
		a.HuellaValidacionSHA256 == b.HuellaValidacionSHA256 &&
		a.Artefacto == b.Artefacto && a.ValidadaEn.Equal(b.ValidadaEn) &&
		a.PerfilFirmaVerificadoClave == b.PerfilFirmaVerificadoClave &&
		a.SelloTiempoVerificadoRef == b.SelloTiempoVerificadoRef &&
		a.HuellaSelloTiempoVerificadaSHA256 == b.HuellaSelloTiempoVerificadaSHA256 &&
		a.AumentoLongevidadVerificadoRef == b.AumentoLongevidadVerificadoRef &&
		a.HuellaAumentoLongevidadVerificadaSHA256 == b.HuellaAumentoLongevidadVerificadaSHA256
}

type SelladorSolicitudBaremacion interface {
	SellarSolicitudBaremacion(context.Context, CargaProtegida) (string, error)
}

type GeneradorReferenciasOpacasBaremacion interface {
	NuevoIDBaremacion() (string, error)
	NuevoIDDecisionTecnica() (string, error)
	NuevaReferenciaManifiestoProbatorio() (string, error)
	NuevaReferenciaCorrelacion() (string, error)
	NuevaReferenciaEfectoAlmacen() (string, error)
}

type Reloj = puertosvec.Reloj

func tokenBase64URLValido(valor string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) < 32 || len(valor) > 128 || strings.Contains(valor, "=") {
		return false
	}
	decodificado, err := base64.RawURLEncoding.DecodeString(valor)
	return err == nil && len(decodificado) >= 24 && len(decodificado) <= 96 &&
		base64.RawURLEncoding.EncodeToString(decodificado) == valor
}

func referenciaValida(valor string, maximo int) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximo {
		return false
	}
	for _, caracter := range valor {
		if caracter < 33 || caracter > 126 {
			return false
		}
	}
	return true
}

func claveValida(valor string) bool {
	return len(valor) <= 128 && valor == strings.TrimSpace(valor) && patronClave.MatchString(valor)
}

func textoValido(valor string, maximo int) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximo {
		return false
	}
	for _, caracter := range valor {
		if caracter < 32 && caracter != '\n' && caracter != '\r' && caracter != '\t' {
			return false
		}
	}
	return true
}

func huellaSHA256Valida(valor string) bool {
	if len(valor) != 64 || valor != strings.ToLower(valor) || valor != strings.TrimSpace(valor) {
		return false
	}
	bytes, err := hex.DecodeString(valor)
	return err == nil && len(bytes) == 32
}

func huellaHMACSHA256Valida(valor string) bool {
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" && claveValida(partes[1]) && huellaSHA256Valida(partes[2])
}

func ventanaValida(inicio, fin time.Time, maxima time.Duration) bool {
	return !inicio.IsZero() && !fin.IsZero() && fin.After(inicio) && fin.Sub(inicio) <= maxima
}

func mimeValido(valor string) bool {
	return referenciaValida(valor, 255) && strings.Contains(valor, "/") && !strings.ContainsAny(valor, ";,")
}

func clonarEvidencia(e dominiobolsa.EvidenciaMerito) dominiobolsa.EvidenciaMerito {
	clon := e
	if e.SubsanacionDe != nil {
		referencia := *e.SubsanacionDe
		clon.SubsanacionDe = &referencia
	}
	return clon
}

func evidenciasIguales(izquierda, derecha dominiobolsa.EvidenciaMerito) bool {
	if izquierda.Referencia != derecha.Referencia {
		return false
	}
	if izquierda.SubsanacionDe == nil || derecha.SubsanacionDe == nil {
		return izquierda.SubsanacionDe == nil && derecha.SubsanacionDe == nil
	}
	return *izquierda.SubsanacionDe == *derecha.SubsanacionDe
}
