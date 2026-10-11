package application

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type repositorioIngestaPrueba struct {
	ports.RepositorioIngestaMayan
	plan            ports.PlanIngestaMayan
	despachos       int
	resoluciones    int
	reservaRecibida ports.ReservaRecursoDocumental
}

func (r *repositorioIngestaPrueba) ReservarRecurso(_ context.Context, o ports.OrdenReservarRecursoMayan) (ports.ReservaRecursoDocumental, error) {
	r.reservaRecibida = o.Reserva
	respuesta := o.Reserva
	respuesta.Estado = ports.EstadoRecursoReservado
	respuesta.Version = 1
	respuesta.CreadaEn = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	respuesta.ExpiraEn = respuesta.CreadaEn.Add(10 * time.Minute)
	return respuesta, nil
}

func (r *repositorioIngestaPrueba) ResolverIngesta(_ context.Context, _ ports.OrdenCambiarIngestaMayan) (ports.PlanIngestaMayan, error) {
	r.resoluciones++
	return r.plan, nil
}

func (r *repositorioIngestaPrueba) ConsultarIngesta(_ context.Context, _ ports.OrdenConsultarIngestaMayan) (ports.PlanIngestaMayan, error) {
	return r.plan, nil
}
func (r *repositorioIngestaPrueba) DespacharIngesta(_ context.Context, _ ports.OrdenCambiarIngestaMayan) (ports.PlanIngestaMayan, bool, error) {
	r.despachos++
	return ports.PlanIngestaMayan{}, false, nil
}

type autoridadIngestaPrueba struct {
	t         *testing.T
	ahora     time.Time
	principal string
}

func (a autoridadIngestaPrueba) AutorizarIngestaMayan(_ context.Context, accion string, preimagen []byte, planRef, expedienteRef string) (ports.AutorizacionV3, error) {
	principal := a.principal
	if principal == "" {
		principal = "per:0001"
	}
	return ports.AutorizacionV3{Material: materialExternaPrueba(a.t, accion, planRef, preimagen, a.ahora),
		Accion: accion, Finalidad: ports.FinalidadIngestaMayan, RecursoRef: planRef, AmbitoRef: expedienteRef,
		PrincipalID: principal, PerfilActivoRef: "perfil:0001", CorrelacionRef: "corr:0001"}, nil
}

func TestReservaRecursoFijaPersonaYContextoV2AntesDeS3(t *testing.T) {
	escenario := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	datos, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioIngestaPrueba{}
	s := ServicioIngestaMayan{Repositorio: repo, Autoridad: autoridadIngestaPrueba{t: t, ahora: escenario.ahora, principal: escenario.resultado.Contexto.PersonaRef}, Reloj: relojExternaPrueba{escenario.ahora}}
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	reserva, err := s.ReservarRecurso(context.Background(), ref("a"), ref("b"), strings.Repeat("c", 64),
		escenario.resultado, datos.VinculoAutenticacionActor)
	if err != nil || reserva.RecursoRef != ref("a") || repo.reservaRecibida.PersonaRef != escenario.resultado.Contexto.PersonaRef ||
		repo.reservaRecibida.RegistroContextoRef != escenario.resultado.RegistroContextoRef ||
		repo.reservaRecibida.HuellaContextoSHA256 != escenario.resultado.HuellaSHA256 {
		t.Fatalf("reserva sin titular V2: err=%v reserva=%+v", err, repo.reservaRecibida)
	}
}

type custodioIngestaPrueba struct {
	llamadas  int
	recibido  ports.DocumentoMayan
	respuesta ports.DocumentoMayan
}

func (c *custodioIngestaPrueba) IngerirOriginal(context.Context, ports.PlanIngestaMayan, io.Reader) (ports.DocumentoMayan, error) {
	c.llamadas++
	return ports.DocumentoMayan{}, errors.New("POST inesperado")
}
func (c *custodioIngestaPrueba) ReconciliarOriginal(_ context.Context, d ports.DocumentoMayan) (ports.DocumentoMayan, error) {
	c.recibido = d
	return c.respuesta, nil
}

func TestConciliarConservaVersionConocidaYRechazaSustitucionRemota(t *testing.T) {
	p := ports.PlanIngestaMayan{Estado: ports.EstadoIngestaIndeterminada,
		Version: 3, DocumentoID: "15", ArchivoID: "20", VersionID: "30", HuellaSHA256: strings.Repeat("a", 64), Tamano: 10, MIME: "application/pdf"}
	d := ports.DocumentoMayan{DocumentoID: "15", ArchivoID: "20", VersionID: "31", HuellaSHA256: p.HuellaSHA256, Tamano: p.Tamano, MIME: p.MIME}
	if evidenciaMayanConservaIDs(p, d) {
		t.Fatal("la versión remota sustituyó el intento conocido")
	}
}

type cargasIngestaPrueba struct {
	vecports.RepositorioCargasDocumentales
}

func TestIngestaIndeterminadaNuncaReenviaPOST(t *testing.T) {
	ahora := time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	repo := &repositorioIngestaPrueba{plan: ports.PlanIngestaMayan{PlanRef: ref("1"), ExpedienteRef: ref("2"), Estado: ports.EstadoIngestaIndeterminada, Version: 3}}
	custodio := &custodioIngestaPrueba{}
	s := ServicioIngestaMayan{Repositorio: repo, Autoridad: autoridadIngestaPrueba{t: t, ahora: ahora}, Custodio: custodio, Cargas: cargasIngestaPrueba{}, Politicas: politicasExternaPrueba{}, Reloj: relojExternaPrueba{ahora}}
	_, err := s.Ingerir(context.Background(), repo.plan.PlanRef, repo.plan.ExpedienteRef, vecdomain.ResultadoContextoActorRegistradoV2{}, vecdomain.VinculoAutenticacionActorV2{}, strings.NewReader("bytes"))
	if !errors.Is(err, ports.ErrIngestaNoReenviable) || custodio.llamadas != 0 || repo.despachos != 0 {
		t.Fatalf("replay indeterminado: err=%v POST=%d despachos=%d", err, custodio.llamadas, repo.despachos)
	}
}
