package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

func cargarEntornoEnsayoBaremo(t *testing.T, cfg configuracionEnsayoBaremo) {
	t.Helper()
	runtime := configuracionConexionEnsayoBaremo(t, cfg.RuntimeDSN)
	evidencia := configuracionConexionEnsayoBaremo(t, cfg.EvidenciaDSN)
	if !mismaBaseEnsayoBaremo(runtime, evidencia) || runtime.ConnConfig.User == evidencia.ConnConfig.User {
		t.Fatal("runtime y evidencia deben usar LOGIN distintos del mismo clon")
	}
	// La composición no debe heredar otra instancia del equipo. Se limpian
	// únicamente sus variables, sin inspeccionar ni imprimir sus valores.
	for _, entrada := range os.Environ() {
		clave, _, _ := strings.Cut(entrada, "=")
		if (strings.HasPrefix(clave, "VEC_") && !strings.HasPrefix(clave, "VEC_BAREMO_PG_")) || strings.HasPrefix(clave, "BOLSA_") {
			t.Setenv(clave, "")
		}
	}
	for clave, valor := range cfg.Entorno {
		if !strings.HasPrefix(clave, "VEC_") || strings.HasPrefix(clave, "VEC_BAREMO_PG_") {
			t.Fatal("variable ajena a configuración VEC")
		}
		if strings.HasSuffix(clave, "DATABASE_URL") {
			c := configuracionConexionEnsayoBaremo(t, valor)
			if !mismaBaseEnsayoBaremo(runtime, c) {
				t.Fatal("la composición contiene otra base distinta del clon")
			}
		}
		t.Setenv(clave, valor)
	}
}

func fronterasEnsayoBaremo(t *testing.T, p *PerfilGobiernoReglasBaremoV3, rutas RutasGobiernoReglasBaremoV3) catalogoFronterasComunDesarrollo {
	t.Helper()
	descriptores := make([]descriptorFronteraComunDesarrollo, 0, 3)
	for _, op := range []struct{ ruta, accion string }{
		{rutas.Alta, "bolsa.reglas_baremo.borrador.crear"},
		{rutas.Consulta, "bolsa.reglas_baremo.version.consultar"},
		{rutas.Recuperar, "bolsa.reglas_baremo.recibo.consultar"},
	} {
		descriptores = append(descriptores, descriptorFronteraComunDesarrollo{Clave: op.accion,
			Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: "POST", Ruta: op.ruta,
			PerfilesActivosRef: []string{p.PerfilRef()}, ClavePolitica: op.accion, ClaveCapacidad: op.accion})
	}
	c, err := nuevoCatalogoFronterasComunDesarrollo(descriptores)
	if err != nil {
		t.Fatal("fronteras nominales del ensayo inválidas")
	}
	return c
}

func (e *ensayoBaremoReal) credenciales(t *testing.T, ctx context.Context, ruta string) (context.Context, app.CredencialesGobiernoV3) {
	t.Helper()
	descriptor, ok := e.fronteras.resolver("POST", ruta)
	if !ok {
		t.Fatal("frontera del ensayo no declarada")
	}
	// La identidad y la vigencia se han cargado y verificado con el material
	// existente. El harness sella la invocación directa; no simula HTTP/mTLS.
	capacidad := capacidadConsultaContratacionTemporalDesarrollo{sello: e.perfil.soporte.sello,
		ruta: ruta, metodo: "POST", principal: e.principal, contextoOperacion: &contextoOperacionCTDesarrollo{},
		certificadoVerificadoEn: time.Now().UTC().Truncate(time.Microsecond), certificadoValidoHasta: e.certificadoValidoHasta}
	pedido := context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	pedido = context.WithValue(pedido, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: "POST", ruta: ruta, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: e.fronteras, descriptor: descriptor})
	c, err := e.broker.Credenciales(pedido)
	if err != nil {
		t.Fatal("credenciales de sesión real PostgreSQL no disponibles")
	}
	return pedido, c
}

