package postgres

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	fuente "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguimientoejercicio"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Solo fixture de transporte binario raíz: cada preimagen se coteja con la
// huella calculada por el dominio público. No es un codec de producción.
func canonRaizFixtureV2(t *testing.T, s dom.EstadoPersistidoSeguimiento) []byte {
	t.Helper()
	var b bytes.Buffer
	cadena := func(v string) { _ = binary.Write(&b, binary.BigEndian, uint32(len(v))); b.WriteString(v) }
	entero := func(v uint64) { _ = binary.Write(&b, binary.BigEndian, v) }
	cadena("vec.dipgra.contratacion-temporal.seguimiento.raiz")
	_ = binary.Write(&b, binary.BigEndian, uint16(1))
	cadena("sha-256")
	for _, v := range []string{s.Referencia, s.OrganizacionRef, s.ExpedienteRef, s.RelacionRef, s.Definicion.Referencia} {
		cadena(v)
	}
	entero(s.Definicion.Version)
	cadena(s.Definicion.HuellaSHA256)
	cadena(string(s.EstadoActual))
	entero(uint64(s.PeriodoPrevisto.Desde.UnixMicro()))
	entero(uint64(s.PeriodoPrevisto.Hasta.UnixMicro()))
	entero(uint64(s.CreadoEn.UnixMicro()))
	if shaRaicesV2(b.Bytes()) != s.HuellaRaizSHA256 {
		t.Fatal("fixture canon no corresponde al dominio")
	}
	return b.Bytes()
}
func filaRaizFixtureV2(t *testing.T, p dom.PublicacionDefinicionSeguimiento, s dom.EstadoPersistidoSeguimiento) filaRaizIncorporacionV2 {
	t.Helper()
	d, e := dom.RestaurarDefinicionSeguimiento(p)
	registroV2Exigir(t, e)
	agregado, e := dom.RehidratarSeguimiento(d, s)
	registroV2Exigir(t, e)
	snap, e := PrepararSnapshotSeguimientoPersistido(d, agregado)
	registroV2Exigir(t, e)
	return filaRaizIncorporacionV2{referencia: s.Referencia, org: s.OrganizacionRef, exp: s.ExpedienteRef, rel: s.RelacionRef, versionExp: "8",
		defRef: p.Referencia, defVersion: strconv.FormatUint(p.Version, 10), defHash: p.HuellaSHA256,
		publicacion: serialTransporte(t, p), raiz: canonRaizFixtureV2(t, s), raizHash: s.HuellaRaizSHA256, estado: snap.EstadoJSON, canon: snap.EstadoCanonico}
}
func fixtureRaicesV2(t *testing.T) (ct.OrdenConfirmacionIncorporacionV2, filaRaizIncorporacionV2, time.Time) {
	t.Helper()
	o, _, ahora := fixtureTransporte(t)
	w, _ := wireTransporte(t, o, ahora)
	// Validar bytes de la fuente C antes de reutilizar su publicación.
	b, e := os.ReadFile("../seguimientoejercicio/testdata/definicion-ejercicio.json")
	registroV2Exigir(t, e)
	f, e := fuente.Nueva(b, fuente.Configuracion{HuellaArchivoSHA256: "937088aff1d4d7ab518b565cd7b6c34d214d849780ddb9751d5a0e2a6e9cf1e1", Definicion: w.Historia.Anterior.Definicion})
	registroV2Exigir(t, e)
	_, e = f.Consultar(context.Background(), w.Historia.Anterior.Definicion, ahora)
	registroV2Exigir(t, e)
	return o, filaRaizFixtureV2(t, w.Historia.Publicacion, w.Historia.Anterior), ahora
}

// Dobles pgx explícitos. La orden y el agregado proceden de APIs nominales;
// ninguna lectura simulada acredita la función SQL78 ni procedencia de commit.
type txRaicesV2 struct {
	pgx.Tx
	t          *testing.T
	filas      []filaRaizIncorporacionV2
	triple     [3]string
	pasos      []string
	fallo      string
	hook       func(string)
	confirmado bool
	destruir   bool
	scans      int
}

