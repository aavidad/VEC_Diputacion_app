package meritoshttp

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	cose "github.com/veraison/go-cose"
	merpg "vec-diputacion-granada/internal/modules/meritos/adapters/postgres"
	merapp "vec-diputacion-granada/internal/modules/meritos/application"
	merdom "vec-diputacion-granada/internal/modules/meritos/domain"
	merports "vec-diputacion-granada/internal/modules/meritos/ports"
	ctxpg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	idpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	authpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridad "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Solo se ejecuta con un fichero privado explícito. Usa las autoridades y
// adaptadores comunes reales; no instala SQL, concede permisos ni carga dobles.
type consultaIntegracionConfig struct {
	Crypto      consultaCryptoConfig           `json:"crypto"`
	Actors      map[string]consultaActorConfig `json:"actors"`
	Keys        map[string]consultaClaveConfig `json:"keys"`
	Passwords   map[string]string              `json:"passwords"`
	SocketDir   string                         `json:"socket_dir"`
	HechoRefs   []consultaCasoReal             `json:"hecho_refs"`
	Motivo      vd.ReferenciaEntradaCatalogo   `json:"motivo"`
	SourceRoot  string                         `json:"source_root"`
	ReadyFile   string                         `json:"ready_file"`
	HoldSeconds int                            `json:"hold_seconds"`
	Auditoria   consultaAuditoriaConfig        `json:"auditoria"`
}

type consultaAuditoriaConfig struct {
	Proceso            string                       `json:"proceso"`
	PlazoMS            int                          `json:"plazo_ms"`
	MotivoDenegacion   vd.ReferenciaEntradaCatalogo `json:"motivo_denegacion"`
	MotivoError        vd.ReferenciaEntradaCatalogo `json:"motivo_error"`
	RecursoConsultaRef string                       `json:"recurso_consulta_ref"`
}

type consultaCasoReal struct {
	HechoRef string `json:"hecho_ref"`
	Expected string `json:"expected"`
	Version  int    `json:"version"`
}

type consultaClaveConfig struct {
	ID                 string    `json:"id"`
	Version            uint64    `json:"version"`
	HMAC               []byte    `json:"hmac"`
	Issuer             string    `json:"issuer"`
	GovernmentRevision uint64    `json:"government_revision"`
	GovernmentSHA      string    `json:"government_sha"`
	From               time.Time `json:"from"`
	Until              time.Time `json:"until"`
}

type consultaCryptoConfig struct {
	Seed               []byte    `json:"seed"`
	HMAC               []byte    `json:"hmac"`
	RootID             string    `json:"root_id"`
	RootVersion        uint64    `json:"root_version"`
	Deployment         string    `json:"deployment"`
	Revision           string    `json:"revision"`
	Sequence           uint64    `json:"sequence"`
	Published          time.Time `json:"published"`
	Expires            time.Time `json:"expires"`
	RootFrom           time.Time `json:"root_from"`
	RootUntil          time.Time `json:"root_until"`
	KeyID              string    `json:"key_id"`
	KeyVersion         uint64    `json:"key_version"`
	Issuer             string    `json:"issuer"`
	GovernmentRevision uint64    `json:"government_revision"`
	GovernmentSHA      string    `json:"government_sha"`
	KeyFrom            time.Time `json:"key_from"`
	KeyUntil           time.Time `json:"key_until"`
}

type consultaActorConfig struct {
	Account           string `json:"account"`
	Profile           string `json:"profile"`
	Authentication    string `json:"authentication"`
	Session           string `json:"session"`
	ContextLogin      string `json:"context_login"`
	RevalidationLogin string `json:"revalidation_login"`
	SourceLogin       string `json:"source_login"`
	RegisterLogin     string `json:"register_login"`
	ReasonLogin       string `json:"reason_login"`
	RuntimeLogin      string `json:"runtime_login"`
	AuditLogin        string `json:"audit_login"`
}

type consultaRelojReal struct{}

