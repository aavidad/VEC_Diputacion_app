package auditoria

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
)

// Vectores de encuadre obtenidos por separado con SHA-256 sobre los campos
// UTF-8 de AD169 y AD3-002. El contexto procede del vector sintético V2.
const contextoMixtoV2 = `{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":"per_rrrrrrrrrrrrrrrrrrrrrr","metodo":"certificado","garantia":"alto","perfil_activo_ref":"prf_pppppppppppppppppppppp","persona_ref":"per_rrrrrrrrrrrrrrrrrrrrrr","contexto_actor_ref":"vca_vvvvvvvvvvvvvvvvvvvvvv","contexto_version":3,"cuenta_ref":"cta_aaaaaaaaaaaaaaaaaaaaaa","cuenta_version":6,"persona_version":4,"perfil_version":5,"estado":"activo","vigente_desde":"2026-07-15T09:30:00.123000Z","vigente_hasta":"2026-07-15T11:30:00.123000Z","resuelto_en":"2026-07-15T10:30:00.123000Z","vinculos":[{"vinculo_ref":"vin_cccccccccccccccccccccc","version":7,"tipo":"candidato","referencia":"can_cccccccccccccccccccccc","estado":"activo","vigente_desde":"2026-07-15T09:30:00.123000Z","vigente_hasta":"2026-07-15T11:30:00.123000Z"},{"vinculo_ref":"vin_eeeeeeeeeeeeeeeeeeeeee","version":9,"tipo":"empleado","referencia":"emp_eeeeeeeeeeeeeeeeeeeeee","estado":"activo","vigente_desde":"2026-07-15T09:30:00.123000Z","vigente_hasta":"2026-07-15T11:30:00.123000Z"}]}`
const materialMixtoSHA = "e82dbeb37bedf8c11268e248c3fc51c4ca7b5978b6208f8f9700abd5b5c5dd9f"
const eslabonIntentoSHA = "459a8adba29e2e4e21ad4aef3c468ed7c0608e8bbba7f7d9901e1e1a7111e328"
const cabezaMixtaSHA = "009cd0bfaad025d8099400e0645ac481636c2428589f17c7eeea5df25facc107"

func vectorMixtoV2() DocumentoVerificacionMixta {
	a := RegistroIntentoV2{
		AuditoriaRef: "aud_v3_i_" + strings.Repeat("1", 32), Secuencia: 1,
		AnteriorSHA256: strings.Repeat("0", 64), HuellaSHA256: eslabonIntentoSHA,
		RegistradaEn: "2026-10-03T10:11:12.123456Z", IntentoRef: "intento_" + strings.Repeat("1", 32),
		IntentoMaterialSHA256: materialMixtoSHA,
		ActorRef:              "per_rrrrrrrrrrrrrrrrrrrrrr", PerfilActivoRef: "prf_pppppppppppppppppppppp",
		RegistroContextoRef: "rca_" + strings.Repeat("2", 32),
		ContextoSHA256:      "18e12e87244ad1d33bbd2ab1d6344bae8a7c6819723d51a3da501d7560cf4798",
		ProcedenciaSHA256:   strings.Repeat("3", 64), AutenticacionRef: "aut_" + strings.Repeat("4", 32),
		SesionRef: "ses_" + strings.Repeat("5", 32), AutenticacionSHA256: strings.Repeat("6", 64),
		Accion: "administracion.perfiles.consultar", ModuloID: "usuarios", RecursoRef: "perfil:xxxx",
		FinalidadRef: "administracion", Resultado: "denegado", MotivoRef: "sin_permiso",
		Proceso: "vec-admin", Canal: "administracion_privilegiada", CorrelacionRef: "corr:1",
		VinculoSHA256: strings.Repeat("7", 64), ContextoCanonicoBase64: base64.StdEncoding.EncodeToString([]byte(contextoMixtoV2)),
	}
	c2 := "bb0b89e4c541aa5bc318d992f1bfdf06df7cee6e608a4e216090212f5c666fda"
	c := RegistroCadenaV3{AuditoriaRef: "aud_v3_" + c2[:32], Secuencia: 2,
		DecisionRef: "decisión:2", EfectoRef: "efecto:2", HuellaEfectoSHA256: strings.Repeat("a", 64),
		AnteriorSHA256: eslabonIntentoSHA, HuellaSHA256: cabezaMixtaSHA, ConsumoHuellaSHA256: c2}
	return DocumentoVerificacionMixta{Esquema: EsquemaVerificacionMixta,
		Manifiesto: CoberturaCadena{CadenaID: "cadena:sintetica:principal", PrimeraSecuencia: 1,
			UltimaSecuencia: 2, AnteriorSHA256: strings.Repeat("0", 64), CabezaSHA256: cabezaMixtaSHA, Registros: 2},
		Registros: []RegistroMixtoV2{{TipoRegistro: "intento_nominal", Intento: &a},
			{TipoRegistro: "consumo_confirmado", Consumo: &c}}}
}

