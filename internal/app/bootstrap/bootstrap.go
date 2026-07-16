package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/server"
	authadapter "vec-diputacion-granada/internal/candidate/adapters/auth"
	"vec-diputacion-granada/internal/candidate/adapters/handler"
	"vec-diputacion-granada/internal/candidate/adapters/repository"
	"vec-diputacion-granada/internal/candidate/application"
	candidatedomain "vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
	"vec-diputacion-granada/internal/candidate/usecases"
	adminmodule "vec-diputacion-granada/internal/modules/administracion"
	bolsamodule "vec-diputacion-granada/internal/modules/bolsa"
	bolsacatalogosvec "vec-diputacion-granada/internal/modules/bolsa/adapters/catalogosvec"
	bolsafichero "vec-diputacion-granada/internal/modules/bolsa/adapters/fichero"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppublico"
	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	cronosmodule "vec-diputacion-granada/internal/modules/cronos"
	dietasmodule "vec-diputacion-granada/internal/modules/dietas"
	personalmodule "vec-diputacion-granada/internal/modules/personal"
	personalcatalogosvec "vec-diputacion-granada/internal/modules/personal/adapters/catalogosvec"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/shared/i18n"
	vecfichero "vec-diputacion-granada/internal/vec/adapters/fichero"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecmemory "vec-diputacion-granada/internal/vec/adapters/memory"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func NewHTTPServer() (*http.Server, error) {
	return NewHTTPServerWithConfig(config.Load())
}

func NewHTTPServerWithConfig(cfg config.Config) (*http.Server, error) {
	api, err := NewDemoAPIWithConfig(cfg)
	if err != nil {
		return nil, err
	}
	return server.NewHTTPServer(cfg, api)
}

func NewDemoAPI() (http.Handler, error) {
	return NewDemoAPIWithConfig(config.Config{AuthMode: config.AuthModeFake})
}

func NewDemoAPIWithConfig(cfg config.Config) (http.Handler, error) {
	cfg = cfg.Normalize()
	credencialesFake, err := cargarAlmacenFakeConfigurado(cfg)
	if err != nil {
		return nil, err
	}
	consultaCategorias, categoriasPersonal, err := nuevasDependenciasCategoriasProfesionales(cfg)
	if err != nil {
		return nil, err
	}
	vecAPI, err := newVECShellAPICompuesta(cfg, credencialesFake, categoriasPersonal)
	if err != nil {
		return nil, err
	}
	publicaBolsaAPI, err := newBolsaPublicAPIConCatalogos(cfg, consultaCategorias)
	if err != nil {
		return nil, err
	}
	if cfg.AuthMode != config.AuthModeFake {
		return composeVECShellAPI(vecAPI, publicaBolsaAPI), nil
	}
	bolsaAPI, err := newBolsaAPIWithConfig(cfg, credencialesFake)
	if err != nil {
		return nil, err
	}
	return composeAPI(vecAPI, bolsaAPI, publicaBolsaAPI), nil
}

func NewVECShellAPI() (http.Handler, error) {
	return NewVECShellAPIWithConfig(config.Config{PersonalCatalogPath: "memory"})
}

func NewVECShellAPIWithConfig(cfg config.Config) (http.Handler, error) {
	cfg = cfg.Normalize()
	credencialesFake, err := cargarAlmacenFakeConfigurado(cfg)
	if err != nil {
		return nil, err
	}
	return newVECShellAPIWithConfig(cfg, credencialesFake)
}

func newVECShellAPIWithConfig(cfg config.Config, credencialesFake *almacenCredencialesFake) (http.Handler, error) {
	_, categoriasPersonal, err := nuevasDependenciasCategoriasProfesionales(cfg)
	if err != nil {
		return nil, err
	}
	return newVECShellAPICompuesta(cfg, credencialesFake, categoriasPersonal)
}

