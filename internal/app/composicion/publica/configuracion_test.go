package publica

import (
	"reflect"
	"testing"

	"vec-diputacion-granada/config"
)

func TestCargarConfiguracionNoLeeSecretosInternos(t *testing.T) {
	for _, clave := range []string{
		config.EnvAddress,
		config.LegacyEnvAddress,
		config.EnvHTTPAllowedCIDRs,
		config.EnvTLSCertFile,
		config.EnvTLSKeyFile,
		config.EnvExecutionProfile,
		config.EnvAuthMode,
		config.LegacyEnvAuthMode,
		config.EnvDevelopmentGuard,
		config.EnvDevelopmentMaterialDir,
		config.EnvBolsaPublicSourcePath,
		config.EnvBolsaCategoriesSourcePath,
		config.EnvBolsaCategoriesCatalogID,
		config.EnvBolsaCategoriesVersion,
		config.EnvBolsaCategoriesSHA256,
		config.EnvBolsaCategoriesPublicProjectionSHA256,
		config.EnvBolsaPublicaDatabaseURL,
		config.EnvBolsaPublicaManifiestoSHA256,
		config.EnvRRHHPresentationEnabled,
		config.EnvRRHHPresentationGuardOne,
		config.EnvRRHHPresentationGuardTwo,
	} {
		t.Setenv(clave, "")
	}

	antes := CargarConfiguracion()
	if antes.CatalogoCategorias != "" || antes.VersionCategorias != 0 || antes.HuellaCategorias != "" ||
		antes.HuellaProyeccionCategorias != "" ||
		antes.FuenteConvocatorias != "" || antes.FuenteCategorias != "" {
		t.Fatalf("la raiz publica heredo selectores DEMO: %+v", antes)
	}
	for _, clave := range []string{
		config.EnvStorageMode,
		config.LegacyEnvStorageMode,
		config.EnvDataDir,
		config.LegacyEnvDataDir,
		config.EnvDataPath,
		config.LegacyEnvDataPath,
		config.EnvFakeCredentialsPath,
		config.EnvTrustedHeaderSubject,
		config.LegacyTrustedHeaderSubject,
		config.EnvTrustedHeaderRoles,
		config.LegacyTrustedHeaderRoles,
		config.EnvTrustedHeaderMechanism,
		config.LegacyTrustedHeaderMechanism,
		config.EnvTrustedProxyCIDRs,
		config.LegacyEnvTrustedProxyCIDRs,
		config.EnvPersonalCatalogPath,
		config.EnvOSRMBaseURL,
		config.EnvOSRMScopeName,
		config.EnvOSRMScopeBounds,
		config.EnvOSRMAllowedCIDRs,
		config.EnvOSRMGraphVersion,
		config.EnvBolsaBorradoresEjecutorConsultaDatabaseURL,
		config.EnvBolsaBorradoresProyectorGobiernoDatabaseURL,
		config.EnvBolsaBorradoresVerificadorReciboDatabaseURL,
	} {
		t.Setenv(clave, "VALOR_INTERNO_NO_LEER")
	}
	despues := CargarConfiguracion()

	if !reflect.DeepEqual(antes, despues) {
		t.Fatalf("la configuracion publica cambio al definir secretos internos:\nantes=%+v\ndespues=%+v", antes, despues)
	}
}

func TestCargarConfiguracionProyectaSoloOpcionesPublicas(t *testing.T) {
	t.Setenv(config.EnvAddress, "127.0.0.1:9090")
	t.Setenv(config.EnvHTTPAllowedCIDRs, "127.0.0.1/32, ::1/128")
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileProduction)
	t.Setenv(config.EnvAuthMode, config.AuthModeDisabled)
	t.Setenv(config.EnvBolsaCategoriesVersion, "7")
	t.Setenv(config.EnvBolsaPublicaDatabaseURL, "postgres://lector:secreto@db-publica/vec")
	t.Setenv(config.EnvBolsaPublicaManifiestoSHA256, "2a85abd0a1e78d828fe27baf619349caf8e4e8a3e0bf20815279dd98a966889a")

	cfg := CargarConfiguracion()
	if cfg.Direccion != "127.0.0.1:9090" || cfg.PerfilEjecucion != config.ExecutionProfileProduction ||
		cfg.AutenticacionSolicitada != config.AuthModeDisabled || cfg.VersionCategorias != 7 ||
		cfg.PostgreSQL.Validar() != nil || cfg.HuellaManifiesto == "" ||
		!reflect.DeepEqual(cfg.RedesPermitidas, []string{"127.0.0.1/32", "::1/128"}) {
		t.Fatalf("proyeccion publica inesperada: %+v", cfg)
	}
}
