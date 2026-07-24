package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type poolContextoActorDoble struct {
	transacciones []*txContextoActorDoble
	opciones      []pgx.TxOptions
	llamadas      int
}

func (p *poolContextoActorDoble) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.opciones = append(p.opciones, o)
	if p.llamadas >= len(p.transacciones) {
		return nil, errors.New("sin transaccion de prueba")
	}
	tx := p.transacciones[p.llamadas]
	p.llamadas++
	return tx, nil
}

func (*poolContextoActorDoble) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaContextoActorDoble{err: errors.New("no usada")}
}

type txContextoActorDoble struct {
	pgx.Tx
	filas      []pgx.Row
	consultas  []string
	argumentos [][]any
	errCommit  error
	commits    int
	rollbacks  int
}

func (t *txContextoActorDoble) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (t *txContextoActorDoble) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	t.consultas = append(t.consultas, q)
	t.argumentos = append(t.argumentos, append([]any(nil), args...))
	if len(t.filas) == 0 {
		return filaContextoActorDoble{err: errors.New("sin fila de prueba")}
	}
	f := t.filas[0]
	t.filas = t.filas[1:]
	return f
}

func (t *txContextoActorDoble) Commit(context.Context) error {
	t.commits++
	return t.errCommit
}

func (t *txContextoActorDoble) Rollback(context.Context) error {
	t.rollbacks++
	return nil
}

type filaContextoActorDoble struct {
	valores []any
	err     error
}

func (f filaContextoActorDoble) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != len(f.valores) {
		return errors.New("columnas inesperadas")
	}
	for i := range destinos {
		d := reflect.ValueOf(destinos[i])
		v := reflect.ValueOf(f.valores[i])
		if d.Kind() != reflect.Pointer || !v.Type().AssignableTo(d.Elem().Type()) {
			return errors.New("tipo de columna inesperado")
		}
		d.Elem().Set(v)
	}
	return nil
}

func TestGeneradorOperacionContextoActorV2Usa192BitsCSPRNG(t *testing.T) {
	material := bytes.Repeat([]byte{0xa7}, bytesAleatoriosReferenciaContextoActorV2)
	g := nuevoGeneradorOperacionContextoActorV2(bytes.NewReader(material))
	ref, err := g.NuevaReferenciaOperacionContextoActorV2(context.Background())
	if err != nil || !strings.HasPrefix(ref, "oca_") {
		t.Fatalf("referencia no generada: %q %v", ref, err)
	}
	decodificada, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(ref, "oca_"))
	if err != nil || !bytes.Equal(decodificada, material) || len(decodificada)*8 < 144 {
		t.Fatal("la referencia no conserva el material CSPRNG esperado")
	}

	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err = g.NuevaReferenciaOperacionContextoActorV2(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("el generador no respeto la cancelacion")
	}
}

func TestResolutorContextoActorPostgreSQLConfirmaCanonicoEnSerializable(t *testing.T) {
	solicitud, fila := solicitudYFilaContextoActorV2(t)
	tx := &txContextoActorDoble{filas: []pgx.Row{fila}}
	pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{tx}}
	adaptador, err := nuevoResolutorRegistroContextoActorPostgreSQLV2(
		pool, bytes.NewReader(bytes.Repeat([]byte{0x55}, bytesAleatoriosReferenciaContextoActorV2)),
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), solicitud)
	if err != nil || confirmacion.ValidarPara(solicitud) != nil {
		t.Fatalf("confirmacion valida rechazada: %v", err)
	}
	if pool.llamadas != 1 || len(pool.opciones) != 1 ||
		pool.opciones[0].IsoLevel != pgx.Serializable || pool.opciones[0].AccessMode != pgx.ReadWrite ||
		tx.commits != 1 || !strings.Contains(tx.consultas[0], "resolver_y_registrar_contexto_actor_v2") {
		t.Fatal("la operacion no uso su unica transaccion SERIALIZABLE cerrada")
	}
	if tx.argumentos[0][0] != solicitud.OperacionRef || tx.argumentos[0][2] != solicitud.Contexto.Cuenta.CuentaRef ||
		tx.argumentos[0][3] != solicitud.Contexto.PerfilActivoRef {
		t.Fatal("la frontera SQL no recibio referencias exactas")
	}
}

