package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	calports "vec-diputacion-granada/internal/modules/calendarios/ports"
	cronoscomp "vec-diputacion-granada/internal/modules/cronos/adapters/composicion"
	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronospg "vec-diputacion-granada/internal/modules/cronos/adapters/postgres"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrMontajeCalendarioLaboralC4 = errors.New("bootstrap: calendario laboral C4 no disponible")

func descriptorMaterialUnidadTrabajoC4() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        personalports.AudienciaUnidadTrabajoPropiaCronos,
		Dominio:          "vec.personal.unidad-trabajo-cronos.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:personal-unidad-cronos:",
		ProveedorNominal: "proveedor-material-personal-unidad-cronos",
	}
}

type configuracionCalendarioLaboralC4 struct {
	redaccionMaterialRutasDietas
	Version                   int                                 `json:"version"`
	Autoridad                 string                              `json:"autoridad"`
	DSNPersonal               string                              `json:"dsn_personal"`
	MotivoUnidad              vecdomain.ReferenciaEntradaCatalogo `json:"motivo_unidad"`
	MotivoOrganizacion        vecdomain.ReferenciaEntradaCatalogo `json:"motivo_organizacion"`
	MotivoIntentoDenegado     vecdomain.ReferenciaEntradaCatalogo `json:"motivo_intento_denegado"`
	MotivoIntentoInvalido     vecdomain.ReferenciaEntradaCatalogo `json:"motivo_intento_invalido"`
	MotivoIntentoNoDisponible vecdomain.ReferenciaEntradaCatalogo `json:"motivo_intento_no_disponible"`
}

type validadorMotivosCalendarioC4 interface {
	ValidarReferenciaMotivoAutorizacionV2(context.Context, vecdomain.ReferenciaEntradaCatalogo, time.Time) error
}

// prepararCalendarioLaboralC4 devuelve el pool sólo si la ruta completa queda
// publicada. Dirección conserva su cierre con los demás pools de Cronos.
func prepararCalendarioLaboralC4(ctx context.Context, cfg config.Config, selector string, rutas map[string]http.Handler,
	a *autoridadCronosEmpleadoDesarrollo, calendarios calports.ConsultaCalendarios, ejecutor *pgxpool.Pool,
	autorizador *cronoscomp.AutorizadorCronos, canal cronosdomain.AcreditacionCanalMarcaje,
	emisor emisorCronosEmpleadoDesarrollo, validador validadorMotivosCalendarioC4, usuarios map[string]bool) (*pgxpool.Pool, string, error) {
	activo, err := config.CronosCalendarioC4DesarrolloActivo(cfg, selector)
	if err != nil {
		return nil, "", ErrMontajeCalendarioLaboralC4
	}
	if !activo {
		return nil, "", nil
	}
	if ctx == nil || a == nil || a.reloj == nil || calendarios == nil || ejecutor == nil || autorizador == nil || validador == nil || usuarios == nil ||
		emisor.porAccion[personalports.AccionUnidadTrabajoPropiaCronos] == nil || emisor.porAccion[personaldomain.AccionConsultaOrganizacionHistorica] == nil {
		return nil, "", ErrMontajeCalendarioLaboralC4
	}
	c, err := leerConfiguracionCalendarioLaboralC4(cfg)
	if err != nil {
		return nil, "", ErrMontajeCalendarioLaboralC4
	}
	for _, m := range []vecdomain.ReferenciaEntradaCatalogo{c.MotivoUnidad, c.MotivoOrganizacion, c.MotivoIntentoDenegado, c.MotivoIntentoInvalido, c.MotivoIntentoNoDisponible} {
		if validador.ValidarReferenciaMotivoAutorizacionV2(ctx, m, a.reloj.Ahora()) != nil {
			return nil, "", ErrMontajeCalendarioLaboralC4
		}
	}
	pool, usuario, err := abrirPoolRutasDietas(ctx, c.DSNPersonal, "vec_personal_ejecutor")
	if err != nil {
		return nil, "", ErrMontajeCalendarioLaboralC4
	}
	completo := false
	defer func() {
		if !completo {
			pool.Close()
		}
	}()
	if usuarios[usuario] {
		return nil, "", ErrMontajeCalendarioLaboralC4
	}
	personal, err := nuevoLectorUnidadTrabajoCalendarioC4(ctx, pool, a, emisor, c.MotivoUnidad)
	if err != nil {
		return nil, "", ErrMontajeCalendarioLaboralC4
	}
	registro, ok := a.auditoriaIntentos.(intentoOrganizacionC4Registrador)
	if !ok {
		return nil, "", ErrMontajeCalendarioLaboralC4
	}
	organizacion, err := nuevoLectorOrganizacionCalendarioC4(ctx, pool, a, emisor, c.MotivoOrganizacion, registro,
		motivosIntentoOrganizacionC4{c.MotivoIntentoDenegado, c.MotivoIntentoInvalido, c.MotivoIntentoNoDisponible})
	if err != nil {
		return nil, "", ErrMontajeCalendarioLaboralC4
	}
	if montarCalendarioLaboralC4(ctx, rutas, a, personal, organizacion, calendarios, ejecutor, autorizador, canal) != nil {
		return nil, "", ErrMontajeCalendarioLaboralC4
	}
	completo = true
	return pool, usuario, nil
}

