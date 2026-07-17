package confianzaatestacionv2

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

type iniciadorDoble struct {
	tx       pgx.Tx
	err      error
	llamadas int
	opciones pgx.TxOptions
}

type relojConfianzaAtestacionV2Prueba struct {
	ahora time.Time
}

func (r *relojConfianzaAtestacionV2Prueba) Ahora() time.Time { return r.ahora }

func (i *iniciadorDoble) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	i.llamadas++
	i.opciones = opciones
	return i.tx, i.err
}

type transaccionDoble struct {
	pgx.Tx
	identidad            identidadTransaccion
	filas                *filasDoble
	errPreparacion       error
	errBloqueo           error
	errConsulta          error
	errIdentidad         error
	errConfirmacion      error
	cancelarBloqueo      context.CancelFunc
	cancelarConsulta     context.CancelFunc
	ejecutadas           []string
	eventos              []string
	consultada           string
	consultasIdentidad   int
	consultaIdentidadSQL string
	confirmaciones       int
	reversiones          int
	errorRollbackCtx     error
	rollbackConPlazo     bool
}

func (t *transaccionDoble) Exec(
	_ context.Context,
	sql string,
	_ ...any,
) (pgconn.CommandTag, error) {
	t.ejecutadas = append(t.ejecutadas, sql)
	if sql == sentenciaBloqueoGobierno {
		t.eventos = append(t.eventos, "bloqueo")
		if t.cancelarBloqueo != nil {
			t.cancelarBloqueo()
		}
		if t.errBloqueo != nil {
			return pgconn.CommandTag{}, t.errBloqueo
		}
		return pgconn.NewCommandTag("SELECT 1"), nil
	}
	if strings.Contains(sql, "SET LOCAL ROLE ") {
		t.eventos = append(t.eventos, "rol")
	} else {
		t.eventos = append(t.eventos, "ajustes")
	}
	if t.errPreparacion != nil {
		return pgconn.CommandTag{}, t.errPreparacion
	}
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (t *transaccionDoble) Query(
	_ context.Context,
	sql string,
	_ ...any,
) (pgx.Rows, error) {
	t.consultada = sql
	t.eventos = append(t.eventos, "datos")
	if t.cancelarConsulta != nil {
		t.cancelarConsulta()
	}
	if t.errConsulta != nil {
		return nil, t.errConsulta
	}
	return t.filas, nil
}

func (t *transaccionDoble) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	t.consultasIdentidad++
	t.consultaIdentidadSQL = sql
	t.eventos = append(t.eventos, "identidad")
	return filaIdentidadDoble{identidad: t.identidad, err: t.errIdentidad}
}

func (t *transaccionDoble) Commit(_ context.Context) error {
	t.confirmaciones++
	t.eventos = append(t.eventos, "commit")
	return t.errConfirmacion
}

func (t *transaccionDoble) Rollback(ctx context.Context) error {
	t.reversiones++
	t.eventos = append(t.eventos, "rollback")
	t.errorRollbackCtx = ctx.Err()
	_, t.rollbackConPlazo = ctx.Deadline()
	return nil
}

type filaIdentidadDoble struct {
	identidad identidadTransaccion
	err       error
}

func (f filaIdentidadDoble) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	valores := []any{
		f.identidad.sesionUsuario,
		f.identidad.usuarioActual,
		f.identidad.instanteConsulta,
		f.identidad.sesionPuedeLogin,
		f.identidad.sesionSuperusuario,
		f.identidad.sesionCreaRoles,
		f.identidad.sesionCreaBases,
		f.identidad.sesionReplica,
		f.identidad.sesionEvitaRLS,
		f.identidad.sesionMembresias,
		f.identidad.sesionMiembroLector,
		f.identidad.actualPuedeLogin,
		f.identidad.actualSuperusuario,
		f.identidad.actualCreaRoles,
		f.identidad.actualCreaBases,
		f.identidad.actualReplica,
		f.identidad.actualEvitaRLS,
		f.identidad.actualSinMembresias,
	}
	return asignarDestinos(destinos, valores)
}

type filasDoble struct {
	pgx.Rows
	filas      []filaConfianza
	indice     int
	err        error
	errEscaneo error
	cerradas   int
}

func (f *filasDoble) Next() bool {
	return f != nil && f.indice < len(f.filas)
}

