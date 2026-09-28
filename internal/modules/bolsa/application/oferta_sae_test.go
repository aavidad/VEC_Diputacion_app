package application

import (
	"context"
	"errors"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type catalogoSAEInexistente struct{ llamadas int }

func (c *catalogoSAEInexistente) CatalogoOfertaSAEVigente(context.Context) (dominiobolsa.CatalogoOfertaSAE, error) {
	c.llamadas++
	return dominiobolsa.CatalogoOfertaSAE{}, puertosbolsa.ErrOfertaSAENoDisponible
}

type ambitoSAEPrueba struct {
	err      error
	llamadas int
}

func (a *ambitoSAEPrueba) ResolverAmbitoOfertaSAE(context.Context, dominiovec.ContextoActor) (puertosbolsa.AmbitoOfertaSAE, error) {
	a.llamadas++
	if a.err != nil {
		return puertosbolsa.AmbitoOfertaSAE{}, a.err
	}
	return puertosbolsa.AmbitoOfertaSAE{UnidadRef: "unidad:rrhh", AmbitoRef: "ambito:seleccion"}, nil
}

type repoSAENoLlamado struct{ llamadas int }

func (r *repoSAENoLlamado) Preparar(context.Context, puertosbolsa.OrdenPrepararOfertaSAE) (puertosbolsa.ReciboOfertaSAE, error) {
	r.llamadas++
	return puertosbolsa.ReciboOfertaSAE{}, nil
}
func (r *repoSAENoLlamado) Actuar(context.Context, puertosbolsa.OrdenActuarOfertaSAE) (puertosbolsa.ReciboOfertaSAE, error) {
	r.llamadas++
	return puertosbolsa.ReciboOfertaSAE{}, nil
}
func (r *repoSAENoLlamado) Consultar(context.Context, puertosbolsa.OrdenConsultarOfertaSAE) (dominiobolsa.OfertaSAE, error) {
	r.llamadas++
	return dominiobolsa.OfertaSAE{}, nil
}

func solicitudSAEContextoPrueba(t *testing.T, ahora time.Time) (dominiovec.ResultadoContextoActorRegistradoV2, dominiovec.VinculoAutenticacionActorV2) {
	t.Helper()
	r, v, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	return r, v
}

func TestOfertaSAESinCatalogoPublicadoNoPreparaNiAutoriza(t *testing.T) {
	ahora := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	resultado, vinculo := solicitudSAEContextoPrueba(t, ahora)
	cat, ambito, repo := &catalogoSAEInexistente{}, &ambitoSAEPrueba{}, &repoSAENoLlamado{}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, err := NuevoServicioOfertaSAE(ambito, cat, autorizador, repo, nil, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Preparar(context.Background(), puertosbolsa.SolicitudPrepararOfertaSAE{
		Vinculo: vinculo, ResultadoContexto: resultado,
		Datos: dominiobolsa.DatosOfertaSAE{CategoriaRef: "categoria:operario", PuestoRef: "puesto:auxiliar", NumeroPlazas: 1,
			Modalidad: "interinidad", Duracion: "Hasta cobertura reglamentaria", Requisitos: "Requisitos publicados"},
		ClaveIdempotencia: "clave-oferta-sae-0001", Correlacion: correlacionBorradorPrueba(t), MotivoAutorizacion: motivoBorradorPrueba(),
	})
	if !errors.Is(err, puertosbolsa.ErrOfertaSAENoDisponible) || cat.llamadas != 1 || ambito.llamadas != 0 || autorizador.llamadas != 0 || repo.llamadas != 0 {
		t.Fatalf("sin catálogo: err=%v catálogo=%d ámbito=%d autorizador=%d repo=%d", err, cat.llamadas, ambito.llamadas, autorizador.llamadas, repo.llamadas)
	}
}

func TestOfertaSAENoConsultaSinAmbitoAcreditado(t *testing.T) {
	ahora := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	resultado, vinculo := solicitudSAEContextoPrueba(t, ahora)
	ambito, repo := &ambitoSAEPrueba{err: dominiovec.ErrAutorizacionDenegada}, &repoSAENoLlamado{}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, _ := NuevoServicioOfertaSAE(ambito, &catalogoSAEInexistente{}, autorizador, repo, nil, func() time.Time { return ahora })
	_, err := servicio.Consultar(context.Background(), puertosbolsa.SolicitudConsultarOfertaSAE{Vinculo: vinculo, ResultadoContexto: resultado,
		OfertaRef: "oferta-sae:001", Correlacion: correlacionBorradorPrueba(t), MotivoAutorizacion: motivoBorradorPrueba()})
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || ambito.llamadas != 1 || autorizador.llamadas != 0 || repo.llamadas != 0 {
		t.Fatalf("denegación: err=%v ámbito=%d autorizador=%d repo=%d", err, ambito.llamadas, autorizador.llamadas, repo.llamadas)
	}
}

func TestOfertaSAEResolucionSinAutoridadPersonaQuedaPendienteSinEfecto(t *testing.T) {
	ahora := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	resultado, vinculo := solicitudSAEContextoPrueba(t, ahora)
	ambito, repo := &ambitoSAEPrueba{}, &repoSAENoLlamado{}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, _ := NuevoServicioOfertaSAE(ambito, &catalogoSAEInexistente{}, autorizador, repo, nil, func() time.Time { return ahora })
	q := puertosbolsa.SolicitudActuarOfertaSAE{Vinculo: vinculo, ResultadoContexto: resultado, OfertaRef: "oferta-sae:001",
		Cambio: dominiobolsa.CambioOfertaSAE{Accion: dominiobolsa.AccionSAEResolver, Clave: "clave-resolver-0001", VersionEsperada: 5,
			CandidatoElegidoRef: "candidato:externo:001"}, Correlacion: correlacionBorradorPrueba(t), MotivoAutorizacion: motivoBorradorPrueba()}
	if _, err := servicio.Actuar(context.Background(), q); !errors.Is(err, dominiobolsa.ErrOfertaSAEConciliacionPendiente) ||
		ambito.llamadas != 0 || autorizador.llamadas != 0 || repo.llamadas != 0 {
		t.Fatalf("sin autoridad Persona: err=%v ámbito=%d autorizador=%d repo=%d", err, ambito.llamadas, autorizador.llamadas, repo.llamadas)
	}
	q.Cambio.Accion = dominiobolsa.AccionSAEConciliarPersona
	if _, err := servicio.Actuar(context.Background(), q); !errors.Is(err, dominiobolsa.ErrOfertaSAEConciliacionPendiente) {
		t.Fatalf("conciliación sin autoridad: %v", err)
	}
	q.Cambio.Accion = dominiobolsa.AccionSAERegistrarCandidato
	q.Cambio.Candidato = &dominiobolsa.CandidatoOfertaSAE{Referencia: "candidato:externo:001", PersonaRef: "per_0123456789abcdefghijkl",
		NombreProtegidoRef: "dato:nombre:001", DocumentoProtegidoRef: "dato:documento:001", ContactoProtegidoRef: "dato:contacto:001"}
	if _, err := servicio.Actuar(context.Background(), q); !errors.Is(err, dominiobolsa.ErrOfertaSAEInvalida) || autorizador.llamadas != 0 || repo.llamadas != 0 {
		t.Fatalf("per_ cliente aceptado: %v", err)
	}
	q.Cambio.Accion = dominiobolsa.AccionSAEResolver
	q.Cambio.Candidato = nil
	q.Cambio.Acreditacion = &dominiobolsa.AcreditacionPersonaSAE{PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1,
		EvidenciaRef: "evidencia:inventada", VerificadaEn: ahora.Add(-time.Minute), ValidaHasta: ahora.Add(time.Minute)}
	if _, err := servicio.Actuar(context.Background(), q); !errors.Is(err, dominiobolsa.ErrOfertaSAEInvalida) || repo.llamadas != 0 {
		t.Fatalf("acreditación del cliente aceptada: %v", err)
	}
}
