package ports

import (
	"bytes"
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

func (p PreparacionSolicitudValidarRC) Validar() error {
	if !domain.ReferenciaOpacaValida(p.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(p.ExpedienteRef) ||
		p.VersionExpediente == 0 ||
		p.VersionExpediente > maximoEnteroSeguroFuenteAnalisis ||
		p.Entrada.Validar() != nil ||
		p.Declaracion.Validar() != nil ||
		!importeFuenteAnalisisValidoDeclaracion(p.Declaracion) {
		return ErrPeticionFuenteAnalisisInvalida
	}
	return nil
}

type PreparacionSelladoSolicitudValidarRC struct {
	datos     DatosSolicitudValidarRC
	preimagen []byte
}

func NuevaPreparacionSelladoSolicitudValidarRC(
	preparacion PreparacionSolicitudValidarRC,
	peticionRef string,
	solicitadaEn time.Time,
) (PreparacionSelladoSolicitudValidarRC, error) {
	if preparacion.Validar() != nil ||
		!referenciaPeticionFuenteAnalisisValida(peticionRef) ||
		!instanteFuenteAnalisisCanonico(solicitadaEn) {
		return PreparacionSelladoSolicitudValidarRC{},
			ErrPeticionFuenteAnalisisInvalida
	}
	datos := DatosSolicitudValidarRC{
		PeticionRef:       peticionRef,
		OrganizacionRef:   preparacion.OrganizacionRef,
		ExpedienteRef:     preparacion.ExpedienteRef,
		VersionExpediente: preparacion.VersionExpediente,
		Entrada:           preparacion.Entrada,
		Declaracion:       preparacion.Declaracion,
		SolicitadaEn:      solicitadaEn,
	}
	canonica, err := canonPeticionValidacionRC(datos)
	if err != nil {
		return PreparacionSelladoSolicitudValidarRC{},
			ErrPeticionFuenteAnalisisInvalida
	}
	return PreparacionSelladoSolicitudValidarRC{
		datos:     datos,
		preimagen: append([]byte(nil), canonica...),
	}, nil
}

func (p PreparacionSelladoSolicitudValidarRC) Preimagen() (
	PreimagenPeticionFuenteAnalisis,
	error,
) {
	if len(p.preimagen) == 0 {
		return PreimagenPeticionFuenteAnalisis{},
			ErrPeticionFuenteAnalisisInvalida
	}
	return PreimagenPeticionFuenteAnalisis{
		contenido: append([]byte(nil), p.preimagen...),
	}, nil
}

func (p PreparacionSelladoSolicitudValidarRC) Completar(
	sello string,
) (SolicitudValidarRC, error) {
	canonica, err := canonPeticionValidacionRC(p.datos)
	if err != nil || !selloPeticionFuenteAnalisisValido(sello) ||
		!bytes.Equal(canonica, p.preimagen) {
		return SolicitudValidarRC{}, ErrPeticionFuenteAnalisisInvalida
	}
	datos := p.datos
	datos.HuellaPeticionHMAC = sello
	return SolicitudValidarRC{
		datos:     &datos,
		preimagen: append([]byte(nil), canonica...),
		sello:     sello,
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

func (p PreparacionSolicitudCalcularCoste) Validar() error {
	if !domain.ReferenciaOpacaValida(p.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(p.ExpedienteRef) ||
		p.VersionExpediente == 0 ||
		p.VersionExpediente > maximoEnteroSeguroFuenteAnalisis ||
		!domain.ReferenciaOpacaValida(p.CategoriaRef) ||
		!domain.GrupoSubgrupoValido(p.GrupoSubgrupo) ||
		!p.ModalidadClave.Valida() ||
		!p.CausaClave.Valida() ||
		!periodoFuenteAnalisisValido(p.Periodo) ||
		p.Jornada.Validar() != nil {
		return ErrPeticionFuenteAnalisisInvalida
	}
	return nil
}

type PreparacionSelladoSolicitudCalcularCoste struct {
	datos     DatosSolicitudCalcularCoste
	preimagen []byte
}

func NuevaPreparacionSelladoSolicitudCalcularCoste(
	preparacion PreparacionSolicitudCalcularCoste,
	peticionRef string,
	solicitadaEn time.Time,
) (PreparacionSelladoSolicitudCalcularCoste, error) {
	if preparacion.Validar() != nil ||
		!referenciaPeticionFuenteAnalisisValida(peticionRef) ||
		!instanteFuenteAnalisisCanonico(solicitadaEn) {
		return PreparacionSelladoSolicitudCalcularCoste{},
			ErrPeticionFuenteAnalisisInvalida
	}
	datos := DatosSolicitudCalcularCoste{
		PeticionRef:       peticionRef,
		OrganizacionRef:   preparacion.OrganizacionRef,
		ExpedienteRef:     preparacion.ExpedienteRef,
		VersionExpediente: preparacion.VersionExpediente,
		CategoriaRef:      preparacion.CategoriaRef,
		GrupoSubgrupo:     preparacion.GrupoSubgrupo,
		ModalidadClave:    preparacion.ModalidadClave,
		CausaClave:        preparacion.CausaClave,
		Periodo:           preparacion.Periodo,
		Jornada:           preparacion.Jornada,
		SolicitadaEn:      solicitadaEn,
	}
	canonica, err := canonPeticionCalculoCoste(datos)
	if err != nil {
		return PreparacionSelladoSolicitudCalcularCoste{},
			ErrPeticionFuenteAnalisisInvalida
	}
	return PreparacionSelladoSolicitudCalcularCoste{
		datos:     datos,
		preimagen: append([]byte(nil), canonica...),
	}, nil
}

func (p PreparacionSelladoSolicitudCalcularCoste) Preimagen() (
	PreimagenPeticionFuenteAnalisis,
	error,
) {
	if len(p.preimagen) == 0 {
		return PreimagenPeticionFuenteAnalisis{},
			ErrPeticionFuenteAnalisisInvalida
	}
	return PreimagenPeticionFuenteAnalisis{
		contenido: append([]byte(nil), p.preimagen...),
	}, nil
}

func (p PreparacionSelladoSolicitudCalcularCoste) Completar(
	sello string,
) (SolicitudCalcularCoste, error) {
	canonica, err := canonPeticionCalculoCoste(p.datos)
	if err != nil || !selloPeticionFuenteAnalisisValido(sello) ||
		!bytes.Equal(canonica, p.preimagen) {
		return SolicitudCalcularCoste{}, ErrPeticionFuenteAnalisisInvalida
	}
	datos := p.datos
	datos.HuellaPeticionHMAC = sello
	return SolicitudCalcularCoste{
		datos:     &datos,
		preimagen: append([]byte(nil), canonica...),
		sello:     sello,
	}, nil
}
