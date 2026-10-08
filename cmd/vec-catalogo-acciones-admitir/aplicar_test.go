package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type filaAdmisionPrueba struct {
	b   []byte
	err error
}

func (f filaAdmisionPrueba) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	*dest[0].(*[]byte) = append([]byte(nil), f.b...)
	return nil
}

type txAdmisionPrueba struct {
	respuesta []byte
	filErr    error
	commitErr error
	commits   int
	rollbacks int
}

func (t *txAdmisionPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *txAdmisionPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaAdmisionPrueba{b: t.respuesta, err: t.filErr}
}
func (t *txAdmisionPrueba) Commit(context.Context) error {
	t.commits++
	return t.commitErr
}
func (t *txAdmisionPrueba) Rollback(context.Context) error {
	t.rollbacks++
	return nil
}

func respuestaAdmisionPrueba(t *testing.T) (planAdmision, string, map[string]any) {
	t.Helper()
	p := planAdmision{OperacionRef: "caa_" + strings.Repeat("a", 22), CatalogoRef: "catalogo:sintetico",
		CatalogoVersion: "1", CatalogoSHA256: strings.Repeat("a", 64), PaqueteRef: "paquete:sintetico",
		PaqueteVersion: "1", PaqueteSHA256: strings.Repeat("b", 64), AprobacionRef: "aprobacion:sintetica",
		AprobacionSHA256: strings.Repeat("c", 64)}
	planSHA := strings.Repeat("d", 64)
	confirmado := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)
	recibo := map[string]any{
		"esquema": "vec.admin.catalogo-acciones.recibo.v1", "operacion_ref": p.OperacionRef,
		"plan_sha256": planSHA, "catalogo_ref": p.CatalogoRef,
		"catalogo_version": p.CatalogoVersion, "catalogo_sha256": p.CatalogoSHA256,
		"paquete_ref": p.PaqueteRef, "paquete_version": 1, "paquete_sha256": p.PaqueteSHA256,
		"censo_sha256": strings.Repeat("e", 64), "aprobacion_ref": p.AprobacionRef,
		"aprobacion_sha256": p.AprobacionSHA256, "aprobador_ref": "actor:sintetico",
		"auditoria_ref": "aud_v3_caa_" + strings.Repeat("a", 32), "auditoria_secuencia": 1,
		"auditoria_huella_sha256": strings.Repeat("f", 64), "confirmado_en": confirmado.Format(time.RFC3339Nano),
	}
	respuesta := map[string]any{"estado": "permitido", "codigo": nil, "recibo": recibo, "replay": false,
		"auditoria_intento": map[string]any{
			"auditoria_ref": "aud_v3_caai_" + strings.Repeat("b", 32), "secuencia": 2,
			"huella_sha256": strings.Repeat("d", 64), "correlacion_ref": "correlacion_" + strings.Repeat("c", 32),
			"registrada_en": confirmado.Add(time.Second).Format(time.RFC3339Nano)}}
	return p, planSHA, respuesta
}

func TestAdmisionNoConfirmaRespuestaPermitidaSinReciboLigado(t *testing.T) {
	p, sha, base := respuestaAdmisionPrueba(t)
	casos := []struct {
		nombre string
		mutar  func(map[string]any)
		commit bool
	}{
		{"recibo exacto", func(map[string]any) {}, true},
		{"recibo ausente", func(x map[string]any) { x["recibo"] = nil }, false},
		{"recibo vacio", func(x map[string]any) { x["recibo"] = map[string]any{} }, false},
		{"recibo malformado", func(x map[string]any) { x["recibo"] = "sin recibo" }, false},
		{"otra operacion", func(x map[string]any) {
			x["recibo"].(map[string]any)["operacion_ref"] = "caa_" + strings.Repeat("z", 22)
		}, false},
		{"otro plan", func(x map[string]any) { x["recibo"].(map[string]any)["plan_sha256"] = strings.Repeat("0", 64) }, false},
		{"otro catalogo", func(x map[string]any) { x["recibo"].(map[string]any)["catalogo_sha256"] = strings.Repeat("0", 64) }, false},
		{"fecha futura", func(x map[string]any) {
			x["recibo"].(map[string]any)["confirmado_en"] = time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond).Format(time.RFC3339Nano)
		}, false},
		{"intento sin auditoria", func(x map[string]any) { x["auditoria_intento"] = nil }, false},
		{"intento anterior al efecto", func(x map[string]any) { x["auditoria_intento"].(map[string]any)["secuencia"] = 1 }, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			b, _ := json.Marshal(base)
			var copia map[string]any
			if err := json.Unmarshal(b, &copia); err != nil {
				t.Fatal(err)
			}
			caso.mutar(copia)
			b, _ = json.Marshal(copia)
			tx := &txAdmisionPrueba{respuesta: b}
			r, err := aplicarPlanEnTransaccion(context.Background(), tx, []byte(`{}`), sha, p)
			if caso.commit && (err != nil || r.Estado != "permitido" || tx.commits != 1) {
				t.Fatalf("recibo válido no confirmado: %v, commits=%d", err, tx.commits)
			}
			if !caso.commit && (!errors.Is(err, errRespuestaAdmision) || tx.commits != 0 || tx.rollbacks != 1) {
				t.Fatalf("respuesta inválida confirmó efecto: %v, commits=%d, rollbacks=%d", err, tx.commits, tx.rollbacks)
			}
		})
	}
}

