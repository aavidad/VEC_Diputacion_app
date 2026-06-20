package domain

import "strings"

// Candidate represents an applicant in the Diputacion de Granada pool.
type Candidate struct {
	ID     string
	DNI    string
	Nombre string
	Email  string
}

func NewCandidate(id, dni, nombre, email string) (Candidate, error) {
	candidate := Candidate{
		ID:     strings.TrimSpace(id),
		DNI:    strings.TrimSpace(dni),
		Nombre: strings.TrimSpace(nombre),
		Email:  strings.TrimSpace(email),
	}
	if err := candidate.Validate(); err != nil {
		return Candidate{}, err
	}
	return candidate, nil
}

func (c Candidate) Validate() error {
	switch {
	case strings.TrimSpace(c.ID) == "":
		return ErrCandidateIDRequired
	case strings.TrimSpace(c.DNI) == "":
		return ErrCandidateDNIRequired
	case strings.TrimSpace(c.Nombre) == "":
		return ErrCandidateNombreRequired
	case strings.TrimSpace(c.Email) == "":
		return ErrCandidateEmailRequired
	default:
		return nil
	}
}
