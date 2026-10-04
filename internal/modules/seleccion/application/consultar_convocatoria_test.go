package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	bolsadomain "vec-diputacion-granada/internal/modules/bolsa/domain"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Este doble solo verifica el consumidor WIP; no acredita autorización V3.
type lectorConvocatoriaPrueba struct {
	respuesta ports.LecturaConvocatoria
	err       error
	llamadas  int
	observar  func(ports.SolicitudConsultaConvocatoria)
}

func (l *lectorConvocatoriaPrueba) ConsultarExacta(_ context.Context, s ports.SolicitudConsultaConvocatoria) (ports.LecturaConvocatoria, error) {
	l.llamadas++
	if l.observar != nil {
		l.observar(s)
	}
	return l.respuesta, l.err
}

func solicitudConvocatoriaPrueba(t *testing.T) ports.SolicitudConsultaConvocatoria {
	t.Helper()
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudConsultaConvocatoria{Selector: bolsaports.SelectorVersionConvocatoriaExacta{ID: "convocatoria:prueba", Secuencia: 2}, Actor: actor, Correlacion: correlacionNominalPrueba(t)}
}

func lecturaConvocatoriaPrueba(s ports.SolicitudConsultaConvocatoria) ports.LecturaConvocatoria {
	h := strings.Repeat("a", 64)
	return ports.LecturaConvocatoria{
		Ficha: ports.FichaConvocatoria{ConvocatoriaID: s.Selector.ID, Secuencia: s.Selector.Secuencia, Revision: 1, FuenteRef: s.Selector.Referencia(), HuellaVersionSHA256: h,
			Bases:            []bolsadomain.ReferenciaDocumentoOficialConvocatoria{{Rol: "bases", PublicacionRef: "publicacion:bases", DocumentoRef: "documento:bases", VersionDocumento: 1, RepresentacionRef: "representacion:bases", HuellaContenidoSHA256: h, FirmaValidadaRef: "firma:prueba", ReciboCustodiaRef: "custodia:prueba"}},
			Requisitos:       []bolsadomain.RequisitoConvocatoria{{Referencia: "requisito:prueba", Orden: 1, Titulo: "Requisito sintético", Obligatorio: true}},
			FlujoProceso:     bolsadomain.ReferenciaConfiguracionConvocatoria{ID: "flujo-prueba", Version: 1, HuellaContenidoSHA256: h},
			ReglasBaremacion: bolsadomain.ReferenciaConfiguracionConvocatoria{ID: "reglas:prueba", Version: 1, HuellaContenidoSHA256: h}, FasesEstado: "pendiente_fuente", ReferenciasCoberturaEstado: "pendiente_fuente"},
		Evidencia: ports.EvidenciaLecturaConvocatoria{ReciboRef: "recibo:prueba", DecisionRef: "decision:prueba", ConsumoHuellaSHA256: h, AuditoriaRef: "auditoria:prueba", CorrelacionRef: correlacionConvocatoria(s), ConsultadaEn: s.Actor.ResueltoEn},
	}
}

func TestConsultaConvocatoriaConservaVersionPendientesYCopias(t *testing.T) {
	s := solicitudConvocatoriaPrueba(t)
	lector := &lectorConvocatoriaPrueba{respuesta: lecturaConvocatoriaPrueba(s)}
	lector.observar = func(recibida ports.SolicitudConsultaConvocatoria) {
		if recibida.Selector != s.Selector || correlacionConvocatoria(recibida) != correlacionConvocatoria(s) {
			t.Fatal("consulta exacta sustituida")
		}
		recibida.Actor.Instantanea.Vinculos[0].Referencia = "emp_" + strings.Repeat("b", 24)
	}
	r, err := ConsultarConvocatoria(context.Background(), s, lector)
	if err != nil || lector.llamadas != 1 {
		t.Fatalf("consumidor: %v", err)
	}
	if r.Ficha.FasesEstado != "pendiente_fuente" || r.Ficha.ReferenciasCoberturaEstado != "pendiente_fuente" || s.Actor.Instantanea.Vinculos[0].Referencia != "emp_"+strings.Repeat("a", 24) {
		t.Fatal("se completaron datos pendientes o se compartió actor")
	}
	r.Ficha.Bases[0].DocumentoRef = "documento:otro"
	r.Ficha.Requisitos[0].Titulo = "Otro"
	if lector.respuesta.Ficha.Bases[0].DocumentoRef != "documento:bases" || lector.respuesta.Ficha.Requisitos[0].Titulo != "Requisito sintético" {
		t.Fatal("respuesta compartida con el lector")
	}
}

