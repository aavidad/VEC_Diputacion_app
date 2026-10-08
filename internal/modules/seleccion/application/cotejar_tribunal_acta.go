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
func CotejarSalidaTribunal(ctx context.Context, salida domain.PreparacionTribunal, huellaSalida string, acta domain.MaterialActaPropuesto) (domain.PreparacionActa, error) {
	if ctx == nil || !basess2.HuellaValida(huellaSalida) {
		return domain.PreparacionActa{}, ErrCotejoTribunalActa
	}
	if err := ctx.Err(); err != nil {
		return domain.PreparacionActa{}, err
	}
	recalculada, err := domain.PrepararTribunal(salida.MaterialPropuesto)
	if err != nil || !reflect.DeepEqual(salida, recalculada) {
		return domain.PreparacionActa{}, ErrCotejoTribunalActa
	}
	tribunal := salida.MaterialPropuesto
	if acta.AntecedenteTribunal.IdentidadMaterial != tribunal.IdentidadMaterial ||
		acta.AntecedenteTribunal.VersionMaterial != tribunal.VersionMaterial ||
		(acta.AntecedenteTribunal.HuellaAportadaSHA256 != "" && acta.AntecedenteTribunal.HuellaAportadaSHA256 != huellaSalida) {
		return domain.PreparacionActa{}, ErrCotejoTribunalActa
	}
	faseEncontrada := false
	for _, fase := range tribunal.FasesPropuestas {
		if fase == acta.FasePropuesta {
			faseEncontrada = true
			break
		}
	}
	if !faseEncontrada {
		return domain.PreparacionActa{}, ErrCotejoTribunalActa
	}
	acta.AntecedenteTribunal.HuellaAportadaSHA256 = huellaSalida
	preparacion, err := PrepararMaterialActa(ctx, acta)
	if err != nil {
		return domain.PreparacionActa{}, err
	}
	var antecedente, fase bool
	for i := range preparacion.Pendientes {
		switch preparacion.Pendientes[i].Campo {
		case "antecedente_tribunal":
			if preparacion.Pendientes[i].Codigo != "antecedente_no_cotejado" {
				return domain.PreparacionActa{}, ErrCotejoTribunalActa
			}
			preparacion.Pendientes[i].Codigo = "antecedente_cotejado_local"
			antecedente = true
		case "fase_propuesta":
			if preparacion.Pendientes[i].Codigo != "pertenencia_no_verificada" {
				return domain.PreparacionActa{}, ErrCotejoTribunalActa
			}
			preparacion.Pendientes[i].Codigo = "fase_cotejada_local"
			fase = true
		}
	}
	if !antecedente || !fase {
		return domain.PreparacionActa{}, ErrCotejoTribunalActa
	}
	return preparacion, nil
}
