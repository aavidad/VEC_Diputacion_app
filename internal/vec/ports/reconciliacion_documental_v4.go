package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

var (
	ErrRetoConsultaReconciliacionDocumentalV4Invalido = errors.New(
		"vec: reto de consulta de reconciliacion documental v4 invalido",
	)
	ErrConsultaReconciliacionDocumentalV4Invalida = errors.New(
		"vec: consulta de reconciliacion documental v4 invalida",
	)
	ErrRespuestaCrudaReconciliacionDocumentalV4Invalida = errors.New(
		"vec: respuesta cruda de reconciliacion documental v4 invalida",
	)
	ErrProyeccionIntentoCASReconciliacionDocumentalV4Invalida = errors.New(
		"vec: proyeccion de intento cas de reconciliacion documental v4 invalida",
	)
	ErrSerializacionReconciliacionDocumentalV4 = errors.New(
		"vec: serializacion generica de reconciliacion documental v4 prohibida",
	)
)

const (
	versionProtocoloReconciliacionDocumentalV4 uint16 = 4

	minimoBytesRetoReconciliacionDocumentalV4 = 32
	maximoBytesRetoReconciliacionDocumentalV4 = 64

	// La ventana es deliberadamente corta: EmitidaEn se admite y ExpiraEn es el
	// limite superior exclusivo. La capa interna debe comprobar ademas su propio
	// reloj al verificar COSE; el tiempo declarado por el remoto no da frescura.
	maximaVigenciaConsultaReconciliacionDocumentalV4 = 2 * time.Minute

	formatoInstanteReconciliacionDocumentalV4 = "2006-01-02T15:04:05.000000Z"
)

// RetoConsultaReconciliacionDocumentalV4 es un valor nominal, opaco y sin
// autoridad. Este paquete solo comprueba tamano y forma. Su generacion mediante
// crypto/rand y la prohibicion de reutilizarlo corresponden al servicio de
// confianza alojado en application/internal.
type RetoConsultaReconciliacionDocumentalV4 struct {
	valor            []byte
	huellaRetoSHA256 string
}

func NuevoRetoConsultaReconciliacionDocumentalV4(
	valor []byte,
) (RetoConsultaReconciliacionDocumentalV4, error) {
	reto := RetoConsultaReconciliacionDocumentalV4{
		valor: append([]byte(nil), valor...),
	}
	reto.huellaRetoSHA256 = huellaBytesReconciliacionDocumentalV4(reto.valor)
	if reto.ValidarSintaxis() != nil {
		return RetoConsultaReconciliacionDocumentalV4{},
			ErrRetoConsultaReconciliacionDocumentalV4Invalido
	}
	return reto, nil
}

// ValidarSintaxis no certifica que el reto proceda de un CSPRNG.
func (r RetoConsultaReconciliacionDocumentalV4) ValidarSintaxis() error {
	if len(r.valor) < minimoBytesRetoReconciliacionDocumentalV4 ||
		len(r.valor) > maximoBytesRetoReconciliacionDocumentalV4 ||
		bytesReconciliacionDocumentalV4Nulos(r.valor) ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(r.huellaRetoSHA256) ||
		r.huellaRetoSHA256 != huellaBytesReconciliacionDocumentalV4(r.valor) {
		return ErrRetoConsultaReconciliacionDocumentalV4Invalido
	}
	return nil
}

// BytesParaProtocolo entrega una copia exclusivamente para construir o
// verificar el payload canonico. Los serializadores generales estan cerrados.
func (r RetoConsultaReconciliacionDocumentalV4) BytesParaProtocolo() ([]byte, error) {
	if r.ValidarSintaxis() != nil {
		return nil, ErrRetoConsultaReconciliacionDocumentalV4Invalido
	}
	return append([]byte(nil), r.valor...), nil
}

func (r RetoConsultaReconciliacionDocumentalV4) HuellaSHA256() (string, error) {
	if r.ValidarSintaxis() != nil {
		return "", ErrRetoConsultaReconciliacionDocumentalV4Invalido
	}
	return r.huellaRetoSHA256, nil
}

func (RetoConsultaReconciliacionDocumentalV4) String() string {
	return "[RETO-CONSULTA-RECONCILIACION-DOCUMENTAL-V4-REDACTADO]"
}

