package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"strconv"
	"strings"
	"unicode"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

// La procedencia histórica se conserva y el estado se presenta según catálogo.
// Se neutralizan las celdas que una
// hoja de cálculo podría interpretar como fórmula, incluidas las cabeceras.
func celdaCSVServiciosPropios(s string) string {
	t := strings.TrimLeftFunc(s, func(r rune) bool { return unicode.IsSpace(r) || r == '\ufeff' || r == '\u200b' })
	if len(t) > 0 && strings.ContainsRune("=+-@", rune(t[0])) {
		return "'" + s
	}
	return s
}
func serializarServiciosPropiosCSV(f domain.FormatoExportacionServiciosPropios, filas []domain.ServicioFichaPropia) ([]byte, string, error) {
	if f.Validar() != nil || filas == nil || len(filas) > domain.LimiteFilasFichaPropia {
		return nil, "", domain.ErrExportacionServiciosPropiosNoDisponible
	}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	datos := f.Datos()
	cab := datos.Cabeceras
	for i := range cab {
		cab[i] = celdaCSVServiciosPropios(cab[i])
	}
	if w.Write(cab) != nil {
		return nil, "", domain.ErrExportacionServiciosPropiosNoDisponible
	}
	for _, s := range filas {
		estado, ok := datos.Estados[s.Estado]
		if !ok {
			return nil, "", domain.ErrExportacionServiciosPropiosNoDisponible
		}
		fila := []string{s.Inicio.Texto(), s.Fin.Texto(), s.Clase, strconv.FormatInt(s.Dias, 10), estado}
		for i := range fila {
			fila[i] = celdaCSVServiciosPropios(fila[i])
		}
		if w.Write(fila) != nil {
			return nil, "", domain.ErrExportacionServiciosPropiosNoDisponible
		}
	}
	w.Flush()
	if w.Error() != nil || b.Len() == 0 || b.Len() > domain.LimiteBytesExportacionServiciosPropios {
		return nil, "", domain.ErrExportacionServiciosPropiosNoDisponible
	}
	contenido := append([]byte(nil), b.Bytes()...)
	h := sha256.Sum256(contenido)
	return contenido, hex.EncodeToString(h[:]), nil
}
