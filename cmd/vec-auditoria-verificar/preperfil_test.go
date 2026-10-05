package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/vec/auditoria"
)

func vectoresPreperfilCLI(t *testing.T) []map[string]any {
	t.Helper()
	b, err := os.ReadFile("testdata/preperfil_ad171_vectores.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Vectores []map[string]any `json:"vectores"`
	}
	if json.Unmarshal(b, &fixture) != nil || len(fixture.Vectores) != 4 {
		t.Fatal("vectores SQL ausentes")
	}
	return fixture.Vectores
}

func documentoPreperfilCLI(t *testing.T, vector map[string]any) (map[string]any, []string) {
	t.Helper()
	evento := vector["evento"].(map[string]any)
	tipo := evento["tipo_registro"].(string)
	delete(evento, "tipo_registro")
	for _, k := range []string{"auditoria_ref", "secuencia", "anterior_sha256", "huella_sha256", "registrada_en", "evento_material_sha256"} {
		evento[k] = vector[k]
	}
	evento["modulo_id"] = "administracion"
	key := "preperfil"
	if tipo == "bootstrap_operador" {
		key = "bootstrap"
	}
	cp := map[string]any{"cadena_id": "cadena:sintetica:ad171", "primera_secuencia": vector["secuencia"], "ultima_secuencia": vector["secuencia"],
		"anterior_sha256": vector["anterior_sha256"], "cabeza_sha256": vector["huella_sha256"], "registros": 1}
	b, _ := json.Marshal(cp)
	ruta := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	return map[string]any{"esquema": auditoria.EsquemaVerificacionPreperfil, "manifiesto": cp, "registros": []any{map[string]any{"tipo_registro": tipo, key: evento}}},
		[]string{"--checkpoint", ruta, "--max-bytes", "16384", "--max-registros", "6"}
}

func TestCLIVerificaCuatroVectoresAD171(t *testing.T) {
	for _, v := range vectoresPreperfilCLI(t) {
		d, args := documentoPreperfilCLI(t, v)
		b, _ := json.Marshal(d)
		var salida bytes.Buffer
		if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != 0 {
			t.Fatalf("vector SQL rechazado: %d %s", codigo, &salida)
		}
		var informe auditoria.InformeVerificacionV3
		if json.Unmarshal(salida.Bytes(), &informe) != nil || !informe.MaterialEventoRecalculado || informe.ActorPerfilContextoCotejados ||
			informe.AutenticidadFuentesHistoricas != "no_comprobada" || informe.AutenticidadCheckpoint != "no_comprobada" {
			t.Fatalf("alcance inventado: %s", &salida)
		}
	}
}

func TestCLIV3DetectaCambiosRechazaCamposCruzadosYNoReflejaDatos(t *testing.T) {
	const privado = "dato_privado_sintetico"
	for _, caso := range []struct {
		nombre string
		indice int
		codigo int
		mutar  func(map[string]any, map[string]any)
	}{
		{"actor", 0, 1, func(d, e map[string]any) { e["actor_ref"] = "per_" + strings.Repeat("B", 22) }},
		{"plan", 3, 1, func(d, e map[string]any) { e["plan_sha256"] = strings.Repeat("3", 64) }},
		{"campo_sha", 0, 1, func(d, e map[string]any) { e["fuente_sha256"] = privado }},
		{"perfil", 0, 2, func(d, e map[string]any) { e["perfil_activo_ref"] = privado }},
		{"contexto", 0, 2, func(d, e map[string]any) { e["contexto_sha256"] = strings.Repeat("a", 64) }},
		{"decision", 0, 2, func(d, e map[string]any) { e["decision_ref"] = privado }},
		{"actor_bootstrap", 3, 2, func(d, e map[string]any) { e["actor_ref"] = privado }},
		{"plan_preperfil", 0, 2, func(d, e map[string]any) { e["plan_sha256"] = strings.Repeat("3", 64) }},
		{"null", 0, 2, func(d, e map[string]any) { e["actor_ref"] = nil }},
		{"ausente", 0, 2, func(d, e map[string]any) { delete(e, "motivo_ref") }},
		{"desconocido", 0, 2, func(d, e map[string]any) { e["secreto"] = privado }},
		{"schema_v2", 0, 2, func(d, e map[string]any) { d["esquema"] = auditoria.EsquemaVerificacionMixta }},
		{"union", 0, 2, func(d, e map[string]any) { d["registros"].([]any)[0].(map[string]any)["bootstrap"] = map[string]any{} }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			d, args := documentoPreperfilCLI(t, vectoresPreperfilCLI(t)[caso.indice])
			r := d["registros"].([]any)[0].(map[string]any)
			key := "preperfil"
			if caso.indice == 3 {
				key = "bootstrap"
			}
			caso.mutar(d, r[key].(map[string]any))
			b, _ := json.Marshal(d)
			var salida bytes.Buffer
			if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != caso.codigo || strings.Contains(salida.String(), privado) {
				t.Fatalf("rechazo o minimización incorrectos: %d %s", codigo, &salida)
			}
		})
	}
}

func TestCLIV3CadenaIntercaladaYFramingAnterior(t *testing.T) {
	b, err := os.ReadFile("testdata/preperfil_mixta_v3.json")
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"--checkpoint", "testdata/preperfil_checkpoint_v3.json", "--max-bytes", "16384", "--max-registros", "6"}
	var salida bytes.Buffer
	if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != 0 {
		t.Fatalf("mezcla rechazada: %d %s", codigo, &salida)
	}
	var informe auditoria.InformeVerificacionV3
	if json.Unmarshal(salida.Bytes(), &informe) != nil || !informe.MaterialEventoRecalculado || !informe.MaterialIntentoRecalculado || !informe.ActorPerfilContextoCotejados || informe.Cobertura.Registros != 6 {
		t.Fatalf("mezcla incompleta: %s", &salida)
	}
	// La primera fila antigua conserva el hash fijo de #506. Los otros eslabones
	// intercalados fueron calculados por Python, fuera del código del verificador.
	var d auditoria.DocumentoVerificacionMixta
	if json.Unmarshal(b, &d) != nil || d.Registros[0].Intento.HuellaSHA256 != "459a8adba29e2e4e21ad4aef3c468ed7c0608e8bbba7f7d9901e1e1a7111e328" {
		t.Fatal("preimagen histórica alterada")
	}
	d.Registros[1], d.Registros[2] = d.Registros[2], d.Registros[1]
	alterado, _ := json.Marshal(d)
	salida.Reset()
	if codigo := ejecutar(args, bytes.NewReader(alterado), &salida); codigo != 1 {
		t.Fatal("orden cambiado aceptado")
	}
	// No se admite capitalización ambigua, duplicados ni null en una fila nueva.
	for _, alterado := range [][]byte{
		bytes.Replace(b, []byte(`"actor_ref":"per_AAAAAAAAAAAAAAAAAAAAAA"`), []byte(`"actor_ref":"otro","actor_ref":"per_AAAAAAAAAAAAAAAAAAAAAA"`), 1),
		bytes.Replace(b, []byte(`"evento_ref":`), []byte(`"EVENTO_REF":`), 1),
		bytes.Replace(b, []byte(`"preperfil":{`), []byte(`"preperfil":null,"preperfil":{`), 1),
	} {
		salida.Reset()
		if codigo := ejecutar(args, bytes.NewReader(alterado), &salida); codigo != 2 {
			t.Fatalf("JSON ambiguo aceptado: %d", codigo)
		}
	}
}
