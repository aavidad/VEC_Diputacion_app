package postgres

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type resultadoRegistroContextoActorV3PostgreSQLPrueba struct {
	concedida    *bool
	codigo       *string
	huella       *string
	registrada   *time.Time
	errorEscaneo error
}

type filasRegistroContextoActorV3PostgreSQLPrueba struct {
	pgx.Rows
	resultados  []resultadoRegistroContextoActorV3PostgreSQLPrueba
	indice      int
	actual      int
	err         error
	cerradas    bool
	eventos     *[]string
	alTerminar  func()
	terminacion bool
}

func (f *filasRegistroContextoActorV3PostgreSQLPrueba) Next() bool {
	*f.eventos = append(*f.eventos, "next")
	if f.indice >= len(f.resultados) {
		if !f.terminacion && f.alTerminar != nil {
			f.terminacion = true
			f.alTerminar()
		}
		return false
	}
	f.actual = f.indice
	f.indice++
	return true
}

func (f *filasRegistroContextoActorV3PostgreSQLPrueba) Scan(destinos ...any) error {
	*f.eventos = append(*f.eventos, "scan")
	resultado := f.resultados[f.actual]
	if resultado.errorEscaneo != nil {
		return resultado.errorEscaneo
	}
	if len(destinos) != 4 || resultado.concedida == nil || resultado.codigo == nil ||
		resultado.huella == nil || resultado.registrada == nil {
		return errors.New("fila nula o contrato de escaneo invalido")
	}
	concedida, okConcedida := destinos[0].(*bool)
	codigo, okCodigo := destinos[1].(*string)
	huella, okHuella := destinos[2].(*string)
	registrada, okRegistrada := destinos[3].(*time.Time)
	if !okConcedida || !okCodigo || !okHuella || !okRegistrada {
		return errors.New("destinos de escaneo invalidos")
	}
	*concedida = *resultado.concedida
	*codigo = *resultado.codigo
	*huella = *resultado.huella
	*registrada = *resultado.registrada
	return nil
}

func (f *filasRegistroContextoActorV3PostgreSQLPrueba) Err() error { return f.err }

func (f *filasRegistroContextoActorV3PostgreSQLPrueba) Close() {
	*f.eventos = append(*f.eventos, "close")
	f.cerradas = true
}

type transaccionRegistroContextoActorV3PostgreSQLPrueba struct {
	pgx.Tx
	eventos           []string
	filas             *filasRegistroContextoActorV3PostgreSQLPrueba
	errorConfigurar   error
	errorConsulta     error
	errorCommit       error
	alConsultar       func()
	alCommit          func()
	consulta          string
	argumentos        []any
	argumentosVivos   []any
	commitInvocado    bool
	rollbackInvocado  bool
	commitConsiderado bool
	configuracion     string
	argumentosConfig  []any
}

func (t *transaccionRegistroContextoActorV3PostgreSQLPrueba) Exec(
	_ context.Context,
	consulta string,
	argumentos ...any,
) (pgconn.CommandTag, error) {
	t.eventos = append(t.eventos, "configurar")
	t.configuracion = consulta
	t.argumentosConfig = append([]any(nil), argumentos...)
	return pgconn.CommandTag{}, t.errorConfigurar
}

func (t *transaccionRegistroContextoActorV3PostgreSQLPrueba) Query(
	_ context.Context,
	consulta string,
	argumentos ...any,
) (pgx.Rows, error) {
	t.eventos = append(t.eventos, "consultar")
	t.consulta = consulta
	t.argumentosVivos = argumentos
	t.argumentos = copiarArgumentosRegistroContextoActorV3Prueba(argumentos)
	if t.alConsultar != nil {
		t.alConsultar()
	}
	if t.filas != nil {
		t.filas.eventos = &t.eventos
	}
	return t.filas, t.errorConsulta
}

