package auditoria

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Ocho pares motivo/resultado sintéticos, con material y eslabones calculados
// fuera del verificador Go. No representan una entrada PostgreSQL instalada.
func documentoPreIdentidadTecnicaPrueba(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	b, err := os.ReadFile("testdata/pre_identidad_tecnica_ad222_sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestPreIdentidadTecnicaCotejaMaterialYCadenaAD207(t *testing.T) {
	casos := map[string]func(*DocumentoVerificacionMixta){
		"valido": nil,
		"metodo_ruta": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PreIdentidadTecnica.MetodoEsperado = "POST"
		},
		"fase": func(d *DocumentoVerificacionMixta) {
			d.Registros[1].PreIdentidadTecnica.Fase = "con_perfil"
		},
		"superficie": func(d *DocumentoVerificacionMixta) {
			d.Registros[2].PreIdentidadTecnica.Superficie = "administracion_privilegiada"
		},
		"canal": func(d *DocumentoVerificacionMixta) {
			d.Registros[3].PreIdentidadTecnica.Canal = "certificado_mtls_atestado_por_frontera"
		},
		"motivo_resultado": func(d *DocumentoVerificacionMixta) {
			d.Registros[6].PreIdentidadTecnica.Resultado = "denegado"
		},
		"recurso": func(d *DocumentoVerificacionMixta) {
			d.Registros[4].PreIdentidadTecnica.RecursoRef = "solicitud_sesion:" + strings.Repeat("a", 32)
		},
		"correlacion": func(d *DocumentoVerificacionMixta) {
			d.Registros[4].PreIdentidadTecnica.CorrelacionRef = "correlacion_" + strings.Repeat("b", 32)
		},
		"operador": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PreIdentidadTecnica.OperadorLogin = "otro_login"
		},
		"material": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PreIdentidadTecnica.EventoMaterialSHA256 = strings.Repeat("c", 64)
		},
		"huella_asiento": func(d *DocumentoVerificacionMixta) {
			d.Registros[1].PreIdentidadTecnica.HuellaSHA256 = strings.Repeat("d", 64)
		},
		"huella_eslabon": func(d *DocumentoVerificacionMixta) {
			d.Registros[2].Eslabon.EslabonSHA256 = strings.Repeat("e", 64)
		},
		"cruce_familia": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PresentacionCertificado = &RegistroPresentacionCertificadoV1{}
		},
		"orden": func(d *DocumentoVerificacionMixta) {
			d.Registros[0], d.Registros[1] = d.Registros[1], d.Registros[0]
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			d := documentoPreIdentidadTecnicaPrueba(t)
			if cambiar != nil {
				cambiar(&d)
			}
			r := VerificarCadenaPreIdentidadTecnicaV1(d, d.Manifiesto, 8)
			if nombre == "valido" {
				if r.Estado != "verificada" || r.Fallo != nil ||
					r.AutenticidadCheckpoint != "no_comprobada" || r.AutenticidadFuentesHistoricas != "no_comprobada" {
					t.Fatalf("fixture sintética rechazada o autenticidad atribuida: %+v", r)
				}
				for _, asiento := range d.Registros {
					if asiento.PreIdentidadTecnica.EventoMaterialSHA256 == asiento.PreIdentidadTecnica.HuellaSHA256 ||
						asiento.PreIdentidadTecnica.HuellaSHA256 == asiento.Eslabon.EslabonSHA256 {
						t.Fatal("material, asiento y sello confundidos")
					}
				}
				return
			}
			if r.Estado != "rechazada" || r.Fallo == nil {
				t.Fatalf("alteración %s aceptada", nombre)
			}
		})
	}
}

func TestPreIdentidadTecnicaExigeEslabonAislado(t *testing.T) {
	d := documentoPreIdentidadTecnicaPrueba(t)
	r := d.Registros[0]
	r.Eslabon = nil
	d.Registros = []RegistroMixtoV2{r}
	d.Manifiesto.PrimeraSecuencia = r.PreIdentidadTecnica.Secuencia
	d.Manifiesto.UltimaSecuencia = d.Manifiesto.PrimeraSecuencia
	d.Manifiesto.Registros = 1
	d.Manifiesto.AnteriorSHA256 = MarcadorSinAnteriorV5
	d.Manifiesto.CabezaSHA256 = r.PreIdentidadTecnica.HuellaSHA256
	informe := VerificarCadenaPreIdentidadTecnicaV1(d, d.Manifiesto, 1)
	if informe.Estado != "rechazada" || informe.Fallo == nil || informe.Fallo.Codigo != "eslabon_ausente" {
		t.Fatalf("asiento AD222 sin eslabón: %+v", informe.Fallo)
	}
}

func TestPreIdentidadTecnicaConservaAD219YAD221(t *testing.T) {
	for _, d := range []DocumentoVerificacionMixta{documentoCatalogoAccionesPrueba(t), documentoPresentacionCertificadoPrueba(t)} {
		antes, err := json.Marshal(d.Registros)
		if err != nil {
			t.Fatal(err)
		}
		d.Esquema = EsquemaVerificacionPreIdentidadTecnica
		informe := VerificarCadenaPreIdentidadTecnicaV1(d, d.Manifiesto, uint64(len(d.Registros)))
		despues, err := json.Marshal(d.Registros)
		if err != nil {
			t.Fatal(err)
		}
		if informe.Estado != "verificada" || string(antes) != string(despues) {
			t.Fatalf("familia anterior rechazada: %+v", informe.Fallo)
		}
	}
	d := documentoPreIdentidadTecnicaPrueba(t)
	d.Esquema = EsquemaVerificacionPresentacionCertificado
	if r := verificarCadenaMixta(d, d.Manifiesto, 8, d.Esquema); r.Estado != "rechazada" {
		t.Fatal("esquema AD221 anterior admitió AD222")
	}
}