func (f *filasDoble) Scan(destinos ...any) error {
	if f.errEscaneo != nil {
		return f.errEscaneo
	}
	if f.indice >= len(f.filas) {
		return errors.New("fila de prueba fuera de rango")
	}
	fila := f.filas[f.indice]
	f.indice++
	valores := []any{
		fila.revision,
		fila.huellaConfiguracionSHA256,
		fila.configuracionPublicadaEn,
		fila.configuracionExpiraEn,
		fila.configuracionEstado,
		fila.configuracionRevocadaEn,
		fila.claveID,
		fila.algoritmoCOSE,
		fila.suite,
		fila.audienciaDespliegue,
		bytes.Clone(fila.clavePublicaSPKI),
		fila.huellaClaveSPKISHA256,
		fila.raizValidaDesde,
		fila.raizValidaHasta,
		fila.raizEstado,
		fila.raizRevocadaEn,
	}
	return asignarDestinos(destinos, valores)
}

func (f *filasDoble) Err() error { return f.err }
func (f *filasDoble) Close()     { f.cerradas++ }

func asignarDestinos(destinos, valores []any) error {
	if len(destinos) != len(valores) {
		return fmt.Errorf("numero de destinos inesperado: %d", len(destinos))
	}
	for indice, destino := range destinos {
		puntero := reflect.ValueOf(destino)
		if puntero.Kind() != reflect.Pointer || puntero.IsNil() {
			return errors.New("destino de prueba no escribible")
		}
		valor := reflect.ValueOf(valores[indice])
		if !valor.IsValid() || !valor.Type().AssignableTo(puntero.Elem().Type()) {
			return fmt.Errorf("tipo de destino inesperado en %d", indice)
		}
		puntero.Elem().Set(valor)
	}
	return nil
}

func TestCargarConfiguracionActualReconstruyeConjuntoExactoYConfirma(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	filas := filasConfianzaValidas(t, ahora, true)
	huella := huellaConfiguracionPrueba(filas)
	for indice := range filas {
		filas[indice].huellaConfiguracionSHA256 = huella
	}
	tx := transaccionValida(ahora, filas)
	iniciador := &iniciadorDoble{tx: tx}

	configuracion, err := cargarConfiguracionActual(context.Background(), iniciador)
	if err != nil {
		t.Fatalf("carga valida rechazada: %v", err)
	}
	if err := configuracion.ValidarHuellaSHA256Esperada(huella); err != nil {
		t.Fatalf("la configuracion no conserva el conjunto exacto: %v", err)
	}
	filas[0].clavePublicaSPKI[0] ^= 0xff
	if err := configuracion.ValidarHuellaSHA256Esperada(huella); err != nil {
		t.Fatalf("la configuracion retuvo memoria mutable del origen: %v", err)
	}
	if iniciador.opciones.IsoLevel != pgx.ReadCommitted ||
		iniciador.opciones.AccessMode != pgx.ReadWrite ||
		iniciador.opciones.DeferrableMode != pgx.NotDeferrable ||
		iniciador.opciones.BeginQuery != "" || iniciador.opciones.CommitQuery != "" {
		t.Fatalf("opciones de transaccion inseguras: %+v", iniciador.opciones)
	}
	if tx.confirmaciones != 1 || tx.reversiones != 1 {
		t.Fatalf("ciclo transaccional inesperado: commit=%d rollback=%d", tx.confirmaciones, tx.reversiones)
	}
	if tx.errorRollbackCtx != nil {
		t.Fatal("el rollback no recibio un contexto independiente util")
	}
	if !tx.rollbackConPlazo {
		t.Fatal("el rollback no quedo acotado por plazo")
	}
	if len(tx.ejecutadas) != 3 ||
		!strings.Contains(tx.ejecutadas[0], "SET LOCAL ROLE "+RolLectorAutoridadPostgreSQL) ||
		!strings.Contains(tx.ejecutadas[1], "'search_path', 'pg_catalog'") ||
		tx.ejecutadas[2] != sentenciaBloqueoGobierno || tx.consultasIdentidad != 1 ||
		!strings.Contains(tx.consultaIdentidadSQL, "pg_catalog.clock_timestamp()") {
		t.Fatalf("preparacion transaccional incompleta: %#v", tx.ejecutadas)
	}
	if tx.consultada != consultaConfianzaActual {
		t.Fatalf("se empleo una consulta no gobernada: %q", tx.consultada)
	}
	esperados := []string{"rol", "ajustes", "bloqueo", "identidad", "datos", "commit", "rollback"}
	if !reflect.DeepEqual(tx.eventos, esperados) {
		t.Fatalf("orden transaccional inseguro: %#v", tx.eventos)
	}
}