func (t *transaccionRegistroContextoActorV3PostgreSQLPrueba) Commit(context.Context) error {
	t.eventos = append(t.eventos, "commit")
	t.commitInvocado = true
	if t.alCommit != nil {
		t.alCommit()
	}
	if t.errorCommit == nil {
		t.commitConsiderado = true
	}
	return t.errorCommit
}

func (t *transaccionRegistroContextoActorV3PostgreSQLPrueba) Rollback(context.Context) error {
	t.eventos = append(t.eventos, "rollback")
	t.rollbackInvocado = true
	return nil
}

type iniciadorRegistroContextoActorV3PostgreSQLPrueba struct {
	tx           *transaccionRegistroContextoActorV3PostgreSQLPrueba
	err          error
	opciones     pgx.TxOptions
	invocaciones int
}

func (i *iniciadorRegistroContextoActorV3PostgreSQLPrueba) BeginTx(
	_ context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.invocaciones++
	i.opciones = opciones
	if i.tx != nil {
		i.tx.eventos = append(i.tx.eventos, "begin")
	}
	return i.tx, i.err
}

func copiarArgumentosRegistroContextoActorV3Prueba(argumentos []any) []any {
	copia := make([]any, len(argumentos))
	for indice, argumento := range argumentos {
		if bytes, ok := argumento.([]byte); ok {
			copia[indice] = append([]byte(nil), bytes...)
			continue
		}
		copia[indice] = argumento
	}
	return copia
}

func punteroRegistroContextoActorV3Prueba[T any](valor T) *T { return &valor }

func TestRegistroConcesionContextoActorV3PostgreSQLUsaContratoExactoYCommit(t *testing.T) {
	escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, true)
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		escenario.solicitud, escenario.decision, escenario.motivo, escenario.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	huella, _ := domain.HuellaSHA256DecisionAutorizacionV3(escenario.decision)
	registradaUTC := escenario.ahora.Add(time.Microsecond)
	registrada := registradaUTC.In(time.FixedZone("Europe/Madrid-prueba", 2*60*60))
	tx := &transaccionRegistroContextoActorV3PostgreSQLPrueba{}
	tx.filas = &filasRegistroContextoActorV3PostgreSQLPrueba{resultados: []resultadoRegistroContextoActorV3PostgreSQLPrueba{{
		concedida:  punteroRegistroContextoActorV3Prueba(true),
		codigo:     punteroRegistroContextoActorV3Prueba("concedida"),
		huella:     punteroRegistroContextoActorV3Prueba(huella),
		registrada: &registrada,
	}}}
	iniciador := &iniciadorRegistroContextoActorV3PostgreSQLPrueba{tx: tx}
	almacen, err := nuevoAlmacenAutorizacion(iniciador)
	if err != nil {
		t.Fatal(err)
	}

	obtenida, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
		context.Background(), orden,
	)
	if err != nil || !obtenida.Equal(registradaUTC) || obtenida.Location() != time.UTC {
		t.Fatalf("registro valido: instante=%v error=%v", obtenida, err)
	}
	if iniciador.opciones.IsoLevel != pgx.Serializable ||
		iniciador.opciones.AccessMode != pgx.ReadWrite {
		t.Fatalf("opciones no cerradas: %#v", iniciador.opciones)
	}
	for _, ajuste := range []string{
		"set_config('search_path', 'pg_catalog', true)",
		"set_config('row_security', 'on', true)",
		"set_config('timezone', 'UTC', true)",
		"set_config('lock_timeout', '2s', true)",
		"set_config('statement_timeout', '8s', true)",
		"set_config('idle_in_transaction_session_timeout', '10s', true)",
	} {
		if !strings.Contains(tx.configuracion, ajuste) {
			t.Fatalf("configuracion transaccional incompleta: falta %q en %s", ajuste, tx.configuracion)
		}
	}
	if len(tx.argumentosConfig) != 0 {
		t.Fatalf("configuracion transaccional recibio argumentos: %#v", tx.argumentosConfig)
	}
	if tx.consulta != consultaRegistrarDecisionContextoActorV3 {
		t.Fatalf("consulta exterior inesperada: %s", tx.consulta)
	}
	decisionCanonica, _ := domain.RepresentacionCanonicaDecisionAutorizacionV3(escenario.decision)
	motivoCanonico, _ := domain.RepresentacionCanonicaMotivoAutorizacionV2(escenario.motivo)
	if len(tx.argumentos) != 4 || !bytes.Equal(tx.argumentos[0].([]byte), decisionCanonica) ||
		!bytes.Equal(tx.argumentos[1].([]byte), motivoCanonico) ||
		tx.argumentos[2] != strconv.FormatUint(^uint64(0), 10) ||
		tx.argumentos[3] != strconv.FormatUint(^uint64(0)-1, 10) {
		t.Fatalf("argumentos divergentes: %#v", tx.argumentos)
	}
	for _, indice := range []int{0, 1} {
		for _, valor := range tx.argumentosVivos[indice].([]byte) {
			if valor != 0 {
				t.Fatalf("buffer canonico %d no borrado", indice)
			}
		}
	}
	esperados := []string{"begin", "configurar", "consultar", "next", "scan", "next", "close", "commit", "rollback"}
	if strings.Join(tx.eventos, ",") != strings.Join(esperados, ",") ||
		!tx.commitInvocado || !tx.commitConsiderado || !tx.rollbackInvocado || !tx.filas.cerradas {
		t.Fatalf("orden/transaccion incorrectos: eventos=%v commit=%t rollback=%t close=%t",
			tx.eventos, tx.commitConsiderado, tx.rollbackInvocado, tx.filas.cerradas)
	}
}

