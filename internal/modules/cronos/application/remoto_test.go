package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type consultaTeletrabajoPrueba struct {
	periodo   domain.PeriodoTeletrabajo
	permitido bool
	err       error
	llamadas  int
	empleado  string
	instante  time.Time
}

func (c *consultaTeletrabajoPrueba) ConsultarTeletrabajoPropio(_ context.Context, _ vecdomain.ContextoActor, empleado string, instante time.Time) (domain.PeriodoTeletrabajo, bool, error) {
	c.llamadas++
	c.empleado, c.instante = empleado, instante
	return c.periodo, c.permitido, c.err
}

type repositorioRemotoPrueba struct {
	llamadas             int
	recuperaciones       int
	recuperable          bool
	consultasContinuidad int
	continuidad          bool
	movimientos          []domain.PunchKind
	continuidadErr       error
	claveConsultada      string
}

func (r *repositorioRemotoPrueba) ConfirmarContinuidadMarcajeRemoto(_ context.Context, _ vecdomain.ContextoActor, _ string, _ domain.PeriodoTeletrabajo, clave string) (ports.EstadoSecuenciaMarcajeRemoto, error) {
	r.consultasContinuidad++
	r.claveConsultada = clave
	return ports.EstadoSecuenciaMarcajeRemoto{ContinuidadConfirmada: r.continuidad, MovimientosPermitidos: r.movimientos}, r.continuidadErr
}

func (r *repositorioRemotoPrueba) RegistrarOriginalRemotoAutorizado(context.Context, domain.MarcajeOriginal, domain.MaterialAutorizacionMarcajePropio, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboMarcajePropio, error) {
	r.llamadas++
	return ports.ReciboMarcajePropio{}, nil
}

func (r *repositorioRemotoPrueba) RecuperarOriginalRemotoAutorizado(_ context.Context, m domain.MaterialRecuperacionMarcajeRemoto, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboMarcajePropio, error) {
	r.recuperaciones++
	if r.recuperable {
		return ports.ReciboMarcajePropio{Referencia: "recibo:cronos:prueba", MarcajeOriginalRef: "marcaje:cronos:" + m.ClaveOperacion, InstanteUTC: time.Now().UTC().Truncate(time.Microsecond), Replay: true}, nil
	}
	return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
}

type proveedorLecturaRemotaPrueba struct {
	llamadas            int
	material            domain.MaterialRecuperacionMarcajeRemoto
	materialEstructural bool
}

type relojRemotoPunteroPrueba struct{ instante time.Time }

func (r *relojRemotoPunteroPrueba) AhoraUTC() time.Time { return r.instante }

type proveedorEscrituraRemotaPunteroPrueba struct{}

func (p *proveedorEscrituraRemotaPunteroPrueba) ProveerMaterialMarcajePropio(context.Context, domain.MaterialAutorizacionMarcajePropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}

func (p *proveedorLecturaRemotaPrueba) ProveerMaterialRecuperacionMarcajeRemoto(_ context.Context, m domain.MaterialRecuperacionMarcajeRemoto) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	p.material = m
	if p.materialEstructural {
		r, err := RecursoRecuperacionMarcajeRemoto(m)
		if err != nil {
			return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
		}
		huella, err := r.HuellaContextoAutorizacionSHA256()
		if err != nil {
			return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
		}
		ahora := time.Now().UTC().Truncate(time.Microsecond)
		resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), AccionRecuperarMarcajeRemoto, r.Referencia, huella, AudienciaRecuperacionMarcajeRemoto, ahora, ahora.Add(3*time.Second))
		if err != nil {
			return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
		}
		raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
		if err != nil {
			return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
		}
		return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	}
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}

func contextoRecuperacionRemotaPrueba(t *testing.T, proveedor ports.ProveedorMaterialRecuperacionMarcajeRemoto) ports.ContextoRecuperacionMarcajeRemoto {
	t.Helper()
	c := contextoRemotoPrueba(t)
	actor, err := c.OrdenConsumo.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenLecturaMarcajeRemoto(actor, proveedor)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoRecuperacionMarcajeRemoto{CanalAcreditado: c.CanalAcreditado, OrdenLectura: orden}
}

