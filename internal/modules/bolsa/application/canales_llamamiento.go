package application

import (
	"context"
	"errors"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

var ErrCanalesLlamamientoNoCompuestos = errors.New("bolsa: canales de llamamiento no compuestos")

// RegistroCanalesLlamamiento une el catálogo de canales con sus adaptadores
// reales. Un canal se publica solo si el catálogo lo activa y su adaptador
// existe y está disponible; si no, no aparece y no se puede elegir.
type RegistroCanalesLlamamiento struct {
	catalogo    dominiobolsa.CatalogoCanalesLlamamiento
	adaptadores map[string]puertosbolsa.CanalAvisoLlamamiento
}

// NuevoRegistroCanalesLlamamiento exige un catálogo válido y adaptadores de
// canales que el catálogo conozca. Un canal activo sin adaptador se admite
// (queda apagado) para que activar el catálogo antes del proveedor no tumbe
// el arranque; se informa con CanalesSinProveedor.
func NuevoRegistroCanalesLlamamiento(catalogo dominiobolsa.CatalogoCanalesLlamamiento, adaptadores ...puertosbolsa.CanalAvisoLlamamiento) (*RegistroCanalesLlamamiento, error) {
	if catalogo.Validar() != nil {
		return nil, ErrCanalesLlamamientoNoCompuestos
	}
	conocidos := map[string]bool{}
	for _, c := range catalogo.Canales {
		conocidos[c.Canal] = true
	}
	r := &RegistroCanalesLlamamiento{catalogo: catalogo, adaptadores: map[string]puertosbolsa.CanalAvisoLlamamiento{}}
	for _, a := range adaptadores {
		if a == nil || !conocidos[a.Canal()] || r.adaptadores[a.Canal()] != nil {
			return nil, ErrCanalesLlamamientoNoCompuestos
		}
		r.adaptadores[a.Canal()] = a
	}
	return r, nil
}

// CanalesSinProveedor lista los canales activos en el catálogo sin adaptador
// compuesto, para avisarlo en el registro técnico del arranque.
func (r *RegistroCanalesLlamamiento) CanalesSinProveedor() []string {
	if r == nil {
		return nil
	}
	var faltan []string
	for _, c := range r.catalogo.Canales {
		if c.Activo && r.adaptadores[c.Canal] == nil {
			faltan = append(faltan, c.Canal)
		}
	}
	return faltan
}

// Activos devuelve, en el orden del catálogo, los canales que se pueden usar
// ahora. El correo va siempre primero por catálogo; una dependencia caída
// solo apaga su canal.
func (r *RegistroCanalesLlamamiento) Activos(ctx context.Context) []dominiobolsa.CanalLlamamiento {
	if r == nil || ctx == nil {
		return nil
	}
	activos := make([]dominiobolsa.CanalLlamamiento, 0, len(r.catalogo.Canales))
	for _, c := range r.catalogo.Canales {
		a := r.adaptadores[c.Canal]
		if c.Activo && a != nil && a.Disponible(ctx) {
			activos = append(activos, c.Copia())
		}
	}
	return activos
}
