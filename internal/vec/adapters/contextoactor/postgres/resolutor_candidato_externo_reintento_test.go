package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	postgresqlcomun "vec-diputacion-granada/internal/shared/postgresql"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func probarConsumidoresContextoExterno(t *testing.T, probar func(*testing.T, bool)) {
	t.Helper()
	for _, consumidor := range []struct {
		nombre   string
		usuarios bool
	}{{"candidato", false}, {"usuarios", true}} {
		t.Run(consumidor.nombre, func(t *testing.T) { probar(t, consumidor.usuarios) })
	}
}

func solicitudYFilaContextoExterno(t *testing.T, usuarios bool) (ports.SolicitudResolucionRegistroContextoActorV2, filaContextoActorDoble) {
	t.Helper()
	solicitud, fila := solicitudYFilaContextoActorV2(t)
	if !usuarios {
		return solicitud, fila
	}
	actor, err := domain.RehidratarContextoActorVinculadoV2(fila.valores[2].([]byte))
	if err != nil {
		t.Fatal(err)
	}
	actor.Instantanea.Vinculos = nil
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	manifiesto, err := domain.RehidratarManifiestoProcedenciaContextoActorV1(fila.valores[4].([]byte))
	if err != nil {
		t.Fatal(err)
	}
	manifiesto.Vinculos = []domain.ProcedenciaVinculoReferenciaContextoActorV1{}
	procedencia, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	h1, h2 := sha256.Sum256(canon), sha256.Sum256(procedencia)
	fila.valores[2], fila.valores[3] = canon, hex.EncodeToString(h1[:])
	fila.valores[4], fila.valores[5] = procedencia, hex.EncodeToString(h2[:])
	return solicitud, fila
}

func nuevoResolutorContextoExternoPrueba(t *testing.T, pool iniciadorContextoActorPostgreSQL, usuarios bool) *ResolutorRegistroContextoActorExternoPostgreSQLV1 {
	t.Helper()
	r, err := nuevoResolutorRegistroContextoActorExternoPostgreSQLV1(pool,
		bytes.NewReader(bytes.Repeat([]byte{0x55}, bytesAleatoriosReferenciaContextoActorV2)))
	if err != nil {
		t.Fatal(err)
	}
	r.usuarios = usuarios
	return r
}

func comprobarIntentosContextoExterno(t *testing.T, pool *poolContextoActorDoble, usuarios bool) {
	t.Helper()
	resolver, reconciliar := consultaResolverContextoCandidatoExternoV1, consultaReconciliarContextoCandidatoExternoV1
	if usuarios {
		resolver, reconciliar = consultaResolverContextoUsuariosExternoV1, consultaReconciliarContextoUsuariosExternoV1
	}
	argumentos := pool.transacciones[0].argumentos[0]
	for i := 0; i < pool.llamadas; i++ {
		tx := pool.transacciones[i]
		if len(tx.consultas) != 1 || len(tx.argumentos) != 1 || !reflect.DeepEqual(tx.argumentos[0], argumentos) ||
			len(argumentos) != 5 || tx.rollbacks != 1 {
			t.Fatalf("intento %d cambió argumentos o no cerró la transacción", i+1)
		}
		if tx.consultas[0] == resolver {
			if pool.opciones[i].IsoLevel != pgx.Serializable || pool.opciones[i].AccessMode != pgx.ReadWrite {
				t.Fatal("escritura sin transacción serializable nueva")
			}
		} else if tx.consultas[0] != reconciliar || pool.opciones[i].IsoLevel != pgx.ReadCommitted {
			t.Fatal("consulta ajena a la población o reconciliación sin snapshot renovable")
		}
	}
}