func TestRegistroDenegacionContextoActorV3PostgreSQLUsaMismaFronteraSinConfirmar(t *testing.T) {
	escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, false)
	orden, err := ports.NuevaOrdenRegistroDenegacionAutorizacionLigadaV3(
		escenario.solicitud, escenario.decision, escenario.motivo, escenario.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, codigo, _ := escenario.decision.Resultado()
	huella, _ := domain.HuellaSHA256DecisionAutorizacionV3(escenario.decision)
	registrada := escenario.ahora.Add(time.Microsecond)
	tx := nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(false, codigo, huella, registrada)
	almacen, _ := nuevoAlmacenAutorizacion(&iniciadorRegistroContextoActorV3PostgreSQLPrueba{tx: tx})
	if err := almacen.RegistrarDenegacionAutorizacionLigadaV3(context.Background(), orden); err != nil {
		t.Fatalf("denegacion durable valida: %v", err)
	}
	if !tx.commitConsiderado || !strings.Contains(tx.consulta, "registrar_decision_contexto_actor_v3") {
		t.Fatalf("denegacion no confirmada por frontera unica: eventos=%v", tx.eventos)
	}
}

func TestRegistroContextoActorV3PostgreSQLExigeUnaFilaCompletaYConcordante(t *testing.T) {
	escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, true)
	orden, _ := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		escenario.solicitud, escenario.decision, escenario.motivo, escenario.resultado,
	)
	huella, _ := domain.HuellaSHA256DecisionAutorizacionV3(escenario.decision)
	valida := escenario.ahora.Add(time.Microsecond)
	filaValida := resultadoRegistroContextoActorV3PostgreSQLPrueba{
		concedida: punteroRegistroContextoActorV3Prueba(true),
		codigo:    punteroRegistroContextoActorV3Prueba("concedida"),
		huella:    &huella, registrada: &valida,
	}
	antes := escenario.ahora.Add(-time.Microsecond)
	limite := escenario.ahora.Add(90 * time.Second)
	submicrosegundo := valida.Add(time.Nanosecond)
	casos := []struct {
		nombre   string
		filas    []resultadoRegistroContextoActorV3PostgreSQLPrueba
		errFilas error
		esperado error
	}{
		{"cero filas", nil, nil, ports.ErrInstantaneaAutorizacionObsoleta},
		{"multiples filas", []resultadoRegistroContextoActorV3PostgreSQLPrueba{filaValida, filaValida}, nil, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible},
		{"NULL", []resultadoRegistroContextoActorV3PostgreSQLPrueba{{concedida: filaValida.concedida}}, nil, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible},
		{"concesion distinta", []resultadoRegistroContextoActorV3PostgreSQLPrueba{{concedida: punteroRegistroContextoActorV3Prueba(false), codigo: filaValida.codigo, huella: filaValida.huella, registrada: filaValida.registrada}}, nil, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible},
		{"codigo distinto", []resultadoRegistroContextoActorV3PostgreSQLPrueba{{concedida: filaValida.concedida, codigo: punteroRegistroContextoActorV3Prueba("otro"), huella: filaValida.huella, registrada: filaValida.registrada}}, nil, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible},
		{"huella distinta", []resultadoRegistroContextoActorV3PostgreSQLPrueba{{concedida: filaValida.concedida, codigo: filaValida.codigo, huella: punteroRegistroContextoActorV3Prueba(strings.Repeat("f", 64)), registrada: filaValida.registrada}}, nil, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible},
		{"instante anterior", []resultadoRegistroContextoActorV3PostgreSQLPrueba{{concedida: filaValida.concedida, codigo: filaValida.codigo, huella: filaValida.huella, registrada: &antes}}, nil, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible},
		{"instante limite", []resultadoRegistroContextoActorV3PostgreSQLPrueba{{concedida: filaValida.concedida, codigo: filaValida.codigo, huella: filaValida.huella, registrada: &limite}}, nil, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible},
		{"instante no canonico", []resultadoRegistroContextoActorV3PostgreSQLPrueba{{concedida: filaValida.concedida, codigo: filaValida.codigo, huella: filaValida.huella, registrada: &submicrosegundo}}, nil, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible},
		{"error de filas", []resultadoRegistroContextoActorV3PostgreSQLPrueba{filaValida}, errors.New("secreto de cursor"), ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &transaccionRegistroContextoActorV3PostgreSQLPrueba{}
			tx.filas = &filasRegistroContextoActorV3PostgreSQLPrueba{resultados: caso.filas, err: caso.errFilas}
			almacen, _ := nuevoAlmacenAutorizacion(&iniciadorRegistroContextoActorV3PostgreSQLPrueba{tx: tx})
			_, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
				context.Background(), orden,
			)
			if !errors.Is(err, caso.esperado) || tx.commitInvocado || !tx.rollbackInvocado {
				t.Fatalf("resultado no cerrado: error=%v eventos=%v", err, tx.eventos)
			}
			if err != nil && strings.Contains(err.Error(), "secreto") {
				t.Fatalf("error no saneado: %q", err)
			}
		})
	}
}