func (r RetoConsultaReconciliacionDocumentalV4) GoString() string { return r.String() }
func (r RetoConsultaReconciliacionDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (r RetoConsultaReconciliacionDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (RetoConsultaReconciliacionDocumentalV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionReconciliacionDocumentalV4
}
func (*RetoConsultaReconciliacionDocumentalV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionReconciliacionDocumentalV4
}
func (RetoConsultaReconciliacionDocumentalV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionReconciliacionDocumentalV4
}
func (*RetoConsultaReconciliacionDocumentalV4) UnmarshalText([]byte) error {
	return ErrSerializacionReconciliacionDocumentalV4
}
func (RetoConsultaReconciliacionDocumentalV4) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionReconciliacionDocumentalV4
}
func (*RetoConsultaReconciliacionDocumentalV4) UnmarshalBinary([]byte) error {
	return ErrSerializacionReconciliacionDocumentalV4
}

// DatosConsultaReconciliacionDocumentalV4 son datos declarativos. La consulta
// resultante sigue sin acreditar frescura, origen servidor ni autorizacion: el
// servicio interno debe generarla y persistir sus claves UNIQUE.
type DatosConsultaReconciliacionDocumentalV4 struct {
	ConsultaRef              string
	Reto                     RetoConsultaReconciliacionDocumentalV4
	ReservaRef               string
	EfectoRef                string
	HuellaPlanSHA256         string
	EstadoEsperado           EstadoEjecucionDocumentalV3
	VersionEsperada          uint64
	SecuenciaCercadoEsperada uint64
	EmitidaEn                time.Time
	ExpiraEn                 time.Time
}

// ConsultaReconciliacionDocumentalV4 conserva una instantanea defensiva y el
// compromiso del payload exacto que debera firmar el sistema consultado.
type ConsultaReconciliacionDocumentalV4 struct {
	datos               DatosConsultaReconciliacionDocumentalV4
	huellaMensajeSHA256 string
}

func NuevaConsultaReconciliacionDocumentalV4(
	datos DatosConsultaReconciliacionDocumentalV4,
) (ConsultaReconciliacionDocumentalV4, error) {
	datos = clonarDatosConsultaReconciliacionDocumentalV4(datos)
	if validarDatosConsultaReconciliacionDocumentalV4(datos) != nil {
		return ConsultaReconciliacionDocumentalV4{},
			ErrConsultaReconciliacionDocumentalV4Invalida
	}
	consulta := ConsultaReconciliacionDocumentalV4{datos: datos}
	consulta.huellaMensajeSHA256 = huellaBytesReconciliacionDocumentalV4(
		consulta.mensajeCanonicoSinValidar(),
	)
	if consulta.ValidarSintaxis() != nil {
		return ConsultaReconciliacionDocumentalV4{},
			ErrConsultaReconciliacionDocumentalV4Invalida
	}
	return consulta, nil
}

// ValidarSintaxis comprueba invariantes, no la aleatoriedad ni la unicidad
// durable de ConsultaRef y reto.
func (c ConsultaReconciliacionDocumentalV4) ValidarSintaxis() error {
	if validarDatosConsultaReconciliacionDocumentalV4(c.datos) != nil ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(c.huellaMensajeSHA256) ||
		c.huellaMensajeSHA256 != huellaBytesReconciliacionDocumentalV4(
			c.mensajeCanonicoSinValidar(),
		) {
		return ErrConsultaReconciliacionDocumentalV4Invalida
	}
	return nil
}

func (c ConsultaReconciliacionDocumentalV4) Datos() (
	DatosConsultaReconciliacionDocumentalV4,
	error,
) {
	if c.ValidarSintaxis() != nil {
		return DatosConsultaReconciliacionDocumentalV4{},
			ErrConsultaReconciliacionDocumentalV4Invalida
	}
	return clonarDatosConsultaReconciliacionDocumentalV4(c.datos), nil
}

// MensajeCanonico es la unica salida binaria intencionada. Todos sus campos,
// incluida la version, llevan longitud big-endian para impedir ambiguedades de
// concatenacion y cambios de representacion.
func (c ConsultaReconciliacionDocumentalV4) MensajeCanonico() ([]byte, error) {
	if c.ValidarSintaxis() != nil {
		return nil, ErrConsultaReconciliacionDocumentalV4Invalida
	}
	return append([]byte(nil), c.mensajeCanonicoSinValidar()...), nil
}

func (c ConsultaReconciliacionDocumentalV4) HuellaMensajeSHA256() (string, error) {
	if c.ValidarSintaxis() != nil {
		return "", ErrConsultaReconciliacionDocumentalV4Invalida
	}
	return c.huellaMensajeSHA256, nil
}

func (c ConsultaReconciliacionDocumentalV4) ClaveConsumoUnico() (
	ClaveConsumoConsultaReconciliacionDocumentalV4,
	error,
) {
	if c.ValidarSintaxis() != nil {
		return ClaveConsumoConsultaReconciliacionDocumentalV4{},
			ErrConsultaReconciliacionDocumentalV4Invalida
	}
	huellaReto, _ := c.datos.Reto.HuellaSHA256()
	clave := ClaveConsumoConsultaReconciliacionDocumentalV4{
		ConsultaRef:          c.datos.ConsultaRef,
		HuellaRetoSHA256:     huellaReto,
		HuellaConsultaSHA256: c.huellaMensajeSHA256,
	}
	if clave.ValidarContra(c) != nil {
		return ClaveConsumoConsultaReconciliacionDocumentalV4{},
			ErrConsultaReconciliacionDocumentalV4Invalida
	}
	return clave, nil
}

func (c ConsultaReconciliacionDocumentalV4) mensajeCanonicoSinValidar() []byte {
	d := c.datos
	mensaje := make([]byte, 0, 512+len(d.Reto.valor))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, []byte("vec.documentos.consulta-reconciliacion"),
	)
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, bytesUint16ReconciliacionDocumentalV4(versionProtocoloReconciliacionDocumentalV4),
	)
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, []byte(d.ConsultaRef))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, d.Reto.valor)
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, []byte(d.ReservaRef))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, []byte(d.EfectoRef))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, []byte(d.HuellaPlanSHA256))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, []byte(d.EstadoEsperado))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, bytesUint64ReconciliacionDocumentalV4(d.VersionEsperada),
	)
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, bytesUint64ReconciliacionDocumentalV4(d.SecuenciaCercadoEsperada),
	)
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, []byte(d.EmitidaEn.Format(formatoInstanteReconciliacionDocumentalV4)),
	)
	return anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, []byte(d.ExpiraEn.Format(formatoInstanteReconciliacionDocumentalV4)),
	)
}