func poolEnsayoBaremo(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()
	c := configuracionConexionEnsayoBaremo(t, dsn)
	c.MaxConns = 1
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	p, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		t.Fatal("pool del clon no disponible")
	}
	t.Cleanup(p.Close)
	if p.Ping(ctx) != nil {
		t.Fatal("pool del clon no conectado")
	}
	return p
}

func mismaBaseEnsayoBaremo(a, b *pgxpool.Config) bool {
	return a.ConnConfig.Host == b.ConnConfig.Host && a.ConnConfig.Port == b.ConnConfig.Port && a.ConnConfig.Database == b.ConnConfig.Database
}

func configuracionConexionEnsayoBaremo(t *testing.T, dsn string) *pgxpool.Config {
	t.Helper()
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c.ConnConfig.User == "" || !loopbackEnsayoBaremo(c.ConnConfig.Host) || c.ConnConfig.Port == 5432 || validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		t.Fatal("DSN de clon local rechazado")
	}
	for _, alternativa := range c.ConnConfig.Fallbacks {
		if alternativa.Host != c.ConnConfig.Host || alternativa.Port != c.ConnConfig.Port {
			t.Fatal("DSN con destino alternativo distinto del clon")
		}
	}
	return c
}

func resumenPGEnsayoBaremo(t *testing.T, ctx context.Context, pool *pgxpool.Pool, cfg configuracionEnsayoBaremo) (resumenEnsayoBaremo, time.Time) {
	t.Helper()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly, IsoLevel: pgx.RepeatableRead})
	if err != nil {
		t.Fatal("lectura de evidencia no disponible")
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout='5s'; SET LOCAL search_path='pg_catalog'"); err != nil {
		t.Fatal("lectura acotada rechazada")
	}
	var r resumenEnsayoBaremo
	var postmaster time.Time
	if tx.QueryRow(ctx, cfg.ResumenSQL, cfg.ClaveOperacion).Scan(&r.Versiones, &r.Recibos, &r.Historias, &r.Outbox, &r.Huella) != nil {
		t.Fatal("resumen durable no disponible")
	}
	if tx.QueryRow(ctx, "SELECT pg_catalog.pg_postmaster_start_time()").Scan(&postmaster) != nil {
		t.Fatal("arranque PostgreSQL no disponible")
	}
	return r, postmaster
}

