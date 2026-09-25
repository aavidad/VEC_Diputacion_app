package ports

import (
	"context"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

// PoliticaSegregacionVigente es la política que la base de datos aplicará a
// la próxima operación de B8, con su versión y su procedencia.
type PoliticaSegregacionVigente struct {
	Version     int64
	CatalogoRef string
	Politica    dominiobolsa.PoliticaSegregacion
}

// PublicacionPoliticaSegregacion traslada una entrada exacta del catálogo
// de separación de funciones a la autoridad durable de Bolsa.
type PublicacionPoliticaSegregacion struct {
	CatalogoRef    string
	CatalogoSHA256 string
	Politica       dominiobolsa.PoliticaSegregacion
}

// ConsultaPoliticaSegregacion lee la política vigente. Un repositorio de
// situación que no la implemente conserva solo el mínimo fijo del dominio.
type ConsultaPoliticaSegregacion interface {
	PoliticaSegregacion(context.Context) (PoliticaSegregacionVigente, error)
}

// PublicadorPoliticaSegregacion registra una nueva versión si difiere de la
// vigente; publicar la misma entrada otra vez no crea historia.
type PublicadorPoliticaSegregacion interface {
	PublicarPoliticaSegregacion(context.Context, PublicacionPoliticaSegregacion) (PoliticaSegregacionVigente, error)
}
