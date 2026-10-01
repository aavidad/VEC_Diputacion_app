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
	esperarPostgresEnsayo(t, x)
	// Builder de datos exclusivamente sintéticos, antes de la ventana de captura.
	sqlEnsayo(t, x, `CREATE ROLE cs06_multi_sintetico; CREATE DATABASE vec_cs11;`)
	sqlBaseEnsayo(t, x, "vec_cs11", `CREATE TABLE public.datos(id int4,valor text); INSERT INTO public.datos VALUES(1,'sintético');`)
	cfg := limitesPrueba()
	cfg.BasesInventariadas = []string{"vec_cs11", "template1", "postgres", "template0"}
	cfg.TiempoMaximo = 90 * time.Second
	cfg.MaxBytes = 32 << 20
	l, e := Nuevo(cfg)
	if e != nil {
		t.Fatal(e)
	}
	// El provider de ensayo observa una copia real de template0 mediante el motor,
	// sin cambiar datallowconn ni escribir en template0. El material se conserva
	// y el clon se retira antes de empezar el ámbito definitivo.
	source := observarMaterialTemplateEnsayo(t, x, cfg)
	capture := func() domain.Snapshot {
		s, e := l.CapturarEjecutorConFuente(context.Background(), x, "postgres", x, source)
		if e != nil {
			t.Fatal(e)
		}
		if !s.Completo || len(domain.Validar(s)) != 0 {
			t.Fatalf("inventario multibase incompleto: %v", s.Motivos)
		}
		return s
	}
	first := capture()
	without, e := l.CapturarEjecutor(context.Background(), x, "postgres", x)
	if e != nil || without.Completo || domain.Comparar(first, without).Estado != domain.NoComprobable {
		t.Fatal("template0 sin proveedor se declaró completo")
	}
	roundtripBaseEnsayo(t, x, "vec_cs11")
	if r := domain.Comparar(first, capture()); r.Estado != domain.Igual {
		t.Fatalf("restauración nombrada cambió inventario completo: %v", r.Razones)
	}
	sqlEnsayo(t, x, `CREATE TABLE public.mantenimiento(id int4,valor text); INSERT INTO public.mantenimiento VALUES(1,'antes');`)
	maintenance := capture()
	if domain.Comparar(first, maintenance).Estado != domain.Diferente {
		t.Fatal("datos de mantenimiento omitidos")
	}
	sqlEnsayo(t, x, `UPDATE public.mantenimiento SET valor='después' WHERE id=1`)
	changed := capture()
	if !claseDifiere(domain.Comparar(maintenance, changed), "tablas") {
		t.Fatal("celda de mantenimiento no contrastada")
	}
	sqlEnsayo(t, x, `UPDATE public.mantenimiento SET valor='antes' WHERE id=1; GRANT SELECT ON public.mantenimiento TO cs06_multi_sintetico`)
	changed = capture()
	if !claseDifiere(domain.Comparar(maintenance, changed), "acl") {
		t.Fatal("ACL de mantenimiento no contrastado")
	}
	sqlEnsayo(t, x, `REVOKE SELECT ON public.mantenimiento FROM cs06_multi_sintetico`)
	sqlBaseEnsayo(t, x, "template1", `CREATE TABLE public.estado_plantilla(valor text); INSERT INTO public.estado_plantilla VALUES('observado')`)
	changed = capture()
	if !claseDifiere(domain.Comparar(maintenance, changed), "tablas") {
		t.Fatal("contenido plantilla conectable omitido")
	}
	sqlEnsayo(t, x, `CREATE DATABASE cs06_extra_sintetica`)
	unlisted, e := l.CapturarEjecutorConFuente(context.Background(), x, "postgres", x, source)
	if e != nil || unlisted.Completo || domain.Comparar(first, unlisted).Estado != domain.NoComprobable {
		t.Fatal("base fuera del ámbito fue omitida silenciosamente")
	}
	sqlEnsayo(t, x, `DROP DATABASE cs06_extra_sintetica`)
	// El conjunto consume un presupuesto común: este límite permite la primera
	// base aislada, pero no la suma de todas las bases inventariadas.
	limited := cfg
	limited.MaxBytes = source.evidence.BytesLeidos + 2000
	small, _ := Nuevo(limited)
	if _, e = small.CapturarEjecutorConFuente(context.Background(), x, "postgres", x, source); e != errLimite {
		t.Fatal("presupuesto bytes se multiplicó por base")
	}
	t.Log("PG18.4 real: postgres/template0/template1/vec_cs11 completos con copia observada pre-ventana; restore nombrado igual; mantenimiento datos/ACL y plantilla difieren; base adicional y provider ausente no_comprobable; presupuesto global limitado")
}
func claseDifiere(r domain.Resultado, clase string) bool {
	if r.Estado != domain.Diferente {
		return false
	}
	for _, v := range r.Razones {
		if v.Clase == clase {
			return true
		}
	}
	return false
}