// Negativos de campos en el broker real antes de emitir material. No mutan
// una decisión firmada ni afirman ensayo del consumidor SQL de AD144.
func camposCruzadosEnsayoBaremo(t *testing.T, ctx context.Context, cfg configuracionEnsayoBaremo, e *ensayoBaremoReal, selector bp.SelectorGobiernoReglasV3, recibo bp.ReciboAltaBorradorReglasV3) {
	t.Helper()
	for _, recuperacion := range []bool{false, true} {
		ruta, operacion, accion, campos := cfg.Rutas.Consulta, "consultar_exacta", "bolsa.reglas_baremo.version.consultar", []string{"estado_reglas_baremo", "recibo"}
		clave, huellaSolicitud := "", ""
		if recuperacion {
			ruta, operacion, accion, campos = cfg.Rutas.Recuperar, "recuperar_recibo", "bolsa.reglas_baremo.recibo.consultar", []string{"recibo"}
			clave, huellaSolicitud = cfg.ClaveOperacion, recibo.HuellaSolicitudSHA256
		}
		pedido, _ := e.credenciales(t, ctx, ruta)
		actual, err := e.broker.contexto(pedido, operacion)
		if err != nil {
			t.Fatal("contexto para cruce de campos no disponible")
		}
		motivo, err := vd.RepresentacionCanonicaMotivoAutorizacionV2(cfg.Motivo)
		if err != nil {
			t.Fatal("motivo no serializable")
		}
		contenido := selector.Estado.Contenido()
		m := app.MaterialGobiernoV3{Esquema: "vec.bolsa.gobierno-borrador.material.v3", Operacion: operacion,
			Accion: accion, ModuloID: "bolsa", TipoRecurso: "version_reglas_baremo_gobernada", Finalidad: "consulta_gobierno_reglas_baremo",
			PersonaRef: actual.Resultado.Contexto.PersonaRef, PerfilRef: e.perfil.PerfilRef(), ConvocatoriaRef: e.perfil.convocatoriaRef, ExpedienteRef: e.perfil.expedienteRef,
			Estado:         app.EstadoMaterialGobiernoV3{Referencia: contenido.Referencia(), Version: contenido.Version(), HuellaContenidoSHA256: contenido.HuellaSHA256(), Revision: selector.Estado.Revision(), HuellaEstadoSHA256: selector.Estado.HuellaEstadoSHA256()},
			ClaveOperacion: clave, HuellaSolicitudSHA256: huellaSolicitud, MotivoCanonico: motivo, SolicitadaEn: time.Now().UTC().Truncate(time.Microsecond).Format("2006-01-02T15:04:05.000000Z")}
		canon, err := json.Marshal(m)
		if err != nil {
			t.Fatal("material para cruce de campos rechazado")
		}
		h := sha256.Sum256(canon)
		s := bp.SolicitudMaterialGobiernoReglasV3{Operacion: operacion, Accion: accion, Finalidad: m.Finalidad, Campos: campos,
			Audiencia: app.AudienciaGobiernoBorradorReglasV3, Motivo: cfg.Motivo, MaterialCanonico: canon}
		s.Recurso.Referencia, s.Recurso.ModuloID, s.Recurso.Tipo = "reglas-baremo:"+m.Estado.HuellaEstadoSHA256, "bolsa", m.TipoRecurso
		s.Recurso.Ambitos = map[string]string{"convocatoria_ref": m.ConvocatoriaRef, "expediente_ref": m.ExpedienteRef}
		s.Recurso.Atributos = map[string]string{"material_sha256": hex.EncodeToString(h[:])}
		cruzados := s.Campos
		s.Campos = []string{"estado_reglas_baremo"}
		if recuperacion {
			s.Campos = []string{"estado_reglas_baremo", "recibo"}
		}
		if e.broker.validarPedido(s, actual.Resultado.Contexto, time.Now().UTC().Truncate(time.Microsecond)) != nil {
			t.Fatal("control nominal del negativo de campos inválido")
		}
		s.Campos = cruzados
		if _, err := e.broker.ProveerMaterialGobiernoReglasV3(pedido, actual.Vinculo, s); err != app.ErrGobiernoV3Prohibido {
			t.Fatal("broker admitió campos cruzados")
		}
	}
}

func loopbackEnsayoBaremo(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func rutaPrivadaEnsayoBaremo(ruta string) bool {
	if !filepath.IsAbs(ruta) {
		return false
	}
	for dir := filepath.Dir(ruta); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return false
		}
		if info, err := os.Lstat(dir); err != nil || info.Mode()&os.ModeSymlink != 0 {
			return false
		}
	}
	return true
}

func ficheroPrivadoEnsayoBaremo(t *testing.T, ruta string) []byte {
	t.Helper()
	info, err := os.Lstat(ruta)
	if !rutaPrivadaEnsayoBaremo(ruta) || err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 1<<20 {
		t.Fatal("fichero privado rechazado")
	}
	b, err := os.ReadFile(ruta)
	if err != nil || len(b) > 1<<20 {
		t.Fatal("fichero privado no disponible")
	}
	return b
}

func leerJSONEnsayoBaremo(t *testing.T, ruta string, destino any) {
	t.Helper()
	b := ficheroPrivadoEnsayoBaremo(t, ruta)
	if validarClavesJSONUnicas(b) != nil {
		t.Fatal("JSON privado con claves repetidas")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(new(any)) != io.EOF {
		t.Fatal("JSON privado rechazado")
	}
}
