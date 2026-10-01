package domain_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	d "vec-diputacion-granada/internal/modules/provision/domain"
)

func diferenciaConfig(t *testing.T) d.Configuracion {
	t.Helper()
	return concConfig(t, concRegla(t, d.Grado), concRegla(t, d.Antiguedad), concRegla(t, d.Cursos), concRegla(t, d.Titulaciones))
}
func copiarConfig(t *testing.T, c d.Configuracion) d.Configuracion {
	t.Helper()
	datos, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var n d.Configuracion
	if err := json.Unmarshal(datos, &n); err != nil {
		t.Fatal(err)
	}
	return n
}
func compararConfig(t *testing.T, a, n d.Configuracion) d.DiferenciaConfiguracion {
	t.Helper()
	diff, err := d.CompararConfiguraciones(a, n)
	if err != nil {
		t.Fatal(err)
	}
	return diff
}
func buscarCambioConfig(t *testing.T, diff d.DiferenciaConfiguracion, ambito, regla, tramo, campo string) d.CambioConfiguracion {
	t.Helper()
	for _, c := range diff.Cambios {
		if c.Ambito == ambito && c.ReglaID == regla && c.TramoID == tramo && c.Campo == campo {
			return c
		}
	}
	t.Fatalf("falta cambio %s/%s/%s/%s", ambito, regla, tramo, campo)
	return d.CambioConfiguracion{}
}

func TestComparacionConcursosValoresExactosYPoliticas(t *testing.T) {
	a := diferenciaConfig(t)
	n := copiarConfig(t, a)
	n.Version = "revision:alpha"
	n.BasesRef = "bases:sinteticas:rectificadas"
	n.FechaCorte = concFecha(t, "2025-02-01")
	n.VentanaDesde = concFecha(t, "2024-02-01")
	n.MaximoTotal = concPuntos(t, 9_000_000_000_000_000)
	n.Reglas[1].Coeficiente = concPuntos(t, 123456789)
	n.Reglas[1].Maximo = concPuntos(t, 987654321)
	n.Reglas[1].Conversion = &d.Conversion{Metodo: "dias_completos", Divisor: 30}
	n.Reglas[1].Jornada = "integra"
	n.Reglas[0].Tramos[0].Coeficiente = concPuntos(t, 10000001)
	n.Reglas[0].Tramos[0].Maximo = concPuntos(t, 10000002)
	diff := compararConfig(t, a, n)
	if diff.Esquema != "vec.provision.diferencia_configuracion.v1" || diff.Alcance != "comparacion_reglas" || len(diff.Cambios) != 10 {
		t.Fatalf("salida inesperada: %+v", diff)
	}
	c := buscarCambioConfig(t, diff, "regla", a.Reglas[1].ID, "", "coeficiente")
	if c.Anterior.Puntos.Micropuntos() != 1000000 || c.Nuevo.Puntos.Micropuntos() != 123456789 {
		t.Fatal("coeficiente inexacto")
	}
	c = buscarCambioConfig(t, diff, "configuracion", "", "", "maximo_total")
	if c.Nuevo.Puntos.Micropuntos() != 9_000_000_000_000_000 {
		t.Fatal("tope perdió precisión")
	}
	c = buscarCambioConfig(t, diff, "regla", a.Reglas[1].ID, "", "conversion")
	if c.Anterior.Conversion.Metodo != "dias_racionales" || c.Nuevo.Conversion.Divisor != 30 {
		t.Fatal("conversión perdida")
	}
	buscarCambioConfig(t, diff, "tramo", a.Reglas[0].ID, a.Reglas[0].Tramos[0].ID, "maximo")
	datos, err := json.Marshal(diff)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(datos, []byte(`"puntos":"9000000000000000"`)) {
		t.Fatal("puntos no serializados exactamente")
	}
}

func TestComparacionConcursosAltasBajasPorID(t *testing.T) {
	a := diferenciaConfig(t)
	n := copiarConfig(t, a)
	n.Version = "revision:z"
	n.Reglas[2].ID = "regla:curso:renombrada"
	n.Reglas[0].Tramos[0].ID = "tramo:renombrado"
	diff := compararConfig(t, a, n)
	if len(diff.Cambios) != 4 {
		t.Fatalf("altas/bajas inesperadas: %+v", diff.Cambios)
	}
	alta := buscarCambioConfig(t, diff, "regla", "regla:curso:renombrada", "", "elemento")
	baja := buscarCambioConfig(t, diff, "regla", a.Reglas[2].ID, "", "elemento")
	if alta.Tipo != "alta" || alta.Anterior != nil || alta.Nuevo.Regla.ID != "regla:curso:renombrada" || baja.Tipo != "baja" || baja.Nuevo != nil {
		t.Fatal("alta/baja de regla incorrecta")
	}
	alta = buscarCambioConfig(t, diff, "tramo", a.Reglas[0].ID, "tramo:renombrado", "elemento")
	if alta.Nuevo.Tramo.MinDiferencia != -3 {
		t.Fatal("alta de tramo sin material")
	}
	// Una regla completa añadida conserva sus tramos sin duplicar cambios hijos.
	n = copiarConfig(t, a)
	n.Version = "revision:nueva"
	n.Reglas[0].ID = "regla:grado:nueva"
	if diff = compararConfig(t, a, n); len(diff.Cambios) != 2 {
		t.Fatal("tramos duplicados en alta/baja de regla")
	}
}

