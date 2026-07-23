//go:build o206postgresql

package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type vectorConfirmacionAltaPostgreSQL18 struct {
	parametros parametrosConfirmacionAlta
	expediente domain.Expediente
}

type iniciadorDecoradoPostgreSQL18 struct {
	pool      *pgxpool.Pool
	mu        sync.Mutex
	inicios   int
	decorar   func(int, pgx.Tx) pgx.Tx
	errorEn   int
	errorBase error
}

func (i *iniciadorDecoradoPostgreSQL18) BeginTx(
	ctx context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.mu.Lock()
	i.inicios++
	indice := i.inicios
	i.mu.Unlock()
	if i.errorEn == indice {
		return nil, i.errorBase
	}
	tx, err := i.pool.BeginTx(ctx, opciones)
	if err != nil {
		return nil, err
	}
	if i.decorar != nil {
		return i.decorar(indice, tx), nil
	}
	return tx, nil
}

type transaccionCommitPerdidoPostgreSQL18 struct {
	pgx.Tx
}

func (t transaccionCommitPerdidoPostgreSQL18) Commit(
	ctx context.Context,
) error {
	if err := t.Tx.Commit(ctx); err != nil {
		return err
	}
	return io.EOF
}

type transaccionCanceladaPreCommitPostgreSQL18 struct {
	pgx.Tx
	cancelar context.CancelFunc
}

func (t transaccionCanceladaPreCommitPostgreSQL18) Query(
	ctx context.Context,
	sql string,
	argumentos ...any,
) (pgx.Rows, error) {
	filas, err := t.Tx.Query(ctx, sql, argumentos...)
	if err != nil {
		return nil, err
	}
	filas.Close()
	_ = t.Tx.Rollback(context.Background())
	t.cancelar()
	return nil, context.Canceled
}

type transaccionReciboAdulteradoPostgreSQL18 struct {
	pgx.Tx
}

func (t transaccionReciboAdulteradoPostgreSQL18) Query(
	ctx context.Context,
	sql string,
	argumentos ...any,
) (pgx.Rows, error) {
	filas, err := t.Tx.Query(ctx, sql, argumentos...)
	if err != nil {
		return nil, err
	}
	return filasReciboAdulteradoPostgreSQL18{Rows: filas}, nil
}

type filasReciboAdulteradoPostgreSQL18 struct {
	pgx.Rows
}

func (f filasReciboAdulteradoPostgreSQL18) Scan(destinos ...any) error {
	if err := f.Rows.Scan(destinos...); err != nil {
		return err
	}
	huella, ok := destinos[7].(*string)
	if !ok {
		return errors.New("destino de huella inesperado")
	}
	*huella = strings.Repeat("f", 64)
	return nil
}

