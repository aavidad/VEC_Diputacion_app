package mibolsa

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ComandoSolicitudDocumentalPortal contiene únicamente lo declarado por la
// persona. Su identidad, vínculo y fecha de registro llegan de la frontera.
type ComandoSolicitudDocumentalPortal struct {
	Bolsa, DocumentoRef, DocumentoSHA256 string
	FechaFinCausa, Clave                 string
}

// PresentarSolicitudDocumental deja una petición pendiente de RRHH. No
// cambia la situación de la participación ni acredita custodia del fichero.
func (p *Portal) PresentarSolicitudDocumental(ctx context.Context, orden Orden, c ComandoSolicitudDocumentalPortal) (puertosbolsa.ReciboSolicitudDocumentalPortal, error) {
	var vacio puertosbolsa.ReciboSolicitudDocumentalPortal
	if err := p.listo(ctx); err != nil {
		return vacio, err
	}
	if !bolsaRefPortal.MatchString(c.Bolsa) || !justificanteRefPortal.MatchString(c.DocumentoRef) ||
		!huellaSHA256Portal.MatchString(c.DocumentoSHA256) || !claveIdempotenciaPortal.MatchString(c.Clave) {
		return vacio, puertosbolsa.ErrPortalCandidatoInvalido
	}
	if (dominiobolsa.JustificanteOperacionSituacion{Tipo: dominiobolsa.JustificanteSolicitudCandidato,
		Referencia: c.DocumentoRef, SHA256: c.DocumentoSHA256}).Validar() != nil {
		return vacio, puertosbolsa.ErrPortalCandidatoInvalido
	}
	if c.FechaFinCausa != "" {
		fecha, err := time.Parse(time.DateOnly, c.FechaFinCausa)
		if err != nil || fecha.Format(time.DateOnly) != c.FechaFinCausa {
			return vacio, puertosbolsa.ErrPortalCandidatoInvalido
		}
	}
	material, candidato, ahora, err := p.autorizar(ctx, orden, puertosbolsa.AccionPresentarSolicitudDocumentalPropia, puertosbolsa.AudienciaPresentarSolicitudDocumentalPropia, c.Bolsa)
	if err != nil {
		return vacio, err
	}
	huella := huellaPortal("solicitud-documental", candidato, c.Bolsa, c.Clave)
	solicitud := puertosbolsa.SolicitudDocumentalPortal{
		SolicitudRef:    "solicitud-documental:" + huella,
		ReciboRef:       "recibo:solicitud-documental:" + huellaPortal("recibo", huella),
		ContenidoSHA256: huellaPortal("contenido-solicitud-documental", candidato, c.Bolsa, c.DocumentoRef, c.DocumentoSHA256, c.FechaFinCausa),
		CandidatoRef:    candidato, Bolsa: c.Bolsa,
		DocumentoRef: c.DocumentoRef, DocumentoSHA256: c.DocumentoSHA256,
		FechaFinCausa: c.FechaFinCausa, Clave: c.Clave,
		RegistradaEn: ahora, Material: material,
	}
	recibo, err := p.registro.SolicitarDocumentalPortal(ctx, solicitud)
	if err != nil {
		return vacio, err
	}
	if recibo.SolicitudRef != solicitud.SolicitudRef || recibo.ReciboRef != solicitud.ReciboRef ||
		recibo.ContenidoSHA256 != solicitud.ContenidoSHA256 || recibo.Version != 1 ||
		(!recibo.Reutilizada && recibo.Estado != "pendiente_rrhh") ||
		(recibo.Reutilizada && recibo.Estado != "pendiente_rrhh" && recibo.Estado != "validada" && recibo.Estado != "rechazada") ||
		recibo.RegistradaEn.IsZero() {
		return vacio, errors.Join(puertosbolsa.ErrPortalCandidatoNoDisponible, puertosbolsa.ErrPortalCandidatoInvalido)
	}
	return recibo, nil
}
