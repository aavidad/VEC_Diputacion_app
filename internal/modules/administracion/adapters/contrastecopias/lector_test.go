package contrastecopias

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	domain "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

func TestHuellaConservaMulticonjunto(t *testing.T) {
	a := []string{`["int4","1"]`, `["text",null]`, `["text",""]`, `["int4","1"]`}
	b := []string{a[3], a[2], a[1], a[0]}
	if huella(a) != huella(b) {
		t.Fatal("orden físico alteró huella")
	}
	for _, c := range [][]string{a[:3], {a[0], a[2], a[2], a[3]}, {`["int8","1"]`, a[1], a[2], a[3]}} {
		if huella(a) == huella(c) {
			t.Fatal("omisión de multiplicidad, null o tipo")
		}
	}
}
func TestErroresNoRevelanConfiguracion(t *testing.T) {
	_, e := Nuevo(Configuracion{DSN: "secreto-dsn", VersionPostgreSQL: "18.4"})
	if e == nil || strings.Contains(e.Error(), "secreto") {
		t.Fatal("configuración no protegida")
	}
	l, e := Nuevo(limitesPrueba())
	if e != nil {
		t.Fatal(e)
	}
	_, e = l.Capturar(context.Background())
	if e != errCaptura {
		t.Fatal("DSN ausente no denegado")
	}
	_, e = l.CapturarEjecutor(context.Background(), nil, "postgres", nil)
	if e != errCaptura {
		t.Fatal("canal sin evidencia no denegado")
	}
}
func TestTransporteSoloLecturaYParametros(t *testing.T) {
	x := &execPrueba{out: []byte("{\"a\":true,\"b\":null,\"c\":\"\\n\\\\\\\"\"}\n")}
	tx := &transporteEjecutor{exec: x, base: "postgres", maxBytes: 4096, timeout: 1000}
	var a bool
	var b *string
	var c string
	if e := tx.QueryRow(context.Background(), `SELECT $1,$2,$3`, "dato'\\n", int64(5), uint32(7)).Scan(&a, &b, &c); e != nil {
		t.Fatal(e)
	}
	if !a || b != nil || c != "\n\\\"" {
		t.Fatal("transporte no conserva valores")
	}
	if !strings.Contains(x.input, "ISOLATION LEVEL REPEATABLE READ READ ONLY") || !strings.Contains(x.input, "'dato''\\n'") || x.tool != "psql" {
		t.Fatal("frontera de consulta incorrecta")
	}
	x.err = errors.New("credencial-dsn-celda")
	if _, e := tx.Query(context.Background(), "SELECT 1"); e != errCaptura {
		t.Fatal("error externo sin redacción")
	}
}
func TestParametrosNoAlteranCatalogo(t *testing.T) {
	sql := `SELECT $1, "nombre$1", 'literal$1', $$texto$1$$, $tag$texto$1$tag$ /* $1 /* $1 */ */ -- $1
 WHERE x=$2`
	got, e := bind(sql, []any{int64(17), "dato"})
	if e != nil {
		t.Fatal(e)
	}
	want := `SELECT 17, "nombre$1", 'literal$1', $$texto$1$$, $tag$texto$1$tag$ /* $1 /* $1 */ */ -- $1
 WHERE x='dato'`
	if got != want {
		t.Fatal("parámetro dentro de objeto catalogado modificado")
	}
}

func TestSelloExclusionCambiaDeniega(t *testing.T) {
	guard := &guardCambiante{}
	c := &captura{guard: guard}
	if ok, e := c.exclusion(context.Background()); e != nil || !ok {
		t.Fatal("primera evidencia inválida")
	}
	if ok, e := c.exclusion(context.Background()); e != nil || ok {
		t.Fatal("ventana sustituida admitida")
	}
}

type guardCambiante struct{ n int }

func (g *guardCambiante) ComprobarExclusion(context.Context) (string, error) {
	g.n++
	return strings.Repeat(string(rune('a'+g.n-1)), 64), nil
}

type execPrueba struct {
	out         []byte
	err         error
	input, tool string
}

func (x *execPrueba) EjecutarPostgreSQL(_ context.Context, tool string, _ []string, input []byte, _ int) ([]byte, error) {
	x.input = string(input)
	x.tool = tool
	return x.out, x.err
}
func limitesPrueba() Configuracion {
	return Configuracion{VersionPostgreSQL: "18.4", TiempoMaximo: 30 * time.Second, MaxFilas: 100000, MaxBytes: 8 << 20, MaxObjetos: 1000}
}