func TestConfirmacionAltaPostgreSQL18Real(t *testing.T) {
	dsnAdmin := os.Getenv("VEC_O206_DSN_ADMIN")
	dsnRuntime := os.Getenv("VEC_O206_DSN_RUNTIME")
	if dsnAdmin == "" || dsnRuntime == "" {
		t.Skip("runner PostgreSQL O2-06 no activado")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsnAdmin)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	runtime, err := pgxpool.New(ctx, dsnRuntime)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()

	t.Run("canon Go coincide byte a byte con SQL", func(t *testing.T) {
		vector := prepararVectorConfirmacionPostgreSQL18(
			t, admin, "o206_canon",
		)
		var documento efectoAltaV2
		if err := json.Unmarshal(vector.parametros.alta, &documento); err != nil {
			t.Fatal(err)
		}
		var sellos sellosAltaV1
		if err := json.Unmarshal(vector.parametros.sellos, &sellos); err != nil {
			t.Fatal(err)
		}
		candidatura := ports.CandidaturaAlta{
			ReservaRef: documento.ReservaRef,
			Referencias: ports.ReferenciasAlta{
				ExpedienteRef: documento.ExpedienteRef,
				NumeroVisible: documento.NumeroVisible,
				ReciboRef:     documento.ReciboRef,
			},
			AmbitoIdempotenciaHMAC: sellos.Activo.AmbitoHMAC,
			HuellaPeticionHMAC:     sellos.Activo.HuellaHMAC,
			OrganizacionRef:        documento.OrganizacionRef,
			ActorRef:               documento.ActorRef,
			PerfilRef:              documento.PerfilRef,
		}
		proyeccion, err := NuevoProyectorEfectoAltaV2().
			ProyectarEfectoAlta(ports.SolicitudProyectarEfectoAlta{
				Expediente:  vector.expediente,
				Candidatura: candidatura,
			})
		if err != nil {
			t.Fatal(err)
		}
		contenido, _, err := proyeccion.Datos()
		if err != nil || !bytes.Equal(contenido, vector.parametros.alta) {
			t.Fatalf("canon Go/SQL divergente: %v", err)
		}
	})

	t.Run("alta y replay exacto", func(t *testing.T) {
		vector := prepararVectorConfirmacionPostgreSQL18(
			t, admin, "o206_exito",
		)
		adaptador := &TransaccionAltasPostgreSQL{pool: runtime}
		primero, err := adaptador.confirmarParametros(
			ctx, vector.expediente, vector.parametros,
		)
		if err != nil {
			t.Fatal(err)
		}
		segundo, err := adaptador.confirmarParametros(
			ctx, vector.expediente, vector.parametros,
		)
		if err != nil || segundo != primero {
			t.Fatalf("replay divergente: %#v / %#v / %v",
				primero, segundo, err)
		}
		afirmarAgregadoConfirmacionPostgreSQL18(t, admin, "o206_exito", 1)
	})

	t.Run("concurrencia devuelve un recibo", func(t *testing.T) {
		vector := prepararVectorConfirmacionPostgreSQL18(
			t, admin, "o206_carrera",
		)
		adaptador := &TransaccionAltasPostgreSQL{pool: runtime}
		const total = 4
		type resultado struct {
			recibo ports.ReciboAlta
			err    error
		}
		resultados := make(chan resultado, total)
		for indice := 0; indice < total; indice++ {
			go func() {
				recibo, err := adaptador.confirmarParametros(
					ctx, vector.expediente, vector.parametros,
				)
				resultados <- resultado{recibo: recibo, err: err}
			}()
		}
		var esperado ports.ReciboAlta
		for indice := 0; indice < total; indice++ {
			obtenido := <-resultados
			if obtenido.err != nil {
				t.Fatal(obtenido.err)
			}
			if indice == 0 {
				esperado = obtenido.recibo
			} else if obtenido.recibo != esperado {
				t.Fatalf("recibos concurrentes divergentes: %#v / %#v",
					esperado, obtenido.recibo)
			}
		}
		afirmarAgregadoConfirmacionPostgreSQL18(
			t, admin, "o206_carrera", 1,
		)
	})

	t.Run("cancelacion antes de commit revierte todo", func(t *testing.T) {
		vector := prepararVectorConfirmacionPostgreSQL18(
			t, admin, "o206_precommit",
		)
		ctxCancelado, cancelar := context.WithCancel(ctx)
		iniciador := &iniciadorDecoradoPostgreSQL18{
			pool: runtime,
			decorar: func(_ int, tx pgx.Tx) pgx.Tx {
				return transaccionCanceladaPreCommitPostgreSQL18{
					Tx: tx, cancelar: cancelar,
				}
			},
		}
		adaptador := &TransaccionAltasPostgreSQL{pool: iniciador}
		_, err := adaptador.confirmarParametros(
			ctxCancelado, vector.expediente, vector.parametros,
		)
		if !errors.Is(err, context.Canceled) || iniciador.inicios != 1 {
			t.Fatalf("cancelación pre-COMMIT no concluyente: %v", err)
		}
		afirmarAgregadoConfirmacionPostgreSQL18(
			t, admin, "o206_precommit", 0,
		)
	})

	t.Run("respuesta perdida tras commit se reconcilia", func(t *testing.T) {
		vector := prepararVectorConfirmacionPostgreSQL18(
			t, admin, "o206_perdida",
		)
		iniciador := &iniciadorDecoradoPostgreSQL18{
			pool: runtime,
			decorar: func(indice int, tx pgx.Tx) pgx.Tx {
				if indice == 1 {
					return transaccionCommitPerdidoPostgreSQL18{Tx: tx}
				}
				return tx
			},
		}
		adaptador := &TransaccionAltasPostgreSQL{pool: iniciador}
		recibo, err := adaptador.confirmarParametros(
			ctx, vector.expediente, vector.parametros,
		)
		if err != nil || recibo.ValidarPara(vector.expediente) != nil ||
			iniciador.inicios != 2 {
			t.Fatalf("COMMIT perdido no reconciliado: %#v / %v / %d",
				recibo, err, iniciador.inicios)
		}
		afirmarAgregadoConfirmacionPostgreSQL18(
			t, admin, "o206_perdida", 1,
		)
	})

	t.Run("segundo fallo conserva indeterminado", func(t *testing.T) {
		vector := prepararVectorConfirmacionPostgreSQL18(
			t, admin, "o206_segundo",
		)
		iniciador := &iniciadorDecoradoPostgreSQL18{
			pool: runtime, errorEn: 2, errorBase: io.EOF,
			decorar: func(indice int, tx pgx.Tx) pgx.Tx {
				if indice == 1 {
					return transaccionCommitPerdidoPostgreSQL18{Tx: tx}
				}
				return tx
			},
		}
		adaptador := &TransaccionAltasPostgreSQL{pool: iniciador}
		recibo, err := adaptador.confirmarParametros(
			ctx, vector.expediente, vector.parametros,
		)
		if recibo != (ports.ReciboAlta{}) ||
			!errors.Is(err, ports.ErrResultadoAltaIndeterminado) ||
			iniciador.inicios != 2 {
			t.Fatalf("segundo fallo no quedó indeterminado: %#v / %v / %d",
				recibo, err, iniciador.inicios)
		}
		afirmarAgregadoConfirmacionPostgreSQL18(
			t, admin, "o206_segundo", 1,
		)
	})

	t.Run("recibo adulterado nunca confirma", func(t *testing.T) {
		vector := prepararVectorConfirmacionPostgreSQL18(
			t, admin, "o206_adulterado",
		)
		iniciador := &iniciadorDecoradoPostgreSQL18{
			pool: runtime,
			decorar: func(_ int, tx pgx.Tx) pgx.Tx {
				return transaccionReciboAdulteradoPostgreSQL18{Tx: tx}
			},
		}
		adaptador := &TransaccionAltasPostgreSQL{pool: iniciador}
		_, err := adaptador.confirmarParametros(
			ctx, vector.expediente, vector.parametros,
		)
		if !errors.Is(err, ports.ErrResultadoAltaNoConfiable) {
			t.Fatalf("recibo adulterado aceptado: %v", err)
		}
		afirmarAgregadoConfirmacionPostgreSQL18(
			t, admin, "o206_adulterado", 0,
		)
	})

	t.Run("replay con pool y proceso logico nuevos", func(t *testing.T) {
		vector := prepararVectorConfirmacionPostgreSQL18(
			t, admin, "o206_reinicio",
		)
		primero, err := pgxpool.New(ctx, dsnRuntime)
		if err != nil {
			t.Fatal(err)
		}
		reciboPrimero, err := (&TransaccionAltasPostgreSQL{
			pool: primero,
		}).confirmarParametros(ctx, vector.expediente, vector.parametros)
		primero.Close()
		if err != nil {
			t.Fatal(err)
		}
		segundo, err := pgxpool.New(ctx, dsnRuntime)
		if err != nil {
			t.Fatal(err)
		}
		defer segundo.Close()
		reciboSegundo, err := (&TransaccionAltasPostgreSQL{
			pool: segundo,
		}).confirmarParametros(ctx, vector.expediente, vector.parametros)
		if err != nil || reciboSegundo != reciboPrimero {
			t.Fatalf("replay tras reinicio divergente: %#v / %#v / %v",
				reciboPrimero, reciboSegundo, err)
		}
	})
}