func (consultaRelojReal) Ahora() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

type consultaProveedorReal struct {
	resolver func(context.Context) (merapp.SolicitudConsultaPropia, error)
}

func (p consultaProveedorReal) SolicitudConsultaPropia(ctx context.Context) (merapp.SolicitudConsultaPropia, error) {
	return p.resolver(ctx)
}

func consultaRaizRepositorio() (string, error) {
	actual, err := os.Getwd()
	if err != nil {
		return "", errors.New("consulta_integracion.directorio")
	}
	for {
		if st, err := os.Stat(filepath.Join(actual, "go.mod")); err == nil && st.Mode().IsRegular() {
			return actual, nil
		}
		padre := filepath.Dir(actual)
		if padre == actual {
			return "", errors.New("consulta_integracion.repositorio")
		}
		actual = padre
	}
}

func consultaFueraRepositorio(ruta, repo string) bool {
	if !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return false
	}
	rel, err := filepath.Rel(repo, ruta)
	return err == nil && (rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func consultaPrivadoPropio(st os.FileInfo, directorio bool) bool {
	u, ok := st.Sys().(*syscall.Stat_t)
	return ok && u.Uid == uint32(os.Geteuid()) && st.Mode().Perm()&0077 == 0 &&
		((directorio && st.IsDir()) || (!directorio && st.Mode().IsRegular()))
}

func consultaLeerConfig(ruta, repo string) (consultaIntegracionConfig, error) {
	var c consultaIntegracionConfig
	if !consultaFueraRepositorio(ruta, repo) {
		return c, errors.New("consulta_integracion.config_ubicacion")
	}
	real, err := filepath.EvalSymlinks(ruta)
	if err != nil || real != ruta {
		return c, errors.New("consulta_integracion.config_enlace")
	}
	st, err := os.Lstat(ruta)
	if err != nil || !consultaPrivadoPropio(st, false) || st.Size() <= 0 || st.Size() > 256*1024 {
		return c, errors.New("consulta_integracion.config_permisos")
	}
	f, err := os.Open(ruta)
	if err != nil {
		return c, errors.New("consulta_integracion.config_lectura")
	}
	defer f.Close()
	abierta, err := f.Stat()
	if err != nil || !os.SameFile(st, abierta) {
		return c, errors.New("consulta_integracion.config_cambiada")
	}
	d := json.NewDecoder(io.LimitReader(f, 256*1024+1))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF {
		return c, errors.New("consulta_integracion.config_json")
	}
	if len(c.Actors) != 1 || c.Actors["lector"].RuntimeLogin != "vec_rum04_lector" ||
		!vd.ReferenciaMotivoAutorizacionV2Valida(c.Motivo) || len(c.HechoRefs) == 0 || len(c.HechoRefs) > 16 || c.HoldSeconds < 0 || c.HoldSeconds > 300 {
		return c, errors.New("consulta_integracion.config_alcance")
	}
	for _, h := range c.HechoRefs {
		if !merdom.ReferenciaValida(h.HechoRef) || h.Version < 0 ||
			(h.Expected != "obtenida" && h.Expected != "no_encontrada" && h.Expected != "denegada" && h.Expected != "no_disponible") || h.Expected == "obtenida" && h.Version == 0 || h.Expected != "obtenida" && h.Version != 0 {
			return c, errors.New("consulta_integracion.config_caso")
		}
	}
	if err := consultaValidarCredenciales(c); err != nil {
		return c, err
	}
	return c, nil
}

func consultaPasswordValida(password string) bool {
	return len(password) > 0 && len(password) <= 1024 && !strings.ContainsAny(password, "\x00\r\n")
}

func consultaValidarCredenciales(c consultaIntegracionConfig) error {
	a, ok := c.Actors["lector"]
	logins := []string{a.ContextLogin, a.RevalidationLogin, a.SourceLogin, a.RegisterLogin, a.ReasonLogin, a.RuntimeLogin, a.AuditLogin}
	if !ok || len(c.Passwords) != len(logins) {
		return errors.New("consulta_integracion.credenciales_no_disponibles")
	}
	vistos := make(map[string]bool, len(logins))
	for _, login := range logins {
		if vistos[login] || !regexp.MustCompile(`^vec_rum0[34]_[a-z0-9_]{1,40}$`).MatchString(login) || !consultaPasswordValida(c.Passwords[login]) {
			return errors.New("consulta_integracion.credenciales_no_disponibles")
		}
		vistos[login] = true
	}
	return nil
}

func TestConsultaCredencialesNominalesConErrorSaneado(t *testing.T) {
	actor := consultaActorConfig{ContextLogin: "vec_rum03_contexto", RevalidationLogin: "vec_rum03_reval",
		SourceLogin: "vec_rum03_fuente", RegisterLogin: "vec_rum03_registro", ReasonLogin: "vec_rum03_motivos",
		RuntimeLogin: "vec_rum04_lector", AuditLogin: "vec_rum04_auditor"}
	for _, caso := range []string{"valida", "desconocida", "ausente", "vacia", "login_duplicado", "login_ajeno"} {
		t.Run(caso, func(t *testing.T) {
			c := consultaIntegracionConfig{Actors: map[string]consultaActorConfig{"lector": actor}, Passwords: map[string]string{}}
			for _, login := range []string{actor.ContextLogin, actor.RevalidationLogin, actor.SourceLogin, actor.RegisterLogin, actor.ReasonLogin, actor.RuntimeLogin, actor.AuditLogin} {
				c.Passwords[login] = "fixture_password_not_for_runtime"
			}
			switch caso {
			case "desconocida":
				delete(c.Passwords, actor.AuditLogin)
				c.Passwords["login_no_admitido"] = "fixture_password_not_for_runtime"
			case "ausente":
				delete(c.Passwords, actor.RuntimeLogin)
			case "vacia":
				c.Passwords[actor.RuntimeLogin] = ""
			case "login_duplicado":
				a := actor
				a.SourceLogin = a.ContextLogin
				c.Actors["lector"] = a
			case "login_ajeno":
				a := actor
				a.SourceLogin = "postgres"
				c.Actors["lector"] = a
			}
			err := consultaValidarCredenciales(c)
			if caso == "valida" {
				if err != nil {
					t.Fatal("rechaza el catálogo nominal de prueba")
				}
			} else if err == nil || err.Error() != "consulta_integracion.credenciales_no_disponibles" {
				t.Fatal("acepta una credencial no nominal o expone su contenido")
			}
		})
	}
	if _, err := consultaAbrirPool(context.Background(), "/socket_no_utilizado", actor.RuntimeLogin, ""); err == nil || err.Error() != "consulta_integracion.credenciales_no_disponibles" {
		t.Fatal("intenta conectar sin credencial o expone un diagnóstico")
	}
}

func consultaAbrirPool(ctx context.Context, socket, login, password string) (*pgxpool.Pool, error) {
	if !consultaPasswordValida(password) {
		return nil, errors.New("consulta_integracion.credenciales_no_disponibles")
	}
	if !regexp.MustCompile(`^vec_rum0[34]_[a-z0-9_]{1,40}$`).MatchString(login) || !filepath.IsAbs(socket) || filepath.Clean(socket) != socket {
		return nil, errors.New("consulta_integracion.pool_alcance")
	}
	real, err := filepath.EvalSymlinks(socket)
	st, statErr := os.Stat(socket)
	if err != nil || statErr != nil || real != socket || !consultaPrivadoPropio(st, true) {
		return nil, errors.New("consulta_integracion.socket_permisos")
	}
	cfg, err := pgxpool.ParseConfig("host=/socket port=5432 dbname=postgres sslmode=disable user=vec_rum04_lector password=fixture_socket_only")
	if err != nil {
		return nil, errors.New("consulta_integracion.pool_config")
	}
	// Fija cada entrada de conexión; no hereda hosts alternativos, contraseñas,
	// TLS o base desde el entorno. El único destino es el socket privado dado.
	cfg.ConnConfig.Host, cfg.ConnConfig.Port, cfg.ConnConfig.Database, cfg.ConnConfig.User = socket, 5432, "postgres", login
	cfg.ConnConfig.Password, cfg.ConnConfig.TLSConfig, cfg.ConnConfig.Fallbacks = password, nil, nil
	cfg.ConnConfig.RuntimeParams = map[string]string{}
	cfg.ConnConfig.ConnectTimeout = 3 * time.Second
	cfg.MaxConns, cfg.MinConns = 1, 0
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.New("consulta_integracion.pool_abrir")
	}
	if p.Ping(ctx) != nil {
		p.Close()
		return nil, errors.New("consulta_integracion.pool_ping")
	}
	return p, nil
}

