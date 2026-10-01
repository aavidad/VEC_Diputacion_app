package main

import (
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

func ejecutarDiferenciaExperiencia(salida, diagnostico io.Writer, rutaAnterior, shaAnterior, rutaNueva, shaNueva string, limite int64) int {
	// Corresponde al maximo de la representacion canonica de reglasbaremo V1.
	const maximoBytesReglasExperiencia = 4 * 1024 * 1024
	if limite > maximoBytesReglasExperiencia {
		limite = maximoBytesReglasExperiencia
	}
	var conjuntos [2]reglasbaremo.ConjuntoReglasBaremo
	for i, archivo := range []struct{ ruta, sha, fase string }{
		{rutaAnterior, shaAnterior, "reglas_anteriores"}, {rutaNueva, shaNueva, "reglas_nuevas"},
	} {
		contenido, err := leerArchivoLimitado(archivo.ruta, limite)
		if err != nil {
			return diagnosticar(diagnostico, "archivo_invalido", archivo.fase, 2)
		}
		conjuntos[i], err = reglasbaremo.RestaurarConjuntoReglasBaremoConHuellaSHA256(contenido, archivo.sha)
		if err != nil {
			return diagnosticar(diagnostico, "reglas_invalidas", archivo.fase, 2)
		}
	}
	diferencia, err := reglasbaremo.CompararConjuntos(conjuntos[0], conjuntos[1])
	if err != nil {
		return diagnosticar(diagnostico, "comparacion_fallida", "comparacion", 2)
	}
	contenido, err := json.Marshal(diferencia)
	if err != nil {
		return diagnosticar(diagnostico, "comparacion_fallida", "resultado", 2)
	}
	n, err := salida.Write(contenido)
	if err != nil || n != len(contenido) {
		return diagnosticar(diagnostico, "salida_fallida", "comparacion", 2)
	}
	return 0
}
