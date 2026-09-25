package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
	dp "vec-diputacion-granada/internal/modules/dietas/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type registroDenegacionCronosPrueba struct {
	ordenes []cronosports.OrdenDenegacionFronteraCronos
	err     error
}

func (r *registroDenegacionCronosPrueba) RegistrarDenegacionFronteraCronos(_ context.Context, o cronosports.OrdenDenegacionFronteraCronos) error {
	if o.Validar() != nil {
		return cronosports.ErrDenegacionFronteraNoRegistrada
	}
	r.ordenes = append(r.ordenes, o)
	return r.err
}

// resolutorContextoCronosPrueba envuelve el resolutor de sesión de prueba y
// permite simular las dos denegaciones cerradas de la proyección {empleado}.
type resolutorContextoCronosPrueba struct {
	base   *resolutorSesionConsultaPrueba
	motivo error
}

func (r *resolutorContextoCronosPrueba) ResolverContextoActorRegistradoV2(ctx context.Context, s core.SolicitudContextoActor) (core.ResultadoContextoActorRegistradoV2, error) {
	if r.motivo != nil {
		return core.ResultadoContextoActorRegistradoV2{}, errors.Join(core.ErrVinculoAutenticacionActorV2Invalido, core.ErrContextoActorNoResuelto, r.motivo)
	}
	return r.base.ResolverContextoActorRegistradoV2(ctx, s)
}

func TestCronosEmpleadoApagadoPorDefectoNoTocaLaRaiz(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	a, err := nuevasRutasCronosEmpleadoDesarrollo(cfg, nil, nil, materialCronosDesdeCTDesarrollo{})
	if err != nil || a != nil {
		t.Fatal("con el selector apagado no debe componerse nada", err)
	}
	raiz := http.NotFoundHandler()
	if h := componerRaizConCronosEmpleado(raiz, nil); h == nil {
		t.Fatal("raíz perdida")
	} else if _, ok := h.(*http.ServeMux); ok {
		t.Fatal("la raíz cambió sin Cronos")
	}
}

func TestCronosEmpleadoActivadoFallaCerradoSinDependencias(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	cfg.CronosEmpleadoEnabled = "si"
	if _, err := nuevasRutasCronosEmpleadoDesarrollo(cfg, nil, nil, materialCronosDesdeCTDesarrollo{}); !errors.Is(err, config.ErrConfiguracionCronosEmpleadoSelector) {
		t.Fatal("selector no canónico aceptado", err)
	}
	cfg.CronosEmpleadoEnabled = "true"
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nuevasRutasCronosEmpleadoDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, materialCronosDesdeCTDesarrollo{}); !errors.Is(err, ErrComposicionCronosEmpleadoNoDisponible) {
		t.Fatal("arranca sin material V3 de Cronos", err)
	}
	var proveedores [8]*proveedorMaterialAltaContratacionTemporalDesarrollo
	for i := range proveedores {
		proveedores[i] = &proveedorMaterialAltaContratacionTemporalDesarrollo{}
	}
	completo := materialCronosDesdeProveedores(proveedores)
	if !completo.completo() {
		t.Fatal("ocho proveedores no componen el material de Cronos")
	}
	proveedores[7] = nil
	if materialCronosDesdeProveedores(proveedores).completo() {
		t.Fatal("material de Cronos completo sin el proveedor de solicitud de permiso")
	}
	proveedores[7] = completo.solicitudPermiso
	if _, err := nuevasRutasCronosEmpleadoDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, completo); !errors.Is(err, ErrComposicionCronosEmpleadoNoDisponible) {
		t.Fatal("arranca sin configuración privada de Cronos", err)
	}
	cfg.CronosEmpleadoEnabled = "true"
	cfg.DevelopmentGuard = ""
	if _, err := nuevasRutasCronosEmpleadoDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, completo); !errors.Is(err, config.ErrConfiguracionCronosEmpleadoActivacion) {
		t.Fatal("arranca fuera de la doble llave", err)
	}
}

