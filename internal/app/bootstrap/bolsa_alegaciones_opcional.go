package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	httpinscripcion "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinscripcion"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

// La ausencia de material o motivo de Alegaciones no impide Inscripción. El
// diagnóstico usa sólo una etapa cerrada: ningún valor privado llega al log.
func registrarAlegacionesBolsaNoDisponibles(etapa string) {
	slog.Warn("bolsa alegaciones: montaje no disponible", "etapa", etapa)
}

// El constructor conserva sólo los descriptores y materiales acreditados.
// Esta segunda barrera rechaza ambas acciones de Alegaciones antes del PDP
// cuando su configuración opcional no está acreditada.
type autoridadInscripcionSinAlegaciones struct{ AutoridadInscripcionBolsa }

func (a autoridadInscripcionSinAlegaciones) CapturarLectura(ctx context.Context, s contextoSeguridadComunDesarrollo,
	acreditacion AcreditacionSesionInscripcionBolsa, accion, recurso string, filtro inscripcion.Filtro) (inscripcion.CapturaLectura, error) {
	if accion == inscripcion.AccionObjetosAlegacion {
		return inscripcion.CapturaLectura{}, inscripcion.ErrNoDisponible
	}
	return a.AutoridadInscripcionBolsa.CapturarLectura(ctx, s, acreditacion, accion, recurso, filtro)
}

func (a autoridadInscripcionSinAlegaciones) AutorizarEscritura(ctx context.Context, s contextoSeguridadComunDesarrollo,
	acreditacion AcreditacionSesionInscripcionBolsa, accion, recurso string, material []byte) (AutorizacionEscrituraInscripcionBolsa, error) {
	if accion == inscripcion.AccionPresentarAlegacion {
		return AutorizacionEscrituraInscripcionBolsa{}, inscripcion.ErrNoDisponible
	}
	return a.AutoridadInscripcionBolsa.AutorizarEscritura(ctx, s, acreditacion, accion, recurso, material)
}

func cerrarRutasAlegacionesExterno(manejador http.Handler) http.Handler {
	noDisponible := consultaPublicaPersonalNoDisponible("servicio_no_disponible")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.URL != nil && (r.URL.Path == httpinscripcion.RutaAlegacionesPropias ||
			strings.HasPrefix(r.URL.Path, httpinscripcion.RutaAlegacionesPropias+"/")) {
			noDisponible.ServeHTTP(w, r)
			return
		}
		manejador.ServeHTTP(w, r)
	})
}

// Si el segundo fichero de un inventario válido aún no está disponible,
// reconstruye únicamente la primera audiencia mediante los mismos cotejos
// de configuración, clave y raíz. Un fallo de la primera sigue siendo fatal.
func proveedorBaseInscripcionPortalExterno(ctx context.Context, directorio string, pool *pgxpool.Pool,
	reloj relojContratacionTemporalDesarrollo) (*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	inv, err := leerInventarioV3PortalExterno(directorio)
	if err != nil || len(inv.Consumidores[consumidorInscripcionPortalExternoV3]) < 1 {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	inv.Consumidores[consumidorInscripcionPortalExternoV3] = inv.Consumidores[consumidorInscripcionPortalExternoV3][:1]
	vigente, err := leerConfiguracionV3PortalExterno(ctx, pool, inv, consumidorInscripcionPortalExternoV3, inv.Configuracion)
	if err != nil || comprobarMaterialV3PortalExterno(ctx, pool, inv, consumidorInscripcionPortalExternoV3, vigente) != nil {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	materiales, err := materialesConsumidorV3PortalExterno(directorio, inv, consumidorInscripcionPortalExternoV3, vigente)
	if err != nil || len(materiales) != 1 || materiales[0].audienciaConsumo != audienciaPresentarInscripcionExternaV3 {
		borrarMaterialesV3PortalExterno(materiales)
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	defer borrarMaterialesV3PortalExterno(materiales)
	fuente, err := nuevaFuenteConfianzaPortalExterno(pool, inv, consumidorInscripcionPortalExternoV3, materiales[0], reloj)
	if err != nil {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	materiales[0].fuenteConfianza = fuente
	return nuevoProveedorMaterialAutorizacionBaseDesarrollo(materiales[0], reloj)
}
