package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	core "vec-diputacion-granada/internal/vec/ports"
)

// Doble privado: demuestra coordinacion y validacion Go, no durabilidad SQL.
type repositorioPreparacionPrueba struct {
	llamadas  int
	esperada  prep.Esperada
	clave     string
	r         ports.ResultadoPreparacionBases
	manipular func(*ports.ResultadoPreparacionBases)
}

func (p *repositorioPreparacionPrueba) GuardarPreparacionBases(_ context.Context, o ports.GuardarPreparacionBases) (ports.ResultadoPreparacionBases, error) {
	p.llamadas++
	h, _ := prep.HuellaIntencion(o.Esperada, o.Material, o.Ambito)
	if p.clave == o.ClaveOperacion && p.clave != "" {
		if p.r.Recibo.HuellaIntencionSHA256 != h {
			return ports.ResultadoPreparacionBases{}, ports.ErrPreparacionBasesClaveReutilizada
		}
	} else {
		if p.clave != "" && p.esperada != o.Esperada {
			return ports.ResultadoPreparacionBases{}, ports.ErrPreparacionBasesConflicto
		}
		huella, _ := o.Material.HuellaSHA256()
		p.r = ports.ResultadoPreparacionBases{Version: prep.Version{Ambito: o.Ambito, Estado: prep.Esperada{PreparacionRef: o.Esperada.PreparacionRef, Revision: o.Esperada.Revision + 1, HuellaMaterialSHA256: huella}, Material: o.Material},
			Recibo: ports.ReciboPreparacionBases{ReciboRef: "recibo:prep:1", HistoriaRef: "historia:prep:1", AuditoriaRef: "auditoria:efecto:1", EventoRef: "evento:prep:1", HuellaIntencionSHA256: h, ConfirmadaEn: o.SolicitadaEn}}
		p.clave = o.ClaveOperacion
		p.esperada = p.r.Version.Estado
	}
	return p.acceso(p.r, o.Autorizacion, o.SolicitadaEn), nil
}

func (p *repositorioPreparacionPrueba) ConsultarPreparacionBases(_ context.Context, o ports.ConsultarPreparacionBases) (ports.ResultadoPreparacionBases, error) {
	p.llamadas++
	return p.acceso(p.r, o.Autorizacion, o.SolicitadaEn), nil
}

func (p *repositorioPreparacionPrueba) acceso(r ports.ResultadoPreparacionBases, e core.EvidenciaUsoDecisionAutorizacion, instante time.Time) ports.ResultadoPreparacionBases {
	d, _ := e.Datos()
	r.AutorizacionRef = d.Decision.DecisionRef
	r.HuellaAutorizacionSHA256 = d.HuellaDecisionSHA256
	r.ConsumoAutorizacionRef = "consumo:prep:1"
	r.AuditoriaAccesoRef = "auditoria:acceso:1"
	r.AccedidaEn = instante
	if p.manipular != nil {
		p.manipular(&r)
	}
	return r
}

