package auditoria

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

// El vector procede del ensayo PostgreSQL 18 post-AD183 del director. Conserva
// configuración, captura y confirmación reales en la cadena técnica sintética.
func TestAD186VectorPostgreSQLRealConWrapBase64(t *testing.T) {
	raw, err := os.ReadFile("testdata/periodica_ad186.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if err := decodificarDocumentoExportacionEstricto(raw, &d); err != nil {
		t.Fatal(err)
	}
	if d.Manifiesto.PrimeraSecuencia != 6241 || d.Manifiesto.UltimaSecuencia != 6243 || d.Manifiesto.Registros != 3 {
		t.Fatal("vector PostgreSQL distinto")
	}
	for _, r := range d.Registros {
		if r.Periodica == nil || !strings.Contains(r.Periodica.DetalleCanonicoBase64, "\n") {
			t.Fatal("sin transporte PostgreSQL natural")
		}
	}
	r := VerificarCadenaPeriodicaV1(d, d.Manifiesto, 3)
	if r.Estado != "verificada" || !r.MaterialPeriodicaRecalculado || r.ActorPerfilContextoCotejados || r.AutenticidadCheckpoint != "no_comprobada" {
		t.Fatalf("vector PostgreSQL: %+v fallo=%+v", r, r.Fallo)
	}
	c := d.Manifiesto
	_, informe, err := VerificarDocumentoExportacionAuditoria(raw, domain.CoberturaCheckpoint{
		CadenaID: c.CadenaID, PrimeraSecuencia: c.PrimeraSecuencia, UltimaSecuencia: c.UltimaSecuencia,
		Registros: c.Registros, AnteriorSHA256: c.AnteriorSHA256, CabezaSHA256: c.CabezaSHA256}, 8192, 3)
	if err != nil || informe.Estado != "verificada" {
		t.Fatalf("parser PostgreSQL: %v %+v", err, informe)
	}
}

// Esta fixture prueba alteraciones locales, no la interoperabilidad SQL.
// El vector producido por PostgreSQL se coteja por separado.
func documentoPeriodicaPrueba() DocumentoVerificacionMixta {
	b := RegistroOperacionMantenimientoV1{AuditoriaRef: "aud_v3_per_" + strings.Repeat("1", 32), Secuencia: 1,
		AnteriorSHA256: strings.Repeat("0", 64), RegistradaEn: "2026-10-04T06:00:00.000000Z",
		EventoRef: "evento_" + strings.Repeat("1", 32), OperadorLogin: "sellador_sintetico",
		Accion: "capturar_sello_periodico_v1", ModuloID: "auditoria", RecursoRef: "auditoria_periodica:comun_interna",
		Resultado: "denegado", MotivoRef: "politica_inactiva", Proceso: "postgresql", Canal: "operacion_tecnica_privada",
		FinalidadRef: "integridad_auditoria_periodica", CorrelacionRef: "correlacion_" + strings.Repeat("2", 32)}
	detalle := `{"perfil_tecnico_ref": "vec_auditoria_periodica_sellador"}`
	m := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.periodica.material.v1", TipoOperacionPeriodica,
		b.EventoRef, b.OperadorLogin, b.Accion, b.ModuloID, b.RecursoRef, b.FinalidadRef, b.Resultado, b.MotivoRef,
		b.Proceso, b.Canal, b.CorrelacionRef, detalle))
	b.EventoMaterialSHA256 = hex.EncodeToString(m[:])
	h := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.periodica.eslabon.v1", strconv.FormatUint(b.Secuencia, 10),
		b.AnteriorSHA256, b.AuditoriaRef, b.EventoMaterialSHA256, b.RegistradaEn))
	b.HuellaSHA256 = hex.EncodeToString(h[:])
	return DocumentoVerificacionMixta{Esquema: EsquemaVerificacionPeriodica,
		Manifiesto: CoberturaCadena{CadenaID: "cadena:sintetica", PrimeraSecuencia: 1, UltimaSecuencia: 1, Registros: 1,
			AnteriorSHA256: b.AnteriorSHA256, CabezaSHA256: b.HuellaSHA256},
		Registros: []RegistroMixtoV2{{TipoRegistro: TipoOperacionPeriodica, Periodica: &RegistroOperacionPeriodicaV1{
			RegistroOperacionMantenimientoV1: b, DetalleCanonicoBase64: base64.StdEncoding.EncodeToString([]byte(detalle))}}}}
}

