package auditoria

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Cinco tipos sintéticos con material AD221 y eslabones AD207 precalculados
// fuera de Go. El fixture no procede de PostgreSQL ni acredita autenticidad.
func documentoPresentacionCertificadoPrueba(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	b, err := os.ReadFile("testdata/presentacion_certificado_ad221_sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestPresentacionCertificadoCotejaCincoTiposYHuellas(t *testing.T) {
	casos := map[string]func(*DocumentoVerificacionMixta){
		"valido": nil,
		"recibo": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PresentacionCertificado.ReciboSHA256 = strings.Repeat("a", 64)
		},
		"sujeto": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PresentacionCertificado.SujetoIDHMAC = strings.Repeat("b", 64)
		},
		"canal": func(d *DocumentoVerificacionMixta) {
			v := strings.Repeat("c", 64)
			d.Registros[1].PresentacionCertificado.CanalSHA256 = &v
		},
		"asercion": func(d *DocumentoVerificacionMixta) {
			v := strings.Repeat("d", 64)
			d.Registros[2].PresentacionCertificado.AsercionActualSHA256 = &v
		},
		"actor_actual": func(d *DocumentoVerificacionMixta) {
			v := "cta_abcdefghijklmnopqrstuv"
			d.Registros[3].PresentacionCertificado.ActorCuentaRef = &v
		},
		"actor_inventado_is2": func(d *DocumentoVerificacionMixta) {
			v := "cta_abcdefghijklmnopqrstuv"
			d.Registros[4].PresentacionCertificado.ActorCuentaRef = &v
		},
		"control_omitido_is2": func(d *DocumentoVerificacionMixta) {
			d.Registros[4].PresentacionCertificado.ControlSesionRef = nil
		},
		"revision_control": func(d *DocumentoVerificacionMixta) {
			v := "07"
			d.Registros[4].PresentacionCertificado.ControlSesionRevision = &v
		},
		"origen_control": func(d *DocumentoVerificacionMixta) {
			v := "opr_abcdefghijklmnopqrstuv"
			d.Registros[4].PresentacionCertificado.ControlOrigenOperacionRef = &v
		},
		"control_en_nominal": func(d *DocumentoVerificacionMixta) {
			v := "cse_abcdefghijklmnopqrstuv"
			d.Registros[0].PresentacionCertificado.ControlSesionRef = &v
		},
		"fase_is2": func(d *DocumentoVerificacionMixta) {
			d.Registros[4].PresentacionCertificado.Fase = "pre_f1_sin_perfil_activo"
		},
		"canal_is2": func(d *DocumentoVerificacionMixta) {
			d.Registros[4].PresentacionCertificado.Canal = "identidad_sesion_vigente"
		},
		"politica_actual": func(d *DocumentoVerificacionMixta) {
			v := "pga_abcdefghijklmnopqrstuv"
			d.Registros[1].PresentacionCertificado.PoliticaActualRef = &v
		},
		"acr_actual": func(d *DocumentoVerificacionMixta) {
			v := "urn:vec:acr:otro"
			d.Registros[0].PresentacionCertificado.ACRActual = &v
		},
		"caducidad": func(d *DocumentoVerificacionMixta) {
			v := d.Registros[0].PresentacionCertificado.RegistradaEn
			d.Registros[0].PresentacionCertificado.PresentacionValidaHasta = &v
		},
		"caducidad_ilegible": func(d *DocumentoVerificacionMixta) {
			v := "contenido_privado_no_fecha"
			d.Registros[0].PresentacionCertificado.PresentacionValidaHasta = &v
		},
		"caducidad_submicrosegundo": func(d *DocumentoVerificacionMixta) {
			v := "2026-10-07T09:00:01.0000011Z"
			d.Registros[0].PresentacionCertificado.PresentacionValidaHasta = &v
		},
		"actual_nulo": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PresentacionCertificado.CanalSHA256 = nil
		},
		"revocacion_con_actual": func(d *DocumentoVerificacionMixta) {
			v := "urn:vec:acr:certificado-desarrollo-protegido"
			d.Registros[3].PresentacionCertificado.ACRActual = &v
		},
		"motivo": func(d *DocumentoVerificacionMixta) {
			d.Registros[2].PresentacionCertificado.MotivoRef = "presentacion_abierta"
		},
		"recurso": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PresentacionCertificado.RecursoRef = "presentacion_certificado:otra"
		},
		"evento_derivado": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PresentacionCertificado.EventoRef = "evento_" + strings.Repeat("e", 32)
		},
		"correlacion_derivada": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PresentacionCertificado.CorrelacionRef = "correlacion_" + strings.Repeat("f", 32)
		},
		"material": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PresentacionCertificado.EventoMaterialSHA256 = strings.Repeat("a", 64)
		},
		"huella_asiento": func(d *DocumentoVerificacionMixta) {
			d.Registros[2].PresentacionCertificado.HuellaSHA256 = strings.Repeat("b", 64)
		},
		"huella_eslabon": func(d *DocumentoVerificacionMixta) {
			d.Registros[3].Eslabon.EslabonSHA256 = strings.Repeat("c", 64)
		},
		"cruce": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].CatalogoAcciones = &RegistroCatalogoAccionesV1{}
		},
		"orden": func(d *DocumentoVerificacionMixta) {
			d.Registros[0], d.Registros[1] = d.Registros[1], d.Registros[0]
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			d := documentoPresentacionCertificadoPrueba(t)
			if cambiar != nil {
				cambiar(&d)
			}
			r := VerificarCadenaPresentacionCertificadoV1(d, d.Manifiesto, 5)
			if nombre == "valido" {
				if r.Estado != "verificada" || r.Fallo != nil || r.AutenticidadCheckpoint != "no_comprobada" {
					t.Fatalf("fixture sintético: estado=%s fallo=%+v autenticidad=%s", r.Estado, r.Fallo, r.AutenticidadCheckpoint)
				}
				return
			}
			if r.Estado != "rechazada" || r.Fallo == nil {
				t.Fatalf("alteración %s aceptada", nombre)
			}
			if nombre == "caducidad_ilegible" {
				b, err := json.Marshal(r)
				if err != nil || strings.Contains(string(b), "contenido_privado") {
					t.Fatalf("fecha privada en el informe: %s", b)
				}
			}
		})
	}
}