// Este ensayo exige un recurso sintético propio creado explícitamente por el
// responsable; jamás usa DSN, principal ni infraestructura compartida.
func TestInventarioPG18Aislado(t *testing.T) {
	name := os.Getenv("VEC_CS06_CONTENEDOR_ENSAYO")
	if name == "" {
		t.Skip("ensayo PostgreSQL aislado no solicitado")
	}
	if name != "vec-cs06-lector-sol-20261001" {
		t.Fatal("recurso de ensayo no propio")
	}
	x := dockerEnsayo{name: name}
	if _, e := x.ComprobarExclusion(context.Background()); e != nil {
		t.Fatal(e)
	}
	fixture := `CREATE ROLE cs06_lector_sintetico;
CREATE TABLE public.datos (id int4, valor text, CONSTRAINT dato_positivo CHECK (id>0));
CREATE SEQUENCE public.numero START 1 INCREMENT 1 CACHE 1;
CREATE TABLE public.padre(id int4 PRIMARY KEY);
CREATE TABLE public.hija(id int4 REFERENCES public.padre(id));
INSERT INTO public.padre VALUES(1); INSERT INTO public.hija VALUES(1);
INSERT INTO public.datos VALUES(2,''),(1,NULL),(2,''),(3,'á');
CREATE TABLE public."nombre$1"("columna$2" text); INSERT INTO public."nombre$1" VALUES('literal$3');`
	sqlEnsayo(t, x, fixture)
	l, e := Nuevo(limitesPrueba())
	if e != nil {
		t.Fatal(e)
	}
	capture := func() domain.Snapshot {
		s, e := l.CapturarEjecutor(context.Background(), x, "postgres", x)
		if e != nil {
			t.Fatal(e)
		}
		if !s.Completo {
			t.Fatalf("captura incompleta: %v", s.Motivos)
		}
		if rs := domain.Validar(s); len(rs) != 0 {
			t.Fatalf("snapshot inválido: %v", rs)
		}
		return s
	}
	first := capture()
	sqlEnsayo(t, x, `DROP TABLE public.datos; CREATE TABLE public.datos (id int4, valor text, CONSTRAINT dato_positivo CHECK (id>0)); INSERT INTO public.datos VALUES(3,'á'),(2,''),(2,''),(1,NULL);`)
	equivalent := capture()
	if r := domain.Comparar(first, equivalent); r.Estado != domain.Igual {
		t.Fatalf("orden/OID alteraron evidencia: %v", r.Razones)
	}
	for _, scenario := range []struct {
		name, change, restore string
		class                 string
	}{
		{"celda", `UPDATE public.datos SET valor='diferente' WHERE id=3`, `UPDATE public.datos SET valor='á' WHERE id=3`, "tablas"},
		{"check", `ALTER TABLE public.datos DROP CONSTRAINT dato_positivo; ALTER TABLE public.datos ADD CONSTRAINT dato_positivo CHECK(id>1 OR id=1)`, `ALTER TABLE public.datos DROP CONSTRAINT dato_positivo; ALTER TABLE public.datos ADD CONSTRAINT dato_positivo CHECK(id>0)`, "esquema"},
		{"secuencia", `SELECT setval('public.numero',17,true)`, `SELECT setval('public.numero',1,false)`, "secuencias"},
		{"acl", `GRANT SELECT ON public.datos TO cs06_lector_sintetico`, `REVOKE SELECT ON public.datos FROM cs06_lector_sintetico`, "acl"},
		{"privilegios_defecto", `ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO cs06_lector_sintetico`, `ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT ON TABLES FROM cs06_lector_sintetico`, "privilegios_defecto"},
		{"privilegios_defecto_vacios", `ALTER DEFAULT PRIVILEGES FOR ROLE cs06_lector_sintetico REVOKE ALL ON TABLES FROM cs06_lector_sintetico`, `ALTER DEFAULT PRIVILEGES FOR ROLE cs06_lector_sintetico GRANT ALL ON TABLES TO cs06_lector_sintetico`, "privilegios_defecto"},
		{"roles", `ALTER ROLE cs06_lector_sintetico NOINHERIT`, `ALTER ROLE cs06_lector_sintetico INHERIT`, "roles"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			sqlEnsayo(t, x, scenario.change)
			changed := capture()
			r := domain.Comparar(first, changed)
			if r.Estado != domain.Diferente {
				t.Fatal("divergencia no detectada")
			}
			found := false
			for _, reason := range r.Razones {
				if reason.Clase == scenario.class {
					found = true
				}
			}
			if !found {
				t.Fatalf("clase de divergencia ausente: %v", r.Razones)
			}
			sqlEnsayo(t, x, scenario.restore)
		})
	}
	t.Run("fk_disparadores_desactivados", func(t *testing.T) {
		sqlEnsayo(t, x, `ALTER TABLE public.hija DISABLE TRIGGER ALL`)
		s, e := l.CapturarEjecutor(context.Background(), x, "postgres", x)
		if e != nil {
			t.Fatal(e)
		}
		if s.Completo || domain.Comparar(first, s).Estado != domain.NoComprobable {
			t.Fatal("FK sin disparadores admitida como equivalente")
		}
		found := false
		for _, m := range s.Motivos {
			if m == "disparadores_internos_no_admitidos" {
				found = true
			}
		}
		if !found {
			t.Fatal("estado de disparadores internos omitido")
		}
		sqlEnsayo(t, x, `ALTER TABLE public.hija ENABLE TRIGGER ALL`)
		if r := domain.Comparar(first, capture()); r.Estado != domain.Igual {
			t.Fatalf("restauración de FK no recupera evidencia: %v", r.Razones)
		}
	})
	sqlEnsayo(t, x, `CREATE VIEW public.vista AS SELECT * FROM public.datos`)
	incomplete, e := l.CapturarEjecutor(context.Background(), x, "postgres", x)
	if e != nil || incomplete.Completo {
		t.Fatal("vista avanzada no denegada")
	}
	sqlEnsayo(t, x, `DROP VIEW public.vista; SELECT lo_from_bytea(0,decode('010200ff','hex'));`)
	incomplete, e = l.CapturarEjecutor(context.Background(), x, "postgres", x)
	if e != nil || incomplete.Completo {
		t.Fatalf("vínculo LO desconocido no denegado: %v %v", e, incomplete.Motivos)
	}
	found := false
	for _, o := range incomplete.Objetos {
		if o.Clase == "objetos_grandes" && o.Clave != "inventario" {
			found = true
		}
	}
	if !found {
		t.Fatal("contenido LO omitido")
	}
	sqlEnsayo(t, x, `SELECT lo_unlink(oid) FROM pg_largeobject_metadata`)
	limited := limitesPrueba()
	limited.MaxBytes = 100
	small, _ := Nuevo(limited)
	if _, e = small.CapturarEjecutor(context.Background(), x, "postgres", x); e != errLimite {
		t.Fatal("límite bytes no denegado")
	}
	// La primaria directa conserva la separación: sin guard observado, no completa.
	// La ruta externa se probó con evidencia runtime real, nunca con booleano libre.
	t.Log("PG18.4 real: orden y OID iguales; celda/CHECK/secuencia/ACL/roles/privilegios_defecto divergentes; ACL vacío distinguido de ausencia, FK desactivada no_comprobable; advanced/LO y límites denegados")
}

