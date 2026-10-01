package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/domain"
)

func ejecutarDiferenciaConcursos(salida, diagnostico io.Writer, rutaAnterior, shaAnterior, rutaNueva, shaNueva string, limite int64) int {
	if limite > simulacion.MaximoBytes {
		limite = simulacion.MaximoBytes
	}
	var configuraciones [2]domain.Configuracion
	for i, archivo := range []struct{ ruta, sha, fase string }{
		{rutaAnterior, shaAnterior, "reglas_anteriores"}, {rutaNueva, shaNueva, "reglas_nuevas"},
	} {
		datos, err := leerArchivoLimitado(archivo.ruta, limite)
		if err != nil {
			return diagnosticar(diagnostico, "archivo_invalido", archivo.fase, 2)
		}
		suma := sha256.Sum256(datos)
		if archivo.sha != hex.EncodeToString(suma[:]) {
			return diagnosticar(diagnostico, "huella_invalida", archivo.fase, 2)
		}
		if err := simulacion.Decodificar(bytes.NewReader(datos), &configuraciones[i]); err != nil {
			return diagnosticar(diagnostico, "json_contrato_invalido", archivo.fase, 2)
		}
		if err := domain.ValidarConfiguracion(configuraciones[i]); err != nil {
			return diagnosticar(diagnostico, "reglas_invalidas", archivo.fase, 2)
		}
	}
	diferencia, err := domain.CompararConfiguraciones(configuraciones[0], configuraciones[1])
	if err != nil {
		return diagnosticar(diagnostico, "comparacion_fallida", "comparacion", 2)
	}
	return emitirConcursos(salida, diagnostico, diferencia)
}
