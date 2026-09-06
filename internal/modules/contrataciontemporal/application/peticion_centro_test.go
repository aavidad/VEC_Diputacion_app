package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadPeticionCentroPrueba struct {
	actor  domain.ActorPeticionCentro
	config domain.ConfiguracionPeticionCentro
}

func (a *autoridadPeticionCentroPrueba) ActorPeticionCentro(context.Context) (domain.ActorPeticionCentro, error) {
	return a.actor, nil
}
func (a *autoridadPeticionCentroPrueba) ConfiguracionPeticionCentro(context.Context, domain.ActorPeticionCentro) (domain.ConfiguracionPeticionCentro, error) {
	return a.config, nil
}

type repositorioPeticionCentroPrueba struct {
	operaciones map[string]ports.MaterialPeticionCentro
	peticiones  map[string]domain.DatosPeticionCentro
	confirmadas int
	ultimo      ports.MaterialPeticionCentro
	reciboMalo  bool
}

func (r *repositorioPeticionCentroPrueba) ConsultarOperacion(_ context.Context, _ domain.ActorPeticionCentro, clave string) (*ports.MaterialPeticionCentro, error) {
	m, existe := r.operaciones[clave]
	if !existe {
		return nil, nil
	}
	return &m, nil
}
func (r *repositorioPeticionCentroPrueba) ObtenerPeticion(_ context.Context, _ domain.ActorPeticionCentro, ref string) (domain.DatosPeticionCentro, error) {
	return r.peticiones[ref], nil
}
func (r *repositorioPeticionCentroPrueba) ConfirmarPeticion(_ context.Context, m ports.MaterialPeticionCentro) (ports.ReciboPeticionCentro, error) {
	r.confirmadas++
	r.ultimo = m
	r.operaciones[m.Comando.ClaveIdempotencia] = m
	r.peticiones[m.Peticion.Referencia] = m.Peticion
	if r.reciboMalo {
		return ports.ReciboPeticionCentro{}, nil
	}
	registrado := m.Peticion.CreadaEn.Add(time.Minute)
	if m.Peticion.RatificadaEn.After(registrado) {
		registrado = m.Peticion.RatificadaEn
	}
	estado := "registrado"
	if r.confirmadas > 1 {
		estado = "replay_confirmado"
	}
	return ports.ReciboPeticionCentro{ReciboRef: "recibo:peticion:001", PeticionRef: m.Peticion.Referencia,
		Version: m.Peticion.Version, Estado: m.Peticion.Estado, ActorRef: m.Actor.ActorRef,
		RegistradoEn: registrado, EstadoLocal: estado}, nil
}

type relojPeticionCentroPrueba struct{ ahora time.Time }

func (r relojPeticionCentroPrueba) Ahora() time.Time { return r.ahora }

func TestServicioPeticionCentroPresentaRatificaYReplayConfirma(t *testing.T) {
	servicio, autoridad, repo, solicitud, ahora := servicioPeticionCentroPrueba(t)
	clave := "12345678-1234-4234-8234-123456789abc"
	presentar := ports.ComandoPeticionCentro{Operacion: ports.OperacionPresentarPeticionCentro, ClaveIdempotencia: clave, Solicitud: solicitud}
	recibo, err := servicio.Ejecutar(context.Background(), presentar)
	if err != nil || recibo.Version != 1 || repo.ultimo.Actor != autoridad.config.Solicitante {
		t.Fatalf("presentar: recibo=%#v material=%#v error=%v", recibo, repo.ultimo, err)
	}
	original := repo.ultimo
	replay, err := servicio.Ejecutar(context.Background(), presentar)
	if err != nil || replay.EstadoLocal != "replay_confirmado" || repo.confirmadas != 2 || !reflect.DeepEqual(repo.ultimo, original) {
		t.Fatalf("replay no reconfirmado con material original: %#v, %v", replay, err)
	}
	solicitud.DocumentosAdjuntos[0] = "documento:alterado"
	if original.Peticion.Solicitud.DocumentosAdjuntos[0] != "documento:001" || original.Comando.ClaveIdempotencia != clave {
		t.Fatal("material no conservó clon y UUID originales")
	}
	autoridad.actor = autoridad.config.Ratificador
	servicio.reloj = relojPeticionCentroPrueba{ahora.Add(2 * time.Minute)}
	ratificar := ports.ComandoPeticionCentro{Operacion: ports.OperacionRatificarPeticionCentro,
		ClaveIdempotencia: "abcdefab-cdef-4abc-8def-abcdefabcdef", PeticionRef: recibo.PeticionRef, VersionEsperada: 1, Motivo: "Confirmada"}
	final, err := servicio.Ejecutar(context.Background(), ratificar)
	if err != nil || final.Version != 2 || final.Estado != "ratificada" || repo.ultimo.Actor != autoridad.config.Ratificador {
		t.Fatalf("ratificar: %#v, %v", final, err)
	}
}

