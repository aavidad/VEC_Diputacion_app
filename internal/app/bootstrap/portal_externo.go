package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	publicatransitoria "vec-diputacion-granada/internal/app/composicion/publicatransitoria"
	"vec-diputacion-granada/internal/app/server"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	bolsapublicahttp "vec-diputacion-granada/internal/modules/bolsa/publico/httpapi"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vecports "vec-diputacion-granada/internal/vec/ports"
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
	if err := validarManifiestoPortalExterno(filepath.Join(cfg.DevelopmentMaterialDir, "manifiesto.json"), ca, certificadoServidor); err != nil {
		return vacio, err
	}
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
			// La CA emite también los certificados de RRHH, Intervención y
			// centros. El proceso externo solo acepta los de las personas
			// que conoce: cualquier otro corta la conexión en el saludo TLS.
			VerifyConnection: verificarClienteConocido(identidad),
		},
		identidad: identidad,
	}, nil
}

// verificarClienteConocido rechaza en el saludo TLS cualquier certificado de
// cliente, aunque lo haya emitido la CA común, cuya huella no esté registrada
// en el resolvedor del proceso.
func verificarClienteConocido(identidad *resolvedorIdentidadDesarrollo) func(tls.ConnectionState) error {
	return func(estado tls.ConnectionState) error {
		if identidad == nil || len(estado.PeerCertificates) == 0 || estado.PeerCertificates[0] == nil {
			return ErrMaterialPortalExternoInvalido
		}
		huella := sha256.Sum256(estado.PeerCertificates[0].Raw)
		if _, conocido := identidad.porHuella[huella]; !conocido {
			return ErrMaterialPortalExternoInvalido
		}
		return nil
	}
}

