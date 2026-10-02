package contrastecopias

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	domain "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

func nuevoPGRevisionAvanzada(t *testing.T, name string) (dockerEnsayo, *Lector) {
	t.Helper()
	if os.Getenv("VEC_CS06_REVISION_AVANZADA_ENSAYO") != "1" {
		t.Skip("regresiones PostgreSQL de revisión no solicitadas")
	}
	cmd := exec.CommandContext(context.Background(), "/usr/bin/docker", "run", "-d", "--name", name, "--network", "none", "--label", "vec.cs06.owner=lector-sol", "--cpus", "1", "--memory", "256m", "--pids-limit", "96", "--tmpfs", "/var/lib/postgresql:rw,size=192m", "--tmpfs", "/var/run/postgresql:rw,size=16m", "-e", "POSTGRES_HOST_AUTH_METHOD=trust", "postgres:18.4")
	if _, e := cmd.Output(); e != nil {
		t.Fatal("runtime propio de regresión no disponible")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if e := exec.CommandContext(ctx, "/usr/bin/docker", "rm", "-f", name).Run(); e != nil {
			t.Error("limpieza_regresion_pg_no_confirmada")
		}
	})
	x := dockerEnsayo{name: name}
	if _, e := x.ComprobarExclusion(context.Background()); e != nil {
		t.Fatal(e)
	}
	esperarPGAvanzado(t, name, x)
	cfg := limitesPrueba()
	cfg.MaxBytes = 32 << 20
	cfg.MaxFilas = 250000
	cfg.TiempoMaximo = 90 * time.Second
	reader, e := Nuevo(cfg)
	if e != nil {
		t.Fatal(e)
	}
	return x, reader
}
func capturarPGRevisionAvanzada(t *testing.T, x dockerEnsayo, reader *Lector) domain.Snapshot {
	t.Helper()
	s, e := reader.CapturarEjecutor(context.Background(), x, "postgres", x)
	if e != nil {
		t.Fatal(e)
	}
	if !s.Completo || len(domain.Validar(s)) != 0 {
		t.Fatalf("regresión no obtuvo captura completa: %v", s.Motivos)
	}
	return s
}
func TestDependenciaExtensionAvanzadaPG18(t *testing.T) {
	x, reader := nuevoPGRevisionAvanzada(t, "vec-cs06-rev-depend-sol-20261002")
	sqlEnsayo(t, x, `CREATE FUNCTION public.puntuar_revision(v int4) RETURNS int4 LANGUAGE SQL IMMUTABLE AS $$SELECT v+1$$;`)
	baseline := capturarPGRevisionAvanzada(t, x, reader)
	sqlEnsayo(t, x, `ALTER FUNCTION public.puntuar_revision(int4) DEPENDS ON EXTENSION plpgsql;`)
	changed := capturarPGRevisionAvanzada(t, x, reader)
	if !claseDifiere(domain.Comparar(baseline, changed), "esquema") {
		t.Fatal("dependencia semántica de extensión no contrastada")
	}
	sqlEnsayo(t, x, `ALTER FUNCTION public.puntuar_revision(int4) NO DEPENDS ON EXTENSION plpgsql;`)
	if domain.Comparar(baseline, capturarPGRevisionAvanzada(t, x, reader)).Estado != domain.Igual {
		t.Fatal("retirada de dependencia no recupera evidencia")
	}
	t.Log("PG18.4: DEPENDS ON EXTENSION difiere; NO DEPENDS conserva igualdad")
}
func TestColumnaIArrayCompuestoAvanzadoPG18(t *testing.T) {
	x, reader := nuevoPGRevisionAvanzada(t, "vec-cs06-rev-array-sol-20261002")
	sqlEnsayo(t, x, `CREATE TYPE public.fila_revision_i AS (a int4[]);
CREATE TABLE public.array_revision(i int4[]);
INSERT INTO public.array_revision VALUES('[0:1][3:4]={{1,NULL},{1,2}}'::int4[]);
CREATE TABLE public.compuesto_revision(i public.fila_revision_i);
INSERT INTO public.compuesto_revision VALUES(ROW('[0:1][3:4]={{1,NULL},{1,2}}'::int4[])::public.fila_revision_i);`)
	baseline := capturarPGRevisionAvanzada(t, x, reader)
	roundtripGrandeEnsayo(t, x)
	if r := domain.Comparar(baseline, capturarPGRevisionAvanzada(t, x, reader)); r.Estado != domain.Igual {
		t.Fatalf("columnas i no conservan igualdad tras restaurar: %v", r.Razones)
	}
	sqlEnsayo(t, x, `UPDATE public.array_revision SET i='[0:1][3:4]={{1,NULL},{1,9}}'::int4[];`)
	if !claseDifiere(domain.Comparar(baseline, capturarPGRevisionAvanzada(t, x, reader)), "tablas") {
		t.Fatal("elemento distinto de array i no contrastado")
	}
	sqlEnsayo(t, x, `UPDATE public.array_revision SET i='[0:1][3:4]={{1,NULL},{1,2}}'::int4[];
UPDATE public.compuesto_revision SET i=ROW('[0:1][3:4]={{1,NULL},{1,9}}'::int4[])::public.fila_revision_i;`)
	if !claseDifiere(domain.Comparar(baseline, capturarPGRevisionAvanzada(t, x, reader)), "tablas") {
		t.Fatal("array dentro de compuesto i no contrastado")
	}
	t.Log("PG18.4: columna i array y compuesto, dimensiones2/límites no1, restore igual y elementos distintos detectados")
}