func TestRegistroContextoActorV3PostgreSQLRollbackCancelacionYCommitAmbiguo(t *testing.T) {
	escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, true)
	orden, _ := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		escenario.solicitud, escenario.decision, escenario.motivo, escenario.resultado,
	)
	huella, _ := domain.HuellaSHA256DecisionAutorizacionV3(escenario.decision)
	registrada := escenario.ahora.Add(time.Microsecond)

	t.Run("cancelacion previa", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		iniciador := &iniciadorRegistroContextoActorV3PostgreSQLPrueba{}
		almacen, _ := nuevoAlmacenAutorizacion(iniciador)
		_, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
		if !errors.Is(err, context.Canceled) || iniciador.invocaciones != 0 {
			t.Fatalf("cancelacion previa alcanzo PostgreSQL: %v, %d", err, iniciador.invocaciones)
		}
	})

	t.Run("cancelacion durante consulta", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		tx := nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(true, "concedida", huella, registrada)
		tx.alConsultar = cancelar
		tx.errorConsulta = context.Canceled
		almacen, _ := nuevoAlmacenAutorizacion(&iniciadorRegistroContextoActorV3PostgreSQLPrueba{tx: tx})
		_, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
		if !errors.Is(err, context.Canceled) || tx.commitInvocado || !tx.rollbackInvocado {
			t.Fatalf("cancelacion durante consulta: %v eventos=%v", err, tx.eventos)
		}
	})

	t.Run("cancelacion antes de commit", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		tx := nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(true, "concedida", huella, registrada)
		tx.filas.alTerminar = cancelar
		almacen, _ := nuevoAlmacenAutorizacion(&iniciadorRegistroContextoActorV3PostgreSQLPrueba{tx: tx})
		_, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
		if !errors.Is(err, context.Canceled) || tx.commitInvocado || !tx.rollbackInvocado {
			t.Fatalf("cancelacion anterior a commit: %v eventos=%v", err, tx.eventos)
		}
	})

	t.Run("cancelacion tardia con commit exitoso", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		tx := nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(true, "concedida", huella, registrada)
		tx.alCommit = cancelar
		almacen, _ := nuevoAlmacenAutorizacion(&iniciadorRegistroContextoActorV3PostgreSQLPrueba{tx: tx})
		obtenida, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
		if err != nil || !obtenida.Equal(registrada) || !tx.commitConsiderado {
			t.Fatalf("commit durable borrado por cancelacion tardia: %v, %v", obtenida, err)
		}
	})

	t.Run("respuesta de commit ambigua", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		tx := nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(true, "concedida", huella, registrada)
		tx.alCommit = func() { tx.commitConsiderado = true; cancelar() }
		tx.errorCommit = errors.New("postgresql://usuario:secreto@servidor respuesta perdida")
		almacen, _ := nuevoAlmacenAutorizacion(&iniciadorRegistroContextoActorV3PostgreSQLPrueba{tx: tx})
		_, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
		if !errors.Is(err, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
			errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "secreto") || !tx.rollbackInvocado {
			t.Fatalf("ambiguedad de commit mal clasificada: %v eventos=%v", err, tx.eventos)
		}
	})
}

