package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
	"vec-diputacion-granada/internal/vec/domain"
)

// Los mismos resultados independientes se cotejan con los constructores SQL
// en los vectores46/45. No se publica un perfil ni una capacidad favorable.
func TestDestinosMantenimientoPasanValidadorSinCambiarHastaIdentidad(t *testing.T) {
	leer := func(nombre string) (domain.AsignacionPerfil, []byte) {
		t.Helper()
		b, err := os.ReadFile(filepath.Join("../../deploy/postgresql/autorizacion/pruebas_sql/testdata", "fechas_mantenimiento_"+nombre+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var d domain.AsignacionPerfil
		if err := json.Unmarshal(b, &d); err != nil {
			t.Fatal(err)
		}
		return d, b
	}
	original, _ := leer("original")
	historico, hb := leer("historico42")
	nuevo42, n42b := leer("nuevo42")
	nuevo45, _ := leer("nuevo45")
	if original.Validar() != nil || historico.Validar() == nil || nuevo42.Validar() != nil || nuevo45.Validar() != nil {
		t.Fatal("fecha_emision_inicio_invalida")
	}
	for _, d := range []domain.AsignacionPerfil{nuevo42, nuevo45} {
		if d.PrincipalID != original.PrincipalID || d.PerfilActivoRef != original.PerfilActivoRef || d.AsignacionID != original.AsignacionID || !d.VigenteHasta.Equal(original.VigenteHasta) || !d.VigenteDesde.Equal(d.EmitidaEn) {
			t.Fatal("destino_renovo_o_cambio_identidad")
		}
		a, err := json.Marshal(d.Ambitos)
		if err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(original.Ambitos)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			t.Fatal("ambitos_alterados")
		}
	}
	// Los recibos de replay histórico y prospectivo tienen hashes diferentes;
	// el driver valida y devuelve los bytes originales, sin reconstruirlos con
	// la fórmula actual ni reemitir un documento de asignación.
	for _, caso := range []struct {
		nombre string
		doc    []byte
		plan   documentoPlan
	}{
		{"historico42", hb, planPrueba()}, {"nuevo42", n42b, planPrueba()},
	} {
		var e envoltura
		if err := json.Unmarshal(envelopePrueba("permitido"), &e); err != nil {
			t.Fatal(err)
		}
		e.Replay = true
		e.Recibo.Asignaciones[0].SHA = hex.EncodeToString(sha256BytesFechas(caso.doc))
		b, err := json.MarshalIndent(e, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		tx := &txPrueba{resultado: b}
		abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
		originalSHA := e.Recibo.PlanSHA256
		raw, res, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, []byte("plan_original_aprobado"), originalSHA, caso.plan, abrir)
		if err != nil || !res.Replay || !tx.confirmada || !bytes.Equal(raw, b) || sha256.Sum256(raw) != sha256.Sum256(b) {
			t.Fatalf("replay_%s_bytes_o_sha_distintos", caso.nombre)
		}
	}
	if sha256.Sum256(hb) == sha256.Sum256(n42b) {
		t.Fatal("fixture_no_distingue_formula_historica")
	}
}
func sha256BytesFechas(b []byte) []byte { h := sha256.Sum256(b); return h[:] }
