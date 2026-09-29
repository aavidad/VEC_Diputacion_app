package bootstrap

import (
	"errors"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
)

// Material y conexiones sintéticos. La conexión apunta a una dirección de
// documentación (192.0.2.1, RFC 5737): si el proceso llegara a abrirla, la
// prueba vería otro error distinto del rechazo de separación.
const conexionInternaSintetica = "postgresql://vec_ct_sintetico:sintetica@192.0.2.1:1/vec?connect_timeout=1"

// vaciarConexionesDelEntorno neutraliza conexiones que el entorno de pruebas
// pudiera traer (por ejemplo, las de las pruebas con PostgreSQL real).
func vaciarConexionesDelEntorno(t *testing.T) {
	t.Helper()
	for _, linea := range os.Environ() {
		nombre, _, _ := strings.Cut(linea, "=")
		if strings.HasPrefix(nombre, "VEC_") || strings.HasPrefix(nombre, "PG") {
			t.Setenv(nombre, "")
		}
	}
}

// separarMaterial convierte el material generado en el de un proceso
// separado: retira las claves privadas de emisión y de clientes y declara el
// portal.
func separarMaterial(t *testing.T, raiz string, portal separacionportales.Portal) {
	t.Helper()
	for _, patron := range []string{"ca/ca.key", "ca/serie", "mtls/*.key", "mtls/*.p12", "mtls/*.password"} {
		coincidencias, err := filepath.Glob(filepath.Join(raiz, filepath.FromSlash(patron)))
		if err != nil {
			t.Fatal(err)
		}
		for _, ruta := range coincidencias {
			if err := os.Remove(ruta); err != nil {
				t.Fatal(err)
			}
		}
	}
	marca := `{"version":1,"portal":"` + string(portal) + `"}`
	if err := os.WriteFile(filepath.Join(raiz, separacionportales.FicheroMarcaPortal), []byte(marca), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestProcesoSeparadoFueraDeDesarrolloNoArranca(t *testing.T) {
	for _, portal := range []string{"interno", "externo"} {
		_, err := NewHTTPServerWithConfig(config.Config{PortalProceso: portal})
		if !errors.Is(err, ErrPortalSeparadoFueraDesarrollo) {
			t.Fatalf("%s fuera de desarrollo: %v", portal, err)
		}
	}
	if _, err := NewHTTPServerWithConfig(config.Config{PortalProceso: "externo "}); !errors.Is(err, separacionportales.ErrPortalDesconocido) {
		t.Fatalf("un portal mal escrito debe impedir arrancar: %v", err)
	}
}

func TestProcesoExternoNoArrancaConConexionInterna(t *testing.T) {
	vaciarConexionesDelEntorno(t)
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	separarMaterial(t, cfg.DevelopmentMaterialDir, separacionportales.PortalExterno)
	// El material generado trae la identidad de RRHH: también debe impedirlo.
	cfg.PortalProceso = "externo"
	t.Setenv("VEC_CT_DATABASE_URL", conexionInternaSintetica)
	_, err := NewHTTPServerWithConfig(cfg)
	var rechazo *separacionportales.ErrorSeparacion
	if !errors.As(err, &rechazo) || rechazo.Elemento != "VEC_CT_DATABASE_URL" {
		t.Fatalf("el externo debe rechazar la conexion interna antes de abrirla: %v", err)
	}
	if strings.Contains(err.Error(), "sintetica") {
		t.Fatalf("el error no debe mostrar la conexion: %v", err)
	}
	t.Setenv("VEC_CT_DATABASE_URL", "")
	_, err = NewHTTPServerWithConfig(cfg)
	if !errors.As(err, &rechazo) || rechazo.Elemento != "identidad/identidad.json" {
		t.Fatalf("el externo debe rechazar la identidad de RRHH de su material: %v", err)
	}
}

func TestProcesoExternoConMaterialPropioQuedaPendienteDeComposicion(t *testing.T) {
	vaciarConexionesDelEntorno(t)
	directorio, err := directorioPersonalPostgreSQL()
	if err != nil {
		t.Fatal(err)
	}
	for _, relativa := range []string{".pgpass", ".pg_service.conf", ".postgresql/postgresql.key"} {
		if _, err := os.Lstat(filepath.Join(directorio, relativa)); err == nil {
			t.Skip("el usuario de la prueba tiene credenciales de PostgreSQL en su directorio personal")
		}
	}
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	separarMaterial(t, cfg.DevelopmentMaterialDir, separacionportales.PortalExterno)
	for _, relativa := range []string{"identidad/identidad.json", "identidad/intervencion.json", "mtls/cliente.crt", "mtls/intervencion.crt"} {
		if err := os.Remove(filepath.Join(cfg.DevelopmentMaterialDir, filepath.FromSlash(relativa))); err != nil {
			t.Fatal(err)
		}
	}
	cfg.PortalProceso = "externo"
	if _, err := NewHTTPServerWithConfig(cfg); !errors.Is(err, ErrComposicionPortalExternoPendiente) {
		t.Fatalf("el externo con material propio debe pararse en la composicion pendiente: %v", err)
	}
}

func TestProcesoSeparadoRechazaDirectorioPersonalDesconocido(t *testing.T) {
	consultarOriginal := consultarUsuarioActualPostgreSQL
	t.Cleanup(func() { consultarUsuarioActualPostgreSQL = consultarOriginal })
	cfg := config.Config{
		PortalProceso:    "externo",
		ExecutionProfile: config.ExecutionProfileDevelopment,
		AuthMode:         config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement,
	}
	falloUsuario := errors.New("fallo sintetico de user.Current")
	for _, caso := range []struct {
		nombre    string
		consultar func() (*user.User, error)
		causa     error
	}{
		{"error", func() (*user.User, error) { return nil, falloUsuario }, falloUsuario},
		{"directorio vacio", func() (*user.User, error) { return &user.User{}, nil }, nil},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			consultarUsuarioActualPostgreSQL = caso.consultar
			_, err := comprobarSeparacionPortalProceso(cfg, entornoProcesoActual())
			if !errors.Is(err, ErrDirectorioPersonalPostgreSQLNoDisponible) {
				t.Fatalf("el proceso separado debe denegar el arranque: %v", err)
			}
			if caso.causa != nil && !errors.Is(err, caso.causa) {
				t.Fatalf("el fallo de user.Current debe conservarse: %v", err)
			}
		})
	}
}

