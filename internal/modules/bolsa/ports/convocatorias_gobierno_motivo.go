package ports

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const DominioCriptograficoMotivoGobiernoConvocatoriaV1 = "bolsa.convocatoria.motivo.v1"

const VigenciaMaximaAtestacionMotivoGobiernoConvocatoria = 5 * time.Minute

var (
	ErrSelladoMotivoGobiernoConvocatoriaInvalido = errors.New("bolsa: sellado HMAC de motivo de convocatoria invalido")
	ErrSerializacionMotivoGobiernoConvocatoria   = errors.New("bolsa: serializacion de motivo de convocatoria prohibida")
)

// SolicitudSemanticaMotivoGobiernoConvocatoria es un valor interno local. Es
// el unico que porta el motivo en claro y nunca cruza el puerto HSM/KMS.
type SolicitudSemanticaMotivoGobiernoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	DominioCriptografico string
	Accion               string
	ConvocatoriaRef      string
	PrincipalRef         string
	CorrelacionRef       string
	Motivo               string
	SolicitadaEn         time.Time
}

func (s SolicitudSemanticaMotivoGobiernoConvocatoria) Validar() error {
	especificacion, conocida := especificacionesAutorizacionConvocatoria[s.Accion]
	if s.DominioCriptografico != DominioCriptograficoMotivoGobiernoConvocatoriaV1 ||
		!conocida || !especificacion.mutacion ||
		!referenciaVersionGobernadaConvocatoriaValida(s.ConvocatoriaRef) ||
		!referenciaGobiernoConvocatoriaValida(s.PrincipalRef) ||
		!referenciaGobiernoConvocatoriaValida(s.CorrelacionRef) ||
		!textoMotivoGobiernoConvocatoriaValido(s.Motivo) ||
		!instanteGobiernoConvocatoriaCanonico(s.SolicitadaEn) {
		return ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return nil
}

// HuellaSHA256 mantiene la preimagen semantica V1. El instante queda fuera:
// los reintentos de una misma intencion producen el mismo compromiso HMAC.
func (s SolicitudSemanticaMotivoGobiernoConvocatoria) HuellaSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return huellaSemanticaMotivoGobiernoConvocatoria(
		s.Accion, s.ConvocatoriaRef, s.PrincipalRef, s.CorrelacionRef, s.Motivo,
	)
}

func huellaSemanticaMotivoGobiernoConvocatoria(
	accion, convocatoriaRef, principalRef, correlacionRef, motivo string,
) (string, error) {
	especificacion, conocida := especificacionesAutorizacionConvocatoria[accion]
	if !conocida || !especificacion.mutacion ||
		!referenciaVersionGobernadaConvocatoriaValida(convocatoriaRef) ||
		!referenciaGobiernoConvocatoriaValida(principalRef) ||
		!referenciaGobiernoConvocatoriaValida(correlacionRef) ||
		!textoMotivoGobiernoConvocatoriaValido(motivo) {
		return "", ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	preimagen := struct {
		Esquema         string `json:"esquema"`
		Dominio         string `json:"dominio"`
		Accion          string `json:"accion"`
		ConvocatoriaRef string `json:"convocatoria_ref"`
		PrincipalRef    string `json:"principal_ref"`
		CorrelacionRef  string `json:"correlacion_ref"`
		Motivo          string `json:"motivo"`
	}{
		"bolsa.convocatoria.solicitud-sellado-motivo.v1",
		DominioCriptograficoMotivoGobiernoConvocatoriaV1,
		accion, convocatoriaRef, principalRef, correlacionRef, motivo,
	}
	contenido, err := json.Marshal(preimagen)
	if err != nil {
		return "", ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

func (SolicitudSemanticaMotivoGobiernoConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMotivoGobiernoConvocatoria
}
func (SolicitudSemanticaMotivoGobiernoConvocatoria) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionMotivoGobiernoConvocatoria
}
func (SolicitudSemanticaMotivoGobiernoConvocatoria) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionMotivoGobiernoConvocatoria
}
func (SolicitudSemanticaMotivoGobiernoConvocatoria) String() string {
	return "[SOLICITUD-SEMANTICA-MOTIVO-CONVOCATORIA-LOCAL]"
}
func (s SolicitudSemanticaMotivoGobiernoConvocatoria) GoString() string { return s.String() }
func (s SolicitudSemanticaMotivoGobiernoConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudSemanticaMotivoGobiernoConvocatoria) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

// SolicitudComprometerMotivoGobiernoConvocatoria es la peticion minimizada al
// HSM/KMS. El servicio de claves recibe exclusivamente dominio y huella; no
// conoce accion, referencia, actor, correlacion ni motivo en claro.
type SolicitudComprometerMotivoGobiernoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	DominioCriptografico  string
	HuellaSolicitudSHA256 string
}

