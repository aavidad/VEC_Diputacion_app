package application

import (
	"context"
	"errors"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type fuenteOrigenPrueba map[string]*dominiobolsa.MarcaOrigenDatosContacto

func (f fuenteOrigenPrueba) OrigenContactoParticipacion(_ context.Context, participacion string) (*dominiobolsa.MarcaOrigenDatosContacto, error) {
	if participacion == "participacion:caida" {
		return nil, errors.New("sin repositorio")
	}
	return f[participacion], nil
}

type repositorioEmisionPrueba struct {
	puertosbolsa.RepositorioEmisionLlamamiento
	emision puertosbolsa.EmisionLlamamiento
}

func (r repositorioEmisionPrueba) Recuperar(context.Context, string, string) (puertosbolsa.EmisionLlamamiento, error) {
	return r.emision, nil
}

func TestEmisionAvisaDelContactoConvocaVencidoSinBloquear(t *testing.T) {
	ahora := time.Date(2027, 10, 1, 8, 0, 0, 0, time.UTC)
	vencida := &dominiobolsa.MarcaOrigenDatosContacto{Origen: dominiobolsa.OrigenDatosContactoConvoca, VigenteHasta: time.Date(2027, 9, 28, 22, 0, 0, 0, time.UTC), UltimoDia: "2027-09-28"}
	vigente := &dominiobolsa.MarcaOrigenDatosContacto{Origen: dominiobolsa.OrigenDatosContactoConvoca, VigenteHasta: time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC), UltimoDia: "2027-12-31"}
	emision := puertosbolsa.EmisionLlamamiento{BolsaRef: "bolsa:1", Estado: "emitido", Participaciones: []string{"participacion:vencida", "participacion:vigente", "participacion:propia", "participacion:caida"}}
	servicio, err := NuevoServicioEmisionLlamamiento(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repositorioEmisionPrueba{emision: emision}, fuenteCorreoPrueba{}, emisorCorreoPrueba{}, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	solicitud := puertosbolsa.SolicitudRecuperarLlamamiento{ContextoActor: dominiovec.ContextoActor{PersonaRef: "per_0123456789abcdefghijkl"}, BolsaRef: "bolsa:1", ClaveIdempotencia: "clave-1"}
	sinAvisos, err := servicio.RecuperarLlamamiento(context.Background(), solicitud)
	if err != nil || sinAvisos.AvisosContacto != nil {
		t.Fatalf("sin regla compuesta no hay avisos: %+v %v", sinAvisos.AvisosContacto, err)
	}
	if servicio.EstablecerAvisoContactoNoConfirmado(nil) == nil {
		t.Fatal("una fuente nula debe rechazarse")
	}
	if err := servicio.EstablecerAvisoContactoNoConfirmado(fuenteOrigenPrueba{"participacion:vencida": vencida, "participacion:vigente": vigente}); err != nil {
		t.Fatal(err)
	}
	conAvisos, err := servicio.RecuperarLlamamiento(context.Background(), solicitud)
	if err != nil || conAvisos.Estado != "emitido" || len(conAvisos.AvisosContacto) != 2 {
		t.Fatalf("avisos: %+v %v", conAvisos.AvisosContacto, err)
	}
	if a := conAvisos.AvisosContacto[0]; a.ParticipacionRef != "participacion:vencida" || a.Aviso != puertosbolsa.AvisoContactoNoConfirmado || a.UltimoDia != "2027-09-28" {
		t.Fatalf("aviso vencido: %+v", a)
	}
	if a := conAvisos.AvisosContacto[1]; a.ParticipacionRef != "participacion:caida" || a.Aviso != puertosbolsa.AvisoEstadoContactoNoDisponible {
		t.Fatalf("un origen ilegible nunca se da por confirmado: %+v", a)
	}
}

type fuenteCorreoPrueba struct{}

func (fuenteCorreoPrueba) CorreoParticipacion(context.Context, string) (string, error) {
	return "persona@ejemplo.es", nil
}

type emisorCorreoPrueba struct{}

func (emisorCorreoPrueba) EnviarCorreo(context.Context, string, string, string, string, time.Time) bool {
	return true
}
