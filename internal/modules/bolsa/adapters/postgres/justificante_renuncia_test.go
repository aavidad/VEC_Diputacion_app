package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestPreparacionJustificanteDecodificaContratoD22(t *testing.T) {
	var respuesta struct {
		bolsaports.ActoJustificanteRenuncia
		DecisionRef  string `json:"decision_ref"`
		AuditoriaRef string `json:"auditoria_ref"`
	}
	contenido := []byte(`{"recurso_ref":"justificante-renuncia:` + string64('a') + `",` +
		`"version_dueno":1,"participacion_ref":"participacion:1","llamamiento_ref":"llamamiento:1",` +
		`"candidato_ref":"can_123","persona_ref":"per_123","decision_ref":"decision:1","auditoria_ref":"auditoria:1"}`)
	if err := decodificarPreparacionJustificante(contenido, &respuesta); err != nil {
		t.Fatal(err)
	}
	if respuesta.RecursoRef != "justificante-renuncia:"+string64('a') ||
		respuesta.VersionDueno != 1 || respuesta.ParticipacionRef != "participacion:1" ||
		respuesta.LlamamientoRef != "llamamiento:1" || respuesta.PersonaRef != "per_123" ||
		respuesta.DecisionRef != "decision:1" || respuesta.AuditoriaRef != "auditoria:1" {
		t.Fatalf("contrato D22 incompleto: %+v", respuesta)
	}
	if err := decodificarPreparacionJustificante(append(contenido[:len(contenido)-1], []byte(`,"extra":"x"}`)...), &respuesta); err == nil {
		t.Fatal("aceptó un campo no pactado")
	}
}

func TestPreparacionRRHHDecodificaContratoD22(t *testing.T) {
	var respuesta struct {
		bolsaports.JustificanteRenunciaRRHH
		DecisionRef  string `json:"decision_ref"`
		AuditoriaRef string `json:"auditoria_ref"`
	}
	contenido := []byte(`{"documento_ref":"ref:` + string64('a') + `",` +
		`"documento_sha256":"` + string64('b') + `","recurso_ref":"justificante-renuncia:` + string64('c') + `",` +
		`"persona_titular_ref":"per_123","decision_ref":"decision:2","auditoria_ref":"auditoria:2"}`)
	if err := decodificarPreparacionJustificante(contenido, &respuesta); err != nil {
		t.Fatal(err)
	}
	if respuesta.DocumentoRef != "ref:"+string64('a') || respuesta.DocumentoSHA256 != string64('b') ||
		respuesta.RecursoRef != "justificante-renuncia:"+string64('c') ||
		respuesta.PersonaTitularRef != "per_123" || respuesta.DecisionRef != "decision:2" ||
		respuesta.AuditoriaRef != "auditoria:2" {
		t.Fatalf("contrato RRHH D22 incompleto: %+v", respuesta)
	}
}

func TestPreparacionJustificanteDistingueDenegacionYFallo(t *testing.T) {
	if err := errorJustificanteRenuncia(t.Context(), &pgconn.PgError{Code: "42501"}); !errors.Is(err, bolsaports.ErrJustificanteRenunciaDenegado) {
		t.Fatalf("denegación perdida: %v", err)
	}
	if err := errorJustificanteRenuncia(t.Context(), &pgconn.PgError{Code: "55000"}); !errors.Is(err, bolsaports.ErrJustificanteRenunciaNoDisponible) {
		t.Fatalf("fallo técnico confundido con denegación: %v", err)
	}
}

func string64(caracter byte) string {
	b := make([]byte, 64)
	for i := range b {
		b[i] = caracter
	}
	return string(b)
}
