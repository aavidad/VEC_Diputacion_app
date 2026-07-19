package config

import "testing"

func TestAutenticacionParteDeshabilitadaYNoInfiereModo(t *testing.T) {
	for _, prueba := range []struct {
		nombre string
		modo   string
		quiere string
	}{
		{nombre: "ausente", quiere: AuthModeDisabled},
		{nombre: "desconocido no se degrada", modo: " Automatico ", quiere: "automatico"},
		{nombre: "deshabilitado expreso", modo: AuthModeDisabled, quiere: AuthModeDisabled},
		{nombre: "demostracion expresa", modo: AuthModeFake, quiere: AuthModeFake},
		{nombre: "cabeceras heredadas expresas", modo: AuthModeTrustedHeaders, quiere: AuthModeTrustedHeaders},
		{nombre: "desarrollo expreso", modo: AuthModeDevelopment, quiere: AuthModeDevelopment},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			obtenido := (Config{AuthMode: prueba.modo}).Normalize().AuthMode
			if obtenido != prueba.quiere {
				t.Fatalf("AuthMode = %q; se esperaba %q", obtenido, prueba.quiere)
			}
		})
	}
}

func TestPerfilDesconocidoNoSeConvierteEnProduccion(t *testing.T) {
	configuracion := (Config{ExecutionProfile: " Produccion-Azul "}).Normalize()
	if configuracion.ExecutionProfile != "produccion-azul" {
		t.Fatalf("perfil desconocido = %q; no debe convertirse en produccion", configuracion.ExecutionProfile)
	}

	t.Setenv(EnvExecutionProfile, " Produccion-Azul ")
	t.Setenv(EnvAuthMode, " Automatico ")
	cargada := Load()
	if cargada.ExecutionProfile != "produccion-azul" || cargada.AuthMode != "automatico" {
		t.Fatalf("Load degrado valores desconocidos: perfil=%q auth=%q", cargada.ExecutionProfile, cargada.AuthMode)
	}
}

func TestAlmacenamientoDesconocidoNoSeConvierteEnMemoria(t *testing.T) {
	configuracion := (Config{StorageMode: " Redis-Temporal "}).Normalize()
	if configuracion.StorageMode != "redis-temporal" {
		t.Fatalf("almacenamiento desconocido = %q; no debe convertirse en memoria", configuracion.StorageMode)
	}
}

func TestPerfilDesarrolloNoEsPredeterminadoYExigeDobleLlave(t *testing.T) {
	predeterminada := (Config{}).Normalize()
	if predeterminada.ExecutionProfile != ExecutionProfileProduction {
		t.Fatalf("perfil predeterminado = %q; debe ser produccion", predeterminada.ExecutionProfile)
	}
	if predeterminada.DevelopmentEnabledByDoubleKey() {
		t.Fatal("perfil desarrollo activado sin configuracion")
	}

	base := Config{
		ExecutionProfile: ExecutionProfileDevelopment,
		AuthMode:         AuthModeDevelopment,
		DevelopmentGuard: DevelopmentGuardAcknowledgement,
	}
	if !base.DevelopmentEnabledByDoubleKey() {
		t.Fatal("las dos llaves expresas y el modo desarrollo no activaron el perfil")
	}

	for nombre, mutar := range map[string]func(*Config){
		"sin perfil": func(c *Config) { c.ExecutionProfile = "" },
		"sin modo":   func(c *Config) { c.AuthMode = "" },
		"sin guarda": func(c *Config) { c.DevelopmentGuard = "" },
		"guarda mal escrita": func(c *Config) {
			c.DevelopmentGuard = "true"
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			configuracion := base
			mutar(&configuracion)
			if configuracion.DevelopmentEnabledByDoubleKey() {
				t.Fatal("una configuracion parcial activo desarrollo")
			}
		})
	}
}

func TestLoadCargaPerfilDesarrolloSinHeredarloDeBolsa(t *testing.T) {
	t.Setenv(EnvExecutionProfile, ExecutionProfileDevelopment)
	t.Setenv(EnvAuthMode, AuthModeDevelopment)
	t.Setenv(EnvDevelopmentGuard, DevelopmentGuardAcknowledgement)
	t.Setenv(EnvDevelopmentMaterialDir, " /estado-local/vec/desarrollo ")
	configuracion := Load()
	if !configuracion.DevelopmentEnabledByDoubleKey() {
		t.Fatal("Load no conservo la activacion explicita")
	}
	if configuracion.DevelopmentMaterialDir != "/estado-local/vec/desarrollo" {
		t.Fatalf("directorio material = %q", configuracion.DevelopmentMaterialDir)
	}
}

