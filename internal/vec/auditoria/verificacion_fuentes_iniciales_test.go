package auditoria

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func documentoFuentesPrueba(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	b, err := os.ReadFile("../../../cmd/vec-auditoria-verificar/testdata/fuentes_iniciales_ad174.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestAD174VectoresYAlcance(t *testing.T) {
	d := documentoFuentesPrueba(t)
	informe := VerificarCadenaFuentesInicialesV1(d, d.Manifiesto, 2)
	if informe.Estado != "verificada" || !informe.MaterialFuentesRecalculado || informe.MaterialEventoRecalculado ||
		informe.ActorPerfilContextoCotejados || informe.MaterialIntentoRecalculado || informe.ContenidoConsumoRecalculado ||
		informe.AutenticidadCheckpoint != "no_comprobada" || informe.AutenticidadFuentesHistoricas != "no_comprobada" {
		t.Fatalf("vector o alcance divergente: %+v", informe)
	}
	for _, esquema := range []string{EsquemaVerificacionMixta, EsquemaVerificacionPreperfil} {
		d.Esquema = esquema
		if r := verificarCadenaMixta(d, d.Manifiesto, 2, esquema); r.Estado != "rechazada" {
			t.Fatal("esquema anterior admitió AD174")
		}
	}
}

func TestAD174FechaNoParseablePropagaMotivoCerrado(t *testing.T) {
	d := documentoFuentesPrueba(t)
	d.Registros[0].FuentesIniciales.RegistradaEn = "dato_privado_sintetico:no_fecha"
	r := VerificarCadenaFuentesInicialesV1(d, d.Manifiesto, 2)
	b, _ := json.Marshal(r)
	if r.Estado != "rechazada" || r.Fallo == nil || r.Fallo.Codigo != MotivoInstanteAD171Invalido ||
		r.Fallo.Clave != "registrada_en" || strings.Contains(string(b), "dato_privado") {
		t.Fatalf("motivo de parseo no propagado o fecha expuesta: %s", b)
	}
}

func TestAD174RechazaCambiosCrucesYDuplicados(t *testing.T) {
	casos := map[string]func(*DocumentoVerificacionMixta){
		"plan": func(d *DocumentoVerificacionMixta) { d.Registros[0].FuentesIniciales.PlanRef = "plan:distinto" },
		"configuracion": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].FuentesIniciales.ConfiguracionSHA256 = strings.Repeat("5", 64)
		},
		"preimagen": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].FuentesIniciales.PreimagenSHA256 = strings.Repeat("5", 64)
		},
		"operador": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].FuentesIniciales.OperadorLogin = "dato_privado_sintetico"
		},
		"alcance": func(d *DocumentoVerificacionMixta) { d.Registros[0].FuentesIniciales.AlcanceFuente = "institucional" },
		"fuente": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].FuentesIniciales.FuenteSHA256 = strings.Repeat("5", 64)
		},
		"fecha": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].FuentesIniciales.RegistradaEn = "2026-10-03T21:00:00.123456+00:00"
		},
		"huella": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].FuentesIniciales.HuellaSHA256 = strings.Repeat("5", 64)
		},
		"resultado": func(d *DocumentoVerificacionMixta) { d.Registros[0].FuentesIniciales.Resultado = "denegado" },
		"cruce":     func(d *DocumentoVerificacionMixta) { d.Registros[0].Bootstrap = &RegistroBootstrapV3{} },
		"tipo":      func(d *DocumentoVerificacionMixta) { d.Registros[0].TipoRegistro = "bootstrap_operador" },
		"secuencia": func(d *DocumentoVerificacionMixta) { d.Registros[0].FuentesIniciales.Secuencia++ },
		"duplicado": func(d *DocumentoVerificacionMixta) {
			d.Registros[1] = d.Registros[0]
			d.Registros[1].FuentesIniciales.Secuencia = 2
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			d := documentoFuentesPrueba(t)
			mutar(&d)
			r := VerificarCadenaFuentesInicialesV1(d, d.Manifiesto, 2)
			b, _ := json.Marshal(r)
			if r.Estado != "rechazada" || r.Fallo == nil || r.MaterialFuentesRecalculado || strings.Contains(string(b), "dato_privado") {
				t.Fatalf("no rechazó: %s", b)
			}
		})
	}
}

func TestAD174ConservaFamiliasHistoricas(t *testing.T) {
	d := vectorMixtoV2()
	d.Esquema = EsquemaVerificacionFuentesIniciales
	if r := VerificarCadenaFuentesInicialesV1(d, d.Manifiesto, 2); r.Estado != "verificada" || !r.MaterialIntentoRecalculado || r.MaterialFuentesRecalculado {
		t.Fatalf("histórico: %+v", r)
	}
	for _, registro := range vectoresAD171Prueba(t) {
		d := documentoAD171Prueba(registro)
		d.Esquema = EsquemaVerificacionFuentesIniciales
		if r := VerificarCadenaFuentesInicialesV1(d, d.Manifiesto, 1); r.Estado != "verificada" || !r.MaterialEventoRecalculado || r.MaterialFuentesRecalculado {
			t.Fatalf("AD171: %+v", r)
		}
		d.Registros[0].FuentesIniciales = &RegistroFuentesInicialesV1{}
		d.Esquema = EsquemaVerificacionPreperfil
		if r := VerificarCadenaMixtaV3(d, d.Manifiesto, 1); r.Estado != "rechazada" {
			t.Fatal("cruce histórico admitido")
		}
	}
}
