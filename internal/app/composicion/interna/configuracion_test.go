package interna

import (
	"errors"
	"reflect"
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
		{"reutiliza clave como CA", func(c *Configuracion) { c.AutoridadClientesTLS = c.ClaveServidorTLS }, ErrConfiguracionTLSIncompleta},
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
