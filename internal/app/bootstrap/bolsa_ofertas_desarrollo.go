package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	reglasadjudicacion "vec-diputacion-granada/internal/modules/bolsa/application/reglasadjudicacion"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// nuevaRutaPoliticaOfertasBolsaDesarrollo compone B47 sólo cuando la raíz ya
// validó el selector, AD3-93, Bolsa 000047, el pool de Bolsa, Calendarios y
// el material V3 de su audiencia propia. No reutiliza la autorización B7.
func nuevaRutaPoliticaOfertasBolsaDesarrollo(
	ctx context.Context,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	preparador *preparadorBorradorLlamamientoDesarrollo,
	emisor *emisorBorradorLlamamientoDesarrollo,
	calendarios calendariosports.ConsultaCalendarios,
) (vechttp.RutaExacta, *reglasadjudicacion.Servicio, error) {
	if ctx == nil || alta == nil || alta.postgresql.bolsa == nil ||
		alta.postgresql.proveedorMaterialPoliticaOfertas == nil || preparador == nil || emisor == nil ||
		emisor.politicaOfertas == nil {
		return vechttp.RutaExacta{}, nil, errBorradorNoDisponibleEn()
	}
	var instalada bool
	if err := alta.postgresql.bolsa.QueryRow(ctx, `SELECT
		to_regclass('vec_bolsa_llamamientos.politica_ofertas_version') IS NOT NULL AND
		to_regprocedure('vec_bolsa_llamamientos.leer_politica_ofertas_v1(text)') IS NOT NULL AND
		to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_politica_ofertas_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL`).Scan(&instalada); err != nil || !instalada {
		return vechttp.RutaExacta{}, nil, errBorradorNoDisponibleEn()
	}
	repo, err := postgresbolsa.NuevoRepositorioPoliticaOfertasPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		return vechttp.RutaExacta{}, nil, errBorradorNoDisponibleEn()
	}
	servicio, err := reglasadjudicacion.NuevoServicio(repo, calendarios)
	if err != nil {
		return vechttp.RutaExacta{}, nil, errBorradorNoDisponibleEn()
	}
	handler, err := bolsahttp.NuevoHandlerPoliticaOfertas(
		&preparadorPoliticaOfertasBolsaDesarrollo{base: preparador, emisor: emisor}, servicio,
	)
	if err != nil {
		return vechttp.RutaExacta{}, nil, errBorradorNoDisponibleEn()
	}
	return vechttp.RutaExacta{Ruta: bolsahttp.RutaPoliticaOfertas, Manejador: handler}, servicio, nil
}

// Con B47 activo, las ofertas consumen la política publicada de su bolsa.
// El calculador antiguo continúa intacto cuando el selector está apagado.
type calculadoraPlazoPoliticaOfertasBolsaDesarrollo struct{ servicio *reglasadjudicacion.Servicio }

func (c calculadoraPlazoPoliticaOfertasBolsaDesarrollo) PlazoDisposicion(context.Context, time.Time) (puertosbolsa.PlazoOferta, time.Time, error) {
	return puertosbolsa.PlazoOferta{}, time.Time{}, puertosbolsa.ErrOfertaNoDisponible
}

func (c calculadoraPlazoPoliticaOfertasBolsaDesarrollo) PlazoDisposicionBolsa(ctx context.Context, bolsa string, publicada time.Time) (puertosbolsa.PlazoOferta, time.Time, error) {
	if c.servicio == nil {
		return puertosbolsa.PlazoOferta{}, time.Time{}, puertosbolsa.ErrOfertaNoDisponible
	}
	return c.servicio.PlazoDisposicionBolsa(ctx, bolsa, publicada)
}

// preparadorPoliticaOfertasBolsaDesarrollo mantiene la entrada de B47 dentro
// de la sesión y los ámbitos Bolsa ya revalidados por B-BACK.
type preparadorPoliticaOfertasBolsaDesarrollo struct {
	base   *preparadorBorradorLlamamientoDesarrollo
	emisor *emisorBorradorLlamamientoDesarrollo
}

