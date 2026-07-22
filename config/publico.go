package config

import (
	"strings"

	"vec-diputacion-granada/internal/shared/limiteshttp"
)

// NormalizePublicTransport aplica únicamente valores de transporte seguros.
// Deliberadamente no toca rutas, catálogos, credenciales, adaptadores ni
// valores de demostración: el binario anónimo no debe enlazarlos por accidente.
func (c Config) NormalizePublicTransport() Config {
	if strings.TrimSpace(c.Address) == "" {
		c.Address = DefaultAddress
	} else {
		c.Address = strings.TrimSpace(c.Address)
	}
	if c.ReadHeaderTimeout <= 0 {
		c.ReadHeaderTimeout = DefaultReadHeaderLimit
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = DefaultReadTimeout
	}
	// El handler público reparte exactamente este presupuesto entre operación,
	// reversión y escritura. Aceptar un valor menor cortaría la respuesta antes
	// de la limpieza; uno mayor ampliaría la ventana de clientes lentos.
	c.WriteTimeout = limiteshttp.DuracionMaximaPeticionPublica
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = DefaultIdleTimeout
	}
	if c.MaxHeaderBytes <= 0 {
		c.MaxHeaderBytes = DefaultMaxHeaderBytes
	}
	if c.MaxRequestBodyBytes <= 0 {
		c.MaxRequestBodyBytes = DefaultMaxRequestBodyBytes
	}
	c.HTTPAllowedCIDRs = normalizeCIDRs(c.HTTPAllowedCIDRs)
	c.TLSCertFile = strings.TrimSpace(c.TLSCertFile)
	c.TLSKeyFile = strings.TrimSpace(c.TLSKeyFile)
	c.AuthMode = strings.ToLower(strings.TrimSpace(c.AuthMode))
	c.ExecutionProfile = strings.ToLower(strings.TrimSpace(c.ExecutionProfile))
	return c
}
