//go:build baremo_pg_http

package bootstrap

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
)

// Los helpers del ensayo directo se cargan mediante overlay desde su commit
// conservado. El runner no instala SQL ni reconstruye una autoridad de sesión.
type configuracionEnsayoBaremoHTTP struct {
	Harness                 configuracionEnsayoBaremo `json:"harness"`
	AuditoriaSQL            string                    `json:"auditoria_sql"`
	AuditoriaPreContextoSQL string                    `json:"auditoria_pre_contexto_sql"`
}

type continuidadEnsayoBaremoHTTP struct {
	Recibo     json.RawMessage     `json:"recibo"`
	Resumen    resumenEnsayoBaremo `json:"resumen"`
	Postmaster time.Time           `json:"postmaster"`
}

func TestGobiernoReglasBaremoHTTPIntegracionPostgreSQL(t *testing.T) {
	if os.Getenv("VEC_BAREMO_HTTP_PG_CONFIG") == "" {
		t.Skip("ensayo HTTP PostgreSQL no configurado")
	}
	if os.Getenv("VEC_BAREMO_PG_DESECHABLE") != "si" {
		t.Fatal("requiere clon desechable explícito")
	}
	var c configuracionEnsayoBaremoHTTP
	leerJSONEnsayoBaremo(t, os.Getenv("VEC_BAREMO_HTTP_PG_CONFIG"), &c)
	cfg := c.Harness
	fase := os.Getenv("VEC_BAREMO_PG_FASE")
	nominales := RutasGobiernoReglasBaremoV3{Alta: bolsahttp.RutaAltaGobiernoReglasBaremoV3, Consulta: bolsahttp.RutaConsultaGobiernoReglasBaremoV3, Recuperar: bolsahttp.RutaRecuperarGobiernoReglasBaremoV3}
	if cfg.Version != 1 || cfg.Rutas != nominales || c.AuditoriaSQL == "" || c.AuditoriaPreContextoSQL == "" || !rutaPrivadaEnsayoBaremo(cfg.Continuidad) || !claveOperacionGobiernoV3(cfg.ClaveOperacion) {
		t.Fatal("configuración HTTP incompleta")
	}
	if fase != "preparar" && fase != "alta" && fase != "recuperar" && fase != "reinicio" {
		t.Fatal("fase desconocida")
	}
	cargarEntornoEnsayoBaremo(t, cfg)
	ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancelar()
	canon := ficheroPrivadoEnsayoBaremo(t, cfg.Conjunto)
	conjunto, err := reglas.RestaurarConjuntoReglasBaremo(canon)
	if err != nil {
		t.Fatal("conjunto sintético rechazado")
	}
	if fase == "preparar" {
		componerEnsayoBaremoReal(t, ctx, cfg, conjunto, true)
		return
	}
	servidor, cliente, ajeno, auditor, personaRef := servidorEnsayoBaremoHTTP(t, ctx, cfg)
	evidencia := poolEnsayoBaremo(t, ctx, cfg.EvidenciaDSN)
	alta := map[string]any{"reglas": json.RawMessage(canon), "motivo": cfg.Motivo, "clave_operacion": cfg.ClaveOperacion}
	antes, arranque := resumenPGEnsayoBaremo(t, ctx, evidencia, cfg)
	var guardado continuidadEnsayoBaremoHTTP
	_, err = os.Lstat(cfg.Continuidad)
	if os.IsNotExist(err) {
		if fase == "reinicio" {
			t.Fatal("reinicio sin continuidad previa")
		}
		esperados, estado := int64(0), 201
		if fase == "recuperar" {
			esperados, estado = 1, 200
		}
		if antes.Versiones != esperados || antes.Recibos != esperados || antes.Historias != esperados || antes.Outbox != esperados {
			t.Fatal("intención previa distinta")
		}
		respuesta := postEnsayoBaremoHTTP(t, ctx, cliente, servidor.URL+cfg.Rutas.Alta, alta, estado)
		var salida struct {
			Recibo json.RawMessage `json:"recibo"`
			Replay bool            `json:"replay"`
		}
		if json.Unmarshal(respuesta, &salida) != nil || salida.Replay != (estado == 200) || len(salida.Recibo) == 0 {
			t.Fatal("alta sin recibo HTTP")
		}
		resumen, postmaster := resumenPGEnsayoBaremo(t, ctx, evidencia, cfg)
		if resumen.Versiones != 1 || resumen.Recibos != 1 || resumen.Historias != 1 || resumen.Outbox != 1 || !shaHexGobiernoV3(resumen.Huella) || (estado == 200 && resumen != antes) {
			t.Fatal("efecto durable no único")
		}
		guardado = continuidadEnsayoBaremoHTTP{salida.Recibo, resumen, postmaster}
		guardarContinuidadHTTPBaremo(t, cfg.Continuidad, guardado)
	} else {
		leerJSONEnsayoBaremo(t, cfg.Continuidad, &guardado)
		if fase == "alta" || antes != guardado.Resumen || (fase == "reinicio" && !arranque.After(guardado.Postmaster)) {
			t.Fatal("continuidad o reinicio no acreditados")
		}
	}
	var recibo struct {
		Selector       json.RawMessage `json:"selector"`
		Huella         string          `json:"huella_solicitud_sha256"`
		Ref            string          `json:"recibo_ref"`
		Fecha          string          `json:"confirmada_en"`
		Disponibilidad string          `json:"disponibilidad"`
	}
	if json.Unmarshal(guardado.Recibo, &recibo) != nil || recibo.Ref == "" || recibo.Fecha == "" || recibo.Disponibilidad != "disponible_para_preparacion" || !shaHexGobiernoV3(recibo.Huella) {
		t.Fatal("recibo conservado inválido")
	}
	var replay struct {
		Recibo json.RawMessage `json:"recibo"`
		Replay bool            `json:"replay"`
	}
	if json.Unmarshal(postEnsayoBaremoHTTP(t, ctx, cliente, servidor.URL+cfg.Rutas.Alta, alta, 200), &replay) != nil || !replay.Replay || !bytes.Equal(replay.Recibo, guardado.Recibo) {
		t.Fatal("replay alteró recibo HTTP")
	}
	consulta := map[string]any{"selector": recibo.Selector, "motivo": cfg.Motivo}
	var version struct {
		Reglas         json.RawMessage `json:"reglas"`
		Disponibilidad string          `json:"disponibilidad"`
	}
	if json.Unmarshal(postEnsayoBaremoHTTP(t, ctx, cliente, servidor.URL+cfg.Rutas.Consulta, consulta, 200), &version) != nil || version.Disponibilidad != recibo.Disponibilidad {
		t.Fatal("consulta HTTP no confirmada")
	}
	canonVersion, err := reglas.CanonicalizarConjuntoReglasBaremoJSON(version.Reglas)
	if err != nil || !bytes.Equal(canonVersion, canon) {
		t.Fatal("consulta cambió canon")
	}
	consulta["clave_operacion"], consulta["huella_solicitud_sha256"] = cfg.ClaveOperacion, recibo.Huella
	var recuperado struct {
		Existe bool            `json:"existe"`
		Recibo json.RawMessage `json:"recibo"`
	}
	if json.Unmarshal(postEnsayoBaremoHTTP(t, ctx, cliente, servidor.URL+cfg.Rutas.Recuperar, consulta, 200), &recuperado) != nil || !recuperado.Existe || !bytes.Equal(recuperado.Recibo, guardado.Recibo) {
		t.Fatal("recuperación alteró recibo HTTP")
	}
	canonAjeno := ficheroPrivadoEnsayoBaremo(t, cfg.ConjuntoAjeno)
	audits := contarAuditoriaHTTPBaremo(t, ctx, evidencia, c.AuditoriaSQL, personaRef)
	precontexto := contarAuditoriaHTTPBaremo(t, ctx, evidencia, c.AuditoriaPreContextoSQL)
	conjuntoAjeno, err := reglas.RestaurarConjuntoReglasBaremo(canonAjeno)
	if err != nil || (conjuntoAjeno.Identidad().ConvocatoriaRef() == conjunto.Identidad().ConvocatoriaRef() && conjuntoAjeno.Identidad().ExpedienteRef() == conjunto.Identidad().ExpedienteRef()) {
		t.Fatal("negativo HTTP sin ámbito distinto")
	}
	altaAjena := map[string]any{"reglas": json.RawMessage(canonAjeno), "motivo": cfg.Motivo, "clave_operacion": cfg.ClaveOperacion}
	postEnsayoBaremoHTTP(t, ctx, cliente, servidor.URL+cfg.Rutas.Alta, altaAjena, 403)
	if contarAuditoriaHTTPBaremo(t, ctx, evidencia, c.AuditoriaSQL, personaRef) != audits+1 ||
		contarAuditoriaHTTPBaremo(t, ctx, evidencia, c.AuditoriaPreContextoSQL) != precontexto {
		t.Fatal("ámbito ajeno sin auditoría durable antes del PDP")
	}
	// Certificado real válido con rol ajeno: denegación de raíz con auditor
	// segregado real. Su cierre local prueba 503 sin tocar PostgreSQL compartido.
	postEnsayoBaremoHTTP(t, ctx, ajeno, servidor.URL+cfg.Rutas.Alta, alta, 403)
	if contarAuditoriaHTTPBaremo(t, ctx, evidencia, c.AuditoriaPreContextoSQL) != precontexto+1 ||
		contarAuditoriaHTTPBaremo(t, ctx, evidencia, c.AuditoriaSQL, personaRef) != audits+1 {
		t.Fatal("403 sin auditoría durable única")
	}
	auditor.Close()
	postEnsayoBaremoHTTP(t, ctx, ajeno, servidor.URL+cfg.Rutas.Alta, alta, 503)
	if contarAuditoriaHTTPBaremo(t, ctx, evidencia, c.AuditoriaPreContextoSQL) != precontexto+1 ||
		contarAuditoriaHTTPBaremo(t, ctx, evidencia, c.AuditoriaSQL, personaRef) != audits+1 {
		t.Fatal("auditor cerrado produjo otra fila")
	}
	final, _ := resumenPGEnsayoBaremo(t, ctx, evidencia, cfg)
	if final != guardado.Resumen {
		t.Fatal("HTTP o negativos modificaron historia conservada")
	}
	t.Log("HTTP real: alta/replay, consulta, recuperación, 403 auditado y 503; negocio conservado")
}

