package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/bootstrap"
)

func TestCLIProvisionCargoSoloPreparaYNoExponeIdentidad(t *testing.T) {
	dir, err := os.MkdirTemp("/var/tmp", "vec-cargo-ct-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	c := bootstrap.ConfiguracionProvisionCargoCT{Version: 1, Cargo: "jefatura_servicio_rrhh",
		PrincipalID: "per:ana-molina", PerfilRef: "perfil:ana-molina:jefatura",
		OrganizacionRef: "org:granada", UnidadRef: "unidad:rrhh",
		VigenteDesde: ahora.Add(time.Hour), VigenteHasta: ahora.Add(48 * time.Hour),
		AprobacionInstalacionRef: "aprobacion:tecnica:001", AprobadorPrincipalID: "per:lucia-ruiz",
		PreimagenSHA256: strings.Repeat("a", 64), VersionRolSHA256: strings.Repeat("b", 64)}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	fuente := filepath.Join(dir, "cargo.json")
	dsn := filepath.Join(dir, "dsn.txt")
	if os.WriteFile(fuente, b, 0600) != nil || os.WriteFile(dsn, []byte("postgres://solo-lectura@127.0.0.1/vec_clon"), 0600) != nil {
		t.Fatal("fixture privado no creado")
	}
	if _, err := bootstrap.LeerMaterialProvisionExterna(fuente, 16<<10); err != nil {
		t.Fatalf("el lector privado rechazó el fixture %s: %v", fuente, err)
	}
	llamadas := 0
	var salida, errores bytes.Buffer
	d := dependencias{reloj: func() time.Time { return ahora }, preparar: func(_ context.Context, recibido string,
		config bootstrap.ConfiguracionProvisionCargoCT, _ time.Time) (bootstrap.ResumenProvisionCargoCT, error) {
		llamadas++
		if recibido != "postgres://solo-lectura@127.0.0.1/vec_clon" || config.Cargo != c.Cargo {
			t.Fatal("la CLI alteró la fuente o la conexión")
		}
		return bootstrap.ResumenProvisionCargoCT{Cargo: c.Cargo, Estado: "pendiente_canal_autorizado",
			PreimagenSHA256: c.PreimagenSHA256, PlanSHA256: strings.Repeat("b", 64), VersionSiguiente: 2}, nil
	}}
	if codigo := ejecutar(context.Background(), []string{"-fuente", fuente, "-dsn-archivo", dsn}, &salida, &errores, d); codigo != 0 || llamadas != 1 {
		t.Fatalf("preparación fallida: código=%d, llamadas=%d, error=%s", codigo, llamadas, errores.String())
	}
	if strings.Contains(salida.String(), c.PrincipalID) || strings.Contains(salida.String(), c.PerfilRef) ||
		strings.Contains(salida.String(), "postgres://") || !strings.Contains(salida.String(), "pendiente_canal_autorizado") {
		t.Fatalf("salida insegura o incompleta: %s", salida.String())
	}
	if codigo := ejecutar(context.Background(), []string{"-fuente", fuente, "-dsn-archivo", dsn, "-aprobar", "si"},
		&salida, &errores, d); codigo == 0 || llamadas != 1 {
		t.Fatal("la CLI admitió aplicar desde argumento")
	}
}