func TestResolutorContextoActorPostgreSQLReconciliaCommitAmbiguoExacto(t *testing.T) {
	solicitud, fila := solicitudYFilaContextoActorV2(t)
	primera := &txContextoActorDoble{filas: []pgx.Row{fila}, errCommit: errors.New("commit ambiguo")}
	segunda := &txContextoActorDoble{filas: []pgx.Row{filaContextoActorDoble{valores: append([]any(nil), fila.valores...)}}}
	pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{primera, segunda}}
	adaptador, _ := nuevoResolutorRegistroContextoActorPostgreSQLV2(
		pool, bytes.NewReader(bytes.Repeat([]byte{0x55}, bytesAleatoriosReferenciaContextoActorV2)),
	)
	confirmacion, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), solicitud)
	if err != nil || confirmacion.ValidarPara(solicitud) != nil || pool.llamadas != 2 {
		t.Fatal("un COMMIT ambiguo confirmado exactamente no se reconcilio")
	}
	if !strings.Contains(segunda.consultas[0], "reconciliar_contexto_actor_v2") ||
		primera.argumentos[0][0] != segunda.argumentos[0][0] ||
		primera.argumentos[0][1] != segunda.argumentos[0][1] {
		t.Fatal("la reconciliacion no conservo operacion y recibo CSPRNG")
	}
	if len(pool.opciones) != 2 || pool.opciones[0].IsoLevel != pgx.Serializable ||
		pool.opciones[1].IsoLevel != pgx.ReadCommitted ||
		pool.opciones[1].AccessMode != pgx.ReadWrite {
		t.Fatalf("aislamientos de escritura/reconciliacion inesperados: %#v", pool.opciones)
	}
}

func TestResolutorContextoActorPostgreSQLDeniegaReconciliacionDistinta(t *testing.T) {
	solicitud, fila := solicitudYFilaContextoActorV2(t)
	distinta := append([]any(nil), fila.valores...)
	distinta[3] = strings.Repeat("0", 64)
	pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{
		{filas: []pgx.Row{fila}, errCommit: errors.New("commit ambiguo")},
		{filas: []pgx.Row{filaContextoActorDoble{valores: distinta}}},
	}}
	adaptador, _ := nuevoResolutorRegistroContextoActorPostgreSQLV2(
		pool, bytes.NewReader(bytes.Repeat([]byte{0x55}, bytesAleatoriosReferenciaContextoActorV2)),
	)
	_, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), solicitud)
	if !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) {
		t.Fatal("una reconciliacion distinta no fallo cerrada")
	}
}

func TestResolutorContextoActorPostgreSQLRechazaManifiestoAdulteradoONoAutoritativo(t *testing.T) {
	solicitud, fila := solicitudYFilaContextoActorV2(t)
	casos := []struct {
		nombre string
		mutar  func([]any)
	}{
		{"bytes", func(valores []any) {
			contenido := append(append([]byte(nil), valores[4].([]byte)...), ' ')
			suma := sha256.Sum256(contenido)
			valores[4], valores[5] = contenido, hex.EncodeToString(suma[:])
		}},
		{"huella", func(valores []any) { valores[5] = strings.Repeat("0", 64) }},
		{"version cuenta no eco", func(valores []any) {
			// Solo cambia Cuenta.Version; el JSON y su huella vuelven a ser
			// internamente coherentes, pero dejan de corresponder al canon V2.
			contenido := []byte(strings.Replace(string(valores[4].([]byte)),
				`,"version":7,"procedencia_ref"`, `,"version":8,"procedencia_ref"`, 1))
			suma := sha256.Sum256(contenido)
			valores[4], valores[5] = contenido, hex.EncodeToString(suma[:])
		}},
		{"version no eco", func(valores []any) {
			// Mutar una version conservando JSON canonico y huella exacta.
			contenido := []byte(strings.Replace(string(valores[4].([]byte)),
				`,"version":4,"procedencia_ref"`, `,"version":99,"procedencia_ref"`, 1))
			suma := sha256.Sum256(contenido)
			valores[4], valores[5] = contenido, hex.EncodeToString(suma[:])
		}},
		{"autoridad", func(valores []any) {
			contenido := []byte(strings.ReplaceAll(string(valores[4].([]byte)),
				string(domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1),
				string(domain.AutoridadProcedenciaContextoActorNoAutoritativaV1)))
			suma := sha256.Sum256(contenido)
			valores[4], valores[5] = contenido, hex.EncodeToString(suma[:])
			valores[6] = string(domain.AutoridadProcedenciaContextoActorNoAutoritativaV1)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			valores := append([]any(nil), fila.valores...)
			caso.mutar(valores)
			pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{{
				filas: []pgx.Row{filaContextoActorDoble{valores: valores}},
			}}}
			adaptador, _ := nuevoResolutorRegistroContextoActorPostgreSQLV2(
				pool, bytes.NewReader(bytes.Repeat([]byte{0x55}, bytesAleatoriosReferenciaContextoActorV2)),
			)
			if _, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), solicitud); !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) {
				t.Fatalf("manifiesto inseguro aceptado: %v", err)
			}
		})
	}
}