// Contraste independiente con encuadrar_mac(text) de PostgreSQL 18.4 en el
// clon frío: acta SHA256 26e3e06b8ecefc2e2ad294fe4590ac08db7c17282932f1ff5185eb4c131dd227.
func TestFramingAD169CoincideConVectoresPostgreSQL(t *testing.T) {
	for _, caso := range []struct{ fecha, eslabon string }{
		{"0001-01-01T00:00:00.000000Z", "dc77fbac6986849a11edab8688fd7b494a5ee8b0fe95a746064520596573de64"},
		{"9999-12-31T23:59:59.999999Z", "5f9935d657ba2451d16b4d0c15b68c9d1ee251a477aaac48d970b82d8c0e3e7a"},
	} {
		a := *vectorMixtoV2().Registros[0].Intento
		a.RegistradaEn, a.HuellaSHA256 = caso.fecha, caso.eslabon
		if codigo, clave, _, _ := cotejarIntentoV2(a); codigo != "" {
			t.Fatalf("vector PostgreSQL %s rechazado: %s/%s", caso.fecha, codigo, clave)
		}
	}
	// Este texto prueba la longitud en bytes y escapes del encuadre; no es un
	// campo admisible en una orden AD169.
	encuadre := encuadrarVerificacion("ñ: \"\\\"\n😃")
	if hex.EncodeToString(encuadre) != "31323ac3b13a20225c220af09f98830a" {
		t.Fatal("longitud UTF-8 de PostgreSQL divergente")
	}
	suma := sha256.Sum256(encuadre)
	if hex.EncodeToString(suma[:]) != "342b5d012ffedfd16554de5b2bd0388ae125e18166ba3c52aee63eb57a5454a7" {
		t.Fatal("huella de encuadre PostgreSQL divergente")
	}
}

func TestCadenaMixtaVectorYAlcance(t *testing.T) {
	d := vectorMixtoV2()
	r := VerificarCadenaMixtaV2(d, d.Manifiesto, 2)
	if r.Estado != "verificada" || !r.CheckpointCotejado || !r.MaterialIntentoRecalculado ||
		!r.ActorPerfilContextoCotejados || r.ContenidoConsumoRecalculado || r.CamposFueraHuellaVerificados ||
		r.AutenticidadCheckpoint != "no_comprobada" || r.AutenticidadFuentesHistoricas != "no_comprobada" {
		t.Fatalf("alcance mixto incorrecto: %+v", r)
	}
}

func TestCadenaMixtaDetectaAlteraciones(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*DocumentoVerificacionMixta)
		code   string
	}{
		{"accion", func(d *DocumentoVerificacionMixta) { d.Registros[0].Intento.Accion = "otra.accion" }, "material_distinto"},
		{"resultado", func(d *DocumentoVerificacionMixta) { d.Registros[0].Intento.Resultado = "error" }, "material_distinto"},
		{"contexto_hash", func(d *DocumentoVerificacionMixta) { d.Registros[0].Intento.ContextoSHA256 = strings.Repeat("8", 64) }, "contexto_distinto"},
		{"contexto_bytes", func(d *DocumentoVerificacionMixta) {
			d.Registros[0].Intento.ContextoCanonicoBase64 = base64.StdEncoding.EncodeToString([]byte("{}"))
		}, "contexto_distinto"},
		{"actor", func(d *DocumentoVerificacionMixta) {
			d.Registros[0].Intento.ActorRef = "per_" + strings.Repeat("x", 22)
		}, "actor_perfil_distinto"},
		{"perfil", func(d *DocumentoVerificacionMixta) {
			d.Registros[0].Intento.PerfilActivoRef = "prf_" + strings.Repeat("x", 22)
		}, "actor_perfil_distinto"},
		{"procedencia", func(d *DocumentoVerificacionMixta) {
			d.Registros[0].Intento.ProcedenciaSHA256 = strings.Repeat("9", 64)
		}, "material_distinto"},
		{"fecha", func(d *DocumentoVerificacionMixta) {
			d.Registros[0].Intento.RegistradaEn = "2026-10-03T10:11:12.123457Z"
		}, "huella_distinta"},
		{"enlace", func(d *DocumentoVerificacionMixta) { d.Registros[1].Consumo.AnteriorSHA256 = strings.Repeat("b", 64) }, "enlace_distinto"},
		{"hueco", func(d *DocumentoVerificacionMixta) { d.Registros[1].Consumo.Secuencia = 3 }, "secuencia_distinta"},
		{"tipo", func(d *DocumentoVerificacionMixta) { d.Registros[0].TipoRegistro = "consumo_confirmado" }, "tipo_invalido"},
		{"truncado", func(d *DocumentoVerificacionMixta) { d.Registros = d.Registros[:1] }, "cantidad_distinta"},
		{"checkpoint", func(d *DocumentoVerificacionMixta) { d.Manifiesto.CabezaSHA256 = strings.Repeat("c", 64) }, "checkpoint_distinto"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := vectorMixtoV2()
			cp := d.Manifiesto
			tc.mutate(&d)
			r := VerificarCadenaMixtaV2(d, cp, 2)
			if r.Estado != "rechazada" || r.Fallo == nil || r.Fallo.Codigo != tc.code {
				t.Fatalf("%s: %+v", tc.name, r)
			}
		})
	}
}