func prepararVectorConfirmacionPostgreSQL18(
	t *testing.T,
	admin *pgxpool.Pool,
	caso string,
) vectorConfirmacionAltaPostgreSQL18 {
	t.Helper()
	ctx := context.Background()
	if _, err := admin.Exec(
		ctx,
		`SELECT public.preparar_vector_o2_05($1, 'valido', 1)`,
		caso,
	); err != nil {
		t.Fatal(err)
	}
	var vector vectorConfirmacionAltaPostgreSQL18
	err := admin.QueryRow(ctx, `
		SELECT capacidad, decision, motivo, contexto,
		       persona_version::text, perfil_version::text,
		       payload, cose, evidencia, spki, alta, sellos
		  FROM public.vectores_o2_05
		 WHERE caso=$1`,
		caso,
	).Scan(
		&vector.parametros.capacidad,
		&vector.parametros.decision,
		&vector.parametros.motivo,
		&vector.parametros.contexto,
		&vector.parametros.personaVersion,
		&vector.parametros.perfilVersion,
		&vector.parametros.payload,
		&vector.parametros.cose,
		&vector.parametros.evidencia,
		&vector.parametros.spki,
		&vector.parametros.alta,
		&vector.parametros.sellos,
	)
	if err != nil {
		t.Fatal(err)
	}
	vector.expediente = reconstruirExpedientePostgreSQL18(
		t, vector.parametros.alta,
	)
	return vector
}

