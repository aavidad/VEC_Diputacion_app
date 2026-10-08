package auditoria

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Fixture sintética con material AD219 y eslabones AD207 precalculados fuera
// del verificador Go. No procede de PostgreSQL ni acredita una firma.
func documentoCatalogoAccionesPrueba(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	b, err := os.ReadFile("testdata/catalogo_acciones_ad219_sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestCatalogoAccionesCotejaMaterialYCadenaAD207(t *testing.T) {
	casos := map[string]func(*DocumentoVerificacionMixta){
		"valido": nil,
		"aprobacion_ref": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].CatalogoAcciones.AprobacionRef = "aprobacion:catalogo:distinta"
		},
		"aprobacion_huella": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].CatalogoAcciones.AprobacionSHA256 = strings.Repeat("a", 64)
		},
		"paquete_huella": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].CatalogoAcciones.PaqueteSHA256 = strings.Repeat("b", 64)
		},
		"censo_huella": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].CatalogoAcciones.CensoSHA256 = strings.Repeat("c", 64)
		},
		"version": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].CatalogoAcciones.CatalogoVersion = "02"
		},
		"cantidad": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].CatalogoAcciones.EntradasNumero = "513"
		},
		"recurso": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].CatalogoAcciones.RecursoRef = "catalogo_acciones_admin:otra"
		},
		"motivo_cruzado": func(d *DocumentoVerificacionMixta) {
			d.Registros[1].IntentoCatalogoAcciones.MotivoRef = "catalogo_acciones_registrado"
		},
		"material": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].CatalogoAcciones.EventoMaterialSHA256 = strings.Repeat("d", 64)
		},
		"huella_asiento": func(d *DocumentoVerificacionMixta) {
			d.Registros[1].IntentoCatalogoAcciones.HuellaSHA256 = strings.Repeat("e", 64)
		},
		"huella_eslabon": func(d *DocumentoVerificacionMixta) {
			d.Registros[1].Eslabon.EslabonSHA256 = strings.Repeat("a", 64)
		},
		"cruce_familia": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].GobiernoUsuarios = &RegistroGobiernoUsuariosV1{}
		},
		"cruce_intento": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].IntentoCatalogoAcciones = &RegistroIntentoCatalogoAccionesV1{}
		},
		"orden": func(d *DocumentoVerificacionMixta) {
			d.Registros[0], d.Registros[1] = d.Registros[1], d.Registros[0]
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			d := documentoCatalogoAccionesPrueba(t)
			if cambiar != nil {
				cambiar(&d)
			}
			r := VerificarCadenaCatalogoAccionesV1(d, d.Manifiesto, 2)
			if nombre == "valido" {
				if r.Estado != "verificada" || r.Fallo != nil {
					t.Fatalf("cadena sintética rechazada: %+v", r.Fallo)
				}
				return
			}
			if r.Estado != "rechazada" || r.Fallo == nil {
				t.Fatalf("alteración %s aceptada", nombre)
			}
		})
	}
}

func TestCatalogoAccionesConservaFamiliasPreviasYNoAmpliaEsquemasAntiguos(t *testing.T) {
	anteriores := []DocumentoVerificacionMixta{
		vectorConsumosMixtosAD173(t), documentoAD193Prueba(t), vectorAD207(t),
		documentoFuentesPrueba(t), documentoIntentoFuentesPrueba(t), documentoUnidadInicialPrueba(t),
		documentoBootstrapIntentosPrueba(t), vectorMantenimientoFijo(t), vectorGobiernoUsuariosPrueba(t),
		documentoContextoPreV2Prueba(t), documentoPerfilesAsignablesPrueba(t), vectorIdentidadInterna(t),
	}
	for i, d := range anteriores {
		antes, err := json.Marshal(d.Registros)
		if err != nil {
			t.Fatal(err)
		}
		d.Esquema = EsquemaVerificacionCatalogoAcciones
		r := VerificarCadenaCatalogoAccionesV1(d, d.Manifiesto, uint64(len(d.Registros)))
		despues, err := json.Marshal(d.Registros)
		if err != nil {
			t.Fatal(err)
		}
		if r.Estado != "verificada" || string(antes) != string(despues) {
			t.Fatalf("familia anterior %d: %+v", i, r.Fallo)
		}
	}
	for _, esquema := range []string{EsquemaVerificacionPerfilesAsignables, EsquemaVerificacionIdentidadInternaSintetica} {
		d := documentoCatalogoAccionesPrueba(t)
		d.Esquema = esquema
		if r := verificarCadenaMixta(d, d.Manifiesto, 2, esquema); r.Estado != "rechazada" {
			t.Fatalf("esquema anterior %s admitió AD219", esquema)
		}
	}
}

// AD219 nace después del corte AD207. Incluso una captura de un solo asiento
// con checkpoint rehecho no puede convertirlo en un asiento histórico.
func TestCatalogoAccionesExigeEslabonAD207EnAmbasFamilias(t *testing.T) {
	for _, indice := range []int{0, 1} {
		t.Run([]string{"exito", "intento"}[indice], func(t *testing.T) {
			d := documentoCatalogoAccionesPrueba(t)
			r := d.Registros[indice]
			r.Eslabon = nil
			d.Registros = []RegistroMixtoV2{r}
			if indice == 0 {
				d.Manifiesto.PrimeraSecuencia = r.CatalogoAcciones.Secuencia
				d.Manifiesto.CabezaSHA256 = r.CatalogoAcciones.HuellaSHA256
			} else {
				d.Manifiesto.PrimeraSecuencia = r.IntentoCatalogoAcciones.Secuencia
				d.Manifiesto.CabezaSHA256 = r.IntentoCatalogoAcciones.HuellaSHA256
			}
			d.Manifiesto.UltimaSecuencia = d.Manifiesto.PrimeraSecuencia
			d.Manifiesto.Registros = 1
			d.Manifiesto.AnteriorSHA256 = MarcadorSinAnteriorV5
			informe := VerificarCadenaCatalogoAccionesV1(d, d.Manifiesto, 1)
			if informe.Estado != "rechazada" || informe.Fallo == nil || informe.Fallo.Codigo != "eslabon_ausente" {
				t.Fatalf("asiento AD219 sin eslabón: %+v", informe.Fallo)
			}
		})
	}
}