func (m *txRaicesV2) paso(s string) {
	m.pasos = append(m.pasos, s)
	if m.hook != nil {
		m.hook(s)
	}
}
func (m *txRaicesV2) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	m.paso("begin")
	if o != (pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly}) {
		m.t.Fatal("no RO serializable")
	}
	if m.fallo == "begin" {
		return m, errors.New("detalle privado")
	}
	if m.fallo == "tx_nil" {
		return nil, nil
	}
	return m, nil
}
func (m *txRaicesV2) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	m.paso("settings")
	if q != ajustesRaicesIncorporacionV2 || len(args) != 0 {
		m.t.Fatal("ajustes")
	}
	if m.fallo == "settings" {
		return pgconn.CommandTag{}, errors.New("detalle privado")
	}
	return pgconn.CommandTag{}, nil
}
func (m *txRaicesV2) Query(_ context.Context, q string, args ...any) (pgx.Rows, error) {
	m.paso("query")
	if q != consultaRaicesIncorporacionV2 || !reflect.DeepEqual(args, []any{m.triple[0], m.triple[1], m.triple[2]}) {
		m.t.Fatal("selector o API")
	}
	if m.fallo == "query" {
		return nil, errors.New("detalle privado")
	}
	if m.fallo == "rows_nil" {
		return nil, nil
	}
	return &filasRaicesV2{m: m}, nil
}
func (m *txRaicesV2) Commit(context.Context) error {
	m.paso("commit")
	if m.fallo == "commit" {
		return errors.New("detalle privado")
	}
	m.confirmado = true
	return nil
}
func (m *txRaicesV2) Rollback(ctx context.Context) error {
	d, ok := ctx.Deadline()
	if ctx.Err() != nil || !ok || time.Until(d) > 2*time.Second || m.confirmado {
		m.t.Fatal("rollback no independiente o postcommit")
	}
	m.paso("rollback")
	return nil
}

type filasRaicesV2 struct {
	pgx.Rows
	m *txRaicesV2
	i int
}

func (r *filasRaicesV2) Next() bool { r.m.paso("next"); r.i++; return r.i <= len(r.m.filas) }
func (r *filasRaicesV2) Scan(d ...any) error {
	r.m.paso("scan")
	r.m.scans++
	if r.m.fallo == "scan" {
		return errors.New("detalle privado")
	}
	if len(d) != 13 {
		r.m.t.Fatal("columnas")
	}
	f := r.m.filas[r.i-1]
	ss := []string{f.referencia, f.org, f.exp, f.rel, f.versionExp, f.defRef, f.defVersion, f.defHash}
	for i, v := range ss {
		*d[i].(*string) = v
	}
	*d[8].(*[]byte) = f.publicacion
	*d[9].(*[]byte) = f.raiz
	*d[10].(*string) = f.raizHash
	*d[11].(*[]byte) = f.estado
	*d[12].(*[]byte) = f.canon
	return nil
}
func (r *filasRaicesV2) Err() error {
	if r.m.fallo == "rows_err" {
		return errors.New("detalle privado")
	}
	return nil
}
func (r *filasRaicesV2) Close() {
	if r.m.destruir {
		for _, f := range r.m.filas {
			for _, b := range [][]byte{f.publicacion, f.raiz, f.estado, f.canon} {
				clear(b)
			}
		}
	}
}
func dobleRaicesV2(t *testing.T, f filaRaizIncorporacionV2) *txRaicesV2 {
	return &txRaicesV2{t: t, filas: []filaRaizIncorporacionV2{f.clonar()}, triple: [3]string{f.org, f.exp, f.rel}}
}
func exigirErrorRaizV2(t *testing.T, ref string, e error, m *txRaicesV2) {
	t.Helper()
	if ref != "" || e == nil || strings.Contains(e.Error(), "detalle privado") {
		t.Fatal("fallo no cerrado")
	}
	if m.confirmado {
		return
	}
	if len(m.pasos) > 0 && m.fallo != "tx_nil" && m.pasos[len(m.pasos)-1] != "rollback" {
		t.Fatal("rollback ausente")
	}
}

