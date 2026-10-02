package contrastecopias

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func tiposCanonPrueba() *canonTipos {
	ct := &canonTipos{tipos: map[uint32]*tipoCanon{}}
	for _, t := range []tipoCanon{
		{OID: 25, Esquema: "pg_catalog", Nombre: "text", Clase: "b", SalidaSegura: true, ModificadorSeguro: true},
		{OID: 23, Esquema: "pg_catalog", Nombre: "int4", Clase: "b", SalidaSegura: true, ModificadorSeguro: true},
		{OID: 26, Esquema: "pg_catalog", Nombre: "oid", Clase: "b", SalidaSegura: true, ModificadorSeguro: true},
		{OID: 16384, Esquema: "public", Nombre: "estado", Clase: "e", SalidaSegura: true, ModificadorSeguro: true},
		{OID: 16385, Esquema: "public", Nombre: "dominio", Clase: "d", Base: 25, SalidaSegura: true, ModificadorSeguro: true},
		{OID: 16386, Esquema: "public", Nombre: "fila", Clase: "c", Relacion: 17000, SalidaSegura: true, ModificadorSeguro: true, campos: []campoCanon{{Nombre: "texto", Tipo: 25}, {Nombre: "estado", Tipo: 16384}, {Nombre: "dominio", Tipo: 16385}}},
		{OID: 16387, Esquema: "public", Nombre: "_fila", Clase: "b", Elemento: 16386, Categoria: "A", SalidaSegura: true, ModificadorSeguro: true, ArraySeguro: true},
		{OID: 16388, Esquema: "public", Nombre: "personalizado", Clase: "b", SalidaSegura: false, ModificadorSeguro: true},
	} {
		v := t
		ct.tipos[v.OID] = &v
	}
	return ct
}

type grafoTiposPrueba struct {
	tipos     []tipoCanon
	campos    []campoCanon
	consultas int
}

func (g *grafoTiposPrueba) EjecutarPostgreSQL(_ context.Context, _ string, _ []string, input []byte, _ int) ([]byte, error) {
	g.consultas++
	var rows []any
	if strings.Contains(string(input), "'safe_typmod'") {
		for _, t := range g.tipos {
			rows = append(rows, t)
		}
	} else {
		for _, f := range g.campos {
			rows = append(rows, f)
		}
	}
	var out []byte
	for _, row := range rows {
		v, _ := json.Marshal(row)
		b, _ := json.Marshal(map[string]string{"v": string(v)})
		out = append(out, b...)
		out = append(out, '\n')
	}
	return out, nil
}

func TestPrepararTiposBloqueaDeparseDeSalidaDesconocida(t *testing.T) {
	for _, scenario := range []struct {
		nombre  string
		tipo    tipoCanon
		bloquea bool
	}{
		{"builtin", tipoCanon{OID: 25, Esquema: "pg_catalog", Nombre: "text", Clase: "b", SalidaSegura: true, ModificadorSeguro: true}, false},
		{"enum", tipoCanon{OID: 17000, Esquema: "public", Nombre: "estado", Clase: "e", SalidaSegura: true, ModificadorSeguro: true}, false},
		{"base_salida_user", tipoCanon{OID: 17000, Esquema: "public", Nombre: "opaco", Clase: "b", ModificadorSeguro: true}, true},
		{"base_salida_builtin", tipoCanon{OID: 17000, Esquema: "public", Nombre: "opaco", Clase: "b", SalidaSegura: true, ModificadorSeguro: true}, true},
		{"typmod_user", tipoCanon{OID: 17000, Esquema: "public", Nombre: "estado", Clase: "e", SalidaSegura: true}, true},
	} {
		t.Run(scenario.nombre, func(t *testing.T) {
			x := &grafoTiposPrueba{tipos: []tipoCanon{scenario.tipo}}
			c := &captura{l: &Lector{limites: limitesPrueba()}, tx: &transporteEjecutor{exec: x, base: "postgres", maxBytes: 8 << 20, timeout: 1000}}
			ct, err := c.prepararTipos(context.Background())
			if err != nil || ct.NoDeparseSeguro() != scenario.bloquea {
				t.Fatalf("admisión de deparse incorrecta: %v", err)
			}
			if x.consultas != 2 || c.filas != 1 || c.bytes == 0 || len(c.s.Objetos) != 0 {
				t.Fatal("grafo salió del presupuesto o se publicó como evidencia")
			}
		})
	}
}

