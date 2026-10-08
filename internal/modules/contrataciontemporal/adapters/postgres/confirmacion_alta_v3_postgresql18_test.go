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

// Cada ejecución consume una fase recién emitida del runner desechable.
// La capacidad VEC-AD-3 caduca a los cinco segundos: no se preparan cuatro
// bundles por adelantado ni se confunde un replay con otra confirmación.
func TestConfirmacionAltaV3PostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_CT_E3_PG18") != "SI" {
		t.Skip("requiere runner E3 en PostgreSQL 18 desechable")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 25*time.Second)
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
	var fixture bool
	if err := admin.QueryRow(ctx, `SELECT current_database()='postgres'
		AND to_regclass('public.ct193_e3_desechable') IS NOT NULL
		AND to_regclass('public.vectores_o2_05') IS NOT NULL`).Scan(&fixture); err != nil || !fixture {
		t.Fatalf("la conexión no apunta al fixture PG18 desechable: %v", err)
	}
	if err := admin.QueryRow(ctx, `SELECT count(*)=1 AND bool_and(marca='SI') FROM public.ct193_e3_desechable`).Scan(&fixture); err != nil || !fixture {
		t.Fatalf("marcador PG18 desechable inválido: %v", err)
	}
	dir := os.Getenv("VEC_CT_E3_VECTORES_DIR")
	if dir == "" {
		t.Fatal("falta VEC_CT_E3_VECTORES_DIR con bundle de fase")
	}
	fase := os.Getenv("VEC_CT_E3_FASE")
	var instalada bool
	if err := admin.QueryRow(ctx, `SELECT to_regprocedure('vec_contratacion_temporal.necesidad_alta_valida_v3(jsonb)') IS NOT NULL`).Scan(&instalada); err != nil {
		t.Fatal(err)
	}
	if instalada != (fase != "e2_pre") {
		t.Fatalf("preimagen CT193 incompatible con fase %q", fase)
	}
	vector := func(nombre string) vectorAltaE3PG {
		return cargarVectorAltaE3PG(t, filepath.Join(dir, nombre+".json"))
	}
	switch fase {
	case "e2_pre":
		e2 := vector("e2")
		if e2.efecto.Esquema != esquemaEfectoAltaV2 {
			t.Fatal("la preimagen debe ser E2")
		}
		resolverCandidaturaAltaE3PG(t, ctx, runtime, e2)
		fila, err := confirmarVectorAltaE3PG(ctx, runtime, e2.argumentos())
		if err != nil || fila.expedienteRef != e2.efecto.ExpedienteRef {
			t.Fatalf("E2 previa a CT193 no confirmada: %+v, %v", fila, err)
		}
		assertBytesVersionAltaE3PG(t, ctx, admin, e2)
		assertCardinalidadAltaE3PG(t, ctx, admin, e2.efecto.ExpedienteRef)
		guardarReciboAltaE3PG(t, "VEC_CT_E3_RECIBO_E2", fila)
	case "e2_post":
		e2 := vector("e2")
		if e2.efecto.Esquema != esquemaEfectoAltaV2 {
			t.Fatal("el replay debe conservar E2")
		}
		assertBytesVersionAltaE3PG(t, ctx, admin, e2)
		antes := estadoEfectosR3B(t, ctx, admin)
		replay, err := confirmarVectorAltaE3PG(ctx, segundo, e2.argumentos())
		if err != nil || replay != leerReciboAltaE3PG(t, "VEC_CT_E3_RECIBO_E2") || estadoEfectosR3B(t, ctx, admin) != antes {
			t.Fatalf("E2 cambió recibo o historia tras CT193: %+v, %v", replay, err)
		}
		assertBytesVersionAltaE3PG(t, ctx, admin, e2)
	case "e3":
		probarAltaE3PG(t, ctx, runtime, segundo, admin, vector("e3"), false)
	case "e3_replay":
		probarReplayReinicioAltaE3PG(t, ctx, runtime, admin, vector("e3"))
	case "abierto":
		probarAltaE3PG(t, ctx, runtime, segundo, admin, vector("abierto"), true)
	case "colision":
		probarColisionAltaE3PG(t, ctx, segundo, admin, vector("e3"), vector("colision"))
	case "concurrente":
		probarConcurrenciaAltaE3PG(t, ctx, runtime, segundo, admin, vector("concurrente"))
	default:
		t.Fatalf("fase E3 no reconocida: %q", fase)
	}
}

