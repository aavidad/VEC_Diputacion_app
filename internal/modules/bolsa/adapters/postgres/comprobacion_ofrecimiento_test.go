package postgres

import (
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

func TestReciboComprobacionAntesDeCommit(t *testing.T) {
	q := ports.SolicitudComprobacion{VersionEsperada: 2, Contexto: ports.ContextoComprobacion{VersionPoliticaOfertas: 7}}
	ref := "comprobacion:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	r := ports.ReciboComprobacion{ComprobacionRef: ref, Version: 3, VersionEsperada: 2, VersionPoliticaOfertas: 7, Resultado: "pendiente", MotivoClave: "antecedentes_incompletos", ReciboRef: "recibo:" + ref, EventoRef: "evento:" + ref, AuditoriaRef: "auditoria:001", RegistradaEn: time.Date(2026, 10, 18, 8, 0, 0, 123000, time.UTC)}
	if !reciboComprobacionValido(r, q) {
		t.Fatal("recibo válido rechazado")
	}
	r.Reutilizada = true
	if !reciboComprobacionValido(r, q) {
		t.Fatal("replay original rechazado")
	}
	r.VersionPoliticaOfertas = 8
	if reciboComprobacionValido(r, q) {
		t.Fatal("recibo de política ajena aceptado")
	}
	r.VersionPoliticaOfertas = 7
	r.ReciboRef = "recibo:otro"
	if reciboComprobacionValido(r, q) {
		t.Fatal("recibo desacoplado aceptado")
	}
}

func TestErroresComprobacionSQL(t *testing.T) {
	for _, c := range []struct {
		code string
		want error
	}{{"VBC01", ports.ErrComprobacionConflicto}, {"42501", vd.ErrAutorizacionDenegada}, {"22023", ports.ErrComprobacionInvalida}, {"VA172", ports.ErrComprobacionDependencia}} {
		err := errorComprobacionSQL(&pgconn.PgError{Code: c.code})
		if !errors.Is(err, c.want) {
			t.Errorf("%s: %v", c.code, err)
		}
	}
	for _, c := range []struct {
		nombre     string
		err        error
		reintentar bool
	}{
		{"serialización", &pgconn.PgError{Code: "40001"}, true},
		{"interbloqueo", &pgconn.PgError{Code: "40P01"}, true},
		{"denegación", &pgconn.PgError{Code: "42501"}, false},
		{"origen ausente", &pgconn.PgError{Code: "VA172"}, false},
		{"conflicto funcional", &pgconn.PgError{Code: "VBC01"}, false},
		{"RAISE de función", &pgconn.PgError{Code: "40001", Routine: "exec_stmt_raise"}, false},
		{"commit incierto sin SQLSTATE", errors.New("resultado de COMMIT desconocido"), false},
	} {
		if got := reintentarComprobacion(c.err); got != c.reintentar {
			t.Errorf("%s: reintentar=%t", c.nombre, got)
		}
	}
}

func TestIntencionComprobacionLigada(t *testing.T) {
	q := ports.SolicitudComprobacion{Contexto: ports.ContextoComprobacion{ParticipacionRef: "participacion:uno"}, Hechos: map[string]ports.HechoComprobacion{}, ClaveIdempotencia: "clave:12345678"}
	b, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	i := ports.IntencionComprobacion{Solicitud: q, CanonPostgreSQL: string(b)}
	if !intencionComprobacionLigada(i) {
		t.Fatal("intención original rechazada")
	}
	i.Solicitud.Contexto.ParticipacionRef = "participacion:otra"
	if intencionComprobacionLigada(i) {
		t.Fatal("intención alterada aceptada")
	}
}