func TestRegistroContextoActorV3PostgreSQLSaneaFallosDeInfraestructura(t *testing.T) {
	escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, true)
	orden, _ := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		escenario.solicitud, escenario.decision, escenario.motivo, escenario.resultado,
	)
	secreto := errors.New("postgresql://usuario:secreto@servidor dato_privado")
	casos := []struct {
		nombre   string
		preparar func(*iniciadorRegistroContextoActorV3PostgreSQLPrueba, *transaccionRegistroContextoActorV3PostgreSQLPrueba)
	}{
		{"begin", func(i *iniciadorRegistroContextoActorV3PostgreSQLPrueba, _ *transaccionRegistroContextoActorV3PostgreSQLPrueba) {
			i.err = secreto
		}},
		{"configurar", func(_ *iniciadorRegistroContextoActorV3PostgreSQLPrueba, tx *transaccionRegistroContextoActorV3PostgreSQLPrueba) {
			tx.errorConfigurar = secreto
		}},
		{"query", func(_ *iniciadorRegistroContextoActorV3PostgreSQLPrueba, tx *transaccionRegistroContextoActorV3PostgreSQLPrueba) {
			tx.errorConsulta = secreto
		}},
		{"filas nulas", func(_ *iniciadorRegistroContextoActorV3PostgreSQLPrueba, tx *transaccionRegistroContextoActorV3PostgreSQLPrueba) {
			tx.filas = nil
		}},
		{"scan", func(_ *iniciadorRegistroContextoActorV3PostgreSQLPrueba, tx *transaccionRegistroContextoActorV3PostgreSQLPrueba) {
			tx.filas.resultados[0].errorEscaneo = secreto
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			huella, _ := domain.HuellaSHA256DecisionAutorizacionV3(escenario.decision)
			registrada := escenario.ahora.Add(time.Microsecond)
			tx := nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(true, "concedida", huella, registrada)
			iniciador := &iniciadorRegistroContextoActorV3PostgreSQLPrueba{tx: tx}
			caso.preparar(iniciador, tx)
			almacen, _ := nuevoAlmacenAutorizacion(iniciador)
			_, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Background(), orden)
			if !errors.Is(err, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
				strings.Contains(err.Error(), "secreto") || tx.commitInvocado {
				t.Fatalf("fallo no cerrado: %v eventos=%v", err, tx.eventos)
			}
		})
	}
}

