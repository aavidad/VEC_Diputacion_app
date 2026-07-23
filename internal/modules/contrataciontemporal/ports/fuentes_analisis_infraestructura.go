package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type TipoPeticionFuenteAnalisis string

const (
	TipoPeticionValidacionRC TipoPeticionFuenteAnalisis = "validacion_rc"
	TipoPeticionCalculoCoste TipoPeticionFuenteAnalisis = "calculo_coste"
)

type GeneradorPeticionFuenteAnalisis interface {
	NuevaReferenciaPeticionFuenteAnalisis(
		context.Context,
		TipoPeticionFuenteAnalisis,
	) (string, error)
}

type SelladorPeticionFuenteAnalisis interface {
	SellarPeticionFuenteAnalisis(
		context.Context,
		PreimagenPeticionFuenteAnalisis,
	) (string, error)
}

type RelojFuenteAnalisis interface {
	Ahora() time.Time
}

type PreparacionSolicitudValidarRC struct {
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	Entrada           domain.VinculoEntradaRC
	Declaracion       domain.DeclaracionRC
}

func NuevaSolicitudValidarRC(
	ctx context.Context,
	generador GeneradorPeticionFuenteAnalisis,
	sellador SelladorPeticionFuenteAnalisis,
	reloj RelojFuenteAnalisis,
	preparacion PreparacionSolicitudValidarRC,
) (SolicitudValidarRC, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(generador) ||
		dependenciaNulaFuenteAnalisis(sellador) ||
		dependenciaNulaFuenteAnalisis(reloj) ||
		!domain.ReferenciaOpacaValida(preparacion.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(preparacion.ExpedienteRef) ||
		preparacion.VersionExpediente == 0 ||
		preparacion.Entrada.Validar() != nil ||
		preparacion.Declaracion.Validar() != nil ||
		!importeFuenteAnalisisValidoDeclaracion(preparacion.Declaracion) {
		return SolicitudValidarRC{}, ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(ctx, TiempoMaximoFuenteAnalisis)
	defer cancelar()
	solicitadaEn := reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return SolicitudValidarRC{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			err,
		)
	}
	if !instanteFuenteAnalisisCanonico(solicitadaEn) {
		return SolicitudValidarRC{}, ErrPeticionFuenteAnalisisInvalida
	}
	peticionRef, errGenerador := generador.NuevaReferenciaPeticionFuenteAnalisis(
		operacion,
		TipoPeticionValidacionRC,
	)
	if err := operacion.Err(); err != nil {
		return SolicitudValidarRC{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			err,
		)
	}
	if errGenerador != nil || !referenciaPeticionFuenteAnalisisValida(peticionRef) {
		return SolicitudValidarRC{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			errGenerador,
		)
	}
	datos := DatosSolicitudValidarRC{
		PeticionRef: peticionRef, OrganizacionRef: preparacion.OrganizacionRef,
		ExpedienteRef:     preparacion.ExpedienteRef,
		VersionExpediente: preparacion.VersionExpediente,
		Entrada:           preparacion.Entrada, Declaracion: preparacion.Declaracion,
		SolicitadaEn: solicitadaEn,
	}
	canonica, err := canonPeticionValidacionRC(datos)
	if err != nil {
		return SolicitudValidarRC{}, ErrPeticionFuenteAnalisisInvalida
	}
	sello, errSellador := sellador.SellarPeticionFuenteAnalisis(
		operacion,
		PreimagenPeticionFuenteAnalisis{contenido: canonica},
	)
	if err := operacion.Err(); err != nil {
		return SolicitudValidarRC{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			err,
		)
	}
	if errSellador != nil || !selloPeticionFuenteAnalisisValido(sello) {
		return SolicitudValidarRC{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			errSellador,
		)
	}
	datos.HuellaPeticionHMAC = sello
	return SolicitudValidarRC{
		datos: &datos, preimagen: append([]byte(nil), canonica...),
	}, nil
}

type PreparacionSolicitudCalcularCoste struct {
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	CategoriaRef      string
	GrupoSubgrupo     string
	ModalidadClave    domain.ClaveCatalogo
	CausaClave        domain.ClaveCatalogo
	Periodo           domain.PeriodoPrevisto
	Jornada           domain.JornadaDiezmilesimas
}

func NuevaSolicitudCalcularCoste(
	ctx context.Context,
	generador GeneradorPeticionFuenteAnalisis,
	sellador SelladorPeticionFuenteAnalisis,
	reloj RelojFuenteAnalisis,
	preparacion PreparacionSolicitudCalcularCoste,
) (SolicitudCalcularCoste, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(generador) ||
		dependenciaNulaFuenteAnalisis(sellador) ||
		dependenciaNulaFuenteAnalisis(reloj) ||
		!domain.ReferenciaOpacaValida(preparacion.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(preparacion.ExpedienteRef) ||
		preparacion.VersionExpediente == 0 ||
		!domain.ReferenciaOpacaValida(preparacion.CategoriaRef) ||
		!domain.GrupoSubgrupoValido(preparacion.GrupoSubgrupo) ||
		!preparacion.ModalidadClave.Valida() ||
		!preparacion.CausaClave.Valida() ||
		!periodoFuenteAnalisisValido(preparacion.Periodo) ||
		preparacion.Jornada.Validar() != nil {
		return SolicitudCalcularCoste{}, ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(ctx, TiempoMaximoFuenteAnalisis)
	defer cancelar()
	solicitadaEn := reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return SolicitudCalcularCoste{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			err,
		)
	}
	if !instanteFuenteAnalisisCanonico(solicitadaEn) {
		return SolicitudCalcularCoste{}, ErrPeticionFuenteAnalisisInvalida
	}
	peticionRef, errGenerador := generador.NuevaReferenciaPeticionFuenteAnalisis(
		operacion,
		TipoPeticionCalculoCoste,
	)
	if err := operacion.Err(); err != nil {
		return SolicitudCalcularCoste{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			err,
		)
	}
	if errGenerador != nil || !referenciaPeticionFuenteAnalisisValida(peticionRef) {
		return SolicitudCalcularCoste{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			errGenerador,
		)
	}
	datos := DatosSolicitudCalcularCoste{
		PeticionRef: peticionRef, OrganizacionRef: preparacion.OrganizacionRef,
		ExpedienteRef:     preparacion.ExpedienteRef,
		VersionExpediente: preparacion.VersionExpediente,
		CategoriaRef:      preparacion.CategoriaRef,
		GrupoSubgrupo:     preparacion.GrupoSubgrupo,
		ModalidadClave:    preparacion.ModalidadClave,
		CausaClave:        preparacion.CausaClave, Periodo: preparacion.Periodo,
		Jornada: preparacion.Jornada, SolicitadaEn: solicitadaEn,
	}
	canonica, err := canonPeticionCalculoCoste(datos)
	if err != nil {
		return SolicitudCalcularCoste{}, ErrPeticionFuenteAnalisisInvalida
	}
	sello, errSellador := sellador.SellarPeticionFuenteAnalisis(
		operacion,
		PreimagenPeticionFuenteAnalisis{contenido: canonica},
	)
	if err := operacion.Err(); err != nil {
		return SolicitudCalcularCoste{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			err,
		)
	}
	if errSellador != nil || !selloPeticionFuenteAnalisisValido(sello) {
		return SolicitudCalcularCoste{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			errSellador,
		)
	}
	datos.HuellaPeticionHMAC = sello
	return SolicitudCalcularCoste{
		datos: &datos, preimagen: append([]byte(nil), canonica...),
	}, nil
}
