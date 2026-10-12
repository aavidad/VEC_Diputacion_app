package application

import (
	"bytes"
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type repositorioVersionPoliticaPrueba struct {
	ports.RepositorioIngestaMayan
	reserva       ports.ReservaRecursoDocumental
	preparaciones int
}

func (r *repositorioVersionPoliticaPrueba) ConsultarReserva(context.Context, ports.OrdenConsultaReservaMayan) (ports.ReservaRecursoDocumental, error) {
	return r.reserva, nil
}
func (r *repositorioVersionPoliticaPrueba) PrepararIngesta(context.Context, ports.OrdenPrepararIngestaMayan) (ports.PlanIngestaMayan, error) {
	r.preparaciones++
	return ports.PlanIngestaMayan{}, errors.New("preparación inesperada")
}

func TestPrepararMayanRechazaVersionPoliticaFueraDeInt64SinEfectos(t *testing.T) {
	e := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	datosActor, err := e.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	for _, caso := range []struct {
		nombre, version string
		valor           uint64
		consultaCarga   int
	}{
		{"max_int64", "aceptada", uint64(math.MaxInt64), 1},
		{"max_int64_mas_uno", "rechazada", uint64(math.MaxInt64) + 1, 0},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud, err := vecports.NuevaSolicitudPoliticaConservacionDocumental(
				ref("1"), ref("2"), ref("3"), ref("4"), ref("5"), caso.valor,
				bytes.Repeat([]byte{0x6a}, 32), ref("6"), e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			politica, err := vecports.NuevaPoliticaConservacionDocumental(solicitud,
				time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC),
				vecports.ProteccionPoliticaConservacionDocumentalOrdinaria, "",
				vecports.EstadoPoliticaConservacionDocumentalAprobada, time.Time{})
			if err != nil {
				t.Fatal(err)
			}
			repo := &repositorioVersionPoliticaPrueba{reserva: ports.ReservaRecursoDocumental{
				RecursoRef: solicitud.ExpedienteRef(), PersonaRef: e.resultado.Contexto.PersonaRef,
				Estado: ports.EstadoRecursoVinculado}}
			fuente := &fuenteCargaNoAutorizadaPrueba{}
			custodio := &custodioIngestaPrueba{}
			s := ServicioIngestaMayan{Repositorio: repo,
				Autoridad: autoridadIngestaPrueba{t: t, ahora: e.ahora, principal: e.resultado.Contexto.PersonaRef},
				Cargas:    fuente, Custodio: custodio, Politicas: politicasExternaPrueba{politica},
				Reloj: relojExternaPrueba{e.ahora}}
			p := ports.PlanIngestaMayan{CargaID: ref("7"), TipoRef: solicitud.TipoDocumentalRef(),
				ClaveReserva: ref("8"), ClaveIdempotencia: ref("9"), EvidenciaRef: ref("a"),
				HuellaSolicitudSHA256: strings.Repeat("b", 64), HuellaSHA256: strings.Repeat("c", 64),
				MIME: "application/pdf", Tamano: 100, CargaVersion: 5}
			_, err = s.Preparar(context.Background(), p, solicitud, e.resultado, datosActor.VinculoAutenticacionActor)
			if caso.consultaCarga == 0 {
				if !errors.Is(err, ports.ErrSolicitudInvalida) {
					t.Fatalf("versión desbordada aceptada: %v", err)
				}
			} else if !errors.Is(err, ports.ErrCapacidadNoDisponible) {
				t.Fatalf("MaxInt64 se desbordó antes de la fuente: %v", err)
			}
			if fuente.consultas != caso.consultaCarga || repo.preparaciones != 0 || custodio.llamadas != 0 {
				t.Fatalf("hubo efectos con versión %s: consultas=%d planes=%d POST=%d",
					caso.version, fuente.consultas, repo.preparaciones, custodio.llamadas)
			}
		})
	}
}