func probarAltaE3PG(t *testing.T, ctx context.Context, runtime, segundo, admin *pgxpool.Pool, v vectorAltaE3PG, abierto bool) {
	t.Helper()
	if v.efecto.Esquema != esquemaEfectoAltaV3 {
		t.Fatal("alta E3 sin esquema V3")
	}
	if !abierto {
		exigirNumeroPersonasE3PG(t, v)
	}
	var forma struct {
		Solicitud struct {
			Periodo map[string]json.RawMessage `json:"periodo"`
		} `json:"solicitud"`
	}
	if err := json.Unmarshal(v.alta, &forma); err != nil {
		t.Fatal(err)
	}
	_, tieneFin := forma.Solicitud.Periodo["fin"]
	_, tieneCausa := forma.Solicitud.Periodo["causa_fin"]
	_, tienePolitica := forma.Solicitud.Periodo["politica_fin"]
	if abierto && (tieneFin || !tieneCausa || !tienePolitica || v.efecto.Solicitud.MotivoClave != "sustitucion") {
		t.Fatal("el vector abierto debe conservar causa_fin y politica_fin sin fecha fin")
	}
	if !abierto && (!tieneFin || tieneCausa) {
		t.Fatal("el vector de fin cerrado debe conservar fecha fin")
	}
	resolverCandidaturaAltaE3PG(t, ctx, runtime, v)
	primero, err := confirmarVectorAltaE3PG(ctx, runtime, v.argumentos())
	if err != nil || primero.expedienteRef != v.efecto.ExpedienteRef || primero.version != 1 {
		t.Fatalf("confirmación E3 inválida: %+v, %v", primero, err)
	}
	assertBytesVersionAltaE3PG(t, ctx, admin, v)
	antes := estadoEfectosR3B(t, ctx, admin)
	replay, err := confirmarVectorAltaE3PG(ctx, segundo, v.argumentos())
	if err != nil || replay != primero || estadoEfectosR3B(t, ctx, admin) != antes {
		t.Fatalf("replay E3 duplicó o alteró recibo: %+v/%+v, %v", primero, replay, err)
	}
	assertCardinalidadAltaE3PG(t, ctx, admin, v.efecto.ExpedienteRef)
	if !abierto {
		guardarReciboAltaE3PG(t, "VEC_CT_E3_RECIBO_E3", primero)
		guardarInicioPGAltaE3PG(t, ctx, admin)
	}
}

