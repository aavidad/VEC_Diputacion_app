package main

import (
	"testing"
	"time"
)

// consumir_decision_mutacion_v3_interna rechaza la sesión si faltan estos
// límites o salen de su rango (1..15 s y 1..20 s).
func TestPoolsADMINFijanLimitesSesionVECAD3(t *testing.T) {
	pc, err := configurarPoolADMIN("host=db.internal user=runtime dbname=vec password=secreto sslmode=verify-full")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []struct {
		clave  string
		maximo time.Duration
	}{{"statement_timeout", 15 * time.Second}, {"idle_in_transaction_session_timeout", 20 * time.Second}, {"lock_timeout", 15 * time.Second}} {
		v, err := time.ParseDuration(pc.ConnConfig.RuntimeParams[l.clave])
		if err != nil || v < time.Second || v > l.maximo {
			t.Fatalf("%s fuera de los límites VEC-AD-3: %q", l.clave, pc.ConnConfig.RuntimeParams[l.clave])
		}
	}
	if _, err := configurarPoolADMIN("host=db.internal port=no-numerico"); err == nil {
		t.Fatal("DSN inválido aceptado")
	}
}

// El pool de cargos lleva además la zona horaria que exige Personal28; los
// demás pools no cambian.
func TestPoolCargosFijaUTC(t *testing.T) {
	dsn := "host=localhost dbname=x user=y sslmode=disable"
	pc, err := configurarPoolCargosADMIN(dsn)
	if err != nil || pc.ConnConfig.RuntimeParams["timezone"] != "UTC" || pc.ConnConfig.RuntimeParams["statement_timeout"] != "10s" {
		t.Fatalf("pool de cargos sin UTC o sin límites: %v", err)
	}
	otro, err := configurarPoolADMIN(dsn)
	if err != nil || otro.ConnConfig.RuntimeParams["timezone"] != "" {
		t.Fatal("la zona horaria de cargos se filtró a los demás pools")
	}
	if acreditarZonaHorariaUTC(t.Context(), nil) == nil {
		t.Fatal("pool ausente acreditado")
	}
}