func (ConsultaReconciliacionDocumentalV4) String() string {
	return "[CONSULTA-RECONCILIACION-DOCUMENTAL-V4-REDACTADA]"
}
func (c ConsultaReconciliacionDocumentalV4) GoString() string { return c.String() }
func (c ConsultaReconciliacionDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(c.String())
}
func (c ConsultaReconciliacionDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (ConsultaReconciliacionDocumentalV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionReconciliacionDocumentalV4
}
func (*ConsultaReconciliacionDocumentalV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionReconciliacionDocumentalV4
}
func (ConsultaReconciliacionDocumentalV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionReconciliacionDocumentalV4
}
func (*ConsultaReconciliacionDocumentalV4) UnmarshalText([]byte) error {
	return ErrSerializacionReconciliacionDocumentalV4
}
func (ConsultaReconciliacionDocumentalV4) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionReconciliacionDocumentalV4
}
func (*ConsultaReconciliacionDocumentalV4) UnmarshalBinary([]byte) error {
	return ErrSerializacionReconciliacionDocumentalV4
}

// ClaveConsumoConsultaReconciliacionDocumentalV4 es comparable y apta para
// indices UNIQUE. El registro durable debe imponer unicidad independiente
// tanto sobre ConsultaRef como sobre HuellaRetoSHA256, ademas de conservar la
// huella completa de consulta. La estructura no concede autoridad.
type ClaveConsumoConsultaReconciliacionDocumentalV4 struct {
	ConsultaRef          string
	HuellaRetoSHA256     string
	HuellaConsultaSHA256 string
}

func (c ClaveConsumoConsultaReconciliacionDocumentalV4) ValidarContra(
	consulta ConsultaReconciliacionDocumentalV4,
) error {
	if consulta.ValidarSintaxis() != nil ||
		!referenciaConsultaServidorReconciliacionDocumentalV4Valida(c.ConsultaRef) ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(c.HuellaRetoSHA256) ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(c.HuellaConsultaSHA256) {
		return ErrConsultaReconciliacionDocumentalV4Invalida
	}
	huellaReto, _ := consulta.datos.Reto.HuellaSHA256()
	if c.ConsultaRef != consulta.datos.ConsultaRef ||
		c.HuellaRetoSHA256 != huellaReto ||
		c.HuellaConsultaSHA256 != consulta.huellaMensajeSHA256 {
		return ErrConsultaReconciliacionDocumentalV4Invalida
	}
	return nil
}

type EstadoResultadoReconciliacionDocumentalV4 string

const (
	EstadoResultadoReconciliacionDocumentalV4Aplicado    EstadoResultadoReconciliacionDocumentalV4 = "aplicado"
	EstadoResultadoReconciliacionDocumentalV4NoAplicado  EstadoResultadoReconciliacionDocumentalV4 = "no_aplicado"
	EstadoResultadoReconciliacionDocumentalV4Desconocido EstadoResultadoReconciliacionDocumentalV4 = "desconocido"
)

func (e EstadoResultadoReconciliacionDocumentalV4) Valido() bool {
	return e == EstadoResultadoReconciliacionDocumentalV4Aplicado ||
		e == EstadoResultadoReconciliacionDocumentalV4NoAplicado ||
		e == EstadoResultadoReconciliacionDocumentalV4Desconocido
}

// DeclaracionRespuestaReconciliacionDocumentalV4 refleja lo que afirma el
// componente remoto. Sigue siendo entrada hostil hasta que application/internal
// verifique el COSE, la clave, el algoritmo, la audiencia y este payload exacto.
type DeclaracionRespuestaReconciliacionDocumentalV4 struct {
	ConsultaRef          string
	HuellaConsultaSHA256 string
	HuellaRetoSHA256     string
	ReservaRef           string
	EfectoRef            string
	HuellaPlanSHA256     string
	// Estos tres valores son un eco firmado de la condicion CAS enviada por
	// el servidor. No representan una lectura remota ni demuestran por si solos
	// el estado local o la existencia/ausencia del efecto.
	EstadoReservaEsperadoEco    EstadoEjecucionDocumentalV3
	VersionReservaEsperadaEco   uint64
	SecuenciaCercadoEsperadaEco uint64
	Resultado                   EstadoResultadoReconciliacionDocumentalV4
	HuellaEfectoAplicadoSHA256  string
	TamanoEfectoAplicado        uint64
	RespondidaEn                time.Time
}

