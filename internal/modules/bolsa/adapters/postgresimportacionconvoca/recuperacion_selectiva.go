package postgresimportacionconvoca

import (
	"context"

	"vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

// DescifrarFilasSeleccionadas recibe exclusivamente el resultado protegido
// que B84 obtuvo de BIC6 dentro de la transacción nominal. No abre una segunda
// consulta ni lee el resto del acta. El llamador comprueba la página y borra
// la referencia al resultado antes de responder.
func (r *RepositorioRecuperacionPostgreSQL) DescifrarFilasSeleccionadas(ctx context.Context, crudo []byte, huella, categoria string, numeros []int) ([]importacionconvoca.FilaAceptada, error) {
	if r == nil || valorNulo(r.protector) || ctx == nil || ctx.Err() != nil ||
		len(crudo) == 0 || len(crudo) > 12<<20 || len(numeros) == 0 || len(numeros) > 5000 {
		return nil, ErrRepositorioNoDisponible
	}
	var pagina loteRecuperadoPostgreSQL
	if decodificarJSONExacto(crudo, &pagina) != nil || pagina.SiguienteNumero != nil {
		return nil, ErrResultadoNoConfiable
	}
	estado, err := restaurarEstado(pagina.Estado)
	if err != nil || estado.Acta.HuellaFicheroSHA256 != huella || estado.Acta.CategoriaRef != categoria ||
		estado.EstadoStaging == "expurgado" || len(pagina.Filas) != len(numeros) {
		return nil, ErrResultadoNoConfiable
	}
	protegidas, err := restaurarFilasProtegidas(pagina.Filas)
	if err != nil {
		return nil, ErrResultadoNoConfiable
	}
	defer borrarFilasProtegidas(protegidas)
	if validarFilasProtegidas(protegidas) != nil {
		return nil, ErrResultadoNoConfiable
	}
	for i, fila := range protegidas {
		if numeros[i] != fila.Numero {
			return nil, ErrResultadoNoConfiable
		}
	}
	filasParaProtector := clonarFilasProtegidas(protegidas)
	defer borrarFilasProtegidas(filasParaProtector)
	recuperadas, err := r.protector.RecuperarStaging(ctx, SolicitudRecuperacionStaging{
		ImportacionRef:      estado.Acta.ImportacionRef,
		HuellaFicheroSHA256: estado.Acta.HuellaFicheroSHA256,
		Esquema:             estado.Acta.Esquema,
		Filas:               filasParaProtector,
	})
	if err != nil || validarCorrespondenciaRecuperada(protegidas, recuperadas, estado.Acta.Esquema) != nil {
		return nil, ErrMaterialNoConfiable
	}
	return clonarFilasDominio(recuperadas), nil
}