func (p *preparadorPoliticaOfertasBolsaDesarrollo) PrepararConsultaPoliticaOfertas(
	ctx context.Context, bolsaRef string,
) (bolsahttp.ConsultaPoliticaOfertasPreparada, error) {
	contexto, solicitud, err := p.solicitud(ctx, bolsaRef)
	if err != nil {
		return bolsahttp.ConsultaPoliticaOfertasPreparada{}, err
	}
	// El PDP prepara la decisión sin registrar una concesión candidata de
	// escritura durante una lectura. El POST vuelve a autorizar y consume.
	autoridad, ok := p.emisor.politicaOfertas.autoridad.(*autorizadorComunDesarrollo)
	if !ok || autoridad.servicio == nil {
		return bolsahttp.ConsultaPoliticaOfertasPreparada{}, errBorradorNoDisponibleEn()
	}
	ctx, err = autoridad.contextoSolicitud(ctx, solicitud)
	if err != nil {
		return bolsahttp.ConsultaPoliticaOfertasPreparada{}, err
	}
	_, _, err = autoridad.servicio.PrepararSolicitudLigadaV3(ctx, solicitud, contexto.Resultado)
	if errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		return bolsahttp.ConsultaPoliticaOfertasPreparada{BolsaRef: bolsaRef, PuedePublicar: false}, nil
	}
	if err != nil {
		return bolsahttp.ConsultaPoliticaOfertasPreparada{}, err
	}
	return bolsahttp.ConsultaPoliticaOfertasPreparada{BolsaRef: bolsaRef, PuedePublicar: true}, nil
}

func (p *preparadorPoliticaOfertasBolsaDesarrollo) PrepararPublicacionPoliticaOfertas(
	ctx context.Context, entrada bolsahttp.EntradaPublicarPoliticaOfertas,
) (puertosbolsa.ComandoPublicarPoliticaOfertas, error) {
	contexto, solicitud, err := p.solicitud(ctx, entrada.BolsaRef)
	if err != nil {
		return puertosbolsa.ComandoPublicarPoliticaOfertas{}, err
	}
	_, _, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, contexto.Resultado)
	if err != nil || exportador == nil {
		return puertosbolsa.ComandoPublicarPoliticaOfertas{}, errBorradorNoDisponibleEn()
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || material.ValidarEstructura() != nil {
		return puertosbolsa.ComandoPublicarPoliticaOfertas{}, errBorradorNoDisponibleEn()
	}
	return puertosbolsa.ComandoPublicarPoliticaOfertas{
		BolsaRef: entrada.BolsaRef, ActorRef: contexto.Resultado.Contexto.Principal.ID,
		VersionEsperada: entrada.VersionEsperada, ClaveIdempotencia: entrada.ClaveIdempotencia,
		Politica: entrada.Politica, Material: material,
	}, nil
}

func (p *preparadorPoliticaOfertasBolsaDesarrollo) solicitud(
	ctx context.Context, bolsaRef string,
) (contextoSeguridadComunDesarrollo, dominiovec.SolicitudAutorizacionLigadaV3, error) {
	if p == nil || p.base == nil || p.emisor == nil || p.emisor.politicaOfertas == nil ||
		p.emisor.politicaOfertas.autoridad == nil {
		return contextoSeguridadComunDesarrollo{}, dominiovec.SolicitudAutorizacionLigadaV3{}, errBorradorNoDisponibleEn()
	}
	contexto, err := p.base.contextoRevalidado(ctx)
	if err != nil {
		return contextoSeguridadComunDesarrollo{}, dominiovec.SolicitudAutorizacionLigadaV3{}, err
	}
	resuelto, err := p.base.ResolverContextoContactosBolsa(ctx, contexto.Resultado.Contexto, bolsaRef)
	if err != nil || resuelto.Validar() != nil {
		return contextoSeguridadComunDesarrollo{}, dominiovec.SolicitudAutorizacionLigadaV3{}, dominiovec.ErrAutorizacionDenegada
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.base.generar)
	if err != nil {
		return contextoSeguridadComunDesarrollo{}, dominiovec.SolicitudAutorizacionLigadaV3{}, errBorradorNoDisponibleEn()
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: contexto.Vinculo, ReferenciaMotivo: motivoPublicarPoliticaOfertasBolsaDesarrollo(),
		Accion:    puertosbolsa.AccionPublicarPoliticaOfertas,
		Recurso:   dominiovec.RecursoAutorizable{Referencia: bolsaRef, ModuloID: "bolsa", Tipo: "bolsa_constituida", Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}},
		Finalidad: puertosbolsa.FinalidadPoliticaOfertas, Correlacion: correlacion,
	})
	if err != nil {
		return contextoSeguridadComunDesarrollo{}, dominiovec.SolicitudAutorizacionLigadaV3{}, dominiovec.ErrAutorizacionDenegada
	}
	return contexto, solicitud, nil
}

var _ bolsahttp.PreparadorPoliticaOfertas = (*preparadorPoliticaOfertasBolsaDesarrollo)(nil)

// Ofertas publicadas de Bolsa (art. 8.1 del Reglamento). Comparten sesión,
// política, motivo y material con la emisión B7: publicar y resolver son
// modalidades del mismo llamamiento, sin acción ni consumidor nuevos.
const (
	claveFronteraPublicarOfertaBolsa  = "bolsa-bof-oferta-publicar"
	claveFronteraConsultarOfertaBolsa = "bolsa-bof-oferta-consultar"
	claveFronteraResolverOfertaBolsa  = "bolsa-bof-oferta-resolver"
)