func TestResolutorRaizIncorporacionV2LimiteDecimalCT72(t *testing.T) {
	// Guardia común a definición y versión observada; no equipara sus valores.
	for _, valor := range []string{"9007199254740991", "9007199254740992", "18446744073709551615"} {
		t.Run("decimal_"+valor, func(t *testing.T) {
			n, e := decimalRaicesV2(valor)
			if valor == "9007199254740991" {
				if e != nil || n != 9007199254740991 {
					t.Fatal("máximo CT72 rechazado")
				}
			} else if e == nil || n != 0 {
				t.Fatal("versión fuera de CT72 aceptada")
			}
		})
	}
	o, f, now := fixtureRaicesV2(t)
	for _, caso := range []string{"observada_maximo", "observada_maximo_mas_uno", "definicion_maximo_mas_uno"} {
		t.Run(caso, func(t *testing.T) {
			m := dobleRaicesV2(t, f)
			switch caso {
			case "observada_maximo":
				m.filas[0].versionExp = "9007199254740991"
			case "observada_maximo_mas_uno":
				m.filas[0].versionExp = "9007199254740992"
			case "definicion_maximo_mas_uno":
				m.filas[0].defVersion = "9007199254740992"
			}
			l, e := nuevoResolverRaizIncorporacionV2(m, relojLecturaHistoriaV2(func() time.Time { return now }))
			registroV2Exigir(t, e)
			ref, e := l.ResolverSeguimientoIncorporacionV2(context.Background(), o)
			if caso == "observada_maximo" {
				if e != nil || ref != f.referencia || !m.confirmado {
					t.Fatal("máximo observado nominal rechazado")
				}
				return
			}
			exigirErrorRaizV2(t, ref, e, m)
			if m.confirmado {
				t.Fatal("commit con versión fuera de contrato")
			}
		})
	}
}

func TestResolutorRaizIncorporacionV2NominalYVersiones(t *testing.T) {
	o, f, now := fixtureRaicesV2(t)
	for _, caso := range []string{"nominal", "version_observada_distinta", "canon_estado_copias", "orden_seguimiento_posterior", "definicion_historica_no_vigente_hoy", "bytes_exactos_limite"} {
		t.Run(caso, func(t *testing.T) {
			actual := o
			if caso == "orden_seguimiento_posterior" {
				d, e := o.Material().Datos()
				registroV2Exigir(t, e)
				d.Confirmacion.VersionSeguimientoEsperada = 1
				actual = ordenCTTransporte(t, d, now)
			}
			m := dobleRaicesV2(t, f)
			if caso == "version_observada_distinta" {
				m.filas[0].versionExp = "7"
			}
			if caso == "definicion_historica_no_vigente_hoy" {
				var p dom.PublicacionDefinicionSeguimiento
				var s dom.EstadoPersistidoSeguimiento
				registroV2Exigir(t, json.Unmarshal(f.publicacion, &p))
				registroV2Exigir(t, json.Unmarshal(f.estado, &s))
				p.Vigencia.Hasta = now.Add(-30 * time.Second)
				d, e := dom.PublicarDefinicionSeguimiento(dom.BorradorDefinicionSeguimiento{Referencia: p.Referencia, Version: p.Version, PublicadoEn: p.PublicadoEn, Vigencia: p.Vigencia, EstadoInicial: p.EstadoInicial, ProhibeCiclosSilenciosos: p.ProhibeCiclosSilenciosos, Estados: p.Estados, Motivos: p.Motivos, Transiciones: p.Transiciones})
				registroV2Exigir(t, e)
				a, e := dom.NuevoSeguimiento(d, dom.AltaSeguimiento{Referencia: s.Referencia, OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, RelacionRef: s.RelacionRef, PeriodoPrevisto: s.PeriodoPrevisto, CreadoEn: s.CreadoEn})
				registroV2Exigir(t, e)
				if d.VigenteEn(now) {
					t.Fatal("fixture no histórica")
				}
				m.filas[0] = filaRaizFixtureV2(t, d.Publicacion(), a.Estado())
			}
			if caso == "bytes_exactos_limite" {
				relleno := MaximoBytesRaicesIncorporacionV2 - m.filas[0].bytes()
				m.filas[0].publicacion = append(bytes.Repeat([]byte(" "), relleno), m.filas[0].publicacion...)
				if m.filas[0].bytes() != MaximoBytesRaicesIncorporacionV2 || !json.Valid(m.filas[0].publicacion) {
					t.Fatal("fixture cota inválida")
				}
			}
			if caso == "canon_estado_copias" {
				m.destruir = true
			}
			l, e := nuevoResolverRaizIncorporacionV2(m, relojLecturaHistoriaV2(func() time.Time { return now }))
			registroV2Exigir(t, e)
			ref, e := l.ResolverSeguimientoIncorporacionV2(context.Background(), actual)
			if e != nil || ref != f.referencia || !m.confirmado || m.scans != 1 {
				t.Fatal("nominal")
			}
			if !reflect.DeepEqual(m.pasos, []string{"begin", "settings", "query", "next", "scan", "next", "commit"}) {
				t.Fatal("secuencia")
			}
			// Una segunda preparación independiente no usa una caché del primero.
			m2 := dobleRaicesV2(t, f)
			l2, e := nuevoResolverRaizIncorporacionV2(m2, relojLecturaHistoriaV2(func() time.Time { return now }))
			registroV2Exigir(t, e)
			ref2, e := l2.ResolverSeguimientoIncorporacionV2(context.Background(), actual)
			if e != nil || ref2 != ref || m2.scans != 1 {
				t.Fatal("segunda lectura independiente")
			}
		})
	}
}

