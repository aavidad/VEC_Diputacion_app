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

const claveEntregaPeticionCentroPrueba = "12345678-1234-4234-8234-123456789abc"

func TestEntregaPeticionCentroAltaUnaVezYReplayNoVuelveAAlta(t *testing.T) {
	repo, registro, servicio, comando, preparado := entregaPeticionCentroPrueba(t)

	primera, err := servicio.Entregar(context.Background(), comando)
	if err != nil {
		t.Fatalf("primera entrega: %v", err)
	}
	if registro.llamadas != 1 || repo.confirmaciones != 1 {
		t.Fatalf("seam invocado con cardinalidad incorrecta: alta=%d confirmación=%d", registro.llamadas, repo.confirmaciones)
	}
	if primera.EstadoEntrega != "confirmada" || primera.ClaveAlta != preparado.ClaveAlta {
		t.Fatalf("entrega no confirmada: %#v", primera)
	}

	repo.preparada.EstadoEntrega = "confirmada"
	repo.preparada.ReciboAlta = primera.ReciboAlta
	segunda, err := servicio.Entregar(context.Background(), comando)
	if err != nil || !reflect.DeepEqual(segunda, primera) {
		t.Fatalf("replay no devuelve la confirmación original: entrega=%#v error=%v", segunda, err)
	}
	if registro.llamadas != 1 || repo.confirmaciones != 1 {
		t.Fatalf("el replay volvió a producir efectos: alta=%d confirmación=%d", registro.llamadas, repo.confirmaciones)
	}
}

func TestEntregaPeticionCentroConservaFuenteRatificadaYReservaActorPerfil(t *testing.T) {
	repo, registro, servicio, comando, preparado := entregaPeticionCentroPrueba(t)
	original := preparado.Peticion

	_, err := servicio.Entregar(context.Background(), comando)
	if err != nil {
		t.Fatalf("entrega: %v", err)
	}
	registro.recibido.Peticion.Solicitud.DocumentosAdjuntos[0] = "documento:alterado"
	if !reflect.DeepEqual(repo.preparada.Peticion, original) {
		t.Fatal("el alta pudo mutar la petición ratificada original")
	}
	if repo.ultimaComando.VersionEsperada != 2 ||
		repo.preparada.ClaveAlta != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" ||
		repo.preparada.ActorRef != "actor:rrhh" || repo.preparada.PerfilRef != "perfil:rrhh" {
		t.Fatalf("reserva incompleta o no única: %#v", repo.preparada)
	}
}

func TestEntregaPeticionCentroFalloAltaNoConfirmaYFalloEnlacePermiteReintento(t *testing.T) {
	t.Run("fallo alta", func(t *testing.T) {
		repo, registro, servicio, comando, _ := entregaPeticionCentroPrueba(t)
		registro.err = errors.New("alta no disponible")
		if _, err := servicio.Entregar(context.Background(), comando); !errors.Is(err, registro.err) {
			t.Fatalf("error de alta perdido: %v", err)
		}
		if repo.confirmaciones != 0 {
			t.Fatalf("se confirmó tras fallar el alta: %d", repo.confirmaciones)
		}
	})

	t.Run("fallo enlace", func(t *testing.T) {
		repo, registro, servicio, comando, _ := entregaPeticionCentroPrueba(t)
		repo.errConfirmar = errors.New("enlace interrumpido")
		if _, err := servicio.Entregar(context.Background(), comando); !errors.Is(err, repo.errConfirmar) {
			t.Fatalf("error de enlace perdido: %v", err)
		}
		repo.errConfirmar = nil
		segunda, err := servicio.Entregar(context.Background(), comando)
		if err != nil || segunda.EstadoEntrega != "confirmada" {
			t.Fatalf("reintento no recuperable: entrega=%#v error=%v", segunda, err)
		}
		if registro.llamadas != 2 || repo.confirmaciones != 2 || registro.recibo.ReciboRef != "recibo:alta:001" ||
			registro.recibido.ClaveAlta != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" ||
			!reflect.DeepEqual(registro.recibido.Peticion, repo.preparada.Peticion) {
			t.Fatalf("reintento no conservó clave/recibo: alta=%d confirmación=%d recibido=%#v", registro.llamadas, repo.confirmaciones, registro.recibido)
		}
	})
}

func TestEntregaPeticionCentroDeniegaDivergentesYCancelacion(t *testing.T) {
	repo, registro, servicio, comando, _ := entregaPeticionCentroPrueba(t)
	divergente := comando
	divergente.PeticionRef = "peticion:centro:otra"
	if _, err := servicio.Entregar(context.Background(), divergente); !errors.Is(err, ports.ErrReciboPeticionCentroNoConfiable) {
		t.Fatalf("petición divergente aceptada: %v", err)
	}
	if repo.preparaciones != 1 || registro.llamadas != 0 || repo.confirmaciones != 0 {
		t.Fatalf("la divergencia produjo alta/confirmación: %#v alta=%d", repo, registro.llamadas)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := servicio.Entregar(ctx, comando); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación no propagada: %v", err)
	}
	if repo.preparaciones != 1 || registro.llamadas != 0 {
		t.Fatalf("la cancelación produjo efectos: %#v alta=%d", repo, registro.llamadas)
	}
}

