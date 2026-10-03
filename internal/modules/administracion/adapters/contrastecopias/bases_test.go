package contrastecopias

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	domain "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

func TestConfiguracionBasesInventariadas(t *testing.T) {
	c := limitesPrueba()
	c.BasesInventariadas = []string{"vec_cs11", "postgres", "template0", "template1"}
	c.ObjetosGrandesSemanticos = true
	c.ReferenciasObjetosGrandes = []ReferenciaObjetoGrande{{"public", "archivos", "referencia", "vec_cs11"}}
	l, e := Nuevo(c)
	if e != nil {
		t.Fatal(e)
	}
	c.BasesInventariadas[0] = "otra"
	c.ReferenciasObjetosGrandes[0].Base = "otra"
	if !contieneBase(l.limites.BasesInventariadas, "vec_cs11") || l.limites.ReferenciasObjetosGrandes[0].Base != "vec_cs11" {
		t.Fatal("ámbito mutable desde configuración externa")
	}
	for _, bases := range [][]string{{"postgres", "postgres"}, {"postgres", "host=privado"}, {"postgres", "postgres://privado"}} {
		c.BasesInventariadas = bases
		c.ReferenciasObjetosGrandes = nil
		if _, e := Nuevo(c); e == nil {
			t.Fatal("ámbito no válido admitido")
		}
	}
	c.BasesInventariadas = []string{"postgres"}
	c.ReferenciasObjetosGrandes = []ReferenciaObjetoGrande{{"public", "archivos", "ref", ""}}
	if _, e := Nuevo(c); e == nil {
		t.Fatal("referencia multibase sin base admitida")
	}
	if _, e = l.Capturar(context.Background()); e == nil || e.Error() != "captura_multibase_requiere_ejecutor" {
		t.Fatal("DSN no soportado abrió captura multibase")
	}
}
func TestCombinarBasesAislaIdentidades(t *testing.T) {
	l, _ := Nuevo(limitesPrueba())
	one := snapshotMinimoBases()
	two := snapshotMinimoBases()
	one.Objetos = append(one.Objetos, domain.Objeto{Clase: "tablas", Clave: `"public"."datos"`, Cantidad: 1, SHA256: strings.Repeat("a", 64)})
	two.Objetos = append(two.Objetos, domain.Objeto{Clase: "tablas", Clave: `"public"."datos"`, Cantidad: 1, SHA256: strings.Repeat("b", 64)})
	for i, o := range one.Objetos {
		if o.Clase == "tablas" && o.Clave == "inventario" {
			one.Objetos[i] = domain.Resumir("tablas", one.Objetos)
		}
	}
	for i, o := range two.Objetos {
		if o.Clase == "tablas" && o.Clave == "inventario" {
			two.Objetos[i] = domain.Resumir("tablas", two.Objetos)
		}
	}
	a, e := l.combinarBases([]baseCapturada{{"postgres", "props_a", one}, {"vec_cs11", "props_b", two}})
	if e != nil || len(domain.Validar(a)) != 0 {
		t.Fatal("composición multibase no válida")
	}
	b, e := l.combinarBases([]baseCapturada{{"vec_cs11", "props_b", two}, {"postgres", "props_a", one}})
	if e != nil || domain.Comparar(a, b).Estado != domain.Igual {
		t.Fatal("orden de bases alteró igualdad")
	}
	found := map[string]bool{}
	for _, o := range a.Objetos {
		if o.Clase == "tablas" && o.Clave != "inventario" {
			found[o.Clave] = true
		}
	}
	if !found[`"postgres"."public"."datos"`] || !found[`"vec_cs11"."public"."datos"`] {
		t.Fatal("dos tablas homónimas se colisionaron")
	}
	limited := limitesPrueba()
	limited.MaxObjetos = 9
	small, _ := Nuevo(limited)
	if _, e = small.combinarBases([]baseCapturada{{"postgres", "p", one}, {"vec_cs11", "q", two}}); e != errLimite {
		t.Fatal("límite objetos se multiplicó por base")
	}
}
func TestBaseNoConectableSinProcedenciaNoSeAsume(t *testing.T) {
	l, _ := Nuevo(limitesPrueba())
	p := &presupuestoCaptura{objetos: 8}
	s, e := l.capturarNoConectable(context.Background(), "template0", strings.Repeat("a", 64), strings.Repeat("b", 64), p, nil)
	if e != nil || s.Completo {
		t.Fatal("base no conectable asumida vacía")
	}
	f := fuentePruebaBases{evidence: EvidenciaBaseNoConectable{Snapshot: snapshotMinimoBases(), Nombre: "template0", PropiedadesSHA256: strings.Repeat("a", 64), SelloExclusion: strings.Repeat("b", 64), InicializacionSHA256: strings.Repeat("c", 64), SnapshotSHA256: strings.Repeat("d", 64), FilasLeidas: 10, BytesLeidos: 10000}}
	s, e = l.capturarNoConectable(context.Background(), "template0", strings.Repeat("a", 64), strings.Repeat("b", 64), p, f)
	if e != nil || s.Completo {
		t.Fatal("hash de material no observado admitido")
	}
	b, _ := json.Marshal(f.evidence.Snapshot)
	f.evidence.SnapshotSHA256 = shaBytes(b)
	f.evidence.PropiedadesSHA256 = strings.Repeat("e", 64)
	s, e = l.capturarNoConectable(context.Background(), "template0", strings.Repeat("a", 64), strings.Repeat("b", 64), p, f)
	if e != nil || s.Completo {
		t.Fatal("procedencia de otra base admitida")
	}
}
func snapshotMinimoBases() domain.Snapshot {
	s := domain.Snapshot{Version: 1, PostgreSQL: "18.4", Completo: true, Motivos: []string{}, Objetos: []domain.Objeto{}}
	for _, clase := range []string{"esquema", "roles", "acl", "extensiones", "privilegios_defecto"} {
		s.Objetos = append(s.Objetos, domain.Objeto{Clase: clase, Clave: "inventario", SHA256: huella(nil)})
	}
	for _, clase := range []string{"tablas", "secuencias", "objetos_grandes"} {
		s.Objetos = append(s.Objetos, domain.Resumir(clase, s.Objetos))
	}
	return s
}

