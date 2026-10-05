package postgres

import (
	"context"
	"testing"
	"time"
	"vec-diputacion-granada/internal/vec/ports"
)

// AUT48: el adaptador no fija la versión del rol de Aplicación; la concesión
// exacta la comprueban el emisor y PostgreSQL.
func TestSourceUsuariosAdmiteVersionVigenteDeAplicacion(t *testing.T) {
	for _, v := range []int{4, 5, 6, 7} {
		ahora := time.Now().UTC().Truncate(time.Microsecond)
		actor, v2 := evidenciaAdminPrueba(t, ahora)
		a := ambito{"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "unidad_admin_sintetica"}
		s := snapshotAdminPrueba(t, actor, ahora, a)
		s.VersionRol.Version = v
		s.AsignacionPerfil.VersionRolRef = s.VersionRol.Referencia()
		s.ControlVigenciaVersionRol.VersionRolRef = s.VersionRol.Referencia()
		emisor := &emisorMutadorPrueba{}
		reg := &capturadorIntentoUsuarios{ahora: ahora}
		f := &Fuente{ambito: a, config: Configuracion{Proceso: "vec_admin", Canal: "administracion_privilegiada", MotivoDenegado: motivoIntentoPrueba(), MotivoError: motivoIntentoPrueba()}, pool: &acreditacionPoolPrueba{}, fuente: fuenteSnapshotPrueba{s}, emisor: emisor, intentos: reg, reloj: relojAdminPrueba{ahora}}
		ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		// El proveedor adversario devuelve error: no hay lectura ni V3 favorable.
		_, err = f.ListarUsuarios(ctx, actor, v2, ports.FiltrosUsuariosAdministrables{})
		if err == nil || emisor.llamadas != 1 {
			t.Fatalf("version%d fronteraincorrecta", v)
		}
	}
}
