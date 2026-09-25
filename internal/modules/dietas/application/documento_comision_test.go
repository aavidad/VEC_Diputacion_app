package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	docdomain "vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type lectorDocumentoPrueba struct {
	resultado dietasports.ResultadoBorradorComision
	llamadas  int
}

func (l *lectorDocumentoPrueba) ObtenerPropio(_ context.Context, _ dietasports.IdentidadEfectivaBorrador, _ string) (dietasports.ResultadoBorradorComision, error) {
	l.llamadas++
	return l.resultado, nil
}

type renderizadorDocumentoPrueba struct{ llamadas int }

func (r *renderizadorDocumentoPrueba) Renderizar(_ context.Context, _ dietasports.InstantaneaDocumentoComision) ([]byte, error) {
	r.llamadas++
	return []byte("%PDF-1.7\ncontenido"), nil
}

type altaDocumentoPrueba struct {
	solicitud docports.AltaGenerado
	llamadas  int
}

var errAltaDocumentoPrueba = errors.New("alta detenida por prueba")

func (a *altaDocumentoPrueba) AltaGenerado(_ context.Context, solicitud docports.AltaGenerado) (docdomain.Documento, error) {
	a.llamadas++
	a.solicitud = solicitud
	return docdomain.Documento{}, errAltaDocumentoPrueba
}

func ordenDocumentoPrueba(t *testing.T) (OrdenDocumentoComision, dietasports.ResultadoBorradorComision) {
	t.Helper()
	ref := "dco_" + strings.Repeat("a", 22)
	agrupacion, err := ReferenciaAgrupacionDocumentoComision(ref, 2)
	if err != nil {
		t.Fatal(err)
	}
	refs := []string{
		"ref:" + strings.Repeat("1", 64), "ref:" + strings.Repeat("2", 64),
		"ref:" + strings.Repeat("3", 64), agrupacion,
		"ref:" + strings.Repeat("4", 64), "ref:" + strings.Repeat("5", 64),
	}
	politica, err := vecports.NuevaSolicitudPoliticaConservacionDocumental(
		refs[0], refs[1], refs[2], refs[3], refs[4], 1,
		[]byte(strings.Repeat("h", 32)), refs[5],
		time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 9, 21, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado := dietasports.ResultadoBorradorComision{
		Comision: domain.ComisionBorrador{
			Referencia: ref, Estado: "borrador", FechaInicio: "2026-09-22", FechaFin: "2026-09-23",
			Motivo: "Visita técnica", CodigosRuta: []string{"granada", "motril"},
			RelacionRef: "rel_" + strings.Repeat("b", 22),
		},
		Recibo: dietasports.ReciboBorradorComision{
			Referencia: "rcd_" + strings.Repeat("c", 36), Version: 2,
			RegistradoEn: time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC),
		},
	}
	return OrdenDocumentoComision{
		ReferenciaComision: ref, VersionEsperada: 2,
		DocumentoID: "documento:comision:1", ClaveIdempotencia: "clave:documento:1",
		TipoDocumentalRef: refs[2], SolicitudPolitica: politica,
		Autorizacion: docports.AutorizacionV3{Accion: docports.AccionAlta},
	}, resultado
}

func TestDocumentoComisionPasaInstantaneaDurableAlPuertoComun(t *testing.T) {
	orden, resultado := ordenDocumentoPrueba(t)
	lector := &lectorDocumentoPrueba{resultado: resultado}
	renderizador := &renderizadorDocumentoPrueba{}
	alta := &altaDocumentoPrueba{}
	servicio, err := NuevoServicioDocumentoComision(lector, renderizador, alta)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Registrar(context.Background(), orden)
	if !errors.Is(err, errAltaDocumentoPrueba) || lector.llamadas != 1 || renderizador.llamadas != 1 || alta.llamadas != 1 {
		t.Fatalf("camino al puerto comun: %v, llamadas %d/%d/%d", err, lector.llamadas, renderizador.llamadas, alta.llamadas)
	}
	agrupacion, _ := ReferenciaAgrupacionDocumentoComision(orden.ReferenciaComision, orden.VersionEsperada)
	if alta.solicitud.ExpedienteRef != agrupacion || alta.solicitud.TipoRef != orden.TipoDocumentalRef ||
		alta.solicitud.ModuloID != "dietas" || alta.solicitud.Version != 2 ||
		alta.solicitud.MIME != "application/pdf" || string(alta.solicitud.Contenido) != "%PDF-1.7\ncontenido" {
		t.Fatalf("solicitud comun discordante: %+v", alta.solicitud)
	}
}

func TestDocumentoComisionRechazaVersionDistintaAntesDeCustodia(t *testing.T) {
	orden, resultado := ordenDocumentoPrueba(t)
	resultado.Recibo.Version = 1
	lector := &lectorDocumentoPrueba{resultado: resultado}
	renderizador := &renderizadorDocumentoPrueba{}
	alta := &altaDocumentoPrueba{}
	servicio, err := NuevoServicioDocumentoComision(lector, renderizador, alta)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Registrar(context.Background(), orden)
	if !errors.Is(err, dietasports.ErrInstantaneaDocumentoComisionInvalida) ||
		lector.llamadas != 1 || renderizador.llamadas != 0 || alta.llamadas != 0 {
		t.Fatalf("version discordante: %v, llamadas %d/%d/%d", err, lector.llamadas, renderizador.llamadas, alta.llamadas)
	}
}
