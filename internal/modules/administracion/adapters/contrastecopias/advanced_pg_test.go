package contrastecopias

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"os/exec"
	"testing"
	"time"

	domain "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

func TestInventarioAvanzadoPG18(t *testing.T) {
	if os.Getenv("VEC_CS06_AVANZADO_ENSAYO") != "1" {
		t.Skip("ensayo PostgreSQL avanzado no solicitado")
	}
	name := "vec-cs06-avanzado-sol-20261002"
	cmd := exec.CommandContext(context.Background(), "/usr/bin/docker", "run", "-d", "--name", name, "--network", "none", "--label", "vec.cs06.owner=lector-sol", "--cpus", "1", "--memory", "256m", "--pids-limit", "96", "--tmpfs", "/var/lib/postgresql:rw,size=192m", "--tmpfs", "/var/run/postgresql:rw,size=16m", "-e", "POSTGRES_HOST_AUTH_METHOD=trust", "postgres:18.4")
	if _, e := cmd.Output(); e != nil {
		t.Fatal("runtime sintético avanzado no disponible")
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if e := exec.CommandContext(ctx, "/usr/bin/docker", "rm", "-f", name).Run(); e != nil {
			t.Error("limpieza_runtime_avanzado_no_confirmada")
		}
	})
	x := dockerEnsayo{name: name}
	if _, e := x.ComprobarExclusion(context.Background()); e != nil {
		t.Fatal(e)
	}
	esperarPGAvanzado(t, name, x)
	secret := secretoSinteticoAvanzado(t)
	sqlEnsayo(t, x, `CREATE ROLE cs06_avanzado_privado LOGIN PASSWORD `+literal(secret)+`; CREATE ROLE cs06_avanzado_lector;
CREATE TYPE public.estado_avanzado AS ENUM('inicial','final');
ALTER TYPE public.estado_avanzado ADD VALUE 'intermedio' BEFORE 'final';
CREATE DOMAIN public.numero_positivo AS int4 CHECK(VALUE>0);
CREATE TYPE public.fila_avanzada AS (texto text,numero public.numero_positivo);
CREATE TABLE public.padre_avanzado(id int4 PRIMARY KEY);
INSERT INTO public.padre_avanzado VALUES(1);
CREATE TABLE public.datos_avanzados(id int4,basura text,estado public.estado_avanzado,numero public.numero_positivo,registro public.fila_avanzada,lista public.fila_avanzada[],cuadro int4[],valor text,CONSTRAINT id_positivo CHECK(id>0),CONSTRAINT dato_padre FOREIGN KEY(id) REFERENCES public.padre_avanzado(id));
ALTER TABLE public.datos_avanzados DROP COLUMN basura;
ALTER TABLE public.datos_avanzados ADD COLUMN recreada text;
INSERT INTO public.datos_avanzados VALUES(1,'intermedio',7,ROW(NULL,NULL)::public.fila_avanzada,ARRAY[ROW('a',1)::public.fila_avanzada,NULL], '[0:1][3:4]={{1,NULL},{1,2}}'::int4[],'dato','vivo');
INSERT INTO public.datos_avanzados VALUES(1,'final',9,NULL,ARRAY[]::public.fila_avanzada[],NULL,'otro','vivo');
INSERT INTO public.datos_avanzados VALUES(1,'final',9,NULL,ARRAY[]::public.fila_avanzada[],NULL,'otro','vivo');
CREATE FUNCTION public.puntuar_avanzado(v int4) RETURNS int4 LANGUAGE SQL IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$SELECT v+1$$;
CREATE FUNCTION public.disparador_avanzado() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RETURN NEW; END$$;
CREATE TRIGGER cambio_avanzado BEFORE UPDATE ON public.datos_avanzados FOR EACH ROW EXECUTE FUNCTION public.disparador_avanzado();
ALTER TABLE public.datos_avanzados ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.datos_avanzados FORCE ROW LEVEL SECURITY;
CREATE POLICY solo_primero ON public.datos_avanzados FOR SELECT TO cs06_avanzado_lector USING(id=1);
CREATE FUNCTION public.vista_no_ejecutar() RETURNS text LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'funcion_de_vista_no_debe_ejecutarse'; END$$;
CREATE VIEW public.vista_avanzada AS SELECT public.vista_no_ejecutar() AS valor;
CREATE MATERIALIZED VIEW public.materializado_avanzado AS SELECT id,valor FROM public.datos_avanzados;
COMMENT ON TABLE public.datos_avanzados IS 'datos sintéticos avanzados';
COMMENT ON TYPE public.fila_avanzada IS 'compuesto sintético';
CREATE STATISTICS public.estadistica_avanzada (dependencies) ON id,valor FROM public.datos_avanzados;
CREATE COLLATION public.orden_avanzado (provider=libc,locale='C');
CREATE TEXT SEARCH CONFIGURATION public.busqueda_avanzada (COPY=pg_catalog.simple);
GRANT SELECT ON public.datos_avanzados TO cs06_avanzado_lector;`)
	cfg := limitesPrueba()
	cfg.MaxBytes = 32 << 20
	cfg.MaxFilas = 250000
	cfg.TiempoMaximo = 90 * time.Second
	reader, e := Nuevo(cfg)
	if e != nil {
		t.Fatal(e)
	}
	capture := func() domain.Snapshot {
		s, e := reader.CapturarEjecutor(context.Background(), x, "postgres", x)
		if e != nil {
			t.Fatal(e)
		}
		if !s.Completo || len(domain.Validar(s)) != 0 || s.Version != domain.VersionCanonica {
			t.Fatalf("captura avanzada incompleta: %v", s.Motivos)
		}
		return s
	}
	baseline := capture()
	// CREATE/RESTORE changes enum/internaltrigger/type/column OIDs and closes
	// dropped attribute holes; definitions and logical values must remain equal.
	roundtripGrandeEnsayo(t, x)
	if r := domain.Comparar(baseline, capture()); r.Estado != domain.Igual {
		t.Fatalf("restauración avanzada alteró evidencia: %v", r.Razones)
	}
	for _, test := range []struct{ name, change, restore, class string }{
		{"funcion", `CREATE OR REPLACE FUNCTION public.puntuar_avanzado(v int4) RETURNS int4 LANGUAGE SQL IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$SELECT v+2$$`, `CREATE OR REPLACE FUNCTION public.puntuar_avanzado(v int4) RETURNS int4 LANGUAGE SQL IMMUTABLE PARALLEL SAFE SET search_path=pg_catalog AS $$SELECT v+1$$`, "esquema"},
		{"rls", `ALTER POLICY solo_primero ON public.datos_avanzados USING(id>0)`, `ALTER POLICY solo_primero ON public.datos_avanzados USING(id=1)`, "esquema"},
		{"dominio", `ALTER DOMAIN public.numero_positivo SET DEFAULT 2`, `ALTER DOMAIN public.numero_positivo DROP DEFAULT`, "esquema"},
		{"trigger", `ALTER TABLE public.datos_avanzados DISABLE TRIGGER cambio_avanzado`, `ALTER TABLE public.datos_avanzados ENABLE TRIGGER cambio_avanzado`, "esquema"},
		{"trigger_fk", `ALTER TABLE public.datos_avanzados DISABLE TRIGGER ALL`, `ALTER TABLE public.datos_avanzados ENABLE TRIGGER ALL`, "esquema"},
		{"check", `ALTER TABLE public.datos_avanzados DROP CONSTRAINT id_positivo; ALTER TABLE public.datos_avanzados ADD CONSTRAINT id_positivo CHECK(id>=0)`, `ALTER TABLE public.datos_avanzados DROP CONSTRAINT id_positivo; ALTER TABLE public.datos_avanzados ADD CONSTRAINT id_positivo CHECK(id>0)`, "esquema"},
		{"enum", `ALTER TYPE public.estado_avanzado RENAME VALUE 'intermedio' TO 'medio'`, `ALTER TYPE public.estado_avanzado RENAME VALUE 'medio' TO 'intermedio'`, "esquema"},
		{"celda_mismo_count", `UPDATE public.datos_avanzados SET valor='cambiado' WHERE estado='intermedio'`, `UPDATE public.datos_avanzados SET valor='dato' WHERE estado='intermedio'`, "tablas"},
		{"compuesto_null", `UPDATE public.datos_avanzados SET registro=NULL WHERE estado='intermedio'`, `UPDATE public.datos_avanzados SET registro=ROW(NULL,NULL)::public.fila_avanzada WHERE estado='intermedio'`, "tablas"},
		{"array_limites", `UPDATE public.datos_avanzados SET cuadro='[1:2][3:4]={{1,NULL},{1,2}}'::int4[] WHERE estado='intermedio'`, `UPDATE public.datos_avanzados SET cuadro='[0:1][3:4]={{1,NULL},{1,2}}'::int4[] WHERE estado='intermedio'`, "tablas"},
		{"comentario", `COMMENT ON TYPE public.fila_avanzada IS 'comentario cambiado'`, `COMMENT ON TYPE public.fila_avanzada IS 'compuesto sintético'`, "esquema"},
	} {
		t.Run(test.name, func(t *testing.T) {
			sqlEnsayo(t, x, test.change)
			r := domain.Comparar(baseline, capture())
			if !claseDifiere(r, test.class) {
				t.Fatalf("divergencia avanzada no detectada: %v", r.Razones)
			}
			sqlEnsayo(t, x, test.restore)
		})
	}
	globals := dumpGlobalesAvanzado(t, name)
	sqlEnsayo(t, x, `ALTER ROLE cs06_avanzado_privado PASSWORD `+literal(secretoSinteticoAvanzado(t)))
	changed := capture()
	if !claseDifiere(domain.Comparar(baseline, changed), "roles") {
		t.Fatal("credencial alterada no contrastada")
	}
	// All roles already exist in this owned fixture; apply dump ALTER statements
	// without replaying their CREATE. No verifier is printed or stored in fixtures.
	lines := bytes.Split(globals, []byte{'\n'})
	out := []byte{}
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("CREATE ROLE ")) {
			continue
		}
		out = append(out, line...)
		out = append(out, '\n')
	}
	sqlEnsayo(t, x, string(out))
	if r := domain.Comparar(baseline, capture()); r.Estado != domain.Igual {
		t.Fatalf("verificador de credencial restaurado no conserva identidad: %v", r.Razones)
	}
	// Register a user base type with a custom C output that would fail if invoked.
	// Admission is decided before deparse/data, so the output never executes.
	sqlEnsayo(t, x, `CREATE TYPE public.tipo_opaco;
CREATE FUNCTION public.opaco_in(cstring) RETURNS public.tipo_opaco AS 'boolin' LANGUAGE internal IMMUTABLE STRICT;
CREATE FUNCTION public.opaco_out(public.tipo_opaco) RETURNS cstring AS 'trigger_out' LANGUAGE internal IMMUTABLE STRICT;
CREATE TYPE public.tipo_opaco (INPUT=public.opaco_in,OUTPUT=public.opaco_out,INTERNALLENGTH=1,PASSEDBYVALUE,ALIGNMENT=char); CREATE TABLE public.opacos_seguridad(valor public.tipo_opaco); INSERT INTO public.opacos_seguridad VALUES('true'::public.tipo_opaco);`)
	unsupported, e := reader.CapturarEjecutor(context.Background(), x, "postgres", x)
	if e != nil || unsupported.Completo || domain.Comparar(baseline, unsupported).Estado != domain.NoComprobable {
		t.Fatal("salida propia opaca ejecutada o admitida")
	}
	t.Log("PG18.4 real: funciones/tipos/RLS/triggers/enum/huecos/compuestos/arrays/vistas/materializados/credenciales; dump/restore igual con OID regenerados; diferencias semánticas y opacos rechazados sin ejecutar salida")
}
func secretoSinteticoAvanzado(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		t.Fatal("material sintético privado no disponible")
	}
	return hex.EncodeToString(b[:])
}
func dumpGlobalesAvanzado(t *testing.T, name string) []byte {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "/usr/bin/docker", "exec", "--user", "postgres", name, "pg_dumpall", "--roles-only")
	out, e := cmd.Output()
	if e != nil || len(out) > 1<<20 {
		t.Fatal("globales sintéticos privados no disponibles")
	}
	return out
}

func esperarPGAvanzado(t *testing.T, name string, x dockerEnsayo) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for {
		cmd := exec.CommandContext(ctx, "/usr/bin/docker", "exec", name, "cat", "/proc/1/comm")
		out, e := cmd.Output()
		if e == nil && bytes.Equal(bytes.TrimSpace(out), []byte("postgres")) {
			esperarPostgresEnsayo(t, x)
			return
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			t.Fatal("PostgreSQL sintético definitivo no disponible")
		case <-timer.C:
		}
	}
}