type dockerEnsayo struct{ name string }

func (x dockerEnsayo) EjecutarPostgreSQL(ctx context.Context, tool string, args []string, input []byte, max int) ([]byte, error) {
	if tool != "psql" {
		return nil, errCaptura
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/docker", append([]string{"exec", "-i", "--user", "postgres", x.name, tool}, args...)...)
	cmd.Stdin = bytes.NewReader(input)
	out, e := cmd.Output()
	if e != nil {
		return nil, errCaptura
	}
	if len(out) > max {
		return nil, errLimite
	}
	return out, nil
}
func (x dockerEnsayo) ComprobarExclusion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/docker", "inspect", x.name)
	out, e := cmd.Output()
	if e != nil {
		return "", errCaptura
	}
	var info []struct {
		Id, Image  string
		Config     struct{ Labels map[string]string }
		HostConfig struct {
			NetworkMode string
			Binds       []string
		}
		Mounts []struct{ Type string }
		State  struct{ Running bool }
	}
	if json.Unmarshal(out, &info) != nil || len(info) != 1 {
		return "", errCaptura
	}
	i := info[0]
	if !i.State.Running || i.HostConfig.NetworkMode != "none" || len(i.HostConfig.Binds) != 0 || len(i.Mounts) != 0 || i.Config.Labels["vec.cs06.owner"] != "lector-sol" || i.Image != "sha256:1bf3d6960db467e87a506daef30feb41fecc23b7c5f96b157e873059f2ffb50a" {
		return "", errCaptura
	}
	// Canal propio secuencial; el responsable creó este único recurso sintético y
	// mantiene la ventana desde la primera hasta la última consulta de captura.
	b, _ := json.Marshal([]string{i.Id, i.Image, i.HostConfig.NetworkMode, "lector-sol:ventana-secuencial"})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func sqlEnsayo(t *testing.T, x dockerEnsayo, sql string) {
	t.Helper()
	if _, e := x.EjecutarPostgreSQL(context.Background(), "psql", []string{"-X", "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1", "--dbname=postgres"}, []byte(sql), 1<<20); e != nil {
		t.Fatal("preparación sintética fallida")
	}
}
