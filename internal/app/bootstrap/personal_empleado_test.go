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
	dp "vec-diputacion-granada/internal/modules/dietas/ports"
	personalhttp "vec-diputacion-granada/internal/modules/personal/adapters/httpinterno"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type registroDenegacionPersonalPrueba struct {
	denegaciones []personalports.DenegacionFichaPropia
	err          error
}

func (r *registroDenegacionPersonalPrueba) RegistrarDenegacionFichaPropia(_ context.Context, d personalports.DenegacionFichaPropia) error {
	r.denegaciones = append(r.denegaciones, d)
	return r.err
}

func TestPersonalEmpleadoApagadoPorDefectoNoTocaLaRaiz(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	a, err := nuevasRutasPersonalEmpleadoDesarrollo(cfg, nil, nil, nil)
	if err != nil || a != nil {
		t.Fatal("con el selector apagado no debe componerse nada", err)
	}
	raiz := http.NotFoundHandler()
	if h := componerRaizConPersonalEmpleado(raiz, nil); h == nil {
		t.Fatal("raíz perdida")
	} else if _, ok := h.(*http.ServeMux); ok {
		t.Fatal("la raíz cambió sin la ficha propia")
	}
	if personalEmpleadoSolicitado("TRUE") || !personalEmpleadoSolicitado("true") {
		t.Fatal("selector de material no canónico")
	}
	d := descriptorMaterialFichaPropiaPersonalDesarrollo()
	if d.Audiencia != personaldomain.AudienciaFichaPropia || !strings.HasPrefix(d.Prefijo, "clave:capacidad:") {
		t.Fatalf("descriptor V3 inesperado: %+v", d)
	}
	// El descriptor convive con los del resto de consumidores (CT, Bolsa,
	// Dietas, Cronos y los ocho de B2) sin colisión de audiencia, dominio ni
	// prefijo, tampoco textual: ningún prefijo puede ser prefijo de otro.
	todos := append(append(descriptoresPreviosPersonalB2Prueba(), descriptoresMaterialPersonalB2Desarrollo()...), d)
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(todos); err != nil {
		t.Fatal("el descriptor de la ficha propia colisiona con otro consumidor", err)
	}
	for _, otro := range todos {
		if otro.Audiencia != d.Audiencia && (strings.HasPrefix(otro.Prefijo, d.Prefijo) || strings.HasPrefix(d.Prefijo, otro.Prefijo) || otro.Dominio == d.Dominio) {
			t.Fatalf("prefijo o dominio solapado con la ficha propia: %s / %s", d.Prefijo, otro.Prefijo)
		}
	}
	for _, b2 := range descriptoresMaterialPersonalB2Desarrollo() {
		for nombre, alterado := range map[string]descriptorMaterialConsumidorV3Desarrollo{
			"audiencia": {Audiencia: b2.Audiencia, Dominio: d.Dominio, Prefijo: d.Prefijo, ProveedorNominal: d.ProveedorNominal},
			"dominio":   {Audiencia: d.Audiencia, Dominio: b2.Dominio, Prefijo: d.Prefijo, ProveedorNominal: d.ProveedorNominal},
			"prefijo":   {Audiencia: d.Audiencia, Dominio: d.Dominio, Prefijo: b2.Prefijo, ProveedorNominal: d.ProveedorNominal},
		} {
			con := append(append(descriptoresPreviosPersonalB2Prueba(), descriptoresMaterialPersonalB2Desarrollo()...), alterado)
			if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(con); err == nil {
				t.Fatalf("colisión de %s entre la ficha propia y %s admitida", nombre, b2.Audiencia)
			}
		}
	}
}

func TestPersonalEmpleadoActivadoFallaCerradoSinDependencias(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	cfg.PersonalEmpleadoEnabled = "si"
	if _, err := nuevasRutasPersonalEmpleadoDesarrollo(cfg, nil, nil, nil); !errors.Is(err, config.ErrConfiguracionPersonalEmpleadoSelector) {
		t.Fatal("selector no canónico aceptado", err)
	}
	cfg.PersonalEmpleadoEnabled = "true"
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nuevasRutasPersonalEmpleadoDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, nil); !errors.Is(err, ErrComposicionPersonalEmpleadoNoDisponible) {
		t.Fatal("arranca sin material V3 de Personal", err)
	}
	material := &proveedorMaterialAltaContratacionTemporalDesarrollo{}
	if _, err := nuevasRutasPersonalEmpleadoDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, material); !errors.Is(err, ErrComposicionPersonalEmpleadoNoDisponible) {
		t.Fatal("arranca sin configuración privada de Personal", err)
	}
	cfg.DevelopmentGuard = ""
	if _, err := nuevasRutasPersonalEmpleadoDesarrollo(cfg, composicion.identidad, composicion.derivadorIdempotencia, material); !errors.Is(err, config.ErrConfiguracionPersonalEmpleadoActivacion) {
		t.Fatal("arranca fuera de la doble llave", err)
	}
}