func TestResolutorExternoReintentaCarrerasConsultaYCommit(t *testing.T) {
	probarConsumidoresContextoExterno(t, func(t *testing.T, usuarios bool) {
		for _, codigo := range []string{"40001", "40P01"} {
			for _, alConfirmar := range []bool{false, true} {
				nombre := codigo + "/consulta"
				if alConfirmar {
					nombre = codigo + "/commit"
				}
				t.Run(nombre, func(t *testing.T) {
					solicitud, fila := solicitudYFilaContextoExterno(t, usuarios)
					transacciones := make([]*txContextoActorDoble, 4)
					for i := 0; i < 3; i++ {
						conflicto := &pgconn.PgError{Code: codigo}
						transacciones[i] = &txContextoActorDoble{filas: []pgx.Row{filaContextoActorDoble{err: conflicto}}}
						if alConfirmar {
							transacciones[i] = &txContextoActorDoble{filas: []pgx.Row{fila}, errCommit: conflicto}
						}
					}
					transacciones[3] = &txContextoActorDoble{filas: []pgx.Row{fila}}
					pool := &poolContextoActorDoble{transacciones: transacciones}
					r := nuevoResolutorContextoExternoPrueba(t, pool, usuarios)
					confirmacion, err := r.ResolverYRegistrarContextoActorV2(context.Background(), solicitud)
					if err != nil || confirmacion.ValidarParaProductiva(solicitud) != nil || pool.llamadas != 4 {
						t.Fatalf("carrera no recuperada: intentos=%d err=%v", pool.llamadas, err)
					}
					comprobarIntentosContextoExterno(t, pool, usuarios)
					for _, tx := range transacciones {
						if tx.consultas[0] != transacciones[0].consultas[0] {
							t.Fatal("se reconcilió un aborto seguro")
						}
					}
				})
			}
		}
	})
}

func TestResolutorExternoAgotaPoliticaComun(t *testing.T) {
	probarConsumidoresContextoExterno(t, func(t *testing.T, usuarios bool) {
		t.Parallel()
		solicitud, _ := solicitudYFilaContextoExterno(t, usuarios)
		transacciones := make([]*txContextoActorDoble, postgresqlcomun.IntentosMaximosCarreraSerializable+1)
		for i := range transacciones {
			transacciones[i] = &txContextoActorDoble{filas: []pgx.Row{filaContextoActorDoble{err: &pgconn.PgError{Code: "40001"}}}}
		}
		pool := &poolContextoActorDoble{transacciones: transacciones}
		r := nuevoResolutorContextoExternoPrueba(t, pool, usuarios)
		confirmacion, err := r.ResolverYRegistrarContextoActorV2(context.Background(), solicitud)
		if !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) || confirmacion.RegistroContextoRef != "" ||
			pool.llamadas != postgresqlcomun.IntentosMaximosCarreraSerializable {
			t.Fatalf("agotamiento inesperado: intentos=%d err=%v", pool.llamadas, err)
		}
		comprobarIntentosContextoExterno(t, pool, usuarios)
	})
}

// Cancela al entrar en el select de la espera común, sin depender del reloj.
type contextoCancelarEsperaExterno struct {
	context.Context
	cancelar context.CancelFunc
}

func (c contextoCancelarEsperaExterno) Done() <-chan struct{} {
	c.cancelar()
	return c.Context.Done()
}

func TestResolutorExternoCancelaDuranteEspera(t *testing.T) {
	probarConsumidoresContextoExterno(t, func(t *testing.T, usuarios bool) {
		solicitud, fila := solicitudYFilaContextoExterno(t, usuarios)
		pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{
			{filas: []pgx.Row{filaContextoActorDoble{err: &pgconn.PgError{Code: "40001"}}}},
			{filas: []pgx.Row{fila}},
		}}
		ctx, cancelar := context.WithCancel(context.Background())
		defer cancelar()
		r := nuevoResolutorContextoExternoPrueba(t, pool, usuarios)
		_, err := r.ResolverYRegistrarContextoActorV2(contextoCancelarEsperaExterno{ctx, cancelar}, solicitud)
		if !errors.Is(err, context.Canceled) || pool.llamadas != 1 {
			t.Fatalf("se escribió tras cancelar la espera: intentos=%d err=%v", pool.llamadas, err)
		}
		comprobarIntentosContextoExterno(t, pool, usuarios)
	})
}

