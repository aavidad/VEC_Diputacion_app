package domain

import (
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/shared/baremacion"
)

func puntosPrueba(t *testing.T, s string) baremacion.Puntos {
	t.Helper()
	p, err := ParsearPuntos(s)
	if err != nil {
		t.Fatalf("puntos %q: %v", s, err)
	}
	return p
}

func convocatoriaPrueba(t *testing.T) Convocatoria {
	t.Helper()
	return Convocatoria{
		Ref: "bolsa-operario-diputacion-2026", Titulo: "Bolsa de Operario",
		AbreEn: time.Date(2026, 9, 20, 22, 0, 0, 0, time.UTC), CierraEn: time.Date(2026, 10, 30, 22, 59, 59, 0, time.UTC),
		PublicadaEn: time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC), FechaReferencia: "2026-10-30",
		Turnos: []Turno{{"libre", "Turno libre"}, {"discapacidad", "Reserva"}},
		Requisitos: []Requisito{
			{Clave: "nacionalidad", Titulo: "Nacionalidad", Obligatorio: true, ImpidePresentar: true},
			{Clave: "titulacion", Titulo: "Titulación", Obligatorio: true},
		},
		Baremo: Baremo{Maximo: puntosPrueba(t, "10"), Redondeo: baremacion.RedondeoMitadArriba, Grupos: []GrupoBaremo{
			{Clave: "experiencia", Titulo: "Experiencia", Maximo: puntosPrueba(t, "6"), Meritos: []MeritoBaremo{
				{Clave: "meses_administracion", Titulo: "Meses AAPP", Unidad: "mes", PuntosPorUnidad: puntosPrueba(t, "0.1"), Maximo: puntosPrueba(t, "6")},
				{Clave: "meses_privado", Titulo: "Meses privado", Unidad: "mes", PuntosPorUnidad: puntosPrueba(t, "0.05"), Maximo: puntosPrueba(t, "3")},
			}},
			{Clave: "formacion", Titulo: "Formación", Maximo: puntosPrueba(t, "4"), Meritos: []MeritoBaremo{
				{Clave: "horas_cursos", Titulo: "Horas", Unidad: "hora", PuntosPorUnidad: puntosPrueba(t, "0.01"), Maximo: puntosPrueba(t, "3")},
			}},
		}},
		Numeracion: Numeracion{Patron: "{anio}/SOL-{numero}", Ancho: 6}, MarcaEjemplo: true,
	}
}

func datosCompletos() DatosPersonales {
	return DatosPersonales{Nombre: " Antonio ", Apellidos: "Reyes  Álvarez", DocumentoIdentidad: "12345678-z", FechaNacimiento: "1988-04-12",
		Nacionalidad: "española", Correo: "antonio.reyes@example.org", Telefono: "600 000 000",
		Direccion: Direccion{Via: "C/ Real 1", CodigoPostal: "18001", Municipio: "Granada", Provincia: "Granada"}}
}

func TestPuntosYCantidades(t *testing.T) {
	for entrada, salida := range map[string]string{"0": "0", "1.4": "1.4", "20": "20", "0.000001": "0.000001", "12.250": "12.25"} {
		if got := FormatearPuntos(puntosPrueba(t, entrada)); got != salida {
			t.Fatalf("%s → %s; se esperaba %s", entrada, got, salida)
		}
	}
	for _, malo := range []string{"", "-1", "01", "1.", ".5", "1,5", "1.0000001", "abc"} {
		if _, err := ParsearPuntos(malo); err == nil {
			t.Fatalf("aceptó puntos %q", malo)
		}
	}
	for _, malo := range []string{"", "-2", "1.2345", "1000001", "x"} {
		if _, err := ParsearCantidad(malo); err == nil {
			t.Fatalf("aceptó cantidad %q", malo)
		}
	}
}

func TestConvocatoriaHuellaEstableYContenidoReversible(t *testing.T) {
	c := convocatoriaPrueba(t)
	h1, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := c.HuellaSHA256()
	if h1 != h2 || len(h1) != 64 {
		t.Fatal("la huella no es estable")
	}
	contenido, err := c.ContenidoJSON()
	if err != nil {
		t.Fatal(err)
	}
	vuelta, err := ConvocatoriaDesdeContenido(c.Ref, c.Titulo, c.AbreEn, c.CierraEn, c.PublicadaEn, contenido)
	if err != nil {
		t.Fatal(err)
	}
	if h3, _ := vuelta.HuellaSHA256(); h3 != h1 {
		t.Fatal("el contenido publicado no reconstruye la misma convocatoria")
	}
	otra := convocatoriaPrueba(t)
	otra.CierraEn = otra.CierraEn.Add(time.Hour)
	if h4, _ := otra.HuellaSHA256(); h4 == h1 {
		t.Fatal("un plazo distinto no cambia la huella")
	}
	mala := convocatoriaPrueba(t)
	mala.Requisitos = append(mala.Requisitos, Requisito{Clave: "x", Titulo: "X", ImpidePresentar: true})
	if mala.Validar() == nil {
		t.Fatal("un requisito que impide presentar debe ser obligatorio")
	}
	mala = convocatoriaPrueba(t)
	mala.Numeracion.Patron = "SOL-{numero}"
	if mala.Validar() == nil {
		t.Fatal("el patrón del justificante exige {anio}")
	}
}

