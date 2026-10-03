package simulacion

import (
	"encoding/json"
	"errors"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

var ErrNotas = errors.New("notas_prueba_invalidas")

// NotaPrueba contiene exclusivamente referencias sintéticas y una nota de ensayo.
type NotaPrueba struct {
	SolicitudRef      string `json:"solicitud_ref"`
	FaseRef           string `json:"fase_ref"`
	PuntosMicropuntos *int64 `json:"puntos_micropuntos"`
}

type notaPruebaJSON struct {
	SolicitudRef      string          `json:"solicitud_ref"`
	FaseRef           string          `json:"fase_ref"`
	PuntosMicropuntos json.RawMessage `json:"puntos_micropuntos"`
}

// El nombre del catálogo procede siempre del ejemplo embebido del servidor.
type NotaEditable struct {
	NotaPrueba
	Nombre string `json:"nombre"`
}

func catalogoNotas(f fixture) []NotaEditable {
	notas := []NotaEditable{}
	for _, solicitud := range f.Entrada.Solicitudes {
		for _, fase := range f.Configuracion.Fases {
			if fase.Tipo == "prueba" {
				notas = append(notas, NotaEditable{NotaPrueba: NotaPrueba{SolicitudRef: solicitud.Referencia, FaseRef: fase.Referencia, PuntosMicropuntos: solicitud.Notas[fase.Referencia]}, Nombre: solicitud.Nombre})
			}
		}
	}
	return notas
}

// PrepararConNotas reconstruye el ejemplo y admite cambios parciales de sus
// pruebas. Omitir la lista conserva sus notas; null explícito deja una pendiente.
func PrepararConNotas(s Solicitud) (domain.Entrada, BaremadorBolsa, error) {
	fixtures, err := leer()
	if err != nil {
		return domain.Entrada{}, BaremadorBolsa{}, err
	}
	for _, f := range fixtures {
		if f.Referencia != s.EjemploRef {
			continue
		}
		if len(s.NotasPrueba) > 0 {
			if err := s.Configuracion.Validar(); err != nil {
				return domain.Entrada{}, BaremadorBolsa{}, err
			}
			catalogo := map[claveNota]bool{}
			for _, nota := range catalogoNotas(f) {
				catalogo[claveNota{nota.SolicitudRef, nota.FaseRef}] = true
			}
			fases := map[string]domain.Fase{}
			for _, fase := range s.Configuracion.Fases {
				fases[fase.Referencia] = fase
			}
			vistas := map[claveNota]bool{}
			for _, nota := range s.NotasPrueba {
				clave := claveNota{nota.SolicitudRef, nota.FaseRef}
				fase, existe := fases[nota.FaseRef]
				if !catalogo[clave] || vistas[clave] || !existe || fase.Tipo != "prueba" || (nota.PuntosMicropuntos != nil && (*nota.PuntosMicropuntos < 0 || *nota.PuntosMicropuntos > fase.MaximoMicropuntos)) {
					return domain.Entrada{}, BaremadorBolsa{}, ErrNotas
				}
				vistas[clave] = true
			}
			for _, nota := range s.NotasPrueba {
				for i := range f.Entrada.Solicitudes {
					solicitud := &f.Entrada.Solicitudes[i]
					if solicitud.Referencia == nota.SolicitudRef {
						if solicitud.Notas == nil {
							solicitud.Notas = map[string]*int64{}
						}
						var puntos *int64
						if nota.PuntosMicropuntos != nil {
							valor := *nota.PuntosMicropuntos
							puntos = &valor
						}
						solicitud.Notas[nota.FaseRef] = puntos
					}
				}
			}
		}
		return f.Entrada, BaremadorBolsa{f.ConvocatoriaRef, f.BasesVersion, f.Baremo, f.Meritos}, nil
	}
	return domain.Entrada{}, BaremadorBolsa{}, ErrEjemplo
}

type claveNota struct{ solicitud, fase string }

// MarcarNotasEditadas añade procedencia al resultado ya calculado; no cambia
// puntos, requisitos, clasificación ni trazas del baremador común.
func MarcarNotasEditadas(r *domain.Resultado, notas []NotaPrueba) {
	editadas := map[claveNota]bool{}
	for _, nota := range notas {
		editadas[claveNota{nota.SolicitudRef, nota.FaseRef}] = true
	}
	for i := range r.Solicitudes {
		solicitud := &r.Solicitudes[i]
		for j := range solicitud.Fases {
			fase := &solicitud.Fases[j]
			if fase.Tipo == "prueba" && editadas[claveNota{solicitud.Referencia, fase.Referencia}] {
				fase.Origen = "prueba_editada"
			}
		}
	}
}
