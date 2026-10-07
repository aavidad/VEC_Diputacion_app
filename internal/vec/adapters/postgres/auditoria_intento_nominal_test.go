package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type filaIntentoAuditoriaPrueba struct {
	acuse ports.AcuseIntentoAuditoria
	err   error
}

func (f filaIntentoAuditoriaPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != 5 {
		return errors.New("destinos invalidos")
	}
	*destinos[0].(*string) = f.acuse.AuditoriaRef
	*destinos[1].(*int64) = f.acuse.Secuencia
	*destinos[2].(*string) = f.acuse.HuellaSHA256
	*destinos[3].(*string) = f.acuse.CorrelacionRef
	*destinos[4].(*time.Time) = f.acuse.RegistradaEn
	return nil
}

type txIntentoAuditoriaPrueba struct {
	pgx.Tx
	eventos     []string
	fila        pgx.Row
	consulta    string
	argumentos  []any
	contextoErr error
	commitErr   error
}

func (t *txIntentoAuditoriaPrueba) Exec(ctx context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	t.eventos = append(t.eventos, "configurar")
	t.contextoErr = ctx.Err()
	return pgconn.CommandTag{}, nil
}
func (t *txIntentoAuditoriaPrueba) QueryRow(ctx context.Context, consulta string, args ...any) pgx.Row {
	t.eventos = append(t.eventos, "consultar")
	t.contextoErr = ctx.Err()
	t.consulta = consulta
	t.argumentos = args
	return t.fila
}
func (t *txIntentoAuditoriaPrueba) Commit(context.Context) error {
	t.eventos = append(t.eventos, "commit")
	return t.commitErr
}
func (t *txIntentoAuditoriaPrueba) Rollback(context.Context) error {
	t.eventos = append(t.eventos, "rollback")
	return nil
}

type poolIntentoAuditoriaPrueba struct {
	tx                  *txIntentoAuditoriaPrueba
	opciones            pgx.TxOptions
	inicios             int
	preflight           bool
	consultaPreflight   string
	argumentosPreflight []any
}

func (p *poolIntentoAuditoriaPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	p.inicios++
	p.opciones = opciones
	p.tx.eventos = append(p.tx.eventos, "begin")
	return p.tx, nil
}
func (p *poolIntentoAuditoriaPrueba) QueryRow(_ context.Context, consulta string, argumentos ...any) pgx.Row {
	p.consultaPreflight = consulta
	p.argumentosPreflight = argumentos
	return filaPreflightIntentoAuditoriaPrueba{permitido: p.preflight}
}

type filaPreflightIntentoAuditoriaPrueba struct{ permitido bool }

func (f filaPreflightIntentoAuditoriaPrueba) Scan(destinos ...any) error {
	if len(destinos) != 1 {
		return errors.New("preflight invalido")
	}
	*destinos[0].(*bool) = f.permitido
	return nil
}

