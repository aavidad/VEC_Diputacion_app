package bootstrap

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type calculadorCosteAnalisisDesarrollo struct {
	*presentadorAutoridadAnalisisDesarrollo
	derivador    *derivadorIdentidadOperacionDesarrollo
	autoridadRef string
	generacion   uint32
	reloj        relojContratacionTemporalDesarrollo
	// retribuciones es el catálogo ct.retribuciones; sin él el coste queda
	// «sin calcular».
	retribuciones *fuenteRetribucionesDesarrollo
	// catalogo es el mismo catálogo de alta que usa el preparador: sin él se
	// rechazarían por desconocidas las categorías de la RPT que el preparador
	// sí admitió.
	catalogo *catalogosAltaContratacionTemporalDesarrollo
}

var _ ports.CalculadorCostePersonal = (*calculadorCosteAnalisisDesarrollo)(nil)

func (c *calculadorCosteAnalisisDesarrollo) CalcularCoste(
	ctx context.Context,
	solicitud ports.SolicitudCalcularCoste,
) (ports.ResultadoCalculoCoste, error) {
	if c == nil || c.derivador == nil || !c.derivador.valido() ||
		c.generacion == 0 || contextoInterfazNulo(ctx) {
		return ports.ResultadoCalculoCoste{},
			ports.ErrCalculadorCosteNoDisponible
	}
	datos, err := solicitud.Datos()
	if err != nil ||
		!datosCalculoCosteAnalisisContratacionTemporalDesarrolloValidos(datos, c.catalogo) {
		return ports.ResultadoCalculoCoste{},
			ports.ErrPeticionFuenteAnalisisInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoCalculoCoste{}, err
	}
	ahora := c.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ahora.Before(datos.SolicitadaEn) {
		return ports.ResultadoCalculoCoste{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	reciboRef, err := referenciaReciboFuenteAnalisisDesarrollo(
		c.derivador,
		"coste",
		datos.PeticionRef,
	)
	if err != nil {
		return ports.ResultadoCalculoCoste{},
			ports.ErrCalculadorCosteNoDisponible
	}
	fila, ok, err := c.retribuciones.retribucion(
		ctx, datos.CategoriaRef, datos.GrupoSubgrupo,
	)
	if err != nil || !ok {
		return ports.ResultadoCalculoCoste{},
			errors.Join(ports.ErrCalculadorCosteNoDisponible, err)
	}
	importe, ok := costeEstimadoAnalisisDesarrollo(
		fila, datos.Periodo, datos.Jornada,
	)
	if !ok {
		return ports.ResultadoCalculoCoste{},
			ports.ErrCalculadorCosteNoDisponible
	}
	metadatos := ports.MetadatosAtestacionRespuestaFuenteAnalisis{
		AutoridadRef: c.autoridadRef,
		Generacion:   c.generacion,
		ReciboRef:    reciboRef,
		EmitidaEn:    ahora,
		ValidaHasta:  ahora.Add(ports.VigenciaMaximaRespuestaFuenteAnalisis),
	}
	preimagen, err := ports.NuevaPreimagenRespuestaCalculoCoste(
		solicitud,
		c.autoridadRef,
		reciboRef,
		importe,
		ahora,
		metadatos,
	)
	if err != nil {
		return ports.ResultadoCalculoCoste{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	atestacion, err := nuevaAtestacionRespuestaFuenteAnalisisDesarrollo(
		c.derivador,
		preimagen,
		metadatos,
	)
	if err != nil {
		return ports.ResultadoCalculoCoste{}, err
	}
	return ports.NuevoResultadoCalculoCoste(
		solicitud,
		c.autoridadRef,
		reciboRef,
		importe,
		ahora,
		atestacion,
	)
}

func datosCalculoCosteAnalisisContratacionTemporalDesarrolloValidos(
	datos ports.DatosSolicitudCalcularCoste,
	catalogo *catalogosAltaContratacionTemporalDesarrollo,
) bool {
	// La solicitud de coste no lleva RC propia: su forma se comprueba con la
	// primera entrada publicada.
	entradaRC := catalogo.opcionesAnalisis().primeraEntradaRC()
	solicitud := ports.SolicitudPrepararArtefactoAnalisis{
		ArtefactoRef:      artefactoAnalisisContratacionTemporalDesarrollo,
		OrganizacionRef:   datos.OrganizacionRef,
		ExpedienteRef:     datos.ExpedienteRef,
		VersionExpediente: datos.VersionExpediente,
		DatosFuncionales: ports.DatosFuncionalesOperacionAnalisis{
			ModalidadClave:    datos.ModalidadClave,
			CategoriaRef:      datos.CategoriaRef,
			GrupoSubgrupo:     datos.GrupoSubgrupo,
			CausaClave:        datos.CausaClave,
			Periodo:           datos.Periodo,
			PorcentajeJornada: datos.Jornada,
			EntradaRC: domain.VinculoEntradaRC{
				Referencia:   entradaRC.Referencia,
				HuellaSHA256: entradaRC.Huella,
			},
		},
		SolicitadaEn: datos.SolicitadaEn,
	}
	return solicitud.Validar() == nil &&
		solicitudAnalisisContratacionTemporalDesarrolloValidaConCatalogo(solicitud, catalogo)
}
