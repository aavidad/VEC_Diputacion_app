package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type relojGobiernoRPTProgresivo struct {
	instantes []time.Time
	llamadas  int
}

func (r *relojGobiernoRPTProgresivo) Ahora() time.Time {
	i := r.llamadas
	r.llamadas++
	if i >= len(r.instantes) {
		i = len(r.instantes) - 1
	}
	return r.instantes[i]
}

func entornoEmisionGobiernoRPT(t *testing.T) (*entornoAutorizacionSolicitudV3Prueba, *confianza.EmisorMaterialAutorizacionAtestadaV3, CredencialesGobiernoCategoriaRPT, ports.PreparacionGobiernoCategoriaRPT) {
	t.Helper()
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	e.servicio.generador = generadorContactoAleatorio{}
	e.fuente.instantanea.VersionRol.Concesiones[0] = domain.ConcesionRol{
		Accion:   ports.AccionAprobarGobiernoCategoriaRPT,
		ModuloID: "bolsa", TipoRecurso: ports.TipoRecursoGobiernoCategoriaRPT,
		Finalidades:      []string{ports.FinalidadGobiernoCategoriaRPT},
		GarantiaMinima:   domain.AuthAssuranceHigh,
		CamposPermitidos: []string{"gobierno", "recibo"},
	}
	e.fuente.instantanea.AsignacionPerfil.Ambitos = []domain.AmbitoPerfil{
		{Clave: "catalogo_id", Valores: []string{"catalogo.ejemplo"}},
		{Clave: "modulo_id", Valores: []string{"bolsa"}},
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { clear(priv) })
	relojEmisor := &relojAutorizacionServicioPrueba{ahora: e.ahora}
	at, err := NuevoServicioAtestacionesAutorizacionV3(domain.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite:          confianza.SuiteAtestacionAutorizacionV3COSEEdDSA,
		ClaveID:        "clave:prueba:contacto", Audiencia: "vec/prueba/contacto",
	}, firmanteContacto{priv, e.ahora})
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		"clave:prueba:contacto", 1, pub, "vec/prueba/contacto",
		confianza.EstadoClaveAtestacionAutorizacionV3Activa,
		e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{},
	)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		"confianza:prueba:contacto", 1, e.ahora.Add(-time.Minute), e.ahora.Add(time.Hour), raiz,
	)
	if err != nil {
		t.Fatal(err)
	}
	verificador, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(cfg, relojEmisor)
	if err != nil {
		t.Fatal(err)
	}
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(
		"clave:capacidad:contacto", 1, bytes.Repeat([]byte{0x71}, 32),
		"emisor:prueba:contacto", ports.AudienciaGobiernoCategoriaRPT,
		confianza.EstadoClaveHMACCapacidadAtestacionV3Emision,
		e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("7", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	emisorCapacidades, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, relojEmisor)
	if err != nil {
		t.Fatal(err)
	}
	emisor, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(e.servicio, at, verificador, emisorCapacidades)
	if err != nil {
		t.Fatal(err)
	}
	base, err := e.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("a", 64)
	p := ports.PreparacionGobiernoCategoriaRPT{
		Accion:          ports.AccionAprobarGobiernoCategoriaRPT,
		Finalidad:       ports.FinalidadGobiernoCategoriaRPT,
		Audiencia:       ports.AudienciaGobiernoCategoriaRPT,
		HuellaPropuesta: h,
		Recurso: domain.RecursoAutorizable{
			Referencia: "propuesta:ejemplo", ModuloID: "bolsa",
			Tipo:      ports.TipoRecursoGobiernoCategoriaRPT,
			Ambitos:   map[string]string{"catalogo_id": "catalogo.ejemplo", "modulo_id": "bolsa"},
			Atributos: map[string]string{"material_sha256": h},
		},
	}
	cred := CredencialesGobiernoCategoriaRPT{
		Actor: e.resultado.Contexto, Vinculo: base.VinculoAutenticacionActor,
		ResultadoContexto: e.resultado, Motivo: base.ReferenciaMotivo,
		Correlacion: base.Correlacion,
	}
	return e, emisor, cred, p
}

