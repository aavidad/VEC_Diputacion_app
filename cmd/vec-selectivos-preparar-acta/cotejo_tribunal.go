package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"syscall"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

// leerSalidaTribunal conserva la huella de los bytes recibidos, incluida la
// presentación JSON. Nunca sustituye esa huella por la de una reserialización.
func leerSalidaTribunal(archivo string) (domain.PreparacionTribunal, string, error) {
	f, err := os.OpenFile(archivo, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return domain.PreparacionTribunal{}, "", errEntradaJSON
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maximoEntradaJSON {
		return domain.PreparacionTribunal{}, "", errEntradaJSON
	}
	raw, err := io.ReadAll(io.LimitReader(f, maximoEntradaJSON+1))
	if err != nil || len(raw) > maximoEntradaJSON {
		return domain.PreparacionTribunal{}, "", errEntradaJSON
	}
	var sobre struct {
		Preparacion domain.PreparacionTribunal `json:"preparacion"`
		Limite      string                     `json:"limite"`
		Mensajes    []pendienteVisible         `json:"mensajes"`
	}
	if leerJSON(bytes.NewReader(raw), &sobre) != nil {
		return domain.PreparacionTribunal{}, "", errEntradaJSON
	}
	huella := sha256.Sum256(raw)
	return sobre.Preparacion, hex.EncodeToString(huella[:]), nil
}
