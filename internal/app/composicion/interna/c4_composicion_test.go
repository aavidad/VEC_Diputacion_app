package interna

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/puddle/v2"
)

func TestMaterialIdentidadPresentadorOpcionalYNominal(t *testing.T) {
	base := t.TempDir()
	identidad := filepath.Join(base, "identidad")
	if err := os.Chmod(base, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(identidad, 0700); err != nil {
		t.Fatal(err)
	}
	ct := MaterialPoolsSeguimiento{}
	for i, entrada := range []*MaterialPoolSeguimiento{
		&ct.AltaPersonal, &ct.RegistroCT, &ct.InicialCT, &ct.LocalizadorCT,
		&ct.LocalizadorPersonal, &ct.LecturaPersonal, &ct.HistoriaRegistroCT,
		&ct.HistoriaAutenticacion, &ct.HistoriaContexto, &ct.HistoriaEvaluacion,
		&ct.HistoriaConcesion,
	} {
		entrada.Login = "ct_" + string(rune('a'+i))
	}
	ruta := filepath.Join(identidad, "pools.json")
	const previos = `"registro":{"login":"id_reg","dsn":"dsn_reg"},` +
		`"revalidacion":{"login":"id_rev","dsn":"dsn_rev"},` +
		`"contexto":{"login":"id_ctx","dsn":"dsn_ctx"},` +
		`"auditoria":{"login":"id_aud","dsn":"dsn_aud"}`
	escribir := func(campo string) {
		t.Helper()
		if err := os.WriteFile(ruta, []byte(`{"version":1,"pools":{`+previos+campo+`}}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	escribir("")
	material, err := cargarMaterialPoolsIdentidad(base, ct)
	if err != nil || material.presentador != (entradaPoolSeguimiento{}) {
		t.Fatalf("configuración previa rechazada: %v", err)
	}
	escribir(`,"presentador":{"login":"id_presentador","dsn":"dsn_presentador"}`)
	material, err = cargarMaterialPoolsIdentidad(base, ct)
	if err != nil || material.presentador.Login != "id_presentador" {
		t.Fatalf("presentador dedicado rechazado: %v", err)
	}
	for nombre, campo := range map[string]string{
		"nulo":            `,"presentador":null`,
		"vacio":           `,"presentador":{}`,
		"incompleto":      `,"presentador":{"login":"id_presentador"}`,
		"reusa identidad": `,"presentador":{"login":"id_reg","dsn":"dsn_presentador"}`,
		"reusa CT":        `,"presentador":{"login":"ct_a","dsn":"dsn_presentador"}`,
		"duplicado":       `,"presentador":{"login":"id_presentador","login":"otro","dsn":"dsn_presentador"}`,
		"desconocido":     `,"presentador":{"login":"id_presentador","dsn":"dsn_presentador","rol":"privilegiado"}`,
	} {
		t.Run(nombre, func(t *testing.T) {
			escribir(campo)
			if _, err := cargarMaterialPoolsIdentidad(base, ct); !errors.Is(err, ErrMaterialSeguimientoNoDisponible) {
				t.Fatalf("presentador inválido admitido: %v", err)
			}
		})
	}
}

func TestCerrarPoolsIdentidadIncluyePresentador(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgresql://presentador:prueba@127.0.0.1:1/vec?sslmode=verify-full")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	PoolsIdentidadInterna{Presentador: pool}.Cerrar()
	if _, err := pool.Acquire(context.Background()); !errors.Is(err, puddle.ErrClosedPool) {
		t.Fatalf("presentador quedó abierto tras cerrar recursos: %v", err)
	}
}
