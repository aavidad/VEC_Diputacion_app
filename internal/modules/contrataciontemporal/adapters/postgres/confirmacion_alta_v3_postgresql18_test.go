package postgres

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// TestConfirmacionAltaV3PostgreSQL18 exige una base desechable PG18 con CT193,
// VEC-AD-3 y la política HMAC de R3B. Los cuatro bundles son salidas reales de
// TestGenerarVectorO205ParaSQL, guardadas en un directorio temporal del runner:
// e2.json (confirmado ANTES de CT193), e3.json, colision.json y concurrente.json.
// colision comparte el ámbito HMAC de e3, pero cambia jornada_minutos y la
// huella de petición; concurrente tiene ámbito y decisión propios. El runner
// instala CT193 una sola vez, después de confirmar e2, y activa este test con
// VEC_CT_E3_PG18=SI, VEC_CT_E3_VECTORES_DIR y ambos DSN VEC_CT_E3_*_DSN.
func TestConfirmacionAltaV3PostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_CT_E3_PG18") != "SI" {
		t.Skip("requiere runner E3 en PostgreSQL 18 desechable")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	runtime := abrirPoolR3B(t, ctx, "VEC_CT_E3_RUNTIME_DSN")
	defer runtime.Close()
	segundo := abrirPoolR3B(t, ctx, "VEC_CT_E3_RUNTIME_DSN")
	defer segundo.Close()
	admin := abrirPoolR3B(t, ctx, "VEC_CT_E3_ADMIN_DSN")
	defer admin.Close()

	var version int
	if err := admin.QueryRow(ctx, `SELECT current_setting('server_version_num')::integer`).Scan(&version); err != nil || version/10000 != 18 {
		t.Fatalf("se exige PostgreSQL 18 desechable: versión=%d error=%v", version, err)
	}
	var instalada bool
	if err := admin.QueryRow(ctx, `SELECT to_regprocedure('vec_contratacion_temporal.necesidad_alta_valida_v3(jsonb)') IS NOT NULL`).Scan(&instalada); err != nil || !instalada {
		t.Fatalf("CT193 no instalada: %v", err)
	}
	dir := os.Getenv("VEC_CT_E3_VECTORES_DIR")
	if dir == "" {
		t.Fatal("falta VEC_CT_E3_VECTORES_DIR con cuatro bundles firmados")
	}
	e2 := cargarVectorAltaE3PG(t, filepath.Join(dir, "e2.json"))
	e3 := cargarVectorAltaE3PG(t, filepath.Join(dir, "e3.json"))
	colision := cargarVectorAltaE3PG(t, filepath.Join(dir, "colision.json"))
	concurrente := cargarVectorAltaE3PG(t, filepath.Join(dir, "concurrente.json"))
	if e2.efecto.Esquema != esquemaEfectoAltaV2 || e3.efecto.Esquema != esquemaEfectoAltaV3 ||
		colision.efecto.Esquema != esquemaEfectoAltaV3 || concurrente.efecto.Esquema != esquemaEfectoAltaV3 {
		t.Fatal("los bundles no representan E2, E3, E3 colisión y E3 concurrente")
	}
	if e3.sellos.Activo.AmbitoHMAC != colision.sellos.Activo.AmbitoHMAC ||
		e3.sellos.Activo.HuellaHMAC == colision.sellos.Activo.HuellaHMAC ||
		e3.efecto.ReservaRef != colision.efecto.ReservaRef ||
		e3.efecto.ExpedienteRef != colision.efecto.ExpedienteRef ||
		e3.efecto.NumeroVisible != colision.efecto.NumeroVisible ||
		e3.efecto.ReciboRef != colision.efecto.ReciboRef ||
		e3.efecto.OrganizacionRef != colision.efecto.OrganizacionRef ||
		e3.efecto.ActorRef != colision.efecto.ActorRef ||
		e3.efecto.PerfilRef != colision.efecto.PerfilRef ||
		!reflect.DeepEqual(e3.efecto.Solicitud, colision.efecto.Solicitud) ||
		bytes.Equal(decodificarPublicoR3B(t, e3.bundle.DecisionB64), decodificarPublicoR3B(t, colision.bundle.DecisionB64)) ||
		concurrente.sellos.Activo.AmbitoHMAC == e3.sellos.Activo.AmbitoHMAC {
		t.Fatal("los vectores no aíslan colisión de campo y concurrencia")
	}
	var necesidadE3, necesidadColision struct {
		Jornada int `json:"jornada_minutos"`
	}
	if err := json.Unmarshal(e3.necesidad, &necesidadE3); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(colision.necesidad, &necesidadColision); err != nil ||
		necesidadE3.Jornada <= 0 || necesidadColision.Jornada <= 0 ||
		necesidadE3.Jornada == necesidadColision.Jornada {
		t.Fatal("la colisión debe cambiar jornada_minutos con firma nueva")
	}

	// El E2 se confirmó antes de instalar CT193: su alta canónica no puede cambiar.
	assertBytesVersionAltaE3PG(t, ctx, admin, e2)
	replayE2, err := confirmarVectorAltaE3PG(ctx, runtime, e2.argumentos())
	if err != nil || replayE2.expedienteRef != e2.efecto.ExpedienteRef {
		t.Fatalf("E2 dejó de recuperar por confirmar_alta_atestada_v3: %+v, %v", replayE2, err)
	}
	assertBytesVersionAltaE3PG(t, ctx, admin, e2)

	resolverCandidaturaAltaE3PG(t, ctx, runtime, e3)
	primero, err := confirmarVectorAltaE3PG(ctx, runtime, e3.argumentos())
	if err != nil || primero.expedienteRef != e3.efecto.ExpedienteRef || primero.version != 1 {
		t.Fatalf("confirmación E3 inválida: %+v, %v", primero, err)
	}
	assertBytesVersionAltaE3PG(t, ctx, admin, e3)
	antes := estadoEfectosR3B(t, ctx, admin)
	// El segundo pool simula un proceso nuevo; debe recuperar el recibo íntegro.
	replay, err := confirmarVectorAltaE3PG(ctx, segundo, e3.argumentos())
	if err != nil || replay != primero || estadoEfectosR3B(t, ctx, admin) != antes {
		t.Fatalf("replay E3 duplicó o alteró el recibo: original=%+v replay=%+v error=%v", primero, replay, err)
	}

	resolutor, err := NuevoResolutorCandidaturaAltaPostgreSQL(segundo)
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolutor.ResolverCandidaturaAlta(ctx, solicitudCandidaturaAltaE3PG(t, colision))
	if !errors.Is(err, ports.ErrClaveIdempotenciaUsada) || estadoEfectosR3B(t, ctx, admin) != antes {
		t.Fatalf("la clave reutilizada no produjo el conflicto 409 del adaptador: %v", err)
	}
	_, err = confirmarVectorAltaE3PG(ctx, segundo, colision.argumentos())
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" || estadoEfectosR3B(t, ctx, admin) != antes {
		t.Fatalf("confirmación directa de clave conflictiva no fue atómica: %v", err)
	}
	assertBytesVersionAltaE3PG(t, ctx, admin, e3)

	resolverCandidaturaAltaE3PG(t, ctx, runtime, concurrente)
	alterada := append([]any(nil), concurrente.argumentos()...)
	alterada[10] = cambiarJornadaAltaE3PG(t, concurrente.alta)
	antes = estadoEfectosR3B(t, ctx, admin)
	_, err = confirmarVectorAltaE3PG(ctx, runtime, alterada)
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" || estadoEfectosR3B(t, ctx, admin) != antes {
		t.Fatalf("digest V3 divergente consumió decisión o escribió historia: %v", err)
	}

	const sesiones = 2
	resultados := make(chan filaConfirmacionAlta, sesiones)
	errores := make(chan error, sesiones)
	var grupo sync.WaitGroup
	for _, pool := range []*pgxpool.Pool{runtime, segundo} {
		grupo.Add(1)
		go func(pool *pgxpool.Pool) {
			defer grupo.Done()
			var fila filaConfirmacionAlta
			var err error
			for intento := 1; intento <= maximoIntentosConfirmarAlta; intento++ {
				fila, err = confirmarVectorAltaE3PG(ctx, pool, concurrente.argumentos())
				if err == nil || !errorPostgreSQLReintentable(err) {
					break
				}
			}
			if err != nil {
				errores <- err
			} else {
				resultados <- fila
			}
		}(pool)
	}
	grupo.Wait()
	close(resultados)
	close(errores)
	for err := range errores {
		t.Errorf("confirmación concurrente: %v", err)
	}
	var primeroConcurrente filaConfirmacionAlta
	cuantos := 0
	for fila := range resultados {
		if cuantos > 0 && fila != primeroConcurrente {
			t.Errorf("recibos concurrentes distintos: %+v / %+v", primeroConcurrente, fila)
		}
		primeroConcurrente = fila
		cuantos++
	}
	if cuantos != sesiones {
		t.Fatalf("solo %d/%d sesiones recibieron confirmación", cuantos, sesiones)
	}
	assertCardinalidadAltaE3PG(t, ctx, admin, concurrente.efecto.ExpedienteRef)
	assertBytesVersionAltaE3PG(t, ctx, admin, concurrente)
}

