package contrastecopias

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	domain "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

func TestConfiguracionReferenciasObjetoGrande(t *testing.T) {
	c := limitesPrueba()
	c.ObjetosGrandesSemanticos = true
	c.ReferenciasObjetosGrandes = []ReferenciaObjetoGrande{{"public", "archivos", "referencia", ""}}
	l, e := Nuevo(c)
	if e != nil {
		t.Fatal(e)
	}
	c.ReferenciasObjetosGrandes[0].Columna = "otra"
	if !l.limites.ObjetosGrandesSemanticos || l.limites.ReferenciasObjetosGrandes[0].Columna != "referencia" {
		t.Fatal("configuración mutable desde fuera")
	}
	for _, bad := range [][]ReferenciaObjetoGrande{{{"", "archivos", "ref", ""}}, {{"public", "archivos", "ref", ""}, {"public", "archivos", "ref", ""}}, {{"public", strings.Repeat("ñ", 32), "ref", ""}}} {
		c.ReferenciasObjetosGrandes = bad
		if _, e := Nuevo(c); e == nil {
			t.Fatal("referencia no válida aceptada")
		}
	}
	c = limitesPrueba()
	c.ReferenciasObjetosGrandes = []ReferenciaObjetoGrande{{"public", "archivos", "ref", ""}}
	if _, e := Nuevo(c); e == nil {
		t.Fatal("referencia habilitada sin modo semántico")
	}
}
func TestTransporteRechazaUTF8Invalido(t *testing.T) {
	out := append([]byte(`{"valor":"`), 0xff)
	out = append(out, []byte(`"}`)...)
	tx := &transporteEjecutor{exec: &execPrueba{out: out}, base: "postgres", maxBytes: 4096, timeout: 1000}
	if _, e := tx.Query(context.Background(), "SELECT 1"); e != errCaptura {
		t.Fatal("UTF8 inválido normalizado como valor válido")
	}
}

func TestSelloGrandeIndependienteDeFragmentos(t *testing.T) {
	meta := `["810001","propietario",[],"5"]`
	a := nuevoSelloGrande(meta)
	a.Write([]byte{0, 0, 3, 4, 0})
	b := nuevoSelloGrande(meta)
	b.Write([]byte{0})
	b.Write([]byte{0, 3})
	b.Write([]byte{4, 0})
	if !bytes.Equal(a.Sum(nil), b.Sum(nil)) {
		t.Fatal("fragmentación física altera sello lógico")
	}
	c := nuevoSelloGrande(strings.Replace(meta, "810001", "810002", 1))
	c.Write([]byte{0, 0, 3, 4, 0})
	if bytes.Equal(a.Sum(nil), c.Sum(nil)) {
		t.Fatal("identidad lógica no conservada")
	}
}

