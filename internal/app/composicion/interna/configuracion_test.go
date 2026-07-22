package interna

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func configuracionInternaValidaPrueba() Configuracion {
	return Configuracion{
		DireccionEscucha:       "10.7.15.40:8443",
		RedesPermitidas:        []string{"10.0.0.0/8"},
		CertificadoServidorTLS: "/run/secrets/vec-interno-servidor.crt",
		ClaveServidorTLS:       "/run/secrets/vec-interno-servidor.key",
		AutoridadClientesTLS:   "/run/secrets/vec-interno-clientes-ca.crt",
		NombreServidorTLS:      "servidor.interna.test",
		Audiencia:              "vec-interna",
		EmisorIdentidad:        "https://identidad.mulhacen.test",
		IdentidadesSANProxy:    []string{"dns:proxy-interno.mulhacen.test"},
	}
}

func TestConfiguracionInternaValidaContratoCorporativo(t *testing.T) {
	if err := configuracionInternaValidaPrueba().Validar(); err != nil {
		t.Fatalf("configuracion interna valida: %v", err)
	}
}

func TestConfiguracionInternaRechazaLimitesInseguros(t *testing.T) {
	pruebas := []struct {
		nombre  string
		alterar func(*Configuracion)
		error   error
	}{
		{"listener universal", func(c *Configuracion) { c.DireccionEscucha = "0.0.0.0:8443" }, ErrConfiguracionInternaInvalida},
		{"red universal", func(c *Configuracion) { c.RedesPermitidas = []string{"0.0.0.0/0"} }, ErrConfiguracionInternaInvalida},
		{"sin identidad proxy", func(c *Configuracion) { c.IdentidadesSANProxy = nil }, ErrConfiguracionInternaInvalida},
		{"sin certificado servidor", func(c *Configuracion) { c.CertificadoServidorTLS = "" }, ErrConfiguracionTLSIncompleta},
		{"certificado relativo", func(c *Configuracion) { c.CertificadoServidorTLS = "certificados/servidor.crt" }, ErrConfiguracionTLSIncompleta},
		{"certificado con control", func(c *Configuracion) { c.CertificadoServidorTLS = "/run/secrets/servidor\n.crt" }, ErrConfiguracionTLSIncompleta},
		{"reutiliza clave como CA", func(c *Configuracion) { c.AutoridadClientesTLS = c.ClaveServidorTLS }, ErrConfiguracionTLSIncompleta},
		{"sin nombre TLS servidor", func(c *Configuracion) { c.NombreServidorTLS = "" }, ErrConfiguracionTLSIncompleta},
		{"nombre TLS hostil", func(c *Configuracion) { c.NombreServidorTLS = "servidor\n.test" }, ErrConfiguracionTLSIncompleta},
		{"nombre TLS comodin", func(c *Configuracion) { c.NombreServidorTLS = "*.interna.test" }, ErrConfiguracionTLSIncompleta},
		{"selector fake", func(c *Configuracion) { c.SelectorAutenticacionHeredado = config.AuthModeFake }, ErrSelectorHeredadoProhibido},
		{"selector memoria", func(c *Configuracion) { c.SelectorAlmacenHeredado = config.StorageModeMemory }, ErrSelectorHeredadoProhibido},
		{"selector desarrollo", func(c *Configuracion) { c.SelectorPerfilHeredado = config.ExecutionProfileDevelopment }, ErrSelectorHeredadoProhibido},
		{"selector presentacion", func(c *Configuracion) { c.SelectorPresentacionHeredado = true }, ErrSelectorHeredadoProhibido},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			cfg := configuracionInternaValidaPrueba()
			prueba.alterar(&cfg)
			if err := cfg.Validar(); !errors.Is(err, prueba.error) {
				t.Fatalf("error = %v; se esperaba %v", err, prueba.error)
			}
		})
	}
}

