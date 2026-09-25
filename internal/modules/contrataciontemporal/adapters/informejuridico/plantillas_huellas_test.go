package informejuridico

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
)

func variantesHuellas() map[string]struct {
	d ports.DetalleExpedienteRRHH
	e EtiquetadorReferencias
} {
	base := detalleInformeDefinitivoPrueba()
	rico := detalleInformeDefinitivoPrueba()
	rico.Analisis.Observaciones = "Nota operativa: {{centro}} 50 % urgente."
	rico.Analisis.CostePrevisto = &ports.ImporteOperativoRRHH{Centimos: 1234507, Moneda: "EUR"}
	rico.Analisis.FuenteCosteRef = "fuente:coste:sintetica"
	rico.Analisis.ModalidadClave, rico.Resumen.ModalidadClave = "otra_modalidad", "otra_modalidad"
	rico.Analisis.PorcentajeJornada = 5005
	rico.Cobertura.Comprobaciones = []ports.ComprobacionOperativaRRHH{{Clave: "hay_candidaturas_disponibles", Resultado: domain.ComprobacionNegativa}, {Clave: "clave_rara", Resultado: domain.ComprobacionNoConsta}, {Clave: "oferta_sae_disponible", Resultado: domain.ComprobacionAfirmativa}, {Clave: "requiere_nueva_convocatoria", Resultado: domain.ComprobacionNoConsta}}
	vacio := detalleInformeDefinitivoPrueba()
	vacio.Cobertura.Comprobaciones = nil
	vacio.Analisis.ModalidadClave, vacio.Resumen.ModalidadClave = "relevo", "relevo"
	et := EtiquetadorReferencias(func(r string) string {
		return map[string]string{"centro:sintetico:001": "Secretaría General", "categoria:sintetica:c2": "Auxiliar", "unidad:sintetica:rrhh": " Servicio "}[r]
	})
	return map[string]struct {
		d ports.DetalleExpedienteRRHH
		e EtiquetadorReferencias
	}{"base": {base, nil}, "rico": {rico, et}, "vacio": {vacio, et}, "v9": {detalleInformeDefinitivoPruebaVersionNueve(), nil}}
}

// huellasPDFAnteriores son los SHA-256 de los PDF que generaba el texto
// escrito en Go antes de pasar al catálogo de plantillas. La resolución de
// formalización conserva la huella de su PDF: el catálogo de ejemplo debe
// reproducir exactamente los mismos bytes.
var huellasPDFAnteriores = map[string]string{
	"base/comunicacion_centro":  "59059b4cb8bdafce4836660d0e33294b1a95aff697b664eb100abfbfc8d7acd2",
	"base/diligencia":           "76f14c94e9f6956a5453ae3678230347fd35722ad4d5b435e6feb9288ee0881d",
	"base/informe_definitivo":   "76a7348d4631306937679b98c692f477e068ded5b76a6e01c3914cdc76653451",
	"base/notificacion":         "c33840d5a2d3a1d9b0534601c89787b3b4065ada152ab56497e0a3a9e2b473aa",
	"base/resolucion":           "a85aed6b6972fdc9fcda6111010a06a6d41cf9a3b9c54934f09018131c9dbd72",
	"base/toma_posesion":        "1c5999411f990cc107087df0e8c63c750ddd0855557e85cdc5fe5d2a61654705",
	"rico/comunicacion_centro":  "6bf1a74a3638f0f2e784deb1c498d0fb6db11a3c00909fc63b13dc22da129f14",
	"rico/diligencia":           "3742e6025cff6593c4d83196e593c2ed4d1aa0318c15a86742a37cdc566e4ad5",
	"rico/informe_definitivo":   "e8c156632ca0114054c289685373063199ed09310fce0ad780772772e56a8425",
	"rico/notificacion":         "24a70e7ed6e0467bb5c0e7eb3ec94efe99940b9333ce93c0f49e54b9035a0a7e",
	"rico/resolucion":           "8e1869884d9d2b6dac025f9a924595414eebfb24d92675c8776e035985bafd25",
	"rico/toma_posesion":        "cab3c7efad95758cdf62b425cf716bdfde0a16605ba7c4dc3c0299552924bc67",
	"v9/comunicacion_centro":    "5e2ba0aa77d8f0585037ac7e9efa09c9b20f5e87466af4e125ac760087bfb69e",
	"v9/diligencia":             "3864942a674aa40ab77e1444ff712959207b09dea5b8447d5af2393175d8a98f",
	"v9/informe_definitivo":     "757ffa2a662a15447c57919ac50b4fe7b8ef45651ae33b1e01fdf6ba56da4175",
	"v9/notificacion":           "da5c1c12d448110747373941711932ccf3daff650ebec45324b7531342c1e110",
	"v9/resolucion":             "853f7352340fae44feb31c335f6e25ba3f5594cce670f4ed29e091b4d14c337f",
	"v9/toma_posesion":          "e2e2b350255cba141edfe95315181e565ec26ed4f2ce70b8869a9e80f788f6ed",
	"vacio/comunicacion_centro": "2bd78a9de577941c08a980bada6bf724fdc7c3df67a61c3252857bfbd5220760",
	"vacio/diligencia":          "3742e6025cff6593c4d83196e593c2ed4d1aa0318c15a86742a37cdc566e4ad5",
	"vacio/informe_definitivo":  "12eb413b0fe13dab1b62e1b22428195cc416f3446003d577733f7c1d8de6598e",
	"vacio/notificacion":        "24a70e7ed6e0467bb5c0e7eb3ec94efe99940b9333ce93c0f49e54b9035a0a7e",
	"vacio/resolucion":          "8cc3bf0f2fadbb962a278ac42746bff2e89c060e0900353f774f006a2da567ee",
	"vacio/toma_posesion":       "a2d3e4e82f8b31f23a384d52dd6e39acf3798f80559c957a6b19dd6f1dd35720",
}

func TestPlantillasReproducenLosBorradoresAnteriores(t *testing.T) {
	plantillas := plantillasPrueba(t)
	vistos := 0
	for nombre, v := range variantesHuellas() {
		for _, tipo := range TiposBorradorConocidos[:6] {
			b, err := RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{}, Etiquetas: v.e, Plantillas: plantillas}.RenderizarBorrador(context.Background(), tipo, v.d)
			if err != nil {
				t.Fatal(err)
			}
			h := sha256.Sum256(b)
			if want := huellasPDFAnteriores[nombre+"/"+string(tipo)]; hex.EncodeToString(h[:]) != want {
				t.Errorf("%s/%s: el PDF cambió respecto al texto anterior", nombre, tipo)
			}
			vistos++
		}
	}
	if vistos != len(huellasPDFAnteriores) {
		t.Fatalf("comprobados %d de %d", vistos, len(huellasPDFAnteriores))
	}
}