func TestPerfilDesarrolloDerivaSoloSusRutasLocalesYNoSeActivaPorVariableHeredada(t *testing.T) {
	t.Setenv(LegacyEnvAuthMode, AuthModeDevelopment)
	heredada := Load()
	if heredada.DevelopmentEnabledByDoubleKey() || heredada.ExecutionProfile != ExecutionProfileProduction {
		t.Fatalf("la variable heredada activo desarrollo: %+v", heredada)
	}

	raiz := "/estado-local/vec/desarrollo"
	configuracion := (Config{
		ExecutionProfile:       ExecutionProfileDevelopment,
		AuthMode:               AuthModeDevelopment,
		DevelopmentGuard:       DevelopmentGuardAcknowledgement,
		DevelopmentMaterialDir: raiz,
		TLSCertFile:            " ", TLSKeyFile: " ",
	}).Normalize()
	rutas := configuracion.DevelopmentPaths()
	if configuracion.TLSCertFile != rutas.ServerCertificate || configuracion.TLSKeyFile != rutas.ServerPrivateKey ||
		rutas.KMSAttestationKey != "/estado-local/vec/desarrollo/kms/atestacion-ed25519.key" ||
		rutas.KMSAttestationPublic != "/estado-local/vec/desarrollo/kms/atestacion-ed25519.pub" ||
		rutas.KMSRevalidationKey != "/estado-local/vec/desarrollo/kms/revalidacion-ed25519.key" ||
		rutas.KMSRevalidationPublic != "/estado-local/vec/desarrollo/kms/revalidacion-ed25519.pub" ||
		rutas.IdempotencyHMACConfig != "/estado-local/vec/desarrollo/idempotencia/configuracion.json" {
		t.Fatalf("rutas de desarrollo incoherentes: %+v / %+v", configuracion, rutas)
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

func TestPresentacionRRHHExigePerfilSelectorYDosGuardasLiterales(t *testing.T) {
	valida := Config{
		ExecutionProfile:         ExecutionProfileRRHHPresentation,
		RRHHPresentationEnabled:  true,
		RRHHPresentationGuardOne: RRHHPresentationGuardOneAcknowledgement,
		RRHHPresentationGuardTwo: RRHHPresentationGuardTwoAcknowledgement,
	}
	if !valida.RRHHPresentationEnabledByDoubleGuard() {
		t.Fatal("la configuracion completa no activo el perfil de presentacion")
	}
	mutaciones := []struct {
		nombre string
		muta   func(*Config)
	}{
		{"sin perfil", func(c *Config) { c.ExecutionProfile = ExecutionProfileProduction }},
		{"sin selector", func(c *Config) { c.RRHHPresentationEnabled = false }},
		{"primera guarda distinta", func(c *Config) { c.RRHHPresentationGuardOne = "ACEPTO" }},
		{"segunda guarda distinta", func(c *Config) { c.RRHHPresentationGuardTwo = "CONFIRMO" }},
	}
	for _, mutacion := range mutaciones {
		t.Run(mutacion.nombre, func(t *testing.T) {
			configuracion := valida
			mutacion.muta(&configuracion)
			if configuracion.RRHHPresentationEnabledByDoubleGuard() {
				t.Fatal("una configuracion parcial habilito la presentacion")
			}
			if !configuracion.HasRRHHPresentationSelectors() {
				t.Fatal("el selector parcial no sera detectable por produccion")
			}
		})
	}
}

func TestCargaGuardasPresentacionSinNormalizarLiterales(t *testing.T) {
	t.Setenv(EnvExecutionProfile, ExecutionProfileRRHHPresentation)
	t.Setenv(EnvRRHHPresentationEnabled, "true")
	t.Setenv(EnvRRHHPresentationGuardOne, RRHHPresentationGuardOneAcknowledgement)
	t.Setenv(EnvRRHHPresentationGuardTwo, RRHHPresentationGuardTwoAcknowledgement)
	if !Load().RRHHPresentationEnabledByDoubleGuard() {
		t.Fatal("Load no conservo la activacion literal completa")
	}
	t.Setenv(EnvRRHHPresentationGuardTwo, "confirmo_datos_sinteticos_sin_validez_administrativa")
	if Load().RRHHPresentationEnabledByDoubleGuard() {
		t.Fatal("una guarda con distinta capitalizacion fue aceptada")
	}
}

func TestCatalogoPersonalEnMemoriaConservaDecisionAlNormalizarVariasVeces(t *testing.T) {
	configuracion := (Config{PersonalCatalogPath: "memory"}).Normalize()
	if !configuracion.PersonalCatalogInMemory || configuracion.PersonalCatalogPath != "" {
		t.Fatalf("primera normalizacion = %+v", configuracion)
	}
	configuracion = configuracion.Normalize()
	if !configuracion.PersonalCatalogInMemory || configuracion.PersonalCatalogPath != "" {
		t.Fatalf("segunda normalizacion perdio el modo en memoria: %+v", configuracion)
	}
	porDefecto := (Config{}).Normalize()
	if porDefecto.PersonalCatalogInMemory || porDefecto.PersonalCatalogPath != DefaultPersonalCatalogPath {
		t.Fatalf("el valor ausente dejo de seleccionar el catalogo durable: %+v", porDefecto)
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
