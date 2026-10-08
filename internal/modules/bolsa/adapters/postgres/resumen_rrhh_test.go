package postgres

import (
	"context"
	"errors"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type contadorConsultasResumenRRHH struct {
	situaciones, politicas, llamamientos, lista atomic.Int64
}

func (c *contadorConsultasResumenRRHH) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if strings.Contains(data.SQL, "leer_resumen_situaciones_bolsas_v1") {
		c.situaciones.Add(1)
	}
	if strings.Contains(data.SQL, "leer_politicas_orden_vigentes_v1") {
		c.politicas.Add(1)
	}
	if strings.Contains(data.SQL, "leer_llamamientos_en_curso_bolsas_v1") {
		c.llamamientos.Add(1)
	}
	if strings.Contains(data.SQL, "FROM vec_bolsa_llamamientos.leer_llamamientos_completos_resumen_v1()") {
		c.lista.Add(1)
	}
	return ctx
}

func (*contadorConsultasResumenRRHH) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {
}

func TestLectorResumenBolsasFallaCerradoSinBase(t *testing.T) {
	var l *LectorResumenBolsasPostgreSQL
	if _, err := l.LeerResumen(context.Background(), time.Now()); !errors.Is(err, ports.ErrResumenBolsasNoDisponible) {
		t.Fatalf("lector nulo: %v", err)
	}
	l = &LectorResumenBolsasPostgreSQL{pool: &pgxpool.Pool{}}
	if _, err := l.LeerResumen(context.Background(), time.Time{}); !errors.Is(err, ports.ErrResumenBolsasNoDisponible) {
		t.Fatalf("corte vacío: %v", err)
	}
	if _, err := NuevoLectorResumenBolsasPostgreSQL(nil); err == nil {
		t.Fatal("pool nulo aceptado")
	}
	if _, err := NuevoLectorResumenBolsasConLlamamientosPostgreSQL(context.Background(), nil); err == nil {
		t.Fatal("pool nulo aceptado en lector B94")
	}
}

