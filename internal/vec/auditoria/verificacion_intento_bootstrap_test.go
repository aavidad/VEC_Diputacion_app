package auditoria

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func documentoBootstrapIntentosPrueba(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	b, err := os.ReadFile("../../../cmd/vec-auditoria-verificar/testdata/bootstrap_intentos_ad179.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err = json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestAD179VectoresYAlcance(t *testing.T) {
	d := documentoBootstrapIntentosPrueba(t)
	r := VerificarCadenaBootstrapCentralV1(d, d.Manifiesto, 4)
	if r.Estado != "verificada" || !r.MaterialIntentosBootstrapRecalculado || r.MaterialEventoRecalculado || r.MaterialUnidadRecalculado ||
		r.ActorPerfilContextoCotejados || r.AutenticidadCheckpoint != "no_comprobada" || r.AutenticidadFuentesHistoricas != "no_comprobada" {
		t.Fatalf("vector o alcance: %+v", r)
	}
	for _, esquema := range []string{EsquemaVerificacionMixta, EsquemaVerificacionPreperfil, EsquemaVerificacionFuentesIniciales, EsquemaVerificacionUnidadInicial} {
		d.Esquema = esquema
		if verificarCadenaMixta(d, d.Manifiesto, 4, esquema).Estado != "rechazada" {
			t.Fatal("esquema anterior admitió AD179")
		}
	}
}

func TestAD179UnionConsumosYFamiliasTecnicas(t *testing.T) {
	b, err := os.ReadFile("../../../cmd/vec-auditoria-verificar/testdata/union_consumos_tecnicos_bootstrap_ad173_ad179.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err = json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	r := VerificarCadenaBootstrapCentralV1(d, d.Manifiesto, 19)
	if r.Estado != "verificada" || !r.ConsumosHistoricosSinFechaLigada || !r.FechaConsumoLigadaCotejada ||
		!r.MaterialFuentesRecalculado || !r.MaterialUnidadRecalculado || !r.MaterialIntentosBootstrapRecalculado || r.ActorPerfilContextoCotejados {
		t.Fatalf("unión bootstrap: %+v", r)
	}
	d.Registros[15].ConsumoFecha = d.Registros[2].ConsumoFecha
	if VerificarCadenaBootstrapCentralV1(d, d.Manifiesto, 19).Estado != "rechazada" {
		t.Fatal("intento bootstrap admitió consumo nominal cruzado")
	}
}

func TestAD179PreservaFamiliasYConfirmacionBootstrap(t *testing.T) {
	documentos := []DocumentoVerificacionMixta{vectorMixtoV2(), documentoFuentesPrueba(t), documentoIntentoFuentesPrueba(t), documentoUnidadInicialPrueba(t)}
	for _, r := range vectoresAD171Prueba(t) {
		documentos = append(documentos, documentoAD171Prueba(r))
	}
	for _, d := range documentos {
		antes, _ := json.Marshal(d.Registros)
		d.Esquema = EsquemaVerificacionBootstrapCentral
		r := VerificarCadenaBootstrapCentralV1(d, d.Manifiesto, uint64(len(d.Registros)))
		despues, _ := json.Marshal(d.Registros)
		if r.Estado != "verificada" || r.MaterialIntentosBootstrapRecalculado || string(antes) != string(despues) {
			t.Fatalf("historia o confirmación alterada: %+v", r)
		}
		d.Registros[0].IntentoBootstrapCentral = &RegistroIntentoBootstrapCentralV1{}
		if VerificarCadenaBootstrapCentralV1(d, d.Manifiesto, uint64(len(d.Registros))).Estado != "rechazada" {
			t.Fatal("histórico admitió cruce bootstrap")
		}
	}
}

func TestAD179RechazaCrucesMaterialYFechaNoParseable(t *testing.T) {
	casos := map[string]func(*RegistroMixtoV2){
		"confirmacion": func(r *RegistroMixtoV2) { r.Bootstrap = &RegistroBootstrapV3{} },
		"unidad":       func(r *RegistroMixtoV2) { r.UnidadInicial = &RegistroUnidadInicialPersonalV1{} },
		"actor":        func(r *RegistroMixtoV2) { r.Preperfil = &RegistroPreperfilV3{} },
		"solicitud":    func(r *RegistroMixtoV2) { r.IntentoBootstrapCentral.SolicitudSHA256 = strings.Repeat("9", 64) },
		"motivo":       func(r *RegistroMixtoV2) { r.IntentoBootstrapCentral.MotivoRef = "unidad_registrada" },
		"resultado":    func(r *RegistroMixtoV2) { r.IntentoBootstrapCentral.Resultado = "denegado" },
		"proceso":      func(r *RegistroMixtoV2) { r.IntentoBootstrapCentral.Proceso = "cli" },
		"recurso": func(r *RegistroMixtoV2) {
			r.IntentoBootstrapCentral.RecursoRef = "solicitud_unidad:" + strings.Repeat("1", 32)
		},
		"fecha": func(r *RegistroMixtoV2) { r.IntentoBootstrapCentral.RegistradaEn = "dato_privado_sintetico:no_fecha" },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			d := documentoBootstrapIntentosPrueba(t)
			mutar(&d.Registros[0])
			r := VerificarCadenaBootstrapCentralV1(d, d.Manifiesto, 4)
			b, _ := json.Marshal(r)
			if r.Estado != "rechazada" || r.Fallo == nil || r.MaterialIntentosBootstrapRecalculado || strings.Contains(string(b), "dato_privado") {
				t.Fatalf("rechazo: %s", b)
			}
			if nombre == "fecha" && (r.Fallo.Codigo != MotivoInstanteAD171Invalido || r.Fallo.Clave != "registrada_en") {
				t.Fatalf("parseo sin diagnóstico: %+v", r.Fallo)
			}
		})
	}
}