func TestProcesoInternoNoArrancaConConexionOMaterialExternos(t *testing.T) {
	vaciarConexionesDelEntorno(t)
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	separarMaterial(t, cfg.DevelopmentMaterialDir, separacionportales.PortalInterno)
	cfg.PortalProceso = "interno"
	t.Setenv("VEC_EXTERNO_USUARIOS_DATABASE_URL", conexionInternaSintetica)
	_, err := NewHTTPServerWithConfig(cfg)
	var rechazo *separacionportales.ErrorSeparacion
	if !errors.As(err, &rechazo) || rechazo.Elemento != "VEC_EXTERNO_USUARIOS_DATABASE_URL" {
		t.Fatalf("el interno debe rechazar la conexion externa: %v", err)
	}
	t.Setenv("VEC_EXTERNO_USUARIOS_DATABASE_URL", "")
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "bolsa-candidato.json")
	if err := os.WriteFile(ruta, []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = NewHTTPServerWithConfig(cfg)
	if !errors.As(err, &rechazo) || rechazo.Elemento != "identidad/bolsa-candidato.json" {
		t.Fatalf("el interno debe rechazar la identidad del candidato: %v", err)
	}
}

func TestProcesoInternoRechazaClaveDeLaAutoridadCertificadora(t *testing.T) {
	vaciarConexionesDelEntorno(t)
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	clave, err := os.ReadFile(rutas.CAPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	separarMaterial(t, cfg.DevelopmentMaterialDir, separacionportales.PortalInterno)
	if err := os.WriteFile(rutas.CAPrivateKey, clave, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg.PortalProceso = "interno"
	_, err = NewHTTPServerWithConfig(cfg)
	var rechazo *separacionportales.ErrorSeparacion
	if !errors.As(err, &rechazo) || rechazo.Elemento != "ca/ca.key" {
		t.Fatalf("el interno no debe arrancar con la clave de la CA: %v", err)
	}
}

// El material de un proceso separado ya no lleva claves privadas de emisión
// ni de clientes, y la composición de seguridad se monta igual. El proceso
// combinado sigue exigiéndolas, como hasta ahora.
func TestMaterialSeparadoComponeSeguridadSinClavesDeEmision(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	separarMaterial(t, cfg.DevelopmentMaterialDir, separacionportales.PortalInterno)
	cfg.PortalProceso = "interno"
	if _, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard); err != nil {
		t.Fatalf("la seguridad del interno separado debe componerse: %v", err)
	}
	cfg.PortalProceso = ""
	if _, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard); err == nil {
		t.Fatal("el proceso combinado conserva la exigencia historica de material")
	}
}

func TestProcesoCombinadoRechazaMaterialSeparado(t *testing.T) {
	vaciarConexionesDelEntorno(t)
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	separarMaterial(t, cfg.DevelopmentMaterialDir, separacionportales.PortalInterno)
	_, err := NewHTTPServerWithConfig(cfg)
	if !errors.Is(err, separacionportales.ErrSeparacionPortales) {
		t.Fatalf("el combinado no debe usar material separado: %v", err)
	}
}

// Con su material propio y sin nada del externo, el proceso interno pasa la
// separación y la composición de seguridad y sigue la composición de siempre:
// aquí se detiene solo porque la prueba no le da el PostgreSQL de RRHH.
func TestProcesoInternoSigueLaComposicionDeSiempre(t *testing.T) {
	vaciarConexionesDelEntorno(t)
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	separarMaterial(t, cfg.DevelopmentMaterialDir, separacionportales.PortalInterno)
	cfg.PortalProceso = "interno"
	_, err := NewHTTPServerWithConfig(cfg)
	if !errors.Is(err, config.ErrConfiguracionPostgreSQLContratacionTemporalIncompleta) {
		t.Fatalf("el interno debe llegar a la composicion de RRHH: %v", err)
	}
}
