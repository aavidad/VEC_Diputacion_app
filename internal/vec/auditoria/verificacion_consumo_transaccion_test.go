package auditoria

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func documentoAD193Prueba(t *testing.T) DocumentoVerificacionMixta {
	t.Helper()
	b, err := os.ReadFile("../../../cmd/vec-auditoria-verificar/testdata/consumo_transaccion_ad193.json")
	if err != nil {
		t.Fatal(err)
	}
	var d DocumentoVerificacionMixta
	if json.Unmarshal(b, &d) != nil {
		t.Fatal("vector AD193 inválido")
	}
	return d
}

func TestAD193VectoresIndependientesYFamiliasHistoricas(t *testing.T) {
	d := documentoAD193Prueba(t)
	for i, xid := range []string{"9007199254740993", "18446744073709551615"} {
		r := d.Registros[3+i].ConsumoTransaccion
		if r == nil || r.TransaccionOrigen != xid || r.TransaccionConsumoOrigen != xid || CotejarConsumoTransaccionV4(*r) != nil {
			t.Fatal("sello uint64 o preimagen de dieciséis campos divergentes")
		}
	}
	antes, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, esquema := range []string{EsquemaVerificacionMixta, EsquemaVerificacionPreperfil,
		EsquemaVerificacionFuentesIniciales, EsquemaVerificacionUnidadInicial, EsquemaVerificacionBootstrapCentral,
		EsquemaVerificacionMantenimientoFijo, EsquemaVerificacionPeriodica, EsquemaVerificacionPreservacionAuditoria,
		EsquemaVerificacionGobiernoUsuarios, EsquemaVerificacionFronteraAdminTecnicaV1} {
		d.Esquema = esquema
		b, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		_, informe, err := VerificarDocumentoExportacionAuditoria(b, domain.CoberturaCheckpoint(d.Manifiesto), 1<<20, 5)
		if err != nil || informe.Estado != "verificada" || !informe.FechaConsumoLigadaCotejada || !informe.ConsumosHistoricosSinFechaLigada || informe.ActorPerfilContextoCotejados {
			t.Fatalf("esquema=%s informe=%+v err=%v", esquema, informe, err)
		}
	}
	d.Esquema = EsquemaVerificacionPreperfil
	despues, err := json.Marshal(d)
	if err != nil || !bytes.Equal(antes, despues) {
		t.Fatal("se reescribió historia al verificar")
	}
	// El formato v1 sin discriminadores no adquiere una familia nueva.
	d.Esquema = EsquemaVerificacion
	b, _ := json.Marshal(d)
	if _, informe, err := VerificarDocumentoExportacionAuditoria(b, domain.CoberturaCheckpoint(d.Manifiesto), 1<<20, 5); err == nil || informe.Estado != "rechazada" {
		t.Fatal("v1 admitió consumo mixto")
	}
}

