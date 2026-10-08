package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

var (
	ErrOrdenFronteraIdentidadTecnicaInvalida  = errors.New("frontera tecnica de identidad: orden invalida")
	ErrAcuseFronteraIdentidadTecnicaInvalido  = errors.New("frontera tecnica de identidad: acuse invalido")
	ErrFronteraIdentidadTecnicaNoDisponible   = errors.New("frontera tecnica de identidad no disponible")
	ErrFronteraIdentidadTecnicaCommitIncierto = errors.New("frontera tecnica de identidad: confirmacion incierta")
)

const (
	MetodoInicioSesionGET              = "GET"
	MetodoInicioSesionPOST             = "POST"
	RutaSesionActual                   = "/api/vec/session"
	RutaInicioSesion                   = "/api/vec/session/start"
	SuperficieFronteraIdentidadTecnica = "interna_corporativa"
)

type MotivoFronteraIdentidadTecnica string

const (
	MotivoCertificadoRequerido   MotivoFronteraIdentidadTecnica = "certificado_requerido"
	MotivoAutenticacionRequerida MotivoFronteraIdentidadTecnica = "autenticacion_requerida"
	MotivoAccesoDenegado         MotivoFronteraIdentidadTecnica = "acceso_denegado"
	MotivoMetodoNoPermitido      MotivoFronteraIdentidadTecnica = "metodo_no_permitido"
	MotivoRecursoNoEncontrado    MotivoFronteraIdentidadTecnica = "recurso_no_encontrado"
	MotivoSolicitudInvalida      MotivoFronteraIdentidadTecnica = "solicitud_invalida"
	MotivoServicioNoDisponible   MotivoFronteraIdentidadTecnica = "servicio_no_disponible"
	MotivoRespuestaIncompatible  MotivoFronteraIdentidadTecnica = "respuesta_incompatible"
)

// OrdenFronteraIdentidadTecnica nace en el servidor con correlación CSPRNG.
// Carece de actor, perfil, decisión, certificado, cuenta, sujeto y payload HTTP.
// Su valor cero no puede autorizar ni registrar una petición.
type OrdenFronteraIdentidadTecnica struct {
	metodoEsperado, ruta, correlacionRef string
	motivo                               MotivoFronteraIdentidadTecnica
}

type DatosOrdenFronteraIdentidadTecnica struct {
	MetodoEsperado string
	Ruta           string
	Motivo         MotivoFronteraIdentidadTecnica
	Resultado      string
	CorrelacionRef string
}

func NuevaOrdenFronteraIdentidadTecnica(
	metodoEsperado, ruta string, motivo MotivoFronteraIdentidadTecnica,
) (OrdenFronteraIdentidadTecnica, error) {
	if !rutaMetodoInicioSesionValido(metodoEsperado, ruta) || resultadoMotivoFronteraIdentidad(motivo) == "" {
		return OrdenFronteraIdentidadTecnica{}, ErrOrdenFronteraIdentidadTecnicaInvalida
	}
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return OrdenFronteraIdentidadTecnica{}, ErrFronteraIdentidadTecnicaNoDisponible
	}
	return OrdenFronteraIdentidadTecnica{
		metodoEsperado: metodoEsperado, ruta: ruta, motivo: motivo,
		correlacionRef: "correlacion_" + hex.EncodeToString(aleatorio[:]),
	}, nil
}

func (o OrdenFronteraIdentidadTecnica) Datos() (DatosOrdenFronteraIdentidadTecnica, error) {
	resultado := resultadoMotivoFronteraIdentidad(o.motivo)
	if !rutaMetodoInicioSesionValido(o.metodoEsperado, o.ruta) ||
		!referenciaHexFronteraIdentidad(o.correlacionRef, "correlacion_") || resultado == "" {
		return DatosOrdenFronteraIdentidadTecnica{}, ErrOrdenFronteraIdentidadTecnicaInvalida
	}
	return DatosOrdenFronteraIdentidadTecnica{
		MetodoEsperado: o.metodoEsperado, Ruta: o.ruta,
		Motivo: o.motivo, Resultado: resultado, CorrelacionRef: o.correlacionRef,
	}, nil
}

func (o OrdenFronteraIdentidadTecnica) CorrelacionRef() string { return o.correlacionRef }

