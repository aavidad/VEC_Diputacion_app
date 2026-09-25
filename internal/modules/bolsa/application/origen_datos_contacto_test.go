package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type politicaOrigenPrueba struct {
	llamadas int
	err      error
}

func (p *politicaOrigenPrueba) MarcaOrigenConvoca(_ context.Context, registradaEn time.Time) (dominiobolsa.MarcaOrigenDatosContacto, error) {
	p.llamadas++
	if p.err != nil {
		return dominiobolsa.MarcaOrigenDatosContacto{}, p.err
	}
	return dominiobolsa.MarcaOrigenDatosContacto{
		Origen: dominiobolsa.OrigenDatosContactoConvoca, VigenteHasta: registradaEn.AddDate(1, 0, 0),
		UltimoDia: registradaEn.AddDate(1, 0, -1).Format(time.DateOnly), ReglaRef: "vec.bolsa.reglas:1:b29.contacto_origen_convoca",
		ReglaHuella: strings.Repeat("d", 64),
	}, nil
}

func solicitudConvocaPrueba(t *testing.T, ahora time.Time, clave string) puertosbolsa.SolicitudRegistrarDatosContactoParticipacion {
	t.Helper()
	s := solicitudDatosContactoPrueba(t, ahora, clave, datosDatosContactoPrueba())
	s.Origen = dominiobolsa.OrigenDatosContactoConvoca
	return s
}

func TestServicioContactoOrigenConvocaSinReglaSeRechazaSinConsumirAutorizacion(t *testing.T) {
	ahora := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	repo := &repositorioDatosContactoPrueba{}
	servicio, autorizador := servicioDatosContactoPrueba(t, ahora, repo, &cifradorDatosContactoPrueba{}, contextoSituacionPrueba{}, true)
	if _, err := servicio.Registrar(context.Background(), solicitudConvocaPrueba(t, ahora, "convoca-1")); !errors.Is(err, puertosbolsa.ErrOrigenDatosContactoNoConfigurado) {
		t.Fatalf("sin regla no hay vigencia inventada: %v", err)
	}
	if autorizador.llamadas != 0 || repo.llamadas != 0 {
		t.Fatalf("no debe autorizar ni registrar: auth=%d repo=%d", autorizador.llamadas, repo.llamadas)
	}
	otra := solicitudConvocaPrueba(t, ahora, "convoca-2")
	otra.Origen = "persona"
	if _, err := servicio.Registrar(context.Background(), otra); err == nil {
		t.Fatal("un origen desconocido debe rechazarse")
	}
}

func TestServicioContactoOrigenConvocaMarcaVersionYSeVenceEnLaLectura(t *testing.T) {
	ahora := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	instante := ahora
	repo := &repositorioDatosContactoPrueba{}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, err := NuevoServicioDatosContactoParticipacion(contextoSituacionPrueba{}, autorizador, pertenenciaDatosContactoPrueba{true}, &cifradorDatosContactoPrueba{}, repo, func() time.Time { return instante })
	if err != nil {
		t.Fatal(err)
	}
	if servicio.EstablecerPoliticaOrigenDatosContacto(nil) == nil {
		t.Fatal("una política nula debe rechazarse")
	}
	politica := &politicaOrigenPrueba{}
	if err := servicio.EstablecerPoliticaOrigenDatosContacto(politica); err != nil {
		t.Fatal(err)
	}
	registro, err := servicio.Registrar(context.Background(), solicitudConvocaPrueba(t, ahora, "convoca-1"))
	if err != nil || registro.Origen == nil || repo.ultimo.Origen == nil || politica.llamadas != 1 {
		t.Fatalf("registro=%+v err=%v", registro, err)
	}
	if !repo.ultimo.Origen.VigenteHasta.Equal(time.Date(2027, 9, 28, 8, 0, 0, 0, time.UTC)) || repo.ultimo.Origen.Validar() != nil {
		t.Fatalf("marca: %+v", repo.ultimo.Origen)
	}
	// Replay con la misma clave sin origen: los datos ya no son los mismos.
	if _, err := servicio.Registrar(context.Background(), solicitudDatosContactoPrueba(t, ahora, "convoca-1", datosDatosContactoPrueba())); !errors.Is(err, dominiobolsa.ErrDatosContactoParticipacionInvalidos) {
		t.Fatalf("replay sin origen: %v", err)
	}
	repetido, err := servicio.Registrar(context.Background(), solicitudConvocaPrueba(t, ahora, "convoca-1"))
	if err != nil || !repetido.Reutilizada || repo.llamadas != 1 {
		t.Fatalf("replay exacto: %+v %v", repetido, err)
	}
	consulta := puertosbolsa.SolicitudConsultarDatosContactoParticipacion{ContextoActor: dominiovec.ContextoActor{PersonaRef: "per_0123456789abcdefghijkl"}, BolsaRef: "bolsa:b4", ParticipacionRef: "participacion:b4"}
	leidos, err := servicio.Consultar(context.Background(), consulta)
	if err != nil || leidos.Origen == nil || leidos.EstadoOrigen != dominiobolsa.EstadoOrigenContactoVigente {
		t.Fatalf("lectura vigente: %+v %v", leidos, err)
	}
	instante = time.Date(2027, 9, 28, 8, 0, 0, 0, time.UTC)
	leidos, err = servicio.Consultar(context.Background(), consulta)
	if err != nil || leidos.EstadoOrigen != dominiobolsa.EstadoOrigenContactoVencido {
		t.Fatalf("lectura vencida: %+v %v", leidos, err)
	}
	// La persona confirma (versión propia): desaparece la marca.
	autorizador.instante = instante
	if _, err := servicio.Registrar(context.Background(), solicitudDatosContactoPrueba(t, instante, "confirmado-1", datosDatosContactoPrueba())); err != nil {
		t.Fatal(err)
	}
	leidos, err = servicio.Consultar(context.Background(), consulta)
	if err != nil || leidos.Version != 2 || leidos.Origen != nil || leidos.EstadoOrigen != "" {
		t.Fatalf("contacto propio: %+v %v", leidos, err)
	}
}

func TestServicioContactoOrigenConvocaReglaCaidaNoRegistra(t *testing.T) {
	ahora := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	repo := &repositorioDatosContactoPrueba{}
	servicio, _ := servicioDatosContactoPrueba(t, ahora, repo, &cifradorDatosContactoPrueba{}, contextoSituacionPrueba{}, true)
	_ = servicio.EstablecerPoliticaOrigenDatosContacto(&politicaOrigenPrueba{err: errors.New("calendario caído")})
	if _, err := servicio.Registrar(context.Background(), solicitudConvocaPrueba(t, ahora, "convoca-1")); !errors.Is(err, puertosbolsa.ErrOrigenDatosContactoNoConfigurado) || repo.llamadas != 0 {
		t.Fatalf("una regla no disponible no puede dar vigencia: %v repo=%d", err, repo.llamadas)
	}
}
