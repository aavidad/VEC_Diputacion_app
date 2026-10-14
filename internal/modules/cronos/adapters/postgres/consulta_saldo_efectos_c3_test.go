package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func TestDecodificarFuenteSaldoEfectosC3ConservaHuellaYSesion(t *testing.T) {
	fecha := "2026-10-20"
	contenido := map[string]any{"saldo": map[string]any{"empleado_ref": "emp_prueba", "desde": fecha, "hasta": fecha, "zona_horaria": "Europe/Madrid", "completo": true,
		"jornadas": []any{}, "marcajes": []any{}, "movimientos_saldo": []any{}}, "programaciones": []any{}, "permisos": []any{}, "politicas": []any{}, "cobertura_permisos": true}
	b, err := json.Marshal(contenido)
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(b)
	huella := hex.EncodeToString(suma[:])
	sobre, err := json.Marshal(map[string]any{"fuente_canonica": string(b), "snapshot_ref": "saldo:cronos:snapshot:" + huella,
		"fuente_sha256": huella, "corte_utc": "2026-10-20T12:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	f, err := decodificarFuenteSaldoEfectosC3(sobre, "emp_prueba", fecha, fecha, "Europe/Madrid")
	if err != nil || f.EfectosC3 == nil || f.EfectosC3.SnapshotRef == "" || !f.EfectosC3.CoberturaPermisos {
		t.Fatalf("fuente=%+v, err=%v", f, err)
	}
	if _, err = decodificarFuenteSaldoEfectosC3(sobre, "emp_ajeno", fecha, fecha, "Europe/Madrid"); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatalf("empleado ajeno: %v", err)
	}
	sobreAlterado := map[string]any{}
	if err = json.Unmarshal(sobre, &sobreAlterado); err != nil {
		t.Fatal(err)
	}
	sobreAlterado["fuente_sha256"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	b, err = json.Marshal(sobreAlterado)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = decodificarFuenteSaldoEfectosC3(b, "emp_prueba", fecha, fecha, "Europe/Madrid"); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatalf("huella falsa: %v", err)
	}
}
