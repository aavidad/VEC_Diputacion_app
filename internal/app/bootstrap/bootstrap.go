package bootstrap

import (
	"context"
	"net/http"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/server"
	authadapter "vec-diputacion-granada/internal/candidate/adapters/auth"
	"vec-diputacion-granada/internal/candidate/adapters/handler"
	"vec-diputacion-granada/internal/candidate/adapters/repository"
	"vec-diputacion-granada/internal/candidate/application"
	candidatedomain "vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
	"vec-diputacion-granada/internal/candidate/usecases"
	bolsamodule "vec-diputacion-granada/internal/modules/bolsa"
	cronosmodule "vec-diputacion-granada/internal/modules/cronos"
	dietasmodule "vec-diputacion-granada/internal/modules/dietas"
	personalmodule "vec-diputacion-granada/internal/modules/personal"
	"vec-diputacion-granada/internal/shared/i18n"
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
	return NewDemoAPIWithConfig(config.Config{})
}

func NewDemoAPIWithConfig(cfg config.Config) (http.Handler, error) {
	bolsaAPI, err := newBolsaAPIWithConfig(cfg)
	if err != nil {
		return nil, err
	}
	vecAPI, err := NewVECShellAPI()
	if err != nil {
		return nil, err
	}
	return composeAPI(vecAPI, bolsaAPI), nil
}

func NewVECShellAPI() (http.Handler, error) {
	store := vecmemory.NewStore()
	service, err := vecapp.NewService(store, store, store)
	if err != nil {
		return nil, err
	}
	for _, manifest := range []vecdomain.ModuleManifest{
		personalmodule.Manifest(),
		cronosmodule.Manifest(),
		dietasmodule.Manifest(),
		bolsamodule.Manifest(),
	} {
		if err := service.RegisterModule(context.Background(), manifest); err != nil {
			return nil, err
		}
	}
	return vechttp.NewHandler(service)
}

func composeAPI(vecAPI http.Handler, fallback http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/vec", vecAPI)
	mux.Handle("/api/vec/", vecAPI)
	mux.Handle("/", fallback)
	return mux
}

func newBolsaAPIWithConfig(cfg config.Config) (http.Handler, error) {
	cfg = cfg.Normalize()
	candidates, merits, results, durable, err := demoRepositories(cfg)
	if err != nil {
		return nil, err
	}

	baremo, err := usecases.NewBaremoUseCase(merits, results)
	if err != nil {
		return nil, err
	}
	rules, err := demoRuleSet(application.DefaultCallID, "v1")
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
	authenticator, err := demoAuthenticator(cfg)
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

func demoAuthenticator(cfg config.Config) (ports.Authenticator, error) {
	cfg = cfg.Normalize()
	if cfg.AuthMode == config.AuthModeTrustedHeaders {
		return authadapter.NewTrustedHeadersAuthenticator(cfg)
	}
	citizen := ports.AuthPrincipal{
		Subject:   "cand-1",
		Role:      ports.AuthRoleCiudadano,
		Mechanism: ports.AuthMechanismClave,
	}
	staff := ports.AuthPrincipal{
		Subject:   "staff",
		Role:      ports.AuthRolePersonalInterno,
		Mechanism: ports.AuthMechanismKerberosAD,
	}
	authenticator, err := authadapter.NewFakeAuthenticator(citizen, staff)
	if err != nil {
		return nil, err
	}
	if err := authenticator.Register(citizen, "citizen-token"); err != nil {
		return nil, err
	}
	if err := authenticator.Register(staff, "staff-token"); err != nil {
		return nil, err
	}
	return authenticator, nil
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
