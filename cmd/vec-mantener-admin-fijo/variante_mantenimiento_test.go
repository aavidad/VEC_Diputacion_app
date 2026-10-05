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
	for _, v := range []uint64{0, 3, 7, 999} {
		if versionMantenimientoAdmitida(v) {
			t.Fatal("version_abierta")
		}
	}
	v1, _ := varianteMantenimiento(1)
	v2, _ := varianteMantenimiento(2)
	if v1.QuerySQL == v2.QuerySQL || !strings.Contains(v2.QuerySQL, "mantener_version_perfil_fijo_lote_admin_v1(") {
		t.Fatal("fachada_no_separada")
	}
}