// Con Bolsa 000082 instalada (VEC_BOLSA_LOTES_PG_DSN, rol ejecutor), las
// lecturas de conjunto coinciden fila a fila con las individuales.
func TestLectorResumenBolsasCoincideConLecturasIndividualesPostgreSQL(t *testing.T) {
	dsn := os.Getenv("VEC_BOLSA_LOTES_PG_DSN")
	if dsn == "" {
		t.Skip("sin VEC_BOLSA_LOTES_PG_DSN")
	}
	ctx := context.Background()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	consultas := &contadorConsultasResumenRRHH{}
	config.ConnConfig.Tracer = consultas
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	lector, _ := NuevoLectorResumenBolsasPostgreSQL(pool)
	situaciones, _ := NuevoRepositorioSituacionParticipacionPostgreSQL(pool)
	ceses, _ := NuevaConsultaEstadoCesePostgreSQL(pool)
	orden, _ := NuevaConsultaOrdenVigentePostgreSQL(pool)
	corte := time.Now()
	resumen, err := lector.LeerResumen(ctx, corte)
	filas, politicas := resumen.Situaciones, resumen.Politicas
	if err != nil || len(filas) == 0 {
		t.Fatalf("resumen: %d filas, %v", len(filas), err)
	}
	if resumen.Llamamientos != nil {
		t.Fatal("lector legado debe distinguir la lista B94 no montada")
	}
	if consultas.situaciones.Load() != 1 || consultas.politicas.Load() != 1 || consultas.llamamientos.Load() != 1 {
		t.Fatalf("consultas por petición: situaciones=%d políticas=%d recuentos=%d",
			consultas.situaciones.Load(), consultas.politicas.Load(), consultas.llamamientos.Load())
	}
	refs := make([]string, 0, len(filas))
	for _, fila := range filas {
		refs = append(refs, fila.ParticipacionRef)
	}
	individuales, err := situaciones.SituacionesVigentes(ctx, refs)
	if err != nil {
		t.Fatal(err)
	}
	cesesIndividuales, err := ceses.ConsultarEstadosCese(ctx, refs, corte)
	if err != nil {
		t.Fatal(err)
	}
	conCese := 0
	for _, fila := range filas {
		esperada := individuales[fila.ParticipacionRef]
		if fila.Situacion == nil || fila.Situacion.Situacion != esperada.Situacion || !fila.Situacion.Desde.Equal(esperada.Desde) ||
			!mismaFechaDisponiblePostgreSQL(fila.Situacion.FechaDisponible, esperada.FechaDisponible) {
			t.Fatalf("situación distinta para %s", fila.ParticipacionRef)
		}
		estado, presente := cesesIndividuales[fila.ParticipacionRef]
		if presente != (fila.Cese != nil) || (presente && (!estado.FechaEfecto.Equal(fila.Cese.FechaEfecto) ||
			!estado.DisponibleDesde.Equal(fila.Cese.DisponibleDesde) || estado.EnRestriccion != fila.Cese.EnRestriccion || estado.TrabajoCesado != fila.Cese.TrabajoCesado)) {
			t.Fatalf("cese distinto para %s", fila.ParticipacionRef)
		}
		if presente {
			conCese++
		}
	}
	if len(politicas) == 0 {
		t.Fatal("sin políticas")
	}
	bolsas := map[string]struct{}{}
	for _, fila := range filas {
		bolsas[fila.BolsaRef] = struct{}{}
	}
	if len(resumen.LlamamientosEnCurso) != len(bolsas) {
		t.Fatalf("recuentos agrupados=%d bolsas=%d", len(resumen.LlamamientosEnCurso), len(bolsas))
	}
	for bolsa := range bolsas {
		var individual int
		if err := pool.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.contar_llamamientos_en_curso_v1($1)`, bolsa).Scan(&individual); err != nil ||
			resumen.LlamamientosEnCurso[bolsa] != individual {
			t.Fatalf("recuento de %s: conjunto=%d individual=%d error=%v", bolsa, resumen.LlamamientosEnCurso[bolsa], individual, err)
		}
	}
	for bolsa, politica := range politicas {
		vigente, err := orden.ConsultarOrdenVigente(ctx, bolsa)
		if err != nil {
			t.Fatal(err)
		}
		esperada := vigente.Politica
		if esperada.PoliticaRef != politica.PoliticaRef || esperada.Version != politica.Version || esperada.TipoLista != politica.TipoLista ||
			esperada.Reposicion != politica.Reposicion || esperada.Provisional != politica.Provisional || !esperada.VigenteDesde.Equal(politica.VigenteDesde) {
			t.Fatalf("política distinta para %s: %+v frente a %+v", bolsa, politica, esperada)
		}
	}
	t.Logf("%d participaciones (%d con cese), %d políticas", len(filas), conCese, len(politicas))
}

// En un clon con B94 instalada y el rol ejecutor, la lista y B85 han de
// corresponder a la misma instantánea y ocupar una cuarta consulta de conjunto.
func TestLectorResumenBolsasConLlamamientosPostgreSQL(t *testing.T) {
	dsn := os.Getenv("VEC_BOLSA_LOTES_PG_DSN")
	if dsn == "" {
		t.Skip("sin VEC_BOLSA_LOTES_PG_DSN")
	}
	ctx := context.Background()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	consultas := &contadorConsultasResumenRRHH{}
	config.ConnConfig.Tracer = consultas
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	lector, err := NuevoLectorResumenBolsasConLlamamientosPostgreSQL(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := lector.LeerResumen(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if resumen.Llamamientos == nil {
		t.Fatal("lista B94 montada devuelta como nil")
	}
	if consultas.situaciones.Load() != 1 || consultas.politicas.Load() != 1 || consultas.llamamientos.Load() != 1 || consultas.lista.Load() != 1 {
		t.Fatalf("consultas por petición: situaciones=%d políticas=%d recuentos=%d lista=%d",
			consultas.situaciones.Load(), consultas.politicas.Load(), consultas.llamamientos.Load(), consultas.lista.Load())
	}
	porBolsa := map[string]int{}
	for _, fila := range resumen.Llamamientos {
		porBolsa[fila.BolsaRef]++
	}
	for bolsa, total := range resumen.LlamamientosEnCurso {
		if porBolsa[bolsa] != total {
			t.Fatalf("lista y recuento distintos para %s", bolsa)
		}
	}
}

// La medición explícita usa el mismo lector de producción y exige un clon con
// al menos 2000 participaciones; no atribuye al adaptador el tiempo HTTP.
func TestLectorResumenBolsasP95ClonPostgreSQL(t *testing.T) {
	dsn := os.Getenv("VEC_BOLSA_LOTES_PG_DSN")
	if dsn == "" {
		t.Skip("sin VEC_BOLSA_LOTES_PG_DSN")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	lector, err := NuevoLectorResumenBolsasPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	primero, err := lector.LeerResumen(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(primero.Situaciones) < 2000 {
		t.Skipf("clon con %d participaciones; se requieren al menos 2000", len(primero.Situaciones))
	}
	const muestras = 100
	duraciones := make([]time.Duration, 0, muestras)
	for range muestras {
		inicio := time.Now()
		if _, err := lector.LeerResumen(ctx, inicio); err != nil {
			t.Fatal(err)
		}
		duraciones = append(duraciones, time.Since(inicio))
	}
	sort.Slice(duraciones, func(i, j int) bool { return duraciones[i] < duraciones[j] })
	p95 := duraciones[(muestras*95+99)/100-1]
	t.Logf("adaptador: %d participaciones, %d muestras, p95=%s", len(primero.Situaciones), muestras, p95)
	if p95 >= 300*time.Millisecond {
		t.Fatalf("p95 del adaptador supera 300 ms: %s", p95)
	}
}

func TestLectorResumenBolsasConLlamamientosP95ClonPostgreSQL(t *testing.T) {
	dsn := os.Getenv("VEC_BOLSA_LOTES_PG_DSN")
	if dsn == "" {
		t.Skip("sin VEC_BOLSA_LOTES_PG_DSN")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	lector, err := NuevoLectorResumenBolsasConLlamamientosPostgreSQL(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	primero, err := lector.LeerResumen(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(primero.Situaciones) < 2000 {
		t.Skipf("clon con %d participaciones; se requieren al menos 2000", len(primero.Situaciones))
	}
	const muestras = 100
	duraciones := make([]time.Duration, 0, muestras)
	for range muestras {
		inicio := time.Now()
		if _, err := lector.LeerResumen(ctx, inicio); err != nil {
			t.Fatal(err)
		}
		duraciones = append(duraciones, time.Since(inicio))
	}
	sort.Slice(duraciones, func(i, j int) bool { return duraciones[i] < duraciones[j] })
	p95 := duraciones[(muestras*95+99)/100-1]
	t.Logf("adaptador con lista: %d participaciones, %d llamamientos, %d muestras, p95=%s",
		len(primero.Situaciones), len(primero.Llamamientos), muestras, p95)
	if p95 >= 300*time.Millisecond {
		t.Fatalf("p95 del adaptador con lista supera 300 ms: %s", p95)
	}
}
