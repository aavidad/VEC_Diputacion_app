package ports

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"time"
)

var ErrFronteraAdminTecnicaNoDisponible = errors.New("vec.admin.frontera_tecnica.no_disponible")
var ErrFronteraAdminTecnicaCommitIncierto = errors.New("vec.admin.frontera_tecnica.commit_incierto")

// EventoFronteraAdminTecnica no transporta persona, perfil, sesión, certificado,
// dirección, ruta ni contenido. El LOGIN pertenece al proceso PostgreSQL.
type EventoFronteraAdminTecnica struct {
	TipoRegistro   string `json:"tipo_registro"`
	EventoRef      string `json:"evento_ref"`
	OperadorLogin  string `json:"operador_login"`
	Accion         string `json:"accion"`
	RecursoRef     string `json:"recurso_ref"`
	Resultado      string `json:"resultado"`
	CodigoRef      string `json:"codigo_ref"`
	Proceso        string `json:"proceso"`
	Canal          string `json:"canal"`
	FinalidadRef   string `json:"finalidad_ref"`
	CorrelacionRef string `json:"correlacion_ref"`
}

type AcuseFronteraAdminTecnica struct {
	AuditoriaRef   string    `json:"auditoria_ref"`
	Secuencia      uint64    `json:"secuencia"`
	HuellaSHA256   string    `json:"huella_sha256"`
	CorrelacionRef string    `json:"correlacion_ref"`
	RegistradaEn   time.Time `json:"registrada_en"`
}

type RegistradorFronteraAdminTecnica interface {
	AppendFronteraAdminTecnica(context.Context, EventoFronteraAdminTecnica) (AcuseFronteraAdminTecnica, error)
}

var eventoFronteraTecnica = regexp.MustCompile(`^evento_[0-9a-f]{32}$`)
var correlacionFronteraTecnica = regexp.MustCompile(`^correlacion_[0-9a-f]{32}$`)
var codigoFronteraTecnica = regexp.MustCompile(`^[a-z][a-z0-9_]{1,127}$`)
var procesoFronteraTecnica = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)
var hashFronteraTecnica = regexp.MustCompile(`^[0-9a-f]{64}$`)

// NuevaReferenciaEventoFronteraAdminTecnica identifica una invocación común;
// no es una identidad ni una capacidad de autorización.
func NuevaReferenciaEventoFronteraAdminTecnica() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", errorFronteraAdminTecnica(err)
	}
	return "evento_" + hex.EncodeToString(b[:]), nil
}

func RecursoFronteraAdminTecnica(correlacion string) (string, error) {
	if !correlacionFronteraTecnica.MatchString(correlacion) {
		return "", ErrFronteraAdminTecnicaNoDisponible
	}
	h := sha256.Sum256([]byte("vec.admin.frontera.solicitud.v1\n" + correlacion))
	return "solicitud_admin:" + hex.EncodeToString(h[:16]), nil
}

func errorFronteraAdminTecnica(err error) error {
	if err != nil {
		return ErrFronteraAdminTecnicaNoDisponible
	}
	return nil
}

func (e EventoFronteraAdminTecnica) Validar() error {
	recurso, err := RecursoFronteraAdminTecnica(e.CorrelacionRef)
	if err != nil {
		return errorFronteraAdminTecnica(err)
	}
	if e.TipoRegistro != "frontera_admin_tecnica" || !eventoFronteraTecnica.MatchString(e.EventoRef) || e.OperadorLogin == "" || len(e.OperadorLogin) > 63 || e.Accion != "controlar_frontera_admin_v1" || e.RecursoRef != recurso || (e.Resultado != "denegado" && e.Resultado != "error") || !codigoFronteraTecnica.MatchString(e.CodigoRef) || !procesoFronteraTecnica.MatchString(e.Proceso) || e.Canal != "administracion_privilegiada" || e.FinalidadRef != "control_frontera_admin" {
		return ErrFronteraAdminTecnicaNoDisponible
	}
	return nil
}

func (a AcuseFronteraAdminTecnica) ValidarPara(e EventoFronteraAdminTecnica) error {
	if e.Validar() != nil || a.AuditoriaRef != "aud_v3_fat_"+e.EventoRef[len("evento_"):] || a.Secuencia == 0 || a.Secuencia > 1<<53-1 || !hashFronteraTecnica.MatchString(a.HuellaSHA256) || a.CorrelacionRef != e.CorrelacionRef || a.RegistradaEn.IsZero() || a.RegistradaEn.Year() < 1 || a.RegistradaEn.Year() > 9999 {
		return ErrFronteraAdminTecnicaNoDisponible
	}
	return nil
}