func TestComponerManejadoresCronosEmpleadoPublicaSoloOchoRutas(t *testing.T) {
	if _, err := componerManejadoresCronosEmpleado(dependenciasCronosEmpleado{}); !errors.Is(err, ErrComposicionCronosEmpleadoNoDisponible) {
		t.Fatal("compone sin dependencias", err)
	}
	// Pools perezosos: la composición no abre conexiones ni lee datos.
	pool, err := pgxpool.New(context.Background(), "postgres://cronos_prueba@127.0.0.1:1/nadie?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	canal, _ := cronosdomain.NuevaAcreditacionCanalMarcaje(cronosdomain.DatosAcreditacionCanalMarcaje{PoliticaVersionRef: "politica:canal:cronos:v1", CanalRef: "portal-empleado-web", OrigenRef: cronosdomain.OrigenMarcajeRemoto, CalidadRef: "mtls-certificado"})
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_cronos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("5", 32)}
	emisor := emisorCronosEmpleadoDesarrollo{porAccion: map[string]emisorMaterialDietasDesarrollo{}}
	autorizador, err := cronoscomp.NuevoAutorizadorCronos(emisor, cronoscomp.MotivosCronos{Saldo: motivo, Marcaje: motivo, Disponibilidad: motivo, Recuperacion: motivo, Movimientos: motivo, Correccion: motivo, Permisos: motivo, Permiso: motivo})
	if err != nil {
		t.Fatal(err)
	}
	zona, _ := time.LoadLocation("Europe/Madrid")
	identidad := seguridadCronosEmpleadoDesarrollo{autoridad: &autoridadCronosEmpleadoDesarrollo{reloj: relojRutasDietas{}}}
	rutas, err := componerManejadoresCronosEmpleado(dependenciasCronosEmpleado{ejecutor: pool, auditor: pool, identidad: identidad, autorizador: autorizador, canal: canal, zona: zona})
	if err != nil || len(rutas) != 8 {
		t.Fatal(len(rutas), err)
	}
	for _, ruta := range []string{cronoshttp.RutaConsultarSaldoPropio, cronoshttp.RutaRegistrarMarcajeRemoto, cronoshttp.RutaDisponibilidadMarcajeRemoto, cronoshttp.RutaRecuperarReciboMarcajeRemoto,
		cronoshttp.RutaConsultarMovimientosPropios, cronoshttp.RutaSolicitarCorreccionPropia, cronoshttp.RutaConsultarPermisosPropios, cronoshttp.RutaSolicitarPermisoPropio} {
		if rutas[ruta] == nil || !strings.HasPrefix(ruta, prefijoRutasCronosEmpleado) {
			t.Fatalf("ruta %s ausente", ruta)
		}
	}
	// Sin identidad registrada en la petición, ningún manejador llega a datos.
	w := httptest.NewRecorder()
	rutas[cronoshttp.RutaConsultarSaldoPropio].ServeHTTP(w, httptest.NewRequest(http.MethodGet, cronoshttp.RutaConsultarSaldoPropio+"?periodo=hoy", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatal("manejador sin identidad no falla cerrado", w.Code)
	}
	for ruta, peticion := range map[string]*http.Request{
		cronoshttp.RutaConsultarMovimientosPropios: httptest.NewRequest(http.MethodGet, cronoshttp.RutaConsultarMovimientosPropios+"?periodo=anio", nil),
		cronoshttp.RutaConsultarPermisosPropios:    httptest.NewRequest(http.MethodGet, cronoshttp.RutaConsultarPermisosPropios, nil),
	} {
		w := httptest.NewRecorder()
		rutas[ruta].ServeHTTP(w, peticion)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatal("manejador sin identidad no falla cerrado", ruta, w.Code)
		}
	}
}

