package bootstrap

import (
	"context"
	"strconv"
	"time"

	denominacion "vec-diputacion-granada/internal/vec/adapters/denominacionpersona"
	"vec-diputacion-granada/internal/vec/ports"
)

// Dominios de derivación dedicados. Una referencia del catálogo nunca puede
// seleccionar la clave de correo, contacto, SMTP o autenticación.
const (
	dominioDenominacionCifrado  = "vec.kms.desarrollo.persona.denominacion.cifrado.v1"
	dominioDenominacionBusqueda = "vec.kms.desarrollo.persona.denominacion.busqueda.v1"
)

type FuenteMetadatosDenominacionPersonaDesarrollo interface {
	CargarMetadatosDenominacionPersona(context.Context) (MetadatosDenominacionPersonaDesarrollo, error)
}

type fuenteClavesDenominacionDesarrollo struct {
	envoltura [32]byte
	fuente    FuenteMetadatosDenominacionPersonaDesarrollo
	norma     ports.NormaDenominacionPersona
}

// ProtectorDenominacionPersonaDesarrollo utiliza el KMS ya cargado sin exponer
// su material. El proveedor vuelve a leer los metadatos en cada operación.
func (c *ComposicionSeguridadDesarrollo) ProtectorDenominacionPersonaDesarrollo(fuente FuenteMetadatosDenominacionPersonaDesarrollo, ahora func() time.Time) (*denominacion.Protector, string, error) {
	if c == nil || c.emisorKMS == nil {
		return nil, "", ErrKMSDesarrolloNoDisponible
	}
	return protectorDenominacionDesdeEnvoltura(c.emisorKMS.claveEnvoltura, fuente, ahora)
}

// NuevoProtectorDenominacionPersonaDesarrollo permite preparar sobres offline
// con el material maestro externo existente. No comprueba Persona ni permisos.
func NuevoProtectorDenominacionPersonaDesarrollo(maestra [32]byte, fuente FuenteMetadatosDenominacionPersonaDesarrollo, ahora func() time.Time) (*denominacion.Protector, string, error) {
	defer clear(maestra[:])
	if maestra == [32]byte{} {
		return nil, "", ErrKMSDesarrolloNoDisponible
	}
	envoltura := derivarClaveDesarrollo(maestra, "vec.kms.desarrollo.envoltura.v1")
	defer clear(envoltura[:])
	return protectorDenominacionDesdeEnvoltura(envoltura, fuente, ahora)
}
func protectorDenominacionDesdeEnvoltura(envoltura [32]byte, fuente FuenteMetadatosDenominacionPersonaDesarrollo, ahora func() time.Time) (*denominacion.Protector, string, error) {
	if envoltura == [32]byte{} || dependenciaContactoKMSNula(fuente) || ahora == nil {
		return nil, "", ErrKMSDesarrolloNoDisponible
	}
	m, err := fuente.CargarMetadatosDenominacionPersona(context.Background())
	if err != nil || m.validar() != nil {
		return nil, "", ErrKMSDesarrolloNoDisponible
	}
	f := &fuenteClavesDenominacionDesarrollo{envoltura: envoltura, fuente: fuente, norma: m.norma()}
	p, err := denominacion.NuevoProtector(f, ahora, f.norma)
	if err != nil {
		clear(f.envoltura[:])
		return nil, "", ErrKMSDesarrolloNoDisponible
	}
	return p, m.Busqueda.referenciaVersionada(), nil
}

func (f *fuenteClavesDenominacionDesarrollo) CargarClavesDenominacionPersona(ctx context.Context) (denominacion.Claves, error) {
	if f == nil || ctx == nil || ctx.Err() != nil || f.envoltura == [32]byte{} {
		return denominacion.Claves{}, ErrKMSDesarrolloNoDisponible
	}
	m, err := f.fuente.CargarMetadatosDenominacionPersona(ctx)
	if err != nil || m.validar() != nil || m.norma() != f.norma {
		return denominacion.Claves{}, ErrKMSDesarrolloNoDisponible
	}
	c := denominacion.Claves{Cifrado: f.clave(m.Cifrado, dominioDenominacionCifrado), Busqueda: f.clave(m.Busqueda, dominioDenominacionBusqueda)}
	for _, retenida := range m.CifradoRetenidas {
		c.CifradoRetenidas = append(c.CifradoRetenidas, f.clave(retenida, dominioDenominacionCifrado))
	}
	return c, nil
}
func (f *fuenteClavesDenominacionDesarrollo) clave(m ClaveDenominacionPersonaDesarrollo, dominio string) denominacion.Clave {
	// Referencia y versión forman coordenadas no ambiguas del mismo proveedor.
	coordenada := dominio + ":" + strconv.Itoa(len(m.Referencia)) + ":" + m.Referencia + ":" + strconv.FormatUint(m.Version, 10)
	return denominacion.Clave{Ref: m.referenciaVersionada(), Material: derivarClaveDesarrollo(f.envoltura, coordenada), Revocada: m.Revocada, RetenerHasta: m.RetenerHasta}
}
