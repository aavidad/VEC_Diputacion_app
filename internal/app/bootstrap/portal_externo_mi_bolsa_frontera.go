package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	mibolsa "vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecmemory "vec-diputacion-granada/internal/vec/adapters/memory"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type motivosMiBolsaPortalExterno map[string]*vecpg.ValidadorReferenciaMotivoPostgreSQLV2

func nuevosMotivosMiBolsaPortalExterno(pool *pgxpool.Pool) (motivosMiBolsaPortalExterno, error) {
	m := motivosMiBolsaPortalExterno{}
	for _, ref := range []core.ReferenciaEntradaCatalogo{motivoMiBolsaDesarrollo(), motivoHistorialMiBolsaDesarrollo(), motivoPortalMiBolsaDesarrollo()} {
		v, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2Externo(pool, ref.CatalogoID)
		if err != nil {
			return nil, errMiBolsaNoDisponible
		}
		m[ref.CatalogoID] = v
	}
	return m, nil
}

func (m motivosMiBolsaPortalExterno) ValidarReferenciaMotivoAutorizacionV2(ctx context.Context,
	ref core.ReferenciaEntradaCatalogo, ahora time.Time,
) error {
	if m[ref.CatalogoID] == nil {
		return core.ErrAutorizacionDenegada
	}
	return m[ref.CatalogoID].ValidarReferenciaMotivoAutorizacionV2(ctx, ref, ahora)
}

type claveOrdenMiBolsaPortalExterno struct{}
type contextoMiBolsaPortalExterno struct {
	autoridad *fronteraMiBolsaPortalExterno
	ruta      string
	orden     mibolsa.Orden
	err       error
}

// La orden solo se crea delante del despachador. El manejador del módulo y
// la autoridad de ruta reciben la misma orden, sin volver a registrar sesión.
type fronteraMiBolsaPortalExterno struct{ preparador bolsahttp.Preparador }

func (a *fronteraMiBolsaPortalExterno) PrepararMiBolsa(r *http.Request) (mibolsa.Orden, error) {
	if r == nil || r.URL == nil {
		return mibolsa.Orden{}, bolsahttp.ErrAutenticacionAusente
	}
	c, ok := r.Context().Value(claveOrdenMiBolsaPortalExterno{}).(contextoMiBolsaPortalExterno)
	if !ok || c.autoridad != a || c.ruta != r.URL.Path || c.err != nil {
		return mibolsa.Orden{}, errMiBolsaNoDisponible
	}
	return c.orden, nil
}

func (a *fronteraMiBolsaPortalExterno) AutorizarRutaExacta(ctx context.Context, ruta string) error {
	if ctx == nil || !bolsahttp.EsRutaPortal(ruta) {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	c, ok := ctx.Value(claveOrdenMiBolsaPortalExterno{}).(contextoMiBolsaPortalExterno)
	if !ok || c.autoridad != a || c.ruta != ruta {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	if errors.Is(c.err, bolsahttp.ErrAutenticacionAusente) {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	if c.err != nil || c.orden.ResultadoContexto.Validar() != nil {
		return vechttp.ErrAutoridadRutaExactaNoDisponible
	}
	if c.orden.Vinculo.ValidarPara(c.orden.ResultadoContexto) != nil ||
		!c.orden.Vinculo.VigenteEn(time.Now().UTC(), c.orden.ResultadoContexto) {
		return vechttp.ErrAccesoRutaExactaDenegado
	}
	return nil
}

func (a *fronteraMiBolsaPortalExterno) proteger(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || !bolsahttp.EsRutaPortal(r.URL.Path) {
			siguiente.ServeHTTP(w, r)
			return
		}
		orden, err := a.preparador.PrepararMiBolsa(r)
		ctx := context.WithValue(r.Context(), claveOrdenMiBolsaPortalExterno{}, contextoMiBolsaPortalExterno{autoridad: a, ruta: r.URL.Path, orden: orden, err: err})
		if err == nil {
			if conActor, e := vechttp.ConActorVerificadoAuditoriaBolsa(ctx, orden.ResultadoContexto.Contexto); e == nil {
				ctx = conActor
			}
		}
		siguiente.ServeHTTP(w, r.WithContext(ctx))
	})
}

func nuevaAPIMiBolsaPortalExterno(identidad *resolvedorIdentidadDesarrollo,
	incidencias vecports.EmisorIncidenciasTecnicas, auditoria vecports.RegistradorAuditoriaFronteraRutaExacta,
	d dependenciasMiBolsaPortalExterno,
) (http.Handler, error) {
	if identidad == nil || incidencias == nil || auditoria == nil || d.preparador == nil {
		return nil, errMiBolsaNoDisponible
	}
	frontera := &fronteraMiBolsaPortalExterno{preparador: d.preparador}
	d.preparador = frontera
	rutas, err := nuevasRutasMiBolsaPortalExterno(d)
	if err != nil {
		return nil, err
	}
	almacen := vecmemory.NewStore()
	servicio, _, err := vecapp.NewServiceWithInternalOperations(almacen, almacen, almacen)
	if err != nil {
		return nil, err
	}
	h, err := vechttp.NewHandlerWithOptions(servicio, vechttp.HandlerOptions{AllowDemoIdentity: true,
		DemoIdentityResolver: identidad, RutasExactas: rutas, AutoridadRutasExactas: frontera,
		RegistradorAuditoriaFronteraRutasExactas: auditoria, EmisorIncidenciasTecnicas: incidencias})
	if err != nil {
		return nil, err
	}
	return frontera.proteger(h), nil
}