func TestConsultaConvocatoriaRechazaRespuestaCruzadaOSinEvidencia(t *testing.T) {
	s := solicitudConvocatoriaPrueba(t)
	for nombre, cambiar := range map[string]func(*ports.LecturaConvocatoria){
		"otra convocatoria": func(r *ports.LecturaConvocatoria) { r.Ficha.ConvocatoriaID = "convocatoria:otra" },
		"otra version":      func(r *ports.LecturaConvocatoria) { r.Ficha.Secuencia++ },
		"sin huella":        func(r *ports.LecturaConvocatoria) { r.Ficha.HuellaVersionSHA256 = "" },
		"fuente ajena":      func(r *ports.LecturaConvocatoria) { r.Ficha.FuenteRef = "convocatoria:otra#2" },
		"sin evidencia":     func(r *ports.LecturaConvocatoria) { r.Evidencia = ports.EvidenciaLecturaConvocatoria{} },
		"otra correlacion":  func(r *ports.LecturaConvocatoria) { r.Evidencia.CorrelacionRef = "correlacion:otra" },
		"fases deducidas":   func(r *ports.LecturaConvocatoria) { r.Ficha.FasesEstado = "completadas" },
		"OEP deducida":      func(r *ports.LecturaConvocatoria) { r.Ficha.ReferenciasCoberturaEstado = "completadas" },
	} {
		t.Run(nombre, func(t *testing.T) {
			lector := &lectorConvocatoriaPrueba{respuesta: lecturaConvocatoriaPrueba(s)}
			cambiar(&lector.respuesta)
			r, err := ConsultarConvocatoria(context.Background(), s, lector)
			if !errors.Is(err, ports.ErrRespuestaConvocatoriaInvalida) || r.Ficha.ConvocatoriaID != "" {
				t.Fatalf("respuesta parcial aceptada: %v", err)
			}
		})
	}
}

func TestConsultaConvocatoriaFallaSinResultado(t *testing.T) {
	s := solicitudConvocatoriaPrueba(t)
	for _, causa := range []error{ports.ErrConsultaConvocatoriaDenegada, ports.ErrConvocatoriaNoDisponible, ports.ErrConvocatoriaNoEncontrada, context.Canceled} {
		lector := &lectorConvocatoriaPrueba{respuesta: lecturaConvocatoriaPrueba(s), err: causa}
		r, err := ConsultarConvocatoria(context.Background(), s, lector)
		if !errors.Is(err, causa) || r.Ficha.ConvocatoriaID != "" {
			t.Fatalf("fallo tratado como éxito: %v", err)
		}
	}
	lector := &lectorConvocatoriaPrueba{err: errors.New("detalle privado del proveedor")}
	_, err := ConsultarConvocatoria(context.Background(), s, lector)
	if err != ports.ErrConvocatoriaNoDisponible {
		t.Fatal("se propagó un detalle del proveedor")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	lector = &lectorConvocatoriaPrueba{}
	_, err = ConsultarConvocatoria(ctx, s, lector)
	if !errors.Is(err, context.Canceled) || lector.llamadas != 0 {
		t.Fatal("se llamó al lector tras cancelación")
	}
	ctx, cancelar = context.WithCancel(context.Background())
	defer cancelar()
	lector = &lectorConvocatoriaPrueba{respuesta: lecturaConvocatoriaPrueba(s), observar: func(ports.SolicitudConsultaConvocatoria) { cancelar() }}
	r, err := ConsultarConvocatoria(ctx, s, lector)
	if !errors.Is(err, context.Canceled) || r.Ficha.ConvocatoriaID != "" {
		t.Fatal("se aceptó una respuesta tras cancelación")
	}
	s.Actor = vecdomain.ContextoActor{}
	lector = &lectorConvocatoriaPrueba{}
	_, err = ConsultarConvocatoria(context.Background(), s, lector)
	if !errors.Is(err, ports.ErrConsultaConvocatoriaInvalida) || lector.llamadas != 0 {
		t.Fatal("se llamó al lector sin contexto servidor")
	}
}

func TestConsultaConvocatoriaSinLectorNoTieneResultado(t *testing.T) {
	s := solicitudConvocatoriaPrueba(t)
	var nulo *lectorConvocatoriaPrueba
	for _, lector := range []ports.LectorConvocatoriaExacta{nil, nulo} {
		r, err := ConsultarConvocatoria(context.Background(), s, lector)
		if err != ports.ErrConvocatoriaNoDisponible || r.Ficha.ConvocatoriaID != "" {
			t.Fatal("se aceptó una dependencia ausente")
		}
	}
}

type generadorCorrelacionConvocatoriaPrueba struct{}

func (generadorCorrelacionConvocatoriaPrueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return "correlacion_" + strings.Repeat("a", 32), nil
}
func correlacionNominalPrueba(t *testing.T) vecdomain.ReferenciaCorrelacionAutorizacionV2 {
	t.Helper()
	r, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), generadorCorrelacionConvocatoriaPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	return r
}
