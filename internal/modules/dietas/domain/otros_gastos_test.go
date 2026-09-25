package domain

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

func calculoSinVehiculoPrueba() CalculoComision {
	regla := reglaTramosPrueba()
	c := CalculoComision{Procedencia: "sin_vehiculo_propio", VersionGrafo: "no_aplica", Motor: "no_aplica", VersionTarifa: regla.VersionTarifaRef, Rotulo: RotuloTarifaProvisional, ReglaRef: regla.ReglaRef, ReglaHuellaSHA256: regla.HuellaSHA256, HoraInicio: "08:00", HoraFin: "12:00", Kilometros: "0.0000", EURPorKM: "0.2600", TramosRuta: []TramoRutaComision{}, Rutas: []RutaCalculadaComision{}}
	for grupo := 1; grupo <= 3; grupo++ {
		c.OpcionesDieta = append(c.OpcionesDieta, OpcionDietaComision{Grupo: grupo, Calculo: CalculoDietasProvisional{Tramos: []TramoDietaProvisional{}, VersionTarifaRef: regla.VersionTarifaRef, Rotulo: RotuloTarifaProvisional, ReglaRef: regla.ReglaRef, ReglaHuellaSHA256: regla.HuellaSHA256}})
	}
	return c
}

func gastoD5Prueba() OtroGastoDeclarado {
	return OtroGastoDeclarado{Tipo: ClaseOtroMedio, TipoGasto: "taxi", CatalogoVersion: VersionCatalogoOtrosGastos, Fecha: "2026-09-23", Concepto: "Taxi estación a sede", ImporteCentimos: 1250, JustificanteRef: "ticket:taxi-0923", JustificanteSHA256: strings.Repeat("b", 64)}
}

func TestOtroGastoD5ExigeCatalogoFechaYJustificante(t *testing.T) {
	if err := gastoD5Prueba().ValidarAlta("2026-09-23", "2026-09-24"); err != nil {
		t.Fatalf("línea D5 válida rechazada: %v", err)
	}
	casos := map[string]func(*OtroGastoDeclarado){
		"tipo fuera del catálogo":   func(o *OtroGastoDeclarado) { o.TipoGasto = "restaurante" },
		"apartado incoherente":      func(o *OtroGastoDeclarado) { o.Tipo = ClaseOtroGasto },
		"versión no publicada":      func(o *OtroGastoDeclarado) { o.CatalogoVersion = "provisional:otros-gastos:20990101" },
		"fecha antes de la salida":  func(o *OtroGastoDeclarado) { o.Fecha = "2026-09-22" },
		"fecha tras el regreso":     func(o *OtroGastoDeclarado) { o.Fecha = "2026-09-25" },
		"fecha inexistente":         func(o *OtroGastoDeclarado) { o.Fecha = "2026-02-30" },
		"sin justificante":          func(o *OtroGastoDeclarado) { o.JustificanteRef, o.JustificanteSHA256 = "", "" },
		"referencia sin huella":     func(o *OtroGastoDeclarado) { o.JustificanteSHA256 = "" },
		"huella en mayúsculas":      func(o *OtroGastoDeclarado) { o.JustificanteSHA256 = strings.Repeat("B", 64) },
		"referencia con espacios":   func(o *OtroGastoDeclarado) { o.JustificanteRef = "ticket taxi" },
		"descripción corta":         func(o *OtroGastoDeclarado) { o.Concepto = "tx" },
		"descripción con salto":     func(o *OtroGastoDeclarado) { o.Concepto = "Taxi\nsede" },
		"importe cero":              func(o *OtroGastoDeclarado) { o.ImporteCentimos = 0 },
		"importe desmesurado":       func(o *OtroGastoDeclarado) { o.ImporteCentimos = 100000001 },
		"forma antigua sin tipo D5": func(o *OtroGastoDeclarado) { o.TipoGasto, o.CatalogoVersion, o.Fecha = "", "", "" },
	}
	for nombre, cambiar := range casos {
		o := gastoD5Prueba()
		cambiar(&o)
		if o.ValidarAlta("2026-09-23", "2026-09-24") == nil {
			t.Errorf("%s: aceptado", nombre)
		}
	}
}

