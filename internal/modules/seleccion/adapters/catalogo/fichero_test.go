package catalogo

import (
	"context"
	"strings"
	"testing"
)

func TestCatalogoDeEjemploDelRepositorio(t *testing.T) {
	f, err := NuevoFichero("../../../../../data/demo/reglas/seleccion_convocatorias.ejemplo.demo.json")
	if err != nil {
		t.Fatalf("el catálogo de ejemplo no es válido: %v", err)
	}
	c, err := f.Convocatorias(context.Background())
	if err != nil || len(c) < 1 {
		t.Fatal("sin convocatorias")
	}
	for _, x := range c {
		if !x.MarcaEjemplo || x.Validar() != nil || strings.Contains(x.Ref, "proceso:") {
			t.Fatalf("convocatoria de ejemplo inesperada: %s", x.Ref)
		}
		if _, err := x.HuellaSHA256(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCatalogoRechazaIncoherencias(t *testing.T) {
	base := `{"version_esquema":1,"fuente":{"revision":"r","actualizada_en":"2026-09-26T00:00:00Z","demostracion":%s,"aviso":"a"},
 "catalogo":{"id":"vec.seleccion.convocatorias","version":1,"revision":1,"modulo_id":"seleccion","nombre":"n","fuente_ref":"%s","motivo_creacion":"m"},
 "convocatorias":[%s]}`
	convocatoria := `{"convocatoria_ref":"bolsa-x-2026","titulo":"X","publicada_en":"2026-09-26T00:00:00Z","plazo":{"abre_en":"2026-09-20T00:00:00Z","cierra_en":"2026-10-20T00:00:00Z"},
 "fecha_referencia":"2026-10-20","turnos":[{"clave":"libre","etiqueta":"Libre"}],"requisitos":[],"baremo":{"maximo":"10","redondeo":"mitad_arriba","grupos":[]},
 "numeracion":{"patron":"{anio}/SOL-{numero}","ancho":6}}`
	sustituir := func(demo, fuente, conv string) []byte {
		s := strings.Replace(base, "%s", demo, 1)
		s = strings.Replace(s, "%s", fuente, 1)
		return []byte(strings.Replace(s, "%s", conv, 1))
	}
	if _, err := Parsear(sustituir("true", FuentePaqueteEjemplo, convocatoria)); err != nil {
		t.Fatalf("catálogo mínimo rechazado: %v", err)
	}
	for nombre, bruto := range map[string][]byte{
		"ejemplo sin demostración": sustituir("false", FuentePaqueteEjemplo, convocatoria),
		"repetida":                 sustituir("true", FuentePaqueteEjemplo, convocatoria+","+convocatoria),
		"plazo invertido":          sustituir("true", FuentePaqueteEjemplo, strings.Replace(convocatoria, "2026-10-20T00:00:00Z", "2026-09-01T00:00:00Z", 1)),
		"redondeo desconocido":     sustituir("true", FuentePaqueteEjemplo, strings.Replace(convocatoria, "mitad_arriba", "a_ojo", 1)),
		"campo desconocido":        sustituir("true", FuentePaqueteEjemplo, strings.Replace(convocatoria, `"titulo":"X"`, `"titulo":"X","extra":1`, 1)),
		"sin convocatorias":        sustituir("true", FuentePaqueteEjemplo, ""),
	} {
		if _, err := Parsear(bruto); err == nil {
			t.Fatalf("%s aceptado", nombre)
		}
	}
}
