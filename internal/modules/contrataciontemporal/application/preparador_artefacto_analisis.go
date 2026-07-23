package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// CapacidadPrepararArtefactoAnalisisO3 orquesta fuentes y validadores desde la
// capa de aplicación. No consume respuestas ni produce efectos durables.
type CapacidadPrepararArtefactoAnalisisO3 struct {
	solicitudes ports.PreparadorSolicitudesFuentesAnalisisO3
	fuenteRC    ports.FuentePresupuestaria
	calculador  ports.CalculadorCostePersonal
	verificador ports.VerificadorRespuestaFuenteAnalisis
	publicador  ports.VerificadorPublicacionMotivoFuenteAnalisis
	confianza   ports.ConfianzaAutoridadesFuenteAnalisis
	reloj       ports.RelojFuenteAnalisis
}

func NuevaCapacidadPrepararArtefactoAnalisisO3ParaComposicionInterna(
	solicitudes ports.PreparadorSolicitudesFuentesAnalisisO3,
	fuenteRC ports.FuentePresupuestaria,
	calculador ports.CalculadorCostePersonal,
	verificador ports.VerificadorRespuestaFuenteAnalisis,
	publicador ports.VerificadorPublicacionMotivoFuenteAnalisis,
	confianza ports.ConfianzaAutoridadesFuenteAnalisis,
	reloj ports.RelojFuenteAnalisis,
) (*CapacidadPrepararArtefactoAnalisisO3, error) {
	if dependenciaNula(solicitudes) || dependenciaNula(fuenteRC) ||
		dependenciaNula(calculador) || dependenciaNula(verificador) ||
		dependenciaNula(publicador) || dependenciaNula(reloj) ||
		confianza.Validar() != nil {
		return nil, ports.ErrSolicitudArtefactoAnalisisInvalida
	}
	return &CapacidadPrepararArtefactoAnalisisO3{
		solicitudes: solicitudes,
		fuenteRC:    fuenteRC,
		calculador:  calculador,
		verificador: verificador,
		publicador:  publicador,
		confianza:   confianza,
		reloj:       reloj,
	}, nil
}

func (c *CapacidadPrepararArtefactoAnalisisO3) PrepararArtefactoAnalisis(
	ctx context.Context,
	solicitud ports.SolicitudPrepararArtefactoAnalisis,
) (ports.ArtefactoAnalisisPreparado, error) {
	if c == nil || ctx == nil || solicitud.Validar() != nil ||
		dependenciaNula(c.solicitudes) || dependenciaNula(c.fuenteRC) ||
		dependenciaNula(c.calculador) || dependenciaNula(c.verificador) ||
		dependenciaNula(c.publicador) || dependenciaNula(c.reloj) ||
		c.confianza.Validar() != nil {
		return ports.ArtefactoAnalisisPreparado{},
			ports.ErrSolicitudArtefactoAnalisisInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ArtefactoAnalisisPreparado{},
			errorArtefactoAnalisisNoDisponible(err)
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		ports.TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	solicitudes, err := c.solicitudes.
		PrepararSolicitudesFuentesAnalisisO3(operacion, solicitud)
	if err != nil {
		return ports.ArtefactoAnalisisPreparado{},
			errorArtefactoAnalisisNoDisponible(operacion.Err())
	}
	if solicitudes.ValidarPara(solicitud) != nil {
		return ports.ArtefactoAnalisisPreparado{},
			ports.ErrArtefactoAnalisisNoConfiable
	}
	evidenciaRC, err := ports.VerificarValidacionRCConFuenteO3(
		operacion,
		c.fuenteRC,
		c.verificador,
		c.publicador,
		c.confianza,
		c.reloj,
		solicitudes.ValidacionRC,
	)
	if err != nil {
		return ports.ArtefactoAnalisisPreparado{}, err
	}
	var evidenciaCoste ports.EvidenciaCalculoCosteVerificadaO3
	if solicitudes.CalculoCoste != nil {
		evidenciaCoste, err = ports.VerificarCalculoCosteConFuenteO3(
			operacion,
			c.calculador,
			c.verificador,
			c.confianza,
			c.reloj,
			*solicitudes.CalculoCoste,
		)
		if err != nil {
			return ports.ArtefactoAnalisisPreparado{}, err
		}
	}
	comprobadaEn := c.reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return ports.ArtefactoAnalisisPreparado{},
			errorArtefactoAnalisisNoDisponible(err)
	}
	if evidenciaRC.ValidarEn(comprobadaEn) != nil ||
		evidenciaCoste.ValidarEn(comprobadaEn) != nil ||
		ports.RevalidarEvidenciasFuenteAnalisisO3(
			operacion,
			c.fuenteRC,
			c.calculador,
			c.verificador,
			c.publicador,
			c.confianza,
			evidenciaRC,
			evidenciaCoste,
			comprobadaEn,
		) != nil {
		return ports.ArtefactoAnalisisPreparado{},
			ports.ErrArtefactoAnalisisNoConfiable
	}
	preparadoEn := c.reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return ports.ArtefactoAnalisisPreparado{},
			errorArtefactoAnalisisNoDisponible(err)
	}
	if evidenciaRC.ValidarEn(preparadoEn) != nil ||
		evidenciaCoste.ValidarEn(preparadoEn) != nil {
		return ports.ArtefactoAnalisisPreparado{},
			ports.ErrArtefactoAnalisisNoConfiable
	}
	return ports.NuevoArtefactoAnalisisVerificadoO3(
		solicitud,
		evidenciaRC,
		evidenciaCoste,
		preparadoEn,
	)
}

func errorArtefactoAnalisisNoDisponible(causa error) error {
	if errors.Is(causa, context.Canceled) ||
		errors.Is(causa, context.DeadlineExceeded) {
		return errors.Join(ports.ErrArtefactoAnalisisNoDisponible, causa)
	}
	return ports.ErrArtefactoAnalisisNoDisponible
}

var _ ports.PreparadorArtefactoAnalisisO3 = (*CapacidadPrepararArtefactoAnalisisO3)(nil)
