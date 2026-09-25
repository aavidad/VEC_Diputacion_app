package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	reglasbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/reglas"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestContactoOrigenConvocaVenceDoceMesesCivilesTrasElRegistro(t *testing.T) {
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""), nil, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	marca, err := reglasbolsa.NuevoContactoOrigen(compuestas.bolsa).MarcaOrigenConvoca(t.Context(), time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	// Fecha a fecha en hora peninsular: vence al acabar el 28/09/2027.
	if marca.UltimoDia != "2027-09-28" || !marca.VigenteHasta.Equal(time.Date(2027, 9, 28, 22, 0, 0, 0, time.UTC)) || marca.Validar() != nil {
		t.Fatalf("marca: %+v", marca)
	}
}

func TestComponerContactoOrigenSinCatalogoNoCambiaNada(t *testing.T) {
	if err := componerContactoOrigenBolsaDesarrollo(nil, nil); err != nil {
		t.Fatal(err)
	}
}

type repositorioDatosContactoOrigenPrueba struct {
	puertosbolsa.RepositorioDatosContactoParticipacion
	registro puertosbolsa.RegistroDatosContactoParticipacion
	err      error
}

func (r repositorioDatosContactoOrigenPrueba) DatosContactoVigentes(context.Context, string) (puertosbolsa.RegistroDatosContactoParticipacion, error) {
	return r.registro, r.err
}

func TestFuenteOrigenContactoLeeLaVersionVigente(t *testing.T) {
	marca := &dominiobolsa.MarcaOrigenDatosContacto{Origen: dominiobolsa.OrigenDatosContactoConvoca}
	fuente := &fuenteCorreoParticipacionB7{repositorio: repositorioDatosContactoOrigenPrueba{registro: puertosbolsa.RegistroDatosContactoParticipacion{Origen: marca}}}
	if got, err := fuente.OrigenContactoParticipacion(t.Context(), "participacion:1"); err != nil || got != marca {
		t.Fatalf("marca=%v err=%v", got, err)
	}
	fuente.repositorio = repositorioDatosContactoOrigenPrueba{err: puertosbolsa.ErrDatosContactoParticipacionNoEncontrados}
	if got, err := fuente.OrigenContactoParticipacion(t.Context(), "participacion:1"); err != nil || got != nil {
		t.Fatalf("sin contacto no hay marca: %v %v", got, err)
	}
	fuente.repositorio = repositorioDatosContactoOrigenPrueba{err: errors.New("caído")}
	if _, err := fuente.OrigenContactoParticipacion(t.Context(), "participacion:1"); err == nil {
		t.Fatal("un repositorio caído no puede darse por contacto propio")
	}
}