func TestCargarConfiguracionActualBloqueaGobiernoAntesDeIdentidadYDatos(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	filas := filasConfianzaValidas(t, ahora, false)
	t.Run("fallo_del_bloqueo", func(t *testing.T) {
		tx := transaccionValida(ahora, filas)
		tx.errBloqueo = errors.New("bloqueo no disponible")
		_, err := cargarConfiguracionActual(context.Background(), &iniciadorDoble{tx: tx})
		if !errors.Is(err, ErrCargaConfianzaAtestacionV2NoDisponible) {
			t.Fatalf("el fallo del bloqueo no cerro la carga: %v", err)
		}
		verificarBloqueoImpideLecturas(t, tx)
	})

	t.Run("cancelacion_al_adquirir", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		tx := transaccionValida(ahora, filas)
		tx.cancelarBloqueo = cancelar
		_, err := cargarConfiguracionActual(ctx, &iniciadorDoble{tx: tx})
		if !errors.Is(err, context.Canceled) ||
			!errors.Is(err, ErrCargaConfianzaAtestacionV2NoDisponible) {
			t.Fatalf("la cancelacion del bloqueo no se preservo: %v", err)
		}
		verificarBloqueoImpideLecturas(t, tx)
	})
}

func verificarBloqueoImpideLecturas(t *testing.T, tx *transaccionDoble) {
	t.Helper()
	if tx.consultasIdentidad != 0 || tx.consultaIdentidadSQL != "" ||
		tx.consultada != "" || tx.confirmaciones != 0 {
		t.Fatal("se consulto identidad o datos sin adquirir el bloqueo de gobierno")
	}
	if len(tx.ejecutadas) != 3 || tx.ejecutadas[2] != sentenciaBloqueoGobierno {
		t.Fatalf("orden transaccional inesperado: %#v", tx.ejecutadas)
	}
	if tx.reversiones != 1 {
		t.Fatal("el fallo del bloqueo no revirtio la transaccion")
	}
	esperados := []string{"rol", "ajustes", "bloqueo", "rollback"}
	if !reflect.DeepEqual(tx.eventos, esperados) {
		t.Fatalf("orden ante fallo del bloqueo inesperado: %#v", tx.eventos)
	}
}

func TestNuevoServicioActualUsaRelojExplicitoYConfiguracionCargada(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	filas := filasConfianzaValidas(t, ahora, false)
	huella := huellaConfiguracionPrueba(filas)
	for indice := range filas {
		filas[indice].huellaConfiguracionSHA256 = huella
	}
	tx := transaccionValida(ahora, filas)
	servicio, err := nuevoServicioActual(
		context.Background(),
		&iniciadorDoble{tx: tx},
		&relojConfianzaAtestacionV2Prueba{ahora: ahora},
	)
	if err != nil || servicio == nil {
		t.Fatalf("no se construyo el servicio productivo: %v", err)
	}
}

func TestNuevoServicioActualRechazaRelojNuloAntesDeConsultar(t *testing.T) {
	iniciador := &iniciadorDoble{}
	if _, err := nuevoServicioActual(context.Background(), iniciador, nil); !errors.Is(err, ErrCargaConfianzaAtestacionV2NoDisponible) {
		t.Fatalf("reloj nulo aceptado: %v", err)
	}
	var relojNulo *relojConfianzaAtestacionV2Prueba
	if _, err := nuevoServicioActual(context.Background(), iniciador, relojNulo); !errors.Is(err, ErrCargaConfianzaAtestacionV2NoDisponible) {
		t.Fatalf("reloj con nil tipado aceptado: %v", err)
	}
	if iniciador.llamadas != 0 {
		t.Fatal("se consulto PostgreSQL pese a carecer de reloj")
	}
}