func TestGobiernoRPTContrastaCapacidadTrasEmision(t *testing.T) {
	for _, caso := range []struct {
		name    string
		segundo time.Duration
		concede bool
	}{
		{"vigente", time.Microsecond, true},
		{"vencida", 10 * time.Second, false},
	} {
		t.Run(caso.name, func(t *testing.T) {
			e, emisor, cred, p := entornoEmisionGobiernoRPT(t)
			reloj := &relojGobiernoRPTProgresivo{instantes: []time.Time{
				e.ahora.Add(-time.Microsecond), e.ahora.Add(caso.segundo),
			}}
			s := &ServicioGobiernoCategoriaRPT{autorizador: emisor, reloj: reloj}
			solicitud, material, err := s.autorizar(t.Context(), cred, p,
				ports.AccionAprobarGobiernoCategoriaRPT, "propuesta:ejemplo", "catalogo.ejemplo", "bolsa", strings.Repeat("a", 64))
			if reloj.llamadas != 2 {
				t.Fatalf("reloj invocado %d veces", reloj.llamadas)
			}
			if caso.concede {
				_, errSolicitud := solicitud.Datos()
				if err != nil || errSolicitud != nil || material.ValidarEstructura() != nil {
					t.Fatalf("capacidad vigente rechazada: %v", err)
				}
			} else if !errors.Is(err, ports.ErrGobiernoCategoriaRPTDenegado) {
				t.Fatalf("capacidad vencida aceptada: %v", err)
			}
		})
	}
}

type preparadorGobiernoRPTPrueba struct {
	llamadas  int
	err       error
	resultado ports.PreparacionPropuestaGobiernoCategoriaRPT
}

func (p *preparadorGobiernoRPTPrueba) PrepararPropuestaGobiernoCategoriaRPT(context.Context, ports.BorradorPropuestaGobiernoCategoriaRPT) (ports.PreparacionPropuestaGobiernoCategoriaRPT, error) {
	p.llamadas++
	return p.resultado, p.err
}
func (p *preparadorGobiernoRPTPrueba) PrepararAprobacionGobiernoCategoriaRPT(context.Context, ports.MaterialAvanceGobiernoCategoriaRPT) (ports.PreparacionGobiernoCategoriaRPT, error) {
	p.llamadas++
	return ports.PreparacionGobiernoCategoriaRPT{}, p.err
}
func (p *preparadorGobiernoRPTPrueba) PrepararConfirmacionGobiernoCategoriaRPT(context.Context, ports.MaterialAvanceGobiernoCategoriaRPT) (ports.PreparacionGobiernoCategoriaRPT, error) {
	p.llamadas++
	return ports.PreparacionGobiernoCategoriaRPT{}, p.err
}

func materialAvanceGobiernoRPTPrueba() ports.MaterialAvanceGobiernoCategoriaRPT {
	return ports.MaterialAvanceGobiernoCategoriaRPT{
		PropuestaRef: "propuesta:ejemplo", HuellaSHA256: strings.Repeat("a", 64),
		ReciboRef: "recibo:ejemplo", RevisionEsperada: 1,
		CatalogoID: "catalogo.ejemplo", ModuloID: "bolsa",
	}
}

func TestServicioGobiernoRPTFallaAntesDeV3ConCASInvalidoYDependenciaCaida(t *testing.T) {
	p := &preparadorGobiernoRPTPrueba{err: errors.New("dsn privado y detalle ajeno")}
	s := &ServicioGobiernoCategoriaRPT{preparador: p}
	o := OrdenAvanzarGobiernoCategoriaRPT{Material: materialAvanceGobiernoRPTPrueba()}
	o.Material.RevisionEsperada = 3
	if _, err := s.Aprobar(t.Context(), o); !errors.Is(err, ErrOrdenGobiernoCategoriaRPTInvalida) || p.llamadas != 0 {
		t.Fatalf("CAS invalido: error=%v preparaciones=%d", err, p.llamadas)
	}
	o.Material.RevisionEsperada = 1
	if _, err := s.Aprobar(t.Context(), o); !errors.Is(err, ports.ErrGobiernoCategoriaRPTNoDisponible) || strings.Contains(err.Error(), "dsn privado") || p.llamadas != 1 {
		t.Fatalf("dependencia caida: error=%v preparaciones=%d", err, p.llamadas)
	}
	p.err = ports.ErrGobiernoCategoriaRPTDenegado
	if _, err := s.Aprobar(t.Context(), o); !errors.Is(err, ports.ErrGobiernoCategoriaRPTDenegado) || p.llamadas != 2 {
		t.Fatalf("denegacion: error=%v preparaciones=%d", err, p.llamadas)
	}
}

