package informejuridico

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docxvec "vec-diputacion-granada/internal/vec/adapters/documentos/docx"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const rutaCatalogoPlantillasPrueba = "../../../../../data/demo/plantillas/ct_plantillas_documentos.ejemplo.demo.json"

var instantePlantillasPrueba = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)

func catalogoPlantillasPrueba(t *testing.T) vecdomain.CatalogoConfigurable {
	t.Helper()
	consulta, err := fichero.NuevaConsultaCatalogos(rutaCatalogoPlantillasPrueba)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := consulta.ObtenerCatalogo(context.Background(), CatalogoPlantillasBorradorID, 1)
	if err != nil {
		t.Fatal(err)
	}
	return catalogo
}

func plantillasPrueba(t *testing.T) *PlantillasBorrador {
	t.Helper()
	plantillas, err := NuevasPlantillasBorrador(catalogoPlantillasPrueba(t), instantePlantillasPrueba)
	if err != nil {
		t.Fatal(err)
	}
	return plantillas
}

func TestCatalogoDeEjemploTieneLosDiezDocumentos(t *testing.T) {
	plantillas := plantillasPrueba(t)
	if got := plantillas.Tipos(); len(got) != len(TiposBorradorConocidos) {
		t.Fatalf("tipos = %v", got)
	}
	if !strings.HasPrefix(plantillas.Referencia(), CatalogoPlantillasBorradorID+":1") || len(plantillas.Huella()) != 64 {
		t.Fatalf("referencia %q huella %q", plantillas.Referencia(), plantillas.Huella())
	}
	cese, _ := plantillas.Plantilla(ports.BorradorCese)
	if cese.RequiereAccion == "" || len(cese.Firmantes) == 0 {
		t.Fatal("el cese debe exigir su actuación y declarar firmantes")
	}
}

func mutarPlantilla(t *testing.T, clave string, mutar func(map[string]string)) error {
	t.Helper()
	catalogo := catalogoPlantillasPrueba(t)
	for i := range catalogo.Entradas {
		if catalogo.Entradas[i].Clave == clave {
			mutar(catalogo.Entradas[i].Atributos)
		}
	}
	_, err := NuevasPlantillasBorrador(catalogo, instantePlantillasPrueba)
	return err
}

func TestCatalogoRechazaCamposYAtributosDesconocidosAlCargar(t *testing.T) {
	casos := map[string]struct {
		clave string
		mutar func(map[string]string)
	}{
		"campo desconocido":   {"resolucion", func(a map[string]string) { a["parrafo.02"] += " {{dni_persona}}" }},
		"campo en mayúsculas": {"resolucion", func(a map[string]string) { a["parrafo.02"] += " {{Centro}}" }},
		"llave suelta":        {"resolucion", func(a map[string]string) { a["parrafo.02"] += " {{ centro }}" }},
		"cierre sin apertura": {"resolucion", func(a map[string]string) { a["parrafo.02"] += "{{/coste}}" }},
		"bloque sin cerrar":   {"resolucion", func(a map[string]string) { a["parrafo.02"] += "{{?coste}}x" }},
		"bloques anidados": {"resolucion", func(a map[string]string) {
			a["parrafo.02"] += "{{?coste}}{{!observaciones}}x{{/observaciones}}{{/coste}}"
		}},
		"condicional en título":  {"resolucion", func(a map[string]string) { a["titulo"] = "{{?coste}}T{{/coste}}" }},
		"atributo desconocido":   {"resolucion", func(a map[string]string) { a["pie"] = "x" }},
		"párrafo con hueco":      {"resolucion", func(a map[string]string) { a["parrafo.40"] = "x" }},
		"sin título":             {"resolucion", func(a map[string]string) { delete(a, "titulo") }},
		"modalidad no válida":    {"nombramiento", func(a map[string]string) { a["modalidades"] = "vacante,Otra Cosa" }},
		"firmante vacío":         {"nombramiento", func(a map[string]string) { a["firmantes"] = "Uno;;Dos" }},
		"acción no válida":       {"cese", func(a map[string]string) { a["requiere_accion"] = "Registrar cese" }},
		"etiqueta desconocida":   {"etiquetas", func(a map[string]string) { a["persona.nombre"] = "x" }},
		"etiqueta con plantilla": {"etiquetas", func(a map[string]string) { a["modalidad.vacante"] = "{{centro}}" }},
		"título demasiado largo": {"resolucion", func(a map[string]string) { a["titulo"] = strings.Repeat("T", maximoBytesTituloPlantilla+1) }},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			if err := mutarPlantilla(t, caso.clave, caso.mutar); !errors.Is(err, ErrPlantillasBorradorInvalidas) {
				t.Fatalf("se aceptó: %v", err)
			}
		})
	}
	catalogo := catalogoPlantillasPrueba(t)
	catalogo.Entradas[0].Clave = "contrato_de_alquiler"
	if _, err := NuevasPlantillasBorrador(catalogo, instantePlantillasPrueba); !errors.Is(err, ErrPlantillasBorradorInvalidas) {
		t.Fatal("se aceptó un tipo de documento desconocido")
	}
	catalogo = catalogoPlantillasPrueba(t)
	catalogo.ID = "otro.catalogo"
	if _, err := NuevasPlantillasBorrador(catalogo, instantePlantillasPrueba); err == nil {
		t.Fatal("se aceptó otro catálogo")
	}
}