func TestCargarConfiguracionActualRechazaIdentidadesNoAisladas(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	casos := []struct {
		nombre string
		mutar  func(*identidadTransaccion)
	}{
		{"rol_efectivo_distinto", func(i *identidadTransaccion) { i.usuarioActual = "otro_rol" }},
		{"login_sin_login", func(i *identidadTransaccion) { i.sesionPuedeLogin = false }},
		{"login_superusuario", func(i *identidadTransaccion) { i.sesionSuperusuario = true }},
		{"login_crea_roles", func(i *identidadTransaccion) { i.sesionCreaRoles = true }},
		{"login_crea_bases", func(i *identidadTransaccion) { i.sesionCreaBases = true }},
		{"login_replica", func(i *identidadTransaccion) { i.sesionReplica = true }},
		{"login_evita_rls", func(i *identidadTransaccion) { i.sesionEvitaRLS = true }},
		{"login_sin_membresia", func(i *identidadTransaccion) { i.sesionMembresias = 0 }},
		{"login_con_otras_membresias", func(i *identidadTransaccion) { i.sesionMembresias = 2 }},
		{"login_no_miembro_directo", func(i *identidadTransaccion) { i.sesionMiembroLector = false }},
		{"instante_submicrosegundo", func(i *identidadTransaccion) { i.instanteConsulta = i.instanteConsulta.Add(time.Nanosecond) }},
		{"autoridad_con_login", func(i *identidadTransaccion) { i.actualPuedeLogin = true }},
		{"autoridad_privilegiada", func(i *identidadTransaccion) { i.actualSuperusuario = true }},
		{"autoridad_hereda_otro_rol", func(i *identidadTransaccion) { i.actualSinMembresias = false }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			filas := filasConfianzaValidas(t, ahora, false)
			tx := transaccionValida(ahora, filas)
			caso.mutar(&tx.identidad)
			_, err := cargarConfiguracionActual(context.Background(), &iniciadorDoble{tx: tx})
			if !errors.Is(err, ErrCargaConfianzaAtestacionV2NoDisponible) {
				t.Fatalf("identidad insegura aceptada: %v", err)
			}
			if tx.consultada != "" || tx.confirmaciones != 0 {
				t.Fatal("se leyeron datos despues de rechazar la identidad")
			}
		})
	}
}

