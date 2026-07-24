package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"time"
)

var (
	ErrSolicitudFlujoFirmaBaremacionInvalida = errors.New("bolsa: solicitud de flujo de firma invalida")
	ErrFlujoFirmaBaremacionNoEncontrado      = errors.New("bolsa: flujo de firma no encontrado")
	ErrClaveFlujoFirmaBaremacionReutilizada  = errors.New("bolsa: clave de flujo reutilizada con otros datos")
	ErrConflictoFlujoFirmaBaremacion         = errors.New("bolsa: conflicto de version del flujo de firma")
	ErrFlujoFirmaBaremacionOcupado           = errors.New("bolsa: flujo de firma ocupado")
	ErrArrendamientoFlujoFirmaInvalido       = errors.New("bolsa: arrendamiento de flujo de firma invalido")
	ErrSerializacionArrendamientoProhibida   = errors.New("bolsa: serializacion de arrendamiento de flujo de firma prohibida")
	ErrEstadoFlujoFirmaAlterado              = errors.New("bolsa: estado protegido del flujo de firma alterado")
	ErrPasoFlujoFirmaNoPermitido             = errors.New("bolsa: paso de flujo de firma no permitido")
	ErrSerializacionEstadoFlujoProhibida     = errors.New("bolsa: serializacion generica de estado de flujo prohibida")
)

const (
	EsquemaEstadoProtegidoFlujoFirmaBaremacion = "bolsa.firma.estado-protegido.v1"
	AlgoritmoProteccionEstadoAES256GCM         = "aes-256-gcm"
	DuracionMaximaArrendamientoFlujoFirma      = 5 * time.Minute
)

// EstadoProtegidoFlujoFirmaBaremacion contiene exclusivamente un sobre AEAD.
// Nunca contiene una autorizacion reconstruible en claro. Los adaptadores
// duraderos deben usar DatosPersistencia e ImportarEstadoProtegido de forma
// deliberada; los codificadores genericos fallan cerrados.
type EstadoProtegidoFlujoFirmaBaremacion struct {
	esquema      string
	algoritmo    string
	claveRef     string
	nonce        []byte
	cifrado      []byte
	huellaSHA256 string
}

type DatosPersistenciaEstadoProtegidoFlujoFirmaBaremacion struct {
	Esquema      string
	Algoritmo    string
	ClaveRef     string
	Nonce        []byte
	Cifrado      []byte
	HuellaSHA256 string
}

func NuevoEstadoProtegidoFlujoFirmaBaremacion(
	algoritmo, claveRef string,
	nonce, cifrado []byte,
) (EstadoProtegidoFlujoFirmaBaremacion, error) {
	suma := sha256.Sum256(cifrado)
	estado := EstadoProtegidoFlujoFirmaBaremacion{
		esquema:      EsquemaEstadoProtegidoFlujoFirmaBaremacion,
		algoritmo:    algoritmo,
		claveRef:     claveRef,
		nonce:        append([]byte(nil), nonce...),
		cifrado:      append([]byte(nil), cifrado...),
		huellaSHA256: hex.EncodeToString(suma[:]),
	}
	if estado.Validar() != nil {
		return EstadoProtegidoFlujoFirmaBaremacion{}, ErrEstadoFlujoFirmaAlterado
	}
	return estado, nil
}

func ImportarEstadoProtegidoFlujoFirmaBaremacion(
	datos DatosPersistenciaEstadoProtegidoFlujoFirmaBaremacion,
) (EstadoProtegidoFlujoFirmaBaremacion, error) {
	estado, err := NuevoEstadoProtegidoFlujoFirmaBaremacion(
		datos.Algoritmo, datos.ClaveRef, datos.Nonce, datos.Cifrado,
	)
	if err != nil || datos.Esquema != EsquemaEstadoProtegidoFlujoFirmaBaremacion ||
		estado.huellaSHA256 != datos.HuellaSHA256 {
		return EstadoProtegidoFlujoFirmaBaremacion{}, ErrEstadoFlujoFirmaAlterado
	}
	return estado, nil
}

func (e EstadoProtegidoFlujoFirmaBaremacion) Validar() error {
	if e.esquema != EsquemaEstadoProtegidoFlujoFirmaBaremacion ||
		e.algoritmo != AlgoritmoProteccionEstadoAES256GCM ||
		!referenciaValida(e.claveRef, 256) || len(e.nonce) != 12 ||
		len(e.cifrado) < 16 || len(e.cifrado) > maximoCargaProtegida+64 ||
		!huellaSHA256Valida(e.huellaSHA256) {
		return ErrEstadoFlujoFirmaAlterado
	}
	suma := sha256.Sum256(e.cifrado)
	if hex.EncodeToString(suma[:]) != e.huellaSHA256 {
		return ErrEstadoFlujoFirmaAlterado
	}
	return nil
}

