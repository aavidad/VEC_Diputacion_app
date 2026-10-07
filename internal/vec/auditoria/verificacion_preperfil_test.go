package auditoria

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func vectoresAD171Prueba(t *testing.T) []RegistroMixtoV2 {
	t.Helper()
	b, err := os.ReadFile("../../../cmd/vec-auditoria-verificar/testdata/preperfil_ad171_vectores.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Vectores []map[string]json.RawMessage `json:"vectores"`
	}
	if json.Unmarshal(b, &fixture) != nil || len(fixture.Vectores) != 4 {
		t.Fatal("vectores SQL ausentes")
	}
	var registros []RegistroMixtoV2
	for _, v := range fixture.Vectores {
		var campos map[string]json.RawMessage
		if json.Unmarshal(v["evento"], &campos) != nil {
			t.Fatal("evento inválido")
		}
		var tipo string
		if json.Unmarshal(campos["tipo_registro"], &tipo) != nil {
			t.Fatal("familia inválida")
		}
		delete(campos, "tipo_registro")
		for _, clave := range []string{"auditoria_ref", "secuencia", "anterior_sha256", "huella_sha256", "registrada_en", "evento_material_sha256"} {
			campos[clave] = v[clave]
		}
		campos["modulo_id"] = json.RawMessage(`"administracion"`)
		b, _ := json.Marshal(campos)
		r := RegistroMixtoV2{TipoRegistro: tipo}
		if tipo == "bootstrap_operador" {
			r.Bootstrap = &RegistroBootstrapV3{}
			err = json.Unmarshal(b, r.Bootstrap)
		} else {
			r.Preperfil = &RegistroPreperfilV3{}
			err = json.Unmarshal(b, r.Preperfil)
		}
		if err != nil {
			t.Fatal(err)
		}
		registros = append(registros, r)
	}
	return registros
}

func documentoAD171Prueba(r RegistroMixtoV2) DocumentoVerificacionMixta {
	e := r.Preperfil
	var c RegistroEventoAdminV3
	if e != nil {
		c = e.RegistroEventoAdminV3
	} else {
		c = r.Bootstrap.RegistroEventoAdminV3
	}
	return DocumentoVerificacionMixta{Esquema: EsquemaVerificacionPreperfil, Registros: []RegistroMixtoV2{r},
		Manifiesto: CoberturaCadena{CadenaID: "cadena:sintetica:ad171", PrimeraSecuencia: c.Secuencia, UltimaSecuencia: c.Secuencia,
			AnteriorSHA256: c.AnteriorSHA256, CabezaSHA256: c.HuellaSHA256, Registros: 1}}
}

func TestAD171CuatroVectoresSQLYAlcance(t *testing.T) {
	for _, r := range vectoresAD171Prueba(t) {
		d := documentoAD171Prueba(r)
		informe := VerificarCadenaMixtaV3(d, d.Manifiesto, 1)
		if informe.Estado != "verificada" || !informe.MaterialEventoRecalculado || informe.MaterialIntentoRecalculado ||
			informe.ActorPerfilContextoCotejados || informe.ContenidoConsumoRecalculado || informe.CamposFueraHuellaVerificados ||
			informe.AutenticidadCheckpoint != "no_comprobada" || informe.AutenticidadFuentesHistoricas != "no_comprobada" {
			t.Fatalf("vector SQL o alcance divergente: %+v", informe)
		}
		if previo := VerificarCadenaMixtaV2(d, d.Manifiesto, 1); previo.Estado != "rechazada" {
			t.Fatal("V2 aceptó el esquema nuevo")
		}
		d.Esquema = EsquemaVerificacionMixta
		if previo := VerificarCadenaMixtaV2(d, d.Manifiesto, 1); previo.Estado != "rechazada" {
			t.Fatal("V2 aceptó una familia nueva")
		}
	}
}

func TestAD171RechazaMaterialDivergenteYCamposCruzados(t *testing.T) {
	casos := []struct {
		nombre string
		indice int
		mutar  func(*RegistroMixtoV2)
	}{
		{"actor", 0, func(r *RegistroMixtoV2) { r.Preperfil.ActorRef = "per_" + strings.Repeat("B", 22) }},
		{"plan", 3, func(r *RegistroMixtoV2) { r.Bootstrap.PlanSHA256 = strings.Repeat("3", 64) }},
		{"fuente", 0, func(r *RegistroMixtoV2) { r.Preperfil.FuenteSHA256 = strings.Repeat("4", 64) }},
		{"modulo", 0, func(r *RegistroMixtoV2) { r.Preperfil.ModuloID = "otro" }},
		{"cruce", 0, func(r *RegistroMixtoV2) { r.Bootstrap = &RegistroBootstrapV3{} }},
		{"consumo", 0, func(r *RegistroMixtoV2) { r.Consumo = &RegistroCadenaV3{} }},
		{"tipo", 0, func(r *RegistroMixtoV2) { r.TipoRegistro = "bootstrap_operador" }},
		{"resultado", 0, func(r *RegistroMixtoV2) { r.Preperfil.Resultado = "autorizado" }},
		{"fecha", 0, func(r *RegistroMixtoV2) { r.Preperfil.RegistradaEn = "2026-10-03T10:00:00.123456+00:00" }},
		{"huella", 0, func(r *RegistroMixtoV2) { r.Preperfil.HuellaSHA256 = strings.Repeat("5", 64) }},
		{"secuencia", 0, func(r *RegistroMixtoV2) { r.Preperfil.Secuencia++ }},
		{"referencia", 0, func(r *RegistroMixtoV2) { r.Preperfil.AuditoriaRef = "dato_privado_sintetico" }},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			r := vectoresAD171Prueba(t)[c.indice]
			d := documentoAD171Prueba(r)
			c.mutar(&d.Registros[0])
			informe := VerificarCadenaMixtaV3(d, d.Manifiesto, 1)
			b, _ := json.Marshal(informe)
			if informe.Estado != "rechazada" || informe.Fallo == nil || strings.Contains(string(b), "dato_privado") || informe.MaterialEventoRecalculado {
				t.Fatalf("material no rechazado: %s", b)
			}
		})
	}
}

func TestV3ConservaConsumosIntentosYRechazaUnionCruzada(t *testing.T) {
	d := vectorMixtoV2()
	antes, _ := json.Marshal(d)
	d.Esquema = EsquemaVerificacionPreperfil
	informe := VerificarCadenaMixtaV3(d, d.Manifiesto, 2)
	if informe.Estado != "verificada" || informe.MaterialEventoRecalculado || !informe.MaterialIntentoRecalculado {
		t.Fatalf("familias históricas divergentes: %+v", informe)
	}
	d.Esquema = EsquemaVerificacionMixta
	despues, _ := json.Marshal(d)
	if string(antes) != string(despues) {
		t.Fatal("cambió el material histórico")
	}
	for _, indice := range []int{0, 1} {
		d := vectorMixtoV2()
		d.Registros[indice].Preperfil = vectoresAD171Prueba(t)[0].Preperfil
		if informe := VerificarCadenaMixtaV2(d, d.Manifiesto, 2); informe.Estado != "rechazada" {
			t.Fatal("V2 aceptó campos cruzados")
		}
	}
}