func TestCatalogoRechazaDemasiadosParrafos(t *testing.T) {
	err := mutarPlantilla(t, "resolucion", func(a map[string]string) {
		for i := 1; i <= maximoParrafosPlantilla+1; i++ {
			a["parrafo."+dosCifras(i)] = "Párrafo."
		}
	})
	if !errors.Is(err, ErrPlantillasBorradorInvalidas) {
		t.Fatalf("se aceptaron %d párrafos: %v", maximoParrafosPlantilla+1, err)
	}
}

func TestSinCatalogoNoHayBorradores(t *testing.T) {
	d := detalleInformeDefinitivoPrueba()
	if _, err := (RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{}}).RenderizarBorrador(context.Background(), ports.BorradorResolucion, d); !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
		t.Fatalf("sin catálogo: %v", err)
	}
	if _, err := contenidoBorradorDesarrollo(ports.BorradorResolucion, d, nil, nil); !errors.Is(err, ports.ErrBorradorRRHHNoDisponible) {
		t.Fatalf("sin plantilla: %v", err)
	}
}

func TestLosValoresNoSeInterpretanComoPlantilla(t *testing.T) {
	d := detalleInformeDefinitivoPrueba()
	d.Analisis.Observaciones = "Texto con {{coste}} y {{?centro}}literal{{/centro}}."
	contenido, err := contenidoBorradorDesarrollo(ports.BorradorResolucion, d, nil, plantillasPrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(contenido.Parrafos, "\n"), "Observaciones del análisis: Texto con {{coste}} y {{?centro}}literal{{/centro}}.") {
		t.Fatal("el valor se reinterpretó")
	}
}

func TestPlantillaEditadaCambiaElDocumentoSinTocarCodigo(t *testing.T) {
	catalogo := catalogoPlantillasPrueba(t)
	for i := range catalogo.Entradas {
		if catalogo.Entradas[i].Clave == "resolucion" {
			catalogo.Entradas[i].Atributos["parrafo.03"] = "Antecedentes que redacta RRHH para {{centro}}"
		}
	}
	plantillas, err := NuevasPlantillasBorrador(catalogo, instantePlantillasPrueba)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := contenidoBorradorDesarrollo(ports.BorradorResolucion, detalleInformeDefinitivoPrueba(), nil, plantillas)
	if err != nil || contenido.Parrafos[2] != "Antecedentes que redacta RRHH para centro:sintetico:001" {
		t.Fatalf("plantilla editada no aplicada: %v %q", err, contenido.Parrafos)
	}
}

