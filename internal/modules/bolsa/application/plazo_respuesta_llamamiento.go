package application

import (
	"context"
	"errors"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// ResolutorReglasPlazo es la parte del resolutor común de reglas que usa
// Bolsa. Un *reglas.Resolutor nulo es válido y significa «sin catálogo».
type ResolutorReglasPlazo interface {
	Vencimiento(ctx context.Context, clave string, inicio time.Time, municipioSede string) (reglas.Regla, reglas.Vencimiento, error)
}

// ServicioPlazoRespuestaLlamamiento propone al asistente B7 el plazo de
// respuesta publicado en el catálogo de reglas de Bolsa, calculado desde el
// instante de la consulta. No escribe estado ni fija el plazo del
// llamamiento: RRHH lo revisa y puede cambiarlo antes de emitir.
type ServicioPlazoRespuestaLlamamiento struct {
	resolutor ResolutorReglasPlazo
	ahora     func() time.Time
}

func NuevoServicioPlazoRespuestaLlamamiento(resolutor ResolutorReglasPlazo, ahora func() time.Time) (*ServicioPlazoRespuestaLlamamiento, error) {
	if resolutor == nil || ahora == nil {
		return nil, puertosbolsa.ErrPlazoRespuestaNoDisponible
	}
	return &ServicioPlazoRespuestaLlamamiento{resolutor: resolutor, ahora: ahora}, nil
}

// ConsultarPlazoRespuesta devuelve Configurada=false sin error cuando no hay
// catálogo compuesto, para que el asistente mantenga el texto libre.
func (s *ServicioPlazoRespuestaLlamamiento) ConsultarPlazoRespuesta(ctx context.Context) (puertosbolsa.PlazoRespuestaLlamamiento, error) {
	var vacio puertosbolsa.PlazoRespuestaLlamamiento
	if s == nil || s.resolutor == nil || s.ahora == nil || ctx == nil {
		return vacio, puertosbolsa.ErrPlazoRespuestaNoDisponible
	}
	ahora := s.ahora().UTC().Truncate(time.Second)
	if ahora.IsZero() {
		return vacio, puertosbolsa.ErrPlazoRespuestaNoDisponible
	}
	regla, vencimiento, err := s.resolutor.Vencimiento(ctx, reglas.BolsaPlazoRespuesta, ahora, "")
	switch {
	case errors.Is(err, reglas.ErrReglasNoConfiguradas):
		return vacio, nil
	case err != nil:
		if ctx.Err() != nil {
			return vacio, ctx.Err()
		}
		return vacio, puertosbolsa.ErrPlazoRespuestaNoDisponible
	}
	if regla.Descripcion == "" || regla.Referencia == "" || vencimiento.UltimoDia == "" || !vencimiento.VenceAntesDe.After(ahora) {
		return vacio, puertosbolsa.ErrPlazoRespuestaNoDisponible
	}
	venceAntesDe := vencimiento.VenceAntesDe.UTC()
	return puertosbolsa.PlazoRespuestaLlamamiento{
		Configurada: true,
		Regla: puertosbolsa.ReglaPlazoRespuesta{
			Etiqueta: regla.Etiqueta, Texto: regla.Descripcion, Referencia: regla.Referencia,
			Origen: string(regla.Origen), Articulo: regla.Articulo, Ejemplo: regla.EsEjemplo(),
		},
		CalculadoEn: ahora, UltimoDia: vencimiento.UltimoDia,
		VenceEn: venceAntesDe.Add(-time.Second), VenceAntesDe: venceAntesDe,
	}, nil
}