func evidenciaPreparacionPrueba(t *testing.T, r vec.RecursoAutorizable, accion string) core.EvidenciaUsoDecisionAutorizacion {
	t.Helper()
	a, v := nuevoContextoYVinculoPanelPrueba(t, vec.AuthMethodCertificate, vec.AuthAssuranceHigh, vec.SuperficieAutenticacionInternaCorporativaV1)
	exigidor := exigidorConsultaConvocatoriaPrueba{instante: instantePanelInternoPrueba}
	politica, _ := nuevaPoliticaConsultaConvocatoria(false)
	e, err := exigidor.ExigirEvidencia(context.Background(), a, v, r, "correlacion:prep:sintetica", "", politica)
	if err != nil {
		t.Fatal(err)
	}
	d, _ := e.Datos()
	d.Decision.Accion = accion
	d.Decision.Finalidad = ports.FinalidadPreparacionBases
	d.Decision.CamposPermitidos = []string{"material_preparacion"}
	if accion == ports.AccionGuardarPreparacionBases {
		d.Decision.CamposPermitidos = []string{"auditoria", "evento_outbox", "historia", "material_preparacion"}
	}
	e, err = core.NuevaEvidenciaUsoDecisionAutorizacion(d.Decision, instantePanelInternoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func ordenGuardarPreparacionPrueba(t *testing.T) ports.GuardarPreparacionBases {
	ambito, _ := bolsa.NuevoAmbitoOrganizativoConvocatoria("org_diputaciongranada", "uni_seleccionexterna")
	o := ports.GuardarPreparacionBases{Ambito: ambito, Esperada: prep.Esperada{PreparacionRef: "prep:sintetica"}, ClaveOperacion: "operacion:sintetica:1", Material: prep.Material{Contenido: bolsa.ContenidoPublicableConvocatoria{Titulo: "Propuesta incompleta sintética"}}}
	r, err := o.RecursoAutorizable()
	if err != nil {
		t.Fatal(err)
	}
	o.Autorizacion = evidenciaPreparacionPrueba(t, r, ports.AccionGuardarPreparacionBases)
	return o
}

func TestPreparacionBasesGuardarReplayYConsultaExacta(t *testing.T) {
	p := &repositorioPreparacionPrueba{}
	s, err := NuevoServicioPreparacionBases(p, relojPanelInternoPrueba{ahora: instantePanelInternoPrueba})
	if err != nil {
		t.Fatal(err)
	}
	o := ordenGuardarPreparacionPrueba(t)
	r, err := s.Guardar(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.Guardar(context.Background(), o)
	if err != nil || !reflect.DeepEqual(r.Recibo, replay.Recibo) || r.Version.Estado != replay.Version.Estado {
		t.Fatalf("replay cambia recibo/version: %v", err)
	}
	q := ports.ConsultarPreparacionBases{Ambito: r.Version.Ambito, Exacta: r.Version.Estado}
	recurso, _ := q.RecursoAutorizable()
	q.Autorizacion = evidenciaPreparacionPrueba(t, recurso, ports.AccionConsultarPreparacionBases)
	lectura, err := s.Consultar(context.Background(), q)
	if err != nil || !reflect.DeepEqual(lectura.Version, r.Version) {
		t.Fatalf("recuperacion cambia material: %v", err)
	}
	o.Material.Contenido.Titulo = "Corrección sintética"
	recurso, _ = o.RecursoAutorizable()
	o.Autorizacion = evidenciaPreparacionPrueba(t, recurso, ports.AccionGuardarPreparacionBases)
	if _, err = s.Guardar(context.Background(), o); !errors.Is(err, ports.ErrPreparacionBasesClaveReutilizada) {
		t.Fatalf("clave reusada: %v", err)
	}
	o.ClaveOperacion = "operacion:sintetica:2"
	if _, err = s.Guardar(context.Background(), o); !errors.Is(err, ports.ErrPreparacionBasesConflicto) {
		t.Fatalf("preimagen vieja: %v", err)
	}
	o.Esperada = r.Version.Estado
	recurso, _ = o.RecursoAutorizable()
	o.Autorizacion = evidenciaPreparacionPrueba(t, recurso, ports.AccionGuardarPreparacionBases)
	corregida, err := s.Guardar(context.Background(), o)
	if err != nil || corregida.Version.Estado.Revision != 2 || corregida.Version.Estado.HuellaMaterialSHA256 == r.Version.Estado.HuellaMaterialSHA256 {
		t.Fatalf("correccion: %v", err)
	}
}

func TestPreparacionBasesDeniegaAntesDeRepositorioYValidaSalida(t *testing.T) {
	p := &repositorioPreparacionPrueba{}
	s, _ := NuevoServicioPreparacionBases(p, relojPanelInternoPrueba{ahora: instantePanelInternoPrueba})
	o := ordenGuardarPreparacionPrueba(t)
	o.Autorizacion = core.EvidenciaUsoDecisionAutorizacion{}
	if _, err := s.Guardar(context.Background(), o); !errors.Is(err, ports.ErrPreparacionBasesInvalida) || p.llamadas != 0 {
		t.Fatalf("capacidad vacia alcanza repositorio: %v", err)
	}
	o = ordenGuardarPreparacionPrueba(t)
	o.Material.Contenido.Titulo = "Material distinto del autorizado"
	if _, err := s.Guardar(context.Background(), o); !errors.Is(err, ports.ErrPreparacionBasesInvalida) || p.llamadas != 0 {
		t.Fatalf("intencion cambiada autorizada: %v", err)
	}
	p.manipular = func(r *ports.ResultadoPreparacionBases) { r.Version.Material.Contenido.Titulo = "Material manipulado" }
	if _, err := s.Guardar(context.Background(), ordenGuardarPreparacionPrueba(t)); !errors.Is(err, ports.ErrResultadoPreparacionBasesInvalido) {
		t.Fatalf("respuesta cambiada aceptada: %v", err)
	}
}

func TestPreparacionBasesFallaCerradaSinDependencias(t *testing.T) {
	var p *repositorioPreparacionPrueba
	if _, err := NuevoServicioPreparacionBases(p, relojPanelInternoPrueba{}); !errors.Is(err, ports.ErrPreparacionBasesNoDisponible) {
		t.Fatal(err)
	}
}

func TestPreparacionBasesConservaCausaDelResultadoInvalido(t *testing.T) {
	p := &repositorioPreparacionPrueba{}
	s, _ := NuevoServicioPreparacionBases(p, relojPanelInternoPrueba{ahora: instantePanelInternoPrueba})
	o := ordenGuardarPreparacionPrueba(t)
	r, err := s.Guardar(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	evidenciaVacia := core.EvidenciaUsoDecisionAutorizacion{}
	err = validarResultadoPreparacion(r, r.Version.Estado, evidenciaVacia, r.AccedidaEn)
	if !errors.Is(err, core.ErrEvidenciaUsoDecisionAutorizacionInvalida) || !errors.Is(err, ports.ErrResultadoPreparacionBasesInvalido) {
		t.Fatalf("causa de evidencia perdida: %v", err)
	}
	p.manipular = func(r *ports.ResultadoPreparacionBases) { r.Version.Material.Contenido.Titulo = "Material manipulado" }
	resultado, err := s.Guardar(context.Background(), o)
	if !errors.Is(err, prep.ErrVersionInvalida) || !errors.Is(err, ports.ErrResultadoPreparacionBasesInvalido) || !reflect.DeepEqual(resultado, ports.ResultadoPreparacionBases{}) {
		t.Fatalf("guardado devuelve material o pierde causa de version: %v", err)
	}
	q := ports.ConsultarPreparacionBases{Ambito: r.Version.Ambito, Exacta: r.Version.Estado}
	recurso, _ := q.RecursoAutorizable()
	q.Autorizacion = evidenciaPreparacionPrueba(t, recurso, ports.AccionConsultarPreparacionBases)
	resultado, err = s.Consultar(context.Background(), q)
	if !errors.Is(err, prep.ErrVersionInvalida) || !errors.Is(err, ports.ErrResultadoPreparacionBasesInvalido) || !reflect.DeepEqual(resultado, ports.ResultadoPreparacionBases{}) {
		t.Fatalf("consulta devuelve material o pierde causa de version: %v", err)
	}
}

func TestPreparacionBasesNoRecuperaVersionAjenaNiSalidaSinAuditoria(t *testing.T) {
	for _, manipular := range []func(*ports.ResultadoPreparacionBases){
		func(r *ports.ResultadoPreparacionBases) { r.Version.Estado.Revision++ },
		func(r *ports.ResultadoPreparacionBases) { r.HuellaAutorizacionSHA256 = "" },
		func(r *ports.ResultadoPreparacionBases) { r.AuditoriaAccesoRef = r.Recibo.AuditoriaRef },
		func(r *ports.ResultadoPreparacionBases) { r.AccedidaEn = r.AccedidaEn.Add(time.Second) },
	} {
		p := &repositorioPreparacionPrueba{}
		s, _ := NuevoServicioPreparacionBases(p, relojPanelInternoPrueba{ahora: instantePanelInternoPrueba})
		r, err := s.Guardar(context.Background(), ordenGuardarPreparacionPrueba(t))
		if err != nil {
			t.Fatal(err)
		}
		q := ports.ConsultarPreparacionBases{Ambito: r.Version.Ambito, Exacta: r.Version.Estado}
		recurso, _ := q.RecursoAutorizable()
		q.Autorizacion = evidenciaPreparacionPrueba(t, recurso, ports.AccionConsultarPreparacionBases)
		p.manipular = manipular
		if _, err := s.Consultar(context.Background(), q); !errors.Is(err, ports.ErrResultadoPreparacionBasesInvalido) {
			t.Fatalf("lectura manipulada aceptada: %v", err)
		}
	}
}

func TestPreparacionBasesReautorizaConRelojActualTambienEnReplay(t *testing.T) {
	p := &repositorioPreparacionPrueba{}
	s, _ := NuevoServicioPreparacionBases(p, relojPanelInternoPrueba{ahora: instantePanelInternoPrueba})
	o := ordenGuardarPreparacionPrueba(t)
	if _, err := s.Guardar(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	s.reloj = relojPanelInternoPrueba{ahora: instantePanelInternoPrueba.Add(time.Minute)}
	// Un instante enviado por el consumidor no prolonga la ventana vigente.
	o.SolicitadaEn = instantePanelInternoPrueba
	if _, err := s.Guardar(context.Background(), o); !errors.Is(err, ports.ErrPreparacionBasesInvalida) || p.llamadas != 1 {
		t.Fatalf("replay usa evidencia caducada: %v", err)
	}
}