func leerConfiguracionCalendarioLaboralC4(cfg config.Config) (configuracionCalendarioLaboralC4, error) {
	var vacia configuracionCalendarioLaboralC4
	contenido, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "cronos-calendario-c4.json"), 32<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return vacia, ErrMontajeCalendarioLaboralC4
	}
	defer borrarBytes(contenido)
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var c configuracionCalendarioLaboralC4
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(new(any)), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa || c.DSNPersonal == "" ||
		!vecdomain.ReferenciaMotivoAutorizacionV2Valida(c.MotivoUnidad) || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(c.MotivoOrganizacion) ||
		!vecdomain.ReferenciaMotivoAutorizacionV2Valida(c.MotivoIntentoDenegado) || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(c.MotivoIntentoInvalido) ||
		!vecdomain.ReferenciaMotivoAutorizacionV2Valida(c.MotivoIntentoNoDisponible) ||
		c.MotivoUnidad.CatalogoID != c.MotivoOrganizacion.CatalogoID || c.MotivoUnidad.CatalogoID != c.MotivoIntentoDenegado.CatalogoID ||
		c.MotivoUnidad.CatalogoID != c.MotivoIntentoInvalido.CatalogoID || c.MotivoUnidad.CatalogoID != c.MotivoIntentoNoDisponible.CatalogoID {
		return vacia, ErrMontajeCalendarioLaboralC4
	}
	return c, nil
}

type emisorUnidadC4 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type proveedorUnidadTrabajoC4 struct {
	autoridad *autoridadCronosEmpleadoDesarrollo
	emisor    emisorUnidadC4
	motivo    vecdomain.ReferenciaEntradaCatalogo
}

func (p proveedorUnidadTrabajoC4) AutorizarUnidadTrabajoPropiaCronos(ctx context.Context, m personaldomain.MaterialUnidadTrabajoCronos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var vacio vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if ctx == nil || p.autoridad == nil || p.emisor == nil || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(p.motivo) || len(m.Canonico()) == 0 {
		return vacio, personaldomain.ErrUnidadTrabajoCronosNoDisponible
	}
	id, err := (seguridadCronosEmpleadoDesarrollo{autoridad: p.autoridad}).ResolverIdentidadRegistradaCronos(ctx)
	if err != nil {
		return vacio, personaldomain.ErrUnidadTrabajoCronosNoDisponible
	}
	actorCanon, err := m.Actor().RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(actorCanon, id.Contexto.RepresentacionCanonica) {
		return vacio, personaldomain.ErrUnidadTrabajoCronosNoDisponible
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, personaldomain.ErrUnidadTrabajoCronosNoDisponible
	}
	sol, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: id.Vinculo, ReferenciaMotivo: p.motivo, Accion: personalports.AccionUnidadTrabajoPropiaCronos,
		Recurso: m.Recurso(), Finalidad: personaldomain.FinalidadUnidadTrabajoPropiaCronos, Correlacion: correlacion,
	})
	if err != nil {
		return vacio, personaldomain.ErrUnidadTrabajoCronosNoDisponible
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, sol, id.Contexto)
	if err != nil {
		if errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return vacio, personaldomain.ErrUnidadTrabajoCronosDenegada
		}
		return vacio, personaldomain.ErrUnidadTrabajoCronosNoDisponible
	}
	if decision.ValidarPara(sol) != nil || exportador == nil {
		return vacio, personaldomain.ErrUnidadTrabajoCronosNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(sol, decision, confirmacion, id.Contexto, p.motivo, material, personalports.AudienciaUnidadTrabajoPropiaCronos) {
		return vacio, personaldomain.ErrUnidadTrabajoCronosNoDisponible
	}
	return material, nil
}

