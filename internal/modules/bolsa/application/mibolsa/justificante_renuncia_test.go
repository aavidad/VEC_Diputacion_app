package mibolsa

import (
	"errors"
	"testing"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestCandidatoDelActoUsaVinculoRegistrado(t *testing.T) {
	entorno := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	datos, err := entorno.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	candidato, err := CandidatoDelActo(entorno.resultado, datos.VinculoAutenticacionActor)
	if err != nil || candidato != referenciaServicioContextoActorPrueba("can_", "c") {
		t.Fatalf("vínculo propio no resuelto: %q, %v", candidato, err)
	}
	sinCandidato := nuevoEntornoAutorizacionSolicitudV3Prueba(t, func(i *vecdomain.InstantaneaContextoActor) {
		i.Vinculos = i.Vinculos[1:]
	})
	datos, err = sinCandidato.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = CandidatoDelActo(sinCandidato.resultado, datos.VinculoAutenticacionActor); !errors.Is(err, bolsaports.ErrJustificanteRenunciaDenegado) {
		t.Fatalf("vínculo candidato ausente aceptado: %v", err)
	}
}