func (OrdenFronteraIdentidadTecnica) String() string { return "[ORDEN-FRONTERA-IDENTIDAD-TECNICA]" }
func (o OrdenFronteraIdentidadTecnica) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, o.String())
}
func (o OrdenFronteraIdentidadTecnica) LogValue() slog.Value { return slog.StringValue(o.String()) }
func (OrdenFronteraIdentidadTecnica) MarshalJSON() ([]byte, error) {
	return nil, ErrOrdenFronteraIdentidadTecnicaInvalida
}

func rutaMetodoInicioSesionValido(metodo, ruta string) bool {
	return metodo == MetodoInicioSesionGET && ruta == RutaSesionActual ||
		metodo == MetodoInicioSesionPOST && ruta == RutaInicioSesion
}

func resultadoMotivoFronteraIdentidad(m MotivoFronteraIdentidadTecnica) string {
	switch m {
	case MotivoCertificadoRequerido, MotivoAutenticacionRequerida, MotivoAccesoDenegado,
		MotivoMetodoNoPermitido, MotivoRecursoNoEncontrado, MotivoSolicitudInvalida:
		return "denegado"
	case MotivoServicioNoDisponible, MotivoRespuestaIncompatible:
		return "error"
	default:
		return ""
	}
}

// RecursoFronteraIdentidadTecnica es una referencia opaca derivada únicamente
// de la correlación del servidor; no permite localizar cuenta ni certificado.
func RecursoFronteraIdentidadTecnica(correlacionRef string) (string, error) {
	if !referenciaHexFronteraIdentidad(correlacionRef, "correlacion_") {
		return "", ErrOrdenFronteraIdentidadTecnicaInvalida
	}
	huella := sha256.Sum256([]byte("vec.identidad.preacreditacion.solicitud.v1\n" + correlacionRef))
	return "solicitud_sesion:" + hex.EncodeToString(huella[:16]), nil
}

type AcuseFronteraIdentidadTecnica struct {
	AuditoriaRef   string
	Secuencia      uint64
	MaterialSHA256 string
	CorrelacionRef string
	RegistradaEn   time.Time
}

// ValidarPara coteja la confirmación material con la orden y el evento exactos
// antes de aceptar COMMIT. MaterialSHA256 no es el sello AD207 posterior.
func (a AcuseFronteraIdentidadTecnica) ValidarPara(
	o OrdenFronteraIdentidadTecnica, eventoRef, materialSHA256 string, ahora time.Time,
) error {
	datos, err := o.Datos()
	if err != nil || !referenciaHexFronteraIdentidad(eventoRef, "evento_") ||
		a.AuditoriaRef != "aud_v3_pit_"+strings.TrimPrefix(eventoRef, "evento_") ||
		a.Secuencia == 0 || a.Secuencia > 1<<53-1 ||
		!huellaFronteraIdentidadValida(a.MaterialSHA256) || a.MaterialSHA256 != materialSHA256 ||
		a.CorrelacionRef != datos.CorrelacionRef ||
		a.RegistradaEn.IsZero() || a.RegistradaEn.Location() != time.UTC ||
		a.RegistradaEn.Nanosecond()%1000 != 0 ||
		a.RegistradaEn.Year() < 1 || a.RegistradaEn.Year() > 9999 ||
		ahora.IsZero() || a.RegistradaEn.After(ahora) {
		return ErrAcuseFronteraIdentidadTecnicaInvalido
	}
	return nil
}

func (AcuseFronteraIdentidadTecnica) String() string { return "[ACUSE-FRONTERA-IDENTIDAD-TECNICA]" }
func (a AcuseFronteraIdentidadTecnica) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, a.String())
}
func (a AcuseFronteraIdentidadTecnica) LogValue() slog.Value { return slog.StringValue(a.String()) }

func referenciaHexFronteraIdentidad(v, prefijo string) bool {
	if len(v) != len(prefijo)+32 || !strings.HasPrefix(v, prefijo) {
		return false
	}
	for _, b := range v[len(prefijo):] {
		if b < '0' || b > '9' {
			if b < 'a' || b > 'f' {
				return false
			}
		}
	}
	return true
}

func huellaFronteraIdentidadValida(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, b := range v {
		if !(b >= '0' && b <= '9' || b >= 'a' && b <= 'f') {
			return false
		}
	}
	return true
}
