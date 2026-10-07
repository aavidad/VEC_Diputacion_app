// Package reglasadjudicacion gobierna la política de ejemplo de ofertas de
// Bolsa. La publicación es durable; la decisión administrativa la confirma
// RRHH mediante el servicio de ofertas existente.
package reglasadjudicacion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
)

var (
	bolsaRef = regexp.MustCompile(`^bolsa:[A-Za-z0-9:_-]{1,250}$`)
	clave    = regexp.MustCompile(`^[A-Za-z0-9:_-]{8,256}$`)
)

type Servicio struct {
	repositorio ports.RepositorioPoliticaOfertas
	calendarios calendariosports.ConsultaCalendarios
}

func NuevoServicio(repo ports.RepositorioPoliticaOfertas, calendarios calendariosports.ConsultaCalendarios) (*Servicio, error) {
	if repo == nil {
		return nil, ports.ErrPoliticaOfertasNoDisponible
	}
	return &Servicio{repositorio: repo, calendarios: calendarios}, nil
}

func (s *Servicio) Vigente(ctx context.Context, bolsa string) (ports.VersionPoliticaOfertas, error) {
	if s == nil || ctx == nil || !bolsaRef.MatchString(bolsa) {
		return ports.VersionPoliticaOfertas{}, domain.ErrPoliticaOfertasInvalida
	}
	v, err := s.repositorio.Vigente(ctx, bolsa)
	if err != nil {
		return ports.VersionPoliticaOfertas{}, err
	}
	if v.Version == 0 {
		return ports.VersionPoliticaOfertas{BolsaRef: bolsa, Ejemplo: true, Configurada: false}, nil
	}
	if !v.Configurada || !v.Ejemplo || v.Politica == nil || v.Politica.Validar() != nil ||
		v.BolsaRef != bolsa || len(v.HuellaSHA256) != 64 {
		return ports.VersionPoliticaOfertas{}, ports.ErrPoliticaOfertasNoDisponible
	}
	return v, nil
}

// ConsultarAutorizada es la única lectura para RRHH. El repositorio consume
// AD3-97 y registra actor, decisión, auditoría y versión en la misma
// transacción que obtiene la política. Vigente queda para el cálculo interno.
func (s *Servicio) ConsultarAutorizada(ctx context.Context, c ports.ConsultaPoliticaOfertasAutorizada) (ports.VersionPoliticaOfertas, error) {
	if s == nil || ctx == nil || !bolsaRef.MatchString(c.BolsaRef) || c.Material.ValidarEstructura() != nil {
		return ports.VersionPoliticaOfertas{}, ports.ErrPoliticaOfertasNoDisponible
	}
	v, err := s.repositorio.ConsultarAutorizada(ctx, c)
	if err != nil {
		return ports.VersionPoliticaOfertas{}, err
	}
	if v.BolsaRef != c.BolsaRef || !v.Ejemplo ||
		(v.Version == 0 && (v.Configurada || v.Politica != nil)) ||
		(v.Version > 0 && (!v.Configurada || v.Politica == nil || v.Politica.Validar() != nil || len(v.HuellaSHA256) != 64)) {
		return ports.VersionPoliticaOfertas{}, ports.ErrPoliticaOfertasNoDisponible
	}
	return v, nil
}

// Publicar admite la forma histórica solo para recuperar una operación: el
// repositorio distingue atómicamente el replay de una versión nueva y exige
// el inicio explícito para esta última, sin alterar el recibo histórico.
func (s *Servicio) Publicar(ctx context.Context, c ports.ComandoPublicarPoliticaOfertas) (ports.VersionPoliticaOfertas, error) {
	if s == nil || ctx == nil || !bolsaRef.MatchString(c.BolsaRef) || c.VersionEsperada < 0 ||
		!clave.MatchString(c.ClaveIdempotencia) || c.ActorRef == "" ||
		c.Politica.Validar() != nil || c.Material.ValidarEstructura() != nil {
		return ports.VersionPoliticaOfertas{}, domain.ErrPoliticaOfertasInvalida
	}
	if c.ReciboRef == "" {
		h := sha256.Sum256([]byte(c.BolsaRef + "\x1f" + c.ActorRef + "\x1f" + c.ClaveIdempotencia))
		c.ReciboRef = "recibo:politica-ofertas:" + hex.EncodeToString(h[:])
	}
	v, err := s.repositorio.Publicar(ctx, c)
	if err != nil {
		return ports.VersionPoliticaOfertas{}, err
	}
	if v.BolsaRef != c.BolsaRef || !v.Configurada || v.Politica == nil || v.Politica.Validar() != nil || v.Version < 1 {
		return ports.VersionPoliticaOfertas{}, ports.ErrPoliticaOfertasNoDisponible
	}
	return v, nil
}