func contextoRemotoPrueba(t *testing.T) ports.ContextoMarcajePropio {
	t.Helper()
	c := contexto(t)
	canal, err := domain.NuevaAcreditacionCanalMarcaje(domain.DatosAcreditacionCanalMarcaje{
		PoliticaVersionRef: "pol_v1", CanalRef: "canal_remoto", OrigenRef: domain.OrigenMarcajeRemoto, CalidadRef: "nivel_1",
	})
	if err != nil {
		t.Fatal(err)
	}
	c.CanalAcreditado = canal
	return c
}

func TestMarcajeRemotoExigeOrigenRemotoYAutorizacionPositiva(t *testing.T) {
	c := contextoRemotoPrueba(t)
	instante := time.Now().UTC().Truncate(time.Microsecond)
	consulta := &consultaTeletrabajoPrueba{}
	repo := &repositorioRemotoPrueba{}
	s, err := NuevoServicioMarcajesRemotos(repo, consulta, relojMarcajePrueba{instante})
	if err != nil {
		t.Fatal(err)
	}
	local := contexto(t)
	_, err = s.RegistrarMarcajeRemoto(context.Background(), local, ports.SolicitudMarcajePropio{Movimiento: domain.PunchEntry, ClaveOperacion: "op_remota_1"})
	if !errors.Is(err, ErrContextoMarcajeNoAcreditado) || consulta.llamadas != 0 || repo.llamadas != 0 {
		t.Fatal("un canal no remoto consulto o escribio", err)
	}
	_, err = s.RegistrarMarcajeRemoto(context.Background(), c, ports.SolicitudMarcajePropio{Movimiento: domain.PunchEntry, ClaveOperacion: "op_remota_1"})
	if !errors.Is(err, ErrTeletrabajoNoAutorizado) || consulta.llamadas != 1 || repo.llamadas != 0 {
		t.Fatal("teletrabajo no autorizado alcanzo escritura", err)
	}
	if consulta.empleado != "emp_0123456789abcdefghijkl" || !consulta.instante.Equal(instante) {
		t.Fatal("la consulta no uso empleado propio y hora del servidor")
	}
}

func TestDisponibilidadRemotaNoRevelaPeriodoSinAutorizacion(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	consulta := &consultaTeletrabajoPrueba{periodo: domain.PeriodoTeletrabajo{DesdeUTC: instante.Add(-time.Hour), HastaUTC: instante.Add(time.Hour)}}
	repo := &repositorioRemotoPrueba{}
	s, _ := NuevoServicioMarcajesRemotos(repo, consulta, relojMarcajePrueba{instante})
	respuesta, err := s.ConsultarDisponibilidadMarcajeRemoto(context.Background(), contextoRemotoPrueba(t))
	if err != nil || respuesta.Autorizado || respuesta.ContinuidadConfirmada || respuesta.Periodo != nil || respuesta.Motivo != "teletrabajo_no_autorizado" || repo.consultasContinuidad != 0 || respuesta.MovimientosPermitidos == nil {
		t.Fatal("la denegacion revelo un periodo", respuesta, err)
	}
	consulta.permitido = true
	respuesta, err = s.ConsultarDisponibilidadMarcajeRemoto(context.Background(), contextoRemotoPrueba(t))
	if err != nil || !respuesta.Autorizado || respuesta.ContinuidadConfirmada || respuesta.Motivo != "continuidad_no_confirmada" || respuesta.Periodo == nil || !respuesta.Periodo.DesdeUTC.Equal(consulta.periodo.DesdeUTC) || repo.consultasContinuidad != 1 {
		t.Fatal("periodo propio autorizado no disponible", respuesta, err)
	}
	repo.continuidad = true
	respuesta, err = s.ConsultarDisponibilidadMarcajeRemoto(context.Background(), contextoRemotoPrueba(t))
	if err != nil || !respuesta.ContinuidadConfirmada || respuesta.Motivo != "secuencia_no_permitida" || len(respuesta.MovimientosPermitidos) != 0 || repo.consultasContinuidad != 2 || repo.claveConsultada != "" {
		t.Fatal("continuidad sola habilito una secuencia", respuesta, err)
	}
	repo.movimientos = []domain.PunchKind{domain.PunchEntry}
	respuesta, err = s.ConsultarDisponibilidadMarcajeRemoto(context.Background(), contextoRemotoPrueba(t))
	if err != nil || !respuesta.ContinuidadConfirmada || respuesta.Motivo != "autorizado" || len(respuesta.MovimientosPermitidos) != 1 || respuesta.MovimientosPermitidos[0] != domain.PunchEntry || repo.consultasContinuidad != 3 || repo.claveConsultada != "" {
		t.Fatal("continuidad no procedio del repositorio nominal", respuesta, err)
	}
	repo.continuidadErr = errors.New("conciliacion no disponible")
	respuesta, err = s.ConsultarDisponibilidadMarcajeRemoto(context.Background(), contextoRemotoPrueba(t))
	if !errors.Is(err, ports.ErrDependenciaNoDisponible) || respuesta.ContinuidadConfirmada {
		t.Fatal("fallo de conciliacion publicado como disponibilidad", respuesta, err)
	}
}