func TestBorradorParcialYCompleto(t *testing.T) {
	c := convocatoriaPrueba(t)
	hoy := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	parcial, err := Borrador{Requisitos: []RequisitoDeclarado{{"nacionalidad", RequisitoCumple}}, Datos: DatosPersonales{Nombre: "Antonio"}}.Preparar(c, hoy)
	if err != nil || parcial.Completo {
		t.Fatalf("el borrador parcial debe admitirse incompleto: %v", err)
	}
	if len(parcial.Borrador.Requisitos) != 2 || parcial.Borrador.Requisitos[1].Estado != RequisitoPendiente {
		t.Fatal("los requisitos no declarados deben quedar pendientes y en orden")
	}
	completo, err := Borrador{Turno: "libre", Datos: datosCompletos(), Meritos: []MeritoDeclarado{
		{ClaveGrupo: "experiencia", ClaveMerito: "meses_administracion", Cantidad: "14"},
		{ClaveGrupo: "formacion", ClaveMerito: "horas_cursos", Cantidad: "450"},
	}}.Preparar(c, hoy)
	if err != nil || !completo.Completo {
		t.Fatalf("borrador completo rechazado: %v", err)
	}
	d := completo.Borrador.Datos
	if d.DocumentoIdentidad != "12345678Z" || d.Apellidos != "Reyes Álvarez" || d.Telefono != "600000000" || d.DocumentoParcial() != "***5678*" ||
		d.NombreVisible() != "Reyes Álvarez, Antonio" {
		t.Fatalf("normalización inesperada: %+v", d)
	}
	// 14 × 0,1 = 1,4; 450 × 0,01 = 4,5 → máximo del mérito 3.
	if got := FormatearPuntos(completo.Autobaremo.Total); got != "4.4" || !completo.Autobaremo.Maximos {
		t.Fatalf("autobaremo %s; se esperaba 4.4 con máximo aplicado", got)
	}
	for nombre, b := range map[string]Borrador{
		"turno desconocido":     {Turno: "otro"},
		"requisito desconocido": {Requisitos: []RequisitoDeclarado{{"x", RequisitoCumple}}},
		"estado inválido":       {Requisitos: []RequisitoDeclarado{{"nacionalidad", "si"}}},
		"requisito repetido":    {Requisitos: []RequisitoDeclarado{{"nacionalidad", RequisitoCumple}, {"nacionalidad", RequisitoPendiente}}},
		"mérito desconocido":    {Meritos: []MeritoDeclarado{{ClaveGrupo: "experiencia", ClaveMerito: "x", Cantidad: "1"}}},
		"mérito repetido": {Meritos: []MeritoDeclarado{{ClaveGrupo: "experiencia", ClaveMerito: "meses_privado", Cantidad: "1"},
			{ClaveGrupo: "experiencia", ClaveMerito: "meses_privado", Cantidad: "2"}}},
		"cantidad inválida": {Meritos: []MeritoDeclarado{{ClaveGrupo: "experiencia", ClaveMerito: "meses_privado", Cantidad: "-1"}}},
		"correo inválido":   {Datos: DatosPersonales{Correo: "no es correo"}},
		"nacido mañana":     {Datos: DatosPersonales{FechaNacimiento: "2026-09-27"}},
		"documento raro":    {Datos: DatosPersonales{DocumentoIdentidad: "12<34>"}},
	} {
		if _, err := b.Preparar(c, hoy); !errors.Is(err, ErrSolicitudInvalida) {
			t.Fatalf("%s: se esperaba solicitud inválida, se obtuvo %v", nombre, err)
		}
	}
}

func TestImpidePresentarSoloNoCumpleEnRequisitoQueImpide(t *testing.T) {
	c := convocatoriaPrueba(t)
	if _, bloquea := ImpidePresentar(c, []RequisitoDeclarado{{"nacionalidad", RequisitoPendiente}, {"titulacion", RequisitoNoCumple}}); bloquea {
		t.Fatal("pendiente o no_cumple en un requisito que no impide presentar no deben bloquear")
	}
	if clave, bloquea := ImpidePresentar(c, []RequisitoDeclarado{{"nacionalidad", RequisitoNoCumple}}); !bloquea || clave != "nacionalidad" {
		t.Fatal("no_cumple en un requisito que impide presentar debe bloquear")
	}
}

func TestBaremoAplicaMaximosDeGrupoYTotal(t *testing.T) {
	c := convocatoriaPrueba(t)
	r, err := c.Baremo.Calcular([]MeritoDeclarado{
		{ClaveGrupo: "experiencia", ClaveMerito: "meses_administracion", Cantidad: "50"},
		{ClaveGrupo: "experiencia", ClaveMerito: "meses_privado", Cantidad: "100"},
		{ClaveGrupo: "formacion", ClaveMerito: "horas_cursos", Cantidad: "1000"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Experiencia: min(5,6)+min(5,3)=8 → grupo 6; formación min(10,3)=3; total 9.
	if got := FormatearPuntos(r.Total); got != "9" || len(r.Lineas) != 3 || r.Lineas[0].Titulo != "Meses AAPP" || r.Lineas[0].Unidad != "mes" {
		t.Fatalf("total %s", got)
	}
}
