package application

import (
	"context"
	"errors"
	"time"

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
	evidenciaRC, err := verificarValidacionRCConFuenteO3(
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
		evidenciaCoste, err = verificarCalculoCosteConFuenteO3(
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
		evidenciaCoste.ValidarEn(comprobadaEn) != nil {
		return ports.ArtefactoAnalisisPreparado{},
			ports.ErrArtefactoAnalisisNoConfiable
	}
	if err := c.revalidarAutoridadesEvidencias(
		operacion,
		evidenciaRC,
		evidenciaCoste,
		comprobadaEn,
	); err != nil {
		return ports.ArtefactoAnalisisPreparado{}, err
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

func (c *CapacidadPrepararArtefactoAnalisisO3) revalidarAutoridadesEvidencias(
	ctx context.Context,
	rc ports.EvidenciaValidacionRCVerificadaO3,
	coste ports.EvidenciaCalculoCosteVerificadaO3,
	comprobadaEn time.Time,
) error {
	preparacion, err :=
		ports.NuevaPreparacionRevalidacionEvidenciasFuenteAnalisisO3(
			rc,
			coste,
			comprobadaEn,
		)
	if err != nil {
		return ports.ErrArtefactoAnalisisNoConfiable
	}
	materialRC, err := preparacion.MaterialRC()
	if err != nil {
		return ports.ErrArtefactoAnalisisNoConfiable
	}
	fuenteRC, err := presentarAutoridadFuenteAnalisisO3(
		ctx,
		c.fuenteRC,
		c.confianza,
		materialRC,
		ports.RolFuentePresupuestaria,
		c.reloj,
	)
	if err != nil {
		return err
	}
	verificadorRC, err := presentarAutoridadFuenteAnalisisO3(
		ctx,
		c.verificador,
		c.confianza,
		materialRC,
		ports.RolVerificadorRespuesta,
		c.reloj,
	)
	if err != nil {
		return err
	}
	publicadorRC, err := presentarAutoridadFuenteAnalisisO3(
		ctx,
		c.publicador,
		c.confianza,
		materialRC,
		ports.RolPublicadorCatalogo,
		c.reloj,
	)
	if err != nil {
		return err
	}
	resultado := ports.ResultadoRevalidacionEvidenciasFuenteAnalisisO3{
		FuenteRC:      fuenteRC,
		VerificadorRC: verificadorRC,
		PublicadorRC:  publicadorRC,
	}
	materialCoste, existeCoste, err := preparacion.MaterialCoste()
	if err != nil {
		return ports.ErrArtefactoAnalisisNoConfiable
	}
	if existeCoste {
		resultado.FuenteCoste, err = presentarAutoridadFuenteAnalisisO3(
			ctx,
			c.calculador,
			c.confianza,
			materialCoste,
			ports.RolCalculadorCoste,
			c.reloj,
		)
		if err != nil {
			return err
		}
		resultado.VerificadorCoste, err =
			presentarAutoridadFuenteAnalisisO3(
				ctx,
				c.verificador,
				c.confianza,
				materialCoste,
				ports.RolVerificadorRespuesta,
				c.reloj,
			)
		if err != nil {
			return err
		}
	}
	comprobadaResultado := c.reloj.Ahora()
	if preparacion.ValidarResultado(
		resultado,
		comprobadaResultado,
	) != nil {
		return ports.ErrArtefactoAnalisisNoConfiable
	}
	return nil
}

func presentarAutoridadFuenteAnalisisO3(
	ctx context.Context,
	presentador ports.PresentadorAutoridadFuenteAnalisis,
	confianza ports.ConfianzaAutoridadesFuenteAnalisis,
	material []byte,
	rol ports.RolAutoridadFuenteAnalisis,
	reloj ports.RelojFuenteAnalisis,
) (ports.ConfirmacionComprobacionAutoridadFuenteAnalisis, error) {
	if ctx == nil || dependenciaNula(presentador) ||
		dependenciaNula(reloj) {
		return ports.ConfirmacionComprobacionAutoridadFuenteAnalisis{},
			ports.ErrSolicitudArtefactoAnalisisInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ConfirmacionComprobacionAutoridadFuenteAnalisis{},
			errorArtefactoAnalisisNoDisponible(err)
	}
	iniciadaEn := reloj.Ahora()
	comprobacion, err := ports.NuevaComprobacionAutoridadFuenteAnalisis(
		confianza,
		material,
		rol,
		iniciadaEn,
	)
	if err != nil {
		return ports.ConfirmacionComprobacionAutoridadFuenteAnalisis{},
			ports.ErrArtefactoAnalisisNoConfiable
	}
	desafio, err := comprobacion.Desafio()
	if err != nil {
		return ports.ConfirmacionComprobacionAutoridadFuenteAnalisis{},
			ports.ErrArtefactoAnalisisNoConfiable
	}
	presentacion, errPresentacion :=
		presentador.PresentarAutoridadFuenteAnalisis(ctx, desafio)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.ConfirmacionComprobacionAutoridadFuenteAnalisis{},
			errorArtefactoAnalisisNoDisponible(errContexto)
	}
	comprobadaEn := reloj.Ahora()
	vinculo, errVerificacion :=
		comprobacion.ValidarPresentacion(presentacion, comprobadaEn)
	if errPresentacion != nil || errVerificacion != nil {
		return ports.ConfirmacionComprobacionAutoridadFuenteAnalisis{},
			ports.ErrArtefactoAnalisisNoConfiable
	}
	return vinculo, nil
}

func errorArtefactoAnalisisNoDisponible(causa error) error {
	if errors.Is(causa, context.Canceled) ||
		errors.Is(causa, context.DeadlineExceeded) {
		return errors.Join(ports.ErrArtefactoAnalisisNoDisponible, causa)
	}
	return ports.ErrArtefactoAnalisisNoDisponible
}

var _ ports.PreparadorArtefactoAnalisisO3 = (*CapacidadPrepararArtefactoAnalisisO3)(nil)