func TestAD193SellosInvalidosYMutaciones(t *testing.T) {
	for _, valor := range []string{"", "0", "00", "01", "+1", "-1", " 1", "1 ", "1e3", "1.0", "１", "18446744073709551616"} {
		for _, segundo := range []bool{false, true} {
			r := *documentoAD193Prueba(t).Registros[3].ConsumoTransaccion
			clave := "transaccion_origen"
			if segundo {
				r.TransaccionConsumoOrigen, clave = valor, "transaccion_consumo_origen"
			} else {
				r.TransaccionOrigen = valor
			}
			if f := CotejarConsumoTransaccionV4(r); f == nil || f.Codigo != "transaccion_invalida" || f.Clave != clave {
				t.Fatalf("sello no canónico admitido: clave=%s", clave)
			}
		}
	}
	for _, caso := range []struct {
		nombre string
		mutar  func(*RegistroConsumoTransaccionV4)
		codigo string
	}{
		{"sello_consumo", func(r *RegistroConsumoTransaccionV4) { r.TransaccionConsumoOrigen = "9007199254740994" }, "transaccion_distinta"},
		{"ambos_sellos", func(r *RegistroConsumoTransaccionV4) {
			r.TransaccionOrigen, r.TransaccionConsumoOrigen = "9007199254740994", "9007199254740994"
		}, "huella_distinta"},
		{"fecha", func(r *RegistroConsumoTransaccionV4) { r.RegistradaEn = "2026-10-03T12:34:56.123457Z" }, "instante_distinto"},
		{"dos_fechas", func(r *RegistroConsumoTransaccionV4) {
			r.RegistradaEn, r.ConsumidaEn = "2026-10-03T12:34:56.123457Z", "2026-10-03T12:34:56.123457Z"
		}, "huella_distinta"},
		{"version", func(r *RegistroConsumoTransaccionV4) { r.VersionConsumo = 3 }, "tipo_invalido"},
		{"tipo", func(r *RegistroConsumoTransaccionV4) { r.TipoRegistro = TipoConsumoFechaV3 }, "tipo_invalido"},
		{"huella_v3", func(r *RegistroConsumoTransaccionV4) { r.HuellaSHA256 = huellaConsumoFechaV3(r.RegistroConsumoFechaV3) }, "huella_distinta"},
		{"proceso", func(r *RegistroConsumoTransaccionV4) { r.Proceso = "otro_proceso" }, "huella_distinta"},
		{"actor", func(r *RegistroConsumoTransaccionV4) { r.ActorRef = "actor:otro" }, "huella_distinta"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			r := *documentoAD193Prueba(t).Registros[3].ConsumoTransaccion
			caso.mutar(&r)
			if f := CotejarConsumoTransaccionV4(r); f == nil || f.Codigo != caso.codigo {
				t.Fatalf("mutación no rechazada: %+v", f)
			}
		})
	}
}

func TestAD193ExclusividadYUnicidadEnCadena(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		mutar  func(*DocumentoVerificacionMixta)
	}{
		{"v3", func(d *DocumentoVerificacionMixta) { d.Registros[3].ConsumoFecha = d.Registros[2].ConsumoFecha }},
		{"v2", func(d *DocumentoVerificacionMixta) { d.Registros[3].ConsumoOrigen = d.Registros[1].ConsumoOrigen }},
		{"v1", func(d *DocumentoVerificacionMixta) { d.Registros[3].Consumo = d.Registros[0].Consumo }},
		{"intento", func(d *DocumentoVerificacionMixta) { d.Registros[3].Intento = &RegistroIntentoV2{} }},
		{"bootstrap", func(d *DocumentoVerificacionMixta) { d.Registros[3].Bootstrap = &RegistroBootstrapV3{} }},
		{"tecnico", func(d *DocumentoVerificacionMixta) {
			d.Registros[3].FronteraAdminTecnica = &RegistroFronteraAdminTecnicaV1{}
		}},
		{"discriminador", func(d *DocumentoVerificacionMixta) { d.Registros[3].TipoRegistro = TipoConsumoFechaV3 }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			d := documentoAD193Prueba(t)
			caso.mutar(&d)
			if f := VerificarCadenaMixtaV3(d, d.Manifiesto, 5); f.Estado != "rechazada" || f.Fallo == nil || f.Fallo.Codigo != "tipo_invalido" {
				t.Fatalf("familia cruzada admitida: %+v", f)
			}
		})
	}
	d := documentoAD193Prueba(t)
	d.Registros[3].ConsumoFecha = d.Registros[2].ConsumoFecha
	if _, err := json.Marshal(d); err == nil {
		t.Fatal("Marshal admitió dos proyecciones consumo")
	}
	// Una decisión ya usada en historia no se vuelve única por añadir un XID.
	d = documentoAD193Prueba(t)
	r := d.Registros[3].ConsumoTransaccion
	r.DecisionRef = d.Registros[0].Consumo.DecisionRef
	r.HuellaSHA256 = huellaConsumoTransaccionV4(*r)
	if f := VerificarCadenaMixtaV3(d, d.Manifiesto, 5); f.Fallo == nil || f.Fallo.Codigo != "consumo_duplicado" {
		t.Fatalf("decisión duplicada admitida: %+v", f)
	}
}