func TestCargarConfiguracionActualFallaCerradoAnteDatosAlterados(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	casos := []struct {
		nombre string
		mutar  func(*testing.T, []filaConfianza) []filaConfianza
	}{
		{
			"huella_configuracion_distinta",
			func(_ *testing.T, filas []filaConfianza) []filaConfianza {
				for indice := range filas {
					filas[indice].huellaConfiguracionSHA256 = strings.Repeat("0", 64)
				}
				return filas
			},
		},
		{
			"huella_spki_distinta",
			func(_ *testing.T, filas []filaConfianza) []filaConfianza {
				filas[0].huellaClaveSPKISHA256 = strings.Repeat("0", 64)
				return recalcularHuellaConfiguracion(filas)
			},
		},
		{
			"spki_no_canonico",
			func(_ *testing.T, filas []filaConfianza) []filaConfianza {
				filas[0].clavePublicaSPKI = append(filas[0].clavePublicaSPKI, 0)
				filas[0].huellaClaveSPKISHA256 = huellaBytes(filas[0].clavePublicaSPKI)
				return recalcularHuellaConfiguracion(filas)
			},
		},
		{
			"spki_no_ed25519",
			func(t *testing.T, filas []filaConfianza) []filaConfianza {
				privada, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				if err != nil {
					t.Fatal(err)
				}
				spki, err := x509.MarshalPKIXPublicKey(&privada.PublicKey)
				if err != nil {
					t.Fatal(err)
				}
				filas[0].clavePublicaSPKI = spki
				filas[0].huellaClaveSPKISHA256 = huellaBytes(spki)
				return recalcularHuellaConfiguracion(filas)
			},
		},
		{
			"ed25519_todo_cero",
			func(t *testing.T, filas []filaConfianza) []filaConfianza {
				spki, err := x509.MarshalPKIXPublicKey(
					ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)),
				)
				if err != nil {
					t.Fatal(err)
				}
				filas[0].clavePublicaSPKI = spki
				filas[0].huellaClaveSPKISHA256 = huellaBytes(spki)
				return recalcularHuellaConfiguracion(filas)
			},
		},
		{
			"clave_id_duplicada",
			func(_ *testing.T, filas []filaConfianza) []filaConfianza {
				filas[1].claveID = filas[0].claveID
				return recalcularHuellaConfiguracion(filas)
			},
		},
		{
			"spki_duplicado",
			func(_ *testing.T, filas []filaConfianza) []filaConfianza {
				filas[1].clavePublicaSPKI = bytes.Clone(filas[0].clavePublicaSPKI)
				filas[1].huellaClaveSPKISHA256 = filas[0].huellaClaveSPKISHA256
				return recalcularHuellaConfiguracion(filas)
			},
		},
		{
			"filas_de_revisiones_distintas",
			func(_ *testing.T, filas []filaConfianza) []filaConfianza {
				filas[1].revision = "confianza:atestacion:v2:otra"
				return recalcularHuellaConfiguracion(filas)
			},
		},
		{
			"algoritmo_fuera_de_perfil",
			func(_ *testing.T, filas []filaConfianza) []filaConfianza {
				filas[0].algoritmoCOSE = "ES256"
				return filas
			},
		},
		{
			"suite_fuera_de_perfil",
			func(_ *testing.T, filas []filaConfianza) []filaConfianza {
				filas[0].suite = "VEC-AD-2-COSE-EDDSA-0"
				return filas
			},
		},
		{
			"configuracion_expirada",
			func(_ *testing.T, filas []filaConfianza) []filaConfianza {
				for indice := range filas {
					filas[indice].configuracionExpiraEn = ahora
				}
				return recalcularHuellaConfiguracion(filas)
			},
		},
		{
			"configuracion_con_precision_submicrosegundo",
			func(_ *testing.T, filas []filaConfianza) []filaConfianza {
				for indice := range filas {
					filas[indice].configuracionPublicadaEn =
						filas[indice].configuracionPublicadaEn.Add(time.Nanosecond)
				}
				return recalcularHuellaConfiguracion(filas)
			},
		},
		{
			"configuracion_revocada",
			func(_ *testing.T, filas []filaConfianza) []filaConfianza {
				for indice := range filas {
					filas[indice].configuracionEstado = "revocada"
					filas[indice].configuracionRevocadaEn = pgtype.Timestamptz{
						Time: ahora.Add(-time.Minute), Valid: true,
					}
				}
				return filas
			},
		},
		{
			"raiz_activa_expirada",
			func(_ *testing.T, filas []filaConfianza) []filaConfianza {
				for indice := range filas {
					filas[indice].raizValidaHasta = ahora
				}
				return recalcularHuellaConfiguracion(filas)
			},
		},
		{
			"sin_raiz_activa_vigente",
			func(_ *testing.T, filas []filaConfianza) []filaConfianza {
				for indice := range filas {
					filas[indice].raizEstado = string(confianzaatestacion.EstadoClaveAtestacionAutorizacionV2Revocada)
					filas[indice].raizRevocadaEn = pgtype.Timestamptz{
						Time: filas[indice].configuracionPublicadaEn.Add(-time.Minute), Valid: true,
					}
				}
				return recalcularHuellaConfiguracion(filas)
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			filas := caso.mutar(t, filasConfianzaValidas(t, ahora, true))
			tx := transaccionValida(ahora, filas)
			_, err := cargarConfiguracionActual(context.Background(), &iniciadorDoble{tx: tx})
			if !errors.Is(err, ErrCargaConfianzaAtestacionV2NoDisponible) {
				t.Fatalf("datos alterados aceptados: %v", err)
			}
			if tx.confirmaciones != 0 {
				t.Fatal("se confirmo una lectura inconsistente")
			}
		})
	}
}

