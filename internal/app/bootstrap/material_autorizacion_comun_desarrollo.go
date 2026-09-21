package bootstrap

import "errors"

var errMaterialAutorizacionComunDesarrolloNoDisponible = errors.New("vec: material de autorización común no disponible")

// descriptorMaterialConsumidorV3Desarrollo es una declaración inmutable de
// composición. El núcleo no deriva claves, publica SQL ni conoce los tipos de
// ningún módulo: entrega sólo la identidad nominal del proveedor autorizado.
type descriptorMaterialConsumidorV3Desarrollo struct {
	Audiencia, Dominio, Prefijo, ProveedorNominal string
}

type catalogoMaterialAutorizacionComunDesarrollo struct {
	porAudiencia map[string]descriptorMaterialConsumidorV3Desarrollo
	porDominio   map[string]struct{}
	porPrefijo   map[string]struct{}
}

func nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptores []descriptorMaterialConsumidorV3Desarrollo) (catalogoMaterialAutorizacionComunDesarrollo, error) {
	c := catalogoMaterialAutorizacionComunDesarrollo{
		porAudiencia: make(map[string]descriptorMaterialConsumidorV3Desarrollo, len(descriptores)),
		porDominio:   make(map[string]struct{}, len(descriptores)),
		porPrefijo:   make(map[string]struct{}, len(descriptores)),
	}
	for _, d := range descriptores {
		if d.Audiencia == "" || d.Dominio == "" || d.Prefijo == "" || d.ProveedorNominal == "" {
			return catalogoMaterialAutorizacionComunDesarrollo{}, errMaterialAutorizacionComunDesarrolloNoDisponible
		}
		if _, existe := c.porAudiencia[d.Audiencia]; existe {
			return catalogoMaterialAutorizacionComunDesarrollo{}, errMaterialAutorizacionComunDesarrolloNoDisponible
		}
		if _, existe := c.porDominio[d.Dominio]; existe {
			return catalogoMaterialAutorizacionComunDesarrollo{}, errMaterialAutorizacionComunDesarrolloNoDisponible
		}
		if _, existe := c.porPrefijo[d.Prefijo]; existe {
			return catalogoMaterialAutorizacionComunDesarrollo{}, errMaterialAutorizacionComunDesarrolloNoDisponible
		}
		c.porAudiencia[d.Audiencia] = d
		c.porDominio[d.Dominio] = struct{}{}
		c.porPrefijo[d.Prefijo] = struct{}{}
	}
	return c, nil
}

func (c catalogoMaterialAutorizacionComunDesarrollo) descriptorPara(audiencia string) (descriptorMaterialConsumidorV3Desarrollo, bool) {
	d, ok := c.porAudiencia[audiencia]
	return d, ok
}