type fuentePruebaBases struct{ evidence EvidenciaBaseNoConectable }

func (f fuentePruebaBases) CapturarBase(context.Context, SolicitudBaseNoConectable) (EvidenciaBaseNoConectable, error) {
	return f.evidence, nil
}

func TestMultibasePG18Aislado(t *testing.T) {
	name := os.Getenv("VEC_CS06_CONTENEDOR_MULTIBASE_ENSAYO")
	if name == "" {
		t.Skip("ensayo multibase PostgreSQL aislado no solicitado")
	}
	if !baseAdmitida.MatchString(name) {
		t.Fatal("referencia de ensayo no admitida")
	}
	x := dockerEnsayo{name: name}
	guard := exclusionACLEnsayo{x}
	if _, err := guard.ComprobarExclusion(context.Background()); err != nil {
		t.Fatal("recurso sintético propio no comprobado")
	}
	sqlEnsayo(t, x, `CREATE ROLE cs06_multi_sintetico; CREATE DATABASE cs06_auxiliar;`)
	preparar := func(base string) {
		sqlBaseMultibaseEnsayo(t, x, base, `CREATE TABLE public.datos(id int4,valor text); INSERT INTO public.datos VALUES(1,'Lucía Morales');`)
	}
	preparar("postgres")
	preparar("cs06_auxiliar")
	cfg := limitesPrueba()
	cfg.BasesInventariadas = []string{"template1", "postgres", "cs06_auxiliar", "template0"}
	cfg.TiempoMaximo = 90 * time.Second
	l, err := Nuevo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	capturar := func() domain.Snapshot {
		t.Helper()
		s, err := l.CapturarEjecutor(context.Background(), x, "postgres", guard)
		if err != nil || s.Completo || !contieneBase(s.Motivos, "base_no_conectable_sin_evidencia") {
			t.Fatalf("template0 sin evidencia debe quedar no comprobable: %v %v", err, s.Motivos)
		}
		return s
	}
	origen := capturar()
	buscar := func(s domain.Snapshot, clase, clave string) string {
		t.Helper()
		for _, o := range s.Objetos {
			if o.Clase == clase && o.Clave == clave {
				return o.SHA256
			}
		}
		t.Fatalf("objeto no inventariado: %s", clase)
		return ""
	}
	principal := buscar(origen, "tablas", `"postgres"."public"."datos"`)
	auxiliar := buscar(origen, "tablas", `"cs06_auxiliar"."public"."datos"`)
	sqlBaseMultibaseEnsayo(t, x, "cs06_auxiliar", `UPDATE public.datos SET valor='Andrés Romero' WHERE id=1`)
	cambio := capturar()
	if buscar(cambio, "tablas", `"postgres"."public"."datos"`) != principal || buscar(cambio, "tablas", `"cs06_auxiliar"."public"."datos"`) == auxiliar {
		t.Fatal("contenido de bases homónimas cruzado u omitido")
	}
	sqlBaseMultibaseEnsayo(t, x, "cs06_auxiliar", `UPDATE public.datos SET valor='Lucía Morales' WHERE id=1`)
	sqlEnsayo(t, x, `GRANT SET ON PARAMETER statement_timeout TO cs06_multi_sintetico`)
	if buscar(origen, "acl", "inventario") == buscar(capturar(), "acl", "inventario") {
		t.Fatal("ACL compartida de parámetros omitida")
	}
	sqlEnsayo(t, x, `REVOKE SET ON PARAMETER statement_timeout FROM cs06_multi_sintetico`)
	sqlBaseMultibaseEnsayo(t, x, "cs06_auxiliar", `REVOKE USAGE ON LANGUAGE plpgsql FROM PUBLIC; REVOKE ALL ON LANGUAGE plpgsql FROM postgres`)
	vacia := capturar()
	sqlBaseMultibaseEnsayo(t, x, "cs06_auxiliar", `ALTER LANGUAGE plpgsql OWNER TO cs06_multi_sintetico`)
	if buscar(vacia, "acl", "inventario") == buscar(capturar(), "acl", "inventario") {
		t.Fatal("propietario de lenguaje con ACL vacía omitido")
	}
	sqlBaseMultibaseEnsayo(t, x, "cs06_auxiliar", `ALTER LANGUAGE plpgsql OWNER TO postgres; GRANT ALL ON LANGUAGE plpgsql TO postgres; GRANT USAGE ON LANGUAGE plpgsql TO PUBLIC`)
	// pg_dump restaura la base auxiliar con propietarios y restricciones intactos.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dump, err := exec.CommandContext(ctx, "/usr/bin/docker", "exec", "--user", "postgres", name, "pg_dump", "--dbname=cs06_auxiliar", "--format=plain", "--clean", "--if-exists", "--create").Output()
	if err != nil || len(dump) > 1<<20 {
		t.Fatal("volcado sintético no disponible")
	}
	if _, err = x.EjecutarPostgreSQL(ctx, "psql", []string{"-X", "-q", "-v", "ON_ERROR_STOP=1", "--dbname=postgres"}, dump, 1<<20); err != nil {
		t.Fatal("restauración sintética fallida")
	}
	if buscar(capturar(), "tablas", `"cs06_auxiliar"."public"."datos"`) != auxiliar {
		t.Fatal("restauración auxiliar no conserva contenido")
	}
	orden := cfg
	orden.BasesInventariadas = []string{"cs06_auxiliar", "template0", "template1", "postgres"}
	otro, _ := Nuevo(orden)
	s, err := otro.CapturarEjecutor(context.Background(), x, "postgres", guard)
	actual := capturar()
	a, _ := json.Marshal(actual)
	b, _ := json.Marshal(s)
	if err != nil || string(a) != string(b) {
		t.Fatal("orden configurado altera evidencia")
	}
	limitado := cfg
	limitado.MaxObjetos = 9
	small, _ := Nuevo(limitado)
	if _, err = small.CapturarEjecutor(context.Background(), x, "postgres", guard); err != errLimite {
		t.Fatal("presupuesto objetos se multiplicó por base")
	}
	sqlEnsayo(t, x, `CREATE DATABASE cs06_fuera_ambito`)
	incompleto, err := l.CapturarEjecutor(context.Background(), x, "postgres", guard)
	if err != nil || incompleto.Completo || !contieneBase(incompleto.Motivos, "bases_no_inventariadas") {
		t.Fatal("base fuera del ámbito omitida silenciosamente")
	}
	t.Log("PG18: bases homónimas separadas; cambio de contenido/ACL/owner detectado; restore auxiliar conserva contenido; orden estable; límite global; template0 y base no declarada no comprobables")
}

