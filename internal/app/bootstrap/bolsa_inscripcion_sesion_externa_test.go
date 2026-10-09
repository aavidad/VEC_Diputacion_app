package bootstrap

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

func TestAdaptadorSesionInscripcionExternaResuelvePersonaSinParticipacion(t *testing.T) {
	if _, err := NuevaSesionAspiranteInscripcionExterna(nil); !errors.Is(err, inscripcion.ErrNoDisponible) {
		t.Fatal("la sesión externa nula debe cerrar")
	}
	sesion, _, entorno, peticion, _ := escenarioSesionExternaInscripcionPrueba(t)
	adaptador, err := NuevaSesionAspiranteInscripcionExterna(sesion)
	if err != nil {
		t.Fatal(err)
	}
	ctx, acreditacion, err := adaptador.ResolverInscripcion(peticion)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := ctx.Vinculo.Datos()
	if err != nil || len(ctx.Resultado.Contexto.Instantanea.Vinculos) != 0 ||
		acreditacion.PersonaRef != ctx.Resultado.Contexto.PersonaRef ||
		acreditacion.PerfilRef != vinculo.PerfilActivoRef ||
		acreditacion.CuentaRef != vinculo.CuentaRef ||
		acreditacion.SesionRef != vinculo.SesionRef ||
		acreditacion.AutenticacionRef != vinculo.AutenticacionRef ||
		acreditacion.CertificadoHuellaSHA256 == "" ||
		len(entorno.registro.altas) != 1 || entorno.revalidador.llamadas != 1 || entorno.resolutor.llamadas != 1 {
		t.Fatal("la adaptación pierde el vínculo nominal o resuelve la sesión más de una vez")
	}
	publico, err := json.Marshal(acreditacion)
	if err != nil || strings.Contains(string(publico), acreditacion.PersonaRef) ||
		strings.Contains(string(publico), acreditacion.CertificadoHuellaSHA256) {
		t.Fatal("la acreditación del adaptador expone identidad privada")
	}
}