func TestPrepararCompuestoOrdenaCamposVivosSinPublicarHuecos(t *testing.T) {
	x := &grafoTiposPrueba{tipos: []tipoCanon{
		{OID: 25, Esquema: "pg_catalog", Nombre: "text", Clase: "b", SalidaSegura: true, ModificadorSeguro: true},
		{OID: 17000, Esquema: "public", Nombre: "fila", Clase: "c", Relacion: 18000, SalidaSegura: true, ModificadorSeguro: true},
	}, campos: []campoCanon{{Relacion: 18000, Posicion: 3, Nombre: "b", Tipo: 25}, {Relacion: 18000, Posicion: 1, Nombre: "a", Tipo: 25}}}
	c := &captura{l: &Lector{limites: limitesPrueba()}, tx: &transporteEjecutor{exec: x, base: "postgres", maxBytes: 8 << 20, timeout: 1000}}
	ct, err := c.prepararTipos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := ct.canonColumna("public", "datos", "valor", 17000, "")
	if err != nil || !ok || strings.Index(got, `'a','pg_catalog'`) > strings.Index(got, `'b','pg_catalog'`) || strings.Contains(got, "18000") || strings.Contains(got, "17000") {
		t.Fatal("orden vivo o identidad lógica alterados")
	}
	// A restore recreates the live fields contiguously; its canonical SQL stays
	// identical because physical attribute numbers are only traversal keys.
	x.campos[0].Posicion = 2
	ct2, err := c.prepararTipos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got2, ok, err := ct2.canonColumna("public", "datos", "valor", 17000, "")
	if err != nil || !ok || got != got2 {
		t.Fatal("hueco físico alteró la forma canónica")
	}
}

func TestCanonTiposSimplesConservaFrameAnterior(t *testing.T) {
	ct := tiposCanonPrueba()
	got, ok, err := ct.canonColumna("public", "datos", "texto", 25, "")
	if err != nil || !ok || got != `jsonb_build_array('texto','text',"texto"::text)` {
		t.Fatalf("frame simple cambiado: %q %v %v", got, ok, err)
	}
	if _, ok, err = ct.canonColumna("public", "datos", "texto", 25, "v"); err != nil || ok {
		t.Fatal("columna virtual admitida")
	}
	if _, ok, err = ct.canonColumna("public", "datos", "texto", 25, "s"); err != nil || !ok {
		t.Fatal("columna almacenada no admitida")
	}
}

func TestCanonCompuestosSinSalidaOpacaNiNullDeFila(t *testing.T) {
	ct := tiposCanonPrueba()
	got, ok, err := ct.canonColumna("public", "datos", "valor", 16386, "")
	if err != nil || !ok {
		t.Fatalf("compuesto seguro no admitido: %v", err)
	}
	for _, wanted := range []string{`pg_catalog.pg_column_size("valor") IS NULL`, `("valor")."texto"::text`, `pg_catalog.enum_out(("valor")."estado")`, `::"pg_catalog"."text"`, `'domain','public','dominio'`} {
		if !strings.Contains(got, wanted) {
			t.Fatalf("representación estructural ausente: %s", wanted)
		}
	}
	for _, forbidden := range []string{`"valor" IS NULL`, `record_out`, `to_json`, `16386`, `17000`} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("salida opaca o OID: %s", forbidden)
		}
	}
}

func TestCanonArrayConservaDimensionesLimitesOrdenYElementos(t *testing.T) {
	ct := tiposCanonPrueba()
	got, ok, err := ct.canonColumna("public", "datos", "valor", 16387, "")
	if err != nil || !ok {
		t.Fatalf("array seguro no admitido: %v", err)
	}
	for _, wanted := range []string{`pg_catalog.array_ndims("valor")`, `pg_catalog.array_dims("valor")`, `WHEN 0 THEN '[]'::pg_catalog.jsonb`, `WHEN 6 THEN`, `pg_catalog.generate_subscripts("valor",6)`, `ORDER BY vec_cs06_a`, `pg_catalog.pg_column_size(("valor")[`} {
		if !strings.Contains(got, wanted) {
			t.Fatalf("semántica array ausente: %s", wanted)
		}
	}
	if strings.Contains(got, "DISTINCT") || strings.Contains(got, "array_out") || strings.Contains(got, "record_out") {
		t.Fatal("multiplicidad o salida estructural alterada")
	}
}

func TestCanonRechazaTipoDesconocidoAnidadoAntesDeExpresion(t *testing.T) {
	ct := tiposCanonPrueba()
	ct.tipos[16386].campos = append(ct.tipos[16386].campos, campoCanon{Nombre: "opaco", Tipo: 16388})
	for _, oid := range []uint32{16388, 16386, 16387, 999999} {
		got, ok, err := ct.canonColumna("public", "datos", "valor", oid, "")
		if err != nil || ok || got != "" {
			t.Fatalf("tipo desconocido emitió expresión: %d", oid)
		}
	}
	ct.tipos[16388].SalidaSegura = true
	if got, ok, _ := ct.canonColumna("public", "datos", "valor", 16388, ""); ok || got != "" {
		t.Fatal("base user con salida builtin admitida")
	}
}