func TestObjetosGrandesSemanticosPG18(t *testing.T) {
	name := os.Getenv("VEC_CS06_CONTENEDOR_ENSAYO")
	if name == "" {
		t.Skip("ensayo PostgreSQL aislado no solicitado")
	}
	if name != "vec-cs06-lector-sol-20261001" {
		t.Fatal("recurso no propio")
	}
	x := dockerEnsayo{name: name}
	if _, e := x.ComprobarExclusion(context.Background()); e != nil {
		t.Fatal(e)
	}
	sqlEnsayo(t, x, `CREATE ROLE cs06_lo_sintetico;
CREATE TABLE public.archivos(id int4 PRIMARY KEY,referencia oid);
SELECT lo_create(810001); SELECT lo_put(810001,4096,decode('00','hex'));
SELECT lo_create(810002);
INSERT INTO public.archivos VALUES(1,810001),(2,0),(3,NULL);`)
	config := limitesPrueba()
	config.ObjetosGrandesSemanticos = true
	config.ReferenciasObjetosGrandes = []ReferenciaObjetoGrande{{"public", "archivos", "referencia", ""}}
	l, e := Nuevo(config)
	if e != nil {
		t.Fatal(e)
	}
	capture := func() domain.Snapshot {
		s, e := l.CapturarEjecutor(context.Background(), x, "postgres", x)
		if e != nil {
			t.Fatal(e)
		}
		if !s.Completo || len(domain.Validar(s)) != 0 {
			t.Fatalf("modo LO no completo: %v", s.Motivos)
		}
		return s
	}
	first := capture()
	var large domain.Objeto
	for _, o := range first.Objetos {
		if o.Clase == "objetos_grandes" && o.Clave == "loid:810001" {
			large = o
		}
	}
	if large.Cantidad != 4097 || len(large.SHA256) != 64 {
		t.Fatal("tamaño o identidad lógica ausente")
	}
	if _, e := hex.DecodeString(large.SHA256); e != nil {
		t.Fatal(e)
	}
	beforePages := paginasEnsayo(t, x)
	// pg_dump/psql restaura la identidad LO referenciada y cambia OID internos de
	// tablas; los huecos cero se materializan, sin alterar bytes lógicos.
	roundtripGrandeEnsayo(t, x)
	if afterPages := paginasEnsayo(t, x); afterPages <= beforePages {
		t.Fatal("ensayo no cambió representación de páginas")
	}
	if r := domain.Comparar(first, capture()); r.Estado != domain.Igual {
		t.Fatalf("restauración lógica alteró evidencia: %v", r.Razones)
	}
	for _, scenario := range []struct{ name, change, restore, class string }{
		{"byte", `SELECT lo_put(810001,4096,decode('01','hex'))`, `SELECT lo_put(810001,4096,decode('00','hex'))`, "objetos_grandes"},
		{"propietario", `ALTER LARGE OBJECT 810001 OWNER TO cs06_lo_sintetico`, `ALTER LARGE OBJECT 810001 OWNER TO postgres`, "objetos_grandes"},
		{"acl", `GRANT SELECT ON LARGE OBJECT 810001 TO cs06_lo_sintetico`, `REVOKE SELECT ON LARGE OBJECT 810001 FROM cs06_lo_sintetico`, "objetos_grandes"},
		{"objeto_ausente", `SELECT lo_unlink(810002)`, `SELECT lo_create(810002)`, "objetos_grandes"},
		{"referencia", `UPDATE public.archivos SET referencia=810002 WHERE id=1`, `UPDATE public.archivos SET referencia=810001 WHERE id=1`, "tablas"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			sqlEnsayo(t, x, scenario.change)
			s := capture()
			r := domain.Comparar(first, s)
			if r.Estado != domain.Diferente {
				t.Fatal("cambio LO no detectado")
			}
			found := false
			for _, reason := range r.Razones {
				if reason.Clase == scenario.class {
					found = true
				}
			}
			if !found {
				t.Fatal("clase divergente ausente")
			}
			sqlEnsayo(t, x, scenario.restore)
		})
	}
	sqlEnsayo(t, x, `UPDATE public.archivos SET referencia=810099 WHERE id=1`)
	dangling, e := l.CapturarEjecutor(context.Background(), x, "postgres", x)
	if e != nil {
		t.Fatal(e)
	}
	if dangling.Completo || domain.Comparar(first, dangling).Estado != domain.NoComprobable {
		t.Fatal("referencia huérfana admitida")
	}
	sqlEnsayo(t, x, `UPDATE public.archivos SET referencia=810001 WHERE id=1`)
	undeclared := config
	undeclared.ReferenciasObjetosGrandes = nil
	reader, _ := Nuevo(undeclared)
	s, e := reader.CapturarEjecutor(context.Background(), x, "postgres", x)
	if e != nil || s.Completo {
		t.Fatal("oid no declarado admitido")
	}
	absent := config
	absent.ReferenciasObjetosGrandes = []ReferenciaObjetoGrande{{"public", "inexistente", "referencia", ""}}
	reader, _ = Nuevo(absent)
	s, e = reader.CapturarEjecutor(context.Background(), x, "postgres", x)
	if e != nil || s.Completo {
		t.Fatal("declaración sin columna observada admitida")
	}
	t.Log("PG18.4 real: LO pg_dump/psql igual con páginas distintas, loid y referencias conservados; byte/owner/ACL/ausencia/referencia divergentes; dangling y oid sin declarar no_comprobable")
}

func paginasEnsayo(t *testing.T, x dockerEnsayo) int64 {
	t.Helper()
	out, e := x.EjecutarPostgreSQL(context.Background(), "psql", []string{"-X", "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1", "--dbname=postgres"}, []byte(`SELECT count(*) FROM pg_largeobject WHERE loid=810001;`), 4096)
	if e != nil {
		t.Fatal(e)
	}
	n, e := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if e != nil {
		t.Fatal("recuento interno no disponible")
	}
	return n
}
func roundtripGrandeEnsayo(t *testing.T, x dockerEnsayo) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "/usr/bin/docker", "exec", "--user", "postgres", x.name, "pg_dump", "--dbname=postgres", "--format=plain", "--clean", "--if-exists")
	dump, e := cmd.Output()
	if e != nil || len(dump) > 1<<20 {
		t.Fatal("volcado sintético no disponible")
	}
	if _, e = x.EjecutarPostgreSQL(context.Background(), "psql", []string{"-X", "-q", "-v", "ON_ERROR_STOP=1", "--dbname=postgres"}, dump, 1<<20); e != nil {
		t.Fatal("restauración sintética no disponible")
	}
}
