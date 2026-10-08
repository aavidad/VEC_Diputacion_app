package application

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type claveNecesidadAltaValidada struct{}

type necesidadAltaValidada struct {
	organizacionRef string
	material        domain.DatosNecesidadAlta
}

// NecesidadAltaValidadaPara permite a la composición distinguir una necesidad
// comprobada por el mismo servicio de una modalidad histórica. La clave del
// contexto es privada y ningún campo HTTP puede crear esta marca.
func NecesidadAltaValidadaPara(ctx context.Context, organizacionRef string, necesidad *domain.DatosNecesidadAlta) bool {
	if ctx == nil || necesidad == nil || necesidad.ValidarInstantanea() != nil {
		return false
	}
	marca, ok := ctx.Value(claveNecesidadAltaValidada{}).(necesidadAltaValidada)
	return ok && marca.organizacionRef == organizacionRef && reflect.DeepEqual(marca.material, *necesidad)
}

func contextoNecesidadAltaValidada(ctx context.Context, organizacionRef string, necesidad *domain.DatosNecesidadAlta) (context.Context, error) {
	if ctx == nil || necesidad == nil {
		return nil, ErrSolicitudRegistroInvalida
	}
	copia, err := necesidad.Clonar()
	if err != nil {
		return nil, ErrSolicitudRegistroInvalida
	}
	return context.WithValue(ctx, claveNecesidadAltaValidada{}, necesidadAltaValidada{organizacionRef, copia}), nil
}

func copiaNecesidadAltaParaPuerto(necesidad *domain.DatosNecesidadAlta) (*domain.DatosNecesidadAlta, error) {
	if necesidad == nil {
		return nil, nil
	}
	copia, err := necesidad.Clonar()
	if err != nil {
		return nil, ErrSolicitudRegistroInvalida
	}
	return &copia, nil
}

// prepararNecesidadAlta usa primero la instantánea confirmada de la misma
// intención. Eso permite repetir un alta después de cambiar la publicación
// activa sin volver a sellar el expediente con otros bytes.
func (s *ServicioRegistroSolicitud) prepararNecesidadAlta(
	ctx context.Context,
	entrada domain.DatosNecesidadAlta,
	ambitos ports.ColeccionSellosHMAC,
	organizacionRef, actorRef, perfilRef string,
) (domain.DatosNecesidadAlta, error) {
	if s == nil || dependenciaNula(s.fuenteNecesidades) || ctx == nil {
		return domain.DatosNecesidadAlta{}, ErrServicioRegistroInvalido
	}
	consulta := ports.ConsultaInstantaneaNecesidadConfirmada{
		AmbitosHMAC: ambitos, OrganizacionRef: organizacionRef,
		ActorRef: actorRef, PerfilRef: perfilRef,
	}
	if consulta.Validar() != nil {
		return domain.DatosNecesidadAlta{}, ErrSolicitudRegistroInvalida
	}
	resultado, err := s.fuenteNecesidades.RecuperarInstantaneaNecesidadConfirmada(ctx, consulta)
	if err != nil {
		return domain.DatosNecesidadAlta{}, errors.Join(ports.ErrPersistenciaNoDisponible, err)
	}
	if resultado.Validar() != nil {
		return domain.DatosNecesidadAlta{}, ErrResultadoRegistroNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return domain.DatosNecesidadAlta{}, err
	}
	var catalogo domain.CatalogoNecesidadesAlta
	switch resultado.Estado {
	case ports.InstantaneaNecesidadConfirmadaV3:
		catalogo, err = domain.RestaurarCatalogoNecesidadesAlta(resultado.Instantanea)
	case ports.InstantaneaNecesidadLegadoV2:
		return domain.DatosNecesidadAlta{}, ports.ErrClaveIdempotenciaUsada
	case ports.InstantaneaNecesidadAusente:
		catalogo, err = s.fuenteNecesidades.ResolverCatalogoNecesidadesAlta(
			ctx, entrada.CatalogoRef, entrada.CatalogoVersion, entrada.CatalogoHuellaSHA256,
		)
	default:
		return domain.DatosNecesidadAlta{}, ErrResultadoRegistroNoConfiable
	}
	if err != nil || catalogo.Validar() != nil {
		if resultado.Estado == ports.InstantaneaNecesidadConfirmadaV3 {
			return domain.DatosNecesidadAlta{}, ErrResultadoRegistroNoConfiable
		}
		return domain.DatosNecesidadAlta{}, errors.Join(ports.ErrFuenteNecesidadesAltaNoDisponible, err)
	}
	if err := ctx.Err(); err != nil {
		return domain.DatosNecesidadAlta{}, err
	}
	if catalogo.ValidarDatos(entrada) != nil {
		if resultado.Estado == ports.InstantaneaNecesidadConfirmadaV3 {
			return domain.DatosNecesidadAlta{}, ports.ErrClaveIdempotenciaUsada
		}
		return domain.DatosNecesidadAlta{}, ErrSolicitudRegistroInvalida
	}
	if resultado.Estado == ports.InstantaneaNecesidadAusente && entrada.Campos["puesto_codigo"] != "" {
		if dependenciaNula(s.verificadorPuestoRPT) {
			return domain.DatosNecesidadAlta{}, ErrServicioRegistroInvalido
		}
		solicitudPuesto, err := ports.SolicitudPuestoRPTDesdeCampos(entrada.Campos)
		if err != nil {
			return domain.DatosNecesidadAlta{}, ErrSolicitudRegistroInvalida
		}
		puesto, err := s.verificadorPuestoRPT.VerificarPuestoRPTAlta(ctx, solicitudPuesto)
		if err != nil {
			return domain.DatosNecesidadAlta{}, errors.Join(ports.ErrFuenteNecesidadesAltaNoDisponible, err)
		}
		if !puesto.ExisteEnPublicacion {
			return domain.DatosNecesidadAlta{}, ErrSolicitudRegistroInvalida
		}
		if puesto.ValidarPara(solicitudPuesto) != nil {
			return domain.DatosNecesidadAlta{}, ErrResultadoRegistroNoConfiable
		}
		if err := ctx.Err(); err != nil {
			return domain.DatosNecesidadAlta{}, err
		}
	}
	sellada, err := catalogo.SellarDatos(entrada)
	if err != nil {
		if resultado.Estado == ports.InstantaneaNecesidadConfirmadaV3 {
			return domain.DatosNecesidadAlta{}, ports.ErrClaveIdempotenciaUsada
		}
		return domain.DatosNecesidadAlta{}, ErrSolicitudRegistroInvalida
	}
	return sellada, nil
}