func TestResolutorContextoActorPostgreSQLReintentaMismaOperacionTrasAusencia(t *testing.T) {
	solicitud, fila := solicitudYFilaContextoActorV2(t)
	primera := &txContextoActorDoble{filas: []pgx.Row{fila}, errCommit: errors.New("rollback confirmado despues")}
	ausente := &txContextoActorDoble{filas: []pgx.Row{filaContextoActorDoble{err: pgx.ErrNoRows}}}
	reintento := &txContextoActorDoble{filas: []pgx.Row{filaContextoActorDoble{valores: append([]any(nil), fila.valores...)}}}
	pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{primera, ausente, reintento}}
	adaptador, _ := nuevoResolutorRegistroContextoActorPostgreSQLV2(
		pool, bytes.NewReader(bytes.Repeat([]byte{0x55}, bytesAleatoriosReferenciaContextoActorV2)),
	)
	if _, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), solicitud); err != nil {
		t.Fatalf("el reintento seguro no se completo: %v", err)
	}
	if pool.llamadas != 3 || primera.argumentos[0][0] != reintento.argumentos[0][0] ||
		primera.argumentos[0][1] != reintento.argumentos[0][1] {
		t.Fatal("el reintento cambio la identidad de la invocacion")
	}
	if len(primera.consultas) != 1 || len(ausente.consultas) != 1 || len(reintento.consultas) != 1 ||
		!strings.Contains(primera.consultas[0], "resolver_y_registrar") ||
		!strings.Contains(ausente.consultas[0], "reconciliar") ||
		!strings.Contains(reintento.consultas[0], "resolver_y_registrar") {
		t.Fatal("se esperaba un intento, una reconciliacion y un unico reintento")
	}
}

func TestResolutorContextoActorPostgreSQLNoHaceSegundoReintento(t *testing.T) {
	solicitud, _ := solicitudYFilaContextoActorV2(t)
	conflicto := func() *txContextoActorDoble {
		return &txContextoActorDoble{filas: []pgx.Row{filaContextoActorDoble{
			err: &pgconn.PgError{Code: "40001", Message: "serializacion"},
		}}}
	}
	pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{
		conflicto(), conflicto(), conflicto(),
	}}
	adaptador, _ := nuevoResolutorRegistroContextoActorPostgreSQLV2(
		pool, bytes.NewReader(bytes.Repeat([]byte{0x55}, bytesAleatoriosReferenciaContextoActorV2)),
	)
	if _, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), solicitud); !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) || pool.llamadas != 2 {
		t.Fatalf("numero de intentos inesperado: llamadas=%d err=%v", pool.llamadas, err)
	}
}

func TestResolutorContextoActorPostgreSQLReintentaCarreraIdempotente(t *testing.T) {
	solicitud, fila := solicitudYFilaContextoActorV2(t)
	conflicto := &txContextoActorDoble{filas: []pgx.Row{filaContextoActorDoble{
		err: &pgconn.PgError{Code: "23505", Message: "colision concurrente"},
	}}}
	relectura := &txContextoActorDoble{filas: []pgx.Row{fila}}
	pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{conflicto, relectura}}
	adaptador, _ := nuevoResolutorRegistroContextoActorPostgreSQLV2(
		pool, bytes.NewReader(bytes.Repeat([]byte{0x55}, bytesAleatoriosReferenciaContextoActorV2)),
	)
	if _, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), solicitud); err != nil {
		t.Fatalf("la carrera idempotente no releyo en nueva transaccion: %v", err)
	}
	if pool.llamadas != 2 || conflicto.argumentos[0][0] != relectura.argumentos[0][0] ||
		conflicto.argumentos[0][1] != relectura.argumentos[0][1] {
		t.Fatal("el retry concurrente cambio oca_ o rca_")
	}
}