// PlazoDisposicion toma la versión vigente y delega el cálculo al módulo
// Calendarios. El recibo de la oferta congela versión, huella y calendarios
// usados; editar la política más tarde no reescribe ofertas anteriores.
func (s *Servicio) PlazoDisposicionBolsa(ctx context.Context, bolsa string, notificada time.Time) (ports.PlazoOferta, time.Time, error) {
	if s == nil || notificada.IsZero() {
		return ports.PlazoOferta{}, time.Time{}, ports.ErrPlazoOfertaNoConfigurado
	}
	v, err := s.Vigente(ctx, bolsa)
	if err != nil {
		return ports.PlazoOferta{}, time.Time{}, err
	}
	if !v.Configurada || v.Politica == nil || v.Politica.ValidarParaOfertasNuevas() != nil {
		return ports.PlazoOferta{}, time.Time{}, ports.ErrPlazoOfertaNoConfigurado
	}
	p := v.Politica.Plazo
	if p.Unidad == "horas_naturales" {
		apertura := notificada.UTC().Truncate(time.Microsecond)
		vence := apertura.Add(time.Duration(p.Cantidad) * time.Hour)
		madrid, err := time.LoadLocation("Europe/Madrid")
		if err != nil {
			return ports.PlazoOferta{}, time.Time{}, ports.ErrOfertaNoDisponible
		}
		return ports.PlazoOferta{
			ReglaRef:       fmt.Sprintf("politica-ofertas:%s:%d", bolsa, v.Version),
			HuellaCatalogo: v.HuellaSHA256, Unidad: p.Unidad, Cantidad: p.Cantidad,
			Computo: p.Computo, UltimoDia: vence.Add(-time.Nanosecond).In(madrid).Format(time.DateOnly),
			Ejemplo: true, Calendarios: []string{"calendario:utc-continuo:v1"},
			PoliticaVersion: v.Version, MunicipioSede: p.MunicipioSede,
			AperturaEn: apertura.Format("2006-01-02T15:04:05.000000Z"),
			VenceEn:    vence.Format("2006-01-02T15:04:05.000000Z"),
		}, vence, nil
	}
	if s.calendarios == nil {
		return ports.PlazoOferta{}, time.Time{}, ports.ErrOfertaNoDisponible
	}
	resultado, err := s.calendarios.CalcularPlazo(ctx, calendariosports.SolicitudCalculoPlazo{
		NotificadoEn: notificada.UTC(), Unidad: calendariosdomain.UnidadPlazo(p.Unidad),
		Cantidad: p.Cantidad, MunicipioSede: "municipio:ine:" + p.MunicipioSede,
	})
	if err != nil || !resultado.VenceAntesDe.After(notificada) {
		return ports.PlazoOferta{}, time.Time{}, errors.Join(ports.ErrOfertaNoDisponible, err)
	}
	versiones := make([]string, 0, len(resultado.VersionesUtilizadas))
	for _, version := range resultado.VersionesUtilizadas {
		versiones = append(versiones, version.ID)
	}
	return ports.PlazoOferta{
		ReglaRef:       fmt.Sprintf("politica-ofertas:%s:%d", bolsa, v.Version),
		HuellaCatalogo: v.HuellaSHA256, Unidad: p.Unidad, Cantidad: p.Cantidad,
		Computo: p.Computo, UltimoDia: resultado.Vencimiento.String(),
		Ejemplo: true, Calendarios: versiones, PoliticaVersion: v.Version,
		MunicipioSede: p.MunicipioSede,
	}, resultado.VenceAntesDe, nil
}

// La firma histórica no contiene bolsa; usarla sobre este catálogo sería
// ambiguo. El servicio de ofertas invoca PlazoDisposicionBolsa.
func (s *Servicio) PlazoDisposicion(context.Context, time.Time) (ports.PlazoOferta, time.Time, error) {
	return ports.PlazoOferta{}, time.Time{}, ports.ErrPlazoOfertaNoConfigurado
}
