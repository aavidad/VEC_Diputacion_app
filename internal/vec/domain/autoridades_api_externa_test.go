package domain_test

import (
	"strings"
	"testing"
	"time"

	domain "vec-diputacion-granada/internal/vec/domain"
)

func TestAPIPublicaFuenteAutoridadCompletaFirmaAsincrona(t *testing.T) {
	creadaEn := time.Date(2026, time.July, 17, 9, 0, 0, 0, time.UTC)
	fuente, err := domain.NuevaFuenteAutoridadBorradorV1(domain.DatosAltaFuenteAutoridadV1{
		ID: "rpt_historica_api_externa",
		Contenido: domain.ContenidoFuenteAutoridad{
			MateriaClave: "plantilla_rpt", Nombre: "Relación de puestos de trabajo histórica",
			Ambitos: []domain.AmbitoFuenteAutoridad{
				{DimensionClave: "entidad", ValoresClave: []string{"diputacion_granada"}},
			},
			Documento: domain.DocumentoFuenteAutoridad{
				DocumentoID: "doc:rpt:2020", DocumentoVersion: 1, RepresentacionRef: "rep:pdfa:rpt:2020",
				HuellaContenidoSHA256: strings.Repeat("a", 64), PublicacionOficialRef: "bop:granada:2020:10",
				ActoOrigenRef: "acto:pleno:rpt:2020", OrganoEmisorRef: "organo:diputacion:pleno",
			},
			Preceptos:  []domain.PreceptoFuenteAutoridad{{Clave: "anexo_rpt", Cita: "Anexo RPT"}},
			Vigencia:   domain.PeriodoFuenteAutoridad{Desde: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)},
			Efectos:    domain.PeriodoFuenteAutoridad{Desde: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)},
			ConocidaEn: creadaEn.Add(-time.Hour),
		},
		CreadaPor: "per_creador_api_externa_0000001", CreadaEn: creadaEn,
		MotivoCreacionCodigo: "incorporacion_historica",
	})
	if err != nil {
		t.Fatal(err)
	}
	preparadaEn := creadaEn.Add(time.Hour)
	solicitud, err := fuente.PrepararSolicitudTransicionV1(domain.DatosPreparacionTransicionFuenteAutoridadV1{
		EstadoNuevo: domain.EstadoFuenteAutoridadPublicada,
		ActorRef:    "per_publicador_api_externa_00001", MotivoCodigo: "validacion_incorporacion",
		SolicitudRef: "solicitud:api:externa:1", PreparadaEn: preparadaEn, ExpiraEn: preparadaEn.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	bytesSolicitud, err := solicitud.BytesCanonicos()
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err = domain.RehidratarSolicitudTransicionFuenteAutoridadV1(bytesSolicitud)
	if err != nil {
		t.Fatalf("reanudar solicitud: %v", err)
	}
	comprobadaEn := preparadaEn.Add(time.Hour)
	mensaje, err := domain.PrepararMensajeAtestacionActoFuenteAutoridadV1(
		solicitud, domain.DatosMensajeAtestacionActoFuenteAutoridadV1{
			EvidenciaRef: "evidencia:api:externa:1", ActoRef: "acto:pleno:rpt:2020",
			DocumentoRef: "doc:rpt:2020", RepresentacionRef: "rep:pdfa:rpt:2020",
			HuellaDocumentoSHA256: strings.Repeat("b", 64), OrganoRef: "organo:diputacion:pleno",
			FirmasRefs: []string{"firma:secretaria:rpt:2020"}, ComprobadorRef: "conector:afirma:1",
			ActoOcurridoEn: time.Date(2020, 1, 15, 12, 0, 0, 0, time.UTC), ComprobadaEn: comprobadaEn,
		})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mensaje.BytesCanonicos(); err != nil {
		t.Fatal(err)
	}
	evidencia, err := mensaje.ConstituirEvidenciaAtestadaV1(domain.DatosSobreAtestacionActoFuenteAutoridadV1{
		AtestacionRef: "atestacion:api:externa:1", HuellaAtestacionSHA256: strings.Repeat("c", 64),
		FirmaAtestacionRef: "firma:atestacion:api:externa:1",
	})
	if err != nil {
		t.Fatal(err)
	}
	publicada, err := fuente.AplicarTransicionV1(solicitud, evidencia, comprobadaEn.Add(time.Minute))
	if err != nil || publicada.Estado != domain.EstadoFuenteAutoridadPublicada {
		t.Fatalf("aplicar callback asíncrono: estado=%s, error=%v", publicada.Estado, err)
	}
}