func TestCargarConfiguracionActualAcotaConjuntoYPropagaErroresDeFilas(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	fila := filasConfianzaValidas(t, ahora, false)[0]
	filasExcesivas := make([]filaConfianza, maximasRaicesConfiables+1)
	for indice := range filasExcesivas {
		filasExcesivas[indice] = fila
	}
	tx := transaccionValida(ahora, filasExcesivas)
	if _, err := cargarConfiguracionActual(context.Background(), &iniciadorDoble{tx: tx}); !errors.Is(err, ErrCargaConfianzaAtestacionV2NoDisponible) {
		t.Fatalf("se aceptaron mas de %d raices: %v", maximasRaicesConfiables, err)
	}

	tx = transaccionValida(ahora, nil)
	if _, err := cargarConfiguracionActual(context.Background(), &iniciadorDoble{tx: tx}); !errors.Is(err, ErrCargaConfianzaAtestacionV2NoDisponible) {
		t.Fatalf("se acepto un conjunto vacio: %v", err)
	}

	tx = transaccionValida(ahora, []filaConfianza{fila})
	tx.filas.err = context.DeadlineExceeded
	if _, err := cargarConfiguracionActual(context.Background(), &iniciadorDoble{tx: tx}); !errors.Is(err, context.DeadlineExceeded) ||
		!errors.Is(err, ErrCargaConfianzaAtestacionV2NoDisponible) {
		t.Fatalf("no se preservo el plazo agotado de filas: %v", err)
	}
}

func TestCargarConfiguracionActualPreservaCancelacionYRollbackIndependiente(t *testing.T) {
	ctxCancelado, cancelarAntes := context.WithCancel(context.Background())
	cancelarAntes()
	iniciador := &iniciadorDoble{}
	if _, err := cargarConfiguracionActual(ctxCancelado, iniciador); !errors.Is(err, context.Canceled) || iniciador.llamadas != 0 {
		t.Fatalf("cancelacion previa no preservada: %v", err)
	}

	iniciador = &iniciadorDoble{err: fmt.Errorf("detalle-sensible: %w", context.DeadlineExceeded)}
	if _, err := cargarConfiguracionActual(context.Background(), iniciador); !errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(err.Error(), "detalle-sensible") {
		t.Fatalf("plazo agotado al iniciar no se preservo de forma opaca: %v", err)
	}

	ahora := time.Now().UTC().Truncate(time.Microsecond)
	ctx, cancelarDurante := context.WithCancel(context.Background())
	tx := transaccionValida(ahora, filasConfianzaValidas(t, ahora, false))
	tx.cancelarConsulta = cancelarDurante
	tx.errConsulta = context.Canceled
	_, err := cargarConfiguracionActual(ctx, &iniciadorDoble{tx: tx})
	if !errors.Is(err, context.Canceled) ||
		!errors.Is(err, ErrCargaConfianzaAtestacionV2NoDisponible) {
		t.Fatalf("cancelacion durante consulta no preservada: %v", err)
	}
	if tx.errorRollbackCtx != nil || !tx.rollbackConPlazo {
		t.Fatal("el rollback reutilizo el contexto cancelado")
	}
}

func TestConstructoresProductivosRechazanPoolNulo(t *testing.T) {
	if _, err := CargarConfiguracionActual(context.Background(), nil); !errors.Is(err, ErrCargaConfianzaAtestacionV2NoDisponible) {
		t.Fatalf("pool nulo aceptado: %v", err)
	}
	if _, err := NuevoServicioActual(context.Background(), nil, nil); !errors.Is(err, ErrCargaConfianzaAtestacionV2NoDisponible) {
		t.Fatalf("pool nulo aceptado por servicio: %v", err)
	}
}

func transaccionValida(ahora time.Time, filas []filaConfianza) *transaccionDoble {
	return &transaccionDoble{
		identidad: identidadTransaccion{
			sesionUsuario:       "vec_confianza_atestacion_v2_login_aislado",
			usuarioActual:       RolLectorAutoridadPostgreSQL,
			instanteConsulta:    ahora,
			sesionPuedeLogin:    true,
			sesionMembresias:    1,
			sesionMiembroLector: true,
			actualSinMembresias: true,
		},
		filas: &filasDoble{filas: filas},
	}
}

func filasConfianzaValidas(t *testing.T, ahora time.Time, incluirRevocada bool) []filaConfianza {
	t.Helper()
	publicadaEn := ahora.Add(-5 * time.Minute).UTC().Truncate(time.Microsecond)
	expiraEn := ahora.Add(30 * time.Minute).UTC().Truncate(time.Microsecond)
	filas := []filaConfianza{
		nuevaFilaConfianza(t, "clave:atestacion:v2:activa", publicadaEn, expiraEn,
			confianzaatestacion.EstadoClaveAtestacionAutorizacionV2Activa, time.Time{}),
	}
	if incluirRevocada {
		filas = append(filas, nuevaFilaConfianza(
			t,
			"clave:atestacion:v2:revocada",
			publicadaEn,
			expiraEn,
			confianzaatestacion.EstadoClaveAtestacionAutorizacionV2Revocada,
			publicadaEn.Add(-time.Minute),
		))
	}
	return recalcularHuellaConfiguracion(filas)
}