func TestAD186VerificaMaterialYEslabonEnParserComun(t *testing.T) {
	d := documentoPeriodicaPrueba()
	r := VerificarCadenaPeriodicaV1(d, d.Manifiesto, 2)
	if r.Estado != "verificada" || !r.MaterialPeriodicaRecalculado || r.ActorPerfilContextoCotejados || r.AutenticidadCheckpoint != "no_comprobada" {
		t.Fatalf("cotejo: %+v", r)
	}
	raw, _ := json.Marshal(d)
	c := d.Manifiesto
	_, informe, err := VerificarDocumentoExportacionAuditoria(raw, domain.CoberturaCheckpoint{
		CadenaID: c.CadenaID, PrimeraSecuencia: c.PrimeraSecuencia, UltimaSecuencia: c.UltimaSecuencia,
		Registros: c.Registros, AnteriorSHA256: c.AnteriorSHA256, CabezaSHA256: c.CabezaSHA256}, 4096, 2)
	if err != nil || informe.Estado != "verificada" {
		t.Fatalf("parser: %v %+v", err, informe)
	}
}

func TestAD186RechazaAlteracionesSinExponerMaterial(t *testing.T) {
	for nombre, alterar := range map[string]func(*DocumentoVerificacionMixta){
		"operador": func(d *DocumentoVerificacionMixta) { d.Registros[0].Periodica.OperadorLogin = "otro_login" },
		"fecha":    func(d *DocumentoVerificacionMixta) { d.Registros[0].Periodica.RegistradaEn = "dato_privado:no_fecha" },
		"tipo":     func(d *DocumentoVerificacionMixta) { d.Registros[0].TipoRegistro = "consumo_confirmado" },
		"cruce":    func(d *DocumentoVerificacionMixta) { d.Registros[0].Bootstrap = &RegistroBootstrapV3{} },
		"clave libre": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].Periodica.DetalleCanonicoBase64 = base64.StdEncoding.EncodeToString([]byte(`{"nota": "dato_privado"}`))
		},
		"perfil": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].Periodica.DetalleCanonicoBase64 = base64.StdEncoding.EncodeToString([]byte(`{"perfil_tecnico_ref": "vec_auditoria_periodica_configurador"}`))
		},
		"detalle no canonico": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].Periodica.DetalleCanonicoBase64 = base64.StdEncoding.EncodeToString([]byte(`{"perfil_tecnico_ref":"vec_auditoria_periodica_sellador"}`))
		},
		"base64 no canonico": func(d *DocumentoVerificacionMixta) { d.Registros[0].Periodica.DetalleCanonicoBase64 += "A" },
		"resultado":          func(d *DocumentoVerificacionMixta) { d.Registros[0].Periodica.Resultado = "permitido" },
		"secuencia":          func(d *DocumentoVerificacionMixta) { d.Registros[0].Periodica.Secuencia = 2 },
		"limite": func(d *DocumentoVerificacionMixta) {
			d.Registros[0].Periodica.DetalleCanonicoBase64 = strings.Repeat("A", 17000)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			d := documentoPeriodicaPrueba()
			alterar(&d)
			r := VerificarCadenaPeriodicaV1(d, d.Manifiesto, 2)
			b, _ := json.Marshal(r)
			if r.Estado != "rechazada" || strings.Contains(string(b), "dato_privado") || r.MaterialPeriodicaRecalculado {
				t.Fatalf("rechazo: %s", b)
			}
		})
	}
}

