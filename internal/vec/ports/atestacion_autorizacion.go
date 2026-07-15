package ports

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrSolicitudFirmaAtestacionInvalida = errors.New("vec: solicitud de firma de atestacion invalida")
	ErrFirmaAtestacionNoDisponible      = errors.New("vec: firma de atestacion no disponible")
	ErrResultadoFirmaAtestacionInvalido = errors.New("vec: resultado de firma de atestacion invalido")
)

const tamanoMaximoFirmaAtestacion = 16 * 1024

// SolicitudFirmaAtestacionAutorizacionV1 es una capacidad inmutable. La
// cabecera se selecciona antes de construir el mensaje y el firmante recibe
// exactamente esos bytes; no puede devolver otra suite, clave o audiencia.
type SolicitudFirmaAtestacionAutorizacionV1 struct {
	cabecera           domain.CabeceraAtestacionAutorizacionV1
	mensaje            []byte
	huellaMensaje      string
	referenciaDecision string
}

func NuevaSolicitudFirmaAtestacionAutorizacionV1(
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	decision domain.DecisionAutorizacion,
) (SolicitudFirmaAtestacionAutorizacionV1, error) {
	mensaje, err := domain.SerializarMensajeAtestacionAutorizacionV1(cabecera, decision)
	if err != nil {
		return SolicitudFirmaAtestacionAutorizacionV1{}, ErrSolicitudFirmaAtestacionInvalida
	}
	suma := sha256.Sum256(mensaje)
	solicitud := SolicitudFirmaAtestacionAutorizacionV1{
		cabecera:           cabecera,
		mensaje:            append([]byte(nil), mensaje...),
		huellaMensaje:      hex.EncodeToString(suma[:]),
		referenciaDecision: decision.DecisionRef,
	}
	if solicitud.Validar() != nil {
		return SolicitudFirmaAtestacionAutorizacionV1{}, ErrSolicitudFirmaAtestacionInvalida
	}
	return solicitud, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV1) Validar() error {
	if s.cabecera.Validar() != nil || !referenciaAtestacionValida(s.referenciaDecision) ||
		len(s.mensaje) < 8 || len(s.mensaje) > domain.TamanoMaximoMensajeAtestacionAutorizacionV1 ||
		!huellaAtestacionValida(s.huellaMensaje) ||
		!mensajeAtestacionContieneCabecera(s.mensaje, s.cabecera) {
		return ErrSolicitudFirmaAtestacionInvalida
	}
	longitud := binary.BigEndian.Uint64(s.mensaje[len(s.mensaje)-8:])
	if longitud != uint64(len(s.mensaje)) {
		return ErrSolicitudFirmaAtestacionInvalida
	}
	suma := sha256.Sum256(s.mensaje)
	esperada, err := hex.DecodeString(s.huellaMensaje)
	if err != nil || subtle.ConstantTimeCompare(suma[:], esperada) != 1 {
		return ErrSolicitudFirmaAtestacionInvalida
	}
	return nil
}

func mensajeAtestacionContieneCabecera(
	mensaje []byte,
	cabecera domain.CabeceraAtestacionAutorizacionV1,
) bool {
	prefijo := append([]byte(domain.EsquemaMensajeAtestacionAutorizacionV1), 0)
	if len(mensaje) < len(prefijo)+2+8 ||
		subtle.ConstantTimeCompare(mensaje[:len(prefijo)], prefijo) != 1 {
		return false
	}
	posicion := len(prefijo)
	if binary.BigEndian.Uint16(mensaje[posicion:posicion+2]) != cabecera.FormatoVersion {
		return false
	}
	posicion += 2
	for _, esperado := range []string{cabecera.Suite, cabecera.ClaveID, cabecera.Audiencia} {
		if posicion > len(mensaje)-4 {
			return false
		}
		longitud := uint64(binary.BigEndian.Uint32(mensaje[posicion : posicion+4]))
		posicion += 4
		if longitud > uint64(len(mensaje)-posicion) {
			return false
		}
		final := posicion + int(longitud)
		if final > len(mensaje)-8 || string(mensaje[posicion:final]) != esperado {
			return false
		}
		posicion = final
	}
	return true
}