func nuevaFilaConfianza(
	t *testing.T,
	claveID string,
	publicadaEn time.Time,
	expiraEn time.Time,
	estado confianzaatestacion.EstadoClaveAtestacionAutorizacionV2,
	revocadaEn time.Time,
) filaConfianza {
	t.Helper()
	publica, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	revocada := pgtype.Timestamptz{}
	if !revocadaEn.IsZero() {
		revocada = pgtype.Timestamptz{Time: revocadaEn, Valid: true}
	}
	return filaConfianza{
		revision:                 "confianza:atestacion:v2:revision:2026-07-17",
		configuracionPublicadaEn: publicadaEn,
		configuracionExpiraEn:    expiraEn,
		configuracionEstado:      "activa",
		claveID:                  claveID,
		algoritmoCOSE:            confianzaatestacion.AlgoritmoCOSEAtestacionAutorizacionV2EdDSA,
		suite:                    confianzaatestacion.SuiteAtestacionAutorizacionV2COSEEdDSA,
		audienciaDespliegue:      "vec-diputacion/pruebas/vec/autorizacion-v2",
		clavePublicaSPKI:         spki,
		huellaClaveSPKISHA256:    huellaBytes(spki),
		raizValidaDesde:          publicadaEn.Add(-time.Hour),
		raizValidaHasta:          expiraEn.Add(time.Hour),
		raizEstado:               string(estado),
		raizRevocadaEn:           revocada,
	}
}

func recalcularHuellaConfiguracion(filas []filaConfianza) []filaConfianza {
	huella := huellaConfiguracionPrueba(filas)
	for indice := range filas {
		filas[indice].huellaConfiguracionSHA256 = huella
	}
	return filas
}

func huellaConfiguracionPrueba(filas []filaConfianza) string {
	if len(filas) == 0 {
		return ""
	}
	type registro struct {
		claveID string
		campos  []string
	}
	registros := make([]registro, 0, len(filas))
	for _, fila := range filas {
		revocadaEn := ""
		if fila.raizRevocadaEn.Valid {
			revocadaEn = fila.raizRevocadaEn.Time.Format(time.RFC3339Nano)
		}
		registros = append(registros, registro{
			claveID: fila.claveID,
			campos: []string{
				fila.claveID,
				confianzaatestacion.AlgoritmoCOSEAtestacionAutorizacionV2EdDSA,
				fila.huellaClaveSPKISHA256,
				confianzaatestacion.SuiteAtestacionAutorizacionV2COSEEdDSA,
				fila.audienciaDespliegue,
				fila.raizEstado,
				fila.raizValidaDesde.Format(time.RFC3339Nano),
				fila.raizValidaHasta.Format(time.RFC3339Nano),
				revocadaEn,
			},
		})
	}
	sort.Slice(registros, func(i, j int) bool {
		return bytes.Compare([]byte(registros[i].claveID), []byte(registros[j].claveID)) < 0
	})
	calculador := sha256.New()
	escribirCampoHuellaPrueba(calculador, "vec.configuracion-confianza-atestacion-autorizacion.v2")
	escribirCampoHuellaPrueba(calculador, filas[0].revision)
	escribirCampoHuellaPrueba(calculador, filas[0].configuracionPublicadaEn.Format(time.RFC3339Nano))
	escribirCampoHuellaPrueba(calculador, filas[0].configuracionExpiraEn.Format(time.RFC3339Nano))
	for _, registro := range registros {
		for _, campo := range registro.campos {
			escribirCampoHuellaPrueba(calculador, campo)
		}
	}
	return hex.EncodeToString(calculador.Sum(nil))
}

func escribirCampoHuellaPrueba(destino hash.Hash, valor string) {
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(len([]byte(valor))))
	_, _ = destino.Write(longitud[:])
	_, _ = destino.Write([]byte(valor))
}