func TestDocumentoSumaOtrosD5YConservaLecturaAnterior(t *testing.T) {
	calculo := calculoSinVehiculoPrueba()
	codigos := []string{"ruta:a", "ruta:b"}
	medio := gastoD5Prueba()
	gasto := OtroGastoDeclarado{Tipo: ClaseOtroGasto, TipoGasto: "aparcamiento", CatalogoVersion: VersionCatalogoOtrosGastos, Fecha: "2026-09-24", Concepto: "Aparcamiento sede", ImporteCentimos: 600, JustificanteRef: "ticket:parking-0924", JustificanteSHA256: strings.Repeat("c", 64)}
	anterior := OtroGastoDeclarado{Tipo: ClaseOtroGasto, Concepto: "Peaje previo a D5", ImporteCentimos: 300}
	doc, err := ConstruirDocumentoComision(calculo, codigos, nil, false, 2, nil, calculo.VersionTarifa, []OtroGastoDeclarado{medio, gasto, anterior})
	if err != nil {
		t.Fatal(err)
	}
	if doc.OtrosCentimos != 2150 || doc.TotalOrientativoCentimos != 2150 || len(doc.Lineas) != 3 {
		t.Fatalf("totales incoherentes: %+v", doc)
	}
	l := doc.Lineas[0]
	if l.TipoGasto != "taxi" || l.CatalogoVersion != VersionCatalogoOtrosGastos || l.Fecha != "2026-09-23" || *l.JustificanteRef != "ticket:taxi-0923" {
		t.Fatalf("línea D5 sin conservar: %+v", l)
	}
	if err := doc.Validar(calculo, codigos); err != nil {
		t.Fatalf("documento D5 no se revalida: %v", err)
	}
	// La línea serializada y la declaración coinciden campo a campo: es lo
	// que PostgreSQL coteja en validar_documento_v2.
	linea, _ := json.Marshal(doc.Lineas[0])
	declarada, _ := json.Marshal(medio)
	var a, b map[string]any
	_ = json.Unmarshal(linea, &a)
	_ = json.Unmarshal(declarada, &b)
	if len(a) != 8 || len(a) != len(b) {
		t.Fatalf("claves de línea %v frente a declaración %v", a, b)
	}
	for k, v := range b {
		if a[k] != v {
			t.Fatalf("campo %s: %v frente a %v", k, a[k], v)
		}
	}
	alterado := doc
	alterado.Lineas = append([]LineaDocumentoComision(nil), doc.Lineas...)
	alterado.Lineas[0].TipoGasto = "aparcamiento"
	if alterado.Validar(calculo, codigos) == nil {
		t.Fatal("tipo cambiado de apartado aceptado")
	}
	incompleto := medio
	incompleto.Fecha = ""
	if _, err := ConstruirDocumentoComision(calculo, codigos, nil, false, 2, nil, calculo.VersionTarifa, []OtroGastoDeclarado{incompleto}); err == nil {
		t.Fatal("línea D5 sin fecha aceptada")
	}
}

// El catálogo de Go es un espejo del publicado por Dietas 000009, que es
// quien lo exige al guardar. Deben coincidir exactamente.
func TestCatalogoOtrosGastosCoincideConMigracion000009(t *testing.T) {
	sql, err := os.ReadFile("../../../../deploy/postgresql/dietas_borradores/migraciones/000009_otros_gastos_justificados.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	fila := regexp.MustCompile(`\('(provisional:otros-gastos:[0-9]{8})','([a-z_]+)','(otro_medio|otro_gasto)'\)`)
	var publicados []TipoOtroGasto
	for _, m := range fila.FindAllStringSubmatch(string(sql), -1) {
		if m[1] != VersionCatalogoOtrosGastos {
			t.Fatalf("versión SQL %s distinta de %s", m[1], VersionCatalogoOtrosGastos)
		}
		publicados = append(publicados, TipoOtroGasto{Codigo: m[2], Clase: m[3]})
	}
	catalogo := CatalogoOtrosGastosVigente()
	if len(publicados) != len(catalogo.Tipos) {
		t.Fatalf("SQL publica %d tipos; Go %d", len(publicados), len(catalogo.Tipos))
	}
	for i := range publicados {
		if publicados[i] != catalogo.Tipos[i] {
			t.Fatalf("tipo %d: SQL %+v, Go %+v", i, publicados[i], catalogo.Tipos[i])
		}
	}
	catalogo.Tipos[0].Codigo = "alterado"
	if CatalogoOtrosGastosVigente().Tipos[0].Codigo == "alterado" {
		t.Fatal("el catálogo no se copia de forma defensiva")
	}
}