func TestMarcajeRemotoExigeMovimientoPermitidoDelRepositorio(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	consulta := &consultaTeletrabajoPrueba{permitido: true, periodo: domain.PeriodoTeletrabajo{DesdeUTC: instante.Add(-time.Hour), HastaUTC: instante.Add(time.Hour)}}
	repo := &repositorioRemotoPrueba{continuidad: true}
	s, _ := NuevoServicioMarcajesRemotos(repo, consulta, relojMarcajePrueba{instante})
	_, err := s.RegistrarMarcajeRemoto(context.Background(), contextoRemotoPrueba(t), ports.SolicitudMarcajePropio{Movimiento: domain.PunchEntry, ClaveOperacion: "op_remota_6"})
	if !errors.Is(err, ErrMovimientoRemotoNoPermitido) || repo.llamadas != 0 {
		t.Fatal("continuidad sin movimientos permitio escritura", err)
	}
	repo.movimientos = []domain.PunchKind{domain.PunchExit}
	_, err = s.RegistrarMarcajeRemoto(context.Background(), contextoRemotoPrueba(t), ports.SolicitudMarcajePropio{Movimiento: domain.PunchEntry, ClaveOperacion: "op_remota_7"})
	if !errors.Is(err, ErrMovimientoRemotoNoPermitido) || repo.llamadas != 0 {
		t.Fatal("movimiento fuera de secuencia alcanzo AD3-51", err)
	}
	repo.movimientos = []domain.PunchKind{domain.PunchEntry, domain.PunchEntry}
	_, err = s.ConsultarDisponibilidadMarcajeRemoto(context.Background(), contextoRemotoPrueba(t))
	if !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("lista duplicada aceptada", err)
	}
	repo.movimientos = []domain.PunchKind{domain.PunchEntry}
	_, err = s.RegistrarMarcajeRemoto(context.Background(), contextoRemotoPrueba(t), ports.SolicitudMarcajePropio{Movimiento: domain.PunchEntry, ClaveOperacion: "op_remota_8"})
	if !errors.Is(err, ErrContextoMarcajeNoAcreditado) || repo.consultasContinuidad != 4 || repo.llamadas != 0 {
		t.Fatal("movimiento permitido no procedio a validacion V3", err)
	}
}

func TestMarcajeRemotoNoAceptaOtraClaveSinContinuidadDurable(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	consulta := &consultaTeletrabajoPrueba{permitido: true, periodo: domain.PeriodoTeletrabajo{DesdeUTC: instante.Add(-time.Hour), HastaUTC: instante.Add(time.Hour)}}
	repo := &repositorioRemotoPrueba{}
	s, _ := NuevoServicioMarcajesRemotos(repo, consulta, relojMarcajePrueba{instante})
	_, err := s.RegistrarMarcajeRemoto(context.Background(), contextoRemotoPrueba(t), ports.SolicitudMarcajePropio{Movimiento: domain.PunchEntry, ClaveOperacion: "op_remota_4"})
	if !errors.Is(err, ErrContinuidadMarcajeNoConfirmada) || repo.llamadas != 0 || repo.consultasContinuidad != 1 || repo.claveConsultada != "op_remota_4" {
		t.Fatal("teletrabajo autorizado sin continuidad alcanzó AD3-51", err)
	}
	repo.continuidadErr = errors.New("sin conciliacion durable")
	_, err = s.RegistrarMarcajeRemoto(context.Background(), contextoRemotoPrueba(t), ports.SolicitudMarcajePropio{Movimiento: domain.PunchExit, ClaveOperacion: "op_remota_5"})
	if !errors.Is(err, ports.ErrDependenciaNoDisponible) || repo.llamadas != 0 {
		t.Fatal("fallo de conciliacion alcanzo escritura", err)
	}
}