func TestPresentacionCertificadoCanonizaFechaJSONBParaMaterial(t *testing.T) {
	d := documentoPresentacionCertificadoPrueba(t)
	// PostgreSQL puede serializar timestamptz con offset en el JSONB. El
	// instante coincide con el material SQL UTC de seis decimales.
	const jsonb = "2026-10-07T10:00:01.000001+01:00"
	d.Registros[0].PresentacionCertificado.PresentacionValidaHasta = new(string)
	*d.Registros[0].PresentacionCertificado.PresentacionValidaHasta = jsonb
	r := VerificarCadenaPresentacionCertificadoV1(d, d.Manifiesto, 5)
	if r.Estado != "verificada" || *d.Registros[0].PresentacionCertificado.PresentacionValidaHasta != jsonb {
		t.Fatalf("fecha JSONB equivalente rechazada o reescrita: %+v", r.Fallo)
	}
}

func TestPresentacionCertificadoExigeEslabonEnAsientoAislado(t *testing.T) {
	for _, caso := range []struct {
		indice int
		nombre string
	}{{0, "apertura"}, {3, "revocacion"}, {4, "control_is2"}} {
		t.Run(caso.nombre, func(t *testing.T) {
			d := documentoPresentacionCertificadoPrueba(t)
			r := d.Registros[caso.indice]
			r.Eslabon = nil
			d.Registros = []RegistroMixtoV2{r}
			d.Manifiesto.PrimeraSecuencia = r.PresentacionCertificado.Secuencia
			d.Manifiesto.UltimaSecuencia = d.Manifiesto.PrimeraSecuencia
			d.Manifiesto.Registros = 1
			d.Manifiesto.AnteriorSHA256 = MarcadorSinAnteriorV5
			d.Manifiesto.CabezaSHA256 = r.PresentacionCertificado.HuellaSHA256
			informe := VerificarCadenaPresentacionCertificadoV1(d, d.Manifiesto, 1)
			if informe.Estado != "rechazada" || informe.Fallo == nil || informe.Fallo.Codigo != "eslabon_ausente" {
				t.Fatalf("AD221 sin eslabón: %+v", informe.Fallo)
			}
		})
	}
}

func TestPresentacionCertificadoConservaFamiliasAnteriores(t *testing.T) {
	anteriores := []DocumentoVerificacionMixta{
		vectorConsumosMixtosAD173(t), documentoAD193Prueba(t), vectorAD207(t),
		documentoFuentesPrueba(t), documentoIntentoFuentesPrueba(t), documentoUnidadInicialPrueba(t),
		documentoBootstrapIntentosPrueba(t), vectorMantenimientoFijo(t), vectorGobiernoUsuariosPrueba(t),
		documentoContextoPreV2Prueba(t), documentoPerfilesAsignablesPrueba(t), vectorIdentidadInterna(t),
		documentoCatalogoAccionesPrueba(t),
	}
	for i, d := range anteriores {
		antes, err := json.Marshal(d.Registros)
		if err != nil {
			t.Fatal(err)
		}
		d.Esquema = EsquemaVerificacionPresentacionCertificado
		r := VerificarCadenaPresentacionCertificadoV1(d, d.Manifiesto, uint64(len(d.Registros)))
		despues, err := json.Marshal(d.Registros)
		if err != nil {
			t.Fatal(err)
		}
		if r.Estado != "verificada" || string(antes) != string(despues) {
			t.Fatalf("familia anterior %d: %+v", i, r.Fallo)
		}
	}
	d := documentoPresentacionCertificadoPrueba(t)
	d.Esquema = EsquemaVerificacionCatalogoAcciones
	if r := verificarCadenaMixta(d, d.Manifiesto, 5, d.Esquema); r.Estado != "rechazada" {
		t.Fatal("esquema AD219 admitió AD221")
	}
}