func TestNuevosDocumentosSegunModalidad(t *testing.T) {
	plantillas := plantillasPrueba(t)
	d := detalleInformeDefinitivoPrueba() // modalidad sustitución: admite los dos
	for _, tipo := range []ports.TipoBorradorRRHH{ports.BorradorContratoLaboral, ports.BorradorNombramiento} {
		contenido, err := contenidoBorradorDesarrollo(tipo, d, nil, plantillas)
		if err != nil {
			t.Fatalf("%s: %v", tipo, err)
		}
		texto := contenido.Titulo + "\n" + strings.Join(contenido.Parrafos, "\n")
		for _, esperado := range []string{"MODELO DE EJEMPLO", "2026/CT-0001", "Sustitución", "01/01/2027", "100,00 %", "Firmas previstas", "D./D.ª ____", CatalogoPlantillasBorradorID + ":1:" + string(tipo)} {
			if !strings.Contains(texto, esperado) {
				t.Fatalf("%s: falta %q en %s", tipo, esperado, texto)
			}
		}
		if strings.Contains(texto, "{{") {
			t.Fatalf("%s: quedan marcas sin rellenar", tipo)
		}
		for _, formato := range []func() ([]byte, error){
			func() ([]byte, error) {
				return RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{}, Plantillas: plantillas}.RenderizarBorrador(context.Background(), tipo, d)
			},
			func() ([]byte, error) {
				return RenderizadorBorradorDOCXDesarrollo{DOCX: docxvec.Renderizador{}, Plantillas: plantillas}.RenderizarBorradorDOCX(context.Background(), tipo, d)
			},
		} {
			if b, err := formato(); err != nil || len(b) == 0 {
				t.Fatalf("%s: %v", tipo, err)
			}
		}
	}
	relevo := detalleInformeDefinitivoPrueba()
	relevo.Analisis.ModalidadClave, relevo.Resumen.ModalidadClave = "relevo", "relevo"
	if _, err := contenidoBorradorDesarrollo(ports.BorradorNombramiento, relevo, nil, plantillas); !errors.Is(err, ports.ErrBorradorRRHHNoDisponible) {
		t.Fatalf("nombramiento para relevo: %v", err)
	}
	if _, err := contenidoBorradorDesarrollo(ports.BorradorContratoLaboral, relevo, nil, plantillas); err != nil {
		t.Fatalf("contrato para relevo: %v", err)
	}
}

// detalleConActuacionPosterior añade tras la propuesta la actuación que exige
// la plantilla (el cese o la modificación los registra otra tarea).
func detalleConActuacionPosterior(accion domain.ClaveCatalogo) ports.DetalleExpedienteRRHH {
	d := detalleInformeDefinitivoPrueba()
	anterior := d.Hitos[len(d.Hitos)-1]
	h := anterior
	h.Secuencia, h.VersionExpediente = anterior.Secuencia+1, anterior.VersionExpediente+1
	h.AccionClave, h.FaseOrigen, h.EstadoOrigen = accion, anterior.FaseDestino, anterior.EstadoDestino
	h.RealizadaEn = anterior.RealizadaEn.Add(time.Minute)
	d.Hitos = append(d.Hitos, h)
	d.Resumen.Version, d.Resumen.ActualizadoEn = h.VersionExpediente, h.RealizadaEn
	return d
}

func TestCeseYModificacionSoloConSuActuacion(t *testing.T) {
	plantillas := plantillasPrueba(t)
	for _, tipo := range []ports.TipoBorradorRRHH{ports.BorradorCese, ports.BorradorModificacionNombramiento} {
		plantilla, _ := plantillas.Plantilla(tipo)
		if _, err := contenidoBorradorDesarrollo(tipo, detalleInformeDefinitivoPrueba(), nil, plantillas); !errors.Is(err, ports.ErrBorradorRRHHNoDisponible) {
			t.Fatalf("%s sin actuación: %v", tipo, err)
		}
		d := detalleConActuacionPosterior(domain.ClaveCatalogo(plantilla.RequiereAccion))
		contenido, err := contenidoBorradorDesarrollo(tipo, d, nil, plantillas)
		if err != nil {
			t.Fatalf("%s: %v", tipo, err)
		}
		texto := strings.Join(contenido.Parrafos, "\n")
		if !strings.Contains(texto, "actuación 7") || !strings.Contains(texto, "actuación 8, de 2026-09-06T01:07:00Z UTC") {
			t.Fatalf("%s: faltan las actuaciones: %s", tipo, texto)
		}
		// Los borradores de formalización dejan de estar disponibles cuando
		// la propuesta ya no es la última actuación reconocida.
		if _, err := contenidoBorradorDesarrollo(ports.BorradorResolucion, d, nil, plantillas); !errors.Is(err, ports.ErrBorradorRRHHNoDisponible) {
			t.Fatalf("resolución tras %s: %v", tipo, err)
		}
	}
	sinPropuesta := detalleConActuacionPosterior("registrar_cese")
	sinPropuesta.Hitos[6].AccionClave = "actuacion_sintetica"
	if _, err := contenidoBorradorDesarrollo(ports.BorradorCese, sinPropuesta, nil, plantillas); !errors.Is(err, ports.ErrBorradorRRHHNoDisponible) {
		t.Fatalf("cese sin propuesta previa: %v", err)
	}
}