func TestResolutorRaizIncorporacionV2Adversarios(t *testing.T) {
	o, f, now := fixtureRaicesV2(t)
	for _, caso := range []string{"cero", "dos", "dieciseis", "diecisiete", "begin", "tx_nil", "settings", "query", "rows_nil", "scan", "rows_err", "commit", "org", "exp", "rel", "ref", "defref", "defversion", "defhash", "raiz", "raiz_hash", "canon", "json", "version_cero", "version_alias", "estado_v1", "publicacion_alias", "publicacion_ausente", "json_duplicado", "bytes_fila", "bytes_agregado"} {
		t.Run(caso, func(t *testing.T) {
			m := dobleRaicesV2(t, f)
			m.fallo = caso
			switch caso {
			case "cero":
				m.filas = nil
			case "dos", "dieciseis", "diecisiete":
				n := 2
				if caso == "dieciseis" {
					n = 16
				}
				if caso == "diecisiete" {
					n = 17
				}
				m.filas = nil
				for i := 0; i < n; i++ {
					var s dom.EstadoPersistidoSeguimiento
					var p dom.PublicacionDefinicionSeguimiento
					registroV2Exigir(t, json.Unmarshal(f.estado, &s))
					registroV2Exigir(t, json.Unmarshal(f.publicacion, &p))
					d, e := dom.RestaurarDefinicionSeguimiento(p)
					registroV2Exigir(t, e)
					a, e := dom.NuevoSeguimiento(d, dom.AltaSeguimiento{Referencia: registroV2Ref("raiz:" + strconv.Itoa(i)), OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, RelacionRef: s.RelacionRef, PeriodoPrevisto: s.PeriodoPrevisto, CreadoEn: s.CreadoEn})
					registroV2Exigir(t, e)
					m.filas = append(m.filas, filaRaizFixtureV2(t, p, a.Estado()))
				}
			case "org":
				m.filas[0].org = registroV2Ref("otra")
			case "exp":
				m.filas[0].exp = registroV2Ref("otro")
			case "rel":
				m.filas[0].rel = registroV2Ref("otra")
			case "ref":
				m.filas[0].referencia = registroV2Ref("otra")
			case "defref":
				m.filas[0].defRef = registroV2Ref("otra")
			case "defversion":
				m.filas[0].defVersion = "2"
			case "defhash":
				m.filas[0].defHash = strings.Repeat("a", 64)
			case "raiz":
				m.filas[0].raiz[0] ^= 1
				m.filas[0].raizHash = shaRaicesV2(m.filas[0].raiz)
			case "raiz_hash":
				m.filas[0].raizHash = strings.Repeat("a", 64)
			case "canon":
				m.filas[0].canon[0] ^= 1
			case "json":
				m.filas[0].estado = []byte("null")
			case "version_cero":
				m.filas[0].versionExp = "0"
			case "version_alias":
				m.filas[0].versionExp = "08"
			case "estado_v1":
				w, _ := wireTransporte(t, o, now)
				m.filas[0].estado = serialTransporte(t, w.Historia.Posterior)
			case "publicacion_alias":
				m.filas[0].publicacion = bytes.Replace(m.filas[0].publicacion, []byte(`"version":`), []byte(`"VERSION":`), 1)
			case "publicacion_ausente":
				var p map[string]any
				registroV2Exigir(t, json.Unmarshal(f.publicacion, &p))
				delete(p, "version")
				m.filas[0].publicacion = serialTransporte(t, p)
			case "json_duplicado":
				m.filas[0].publicacion = append([]byte(`{"version":1,`), f.publicacion[1:]...)
			case "bytes_fila":
				m.filas[0].publicacion = append(bytes.Repeat([]byte(" "), MaximoBytesRaicesIncorporacionV2), f.publicacion...)
			case "bytes_agregado":
				// Cada candidato es JSON y estado válidos; solo la SUMA cruza 32MiB.
				padded := f.clonar()
				padded.publicacion = append(bytes.Repeat([]byte(" "), 17<<20), padded.publicacion...)
				m.filas = []filaRaizIncorporacionV2{padded, padded.clonar()}
			}
			l, e := nuevoResolverRaizIncorporacionV2(m, relojLecturaHistoriaV2(func() time.Time { return now }))
			registroV2Exigir(t, e)
			ref, e := l.ResolverSeguimientoIncorporacionV2(context.Background(), o)
			exigirErrorRaizV2(t, ref, e, m)
			if m.confirmado {
				t.Fatal("commit indebido")
			}
			if caso == "diecisiete" && m.scans != 16 {
				t.Fatal("truncación/cota")
			}
			if caso == "bytes_agregado" && m.scans != 2 {
				t.Fatal("no ejercita suma")
			}
		})
	}
}
func TestResolutorRaizIncorporacionV2CancelacionYReloj(t *testing.T) {
	o, f, now := fixtureRaicesV2(t)
	for _, caso := range []string{"antes", "reloj_inicial", "begin", "settings", "query", "next", "scan", "commit", "reloj_final", "retroceso", "caducada", "orden_cero", "ctx_nil"} {
		t.Run(caso, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			m := dobleRaicesV2(t, f)
			actual := o
			m.hook = func(p string) {
				if p == caso {
					cancel()
				}
			}
			n := 0
			reloj := relojLecturaHistoriaV2(func() time.Time {
				n++
				if caso == "reloj_inicial" || caso == "reloj_final" && m.confirmado {
					cancel()
				}
				if caso == "retroceso" && n > 1 {
					return now.Add(-time.Microsecond)
				}
				if caso == "caducada" {
					return now.Add(24 * time.Hour)
				}
				return now
			})
			if caso == "antes" {
				cancel()
			}
			if caso == "orden_cero" {
				actual = ct.OrdenConfirmacionIncorporacionV2{}
			}
			l, e := nuevoResolverRaizIncorporacionV2(m, reloj)
			registroV2Exigir(t, e)
			var c context.Context = ctx
			if caso == "ctx_nil" {
				c = nil
			}
			ref, e := l.ResolverSeguimientoIncorporacionV2(c, actual)
			exigirErrorRaizV2(t, ref, e, m)
			if ctx.Err() != nil && !errors.Is(e, context.Canceled) {
				t.Fatal("cancelación no prioritaria")
			}
			if caso == "commit" || caso == "reloj_final" {
				if !m.confirmado || m.pasos[len(m.pasos)-1] != "commit" {
					t.Fatal("rollback postcommit")
				}
			}
		})
	}
	var p *txRaicesV2
	var r *relojTransporteV2
	if _, e := nuevoResolverRaizIncorporacionV2(p, &relojTransporteV2{t: now}); e == nil {
		t.Fatal("pool nil")
	}
	if _, e := nuevoResolverRaizIncorporacionV2(dobleRaicesV2(t, f), r); e == nil {
		t.Fatal("reloj nil")
	}
}