func TestServicioGobiernoRPTRechazaMaterialPreparadoConContenidoAjeno(t *testing.T) {
	id, revision, h := "categoria.ejemplo", int64(7), strings.Repeat("a", 64)
	motivo := domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos", CatalogoVersion: 1,
		CatalogoHuellaSHA256: h, EntradaClave: "motivo_0123456789abcdef0123456789abcdef",
	}
	b := ports.BorradorPropuestaGobiernoCategoriaRPT{
		PropuestaRef: "propuesta:ejemplo", ReciboRef: "recibo:ejemplo",
		Contenido: domain.ContenidoGobiernoCategoriaRPT{
			Accion:     domain.AccionGobiernoCategoriaRPTDeshabilitar,
			CatalogoID: "catalogo.ejemplo", ModuloID: "bolsa", Version: 2,
			CategoriaID: &id, RevisionEsperada: &revision,
			PreimagenesControl: map[string]domain.PreimagenControlGobiernoCategoriaRPT{
				id: {Version: 2, Revision: revision, HuellaSHA256: h, Estado: "habilitada"},
			},
			MotivoRef: motivo.Referencia(), FuenteRef: "resolucion-2026-99",
		},
	}
	preparado := ports.MaterialPropuestaGobiernoCategoriaRPT{
		PropuestaRef: b.PropuestaRef, ReciboRef: b.ReciboRef,
		Contenido: b.Contenido, HuellaSHA256: h,
	}
	preparado.Contenido.PreimagenesHuellaSHA256 = h
	preparado.Contenido.FuenteRef = "resolucion-ajena"
	p := &preparadorGobiernoRPTPrueba{resultado: ports.PreparacionPropuestaGobiernoCategoriaRPT{Material: preparado}}
	s := &ServicioGobiernoCategoriaRPT{preparador: p}
	_, err := s.Proponer(t.Context(), OrdenProponerGobiernoCategoriaRPT{
		Credenciales: CredencialesGobiernoCategoriaRPT{Motivo: motivo}, Borrador: b,
	})
	if !errors.Is(err, ports.ErrGobiernoCategoriaRPTNoConfiable) || p.llamadas != 1 {
		t.Fatalf("contenido preparado ajeno: error=%v preparaciones=%d", err, p.llamadas)
	}
}

func TestResultadoGobiernoRPTConservaReciboEnReplayConDecisionNueva(t *testing.T) {
	fecha := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	h := strings.Repeat("a", 64)
	for _, decision := range []string{"decision-primera", "decision-replay"} {
		resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(
			decision, h, h, "contexto-1", h,
			ports.AccionAprobarGobiernoCategoriaRPT, "propuesta:ejemplo", h,
			ports.AudienciaGobiernoCategoriaRPT, fecha, fecha.Add(time.Second),
		)
		if err != nil {
			t.Fatal(err)
		}
		r := ports.ResultadoGobiernoCategoriaRPT{
			PropuestaRef: "propuesta:ejemplo", HuellaSHA256: h,
			Revision: 2, Estado: domain.EstadoGobiernoCategoriaRPTUnaAprobacion,
			ReciboRef: "recibo:ejemplo",
			Evidencia: ports.EvidenciaGobiernoCategoriaRPT{
				DecisionRef: decision, EfectoRef: "propuesta:ejemplo", HuellaEfectoSHA256: h,
				ConsumoHuellaSHA256: h, AuditoriaRef: "auditoria-1", ConsumidaEn: fecha, ConsumoNuevo: true,
			},
		}
		if !resultadoGobiernoCategoriaRPTValido(r, r.PropuestaRef, h, r.ReciboRef, 2, r.Estado, resumen) {
			t.Fatalf("replay con decision nueva rechazado: %s", decision)
		}
		r.Evidencia.HuellaEfectoSHA256 = strings.Repeat("b", 64)
		if resultadoGobiernoCategoriaRPTValido(r, r.PropuestaRef, h, r.ReciboRef, 2, r.Estado, resumen) {
			t.Fatal("huella de recurso ajena aceptada")
		}
	}
}