// validarManifiestoPortalExterno comprueba lo que el manifiesto del material
// dice del propio proceso externo: perfil de desarrollo no autoritativo, no
// migrable, y huellas de su CA y de su certificado de servidor. Los demás
// campos describen material que el externo no tiene y no se miran.
func validarManifiestoPortalExterno(ruta string, ca, servidor *x509.Certificate) error {
	contenido, err := leerFicheroMaterialSeguro(ruta, 64<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return ErrMaterialPortalExternoInvalido
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var manifiesto archivoManifiestoDesarrollo
	var sobrante any
	if decodificador.Decode(&manifiesto) != nil || !errors.Is(decodificador.Decode(&sobrante), io.EOF) ||
		manifiesto.Perfil != config.ExecutionProfileDevelopment ||
		manifiesto.Autoridad != AutoridadNoAutoritativa || manifiesto.MigrableAProduccion {
		return ErrMaterialPortalExternoInvalido
	}
	for _, dato := range []struct {
		declarada string
		cert      *x509.Certificate
	}{{manifiesto.HuellaCASHA256, ca}, {manifiesto.HuellaServidorSHA256, servidor}} {
		huella := sha256.Sum256(dato.cert.Raw)
		declarada, err := hex.DecodeString(strings.ToLower(dato.declarada))
		if err != nil || len(declarada) != sha256.Size || subtle.ConstantTimeCompare(declarada, huella[:]) != 1 {
			return ErrMaterialPortalExternoInvalido
		}
	}
	return nil
}

// nuevoServidorPortalExternoDesarrollo compone el proceso del portal externo:
// mTLS con la identidad de la persona candidata, consulta pública y los
// ficheros del Área personal. Las capacidades personales usan conexiones
// nominales externas; no abre conexiones de RRHH ni carga material interno.
func nuevoServidorPortalExternoDesarrollo(cfg config.Config, registro io.Writer, emisor vecports.EmisorIncidenciasTecnicas) (*http.Server, error) {
	cfg = cfg.Normalize()
	if registro == nil {
		return nil, marcarFalloComponenteArranque("registro_arranque", ErrRegistroArranqueDesarrollo)
	}
	if err := validarRedLocalDesarrollo(cfg); err != nil {
		return nil, marcarFalloComponenteArranque("red_local", err)
	}
	material, err := cargarMaterialPortalExterno(cfg)
	if err != nil {
		return nil, marcarFalloComponenteArranque("material_portal_externo", err)
	}
	consultaCategorias, _, err := nuevasDependenciasCategoriasProfesionales(cfg)
	if err != nil {
		return nil, marcarFalloComponenteArranque("categorias_profesionales", err)
	}
	cfgPublica := cfg
	cfgPublica.AuthMode = config.AuthModeDisabled
	publicaBolsaAPI, err := publicatransitoria.NuevaAPIConCatalogos(cfgPublica, consultaCategorias)
	if err != nil {
		return nil, marcarFalloComponenteArranque("bolsa_publica_api", err)
	}
	api := http.NewServeMux()
	api.Handle("/api/publico/", publicaBolsaAPI)
	dsnPublico, _ := cfg.ExternoBolsaPublicaPostgreSQL.DSN()
	ctxPublico, cancelarPublico := context.WithTimeout(context.Background(), plazoarranque.Ampliar(30*time.Second))
	bolsasPublicas, cerrarBolsasPublicas, err := nuevasBolsasPublicasPortalExterno(ctxPublico, cfg, dsnPublico)
	cancelarPublico()
	if err != nil {
		return nil, marcarFalloComponenteArranque("bolsas_publicas", err)
	}
	bolsapublicahttp.RegistrarRutasBolsasPublicas(api, bolsasPublicas)
	personal, cerrarPersonal, err := nuevasCapacidadesPersonalesPortalExterno(cfg, material.identidad, emisor)
	if err != nil {
		cerrarBolsasPublicas()
		return nil, marcarFalloComponenteArranque("capacidades_personales", err)
	}
	if personal != nil {
		api.Handle("/api/vec/", personal)
	}
	if err := avisarArranquePortalExterno(registro); err != nil {
		cerrarPersonal()
		cerrarBolsasPublicas()
		return nil, marcarFalloComponenteArranque("aviso_arranque_portal_externo", err)
	}
	servidor, err := server.NewHTTPServer(cfg, api)
	if err != nil {
		cerrarPersonal()
		cerrarBolsasPublicas()
		return nil, marcarFalloComponenteArranque("servidor_http", err)
	}
	servidor.TLSConfig = material.tls.Clone()
	servidor.RegisterOnShutdown(cerrarPersonal)
	servidor.RegisterOnShutdown(cerrarBolsasPublicas)
	return servidor, nil
}

// nuevasCapacidadesPersonalesPortalExterno compone las capacidades del Área
// personal que el proceso externo tenga encendidas: preferencias, consulta de
// Mi bolsa y, con el selector, acciones del candidato. Una capacidad activa
// sin infraestructura completa impide arrancar.
func nuevasCapacidadesPersonalesPortalExterno(cfg config.Config, identidad *resolvedorIdentidadDesarrollo, emisor vecports.EmisorIncidenciasTecnicas) (http.Handler, func(), error) {
	nada := func() {}
	for _, selector := range []string{envUsuariosCorreosDesarrollo, envUsuariosImagenDesarrollo} {
		if activo, err := selectorCapacidadRRHHDesarrollo(cfg, selector); err != nil || activo {
			return nil, nada, ErrUsuariosPortalExternoNoDisponible
		}
	}
	preferencias, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosPreferenciasDesarrollo)
	if err != nil {
		return nil, nada, err
	}
	portalCandidato, err := cfg.BolsaPortalCandidatoDesarrolloActivo()
	if err != nil {
		return nil, nada, err
	}
	_, sinBolsa := cfg.ExternoBolsaPostgreSQL.DSN()
	miBolsa := sinBolsa == nil
	if portalCandidato && !miBolsa {
		return nil, nada, errMiBolsaNoDisponible
	}
	if !preferencias && !miBolsa {
		return nil, nada, nil
	}
	if emisor == nil {
		return nil, nada, ErrEmisorIncidenciasRequerido
	}
	raiz := cfg.DevelopmentMaterialDir
	idempotencia, err := cargarMaterialIdempotenciaDesarrollo(raiz, filepath.Join(raiz, config.DevelopmentIdempotencyHMACConfigRelativePath))
	if err != nil {
		return nil, nada, ErrUsuariosPortalExternoNoDisponible
	}
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&idempotencia)
	idempotencia.borrar()
	if err != nil {
		return nil, nada, ErrUsuariosPortalExternoNoDisponible
	}
	// Los alias de sesión del externo viven en su propio espacio de clave.
	derivador.espacioSeudonimos = espacioSeudonimosPortalExterno
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(30*time.Second))
	defer cancelar()
	preflight, err := abrirPoolPreflightV3PortalExterno(ctx, cfg.ExternoPreflightV3DatabaseURL)
	if err != nil {
		derivador.borrar()
		return nil, nada, ErrUsuariosPortalExternoNoDisponible
	}
	cerrar := func() { preflight.Close(); derivador.borrar() }
	var personal http.Handler
	var autoridadPreferencias *autoridadPreferenciasUsuariosDesarrollo
	cerrarPreferencias := nada
	if preferencias {
		autoridadPreferencias, err = nuevasPreferenciasPortalExterno(ctx, cfg, identidad, derivador, emisor, preflight)
		if err != nil {
			cerrar()
			return nil, nada, err
		}
		cerrarPreferencias = autoridadPreferencias.cerrar
		cerrarTodo := func() { cerrarPreferencias(); cerrar() }
		manejador, err := nuevaAPIPersonalPortalExterno(identidad, emisor, autoridadPreferencias)
		if err != nil {
			cerrarTodo()
			return nil, nada, err
		}
		personal = manejador
	}
	bolsa, cerrarBolsa, err := nuevaMiBolsaPortalExterno(ctx, cfg, identidad, derivador, preflight, emisor)
	if err != nil {
		cerrarPreferencias()
		cerrar()
		return nil, nada, err
	}
	cerrarTodo := func() { cerrarBolsa(); cerrarPreferencias(); cerrar() }
	if bolsa == nil {
		return personal, cerrarTodo, nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.URL != nil && bolsahttp.EsRutaPortal(r.URL.Path) {
			bolsa.ServeHTTP(w, r)
			return
		}
		if personal != nil {
			personal.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	}), cerrarTodo, nil
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
