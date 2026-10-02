package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestOperacionRenunciaJustificadaRRHHConservaJustificanteYRecibo(t *testing.T) {
	reglas := reglasTransicionesPrueba{destinos: map[string][]string{domain.SituacionRenuncia: {domain.SituacionDisponible, domain.SituacionExcluido}}}
	servicio, repo, ahora := servicioDesdeRenunciaPrueba(t, reglas)
	q := ports.SolicitudOperacionSituacion{
		SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora),
		Operacion:                              domain.OperacionReactivar,
		Justificante: domain.JustificanteOperacionSituacion{
			Tipo: domain.JustificanteSolicitudCandidato, Referencia: "justificante:renuncia", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		Validador: "persona:rrhh",
	}
	q.Destino = domain.SituacionDisponible
	q.Motivo = "Justificante de renuncia validado por RRHH"
	primero, err := servicio.Operar(context.Background(), q)
	if err != nil || primero.Situacion != domain.SituacionDisponible || primero.ReciboRef == "" || repo.escrituras != 1 {
		t.Fatalf("reactivación: resultado=%+v error=%v escrituras=%d", primero, err, repo.escrituras)
	}
	if repo.operacion.Justificante != q.Justificante || repo.operacion.Validador != q.Validador || repo.operacion.Actor != q.ResultadoContexto.Contexto.PersonaRef {
		t.Fatal("la operación no conserva su justificante, validador y actor")
	}
	// La situación vigente ya refleja el efecto: recuperar no añade otra fila.
	repo.vigente = primero.SituacionParticipacion
	segundo, err := servicio.Operar(context.Background(), q)
	if err != nil || !segundo.Reutilizada || segundo.ReciboRef != primero.ReciboRef || !segundo.Desde.Equal(primero.Desde) || repo.escrituras != 1 {
		t.Fatalf("recuperación: resultado=%+v error=%v escrituras=%d", segundo, err, repo.escrituras)
	}
	q.Justificante.Referencia = "justificante:otro"
	if _, err := servicio.Operar(context.Background(), q); !errors.Is(err, ports.ErrClaveOperacionReutilizada) || repo.escrituras != 1 {
		t.Fatalf("otro justificante con la misma clave: error=%v escrituras=%d", err, repo.escrituras)
	}
}

func TestOperacionRenunciaRRHHNoReincorporaSinValidacionNiRevierteExclusion(t *testing.T) {
	for _, caso := range []string{"sin_justificante", "sin_validador", "exclusion_adoptada"} {
		t.Run(caso, func(t *testing.T) {
			servicio, repo, ahora := servicioDesdeRenunciaPrueba(t, nil)
			q := ports.SolicitudOperacionSituacion{
				SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora),
				Operacion:                              domain.OperacionReactivar,
				Justificante: domain.JustificanteOperacionSituacion{
					Tipo: domain.JustificanteSolicitudCandidato, Referencia: "justificante:renuncia", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
				Validador: "persona:rrhh",
			}
			q.Destino = domain.SituacionDisponible
			esperado := domain.ErrOperacionSituacionParticipacionInvalida
			switch caso {
			case "sin_justificante":
				q.Justificante = domain.JustificanteOperacionSituacion{}
			case "sin_validador":
				q.Validador = ""
			case "exclusion_adoptada":
				repo.vigente.Situacion = domain.SituacionExcluido
				esperado = domain.ErrCambioSituacionParticipacionInvalido
			}
			if _, err := servicio.Operar(context.Background(), q); !errors.Is(err, esperado) || repo.escrituras != 0 {
				t.Fatalf("operación rechazada: error=%v escrituras=%d", err, repo.escrituras)
			}
		})
	}
}