func (e EstadoProtegidoFlujoFirmaBaremacion) DatosPersistencia() (
	DatosPersistenciaEstadoProtegidoFlujoFirmaBaremacion,
	error,
) {
	if e.Validar() != nil {
		return DatosPersistenciaEstadoProtegidoFlujoFirmaBaremacion{}, ErrEstadoFlujoFirmaAlterado
	}
	return DatosPersistenciaEstadoProtegidoFlujoFirmaBaremacion{
		Esquema: e.esquema, Algoritmo: e.algoritmo, ClaveRef: e.claveRef,
		Nonce: append([]byte(nil), e.nonce...), Cifrado: append([]byte(nil), e.cifrado...),
		HuellaSHA256: e.huellaSHA256,
	}, nil
}

func (e EstadoProtegidoFlujoFirmaBaremacion) Clonar() (EstadoProtegidoFlujoFirmaBaremacion, error) {
	datos, err := e.DatosPersistencia()
	if err != nil {
		return EstadoProtegidoFlujoFirmaBaremacion{}, err
	}
	return ImportarEstadoProtegidoFlujoFirmaBaremacion(datos)
}

func (EstadoProtegidoFlujoFirmaBaremacion) String() string {
	return "[ESTADO-FLUJO-FIRMA-PROTEGIDO]"
}
func (e EstadoProtegidoFlujoFirmaBaremacion) GoString() string { return e.String() }
func (e EstadoProtegidoFlujoFirmaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.String())
}
func (EstadoProtegidoFlujoFirmaBaremacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEstadoFlujoProhibida
}
func (EstadoProtegidoFlujoFirmaBaremacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEstadoFlujoProhibida
}
func (EstadoProtegidoFlujoFirmaBaremacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionEstadoFlujoProhibida
}

type ProtectorEstadoFlujoFirmaBaremacion interface {
	ProtegerEstadoFlujoFirmaBaremacion(context.Context, CargaProtegida) (EstadoProtegidoFlujoFirmaBaremacion, error)
	DesprotegerEstadoFlujoFirmaBaremacion(context.Context, EstadoProtegidoFlujoFirmaBaremacion) (CargaProtegida, error)
}

type EstadoExpedienteFlujoFirmaBaremacion string

const (
	EstadoExpedienteFirmaPreparando           EstadoExpedienteFlujoFirmaBaremacion = "preparando"
	EstadoExpedienteFirmaPendienteInteraccion EstadoExpedienteFlujoFirmaBaremacion = "pendiente_interaccion"
	EstadoExpedienteFirmaFinalizando          EstadoExpedienteFlujoFirmaBaremacion = "finalizando"
	EstadoExpedienteFirmaCompletado           EstadoExpedienteFlujoFirmaBaremacion = "completado"
)

func (e EstadoExpedienteFlujoFirmaBaremacion) Valido() bool {
	switch e {
	case EstadoExpedienteFirmaPreparando, EstadoExpedienteFirmaPendienteInteraccion,
		EstadoExpedienteFirmaFinalizando, EstadoExpedienteFirmaCompletado:
		return true
	default:
		return false
	}
}

type PasoFlujoFirmaBaremacion string

const (
	PasoPrepararFirmaBaremacion  PasoFlujoFirmaBaremacion = "preparar_firma"
	PasoCompletarFirmaBaremacion PasoFlujoFirmaBaremacion = "completar_firma"
	PasoCustodiarFirmaBaremacion PasoFlujoFirmaBaremacion = "custodiar_firma"
	PasoRetenerFirmaBaremacion   PasoFlujoFirmaBaremacion = "retener_firma"
	PasoReservarFirmaBaremacion  PasoFlujoFirmaBaremacion = "reservar_cambio"
	PasoConfirmarFirmaBaremacion PasoFlujoFirmaBaremacion = "confirmar_cambio"
)

var pasosFlujoFirmaBaremacion = [...]PasoFlujoFirmaBaremacion{
	PasoPrepararFirmaBaremacion,
	PasoCompletarFirmaBaremacion,
	PasoCustodiarFirmaBaremacion,
	PasoRetenerFirmaBaremacion,
	PasoReservarFirmaBaremacion,
	PasoConfirmarFirmaBaremacion,
}

func PasosFlujoFirmaBaremacion() []PasoFlujoFirmaBaremacion {
	return append([]PasoFlujoFirmaBaremacion(nil), pasosFlujoFirmaBaremacion[:]...)
}

func (p PasoFlujoFirmaBaremacion) Valido() bool {
	for _, permitido := range pasosFlujoFirmaBaremacion {
		if p == permitido {
			return true
		}
	}
	return false
}

type EstadoPuntoControlFirmaBaremacion string

const (
	EstadoPuntoControlFirmaDeclarado  EstadoPuntoControlFirmaBaremacion = "declarado"
	EstadoPuntoControlFirmaCompletado EstadoPuntoControlFirmaBaremacion = "completado"
)