func TestCanonCicloRechazadoYProfundidadAcotada(t *testing.T) {
	ct := tiposCanonPrueba()
	ct.tipos[16386].campos = append(ct.tipos[16386].campos, campoCanon{Nombre: "ciclo", Tipo: 16386})
	if got, ok, err := ct.canonColumna("public", "datos", "valor", 16386, ""); err != nil || ok || got != "" {
		t.Fatal("ciclo no quedó no admitido")
	}
	previous := uint32(25)
	for id := uint32(20000); id < 20040; id++ {
		ct.tipos[id] = &tipoCanon{OID: id, Esquema: "public", Nombre: "dominio", Clase: "d", Base: previous, SalidaSegura: true, ModificadorSeguro: true}
		previous = id
	}
	if got, ok, err := ct.canonColumna("public", "datos", "valor", previous, ""); err != errLimite || ok || got != "" {
		t.Fatal("profundidad sin límite")
	}
}

func TestCanonOIDExigeDeclaracionExactaSoloEnColumna(t *testing.T) {
	ct := tiposCanonPrueba()
	ct.c = &captura{l: &Lector{limites: Configuracion{ObjetosGrandesSemanticos: true, ReferenciasObjetosGrandes: []ReferenciaObjetoGrande{{Esquema: "public", Tabla: "datos", Columna: "lo"}}}}}
	if _, ok, _ := ct.canonColumna("public", "datos", "lo", 26, ""); !ok {
		t.Fatal("referencia declarada rechazada")
	}
	if got, ok, _ := ct.canonColumna("public", "datos", "otro", 26, ""); ok || got != "" {
		t.Fatal("OID sin declaración admitido")
	}
	ct.tipos[16386].campos = []campoCanon{{Nombre: "lo", Tipo: 26}}
	if got, ok, _ := ct.canonColumna("public", "datos", "lo", 16386, ""); ok || got != "" {
		t.Fatal("OID interno inferido como LO")
	}
	ct.c.l.limites.ObjetosGrandesSemanticos = false
	if _, ok, _ := ct.canonColumna("public", "datos", "lo", 26, ""); ok {
		t.Fatal("modo LO desactivado ignorado")
	}
}

func TestCanonIdentificadoresYParametrosNoSeConfunden(t *testing.T) {
	ct := tiposCanonPrueba()
	ct.tipos[16386].Nombre = `fila$1'"`
	ct.tipos[16386].campos = []campoCanon{{Nombre: `campo$2'"`, Tipo: 25}}
	got, ok, err := ct.canonColumna("public", "datos", `columna$3'"`, 16386, "")
	if err != nil || !ok {
		t.Fatal("identificadores válidos rechazados")
	}
	bound, err := bind("SELECT "+got+", $1", []any{int64(17)})
	if err != nil || !strings.Contains(bound, `"columna$3'"""`) || !strings.Contains(bound, `"campo$2'"""`) || !strings.HasSuffix(bound, ", 17") {
		t.Fatalf("literal o identificador alterado: %v", err)
	}
}

func TestCanonAliasNoOcultaColumnaNiTabla(t *testing.T) {
	ct := tiposCanonPrueba()
	got, ok, err := ct.canonColumna("public", "vec_cs06_a2", "vec_cs06_a1", 16387, "")
	if err != nil || !ok {
		t.Fatal("array no admitido")
	}
	if strings.Contains(got, "AS vec_cs06_a1(") || strings.Contains(got, "AS vec_cs06_a2(") {
		t.Fatal("alias oculta identificador del origen")
	}
}

func TestCanonIndiceInternoNoOcultaColumnaI(t *testing.T) {
	ct := tiposCanonPrueba()
	ct.tipos[1007] = &tipoCanon{OID: 1007, Esquema: "pg_catalog", Nombre: "_int4", Clase: "b", Elemento: 23, Categoria: "A", SalidaSegura: true, ModificadorSeguro: true, ArraySeguro: true}
	ct.tipos[16386].campos = []campoCanon{{Nombre: "elementos", Tipo: 1007}}
	for _, oid := range []uint32{1007, 16386} {
		got, ok, err := ct.canonColumna("public", "datos", "i", oid, "")
		if err != nil || !ok {
			t.Fatalf("array o compuesto de origen i no admitido: %v", err)
		}
		// Every generated index uses its reserved token both as relation and
		// column name. No FROM column named i may shadow the source expression
		// in later lateral dimensions or in the aggregate's element getter.
		for _, forbidden := range []string{`(i)`, `.i`, ` AS i(`} {
			if strings.Contains(got, forbidden) {
				t.Fatalf("índice oculta origen: %s", forbidden)
			}
		}
		for _, wanted := range []string{`AS vec_cs06_a1(vec_cs06_a1)`, `[vec_cs06_a1.vec_cs06_a1]`, `ORDER BY vec_cs06_a1.vec_cs06_a1`, `WHEN 2 THEN`} {
			if !strings.Contains(got, wanted) {
				t.Fatalf("índice reservado no conserva dimensión: %s", wanted)
			}
		}
		if !strings.Contains(got, `pg_catalog.generate_subscripts("i",2)`) && !strings.Contains(got, `pg_catalog.generate_subscripts(("i")."elementos",2)`) {
			t.Fatal("segunda dimensión perdió el origen")
		}
	}
}