type repositorioEntregaPeticionCentroPrueba struct {
	preparada      ports.EntregaPeticionCentro
	ultimaComando  ports.ComandoEntregarPeticionCentro
	confirmaciones int
	preparaciones  int
	errConfirmar   error
}

func (r *repositorioEntregaPeticionCentroPrueba) PrepararEntrega(_ context.Context, comando ports.ComandoEntregarPeticionCentro) (ports.EntregaPeticionCentro, error) {
	r.preparaciones++
	r.ultimaComando = comando
	return r.preparada, nil
}
func (r *repositorioEntregaPeticionCentroPrueba) ConfirmarEntrega(_ context.Context, _ ports.ComandoEntregarPeticionCentro, alta ports.AltaDePeticionCentro) (ports.EntregaPeticionCentro, error) {
	r.confirmaciones++
	if r.errConfirmar != nil {
		return ports.EntregaPeticionCentro{}, r.errConfirmar
	}
	r.preparada.EstadoEntrega = "confirmada"
	r.preparada.ReciboAlta = &alta.Recibo
	return r.preparada, nil
}
func (r *repositorioEntregaPeticionCentroPrueba) ListarPeticionesRRHH(context.Context) ([]ports.EntregaPeticionCentro, error) {
	return nil, nil
}

type registradorEntregaPeticionCentroPrueba struct {
	llamadas int
	err      error
	recibido ports.EntregaPeticionCentro
	recibo   ports.ReciboAlta
}

func (r *registradorEntregaPeticionCentroPrueba) RegistrarExpedientePeticion(_ context.Context, peticion ports.EntregaPeticionCentro) (ports.AltaDePeticionCentro, error) {
	r.llamadas++
	r.recibido = peticion
	if r.err != nil {
		return ports.AltaDePeticionCentro{}, r.err
	}
	r.recibo = reciboAltaPeticionCentroPruebaApplication()
	return ports.AltaDePeticionCentro{Recibo: r.recibo, AmbitoHMAC: selloEntregaPeticionCentroPruebaApplication}, nil
}

const selloEntregaPeticionCentroPruebaApplication = "hmac-sha256:contratacion-temporal.entrega-alta/v2:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func reciboAltaPeticionCentroPruebaApplication() ports.ReciboAlta {
	return ports.ReciboAlta{ExpedienteRef: "expediente:ct:001", NumeroVisible: "2026/5487", Version: 1,
		ReciboRef: "recibo:alta:001", AuditoriaRef: "auditoria:alta:001", EventoRef: "evento:alta:001",
		ConfirmadaEn: time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)}
}

func entregaPeticionCentroPrueba(t *testing.T) (*repositorioEntregaPeticionCentroPrueba, *registradorEntregaPeticionCentroPrueba, *ServicioEntregaPeticionCentro, ports.ComandoEntregarPeticionCentro, ports.EntregaPeticionCentro) {
	t.Helper()
	peticion := peticionCentroRatificadaPrueba(t)
	preparada := ports.EntregaPeticionCentro{Peticion: peticion, EstadoEntrega: "preparada", ClaveAlta: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", AmbitoAltaHMAC: selloEntregaPeticionCentroPruebaApplication, ActorRef: "actor:rrhh", PerfilRef: "perfil:rrhh"}
	repo := &repositorioEntregaPeticionCentroPrueba{preparada: preparada}
	registro := &registradorEntregaPeticionCentroPrueba{}
	servicio, err := NuevoServicioEntregaPeticionCentro(repo, registro)
	if err != nil {
		t.Fatal(err)
	}
	comando := ports.ComandoEntregarPeticionCentro{PeticionRef: peticion.Referencia, VersionEsperada: 2}
	return repo, registro, servicio, comando, preparada
}

func peticionCentroRatificadaPrueba(t *testing.T) domain.DatosPeticionCentro {
	t.Helper()
	config := domain.ConfiguracionPeticionCentro{Referencia: "config:peticion:001", Version: 1,
		Solicitante: domain.ActorPeticionCentro{ActorRef: "actor:solicitante", PerfilRef: "perfil:responsable", CentroRef: "centro:sintetico:001", PuestoRef: "puesto:jefatura"},
		Ratificador: domain.ActorPeticionCentro{ActorRef: "actor:ratificador", PerfilRef: "perfil:responsable", CentroRef: "centro:sintetico:001", PuestoRef: "puesto:direccion"}}
	inicio := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	peticion, err := domain.NuevaPeticionCentro("peticion:centro:001", config, domain.SolicitudCentro{CentroRef: "centro:sintetico:001", ContactoRef: "contacto:sintetico:001", CategoriaRef: "categoria:tecnica", GrupoSubgrupo: "A1", MotivoClave: "necesidad.temporal", Detalle: "Necesidad sintética", Periodo: domain.PeriodoPrevisto{Inicio: inicio, Fin: inicio.Add(24 * time.Hour)}, DocumentosAdjuntos: []string{"documento:001"}}, inicio)
	if err != nil {
		t.Fatal(err)
	}
	ratificada, err := peticion.Ratificar(config.Ratificador, 1, "Necesidad confirmada", inicio.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	return ratificada.Datos()
}