func newVECShellAPICompuesta(
	cfg config.Config,
	credencialesFake *almacenCredencialesFake,
	categoriasPersonal *personalapp.ServicioConsultaCategoriasProfesionales,
) (http.Handler, error) {
	store := vecmemory.NewStore()
	service, internalOperations, err := vecapp.NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		return nil, err
	}
	for _, manifest := range []vecdomain.ModuleManifest{
		personalmodule.Manifest(),
		cronosmodule.Manifest(),
		dietasmodule.Manifest(),
		bolsamodule.Manifest(),
		adminmodule.Manifest(),
	} {
		if err := internalOperations.RegisterModule(context.Background(), manifest); err != nil {
			return nil, err
		}
	}
	return vechttp.NewHandlerWithOptions(service, vechttp.HandlerOptions{
		InternalOperations:      internalOperations,
		PersonalCatalogPath:     cfg.PersonalCatalogPath,
		CategoriasProfesionales: categoriasPersonal,
		OSRMBaseURL:             cfg.OSRMBaseURL,
		OSRMScopeName:           cfg.OSRMScopeName,
		OSRMScopeBounds:         cfg.OSRMScopeBounds,
		OSRMAllowedCIDRs:        append([]string(nil), cfg.OSRMAllowedCIDRs...),
		AllowDemoIdentity:       cfg.AuthMode == config.AuthModeFake,
		DemoIdentityResolver:    credencialesFake,
		TrustIdentityHeaders:    cfg.AuthMode == config.AuthModeTrustedHeaders,
		TrustedProxyCIDRs:       append([]string(nil), cfg.TrustedProxyCIDRs...),
		IdentitySubjectHeader:   cfg.TrustedHeaderSubject,
		IdentityRolesHeader:     cfg.TrustedHeaderRoles,
		IdentityMechanismHeader: cfg.TrustedHeaderMechanism,
	})
}

func composeAPI(vecAPI http.Handler, fallback http.Handler, publicaBolsaAPI http.Handler) http.Handler {
	mux := http.NewServeMux()
	registrarBolsaPublica(mux, publicaBolsaAPI)
	mux.Handle("/api/vec", vecAPI)
	mux.Handle("/api/vec/", vecAPI)
	mux.Handle("/", fallback)
	return mux
}

// composeVECShellAPI no registra una ruta comodin. La API heredada de Bolsa
// solo se publica en el modo fake de demostracion hasta que autorice cada
// operacion mediante el PDP V1; en cualquier otro modo sus rutas no existen.
func composeVECShellAPI(vecAPI http.Handler, publicaBolsaAPI http.Handler) http.Handler {
	mux := http.NewServeMux()
	registrarBolsaPublica(mux, publicaBolsaAPI)
	mux.Handle("/api/vec", vecAPI)
	mux.Handle("/api/vec/", vecAPI)
	return mux
}

func registrarBolsaPublica(mux *http.ServeMux, publica http.Handler) {
	mux.Handle(bolsahttp.RutaConvocatorias, publica)
	mux.Handle(bolsahttp.RutaConvocatorias+"/", publica)
	mux.Handle(bolsahttp.RutaCategorias, publica)
}

func newBolsaPublicAPIConCatalogos(cfg config.Config, consultaCatalogos *vecfichero.ConsultaCatalogos) (http.Handler, error) {
	ruta, err := resolverRutaFuentePublica(cfg.BolsaPublicSourcePath)
	if err != nil {
		return nil, err
	}
	adaptador, err := bolsafichero.NuevaConsultaConvocatorias(ruta)
	if err != nil {
		return nil, err
	}
	categorias, err := bolsacatalogosvec.NuevaConsultaCategorias(
		consultaCatalogos,
		cfg.BolsaCategoriesCatalogID,
		cfg.BolsaCategoriesVersion,
	)
	if err != nil {
		return nil, err
	}
	servicio, err := bolsaapp.NuevoServicioConsultaPublica(adaptador, categorias, bolsaapp.RelojSistemaConsultaPublica{})
	if err != nil {
		return nil, err
	}
	ctxValidacion, cancelarValidacion := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelarValidacion()
	if err := servicio.ValidarConfiguracion(ctxValidacion); err != nil {
		return nil, errors.Join(errors.New("bootstrap: fuentes publicas de Bolsa incompatibles"), err)
	}
	return bolsahttp.NuevoHandler(servicio)
}

