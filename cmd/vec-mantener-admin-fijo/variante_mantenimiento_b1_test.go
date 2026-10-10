package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMantenimientoRol9B1ExigePlanYReciboExactos(t *testing.T) {
	p := planPrueba()
	p.Version = 5
	p.RolDestino = json.RawMessage(`{"version":9}`)
	for i := range p.Asignaciones {
		p.Asignaciones[i].AsignacionRef = strings.TrimSuffix(p.Asignaciones[i].AsignacionRef, ":v1") + ":v5"
	}
	var e envoltura
	if err := json.Unmarshal(envelopePrueba("permitido"), &e); err != nil {
		t.Fatal(err)
	}
	e.Recibo.Esquema = "vec.admin.mantenimiento-fijo.v5"
	e.Recibo.RolOrigenRef = "rol:administracion_perfiles:v8"
	e.Recibo.RolDestinoRef = "rol:administracion_perfiles:v9"
	for i := range e.Recibo.Asignaciones {
		e.Recibo.Asignaciones[i].Ref = strings.TrimSuffix(e.Recibo.Asignaciones[i].Ref, ":v2") + ":v6"
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.Repeat("a", 64)
	tx := &txPrueba{resultado: b}
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	_, resultado, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second,
		[]byte("plan_aprobado"), sha, p, abrir)
	if err != nil || resultado.Estado != "permitido" || !tx.confirmada || !tx.cerrada {
		t.Fatal("Rol9_no_confirmado")
	}
	v5, ok := varianteMantenimiento(5)
	if !ok || v5.OrigenRef != "rol:administracion_perfiles:v8" || v5.DestinoRef != "rol:administracion_perfiles:v9" ||
		!strings.Contains(v5.QuerySQL, "mantener_version_perfil_fijo_version_bolsa_admin_v1(") {
		t.Fatal("fachada_AUT64_no_cerrada")
	}
	e.Recibo.Asignaciones[0].Ref = strings.TrimSuffix(e.Recibo.Asignaciones[0].Ref, ":v6") + ":v5"
	b, err = json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validarEnvoltura(b, p, sha); err == nil {
		t.Fatal("asignacion_anterior_prestada_al_recibo")
	}
}
