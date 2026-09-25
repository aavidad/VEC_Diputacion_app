// Package reglas traduce las reglas del catálogo de Bolsa a las políticas que
// consumen sus casos de uso. Los valores viven en el catálogo; aquí solo se
// nombran claves y atributos.
package reglas

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecreglas "vec-diputacion-granada/internal/vec/reglas"
)

// atributoOrigenContacto nombra el origen al que se aplica la regla b29.
const atributoOrigenContacto = "origen_contacto"

var errReglaContactoOrigen = errors.New("bolsa: regla de contacto de origen no valida")

var _ puertosbolsa.PoliticaOrigenDatosContacto = (*ContactoOrigen)(nil)

// ContactoOrigen traduce b29 (duda 45): vigencia del contacto de origen
// CONVOCA desde su registro en VEC. Es válido con resolutor nulo: responde
// «no configurado» y el alta con origen se rechaza.
type ContactoOrigen struct {
	resolutor *vecreglas.Resolutor
}

func NuevoContactoOrigen(resolutor *vecreglas.Resolutor) *ContactoOrigen {
	return &ContactoOrigen{resolutor: resolutor}
}

// Configurada indica si hay catálogo compuesto.
func (c *ContactoOrigen) Configurada() bool { return c != nil && c.resolutor.Disponible() }

// MarcaOrigenConvoca calcula la vigencia con el cómputo que declara la regla y
// conserva su referencia y la huella del catálogo.
func (c *ContactoOrigen) MarcaOrigenConvoca(ctx context.Context, registradaEn time.Time) (dominiobolsa.MarcaOrigenDatosContacto, error) {
	if !c.Configurada() {
		return dominiobolsa.MarcaOrigenDatosContacto{}, puertosbolsa.ErrOrigenDatosContactoNoConfigurado
	}
	regla, vencimiento, err := c.resolutor.Vencimiento(ctx, vecreglas.BolsaContactoOrigenConvoca, registradaEn, "")
	if err != nil {
		return dominiobolsa.MarcaOrigenDatosContacto{}, errors.Join(puertosbolsa.ErrOrigenDatosContactoNoConfigurado, err)
	}
	if regla.Atributos[atributoOrigenContacto] != dominiobolsa.OrigenDatosContactoConvoca {
		return dominiobolsa.MarcaOrigenDatosContacto{}, errors.Join(puertosbolsa.ErrOrigenDatosContactoNoConfigurado, errReglaContactoOrigen)
	}
	marca := dominiobolsa.MarcaOrigenDatosContacto{
		Origen: dominiobolsa.OrigenDatosContactoConvoca, VigenteHasta: vencimiento.VenceAntesDe.UTC(),
		UltimoDia: vencimiento.UltimoDia, ReglaRef: regla.Referencia, ReglaHuella: regla.HuellaCatalogo,
	}
	if marca.Validar() != nil {
		return dominiobolsa.MarcaOrigenDatosContacto{}, errors.Join(puertosbolsa.ErrOrigenDatosContactoNoConfigurado, errReglaContactoOrigen)
	}
	return marca, nil
}
