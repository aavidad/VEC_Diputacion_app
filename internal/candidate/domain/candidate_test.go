package domain

import (
	"errors"
	"testing"
)

func TestNewCandidateTrimsFields(t *testing.T) {
	candidate, err := NewCandidate(" cand-1 ", " 12345678A ", " Ana Perez ", " ana@example.test ")
	if err != nil {
		t.Fatalf("NewCandidate() error = %v", err)
	}

	if candidate.ID != "cand-1" {
		t.Fatalf("ID = %q, want %q", candidate.ID, "cand-1")
	}
	if candidate.DNI != "12345678A" {
		t.Fatalf("DNI = %q, want %q", candidate.DNI, "12345678A")
	}
	if candidate.Nombre != "Ana Perez" {
		t.Fatalf("Nombre = %q, want %q", candidate.Nombre, "Ana Perez")
	}
	if candidate.Email != "ana@example.test" {
		t.Fatalf("Email = %q, want %q", candidate.Email, "ana@example.test")
	}
}

func TestCandidateValidateRequiredFields(t *testing.T) {
	valid := Candidate{
		ID:     "cand-1",
		DNI:    "12345678A",
		Nombre: "Ana Perez",
		Email:  "ana@example.test",
	}
	tests := []struct {
		name      string
		candidate Candidate
		wantErr   error
	}{
		{
			name:      "id required",
			candidate: Candidate{DNI: valid.DNI, Nombre: valid.Nombre, Email: valid.Email},
			wantErr:   ErrCandidateIDRequired,
		},
		{
			name:      "dni required",
			candidate: Candidate{ID: valid.ID, Nombre: valid.Nombre, Email: valid.Email},
			wantErr:   ErrCandidateDNIRequired,
		},
		{
			name:      "nombre required",
			candidate: Candidate{ID: valid.ID, DNI: valid.DNI, Email: valid.Email},
			wantErr:   ErrCandidateNombreRequired,
		},
		{
			name:      "email required",
			candidate: Candidate{ID: valid.ID, DNI: valid.DNI, Nombre: valid.Nombre},
			wantErr:   ErrCandidateEmailRequired,
		},
		{
			name:      "blank strings rejected",
			candidate: Candidate{ID: " ", DNI: valid.DNI, Nombre: valid.Nombre, Email: valid.Email},
			wantErr:   ErrCandidateIDRequired,
		},
		{
			name:      "valid candidate",
			candidate: valid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.candidate.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