func (s SolicitudFirmaAtestacionAutorizacionV1) Cabecera() (domain.CabeceraAtestacionAutorizacionV1, error) {
	if s.Validar() != nil {
		return domain.CabeceraAtestacionAutorizacionV1{}, ErrSolicitudFirmaAtestacionInvalida
	}
	return s.cabecera, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV1) Mensaje() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrSolicitudFirmaAtestacionInvalida
	}
	return append([]byte(nil), s.mensaje...), nil
}

func (s SolicitudFirmaAtestacionAutorizacionV1) HuellaMensajeSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrSolicitudFirmaAtestacionInvalida
	}
	return s.huellaMensaje, nil
}

func (s SolicitudFirmaAtestacionAutorizacionV1) ReferenciaDecision() (string, error) {
	if s.Validar() != nil {
		return "", ErrSolicitudFirmaAtestacionInvalida
	}
	return s.referenciaDecision, nil
}

// ResultadoFirmaAtestacionAutorizacionV1 conserva solo la firma y evidencia
// opaca del proveedor. No puede sustituir la cabecera preseleccionada.
type ResultadoFirmaAtestacionAutorizacionV1 struct {
	firma                 []byte
	huellaMensaje         string
	evidenciaOperacionRef string
	firmadaEn             time.Time
}

// AtestacionAutorizacionV1 agrupa la cabecera preseleccionada y el resultado
// opaco del firmante. No otorga permiso por si sola: registro y consumidor
// deben verificarla y revalidar la decision en su propia transaccion.
type AtestacionAutorizacionV1 struct {
	cabecera  domain.CabeceraAtestacionAutorizacionV1
	resultado ResultadoFirmaAtestacionAutorizacionV1
}

func NuevaAtestacionAutorizacionV1(
	solicitud SolicitudFirmaAtestacionAutorizacionV1,
	resultado ResultadoFirmaAtestacionAutorizacionV1,
) (AtestacionAutorizacionV1, error) {
	if solicitud.Validar() != nil || resultado.ValidarPara(solicitud) != nil {
		return AtestacionAutorizacionV1{}, ErrResultadoFirmaAtestacionInvalido
	}
	cabecera, _ := solicitud.Cabecera()
	atestacion := AtestacionAutorizacionV1{cabecera: cabecera, resultado: resultado}
	if atestacion.ValidarPara(solicitud) != nil {
		return AtestacionAutorizacionV1{}, ErrResultadoFirmaAtestacionInvalido
	}
	return atestacion, nil
}

func (a AtestacionAutorizacionV1) ValidarPara(
	solicitud SolicitudFirmaAtestacionAutorizacionV1,
) error {
	if solicitud.Validar() != nil || a.cabecera.Validar() != nil ||
		a.resultado.ValidarPara(solicitud) != nil {
		return ErrResultadoFirmaAtestacionInvalido
	}
	cabecera, _ := solicitud.Cabecera()
	if a.cabecera != cabecera {
		return ErrResultadoFirmaAtestacionInvalido
	}
	return nil
}

func (a AtestacionAutorizacionV1) Cabecera() (domain.CabeceraAtestacionAutorizacionV1, error) {
	if a.cabecera.Validar() != nil || a.resultado.Validar() != nil {
		return domain.CabeceraAtestacionAutorizacionV1{}, ErrResultadoFirmaAtestacionInvalido
	}
	return a.cabecera, nil
}

func (a AtestacionAutorizacionV1) Resultado() (ResultadoFirmaAtestacionAutorizacionV1, error) {
	if a.cabecera.Validar() != nil || a.resultado.Validar() != nil {
		return ResultadoFirmaAtestacionAutorizacionV1{}, ErrResultadoFirmaAtestacionInvalido
	}
	return a.resultado, nil
}