// La fábrica separa las cuentas técnicas. El consumo V3 audita éxito y
// ausencia dentro de la lectura; la autoridad común registra los fallos.
func consultaComponerReal(ctx context.Context, c consultaIntegracionConfig) (http.Handler, func(), error) {
	pools := make([]*pgxpool.Pool, 0, 7)
	var privada ed25519.PrivateKey
	cerrar := func() {
		clear(privada)
		for _, p := range pools {
			p.Close()
		}
	}
	fallo := func(codigo string) (http.Handler, func(), error) { cerrar(); return nil, func() {}, errors.New(codigo) }
	a, ok := c.Actors["lector"]
	if !ok {
		return fallo("consulta_integracion.actor")
	}
	abrir := func(login string) (*pgxpool.Pool, error) {
		p, err := consultaAbrirPool(ctx, c.SocketDir, login, c.Passwords[login])
		if err == nil {
			pools = append(pools, p)
		}
		return p, err
	}
	cp, err := abrir(a.ContextLogin)
	if err != nil {
		return fallo("consulta_integracion.pool_contexto")
	}
	rp, err := abrir(a.RevalidationLogin)
	if err != nil {
		return fallo("consulta_integracion.pool_revalidacion")
	}
	sp, err := abrir(a.SourceLogin)
	if err != nil {
		return fallo("consulta_integracion.pool_fuente")
	}
	dp, err := abrir(a.RegisterLogin)
	if err != nil {
		return fallo("consulta_integracion.pool_registro")
	}
	mp, err := abrir(a.ReasonLogin)
	if err != nil {
		return fallo("consulta_integracion.pool_motivo")
	}
	ep, err := abrir(a.RuntimeLogin)
	if err != nil {
		return fallo("consulta_integracion.pool_lectura")
	}
	resolver, err := ctxpg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, cp)
	if err != nil {
		return fallo("consulta_integracion.contexto_resolutor")
	}
	contextService, err := app.NuevoServicioContextoActorProductivoV2(resolver, ctxpg.NuevoGeneradorOperacionContextoActorV2Criptografico(), consultaRelojReal{})
	if err != nil {
		return fallo("consulta_integracion.contexto_servicio")
	}
	actorAuthority, err := app.NuevaAutoridadContextoActorRegistradoV2(contextService)
	if err != nil {
		return fallo("consulta_integracion.contexto_autoridad")
	}
	revalidator, err := idpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, rp)
	if err != nil {
		return fallo("consulta_integracion.revalidador")
	}
	provider := consultaProveedorReal{resolver: func(requestContext context.Context) (merapp.SolicitudConsultaPropia, error) {
		link, result, err := vd.CrearVinculoAutenticacionActorV2ConResultado(requestContext, revalidator,
			vd.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: a.Authentication, SesionRef: a.Session}, actorAuthority,
			vd.SolicitudContextoActor{Cuenta: vd.CuentaAutenticadaContextoActor{CuentaRef: a.Account, Metodo: vd.AuthMethodCertificate, Garantia: vd.AuthAssuranceHigh}, PerfilActivoRef: a.Profile}, consultaRelojReal{})
		if err != nil {
			if merapp.DenegacionConsultaReal(err) {
				return merapp.SolicitudConsultaPropia{}, vd.ErrAutorizacionDenegada
			}
			return merapp.SolicitudConsultaPropia{}, merports.ErrConsultaNoDisponible
		}
		correlation, err := vd.GenerarReferenciaCorrelacionAutorizacionV2(requestContext, seguridad.GeneradorReferenciasCriptograficas{})
		if err != nil {
			return merapp.SolicitudConsultaPropia{}, merports.ErrConsultaNoDisponible
		}
		return merapp.SolicitudConsultaPropia{Vinculo: link, Contexto: result, Correlacion: correlation, Motivo: c.Motivo}, nil
	}}
	source, err := authpg.NuevoAlmacenAutorizacion(sp)
	if err != nil {
		return fallo("consulta_integracion.fuente")
	}
	registration, err := authpg.NuevoAlmacenAutorizacion(dp)
	if err != nil {
		return fallo("consulta_integracion.registro")
	}
	reasons, err := authpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(mp, c.Motivo.CatalogoID)
	if err != nil {
		return fallo("consulta_integracion.motivos")
	}
	authorization, err := app.NuevoServicioAutorizacionSolicitudLigadaV3(source, registration, registration, reasons, consultaRelojReal{}, seguridad.GeneradorReferenciasCriptograficas{}, app.ConfiguracionServicioAutorizacion{})
	if err != nil {
		return fallo("consulta_integracion.autorizacion")
	}
	crypto := c.Crypto
	if len(c.Keys) > 0 {
		k, ok := c.Keys[merapp.AudienciaConsultaPropia]
		if !ok {
			return fallo("consulta_integracion.clave_audiencia")
		}
		crypto.KeyID, crypto.KeyVersion, crypto.HMAC, crypto.Issuer = k.ID, k.Version, k.HMAC, k.Issuer
		crypto.GovernmentRevision, crypto.GovernmentSHA, crypto.KeyFrom, crypto.KeyUntil = k.GovernmentRevision, k.GovernmentSHA, k.From, k.Until
	}
	cfg, key, err := consultaConfianzaReal(crypto)
	if err != nil {
		return fallo("consulta_integracion.confianza")
	}
	privada = key
	verification, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(cfg, consultaRelojReal{})
	if err != nil {
		return fallo("consulta_integracion.verificador")
	}
	header := vd.CabeceraAtestacionAutorizacionV3{FormatoVersion: vd.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: crypto.RootID, Audiencia: crypto.Deployment}
	attestor, err := app.NuevoServicioAtestacionesAutorizacionV3(header, consultaFirmanteReal{privada, header})
	if err != nil {
		return fallo("consulta_integracion.atestador")
	}
	hmacKey, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(crypto.KeyID, crypto.KeyVersion, crypto.HMAC, crypto.Issuer, merapp.AudienciaConsultaPropia, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, crypto.KeyFrom, crypto.KeyUntil, time.Time{}, crypto.GovernmentRevision, crypto.GovernmentSHA)
	if err != nil {
		return fallo("consulta_integracion.clave_hmac")
	}
	emitter, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(hmacKey, consultaRelojReal{})
	if err != nil {
		return fallo("consulta_integracion.emisor")
	}
	common, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(authorization, attestor, verification, emitter)
	if err != nil {
		return fallo("consulta_integracion.material")
	}
	repository, err := merpg.NuevaConsulta(ep)
	if err != nil {
		return fallo("consulta_integracion.repositorio")
	}
	ap, err := abrir(a.AuditLogin)
	if err != nil {
		return fallo("consulta_integracion.pool_auditoria")
	}
	if c.Auditoria.PlazoMS <= 0 || c.Auditoria.PlazoMS > 30000 {
		return fallo("consulta_integracion.config_auditoria")
	}
	plazoAuditoria := time.Duration(c.Auditoria.PlazoMS) * time.Millisecond
	auditoria, err := authpg.NuevoRegistradorIntentosAuditoriaPostgreSQL(ap, c.Auditoria.Proceso,
		string(vd.SuperficieAutenticacionInternaCorporativaV1), plazoAuditoria)
	if err != nil || auditoria.PreflightIntentoAuditoria(ctx) != nil {
		return fallo("consulta_integracion.auditoria")
	}
	auditReasons, err := authpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(mp, c.Auditoria.MotivoDenegacion.CatalogoID)
	if err != nil {
		return fallo("consulta_integracion.motivos_auditoria")
	}
	service, err := merapp.NuevoServicioConsultaPropia(common, repository,
		merapp.ConfiguracionAuditoriaConsulta{Registrador: auditoria, Proceso: c.Auditoria.Proceso, Plazo: plazoAuditoria,
			ValidadorMotivos: auditReasons, MotivoDenegacion: c.Auditoria.MotivoDenegacion, MotivoError: c.Auditoria.MotivoError, RecursoConsultaRef: c.Auditoria.RecursoConsultaRef}, consultaRelojReal{})
	if err != nil {
		return fallo("consulta_integracion.servicio")
	}
	return NuevaConsultaPropia(provider, service), cerrar, nil
}

