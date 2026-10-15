package ports

import (
	"reflect"
	"slices"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
)

// CriteriosOfrecimientoAplicados fija la política de la selección original.
// Su ausencia corresponde a lecturas anteriores a los criterios por oferta.
type CriteriosOfrecimientoAplicados struct {
	VersionPoliticaOfertas      int64                        `json:"version_politica_ofertas"`
	HuellaPoliticaOfertasSHA256 string                       `json:"huella_politica_ofertas_sha256"`
	ModalidadOfrecida           string                       `json:"modalidad_ofrecida"`
	Configuracion               domain.CriteriosOfrecimiento `json:"configuracion"`
}

func (c *CriteriosOfrecimientoAplicados) Validar() error {
	if c == nil {
		return nil
	}
	if c.VersionPoliticaOfertas < 1 || len(c.HuellaPoliticaOfertasSHA256) != 64 ||
		c.ModalidadOfrecida == "" || c.Configuracion.Validar() != nil {
		return ErrOrigenLlamamientoCTNoDisponible
	}
	for _, b := range c.HuellaPoliticaOfertasSHA256 {
		if !(b >= '0' && b <= '9' || b >= 'a' && b <= 'f') {
			return ErrOrigenLlamamientoCTNoDisponible
		}
	}
	return nil
}

func (c *CriteriosOfrecimientoAplicados) Iguales(otros *CriteriosOfrecimientoAplicados) bool {
	return reflect.DeepEqual(c, otros)
}

// Clonar evita compartir listas o límites entre el borrador y su recibo.
func (c *CriteriosOfrecimientoAplicados) Clonar() *CriteriosOfrecimientoAplicados {
	if c == nil {
		return nil
	}
	copia := *c
	copia.Configuracion.ModalidadesAceptables = slices.Clone(c.Configuracion.ModalidadesAceptables)
	if c.Configuracion.Mejora != nil {
		mejora := *c.Configuracion.Mejora
		mejora.ModalidadesDestino = slices.Clone(mejora.ModalidadesDestino)
		mejora.CondicionesActuales = slices.Clone(mejora.CondicionesActuales)
		for i := range mejora.CondicionesActuales {
			condicion := &mejora.CondicionesActuales[i]
			if condicion.MesesTranscurridos != nil {
				limite := *condicion.MesesTranscurridos
				condicion.MesesTranscurridos = &limite
			}
			if condicion.MesesDuracionInicial != nil {
				limite := *condicion.MesesDuracionInicial
				condicion.MesesDuracionInicial = &limite
			}
		}
		copia.Configuracion.Mejora = &mejora
	}
	return &copia
}