func guardarContinuidadHTTPBaremo(t *testing.T, ruta string, c continuidadEnsayoBaremoHTTP) {
	t.Helper()
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal("continuidad HTTP no serializable")
	}
	f, err := os.OpenFile(ruta, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal("continuidad HTTP no creada")
	}
	_, err = f.Write(b)
	syncErr, closeErr := f.Sync(), f.Close()
	if err != nil || syncErr != nil || closeErr != nil {
		t.Fatal("continuidad HTTP no confirmada")
	}
}

func contarAuditoriaHTTPBaremo(t *testing.T, ctx context.Context, p *pgxpool.Pool, consulta string, argumentos ...any) int64 {
	t.Helper()
	tx, err := p.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly, IsoLevel: pgx.RepeatableRead})
	if err != nil {
		t.Fatal("evidencia de auditoría no disponible")
	}
	defer tx.Rollback(context.Background())
	var n int64
	if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout='5s'; SET LOCAL search_path='pg_catalog'"); err != nil {
		t.Fatal("lectura acotada no disponible")
	}
	if tx.QueryRow(ctx, consulta, argumentos...).Scan(&n) != nil {
		t.Fatal("auditoría de rechazo no consultable")
	}
	return n
}

func postEnsayoBaremoHTTP(t *testing.T, ctx context.Context, cliente *http.Client, url string, cuerpo any, estado int) []byte {
	t.Helper()
	b, err := json.Marshal(cuerpo)
	if err != nil {
		t.Fatal("entrada HTTP no serializable")
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		t.Fatal("petición HTTP no creada")
	}
	r.Header.Set("Content-Type", "application/json")
	resp, err := cliente.Do(r)
	if err != nil {
		t.Fatal("transporte HTTP/mTLS no disponible")
	}
	defer resp.Body.Close()
	b, err = io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil || resp.StatusCode != estado || resp.Header.Get("Set-Cookie") != "" || resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("HTTP esperado=%d recibido=%d", estado, resp.StatusCode)
	}
	t.Logf("POST nominal HTTP %d", resp.StatusCode)
	return b
}

