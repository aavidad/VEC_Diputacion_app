package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain"
	"vec-diputacion-granada/internal/modules/administracion/ports"
)

type autorizadorAparienciaPrueba struct {
	llamadas int
	alterar  func(*ports.ConcesionApariencia)
	err      error
}

func (a *autorizadorAparienciaPrueba) ExigirApariencia(_ context.Context, p ports.SolicitudAutorizacionApariencia) (ports.ConcesionApariencia, error) {
	a.llamadas++
	if a.err != nil {
		return ports.ConcesionApariencia{}, a.err
	}
	ahora := time.Now().UTC()
	c := ports.ConcesionApariencia{
		Permitida: true, Accion: p.Accion, RecursoRef: p.RecursoRef,
		AmbitoRef: p.AmbitoRef, Finalidad: p.Finalidad, Tema: p.Tema,
		RevisionGlobalEsperada: p.RevisionGlobalEsperada, ClaveIdempotencia: p.ClaveIdempotencia,
		ActorRef: "actor:publicador", PerfilRef: "perfil:admin", DecisionRef: "decision:tema:1",
		EmitidaEn: ahora.Add(-time.Minute), ExpiraEn: ahora.Add(time.Minute),
	}
	if p.Accion == ports.AccionPublicarApariencia {
		c.AprobacionRef = "aprobacion:tema:1"
		c.AprobadorRef = "actor:aprobador"
	}
	if a.alterar != nil {
		a.alterar(&c)
	}
	return c, nil
}

type registroAparienciaPrueba struct {
	consultas, publicaciones int
	alterarRecibo            func(*ports.ReciboPublicacion)
	alterarConsulta          func(*ports.ResultadoConsultaApariencia)
	err                      error
}

func (r *registroAparienciaPrueba) ConsultarAuditable(_ context.Context, o ports.OrdenConsultaApariencia) (ports.ResultadoConsultaApariencia, error) {
	r.consultas++
	if r.err != nil {
		return ports.ResultadoConsultaApariencia{}, r.err
	}
	resultado := ports.ResultadoConsultaApariencia{
		Estado: domain.EstadoApariencia{}, DecisionRef: o.Concesion.DecisionRef,
		AuditoriaRef: "auditoria:lectura:1", ConsultadaEn: time.Now().UTC(),
	}
	if r.alterarConsulta != nil {
		r.alterarConsulta(&resultado)
	}
	return resultado, nil
}

func (r *registroAparienciaPrueba) PublicarAtomico(_ context.Context, o ports.OrdenConfirmarPublicacion) (ports.ReciboPublicacion, error) {
	r.publicaciones++
	if r.err != nil {
		return ports.ReciboPublicacion{}, r.err
	}
	ahora := time.Now().UTC()
	recibo := ports.ReciboPublicacion{
		Estado: domain.EstadoApariencia{
			Configurado: true, Tema: o.Publicacion.Tema,
			RevisionGlobal: o.Publicacion.RevisionGlobalEsperada + 1,
			ReciboRef:      "recibo:tema:1", HistoriaRef: "historia:tema:1", PublicadaEn: ahora,
		},
		RevisionAnterior:  o.Publicacion.RevisionGlobalEsperada,
		ClaveIdempotencia: o.Publicacion.ClaveIdempotencia,
		ActorRef:          o.Concesion.ActorRef, DecisionRef: o.Concesion.DecisionRef,
		AprobacionRef: o.Concesion.AprobacionRef, AuditoriaRef: "auditoria:publicacion:1",
		EventoRef: "evento:tema:1", ConfirmadaEn: ahora,
	}
	if r.alterarRecibo != nil {
		r.alterarRecibo(&recibo)
	}
	return recibo, nil
}

func ordenTemaPrueba() domain.OrdenPublicacion {
	return domain.OrdenPublicacion{
		Tema:                   domain.Tema{ID: domain.TemaGranate, Revision: 1},
		RevisionGlobalEsperada: 3, ClaveIdempotencia: "tema-publicacion-0001",
	}
}

func TestAparienciaFallaCerradaSinPuertosOConcesionExacta(t *testing.T) {
	var nulo *autorizadorAparienciaPrueba
	if _, err := NuevoServicioApariencia(nulo, &registroAparienciaPrueba{}); !errors.Is(err, ErrAparienciaNoDisponible) {
		t.Fatalf("autorizador tipado nulo: %v", err)
	}
	if _, err := NuevoServicioApariencia(&autorizadorAparienciaPrueba{}, nil); !errors.Is(err, ErrAparienciaNoDisponible) {
		t.Fatalf("registro nulo: %v", err)
	}
	var registroNulo *registroAparienciaPrueba
	if _, err := NuevoServicioApariencia(&autorizadorAparienciaPrueba{}, registroNulo); !errors.Is(err, ErrAparienciaNoDisponible) {
		t.Fatalf("registro tipado nulo: %v", err)
	}
	for nombre, alterar := range map[string]func(*ports.ConcesionApariencia){
		"denegada":       func(c *ports.ConcesionApariencia) { c.Permitida = false },
		"otro recurso":   func(c *ports.ConcesionApariencia) { c.RecursoRef = "vec.otro" },
		"otra version":   func(c *ports.ConcesionApariencia) { c.RevisionGlobalEsperada++ },
		"caducada":       func(c *ports.ConcesionApariencia) { c.ExpiraEn = c.EmitidaEn.Add(time.Second) },
		"sin aprobacion": func(c *ports.ConcesionApariencia) { c.AprobacionRef = "" },
		"mismo actor":    func(c *ports.ConcesionApariencia) { c.AprobadorRef = c.ActorRef },
	} {
		t.Run(nombre, func(t *testing.T) {
			a := &autorizadorAparienciaPrueba{alterar: alterar}
			r := &registroAparienciaPrueba{}
			s, err := NuevoServicioApariencia(a, r)
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.PublicarTemaGlobal(context.Background(), ordenTemaPrueba())
			if !errors.Is(err, ErrAparienciaDenegada) || r.publicaciones != 0 {
				t.Fatalf("efecto no bloqueado: err=%v llamadas=%d", err, r.publicaciones)
			}
		})
	}
	a, r := &autorizadorAparienciaPrueba{}, &registroAparienciaPrueba{}
	s, _ := NuevoServicioApariencia(a, r)
	mala := ordenTemaPrueba()
	mala.Tema.ID = "externo"
	if _, err := s.PublicarTemaGlobal(context.Background(), mala); !errors.Is(err, domain.ErrOrdenPublicacionInvalida) || a.llamadas != 0 {
		t.Fatalf("orden inválida pasó a autorización: err=%v llamadas=%d", err, a.llamadas)
	}
}

