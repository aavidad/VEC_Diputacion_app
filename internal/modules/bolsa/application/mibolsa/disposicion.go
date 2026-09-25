package mibolsa

import (
	"context"
	"regexp"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var ofertaRefPortal = regexp.MustCompile(`^oferta:[0-9a-f]{64}$`)

// ConRegistroOfertas devuelve una copia del portal que atiende también la
// disposición a ofertas publicadas (Bolsa 000029).
func (p *Portal) ConRegistroOfertas(registro puertosbolsa.RegistroDisposicionOferta) (*Portal, error) {
	if p == nil || nula(registro) {
		return nil, ErrServicioMiBolsaInvalido
	}
	copia := *p
	copia.ofertas = registro
	return &copia, nil
}

// ManifestarDisposicion registra que la persona se ofrece para una oferta
// abierta de su bolsa. El recurso autorizado es la propia oferta; que la
// oferta sea de una bolsa del candidato lo comprueba PostgreSQL al cotejarlo
// con el vínculo del contexto atestado.
func (p *Portal) ManifestarDisposicion(ctx context.Context, orden Orden, oferta, clave string) (puertosbolsa.ReciboDisposicionPortal, error) {
	var vacio puertosbolsa.ReciboDisposicionPortal
	if err := p.listo(ctx); err != nil {
		return vacio, err
	}
	if nula(p.ofertas) {
		return vacio, puertosbolsa.ErrPortalCandidatoNoDisponible
	}
	if !ofertaRefPortal.MatchString(oferta) || !claveIdempotenciaPortal.MatchString(clave) {
		return vacio, puertosbolsa.ErrPortalCandidatoInvalido
	}
	material, candidato, ahora, err := p.autorizarRecurso(ctx, orden, puertosbolsa.AccionManifestarDisposicionPropia, puertosbolsa.AudienciaManifestarDisposicionPropia,
		func(candidato string) dominiovec.RecursoAutorizable {
			return dominiovec.RecursoAutorizable{
				Referencia: oferta, ModuloID: puertosbolsa.ModuloMiBolsa, Tipo: puertosbolsa.TipoRecursoOfertaBolsa,
				Ambitos:   map[string]string{"candidato_ref": candidato},
				Atributos: map[string]string{"propiedad": "candidato"},
			}
		})
	if err != nil {
		return vacio, err
	}
	huella := huellaPortal("disposicion", candidato, oferta, clave)
	return p.ofertas.ManifestarDisposicion(ctx, puertosbolsa.DisposicionPortalCandidato{
		OfertaRef: oferta, ReciboRef: "recibo:disposicion:" + huella, CandidatoRef: candidato, Clave: clave,
		ManifestadaEn: ahora, Material: material,
	})
}
