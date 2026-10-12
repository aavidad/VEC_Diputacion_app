package application

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
)

var ErrReferenciaHechoNoDisponible = errors.New("meritos.error.referencia_hecho_no_disponible")

// GeneradorReferenciaHecho emite una identidad aleatoria previa a la carga.
// La reserva, propiedad e idempotencia pertenecen al plan durable de Documentos;
// generar una referencia no concede acceso ni declara un hecho en RUM.
type GeneradorReferenciaHecho struct {
	entropia io.Reader
}

func NuevoGeneradorReferenciaHecho() *GeneradorReferenciaHecho {
	return &GeneradorReferenciaHecho{entropia: rand.Reader}
}

// NuevoHechoRef cumple por forma los puertos neutrales de composicion. No
// incluye persona, DNI, cuenta, fecha ni clave de idempotencia en el ID.
func (g *GeneradorReferenciaHecho) NuevoHechoRef() (string, error) {
	if g == nil || g.entropia == nil {
		return "", ErrReferenciaHechoNoDisponible
	}
	var valor [32]byte
	if _, err := io.ReadFull(g.entropia, valor[:]); err != nil {
		return "", ErrReferenciaHechoNoDisponible
	}
	todoCero := true
	for _, b := range valor {
		if b != 0 {
			todoCero = false
			break
		}
	}
	if todoCero {
		return "", ErrReferenciaHechoNoDisponible
	}
	return "ref:" + hex.EncodeToString(valor[:]), nil
}