func TestComponerManejadorFichaPropiaFallaCerradoSinIdentidad(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	if _, err := componerManejadorFichaPropia(nil, seguridadPersonalEmpleadoDesarrollo{}, nil, core.ReferenciaEntradaCatalogo{}, nil, zona); !errors.Is(err, ErrComposicionPersonalEmpleadoNoDisponible) {
		t.Fatal("compone sin dependencias", err)
	}
	// Pool perezoso: la composición no abre conexiones ni lee datos.
	pool, err := pgxpool.New(context.Background(), "postgres://personal_prueba@127.0.0.1:1/nadie?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_personal", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("5", 32)}
	identidad := seguridadPersonalEmpleadoDesarrollo{autoridad: &autoridadPersonalEmpleadoDesarrollo{reloj: relojRutasDietas{}}}
	registro := &registroDenegacionPersonalPrueba{}
	manejador, err := componerManejadorFichaPropia(pool, identidad, emisorCronosEmpleadoDesarrollo{porAccion: map[string]emisorMaterialDietasDesarrollo{}}, motivo, registro, zona)
	if err != nil || manejador == nil {
		t.Fatal("ficha propia no compuesta", err)
	}
	w := httptest.NewRecorder()
	manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet, personalhttp.RutaFichaPropia, nil))
	if w.Code != http.StatusServiceUnavailable || len(registro.denegaciones) != 1 {
		t.Fatal("manejador sin identidad registrada no falla cerrado", w.Code)
	}
}

// TLS y resolvedor de certificado son reales; sesión, contexto y auditoría
// son dobles. Acredita la frontera, no PostgreSQL ni el recorrido publicado.
func TestPersonalEmpleadoFronteraMTLSDeniegaConMotivoYAudita(t *testing.T) {
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
	auditoria := &registroDenegacionPersonalPrueba{}
	autoridad := &autoridadPersonalEmpleadoDesarrollo{base: base, reloj: reloj, cuentas: cuentas, registro: auditoria}
	datos := 0
	autoridad.rutas = map[string]http.Handler{personalhttp.RutaFichaPropia: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := seguridadPersonalEmpleadoDesarrollo{autoridad: autoridad}
		actor, err := s.ResolverActorFichaPropia(r.Context())
		id, errID := s.ResolverIdentidadFichaPropia(r.Context())
		if err != nil || errID != nil || actor.PersonaRef != fixture.resultado.Contexto.PersonaRef || id.Resultado.Contexto.PersonaRef != actor.PersonaRef {
			t.Error("la identidad registrada no llega al manejador", err, errID)
		}
		datos++
		w.WriteHeader(http.StatusOK)
	})}
	raizLlamadas := 0
	raiz := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { raizLlamadas++; w.WriteHeader(http.StatusTeapot) })
	servidor := httptest.NewUnstartedServer(componerRaizConPersonalEmpleado(raiz, autoridad))
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
	pedir(personalhttp.RutaFichaPropia, http.StatusOK, "")
	if datos != 1 || len(auditoria.denegaciones) != 0 || len(registro.altas) != 1 {
		t.Fatal("petición autenticada sin sesión nominal o con auditoría espuria")
	}
	pedir(prefijoRutasPersonalEmpleado+"empleados/emp_otro", http.StatusNotFound, "no_disponible")
	contextos.motivo = vp.ErrProyeccionEmpleadoContextoActorAusente
	pedir(personalhttp.RutaFichaPropia, http.StatusForbidden, "sin_empleado")
	contextos.motivo = vp.ErrProyeccionEmpleadoContextoActorAmbigua
	pedir(personalhttp.RutaFichaPropia, http.StatusForbidden, "empleado_ambiguo")
	contextos.motivo = errors.New("contexto caído")
	pedir(personalhttp.RutaFichaPropia, http.StatusServiceUnavailable, "no_disponible")
	if len(auditoria.denegaciones) != 4 || auditoria.denegaciones[0].Motivo != "no_encontrada" || auditoria.denegaciones[1].Motivo != "sin_empleado" ||
		auditoria.denegaciones[2].Motivo != "empleado_ambiguo" || auditoria.denegaciones[3].EstadoHTTP != http.StatusServiceUnavailable {
		t.Fatalf("denegaciones sin motivo auditado: %+v", auditoria.denegaciones)
	}
	contextos.motivo = nil
	delete(cuentas, huella)
	pedir(personalhttp.RutaFichaPropia, http.StatusForbidden, "acceso_denegado")
	cuentas[huella] = cuenta
	auditoria.err = errors.New("auditor caído")
	contextos.motivo = vp.ErrProyeccionEmpleadoContextoActorAusente
	pedir(personalhttp.RutaFichaPropia, http.StatusServiceUnavailable, "no_disponible")
	if datos != 1 {
		t.Fatal("una denegación alcanzó datos")
	}
	pedir("/api/vec/otra", http.StatusTeapot, "")
	if raizLlamadas != 1 {
		t.Fatal("la raíz existente dejó de atender fuera del prefijo")
	}
	// Un anónimo sin mTLS no escribe en la auditoría durable.
	auditoria.err = nil
	previas := len(auditoria.denegaciones)
	for _, ruta := range []string{personalhttp.RutaFichaPropia, prefijoRutasPersonalEmpleado + "inexistente"} {
		anonimo := httptest.NewRecorder()
		componerRaizConPersonalEmpleado(raiz, autoridad).ServeHTTP(anonimo, httptest.NewRequest(http.MethodGet, ruta, nil))
		if anonimo.Code != http.StatusUnauthorized && anonimo.Code != http.StatusNotFound {
			t.Fatal("anónimo no denegado", ruta, anonimo.Code)
		}
	}
	if len(auditoria.denegaciones) != previas {
		t.Fatal("peticiones anónimas escribieron auditoría")
	}
}