type RespuestaCrudaReconciliacionDocumentalV4 struct {
	consulta            ConsultaReconciliacionDocumentalV4
	declaracion         DeclaracionRespuestaReconciliacionDocumentalV4
	atestacion          AtestacionCrudaReconciliacionDocumentalV4
	mensaje             []byte
	huellaMensajeSHA256 string
}

func NuevaRespuestaCrudaReconciliacionDocumentalV4(
	consulta ConsultaReconciliacionDocumentalV4,
	declaracion DeclaracionRespuestaReconciliacionDocumentalV4,
	atestacion AtestacionCrudaReconciliacionDocumentalV4,
) (RespuestaCrudaReconciliacionDocumentalV4, error) {
	if consulta.ValidarSintaxis() != nil ||
		validarDeclaracionRespuestaReconciliacionDocumentalV4(declaracion, consulta) != nil ||
		atestacion.ValidarSintaxis() != nil {
		return RespuestaCrudaReconciliacionDocumentalV4{},
			ErrRespuestaCrudaReconciliacionDocumentalV4Invalida
	}
	consulta = clonarConsultaReconciliacionDocumentalV4(consulta)
	respuesta := RespuestaCrudaReconciliacionDocumentalV4{
		consulta: consulta, declaracion: declaracion, atestacion: atestacion,
	}
	respuesta.mensaje = respuesta.mensajeCanonicoSinValidar()
	respuesta.huellaMensajeSHA256 = huellaBytesReconciliacionDocumentalV4(respuesta.mensaje)
	if respuesta.ValidarSintaxisContra(consulta) != nil {
		return RespuestaCrudaReconciliacionDocumentalV4{},
			ErrRespuestaCrudaReconciliacionDocumentalV4Invalida
	}
	return respuesta, nil
}

// ValidarSintaxisContra no verifica COSE ni convierte la declaracion en hecho.
func (r RespuestaCrudaReconciliacionDocumentalV4) ValidarSintaxisContra(
	consulta ConsultaReconciliacionDocumentalV4,
) error {
	if consulta.ValidarSintaxis() != nil || r.consulta.ValidarSintaxis() != nil ||
		r.atestacion.ValidarSintaxis() != nil ||
		!consultasReconciliacionDocumentalV4Iguales(r.consulta, consulta) ||
		validarDeclaracionRespuestaReconciliacionDocumentalV4(
			r.declaracion, consulta,
		) != nil || len(r.mensaje) == 0 ||
		!bytes.Equal(r.mensaje, r.mensajeCanonicoSinValidar()) ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(r.huellaMensajeSHA256) ||
		r.huellaMensajeSHA256 != huellaBytesReconciliacionDocumentalV4(r.mensaje) {
		return ErrRespuestaCrudaReconciliacionDocumentalV4Invalida
	}
	return nil
}

func (r RespuestaCrudaReconciliacionDocumentalV4) Declaracion() (
	DeclaracionRespuestaReconciliacionDocumentalV4,
	error,
) {
	if r.ValidarSintaxisContra(r.consulta) != nil {
		return DeclaracionRespuestaReconciliacionDocumentalV4{},
			ErrRespuestaCrudaReconciliacionDocumentalV4Invalida
	}
	return r.declaracion, nil
}

func (r RespuestaCrudaReconciliacionDocumentalV4) MensajeCanonico() ([]byte, error) {
	if r.ValidarSintaxisContra(r.consulta) != nil {
		return nil, ErrRespuestaCrudaReconciliacionDocumentalV4Invalida
	}
	return append([]byte(nil), r.mensaje...), nil
}

func (r RespuestaCrudaReconciliacionDocumentalV4) HuellaMensajeSHA256() (string, error) {
	if r.ValidarSintaxisContra(r.consulta) != nil {
		return "", ErrRespuestaCrudaReconciliacionDocumentalV4Invalida
	}
	return r.huellaMensajeSHA256, nil
}

func (r RespuestaCrudaReconciliacionDocumentalV4) AtestacionCruda() (
	AtestacionCrudaReconciliacionDocumentalV4,
	error,
) {
	if r.ValidarSintaxisContra(r.consulta) != nil {
		return AtestacionCrudaReconciliacionDocumentalV4{},
			ErrRespuestaCrudaReconciliacionDocumentalV4Invalida
	}
	return r.atestacion, nil
}

