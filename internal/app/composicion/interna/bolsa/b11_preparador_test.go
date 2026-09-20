package bolsa

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	httpinternobolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type extractorConfirmacionB11Prueba struct {
	confirmacion httpseguridad.ConfirmacionPeticionSesion
	err          error
	llamadas     int
}

func (e *extractorConfirmacionB11Prueba) ExtraerConfirmacionVinculada(context.Context) (httpseguridad.ConfirmacionPeticionSesion, error) {
	e.llamadas++
	return e.confirmacion, e.err
}

type revalidadorB11PreparadorPrueba struct {
	llamadas  int
	solicitud domain.SolicitudRevalidacionAutenticacionActorV1
	err       error
}

func (r *revalidadorB11PreparadorPrueba) RevalidarAutenticacionActorV1(_ context.Context, solicitud domain.SolicitudRevalidacionAutenticacionActorV1) (domain.AutenticacionRevalidadaV1, error) {
	r.llamadas++
	r.solicitud = solicitud
	return domain.AutenticacionRevalidadaV1{}, r.err
}

type resolutorB11PreparadorPrueba struct{ llamadas int }

func (r *resolutorB11PreparadorPrueba) ResolverContextoActorGobernadoV1(context.Context, domain.AutenticacionRevalidadaV1) (domain.ResultadoContextoActorRegistradoV2, error) {
	r.llamadas++
	return domain.ResultadoContextoActorRegistradoV2{}, nil
}

type relojB11PreparadorPrueba struct{ ahora time.Time }

func (r relojB11PreparadorPrueba) Ahora() time.Time { return r.ahora }

func TestPreparadorB11FallaCerradoAntesDeRevalidarConfirmacionNoValida(t *testing.T) {
	ahora := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	extractor := &extractorConfirmacionB11Prueba{confirmacion: httpseguridad.ConfirmacionPeticionSesion{
		SesionRef:         "ses_" + strings.Repeat("s", 22),
		SesionValidaHasta: ahora.Add(time.Minute),
	}}
	revalidador := &revalidadorB11PreparadorPrueba{}
	resolutor := &resolutorB11PreparadorPrueba{}
	preparador, err := NuevoPreparadorOrdenParticipacionesPropiasB11(extractor, revalidador, resolutor, relojB11PreparadorPrueba{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = preparador.PrepararOrdenConsultaParticipacionesPropias(context.Background()); !errors.Is(err, ErrPreparadorParticipacionesPropiasB11Invalido) || !errors.Is(err, httpinternobolsa.ErrAutenticacionInternaAusente) {
		t.Fatalf("confirmacion inválida aceptada: %v", err)
	}
	if extractor.llamadas != 1 || revalidador.llamadas != 0 || resolutor.llamadas != 0 {
		t.Fatalf("autoridades consultadas con confirmación inválida: extractor=%d revalidador=%d resolutor=%d", extractor.llamadas, revalidador.llamadas, resolutor.llamadas)
	}
}

func TestPreparadorB11ExtraeUnaVezYEntregaSoloReferenciasCanonicasALaFabrica(t *testing.T) {
	ahora := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	extractor := &extractorConfirmacionB11Prueba{confirmacion: httpseguridad.ConfirmacionPeticionSesion{
		AutenticacionRef:  "aut_" + strings.Repeat("a", 22),
		SesionRef:         "ses_" + strings.Repeat("s", 22),
		SesionValidaHasta: ahora.Add(time.Minute),
	}}
	revalidador := &revalidadorB11PreparadorPrueba{err: errors.New("revalidacion rechazada")}
	resolutor := &resolutorB11PreparadorPrueba{}
	preparador, err := NuevoPreparadorOrdenParticipacionesPropiasB11(extractor, revalidador, resolutor, relojB11PreparadorPrueba{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = preparador.PrepararOrdenConsultaParticipacionesPropias(context.Background()); !errors.Is(err, ErrPreparadorParticipacionesPropiasB11Invalido) || !errors.Is(err, httpinternobolsa.ErrAutenticacionInternaAusente) {
		t.Fatalf("revalidación fallida aceptada: %v", err)
	}
	if extractor.llamadas != 1 || revalidador.llamadas != 1 || resolutor.llamadas != 0 {
		t.Fatalf("llamadas no mínimas: extractor=%d revalidador=%d resolutor=%d", extractor.llamadas, revalidador.llamadas, resolutor.llamadas)
	}
	if revalidador.solicitud.AutenticacionRef != extractor.confirmacion.AutenticacionRef ||
		revalidador.solicitud.SesionRef != extractor.confirmacion.SesionRef {
		t.Fatalf("la fábrica no recibió las referencias canónicas: %+v", revalidador.solicitud)
	}
}

func TestPreparadorB11ConservaIndisponibilidadSinPresentarlaComoRevocacion(t *testing.T) {
	ahora := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	extractor := &extractorConfirmacionB11Prueba{confirmacion: httpseguridad.ConfirmacionPeticionSesion{
		AutenticacionRef:  "aut_" + strings.Repeat("a", 22),
		SesionRef:         "ses_" + strings.Repeat("s", 22),
		SesionValidaHasta: ahora.Add(time.Minute),
	}}
	revalidador := &revalidadorB11PreparadorPrueba{err: puertosvec.ErrRevalidacionAutenticacionActorNoDisponible}
	preparador, err := NuevoPreparadorOrdenParticipacionesPropiasB11(
		extractor, revalidador, &resolutorB11PreparadorPrueba{}, relojB11PreparadorPrueba{ahora: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = preparador.PrepararOrdenConsultaParticipacionesPropias(context.Background())
	if !errors.Is(err, puertosvec.ErrRevalidacionAutenticacionActorNoDisponible) ||
		errors.Is(err, httpinternobolsa.ErrAutenticacionInternaAusente) {
		t.Fatalf("indisponibilidad mal clasificada: %v", err)
	}
}

func TestNuevoPreparadorB11RechazaDependenciasAusentes(t *testing.T) {
	if _, err := NuevoPreparadorOrdenParticipacionesPropiasB11(nil, nil, nil, nil); !errors.Is(err, ErrPreparadorParticipacionesPropiasB11Invalido) {
		t.Fatalf("dependencias ausentes aceptadas: %v", err)
	}
}
