package bootstrap

import (
	"context"
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
		!datosCalculoCosteAnalisisContratacionTemporalDesarrolloValidos(datos) {
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
	importe, ok := costeEstimadoAnalisisDesarrollo(
		datos.GrupoSubgrupo, datos.Periodo, datos.Jornada,
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
) bool {
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
				Referencia:   entradaRCAnalisisContratacionTemporalDesarrollo,
				HuellaSHA256: huellaEntradaRCAnalisisContratacionTemporalDesarrollo,
			},
		},
		SolicitadaEn: datos.SolicitadaEn,
	}
	return solicitud.Validar() == nil &&
		solicitudAnalisisContratacionTemporalDesarrolloValida(solicitud)
}

// costeMensualReferenciaDesarrollo es la tabla de referencia de desarrollo:
// coste empresa mensual aproximado por grupo (céntimos de euro, jornada
// completa). Sirve para que la estimación varíe con la categoría, el periodo y
// la jornada en las demostraciones; no es la tabla oficial. La fuente real
// (GINPIX o la que fije Intervención) se decide con RRHH (dudas, pregunta 8).
var costeMensualReferenciaDesarrollo = map[string]int64{
	"A1": 460_000,
	"A2": 390_000,
	"B":  330_000,
	"C1": 300_000,
	"C2": 260_000,
	"AP": 230_000,
}

// costeEstimadoAnalisisDesarrollo prorratea el coste mensual del grupo por los
// días naturales del periodo (ambos inclusive) y por la jornada, redondeando al
// céntimo. Devuelve false si el grupo no está en la tabla o el periodo no es
// posterior o igual a su inicio: entonces el coste queda «sin calcular».
func costeEstimadoAnalisisDesarrollo(
	grupo string,
	periodo domain.PeriodoPrevisto,
	jornada domain.JornadaDiezmilesimas,
) (domain.Importe, bool) {
	mensual, ok := costeMensualReferenciaDesarrollo[grupo]
	if !ok || jornada == 0 || periodo.Fin.Before(periodo.Inicio) {
		return domain.Importe{}, false
	}
	dias := int64(periodo.Fin.Sub(periodo.Inicio).Hours()/24) + 1
	if dias <= 0 || dias > 3_660 {
		return domain.Importe{}, false
	}
	// céntimos = mensual × días × jornada / (días por mes × 10 000 diezmilésimas)
	// con la jornada en diezmilésimas; se agrupa para redondear una sola vez.
	numerador := mensual * dias * int64(jornada)
	divisor := diasMesDiezmilesimasCosteDesarrollo
	centimos := (numerador + divisor/2) / divisor
	if centimos <= 0 {
		return domain.Importe{}, false
	}
	return domain.Importe{Centimos: centimos, Moneda: "EUR"}, true
}