type consultaFirmanteReal struct {
	private ed25519.PrivateKey
	header  vd.CabeceraAtestacionAutorizacionV3
}

func (s consultaFirmanteReal) FirmarAtestacionAutorizacionV3(ctx context.Context, request vp.SolicitudFirmaAtestacionAutorizacionV3) (vp.ResultadoFirmaAtestacionAutorizacionV3, error) {
	if err := ctx.Err(); err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	header, err := request.Cabecera()
	if err != nil || header != s.header {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, errors.New("consulta_integracion.firma_cabecera")
	}
	payload, err := request.Mensaje()
	if err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	aad, err := confianza.AADExternoAtestacionAutorizacionV3(header.Audiencia)
	if err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	message := cose.NewSign1Message()
	message.Headers.Protected.SetAlgorithm(cose.AlgorithmEdDSA)
	message.Headers.Protected[cose.HeaderLabelKeyID] = []byte(header.ClaveID)
	message.Payload = payload
	provider, err := cose.NewSigner(cose.AlgorithmEdDSA, s.private)
	if err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	if err = message.Sign(rand.Reader, aad, provider); err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	message.Payload = nil
	message.Headers.RawProtected = nil
	message.Headers.RawUnprotected = nil
	signed, err := message.MarshalCBOR()
	if err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	return vp.NuevoResultadoFirmaAtestacionAutorizacionV3(request, signed, "evidencia:rum04:ejercicio-sintetico", consultaRelojReal{}.Ahora())
}