type vectorAltaE3PG struct {
	bundle    bundlePublicoR3B
	efecto    efectoAltaCanonico
	sellos    sellosAltaCanonicos
	alta      []byte
	necesidad json.RawMessage
}

func cargarVectorAltaE3PG(t *testing.T, ruta string) vectorAltaE3PG {
	t.Helper()
	var bundle bundlePublicoR3B
	leerJSONPublicoR3B(t, ruta, &bundle)
	alta := decodificarPublicoR3B(t, bundle.AltaB64)
	sellos := decodificarPublicoR3B(t, bundle.SellosB64)
	var efecto efectoAltaCanonico
	var par sellosAltaCanonicos
	var bruto struct {
		Solicitud struct {
			Necesidad json.RawMessage `json:"necesidad"`
		} `json:"solicitud"`
	}
	if !json.Valid(alta) || json.Unmarshal(alta, &efecto) != nil ||
		json.Unmarshal(alta, &bruto) != nil || json.Unmarshal(sellos, &par) != nil ||
		efecto.ExpedienteRef == "" || par.Activo.AmbitoHMAC == "" || bundle.DecisionB64 == "" {
		t.Fatalf("bundle firmado incompleto: %s", filepath.Base(ruta))
	}
	for _, codificado := range []string{bundle.CapacidadB64, bundle.DecisionB64,
		bundle.MotivoB64, bundle.ContextoB64, bundle.PayloadB64,
		bundle.COSEB64, bundle.EvidenciaB64, bundle.SPKIB64} {
		if codificado == "" {
			t.Fatalf("material V3 incompleto: %s", filepath.Base(ruta))
		}
		if _, err := base64.StdEncoding.DecodeString(codificado); err != nil {
			t.Fatalf("material V3 no es base64: %s: %v", filepath.Base(ruta), err)
		}
	}
	return vectorAltaE3PG{bundle: bundle, efecto: efecto, sellos: par, alta: alta, necesidad: bruto.Solicitud.Necesidad}
}