// Regresión de la publicación pública usada por PostgreSQL CT78: el extremo
// abierto no es un instante de autenticación y no debe rechazarse como tal.
func TestRaizVigenciaAbiertaOriginal(t *testing.T) {
	_, original, _ := fixtureRaicesV2(t)
	var pub dom.PublicacionDefinicionSeguimiento
	var estado dom.EstadoPersistidoSeguimiento
	registroV2Exigir(t, json.Unmarshal(original.publicacion, &pub))
	registroV2Exigir(t, json.Unmarshal(original.estado, &estado))
	// Fixture nominal local: ninguna ruta absoluta ni dependencia del ensayo PG.
	pub.Vigencia.Hasta = time.Time{}
	def, e := dom.PublicarDefinicionSeguimiento(dom.BorradorDefinicionSeguimiento{Referencia: pub.Referencia, Version: pub.Version, PublicadoEn: pub.PublicadoEn,
		Vigencia: pub.Vigencia, EstadoInicial: pub.EstadoInicial, ProhibeCiclosSilenciosos: pub.ProhibeCiclosSilenciosos, Estados: pub.Estados, Motivos: pub.Motivos, Transiciones: pub.Transiciones})
	registroV2Exigir(t, e)
	raiz, e := dom.NuevoSeguimiento(def, dom.AltaSeguimiento{Referencia: estado.Referencia, OrganizacionRef: estado.OrganizacionRef, ExpedienteRef: estado.ExpedienteRef,
		RelacionRef: estado.RelacionRef, PeriodoPrevisto: estado.PeriodoPrevisto, CreadoEn: estado.CreadoEn})
	registroV2Exigir(t, e)
	pub = def.Publicacion()
	f := filaRaizFixtureV2(t, pub, raiz.Estado())
	if e = validarFilaRaicesV2(context.Background(), f, [3]string{f.org, f.exp, f.rel}); e != nil {
		t.Fatal("vigencia_abierta_rechazada")
	}
	a, e := arbolJSONRegistroV2(f.publicacion)
	if e != nil {
		t.Fatal("arbol")
	}
	forma, e := normalizarPublicacionRaicesV2(a)
	if e != nil {
		t.Fatal("forma")
	}
	serial, e := json.Marshal(forma)
	if e != nil {
		t.Fatal("serial")
	}
	var recuperada dom.PublicacionDefinicionSeguimiento
	if json.Unmarshal(serial, &recuperada) != nil || !reflect.DeepEqual(recuperada, pub) {
		t.Fatal("publicacion_original_alterada")
	}
	for _, c := range []struct{ nombre, campo, valor string }{
		{"publicado_cero", "publicado_en", "0001-01-01T00:00:00Z"},
		{"desde_cero", "desde", "0001-01-01T00:00:00Z"},
		{"hasta_alias_cero", "hasta", "0001-01-01T00:00:00.000000Z"},
		{"hasta_offset", "hasta", "2026-09-09T00:00:00+00:00"},
		{"hasta_nanos", "hasta", "2026-09-09T00:00:00.000000001Z"},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			ar, _ := arbolJSONRegistroV2(f.publicacion)
			m := ar.(map[string]any)
			if c.campo == "publicado_en" {
				m[c.campo] = c.valor
			} else {
				m["vigencia"].(map[string]any)[c.campo] = c.valor
			}
			if _, err := normalizarPublicacionRaicesV2(ar); err == nil {
				t.Fatal("fecha_invalida_aceptada")
			}
		})
	}
}