func ordenIntentoAuditoriaPrueba(t *testing.T) ports.OrdenIntentoAuditoria {
	t.Helper()
	ahora := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenIntentoAuditoria(
		"intento_11111111111111111111111111111111", resultado, vinculo,
		domain.DatosIntentoAuditoria{
			Accion: "administracion.perfiles.consultar", ModuloID: "administracion",
			RecursoRef: "persona:11111111111111111111111111111111", FinalidadRef: "administracion_perfiles",
			Resultado: domain.ResultadoIntentoAuditoriaDenegado,
			Motivo: domain.ReferenciaEntradaCatalogo{
				CatalogoID: "motivos_auditoria", CatalogoVersion: 1,
				CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "acceso_denegado",
			},
			Proceso: "vec-server", Canal: "interna_corporativa",
			CorrelacionRef: "correlacion_11111111111111111111111111111111",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return orden
}

func TestIntentoAuditoriaConfirmaYAcusaTrasCancelacionHTTP(t *testing.T) {
	orden := ordenIntentoAuditoriaPrueba(t)
	tx := &txIntentoAuditoriaPrueba{fila: filaIntentoAuditoriaPrueba{acuse: ports.AcuseIntentoAuditoria{
		AuditoriaRef: "aud_v3_intento_11111111111111111111111111111111", Secuencia: 7,
		HuellaSHA256:   strings.Repeat("b", 64),
		CorrelacionRef: "correlacion_11111111111111111111111111111111",
		RegistradaEn:   time.Date(2026, 10, 3, 10, 1, 0, 0, time.UTC),
	}}}
	pool := &poolIntentoAuditoriaPrueba{tx: tx}
	r, err := nuevoRegistradorIntentosAuditoriaPostgreSQL(pool, "vec-server", "interna_corporativa", 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	acuse, err := r.AppendIntentoAuditoria(ctx, orden)
	if err != nil {
		t.Fatal(err)
	}
	if acuse.Secuencia != 7 || tx.contextoErr != nil ||
		pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite ||
		strings.Join(tx.eventos[:4], ",") != "begin,configurar,consultar,commit" {
		t.Fatalf("secuencia/tx/contexto inesperados: acuse=%v eventos=%v error=%v", acuse.Secuencia, tx.eventos, tx.contextoErr)
	}
	if !strings.Contains(tx.consulta, "registrar_intento_nominal_v1") || len(tx.argumentos) != 3 {
		t.Fatal("contrato SQL distinto")
	}
	var ordenSQL map[string]any
	if err := json.Unmarshal(tx.argumentos[2].([]byte), &ordenSQL); err != nil {
		t.Fatal(err)
	}
	if len(ordenSQL) != 16 || ordenSQL["proceso"] != "vec-server" || ordenSQL["canal"] != "interna_corporativa" ||
		ordenSQL["resultado"] != "denegado" || ordenSQL["intento_ref"] != "intento_11111111111111111111111111111111" {
		t.Fatalf("orden minimizada incorrecta: claves=%d proceso=%v canal=%v", len(ordenSQL), ordenSQL["proceso"], ordenSQL["canal"])
	}
}

func TestIntentoAuditoriaSinCommitNoDevuelveAcuse(t *testing.T) {
	orden := ordenIntentoAuditoriaPrueba(t)
	tx := &txIntentoAuditoriaPrueba{
		fila: filaIntentoAuditoriaPrueba{acuse: ports.AcuseIntentoAuditoria{
			AuditoriaRef: "aud_v3_intento_11111111111111111111111111111111", Secuencia: 7,
			HuellaSHA256:   strings.Repeat("b", 64),
			CorrelacionRef: "correlacion_11111111111111111111111111111111",
			RegistradaEn:   time.Date(2026, 10, 3, 10, 1, 0, 0, time.UTC),
		}},
		commitErr: errors.New("postgresql://usuario:secreto@host/auditoria"),
	}
	r, err := nuevoRegistradorIntentosAuditoriaPostgreSQL(&poolIntentoAuditoriaPrueba{tx: tx}, "vec-server", "interna_corporativa", 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	acuse, err := r.AppendIntentoAuditoria(context.Background(), orden)
	if acuse != (ports.AcuseIntentoAuditoria{}) || !errors.Is(err, ports.ErrIntentoAuditoriaNoDisponible) || strings.Contains(err.Error(), "secreto") {
		t.Fatalf("acuse sin commit o error sensible: acuse=%+v err=%v", acuse, err)
	}
}

func TestIntentoAuditoriaRechazaOrigenAjenoAntesDeAbrirTX(t *testing.T) {
	orden := ordenIntentoAuditoriaPrueba(t)
	pool := &poolIntentoAuditoriaPrueba{tx: &txIntentoAuditoriaPrueba{}}
	r, err := nuevoRegistradorIntentosAuditoriaPostgreSQL(pool, "otro-proceso", "interna_corporativa", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.AppendIntentoAuditoria(context.Background(), orden)
	if !errors.Is(err, ports.ErrOrdenIntentoAuditoriaInvalida) || pool.inicios != 0 {
		t.Fatalf("origen ajeno aceptado: err=%v inicios=%d", err, pool.inicios)
	}
}

func TestIntentoAuditoriaPreflightCotejaProcesoCanalYRol(t *testing.T) {
	pool := &poolIntentoAuditoriaPrueba{preflight: true}
	r, err := nuevoRegistradorIntentosAuditoriaPostgreSQL(pool, "vec-server", "interna_corporativa", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.PreflightIntentoAuditoria(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pool.consultaPreflight, "preflight_registrador_intentos_v1") ||
		len(pool.argumentosPreflight) != 2 || pool.argumentosPreflight[0] != "vec-server" ||
		pool.argumentosPreflight[1] != "interna_corporativa" {
		t.Fatalf("preflight no coteja rol/configuracion exactos: consulta=%q argumentos=%v", pool.consultaPreflight, pool.argumentosPreflight)
	}
	pool.preflight = false
	if err := r.PreflightIntentoAuditoria(context.Background()); !errors.Is(err, ports.ErrIntentoAuditoriaNoDisponible) {
		t.Fatalf("preflight falso aceptado: %v", err)
	}
}

// La pareja privada procede de una atestación y un registro de contexto ya
// confirmados en el clon. Ningún dato ni DSN se versiona o imprime.
func TestIntegracionIntentoAuditoriaPostgreSQL(t *testing.T) {
	ruta, dsn := os.Getenv("VEC_INTENTOS_AUDITORIA_FIXTURE"), os.Getenv("VEC_INTENTOS_AUDITORIA_DSN")
	if ruta == "" || dsn == "" {
		t.Skip("requiere fixture histórico y clon PG privados")
	}
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal("fixture privado no disponible")
	}
	var fixture struct {
		RegistroContextoRef string    `json:"registro_contexto_ref"`
		ContextoBase64      string    `json:"contexto_actor_canonico_base64"`
		ContextoSHA256      string    `json:"contexto_actor_huella_sha256"`
		ManifiestoBase64    string    `json:"manifiesto_procedencia_canonico_base64"`
		ManifiestoSHA256    string    `json:"manifiesto_procedencia_huella_sha256"`
		VinculoBase64       string    `json:"vinculo_canonico_base64"`
		AutoridadEfectiva   string    `json:"autoridad_efectiva"`
		ResueltoEn          time.Time `json:"resuelto_en"`
	}
	if json.Unmarshal(contenido, &fixture) != nil {
		t.Fatal("fixture privado invalido")
	}
	contextoBytes, errContexto := base64.StdEncoding.DecodeString(fixture.ContextoBase64)
	manifiestoBytes, errManifiesto := base64.StdEncoding.DecodeString(fixture.ManifiestoBase64)
	vinculoBytes, errVinculo := base64.StdEncoding.DecodeString(fixture.VinculoBase64)
	if errContexto != nil || errManifiesto != nil || errVinculo != nil {
		t.Fatal("fixture privado invalido")
	}
	actor, err := domain.RehidratarContextoActorVinculadoV2(contextoBytes)
	if err != nil {
		t.Fatal("contexto historico invalido")
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: fixture.RegistroContextoRef,
		Contexto:            actor, RepresentacionCanonica: contextoBytes,
		HuellaSHA256:                      fixture.ContextoSHA256,
		ManifiestoProcedenciaCanonico:     manifiestoBytes,
		ManifiestoProcedenciaHuellaSHA256: fixture.ManifiestoSHA256,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorV1(fixture.AutoridadEfectiva),
		ResueltoEnAutoritativo:            fixture.ResueltoEn.UTC().Truncate(time.Microsecond),
	}
	if resultado.Validar() != nil {
		t.Fatal("recibo historico de contexto invalido")
	}
	var datosVinculo domain.DatosVinculoAutenticacionActorV2
	if json.Unmarshal(vinculoBytes, &datosVinculo) != nil || datosVinculo.Validar() != nil {
		t.Fatal("vinculo historico invalido")
	}
	ahora := datosVinculo.SesionRevalidadaEn
	if resultado.ResueltoEnAutoritativo.After(ahora) {
		ahora = resultado.ResueltoEnAutoritativo
	}
	ahora = ahora.Add(time.Microsecond)
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: actor.Instantanea.CuentaRef,
		Metodo:    actor.Principal.AuthMethod, Garantia: actor.Principal.AuthAssurance,
	}
	vinculo, resultadoLigado, err := domain.CrearVinculoAutenticacionActorV2ConResultado(
		context.Background(),
		revalidadorRegistroContextoActorV3PostgreSQLPrueba{datosVinculo.Autenticacion()},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: datosVinculo.AutenticacionRef, SesionRef: datosVinculo.SesionRef,
		},
		resolutorRegistroContextoActorV3PostgreSQLPrueba{resultado},
		domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef},
		relojRegistroContextoActorV3PostgreSQLPrueba{ahora: ahora},
	)
	if err != nil || vinculo.ValidarPara(resultadoLigado) != nil {
		t.Fatal("vinculo historico no enlazado")
	}
	intentoRef := os.Getenv("VEC_INTENTOS_AUDITORIA_REF")
	if intentoRef == "" {
		intentoRef, err = ports.NuevaReferenciaIntentoAuditoria()
		if err != nil {
			t.Fatal(err)
		}
	}
	orden, err := ports.NuevaOrdenIntentoAuditoria(intentoRef, resultadoLigado, vinculo, domain.DatosIntentoAuditoria{
		Accion: "auditoria.intento.verificar", ModuloID: "auditoria",
		RecursoRef: "auditoria:prueba", FinalidadRef: "verificacion_ensayo",
		Resultado: domain.ResultadoIntentoAuditoriaDenegado,
		Motivo: domain.ReferenciaEntradaCatalogo{
			CatalogoID: "motivos_auditoria", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "acceso_denegado",
		},
		Proceso: "vec_server_ensayo", Canal: string(datosVinculo.Superficie),
		CorrelacionRef: "correlacion_11111111111111111111111111111111",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("clon PG no disponible")
	}
	defer pool.Close()
	registrador, err := NuevoRegistradorIntentosAuditoriaPostgreSQL(pool, "vec_server_ensayo", string(datosVinculo.Superficie), 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := registrador.PreflightIntentoAuditoria(ctx); err != nil {
		t.Fatal("preflight registrador denegado")
	}
	acuse, err := registrador.AppendIntentoAuditoria(ctx, orden)
	if err != nil {
		t.Fatalf("append real fallido: %v", err)
	}
	replay, err := registrador.AppendIntentoAuditoria(ctx, orden)
	if err != nil || replay.AuditoriaRef != acuse.AuditoriaRef || replay.Secuencia != acuse.Secuencia ||
		replay.HuellaSHA256 != acuse.HuellaSHA256 || !replay.RegistradaEn.Equal(acuse.RegistradaEn) {
		t.Fatal("replay no recupera acuse confirmado exacto")
	}
}
