package postgresimportacionconvoca

import (
	"context"
	"errors"
	"math"
	"sort"

	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

const (
	maximoFilasRecuperacionSelectiva = 101 // página de 100 más su turno
	maximoBytesFilasSelectivas       = 4_194_000
	maximoJSONRecuperacionSelectiva  = maximoBytesJSONActa + 2*maximoBytesFilasSelectivas + (1 << 20)
)

// DescifrarFilasSeleccionadas acepta únicamente el JSON cifrado devuelto por
// la consulta propietaria después de que el llamador haya confirmado su
// transacción nominal. El JSON no es una autorización y nunca se toma del
// cuerpo HTTP. El llamador conserva la obligación de borrar crudo al terminar.
// Esta función no abre conexiones, no confirma transacciones ni reconstruye
// las filas del acta que no formaron parte de la selección autorizada.
func (r *RepositorioRecuperacionPostgreSQL) DescifrarFilasSeleccionadas(
	ctx context.Context, crudo []byte, huella, categoria string, numeros []int,
) ([]dominio.FilaAceptada, error) {
	if r == nil || valorNulo(r.protector) || ctx == nil {
		return nil, ErrRepositorioNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ordenados, err := ordenarNumerosSeleccionados(numeros)
	if err != nil {
		return nil, err
	}
	if len(crudo) < 2 || len(crudo) > maximoJSONRecuperacionSelectiva {
		return nil, ErrResultadoNoConfiable
	}
	var pagina loteRecuperadoPostgreSQL
	if decodificarJSONExacto(crudo, &pagina) != nil || pagina.SiguienteNumero != nil {
		return nil, ErrResultadoNoConfiable
	}
	estado, err := restaurarEstado(pagina.Estado)
	if err != nil || estado.Acta.HuellaFicheroSHA256 != huella ||
		estado.Acta.CategoriaRef != categoria {
		return nil, ErrResultadoNoConfiable
	}
	if estado.EstadoStaging == aplicacion.EstadoStagingExpurgado {
		return nil, aplicacion.ErrStagingExpurgado
	}
	if len(pagina.Filas) != len(ordenados) || len(ordenados) > estado.Acta.FilasAceptadas {
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
	bytesSeleccionados := 0
	for i := range protegidas {
		fila := protegidas[i]
		bytesFila := len(fila.Nonce) + len(fila.ContenidoCifrado) +
			len(fila.DerivacionDocumentoHMACSHA256) + len(fila.AtestacionFilaHMACSHA256)
		if fila.Numero != ordenados[i] || bytesFila > maximoBytesFilasSelectivas-bytesSeleccionados {
			return nil, ErrResultadoNoConfiable
		}
		bytesSeleccionados += bytesFila
	}
	filasParaProtector := clonarFilasProtegidas(protegidas)
	defer borrarFilasProtegidas(filasParaProtector)
	recuperadas, err := r.protector.RecuperarStaging(ctx, SolicitudRecuperacionStaging{
		ImportacionRef:      estado.Acta.ImportacionRef,
		HuellaFicheroSHA256: estado.Acta.HuellaFicheroSHA256,
		Esquema:             estado.Acta.Esquema,
		Filas:               filasParaProtector,
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if errors.Is(err, ErrMaterialNoConfiable) {
			return nil, ErrMaterialNoConfiable
		}
		return nil, ErrProteccionNoDisponible
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if validarCorrespondenciaRecuperada(protegidas, recuperadas, estado.Acta.Esquema) != nil {
		return nil, ErrMaterialNoConfiable
	}
	clonadas := clonarFilasDominio(recuperadas)
	porNumero := make(map[int]dominio.FilaAceptada, len(clonadas))
	for _, fila := range clonadas {
		porNumero[fila.Numero] = fila
	}
	salida := make([]dominio.FilaAceptada, len(numeros))
	for i, numero := range numeros {
		salida[i] = porNumero[numero]
	}
	return salida, nil
}

func ordenarNumerosSeleccionados(numeros []int) ([]int, error) {
	if len(numeros) == 0 || len(numeros) > maximoFilasRecuperacionSelectiva {
		return nil, ErrLoteNoConfiable
	}
	ordenados := append([]int(nil), numeros...)
	sort.Ints(ordenados)
	anterior := 1
	for _, numero := range ordenados {
		if numero <= anterior || numero > math.MaxInt32 {
			return nil, ErrLoteNoConfiable
		}
		anterior = numero
	}
	return ordenados, nil
}