func reconstruirExpedientePostgreSQL18(
	t *testing.T,
	alta []byte,
) domain.Expediente {
	t.Helper()
	var documento efectoAltaV2
	if err := json.Unmarshal(alta, &documento); err != nil {
		t.Fatal(err)
	}
	parsearInstante := func(valor string) time.Time {
		instante, err := time.Parse("2006-01-02T15:04:05.000000Z", valor)
		if err != nil {
			t.Fatal(err)
		}
		return instante
	}
	parsearFecha := func(valor string) time.Time {
		fecha, err := time.Parse("2006-01-02", valor)
		if err != nil {
			t.Fatal(err)
		}
		return fecha
	}
	solicitud := domain.SolicitudCentro{
		CentroRef:     documento.Solicitud.CentroRef,
		ContactoRef:   documento.Solicitud.ContactoRef,
		CategoriaRef:  documento.Solicitud.CategoriaRef,
		GrupoSubgrupo: documento.Solicitud.GrupoSubgrupo,
		MotivoClave:   domain.ClaveCatalogo(documento.Solicitud.MotivoClave),
		Detalle:       documento.Solicitud.Detalle,
		Periodo: domain.PeriodoPrevisto{
			Inicio: parsearFecha(documento.Solicitud.Periodo.Inicio),
			Fin:    parsearFecha(documento.Solicitud.Periodo.Fin),
		},
		DocumentosAdjuntos: append(
			[]string(nil), documento.Solicitud.DocumentosAdjuntos...,
		),
		Observaciones: documento.Solicitud.Observaciones,
	}
	expediente, err := domain.NuevoExpediente(domain.AltaExpediente{
		Referencia:      documento.ExpedienteRef,
		OrganizacionRef: documento.OrganizacionRef,
		NumeroVisible:   documento.NumeroVisible,
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: documento.Flujo.DefinicionRef,
			Version:       documento.Flujo.Version,
			HuellaSHA256:  documento.Flujo.HuellaSHA256,
		},
		FaseInicial: domain.ClaveFase(documento.FaseActual),
		Solicitud:   solicitud,
		Actuacion: domain.DatosActuacion{
			AccionClave: domain.ClaveCatalogo(
				documento.Actuacion.AccionClave,
			),
			ActorRef:    documento.Actuacion.ActorRef,
			UnidadRef:   documento.Actuacion.UnidadRef,
			ReciboRef:   documento.Actuacion.ReciboRef,
			RealizadaEn: parsearInstante(documento.Actuacion.RealizadaEn),
			FaseDestino: domain.ClaveFase(
				documento.Actuacion.FaseDestino,
			),
			EstadoDestino: domain.EstadoOperativo(
				documento.Actuacion.EstadoDestino,
			),
			Observaciones: documento.Actuacion.Observaciones,
			DocumentosRef: append(
				[]string(nil), documento.Actuacion.DocumentosRef...,
			),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return expediente
}

func afirmarAgregadoConfirmacionPostgreSQL18(
	t *testing.T,
	admin *pgxpool.Pool,
	caso string,
	esperado int,
) {
	t.Helper()
	var total int
	err := admin.QueryRow(context.Background(), `
		SELECT count(*)
		  FROM vec_contratacion_temporal.confirmacion_agregado_alta
		 WHERE expediente_ref='expediente:ct:o205:' || $1`,
		caso,
	).Scan(&total)
	if err != nil || total != esperado {
		t.Fatalf("agregado %s=%d, esperado=%d, error=%v",
			caso, total, esperado, err)
	}
}