func NuevaSolicitudComprometerMotivoGobiernoConvocatoria(
	semantica SolicitudSemanticaMotivoGobiernoConvocatoria,
) (SolicitudComprometerMotivoGobiernoConvocatoria, error) {
	huella, err := semantica.HuellaSHA256()
	solicitud := SolicitudComprometerMotivoGobiernoConvocatoria{
		DominioCriptografico:  semantica.DominioCriptografico,
		HuellaSolicitudSHA256: huella,
	}
	if err != nil || solicitud.Validar() != nil {
		return SolicitudComprometerMotivoGobiernoConvocatoria{}, ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return solicitud, nil
}

func (s SolicitudComprometerMotivoGobiernoConvocatoria) Validar() error {
	if s.DominioCriptografico != DominioCriptograficoMotivoGobiernoConvocatoriaV1 ||
		!huellaGobiernoConvocatoriaValida(s.HuellaSolicitudSHA256) {
		return ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return nil
}

func (SolicitudComprometerMotivoGobiernoConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMotivoGobiernoConvocatoria
}
func (SolicitudComprometerMotivoGobiernoConvocatoria) String() string {
	return "[SOLICITUD-HSM-COMPROMISO-MOTIVO-CONVOCATORIA-INTERNA]"
}
func (s SolicitudComprometerMotivoGobiernoConvocatoria) GoString() string { return s.String() }
func (s SolicitudComprometerMotivoGobiernoConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudComprometerMotivoGobiernoConvocatoria) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

// HMACMotivoGobiernoConvocatoria es efimero y no serializable: identifica
// dominio, entrada y generacion sin contener la clave. Su autenticidad se
// verifica otra vez en el HSM al materializar.
type HMACMotivoGobiernoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	DominioCriptografico string
	GeneracionClave      uint32
	ClaveHMACRef         string
	HuellaEntradaSHA256  string
	ValorHMACSHA256      string
}

func (h HMACMotivoGobiernoConvocatoria) Validar() error {
	if h.DominioCriptografico != DominioCriptograficoMotivoGobiernoConvocatoriaV1 ||
		h.GeneracionClave < 1 || !claveValida(h.ClaveHMACRef) ||
		!huellaGobiernoConvocatoriaValida(h.HuellaEntradaSHA256) ||
		!huellaGobiernoConvocatoriaValida(h.ValorHMACSHA256) {
		return ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return nil
}

func (h HMACMotivoGobiernoConvocatoria) representacionMaterial() (string, error) {
	proyeccion, err := h.ProyeccionDurable()
	if err != nil {
		return "", ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return proyeccion.representacionMaterial()
}

func (h HMACMotivoGobiernoConvocatoria) igualConstante(
	otro HMACMotivoGobiernoConvocatoria,
) bool {
	if h.Validar() != nil || otro.Validar() != nil {
		return false
	}
	proyeccion, err := h.ProyeccionDurable()
	proyeccionOtra, errOtra := otro.ProyeccionDurable()
	if err != nil || errOtra != nil {
		return false
	}
	coincide := boolAEnteroConstante(proyeccion.igualConstante(proyeccionOtra))
	coincide &= boolAEnteroConstante(huellaMotivoGobiernoIgualConstante(
		h.HuellaEntradaSHA256, otro.HuellaEntradaSHA256,
	))
	return coincide == 1
}

// ProyeccionHMACMotivoGobiernoConvocatoriaDurable conserva unicamente el
// compromiso con clave que puede persistirse en la atestacion. Deliberadamente
// no contiene HuellaEntradaSHA256 ni permite recuperar la huella semantica.
type ProyeccionHMACMotivoGobiernoConvocatoriaDurable struct {
	bloqueoSerializacionGobiernoConvocatoria
	DominioCriptografico string
	GeneracionClave      uint32
	ClaveHMACRef         string
	ValorHMACSHA256      string
}

func (p ProyeccionHMACMotivoGobiernoConvocatoriaDurable) Validar() error {
	if p.DominioCriptografico != DominioCriptograficoMotivoGobiernoConvocatoriaV1 ||
		p.GeneracionClave < 1 || !claveValida(p.ClaveHMACRef) ||
		!huellaGobiernoConvocatoriaValida(p.ValorHMACSHA256) {
		return ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return nil
}

func (p ProyeccionHMACMotivoGobiernoConvocatoriaDurable) representacionMaterial() (string, error) {
	if p.Validar() != nil {
		return "", ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return "hmac-sha256:" + p.ClaveHMACRef + ":" + p.ValorHMACSHA256, nil
}

func (p ProyeccionHMACMotivoGobiernoConvocatoriaDurable) igualConstante(
	otra ProyeccionHMACMotivoGobiernoConvocatoriaDurable,
) bool {
	if p.Validar() != nil || otra.Validar() != nil {
		return false
	}
	representacion, err := p.representacionMaterial()
	representacionOtra, errOtra := otra.representacionMaterial()
	if err != nil || errOtra != nil {
		return false
	}
	coincide := subtle.ConstantTimeEq(int32(p.GeneracionClave), int32(otra.GeneracionClave))
	coincide &= boolAEnteroConstante(representacionHMACMotivoGobiernoIgualConstante(
		representacion, representacionOtra,
	))
	return coincide == 1
}

// ProyeccionDurable elimina la huella de entrada antes de cruzar al almacen.
func (h HMACMotivoGobiernoConvocatoria) ProyeccionDurable() (
	ProyeccionHMACMotivoGobiernoConvocatoriaDurable,
	error,
) {
	proyeccion := ProyeccionHMACMotivoGobiernoConvocatoriaDurable{
		DominioCriptografico: h.DominioCriptografico,
		GeneracionClave:      h.GeneracionClave, ClaveHMACRef: h.ClaveHMACRef,
		ValorHMACSHA256: h.ValorHMACSHA256,
	}
	if h.Validar() != nil || proyeccion.Validar() != nil {
		return ProyeccionHMACMotivoGobiernoConvocatoriaDurable{}, ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return proyeccion, nil
}

func (ProyeccionHMACMotivoGobiernoConvocatoriaDurable) String() string {
	return "[PROYECCION-HMAC-MOTIVO-CONVOCATORIA-DURABLE]"
}
func (p ProyeccionHMACMotivoGobiernoConvocatoriaDurable) GoString() string { return p.String() }
func (p ProyeccionHMACMotivoGobiernoConvocatoriaDurable) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (p ProyeccionHMACMotivoGobiernoConvocatoriaDurable) LogValue() slog.Value {
	return slog.StringValue(p.String())
}

func huellaMotivoGobiernoIgualConstante(izquierda, derecha string) bool {
	return len(izquierda) == sha256.Size*2 && len(derecha) == sha256.Size*2 &&
		subtle.ConstantTimeCompare([]byte(izquierda), []byte(derecha)) == 1
}

func representacionHMACMotivoGobiernoIgualConstante(izquierda, derecha string) bool {
	const longitudMaxima = 256
	if len(izquierda) > longitudMaxima || len(derecha) > longitudMaxima {
		return false
	}
	var canonicaIzquierda, canonicaDerecha [longitudMaxima]byte
	copy(canonicaIzquierda[:], izquierda)
	copy(canonicaDerecha[:], derecha)
	coincide := subtle.ConstantTimeEq(int32(len(izquierda)), int32(len(derecha)))
	coincide &= subtle.ConstantTimeCompare(canonicaIzquierda[:], canonicaDerecha[:])
	return coincide == 1
}

func boolAEnteroConstante(valor bool) int {
	if valor {
		return 1
	}
	return 0
}

func (HMACMotivoGobiernoConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMotivoGobiernoConvocatoria
}
func (HMACMotivoGobiernoConvocatoria) String() string     { return "[HMAC-MOTIVO-CONVOCATORIA-OCULTO]" }
func (h HMACMotivoGobiernoConvocatoria) GoString() string { return h.String() }
func (h HMACMotivoGobiernoConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, h.String())
}
func (h HMACMotivoGobiernoConvocatoria) LogValue() slog.Value {
	return slog.StringValue(h.String())
}

// DatosCompromisoMotivoGobiernoConvocatoria no contiene identificadores de
// atestacion ni tokens consumibles. Es el resultado determinista y no durable
// necesario para formar la intencion que evalua el PDP.
type DatosCompromisoMotivoGobiernoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	HMAC                  HMACMotivoGobiernoConvocatoria
	Accion                string
	ConvocatoriaRef       string
	PrincipalRef          string
	CorrelacionRef        string
	HuellaSolicitudSHA256 string
}

type CompromisoMotivoGobiernoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	datos *DatosCompromisoMotivoGobiernoConvocatoria
}

func NuevoCompromisoMotivoGobiernoConvocatoria(
	solicitud SolicitudSemanticaMotivoGobiernoConvocatoria,
	hmac HMACMotivoGobiernoConvocatoria,
) (CompromisoMotivoGobiernoConvocatoria, error) {
	huella, err := solicitud.HuellaSHA256()
	datos := DatosCompromisoMotivoGobiernoConvocatoria{
		HMAC: hmac, Accion: solicitud.Accion, ConvocatoriaRef: solicitud.ConvocatoriaRef,
		PrincipalRef: solicitud.PrincipalRef, CorrelacionRef: solicitud.CorrelacionRef,
		HuellaSolicitudSHA256: huella,
	}
	if err != nil || validarDatosCompromisoMotivo(datos) != nil ||
		hmac.DominioCriptografico != solicitud.DominioCriptografico ||
		!huellaMotivoGobiernoIgualConstante(hmac.HuellaEntradaSHA256, huella) {
		return CompromisoMotivoGobiernoConvocatoria{}, ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	copia := datos
	return CompromisoMotivoGobiernoConvocatoria{datos: &copia}, nil
}

func (c CompromisoMotivoGobiernoConvocatoria) DatosParaMaterial() (
	DatosCompromisoMotivoGobiernoConvocatoria,
	error,
) {
	if c.datos == nil || validarDatosCompromisoMotivo(*c.datos) != nil {
		return DatosCompromisoMotivoGobiernoConvocatoria{}, ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return *c.datos, nil
}

func (c CompromisoMotivoGobiernoConvocatoria) material() (
	HMACMotivoGobiernoConvocatoria,
	error,
) {
	datos, err := c.DatosParaMaterial()
	if err != nil {
		return HMACMotivoGobiernoConvocatoria{}, err
	}
	return datos.HMAC, nil
}

func (c CompromisoMotivoGobiernoConvocatoria) coincideMaterial(
	material MaterialIntencionGobiernoConvocatoria,
) bool {
	datos, err := c.DatosParaMaterial()
	representacion, errHMAC := datos.HMAC.representacionMaterial()
	return err == nil && errHMAC == nil && datos.Accion == material.Accion &&
		datos.ConvocatoriaRef == material.EstadoPrincipalNuevo.Referencia &&
		datos.HMAC.DominioCriptografico == material.DominioCriptograficoMotivo &&
		datos.HMAC.GeneracionClave == material.GeneracionClaveMotivo &&
		representacionHMACMotivoGobiernoIgualConstante(representacion, material.HuellaMotivoHMACSHA256)
}

func (CompromisoMotivoGobiernoConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMotivoGobiernoConvocatoria
}
func (CompromisoMotivoGobiernoConvocatoria) String() string {
	return "[COMPROMISO-MOTIVO-CONVOCATORIA-INTERNO]"
}
func (c CompromisoMotivoGobiernoConvocatoria) GoString() string { return c.String() }
func (c CompromisoMotivoGobiernoConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (c CompromisoMotivoGobiernoConvocatoria) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

func validarDatosCompromisoMotivo(datos DatosCompromisoMotivoGobiernoConvocatoria) error {
	especificacion, conocida := especificacionesAutorizacionConvocatoria[datos.Accion]
	if datos.HMAC.Validar() != nil || !conocida || !especificacion.mutacion ||
		!referenciaVersionGobernadaConvocatoriaValida(datos.ConvocatoriaRef) ||
		!referenciaGobiernoConvocatoriaValida(datos.PrincipalRef) ||
		!referenciaGobiernoConvocatoriaValida(datos.CorrelacionRef) ||
		!huellaGobiernoConvocatoriaValida(datos.HuellaSolicitudSHA256) ||
		!huellaMotivoGobiernoIgualConstante(datos.HMAC.HuellaEntradaSHA256, datos.HuellaSolicitudSHA256) {
		return ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return nil
}

type DatosSolicitudMaterializarSelladoMotivoGobiernoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	Compromiso                         CompromisoMotivoGobiernoConvocatoria
	HuellaIntencionSHA256              string
	DecisionRef                        string
	HuellaDecisionSHA256               string
	IndiceIdempotenciaHMACSHA256       string
	AtestacionIdempotenciaRef          string
	HuellaAtestacionIdempotenciaSHA256 string
	IdempotenciaEmitidaEn              time.Time
	IdempotenciaValidaHasta            time.Time
	PrincipalRef                       string
	CorrelacionRef                     string
	AutorizacionVerificadaEn           time.Time
	DecisionValidaHasta                time.Time
	SolicitadaEn                       time.Time
}

// SolicitudMaterializarSelladoMotivoGobiernoConvocatoria nace solo despues de
// una concesion PDP exacta. No porta el motivo en claro ni un token consumible.
type SolicitudMaterializarSelladoMotivoGobiernoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	datos *DatosSolicitudMaterializarSelladoMotivoGobiernoConvocatoria
}

func NuevaSolicitudMaterializarSelladoMotivoGobiernoConvocatoria(
	compromiso CompromisoMotivoGobiernoConvocatoria,
	material MaterialIntencionGobiernoConvocatoria,
	autorizacion puertosvec.EvidenciaUsoDecisionAutorizacion,
	testimonio TestimonioIdempotenciaConvocatoria,
	version dominiobolsa.VersionConvocatoriaGobernada,
	solicitadaEn time.Time,
) (SolicitudMaterializarSelladoMotivoGobiernoConvocatoria, error) {
	datosAutorizacion, errAutorizacion := autorizacion.Datos()
	datosCompromiso, errCompromiso := compromiso.DatosParaMaterial()
	datosIdempotencia, errIdempotencia := testimonio.Datos()
	huellaIntencion, errHuella := material.HuellaSHA256()
	datos := DatosSolicitudMaterializarSelladoMotivoGobiernoConvocatoria{
		Compromiso: compromiso, HuellaIntencionSHA256: huellaIntencion,
		DecisionRef:                        datosAutorizacion.Decision.DecisionRef,
		HuellaDecisionSHA256:               datosAutorizacion.HuellaDecisionSHA256,
		IndiceIdempotenciaHMACSHA256:       datosIdempotencia.IndiceOperacionHMACSHA256,
		AtestacionIdempotenciaRef:          datosIdempotencia.AtestacionRef,
		HuellaAtestacionIdempotenciaSHA256: datosIdempotencia.HuellaAtestacionSHA256,
		IdempotenciaEmitidaEn:              datosIdempotencia.EmitidoEn,
		IdempotenciaValidaHasta:            datosIdempotencia.ValidoHasta,
		PrincipalRef:                       datosAutorizacion.Decision.PrincipalID,
		CorrelacionRef:                     datosAutorizacion.Decision.CorrelacionRef,
		AutorizacionVerificadaEn:           datosAutorizacion.VerificadaEn,
		DecisionValidaHasta:                datosAutorizacion.Decision.ValidaHasta,
		SolicitadaEn:                       solicitadaEn,
	}
	if errAutorizacion != nil || errCompromiso != nil || errIdempotencia != nil || errHuella != nil ||
		validarUsoAutorizacionMutacionConvocatoria(
			autorizacion, material.Accion, material, version, solicitadaEn,
		) != nil || !compromiso.coincideMaterial(material) ||
		testimonio.ValidarPara(material, datosAutorizacion.Decision.PrincipalID) != nil ||
		datosCompromiso.PrincipalRef != datos.PrincipalRef ||
		datosCompromiso.CorrelacionRef != datos.CorrelacionRef ||
		validarDatosSolicitudMaterializacionMotivo(datos) != nil {
		return SolicitudMaterializarSelladoMotivoGobiernoConvocatoria{}, ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	copia := datos
	return SolicitudMaterializarSelladoMotivoGobiernoConvocatoria{datos: &copia}, nil
}

func (s SolicitudMaterializarSelladoMotivoGobiernoConvocatoria) DatosParaMaterializacion() (
	DatosSolicitudMaterializarSelladoMotivoGobiernoConvocatoria,
	error,
) {
	if s.datos == nil || validarDatosSolicitudMaterializacionMotivo(*s.datos) != nil {
		return DatosSolicitudMaterializarSelladoMotivoGobiernoConvocatoria{}, ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return *s.datos, nil
}

func validarDatosSolicitudMaterializacionMotivo(
	datos DatosSolicitudMaterializarSelladoMotivoGobiernoConvocatoria,
) error {
	datosCompromiso, errCompromiso := datos.Compromiso.DatosParaMaterial()
	if errCompromiso != nil || !huellaGobiernoConvocatoriaValida(datos.HuellaIntencionSHA256) ||
		!referenciaGobiernoConvocatoriaValida(datos.DecisionRef) ||
		!huellaGobiernoConvocatoriaValida(datos.HuellaDecisionSHA256) ||
		!huellaHMACGobiernoConvocatoriaValida(datos.IndiceIdempotenciaHMACSHA256) ||
		!referenciaGobiernoConvocatoriaValida(datos.AtestacionIdempotenciaRef) ||
		!huellaGobiernoConvocatoriaValida(datos.HuellaAtestacionIdempotenciaSHA256) ||
		!instanteGobiernoConvocatoriaCanonico(datos.IdempotenciaEmitidaEn) ||
		!instanteGobiernoConvocatoriaCanonico(datos.IdempotenciaValidaHasta) ||
		datos.SolicitadaEn.Before(datos.IdempotenciaEmitidaEn) ||
		!datos.SolicitadaEn.Before(datos.IdempotenciaValidaHasta) ||
		!referenciaGobiernoConvocatoriaValida(datos.PrincipalRef) ||
		!referenciaGobiernoConvocatoriaValida(datos.CorrelacionRef) ||
		datos.PrincipalRef != datosCompromiso.PrincipalRef ||
		datos.CorrelacionRef != datosCompromiso.CorrelacionRef ||
		!instanteGobiernoConvocatoriaCanonico(datos.AutorizacionVerificadaEn) ||
		datos.SolicitadaEn.Before(datos.AutorizacionVerificadaEn) ||
		!instanteGobiernoConvocatoriaCanonico(datos.DecisionValidaHasta) ||
		!datos.DecisionValidaHasta.After(datos.SolicitadaEn) ||
		!instanteGobiernoConvocatoriaCanonico(datos.SolicitadaEn) {
		return ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return nil
}

// HuellaReconciliacionSHA256 identifica de forma estable la materializacion.
// Excluye instantes de reintento: la misma intencion, decision e idempotencia
// debe recuperar la misma atestacion durable.
func (s SolicitudMaterializarSelladoMotivoGobiernoConvocatoria) HuellaReconciliacionSHA256() (
	string,
	error,
) {
	datos, err := s.DatosParaMaterializacion()
	if err != nil {
		return "", ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	datosCompromiso, err := datos.Compromiso.DatosParaMaterial()
	representacionHMAC, errHMAC := datosCompromiso.HMAC.representacionMaterial()
	if err != nil || errHMAC != nil {
		return "", ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	preimagen := struct {
		Esquema                            string `json:"esquema"`
		Accion                             string `json:"accion"`
		ConvocatoriaRef                    string `json:"convocatoria_ref"`
		PrincipalRef                       string `json:"principal_ref"`
		CorrelacionRef                     string `json:"correlacion_ref"`
		HMAC                               string `json:"hmac"`
		HuellaIntencionSHA256              string `json:"huella_intencion_sha256"`
		DecisionRef                        string `json:"decision_ref"`
		HuellaDecisionSHA256               string `json:"huella_decision_sha256"`
		IndiceIdempotenciaHMACSHA256       string `json:"indice_idempotencia_hmac_sha256"`
		AtestacionIdempotenciaRef          string `json:"atestacion_idempotencia_ref"`
		HuellaAtestacionIdempotenciaSHA256 string `json:"huella_atestacion_idempotencia_sha256"`
	}{
		"bolsa.convocatoria.materializacion-sellado-motivo.v2",
		datosCompromiso.Accion, datosCompromiso.ConvocatoriaRef,
		datos.PrincipalRef, datos.CorrelacionRef, representacionHMAC,
		datos.HuellaIntencionSHA256, datos.DecisionRef,
		datos.HuellaDecisionSHA256, datos.IndiceIdempotenciaHMACSHA256,
		datos.AtestacionIdempotenciaRef, datos.HuellaAtestacionIdempotenciaSHA256,
	}
	contenido, err := json.Marshal(preimagen)
	if err != nil {
		return "", ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

func (SolicitudMaterializarSelladoMotivoGobiernoConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMotivoGobiernoConvocatoria
}
func (SolicitudMaterializarSelladoMotivoGobiernoConvocatoria) String() string {
	return "[SOLICITUD-MATERIALIZAR-SELLADO-MOTIVO-CONVOCATORIA-INTERNA]"
}
func (s SolicitudMaterializarSelladoMotivoGobiernoConvocatoria) GoString() string { return s.String() }
func (s SolicitudMaterializarSelladoMotivoGobiernoConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudMaterializarSelladoMotivoGobiernoConvocatoria) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

type DatosAtestacionSelladoMotivoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	HMAC                               ProyeccionHMACMotivoGobiernoConvocatoriaDurable
	Accion                             string
	ConvocatoriaRef                    string
	PrincipalRef                       string
	CorrelacionRef                     string
	HuellaIntencionSHA256              string
	DecisionRef                        string
	HuellaDecisionSHA256               string
	IndiceIdempotenciaHMACSHA256       string
	AtestacionIdempotenciaRef          string
	HuellaAtestacionIdempotenciaSHA256 string
	MaterializadorRef                  string
	AtestacionRef                      string
	HuellaAtestacionSHA256             string
	TokenConsumoRef                    string
	AtestacionEmitidaEn                time.Time
	AtestacionValidaHasta              time.Time
}

// AtestacionSelladoMotivoConvocatoria es la unica fase consumible. La barrera
// durable debe releerla y consumir su token en la transaccion de la mutacion.
type AtestacionSelladoMotivoConvocatoria struct {
	bloqueoSerializacionGobiernoConvocatoria
	datos *DatosAtestacionSelladoMotivoConvocatoria
}

func NuevaAtestacionSelladoMotivoConvocatoria(
	solicitud SolicitudMaterializarSelladoMotivoGobiernoConvocatoria,
	datos DatosAtestacionSelladoMotivoConvocatoria,
) (AtestacionSelladoMotivoConvocatoria, error) {
	datosSolicitud, errSolicitud := solicitud.DatosParaMaterializacion()
	datosCompromiso, errCompromiso := datosSolicitud.Compromiso.DatosParaMaterial()
	proyeccionHMAC, errProyeccion := datosCompromiso.HMAC.ProyeccionDurable()
	if errSolicitud != nil || errCompromiso != nil || validarDatosAtestacionSelladoMotivo(datos) != nil ||
		errProyeccion != nil || !datos.HMAC.igualConstante(proyeccionHMAC) ||
		datos.Accion != datosCompromiso.Accion ||
		datos.ConvocatoriaRef != datosCompromiso.ConvocatoriaRef ||
		datos.PrincipalRef != datosSolicitud.PrincipalRef ||
		datos.CorrelacionRef != datosSolicitud.CorrelacionRef ||
		!huellaMotivoGobiernoIgualConstante(
			datos.HuellaIntencionSHA256, datosSolicitud.HuellaIntencionSHA256,
		) ||
		datos.DecisionRef != datosSolicitud.DecisionRef ||
		!huellaMotivoGobiernoIgualConstante(
			datos.HuellaDecisionSHA256, datosSolicitud.HuellaDecisionSHA256,
		) || !representacionHMACMotivoGobiernoIgualConstante(
		datos.IndiceIdempotenciaHMACSHA256, datosSolicitud.IndiceIdempotenciaHMACSHA256,
	) ||
		datos.AtestacionIdempotenciaRef != datosSolicitud.AtestacionIdempotenciaRef ||
		!huellaMotivoGobiernoIgualConstante(
			datos.HuellaAtestacionIdempotenciaSHA256,
			datosSolicitud.HuellaAtestacionIdempotenciaSHA256,
		) ||
		datos.AtestacionEmitidaEn.Before(datosSolicitud.AutorizacionVerificadaEn) ||
		datos.AtestacionEmitidaEn.Before(datosSolicitud.IdempotenciaEmitidaEn) ||
		!datosSolicitud.SolicitadaEn.Before(datos.AtestacionValidaHasta) ||
		datos.AtestacionEmitidaEn.After(datosSolicitud.DecisionValidaHasta) ||
		datos.AtestacionEmitidaEn.After(datosSolicitud.IdempotenciaValidaHasta) ||
		datos.AtestacionValidaHasta.After(datosSolicitud.DecisionValidaHasta) ||
		datos.AtestacionValidaHasta.After(datosSolicitud.IdempotenciaValidaHasta) {
		return AtestacionSelladoMotivoConvocatoria{}, ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	copia := datos
	return AtestacionSelladoMotivoConvocatoria{datos: &copia}, nil
}

func (a AtestacionSelladoMotivoConvocatoria) DatosParaConsumo() (
	DatosAtestacionSelladoMotivoConvocatoria,
	error,
) {
	if a.datos == nil || validarDatosAtestacionSelladoMotivo(*a.datos) != nil {
		return DatosAtestacionSelladoMotivoConvocatoria{}, ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return *a.datos, nil
}

func (a AtestacionSelladoMotivoConvocatoria) validarPara(
	compromiso CompromisoMotivoGobiernoConvocatoria,
	material MaterialIntencionGobiernoConvocatoria,
	autorizacion puertosvec.EvidenciaUsoDecisionAutorizacion,
	testimonio TestimonioIdempotenciaConvocatoria,
	instante time.Time,
) error {
	datos, err := a.DatosParaConsumo()
	datosCompromiso, errCompromiso := compromiso.DatosParaMaterial()
	proyeccionHMAC, errProyeccion := datosCompromiso.HMAC.ProyeccionDurable()
	datosAutorizacion, errAutorizacion := autorizacion.Datos()
	datosIdempotencia, errIdempotencia := testimonio.Datos()
	huellaIntencion, errHuella := material.HuellaSHA256()
	representacion, errHMAC := datos.HMAC.representacionMaterial()
	if err != nil || errCompromiso != nil || errProyeccion != nil || errAutorizacion != nil || errIdempotencia != nil || errHuella != nil || errHMAC != nil ||
		material.Validar() != nil || !compromiso.coincideMaterial(material) ||
		testimonio.ValidarPara(material, datosAutorizacion.Decision.PrincipalID) != nil ||
		!datos.HMAC.igualConstante(proyeccionHMAC) || datos.Accion != material.Accion ||
		datos.ConvocatoriaRef != material.EstadoPrincipalNuevo.Referencia ||
		datos.PrincipalRef != datosAutorizacion.Decision.PrincipalID ||
		datos.CorrelacionRef != datosAutorizacion.Decision.CorrelacionRef ||
		!huellaMotivoGobiernoIgualConstante(datos.HuellaIntencionSHA256, huellaIntencion) ||
		datos.DecisionRef != datosAutorizacion.Decision.DecisionRef ||
		!huellaMotivoGobiernoIgualConstante(
			datos.HuellaDecisionSHA256, datosAutorizacion.HuellaDecisionSHA256,
		) || !representacionHMACMotivoGobiernoIgualConstante(
		datos.IndiceIdempotenciaHMACSHA256, datosIdempotencia.IndiceOperacionHMACSHA256,
	) ||
		datos.AtestacionIdempotenciaRef != datosIdempotencia.AtestacionRef ||
		!huellaMotivoGobiernoIgualConstante(
			datos.HuellaAtestacionIdempotenciaSHA256, datosIdempotencia.HuellaAtestacionSHA256,
		) ||
		datos.AtestacionEmitidaEn.Before(datosAutorizacion.VerificadaEn) ||
		datos.AtestacionEmitidaEn.Before(datosIdempotencia.EmitidoEn) ||
		datos.AtestacionValidaHasta.After(datosAutorizacion.Decision.ValidaHasta) ||
		datos.AtestacionValidaHasta.After(datosIdempotencia.ValidoHasta) ||
		!representacionHMACMotivoGobiernoIgualConstante(
			representacion, material.HuellaMotivoHMACSHA256,
		) ||
		!instanteGobiernoConvocatoriaCanonico(instante) ||
		instante.Before(datos.AtestacionEmitidaEn) || !instante.Before(datos.AtestacionValidaHasta) {
		return ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return nil
}

func (AtestacionSelladoMotivoConvocatoria) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionMotivoGobiernoConvocatoria
}
func (AtestacionSelladoMotivoConvocatoria) String() string {
	return "[ATESTACION-SELLADO-MOTIVO-CONVOCATORIA-INTERNA]"
}
func (a AtestacionSelladoMotivoConvocatoria) GoString() string { return a.String() }
func (a AtestacionSelladoMotivoConvocatoria) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, a.String())
}
func (a AtestacionSelladoMotivoConvocatoria) LogValue() slog.Value {
	return slog.StringValue(a.String())
}

func validarDatosAtestacionSelladoMotivo(datos DatosAtestacionSelladoMotivoConvocatoria) error {
	especificacion, conocida := especificacionesAutorizacionConvocatoria[datos.Accion]
	if datos.HMAC.Validar() != nil || !conocida || !especificacion.mutacion ||
		!referenciaVersionGobernadaConvocatoriaValida(datos.ConvocatoriaRef) ||
		!referenciaGobiernoConvocatoriaValida(datos.PrincipalRef) ||
		!referenciaGobiernoConvocatoriaValida(datos.CorrelacionRef) ||
		!huellaGobiernoConvocatoriaValida(datos.HuellaIntencionSHA256) ||
		!referenciaGobiernoConvocatoriaValida(datos.DecisionRef) ||
		!huellaGobiernoConvocatoriaValida(datos.HuellaDecisionSHA256) ||
		!huellaHMACGobiernoConvocatoriaValida(datos.IndiceIdempotenciaHMACSHA256) ||
		!referenciaGobiernoConvocatoriaValida(datos.AtestacionIdempotenciaRef) ||
		!huellaGobiernoConvocatoriaValida(datos.HuellaAtestacionIdempotenciaSHA256) ||
		!referenciaGobiernoConvocatoriaValida(datos.MaterializadorRef) ||
		!referenciaGobiernoConvocatoriaValida(datos.AtestacionRef) ||
		!huellaGobiernoConvocatoriaValida(datos.HuellaAtestacionSHA256) ||
		!referenciaGobiernoConvocatoriaValida(datos.TokenConsumoRef) ||
		!referenciasGobiernoConvocatoriaDistintas(
			datos.DecisionRef, datos.AtestacionIdempotenciaRef, datos.MaterializadorRef,
			datos.AtestacionRef, datos.TokenConsumoRef,
		) || !instanteGobiernoConvocatoriaCanonico(datos.AtestacionEmitidaEn) ||
		!instanteGobiernoConvocatoriaCanonico(datos.AtestacionValidaHasta) ||
		!datos.AtestacionValidaHasta.After(datos.AtestacionEmitidaEn) ||
		datos.AtestacionValidaHasta.Sub(datos.AtestacionEmitidaEn) > VigenciaMaximaAtestacionMotivoGobiernoConvocatoria {
		return ErrSelladoMotivoGobiernoConvocatoriaInvalido
	}
	return nil
}

// ComprometedorMotivoGobiernoConvocatoria calcula de forma determinista
// HMAC-SHA256(DominioCriptografico || 0x00 || HuellaSolicitudSHA256), sin crear
// estado durable ni una capacidad consumible antes de consultar al PDP.
type ComprometedorMotivoGobiernoConvocatoria interface {
	ComprometerMotivo(
		context.Context,
		SolicitudComprometerMotivoGobiernoConvocatoria,
	) (HMACMotivoGobiernoConvocatoria, error)
}

// MaterializadorSelladoMotivoGobiernoConvocatoria crea la atestacion durable
// y consumible exclusivamente despues de recibir la decision PDP ligada. Debe
// pedir al HSM/KMS que verifique de nuevo el HMAC sobre HuellaEntradaSHA256 con
// la generacion y ClaveHMACRef exactas, incluida la vigencia/no revocacion de
// esa clave. Si la clave ha rotado o ha sido revocada, falla con el error
// generico de sellado: nunca recalcula con la clave actual porque cambiaria el
// material ya autorizado. Tambien debe ser idempotente y reconciliable por
// intencion+indice HMAC+atestacion de idempotencia: la misma solicitud devuelve
// la misma atestacion y cualquier colision con otro contenido falla cerrada.
type MaterializadorSelladoMotivoGobiernoConvocatoria interface {
	VerificarYMaterializarSelladoMotivo(
		context.Context,
		SolicitudMaterializarSelladoMotivoGobiernoConvocatoria,
	) (AtestacionSelladoMotivoConvocatoria, error)
}

func textoMotivoGobiernoConvocatoriaValido(valor string) bool {
	if valor == "" || len(valor) > 8000 || valor != strings.TrimSpace(valor) {
		return false
	}
	for _, caracter := range valor {
		if caracter < 32 && caracter != '\n' && caracter != '\r' && caracter != '\t' {
			return false
		}
	}
	return true
}