func (v vectorAltaE3PG) argumentos() []any {
	dec := func(s string) []byte { b, _ := base64.StdEncoding.DecodeString(s); return b }
	return []any{dec(v.bundle.CapacidadB64), dec(v.bundle.DecisionB64),
		dec(v.bundle.MotivoB64), dec(v.bundle.ContextoB64),
		int64(v.bundle.PersonaVersion), int64(v.bundle.PerfilVersion),
		dec(v.bundle.PayloadB64), dec(v.bundle.COSEB64),
		dec(v.bundle.EvidenciaB64), dec(v.bundle.SPKIB64),
		append([]byte(nil), v.alta...), dec(v.bundle.SellosB64)}
}

func confirmarVectorAltaE3PG(ctx context.Context, pool *pgxpool.Pool, args []any) (filaConfirmacionAlta, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return filaConfirmacionAlta{}, err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT set_config('timezone','UTC',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return filaConfirmacionAlta{}, err
	}
	var fila filaConfirmacionAlta
	err = tx.QueryRow(ctx, consultaConfirmarAltaV3(), args...).Scan(
		&fila.expedienteRef, &fila.numeroVisible, &fila.version,
		&fila.reciboRef, &fila.auditoriaRef, &fila.eventoRef,
		&fila.confirmadaEn, &fila.huellaRecibo)
	if err != nil {
		return filaConfirmacionAlta{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return filaConfirmacionAlta{}, err
	}
	fila.confirmadaEn = fila.confirmadaEn.UTC()
	return fila, nil
}

func resolverCandidaturaAltaE3PG(t *testing.T, ctx context.Context, pool *pgxpool.Pool, v vectorAltaE3PG) {
	t.Helper()
	solicitud := solicitudCandidaturaAltaE3PG(t, v)
	resolutor, err := NuevoResolutorCandidaturaAltaPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = resolutor.ResolverCandidaturaAlta(ctx, solicitud); err != nil {
		t.Fatal(err)
	}
}

func solicitudCandidaturaAltaE3PG(t *testing.T, v vectorAltaE3PG) ports.SolicitudResolverCandidaturaAlta {
	t.Helper()
	ambitos, huellas := coleccionesPublicasR3B(t, v.sellos)
	candidatura, err := ports.NuevaCandidaturaAlta(ports.DatosCandidaturaAlta{
		ReservaRef: v.efecto.ReservaRef,
		Referencias: ports.ReferenciasAlta{ExpedienteRef: v.efecto.ExpedienteRef,
			NumeroVisible: v.efecto.NumeroVisible, ReciboRef: v.efecto.ReciboRef},
		AmbitoIdempotenciaHMAC: v.sellos.Activo.AmbitoHMAC,
		HuellaPeticionHMAC:     v.sellos.Activo.HuellaHMAC,
		OrganizacionRef:        v.efecto.OrganizacionRef, ActorRef: v.efecto.ActorRef,
		PerfilRef: v.efecto.PerfilRef, InstanteEfecto: instantePublicoR3B(t, v.efecto.CreadoEn),
	})
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudResolverCandidaturaAlta(ports.DatosSolicitudResolverCandidaturaAlta{
		AmbitosIdempotenciaHMAC: ambitos, HuellasPeticionHMAC: huellas,
		OrganizacionRef: v.efecto.OrganizacionRef, ActorRef: v.efecto.ActorRef,
		PerfilRef: v.efecto.PerfilRef, Propuesta: candidatura,
	})
	if err != nil {
		t.Fatal(err)
	}
	return solicitud
}

func assertBytesVersionAltaE3PG(t *testing.T, ctx context.Context, admin *pgxpool.Pool, v vectorAltaE3PG) {
	t.Helper()
	var original []byte
	if err := admin.QueryRow(ctx, `SELECT alta_canonica FROM vec_contratacion_temporal.expediente_alta_version WHERE expediente_ref=$1 AND version=1`, v.efecto.ExpedienteRef).Scan(&original); err != nil || !bytes.Equal(original, v.alta) {
		t.Fatalf("bytes E2/E3 alterados para %s: %v", v.efecto.ExpedienteRef, err)
	}
}

func assertCardinalidadAltaE3PG(t *testing.T, ctx context.Context, admin *pgxpool.Pool, expediente string) {
	t.Helper()
	var e, v, a, u, o, c int
	err := admin.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM vec_contratacion_temporal.expediente_alta WHERE expediente_ref=$1),
		(SELECT count(*) FROM vec_contratacion_temporal.expediente_alta_version WHERE expediente_ref=$1),
		(SELECT count(*) FROM vec_contratacion_temporal.actuacion_alta WHERE expediente_ref=$1),
		(SELECT count(*) FROM vec_contratacion_temporal.auditoria_alta WHERE expediente_ref=$1),
		(SELECT count(*) FROM vec_contratacion_temporal.outbox_alta WHERE expediente_ref=$1),
		(SELECT count(*) FROM vec_contratacion_temporal.confirmacion_agregado_alta WHERE expediente_ref=$1)`, expediente).Scan(&e, &v, &a, &u, &o, &c)
	if err != nil || e != 1 || v != 1 || a != 1 || u != 1 || o != 1 || c != 1 {
		t.Fatalf("agregado concurrente duplicado: %d/%d/%d/%d/%d/%d, %v", e, v, a, u, o, c, err)
	}
}

func cambiarJornadaAltaE3PG(t *testing.T, alta []byte) []byte {
	t.Helper()
	marca := []byte(`"jornada_minutos":`)
	inicio := bytes.Index(alta, marca)
	if inicio < 0 || bytes.Count(alta, marca) != 1 {
		t.Fatal("jornada E3 ausente o repetida")
	}
	inicio += len(marca)
	fin := inicio
	for fin < len(alta) && alta[fin] >= '0' && alta[fin] <= '9' {
		fin++
	}
	valor, err := strconv.Atoi(string(alta[inicio:fin]))
	if err != nil || valor <= 1 {
		t.Fatal("jornada E3 no admite mutación válida")
	}
	mutada := bytes.Join([][]byte{alta[:inicio], []byte(strconv.Itoa(valor - 1)), alta[fin:]}, nil)
	if !json.Valid(mutada) || bytes.Equal(mutada, alta) {
		t.Fatal("la mutación de jornada no cambió E3")
	}
	return mutada
}