func TestRegistroContextoActorV3PostgreSQLCierraFilasDevueltasJuntoAError(t *testing.T) {
	escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, true)
	orden, _ := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		escenario.solicitud, escenario.decision, escenario.motivo, escenario.resultado,
	)
	tx := &transaccionRegistroContextoActorV3PostgreSQLPrueba{
		filas:         &filasRegistroContextoActorV3PostgreSQLPrueba{},
		errorConsulta: errors.New("postgresql://usuario:secreto@servidor respuesta parcial"),
	}
	almacen, _ := nuevoAlmacenAutorizacion(&iniciadorRegistroContextoActorV3PostgreSQLPrueba{tx: tx})
	_, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
		context.Background(), orden,
	)
	if !errors.Is(err, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
		!tx.filas.cerradas || tx.commitInvocado || !tx.rollbackInvocado ||
		strings.Contains(err.Error(), "secreto") {
		t.Fatalf("filas parciales no cerradas: error=%v eventos=%v", err, tx.eventos)
	}
}

func TestRegistroContextoActorV3PostgreSQLFallaCerradoSinDependencias(t *testing.T) {
	var almacen *AlmacenAutorizacion
	if _, err := almacen.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
		context.Background(), ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3{},
	); !errors.Is(err, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) {
		t.Fatalf("concesiones sin almacen: %v", err)
	}
	if err := almacen.RegistrarDenegacionAutorizacionLigadaV3(
		context.Background(), ports.OrdenRegistroDenegacionAutorizacionLigadaV3{},
	); !errors.Is(err, ports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) {
		t.Fatalf("denegaciones sin almacen: %v", err)
	}
}

func TestRegistroContextoActorV3PostgreSQLNormalizaTimestamptzBinarioRealDePGXEnUTC(t *testing.T) {
	mapaTipos := pgtype.NewMap()
	origen := time.Date(
		2026, time.July, 15, 10, 0, 0, 123_456_000,
		time.FixedZone("Europe/Madrid-prueba", 2*60*60),
	)
	binario, err := mapaTipos.Encode(
		pgtype.TimestamptzOID, pgtype.BinaryFormatCode, origen, nil,
	)
	if err != nil {
		t.Fatalf("codificar timestamptz con pgx: %v", err)
	}
	var decodificado time.Time
	if err = mapaTipos.Scan(
		pgtype.TimestamptzOID, pgtype.BinaryFormatCode, binario, &decodificado,
	); err != nil {
		t.Fatalf("escanear timestamptz con pgx: %v", err)
	}
	canonico := decodificado.UTC()
	if !canonico.Equal(origen) || canonico.Location() != time.UTC ||
		!instanteRegistroContextoActorV3PostgreSQLValido(canonico) {
		t.Fatalf("pgx no entrego instante UTC canonico: origen=%v resultado=%v zona=%v",
			origen, canonico, canonico.Location())
	}
}