func (r RespuestaCrudaReconciliacionDocumentalV4) mensajeCanonicoSinValidar() []byte {
	d := r.declaracion
	mensaje := make([]byte, 0, 640+len(r.consulta.datos.Reto.valor))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, []byte("vec.documentos.respuesta-reconciliacion"),
	)
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, bytesUint16ReconciliacionDocumentalV4(versionProtocoloReconciliacionDocumentalV4),
	)
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, []byte(d.ConsultaRef))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, []byte(d.HuellaConsultaSHA256))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, r.consulta.datos.Reto.valor)
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, []byte(d.HuellaRetoSHA256))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, []byte(d.ReservaRef))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, []byte(d.EfectoRef))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, []byte(d.HuellaPlanSHA256))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, []byte(d.EstadoReservaEsperadoEco),
	)
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, bytesUint64ReconciliacionDocumentalV4(d.VersionReservaEsperadaEco),
	)
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, bytesUint64ReconciliacionDocumentalV4(d.SecuenciaCercadoEsperadaEco),
	)
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(mensaje, []byte(d.Resultado))
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, []byte(d.HuellaEfectoAplicadoSHA256),
	)
	mensaje = anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, bytesUint64ReconciliacionDocumentalV4(d.TamanoEfectoAplicado),
	)
	return anexarCampoCanonicoReconciliacionDocumentalV4(
		mensaje, []byte(d.RespondidaEn.Format(formatoInstanteReconciliacionDocumentalV4)),
	)
}

