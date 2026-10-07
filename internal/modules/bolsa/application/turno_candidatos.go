package application

import (
	"sort"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

// CandidatoTurno proyecta el orden vigente B6. OrdenActa solo desempata;
// nunca sustituye un orden vigente ausente.
type CandidatoTurno struct {
	ParticipacionRef string
	NombreVisible    string
	Orden            *int
	OrdenActa        int
	Estado           string
}

type UltimoLlamadoTurno struct {
	Candidato CandidatoTurno
	Contacto  dominiobolsa.ContactoParticipacion
}

const (
	EstadoSiguientePrimeroDisponible = "primero_disponible"
	EstadoSiguienteSinDisponibles    = "sin_disponibles"
)

type TurnoCandidatos struct {
	PoliticaRef     string
	PoliticaVersion uint64
	Provisional     bool
	UltimoLlamado   *UltimoLlamadoTurno
	Siguiente       *CandidatoTurno
	EstadoSiguiente string
}

// ProyectarTurnoCandidatos informa el último contacto comunicado y el primer
// disponible en el orden vigente B6. No evalúa elegibilidad para una oferta.
func ProyectarTurnoCandidatos(politicaRef string, politicaVersion uint64, provisional bool, candidatos []CandidatoTurno, contactos []dominiobolsa.ContactoParticipacion) TurnoCandidatos {
	turno := TurnoCandidatos{
		PoliticaRef: politicaRef, PoliticaVersion: politicaVersion,
		Provisional: provisional, UltimoLlamado: UltimoLlamadoDesdeContactos(candidatos, contactos),
		EstadoSiguiente: EstadoSiguienteSinDisponibles,
	}
	for _, candidato := range OrdenarCandidatosTurno(candidatos) {
		if candidato.Orden != nil {
			seleccionado := candidato
			turno.Siguiente = &seleccionado
			turno.EstadoSiguiente = EstadoSiguientePrimeroDisponible
			break
		}
	}
	return turno
}

// OrdenarCandidatosTurno devuelve una copia con orden total estable. Una
// posición sin orden vigente queda detrás de todas las posiciones vigentes.
func OrdenarCandidatosTurno(candidatos []CandidatoTurno) []CandidatoTurno {
	ordenados := append([]CandidatoTurno(nil), candidatos...)
	sort.Slice(ordenados, func(i, j int) bool {
		a, b := ordenados[i], ordenados[j]
		if (a.Orden == nil) != (b.Orden == nil) {
			return a.Orden != nil
		}
		if a.Orden != nil && *a.Orden != *b.Orden {
			return *a.Orden < *b.Orden
		}
		if a.OrdenActa != b.OrdenActa {
			return a.OrdenActa < b.OrdenActa
		}
		return a.ParticipacionRef < b.ParticipacionRef
	})
	return ordenados
}

// UltimoLlamadoDesdeContactos solo considera comunicaciones ligadas a un
// llamamiento. Un envío fallido no acredita haber llamado a la persona.
func UltimoLlamadoDesdeContactos(candidatos []CandidatoTurno, contactos []dominiobolsa.ContactoParticipacion) *UltimoLlamadoTurno {
	porRef := make(map[string]CandidatoTurno, len(candidatos))
	for _, candidato := range candidatos {
		porRef[candidato.ParticipacionRef] = candidato
	}
	var ultimo *UltimoLlamadoTurno
	for _, contacto := range contactos {
		candidato, existe := porRef[contacto.ParticipacionRef]
		if !existe || contacto.LlamamientoRef == "" || contacto.Resultado == dominiobolsa.ResultadoContactoNoEnviado {
			continue
		}
		if ultimo == nil || contacto.Instante.After(ultimo.Contacto.Instante) ||
			(contacto.Instante.Equal(ultimo.Contacto.Instante) && contacto.ContactoRef > ultimo.Contacto.ContactoRef) {
			ultimo = &UltimoLlamadoTurno{Candidato: candidato, Contacto: contacto}
		}
	}
	return ultimo
}
