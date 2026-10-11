package mibolsa

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type escrituraPreferenciasPrueba struct {
	llamadas  int
	solicitud bolsa.OrdenarPreferenciasBolsas
}

func (r *escrituraPreferenciasPrueba) OrdenarPreferenciasBolsas(_ context.Context,
	s bolsa.OrdenarPreferenciasBolsas,
) (bolsa.ReciboPreferenciasBolsas, error) {
	r.llamadas++
	r.solicitud = s
	return bolsa.ReciboPreferenciasBolsas{ReciboRef: "recibo:preferencias-bolsas:prueba",
		Version: s.VersionEsperada + 1, Orden: slices.Clone(s.Orden), RegistradoEn: s.ConsultadaEn}, nil
}

type proveedorPreferenciasPrueba struct{ emisor *emisorObservado }

func (p proveedorPreferenciasPrueba) EmitirMaterialPreferenciasBolsas(ctx context.Context,
	s domain.SolicitudAutorizacionLigadaV3, r domain.ResultadoContextoActorRegistradoV2,
	d domain.DecisionAutorizacionLigadaV3, c ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
) (ports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	return p.emisor.EmitirMaterialMiBolsa(ctx, s, r, d, c)
}

func nuevoServicioPreferenciasPrueba(t *testing.T) (*entornoMiBolsa, *ServicioPreferenciasBolsas, *escrituraPreferenciasPrueba) {
	t.Helper()
	e := nuevoEntorno(t)
	concesion := &e.fuente.instantanea.VersionRol.Concesiones[0]
	concesion.Accion = bolsa.AccionOrdenarPreferenciasBolsas
	concesion.Finalidades = []string{bolsa.FinalidadPreferenciasBolsas}
	concesion.CamposPermitidos = nil
	reloj := &relojAutorizacionServicioPrueba{ahora: e.ahora}
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(
		"clave:capacidad:preferencias", 1, bytes.Repeat([]byte{0x73}, 32),
		"emisor:prueba:preferencias", bolsa.AudienciaOrdenarPreferenciasBolsas,
		confianza.EstadoClaveHMACCapacidadAtestacionV3Emision,
		e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("9", 64))
	exigir(t, err)
	e.emisor.capacidades, err = confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
	exigir(t, err)
	repositorio := new(escrituraPreferenciasPrueba)
	servicio, err := NuevoServicioPreferenciasBolsas(repositorio, e.autorizador,
		proveedorPreferenciasPrueba{e.emisor}, reloj)
	exigir(t, err)
	return e, servicio, repositorio
}

func TestPreferenciasEscrituraNominalV3Propia(t *testing.T) {
	e, servicio, repo := nuevoServicioPreferenciasPrueba(t)
	orden := []string{"bolsa:b", "bolsa:a"}
	recibo, err := servicio.Ordenar(t.Context(), e.orden, orden, 2, "clave-preferencia-1")
	if err != nil || repo.llamadas != 1 || e.emisor.llamadas != 1 || e.emisor.exportaciones != 1 ||
		recibo.Version != 3 || repo.solicitud.CandidatoRef != referenciaServicioContextoActorPrueba("can_", "c") ||
		repo.solicitud.Material.ValidarEstructura() != nil || !slices.Equal(repo.solicitud.Orden, orden) {
		t.Fatalf("escritura nominal incompleta: %+v %v", recibo, err)
	}
	datos, err := e.concesiones.orden.Datos()
	exigir(t, err)
	nominal, err := datos.Solicitud.Datos()
	exigir(t, err)
	if nominal.Accion != bolsa.AccionOrdenarPreferenciasBolsas || nominal.Finalidad != bolsa.FinalidadPreferenciasBolsas ||
		nominal.Recurso.Referencia != "mi-bolsa:"+repo.solicitud.CandidatoRef {
		t.Fatal("decisión V3 no ligada al candidato propio")
	}
}

func TestPreferenciasNoEscribenConOrdenDuplicadoNiContextoAjeno(t *testing.T) {
	e, servicio, repo := nuevoServicioPreferenciasPrueba(t)
	if _, err := servicio.Ordenar(t.Context(), e.orden, []string{"bolsa:a", "bolsa:a"}, 0,
		"clave-preferencia-2"); !errors.Is(err, bolsa.ErrOrdenPreferenciasInvalido) || repo.llamadas != 0 {
		t.Fatalf("orden duplicado alcanzó la escritura: %v", err)
	}
	ordenAjena := e.orden
	ordenAjena.Vinculo = domain.VinculoAutenticacionActorV2{}
	if _, err := servicio.Ordenar(t.Context(), ordenAjena, []string{"bolsa:a"}, 0,
		"clave-preferencia-3"); !errors.Is(err, domain.ErrAutorizacionDenegada) || repo.llamadas != 0 {
		t.Fatalf("contexto ajeno alcanzó la escritura: %v", err)
	}
}
