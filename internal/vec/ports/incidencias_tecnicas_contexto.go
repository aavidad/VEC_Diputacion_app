package ports

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"sync/atomic"

	"vec-diputacion-granada/internal/vec/domain"
)

type claveCorrelacionIncidenciasPeticion struct{}

// ErrCorrelacionIncidenciasNoDisponible no revela errores ni detalles de la
// fuente aleatoria.
var ErrCorrelacionIncidenciasNoDisponible = errors.New("incidencias: correlacion no disponible")

// ConCorrelacionIncidenciasPeticion crea una correlación interna aleatoria.
// Debe invocarse una vez en la frontera, nunca a partir de cabeceras, rutas ni
// datos de personas. El contexto derivado se propaga a adaptadores y efectos.
func ConCorrelacionIncidenciasPeticion(ctx context.Context) (context.Context, error) {
	return conCorrelacionIncidenciasPeticion(ctx, rand.Reader)
}

func conCorrelacionIncidenciasPeticion(ctx context.Context, aleatorio io.Reader) (context.Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var bruto [16]byte
	if _, err := io.ReadFull(aleatorio, bruto[:]); err != nil {
		// Sombrea también una eventual correlación anterior: ninguna petición
		// fallida hereda una coincidencia de otro contexto.
		return context.WithValue(ctx, claveCorrelacionIncidenciasPeticion{}, ""), ErrCorrelacionIncidenciasNoDisponible
	}
	return context.WithValue(ctx, claveCorrelacionIncidenciasPeticion{}, hex.EncodeToString(bruto[:])), nil
}

// CorrelacionIncidenciasPeticion consulta únicamente la clave privada que
// coloca la frontera. Ausencia y fallo nunca constituyen una coincidencia.
func CorrelacionIncidenciasPeticion(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	correlacion, ok := ctx.Value(claveCorrelacionIncidenciasPeticion{}).(string)
	return correlacion, ok && domain.EsCorrelacionTecnicaValida(correlacion)
}

// El emisor de incidencias técnicas se inyecta por constructor (opciones del
// adaptador); junto a la correlación, el contexto transporta una marca: si en
// esa petición ya se declaró una incidencia específica del catálogo, el
// middleware común no añade HTTP_INTERNO_FALLIDO por el mismo fallo.

// claveMarcaIncidenciaPeticion es privada: solo este paquete coloca o lee la
// marca del contexto.
type claveMarcaIncidenciaPeticion struct{}

type marcaIncidenciaPeticion struct {
	declarada atomic.Bool
}

// ConMarcaIncidenciasPeticion devuelve un contexto con la marca de la
// petición y la función que indica si durante ella se declaró una incidencia
// específica. La coloca la frontera de transporte (middleware del servidor).
func ConMarcaIncidenciasPeticion(ctx context.Context) (context.Context, func() bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	marca := &marcaIncidenciaPeticion{}
	return context.WithValue(ctx, claveMarcaIncidenciaPeticion{}, marca), marca.declarada.Load
}

// EmitirIncidenciaTecnicaEnPeticion declara la solicitud con el emisor que el
// adaptador recibió por constructor y marca la petición para que el mismo
// fallo no se cuente dos veces. Un emisor nil no emite ni marca: la
// supervisión es una capa y su ausencia nunca cambia la respuesta.
func EmitirIncidenciaTecnicaEnPeticion(ctx context.Context, emisor EmisorIncidenciasTecnicas, solicitud domain.SolicitudIncidenciaTecnica) {
	if emisor == nil {
		return
	}
	if contextual, ok := emisor.(EmisorIncidenciasTecnicasConContexto); ok {
		contextual.EmitirConContexto(ctx, solicitud)
	} else {
		emisor.Emitir(solicitud)
	}
	if ctx == nil {
		return
	}
	if marca, ok := ctx.Value(claveMarcaIncidenciaPeticion{}).(*marcaIncidenciaPeticion); ok {
		marca.declarada.Store(true)
	}
}

// EmisorIncidenciasTecnicasNulo descarta las solicitudes. Es solo el valor
// por defecto de los adaptadores construidos sin emisor (pruebas y
// composiciones auxiliares); la composición raíz de vec-server exige un
// emisor real y rechaza nil.
type EmisorIncidenciasTecnicasNulo struct{}

// Emitir no hace nada.
func (EmisorIncidenciasTecnicasNulo) Emitir(domain.SolicitudIncidenciaTecnica) {}

var _ EmisorIncidenciasTecnicas = EmisorIncidenciasTecnicasNulo{}
