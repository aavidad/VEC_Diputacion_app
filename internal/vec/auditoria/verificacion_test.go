package auditoria

import (
	"strings"
	"testing"
)

// Fixed independent vectors use AD3-002 UTF-8 octet framing. The accented
// decision reference catches implementations that count characters as bytes.
func vectorCadena() DocumentoVerificacion {
	c1 := "db6527bc20d0ea2e6c20f7f5b07fba88a3e20e28837b36822279f5d1e691c060"
	c2 := "bb0b89e4c541aa5bc318d992f1bfdf06df7cee6e608a4e216090212f5c666fda"
	h1 := "a195617c99d7d7ec006b0afb951024bb11e8abb1b26421639db4059dc27df678"
	h2 := "7d411c4cb1765927e6b444643a984e81435fad4b0d99488dfda18a4d7a074fa2"
	return DocumentoVerificacion{
		Esquema: EsquemaVerificacion,
		Manifiesto: CoberturaCadena{CadenaID: "cadena:sintetica:principal", PrimeraSecuencia: 1,
			UltimaSecuencia: 2, AnteriorSHA256: strings.Repeat("0", 64), CabezaSHA256: h2, Registros: 2},
		Registros: []RegistroCadenaV3{
			{AuditoriaRef: "aud_v3_" + c1[:32], Secuencia: 1, DecisionRef: "decisión:1", EfectoRef: "efecto:1",
				HuellaEfectoSHA256: strings.Repeat("a", 64), AnteriorSHA256: strings.Repeat("0", 64), HuellaSHA256: h1, ConsumoHuellaSHA256: c1},
			{AuditoriaRef: "aud_v3_" + c2[:32], Secuencia: 2, DecisionRef: "decisión:2", EfectoRef: "efecto:2",
				HuellaEfectoSHA256: strings.Repeat("a", 64), AnteriorSHA256: h1, HuellaSHA256: h2, ConsumoHuellaSHA256: c2},
		},
	}
}

func TestVerificarCadenaV3VectorYLimitesDeLaGarantia(t *testing.T) {
	d := vectorCadena()
	r := VerificarCadenaV3(d, d.Manifiesto, 2)
	if r.Estado != "verificada" || !r.CheckpointCotejado || r.Fallo != nil ||
		r.AutenticidadCheckpoint != "no_comprobada" || r.ContenidoConsumoRecalculado || r.CamposFueraHuellaVerificados {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestVerificarCadenaV3DetectaAlteraciones(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		mutar  func(*DocumentoVerificacion)
		codigo string
	}{
		{"payload_coordenada", func(d *DocumentoVerificacion) { d.Registros[0].EfectoRef = "efecto:otro" }, "huella_distinta"},
		{"consumo", func(d *DocumentoVerificacion) { d.Registros[0].ConsumoHuellaSHA256 = strings.Repeat("b", 64) }, "referencia_distinta"},
		{"hueco", func(d *DocumentoVerificacion) { d.Registros[1].Secuencia = 3 }, "secuencia_distinta"},
		{"enlace", func(d *DocumentoVerificacion) { d.Registros[1].AnteriorSHA256 = strings.Repeat("b", 64) }, "enlace_distinto"},
		{"huella", func(d *DocumentoVerificacion) { d.Registros[0].HuellaSHA256 = strings.Repeat("b", 64) }, "huella_distinta"},
		{"duplicado", func(d *DocumentoVerificacion) { d.Registros[1].DecisionRef = d.Registros[0].DecisionRef }, "decision_duplicada"},
		{"consumo_duplicado", func(d *DocumentoVerificacion) {
			d.Registros[1].ConsumoHuellaSHA256 = d.Registros[0].ConsumoHuellaSHA256
			d.Registros[1].AuditoriaRef = d.Registros[0].AuditoriaRef
		}, "consumo_duplicado"},
		{"truncado", func(d *DocumentoVerificacion) { d.Registros = d.Registros[:1] }, "cantidad_distinta"},
		{"truncado_y_manifiesto", func(d *DocumentoVerificacion) {
			d.Registros = d.Registros[:1]
			d.Manifiesto.UltimaSecuencia, d.Manifiesto.Registros = 1, 1
			d.Manifiesto.CabezaSHA256 = d.Registros[0].HuellaSHA256
		}, "checkpoint_distinto"},
		{"referencia_personal", func(d *DocumentoVerificacion) { d.Registros[0].DecisionRef = "valor con espacios" }, "registro_invalido"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			d := vectorCadena()
			checkpoint := d.Manifiesto
			caso.mutar(&d)
			r := VerificarCadenaV3(d, checkpoint, 2)
			if r.Estado != "rechazada" || r.Fallo == nil || r.Fallo.Codigo != caso.codigo {
				t.Fatalf("unexpected result: %+v", r)
			}
		})
	}
}

func TestVerificarCadenaV3RangoParcialYVacio(t *testing.T) {
	d := vectorCadena()
	d.Registros = d.Registros[1:]
	d.Manifiesto.PrimeraSecuencia, d.Manifiesto.Registros = 2, 1
	d.Manifiesto.AnteriorSHA256 = d.Registros[0].AnteriorSHA256
	if r := VerificarCadenaV3(d, d.Manifiesto, 1); r.Estado != "verificada" {
		t.Fatalf("partial range rejected: %+v", r)
	}
	d.Manifiesto.AnteriorSHA256 = strings.Repeat("0", 64)
	if r := VerificarCadenaV3(d, d.Manifiesto, 1); r.Fallo == nil || r.Fallo.Codigo != "enlace_distinto" {
		t.Fatalf("partial predecessor ignored: %+v", r)
	}
	d = DocumentoVerificacion{Esquema: EsquemaVerificacion, Manifiesto: CoberturaCadena{
		CadenaID: "cadena:sintetica:vacia", AnteriorSHA256: strings.Repeat("0", 64), CabezaSHA256: strings.Repeat("0", 64)}, Registros: []RegistroCadenaV3{}}
	if r := VerificarCadenaV3(d, d.Manifiesto, 1); r.Estado != "verificada" {
		t.Fatalf("empty genesis rejected: %+v", r)
	}
}

func TestVerificarCadenaV3CabezaYLimites(t *testing.T) {
	d := vectorCadena()
	if r := VerificarCadenaV3(d, d.Manifiesto, 1); r.Fallo == nil || r.Fallo.Codigo != "limite_registros" || r.Fallo.Esperado != "1" || r.Fallo.Obtenido != "2" {
		t.Fatalf("row limit ignored: %+v", r)
	}
	d.Manifiesto.CabezaSHA256 = strings.Repeat("a", 64)
	if r := VerificarCadenaV3(d, d.Manifiesto, 2); r.Fallo == nil || r.Fallo.Codigo != "cabeza_distinta" {
		t.Fatalf("final anchor ignored: %+v", r)
	}
	d.Manifiesto.PrimeraSecuencia, d.Manifiesto.UltimaSecuencia = maxSecuenciaVerificacion, maxSecuenciaVerificacion+1
	if r := VerificarCadenaV3(d, d.Manifiesto, 2); r.Fallo == nil || r.Fallo.Codigo != "cobertura_invalida" {
		t.Fatalf("sequence limit ignored: %+v", r)
	}
}
