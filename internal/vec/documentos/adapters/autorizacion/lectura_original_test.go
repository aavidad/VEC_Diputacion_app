package autorizacion

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type relojFijo struct{ instante time.Time }

func (r relojFijo) Ahora() time.Time { return r.instante }

type resolutorContador struct{ llamadas int }

func (r *resolutorContador) SeudonimosLecturaOriginal(context.Context, docports.AutorizacionV3) (DatosSeudonimosLectura, error) {
	r.llamadas++
	return DatosSeudonimosLectura{}, nil
}

func (r *resolutorContador) ExigirLecturaOriginal(context.Context, SolicitudDecisionLecturaOriginal) (vecdomain.DecisionAutorizacion, error) {
	r.llamadas++
	return vecdomain.DecisionAutorizacion{}, nil
}

func TestFabricaDeniegaSinDependenciasYAntesDelPDP(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	if _, err := NuevaFabricaContextoLecturaOriginal(nil, relojFijo{ahora}); !errors.Is(err, vecports.ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("sin resolutor: %v", err)
	}
	var ausente *resolutorContador
	if _, err := NuevaFabricaContextoLecturaOriginal(ausente, relojFijo{ahora}); !errors.Is(err, vecports.ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("resolutor tipado nulo: %v", err)
	}
	resolutor := &resolutorContador{}
	fabrica, err := NuevaFabricaContextoLecturaOriginal(resolutor, relojFijo{ahora})
	if err != nil {
		t.Fatal(err)
	}
	d := documentoValidoPrueba(ahora)
	a := docports.AutorizacionV3{RecursoRef: d.ID, AmbitoRef: d.ExpedienteRef}
	if _, err := fabrica.ContextoLecturaOriginal(context.Background(), d, a); !errors.Is(err, vecports.ErrAutorizacionAlmacenInvalida) || resolutor.llamadas != 0 {
		t.Fatalf("material V3 ausente llego al PDP: %v, llamadas=%d", err, resolutor.llamadas)
	}
	a.AmbitoRef = "ref:" + strings.Repeat("9", 64)
	if _, err := fabrica.ContextoLecturaOriginal(context.Background(), d, a); !errors.Is(err, vecports.ErrAutorizacionAlmacenInvalida) || resolutor.llamadas != 0 {
		t.Fatalf("ambito ajeno llego al PDP: %v, llamadas=%d", err, resolutor.llamadas)
	}
}

func TestRecursoLecturaConservaObjetoYVersionExactos(t *testing.T) {
	d := documentoValidoPrueba(time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	a := docports.AutorizacionV3{AmbitoRef: d.ExpedienteRef}
	v := vecports.VinculosOperacionAlmacen{OperacionRef: "operacion:1", CargaRef: d.ID,
		Clasificacion: d.Proteccion, EfectoRef: d.ID,
		SujetoSeudonimoHMAC: "hmac-sha256:sujeto_v1:" + strings.Repeat("a", 64),
		HuellaSolicitudHMAC: "hmac-sha256:solicitud_v1:" + strings.Repeat("b", 64),
		ObjetoVinculado:     vecports.ReferenciaObjetoAlmacen{Referencia: d.ObjetoRef, Version: d.ObjetoVersion},
	}
	r := recursoLecturaOriginal(d, a, v)
	if r.Validar() != nil || r.Referencia != d.ID || r.ModuloID != "documentos" ||
		r.Ambitos["expediente_ref"] != d.ExpedienteRef ||
		r.Atributos[vecports.AtributoAlmacenObjetoRef] != d.ObjetoRef ||
		r.Atributos[vecports.AtributoAlmacenObjetoVersion] != d.ObjetoVersion ||
		r.Atributos["documento_version"] != "3" ||
		r.Atributos["documento_huella_sha256"] != d.HuellaSHA256 {
		t.Fatal("recurso PDP no liga metadatos y objeto originales")
	}
	d.ObjetoVersion = "objeto:version:ajena"
	rAjeno := recursoLecturaOriginal(d, a, v)
	h, _ := r.HuellaContextoAutorizacionSHA256()
	hAjeno, _ := rAjeno.HuellaContextoAutorizacionSHA256()
	if h == hAjeno {
		t.Fatal("cambio de version de objeto no cambia recurso PDP")
	}
}

func documentoValidoPrueba(ahora time.Time) domain.Documento {
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	return domain.Documento{
		ID: ref("1"), NumeroVEC: "VEC-2026-1", ModuloID: "dietas", ExpedienteRef: ref("2"),
		TipoRef: ref("3"), Version: 3, MIME: "application/pdf", HuellaSHA256: strings.Repeat("a", 64),
		Tamano: 4, ObjetoRef: "objeto:original:1", ObjetoVersion: "objeto:version:1",
		PoliticaRef: ref("4"), VersionPolitica: 1, HuellaPoliticaSHA256: strings.Repeat("b", 64),
		ConservacionHasta: ahora.AddDate(10, 0, 0), Proteccion: "conservacion",
		EstadoFirma: domain.EstadoFirmaPendienteProveedor, CreadoEn: ahora, Custodia: domain.CustodiaVEC,
	}
}

// La denegación conserva su causa (M2a): sigue siendo
// ErrAutorizacionAlmacenInvalida para todo consumidor, pero errors.Is alcanza
// el fallo que la originó para el registro interno.
func TestFabricaDenegacionConservaLaCausa(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	resolutor := &resolutorContador{}
	fabrica, err := NuevaFabricaContextoLecturaOriginal(resolutor, relojFijo{ahora})
	if err != nil {
		t.Fatal(err)
	}
	d := documentoValidoPrueba(ahora)
	a := docports.AutorizacionV3{RecursoRef: d.ID, AmbitoRef: d.ExpedienteRef}
	_, err = fabrica.ContextoLecturaOriginal(context.Background(), d, a)
	if !errors.Is(err, vecports.ErrAutorizacionAlmacenInvalida) || !errors.Is(err, docports.ErrSolicitudInvalida) {
		t.Fatalf("la validación V3 fallida no queda como causa: %v", err)
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err = fabrica.ContextoLecturaOriginal(cancelado, d, a)
	if !errors.Is(err, vecports.ErrAutorizacionAlmacenInvalida) || !errors.Is(err, context.Canceled) {
		t.Fatalf("la cancelación no queda como causa: %v", err)
	}
	if err := denegadoPor(nil); err != vecports.ErrAutorizacionAlmacenInvalida {
		t.Fatalf("sin causa la denegación debe ser el motivo cerrado: %v", err)
	}
	if resolutor.llamadas != 0 {
		t.Fatalf("la denegación llegó al PDP: %d", resolutor.llamadas)
	}
}
