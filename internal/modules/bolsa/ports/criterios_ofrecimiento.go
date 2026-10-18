package ports

import (
	"reflect"
	"regexp"
	"slices"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
)

var claveCausaOfrecida = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)

// CriteriosOfrecimientoAplicados fija la política de la selección original.
// Su ausencia corresponde a lecturas anteriores a los criterios por oferta.
type CriteriosOfrecimientoAplicados struct {
	VersionPoliticaOfertas      int64                        `json:"version_politica_ofertas"`
	HuellaPoliticaOfertasSHA256 string                       `json:"huella_politica_ofertas_sha256"`
	ModalidadOfrecida           string                       `json:"modalidad_ofrecida"`
	CausaOfrecida               string                       `json:"causa_ofrecida"`
	Configuracion               domain.CriteriosOfrecimiento `json:"configuracion"`
}

func (c *CriteriosOfrecimientoAplicados) Validar() error {
	if c == nil {
		return nil
	}
	if c.VersionPoliticaOfertas < 1 || len(c.HuellaPoliticaOfertasSHA256) != 64 ||
		c.ModalidadOfrecida == "" || c.Configuracion.Validar() != nil ||
		(c.CausaOfrecida != "" && !claveCausaOfrecida.MatchString(c.CausaOfrecida)) {
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
	if c.Configuracion.Mejora != nil {
		mejora := *c.Configuracion.Mejora
		mejora.ModalidadesDestino = slices.Clone(mejora.ModalidadesDestino)
		mejora.CausasDestino = slices.Clone(mejora.CausasDestino)
		mejora.SupuestosReduccionVacante = slices.Clone(mejora.SupuestosReduccionVacante)
		mejora.ReduccionPorModalidadesPrevias = slices.Clone(mejora.ReduccionPorModalidadesPrevias)
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
