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

const rutaCatalogoPlantillasV3Prueba = "../../../../../data/catalogos/contratacion-temporal/plantillas_documentos.v3.json"

var huellasPDFV3 = map[ports.TipoBorradorRRHH]string{
	ports.BorradorInformeDefinitivo:  "e506fb6bee374eb01f0813eee9f91409b0e3d71c07cb72eb66b5781fcf3f7248",
	ports.BorradorResolucion:         "0946ca378ef63b80776785f1b9be9664fa23e6d0fd6d476d37333966d2345f2d",
	ports.BorradorDiligencia:         "581a713b9a9f099d1a81ec97ff4a9ec992e1f13858b00713ef283e995f73b514",
	ports.BorradorTomaPosesion:       "742f3f6b8c229df930ab61872f4d445686b33dce802a52baf11f8dbfc2ca4ac9",
	ports.BorradorNotificacion:       "3014b96e015e201b96b5bb9c96f5caac220043eb560576dfdbc650e4da948daf",
	ports.BorradorComunicacionCentro: "40e3d2550a029ff415fdc37339414fce42af604173b357e2aa83321064856c33",
}

func TestPlantillasV3SinMarcasDeEnsayoNiFirmaFingida(t *testing.T) {
	consulta, err := fichero.NuevaConsultaCatalogoProvisionable(rutaCatalogoPlantillasV3Prueba)
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
	prohibidas := []string{"ejemplo", "demo", "sintétic", "borrador", "sin valor", "sin validez", "sin efectos", "desarrollo", "ensayo"}
	visibles := []string{catalogo.Nombre, catalogo.FuenteRef, catalogo.MotivoCreacion, catalogo.AprobacionRef, catalogo.MotivoPublicacion}
	for _, entrada := range catalogo.Entradas {
		visibles = append(visibles, entrada.Etiqueta, entrada.Descripcion)
		for _, valor := range entrada.Atributos {
			visibles = append(visibles, valor)
		}
	}
	for _, valor := range visibles {
		for _, marca := range prohibidas {
			if strings.Contains(strings.ToLower(valor), marca) {
				t.Errorf("el catálogo conserva la marca %q: %s", marca, valor)
			}
		}
	}
	for tipo, esperado := range huellasPDFV3 {
		contenido, err := contenidoBorradorDesarrollo(tipo, detalleInformeDefinitivoPrueba(), nil, plantillas)
		if err != nil {
			t.Fatal(err)
		}
		texto := strings.Join(contenido.Parrafos, "\n")
		// El documento deja la firma por hacer: nunca afirma que ya está firmado.
		if strings.Contains(strings.ToLower(texto), "firmado por") || strings.Contains(strings.ToLower(texto), "firmado electrónicamente") {
			t.Errorf("%s: el texto da por hecha una firma", tipo)
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