func NuevoResultadoFirmaAtestacionAutorizacionV1(
	solicitud SolicitudFirmaAtestacionAutorizacionV1,
	firma []byte,
	evidenciaOperacionRef string,
	firmadaEn time.Time,
) (ResultadoFirmaAtestacionAutorizacionV1, error) {
	if solicitud.Validar() != nil {
		return ResultadoFirmaAtestacionAutorizacionV1{}, ErrResultadoFirmaAtestacionInvalido
	}
	huella, _ := solicitud.HuellaMensajeSHA256()
	resultado := ResultadoFirmaAtestacionAutorizacionV1{
		firma:                 append([]byte(nil), firma...),
		huellaMensaje:         huella,
		evidenciaOperacionRef: evidenciaOperacionRef,
		firmadaEn:             firmadaEn,
	}
	if resultado.ValidarPara(solicitud) != nil {
		return ResultadoFirmaAtestacionAutorizacionV1{}, ErrResultadoFirmaAtestacionInvalido
	}
	return resultado, nil
}

func (r ResultadoFirmaAtestacionAutorizacionV1) Validar() error {
	// Esta validacion solo protege el envoltorio generico. No verifica la firma
	// ni aprueba su longitud para una suite: esa frontera debe conocer el perfil,
	// exigir su tamano exacto y ejecutar el verificador criptografico.
	if len(r.firma) == 0 || len(r.firma) > tamanoMaximoFirmaAtestacion ||
		!huellaAtestacionValida(r.huellaMensaje) ||
		!referenciaAtestacionValida(r.evidenciaOperacionRef) ||
		!instanteAtestacionCanonico(r.firmadaEn) {
		return ErrResultadoFirmaAtestacionInvalido
	}
	return nil
}

func (r ResultadoFirmaAtestacionAutorizacionV1) ValidarPara(
	solicitud SolicitudFirmaAtestacionAutorizacionV1,
) error {
	if r.Validar() != nil || solicitud.Validar() != nil {
		return ErrResultadoFirmaAtestacionInvalido
	}
	huella, _ := solicitud.HuellaMensajeSHA256()
	esperada, errEsperada := hex.DecodeString(huella)
	recibida, errRecibida := hex.DecodeString(r.huellaMensaje)
	if errEsperada != nil || errRecibida != nil ||
		subtle.ConstantTimeCompare(esperada, recibida) != 1 {
		return ErrResultadoFirmaAtestacionInvalido
	}
	return nil
}

func (r ResultadoFirmaAtestacionAutorizacionV1) Firma() ([]byte, error) {
	if r.Validar() != nil {
		return nil, ErrResultadoFirmaAtestacionInvalido
	}
	return append([]byte(nil), r.firma...), nil
}

func (r ResultadoFirmaAtestacionAutorizacionV1) HuellaMensajeSHA256() (string, error) {
	if r.Validar() != nil {
		return "", ErrResultadoFirmaAtestacionInvalido
	}
	return r.huellaMensaje, nil
}

func (r ResultadoFirmaAtestacionAutorizacionV1) EvidenciaOperacionRef() (string, error) {
	if r.Validar() != nil {
		return "", ErrResultadoFirmaAtestacionInvalido
	}
	return r.evidenciaOperacionRef, nil
}

func (r ResultadoFirmaAtestacionAutorizacionV1) FirmadaEn() (time.Time, error) {
	if r.Validar() != nil {
		return time.Time{}, ErrResultadoFirmaAtestacionInvalido
	}
	return r.firmadaEn, nil
}

// FirmanteAtestacionesAutorizacionV1 es un puerto de salida. La implementacion
// productiva debe usar identidad exclusiva del PDP y una clave no exportable.
type FirmanteAtestacionesAutorizacionV1 interface {
	FirmarAtestacionAutorizacionV1(
		context.Context,
		SolicitudFirmaAtestacionAutorizacionV1,
	) (ResultadoFirmaAtestacionAutorizacionV1, error)
}

func huellaAtestacionValida(valor string) bool {
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) {
		return false
	}
	contenido, err := hex.DecodeString(valor)
	return err == nil && len(contenido) == sha256.Size
}

func referenciaAtestacionValida(valor string) bool {
	if valor == "" || len(valor) > 512 || valor != strings.TrimSpace(valor) ||
		!utf8.ValidString(valor) || strings.ContainsRune(valor, '*') {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.IsSpace(caracter) {
			return false
		}
	}
	return true
}

func instanteAtestacionCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}
