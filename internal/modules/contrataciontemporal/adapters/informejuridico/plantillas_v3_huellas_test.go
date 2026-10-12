package informejuridico

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
)

const rutaCatalogoPlantillasV3Prueba = "../../../../../data/demo/plantillas/ct_plantillas_documentos.ejemplo.v3.demo.json"

var huellasPDFV3 = map[ports.TipoBorradorRRHH]string{
	ports.BorradorInformeDefinitivo:  "c46a1ac69bfe53f1ae407bcad6b455fbfa1820102d1e3df7e53917ffc0422efa",
	ports.BorradorResolucion:         "66f7da9d21d2f12bef740d0894f0933371900f6c4f92ef9c6b42f82166b4b17b",
	ports.BorradorDiligencia:         "338de3163702c29f430ef83058b462c049d96fe9f75b3c9eb92cc2bf89642bd0",
	ports.BorradorTomaPosesion:       "dfa0dc7d5be6ee2377bfe1de80b1b86c124c2618a258db7ae9e06508f3fffc06",
	ports.BorradorNotificacion:       "c1bbd7f6fcfe4ae83ae5cebbdffe6995ac05603ce30535df0cbbe69181d1a358",
	ports.BorradorComunicacionCentro: "c2bde6517069692390bcf90b1822a92d96f113c1abbf4a05e99871d3b7b5a53b",
}

func TestPlantillasV3ConservanBorradoresSinMarcadoresDeEnsayo(t *testing.T) {
	consulta, err := fichero.NuevaConsultaCatalogos(rutaCatalogoPlantillasV3Prueba)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := consulta.ObtenerCatalogo(context.Background(), CatalogoPlantillasBorradorID, 3)
	if err != nil {
		t.Fatal(err)
	}
	plantillas, err := NuevasPlantillasBorrador(catalogo, time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(plantillas.Referencia(), CatalogoPlantillasBorradorID+":3") {
		t.Fatalf("referencia: %s", plantillas.Referencia())
	}
	for _, entrada := range catalogo.Entradas {
		if entrada.Clave == "etiquetas" {
			continue
		}
		visible := entrada.Etiqueta + " " + entrada.Descripcion + " " + entrada.Atributos["titulo"] + " " + entrada.Atributos["parrafo.01"]
		if strings.Contains(visible, "Datos sintéticos") || strings.Contains(visible, "modelo de ejemplo") || strings.Contains(visible, "BORRADOR DE DESARROLLO") {
			t.Errorf("%s: el catálogo conserva un marcador visible", entrada.Clave)
		}
	}
	for tipo, esperado := range huellasPDFV3 {
		contenido, err := contenidoBorradorDesarrollo(tipo, detalleInformeDefinitivoPrueba(), nil, plantillas)
		if err != nil {
			t.Fatal(err)
		}
		texto := strings.Join(contenido.Parrafos, "\n")
		if strings.Contains(texto, "Datos sintéticos") || strings.Contains(texto, "BORRADOR DE DESARROLLO") || !strings.Contains(texto, "Borrador sin firmar ni validar") {
			t.Errorf("%s: texto visible incorrecto", tipo)
		}
		pdfBytes, err := (RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{}, Plantillas: plantillas}).RenderizarBorrador(context.Background(), tipo, detalleInformeDefinitivoPrueba())
		if err != nil {
			t.Fatal(err)
		}
		suma := sha256.Sum256(pdfBytes)
		actual := hex.EncodeToString(suma[:])
		if actual != esperado {
			t.Errorf("%s: huella obtenida %s, esperada %s", tipo, actual, esperado)
		}
	}
}