func nuevoLectorUnidadTrabajoCalendarioC4(ctx context.Context, pool *pgxpool.Pool, autoridad *autoridadCronosEmpleadoDesarrollo,
	emisor emisorUnidadC4, motivo vecdomain.ReferenciaEntradaCatalogo) (personalports.LectorUnidadTrabajoPropiaCronos, error) {
	if ctx == nil || pool == nil || autoridad == nil || emisor == nil || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivo) || personalpg.PreflightEjecutorUnidadTrabajoCronos(ctx, pool) != nil {
		return nil, ErrMontajeCalendarioLaboralC4
	}
	repo, err := personalpg.NuevoRepositorioUnidadTrabajoCronosPostgreSQL(pool)
	if err != nil {
		return nil, ErrMontajeCalendarioLaboralC4
	}
	s, err := personalapp.NuevoServicioUnidadTrabajoCronos(proveedorUnidadTrabajoC4{autoridad, emisor, motivo}, repo)
	if err != nil {
		return nil, ErrMontajeCalendarioLaboralC4
	}
	return s, nil
}

type lectorOrganizacionCalendarioC4 interface {
	ConsultarC4(ctx context.Context, solicitud personaldomain.SolicitudConsultaOrganizacionHistorica, unidad personalports.ResultadoUnidadTrabajoPropiaCronos) (personalports.ResultadoConsultaOrganizacionHistorica, error)
}

// montarCalendarioLaboralC4 sólo publica la ruta cuando las cuatro fuentes
// y la identidad de la frontera registrada están compuestas. La lectura de
// programación reutiliza la misma autoridad V3 nominal que saldo propio.
func montarCalendarioLaboralC4(ctx context.Context, rutas map[string]http.Handler, autoridad *autoridadCronosEmpleadoDesarrollo,
	personal personalports.LectorUnidadTrabajoPropiaCronos, organizacion lectorOrganizacionCalendarioC4,
	calendarios calports.ConsultaCalendarios, ejecutor *pgxpool.Pool,
	autorizador *cronoscomp.AutorizadorCronos, canal cronosdomain.AcreditacionCanalMarcaje) error {
	if ctx == nil || rutas == nil || autoridad == nil || autoridad.base == nil || personal == nil || organizacion == nil || calendarios == nil || ejecutor == nil || autorizador == nil || cronospg.PreflightProgramacionLaboralC4(ctx, ejecutor) != nil {
		return ErrMontajeCalendarioLaboralC4
	}
	if _, existe := rutas[cronoshttp.RutaCalendarioLaboralPropioC4]; existe {
		return ErrMontajeCalendarioLaboralC4
	}
	resolutor, err := cronoscomp.NuevoResolutorPeticionCronos(seguridadCronosEmpleadoDesarrollo{autoridad: autoridad}, autorizador, canal)
	if err != nil {
		return ErrMontajeCalendarioLaboralC4
	}
	programacion, err := cronospg.NuevoRepositorioProgramacionLaboralC4(ejecutor)
	if err != nil {
		return ErrMontajeCalendarioLaboralC4
	}
	servicio, err := cronosapp.NuevoServicioCalendarioLaboralC4ConProgramacion(personal, organizacion, calendarios, relojCronosEmpleado{}, programacion)
	if err != nil {
		return ErrMontajeCalendarioLaboralC4
	}
	manejador, err := cronoshttp.NuevoManejadorCalendarioLaboralC4(servicio, resolutor)
	if err != nil {
		return ErrMontajeCalendarioLaboralC4
	}
	rutas[cronoshttp.RutaCalendarioLaboralPropioC4] = manejador
	return nil
}