type fuenteTemplateEnsayo struct{ evidence EvidenciaBaseNoConectable }

func (f fuenteTemplateEnsayo) CapturarBase(_ context.Context, req SolicitudBaseNoConectable) (EvidenciaBaseNoConectable, error) {
	if req.Nombre != f.evidence.Nombre || req.PropiedadesSHA256 != f.evidence.PropiedadesSHA256 || req.SelloExclusion != f.evidence.SelloExclusion {
		return EvidenciaBaseNoConectable{}, errCaptura
	}
	return f.evidence, nil
}
func observarMaterialTemplateEnsayo(t *testing.T, x dockerEnsayo, cfg Configuracion) fuenteTemplateEnsayo {
	t.Helper()
	sqlEnsayo(t, x, `CREATE DATABASE cs06_material_template TEMPLATE template0`)
	cloneCfg := cfg
	cloneCfg.BasesInventariadas = nil
	cloneCfg.ReferenciasObjetosGrandes = nil
	reader, _ := Nuevo(cloneCfg)
	p := &presupuestoCaptura{objetos: 8}
	tx := &transporteEjecutor{exec: x, base: "cs06_material_template", maxBytes: cfg.MaxBytes, timeout: cfg.TiempoMaximo.Milliseconds()}
	snapshot, e := reader.capturarSQLAcotada(context.Background(), tx, x, true, p, "template0")
	if e != nil || !snapshot.Completo {
		t.Fatal("material lógico de inicialización no observable")
	}
	sourceTx := &transporteEjecutor{exec: x, base: "postgres", maxBytes: cfg.MaxBytes, timeout: cfg.TiempoMaximo.Milliseconds()}
	properties, e := reader.leerBases(context.Background(), sourceTx, &presupuestoCaptura{objetos: 8})
	if e != nil {
		t.Fatal(e)
	}
	props := ""
	for _, b := range properties {
		if b.Nombre == "template0" {
			if b.Conectable {
				t.Fatal("template0 se activó durante el ensayo")
			}
			props = shaBytes(b.Propiedades)
		}
	}
	if props == "" {
		t.Fatal("template0 no observado")
	}
	sqlEnsayo(t, x, `DROP DATABASE cs06_material_template`)
	seal, e := x.ComprobarExclusion(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	encoded, _ := json.Marshal(snapshot)
	receipt, _ := json.Marshal([]string{seal, props, shaBytes(encoded), "copia_motor_template0_observada_pre_ventana"})
	return fuenteTemplateEnsayo{EvidenciaBaseNoConectable{Snapshot: snapshot, Nombre: "template0", PropiedadesSHA256: props, SelloExclusion: seal, InicializacionSHA256: shaBytes(receipt), SnapshotSHA256: shaBytes(encoded), FilasLeidas: p.filas, BytesLeidos: p.bytes}}
}
func sqlBaseEnsayo(t *testing.T, x dockerEnsayo, base, sql string) {
	t.Helper()
	if _, e := x.EjecutarPostgreSQL(context.Background(), "psql", []string{"-X", "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1", "--dbname=" + base}, []byte(sql), 1<<20); e != nil {
		t.Fatal("preparación de base sintética fallida")
	}
}
func roundtripBaseEnsayo(t *testing.T, x dockerEnsayo, base string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "/usr/bin/docker", "exec", "--user", "postgres", x.name, "pg_dump", "--dbname="+base, "--format=plain", "--clean", "--if-exists", "--create")
	dump, e := cmd.Output()
	if e != nil || len(dump) > 1<<20 {
		t.Fatal("volcado sintético de base no disponible")
	}
	if _, e = x.EjecutarPostgreSQL(context.Background(), "psql", []string{"-X", "-q", "-v", "ON_ERROR_STOP=1", "--dbname=postgres"}, dump, 1<<20); e != nil {
		t.Fatal("restauración nombrada sintética fallida")
	}
}

func esperarPostgresEnsayo(t *testing.T, x dockerEnsayo) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		if _, e := x.EjecutarPostgreSQL(ctx, "psql", []string{"-X", "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1", "--dbname=postgres"}, []byte("SELECT 1;"), 4096); e == nil {
			return
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			t.Fatal("PostgreSQL sintético no disponible")
		case <-timer.C:
		}
	}
}
