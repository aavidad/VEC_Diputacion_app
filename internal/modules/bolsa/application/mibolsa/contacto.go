package mibolsa

import (
	"context"
	"strconv"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ConRegistroContacto devuelve una copia del portal que atiende también la
// confirmación del contacto propio (AD3-86 y Bolsa 000040).
func (p *Portal) ConRegistroContacto(registro puertosbolsa.RegistroConfirmacionContacto) (*Portal, error) {
	if p == nil || nula(registro) {
		return nil, ErrServicioMiBolsaInvalido
	}
	copia := *p
	copia.contacto = registro
	return &copia, nil
}

// ConfirmarContacto registra que la versión del contacto que la persona vio
// en su bolsa sigue siendo suya. PostgreSQL rechaza la confirmación si RRHH
// registró otra versión entre medias.
func (p *Portal) ConfirmarContacto(ctx context.Context, orden Orden, bolsa string, version int64, clave string) (puertosbolsa.ReciboConfirmacionContacto, error) {
	var vacio puertosbolsa.ReciboConfirmacionContacto
	if err := p.listo(ctx); err != nil {
		return vacio, err
	}
	if nula(p.contacto) {
		return vacio, puertosbolsa.ErrPortalCandidatoNoDisponible
	}
	if !bolsaRefPortal.MatchString(bolsa) || version < 1 || !claveIdempotenciaPortal.MatchString(clave) {
		return vacio, puertosbolsa.ErrPortalCandidatoInvalido
	}
	material, candidato, ahora, err := p.autorizar(ctx, orden, puertosbolsa.AccionConfirmarContactoPropio, puertosbolsa.AudienciaConfirmarContactoPropio, bolsa)
	if err != nil {
		return vacio, err
	}
	huella := huellaPortal("confirmacion-contacto", candidato, bolsa, strconv.FormatInt(version, 10), clave)
	return p.contacto.ConfirmarContacto(ctx, puertosbolsa.ConfirmacionContactoPortal{
		CandidatoRef: candidato, Bolsa: bolsa, Version: version, Clave: clave,
		ReciboRef: "recibo:confirmacion-contacto:" + huella, ConfirmadaEn: ahora, Material: material,
	})
}