func probarReplayReinicioAltaE3PG(t *testing.T, ctx context.Context, runtime, admin *pgxpool.Pool, v vectorAltaE3PG) {
	t.Helper()
	exigirNumeroPersonasE3PG(t, v)
	var inicio time.Time
	if err := admin.QueryRow(ctx, `SELECT pg_postmaster_start_time()`).Scan(&inicio); err != nil {
		t.Fatal(err)
	}
	var anterior time.Time
	leerJSONPrivadoAltaE3PG(t, "VEC_CT_E3_INICIO_PG", &anterior)
	if !inicio.After(anterior) {
		t.Fatalf("PostgreSQL no se reinició: inicio=%s anterior=%s", inicio, anterior)
	}
	assertBytesVersionAltaE3PG(t, ctx, admin, v)
	antes := estadoEfectosR3B(t, ctx, admin)
	replay, err := confirmarVectorAltaE3PG(ctx, runtime, v.argumentos())
	if err != nil || replay != leerReciboAltaE3PG(t, "VEC_CT_E3_RECIBO_E3") || estadoEfectosR3B(t, ctx, admin) != antes {
		t.Fatalf("replay E3 tras reinicio alteró recibo o historia: %+v, %v", replay, err)
	}
	var necesidad struct {
		CatalogoInstantanea string `json:"catalogo_instantanea"`
	}
	if err := json.Unmarshal(v.necesidad, &necesidad); err != nil {
		t.Fatal(err)
	}
	esperada := decodificarPublicoR3B(t, necesidad.CatalogoInstantanea)
	tx, err := runtime.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	var estado string
	var instantanea []byte
	err = tx.QueryRow(ctx, `SELECT estado,instantanea
		FROM vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3($1,$2,$3,$4)`,
		v.sellos.Activo.AmbitoHMAC, v.efecto.OrganizacionRef,
		v.efecto.ActorRef, v.efecto.PerfilRef).Scan(&estado, &instantanea)
	if err != nil || estado != "confirmada_v3" || !bytes.Equal(instantanea, esperada) {
		t.Fatalf("instantánea E3 no sobrevivió al reinicio: estado=%s error=%v", estado, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	assertCardinalidadAltaE3PG(t, ctx, admin, v.efecto.ExpedienteRef)
}

func probarColisionAltaE3PG(t *testing.T, ctx context.Context, segundo, admin *pgxpool.Pool, e3, colision vectorAltaE3PG) {
	t.Helper()
	exigirNumeroPersonasE3PG(t, e3)
	exigirNumeroPersonasE3PG(t, colision)
	if e3.efecto.Esquema != esquemaEfectoAltaV3 || colision.efecto.Esquema != esquemaEfectoAltaV3 {
		t.Fatal("la colisión requiere dos altas E3")
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
		bytes.Equal(decodificarPublicoR3B(t, e3.bundle.DecisionB64), decodificarPublicoR3B(t, colision.bundle.DecisionB64)) {
		t.Fatal("los vectores no aíslan la colisión de campo")
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
	assertBytesVersionAltaE3PG(t, ctx, admin, e3)
	antes := estadoEfectosR3B(t, ctx, admin)
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
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" || estadoEfectosR3B(t, ctx, admin) != antes {
		t.Fatalf("confirmación directa sin candidatura exacta no fue denegada atómicamente: %v", err)
	}
	assertBytesVersionAltaE3PG(t, ctx, admin, e3)
}

func probarConcurrenciaAltaE3PG(t *testing.T, ctx context.Context, runtime, segundo, admin *pgxpool.Pool, concurrente vectorAltaE3PG) {
	t.Helper()
	exigirNumeroPersonasE3PG(t, concurrente)
	if concurrente.efecto.Esquema != esquemaEfectoAltaV3 {
		t.Fatal("la carrera requiere un alta E3")
	}
	resolverCandidaturaAltaE3PG(t, ctx, runtime, concurrente)
	alterada := append([]any(nil), concurrente.argumentos()...)
	alterada[10] = cambiarJornadaAltaE3PG(t, concurrente.alta)
	antes := estadoEfectosR3B(t, ctx, admin)
	_, err := confirmarVectorAltaE3PG(ctx, runtime, alterada)
	var pgErr *pgconn.PgError
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

func guardarReciboAltaE3PG(t *testing.T, variable string, fila filaConfirmacionAlta) {
	t.Helper()
	datos, err := json.Marshal([]any{fila.expedienteRef, fila.numeroVisible, fila.version,
		fila.reciboRef, fila.auditoriaRef, fila.eventoRef, fila.confirmadaEn, fila.huellaRecibo})
	if err != nil {
		t.Fatal(err)
	}
	guardarBytesPrivadosAltaE3PG(t, variable, datos)
}

func leerReciboAltaE3PG(t *testing.T, variable string) filaConfirmacionAlta {
	t.Helper()
	var valores []json.RawMessage
	leerJSONPrivadoAltaE3PG(t, variable, &valores)
	if len(valores) != 8 {
		t.Fatal("recibo previo inválido")
	}
	var fila filaConfirmacionAlta
	campos := []any{&fila.expedienteRef, &fila.numeroVisible, &fila.version,
		&fila.reciboRef, &fila.auditoriaRef, &fila.eventoRef, &fila.confirmadaEn, &fila.huellaRecibo}
	for i, campo := range campos {
		if err := json.Unmarshal(valores[i], campo); err != nil {
			t.Fatal(err)
		}
	}
	return fila
}

func guardarInicioPGAltaE3PG(t *testing.T, ctx context.Context, admin *pgxpool.Pool) {
	t.Helper()
	var inicio time.Time
	if err := admin.QueryRow(ctx, `SELECT pg_postmaster_start_time()`).Scan(&inicio); err != nil {
		t.Fatal(err)
	}
	datos, err := json.Marshal(inicio)
	if err != nil {
		t.Fatal(err)
	}
	guardarBytesPrivadosAltaE3PG(t, "VEC_CT_E3_INICIO_PG", datos)
}

func guardarBytesPrivadosAltaE3PG(t *testing.T, variable string, datos []byte) {
	t.Helper()
	ruta := os.Getenv(variable)
	if ruta == "" {
		t.Fatalf("falta %s en scratch privado", variable)
	}
	archivo, err := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := archivo.Write(datos); err != nil {
		archivo.Close()
		t.Fatal(err)
	}
	if err := archivo.Close(); err != nil {
		t.Fatal(err)
	}
}

func leerJSONPrivadoAltaE3PG(t *testing.T, variable string, destino any) {
	t.Helper()
	ruta := os.Getenv(variable)
	if ruta == "" {
		t.Fatalf("falta %s en scratch privado", variable)
	}
	datos, err := os.ReadFile(ruta)
	if err != nil || json.Unmarshal(datos, destino) != nil {
		t.Fatalf("dato previo %s ausente o inválido: %v", variable, err)
	}
}

type vectorAltaE3PG struct {
	bundle    bundlePublicoR3B
	efecto    efectoAltaCanonico
	sellos    sellosAltaCanonicos
	alta      []byte
	necesidad json.RawMessage
}

func exigirNumeroPersonasE3PG(t *testing.T, v vectorAltaE3PG) {
	t.Helper()
	var necesidad struct {
		CatalogoVersion     uint64            `json:"catalogo_version"`
		Campos              map[string]string `json:"campos"`
		CatalogoInstantanea string            `json:"catalogo_instantanea"`
	}
	if err := json.Unmarshal(v.necesidad, &necesidad); err != nil ||
		necesidad.CatalogoVersion != 2 || necesidad.Campos["numero_personas"] != "2" ||
		bytes.Count(v.alta, []byte(`"numero_personas":"2"`)) != 1 {
		t.Fatalf("E3 no selló número de personas dentro del efecto: %v", err)
	}
	var catalogo struct {
		Version uint64 `json:"version"`
		Causas  []struct {
			Clave              string   `json:"clave"`
			CamposPermitidos   []string `json:"campos_permitidos"`
			CamposObligatorios []string `json:"campos_obligatorios"`
		} `json:"causas"`
	}
	if err := json.Unmarshal(decodificarPublicoR3B(t, necesidad.CatalogoInstantanea), &catalogo); err != nil ||
		catalogo.Version != 2 || len(catalogo.Causas) != 1 ||
		catalogo.Causas[0].Clave != "acumulacion_tareas" {
		t.Fatalf("instantánea E3 nueva no corresponde al catálogo de necesidad: %v", err)
	}
	permitido, obligatorio := false, false
	for _, campo := range catalogo.Causas[0].CamposPermitidos {
		permitido = permitido || campo == "numero_personas"
	}
	for _, campo := range catalogo.Causas[0].CamposObligatorios {
		obligatorio = obligatorio || campo == "numero_personas"
	}
	if !permitido || !obligatorio {
		t.Fatal("el catálogo sellado no exige número de personas")
	}
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