func TestComponerManejadoresCronosConResolucionPublicaDoceRutas(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://cronos_prueba@127.0.0.1:1/nadie?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	canal, _ := cronosdomain.NuevaAcreditacionCanalMarcaje(cronosdomain.DatosAcreditacionCanalMarcaje{PoliticaVersionRef: "politica:canal:cronos:v1", CanalRef: "portal-empleado-web", OrigenRef: cronosdomain.OrigenMarcajeRemoto, CalidadRef: "mtls-certificado"})
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_cronos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("5", 32)}
	emisor := emisorCronosEmpleadoDesarrollo{porAccion: map[string]emisorMaterialDietasDesarrollo{}}
	autorizador, err := cronoscomp.NuevoAutorizadorCronos(emisor, cronoscomp.MotivosCronos{Saldo: motivo, Marcaje: motivo, Disponibilidad: motivo, Recuperacion: motivo, Movimientos: motivo, Correccion: motivo, Permisos: motivo, Permiso: motivo,
		Bandeja: motivo, Resolucion: motivo, Avisos: motivo, ArchivoAviso: motivo})
	if err != nil || !autorizador.ResolucionConfigurada() {
		t.Fatal(err)
	}
	zona, _ := time.LoadLocation("Europe/Madrid")
	identidad := seguridadCronosEmpleadoDesarrollo{autoridad: &autoridadCronosEmpleadoDesarrollo{reloj: relojRutasDietas{}}}
	rutas, err := componerManejadoresCronosEmpleado(dependenciasCronosEmpleado{ejecutor: pool, auditor: pool, identidad: identidad, autorizador: autorizador, canal: canal, zona: zona})
	if err != nil || len(rutas) != 12 {
		t.Fatal(len(rutas), err)
	}
	for ruta, peticion := range map[string]*http.Request{
		cronoshttp.RutaBandejaPermisos: httptest.NewRequest(http.MethodGet, cronoshttp.RutaBandejaPermisos+"?paso=responsable", nil),
		cronoshttp.RutaAvisosPropios:   httptest.NewRequest(http.MethodGet, cronoshttp.RutaAvisosPropios, nil),
	} {
		if !strings.HasPrefix(ruta, prefijoRutasCronosEmpleado) {
			t.Fatal("ruta fuera del prefijo", ruta)
		}
		w := httptest.NewRecorder()
		rutas[ruta].ServeHTTP(w, peticion)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatal("manejador sin identidad no falla cerrado", ruta, w.Code)
		}
	}
	for _, ruta := range []string{cronoshttp.RutaResolverPermiso, cronoshttp.RutaArchivarAviso} {
		if rutas[ruta] == nil {
			t.Fatal("ruta ausente", ruta)
		}
	}
	// Manejadores de resolución a medias: no se publica nada.
	if _, err := PrepararManejadoresCronos(DependenciasManejadoresCronos{Resolucion: &cronosapp.ServicioResolucionPermisos{}}); !errors.Is(err, ErrManejadoresCronosNoDisponibles) {
		t.Fatal("prepara manejadores con la resolución incompleta", err)
	}
}

func TestCronosResolucionSinCronosFallaCerrado(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	cfg.CronosEmpleadoEnabled = "true"
	cfg.CronosResolucionEnabled = "si"
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nuevasRutasCronosEmpleadoDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, materialCronosDesdeCTDesarrollo{}); !errors.Is(err, config.ErrConfiguracionCronosResolucionSelector) {
		t.Fatal("selector de resolución no canónico aceptado", err)
	}
	var proveedores [8]*proveedorMaterialAltaContratacionTemporalDesarrollo
	for i := range proveedores {
		proveedores[i] = &proveedorMaterialAltaContratacionTemporalDesarrollo{}
	}
	cfg.CronosResolucionEnabled = "true"
	if _, err := nuevasRutasCronosEmpleadoDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, materialCronosDesdeProveedores(proveedores)); !errors.Is(err, ErrComposicionCronosEmpleadoNoDisponible) {
		t.Fatal("arranca la resolución sin su material V3", err)
	}
	if !cronosResolucionSolicitada("true", "true") || cronosResolucionSolicitada("false", "true") || cronosResolucionSolicitada("true", "") {
		t.Fatal("selector combinado distinto")
	}
}