func TestCadenaMixtaRechazaDuplicadosYLimites(t *testing.T) {
	d := vectorMixtoV2()
	copia := *d.Registros[0].Intento
	copia.Secuencia = 3
	copia.AnteriorSHA256 = cabezaMixtaSHA
	d.Registros = append(d.Registros, RegistroMixtoV2{TipoRegistro: "intento_nominal", Intento: &copia})
	d.Manifiesto.UltimaSecuencia = 3
	d.Manifiesto.Registros = 3
	// El duplicado se detecta antes de aceptar una repetición de intento.
	if r := VerificarCadenaMixtaV2(d, d.Manifiesto, 3); r.Fallo == nil || r.Fallo.Codigo != "intento_duplicado" {
		t.Fatalf("duplicado: %+v", r)
	}
	d = vectorMixtoV2()
	if r := VerificarCadenaMixtaV2(d, d.Manifiesto, 1); r.Fallo == nil || r.Fallo.Codigo != "limite_registros" {
		t.Fatalf("limite: %+v", r)
	}
}

func TestMaterialIntentoComprometeOrdenNominalCompleto(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		mutar  func(*RegistroIntentoV2)
		codigo string
	}{
		{"intento_ref", func(a *RegistroIntentoV2) { a.IntentoRef = "intento_" + strings.Repeat("2", 32) }, "referencia_distinta"},
		{"registro_contexto_ref", func(a *RegistroIntentoV2) { a.RegistroContextoRef = "rca_" + strings.Repeat("9", 32) }, "material_distinto"},
		{"autenticacion_ref", func(a *RegistroIntentoV2) { a.AutenticacionRef = "aut_" + strings.Repeat("9", 32) }, "material_distinto"},
		{"sesion_ref", func(a *RegistroIntentoV2) { a.SesionRef = "ses_" + strings.Repeat("9", 32) }, "material_distinto"},
		{"autenticacion_sha256", func(a *RegistroIntentoV2) { a.AutenticacionSHA256 = strings.Repeat("9", 64) }, "material_distinto"},
		{"modulo_id", func(a *RegistroIntentoV2) { a.ModuloID = "personal" }, "material_distinto"},
		{"recurso_ref", func(a *RegistroIntentoV2) { a.RecursoRef = "perfil:otro" }, "material_distinto"},
		{"finalidad_ref", func(a *RegistroIntentoV2) { a.FinalidadRef = "consulta" }, "material_distinto"},
		{"motivo_ref", func(a *RegistroIntentoV2) { a.MotivoRef = "sin_contexto" }, "material_distinto"},
		{"proceso", func(a *RegistroIntentoV2) { a.Proceso = "vec-interno" }, "material_distinto"},
		{"canal", func(a *RegistroIntentoV2) { a.Canal = "interna_corporativa" }, "material_distinto"},
		{"correlacion_ref", func(a *RegistroIntentoV2) { a.CorrelacionRef = "corr:2" }, "material_distinto"},
		{"vinculo_sha256", func(a *RegistroIntentoV2) { a.VinculoSHA256 = strings.Repeat("8", 64) }, "material_distinto"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			d := vectorMixtoV2()
			caso.mutar(d.Registros[0].Intento)
			r := VerificarCadenaMixtaV2(d, d.Manifiesto, 2)
			if r.Fallo == nil || r.Fallo.Codigo != caso.codigo {
				t.Fatalf("campo %s no comprometido: %+v", caso.nombre, r)
			}
		})
	}
}
