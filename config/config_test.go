package config

import "testing"

func TestAutenticacionParteDeshabilitadaYNoInfiereModo(t *testing.T) {
	for _, prueba := range []struct {
		nombre string
		modo   string
		quiere string
	}{
		{nombre: "ausente", quiere: AuthModeDisabled},
		{nombre: "desconocido", modo: "automatico", quiere: AuthModeDisabled},
		{nombre: "deshabilitado expreso", modo: AuthModeDisabled, quiere: AuthModeDisabled},
		{nombre: "demostracion expresa", modo: AuthModeFake, quiere: AuthModeFake},
		{nombre: "cabeceras heredadas expresas", modo: AuthModeTrustedHeaders, quiere: AuthModeTrustedHeaders},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			obtenido := (Config{AuthMode: prueba.modo}).Normalize().AuthMode
			if obtenido != prueba.quiere {
				t.Fatalf("AuthMode = %q; se esperaba %q", obtenido, prueba.quiere)
			}
		})
	}
}

func TestRedHTTPParteCerradaEnLoopbackYNoInfiereInternet(t *testing.T) {
	configuracion := (Config{}).Normalize()
	if configuracion.Address != "127.0.0.1:8080" {
		t.Fatalf("listener predeterminado = %q; debe ser loopback literal", configuracion.Address)
	}
	if len(configuracion.HTTPAllowedCIDRs) != 2 || configuracion.HTTPAllowedCIDRs[0] != "127.0.0.1/32" ||
		configuracion.HTTPAllowedCIDRs[1] != "::1/128" {
		t.Fatalf("red predeterminada = %#v; debe limitarse a loopback", configuracion.HTTPAllowedCIDRs)
	}
	explicita := (Config{HTTPAllowedCIDRs: []string{"0.0.0.0/0"}}).Normalize()
	if len(explicita.HTTPAllowedCIDRs) != 1 || explicita.HTTPAllowedCIDRs[0] != "0.0.0.0/0" {
		t.Fatalf("red explicita alterada: %#v", explicita.HTTPAllowedCIDRs)
	}
}

func TestCargaRutaCredencialesFakeSinInventarValor(t *testing.T) {
	t.Setenv(EnvFakeCredentialsPath, " /ruta/local/credenciales.json ")
	configuracion := Load()
	if configuracion.FakeCredentialsPath != "/ruta/local/credenciales.json" {
		t.Fatalf("FakeCredentialsPath = %q", configuracion.FakeCredentialsPath)
	}
}

func TestPresentacionRRHHParteDeshabilitadaYExigeActivacionExpresa(t *testing.T) {
	if Load().RRHHPresentationEnabled {
		t.Fatal("la presentacion RRHH se activo sin configuracion expresa")
	}
	for _, valor := range []string{"1", "true", "yes", "on"} {
		t.Run(valor, func(t *testing.T) {
			t.Setenv(EnvRRHHPresentationEnabled, valor)
			if !Load().RRHHPresentationEnabled {
				t.Fatalf("valor positivo %q rechazado", valor)
			}
		})
	}
	for _, valor := range []string{"0", "false", "quizas", "activado"} {
		t.Run("cerrado_"+valor, func(t *testing.T) {
			t.Setenv(EnvRRHHPresentationEnabled, valor)
			if Load().RRHHPresentationEnabled {
				t.Fatalf("valor ambiguo %q concedio acceso", valor)
			}
		})
	}
}

func TestFuentePublicaBolsaEsDemoLocalSustituible(t *testing.T) {
	configuracion := (Config{}).Normalize()
	if configuracion.BolsaPublicSourcePath != DefaultBolsaPublicSourcePath {
		t.Fatalf("BolsaPublicSourcePath = %q", configuracion.BolsaPublicSourcePath)
	}
	t.Setenv(EnvBolsaPublicSourcePath, " /fuentes/convocatorias.json ")
	if obtenida := Load().BolsaPublicSourcePath; obtenida != "/fuentes/convocatorias.json" {
		t.Fatalf("ruta configurada = %q", obtenida)
	}
}

func TestCatalogoCategoriasBolsaFijaFuenteIDYVersionSinRecompilar(t *testing.T) {
	configuracion := (Config{}).Normalize()
	if configuracion.BolsaCategoriesSourcePath != DefaultBolsaCategoriesSourcePath ||
		configuracion.BolsaCategoriesCatalogID != DefaultBolsaCategoriesCatalogID ||
		configuracion.BolsaCategoriesVersion != DefaultBolsaCategoriesVersion ||
		configuracion.BolsaCategoriesSHA256 != DefaultBolsaCategoriesSHA256 {
		t.Fatalf("catalogo predeterminado inesperado: %+v", configuracion)
	}

	t.Setenv(EnvBolsaCategoriesSourcePath, " /fuentes/catalogo-v7.json ")
	t.Setenv(EnvBolsaCategoriesCatalogID, " categorias-profesionales-provincia ")
	t.Setenv(EnvBolsaCategoriesVersion, "7")
	t.Setenv(EnvBolsaCategoriesSHA256, " aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa ")
	configuracion = Load()
	if configuracion.BolsaCategoriesSourcePath != "/fuentes/catalogo-v7.json" ||
		configuracion.BolsaCategoriesCatalogID != "categorias-profesionales-provincia" ||
		configuracion.BolsaCategoriesVersion != 7 ||
		configuracion.BolsaCategoriesSHA256 != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("configuracion explicita alterada: %+v", configuracion)
	}
}

func TestVersionCatalogoCategoriasInvalidaNoRetrocedeSilenciosamente(t *testing.T) {
	for _, valor := range []string{"abc", "0", "-1"} {
		t.Run(valor, func(t *testing.T) {
			t.Setenv(EnvBolsaCategoriesVersion, valor)
			if obtenida := Load().BolsaCategoriesVersion; obtenida >= 1 {
				t.Fatalf("version invalida %q se convirtio en %d", valor, obtenida)
			}
		})
	}
}

func TestOSRMNoInfiereRedesYCargaSoloLasExplicitas(t *testing.T) {
	configuracion := (Config{}).Normalize()
	if len(configuracion.OSRMAllowedCIDRs) != 0 {
		t.Fatalf("redes OSRM inferidas: %#v", configuracion.OSRMAllowedCIDRs)
	}

	t.Setenv(EnvOSRMBaseURL, "http://127.0.0.1:5000")
	t.Setenv(EnvOSRMScopeName, "Granada provincia + 15 km")
	t.Setenv(EnvOSRMScopeBounds, "36.45,-4.6,38.25,-2.15")
	t.Setenv(EnvOSRMAllowedCIDRs, "127.0.0.1/32, ::1/128")
	configuracion = Load()
	if len(configuracion.OSRMAllowedCIDRs) != 2 || configuracion.OSRMAllowedCIDRs[0] != "127.0.0.1/32" ||
		configuracion.OSRMAllowedCIDRs[1] != "::1/128" {
		t.Fatalf("redes OSRM explicitas alteradas: %#v", configuracion.OSRMAllowedCIDRs)
	}
}