func TestComparacionConcursosReordenacionVersionOpacaYHuella(t *testing.T) {
	a := diferenciaConfig(t)
	a.Reglas[2].Tipos = []string{"tipo:z", "tipo:a"}
	n := copiarConfig(t, a)
	n.Reglas[0].Tramos[0], n.Reglas[0].Tramos[2] = n.Reglas[0].Tramos[2], n.Reglas[0].Tramos[0]
	n.Reglas[2].Tipos[0], n.Reglas[2].Tipos[1] = n.Reglas[2].Tipos[1], n.Reglas[2].Tipos[0]
	n.Reglas[0], n.Reglas[3] = n.Reglas[3], n.Reglas[0]
	antes, _ := json.Marshal(n)
	diff := compararConfig(t, a, n)
	despues, _ := json.Marshal(n)
	if len(diff.Cambios) != 0 || diff.Anterior.HuellaSHA256 != diff.Nuevo.HuellaSHA256 || !bytes.Equal(antes, despues) {
		t.Fatal("reordenación cambia semántica o entrada")
	}
	// La misma huella que el cálculo existente, sin necesitar hechos en el comparador.
	resultado, err := d.Calcular(a, concEntrada())
	if err != nil {
		t.Fatal(err)
	}
	if resultado.HuellaReglas != diff.Anterior.HuellaSHA256 {
		t.Fatal("autoridad canónica divergente")
	}
	a.Version = "revision:z"
	n.Version = "revision:a"
	diff = compararConfig(t, a, n)
	if len(diff.Cambios) != 0 || diff.Anterior.Version != "revision:z" || diff.Nuevo.Version != "revision:a" || diff.Anterior.HuellaSHA256 == diff.Nuevo.HuellaSHA256 {
		t.Fatal("versión opaca interpretada u omitida")
	}
	n.Reglas[1].Maximo = concPuntos(t, 0)
	uno, _ := json.Marshal(compararConfig(t, a, n))
	dos, _ := json.Marshal(compararConfig(t, a, n))
	if !bytes.Equal(uno, dos) {
		t.Fatal("comparación no determinista")
	}
}

func TestComparacionConcursosRechazo(t *testing.T) {
	for _, caso := range []string{"convocatoria", "version", "invalida_anterior", "invalida_nueva"} {
		t.Run(caso, func(t *testing.T) {
			a := diferenciaConfig(t)
			n := copiarConfig(t, a)
			switch caso {
			case "convocatoria":
				n.ConvocatoriaRef = "concurso:otro"
			case "version":
				n.MaximoTotal = concPuntos(t, 1)
			case "invalida_anterior":
				a.Reglas[1].Conversion = nil
			case "invalida_nueva":
				n.Reglas[0].Tramos[0].MinDiferencia = -2
			}
			_, err := d.CompararConfiguraciones(a, n)
			var nominal *d.Error
			if !errors.As(err, &nominal) {
				t.Fatalf("se aceptó %s: %v", caso, err)
			}
		})
	}
}

func TestComparacionConcursosAislamientoYUnionTipada(t *testing.T) {
	a := diferenciaConfig(t)
	n := copiarConfig(t, a)
	n.Version = "revision:otra"
	n.Reglas[1].Conversion.Divisor = 30
	n.Reglas[2].ID = "regla:curso:otra"
	diff := compararConfig(t, a, n)
	c := buscarCambioConfig(t, diff, "regla", a.Reglas[1].ID, "", "conversion")
	c.Nuevo.Conversion.Divisor = 999
	alta := buscarCambioConfig(t, diff, "regla", "regla:curso:otra", "", "elemento")
	*alta.Nuevo.Regla.HorasMinimas = concRacional(t, 999, 1)
	alta.Nuevo.Regla.Tipos[0] = "tipo:editado"
	if n.Reglas[1].Conversion.Divisor != 30 || n.Reglas[2].HorasMinimas.Numerador() != 5 || n.Reglas[2].Tipos[0] == "tipo:editado" {
		t.Fatal("el resultado modifica su entrada")
	}
	if _, err := json.Marshal(d.ValorCambioConfiguracion{}); err == nil {
		t.Fatal("unión vacía aceptada")
	}
	x := "x"
	cero := int64(0)
	if _, err := json.Marshal(d.ValorCambioConfiguracion{Tipo: "texto", Texto: &x, Entero: &cero}); err == nil {
		t.Fatal("dos variantes aceptadas")
	}
	no := false
	datos, err := json.Marshal(d.ValorCambioConfiguracion{Tipo: "booleano", Booleano: &no})
	if err != nil || !bytes.Contains(datos, []byte(`"booleano":false`)) {
		t.Fatal("false perdido")
	}
}
