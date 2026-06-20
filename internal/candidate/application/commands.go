package application

import "vec-diputacion-granada/internal/candidate/domain"

type CreateCandidateCommand struct {
	ID     string `json:"id"`
	DNI    string `json:"dni"`
	Nombre string `json:"nombre"`
	Email  string `json:"email"`
	CallID string `json:"call_id,omitempty"`
}

type AddMeritCommand struct {
	ID     string            `json:"id"`
	Tipo   domain.MeritType  `json:"tipo"`
	Datos  MeritDataCommand  `json:"datos"`
	Estado domain.MeritState `json:"estado,omitempty"`
}

type MeritDataCommand struct {
	Meses       int     `json:"meses,omitempty"`
	Horas       int     `json:"horas,omitempty"`
	PuntosFijos float64 `json:"puntos_fijos,omitempty"`
}
