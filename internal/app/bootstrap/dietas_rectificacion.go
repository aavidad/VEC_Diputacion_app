package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	personalcomp "vec-diputacion-granada/internal/modules/personal/adapters/composicion"
	personalhttp "vec-diputacion-granada/internal/modules/personal/adapters/httpinterno"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	core "vec-diputacion-granada/internal/vec/domain"
)

func esRutaRectificacionDietas(ruta string) bool {
	return ruta == personalhttp.RutaSolicitudesRectificacionDietas || strings.HasPrefix(ruta, personalhttp.RutaSolicitudesRectificacionDietas+"/")
}

// La solicitud textual no sustituye la asignación vigente. La confirmación
// completa sigue cerrada mientras D7 no pueda validar a sus destinatarios.
func metodoRectificacionDietasValido(ruta, metodo string) bool {
	if ruta == personalhttp.RutaSolicitudesRectificacionDietas {
		return metodo == http.MethodGet || metodo == http.MethodPost
	}
	return ruta == personalhttp.RutaSolicitudesRectificacionDietas+"/competentes" && metodo == http.MethodGet
}

// Personal 15 es opcional en el arranque heredado. Sólo se monta el circuito
// nuevo cuando su consumidor y su auditoría tienen EXECUTE nominal positivo.
func acreditarRectificacionesDietas(ctx context.Context, pool, auditoria *pgxpool.Pool) (bool, error) {
	if ctx == nil || pool == nil || auditoria == nil {
		return false, ErrComposicionBorradoresDietasNoDisponible
	}
	var instalada bool
	if err := pool.QueryRow(ctx, `SELECT pg_catalog.to_regprocedure('vec_personal.solicitar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL`).Scan(&instalada); err != nil {
		return false, ErrComposicionBorradoresDietasNoDisponible
	}
	if !instalada {
		return false, nil
	}
	funciones := []string{
		"vec_personal.solicitar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
		"vec_personal.consultar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
		"vec_personal.consultar_rectificaciones_competentes_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
	}
	var permitido bool
	if err := pool.QueryRow(ctx, `SELECT COALESCE(bool_and(CASE WHEN pg_catalog.to_regprocedure(f.nombre) IS NULL THEN false ELSE pg_catalog.has_function_privilege(current_user,f.nombre,'EXECUTE') END),false) FROM pg_catalog.unnest($1::text[]) AS f(nombre)`, funciones).Scan(&permitido); err != nil || !permitido {
		return false, ErrComposicionBorradoresDietasNoDisponible
	}
	r, err := personalpg.NuevoRegistradorAuditoriaFronteraRectificacionPostgreSQL(auditoria)
	if err != nil || r.Preflight(ctx) != nil {
		return false, ErrComposicionBorradoresDietasNoDisponible
	}
	return true, nil
}

func nuevoManejadorRectificacionesDietas(i *identidadPersonalDietas, emisor personalcomp.EmisorMaterialRelacionDietasV3, motivo core.ReferenciaEntradaCatalogo, pool *pgxpool.Pool, auditoria personalports.RegistradorAuditoriaFronteraRectificacionDietas) (http.Handler, error) {
	p, err := personalcomp.NuevoProveedorAutorizacionRectificacionDietas(i, emisor, motivo)
	if err != nil {
		return nil, err
	}
	r, err := personalpg.NuevoRepositorioRectificacionDietasPostgreSQL(pool)
	if err != nil {
		return nil, err
	}
	// Sin proveedor de corrección no es posible confirmar otra asignación.
	s, err := personalapp.NuevoServicioRectificacionDietas(p, nil, r)
	if err != nil {
		return nil, err
	}
	pb, err := personalcomp.NuevoProveedorAutorizacionRectificacionesCompetentesDietas(i, emisor, motivo)
	if err != nil {
		return nil, err
	}
	rb, err := personalpg.NuevoRepositorioRectificacionesCompetentesDietasPostgreSQL(pool)
	if err != nil {
		return nil, err
	}
	sb, err := personalapp.NuevoServicioRectificacionesCompetentesDietas(pb, rb)
	if err != nil {
		return nil, err
	}
	m, err := personalhttp.NuevoManejadorRectificacionDietas(i, s, sb, auditoria)
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || !metodoRectificacionDietasValido(r.URL.Path, r.Method) {
			responderDenegacionComisionesDietas(w, http.StatusNotFound)
			return
		}
		ctx := context.WithValue(r.Context(), claveCacheSeguridadComunDesarrollo{}, &cacheSeguridadComunDesarrollo{})
		m.ServeHTTP(w, r.WithContext(ctx))
	}), nil
}

func (a *autoridadComisionesDietasDesarrollo) registrarDenegacionRectificacion(ctx context.Context, ruta, metodo string, estado int, actor string) error {
	if a == nil || ctx == nil || a.registradorRectificacion == nil {
		return ErrComposicionBorradoresDietasNoDisponible
	}
	motivo := personalports.MotivoFronteraPersonalDenegado
	if estado == http.StatusServiceUnavailable {
		motivo = personalports.MotivoFronteraPersonalDependencia
	}
	if estado == http.StatusUnauthorized {
		motivo = personalports.MotivoFronteraPersonalAutenticacion
	}
	if estado == http.StatusForbidden {
		motivo = personalports.MotivoFronteraPersonalDenegado
	}
	accion := "consultar"
	if metodo == http.MethodPost {
		accion = "solicitar"
	}
	if ruta == personalhttp.RutaSolicitudesRectificacionDietas+"/competentes" {
		accion = "consultar_competentes"
	}
	if !metodoRectificacionDietasValido(ruta, metodo) {
		accion = "metodo_no_admitido"
	}
	var aleatorio [16]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		return ErrComposicionBorradoresDietasNoDisponible
	}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
	defer cancelar()
	return a.registradorRectificacion.RegistrarAuditoriaFronteraRectificacionDietas(ctxAuditoria, personalports.OrdenAuditoriaFronteraRectificacionDietas{CorrelacionRef: "corr_" + hex.EncodeToString(aleatorio[:]), Motivo: motivo, Ruta: personalports.RutaSolicitudesRectificacionDietas, Accion: accion, ActorRef: actor, EstadoHTTP: estado})
}
