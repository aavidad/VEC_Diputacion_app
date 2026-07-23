package ports

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type TipoRespuestaFuenteAnalisis string

const (
	RespuestaValidacionRC TipoRespuestaFuenteAnalisis = "validacion_rc"
	RespuestaCalculoCoste TipoRespuestaFuenteAnalisis = "calculo_coste"
)

type OrdenConsumoRespuestaFuenteAnalisis struct {
	datos *DatosOrdenConsumoRespuestaFuenteAnalisis
}

type DatosOrdenConsumoRespuestaFuenteAnalisis struct {
	Tipo                    TipoRespuestaFuenteAnalisis
	PeticionRef             string
	OrganizacionRef         string
	ExpedienteRef           string
	VersionExpediente       uint64
	HuellaRespuestaSHA256   string
	Atestacion              AtestacionRespuestaFuenteAnalisis
	ConfirmacionRespuesta   ConfirmacionRespuestaFuenteAnalisis
	Motivo                  MotivoFuenteAnalisis
	ConfirmacionPublicacion *ConfirmacionPublicacionMotivoFuenteAnalisis
}

func nuevaOrdenConsumoResultadoRC(
	solicitud SolicitudValidarRC,
	resultado ResultadoValidacionRC,
	confirmacion ConfirmacionRespuestaFuenteAnalisis,
	confirmacionMotivo *ConfirmacionPublicacionMotivoFuenteAnalisis,
) (OrdenConsumoRespuestaFuenteAnalisis, error) {
	s, errSolicitud := solicitud.Datos()
	r, errResultado := resultado.Datos()
	datos := DatosOrdenConsumoRespuestaFuenteAnalisis{
		Tipo: RespuestaValidacionRC, PeticionRef: s.PeticionRef,
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		VersionExpediente:     s.VersionExpediente,
		HuellaRespuestaSHA256: r.HuellaRespuestaSHA256,
		Atestacion:            r.Atestacion, ConfirmacionRespuesta: confirmacion,
		Motivo: r.Motivo, ConfirmacionPublicacion: confirmacionMotivo,
	}
	if errSolicitud != nil || errResultado != nil ||
		validarOrdenConsumoRespuesta(datos, resultado.solicitudVerificacion()) != nil {
		return OrdenConsumoRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return nuevaOrdenConsumoRespuesta(datos), nil
}

func nuevaOrdenConsumoResultadoCoste(
	solicitud SolicitudCalcularCoste,
	resultado ResultadoCalculoCoste,
	confirmacion ConfirmacionRespuestaFuenteAnalisis,
) (OrdenConsumoRespuestaFuenteAnalisis, error) {
	s, errSolicitud := solicitud.Datos()
	r, errResultado := resultado.Datos()
	datos := DatosOrdenConsumoRespuestaFuenteAnalisis{
		Tipo: RespuestaCalculoCoste, PeticionRef: s.PeticionRef,
		OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		VersionExpediente:     s.VersionExpediente,
		HuellaRespuestaSHA256: r.HuellaRespuestaSHA256,
		Atestacion:            r.Atestacion, ConfirmacionRespuesta: confirmacion,
	}
	if errSolicitud != nil || errResultado != nil ||
		validarOrdenConsumoRespuesta(datos, resultado.solicitudVerificacion()) != nil {
		return OrdenConsumoRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return nuevaOrdenConsumoRespuesta(datos), nil
}

func nuevaOrdenConsumoRespuesta(
	datos DatosOrdenConsumoRespuestaFuenteAnalisis,
) OrdenConsumoRespuestaFuenteAnalisis {
	clon := datos
	clon.Motivo = datos.Motivo.clonar()
	if datos.ConfirmacionPublicacion != nil {
		confirmacion := *datos.ConfirmacionPublicacion
		clon.ConfirmacionPublicacion = &confirmacion
	}
	return OrdenConsumoRespuestaFuenteAnalisis{datos: &clon}
}

func validarOrdenConsumoRespuesta(
	datos DatosOrdenConsumoRespuestaFuenteAnalisis,
	solicitudVerificacion SolicitudVerificarRespuestaFuenteAnalisis,
) error {
	confirmacion, err := datos.ConfirmacionRespuesta.Datos()
	if (datos.Tipo != RespuestaValidacionRC &&
		datos.Tipo != RespuestaCalculoCoste) ||
		!referenciaPeticionFuenteAnalisisValida(datos.PeticionRef) ||
		!domain.ReferenciaOpacaValida(datos.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(datos.ExpedienteRef) ||
		datos.VersionExpediente == 0 ||
		!huellaSHA256FuenteAnalisisValida(datos.HuellaRespuestaSHA256) ||
		datos.Atestacion.Validar() != nil || err != nil ||
		confirmacion.HuellaMaterialSHA256 != datos.HuellaRespuestaSHA256 ||
		datos.ConfirmacionRespuesta.ValidarPara(
			solicitudVerificacion,
			confirmacion.VerificadaEn,
		) != nil {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	if datos.Tipo == RespuestaCalculoCoste {
		if datos.Motivo.datos != nil || datos.ConfirmacionPublicacion != nil {
			return ErrResultadoFuenteAnalisisNoConfiable
		}
		return nil
	}
	if datos.Motivo.datos == nil {
		if datos.ConfirmacionPublicacion != nil {
			return ErrResultadoFuenteAnalisisNoConfiable
		}
		return nil
	}
	if datos.ConfirmacionPublicacion == nil {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	solicitudMotivo := SolicitudVerificarPublicacionMotivoFuenteAnalisis{
		Motivo:                datos.Motivo,
		HuellaRespuestaSHA256: datos.HuellaRespuestaSHA256,
		AutoridadRespuestaRef: datos.Atestacion.Metadatos.AutoridadRef,
		GeneracionRespuesta:   datos.Atestacion.Metadatos.Generacion,
	}
	confirmacionMotivo, err := datos.ConfirmacionPublicacion.Datos()
	if err != nil || datos.ConfirmacionPublicacion.ValidarPara(
		solicitudMotivo,
		confirmacionMotivo.VerificadaEn,
	) != nil {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

func (o OrdenConsumoRespuestaFuenteAnalisis) Datos() (
	DatosOrdenConsumoRespuestaFuenteAnalisis,
	error,
) {
	if o.datos == nil {
		return DatosOrdenConsumoRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	datos := *o.datos
	datos.Motivo = datos.Motivo.clonar()
	if o.datos.ConfirmacionPublicacion != nil {
		confirmacion := *o.datos.ConfirmacionPublicacion
		datos.ConfirmacionPublicacion = &confirmacion
	}
	return datos, nil
}

func (OrdenConsumoRespuestaFuenteAnalisis) String() string {
	return "[ORDEN-CONSUMO-RESPUESTA-FUENTE-ANALISIS-REDACTADA]"
}

func (o OrdenConsumoRespuestaFuenteAnalisis) GoString() string { return o.String() }
func (o OrdenConsumoRespuestaFuenteAnalisis) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, o.String())
}
func (o OrdenConsumoRespuestaFuenteAnalisis) LogValue() slog.Value {
	return slog.StringValue(o.String())
}

type ReciboConsumoRespuestaFuenteAnalisis struct {
	ConsumoRef            string
	Tipo                  TipoRespuestaFuenteAnalisis
	PeticionRef           string
	ReciboRespuestaRef    string
	HuellaRespuestaSHA256 string
	ConsumidaEn           time.Time
}

func NuevoReciboConsumoRespuestaFuenteAnalisis(
	orden OrdenConsumoRespuestaFuenteAnalisis,
	consumoRef string,
	consumidaEn time.Time,
) (ReciboConsumoRespuestaFuenteAnalisis, error) {
	datos, err := orden.Datos()
	recibo := ReciboConsumoRespuestaFuenteAnalisis{
		ConsumoRef: consumoRef, Tipo: datos.Tipo,
		PeticionRef:           datos.PeticionRef,
		ReciboRespuestaRef:    datos.Atestacion.Metadatos.ReciboRef,
		HuellaRespuestaSHA256: datos.HuellaRespuestaSHA256,
		ConsumidaEn:           consumidaEn,
	}
	if err != nil || recibo.ValidarPara(orden) != nil {
		return ReciboConsumoRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return recibo, nil
}

func (r ReciboConsumoRespuestaFuenteAnalisis) ValidarPara(
	orden OrdenConsumoRespuestaFuenteAnalisis,
) error {
	datos, err := orden.Datos()
	confirmacion, errConfirmacion := datos.ConfirmacionRespuesta.Datos()
	if err != nil || errConfirmacion != nil ||
		!domain.ReferenciaOpacaValida(r.ConsumoRef) ||
		r.Tipo != datos.Tipo || r.PeticionRef != datos.PeticionRef ||
		r.ReciboRespuestaRef != datos.Atestacion.Metadatos.ReciboRef ||
		r.HuellaRespuestaSHA256 != datos.HuellaRespuestaSHA256 ||
		!instanteFuenteAnalisisCanonico(r.ConsumidaEn) ||
		r.ConsumidaEn.Before(confirmacion.VerificadaEn) ||
		!r.ConsumidaEn.Before(confirmacion.ValidaHasta) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

// ConsumidorRespuestaFuenteAnalisis debe conservar de forma durable el primer
// consumo por autoridad, generación y ReciboRespuestaRef. Una repetición
// exacta devuelve el mismo recibo; la misma clave con otra huella devuelve
// ErrRespuestaFuenteAnalisisYaConsumida. El contrato no afirma que exista ya
// un adaptador productivo.
type ConsumidorRespuestaFuenteAnalisis interface {
	ConsumirRespuestaFuenteAnalisis(
		context.Context,
		OrdenConsumoRespuestaFuenteAnalisis,
	) (ReciboConsumoRespuestaFuenteAnalisis, error)
}
