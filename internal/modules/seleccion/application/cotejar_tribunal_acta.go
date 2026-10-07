package application

import (
	"context"
	"errors"
	"reflect"

	basess2 "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

var ErrCotejoTribunalActa = errors.New("seleccion.tribunal_acta.cotejo_invalido")

// CotejarSalidaTribunal enlaza un borrador de acta con la salida local del
// preparador S5. La huella corresponde a los bytes exactos de esa salida.
// El cotejo no acredita procedencia institucional, designación o habilitación.
func CotejarSalidaTribunal(ctx context.Context, salida domain.PreparacionTribunal, huellaSalida string, acta domain.MaterialActaPropuesto) (domain.MaterialActaPropuesto, error) {
	if ctx == nil || !basess2.HuellaValida(huellaSalida) {
		return domain.MaterialActaPropuesto{}, ErrCotejoTribunalActa
	}
	if err := ctx.Err(); err != nil {
		return domain.MaterialActaPropuesto{}, err
	}
	recalculada, err := domain.PrepararTribunal(salida.MaterialPropuesto)
	if err != nil || !reflect.DeepEqual(salida, recalculada) {
		return domain.MaterialActaPropuesto{}, ErrCotejoTribunalActa
	}
	tribunal := salida.MaterialPropuesto
	if acta.AntecedenteTribunal.IdentidadMaterial != tribunal.IdentidadMaterial ||
		acta.AntecedenteTribunal.VersionMaterial != tribunal.VersionMaterial ||
		(acta.AntecedenteTribunal.HuellaAportadaSHA256 != "" && acta.AntecedenteTribunal.HuellaAportadaSHA256 != huellaSalida) {
		return domain.MaterialActaPropuesto{}, ErrCotejoTribunalActa
	}
	faseEncontrada := false
	for _, fase := range tribunal.FasesPropuestas {
		if fase == acta.FasePropuesta {
			faseEncontrada = true
			break
		}
	}
	if !faseEncontrada {
		return domain.MaterialActaPropuesto{}, ErrCotejoTribunalActa
	}
	acta.AntecedenteTribunal.HuellaAportadaSHA256 = huellaSalida
	if err := ctx.Err(); err != nil {
		return domain.MaterialActaPropuesto{}, err
	}
	return acta, nil
}
