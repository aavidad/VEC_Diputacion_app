package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMantenimientoRol6ConservaCadaEstadoYCommitIncierto(t *testing.T) {
	for _, estado := range []string{"permitido", "denegado", "error"} {
		t.Run(estado, func(t *testing.T) {
			p := planPrueba()
			p.Version = 2
			p.RolDestino = json.RawMessage(`{"version":6}`)
			for i := range p.Asignaciones {
				p.Asignaciones[i].AsignacionRef = strings.TrimSuffix(p.Asignaciones[i].AsignacionRef, ":v1") + ":v2"
			}
			var e envoltura
			if err := json.Unmarshal(envelopePrueba(estado), &e); err != nil {
				t.Fatal(err)
			}
			if e.Recibo != nil {
				e.Recibo.Esquema = "vec.admin.mantenimiento-fijo.v2"
				e.Recibo.RolOrigenRef = "rol:administracion_perfiles:v5"
				e.Recibo.RolDestinoRef = "rol:administracion_perfiles:v6"
				for i := range e.Recibo.Asignaciones {
					e.Recibo.Asignaciones[i].Ref = strings.TrimSuffix(e.Recibo.Asignaciones[i].Ref, ":v2") + ":v3"
				}
			}
			b, err := json.Marshal(e)
			if err != nil {
				t.Fatal(err)
			}
			sha := strings.Repeat("a", 64)
			tx := &txPrueba{resultado: b}
			abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
			_, got, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, []byte("plan_aprobado"), sha, p, abrir)
			if err != nil || got.Estado != estado || !tx.confirmada || !tx.cerrada {
				t.Fatal("Rol6_no_confirmado")
			}
			if estado == "permitido" {
				tx.commitError = true
				_, _, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, []byte("plan_aprobado"), sha, p, abrir)
				if err != errCommit {
					t.Fatal("Rol6_commit_incierto_ack")
				}
				e.Recibo.RolDestinoRef = "rol:administracion_perfiles:v5"
				b, err = json.Marshal(e)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = validarEnvoltura(b, p, sha); err == nil {
					t.Fatal("acuse_Rol5_prestado_Rol6")
				}
			}
		})
	}
	for _, v := range []uint64{0, 5, 7, 999} {
		if versionMantenimientoAdmitida(v) {
			t.Fatal("version_abierta")
		}
	}
	v1, _ := varianteMantenimiento(1)
	v2, _ := varianteMantenimiento(2)
	v3, _ := varianteMantenimiento(3)
	if v1.QuerySQL == v2.QuerySQL || v2.QuerySQL == v3.QuerySQL || !strings.Contains(v2.QuerySQL, "mantener_version_perfil_fijo_lote_admin_v1(") ||
		!strings.Contains(v3.QuerySQL, "mantener_version_perfil_fijo_plan_firma_admin_v1(") {
		t.Fatal("fachada_no_separada")
	}
}

// AUT59: el acuse de Rol8 sólo puede confirmar el plan v4 aprobado. Un
// recibo Rol7 (o una asignación v4 reutilizada) no se acepta como efecto.
func TestMantenimientoRol8GobiernoDefinicionesExigeReciboExacto(t *testing.T) {
	p := planPrueba()
	p.Version = 4
	p.RolDestino = json.RawMessage(`{"version":8}`)
	for i := range p.Asignaciones {
		p.Asignaciones[i].AsignacionRef = strings.TrimSuffix(p.Asignaciones[i].AsignacionRef, ":v1") + ":v4"
	}
	var e envoltura
	if err := json.Unmarshal(envelopePrueba("permitido"), &e); err != nil {
		t.Fatal(err)
	}
	e.Recibo.Esquema = "vec.admin.mantenimiento-fijo.v4"
	e.Recibo.RolOrigenRef = "rol:administracion_perfiles:v7"
	e.Recibo.RolDestinoRef = "rol:administracion_perfiles:v8"
	for i := range e.Recibo.Asignaciones {
		e.Recibo.Asignaciones[i].Ref = strings.TrimSuffix(e.Recibo.Asignaciones[i].Ref, ":v2") + ":v5"
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.Repeat("a", 64)
	tx := &txPrueba{resultado: b}
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	_, got, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second,
		[]byte("plan_aprobado"), sha, p, abrir)
	if err != nil || got.Estado != "permitido" || !tx.confirmada || !tx.cerrada {
		t.Fatal("Rol8_no_confirmado")
	}
	v4, ok := varianteMantenimiento(4)
	if !ok || v4.OrigenRef != "rol:administracion_perfiles:v7" || v4.DestinoRef != "rol:administracion_perfiles:v8" ||
		!strings.Contains(v4.QuerySQL, "mantener_version_perfil_fijo_gobierno_definiciones_admin_v1(") {
		t.Fatal("fachada_Aut59_no_cerrada")
	}
	e.Recibo.RolDestinoRef = "rol:administracion_perfiles:v7"
	b, _ = json.Marshal(e)
	if _, err := validarEnvoltura(b, p, sha); err == nil {
		t.Fatal("acuse_Rol7_prestado_Rol8")
	}
	e.Recibo.RolDestinoRef = "rol:administracion_perfiles:v8"
	e.Recibo.Asignaciones[0].Ref = strings.TrimSuffix(e.Recibo.Asignaciones[0].Ref, ":v5") + ":v4"
	b, _ = json.Marshal(e)
	if _, err := validarEnvoltura(b, p, sha); err == nil {
		t.Fatal("asignacion_v4_prestada_Rol8")
	}
}

// AUT51: el acuse de Rol7 sólo vale para un plan versión 3 y no se presta a Rol6.
func TestMantenimientoRol7PlanFirmaConservaEstadosYNoPrestaAcuse(t *testing.T) {
	for _, estado := range []string{"permitido", "denegado", "error"} {
		t.Run(estado, func(t *testing.T) {
			p := planPrueba()
			p.Version = 3
			p.RolDestino = json.RawMessage(`{"version":7}`)
			for i := range p.Asignaciones {
				p.Asignaciones[i].AsignacionRef = strings.TrimSuffix(p.Asignaciones[i].AsignacionRef, ":v1") + ":v3"
			}
			var e envoltura
			if err := json.Unmarshal(envelopePrueba(estado), &e); err != nil {
				t.Fatal(err)
			}
			if e.Recibo != nil {
				e.Recibo.Esquema = "vec.admin.mantenimiento-fijo.v3"
				e.Recibo.RolOrigenRef = "rol:administracion_perfiles:v6"
				e.Recibo.RolDestinoRef = "rol:administracion_perfiles:v7"
				for i := range e.Recibo.Asignaciones {
					e.Recibo.Asignaciones[i].Ref = strings.TrimSuffix(e.Recibo.Asignaciones[i].Ref, ":v2") + ":v4"
				}
			}
			b, err := json.Marshal(e)
			if err != nil {
				t.Fatal(err)
			}
			sha := strings.Repeat("a", 64)
			tx := &txPrueba{resultado: b}
			abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
			_, got, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, []byte("plan_aprobado"), sha, p, abrir)
			if err != nil || got.Estado != estado || !tx.confirmada || !tx.cerrada {
				t.Fatal("Rol7_no_confirmado")
			}
			if estado == "permitido" {
				e.Recibo.RolDestinoRef = "rol:administracion_perfiles:v6"
				b, err = json.Marshal(e)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = validarEnvoltura(b, p, sha); err == nil {
					t.Fatal("acuse_Rol6_prestado_Rol7")
				}
			}
		})
	}
}
