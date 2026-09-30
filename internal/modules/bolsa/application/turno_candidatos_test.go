package application

import (
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

func TestOrdenarCandidatosTurnoUsaOrdenVigenteActaYReferencia(t *testing.T) {
	uno, dos := 1, 2
	entrada := []CandidatoTurno{
		{ParticipacionRef: "nulo", OrdenActa: 1},
		{ParticipacionRef: "b", Orden: &uno, OrdenActa: 2},
		{ParticipacionRef: "d", Orden: &dos, OrdenActa: 1},
		{ParticipacionRef: "a", Orden: &uno, OrdenActa: 2},
		{ParticipacionRef: "c", Orden: &uno, OrdenActa: 1},
	}
	ordenados := OrdenarCandidatosTurno(entrada)
	for i, ref := range []string{"c", "a", "b", "d", "nulo"} {
		if ordenados[i].ParticipacionRef != ref {
			t.Fatalf("posición %d = %s; se esperaba %s", i, ordenados[i].ParticipacionRef, ref)
		}
	}
	if entrada[0].ParticipacionRef != "nulo" {
		t.Fatal("la ordenación modificó la entrada")
	}
}

func TestUltimoLlamadoDesdeContactosExcluyeNoEnviadoYSinLlamamiento(t *testing.T) {
	instante := time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC)
	candidatos := []CandidatoTurno{{ParticipacionRef: "a"}, {ParticipacionRef: "b"}}
	contactos := []dominiobolsa.ContactoParticipacion{
		{ContactoRef: "1", ParticipacionRef: "a", LlamamientoRef: "llamamiento:a", Resultado: dominiobolsa.ResultadoContactoEnviado, Instante: instante},
		{ContactoRef: "3", ParticipacionRef: "b", LlamamientoRef: "llamamiento:b", Resultado: dominiobolsa.ResultadoContactoNoEnviado, Instante: instante.Add(3 * time.Hour)},
		{ContactoRef: "4", ParticipacionRef: "b", Resultado: dominiobolsa.ResultadoContactoContactado, Instante: instante.Add(4 * time.Hour)},
		{ContactoRef: "2", ParticipacionRef: "b", LlamamientoRef: "llamamiento:b", Resultado: dominiobolsa.ResultadoContactoContactado, Instante: instante},
		{ContactoRef: "5", ParticipacionRef: "ajeno", LlamamientoRef: "llamamiento:ajeno", Resultado: dominiobolsa.ResultadoContactoContactado, Instante: instante.Add(5 * time.Hour)},
	}
	ultimo := UltimoLlamadoDesdeContactos(candidatos, contactos)
	if ultimo == nil || ultimo.Candidato.ParticipacionRef != "b" || ultimo.Contacto.ContactoRef != "2" {
		t.Fatalf("último llamado = %#v", ultimo)
	}
	if UltimoLlamadoDesdeContactos(candidatos, nil) != nil {
		t.Fatal("sin contactos se inventó un llamamiento")
	}
}

func TestProyectarTurnoCandidatosEligePrimeroVigenteSinCursorDeContacto(t *testing.T) {
	uno, dos := 1, 2
	candidatos := []CandidatoTurno{
		{ParticipacionRef: "segundo", NombreVisible: "Segundo", Orden: &dos, OrdenActa: 2, Estado: "disponible"},
		{ParticipacionRef: "primero", NombreVisible: "Primero", Orden: &uno, OrdenActa: 1, Estado: "disponible"},
		{ParticipacionRef: "sin-orden", OrdenActa: 3, Estado: "disponible"},
	}
	contactos := []dominiobolsa.ContactoParticipacion{{ContactoRef: "contacto:2", ParticipacionRef: "segundo", LlamamientoRef: "llamamiento:2", Resultado: dominiobolsa.ResultadoContactoContactado, Instante: time.Now().UTC()}}
	turno := ProyectarTurnoCandidatos("politica:1", 3, true, candidatos, contactos)
	if turno.Siguiente == nil || turno.Siguiente.ParticipacionRef != "primero" || turno.EstadoSiguiente != EstadoSiguientePrimeroDisponible || turno.UltimoLlamado == nil || turno.UltimoLlamado.Candidato.ParticipacionRef != "segundo" || turno.PoliticaRef != "politica:1" || turno.PoliticaVersion != 3 || !turno.Provisional {
		t.Fatalf("turno = %#v", turno)
	}
	candidatos[1].Estado = "disponible_desde"
	turno = ProyectarTurnoCandidatos("politica:1", 3, true, candidatos, contactos)
	if turno.Siguiente == nil || turno.Siguiente.ParticipacionRef != "primero" {
		t.Fatalf("B6 dejó posición vigente disponible_desde: %#v", turno.Siguiente)
	}
	candidatos[1].Estado = "no_disponible"
	candidatos[1].Orden = nil
	turno = ProyectarTurnoCandidatos("politica:1", 3, true, candidatos, contactos)
	if turno.Siguiente == nil || turno.Siguiente.ParticipacionRef != "segundo" {
		t.Fatalf("primer disponible = %#v", turno.Siguiente)
	}
	candidatos[0].Estado = "no_disponible"
	candidatos[0].Orden = nil
	turno = ProyectarTurnoCandidatos("politica:1", 3, true, candidatos, contactos)
	if turno.Siguiente != nil || turno.EstadoSiguiente != EstadoSiguienteSinDisponibles {
		t.Fatalf("sin posiciones vigentes disponibles = %#v", turno)
	}
}