func TestAdmisionDenegadaAuditaYCommitAmbiguoNoSeDaPorExito(t *testing.T) {
	p, sha, base := respuestaAdmisionPrueba(t)
	base["estado"], base["codigo"], base["recibo"] = "denegado", "catalogo_acciones_rechazado", nil
	b, _ := json.Marshal(base)
	tx := &txAdmisionPrueba{respuesta: b}
	r, err := aplicarPlanEnTransaccion(context.Background(), tx, []byte(`{}`), sha, p)
	if err != nil || r.Estado != "denegado" || tx.commits != 1 {
		t.Fatalf("denegación auditada no confirmada: %v, commits=%d", err, tx.commits)
	}
	base["recibo"] = map[string]any{"operacion_ref": p.OperacionRef}
	bConRecibo, _ := json.Marshal(base)
	tx = &txAdmisionPrueba{respuesta: bConRecibo}
	if _, err := aplicarPlanEnTransaccion(context.Background(), tx, []byte(`{}`), sha, p); !errors.Is(err, errRespuestaAdmision) || tx.commits != 0 {
		t.Fatalf("denegación con recibo confirmó efecto: %v, commits=%d", err, tx.commits)
	}
	tx = &txAdmisionPrueba{respuesta: b, commitErr: errors.New("commit incierto")}
	if _, err := aplicarPlanEnTransaccion(context.Background(), tx, []byte(`{}`), sha, p); !errors.Is(err, errCommitAdmision) || tx.commits != 1 {
		t.Fatalf("commit ambiguo presentado como éxito: %v", err)
	}
	tx = &txAdmisionPrueba{filErr: errors.New("sin respuesta")}
	if _, err := aplicarPlanEnTransaccion(context.Background(), tx, []byte(`{}`), sha, p); err == nil || tx.commits != 0 {
		t.Fatalf("sin respuesta confirmó efecto: %v", err)
	}
}

func TestFechaUTCConservaCausaDeParseo(t *testing.T) {
	err := fechaUTC([]byte(`"fecha-invalida"`), time.Now().UTC())
	var parseo *time.ParseError
	if !errors.Is(err, errRespuestaAdmision) || !errors.As(err, &parseo) {
		t.Fatalf("se perdió la causa temporal: %v", err)
	}
}

func TestAdmisionPropagaCausaTemporalSinConfirmar(t *testing.T) {
	p, sha, respuesta := respuestaAdmisionPrueba(t)
	respuesta["recibo"].(map[string]any)["confirmado_en"] = "fecha-invalida"
	b, err := json.Marshal(respuesta)
	if err != nil {
		t.Fatal(err)
	}
	tx := &txAdmisionPrueba{respuesta: b}
	_, err = aplicarPlanEnTransaccion(context.Background(), tx, []byte(`{}`), sha, p)
	var parseo *time.ParseError
	if !errors.Is(err, errRespuestaAdmision) || !errors.As(err, &parseo) || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("error temporal perdido o efecto confirmado: %v, commits=%d", err, tx.commits)
	}
}