func sqlBaseMultibaseEnsayo(t *testing.T, x dockerEnsayo, base, sql string) {
	t.Helper()
	if _, err := x.EjecutarPostgreSQL(context.Background(), "psql", []string{"-X", "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1", "--dbname=" + base}, []byte(sql), 1<<20); err != nil {
		t.Fatal("preparación sintética de base fallida")
	}
}

func TestPresupuestoCompartidoFilasBytes(t *testing.T) {
	for _, tipo := range []string{"filas", "bytes"} {
		t.Run(tipo, func(t *testing.T) {
			cfg := limitesPrueba()
			l, _ := Nuevo(cfg)
			p := &presupuestoCaptura{objetos: 8}
			if tipo == "filas" {
				p.filas = cfg.MaxFilas
			} else {
				p.bytes = cfg.MaxBytes
			}
			x := &execPrueba{out: []byte("{\"v\":\"dato\"}\n")}
			tx := &transporteEjecutor{exec: x, base: "postgres", maxBytes: cfg.MaxBytes, timeout: 1000}
			c := &captura{tx: tx, l: l, presupuesto: p, filas: p.filas, bytes: p.bytes}
			if _, err := c.leer(context.Background(), "SELECT 'dato'::text"); err != errLimite {
				t.Fatal("segunda base ignora presupuesto consumido")
			}
		})
	}
}
