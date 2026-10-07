package ports

import "vec-diputacion-granada/internal/modules/meritos/domain"

// Estos puertos sirven al consumidor de preparación. No son lectores de Persona
// ni Documentos: solo transportan referencias y afirmaciones sintéticas aportadas.
type EntradaPreparacion interface {
	Leer() (domain.Paquete, error)
}

type SalidaPreparacion interface {
	Escribir(domain.Preparacion) error
}