// TLS y resolvedor de certificado son reales; sesión, contexto y auditoría
// son dobles. Acredita la frontera, no PostgreSQL ni el recorrido publicado.
func TestCronosEmpleadoFronteraMTLSDeniegaConMotivoYAudita(t *testing.T) {
	cfg, rutasMaterial := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	identidad := composicion.identidad.(*resolvedorIdentidadDesarrollo)
	clienteCert, err := tls.LoadX509KeyPair(rutasMaterial.ClientCertificate, rutasMaterial.ClientPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(clienteCert.Certificate[0])
	huella := hex.EncodeToString(digest[:])
	principal := identidad.porHuella[digest]
	fixture := nuevoEscenarioMaterialRutasDietasPrueba(t, dp.AccionConsultarCatalogoRutasDietas, time.Now().UTC().Truncate(time.Microsecond))
	reloj := &relojSesionConsultaPrueba{ahora: fixture.ahora}
	cuenta := cuentaRutasDietasDesarrollo{CertificadoSHA256: huella, Sujeto: principal.ID, CuentaRef: fixture.resultado.Contexto.Instantanea.CuentaRef, PerfilRef: fixture.resultado.Contexto.PerfilActivoRef}
	registro := &registroSesionConsultaPrueba{reloj: reloj, cuenta: cuenta.CuentaRef}
	contextos := &resolutorContextoCronosPrueba{base: &resolutorSesionConsultaPrueba{base: fixture.resultado, reloj: reloj}}
	cuentas := map[string]cuentaRutasDietasDesarrollo{huella: cuenta}
	base := &autoridadRutasDietasDesarrollo{resolvedor: identidad, cuentas: cuentas, registro: registro, revalidador: &revalidadorSesionConsultaPrueba{registro: registro}, contextos: contextos, reloj: reloj, instancia: strings.Repeat("a", 64)}
	auditoria := &registroDenegacionCronosPrueba{}
	autoridad := &autoridadCronosEmpleadoDesarrollo{base: base, reloj: reloj, cuentas: cuentas, registrador: auditoria}
	datos := 0
	autoridad.rutas = map[string]http.Handler{cronoshttp.RutaConsultarSaldoPropio: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := seguridadCronosEmpleadoDesarrollo{autoridad: autoridad}.ResolverIdentidadRegistradaCronos(r.Context())
		if err != nil || id.Contexto.Contexto.PersonaRef != fixture.resultado.Contexto.PersonaRef {
			t.Error("la identidad registrada no llega al manejador", err)
		}
		datos++
		w.WriteHeader(http.StatusOK)
	})}
	raizLlamadas := 0
	raiz := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { raizLlamadas++; w.WriteHeader(http.StatusTeapot) })
	servidor := httptest.NewUnstartedServer(componerRaizConCronosEmpleado(raiz, autoridad))
	servidor.TLS = composicion.tls.Clone()
	servidor.StartTLS()
	t.Cleanup(servidor.Close)
	ca, err := os.ReadFile(rutasMaterial.CACertificate)
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	raices.AppendCertsFromPEM(ca)
	transport := &http.Transport{TLSClientConfig: &tls.Config{Certificates: []tls.Certificate{clienteCert}, RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13}}
	t.Cleanup(transport.CloseIdleConnections)
	cliente := &http.Client{Transport: transport}
	pedir := func(ruta string, estado int, codigo string) {
		t.Helper()
		res, err := cliente.Get(servidor.URL + ruta)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		cuerpo, _ := io.ReadAll(res.Body)
		if res.StatusCode != estado || (codigo != "" && !strings.Contains(string(cuerpo), `"`+codigo+`"`)) || res.Header.Get("Set-Cookie") != "" {
			t.Fatalf("%s: %d %s (esperado %d %s)", ruta, res.StatusCode, cuerpo, estado, codigo)
		}
	}
	pedir(cronoshttp.RutaConsultarSaldoPropio+"?periodo=hoy", http.StatusOK, "")
	if datos != 1 || len(auditoria.ordenes) != 0 || len(registro.altas) != 1 {
		t.Fatal("petición autenticada sin sesión nominal o con auditoría espuria")
	}
	pedir(prefijoRutasCronosEmpleado+"marcajes/propio", http.StatusNotFound, "no_disponible")
	if o := auditoria.ordenes[0]; o.Ruta != "otra" || o.Motivo != cronosports.MotivoFronteraAccesoDenegado {
		t.Fatalf("ruta no publicada sin auditar: %+v", o)
	}
	contextos.motivo = vp.ErrProyeccionEmpleadoContextoActorAusente
	pedir(cronoshttp.RutaConsultarSaldoPropio+"?periodo=hoy", http.StatusForbidden, "sin_empleado")
	contextos.motivo = vp.ErrProyeccionEmpleadoContextoActorAmbigua
	pedir(cronoshttp.RutaConsultarSaldoPropio+"?periodo=hoy", http.StatusForbidden, "empleado_ambiguo")
	contextos.motivo = errors.New("contexto caído")
	pedir(cronoshttp.RutaConsultarSaldoPropio+"?periodo=hoy", http.StatusServiceUnavailable, "no_disponible")
	if len(auditoria.ordenes) != 4 || auditoria.ordenes[1].Motivo != cronosports.MotivoFronteraSinEmpleado ||
		auditoria.ordenes[2].Motivo != cronosports.MotivoFronteraEmpleadoAmbiguo || auditoria.ordenes[3].Motivo != cronosports.MotivoFronteraDependencia ||
		auditoria.ordenes[1].Ruta != cronoshttp.RutaConsultarSaldoPropio {
		t.Fatalf("denegaciones sin motivo auditado: %+v", auditoria.ordenes)
	}
	contextos.motivo = nil
	delete(cuentas, huella)
	pedir(cronoshttp.RutaConsultarSaldoPropio+"?periodo=hoy", http.StatusForbidden, "acceso_denegado")
	cuentas[huella] = cuenta
	auditoria.err = errors.New("auditor caído")
	contextos.motivo = vp.ErrProyeccionEmpleadoContextoActorAusente
	pedir(cronoshttp.RutaConsultarSaldoPropio+"?periodo=hoy", http.StatusServiceUnavailable, "no_disponible")
	if datos != 1 {
		t.Fatal("una denegación alcanzó datos")
	}
	pedir("/api/vec/otra", http.StatusTeapot, "")
	if raizLlamadas != 1 {
		t.Fatal("la raíz existente dejó de atender fuera del prefijo")
	}
	sinTLS := httptest.NewRecorder()
	auditoria.err = nil
	previas := len(auditoria.ordenes)
	componerRaizConCronosEmpleado(raiz, autoridad).ServeHTTP(sinTLS, httptest.NewRequest(http.MethodGet, cronoshttp.RutaConsultarSaldoPropio, nil))
	if sinTLS.Code != http.StatusUnauthorized || !strings.Contains(sinTLS.Body.String(), "autenticacion_requerida") {
		t.Fatal("sin mTLS no denegado", sinTLS.Code)
	}
	// Un anónimo sin mTLS no escribe en la auditoría durable, ni en rutas
	// publicadas ni en rutas inexistentes bajo el prefijo.
	for i := 0; i < 3; i++ {
		anonimo := httptest.NewRecorder()
		componerRaizConCronosEmpleado(raiz, autoridad).ServeHTTP(anonimo, httptest.NewRequest(http.MethodPost, prefijoRutasCronosEmpleado+"inexistente", nil))
		if anonimo.Code != http.StatusNotFound || !strings.Contains(anonimo.Body.String(), "no_disponible") {
			t.Fatal("ruta no publicada anónima no denegada", anonimo.Code)
		}
	}
	if len(auditoria.ordenes) != previas {
		t.Fatalf("peticiones anónimas escribieron auditoría: %d > %d", len(auditoria.ordenes), previas)
	}
	// Con identidad TLS acreditada la ruta no publicada sí se audita, y si la
	// auditoría no confirma la respuesta es 503.
	auditoria.err = errors.New("auditor caído")
	pedir(prefijoRutasCronosEmpleado+"inexistente", http.StatusServiceUnavailable, "no_disponible")
	if len(auditoria.ordenes) != previas+1 || auditoria.ordenes[previas].Ruta != "otra" {
		t.Fatalf("denegación acreditada sin intento de auditoría: %+v", auditoria.ordenes)
	}
	auditoria.err = nil
}
