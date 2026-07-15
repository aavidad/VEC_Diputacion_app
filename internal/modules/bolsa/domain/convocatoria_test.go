package domain

import (
	"errors"
	"testing"
	"time"
)

func convocatoriaPublicaValidaPrueba() Convocatoria {
	publicada := time.Date(2026, time.July, 1, 8, 0, 0, 0, time.UTC)
	return Convocatoria{
		ID: "proceso:bolsa:auxiliar-2026", Version: "v1", Estado: "inscripcion",
		DatosPublicos: &DatosPublicosConvocatoria{
			IdentificadorPublico: "auxiliar-administrativo-2026",
			Tipo:                 "bolsa_temporal", Categorias: []string{"auxiliar_administrativo"},
			Titulo: "Bolsa temporal de demostracion", Resumen: "Resumen publico de demostracion.",
			Descripcion: "Descripcion publica de demostracion.", PublicadaEn: publicada, ActualizadaEn: publicada,
			Plazos: []PlazoConvocatoria{{
				Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripcion",
				Descripcion: "Plazo publico.", AbreEn: publicada, CierraEn: publicada.Add(30 * 24 * time.Hour),
			}},
			Requisitos: []RequisitoConvocatoria{{Referencia: "req:edad", Orden: 1, Titulo: "Edad", Descripcion: "Requisito de demostracion.", Obligatorio: true}},
			Documentos: []DocumentoConvocatoria{{
				Referencia: "doc:bases", Tipo: "bases", Orden: 1, Titulo: "Bases", Descripcion: "Documento de demostracion.",
				Formato: "html", URL: "/bolsa/documentos/bases-demo.html", PublicadoEn: publicada,
			}},
			Ayuda: []AyudaConvocatoria{{Referencia: "ayuda:alta", Categoria: "inscripcion", Orden: 1, Pregunta: "¿Como participar?", Respuesta: "Consulte las bases."}},
		},
	}
}

func TestConvocatoriaPublicaCanonicaYConHuellaEstable(t *testing.T) {
	convocatoria := convocatoriaPublicaValidaPrueba()
	if err := convocatoria.ValidarPublicacion(); err != nil {
		t.Fatalf("ValidarPublicacion() error = %v", err)
	}
	huella, err := convocatoria.HuellaPublicaSHA256()
	if err != nil || len(huella) != 64 {
		t.Fatalf("HuellaPublicaSHA256() = %q, %v", huella, err)
	}
	clon := convocatoria.Clonar()
	clon.DatosPublicos.Categorias[0] = "otra"
	if convocatoria.DatosPublicos.Categorias[0] != "auxiliar_administrativo" {
		t.Fatal("Clonar() comparte memoria con el agregado")
	}
}

func TestConvocatoriaPublicaRechazaZonaNoUTCYEspaciosNoCanonicos(t *testing.T) {
	for nombre, mutar := range map[string]func(*Convocatoria){
		"zona +01": func(c *Convocatoria) {
			c.DatosPublicos.PublicadaEn = c.DatosPublicos.PublicadaEn.In(time.FixedZone("+01", 3600))
		},
		"precision inferior al microsegundo": func(c *Convocatoria) {
			c.DatosPublicos.PublicadaEn = c.DatosPublicos.PublicadaEn.Add(time.Nanosecond)
		},
		"espacio en referencia": func(c *Convocatoria) { c.ID = " proceso:bolsa:auxiliar-2026" },
		"espacio en clave":      func(c *Convocatoria) { c.DatosPublicos.Tipo = "bolsa_temporal " },
		"espacio en texto":      func(c *Convocatoria) { c.DatosPublicos.Titulo = " Titulo" },
		"marca bidi":            func(c *Convocatoria) { c.DatosPublicos.Titulo = "Titulo\u202edesviado" },
	} {
		t.Run(nombre, func(t *testing.T) {
			convocatoria := convocatoriaPublicaValidaPrueba()
			mutar(&convocatoria)
			if err := convocatoria.ValidarPublicacion(); !errors.Is(err, ErrConvocatoriaInvalida) {
				t.Fatalf("ValidarPublicacion() error = %v", err)
			}
		})
	}
}

func TestConvocatoriaPublicaRechazaDuplicadosYURLExterna(t *testing.T) {
	convocatoria := convocatoriaPublicaValidaPrueba()
	convocatoria.DatosPublicos.Categorias = append(convocatoria.DatosPublicos.Categorias, convocatoria.DatosPublicos.Categorias[0])
	if err := convocatoria.ValidarPublicacion(); !errors.Is(err, ErrConvocatoriaInvalida) {
		t.Fatalf("categorias duplicadas: %v", err)
	}
	convocatoria = convocatoriaPublicaValidaPrueba()
	convocatoria.DatosPublicos.Documentos[0].URL = "https://example.invalid/bases.pdf"
	if err := convocatoria.ValidarPublicacion(); !errors.Is(err, ErrConvocatoriaInvalida) {
		t.Fatalf("URL externa: %v", err)
	}
}