func TestCargarConfiguracionInternaUsaListaPositiva(t *testing.T) {
	for _, clave := range []string{
		EnvDireccionEscuchaInterna,
		EnvRedesPermitidasInternas,
		EnvCertificadoTLSInterno,
		EnvClaveTLSInterna,
		EnvAutoridadClientesTLS,
		EnvNombreServidorTLS,
		EnvAudienciaInterna,
		EnvEmisorIdentidadInterna,
		EnvHuellasProxyTLSInternas,
		EnvIdentidadesSANProxy,
		config.EnvExecutionProfile,
		config.EnvAuthMode,
		config.LegacyEnvAuthMode,
		config.EnvStorageMode,
		config.LegacyEnvStorageMode,
		config.EnvDevelopmentGuard,
		config.EnvDevelopmentMaterialDir,
		config.EnvRRHHPresentationEnabled,
		config.EnvRRHHPresentationGuardOne,
		config.EnvRRHHPresentationGuardTwo,
	} {
		t.Setenv(clave, "")
	}
	t.Setenv(EnvDireccionEscuchaInterna, "10.7.15.40:8443")
	t.Setenv(EnvRedesPermitidasInternas, "10.0.0.0/8")
	t.Setenv(EnvCertificadoTLSInterno, "/run/secrets/servidor.crt")
	t.Setenv(EnvClaveTLSInterna, "/run/secrets/servidor.key")
	t.Setenv(EnvAutoridadClientesTLS, "/run/secrets/clientes-ca.crt")
	t.Setenv(EnvNombreServidorTLS, "servidor.interna.test")
	t.Setenv(EnvAudienciaInterna, "vec-interna")
	t.Setenv(EnvEmisorIdentidadInterna, "https://identidad.mulhacen.test")
	t.Setenv(EnvIdentidadesSANProxy, "dns:proxy-interno.mulhacen.test")

	antes := CargarConfiguracion()
	t.Setenv(config.EnvBolsaBorradoresEjecutorConsultaDatabaseURL, "VALOR_INTERNO_NO_LEER_UNO")
	t.Setenv(config.EnvBolsaBorradoresProyectorGobiernoDatabaseURL, "VALOR_INTERNO_NO_LEER_DOS")
	t.Setenv(config.EnvBolsaBorradoresVerificadorReciboDatabaseURL, "VALOR_INTERNO_NO_LEER_TRES")
	t.Setenv(config.EnvFakeCredentialsPath, "/interno/credenciales-no-autoritativas.json")
	despues := CargarConfiguracion()
	if !reflect.DeepEqual(antes, despues) {
		t.Fatalf("la configuracion interna leyo opciones fuera de su lista positiva:\nantes=%+v\ndespues=%+v", antes, despues)
	}
	if err := despues.Validar(); err != nil {
		t.Fatalf("configuracion ambiental interna: %v", err)
	}
}

func TestConfiguracionInternaNoReflejaValoresInvalidos(t *testing.T) {
	pruebas := []struct {
		nombre        string
		marcador      string
		errorEsperado error
		alterar       func(*Configuracion, string)
	}{
		{
			nombre:        "direccion",
			marcador:      "MARCADOR_DIRECCION_NO_REFLEJAR",
			errorEsperado: ErrConfiguracionInternaInvalida,
			alterar: func(cfg *Configuracion, marcador string) {
				cfg.DireccionEscucha = "10.7.15.40:8443\n" + marcador
			},
		},
		{
			nombre:        "CIDR",
			marcador:      "MARCADOR_CIDR_NO_REFLEJAR",
			errorEsperado: ErrConfiguracionInternaInvalida,
			alterar: func(cfg *Configuracion, marcador string) {
				cfg.RedesPermitidas = []string{"10.0.0.0/8\n" + marcador}
			},
		},
		{
			nombre:        "ruta certificado TLS",
			marcador:      "MARCADOR_CERTIFICADO_NO_REFLEJAR",
			errorEsperado: ErrConfiguracionTLSIncompleta,
			alterar: func(cfg *Configuracion, marcador string) {
				cfg.CertificadoServidorTLS = "certificados/" + marcador
			},
		},
		{
			nombre:        "ruta clave TLS",
			marcador:      "MARCADOR_CLAVE_NO_REFLEJAR",
			errorEsperado: ErrConfiguracionTLSIncompleta,
			alterar: func(cfg *Configuracion, marcador string) {
				cfg.ClaveServidorTLS = "claves/" + marcador
			},
		},
		{
			nombre:        "ruta CA TLS",
			marcador:      "MARCADOR_CA_NO_REFLEJAR",
			errorEsperado: ErrConfiguracionTLSIncompleta,
			alterar: func(cfg *Configuracion, marcador string) {
				cfg.AutoridadClientesTLS = "autoridades/" + marcador
			},
		},
		{
			nombre:        "audiencia identidad",
			marcador:      "MARCADOR_AUDIENCIA_NO_REFLEJAR",
			errorEsperado: ErrConfiguracionInternaInvalida,
			alterar: func(cfg *Configuracion, marcador string) {
				cfg.Audiencia = "vec-interna\n" + marcador
			},
		},
		{
			nombre:        "emisor identidad",
			marcador:      "MARCADOR_EMISOR_NO_REFLEJAR",
			errorEsperado: ErrConfiguracionInternaInvalida,
			alterar: func(cfg *Configuracion, marcador string) {
				cfg.EmisorIdentidad = "https://identidad.test/\n" + marcador
			},
		},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			cfg := configuracionInternaValidaPrueba()
			prueba.alterar(&cfg, prueba.marcador)

			comprobarErrorConfiguracionSaneado(
				t, cfg.Validar(), prueba.errorEsperado, prueba.marcador,
			)
			servidor, err := NuevoServidor(cfg)
			if servidor != nil {
				t.Fatalf("NuevoServidor devolvio servidor para configuracion hostil")
			}
			comprobarErrorConfiguracionSaneado(t, err, prueba.errorEsperado, prueba.marcador)
			servidor, err = construirServidorInterno(cfg, http.NotFoundHandler())
			if servidor != nil {
				t.Fatalf("constructor interno devolvio servidor para configuracion hostil")
			}
			comprobarErrorConfiguracionSaneado(t, err, prueba.errorEsperado, prueba.marcador)
		})
	}
}

