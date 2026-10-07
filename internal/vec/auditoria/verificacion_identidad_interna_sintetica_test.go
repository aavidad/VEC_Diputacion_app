package auditoria

import (
	"strings"
	"testing"
)

// Huellas de este vector calculadas por separado con SHA-256 y el encuadre
// PostgreSQL length:UTF8+LF; no se obtienen de funciones del verificador.
func vectorIdentidadInterna(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	const marca = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	const cero = "0000000000000000000000000000000000000000000000000000000000000000"
	primero := RegistroProvisionIdentidadInternaSinteticaV1{
		RegistroOperacionMantenimientoV1: RegistroOperacionMantenimientoV1{
			AuditoriaRef: "aud_v3_ii_" + strings.Repeat("a", 32), Secuencia: 1, AnteriorSHA256: marca,
			HuellaSHA256: "4db3729d9053cacde1e891d3a271afa94f2826b9c6e0240204705fe15428bfc4",
			RegistradaEn: "2026-10-07T12:00:00.123456Z", EventoRef: "evento_" + strings.Repeat("a", 32),
			EventoMaterialSHA256: "db2227c537464191633346f7996485951b70f7d7357b4ee84be2c043f270bb75",
			OperadorLogin:        "operador_sintetico", Accion: "provisionar_identidad_interna_sintetica_v1",
			ModuloID: "administracion", RecursoRef: "identidad_interna_sintetica:piis_" + strings.Repeat("B", 22),
			Resultado: "permitido", MotivoRef: "identidad_interna_registrada", Proceso: "postgresql",
			Canal: "operacion_tecnica_privada", FinalidadRef: "identidad_interna_sintetica",
			CorrelacionRef: "correlacion_" + strings.Repeat("c", 32),
		},
		OperacionRef: "piis_" + strings.Repeat("B", 22), PlanRef: "plan:1", PlanSHA256: strings.Repeat("1", 64),
		PreimagenSHA256: strings.Repeat("2", 64), ConfiguracionSHA256: strings.Repeat("3", 64),
		AprobacionRef: "aprobacion:1", AlcanceFuente: "sintetico_declarado", FuenteRef: "fuente:1", FuenteSHA256: strings.Repeat("4", 64),
	}
	segundo := RegistroIntentoIdentidadInternaSinteticaV1{
		RegistroOperacionMantenimientoV1: RegistroOperacionMantenimientoV1{
			AuditoriaRef: "aud_v3_iii_" + strings.Repeat("b", 32), Secuencia: 2, AnteriorSHA256: marca,
			HuellaSHA256: "3be997ab0eb34e3431b23f9244e58c30279da3ab22299eaf729e1a3a5b3dc50f",
			RegistradaEn: "2026-10-07T12:00:02.123456Z", EventoRef: "evento_" + strings.Repeat("b", 32),
			EventoMaterialSHA256: "295b88c951147328192afc0d7f28e07422b3e1a38bd2a93b04c73be9b05e3188",
			OperadorLogin:        "operador_sintetico", Accion: "recuperar_identidad_interna_sintetica_v1",
			ModuloID: "administracion", RecursoRef: "solicitud_identidad_interna:" + strings.Repeat("d", 32),
			Resultado: "permitido", MotivoRef: "identidad_interna_recuperada", Proceso: "postgresql",
			Canal: "operacion_tecnica_privada", FinalidadRef: "identidad_interna_sintetica",
			CorrelacionRef: "correlacion_" + strings.Repeat("e", 32),
		}, SolicitudSHA256: strings.Repeat("5", 64),
	}
	return DocumentoVerificacionMixta{
		Esquema: EsquemaVerificacionIdentidadInternaSintetica,
		Manifiesto: CoberturaCadena{CadenaID: "interna", PrimeraSecuencia: 1, UltimaSecuencia: 2,
			AnteriorSHA256: cero, CabezaSHA256: "a3810defa5fce3a2a2d1ae6894f5ba40325b906e6322e35c7151db3176d7d5fd", Registros: 2},
		Registros: []RegistroMixtoV2{
			{TipoRegistro: "provision_identidad_interna_sintetica", ProvisionIdentidadInterna: &primero,
				Eslabon: &EslabonCadenaV5{Posicion: 1, Secuencia: 1, AnteriorSHA256: cero,
					EslabonSHA256: "24e339097f29d9d384f041b5a9654cbf5e6e2e3af6c8577b7979d4b7ab4a5c86",
					RegistradaEn:  primero.RegistradaEn, SelladoEn: "2026-10-07T12:00:01.123456Z"}},
			{TipoRegistro: "intento_identidad_interna_sintetica", IntentoIdentidadInterna: &segundo,
				Eslabon: &EslabonCadenaV5{Posicion: 2, Secuencia: 2,
					AnteriorSHA256: "24e339097f29d9d384f041b5a9654cbf5e6e2e3af6c8577b7979d4b7ab4a5c86",
					EslabonSHA256:  "a3810defa5fce3a2a2d1ae6894f5ba40325b906e6322e35c7151db3176d7d5fd",
					RegistradaEn:   segundo.RegistradaEn, SelladoEn: "2026-10-07T12:00:03.123456Z"}},
		},
	}
}

func TestIdentidadInternaVectorIndependienteYMutaciones(t *testing.T) {
	valido := vectorIdentidadInterna(t)
	if r := VerificarCadenaIdentidadInternaSinteticaV1(valido, valido.Manifiesto, 2); r.Estado != "verificada" {
		t.Fatalf("vector independiente: %+v", r.Fallo)
	}
	for _, caso := range []struct {
		nombre string
		mutar  func(*DocumentoVerificacionMixta)
	}{
		{"material", func(d *DocumentoVerificacionMixta) { d.Registros[0].ProvisionIdentidadInterna.PlanRef = "plan:2" }},
		{"retiro", func(d *DocumentoVerificacionMixta) { d.Registros = d.Registros[:1] }},
		{"version", func(d *DocumentoVerificacionMixta) {
			d.Esquema = "vec.auditoria.verificacion.identidad-interna-sintetica.v2"
		}},
		{"eslabon", func(d *DocumentoVerificacionMixta) { d.Registros[1].Eslabon.SelladoEn = "2026-10-07T12:00:04.123456Z" }},
		{"familia_ajena", func(d *DocumentoVerificacionMixta) { d.Registros[1].TipoRegistro = "intento_fuentes_iniciales_admin" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			d := vectorIdentidadInterna(t)
			caso.mutar(&d)
			if r := VerificarCadenaIdentidadInternaSinteticaV1(d, d.Manifiesto, 2); r.Estado != "rechazada" {
				t.Fatal("alteración aceptada")
			}
		})
	}
	viejo := vectorIdentidadInterna(t)
	viejo.Esquema = EsquemaVerificacionMantenimientoFijo
	if r := VerificarCadenaMantenimientoFijoV1(viejo, viejo.Manifiesto, 2); r.Estado != "rechazada" {
		t.Fatal("un esquema previo aceptó AD215")
	}
}