func TestMarcajeRemotoFallaCerradoConFuenteAusenteOPeriodoInvalido(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	consulta := &consultaTeletrabajoPrueba{err: errors.New("fuente no disponible")}
	repo := &repositorioRemotoPrueba{}
	s, _ := NuevoServicioMarcajesRemotos(repo, consulta, relojMarcajePrueba{instante})
	c := contextoRemotoPrueba(t)
	_, err := s.ConsultarDisponibilidadMarcajeRemoto(context.Background(), c)
	if !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("fallo de fuente interpretado como disponibilidad", err)
	}
	consulta.err = nil
	consulta.permitido = true
	consulta.periodo = domain.PeriodoTeletrabajo{DesdeUTC: instante.Add(time.Second), HastaUTC: instante.Add(time.Hour)}
	_, err = s.RegistrarMarcajeRemoto(context.Background(), c, ports.SolicitudMarcajePropio{Movimiento: domain.PunchPauseStart, ClaveOperacion: "op_remota_2"})
	if !errors.Is(err, ports.ErrDependenciaNoDisponible) || repo.llamadas != 0 {
		t.Fatal("periodo fuera de vigencia alcanzo escritura", err)
	}
}

func TestMarcajeRemotoReutilizaValidacionV3DeAD351(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	consulta := &consultaTeletrabajoPrueba{permitido: true, periodo: domain.PeriodoTeletrabajo{DesdeUTC: instante.Add(-time.Hour), HastaUTC: instante.Add(time.Hour)}}
	repo := &repositorioRemotoPrueba{continuidad: true, movimientos: []domain.PunchKind{domain.PunchExit}}
	s, _ := NuevoServicioMarcajesRemotos(repo, consulta, relojMarcajePrueba{instante})
	_, err := s.RegistrarMarcajeRemoto(context.Background(), contextoRemotoPrueba(t), ports.SolicitudMarcajePropio{Movimiento: domain.PunchExit, ClaveOperacion: "op_remota_3"})
	if !errors.Is(err, ErrContextoMarcajeNoAcreditado) || repo.llamadas != 0 || repo.consultasContinuidad != 1 {
		t.Fatal("V3 no atestado alcanzo el repositorio remoto", err)
	}
}

func TestRecuperacionRemotaTrasRevocacionNoExigeTeletrabajoDeEscritura(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	consulta := &consultaTeletrabajoPrueba{permitido: false}
	repo := &repositorioRemotoPrueba{}
	proveedor := &proveedorLecturaRemotaPrueba{}
	s, _ := NuevoServicioMarcajesRemotos(repo, consulta, relojMarcajePrueba{instante})
	_, err := s.RecuperarReciboMarcajeRemoto(context.Background(), contextoRecuperacionRemotaPrueba(t, proveedor), ports.SolicitudMarcajePropio{
		Movimiento: domain.PunchEntry, ClaveOperacion: "op_remota_9",
	})
	if !errors.Is(err, ErrContextoMarcajeNoAcreditado) || proveedor.llamadas != 1 || consulta.llamadas != 0 ||
		repo.consultasContinuidad != 0 || repo.llamadas != 0 || repo.recuperaciones != 0 {
		t.Fatal("recuperacion requirio escritura o acepto V3 vacio", err)
	}
	if proveedor.material.EmpleadoRef != "emp_0123456789abcdefghijkl" || proveedor.material.ClaveOperacion != "op_remota_9" || proveedor.material.Movimiento != domain.PunchEntry {
		t.Fatal("material de lectura no ligo actor y marcaje propio")
	}
	// La exportación sintética sólo prueba el cableado del caso de uso; el
	// repositorio real debe validar firmas, ACL, lock y auditoría en PostgreSQL.
	proveedor.materialEstructural = true
	repo.recuperable = true
	materialPrueba, err := proveedor.ProveerMaterialRecuperacionMarcajeRemoto(context.Background(), proveedor.material)
	if err != nil || materialPrueba.ValidarEstructura() != nil || !recuperacionV3Ligada(materialPrueba, proveedor.material) {
		t.Fatal("fixture estructural V3 de lectura invalida", err, materialPrueba.ValidarEstructura())
	}
	recibo, err := s.RecuperarReciboMarcajeRemoto(context.Background(), contextoRecuperacionRemotaPrueba(t, proveedor), ports.SolicitudMarcajePropio{
		Movimiento: domain.PunchEntry, ClaveOperacion: "op_remota_9",
	})
	if err != nil || !recibo.Replay || repo.recuperaciones != 1 || repo.llamadas != 0 || consulta.llamadas != 0 {
		t.Fatal("recibo propio no recuperado tras revocacion simulada", recibo, err)
	}
}