func TestConfiguracionInternaSeRedactaAlSerializarYRegistrar(t *testing.T) {
	const marcador = "MARCADOR_CONFIGURACION_PRIVADA"
	cfg := configuracionInternaValidaPrueba()
	cfg.DireccionEscucha = "10.7.15.40:8443-" + marcador
	cfg.RedesPermitidas = []string{"10.0.0.0/8-" + marcador}
	cfg.CertificadoServidorTLS = "/run/secrets/" + marcador + ".crt"
	cfg.Audiencia = "audiencia-" + marcador

	jsonCfg, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	textoCfg, err := cfg.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	var registroEstructurado bytes.Buffer
	slog.New(slog.NewTextHandler(&registroEstructurado, nil)).Info(
		"configuracion", "valor", cfg,
	)
	var registroClasico bytes.Buffer
	log.New(&registroClasico, "", 0).Printf("configuracion=%+v", cfg)

	for nombre, salida := range map[string]string{
		"fmt valor":        fmt.Sprintf("%v", cfg),
		"fmt detallado":    fmt.Sprintf("%+v", cfg),
		"fmt Go":           fmt.Sprintf("%#v", cfg),
		"JSON":             string(jsonCfg),
		"texto":            string(textoCfg),
		"slog":             registroEstructurado.String(),
		"registro clasico": registroClasico.String(),
	} {
		if strings.Contains(salida, marcador) || strings.Contains(salida, "/run/secrets/") ||
			strings.Contains(salida, "10.7.15.40") || strings.Contains(salida, "10.0.0.0") {
			t.Errorf("%s revelo configuracion: %q", nombre, salida)
		}
		if !strings.Contains(salida, configuracionInternaRedactada) {
			t.Errorf("%s no aplico la marca de redaccion: %q", nombre, salida)
		}
	}
}

func comprobarErrorConfiguracionSaneado(t *testing.T, err, errorEsperado error, marcador string) {
	t.Helper()
	if !errors.Is(err, errorEsperado) {
		t.Fatalf("error = %v; se esperaba %v", err, errorEsperado)
	}
	var registro bytes.Buffer
	log.New(&registro, "", 0).Printf("validar configuracion: %v", err)
	for _, salida := range []string{err.Error(), registro.String()} {
		salida = strings.TrimSuffix(strings.TrimSuffix(salida, "\n"), "\r")
		if strings.Contains(salida, marcador) || strings.ContainsAny(salida, "\r\n") {
			t.Fatalf("error o log reflejo entrada hostil: %q", salida)
		}
	}
}