func TestResolutorExternoDeniegaRevocacionTrasCarrera(t *testing.T) {
	probarConsumidoresContextoExterno(t, func(t *testing.T, usuarios bool) {
		solicitud, fila := solicitudYFilaContextoExterno(t, usuarios)
		pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{
			{filas: []pgx.Row{filaContextoActorDoble{err: &pgconn.PgError{Code: "40001"}}}},
			{filas: []pgx.Row{filaContextoActorDoble{err: &pgconn.PgError{Code: "42501"}}}},
			{filas: []pgx.Row{fila}},
		}}
		r := nuevoResolutorContextoExternoPrueba(t, pool, usuarios)
		_, err := r.ResolverYRegistrarContextoActorV2(context.Background(), solicitud)
		if !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) || pool.llamadas != 2 {
			t.Fatalf("se eludió la denegación: intentos=%d err=%v", pool.llamadas, err)
		}
		comprobarIntentosContextoExterno(t, pool, usuarios)
	})
}

func TestResolutorExternoNoReintentaErroresDefinitivos(t *testing.T) {
	probarConsumidoresContextoExterno(t, func(t *testing.T, usuarios bool) {
		for _, codigo := range []string{"23505", "42501", "57014", "PCA01", "PCA02"} {
			t.Run(codigo, func(t *testing.T) {
				solicitud, fila := solicitudYFilaContextoExterno(t, usuarios)
				pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{
					{filas: []pgx.Row{filaContextoActorDoble{err: &pgconn.PgError{Code: codigo}}}},
					{filas: []pgx.Row{fila}},
				}}
				r := nuevoResolutorContextoExternoPrueba(t, pool, usuarios)
				_, err := r.ResolverYRegistrarContextoActorV2(context.Background(), solicitud)
				if !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) || pool.llamadas != 1 || pool.transacciones[0].commits != 0 {
					t.Fatalf("error definitivo repetido: intentos=%d err=%v", pool.llamadas, err)
				}
				comprobarIntentosContextoExterno(t, pool, usuarios)
			})
		}
	})
}

type txPreparacionContextoExternoPrueba struct {
	*txContextoActorDoble
	fallo error
}

func (tx txPreparacionContextoExternoPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, tx.fallo
}

type poolFalloContextoExternoPrueba struct {
	poolContextoActorDoble
	falloInicio      error
	falloPreparacion error
	tx               *txContextoActorDoble
}

func (p *poolFalloContextoExternoPrueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	p.llamadas++
	if p.falloInicio != nil {
		return nil, p.falloInicio
	}
	return txPreparacionContextoExternoPrueba{p.tx, p.falloPreparacion}, nil
}

func TestResolutorExternoInicioYPreparacionFallanConErrorNominal(t *testing.T) {
	probarConsumidoresContextoExterno(t, func(t *testing.T, usuarios bool) {
		for _, etapa := range []string{"inicio", "preparacion"} {
			t.Run(etapa, func(t *testing.T) {
				solicitud, _ := solicitudYFilaContextoExterno(t, usuarios)
				fallo := &pgconn.PgError{Code: "40001", Message: "detalle SQL reservado"}
				pool := &poolFalloContextoExternoPrueba{tx: &txContextoActorDoble{}}
				if etapa == "inicio" {
					pool.falloInicio = fallo
				} else {
					pool.falloPreparacion = fallo
				}
				r := nuevoResolutorContextoExternoPrueba(t, pool, usuarios)
				confirmacion, err := r.ResolverYRegistrarContextoActorV2(context.Background(), solicitud)
				var errorSQL *pgconn.PgError
				if err != ports.ErrResolutorRegistroContextoActorNoDisponible || errors.As(err, &errorSQL) ||
					confirmacion.RegistroContextoRef != "" || pool.llamadas != 1 || pool.tx.commits != 0 || len(pool.tx.consultas) != 0 {
					t.Fatalf("fallo de %s no terminó cerrado: llamadas=%d err=%v", etapa, pool.llamadas, err)
				}
				if etapa == "preparacion" && pool.tx.rollbacks != 1 {
					t.Fatal("transacción de preparación fallida sin rollback")
				}
			})
		}
	})
}