func consultaConfianzaReal(c consultaCryptoConfig) (confianza.ConfiguracionConfianzaAtestacionAutorizacionV3, ed25519.PrivateKey, error) {
	if len(c.Seed) != ed25519.SeedSize {
		return confianza.ConfiguracionConfianzaAtestacionAutorizacionV3{}, nil, errors.New("consulta_integracion.semilla")
	}
	key := ed25519.NewKeyFromSeed(c.Seed)
	root, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(c.RootID, c.RootVersion, key.Public().(ed25519.PublicKey), c.Deployment, confianza.EstadoClaveAtestacionAutorizacionV3Activa, c.RootFrom, c.RootUntil, time.Time{})
	if err != nil {
		clear(key)
		return confianza.ConfiguracionConfianzaAtestacionAutorizacionV3{}, nil, err
	}
	cfg, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(c.Revision, c.Sequence, c.Published, c.Expires, root)
	if err != nil {
		clear(key)
		return cfg, nil, err
	}
	return cfg, key, nil
}

// El servidor es un ensamblaje de ejercicio interno sintético. La cuenta
// confiable se fija en configuración privada: nunca se ofrece como login ni
// como frontera pública. Solo sirve esta lectura y sus recursos explícitos.
func consultaServidorLoopback(c consultaIntegracionConfig, repo string, handler http.Handler) (*httptest.Server, error) {
	if c.SourceRoot != repo || c.HoldSeconds == 0 || c.HechoRefs[0].Expected != "obtenida" {
		return nil, errors.New("consulta_integracion.servidor_alcance")
	}
	raiz, err := os.OpenRoot(repo)
	if err != nil {
		return nil, errors.New("consulta_integracion.servidor_raiz")
	}
	recursos := map[string]string{
		"/consulta-merito/": "web/static/portal-empleado/modulos/meritos/consulta-index.html",
		"/portal-empleado/modulos/meritos/consulta-index.html":      "web/static/portal-empleado/modulos/meritos/consulta-index.html",
		"/portal-empleado/modulos/meritos/consulta-montaje.js":      "web/static/portal-empleado/modulos/meritos/consulta-montaje.js",
		"/portal-empleado/modulos/meritos/consulta-cliente-http.js": "web/static/portal-empleado/modulos/meritos/consulta-cliente-http.js",
		"/portal-empleado/modulos/meritos/consulta-contrato.js":     "web/static/portal-empleado/modulos/meritos/consulta-contrato.js",
		"/portal-empleado/modulos/meritos/consulta-i18n.js":         "web/static/portal-empleado/modulos/meritos/consulta-i18n.js",
		"/portal-empleado/modulos/meritos/consulta-vista.js":        "web/static/portal-empleado/modulos/meritos/consulta-vista.js",
		"/portal-empleado/modulos/meritos/consulta-estilos.css":     "web/static/portal-empleado/modulos/meritos/consulta-estilos.css",
		"/comun/tema-vec.css":                     "web/static/comun/tema-vec.css",
		"/comun/textos.js":                        "web/static/comun/textos.js",
		"/comun/idioma.js":                        "web/static/comun/idioma.js",
		"/portal-empleado/portal.css":             "web/static/portal-empleado/portal.css",
		"/portal-empleado/portal-componentes.css": "web/static/portal-empleado/portal-componentes.css",
		"/textos/idiomas.json":                    "web/static/textos/idiomas.json",
		"/textos/es/meritos-consulta.json":        "web/static/textos/es/meritos-consulta.json",
		"/textos/en/meritos-consulta.json":        "web/static/textos/en/meritos-consulta.json",
	}
	var permitidoHost string
	servidor := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		if r.Host != permitidoHost || r.URL.EscapedPath() != r.URL.Path ||
			(r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+permitidoHost) ||
			(r.Header.Get("Sec-Fetch-Site") != "" && r.Header.Get("Sec-Fetch-Site") != "same-origin" && r.Header.Get("Sec-Fetch-Site") != "none") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.URL.Path == RutaConsultaPropia {
			ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
			defer cancel()
			handler.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if r.URL.Path == "/api/meritos/consulta-contexto" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			if r.Method != http.MethodGet || r.URL.RawQuery != "" || r.URL.ForceQuery {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()
			if _, err := handler.(*ConsultaPropia).proveedor.SolicitudConsultaPropia(ctx); err != nil {
				responderFalloConsulta(w, err)
				return
			}
			_ = json.NewEncoder(w).Encode(struct {
				HechoRef  string `json:"hecho_ref"`
				Sintetico bool   `json:"sintetico"`
			}{c.HechoRefs[0].HechoRef, true})
			return
		}
		relativa, ok := recursos[r.URL.Path]
		if !ok || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		f, err := raiz.Open(relativa)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil || !st.Mode().IsRegular() || st.Size() > 2*1024*1024 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.ServeContent(w, r, filepath.Base(relativa), st.ModTime(), f)
	}))
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		raiz.Close()
		return nil, errors.New("consulta_integracion.servidor_socket")
	}
	servidor.Listener.Close()
	servidor.Listener = listener
	permitidoHost = listener.Addr().String()
	servidor.Config.ReadHeaderTimeout = 3 * time.Second
	servidor.Config.ReadTimeout = 5 * time.Second
	servidor.Config.WriteTimeout = 25 * time.Second
	servidor.Config.IdleTimeout = 5 * time.Second
	servidor.Config.MaxHeaderBytes = 16 * 1024
	servidor.Config.RegisterOnShutdown(func() { _ = raiz.Close() })
	servidor.Start()
	return servidor, nil
}

