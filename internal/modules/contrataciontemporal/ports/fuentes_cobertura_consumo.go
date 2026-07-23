package ports

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// DatosOrdenConsumoCobertura liga el efecto durable a la respuesta completa,
// a su verificación independiente y al catálogo gobernado usado para decidir.
type DatosOrdenConsumoCobertura struct {
	PeticionRef           string
	OrganizacionRef       string
	ExpedienteRef         string
	VersionExpediente     uint64
	AutoridadRef          string
	Generacion            uint32
	ReciboRespuestaRef    string
	HuellaRespuestaSHA256 string
	Atestacion            AtestacionRespuestaCobertura
	ConfirmacionRespuesta ConfirmacionRespuestaCobertura
	ConfirmacionCatalogo  ConfirmacionPublicacionCobertura
}

type OrdenConsumoCobertura struct {
	datos *DatosOrdenConsumoCobertura
}

func nuevaOrdenConsumoCobertura(
	solicitud SolicitudConsultarCobertura,
	resultado ResultadoConsultaCobertura,
	confirmacion ConfirmacionRespuestaCobertura,
	confirmacionCatalogo ConfirmacionPublicacionCobertura,
) (OrdenConsumoCobertura, error) {
	if resultado.ValidarPara(solicitud) != nil {
		return OrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	huella, errHuella := resultado.preimagen.huellaSHA256()
	datosConfirmacion, errConfirmacion := confirmacion.Datos()
	datos := DatosOrdenConsumoCobertura{
		PeticionRef:           solicitud.PeticionRef,
		OrganizacionRef:       solicitud.OrganizacionRef,
		ExpedienteRef:         solicitud.ExpedienteRef,
		VersionExpediente:     solicitud.VersionExpediente,
		AutoridadRef:          resultado.atestacion.Metadatos.AutoridadRef,
		Generacion:            resultado.atestacion.Metadatos.Generacion,
		ReciboRespuestaRef:    resultado.atestacion.Metadatos.ReciboRef,
		HuellaRespuestaSHA256: huella,
		Atestacion:            resultado.atestacion,
		ConfirmacionRespuesta: confirmacion,
		ConfirmacionCatalogo:  confirmacionCatalogo,
	}
	if errHuella != nil || errConfirmacion != nil ||
		datosConfirmacion.HuellaMaterialSHA256 != huella ||
		validarOrdenConsumoCobertura(
			datos,
			solicitud,
			resultado,
			datosConfirmacion.VerificadaEn,
		) != nil {
		return OrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return OrdenConsumoCobertura{datos: &datos}, nil
}

func validarOrdenConsumoCobertura(
	datos DatosOrdenConsumoCobertura,
	solicitud SolicitudConsultarCobertura,
	resultado ResultadoConsultaCobertura,
	comprobadaEn time.Time,
) error {
	if solicitud.Validar() != nil || resultado.ValidarPara(solicitud) != nil ||
		!domain.ReferenciaOpacaValida(datos.PeticionRef) ||
		datos.PeticionRef != solicitud.PeticionRef ||
		datos.OrganizacionRef != solicitud.OrganizacionRef ||
		datos.ExpedienteRef != solicitud.ExpedienteRef ||
		datos.VersionExpediente != solicitud.VersionExpediente ||
		datos.VersionExpediente > maximoEnteroSeguroFuenteAnalisis ||
		datos.AutoridadRef != resultado.atestacion.Metadatos.AutoridadRef ||
		datos.Generacion != resultado.atestacion.Metadatos.Generacion ||
		datos.ReciboRespuestaRef !=
			resultado.atestacion.Metadatos.ReciboRef ||
		!huellaSHA256FuenteAnalisisValida(datos.HuellaRespuestaSHA256) ||
		datos.Atestacion.Validar() != nil {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	solicitudVerificacion, err := nuevaSolicitudVerificarRespuestaCobertura(
		resultado.preimagen,
		datos.Atestacion,
	)
	if err != nil ||
		datos.ConfirmacionRespuesta.ValidarPara(
			solicitudVerificacion,
			comprobadaEn,
		) != nil ||
		datos.ConfirmacionCatalogo.ValidarPara(
			solicitud,
			comprobadaEn,
		) != nil {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func (o OrdenConsumoCobertura) Datos() (
	DatosOrdenConsumoCobertura,
	error,
) {
	if o.datos == nil {
		return DatosOrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return *o.datos, nil
}

func (OrdenConsumoCobertura) String() string {
	return "[ORDEN-CONSUMO-COBERTURA-REDACTADA]"
}

func (o OrdenConsumoCobertura) GoString() string { return o.String() }
func (o OrdenConsumoCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, o.String())
}
func (o OrdenConsumoCobertura) LogValue() slog.Value {
	return slog.StringValue(o.String())
}

type ReciboConsumoCobertura struct {
	ConsumoRef            string
	PeticionRef           string
	AutoridadRef          string
	Generacion            uint32
	ReciboRespuestaRef    string
	HuellaRespuestaSHA256 string
	ConsumidaEn           time.Time
}

func NuevoReciboConsumoCobertura(
	orden OrdenConsumoCobertura,
	consumoRef string,
	consumidaEn time.Time,
) (ReciboConsumoCobertura, error) {
	datos, err := orden.Datos()
	recibo := ReciboConsumoCobertura{
		ConsumoRef:            consumoRef,
		PeticionRef:           datos.PeticionRef,
		AutoridadRef:          datos.AutoridadRef,
		Generacion:            datos.Generacion,
		ReciboRespuestaRef:    datos.ReciboRespuestaRef,
		HuellaRespuestaSHA256: datos.HuellaRespuestaSHA256,
		ConsumidaEn:           consumidaEn,
	}
	if err != nil || recibo.ValidarPara(orden) != nil {
		return ReciboConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return recibo, nil
}

func (r ReciboConsumoCobertura) ValidarPara(
	orden OrdenConsumoCobertura,
) error {
	datos, err := orden.Datos()
	confirmacion, errConfirmacion := datos.ConfirmacionRespuesta.Datos()
	if err != nil || errConfirmacion != nil ||
		!domain.ReferenciaOpacaValida(r.ConsumoRef) ||
		r.PeticionRef != datos.PeticionRef ||
		r.AutoridadRef != datos.AutoridadRef ||
		r.Generacion != datos.Generacion ||
		r.ReciboRespuestaRef != datos.ReciboRespuestaRef ||
		r.HuellaRespuestaSHA256 != datos.HuellaRespuestaSHA256 ||
		!instanteFuenteAnalisisCanonico(r.ConsumidaEn) ||
		r.ConsumidaEn.Before(confirmacion.VerificadaEn) ||
		!r.ConsumidaEn.Before(confirmacion.ValidaHasta) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

// ConsumidorCobertura debe almacenar de forma durable el primer consumo por
// (autoridad, generación, recibo). Un replay exacto devuelve el mismo recibo;
// la misma clave con otra huella devuelve ErrRespuestaCoberturaYaConsumida.
// El puerto no afirma que exista un adaptador productivo.
type ConsumidorCobertura interface {
	ConsumirCobertura(
		context.Context,
		OrdenConsumoCobertura,
	) (ReciboConsumoCobertura, error)
}