func descriptoresFronterasOfertasBolsaDesarrollo(perfilActivoRef string) []descriptorFronteraComunDesarrollo {
	return []descriptorFronteraComunDesarrollo{
		{Clave: claveFronteraPublicarOfertaBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: bolsahttp.RutaOfertasPublicadas, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa},
		{Clave: claveFronteraConsultarOfertaBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaOfertasPublicadas, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa},
		{Clave: claveFronteraResolverOfertaBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: bolsahttp.RutaResolucionesOferta, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa},
	}
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudPublicarOferta(ctx context.Context, entrada bolsahttp.EntradaPublicarOferta) (puertosbolsa.SolicitudPublicarOferta, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudPublicarOferta{}, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return puertosbolsa.SolicitudPublicarOferta{}, err
	}
	return puertosbolsa.SolicitudPublicarOferta{Vinculo: contexto.Vinculo, ResultadoContexto: contexto.Resultado, BolsaRef: entrada.BolsaRef, Datos: entrada.Datos, ClaveIdempotencia: entrada.ClaveIdempotencia, Correlacion: correlacion, MotivoAutorizacion: motivoEmitirLlamamientoBolsaDesarrollo()}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudResolverOferta(ctx context.Context, entrada bolsahttp.EntradaResolverOferta) (puertosbolsa.SolicitudResolverOferta, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudResolverOferta{}, err
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
	if err != nil {
		return puertosbolsa.SolicitudResolverOferta{}, err
	}
	return puertosbolsa.SolicitudResolverOferta{Vinculo: contexto.Vinculo, ResultadoContexto: contexto.Resultado, BolsaRef: entrada.BolsaRef, OfertaRef: entrada.OfertaRef, ParticipacionRef: entrada.ParticipacionRef, ClaveIdempotencia: entrada.ClaveIdempotencia, Correlacion: correlacion, MotivoAutorizacion: motivoEmitirLlamamientoBolsaDesarrollo()}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudConsultarOfertas(ctx context.Context, bolsaRef string, limite int) (puertosbolsa.SolicitudConsultarOfertas, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudConsultarOfertas{}, err
	}
	return puertosbolsa.SolicitudConsultarOfertas{ContextoActor: contexto.Resultado.Contexto, BolsaRef: bolsaRef, Limite: limite}, nil
}

var _ bolsahttp.PreparadorOfertasPublicadas = (*preparadorBorradorLlamamientoDesarrollo)(nil)

// calculadoraPlazoOfertaDesarrollo resuelve la regla b10 del catálogo de
// reglas de Bolsa. El resolutor se fija una sola vez al componer, antes de
// servir; sin él no hay plazo y la publicación se rechaza.
type calculadoraPlazoOfertaDesarrollo struct {
	resolutor atomic.Pointer[reglas.Resolutor]
}

func (c *calculadoraPlazoOfertaDesarrollo) fijar(resolutor *reglas.Resolutor) {
	if c != nil && resolutor != nil {
		c.resolutor.CompareAndSwap(nil, resolutor)
	}
}

func (c *calculadoraPlazoOfertaDesarrollo) PlazoDisposicion(ctx context.Context, publicadaEn time.Time) (puertosbolsa.PlazoOferta, time.Time, error) {
	if c == nil || c.resolutor.Load() == nil {
		return puertosbolsa.PlazoOferta{}, time.Time{}, puertosbolsa.ErrPlazoOfertaNoConfigurado
	}
	regla, vencimiento, err := c.resolutor.Load().Vencimiento(ctx, reglas.BolsaPlazoPublicacion, publicadaEn, "")
	if err != nil {
		if errors.Is(err, reglas.ErrReglaNoEncontrada) || errors.Is(err, reglas.ErrReglasNoConfiguradas) {
			return puertosbolsa.PlazoOferta{}, time.Time{}, errors.Join(puertosbolsa.ErrPlazoOfertaNoConfigurado, err)
		}
		return puertosbolsa.PlazoOferta{}, time.Time{}, errors.Join(puertosbolsa.ErrOfertaNoDisponible, err)
	}
	return puertosbolsa.PlazoOferta{
		ReglaRef: regla.Referencia, HuellaCatalogo: regla.HuellaCatalogo, Unidad: string(regla.Unidad),
		Cantidad: regla.Cantidad, Computo: string(regla.Computo), UltimoDia: vencimiento.UltimoDia,
		Ejemplo: regla.EsEjemplo(), Articulo: regla.Articulo, Calendarios: vencimiento.Calendarios,
	}, vencimiento.VenceAntesDe, nil
}

var _ puertosbolsa.CalculadoraPlazoOferta = (*calculadoraPlazoOfertaDesarrollo)(nil)
