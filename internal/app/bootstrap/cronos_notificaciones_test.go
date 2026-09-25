package bootstrap

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	cronoscomp "vec-diputacion-granada/internal/modules/cronos/adapters/composicion"
	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
	cronosports "vec-diputacion-granada/internal/modules/cronos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestComponerManejadoresCronosConNotificacionesPublicaDoceRutas(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://cronos_prueba@127.0.0.1:1/nadie?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	canal, _ := cronosdomain.NuevaAcreditacionCanalMarcaje(cronosdomain.DatosAcreditacionCanalMarcaje{PoliticaVersionRef: "politica:canal:cronos:v1", CanalRef: "portal-empleado-web", OrigenRef: cronosdomain.OrigenMarcajeRemoto, CalidadRef: "mtls-certificado"})
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_cronos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("6", 32)}
	emisor := emisorCronosEmpleadoDesarrollo{porAccion: map[string]emisorMaterialDietasDesarrollo{}}
	autorizador, err := cronoscomp.NuevoAutorizadorCronos(emisor, cronoscomp.MotivosCronos{Saldo: motivo, Marcaje: motivo, Disponibilidad: motivo, Recuperacion: motivo, Movimientos: motivo, Correccion: motivo, Permisos: motivo, Permiso: motivo,
		Notificacion: motivo, Notificaciones: motivo, BandejaNotificaciones: motivo, AtencionNotificacion: motivo})
	if err != nil || !autorizador.NotificacionesConfiguradas() || autorizador.ResolucionConfigurada() {
		t.Fatal(err)
	}
	zona, _ := time.LoadLocation("Europe/Madrid")
	identidad := seguridadCronosEmpleadoDesarrollo{autoridad: &autoridadCronosEmpleadoDesarrollo{reloj: relojRutasDietas{}}}
	rutas, err := componerManejadoresCronosEmpleado(dependenciasCronosEmpleado{ejecutor: pool, auditor: pool, identidad: identidad, autorizador: autorizador, canal: canal, zona: zona})
	if err != nil || len(rutas) != 12 {
		t.Fatal(len(rutas), err)
	}
	for _, ruta := range []string{cronoshttp.RutaNotificacionesPropias, cronoshttp.RutaBandejaNotificaciones} {
		if !strings.HasPrefix(ruta, prefijoRutasCronosEmpleado) {
			t.Fatal("ruta fuera del prefijo", ruta)
		}
		w := httptest.NewRecorder()
		rutas[ruta].ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta, nil))
		if w.Code != http.StatusServiceUnavailable {
			t.Fatal("manejador sin identidad no falla cerrado", ruta, w.Code)
		}
	}
	for _, ruta := range []string{cronoshttp.RutaRegistrarNotificacion, cronoshttp.RutaAtenderNotificacion} {
		if rutas[ruta] == nil {
			t.Fatal("ruta ausente", ruta)
		}
	}
	if rutas[cronoshttp.RutaBandejaPermisos] != nil {
		t.Fatal("publica la resolución sin sus motivos")
	}
	if _, err := PrepararManejadoresCronos(DependenciasManejadoresCronos{NotificacionesPropias: &cronosapp.ServicioNotificacionesPropias{}}); !errors.Is(err, ErrManejadoresCronosNoDisponibles) {
		t.Fatal("prepara manejadores con las notificaciones incompletas", err)
	}
	for _, ruta := range []string{cronoshttp.RutaNotificacionesPropias, cronoshttp.RutaRegistrarNotificacion, cronoshttp.RutaBandejaNotificaciones, cronoshttp.RutaAtenderNotificacion,
		cronoshttp.RutaBandejaPermisos, cronoshttp.RutaResolverPermiso, cronoshttp.RutaAvisosPropios, cronoshttp.RutaArchivarAviso} {
		o := cronosports.OrdenDenegacionFronteraCronos{CorrelacionRef: "corr_no_disponible", Motivo: cronosports.MotivoFronteraAccesoDenegado, Ruta: ruta, Metodo: "GET"}
		if o.Validar() != nil {
			t.Fatal("la frontera no puede auditar la denegación de", ruta)
		}
	}
}

func TestCronosNotificacionesSinCronosFallaCerrado(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	cfg.CronosEmpleadoEnabled = "true"
	cfg.CronosNotificacionesEnabled = "si"
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nuevasRutasCronosEmpleadoDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, materialCronosDesdeCTDesarrollo{}); !errors.Is(err, config.ErrConfiguracionCronosNotificacionesSelector) {
		t.Fatal("selector de notificaciones no canónico aceptado", err)
	}
	var proveedores [8]*proveedorMaterialAltaContratacionTemporalDesarrollo
	for i := range proveedores {
		proveedores[i] = &proveedorMaterialAltaContratacionTemporalDesarrollo{}
	}
	cfg.CronosNotificacionesEnabled = "true"
	if _, err := nuevasRutasCronosEmpleadoDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, materialCronosDesdeProveedores(proveedores)); !errors.Is(err, ErrComposicionCronosEmpleadoNoDisponible) {
		t.Fatal("arranca las notificaciones sin su material V3", err)
	}
	if !cronosNotificacionesSolicitadas("true", "true") || cronosNotificacionesSolicitadas("false", "true") || cronosNotificacionesSolicitadas("true", "") {
		t.Fatal("selector combinado distinto")
	}
}

func TestAudienciasNotificacionesCronosPublicablesPorElGobiernoCT(t *testing.T) {
	descriptores := descriptoresMaterialCronosNotificacionesDesarrollo()
	audiencias := audienciasCronosNotificacionesDesarrollo()
	if len(descriptores) != len(audiencias) {
		t.Fatal("descriptores y audiencias de las notificaciones divergen")
	}
	for i, d := range descriptores {
		if d.Audiencia != audiencias[i] || !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
			t.Fatalf("audiencia de las notificaciones no publicable por CT: %s", d.Audiencia)
		}
	}
	todos := append(append(append(descriptoresMaterialDietasDesarrollo(), descriptoresMaterialCronosDesarrollo()...), descriptoresMaterialCronosResolucionDesarrollo()...), descriptores...)
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(todos); err != nil {
		t.Fatal("las notificaciones colisionan en el catálogo común", err)
	}
}