func TestServicioPeticionCentroConflictosDenegacionesYFallosNoEscriben(t *testing.T) {
	servicio, autoridad, repo, solicitud, _ := servicioPeticionCentroPrueba(t)
	clave := "12345678-1234-4234-8234-123456789abc"
	base := ports.ComandoPeticionCentro{Operacion: ports.OperacionPresentarPeticionCentro, ClaveIdempotencia: clave, Solicitud: solicitud}
	if _, err := servicio.Ejecutar(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	escritas := repo.confirmadas
	divergente := base
	copia := *solicitud
	copia.Detalle = "Otra necesidad"
	divergente.Solicitud = &copia
	if _, err := servicio.Ejecutar(context.Background(), divergente); !errors.Is(err, ports.ErrClavePeticionCentroUsada) || repo.confirmadas != escritas {
		t.Fatalf("UUID divergente: error=%v confirmaciones=%d", err, repo.confirmadas)
	}
	actorAjeno := autoridad.config.Ratificador
	actorAjeno.PuestoRef = "puesto:otro"
	autoridad.actor = actorAjeno
	ratificar := ports.ComandoPeticionCentro{Operacion: ports.OperacionRatificarPeticionCentro,
		ClaveIdempotencia: "abcdefab-cdef-4abc-8def-abcdefabcdef", PeticionRef: repo.ultimo.Peticion.Referencia, VersionEsperada: 1, Motivo: "Confirmada"}
	if _, err := servicio.Ejecutar(context.Background(), ratificar); !errors.Is(err, domain.ErrRatificacionCentroDenegada) || repo.confirmadas != escritas {
		t.Fatalf("actor ajeno: error=%v confirmaciones=%d", err, repo.confirmadas)
	}
	ratificar.VersionEsperada = 2
	if _, err := servicio.Ejecutar(context.Background(), ratificar); !errors.Is(err, domain.ErrPeticionCentroInvalida) || repo.confirmadas != escritas {
		t.Fatalf("versión inválida: error=%v confirmaciones=%d", err, repo.confirmadas)
	}
	autoridad.actor = autoridad.config.Solicitante
	repo.reciboMalo = true
	base.ClaveIdempotencia = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	if _, err := servicio.Ejecutar(context.Background(), base); !errors.Is(err, ports.ErrReciboPeticionCentroNoConfiable) {
		t.Fatalf("recibo inválido: %v", err)
	}
	antes := repo.confirmadas
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := servicio.Ejecutar(ctx, base); !errors.Is(err, context.Canceled) || repo.confirmadas != antes {
		t.Fatalf("cancelación confirmó: error=%v confirmaciones=%d", err, repo.confirmadas)
	}
}

func servicioPeticionCentroPrueba(t *testing.T) (*ServicioPeticionCentro, *autoridadPeticionCentroPrueba, *repositorioPeticionCentroPrueba, *domain.SolicitudCentro, time.Time) {
	t.Helper()
	centro := "centro:sintetico:001"
	config := domain.ConfiguracionPeticionCentro{Referencia: "config:peticion:001", Version: 1,
		Solicitante: domain.ActorPeticionCentro{ActorRef: "actor:solicitante", PerfilRef: "perfil:responsable", CentroRef: centro, PuestoRef: "puesto:jefatura"},
		Ratificador: domain.ActorPeticionCentro{ActorRef: "actor:ratificador", PerfilRef: "perfil:responsable", CentroRef: centro, PuestoRef: "puesto:direccion"}}
	autoridad := &autoridadPeticionCentroPrueba{actor: config.Solicitante, config: config}
	repo := &repositorioPeticionCentroPrueba{operaciones: map[string]ports.MaterialPeticionCentro{}, peticiones: map[string]domain.DatosPeticionCentro{}}
	ahora := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	inicio := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	solicitud := &domain.SolicitudCentro{CentroRef: centro, ContactoRef: "contacto:sintetico:001", CategoriaRef: "categoria:tecnica",
		GrupoSubgrupo: "A1", MotivoClave: "necesidad.temporal", Detalle: "Necesidad sintética",
		Periodo: domain.PeriodoPrevisto{Inicio: inicio, Fin: inicio.Add(24 * time.Hour)}, DocumentosAdjuntos: []string{"documento:001"}}
	servicio, err := NuevoServicioPeticionCentro(autoridad, repo, relojPeticionCentroPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	return servicio, autoridad, repo, solicitud, ahora
}
