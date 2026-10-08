package auditoria

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func documentoIntentoFuentesPrueba(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	b, err := os.ReadFile("../../../cmd/vec-auditoria-verificar/testdata/intentos_fuentes_ad174.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestAD174IntentosVectoresResultadoYAlcance(t *testing.T) {
	d := documentoIntentoFuentesPrueba(t)
	r := VerificarCadenaFuentesInicialesV1(d, d.Manifiesto, 4)
	if r.Estado != "verificada" || !r.MaterialIntentosFuentesRecalculado || r.MaterialFuentesRecalculado || r.ActorPerfilContextoCotejados || r.AutenticidadFuentesHistoricas != "no_comprobada" {
		t.Fatalf("alcance: %+v", r)
	}
	for _, esquema := range []string{EsquemaVerificacionMixta, EsquemaVerificacionPreperfil} {
		d.Esquema = esquema
		if verificarCadenaMixta(d, d.Manifiesto, 4, esquema).Estado != "rechazada" {
			t.Fatal("esquema previo admitió intento fuentes")
		}
	}
}

func TestAD174IntentoFechaNoParseablePropagaMotivoCerrado(t *testing.T) {
	d := documentoIntentoFuentesPrueba(t)
	d.Registros[0].IntentoFuentesIniciales.RegistradaEn = "dato_privado_sintetico:no_fecha"
	r := VerificarCadenaFuentesInicialesV1(d, d.Manifiesto, 4)
	b, _ := json.Marshal(r)
	if r.Estado != "rechazada" || r.Fallo == nil || r.Fallo.Codigo != MotivoInstanteAD171Invalido ||
		r.Fallo.Clave != "registrada_en" || strings.Contains(string(b), "dato_privado") {
		t.Fatalf("motivo de parseo no propagado o fecha expuesta: %s", b)
	}
}

func TestAD174IntentosRechazaAmbiguedad(t *testing.T) {
	casos := map[string]func(*RegistroMixtoV2){
		"fuente_confirmada": func(r *RegistroMixtoV2) { r.FuentesIniciales = &RegistroFuentesInicialesV1{} },
		"perfil":            func(r *RegistroMixtoV2) { r.Preperfil = &RegistroPreperfilV3{} },
		"solicitud":         func(r *RegistroMixtoV2) { r.IntentoFuentesIniciales.SolicitudSHA256 = strings.Repeat("9", 64) },
		"resultado":         func(r *RegistroMixtoV2) { r.IntentoFuentesIniciales.Resultado = "denegado" },
		"motivo":            func(r *RegistroMixtoV2) { r.IntentoFuentesIniciales.MotivoRef = "dato_privado_sintetico" },
		"proceso":           func(r *RegistroMixtoV2) { r.IntentoFuentesIniciales.Proceso = "cli" },
		"recurso":           func(r *RegistroMixtoV2) { r.IntentoFuentesIniciales.RecursoRef = "plan:declarativo" },
		"operador":          func(r *RegistroMixtoV2) { r.IntentoFuentesIniciales.OperadorLogin = "" },
		"fecha":             func(r *RegistroMixtoV2) { r.IntentoFuentesIniciales.RegistradaEn = "2026-10-03T21:00:01.123456+00:00" },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			d := documentoIntentoFuentesPrueba(t)
			mutar(&d.Registros[0])
			r := VerificarCadenaFuentesInicialesV1(d, d.Manifiesto, 4)
			b, _ := json.Marshal(r)
			if r.Estado != "rechazada" || r.MaterialIntentosFuentesRecalculado || strings.Contains(string(b), "dato_privado") {
				t.Fatalf("rechazo: %s", b)
			}
		})
	}
}
