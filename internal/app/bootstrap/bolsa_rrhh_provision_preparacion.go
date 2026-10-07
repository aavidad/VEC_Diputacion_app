package bootstrap

import (
	"context"
	"crypto/x509"
	"path/filepath"
	"time"

	"vec-diputacion-granada/config"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// PreparacionProvisionLecturasRRHHBolsa describe una propuesta, nunca una
// concesión ni la constancia de aprobación. No expone DNI, certificado o DSN.
type PreparacionProvisionLecturasRRHHBolsa struct {
	Estado           string `json:"estado"`
	VersionPreimagen int    `json:"version_preimagen"`
	VersionObjetivo  int    `json:"version_objetivo"`
	PreimagenSHA256  string `json:"preimagen_sha256"`
	ObjetivoSHA256   string `json:"objetivo_sha256"`
}

const estadoProvisionRRHHBolsaPendienteAprobacion = "pendiente_aprobacion"

// PrepararProvisionLecturasRRHHBolsaSinServidor sirve a una CLI privada de
// preparación. Carga la identidad sintética actual, comprueba el manifiesto
// nominal de Bolsa y lee el gobierno central en una transacción ReadOnly.
// No crea servidor, emisor V3 ni material de consumo; tampoco invoca el
// publicador de asignaciones. La fase de aplicación necesita un acto real
// aprobado y una operación CAS separada.
func PrepararProvisionLecturasRRHHBolsaSinServidor(
	ctx context.Context, cfg config.Config,
) (PreparacionProvisionLecturasRRHHBolsa, error) {
	var vacia PreparacionProvisionLecturasRRHHBolsa
	if ctx == nil || ctx.Err() != nil {
		return vacia, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	cfg = cfg.Normalize()
	if !cfg.DevelopmentEnabledByDoubleKey() || !cfg.BolsaBorradoresEnabled {
		return vacia, ErrActivacionDesarrolloInvalida
	}
	reloj := relojContratacionTemporalDesarrollo{}
	principal, err := cargarPrincipalPreparacionRRHHBolsa(cfg, reloj.Ahora())
	if err != nil {
		return vacia, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	contextoCT, err := nuevoContextoAltaContratacionTemporalDesarrollo(principal, reloj.Ahora())
	if err != nil {
		return vacia, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	soporteCT := &soporteAltaContratacionTemporalDesarrollo{
		sello:       &selloConsultasContratacionTemporalDesarrollo{},
		principalID: principal.ID, certificadoSHA256: principal.Attributes["certificate_sha256"],
		contexto: contextoCT, reloj: reloj,
	}
	soporteBolsa, err := nuevoSoporteSesionBorradorBolsaDesarrollo(cfg.DevelopmentMaterialDir, soporteCT, reloj.Ahora())
	if err != nil {
		return vacia, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	_, dsnGobierno, err := cfg.ContratacionTemporalPostgreSQL.DSNSeparados()
	if err != nil {
		return vacia, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	pool, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(
		ctx, dsnGobierno, "vec-bolsa-rrhh-provision-preparar", rolGobiernoPostgreSQLContratacionTemporalDesarrollo,
	)
	if err != nil {
		return vacia, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	defer pool.Close()
	lector := &lectorSoloLecturaProvisionRRHHBolsa{
		leer: func(ctx context.Context, perfilRef string) (instantaneaPublicadaDesarrollo, bool, error) {
			return leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, pool, perfilRef)
		},
	}
	return prepararProvisionLecturasRRHHBolsaDesdeLectura(ctx, soporteBolsa, lector, reloj)
}

func cargarPrincipalPreparacionRRHHBolsa(cfg config.Config, ahora time.Time) (dominiovec.Principal, error) {
	if !filepath.IsAbs(cfg.DevelopmentMaterialDir) ||
		filepath.Clean(cfg.DevelopmentMaterialDir) != cfg.DevelopmentMaterialDir ||
		dentroDeRepositorioGit(cfg.DevelopmentMaterialDir) {
		return dominiovec.Principal{}, ErrMaterialDesarrolloInvalido
	}
	canonico, err := filepath.EvalSymlinks(cfg.DevelopmentMaterialDir)
	if err != nil || canonico != cfg.DevelopmentMaterialDir {
		return dominiovec.Principal{}, ErrMaterialDesarrolloInvalido
	}
	rutas := cfg.DevelopmentPaths()
	caPEM, err := leerFicheroMaterialSeguro(rutas.CACertificate, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil {
		return dominiovec.Principal{}, ErrMaterialDesarrolloInvalido
	}
	clientePEM, err := leerFicheroMaterialSeguro(rutas.ClientCertificate, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil {
		return dominiovec.Principal{}, ErrMaterialDesarrolloInvalido
	}
	ca, err := decodificarCertificadoUnico(caPEM)
	if err != nil || !ca.IsCA || ca.KeyUsage&x509.KeyUsageCertSign == 0 {
		return dominiovec.Principal{}, ErrMaterialDesarrolloInvalido
	}
	cliente, err := decodificarCertificadoUnico(clientePEM)
	if err != nil {
		return dominiovec.Principal{}, ErrMaterialDesarrolloInvalido
	}
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	if _, err := cliente.Verify(x509.VerifyOptions{
		Roots: raices, CurrentTime: ahora, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}); err != nil {
		return dominiovec.Principal{}, ErrMaterialDesarrolloInvalido
	}
	identidad, err := cargarIdentidadDesarrollo(rutas.Identity, cliente, rolTecnicoRRHHContratacionTemporalDesarrollo)
	if err != nil || !principalContratacionTemporalDesarrolloValido(identidad.principal) {
		return dominiovec.Principal{}, ErrMaterialDesarrolloInvalido
	}
	return clonarPrincipalDesarrollo(identidad.principal), nil
}

// Este adaptador no posee método de escritura operativo. Satisface el contrato
// histórico de la política para reutilizar su comparación exacta de plantilla.
type lectorSoloLecturaProvisionRRHHBolsa struct {
	leer       func(context.Context, string) (instantaneaPublicadaDesarrollo, bool, error)
	mutaciones int
}

func (l *lectorSoloLecturaProvisionRRHHBolsa) leerAsignacionPublicada(
	ctx context.Context, perfilRef string,
) (instantaneaPublicadaDesarrollo, bool, error) {
	if l == nil || l.leer == nil {
		return instantaneaPublicadaDesarrollo{}, false, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	return l.leer(ctx, perfilRef)
}

func (l *lectorSoloLecturaProvisionRRHHBolsa) prepararInstantanea(
	context.Context, dominiovec.InstantaneaAutorizacion, bool,
) (dominiovec.InstantaneaAutorizacion, error) {
	if l != nil {
		l.mutaciones++
	}
	return dominiovec.InstantaneaAutorizacion{}, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
}

func (l *lectorSoloLecturaProvisionRRHHBolsa) publicarInstantaneaDesdePreimagen(
	context.Context, dominiovec.InstantaneaAutorizacion, dominiovec.InstantaneaAutorizacion,
) error {
	if l != nil {
		l.mutaciones++
	}
	return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
}

func prepararProvisionLecturasRRHHBolsaDesdeLectura(
	ctx context.Context, soporte *soporteSesionBorradorBolsaDesarrollo,
	lector *lectorSoloLecturaProvisionRRHHBolsa, reloj relojContratacionTemporalDesarrollo,
) (PreparacionProvisionLecturasRRHHBolsa, error) {
	var vacia PreparacionProvisionLecturasRRHHBolsa
	if lector == nil || lector.leer == nil || soporte == nil {
		return vacia, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	politica := &politicaBorradorLlamamientoBolsaDesarrollo{soporte: soporte, autoridad: lector, reloj: reloj}
	actual, objetivo, err := prepararProvisionLecturasNominalesRRHHBolsa(ctx, politica)
	if err != nil {
		return vacia, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	preimagen, errPre := actual.AsignacionPerfil.HuellaSHA256()
	huellaObjetivo, errObjetivo := objetivo.AsignacionPerfil.HuellaSHA256()
	if errPre != nil || errObjetivo != nil || preimagen == huellaObjetivo {
		return vacia, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	return PreparacionProvisionLecturasRRHHBolsa{
		Estado:           estadoProvisionRRHHBolsaPendienteAprobacion,
		VersionPreimagen: actual.VersionRol.Version, VersionObjetivo: objetivo.VersionRol.Version,
		PreimagenSHA256: preimagen, ObjetivoSHA256: huellaObjetivo,
	}, nil
}
