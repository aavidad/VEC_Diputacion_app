package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	appserver "vec-diputacion-granada/internal/app/server"
	"vec-diputacion-granada/internal/candidate/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const tokenFakePruebas = "M6wW0bkGMl7xN-kU9sQ2zvCe5tHY8aJ4RfP1dXo3LnA7qB2iK9uV6cE8"

func configurarFuentesProduccionPrueba(t *testing.T, cfg config.Config) config.Config {
	t.Helper()
	destino := t.TempDir()
	copiar := func(origen, nombre string) string {
		t.Helper()
		contenido, err := os.ReadFile(filepath.Join("..", "..", "..", origen))
		if err != nil {
			t.Fatalf("leer fuente de prueba %s: %v", origen, err)
		}
		ruta := filepath.Join(destino, nombre)
		if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
			t.Fatalf("copiar fuente de prueba %s: %v", origen, err)
		}
		return ruta
	}
	if strings.TrimSpace(cfg.BolsaPublicSourcePath) == "" || cfg.BolsaPublicSourcePath == config.DefaultBolsaPublicSourcePath {
		cfg.BolsaPublicSourcePath = copiar(config.DefaultBolsaPublicSourcePath, "convocatorias_publicas.json")
	}
	if strings.TrimSpace(cfg.BolsaCategoriesSourcePath) == "" || cfg.BolsaCategoriesSourcePath == config.DefaultBolsaCategoriesSourcePath {
		cfg.BolsaCategoriesSourcePath = copiar(config.DefaultBolsaCategoriesSourcePath, "categorias_profesionales.json")
	}
	return cfg
}

func configuracionAPIPrueba(cfg config.Config) config.Config {
	// Las raices exportadas fallan cerradas en produccion. Las pruebas de los
	// adaptadores transitorios declaran expresamente un perfil no productivo.
	cfg.ExecutionProfile = config.ExecutionProfileRRHHPresentation
	return cfg
}