func TestAD186AdmiteWrapPGSinCambiarBytesAuditados(t *testing.T) {
	for _, salto := range []string{"\n", "\r\n"} {
		d := documentoPeriodicaPrueba()
		p := d.Registros[0].Periodica
		p.DetalleCanonicoBase64 = p.DetalleCanonicoBase64[:76] + salto + p.DetalleCanonicoBase64[76:]
		original := p.DetalleCanonicoBase64
		r := VerificarCadenaPeriodicaV1(d, d.Manifiesto, 2)
		if r.Estado != "verificada" || !r.MaterialPeriodicaRecalculado || p.DetalleCanonicoBase64 != original {
			t.Fatal("wrap PG alteró detalle o proyección")
		}
	}
}

func TestAD186DetallePGCanonicoYCoordenadasPrevias(t *testing.T) {
	b := documentoPeriodicaPrueba().Registros[0].Periodica.RegistroOperacionMantenimientoV1
	b.Secuencia = 2
	b.AnteriorSHA256 = strings.Repeat("a", 64)
	b.Resultado, b.MotivoRef = "permitido", "captura_registrada"
	raw := `{"captura_ref": "captura_` + strings.Repeat("3", 32) + `", "previa_secuencia": 1, "perfil_tecnico_ref": "vec_auditoria_periodica_sellador", "configuracion_sha256": "` + strings.Repeat("b", 64) + `", "previa_cabeza_sha256": "` + strings.Repeat("a", 64) + `"}`
	if !detallePeriodicaValido([]byte(raw), b) {
		t.Fatal("detalle PG orden longitud/léxico rechazado")
	}
	for _, mala := range []string{
		strings.Replace(raw, `"previa_secuencia": 1`, `"previa_secuencia": 2`, 1),
		strings.Replace(raw, `"previa_secuencia": 1`, `"previa_secuencia": 1.0`, 1),
		strings.Replace(raw, `"previa_secuencia": 1`, `"previa_secuencia": "1"`, 1),
		strings.Replace(raw, `"previa_secuencia": 1`, `"previa_secuencia": 1, "previa_secuencia": 1`, 1),
		strings.Replace(raw, `"previa_secuencia": 1`, `"previa_secuencia": null`, 1),
		strings.Replace(raw, strings.Repeat("a", 64), strings.Repeat("c", 64), 1),
	} {
		if detallePeriodicaValido([]byte(mala), b) {
			t.Fatal("detalle alterado aceptado")
		}
	}
}

func TestAD186ConservaFamiliasAnterioresYEsquemas(t *testing.T) {
	for _, d := range []DocumentoVerificacionMixta{vectorConsumosMixtosAD173(t), documentoFuentesPrueba(t), documentoUnidadInicialPrueba(t), documentoBootstrapIntentosPrueba(t), vectorMantenimientoFijo(t)} {
		rawAntes, _ := json.Marshal(d.Registros)
		d.Esquema = EsquemaVerificacionPeriodica
		r := VerificarCadenaPeriodicaV1(d, d.Manifiesto, uint64(len(d.Registros)))
		rawDespues, _ := json.Marshal(d.Registros)
		if r.Estado != "verificada" || r.MaterialPeriodicaRecalculado || string(rawAntes) != string(rawDespues) {
			t.Fatalf("familia histórica: %+v", r)
		}
		d.Registros[0].Periodica = documentoPeriodicaPrueba().Registros[0].Periodica
		if VerificarCadenaPeriodicaV1(d, d.Manifiesto, uint64(len(d.Registros))).Estado != "rechazada" {
			t.Fatal("familia cruzada aceptada")
		}
	}
	d := documentoPeriodicaPrueba()
	d.Esquema = EsquemaVerificacionMantenimientoFijo
	if VerificarCadenaMantenimientoFijoV1(d, d.Manifiesto, 2).Estado != "rechazada" {
		t.Fatal("esquema anterior admite familia nueva")
	}
}
