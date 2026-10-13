package informejuridico

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/docx"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const rutaCatalogoPlantillasV3Prueba = "../../../../../data/catalogos/contratacion-temporal/plantillas_documentos.v3.json"

var huellasPDFV3 = map[ports.TipoBorradorRRHH]string{
	ports.BorradorInformeDefinitivo:  "6f9c632d14fccd44bf8f30d2478f60f7feb0ab161b1de5fd420d0432f33ea095",
	ports.BorradorResolucion:         "18c8420afd0b773610159c1b9d0b900cd23c141170e65c969cb7bf1dd375680c",
	ports.BorradorDiligencia:         "9e1d2c999a0673fc6ccedafb3ff687e45333823db9c4588087369a3e4bbab19f",
	ports.BorradorTomaPosesion:       "bb79430b07e2cd0e01620a8cc4458a4ff3bc03ac4ae568e84bdb60382f35456a",
	ports.BorradorNotificacion:       "154daeb3b30081298fe35d520614a6b7a80273517646b8841df6f97c19147890",
	ports.BorradorComunicacionCentro: "397ebc52a509ed59397f60838f34d4a013cf5751cefcb38eb736d50325e14c2e",
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
		contenido, err := contenidoBorradorDesarrollo(tipo, detalleInformeDefinitivoPrueba(), nil, plantillas, false)
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
		// La misma v3 publicada permite descargar nombres sin cambiar el original.
		detalle := detalleInformeDefinitivoPrueba()
		nombres := map[string]string{
			detalle.Resumen.CentroRef:    "Secretaría General",
			detalle.Resumen.CategoriaRef: "Auxiliar administrativo/a",
			detalle.Asignacion.UnidadRef: "Servicio de Personal",
		}
		for _, documento := range []vecports.RenderizadorDocumento{
			pdf.Renderizador{Membrete: true}, docx.Renderizador{Membrete: true},
		} {
			t.Run(string(tipo)+"/"+string(documento.Formato()), func(t *testing.T) {
				captura := &renderizadorCapturaNombres{RenderizadorDocumento: documento}
				proveedor := &proveedorPlantillasVariable{catalogo: catalogo}
				r := RenderizadorBorradorCatalogoVivo{
					Proveedor: proveedor, PDF: captura, DOCX: captura,
					Etiquetas:         func(ref string) string { return " " + nombres[ref] + " " },
					Ahora:             func() time.Time { return time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC) },
					OmitirReferencias: true,
				}
				var resultado BorradorCatalogado
				var err error
				if documento.Formato() == vecdomain.FormatoDocumentoPDF {
					resultado, err = r.GenerarPDF(context.Background(), tipo, detalle)
				} else {
					resultado, err = r.GenerarDOCX(context.Background(), tipo, detalle)
				}
				if err != nil {
					t.Fatal(err)
				}
				if resultado.CatalogoRef != plantillas.Referencia() || resultado.CatalogoHuellaSHA256 != plantillas.Huella() || proveedor.llamadas != 1 {
					t.Fatal("la descarga debe conservar la misma publicación v3 y una sola lectura")
				}
				texto := strings.Join(captura.contenido.Parrafos, "\n")
				for ref, nombre := range nombres {
					if strings.Contains(texto, ref) {
						t.Errorf("referencia interna junto al nombre: %s", ref)
					}
					if ref != detalle.Asignacion.UnidadRef && !strings.Contains(texto, nombre) {
						t.Errorf("falta el nombre %q", nombre)
					}
				}
			})
		}
	}
}

// Captura el contenido y delega en los generadores PDF/Word reales y su validación.
type renderizadorCapturaNombres struct {
	vecports.RenderizadorDocumento
	contenido vecdomain.ContenidoDocumento
}

func (r *renderizadorCapturaNombres) Renderizar(ctx context.Context, contenido vecdomain.ContenidoDocumento) ([]byte, error) {
	r.contenido = contenido
	return r.RenderizadorDocumento.Renderizar(ctx, contenido)
}