func TestPublicarExigeCASAuditoriaEHistoriaEnRecibo(t *testing.T) {
	a, r := &autorizadorAparienciaPrueba{}, &registroAparienciaPrueba{}
	s, _ := NuevoServicioApariencia(a, r)
	recibo, err := s.PublicarTemaGlobal(context.Background(), ordenTemaPrueba())
	if err != nil || recibo.Estado.RevisionGlobal != 4 || r.publicaciones != 1 {
		t.Fatalf("confirmación: recibo=%+v err=%v llamadas=%d", recibo, err, r.publicaciones)
	}
	r.alterarRecibo = func(x *ports.ReciboPublicacion) { x.AuditoriaRef = "" }
	if _, err := s.PublicarTemaGlobal(context.Background(), ordenTemaPrueba()); !errors.Is(err, ErrResultadoAparienciaIncierto) {
		t.Fatalf("recibo sin auditoría aceptado: %v", err)
	}
	r.alterarRecibo = func(x *ports.ReciboPublicacion) { x.Estado.RevisionGlobal++ }
	if _, err := s.PublicarTemaGlobal(context.Background(), ordenTemaPrueba()); !errors.Is(err, ErrResultadoAparienciaIncierto) {
		t.Fatalf("versión ajena aceptada: %v", err)
	}
	r.alterarRecibo = func(x *ports.ReciboPublicacion) {
		x.Replay = true
		x.DecisionRef = "decision:original:1"
		x.ConfirmadaEn = time.Now().UTC().Add(-time.Hour)
		x.Estado.PublicadaEn = x.ConfirmadaEn
	}
	if recuperado, err := s.PublicarTemaGlobal(context.Background(), ordenTemaPrueba()); err != nil || !recuperado.Replay {
		t.Fatalf("recibo histórico de replay: %+v err=%v", recuperado, err)
	}
	r.alterarRecibo = nil
	r.err = ports.ErrRevisionGlobalObsoleta
	if _, err := s.PublicarTemaGlobal(context.Background(), ordenTemaPrueba()); !errors.Is(err, ports.ErrRevisionGlobalObsoleta) {
		t.Fatalf("conflicto CAS oculto: %v", err)
	}
	r.err = errors.New("detalle privado del registro")
	if _, err := s.PublicarTemaGlobal(context.Background(), ordenTemaPrueba()); !errors.Is(err, ErrResultadoAparienciaIncierto) || err.Error() == r.err.Error() {
		t.Fatalf("error privado expuesto o confirmado: %v", err)
	}
}

func TestConsultarExigePermisoYAUDitoriaAntesDeDevolverEstado(t *testing.T) {
	a, r := &autorizadorAparienciaPrueba{}, &registroAparienciaPrueba{}
	s, _ := NuevoServicioApariencia(a, r)
	consulta, err := s.ConsultarTemaGlobal(context.Background())
	if err != nil || consulta.Estado.Configurado || r.consultas != 1 {
		t.Fatalf("consulta auditable: %+v err=%v llamadas=%d", consulta, err, r.consultas)
	}
	r.alterarConsulta = func(x *ports.ResultadoConsultaApariencia) {
		x.Estado = domain.EstadoApariencia{
			Configurado: true, Tema: domain.Tema{ID: domain.TemaInstitucional, Revision: 1},
			RevisionGlobal: 2, ReciboRef: "recibo:tema:2", HistoriaRef: "historia:tema:2",
			PublicadaEn: time.Now().UTC().Add(-time.Hour),
		}
	}
	if actual, err := s.ConsultarTemaGlobal(context.Background()); err != nil || actual.Estado.RevisionGlobal != 2 {
		t.Fatalf("lectura de estado publicado: %+v err=%v", actual, err)
	}
	r.alterarConsulta = func(x *ports.ResultadoConsultaApariencia) { x.AuditoriaRef = "" }
	if _, err := s.ConsultarTemaGlobal(context.Background()); !errors.Is(err, ErrResultadoAparienciaIncierto) {
		t.Fatalf("lectura sin auditoría aceptada: %v", err)
	}
	a.alterar = func(c *ports.ConcesionApariencia) { c.Permitida = false }
	antes := r.consultas
	if _, err := s.ConsultarTemaGlobal(context.Background()); !errors.Is(err, ErrAparienciaDenegada) || r.consultas != antes {
		t.Fatalf("consulta denegada llegó al registro: err=%v llamadas=%d", err, r.consultas)
	}
}