func TestRecursoRecuperacionRemotaLigaMovimientoYEmpleado(t *testing.T) {
	canal := contextoRemotoPrueba(t).CanalAcreditado
	m := domain.MaterialRecuperacionMarcajeRemoto{
		ActorRef: "per_0123456789abcdefghijkl", PerfilRef: "prf_0123456789abcdefghijkl",
		EmpleadoRef: "emp_0123456789abcdefghijkl", ClaveOperacion: "op_remota_10",
		Movimiento: domain.PunchEntry, Canal: canal,
	}
	r1, err := RecursoRecuperacionMarcajeRemoto(m)
	if err != nil {
		t.Fatal(err)
	}
	h1, err := r1.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	m.Movimiento = domain.PunchExit
	r2, err := RecursoRecuperacionMarcajeRemoto(m)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := r2.HuellaContextoAutorizacionSHA256()
	if err != nil || h1 == h2 {
		t.Fatal("lectura no ligo movimiento exacto", err)
	}
}

func TestDependenciasRemotasConPunteroTipadoNilFallanCerradas(t *testing.T) {
	actor, err := contexto(t).OrdenConsumo.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	var proveedorLectura *proveedorLecturaRemotaPrueba
	if _, err := ports.NuevaOrdenLecturaMarcajeRemoto(actor, proveedorLectura); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("orden acepto proveedor de lectura typed nil", err)
	}
	var repositorio *repositorioRemotoPrueba
	var consulta *consultaTeletrabajoPrueba
	var reloj *relojRemotoPunteroPrueba
	for _, caso := range []struct {
		nombre   string
		repo     ports.RepositorioMarcajesRemotos
		consulta ports.ConsultaAutorizacionTeletrabajo
		reloj    ports.Reloj
	}{
		{"repositorio", repositorio, &consultaTeletrabajoPrueba{}, relojMarcajePrueba{time.Now().UTC()}},
		{"consulta", &repositorioRemotoPrueba{}, consulta, relojMarcajePrueba{time.Now().UTC()}},
		{"reloj", &repositorioRemotoPrueba{}, &consultaTeletrabajoPrueba{}, reloj},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if servicio, err := NuevoServicioMarcajesRemotos(caso.repo, caso.consulta, caso.reloj); servicio != nil || !errors.Is(err, ports.ErrDependenciaNoDisponible) {
				t.Fatal("constructor acepto dependencia typed nil", servicio, err)
			}
		})
	}
	var proveedorEscritura *proveedorEscrituraRemotaPunteroPrueba
	orden, err := ports.NuevaOrdenConsumoAutorizacion(actor, proveedorEscritura)
	if err != nil {
		t.Fatal(err)
	} // Contrato AD3-51 existente; remoto añade su propia guarda.
	c := contextoRemotoPrueba(t)
	c.OrdenConsumo = orden
	s, _ := NuevoServicioMarcajesRemotos(&repositorioRemotoPrueba{}, &consultaTeletrabajoPrueba{}, relojMarcajePrueba{time.Now().UTC()})
	_, err = s.RegistrarMarcajeRemoto(context.Background(), c, ports.SolicitudMarcajePropio{Movimiento: domain.PunchEntry, ClaveOperacion: "op_remota_nil"})
	if !errors.Is(err, ErrContextoMarcajeNoAcreditado) {
		t.Fatal("proveedor de escritura typed nil alcanzo llamada", err)
	}
}
