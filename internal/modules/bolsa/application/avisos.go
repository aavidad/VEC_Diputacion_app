package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	RotuloComputoTresAnosPendiente = "Cómputo legal de encadenamiento pendiente de RRHH (duda 13; art. 15.5 ET tras la Ley 20/2021)"
	vigenciaCursorAvisos           = 15 * time.Minute
	maximoOffsetAvisos             = 100_000
)

type ConsultaAvisos struct {
	Cursor string
	Limite int
}

type PaginaAvisos struct {
	Avisos          []dominiobolsa.AvisoRRHH
	CursorSiguiente string
	GeneradaEn      time.Time
	Provisionalidad string
	Conteos         map[string]int
	Desde           int
	Hasta           int
	Total           int
}

type cursorAvisos struct {
	Corte  time.Time `json:"corte"`
	Offset int       `json:"offset"`
}

type ServicioAvisosRRHH struct {
	consulta puertosbolsa.ConsultaAvisosRRHH
	ahora    func() time.Time
}

func NuevoServicioAvisosRRHH(consulta puertosbolsa.ConsultaAvisosRRHH, ahora func() time.Time) (*ServicioAvisosRRHH, error) {
	if consulta == nil || ahora == nil {
		return nil, puertosbolsa.ErrConsultaAvisosNoDisponible
	}
	return &ServicioAvisosRRHH{consulta: consulta, ahora: ahora}, nil
}

func (s *ServicioAvisosRRHH) Consultar(ctx context.Context, peticion ConsultaAvisos) (PaginaAvisos, error) {
	if s == nil || s.consulta == nil || s.ahora == nil || ctx == nil || peticion.Limite < 1 || peticion.Limite > 100 {
		return PaginaAvisos{}, puertosbolsa.ErrConsultaAvisosNoDisponible
	}
	ahora := s.ahora().UTC().Truncate(time.Microsecond)
	cursor := cursorAvisos{Corte: ahora}
	if peticion.Cursor != "" {
		bytes, err := base64.RawURLEncoding.DecodeString(peticion.Cursor)
		if err != nil || json.Unmarshal(bytes, &cursor) != nil || cursor.Corte.IsZero() || cursor.Offset < 0 || cursor.Offset > maximoOffsetAvisos {
			return PaginaAvisos{}, puertosbolsa.ErrConsultaAvisosNoDisponible
		}
		cursor.Corte = cursor.Corte.UTC().Truncate(time.Microsecond)
		if cursor.Corte.After(ahora) || cursor.Corte.Before(ahora.Add(-vigenciaCursorAvisos)) {
			return PaginaAvisos{}, puertosbolsa.ErrConsultaAvisosNoDisponible
		}
	}
	filas, err := s.consulta.ListarAvisosRRHH(ctx, cursor.Corte, cursor.Offset, peticion.Limite+1)
	if err != nil {
		return PaginaAvisos{}, err
	}
	conteos, err := s.consulta.ContarAvisosRRHH(ctx, cursor.Corte)
	if err != nil || conteos[dominiobolsa.AvisoSaltoOrden] < 0 || conteos[dominiobolsa.AvisoTresAnos] < 0 {
		return PaginaAvisos{}, puertosbolsa.ErrConsultaAvisosNoDisponible
	}
	for _, aviso := range filas {
		if aviso.Validar() != nil {
			return PaginaAvisos{}, puertosbolsa.ErrConsultaAvisosNoDisponible
		}
	}
	pagina := PaginaAvisos{GeneradaEn: cursor.Corte, Provisionalidad: RotuloComputoTresAnosPendiente, Conteos: map[string]int{dominiobolsa.AvisoSaltoOrden: conteos[dominiobolsa.AvisoSaltoOrden], dominiobolsa.AvisoTresAnos: conteos[dominiobolsa.AvisoTresAnos]}}
	pagina.Total = pagina.Conteos[dominiobolsa.AvisoSaltoOrden] + pagina.Conteos[dominiobolsa.AvisoTresAnos]
	if len(filas) > peticion.Limite {
		pagina.Avisos = append([]dominiobolsa.AvisoRRHH(nil), filas[:peticion.Limite]...)
		siguiente, _ := json.Marshal(cursorAvisos{Corte: cursor.Corte, Offset: cursor.Offset + peticion.Limite})
		pagina.CursorSiguiente = base64.RawURLEncoding.EncodeToString(siguiente)
	} else {
		pagina.Avisos = append([]dominiobolsa.AvisoRRHH(nil), filas...)
	}
	if len(pagina.Avisos) != 0 {
		pagina.Desde, pagina.Hasta = cursor.Offset+1, cursor.Offset+len(pagina.Avisos)
	}
	return pagina, nil
}
