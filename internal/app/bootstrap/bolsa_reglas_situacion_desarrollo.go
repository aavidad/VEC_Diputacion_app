package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	reglasbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/reglas"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/reglas"
)

// rutaReglasSituacionBolsaDesarrollo va bajo la misma frontera mTLS de
// consulta RRHH que el resto de lecturas de Bolsa.
const rutaReglasSituacionBolsaDesarrollo = bolsahttp.RutaReglasSituacion

// plazoPublicacionTransicionesBolsa acota la publicación al arrancar.
const plazoPublicacionTransicionesBolsa = 5 * time.Second

var errPoliticaTransicionesEjemploNoValida = errors.New("bootstrap: politica de transiciones de Bolsa no valida")

// componerReglasSituacionBolsaDesarrollo engancha el catálogo de reglas de
// Bolsa al cambio de situación: publica en la base la política de
// transiciones que resulta del catálogo, restringe con él el servicio ya
// compuesto y publica la lectura que usa la pantalla de RRHH, con los
// destinos que el servicio admitirá. Con resolutor nulo (sin catálogo) la
// ruta responde «configuradas: false» y el servicio aplica la política que ya
// tenga la base (o la compilada, si la base no tiene la migración 000032).
func componerReglasSituacionBolsaDesarrollo(resolutor *reglas.Resolutor, mutador http.Handler) (vechttp.RutaExacta, error) {
	consulta := reglasbolsa.NuevasReglasSituacion(resolutor)
	var servicio *aplicacionbolsa.ServicioSituacionParticipacion
	if participacion, ok := mutador.(*manejadorParticipacionBolsaDesarrollo); ok && participacion != nil {
		servicio = participacion.servicioSituacion
	}
	var fuente bolsahttp.FuenteTransicionesSituacion
	if servicio != nil {
		if consulta.Configurada() {
			if err := publicarPoliticaTransicionesBolsaDesarrollo(consulta, servicio); err != nil {
				return vechttp.RutaExacta{}, err
			}
			servicio.EstablecerReglasTransiciones(consulta)
		}
		fuente = servicio
	}
	manejador, err := bolsahttp.NuevoHandlerReglasSituacionConTransiciones(consulta, fuente)
	if err != nil {
		return vechttp.RutaExacta{}, err
	}
	return vechttp.RutaExacta{Ruta: bolsahttp.RutaReglasSituacion, Manejador: manejador}, nil
}

// publicarPoliticaTransicionesBolsaDesarrollo traslada a la base las
// transiciones del catálogo. Un catálogo inválido o una base que lo rechaza
// impiden arrancar en lugar de ignorar la configuración. Una base sin la
// migración 000032 conserva su tabla, que es la compilada, y el catálogo
// solo puede restringirla.
func publicarPoliticaTransicionesBolsaDesarrollo(consulta *reglasbolsa.ReglasSituacion, servicio *aplicacionbolsa.ServicioSituacionParticipacion) error {
	ctx, cancelar := context.WithTimeout(context.Background(), plazoPublicacionTransicionesBolsa)
	defer cancelar()
	publicacion, hay, err := consulta.PoliticaTransiciones(ctx)
	if err != nil {
		return errors.Join(errPoliticaTransicionesEjemploNoValida, err)
	}
	if !hay {
		return nil
	}
	if _, err := servicio.PublicarPoliticaTransiciones(ctx, publicacion); err != nil && !errors.Is(err, puertosbolsa.ErrPoliticaTransicionesNoInstalada) {
		return errors.Join(errPoliticaTransicionesEjemploNoValida, err)
	}
	return nil
}