func (RespuestaCrudaReconciliacionDocumentalV4) String() string {
	return "[RESPUESTA-CRUDA-RECONCILIACION-DOCUMENTAL-V4-REDACTADA]"
}
func (r RespuestaCrudaReconciliacionDocumentalV4) GoString() string { return r.String() }
func (r RespuestaCrudaReconciliacionDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (r RespuestaCrudaReconciliacionDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (RespuestaCrudaReconciliacionDocumentalV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionReconciliacionDocumentalV4
}
func (*RespuestaCrudaReconciliacionDocumentalV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionReconciliacionDocumentalV4
}
func (RespuestaCrudaReconciliacionDocumentalV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionReconciliacionDocumentalV4
}
func (*RespuestaCrudaReconciliacionDocumentalV4) UnmarshalText([]byte) error {
	return ErrSerializacionReconciliacionDocumentalV4
}
func (RespuestaCrudaReconciliacionDocumentalV4) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionReconciliacionDocumentalV4
}
func (*RespuestaCrudaReconciliacionDocumentalV4) UnmarshalBinary([]byte) error {
	return ErrSerializacionReconciliacionDocumentalV4
}

type AccionProyectadaReconciliacionDocumentalV4 string

const (
	// Una firma autentica al declarante, pero la respuesta V4 no contiene un
	// resultado remoto completo ni una prueba de inexistencia. Por ello ningun
	// estado permite confirmar o abandonar: los tres solo proyectan evidencia.
	AccionReconciliacionDocumentalV4RegistrarSoloEvidencia AccionProyectadaReconciliacionDocumentalV4 = "registrar_solo_evidencia"
)

func (a AccionProyectadaReconciliacionDocumentalV4) Valida() bool {
	return a == AccionReconciliacionDocumentalV4RegistrarSoloEvidencia
}

type CondicionCASReconciliacionDocumentalV4 struct {
	EstadoEsperado           EstadoEjecucionDocumentalV3
	VersionEsperada          uint64
	SecuenciaCercadoEsperada uint64
}

func (c CondicionCASReconciliacionDocumentalV4) ValidarContra(
	consulta ConsultaReconciliacionDocumentalV4,
) error {
	if consulta.ValidarSintaxis() != nil ||
		c.EstadoEsperado != EstadoEjecucionDocumentalV3Indeterminada ||
		c.EstadoEsperado != consulta.datos.EstadoEsperado ||
		c.VersionEsperada == 0 || c.VersionEsperada != consulta.datos.VersionEsperada ||
		c.SecuenciaCercadoEsperada == 0 ||
		c.SecuenciaCercadoEsperada != consulta.datos.SecuenciaCercadoEsperada {
		return ErrProyeccionIntentoCASReconciliacionDocumentalV4Invalida
	}
	return nil
}

// ProyeccionIntentoCASReconciliacionDocumentalV4 describe el registro de
// evidencia que podria solicitarse despues de una verificacion criptografica
// fresca. V4 no proyecta ninguna transicion de estado. Nunca es una capacidad
// ni debe aceptarse directamente por un repositorio.
//
// Queda pendiente el puerto durable que, en una sola transaccion y antes de
// COMMIT, debera consumir con UNIQUE ConsultaRef y HuellaRetoSHA256, comprobar
// estado='indeterminada' AND version=N AND cercado=S y anexar evidencia sin
// confirmar ni abandonar. Una futura version solo podra mutar estado con un
// resultado remoto completo o una prueba fuerte de inexistencia. Este tipo no
// simula esas garantias.
type ProyeccionIntentoCASReconciliacionDocumentalV4 struct {
	huellaConsultaSHA256  string
	huellaRespuestaSHA256 string
	condicion             CondicionCASReconciliacionDocumentalV4
	claveConsumo          ClaveConsumoConsultaReconciliacionDocumentalV4
	accion                AccionProyectadaReconciliacionDocumentalV4
}

func ProyectarIntentoCASReconciliacionDocumentalV4(
	consulta ConsultaReconciliacionDocumentalV4,
	respuesta RespuestaCrudaReconciliacionDocumentalV4,
) (ProyeccionIntentoCASReconciliacionDocumentalV4, error) {
	if consulta.ValidarSintaxis() != nil ||
		respuesta.ValidarSintaxisContra(consulta) != nil {
		return ProyeccionIntentoCASReconciliacionDocumentalV4{},
			ErrProyeccionIntentoCASReconciliacionDocumentalV4Invalida
	}
	clave, _ := consulta.ClaveConsumoUnico()
	accion := accionParaResultadoReconciliacionDocumentalV4(respuesta.declaracion.Resultado)
	proyeccion := ProyeccionIntentoCASReconciliacionDocumentalV4{
		huellaConsultaSHA256:  consulta.huellaMensajeSHA256,
		huellaRespuestaSHA256: respuesta.huellaMensajeSHA256,
		condicion: CondicionCASReconciliacionDocumentalV4{
			EstadoEsperado:           consulta.datos.EstadoEsperado,
			VersionEsperada:          consulta.datos.VersionEsperada,
			SecuenciaCercadoEsperada: consulta.datos.SecuenciaCercadoEsperada,
		},
		claveConsumo: clave,
		accion:       accion,
	}
	if proyeccion.ValidarContra(consulta, respuesta) != nil {
		return ProyeccionIntentoCASReconciliacionDocumentalV4{},
			ErrProyeccionIntentoCASReconciliacionDocumentalV4Invalida
	}
	return proyeccion, nil
}

func (p ProyeccionIntentoCASReconciliacionDocumentalV4) ValidarContra(
	consulta ConsultaReconciliacionDocumentalV4,
	respuesta RespuestaCrudaReconciliacionDocumentalV4,
) error {
	if consulta.ValidarSintaxis() != nil || respuesta.ValidarSintaxisContra(consulta) != nil ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(p.huellaConsultaSHA256) ||
		p.huellaConsultaSHA256 != consulta.huellaMensajeSHA256 ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(p.huellaRespuestaSHA256) ||
		p.huellaRespuestaSHA256 != respuesta.huellaMensajeSHA256 ||
		p.condicion.ValidarContra(consulta) != nil ||
		p.claveConsumo.ValidarContra(consulta) != nil ||
		!p.accion.Valida() ||
		p.accion != accionParaResultadoReconciliacionDocumentalV4(
			respuesta.declaracion.Resultado,
		) {
		return ErrProyeccionIntentoCASReconciliacionDocumentalV4Invalida
	}
	return nil
}

func (p ProyeccionIntentoCASReconciliacionDocumentalV4) CondicionCAS() (
	CondicionCASReconciliacionDocumentalV4,
	error,
) {
	if !p.accion.Valida() ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(p.huellaConsultaSHA256) ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(p.huellaRespuestaSHA256) {
		return CondicionCASReconciliacionDocumentalV4{},
			ErrProyeccionIntentoCASReconciliacionDocumentalV4Invalida
	}
	return p.condicion, nil
}

func (p ProyeccionIntentoCASReconciliacionDocumentalV4) ClaveConsumoUnico() (
	ClaveConsumoConsultaReconciliacionDocumentalV4,
	error,
) {
	if !p.accion.Valida() ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(p.huellaConsultaSHA256) ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(p.huellaRespuestaSHA256) {
		return ClaveConsumoConsultaReconciliacionDocumentalV4{},
			ErrProyeccionIntentoCASReconciliacionDocumentalV4Invalida
	}
	return p.claveConsumo, nil
}

func (p ProyeccionIntentoCASReconciliacionDocumentalV4) AccionProyectada() (
	AccionProyectadaReconciliacionDocumentalV4,
	error,
) {
	if !p.accion.Valida() ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(p.huellaConsultaSHA256) ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(p.huellaRespuestaSHA256) {
		return "", ErrProyeccionIntentoCASReconciliacionDocumentalV4Invalida
	}
	return p.accion, nil
}

// RequiereVerificacionCriptograficaFresca es siempre cierto: ni siquiera una
// respuesta "aplicado" puede mutar estado a partir de esta proyeccion cruda.
func (p ProyeccionIntentoCASReconciliacionDocumentalV4) RequiereVerificacionCriptograficaFresca() bool {
	return p.accion.Valida()
}

type ResultadoIntentoCASReconciliacionDocumentalV4 string

const (
	ResultadoIntentoCASReconciliacionDocumentalV4Aplicado  ResultadoIntentoCASReconciliacionDocumentalV4 = "aplicado"
	ResultadoIntentoCASReconciliacionDocumentalV4Conflicto ResultadoIntentoCASReconciliacionDocumentalV4 = "conflicto"
)

func (r ResultadoIntentoCASReconciliacionDocumentalV4) Valido() bool {
	return r == ResultadoIntentoCASReconciliacionDocumentalV4Aplicado ||
		r == ResultadoIntentoCASReconciliacionDocumentalV4Conflicto
}

// SoloPermiteRegistrarEvidencia es cierto para todo resultado reconocido: que
// el registro haya aplicado su operacion o detectado conflicto no convierte la
// afirmacion remota en permiso para confirmar o abandonar el efecto.
func (r ResultadoIntentoCASReconciliacionDocumentalV4) SoloPermiteRegistrarEvidencia() bool {
	return r.Valido()
}

func validarDeclaracionRespuestaReconciliacionDocumentalV4(
	d DeclaracionRespuestaReconciliacionDocumentalV4,
	consulta ConsultaReconciliacionDocumentalV4,
) error {
	if consulta.ValidarSintaxis() != nil {
		return ErrRespuestaCrudaReconciliacionDocumentalV4Invalida
	}
	huellaReto, _ := consulta.datos.Reto.HuellaSHA256()
	if d.ConsultaRef != consulta.datos.ConsultaRef ||
		d.HuellaConsultaSHA256 != consulta.huellaMensajeSHA256 ||
		d.HuellaRetoSHA256 != huellaReto ||
		d.ReservaRef != consulta.datos.ReservaRef ||
		d.EfectoRef != consulta.datos.EfectoRef ||
		d.HuellaPlanSHA256 != consulta.datos.HuellaPlanSHA256 ||
		d.EstadoReservaEsperadoEco != EstadoEjecucionDocumentalV3Indeterminada ||
		d.EstadoReservaEsperadoEco != consulta.datos.EstadoEsperado ||
		d.VersionReservaEsperadaEco == 0 ||
		d.VersionReservaEsperadaEco != consulta.datos.VersionEsperada ||
		d.SecuenciaCercadoEsperadaEco == 0 ||
		d.SecuenciaCercadoEsperadaEco != consulta.datos.SecuenciaCercadoEsperada ||
		!d.Resultado.Valido() ||
		!instanteReconciliacionDocumentalV4Valido(d.RespondidaEn) ||
		d.RespondidaEn.Before(consulta.datos.EmitidaEn) ||
		!d.RespondidaEn.Before(consulta.datos.ExpiraEn) {
		return ErrRespuestaCrudaReconciliacionDocumentalV4Invalida
	}
	if d.Resultado == EstadoResultadoReconciliacionDocumentalV4Aplicado {
		if !huellaSHA256ReconciliacionDocumentalV4Valida(d.HuellaEfectoAplicadoSHA256) ||
			d.TamanoEfectoAplicado == 0 {
			return ErrRespuestaCrudaReconciliacionDocumentalV4Invalida
		}
		return nil
	}
	if d.HuellaEfectoAplicadoSHA256 != "" || d.TamanoEfectoAplicado != 0 {
		return ErrRespuestaCrudaReconciliacionDocumentalV4Invalida
	}
	return nil
}

func validarDatosConsultaReconciliacionDocumentalV4(
	d DatosConsultaReconciliacionDocumentalV4,
) error {
	if !referenciaConsultaServidorReconciliacionDocumentalV4Valida(d.ConsultaRef) ||
		d.Reto.ValidarSintaxis() != nil ||
		!referenciaReconciliacionDocumentalV4Valida(d.ReservaRef) ||
		!referenciaReconciliacionDocumentalV4Valida(d.EfectoRef) ||
		d.ConsultaRef == d.ReservaRef || d.ConsultaRef == d.EfectoRef ||
		d.ReservaRef == d.EfectoRef ||
		!huellaSHA256ReconciliacionDocumentalV4Valida(d.HuellaPlanSHA256) ||
		d.EstadoEsperado != EstadoEjecucionDocumentalV3Indeterminada ||
		d.VersionEsperada == 0 || d.SecuenciaCercadoEsperada == 0 ||
		!ventanaConsultaReconciliacionDocumentalV4Valida(d.EmitidaEn, d.ExpiraEn) {
		return ErrConsultaReconciliacionDocumentalV4Invalida
	}
	return nil
}

func accionParaResultadoReconciliacionDocumentalV4(
	resultado EstadoResultadoReconciliacionDocumentalV4,
) AccionProyectadaReconciliacionDocumentalV4 {
	switch resultado {
	case EstadoResultadoReconciliacionDocumentalV4Aplicado,
		EstadoResultadoReconciliacionDocumentalV4NoAplicado,
		EstadoResultadoReconciliacionDocumentalV4Desconocido:
		return AccionReconciliacionDocumentalV4RegistrarSoloEvidencia
	default:
		return ""
	}
}

func clonarDatosConsultaReconciliacionDocumentalV4(
	d DatosConsultaReconciliacionDocumentalV4,
) DatosConsultaReconciliacionDocumentalV4 {
	d.Reto.valor = append([]byte(nil), d.Reto.valor...)
	return d
}

func clonarConsultaReconciliacionDocumentalV4(
	c ConsultaReconciliacionDocumentalV4,
) ConsultaReconciliacionDocumentalV4 {
	c.datos = clonarDatosConsultaReconciliacionDocumentalV4(c.datos)
	return c
}

func consultasReconciliacionDocumentalV4Iguales(
	primera, segunda ConsultaReconciliacionDocumentalV4,
) bool {
	a := primera.datos
	b := segunda.datos
	return primera.huellaMensajeSHA256 == segunda.huellaMensajeSHA256 &&
		a.ConsultaRef == b.ConsultaRef && bytes.Equal(a.Reto.valor, b.Reto.valor) &&
		a.Reto.huellaRetoSHA256 == b.Reto.huellaRetoSHA256 &&
		a.ReservaRef == b.ReservaRef && a.EfectoRef == b.EfectoRef &&
		a.HuellaPlanSHA256 == b.HuellaPlanSHA256 &&
		a.EstadoEsperado == b.EstadoEsperado &&
		a.VersionEsperada == b.VersionEsperada &&
		a.SecuenciaCercadoEsperada == b.SecuenciaCercadoEsperada &&
		a.EmitidaEn.Equal(b.EmitidaEn) && a.ExpiraEn.Equal(b.ExpiraEn)
}

func ventanaConsultaReconciliacionDocumentalV4Valida(emitidaEn, expiraEn time.Time) bool {
	return instanteReconciliacionDocumentalV4Valido(emitidaEn) &&
		instanteReconciliacionDocumentalV4Valido(expiraEn) &&
		expiraEn.After(emitidaEn) &&
		expiraEn.Sub(emitidaEn) <= maximaVigenciaConsultaReconciliacionDocumentalV4
}

func instanteReconciliacionDocumentalV4Valido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func referenciaConsultaServidorReconciliacionDocumentalV4Valida(valor string) bool {
	const prefijo = "consulta:reconciliacion:v4:"
	if !strings.HasPrefix(valor, prefijo) ||
		!referenciaReconciliacionDocumentalV4Valida(valor) {
		return false
	}
	sufijo := strings.TrimPrefix(valor, prefijo)
	if len(sufijo) < 32 || len(sufijo) > 64 {
		return false
	}
	for _, caracter := range sufijo {
		if !((caracter >= '0' && caracter <= '9') ||
			(caracter >= 'a' && caracter <= 'f')) {
			return false
		}
	}
	return true
}

func referenciaReconciliacionDocumentalV4Valida(valor string) bool {
	if !referenciaEjecucionDocumentalV3Valida(valor) {
		return false
	}
	for _, fragmento := range strings.FieldsFunc(strings.ToLower(valor), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	}) {
		if fragmentoPareceDNINIEReconciliacionDocumentalV4(fragmento) {
			return false
		}
	}
	return true
}

