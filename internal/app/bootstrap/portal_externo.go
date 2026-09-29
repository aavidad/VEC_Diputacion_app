package bootstrap

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"vec-diputacion-granada/config"
	publicatransitoria "vec-diputacion-granada/internal/app/composicion/publicatransitoria"
	"vec-diputacion-granada/internal/app/server"
)

// ErrMaterialPortalExternoInvalido rechaza un material del proceso externo
// incompleto o incoherente. No detalla qué fichero falla para no orientar a
// quien lo manipula; el operador lo localiza con comprobar-separacion-portales.
var ErrMaterialPortalExternoInvalido = errors.New("bootstrap: material del portal externo no valido")

// materialPortalExterno es lo único que el proceso externo lee de su
// directorio para atender: su TLS y la identidad de la persona candidata. No
// hay identidad de RRHH, Intervención ni centros, ni clave maestra de cifrado.
type materialPortalExterno struct {
	tls       *tls.Config
	identidad *resolvedorIdentidadDesarrollo
}

// cargarMaterialPortalExterno valida y carga el material del proceso externo.
// Reutiliza los mismos lectores acotados que la composición de desarrollo.
func cargarMaterialPortalExterno(cfg config.Config) (materialPortalExterno, error) {
	vacio := materialPortalExterno{}
	cfg = cfg.Normalize()
	if !cfg.DevelopmentEnabledByDoubleKey() || !filepath.IsAbs(cfg.DevelopmentMaterialDir) {
		return vacio, ErrActivacionDesarrolloInvalida
	}
	evaluada, err := filepath.EvalSymlinks(cfg.DevelopmentMaterialDir)
	if err != nil || evaluada != filepath.Clean(cfg.DevelopmentMaterialDir) || dentroDeRepositorioGit(cfg.DevelopmentMaterialDir) {
		return vacio, ErrMaterialPortalExternoInvalido
	}
	rutas := cfg.DevelopmentPaths()
	if cfg.TLSCertFile != rutas.ServerCertificate || cfg.TLSKeyFile != rutas.ServerPrivateKey {
		return vacio, ErrMaterialPortalExternoInvalido
	}
	if err := validarArbolMaterialDesarrollo(cfg.DevelopmentMaterialDir); err != nil {
		return vacio, ErrMaterialPortalExternoInvalido
	}
	caPEM, err := leerFicheroMaterialSeguro(rutas.CACertificate, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil {
		return vacio, ErrMaterialPortalExternoInvalido
	}
	servidorPEM, err := leerFicheroMaterialSeguro(rutas.ServerCertificate, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil {
		return vacio, ErrMaterialPortalExternoInvalido
	}
	servidorClavePEM, err := leerFicheroMaterialSeguro(rutas.ServerPrivateKey, tamanoMaximoFicheroMaterialDesarrollo)
	if err != nil {
		return vacio, ErrMaterialPortalExternoInvalido
	}
	defer borrarBytes(servidorClavePEM)
	ca, err := decodificarCertificadoUnico(caPEM)
	if err != nil || !ca.IsCA || ca.KeyUsage&x509.KeyUsageCertSign == 0 {
		return vacio, ErrMaterialPortalExternoInvalido
	}
	par, err := tls.X509KeyPair(servidorPEM, servidorClavePEM)
	if err != nil {
		return vacio, ErrMaterialPortalExternoInvalido
	}
	certificadoServidor, err := decodificarCertificadoUnico(servidorPEM)
	if err != nil {
		return vacio, ErrMaterialPortalExternoInvalido
	}
	par.Leaf = certificadoServidor
	raices := x509.NewCertPool()
	raices.AddCert(ca)
	if _, err := certificadoServidor.Verify(x509.VerifyOptions{
		DNSName: "localhost", Roots: raices, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}); err != nil {
		return vacio, ErrMaterialPortalExternoInvalido
	}
	// Sin persona candidata no hay nada que atender en el Área personal: el
	// proceso no arranca vacío.
	candidato, err := cargarIdentidadCandidatoBolsaDesarrollo(cfg.DevelopmentMaterialDir, ca)
	if err != nil || candidato == nil {
		return vacio, ErrMaterialPortalExternoInvalido
	}
	identidad, err := nuevoResolvedorIdentidadDesarrollo(candidato.identidad)
	if err != nil || identidad.registrarCandidatoBolsa(*candidato) != nil {
		return vacio, ErrMaterialPortalExternoInvalido
	}
	return materialPortalExterno{
		tls: &tls.Config{
			Certificates: []tls.Certificate{par},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    raices,
			MinVersion:   tls.VersionTLS13,
			MaxVersion:   tls.VersionTLS13,
		},
		identidad: identidad,
	}, nil
}

// nuevoServidorPortalExternoDesarrollo compone el proceso del portal externo:
// mTLS con la identidad de la persona candidata, consulta pública y los
// ficheros del Área personal. No abre ninguna conexión de RRHH ni carga
// material interno; las capacidades personales (Mi bolsa, preferencias,
// correos, imagen) se añaden en sus propias minitareas.
func nuevoServidorPortalExternoDesarrollo(cfg config.Config, registro io.Writer) (*http.Server, error) {
	cfg = cfg.Normalize()
	if registro == nil {
		return nil, ErrRegistroArranqueDesarrollo
	}
	if err := validarRedLocalDesarrollo(cfg); err != nil {
		return nil, err
	}
	material, err := cargarMaterialPortalExterno(cfg)
	if err != nil {
		return nil, err
	}
	consultaCategorias, _, err := nuevasDependenciasCategoriasProfesionales(cfg)
	if err != nil {
		return nil, err
	}
	cfgPublica := cfg
	cfgPublica.AuthMode = config.AuthModeDisabled
	publicaBolsaAPI, err := publicatransitoria.NuevaAPIConCatalogos(cfgPublica, consultaCategorias)
	if err != nil {
		return nil, err
	}
	api := http.NewServeMux()
	api.Handle("/api/publico/", publicaBolsaAPI)
	if err := avisarArranquePortalExterno(registro); err != nil {
		return nil, err
	}
	servidor, err := server.NewHTTPServer(cfg, api)
	if err != nil {
		return nil, err
	}
	servidor.TLSConfig = material.tls.Clone()
	return servidor, nil
}

// avisarArranquePortalExterno deja constancia ruidosa, como el resto de la
// composición de desarrollo, de que el proceso usa credenciales no
// autoritativas y atiende solo el portal externo.
func avisarArranquePortalExterno(registro io.Writer) error {
	aviso := struct {
		Nivel     string `json:"nivel"`
		Evento    string `json:"evento"`
		Perfil    string `json:"perfil"`
		Autoridad string `json:"autoridad"`
		Portal    string `json:"portal"`
	}{
		Nivel: "ADVERTENCIA", Evento: "arranque_con_credenciales_no_autoritativas",
		Perfil: config.ExecutionProfileDevelopment, Autoridad: AutoridadNoAutoritativa, Portal: "externo",
	}
	if err := json.NewEncoder(registro).Encode(aviso); err != nil {
		return fmt.Errorf("%w: %w", ErrRegistroArranqueDesarrollo, err)
	}
	return nil
}