func servidorEnsayoBaremoHTTP(t *testing.T, ctx context.Context, cfg configuracionEnsayoBaremo) (*httptest.Server, *http.Client, *http.Client, *pgxpool.Pool, string) {
	t.Helper()
	c := config.Load()
	composicion, err := NuevaComposicionSeguridadDesarrollo(c, io.Discard)
	if err != nil {
		t.Fatal("composición común no disponible")
	}
	t.Cleanup(composicion.derivadorIdempotencia.borrar)
	d, err := nuevasDependenciasCT(c, composicion.identidad, composicion.derivadorIdempotencia, composicion.emisorKMS, io.Discard)
	if err != nil {
		t.Fatal("dependencias comunes no disponibles")
	}
	t.Cleanup(d.Cerrar)
	alta, material := baseNominalEnsayoBaremo(t, ctx, c, d)
	defer material.borrarCopiasEfimeras()
	alta.postgresql.proveedorMaterial, err = nuevoProveedorMaterialAltaContratacionTemporalDesarrollo(material, alta.soporte, d.reloj)
	if err != nil {
		t.Fatal("material común no disponible")
	}
	dsn, err := c.ContratacionTemporalPostgreSQL.DSNAuditoriaFronteraSeparado()
	if err != nil || !mismaBaseEnsayoBaremo(configuracionConexionEnsayoBaremo(t, dsn), configuracionConexionEnsayoBaremo(t, cfg.RuntimeDSN)) {
		t.Fatal("auditoría fuera del clon")
	}
	auditor, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, dsn, "vec-ensayo-http-baremo-audit", rolAuditoriaFronteraPostgreSQLContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatal("pool nominal auditor no disponible")
	}
	t.Cleanup(auditor.Close)
	if auditor.Config().ConnConfig.User == alta.postgresql.gobierno.Config().ConnConfig.User || auditor.Config().ConnConfig.User == alta.postgresql.registroAutorizacion.Config().ConnConfig.User || auditor.Config().ConnConfig.User == configuracionConexionEnsayoBaremo(t, cfg.RuntimeDSN).ConnConfig.User {
		t.Fatal("auditoría comparte LOGIN con otra autoridad")
	}
	registrador, err := pgvec.NuevoRegistradorAuditoriaFronteraRutaExactaPostgreSQL(auditor)
	if err != nil || registrador.PreflightAuditoriaFronteraRutaExacta(ctx) != nil {
		t.Fatal("CT168 no disponible")
	}
	alta.postgresql.registradorAuditoriaFrontera = registrador
	m, err := prepararMontajeGobiernoReglasBaremoHTTPV3(c, alta.soporte, d.reloj)
	if err != nil || m.perfil == nil || m.configuracion.ProvisionarPerfil {
		t.Fatal("binder exige perfil ya provisionado")
	}
	descriptores, err := fronterasGobiernoReglasBaremoHTTPV3(m.perfilRef)
	if err != nil {
		t.Fatal("descriptores nominales no disponibles")
	}
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo(descriptores)
	if err != nil {
		t.Fatal("catálogo común no disponible")
	}
	identidad, cerrar, err := nuevasDependenciasIdentidadConsultasDesarrollo(ctx, c.ContratacionTemporalPostgreSQL, &alta, composicion.derivadorIdempotencia, d.reloj, alta.soporte, fronteras)
	if err != nil {
		t.Fatal("sesión PostgreSQL común no disponible")
	}
	t.Cleanup(cerrar)
	rutas, cerrar, err := m.rutas(ctx, c, &alta, identidad, fronteras, composicion.derivadorIdempotencia, d.reloj)
	if err != nil {
		t.Fatal("binder final no compuesto")
	}
	t.Cleanup(cerrar)
	a := &autoridadConsultasContratacionTemporalDesarrollo{sello: d.sello, resolvedor: d.resolvedor, fronterasSeguridadComun: fronteras, registradorAuditoriaFronteraRutasExactas: registrador}
	h, err := vechttp.NewHandlerSoloRutasExactas(rutas, a, registrador)
	if err != nil {
		t.Fatal("raíz de rutas exactas no disponible")
	}
	s := httptest.NewUnstartedServer(a.proteger(h))
	s.TLS = composicion.tls.Clone()
	s.Config.ReadHeaderTimeout, s.Config.ReadTimeout, s.Config.WriteTimeout = 5*time.Second, 15*time.Second, 20*time.Second
	s.Config.ErrorLog = nil
	s.StartTLS()
	t.Cleanup(s.Close)
	p := c.DevelopmentPaths()
	cliente := clienteEnsayoBaremoHTTP(t, s, p.CACertificate, p.ClientCertificate, p.ClientPrivateKey)
	ajeno := clienteEnsayoBaremoHTTP(t, s, p.CACertificate, p.IntervencionCertificate, p.IntervencionPrivateKey)
	return s, cliente, ajeno, auditor, m.perfil.soporte.contexto.Resultado.Contexto.PersonaRef
}

func clienteEnsayoBaremoHTTP(t *testing.T, s *httptest.Server, ca, certificado, clave string) *http.Client {
	t.Helper()
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(ficheroPrivadoEnsayoBaremo(t, ca)) {
		t.Fatal("CA privada no disponible")
	}
	par, err := tls.X509KeyPair(ficheroPrivadoEnsayoBaremo(t, certificado), ficheroPrivadoEnsayoBaremo(t, clave))
	if err != nil {
		t.Fatal("certificado cliente no disponible")
	}
	servidor := s.Certificate()
	nombre := "127.0.0.1"
	if len(servidor.DNSNames) != 0 {
		nombre = servidor.DNSNames[0]
	}
	transporte := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: raices, Certificates: []tls.Certificate{par}, ServerName: nombre}, DisableKeepAlives: true}
	t.Cleanup(transporte.CloseIdleConnections)
	return &http.Client{Transport: transporte, Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