func fragmentoPareceDNINIEReconciliacionDocumentalV4(valor string) bool {
	if len(valor) != 9 {
		return false
	}
	if valor[0] == 'x' || valor[0] == 'y' || valor[0] == 'z' {
		for indice := 1; indice < 8; indice++ {
			if valor[indice] < '0' || valor[indice] > '9' {
				return false
			}
		}
		return valor[8] >= 'a' && valor[8] <= 'z'
	}
	for indice := 0; indice < 8; indice++ {
		if valor[indice] < '0' || valor[indice] > '9' {
			return false
		}
	}
	return valor[8] >= 'a' && valor[8] <= 'z'
}

func anexarCampoCanonicoReconciliacionDocumentalV4(destino, campo []byte) []byte {
	longitud := make([]byte, 4)
	binary.BigEndian.PutUint32(longitud, uint32(len(campo)))
	destino = append(destino, longitud...)
	return append(destino, campo...)
}

func bytesUint16ReconciliacionDocumentalV4(valor uint16) []byte {
	resultado := make([]byte, 2)
	binary.BigEndian.PutUint16(resultado, valor)
	return resultado
}

func bytesUint64ReconciliacionDocumentalV4(valor uint64) []byte {
	resultado := make([]byte, 8)
	binary.BigEndian.PutUint64(resultado, valor)
	return resultado
}

func huellaBytesReconciliacionDocumentalV4(valor []byte) string {
	huella := sha256.Sum256(valor)
	return hex.EncodeToString(huella[:])
}

func huellaSHA256ReconciliacionDocumentalV4Valida(valor string) bool {
	return esSHA256Hexadecimal(valor) && strings.Trim(valor, "0") != ""
}

func bytesReconciliacionDocumentalV4Nulos(valor []byte) bool {
	if len(valor) == 0 {
		return true
	}
	for _, octeto := range valor {
		if octeto != 0 {
			return false
		}
	}
	return true
}