func consultaPublicarPreparado(ruta, repo, serverURL string) error {
	if !consultaFueraRepositorio(ruta, repo) {
		return errors.New("consulta_integracion.ready_ubicacion")
	}
	padre := filepath.Dir(ruta)
	real, err := filepath.EvalSymlinks(padre)
	st, statErr := os.Stat(padre)
	if err != nil || statErr != nil || real != padre || !consultaPrivadoPropio(st, true) {
		return errors.New("consulta_integracion.ready_permisos")
	}
	f, err := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("consulta_integracion.ready_creacion")
	}
	defer f.Close()
	if json.NewEncoder(f).Encode(struct {
		URL       string `json:"url"`
		Sintetico bool   `json:"sintetico"`
	}{serverURL + "/consulta-merito/", true}) != nil || f.Sync() != nil {
		return errors.New("consulta_integracion.ready_escritura")
	}
	return nil
}

func TestConsultaIntegracionReal(t *testing.T) {
	ruta := os.Getenv("VEC_RUM04_CONFIG")
	if ruta == "" {
		t.Skip("consulta_integracion.config_no_indicada")
	}
	repo, err := consultaRaizRepositorio()
	if err != nil {
		t.Fatal("consulta_integracion.repositorio")
	}
	c, err := consultaLeerConfig(ruta, repo)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer clear(c.Crypto.Seed)
	defer clear(c.Crypto.HMAC)
	defer clear(c.Passwords)
	defer func() {
		for _, k := range c.Keys {
			clear(k.HMAC)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	handler, cleanup, err := consultaComponerReal(ctx, c)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer cleanup()
	for i, h := range c.HechoRefs {
		body, _ := json.Marshal(struct {
			HechoRef string `json:"hecho_ref"`
		}{h.HechoRef})
		r := httptest.NewRequest(http.MethodPost, RutaConsultaPropia, bytes.NewReader(body)).WithContext(ctx)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if h.Expected == "denegada" || h.Expected == "no_disponible" {
			estado := http.StatusForbidden
			if h.Expected == "no_disponible" {
				estado = http.StatusServiceUnavailable
			}
			if w.Code != estado || bytes.Contains(w.Body.Bytes(), []byte("hecho_actual")) || bytes.Contains(w.Body.Bytes(), []byte("recibo_consulta")) {
				t.Fatal("consulta_integracion.denegacion")
			}
		} else {
			var result merports.ResultadoConsultaPropia
			decodeErr := json.Unmarshal(w.Body.Bytes(), &result)
			if w.Code != http.StatusOK || decodeErr != nil || result.Codigo != h.Expected || result.ReciboConsulta == nil || result.ReciboConsulta.VersionConsultada != h.Version {
				t.Fatalf("consulta_integracion.resultado estado_http=%d codigo=%q", w.Code, result.Codigo)
			}
		}
		t.Logf("consulta_integracion caso=%d estado_http=%d", i, w.Code)
	}
	if c.HoldSeconds > 0 {
		server, err := consultaServidorLoopback(c, repo, handler)
		if err != nil {
			t.Fatal(err.Error())
		}
		defer server.Close()
		if err := consultaPublicarPreparado(c.ReadyFile, repo, server.URL); err != nil {
			t.Fatal(err.Error())
		}
		t.Log("consulta_integracion.loopback_preparado")
		timer := time.NewTimer(time.Duration(c.HoldSeconds) * time.Second)
		defer timer.Stop()
		<-timer.C
	}
}