func configurarFakePrueba(t *testing.T, cfg config.Config) config.Config {
	t.Helper()
	huella := sha256.Sum256([]byte(tokenFakePruebas))
	contenido, err := json.Marshal(archivoCredencialesFake{
		Version: versionCredencialesFake,
		Credentials: []registroCredencialFake{{
			TokenSHA256: hex.EncodeToString(huella[:]),
			Subject:     "principal-configurado-prueba",
			DisplayName: "Principal configurado de prueba",
			Roles:       []string{"administrador"},
			Mechanism:   ports.AuthMechanismKerberosAD,
			Assurance:   vecdomain.AuthAssuranceHigh,
			LegacyRole:  ports.AuthRoleSystemAdmin,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "credenciales.json")
	if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg.AuthMode = config.AuthModeFake
	cfg.FakeCredentialsPath = ruta
	if strings.TrimSpace(cfg.StorageMode) == "" || cfg.StorageMode == config.StorageModeMemory {
		cfg.StorageMode = config.StorageModeLocalDurable
		cfg.DataDir = t.TempDir()
		cfg.DataPath = ""
	}
	return configurarFuentesProduccionPrueba(t, cfg)
}

func nuevoServidorFakeAisladoPrueba(t *testing.T, cfg config.Config) *http.Server {
	t.Helper()
	cfg = configuracionAPIPrueba(configurarFakePrueba(t, cfg))
	api, err := NewDemoAPIWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	servidor, err := appserver.NewHTTPServer(cfg, api)
	if err != nil {
		t.Fatal(err)
	}
	return servidor
}

func registroFakePrueba(token string) registroCredencialFake {
	huella := sha256.Sum256([]byte(token))
	return registroCredencialFake{
		TokenSHA256: hex.EncodeToString(huella[:]),
		Subject:     "principal-configurado-prueba",
		DisplayName: "Principal configurado de prueba",
		Roles:       []string{"administrador"},
		Mechanism:   ports.AuthMechanismKerberosAD,
		Assurance:   vecdomain.AuthAssuranceHigh,
		LegacyRole:  ports.AuthRoleSystemAdmin,
	}
}

func escribirCredencialesFakePrueba(t *testing.T, modo os.FileMode, registros ...registroCredencialFake) string {
	t.Helper()
	contenido, err := json.Marshal(archivoCredencialesFake{Version: versionCredencialesFake, Credentials: registros})
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "credenciales.json")
	if err := os.WriteFile(ruta, contenido, modo); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ruta, modo); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func TestComposicionCatalogoPersonalNoCreaSnapshotAlArrancar(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "personal", "catalogo.json")
	servicio, err := nuevoServicioCatalogoPersonal(ruta)
	if err != nil {
		t.Fatalf("componer catalogo Personal durable: %v", err)
	}
	estadisticas, err := servicio.Stats(context.Background())
	if err != nil {
		t.Fatalf("consultar catalogo Personal vacio: %v", err)
	}
	if estadisticas.Positions != 0 || estadisticas.Categories != 0 || estadisticas.CatalogEntries != 0 {
		t.Fatalf("el arranque inyecto datos implicitos: %+v", estadisticas)
	}
	for _, candidata := range []string{ruta, ruta + ".bak"} {
		if _, err := os.Stat(candidata); !os.IsNotExist(err) {
			t.Fatalf("el arranque creo %s: %v", candidata, err)
		}
	}
}

func TestComposicionCatalogoPersonalEnMemoriaEsExplicita(t *testing.T) {
	servicio, err := nuevoServicioCatalogoPersonal("")
	if err != nil {
		t.Fatalf("componer catalogo Personal en memoria: %v", err)
	}
	if servicio == nil {
		t.Fatal("la raiz de composicion no inyecto el servicio de Personal")
	}
}

func TestNewHTTPServerWithConfigComponeAPISinDarFuncionBolsaAlAdministradorTecnico(t *testing.T) {
	srv := nuevoServidorFakeAisladoPrueba(t, config.Config{
		Address:             "127.0.0.1:0",
		APIBasePath:         "/api",
		ReadHeaderTimeout:   time.Second,
		PersonalCatalogPath: "memory",
	})
	if srv.Addr != "127.0.0.1:0" || srv.ReadHeaderTimeout != time.Second {
		t.Fatalf("server config = %s/%s", srv.Addr, srv.ReadHeaderTimeout)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/demo", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Authorization", "Bearer "+tokenFakePruebas)
	srv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("/api/demo status = %d, want 403: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "Permisos insuficientes") || strings.Contains(body, "demo-puestos-rpt") {
		t.Fatalf("el administrador tecnico recibio datos funcionales de Bolsa: %s", body)
	}
}

func TestProduccionIntegradaRechazaAutenticacionDeshabilitada(t *testing.T) {
	srv, err := NewHTTPServerWithConfig(configurarFuentesProduccionPrueba(t, config.Config{
		Address:             "127.0.0.1:0",
		PersonalCatalogPath: "memory",
		StorageMode:         config.StorageModeLocalDurable,
		DataDir:             t.TempDir(),
	}))
	if srv != nil || !errors.Is(err, ErrComposicionProductivaNoDisponible) {
		t.Fatalf("produccion integrada disabled = (%v, %v)", srv, err)
	}
}

func TestModoDisabledPublicaSoloConsultaAnonimaBolsa(t *testing.T) {
	api, err := NewDemoAPIWithConfig(configuracionAPIPrueba(config.Config{PersonalCatalogPath: "memory"}))
	if err != nil {
		t.Fatalf("NewDemoAPIWithConfig() error = %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/publico/bolsa/convocatorias", nil)
	api.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "vec.bolsa.publico.convocatorias.v1") {
		t.Fatalf("consulta pública = %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	api.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/publico/bolsa/categorias", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"total":68`) {
		t.Fatalf("directorio de categorias = %d %s", rec.Code, rec.Body.String())
	}
	for _, ruta := range []string{"/api/publico/bolsa/personas", "/api/publico/bolsa/solicitudes", "/api/demo"} {
		rec = httptest.NewRecorder()
		api.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ruta, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("ruta %s publicada: %d", ruta, rec.Code)
		}
	}
}

func TestSmokeLoopbackPortalYAPIPublicaSinCabecerasConfiables(t *testing.T) {
	servidor := nuevoServidorFakeAisladoPrueba(t, config.Config{
		Address: "127.0.0.1:0", PersonalCatalogPath: "memory",
	})
	escucha, err := net.Listen("tcp", servidor.Addr)
	if err != nil {
		t.Fatal(err)
	}
	finalizado := make(chan error, 1)
	go func() { finalizado <- servidor.Serve(escucha) }()
	base := "http://" + escucha.Addr().String()
	for _, ruta := range []string{"/bolsa/", "/api/publico/bolsa/convocatorias", "/api/publico/bolsa/categorias"} {
		respuesta, err := http.Get(base + ruta)
		if err != nil {
			t.Fatalf("GET %s: %v", ruta, err)
		}
		cuerpo, err := io.ReadAll(respuesta.Body)
		respuesta.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if respuesta.StatusCode != http.StatusOK || len(cuerpo) == 0 {
			t.Fatalf("GET %s = %d, %q", ruta, respuesta.StatusCode, cuerpo)
		}
	}
	if err := servidor.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := <-finalizado; err != nil && err != http.ErrServerClosed {
		t.Fatal(err)
	}
}

func TestArranqueRechazaVersionCatalogoCategoriasInvalida(t *testing.T) {
	_, err := NewDemoAPIWithConfig(configuracionAPIPrueba(config.Config{
		PersonalCatalogPath:       "memory",
		BolsaCategoriesVersion:    -1,
		BolsaCategoriesSourcePath: config.DefaultBolsaCategoriesSourcePath,
		BolsaCategoriesCatalogID:  config.DefaultBolsaCategoriesCatalogID,
	}))
	if err == nil || !strings.Contains(err.Error(), "version de catalogo") {
		t.Fatalf("version invalida no rechazo el arranque: %v", err)
	}
}

func TestArranqueRechazaCatalogoCategoriasInexistente(t *testing.T) {
	for _, prueba := range []struct {
		nombre  string
		id      string
		version int
	}{
		{nombre: "id", id: "catalogo-profesional-inexistente", version: 1},
		{nombre: "version", id: config.DefaultBolsaCategoriesCatalogID, version: 2},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			_, err := NewDemoAPIWithConfig(configuracionAPIPrueba(config.Config{
				PersonalCatalogPath:       "memory",
				BolsaCategoriesSourcePath: config.DefaultBolsaCategoriesSourcePath,
				BolsaCategoriesCatalogID:  prueba.id,
				BolsaCategoriesVersion:    prueba.version,
			}))
			if err == nil || !strings.Contains(err.Error(), "catalogo profesional gobernado incompatible") {
				t.Fatalf("seleccion inexistente no rechazo el arranque: %v", err)
			}
		})
	}
}

func TestArranqueRechazaHuellaCatalogoCategoriasDistinta(t *testing.T) {
	_, err := NewDemoAPIWithConfig(configuracionAPIPrueba(config.Config{
		PersonalCatalogPath:       "memory",
		BolsaCategoriesSourcePath: config.DefaultBolsaCategoriesSourcePath,
		BolsaCategoriesCatalogID:  config.DefaultBolsaCategoriesCatalogID,
		BolsaCategoriesVersion:    config.DefaultBolsaCategoriesVersion,
		BolsaCategoriesSHA256:     "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}))
	if err == nil || !strings.Contains(err.Error(), "catalogo profesional gobernado incompatible") {
		t.Fatalf("huella distinta no rechazo el arranque: %v", err)
	}
}

func TestArranqueRechazaCategoriaDeConvocatoriaDesconocida(t *testing.T) {
	base, err := os.ReadFile("../../../data/demo/convocatorias_publicas.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	alterado := strings.Replace(string(base), `"categorias": ["operario"]`, `"categorias": ["categoria-inexistente"]`, 1)
	if alterado == string(base) {
		t.Fatal("no se altero la categoria de prueba")
	}
	ruta := filepath.Join(t.TempDir(), "convocatorias.json")
	if err := os.WriteFile(ruta, []byte(alterado), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = NewDemoAPIWithConfig(configuracionAPIPrueba(config.Config{
		PersonalCatalogPath:       "memory",
		BolsaPublicSourcePath:     ruta,
		BolsaCategoriesSourcePath: config.DefaultBolsaCategoriesSourcePath,
		BolsaCategoriesCatalogID:  config.DefaultBolsaCategoriesCatalogID,
		BolsaCategoriesVersion:    1,
	}))
	if err == nil || !strings.Contains(err.Error(), "fuentes publicas de Bolsa incompatibles") {
		t.Fatalf("categoria desconocida no rechazo el arranque: %v", err)
	}
}

func TestComposicionIntegradaRechazaCabecerasConfiablesHeredadas(t *testing.T) {
	cfg := config.Config{
		Address:             "127.0.0.1:0",
		PersonalCatalogPath: "memory",
		AuthMode:            config.AuthModeTrustedHeaders,
		TrustedProxyCIDRs:   []string{"127.0.0.1/32"},
	}
	cfg = configurarFuentesProduccionPrueba(t, cfg)

	if servidor, err := NewHTTPServerWithConfig(cfg); !errors.Is(err, ErrModoCabecerasConfiablesRetirado) || servidor != nil {
		t.Fatalf("NewHTTPServerWithConfig() = (%v, %v); debe rechazar trusted_headers", servidor, err)
	}
	if api, err := NewDemoAPIWithConfig(cfg); !errors.Is(err, ErrModoCabecerasConfiablesRetirado) || api != nil {
		t.Fatalf("NewDemoAPIWithConfig() = (%v, %v); debe rechazar trusted_headers", api, err)
	}
	if api, err := NewVECShellAPIWithConfig(cfg); !errors.Is(err, ErrModoCabecerasConfiablesRetirado) || api != nil {
		t.Fatalf("NewVECShellAPIWithConfig() = (%v, %v); debe rechazar trusted_headers", api, err)
	}

	servidorPublico, err := NewHTTPServerPublicoWithConfig(cfg)
	if !errors.Is(err, ErrComposicionProductivaNoDisponible) || servidorPublico != nil {
		t.Fatalf("NewHTTPServerPublicoWithConfig() = (%v, %v); produccion aun no esta disponible", servidorPublico, err)
	}
}

func TestModosNoFakeNoConstruyenNiMontanBolsaHeredada(t *testing.T) {
	for _, modo := range []string{config.AuthModeDisabled} {
		t.Run(modo, func(t *testing.T) {
			// Un directorio no es un fichero durable valido. Si la composicion
			// intentase construir el almacenamiento de Bolsa, el arranque fallaria.
			api, err := NewDemoAPIWithConfig(configuracionAPIPrueba(config.Config{
				AuthMode:            modo,
				StorageMode:         config.StorageModeFile,
				DataPath:            t.TempDir(),
				PersonalCatalogPath: "memory",
				TrustedProxyCIDRs:   []string{"127.0.0.1/32"},
			}))
			if err != nil {
				t.Fatalf("el modo %s intento construir Bolsa heredada: %v", modo, err)
			}

			for _, ruta := range []string{
				"/api/demo",
				"/api/candidates",
				"/api/audit",
				"/api/admin/status",
			} {
				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodGet, ruta, nil)
				req.RemoteAddr = "127.0.0.1:12345"
				api.ServeHTTP(rec, req)
				if rec.Code != http.StatusNotFound {
					t.Fatalf("%s publico %s: estado=%d cuerpo=%s", modo, ruta, rec.Code, rec.Body.String())
				}
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			api.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s altero el cierre de /api/vec: estado=%d cuerpo=%s", modo, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestNewHTTPServerExposesUnifiedVECShellModules(t *testing.T) {
	srv := nuevoServidorFakeAisladoPrueba(t, config.Config{
		Address:             "127.0.0.1:0",
		APIBasePath:         "/api",
		ReadHeaderTimeout:   time.Second,
		PersonalCatalogPath: "memory",
	})

	for _, tc := range []struct {
		method    string
		path      string
		want      string
		forbidden []string
		status    int
	}{
		{method: http.MethodGet, path: "/api/vec/modules", want: "vec.module.personal", status: http.StatusOK},
		{method: http.MethodGet, path: "/api/vec/modules", want: "vec.module.cronos", status: http.StatusOK},
		{method: http.MethodGet, path: "/api/vec/modules", want: "vec.module.dietas", status: http.StatusOK},
		{method: http.MethodGet, path: "/api/vec/modules", want: "vec.module.bolsa", status: http.StatusOK},
		{method: http.MethodGet, path: "/api/vec/modules", want: "vec.module.administracion", status: http.StatusOK},
		{method: http.MethodGet, path: "/api/vec/workspace", want: "vec permission denied", status: http.StatusForbidden},
		{method: http.MethodGet, path: "/api/vec/menu", want: "admin.catalogos", forbidden: []string{"personal.", "cronos.", "dietas.", "bolsa."}, status: http.StatusOK},
		{method: http.MethodPost, path: "/api/vec/modules/cronos/action", want: "vec permission denied", status: http.StatusForbidden},
		{method: http.MethodPost, path: "/api/vec/modules/horarios/action", want: "vec permission denied", status: http.StatusForbidden},
		{method: http.MethodPost, path: "/api/vec/modules/permisos/action", want: "vec permission denied", status: http.StatusForbidden},
		{method: http.MethodPost, path: "/api/vec/modules/dietas/action", want: "vec permission denied", status: http.StatusForbidden},
		{method: http.MethodPost, path: "/api/vec/modules/rutas/action", want: "vec permission denied", status: http.StatusForbidden},
		{method: http.MethodPost, path: "/api/vec/modules/administracion/action", want: "vec.catalog.publish", status: http.StatusAccepted},
		{method: http.MethodPost, path: "/api/vec/modules/personal/action", want: "vec permission denied", status: http.StatusForbidden},
		{method: http.MethodPost, path: "/api/vec/modules/nominas/action", want: "vec permission denied", status: http.StatusForbidden},
		{method: http.MethodPost, path: "/api/vec/modules/bolsa/action", want: "vec permission denied", status: http.StatusForbidden},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set("Authorization", "Bearer "+tokenFakePruebas)
		srv.Handler.ServeHTTP(rec, req)
		if rec.Code != tc.status {
			t.Fatalf("%s %s status = %d, want %d: %s", tc.method, tc.path, rec.Code, tc.status, rec.Body.String())
		}
		if body := rec.Body.String(); !strings.Contains(body, tc.want) {
			t.Fatalf("%s %s body missing %q: %s", tc.method, tc.path, tc.want, body)
		} else {
			for _, forbidden := range tc.forbidden {
				if strings.Contains(body, forbidden) {
					t.Fatalf("%s %s body contiene capacidad funcional %q: %s", tc.method, tc.path, forbidden, body)
				}
			}
		}
	}
}

func TestFakeFallaCerradoSinFicheroDeCredenciales(t *testing.T) {
	if api, err := NewDemoAPI(); !errors.Is(err, ErrComposicionProductivaNoDisponible) || api != nil {
		t.Fatalf("NewDemoAPI() = (%v, %v); debe cerrar produccion", api, err)
	}
	if api, err := NewDemoAPIWithConfig(configuracionAPIPrueba(config.Config{
		AuthMode: config.AuthModeFake,
		Address:  "127.0.0.1:8080",
	})); err == nil || api != nil {
		t.Fatalf("fake sin fichero = (%v, %v); debe fallar cerrado", api, err)
	}
}

func TestFakeSoloAceptaBearerConfiguradoYNoCabecerasDeIdentidad(t *testing.T) {
	srv := nuevoServidorFakeAisladoPrueba(t, config.Config{
		Address:             "127.0.0.1:0",
		PersonalCatalogPath: "memory",
	})
	probar := func(nombre string, preparar func(*http.Request), esperado int) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		if preparar != nil {
			preparar(req)
		}
		srv.Handler.ServeHTTP(rec, req)
		if rec.Code != esperado {
			t.Fatalf("%s: estado=%d, esperado=%d, cuerpo=%s", nombre, rec.Code, esperado, rec.Body.String())
		}
		return rec
	}

	probar("sin token", nil, http.StatusUnauthorized)
	probar("token no registrado", func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ")
	}, http.StatusUnauthorized)
	probar("solo cabeceras falsas", func(req *http.Request) {
		req.Header.Set("X-VEC-Subject", "atacante")
		req.Header.Set("X-VEC-Roles", "administrador")
		req.Header.Set("X-VEC-Auth-Mechanism", "dnie")
		req.Header.Set("X-Auth-Subject", "atacante")
		req.Header.Set("X-Auth-Roles", "system_admin")
		req.Header.Set("X-Auth-Mechanism", "dnie")
		req.Header.Set("X-Auth-Token", tokenFakePruebas)
	}, http.StatusBadRequest)

	probar("bearer valido con cabeceras falsas", func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+tokenFakePruebas)
		req.Header.Set("X-VEC-Subject", "atacante")
		req.Header.Set("X-VEC-Roles", "ciudadano")
		req.Header.Set("X-VEC-Auth-Mechanism", "clave")
		req.Header.Set("X-Auth-Assurance", "bajo")
	}, http.StatusBadRequest)

	valido := probar("bearer valido", func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+tokenFakePruebas)
	}, http.StatusOK)
	cuerpo := valido.Body.String()
	if !strings.Contains(cuerpo, `"id":"principal-configurado-prueba"`) ||
		!strings.Contains(cuerpo, `"roles":["administrador"]`) ||
		!strings.Contains(cuerpo, `"auth_method":"kerberos_ad"`) ||
		!strings.Contains(cuerpo, `"auth_assurance":"alto"`) || strings.Contains(cuerpo, "atacante") {
		t.Fatalf("la identidad no procede exclusivamente del fichero: %s", cuerpo)
	}
}

func TestFakeRechazaFicheroPermisivoYRegistrosDuplicados(t *testing.T) {
	registro := registroFakePrueba(tokenFakePruebas)
	rutaPermisiva := escribirCredencialesFakePrueba(t, 0o640, registro)
	if almacen, err := cargarCredencialesFake(rutaPermisiva); err == nil || almacen != nil {
		t.Fatalf("fichero 0640 aceptado: almacen=%v err=%v", almacen, err)
	}

	rutaDuplicada := escribirCredencialesFakePrueba(t, 0o600, registro, registro)
	if almacen, err := cargarCredencialesFake(rutaDuplicada); err == nil || almacen != nil {
		t.Fatalf("token duplicado aceptado: almacen=%v err=%v", almacen, err)
	}

	rutaClaveDuplicada := filepath.Join(t.TempDir(), "credenciales-clave-duplicada.json")
	contenidoClaveDuplicada := `{"version":1,"version":1,"credentials":[]}`
	if err := os.WriteFile(rutaClaveDuplicada, []byte(contenidoClaveDuplicada), 0o600); err != nil {
		t.Fatal(err)
	}
	if almacen, err := cargarCredencialesFake(rutaClaveDuplicada); err == nil || almacen != nil {
		t.Fatalf("clave JSON duplicada aceptada: almacen=%v err=%v", almacen, err)
	}
}

func TestFakeSoloAdmiteTablaPositivaUnivocaDeRoles(t *testing.T) {
	compatibles := []struct {
		rolVEC    string
		rolLegacy ports.AuthRole
	}{
		{rolVEC: "ciudadano", rolLegacy: ports.AuthRoleCandidate},
		{rolVEC: "administrativo", rolLegacy: ports.AuthRoleValidatorL1},
		{rolVEC: "tecnico_rrhh", rolLegacy: ports.AuthRoleValidatorL2},
		{rolVEC: "jefatura_rrhh", rolLegacy: ports.AuthRoleValidatorL2},
		{rolVEC: "administrador", rolLegacy: ports.AuthRoleSystemAdmin},
	}
	for _, caso := range compatibles {
		t.Run("admite_"+caso.rolVEC, func(t *testing.T) {
			registro := registroFakePrueba(tokenFakePruebas)
			registro.Roles = []string{caso.rolVEC}
			registro.LegacyRole = caso.rolLegacy
			ruta := escribirCredencialesFakePrueba(t, 0o600, registro)
			if almacen, err := cargarCredencialesFake(ruta); err != nil || almacen == nil {
				t.Fatalf("combinacion compatible rechazada: almacen=%v err=%v", almacen, err)
			}
		})
	}

	incompatibles := []struct {
		nombre    string
		rolesVEC  []string
		rolLegacy ports.AuthRole
	}{
		{nombre: "ciudadano privilegiado", rolesVEC: []string{"ciudadano"}, rolLegacy: ports.AuthRoleSystemAdmin},
		{nombre: "rol VEC desconocido", rolesVEC: []string{"superadministrador"}, rolLegacy: ports.AuthRoleSystemAdmin},
		{nombre: "alias ingles", rolesVEC: []string{"system_admin"}, rolLegacy: ports.AuthRoleSystemAdmin},
		{nombre: "empleado ordinario como tramitador", rolesVEC: []string{"personal_interno"}, rolLegacy: ports.AuthRoleValidatorL1},
		{nombre: "jefatura de servicio como tramitador", rolesVEC: []string{"jefe_servicio"}, rolLegacy: ports.AuthRoleValidatorL1},
		{nombre: "roles VEC multiples", rolesVEC: []string{"administrador", "tecnico_rrhh"}, rolLegacy: ports.AuthRoleSystemAdmin},
		{nombre: "rol VEC repetido", rolesVEC: []string{"administrador", "administrador"}, rolLegacy: ports.AuthRoleSystemAdmin},
	}
	for _, caso := range incompatibles {
		t.Run("rechaza_"+caso.nombre, func(t *testing.T) {
			registro := registroFakePrueba(tokenFakePruebas)
			registro.Roles = append([]string(nil), caso.rolesVEC...)
			registro.LegacyRole = caso.rolLegacy
			ruta := escribirCredencialesFakePrueba(t, 0o600, registro)
			if almacen, err := cargarCredencialesFake(ruta); err == nil || almacen != nil {
				t.Fatalf("combinacion incompatible aceptada: almacen=%v err=%v", almacen, err)
			}
		})
	}

	registroValido := registroFakePrueba(tokenFakePruebas)
	registroInvalido := registroFakePrueba("N7xX1clHNn8yO-lV0tR3awDf6uIZ9bK5SgQ2eYp4MoB8rC3jL0vW7dF9")
	registroInvalido.Roles = []string{"ciudadano"}
	registroInvalido.LegacyRole = ports.AuthRoleSystemAdmin
	rutaMixta := escribirCredencialesFakePrueba(t, 0o600, registroValido, registroInvalido)
	if almacen, err := cargarCredencialesFake(rutaMixta); err == nil || almacen != nil {
		t.Fatalf("el fichero mixto no fallo de forma atomica: almacen=%v err=%v", almacen, err)
	}
}

func TestHandlerFakeDirectoMantieneFronteraLoopback(t *testing.T) {
	api, err := NewDemoAPIWithConfig(configuracionAPIPrueba(configurarFakePrueba(t, config.Config{
		Address:             "127.0.0.1:0",
		PersonalCatalogPath: "memory",
	})))
	if err != nil {
		t.Fatal(err)
	}

	probar := func(nombre, remoto string, cabecera, valor string, esperado int) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
		req.RemoteAddr = remoto
		req.Header.Set("Authorization", "Bearer "+tokenFakePruebas)
		if cabecera != "" {
			req.Header.Set(cabecera, valor)
		}
		api.ServeHTTP(rec, req)
		if rec.Code != esperado {
			t.Fatalf("%s: estado=%d, esperado=%d, cuerpo=%s", nombre, rec.Code, esperado, rec.Body.String())
		}
	}

	probar("IPv4 local", "127.0.0.1:12345", "", "", http.StatusOK)
	probar("IPv6 local", "[::1]:12345", "", "", http.StatusOK)
	probar("direccion externa", "203.0.113.25:12345", "", "", http.StatusUnauthorized)
	probar("loopback sin puerto", "127.0.0.1", "", "", http.StatusUnauthorized)
	probar("puerto no numerico", "127.0.0.1:http", "", "", http.StatusUnauthorized)
	probar("Forwarded", "127.0.0.1:12345", "Forwarded", "for=203.0.113.25", http.StatusUnauthorized)
	probar("Via", "127.0.0.1:12345", "Via", "1.1 proxy", http.StatusUnauthorized)
	probar("X-Forwarded-For", "127.0.0.1:12345", "X-Forwarded-For", "203.0.113.25", http.StatusUnauthorized)
	probar("X-Forwarded-Proto", "127.0.0.1:12345", "X-Forwarded-Proto", "https", http.StatusUnauthorized)
}
