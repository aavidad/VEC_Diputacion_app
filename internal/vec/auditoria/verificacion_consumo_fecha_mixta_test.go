package auditoria

import (
	"encoding/json"
	"os"
	"testing"
)

// La cadena se encuadró con Python hashlib, fuera del cotejador Go.
func vectorConsumosMixtosAD173(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	b, err := os.ReadFile("../../../cmd/vec-auditoria-verificar/testdata/consumos_ad173_mixtos.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestCadenaConsumosMixtosAD173YAlcanceHistorico(t *testing.T) {
	d := vectorConsumosMixtosAD173(t)
	r := VerificarCadenaMixtaV3(d, d.Manifiesto, 3)
	if r.Estado != "verificada" || !r.ConsumosHistoricosSinFechaLigada || !r.FechaConsumoLigadaCotejada ||
		r.ContenidoConsumoRecalculado || r.CamposFueraHuellaVerificados || r.ActorPerfilContextoCotejados {
		t.Fatalf("alcance de cadena mixto incorrecto: %+v", r)
	}
	if d.Registros[0].Consumo == nil || d.Registros[1].ConsumoOrigen == nil || d.Registros[2].ConsumoFecha == nil {
		t.Fatal("dispatch de consumo incorrecto")
	}
	// El esquema mixto sin eventos administrativos usa las mismas versiones.
	d.Esquema = EsquemaVerificacionMixta
	if r := VerificarCadenaMixtaV2(d, d.Manifiesto, 3); r.Estado != "verificada" {
		t.Fatalf("consumos mixtos v2 rechazados: %+v", r)
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var recuperado DocumentoVerificacionMixta
	if err := json.Unmarshal(b, &recuperado); err != nil {
		t.Fatal(err)
	}
	if r := VerificarCadenaMixtaV2(recuperado, d.Manifiesto, 3); r.Estado != "verificada" {
		t.Fatal("el JSON consumo no conserva las proyecciones")
	}
	// Un subrango solo AD173 no hereda una afirmación sobre fecha histórica.
	d = vectorConsumosMixtosAD173(t)
	d.Registros = d.Registros[2:]
	d.Manifiesto.PrimeraSecuencia, d.Manifiesto.Registros = 3, 1
	d.Manifiesto.AnteriorSHA256 = d.Registros[0].ConsumoFecha.AnteriorSHA256
	r = VerificarCadenaMixtaV3(d, d.Manifiesto, 1)
	if r.Estado != "verificada" || r.ConsumosHistoricosSinFechaLigada || !r.FechaConsumoLigadaCotejada {
		t.Fatalf("alcance del subrango AD173 incorrecto: %+v", r)
	}
}

func TestCadenaConsumosMixtosAD173AlteracionesYUnicidad(t *testing.T) {
	for _, tc := range []struct {
		nombre string
		mutar  func(*DocumentoVerificacionMixta)
		codigo string
	}{
		{"fecha", func(d *DocumentoVerificacionMixta) {
			d.Registros[2].ConsumoFecha.RegistradaEn = "2026-10-03T12:34:56.123457Z"
		}, "instante_distinto"},
		{"actor", func(d *DocumentoVerificacionMixta) {
			d.Registros[2].ConsumoFecha.ActorRef = "per_cccccccccccccccccccccc"
		}, "huella_distinta"},
		{"perfil", func(d *DocumentoVerificacionMixta) {
			d.Registros[2].ConsumoFecha.PerfilActivoRef = "prf_cccccccccccccccccccccc"
		}, "huella_distinta"},
		{"finalidad", func(d *DocumentoVerificacionMixta) { d.Registros[2].ConsumoFecha.FinalidadRef = "otra_finalidad" }, "huella_distinta"},
		{"origen", func(d *DocumentoVerificacionMixta) { d.Registros[1].ConsumoOrigen.Proceso = "otro_proceso" }, "huella_distinta"},
		{"version", func(d *DocumentoVerificacionMixta) { d.Registros[2].ConsumoFecha.VersionConsumo = 2 }, "tipo_invalido"},
		{"orden", func(d *DocumentoVerificacionMixta) { d.Registros[1], d.Registros[2] = d.Registros[2], d.Registros[1] }, "secuencia_distinta"},
		{"union", func(d *DocumentoVerificacionMixta) { d.Registros[2].ConsumoOrigen = d.Registros[1].ConsumoOrigen }, "tipo_invalido"},
		{"union_intento", func(d *DocumentoVerificacionMixta) { d.Registros[2].Intento = &RegistroIntentoV2{} }, "tipo_invalido"},
		{"duplicado", func(d *DocumentoVerificacionMixta) {
			c := d.Registros[2].ConsumoFecha
			c.DecisionRef = d.Registros[0].Consumo.DecisionRef
			c.HuellaSHA256 = huellaConsumoFechaV3(*c)
			d.Manifiesto.CabezaSHA256 = c.HuellaSHA256
		}, "consumo_duplicado"},
		{"duplicado_huella", func(d *DocumentoVerificacionMixta) {
			c := d.Registros[2].ConsumoFecha
			c.ConsumoHuellaSHA256, c.AuditoriaRef = d.Registros[0].Consumo.ConsumoHuellaSHA256, d.Registros[0].Consumo.AuditoriaRef
			c.HuellaSHA256 = huellaConsumoFechaV3(*c)
			d.Manifiesto.CabezaSHA256 = c.HuellaSHA256
		}, "consumo_duplicado"},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			d := vectorConsumosMixtosAD173(t)
			tc.mutar(&d)
			r := VerificarCadenaMixtaV3(d, d.Manifiesto, 3)
			if r.Fallo == nil || r.Fallo.Codigo != tc.codigo || r.FechaConsumoLigadaCotejada {
				t.Fatalf("alteración no rechazada: %+v", r)
			}
		})
	}
}
