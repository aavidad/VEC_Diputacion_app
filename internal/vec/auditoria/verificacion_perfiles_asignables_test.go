package auditoria

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Los cuatro registros proceden del ensayo AUT49/AD196 en un clon desechable:
// registro, intento registrado, replay y un intento denegado.
func documentoPerfilesAsignablesPrueba(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	b, err := os.ReadFile("testdata/perfiles_asignables_ad196.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestPerfilesAsignablesVerificaRegistrosRealesYRechazaCambios(t *testing.T) {
	casos := map[string]func(*DocumentoVerificacionMixta){
		"valido": func(*DocumentoVerificacionMixta) {},
		"lista": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PerfilesAsignables.PerfilesSHA256 = strings.Repeat("e", 64)
		},
		"aprobacion": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PerfilesAsignables.AprobacionSHA256 = strings.Repeat("a", 64)
		},
		"numero": func(d *DocumentoVerificacionMixta) { d.Registros[0].PerfilesAsignables.PerfilesNumero = "03" },
		"operador": func(d *DocumentoVerificacionMixta) {
			d.Registros[1].IntentoPerfilesAsignables.OperadorLogin = "otro_login"
		},
		"motivo_cruzado": func(d *DocumentoVerificacionMixta) {
			d.Registros[3].IntentoPerfilesAsignables.MotivoRef = "perfiles_asignables_registrado"
		},
		"recurso": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].PerfilesAsignables.RecursoRef = "perfiles_asignables:otro"
		},
		"fecha": func(d *DocumentoVerificacionMixta) {
			d.Registros[2].IntentoPerfilesAsignables.RegistradaEn = "contenido_privado_no_fecha"
		},
		"orden": func(d *DocumentoVerificacionMixta) { d.Registros[1], d.Registros[2] = d.Registros[2], d.Registros[1] },
		"cruce": func(d *DocumentoVerificacionMixta) { d.Registros[0].GobiernoUsuarios = &RegistroGobiernoUsuariosV1{} },
		"familia_ajena": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].IntentoPerfilesAsignables = &RegistroIntentoPerfilesAsignablesV1{}
		},
		"esquema_antiguo": func(d *DocumentoVerificacionMixta) { d.Esquema = EsquemaVerificacionContextoAdminPreV2 },
	}
	for caso, cambiar := range casos {
		t.Run(caso, func(t *testing.T) {
			d := documentoPerfilesAsignablesPrueba(t)
			cambiar(&d)
			var r InformeVerificacion
			if d.Esquema == EsquemaVerificacionContextoAdminPreV2 {
				r = VerificarCadenaContextoAdminPreV2(d, d.Manifiesto, 4)
			} else {
				r = VerificarCadenaPerfilesAsignablesV1(d, d.Manifiesto, 4)
			}
			raw, _ := json.Marshal(r)
			if caso == "valido" {
				if r.Estado != "verificada" {
					t.Fatalf("estado=%s fallo=%+v", r.Estado, r.Fallo)
				}
				return
			}
			if r.Estado != "rechazada" || strings.Contains(string(raw), "contenido_privado") {
				t.Fatalf("alteracion aceptada o exportada: %s", raw)
			}
		})
	}
}

// El esquema nuevo debe seguir verificando todas las familias que ya puede
// contener la cadena común de la principal.
func TestPerfilesAsignablesConservaFamiliasPrevias(t *testing.T) {
	ds := []DocumentoVerificacionMixta{vectorConsumosMixtosAD173(t), documentoFuentesPrueba(t), documentoUnidadInicialPrueba(t),
		documentoBootstrapIntentosPrueba(t), vectorMantenimientoFijo(t), vectorGobiernoUsuariosPrueba(t)}
	b, err := os.ReadFile("testdata/contexto_admin_prev2.json")
	if err != nil {
		t.Fatal(err)
	}
	var prev2 DocumentoVerificacionMixta
	if err := json.Unmarshal(b, &prev2); err != nil {
		t.Fatal(err)
	}
	ds = append(ds, prev2)
	for i, d := range ds {
		antes, _ := json.Marshal(d.Registros)
		d.Esquema = EsquemaVerificacionPerfilesAsignables
		r := VerificarCadenaPerfilesAsignablesV1(d, d.Manifiesto, uint64(len(d.Registros)))
		despues, _ := json.Marshal(d.Registros)
		if r.Estado != "verificada" || string(antes) != string(despues) {
			t.Fatalf("familia previa %d no conservada: %+v", i, r.Fallo)
		}
	}
}

// Ningún esquema anterior admite los tipos de AD196.
func TestPerfilesAsignablesRechazadosEnEsquemasAnteriores(t *testing.T) {
	for _, esquema := range []string{EsquemaVerificacionContextoAdminPreV2, EsquemaVerificacionFronteraAdminTecnicaV1, EsquemaVerificacionGobiernoUsuarios, EsquemaVerificacionMantenimientoFijo} {
		d := documentoPerfilesAsignablesPrueba(t)
		d.Esquema = esquema
		if r := verificarCadenaMixta(d, d.Manifiesto, 4, esquema); r.Estado != "rechazada" {
			t.Fatalf("esquema %s admitio AD196", esquema)
		}
	}
}