// nuevasDependenciasCategoriasProfesionales construye una sola instantanea
// inmutable para Bolsa y Personal. La tupla ID/version/huella se configura de
// forma expresa: nunca se resuelve implicitamente «la ultima version».
func nuevasDependenciasCategoriasProfesionales(
	cfg config.Config,
) (*vecfichero.ConsultaCatalogos, *personalapp.ServicioConsultaCategoriasProfesionales, error) {
	if cfg.BolsaCategoriesVersion < 1 {
		return nil, nil, errors.New("bootstrap: version de catalogo de categorias no valida")
	}
	rutaCategorias, err := resolverRutaFuentePublica(cfg.BolsaCategoriesSourcePath)
	if err != nil {
		return nil, nil, err
	}
	consultaCatalogos, err := vecfichero.NuevaConsultaCatalogos(rutaCategorias)
	if err != nil {
		return nil, nil, err
	}
	referencia := personalports.ReferenciaCatalogoCategoriasProfesionales{
		CatalogoID:           cfg.BolsaCategoriesCatalogID,
		CatalogoVersion:      cfg.BolsaCategoriesVersion,
		CatalogoHuellaSHA256: cfg.BolsaCategoriesSHA256,
	}
	adaptador, err := personalcatalogosvec.NuevaConsultaCategoriasProfesionales(consultaCatalogos, referencia)
	if err != nil {
		return nil, nil, errors.Join(errors.New("bootstrap: referencia de categorias profesionales no valida"), err)
	}
	servicio, err := personalapp.NuevoServicioConsultaCategoriasProfesionales(
		adaptador, personalapp.RelojSistemaCategoriasProfesionales{},
	)
	if err != nil {
		return nil, nil, err
	}
	ctxValidacion, cancelarValidacion := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelarValidacion()
	if _, err := servicio.ListarVigentes(ctxValidacion); err != nil {
		return nil, nil, errors.Join(errors.New("bootstrap: catalogo profesional gobernado incompatible"), err)
	}
	return consultaCatalogos, servicio, nil
}

func resolverRutaFuentePublica(configurada string) (string, error) {
	if filepath.IsAbs(configurada) {
		if info, err := os.Stat(configurada); err == nil && !info.IsDir() {
			return configurada, nil
		}
		return "", errors.New("bootstrap: fuente publica de Bolsa no disponible")
	}
	for _, candidata := range []string{configurada, filepath.Join("../../..", configurada)} {
		if info, err := os.Stat(candidata); err == nil && !info.IsDir() {
			return candidata, nil
		}
	}
	return "", errors.New("bootstrap: fuente publica de Bolsa no disponible")
}

func newBolsaAPIWithConfig(cfg config.Config, credencialesFake *almacenCredencialesFake) (http.Handler, error) {
	cfg = cfg.Normalize()
	candidates, merits, results, durable, err := demoRepositories(cfg)
	if err != nil {
		return nil, err
	}

	baremo, err := usecases.NewBaremoUseCase(merits, results)
	if err != nil {
		return nil, err
	}
	rules, err := demoRuleSet("convocatoria-demostracion", "v1")
	if err != nil {
		return nil, err
	}
	service, err := application.NewCandidateApplicationService(candidates, merits, baremo, rules)
	if err != nil {
		return nil, err
	}

	convocatorias, solicitudes := demoProcedureRepositories(durable)
	procedure, err := usecases.NewProcedureUseCase(convocatorias, solicitudes)
	if err != nil {
		return nil, err
	}
	administrative, err := demoAdministrativeFlow(durable)
	if err != nil {
		return nil, err
	}
	authenticator, err := demoAuthenticator(cfg, credencialesFake)
	if err != nil {
		return nil, err
	}
	catalog, _ := i18n.Load()
	return handler.NewHTTPHandlerWithModulesAndStatus(
		service,
		handler.NewProcedureDemoRunner(procedure),
		administrative,
		authenticator,
		catalog,
		bolsamodule.OperationalStatusForModes(true, cfg.AuthMode, cfg.StorageMode),
	)
}

func demoProcedureRepositories(durable *repository.DurableFileStore) (ports.ConvocatoriaRepository, ports.SolicitudRepository) {
	if durable != nil {
		return durable.ProcedureConvocatoriaRepository(), durable.ProcedureSolicitudRepository()
	}
	return repository.NewProcedureRepositories()
}

func demoRepositories(
	cfg config.Config,
) (ports.CandidateRepository, ports.MeritRepository, usecases.BaremoResultRepository, *repository.DurableFileStore, error) {
	if cfg.StorageMode == config.StorageModeFile {
		store, err := repository.NewDurableFileStore(cfg.DataPath)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		return store.CandidateRepository(), store.MeritRepository(), store.BaremoResultRepository(), store, nil
	}
	store := repository.NewMemoryStore()
	return repository.NewCandidateRepository(store), repository.NewMeritRepository(store), repository.NewBaremoResultRepository(), nil, nil
}