type PuntoControlFirmaBaremacion struct {
	Paso                  PasoFlujoFirmaBaremacion
	Estado                EstadoPuntoControlFirmaBaremacion
	EfectoRef             string
	ClaveIdempotenciaHMAC string
	ResultadoRef          string
	HuellaResultadoSHA256 string
	DeclaradoEn           time.Time
	CompletadoEn          time.Time
}

func (p PuntoControlFirmaBaremacion) Validar() error {
	if !p.Paso.Valido() || !referenciaValida(p.EfectoRef, 512) ||
		!huellaHMACSHA256Valida(p.ClaveIdempotenciaHMAC) || p.DeclaradoEn.IsZero() {
		return ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	switch p.Estado {
	case EstadoPuntoControlFirmaDeclarado:
		if p.ResultadoRef != "" || p.HuellaResultadoSHA256 != "" || !p.CompletadoEn.IsZero() {
			return ErrSolicitudFlujoFirmaBaremacionInvalida
		}
	case EstadoPuntoControlFirmaCompletado:
		if !referenciaValida(p.ResultadoRef, 512) || !huellaSHA256Valida(p.HuellaResultadoSHA256) ||
			p.CompletadoEn.IsZero() || p.CompletadoEn.Before(p.DeclaradoEn) {
			return ErrSolicitudFlujoFirmaBaremacionInvalida
		}
	default:
		return ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	return nil
}

// ProyeccionLanzamientoFirmaBaremacion no transporta bytes, URL firmada,
// autorizacion ni estado del expediente. LanzamientoRef es una referencia
// opaca que el adaptador interno debe resolver tras reautenticar la peticion.
type ProyeccionLanzamientoFirmaBaremacion struct {
	FlujoRef              string
	SesionFirmaRef        string
	LanzamientoRef        string
	CanalLanzamientoClave string
	PreparadaEn           time.Time
	ExpiraEn              time.Time
}

func (p ProyeccionLanzamientoFirmaBaremacion) Validar() error {
	if !referenciaValida(p.FlujoRef, 512) || !referenciaValida(p.SesionFirmaRef, 512) ||
		!referenciaValida(p.LanzamientoRef, 512) || !claveValida(p.CanalLanzamientoClave) ||
		p.PreparadaEn.IsZero() || p.ExpiraEn.IsZero() || !p.ExpiraEn.After(p.PreparadaEn) ||
		p.ExpiraEn.Sub(p.PreparadaEn) > VentanaMaximaSesionFirmaInteractiva {
		return ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	return nil
}

type ResultadoFinalFlujoFirmaBaremacion struct {
	FlujoRef                     string
	DecisionRef                  string
	DocumentoFirmadoRef          string
	HuellaDocumentoFirmadoSHA256 string
	VersionBaremacion            uint64
	EvidenciaConfirmacionRef     string
	HuellaResultadoSHA256        string
	CompletadoEn                 time.Time
}

func (r ResultadoFinalFlujoFirmaBaremacion) Validar() error {
	if !referenciaValida(r.FlujoRef, 512) || !referenciaValida(r.DecisionRef, 512) ||
		!referenciaValida(r.DocumentoFirmadoRef, 512) ||
		!huellaSHA256Valida(r.HuellaDocumentoFirmadoSHA256) || r.VersionBaremacion < 2 ||
		!referenciaValida(r.EvidenciaConfirmacionRef, 512) ||
		!huellaSHA256Valida(r.HuellaResultadoSHA256) || r.CompletadoEn.IsZero() {
		return ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	return nil
}

// ExpedienteFlujoFirmaBaremacion es la saga durable. Solo persiste referencias,
// huellas, recibos y un sobre AEAD; las capacidades de autorizacion se obtienen
// de nuevo para cada efecto y nunca forman parte de esta estructura.
type ExpedienteFlujoFirmaBaremacion struct {
	FlujoRef               string
	Version                uint64
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	VinculoActorHMAC       string
	PerfilActorClave       string
	ProcesoRef             string
	SolicitudRef           string
	BaremacionMeritoRef    string
	DecisionRef            string
	Estado                 EstadoExpedienteFlujoFirmaBaremacion
	EstadoProtegido        EstadoProtegidoFlujoFirmaBaremacion
	PuntosControl          []PuntoControlFirmaBaremacion
	ProyeccionLanzamiento  *ProyeccionLanzamientoFirmaBaremacion
	Resultado              *ResultadoFinalFlujoFirmaBaremacion
	CreadoEn               time.Time
	ActualizadoEn          time.Time
	SelloEstadoHMAC        string
}

func (e ExpedienteFlujoFirmaBaremacion) Validar() error {
	if e.validarSinSello() != nil || !huellaHMACSHA256Valida(e.SelloEstadoHMAC) {
		return ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	return nil
}

func (e ExpedienteFlujoFirmaBaremacion) validarSinSello() error {
	if !referenciaValida(e.FlujoRef, 512) || e.Version < 1 ||
		!huellaHMACSHA256Valida(e.IndiceIdempotenciaHMAC) ||
		!huellaHMACSHA256Valida(e.HuellaSolicitudHMAC) ||
		!huellaHMACSHA256Valida(e.VinculoActorHMAC) || !claveValida(e.PerfilActorClave) ||
		!referenciaValida(e.ProcesoRef, 512) || !referenciaValida(e.SolicitudRef, 512) ||
		!referenciaValida(e.BaremacionMeritoRef, 512) || !referenciaValida(e.DecisionRef, 512) ||
		!e.Estado.Valido() || e.EstadoProtegido.Validar() != nil ||
		e.CreadoEn.IsZero() || e.ActualizadoEn.IsZero() || e.ActualizadoEn.Before(e.CreadoEn) ||
		len(e.PuntosControl) > len(pasosFlujoFirmaBaremacion) {
		return ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	for indice, punto := range e.PuntosControl {
		if punto.Validar() != nil || punto.Paso != pasosFlujoFirmaBaremacion[indice] ||
			punto.DeclaradoEn.Before(e.CreadoEn) || punto.DeclaradoEn.After(e.ActualizadoEn) ||
			(!punto.CompletadoEn.IsZero() && punto.CompletadoEn.After(e.ActualizadoEn)) ||
			(indice < len(e.PuntosControl)-1 && punto.Estado != EstadoPuntoControlFirmaCompletado) {
			return ErrSolicitudFlujoFirmaBaremacionInvalida
		}
	}
	preparacionCompletada := len(e.PuntosControl) > 0 &&
		e.PuntosControl[0].Estado == EstadoPuntoControlFirmaCompletado
	if preparacionCompletada {
		if e.ProyeccionLanzamiento == nil || e.ProyeccionLanzamiento.Validar() != nil ||
			e.ProyeccionLanzamiento.FlujoRef != e.FlujoRef ||
			!e.ProyeccionLanzamiento.PreparadaEn.Equal(e.PuntosControl[0].CompletadoEn) {
			return ErrSolicitudFlujoFirmaBaremacionInvalida
		}
	} else if e.ProyeccionLanzamiento != nil {
		return ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	completado := len(e.PuntosControl) == len(pasosFlujoFirmaBaremacion) &&
		e.PuntosControl[len(e.PuntosControl)-1].Estado == EstadoPuntoControlFirmaCompletado
	switch {
	case completado:
		ultimo := e.PuntosControl[len(e.PuntosControl)-1]
		if e.Estado != EstadoExpedienteFirmaCompletado || e.Resultado == nil ||
			e.Resultado.Validar() != nil || e.Resultado.FlujoRef != e.FlujoRef ||
			e.Resultado.DecisionRef != e.DecisionRef ||
			!e.Resultado.CompletadoEn.Equal(ultimo.CompletadoEn) {
			return ErrSolicitudFlujoFirmaBaremacionInvalida
		}
	case len(e.PuntosControl) == 0 || !preparacionCompletada:
		if e.Estado != EstadoExpedienteFirmaPreparando || e.Resultado != nil {
			return ErrSolicitudFlujoFirmaBaremacionInvalida
		}
	case len(e.PuntosControl) == 1:
		if e.Estado != EstadoExpedienteFirmaPendienteInteraccion || e.Resultado != nil {
			return ErrSolicitudFlujoFirmaBaremacionInvalida
		}
	default:
		if e.Estado != EstadoExpedienteFirmaFinalizando || e.Resultado != nil {
			return ErrSolicitudFlujoFirmaBaremacionInvalida
		}
	}
	return nil
}

func (e ExpedienteFlujoFirmaBaremacion) Clonar() (ExpedienteFlujoFirmaBaremacion, error) {
	clon := e
	estado, err := e.EstadoProtegido.Clonar()
	if err != nil {
		return ExpedienteFlujoFirmaBaremacion{}, err
	}
	clon.EstadoProtegido = estado
	clon.PuntosControl = append([]PuntoControlFirmaBaremacion(nil), e.PuntosControl...)
	if e.ProyeccionLanzamiento != nil {
		proyeccion := *e.ProyeccionLanzamiento
		clon.ProyeccionLanzamiento = &proyeccion
	}
	if e.Resultado != nil {
		resultado := *e.Resultado
		clon.Resultado = &resultado
	}
	if clon.Validar() != nil {
		return ExpedienteFlujoFirmaBaremacion{}, ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	return clon, nil
}

func (e ExpedienteFlujoFirmaBaremacion) PrepararSellado() (
	ExpedienteFlujoFirmaBaremacion,
	CargaProtegida,
	error,
) {
	preparado := e
	preparado.SelloEstadoHMAC = ""
	if preparado.validarSinSello() != nil {
		return ExpedienteFlujoFirmaBaremacion{}, CargaProtegida{}, ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	carga, err := representacionCanonicaExpedienteFlujoFirma(preparado)
	if err != nil {
		return ExpedienteFlujoFirmaBaremacion{}, CargaProtegida{}, err
	}
	return preparado, carga, nil
}

func (e ExpedienteFlujoFirmaBaremacion) IncorporarSello(sello string) (
	ExpedienteFlujoFirmaBaremacion,
	error,
) {
	if e.SelloEstadoHMAC != "" || !huellaHMACSHA256Valida(sello) || e.validarSinSello() != nil {
		return ExpedienteFlujoFirmaBaremacion{}, ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	e.SelloEstadoHMAC = sello
	if e.Validar() != nil {
		return ExpedienteFlujoFirmaBaremacion{}, ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	return e, nil
}

func RepresentacionCanonicaExpedienteFlujoFirmaBaremacion(
	e ExpedienteFlujoFirmaBaremacion,
) (CargaProtegida, error) {
	preparado, carga, err := e.PrepararSellado()
	_ = preparado
	return carga, err
}

func representacionCanonicaExpedienteFlujoFirma(e ExpedienteFlujoFirmaBaremacion) (CargaProtegida, error) {
	datosEstado, err := e.EstadoProtegido.DatosPersistencia()
	if err != nil {
		return CargaProtegida{}, err
	}
	partes := []string{
		"expediente_flujo_firma_baremacion_v1", e.FlujoRef, strconv.FormatUint(e.Version, 10),
		e.IndiceIdempotenciaHMAC, e.HuellaSolicitudHMAC, e.VinculoActorHMAC, e.PerfilActorClave,
		e.ProcesoRef, e.SolicitudRef, e.BaremacionMeritoRef, e.DecisionRef, string(e.Estado),
		datosEstado.Esquema, datosEstado.Algoritmo, datosEstado.ClaveRef,
		hex.EncodeToString(datosEstado.Nonce), datosEstado.HuellaSHA256,
		e.CreadoEn.UTC().Format(time.RFC3339Nano), e.ActualizadoEn.UTC().Format(time.RFC3339Nano),
		strconv.Itoa(len(e.PuntosControl)),
	}
	for _, punto := range e.PuntosControl {
		partes = append(partes, string(punto.Paso), string(punto.Estado), punto.EfectoRef,
			punto.ClaveIdempotenciaHMAC, punto.ResultadoRef, punto.HuellaResultadoSHA256,
			punto.DeclaradoEn.UTC().Format(time.RFC3339Nano), punto.CompletadoEn.UTC().Format(time.RFC3339Nano))
	}
	if e.ProyeccionLanzamiento == nil {
		partes = append(partes, "sin_proyeccion")
	} else {
		p := e.ProyeccionLanzamiento
		partes = append(partes, "con_proyeccion", p.FlujoRef, p.SesionFirmaRef, p.LanzamientoRef,
			p.CanalLanzamientoClave, p.PreparadaEn.UTC().Format(time.RFC3339Nano), p.ExpiraEn.UTC().Format(time.RFC3339Nano))
	}
	if e.Resultado == nil {
		partes = append(partes, "sin_resultado")
	} else {
		r := e.Resultado
		partes = append(partes, "con_resultado", r.FlujoRef, r.DecisionRef, r.DocumentoFirmadoRef,
			r.HuellaDocumentoFirmadoSHA256, strconv.FormatUint(r.VersionBaremacion, 10),
			r.EvidenciaConfirmacionRef, r.HuellaResultadoSHA256, r.CompletadoEn.UTC().Format(time.RFC3339Nano))
	}
	return cargaPartesCanonicas(partes)
}

type SolicitudVerificarEstadoFlujoFirmaBaremacion struct {
	RepresentacionCanonica CargaProtegida
	SelloHMAC              string
}

func (s SolicitudVerificarEstadoFlujoFirmaBaremacion) Validar() error {
	if s.RepresentacionCanonica.Validar() != nil || !huellaHMACSHA256Valida(s.SelloHMAC) {
		return ErrEstadoFlujoFirmaAlterado
	}
	return nil
}

type VerificadorEstadoFlujoFirmaBaremacion interface {
	VerificarEstadoFlujoFirmaBaremacion(context.Context, SolicitudVerificarEstadoFlujoFirmaBaremacion) error
}

type SolicitudCrearORecuperarFlujoFirmaBaremacion struct {
	Expediente ExpedienteFlujoFirmaBaremacion
}

type ResultadoCrearORecuperarFlujoFirmaBaremacion struct {
	Expediente ExpedienteFlujoFirmaBaremacion
	Creado     bool
}

type SolicitudObtenerFlujoFirmaBaremacion struct {
	FlujoRef               string
	IndiceIdempotenciaHMAC string
	VinculoActorHMAC       string
}

func (s SolicitudObtenerFlujoFirmaBaremacion) Validar() error {
	if !referenciaValida(s.FlujoRef, 512) || !huellaHMACSHA256Valida(s.IndiceIdempotenciaHMAC) ||
		!huellaHMACSHA256Valida(s.VinculoActorHMAC) {
		return ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	return nil
}

type SolicitudAdquirirArrendamientoFlujoFirmaBaremacion struct {
	Consulta        SolicitudObtenerFlujoFirmaBaremacion
	VersionEsperada uint64
	PropietarioRef  string
	Duracion        time.Duration
}

func (s SolicitudAdquirirArrendamientoFlujoFirmaBaremacion) Validar() error {
	if s.Consulta.Validar() != nil || s.VersionEsperada < 1 ||
		!referenciaValida(s.PropietarioRef, 512) || s.Duracion < time.Second ||
		s.Duracion > DuracionMaximaArrendamientoFlujoFirma {
		return ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	return nil
}

type ResultadoAdquirirArrendamientoFlujoFirmaBaremacion struct {
	Expediente    ExpedienteFlujoFirmaBaremacion
	Arrendamiento ArrendamientoFlujoFirmaBaremacion
}

type SolicitudGuardarFlujoFirmaBaremacion struct {
	VersionEsperada uint64
	Arrendamiento   ArrendamientoFlujoFirmaBaremacion
	Siguiente       ExpedienteFlujoFirmaBaremacion
}

func (s SolicitudGuardarFlujoFirmaBaremacion) Validar() error {
	if s.VersionEsperada < 1 || s.Arrendamiento.Validar() != nil || s.Siguiente.Validar() != nil ||
		s.Siguiente.FlujoRef != s.Arrendamiento.FlujoRef || s.Siguiente.Version != s.VersionEsperada+1 {
		return ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	return nil
}

type SolicitudLiberarArrendamientoFlujoFirmaBaremacion struct {
	Arrendamiento ArrendamientoFlujoFirmaBaremacion
}

type RepositorioFlujosFirmaBaremacion interface {
	CrearORecuperarFlujoFirmaBaremacion(context.Context, SolicitudCrearORecuperarFlujoFirmaBaremacion) (ResultadoCrearORecuperarFlujoFirmaBaremacion, error)
	ObtenerFlujoFirmaBaremacion(context.Context, SolicitudObtenerFlujoFirmaBaremacion) (ExpedienteFlujoFirmaBaremacion, error)
	AdquirirArrendamientoFlujoFirmaBaremacion(context.Context, SolicitudAdquirirArrendamientoFlujoFirmaBaremacion) (ResultadoAdquirirArrendamientoFlujoFirmaBaremacion, error)
	GuardarFlujoFirmaBaremacion(context.Context, SolicitudGuardarFlujoFirmaBaremacion) (ExpedienteFlujoFirmaBaremacion, error)
	LiberarArrendamientoFlujoFirmaBaremacion(context.Context, SolicitudLiberarArrendamientoFlujoFirmaBaremacion) error
}

// MismaSolicitudInicialFlujoFirmaBaremacion permite reconciliar una respuesta
// perdida sin exigir que el segundo intento repita la referencia aleatoria, el
// nonce AEAD o el instante del primer intento. Sí liga todos los identificadores
// funcionales y las derivaciones HMAC que definen la intención.
func MismaSolicitudInicialFlujoFirmaBaremacion(
	primera, segunda ExpedienteFlujoFirmaBaremacion,
) bool {
	return primera.Validar() == nil && segunda.Validar() == nil &&
		primera.Version == 1 && segunda.Version == 1 &&
		len(primera.PuntosControl) == 0 && len(segunda.PuntosControl) == 0 &&
		primera.Estado == EstadoExpedienteFirmaPreparando &&
		segunda.Estado == EstadoExpedienteFirmaPreparando &&
		primera.IndiceIdempotenciaHMAC == segunda.IndiceIdempotenciaHMAC &&
		primera.HuellaSolicitudHMAC == segunda.HuellaSolicitudHMAC &&
		primera.VinculoActorHMAC == segunda.VinculoActorHMAC &&
		primera.PerfilActorClave == segunda.PerfilActorClave &&
		primera.ProcesoRef == segunda.ProcesoRef &&
		primera.SolicitudRef == segunda.SolicitudRef &&
		primera.BaremacionMeritoRef == segunda.BaremacionMeritoRef &&
		primera.DecisionRef == segunda.DecisionRef
}

// ValidarTransicionFlujoFirmaBaremacion es la única matriz de evolución del
// expediente durable. Una versión declara el siguiente efecto o completa el
// último efecto declarado; nunca puede reescribir historia, identidad,
// idempotencia ni resultados ya confirmados.
func ValidarTransicionFlujoFirmaBaremacion(
	anterior, siguiente ExpedienteFlujoFirmaBaremacion,
) error {
	if anterior.Validar() != nil || siguiente.Validar() != nil ||
		anterior.FlujoRef != siguiente.FlujoRef ||
		siguiente.Version != anterior.Version+1 ||
		anterior.IndiceIdempotenciaHMAC != siguiente.IndiceIdempotenciaHMAC ||
		anterior.HuellaSolicitudHMAC != siguiente.HuellaSolicitudHMAC ||
		anterior.VinculoActorHMAC != siguiente.VinculoActorHMAC ||
		anterior.PerfilActorClave != siguiente.PerfilActorClave ||
		anterior.ProcesoRef != siguiente.ProcesoRef ||
		anterior.SolicitudRef != siguiente.SolicitudRef ||
		anterior.BaremacionMeritoRef != siguiente.BaremacionMeritoRef ||
		anterior.DecisionRef != siguiente.DecisionRef ||
		!anterior.CreadoEn.Equal(siguiente.CreadoEn) ||
		siguiente.ActualizadoEn.Before(anterior.ActualizadoEn) {
		return ErrEstadoFlujoFirmaAlterado
	}
	if len(siguiente.PuntosControl) == len(anterior.PuntosControl)+1 {
		if !puntosControlFlujoFirmaIguales(
			anterior.PuntosControl,
			siguiente.PuntosControl[:len(anterior.PuntosControl)],
		) ||
			siguiente.PuntosControl[len(siguiente.PuntosControl)-1].Estado !=
				EstadoPuntoControlFirmaDeclarado ||
			!estadosProtegidosFlujoFirmaIguales(
				anterior.EstadoProtegido,
				siguiente.EstadoProtegido,
			) ||
			!reflect.DeepEqual(
				anterior.ProyeccionLanzamiento,
				siguiente.ProyeccionLanzamiento,
			) ||
			!reflect.DeepEqual(anterior.Resultado, siguiente.Resultado) {
			return ErrEstadoFlujoFirmaAlterado
		}
		return nil
	}
	if len(siguiente.PuntosControl) != len(anterior.PuntosControl) ||
		len(anterior.PuntosControl) == 0 ||
		!puntosControlFlujoFirmaIguales(
			anterior.PuntosControl[:len(anterior.PuntosControl)-1],
			siguiente.PuntosControl[:len(siguiente.PuntosControl)-1],
		) {
		return ErrEstadoFlujoFirmaAlterado
	}
	puntoAnterior := anterior.PuntosControl[len(anterior.PuntosControl)-1]
	puntoSiguiente := siguiente.PuntosControl[len(siguiente.PuntosControl)-1]
	if puntoAnterior.Estado != EstadoPuntoControlFirmaDeclarado ||
		puntoSiguiente.Estado != EstadoPuntoControlFirmaCompletado ||
		puntoAnterior.Paso != puntoSiguiente.Paso ||
		puntoAnterior.EfectoRef != puntoSiguiente.EfectoRef ||
		puntoAnterior.ClaveIdempotenciaHMAC !=
			puntoSiguiente.ClaveIdempotenciaHMAC ||
		!puntoAnterior.DeclaradoEn.Equal(puntoSiguiente.DeclaradoEn) {
		return ErrEstadoFlujoFirmaAlterado
	}
	switch puntoSiguiente.Paso {
	case PasoPrepararFirmaBaremacion:
		if anterior.ProyeccionLanzamiento == nil &&
			siguiente.ProyeccionLanzamiento != nil &&
			anterior.Resultado == nil && siguiente.Resultado == nil {
			return nil
		}
	case PasoConfirmarFirmaBaremacion:
		if reflect.DeepEqual(
			anterior.ProyeccionLanzamiento,
			siguiente.ProyeccionLanzamiento,
		) && anterior.Resultado == nil && siguiente.Resultado != nil {
			return nil
		}
	default:
		if reflect.DeepEqual(
			anterior.ProyeccionLanzamiento,
			siguiente.ProyeccionLanzamiento,
		) && reflect.DeepEqual(anterior.Resultado, siguiente.Resultado) {
			return nil
		}
	}
	return ErrEstadoFlujoFirmaAlterado
}

func puntosControlFlujoFirmaIguales(
	primero, segundo []PuntoControlFirmaBaremacion,
) bool {
	if len(primero) != len(segundo) {
		return false
	}
	for indice := range primero {
		if !reflect.DeepEqual(primero[indice], segundo[indice]) {
			return false
		}
	}
	return true
}

func estadosProtegidosFlujoFirmaIguales(
	primero, segundo EstadoProtegidoFlujoFirmaBaremacion,
) bool {
	datosPrimero, errPrimero := primero.DatosPersistencia()
	datosSegundo, errSegundo := segundo.DatosPersistencia()
	return errPrimero == nil && errSegundo == nil &&
		datosPrimero.Esquema == datosSegundo.Esquema &&
		datosPrimero.Algoritmo == datosSegundo.Algoritmo &&
		datosPrimero.ClaveRef == datosSegundo.ClaveRef &&
		datosPrimero.HuellaSHA256 == datosSegundo.HuellaSHA256 &&
		bytes.Equal(datosPrimero.Nonce, datosSegundo.Nonce) &&
		bytes.Equal(datosPrimero.Cifrado, datosSegundo.Cifrado)
}

// SolicitudEjecutarPasoFirmaBaremacion porta un estado de trabajo en claro
// solo dentro del proceso. El ejecutor productivo debe rederivar identidad y
// autorizacion desde ctx y recuperar por (EfectoRef, ClaveIdempotenciaHMAC)
// antes de iniciar un efecto remoto.
type SolicitudEjecutarPasoFirmaBaremacion struct {
	FlujoRef              string
	Paso                  PasoFlujoFirmaBaremacion
	EfectoRef             string
	ClaveIdempotenciaHMAC string
	VinculoActorHMAC      string
	PerfilActorClave      string
	ProcesoRef            string
	SolicitudRef          string
	BaremacionMeritoRef   string
	DecisionRef           string
	EstadoTrabajo         CargaProtegida
	PuntosPrevios         []PuntoControlFirmaBaremacion
}

func (s SolicitudEjecutarPasoFirmaBaremacion) Validar() error {
	if !referenciaValida(s.FlujoRef, 512) || !s.Paso.Valido() || !referenciaValida(s.EfectoRef, 512) ||
		!huellaHMACSHA256Valida(s.ClaveIdempotenciaHMAC) ||
		!huellaHMACSHA256Valida(s.VinculoActorHMAC) || !claveValida(s.PerfilActorClave) ||
		!referenciaValida(s.ProcesoRef, 512) || !referenciaValida(s.SolicitudRef, 512) ||
		!referenciaValida(s.BaremacionMeritoRef, 512) || !referenciaValida(s.DecisionRef, 512) ||
		s.EstadoTrabajo.Validar() != nil || len(s.PuntosPrevios) >= len(pasosFlujoFirmaBaremacion) {
		return ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	for indice, punto := range s.PuntosPrevios {
		if punto.Validar() != nil || punto.Estado != EstadoPuntoControlFirmaCompletado ||
			punto.Paso != pasosFlujoFirmaBaremacion[indice] {
			return ErrSolicitudFlujoFirmaBaremacionInvalida
		}
	}
	if s.Paso != pasosFlujoFirmaBaremacion[len(s.PuntosPrevios)] {
		return ErrPasoFlujoFirmaNoPermitido
	}
	return nil
}

type ResultadoEjecutarPasoFirmaBaremacion struct {
	Paso                  PasoFlujoFirmaBaremacion
	EfectoRef             string
	ResultadoRef          string
	HuellaResultadoSHA256 string
	EstadoTrabajo         CargaProtegida
	ProyeccionLanzamiento *ProyeccionLanzamientoFirmaBaremacion
	ResultadoFinal        *ResultadoFinalFlujoFirmaBaremacion
	EjecutadoEn           time.Time
}

func (r ResultadoEjecutarPasoFirmaBaremacion) ValidarPara(s SolicitudEjecutarPasoFirmaBaremacion) error {
	if s.Validar() != nil || r.Paso != s.Paso || r.EfectoRef != s.EfectoRef ||
		!referenciaValida(r.ResultadoRef, 512) || !huellaSHA256Valida(r.HuellaResultadoSHA256) ||
		r.EstadoTrabajo.Validar() != nil || r.EjecutadoEn.IsZero() {
		return ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	switch r.Paso {
	case PasoPrepararFirmaBaremacion:
		if r.ProyeccionLanzamiento == nil || r.ProyeccionLanzamiento.Validar() != nil ||
			r.ProyeccionLanzamiento.FlujoRef != s.FlujoRef ||
			!r.ProyeccionLanzamiento.PreparadaEn.Equal(r.EjecutadoEn) || r.ResultadoFinal != nil {
			return ErrSolicitudFlujoFirmaBaremacionInvalida
		}
	case PasoConfirmarFirmaBaremacion:
		if r.ProyeccionLanzamiento != nil || r.ResultadoFinal == nil || r.ResultadoFinal.Validar() != nil ||
			r.ResultadoFinal.FlujoRef != s.FlujoRef || r.ResultadoFinal.DecisionRef != s.DecisionRef ||
			!r.ResultadoFinal.CompletadoEn.Equal(r.EjecutadoEn) {
			return ErrSolicitudFlujoFirmaBaremacionInvalida
		}
	default:
		if r.ProyeccionLanzamiento != nil || r.ResultadoFinal != nil {
			return ErrSolicitudFlujoFirmaBaremacionInvalida
		}
	}
	return nil
}

type EjecutorPasosFirmaBaremacion interface {
	EjecutarPasoFirmaBaremacion(context.Context, SolicitudEjecutarPasoFirmaBaremacion) (ResultadoEjecutarPasoFirmaBaremacion, error)
}

type GeneradorReferenciasFlujoFirmaBaremacion interface {
	NuevaReferenciaFlujoFirmaBaremacion() (string, error)
	NuevaReferenciaPropietarioArrendamientoFirmaBaremacion() (string, error)
	NuevaReferenciaEfectoFirmaBaremacion(PasoFlujoFirmaBaremacion) (string, error)
}
