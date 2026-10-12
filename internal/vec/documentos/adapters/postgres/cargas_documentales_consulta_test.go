package postgres

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestConsultaCargaPropiaValidaManifiestoConInstantaneaPreparada(t *testing.T) {
	t0 := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	reservada, err := vecdomain.NuevaCargaDocumental(
		"ref:"+strings.Repeat("1", 64), "per_"+strings.Repeat("a", 32), "ref:"+strings.Repeat("2", 64),
		"bolsa", "solicitud", "operacion:carga:0123456789abcdef", "correlacion:0123456789abcdef",
		"aportar_documentacion_bolsa", "datos_personales", "application/pdf", 1024,
		strings.Repeat("a", 64), "hmac-sha256:indice_v1:"+strings.Repeat("c", 64),
		"hmac-sha256:solicitud_v1:"+strings.Repeat("b", 64), t0, t0.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	preparada, err := reservada.Preparar("hmac-sha256:sesion_v1:"+strings.Repeat("d", 64),
		"decision:preparar:0123456789abcdef", t0.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	manifiesto, err := vecdomain.NuevoManifiestoPreparacionCargaDirectaV1(preparada,
		vecdomain.ContextoManifiestoPreparacionCargaDirectaV1{
			CargaRef: preparada.IndiceIdempotenciaHMAC, SujetoSeudonimoHMAC: "hmac-sha256:seudonimo_v1:" + strings.Repeat("d", 64),
			HuellaRecursoBaseSHA256: strings.Repeat("2", 64), HuellaRecursoSHA256: strings.Repeat("e", 64),
			ConectorAlmacenID: "almacen_s3_corporativo", EsquemaContexto: "vec.almacen.contexto-operacion.v1",
			AccionNegocio: "vec.documentos.carga.preparar", AccionTecnica: "preparar_carga_directa",
			PasoRef: "01_preparar_carga_directa", EfectoRef: "efecto:carga:preparar:0123456789abcdef",
			HuellaPlanEfectoSHA256: strings.Repeat("f", 64),
			EsquemaHuellaDecision:  "vec.autorizacion.decision.reforzada.v1.autenticacion-actor",
			DecisionRef:            preparada.AutorizacionPreparacionRef, HuellaDecisionSHA256: strings.Repeat("1", 64),
			ContextoVerificadoEn: t0.Add(500 * time.Millisecond), DecisionValidaHasta: preparada.ExpiraEn.Add(time.Minute),
		})
	if err != nil {
		t.Fatal(err)
	}
	datos, err := manifiesto.Datos()
	if err != nil {
		t.Fatal(err)
	}
	contenido := vecdomain.ContenidoCargaDocumental{ConectorID: "almacen:corporativo", Referencia: "objeto:cuarentena:uno",
		Version: "version:1", Zona: vecdomain.ZonaContenidoCargaCuarentena, MIME: preparada.MIMEDeclarado,
		Tamano: preparada.TamanoDeclarado, HuellaSHA256: preparada.HuellaDeclaradaSHA256,
		EvidenciaRef: "evidencia:almacen:uno", RegistradoEn: t0.Add(2 * time.Second)}
	cuarentena, err := preparada.RegistrarCuarentena(contenido, "decision:confirmar:uno", contenido.RegistradoEn)
	if err != nil {
		t.Fatal(err)
	}
	analisis := vecdomain.AnalisisCargaDocumental{ObjetoReferencia: contenido.Referencia, ObjetoVersion: contenido.Version,
		HuellaObjetoSHA256: contenido.HuellaSHA256, ConectorAnalizadorID: "analizador:icap:corporativo",
		VersionConector: 1, Estado: vecdomain.EstadoAnalisisCargaLimpio, CodigoResultado: "limpio",
		EvidenciaRef: "evidencia:analisis:uno", HuellaEvidenciaSHA256: strings.Repeat("d", 64),
		CompletadoEn: t0.Add(3 * time.Second)}
	limpia, err := cuarentena.RegistrarAnalisis(analisis, "decision:analizar:uno", analisis.CompletadoEn)
	if err != nil {
		t.Fatal(err)
	}
	admitido := contenido
	admitido.Referencia, admitido.Zona = "objeto:admitido:uno", vecdomain.ZonaContenidoCargaAdmitida
	admitido.EvidenciaRef, admitido.RegistradoEn = "evidencia:promocion:uno", t0.Add(4*time.Second)
	final, err := limpia.Admitir(admitido, "decision:promover:uno", admitido.RegistradoEn)
	if err != nil {
		t.Fatal(err)
	}

	para := func(actual vecdomain.CargaDocumental, snapshot vecdomain.CargaDocumental, d vecdomain.DatosManifiestoPreparacionCargaDirectaV1) []byte {
		t.Helper()
		raw, err := json.Marshal(map[string]any{"carga": actual, "carga_preparada": snapshot,
			"manifiesto": d, "estado_operativo": string(vecports.EstadoCargaOperacionReciboRegistrado)})
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	for _, actual := range []vecdomain.CargaDocumental{preparada, cuarentena, final} {
		consulta, err := decodificarConsultaCargaPropiaV3(para(actual, preparada, datos), actual.ID,
			actual.RecursoRef, actual.PrincipalID, true)
		if err != nil || consulta.Manifiesto == nil || consulta.Carga.Version != actual.Version {
			t.Fatalf("consulta v%d: %#v, %v", actual.Version, consulta, err)
		}
		if _, err := decodificarConsultaCargaPropiaV3(para(actual, preparada, datos), actual.ID,
			"ref:"+strings.Repeat("9", 64), actual.PrincipalID, true); err == nil {
			t.Fatal("se expuso una carga bajo recurso ajeno")
		}
	}
	ajena := preparada
	ajena.ID = "ref:" + strings.Repeat("9", 64)
	if _, err := decodificarConsultaCargaPropiaV3(para(final, ajena, datos), final.ID,
		final.RecursoRef, final.PrincipalID, true); err == nil {
		t.Fatal("se acepto instantanea de otra carga")
	}
	alterado := datos
	alterado.MIME = "application/zip"
	if _, err := decodificarConsultaCargaPropiaV3(para(final, preparada, alterado), final.ID,
		final.RecursoRef, final.PrincipalID, true); err == nil {
		t.Fatal("se acepto manifiesto alterado")
	}
}