func TestResolutorContextoActorPostgreSQLRechazaTypedNil(t *testing.T) {
	var pool *poolContextoActorDoble
	var interfaz iniciadorContextoActorPostgreSQL = pool
	if _, err := nuevoResolutorRegistroContextoActorPostgreSQLV2(interfaz, bytes.NewReader(make([]byte, 24))); !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) {
		t.Fatal("un pool typed nil fue aceptado")
	}
	var lector *bytes.Reader
	if _, err := nuevoResolutorRegistroContextoActorPostgreSQLV2(&poolContextoActorDoble{}, lector); !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) {
		t.Fatal("un CSPRNG typed nil fue aceptado")
	}
}

func solicitudYFilaContextoActorV2(t *testing.T) (ports.SolicitudResolucionRegistroContextoActorV2, filaContextoActorDoble) {
	t.Helper()
	ref := func(prefijo, relleno string) string { return prefijo + strings.Repeat(relleno, 24) }
	desde := time.Date(2026, 7, 21, 10, 0, 0, 123000, time.UTC)
	hasta := desde.Add(time.Hour)
	resuelto := desde.Add(time.Second)
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: ref("cta_", "a"), Metodo: domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh,
	}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef: ref("vca_", "b"), VinculoVersion: 3,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: 7,
		PersonaRef: ref("per_", "c"), PersonaVersion: 4,
		PerfilActivoRef: ref("prf_", "d"), PerfilVersion: 5,
		Estado: domain.EstadoVinculoContextoActorActivo, VigenteDesde: desde, VigenteHasta: hasta,
		Vinculos: []domain.VinculoReferenciaContextoActor{{
			VinculoRef: ref("vin_", "e"), Version: 6,
			Tipo: domain.TipoReferenciaContextoActorCandidato, Referencia: ref("can_", "f"),
			Estado: domain.EstadoVinculoContextoActorActivo, VigenteDesde: desde, VigenteHasta: hasta,
		}},
	}
	contexto, err := domain.NuevoContextoActor(cuenta, instantanea, resuelto)
	if err != nil {
		t.Fatal(err)
	}
	representacion, err := contexto.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(representacion)
	acreditacion := func(version uint64) domain.AcreditacionProcedenciaComponenteContextoActorV1 {
		return domain.AcreditacionProcedenciaComponenteContextoActorV1{
			ProcedenciaRef: ref("prc_", "p"), ProcedenciaVersion: version,
			ProcedenciaHuellaSHA256: strings.Repeat("1", 64),
			ProcedenciaAutoridad:    domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		}
	}
	manifiesto := domain.ManifiestoProcedenciaContextoActorV1{
		Esquema:           domain.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: domain.ProcedenciaCuentaContextoActorV1{
			CuentaRef: cuenta.CuentaRef, Version: instantanea.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion(1),
		},
		Persona: domain.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantanea.PersonaRef, Version: instantanea.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion(1),
		},
		Perfil: domain.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantanea.PerfilActivoRef, Version: instantanea.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion(1),
		},
		Contexto: domain.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantanea.VinculoRef, Version: instantanea.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion(1),
		},
		Vinculos: []domain.ProcedenciaVinculoReferenciaContextoActorV1{{
			VinculoRef: instantanea.Vinculos[0].VinculoRef,
			Version:    instantanea.Vinculos[0].Version, Tipo: instantanea.Vinculos[0].Tipo,
			Referencia: instantanea.Vinculos[0].Referencia,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion(1),
		}},
	}
	representacionManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	sumaManifiesto := sha256.Sum256(representacionManifiesto)
	solicitud := ports.SolicitudResolucionRegistroContextoActorV2{
		OperacionRef: ref("oca_", "g"),
		Contexto:     domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: instantanea.PerfilActivoRef},
		SolicitadoEn: desde.Add(500 * time.Millisecond),
	}
	return solicitud, filaContextoActorDoble{valores: []any{
		solicitud.OperacionRef, "rca_" + base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x55}, 24)),
		representacion, hex.EncodeToString(suma[:]), representacionManifiesto,
		hex.EncodeToString(sumaManifiesto[:]), string(domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1),
		resuelto,
	}}
}
