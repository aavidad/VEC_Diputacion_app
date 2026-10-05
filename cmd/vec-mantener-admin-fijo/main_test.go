package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type txPrueba struct {
	resultado   []byte
	commitError bool
	confirmada  bool
	cerrada     bool
}

func (t *txPrueba) PrepararUTC(context.Context) error { return nil }
func (t *txPrueba) Provisionar(context.Context, string, string) ([]byte, error) {
	return append([]byte(nil), t.resultado...), nil
}
func (t *txPrueba) Confirmar(context.Context) error {
	if t.commitError {
		return errors.New("secreto_no_exponer")
	}
	t.confirmada = true
	return nil
}
func (t *txPrueba) Cerrar(context.Context) { t.cerrada = true }
func planPrueba() documentoPlan {
	return documentoPlan{Version: 1, OperacionRef: "pmf_aaaaaaaaaaaaaaaaaaaaaaaa", RolDestino: json.RawMessage(`{"version":5}`), Asignaciones: []objetivo{{AsignacionRef: "asignacion:uno:v1"}, {AsignacionRef: "asignacion:dos:v1"}}}
}
func envelopePrueba(estado string) []byte {
	h := strings.Repeat("a", 64)
	a := auditoriaIntento{AuditoriaRef: "aud_v3_mfi_" + strings.Repeat("a", 32), Secuencia: 3, HuellaSHA256: h, CorrelacionRef: "correlacion_" + strings.Repeat("a", 32), RegistradaEn: "2026-10-04T00:00:00.000001Z"}
	e := envoltura{Estado: estado, AuditoriaIntento: a}
	if estado == "permitido" {
		e.Recibo = &reciboMantenimiento{Esquema: "vec.admin.mantenimiento-fijo.v1", OperacionRef: planPrueba().OperacionRef, PlanSHA256: h, RolOrigenRef: "rol:administracion_perfiles:v4", RolDestinoRef: "rol:administracion_perfiles:v5", RolDestinoSHA: h, Asignaciones: []referenciaAsignacion{{Ref: "asignacion:uno:v2", SHA: h}, {Ref: "asignacion:dos:v2", SHA: h}}, AuditoriaRef: "aud_v3_mf_" + strings.Repeat("a", 32), AuditoriaSecuencia: 2, AuditoriaHuella: h, ConfirmadoEn: "2026-10-04T00:00:00.000001Z"}
	} else {
		c := "mantenimiento_rechazado"
		if estado == "error" {
			c = "mantenimiento_no_disponible"
		}
		e.Codigo = &c
	}
	b, _ := json.Marshal(e)
	return b
}
func TestMantenimientoCommitCadaEstadoYErrorIncierto(t *testing.T) {
	for _, estado := range []string{"permitido", "denegado", "error"} {
		t.Run(estado, func(t *testing.T) {
			tx := &txPrueba{resultado: envelopePrueba(estado)}
			abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
			b, e, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, []byte("plan"), strings.Repeat("a", 64), planPrueba(), abrir)
			if err != nil || len(b) == 0 || e.Estado != estado || !tx.confirmada || !tx.cerrada {
				t.Fatal("estado_no_confirmado")
			}
		})
	}
	tx := &txPrueba{resultado: envelopePrueba("permitido"), commitError: true}
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	b, _, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, []byte("plan"), strings.Repeat("a", 64), planPrueba(), abrir)
	if err != errCommit || len(b) != 0 || !tx.cerrada {
		t.Fatal("COMMIT_ambiguo_se_presento_confirmado")
	}
}
func TestMantenimientoNoAceptaTercerPerfilNiFalsaConfirmacion(t *testing.T) {
	b := envelopePrueba("permitido")
	var e envoltura
	if json.Unmarshal(b, &e) != nil {
		t.Fatal("fixture")
	}
	e.Recibo.Asignaciones = append(e.Recibo.Asignaciones, referenciaAsignacion{Ref: "asignacion:ajena:v2", SHA: strings.Repeat("a", 64)})
	b, _ = json.Marshal(e)
	if _, err := validarEnvoltura(b, planPrueba(), strings.Repeat("a", 64)); err == nil {
		t.Fatal("asignacion_ajena_admitida")
	}
	var d envoltura
	if json.Unmarshal(envelopePrueba("denegado"), &d) != nil {
		t.Fatal("fixture")
	}
	d.Replay = true
	b, _ = json.Marshal(d)
	if _, err := validarEnvoltura(b, planPrueba(), strings.Repeat("a", 64)); err == nil {
		t.Fatal("denegacion_replay_falsa")
	}
	if decodificarEstricto([]byte(`{"version":1,"version":1}`), new(documentoPlan)) == nil {
		t.Fatal("claves_duplicadas_admitidas")
	}
	if bytes.Contains(b, []byte("secreto_no_exponer")) {
		t.Fatal("diagnostico_privado")
	}
}

func TestMantenimientoArchivoPrivadoYClavesExactas(t *testing.T) {
	dir := t.TempDir()
	if os.Chmod(dir, 0700) != nil {
		t.Fatal("directorio")
	}
	p := filepath.Join(dir, "plan.json")
	if os.WriteFile(p, []byte(`{"dato":"sintetico"}`), 0600) != nil {
		t.Fatal("fixture")
	}
	if _, e := leerPrivado(p); e != nil {
		t.Fatal("privado_rechazado")
	}
	if os.Chmod(p, 0644) != nil {
		t.Fatal("chmod")
	}
	if _, e := leerPrivado(p); e == nil {
		t.Fatal("archivo_publico_admitido")
	}
	if os.Chmod(p, 0600) != nil {
		t.Fatal("chmod")
	}
	alias := filepath.Join(dir, "alias.json")
	if os.Symlink(p, alias) != nil {
		t.Fatal("enlace")
	}
	if _, e := leerPrivado(alias); e == nil {
		t.Fatal("enlace_admitido")
	}
	if decodificarEstricto([]byte(`{"huella_plan_sha256":"x","actor_persona_ref":"inventado"}`), new(aprobacionPrivada)) == nil {
		t.Fatal("actor_por_peticion")
	}
}
