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

// huellasPDFAnteriores fijan los PDF del catálogo que reproducen el contenido
// antes escrito en Go, con cabecera 1.4 y /ID. Los originales ya custodiados
// conservan sus bytes y huellas previas.
var huellasPDFAnteriores = map[string]string{
	"base/comunicacion_centro":  "9e7fd76d3a1b9fb06e6efdb9632f8c69ed4ad1b4b6c305ed86031b6b3d7748a0",
	"base/diligencia":           "2f10bbf12311eb513c25554915ad07464bba322d2be8cc4e0e6c5aac55aa7308",
	"base/informe_definitivo":   "154f25f175ccb995df36a69e0491269a2989f70ef8d9be88d7b5af193aefb994",
	"base/notificacion":         "b13c6285c9e85de38341a3f47ad76ccfbafd51ec1eaabebd0c3bb3d949d651ef",
	"base/resolucion":           "7c7425bdff85fe7c46bb19e4251cf3a44e65e890eba774965f6e93d5acdef05d",
	"base/toma_posesion":        "18d2b433995d66972adc7d02fde9049ad30f7f2b07caaefce5e5793cb87a5381",
	"rico/comunicacion_centro":  "d4f1ca39ac7be85943c38f5e75effba16ce64a2470883798f781527d2315cf23",
	"rico/diligencia":           "108d1ae6889851aa3b597cdf33299daedb6eb140a5040f1489c519206d590d70",
	"rico/informe_definitivo":   "9180f76a15955c9e1ad379c68887fadfc80426093ee10ea72132cb8e3a8eca3a",
	"rico/notificacion":         "0cdeb57f20ea5f0a27162088d503bed4c1bcdea50d158b88dd207380a4609e7f",
	"rico/resolucion":           "2ad80323360c39ec8eecda38540e7f0e4dc7ab2b264570f2da0f257bf9503337",
	"rico/toma_posesion":        "f98e0d80227b2a61b11d25b1898a82726d1ec1954f70dfb59b238ad3e9a7a85b",
	"v9/comunicacion_centro":    "c7f8768076ba75b7142888cbfec5e3ddb4786f8ff1963f1808ea7adca697b5ee",
	"v9/diligencia":             "9ec0fb252b27519a3cc15e51d27a33f027bd50171f0a9e13fa5a62a0a5b94b3c",
	"v9/informe_definitivo":     "360eaba25b6471219c5956d1d29f68f911360ba210273156117a8787c49a978b",
	"v9/notificacion":           "244b1b6af43462e1e86f46714787edb8cd473e8b2659d06b866f4a3c6456d87a",
	"v9/resolucion":             "55b4c646b4a21d6bf1f249b543d2e1563ebc9e6a71611fa84067c52dbe1b42d7",
	"v9/toma_posesion":          "6b754d1b10e10289e7142e94c441ff10af87ad7dd6c36344b03f54707d378fea",
	"vacio/comunicacion_centro": "f3149943c5c503a5d93457a86933a72ebca625e5786aff1e95c29a63edd1a046",
	"vacio/diligencia":          "108d1ae6889851aa3b597cdf33299daedb6eb140a5040f1489c519206d590d70",
	"vacio/informe_definitivo":  "1a67f737d177ac5f348a50e9ce9a016a9010f8048506dcddb96b3e0c7a888173",
	"vacio/notificacion":        "0cdeb57f20ea5f0a27162088d503bed4c1bcdea50d158b88dd207380a4609e7f",
	"vacio/resolucion":          "0be70f4cbaebd435b88b1a6d7322eec5c79d4ef5fd95688a38092d1411e291a8",
	"vacio/toma_posesion":       "d2d18557879c4facc904bc29014012a00b0b0f197e34079cf15bcf90f37b6722",
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
				t.Errorf("%s/%s: huella obtenida %s, esperada %s", nombre, tipo, hex.EncodeToString(h[:]), want)
			}
			vistos++
		}
	}
	if vistos != len(huellasPDFAnteriores) {
		t.Fatalf("comprobados %d de %d", vistos, len(huellasPDFAnteriores))
	}
}