func nuevaTransaccionRegistroContextoActorV3PostgreSQLPrueba(
	concedida bool,
	codigo, huella string,
	registrada time.Time,
) *transaccionRegistroContextoActorV3PostgreSQLPrueba {
	tx := &transaccionRegistroContextoActorV3PostgreSQLPrueba{}
	tx.filas = &filasRegistroContextoActorV3PostgreSQLPrueba{resultados: []resultadoRegistroContextoActorV3PostgreSQLPrueba{{
		concedida: &concedida, codigo: &codigo, huella: &huella, registrada: &registrada,
	}}}
	return tx
}

type revalidadorRegistroContextoActorV3PostgreSQLPrueba struct {
	resultado domain.AutenticacionRevalidadaV1
}

func (r revalidadorRegistroContextoActorV3PostgreSQLPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type resolutorRegistroContextoActorV3PostgreSQLPrueba struct {
	resultado domain.ResultadoContextoActorRegistradoV2
}

func (r resolutorRegistroContextoActorV3PostgreSQLPrueba) ResolverContextoActorRegistradoV2(
	context.Context,
	domain.SolicitudContextoActor,
) (domain.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type relojRegistroContextoActorV3PostgreSQLPrueba struct{ ahora time.Time }

func (r relojRegistroContextoActorV3PostgreSQLPrueba) Ahora() time.Time { return r.ahora }

type generadorCorrelacionRegistroContextoActorV3PostgreSQLPrueba struct{ valor string }

func (g generadorCorrelacionRegistroContextoActorV3PostgreSQLPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

type escenarioRegistroContextoActorV3PostgreSQLPrueba struct {
	ahora     time.Time
	solicitud domain.SolicitudAutorizacionLigadaV3
	decision  domain.DecisionAutorizacionLigadaV3
	motivo    domain.ReferenciaEntradaCatalogo
	resultado domain.ResultadoContextoActorRegistradoV2
}

func nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(
	t *testing.T,
	concedida bool,
) escenarioRegistroContextoActorV3PostgreSQLPrueba {
	t.Helper()
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: domain.AuthMethodCertificate,
		Garantia: domain.AuthAssuranceHigh,
	}
	instantaneaActor := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 3,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: 4,
		PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: ^uint64(0),
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: ^uint64(0) - 1,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantaneaActor, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	representacion, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatalf("canon de contexto: %v", err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatalf("huella de contexto: %v", err)
	}
	acreditacion := domain.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef: "prc_0123456789abcdefghijkl", ProcedenciaVersion: 1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad:    domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := domain.ManifiestoProcedenciaContextoActorV1{
		Esquema:           domain.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta:            domain.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: instantaneaActor.CuentaVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Persona:           domain.ProcedenciaPersonaContextoActorV1{PersonaRef: instantaneaActor.PersonaRef, Version: instantaneaActor.PersonaVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Perfil:            domain.ProcedenciaPerfilContextoActorV1{PerfilRef: instantaneaActor.PerfilActivoRef, Version: instantaneaActor.PerfilVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Contexto:          domain.ProcedenciaVinculoContextoActorV1{VinculoRef: instantaneaActor.VinculoRef, Version: instantaneaActor.VinculoVersion, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Vinculos:          make([]domain.ProcedenciaVinculoReferenciaContextoActorV1, 0),
	}
	canonManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatalf("canon de manifiesto: %v", err)
	}
	huellaManifiesto, err := domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(canonManifiesto)
	if err != nil {
		t.Fatalf("huella de manifiesto: %v", err)
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn", Contexto: actor,
		RepresentacionCanonica: representacion, HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico:     canonManifiesto,
		ManifiestoProcedenciaHuellaSHA256: huellaManifiesto,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
	if err := actor.Validar(); err != nil {
		t.Fatalf("actor: %v", err)
	}
	manifiestoRehidratado, err := domain.RehidratarManifiestoProcedenciaContextoActorV1(canonManifiesto)
	if err != nil {
		t.Fatalf("rehidratar manifiesto: %v", err)
	}
	if err := manifiestoRehidratado.ValidarParaContexto(actor); err != nil {
		t.Fatalf("manifiesto para contexto: %v", err)
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef: "aut_0123456789abcdefghijkl", AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef: "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 2,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64), CuentaRef: cuenta.CuentaRef,
		CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: domain.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef: "pga_0123456789abcdefghijkl", PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn: ahora.Add(-10 * time.Minute), SesionEmitidaEn: ahora.Add(-9 * time.Minute),
		SesionRevalidadaEn: ahora.Add(-3 * time.Minute), SesionValidaHasta: ahora.Add(20 * time.Minute),
	}
	if err := resultado.Validar(); err != nil {
		t.Fatalf("resultado de contexto: %v", err)
	}
	if err := autenticacion.Validar(); err != nil {
		t.Fatalf("autenticacion: %v", err)
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(
		context.Background(), revalidadorRegistroContextoActorV3PostgreSQLPrueba{autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef},
		resolutorRegistroContextoActorV3PostgreSQLPrueba{resultado},
		domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: instantaneaActor.PerfilActivoRef},
		relojRegistroContextoActorV3PostgreSQLPrueba{ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo := domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64),
		EntradaClave:         "motivo_11111111111111111111111111111111",
	}
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(), generadorCorrelacionRegistroContextoActorV3PostgreSQLPrueba{
			valor: "correlacion_11111111111111111111111111111111",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo,
		Accion:    "bolsa.expediente.leer",
		Recurso:   domain.RecursoAutorizable{Referencia: "expediente:1", ModuloID: "bolsa", Tipo: "expediente", Ambitos: map[string]string{"unidad": "seleccion"}},
		Finalidad: "gestion_bolsa", Correlacion: correlacion,
	})
	if err != nil {
		t.Fatal(err)
	}
	version := domain.VersionRol{
		RolID: "tecnico_bolsa", Version: 1, Nombre: "Tecnico de bolsa", Estado: domain.EstadoVersionRolPublicada,
		Concesiones:  []domain.ConcesionRol{{Accion: "bolsa.expediente.leer", ModuloID: "bolsa", TipoRecurso: "expediente", Finalidades: []string{"gestion_bolsa"}, GarantiaMinima: domain.AuthAssuranceSubstantial}},
		PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-24 * time.Hour),
	}
	politicas := []domain.PoliticaRestrictiva{}
	if !concedida {
		politicas = append(politicas, domain.PoliticaRestrictiva{
			PoliticaID: "bloqueo_expediente", Version: 1, Nombre: "Bloqueo expediente",
			Estado: domain.EstadoPoliticaRestrictivaPublicada, Efecto: domain.EfectoPoliticaDenegar,
			Acciones: []string{"bolsa.expediente.leer"}, Modulos: []string{"bolsa"},
			TiposRecurso: []string{"expediente"}, FinalidadesPermitidas: []string{"gestion_bolsa"},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-2 * time.Hour),
		})
	}
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(politicas)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := domain.InstantaneaAutorizacion{
		AsignacionPerfil: domain.AsignacionPerfil{
			AsignacionID: "asig-bolsa", Version: 1, PerfilActivoRef: instantaneaActor.PerfilActivoRef,
			PrincipalID: instantaneaActor.PersonaRef, VersionRolRef: version.Referencia(),
			Estado: domain.EstadoAsignacionPerfilActiva, Ambitos: []domain.AmbitoPerfil{{Clave: "unidad", Valores: []string{"seleccion"}}},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			EmitidaPor: "administrador-identidades", EmitidaEn: ahora.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn,
		},
		Politicas: politicas, RevisionCatalogoPoliticas: 1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, "dec_0123456789abcdef0123456789abcdef",
		ahora, ahora.Add(90*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioRegistroContextoActorV3PostgreSQLPrueba{
		ahora: ahora, solicitud: solicitud, decision: decision, motivo: motivo, resultado: resultado,
	}
}