func demoAdministrativeFlow(durable *repository.DurableFileStore) (handler.AdministrativeFlowService, error) {
	var documents ports.CandidateDocumentRepository
	var claims ports.ClaimRepository
	var notifications ports.NotificationRepository
	var audit ports.AdministrativeAuditTrail
	if durable != nil {
		documents = durable.CandidateDocumentRepository()
		claims = durable.ClaimRepository()
		notifications = durable.NotificationRepository()
		audit = durable.AdministrativeAuditTrail()
	} else {
		store := repository.NewAdministrativeFlowMemoryStore()
		documents = repository.NewAdministrativeCandidateDocumentRepository(store)
		claims = repository.NewAdministrativeClaimRepository(store)
		notifications = repository.NewAdministrativeNotificationRepository(store)
		audit = repository.NewAdministrativeAuditTrail(store)
	}
	usecase, err := usecases.NewAdministrativeFlowUseCase(documents, claims, notifications, audit)
	if err != nil {
		return nil, err
	}
	return handler.NewAdministrativeFlowService(documents, usecase), nil
}

func cargarAlmacenFakeConfigurado(cfg config.Config) (*almacenCredencialesFake, error) {
	if cfg.AuthMode != config.AuthModeFake {
		return nil, nil
	}
	return cargarCredencialesFake(cfg.FakeCredentialsPath)
}

func demoAuthenticator(cfg config.Config, credencialesFake *almacenCredencialesFake) (ports.Authenticator, error) {
	cfg = cfg.Normalize()
	if cfg.AuthMode != config.AuthModeFake {
		// La API heredada de Bolsa aun autoriza por roles gruesos. Ni siquiera una
		// cabecera procedente de un proxy admitido puede convertir esos roles en
		// autoridad funcional: permanece cerrada hasta consumir el autorizador
		// RBAC+ABAC por operacion. La carcasa VEC puede autenticar la identidad por
		// su frontera separada, pero tampoco hereda permisos de la cabecera.
		return authadapter.NewFakeAuthenticator()
	}
	if credencialesFake == nil {
		return nil, errCredencialFakeNoValida
	}
	return credencialesFake, nil
}

func demoRuleSet(convocatoriaID string, version string) (candidatedomain.BaremoRuleSet, error) {
	return candidatedomain.NewBaremoRuleSet(candidatedomain.BaremoRuleSetConfig{
		ConvocatoriaID: convocatoriaID,
		Version:        version,
		SorteoLetra:    "A",
		MeritRules: []candidatedomain.BaremoMeritRule{
			{
				MeritType:     candidatedomain.MeritTypeExperienciaMismaCategoria,
				Section:       candidatedomain.BaremoSectionExperiencia,
				Unit:          candidatedomain.BaremoUnitMeses,
				PointsPerUnit: 0.2,
			},
			{
				MeritType:     candidatedomain.MeritTypeExperienciaOtraCategoria,
				Section:       candidatedomain.BaremoSectionExperiencia,
				Unit:          candidatedomain.BaremoUnitMeses,
				PointsPerUnit: 0.1,
			},
			{
				MeritType:     candidatedomain.MeritTypeFormacionTitulo,
				Section:       candidatedomain.BaremoSectionFormacion,
				Unit:          candidatedomain.BaremoUnitPuntosDeclarado,
				PointsPerUnit: 1,
			},
			{
				MeritType:     candidatedomain.MeritTypeFormacionCurso,
				Section:       candidatedomain.BaremoSectionFormacion,
				Unit:          candidatedomain.BaremoUnitHoras,
				PointsPerUnit: 0.05,
			},
			{
				MeritType:     candidatedomain.MeritTypeOtros,
				Section:       candidatedomain.BaremoSectionOtros,
				Unit:          candidatedomain.BaremoUnitPuntosDeclarado,
				PointsPerUnit: 1,
			},
		},
		SectionCaps: []candidatedomain.BaremoSectionCap{
			{Section: candidatedomain.BaremoSectionExperiencia, MaxPoints: 50},
			{Section: candidatedomain.BaremoSectionFormacion, MaxPoints: 30},
		},
		TieBreakRules: []candidatedomain.BaremoTieBreakRule{
			candidatedomain.BaremoTieMayorExperiencia,
			candidatedomain.BaremoTieMayorFormacion,
			candidatedomain.BaremoTieLetraSorteo,
		},
	})
}
