package auditoria

import (
	"os"
	"strings"
	"testing"
)

// El vector procede del laboratorio PostgreSQL 18 de AD207 (copia fría
// post-AD197 + AD198 + AD207): tres asientos anteriores al corte y cuatro
// sellados después, entre ellos la política y una captura del sello periódico
// que fija la cabeza sellada. Se extrajo tras reiniciar PostgreSQL.
func vectorAD207(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	raw, err := os.ReadFile("testdata/periodica_ad207.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err := decodificarDocumentoExportacionEstricto(raw, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestAD207VectorPostgreSQLCruzaElCorte(t *testing.T) {
	d := vectorAD207(t)
	antes, despues := 0, 0
	for _, r := range d.Registros {
		if r.Eslabon == nil {
			antes++
		} else if r.Periodica.AnteriorSHA256 == MarcadorSinAnteriorV5 {
			despues++
		}
	}
	if antes != 3 || despues != 4 {
		t.Fatalf("vector distinto: antes=%d despues=%d", antes, despues)
	}
	if r := VerificarCadenaPeriodicaV1(d, d.Manifiesto, 10); r.Estado != "verificada" {
		t.Fatalf("vector AD207: %+v fallo=%+v", r, r.Fallo)
	}
}

func TestAD207AlteracionesDetectadas(t *testing.T) {
	casos := map[string]func(*DocumentoVerificacionMixta){
		"eslabon alterado": func(d *DocumentoVerificacionMixta) {
			d.Registros[4].Eslabon.EslabonSHA256 = strings.Repeat("a", 64)
		},
		"eslabon que no enlaza": func(d *DocumentoVerificacionMixta) {
			d.Registros[5].Eslabon.AnteriorSHA256 = d.Registros[3].Eslabon.EslabonSHA256
		},
		"posicion cambiada": func(d *DocumentoVerificacionMixta) {
			d.Registros[3].Eslabon.Posicion++
		},
		"numero de otro asiento": func(d *DocumentoVerificacionMixta) {
			d.Registros[4].Eslabon.Secuencia = d.Registros[3].Eslabon.Secuencia
		},
		"asiento posterior sin eslabon": func(d *DocumentoVerificacionMixta) {
			d.Registros[5].Eslabon = nil
		},
		"asiento posterior sin marcador": func(d *DocumentoVerificacionMixta) {
			d.Registros[5].Periodica.AnteriorSHA256 = d.Registros[4].Eslabon.EslabonSHA256
		},
		"asientos intercambiados": func(d *DocumentoVerificacionMixta) {
			d.Registros[5], d.Registros[6] = d.Registros[6], d.Registros[5]
		},
		"asiento suprimido": func(d *DocumentoVerificacionMixta) {
			d.Registros = append(d.Registros[:4], d.Registros[5:]...)
			d.Manifiesto.UltimaSecuencia--
			d.Manifiesto.Registros--
		},
		"contenido alterado": func(d *DocumentoVerificacionMixta) {
			d.Registros[6].Periodica.CorrelacionRef = "correlacion_" + strings.Repeat("0", 32)
		},
		"fecha del asiento distinta": func(d *DocumentoVerificacionMixta) {
			d.Registros[5].Eslabon.RegistradaEn = d.Registros[4].Eslabon.RegistradaEn
		},
		"fecha de sellado alterada": func(d *DocumentoVerificacionMixta) {
			d.Registros[6].Eslabon.SelladoEn = "2026-01-01T00:00:00.000000Z"
		},
		"numero por debajo del corte": func(d *DocumentoVerificacionMixta) {
			d.Registros[3].Eslabon.Secuencia = d.Registros[2].Periodica.Secuencia
		},
		"cabeza distinta": func(d *DocumentoVerificacionMixta) {
			d.Manifiesto.CabezaSHA256 = strings.Repeat("b", 64)
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			d := vectorAD207(t)
			alterar(&d)
			if r := VerificarCadenaPeriodicaV1(d, d.Manifiesto, 10); r.Estado == "verificada" {
				t.Fatal("alteración no detectada")
			}
		})
	}
}

func TestHuellaEslabonV5CoincideConPostgreSQL(t *testing.T) {
	d := vectorAD207(t)
	r := d.Registros[3]
	got := HuellaEslabonV5("interna", r.Eslabon.Posicion, r.Eslabon.AnteriorSHA256, r.Eslabon.Secuencia,
		r.Periodica.AuditoriaRef, r.TipoRegistro, r.Periodica.HuellaSHA256, r.Eslabon.RegistradaEn, r.Eslabon.SelladoEn)
	if got != r.Eslabon.EslabonSHA256 || r.Eslabon.AnteriorSHA256 != d.Registros[2].Periodica.HuellaSHA256 {
		t.Fatalf("eslabón %s, PostgreSQL %s", got, r.Eslabon.EslabonSHA256)
	}
}
