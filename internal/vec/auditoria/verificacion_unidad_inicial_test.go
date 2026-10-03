package auditoria

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func documentoUnidadInicialPrueba(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	b, err := os.ReadFile("../../../cmd/vec-auditoria-verificar/testdata/unidad_inicial_ad176.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err = json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestAD176VectoresFamiliasYAlcance(t *testing.T) {
	d := documentoUnidadInicialPrueba(t)
	r := VerificarCadenaUnidadInicialV1(d, d.Manifiesto, 6)
	if r.Estado != "verificada" || !r.MaterialUnidadRecalculado || !r.MaterialIntentosUnidadRecalculado || r.MaterialFuentesRecalculado || r.ActorPerfilContextoCotejados ||
		r.AutenticidadCheckpoint != "no_comprobada" || r.AutenticidadFuentesHistoricas != "no_comprobada" {
		t.Fatalf("vector o alcance divergente: %+v", r)
	}
	for _, esquema := range []string{EsquemaVerificacionMixta, EsquemaVerificacionPreperfil, EsquemaVerificacionFuentesIniciales} {
		d.Esquema = esquema
		if verificarCadenaMixta(d, d.Manifiesto, 6, esquema).Estado != "rechazada" {
			t.Fatal("esquema previo admitió AD176")
		}
	}
}

func TestAD176FechaNoParseablePropagaMotivoCerrado(t *testing.T) {
	for _, familia := range []string{"confirmacion", "intento"} {
		t.Run(familia, func(t *testing.T) {
			d := documentoUnidadInicialPrueba(t)
			if familia == "confirmacion" {
				d.Registros[0].UnidadInicial.RegistradaEn = "dato_privado_sintetico:no_fecha"
			} else {
				d.Registros[2].IntentoUnidadInicial.RegistradaEn = "dato_privado_sintetico:no_fecha"
			}
			r := VerificarCadenaUnidadInicialV1(d, d.Manifiesto, 6)
			b, _ := json.Marshal(r)
			if r.Estado != "rechazada" || r.Fallo == nil || r.Fallo.Codigo != MotivoInstanteAD171Invalido ||
				r.Fallo.Clave != "registrada_en" || strings.Contains(string(b), "dato_privado") {
				t.Fatalf("motivo de parseo no propagado o fecha expuesta: %s", b)
			}
		})
	}
}

func TestAD176ConservaHistoriaYRecibos(t *testing.T) {
	documentos := []DocumentoVerificacionMixta{documentoFuentesPrueba(t), documentoIntentoFuentesPrueba(t), vectorMixtoV2()}
	for _, r := range vectoresAD171Prueba(t) {
		documentos = append(documentos, documentoAD171Prueba(r))
	}
	for _, d := range documentos {
		antes, _ := json.Marshal(d.Registros)
		d.Esquema = EsquemaVerificacionUnidadInicial
		r := VerificarCadenaUnidadInicialV1(d, d.Manifiesto, uint64(len(d.Registros)))
		despues, _ := json.Marshal(d.Registros)
		if r.Estado != "verificada" || r.MaterialUnidadRecalculado || r.MaterialIntentosUnidadRecalculado || string(antes) != string(despues) {
			t.Fatalf("historia divergente: %+v", r)
		}
		d.Registros[0].UnidadInicial = &RegistroUnidadInicialPersonalV1{}
		if VerificarCadenaUnidadInicialV1(d, d.Manifiesto, uint64(len(d.Registros))).Estado != "rechazada" {
			t.Fatal("histórico aceptó campos nuevos cruzados")
		}
	}
}

func TestAD176RechazaMaterialCrucesYReferenciaAjena(t *testing.T) {
	casos := map[string]func(*DocumentoVerificacionMixta){
		"fuente": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].UnidadInicial.FuenteSHA256 = strings.Repeat("8", 64)
		},
		"recibo": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].UnidadInicial.ReciboSHA256 = strings.Repeat("8", 64)
		},
		"plan": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].UnidadInicial.PlanRef = "prc_" + strings.Repeat("a", 22)
		},
		"accion_fuentes": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].UnidadInicial.Accion = "provisionar_fuentes_iniciales_admin_v1"
		},
		"proceso": func(d *DocumentoVerificacionMixta) { d.Registros[0].UnidadInicial.Proceso = "cli" },
		"fecha": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].UnidadInicial.RegistradaEn = "2026-10-04T00:00:01.123456+00:00"
		},
		"cruce_ad174": func(d *DocumentoVerificacionMixta) { d.Registros[0].FuentesIniciales = &RegistroFuentesInicialesV1{} },
		"cruce_intento": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].IntentoUnidadInicial = &RegistroIntentoUnidadInicialPersonalV1{}
		},
		"cruce_confirmado": func(d *DocumentoVerificacionMixta) { d.Registros[2].UnidadInicial = &RegistroUnidadInicialPersonalV1{} },
		"motivo_libre": func(d *DocumentoVerificacionMixta) {
			d.Registros[2].IntentoUnidadInicial.MotivoRef = "dato_privado_sintetico"
		},
		"resultado": func(d *DocumentoVerificacionMixta) { d.Registros[2].IntentoUnidadInicial.Resultado = "error" },
		"requestsha": func(d *DocumentoVerificacionMixta) {
			d.Registros[2].IntentoUnidadInicial.SolicitudSHA256 = strings.Repeat("8", 64)
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			d := documentoUnidadInicialPrueba(t)
			mutar(&d)
			r := VerificarCadenaUnidadInicialV1(d, d.Manifiesto, 6)
			b, _ := json.Marshal(r)
			if r.Estado != "rechazada" || r.Fallo == nil || r.MaterialUnidadRecalculado || r.MaterialIntentosUnidadRecalculado || strings.Contains(string(b), "dato_privado") {
				t.Fatalf("rechazo divergente: %s", b)
			}
		})
	}
}
