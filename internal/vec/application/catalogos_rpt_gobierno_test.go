package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
	v, err := base.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	autenticacion := v.Autenticacion()
	autenticacion.CuentaPrivilegiada = false
	autenticacion.Superficie = domain.SuperficieAutenticacionInternaCorporativaV1
	autenticacion.CuentaOrdinariaRef = autenticacion.CuentaRef
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(t.Context(),
		&revalidadorVinculoAplicacionAdversarial{resultado: autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef},
		resolutorContextoAutorizacionV3Prueba{resultado: e.resultado},
		domain.SolicitudContextoActor{Cuenta: domain.CuentaAutenticadaContextoActor{CuentaRef: v.CuentaRef, Metodo: v.MetodoObservado, Garantia: v.GarantiaObservada}, PerfilActivoRef: v.PerfilActivoRef},
		&relojAutorizacionServicioPrueba{ahora: e.ahora})
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
		Actor: e.resultado.Contexto, Vinculo: vinculo,
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
			s := &ServicioGobiernoCategoriaRPT{autorizador: emisor, reloj: reloj, versionRolRef: e.fuente.instantanea.VersionRol.Referencia()}
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

func TestGobiernoRPTDenegacionSanitizaErrorDeDependencia(t *testing.T) {
	causa := errors.New("dni sintético y detalle interno de la decisión")
	err := denegacionValidacionGobiernoCategoriaRPT(causa)
	if !errors.Is(err, ports.ErrGobiernoCategoriaRPTDenegado) ||
		errors.Is(err, causa) || strings.Contains(err.Error(), "dni sintético") ||
		strings.Contains(err.Error(), "detalle interno") {
		t.Fatalf("error de validación expuso causa privada: %v", err)
	}
	if err := concesionGobiernoCategoriaRPTValida(
		ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
		domain.SolicitudAutorizacionLigadaV3{}, domain.DecisionAutorizacionLigadaV3{},
		ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
		domain.ResultadoContextoActorRegistradoV2{}, domain.ContextoActor{},
		ports.AccionAprobarGobiernoCategoriaRPT, domain.RecursoAutorizable{}, time.Time{}, "rol:prueba:v1",
	); !errors.Is(err, ports.ErrGobiernoCategoriaRPTDenegado) {
		t.Fatalf("material inválido no denegado: %v", err)
	}
}

type preparadorGobiernoRPTPrueba struct {
	llamadas  int
	err       error
	resultado ports.PreparacionPropuestaGobiernoCategoriaRPT
	avance    ports.PreparacionGobiernoCategoriaRPT
}

func (p *preparadorGobiernoRPTPrueba) PrepararPropuestaGobiernoCategoriaRPT(context.Context, ports.BorradorPropuestaGobiernoCategoriaRPT) (ports.PreparacionPropuestaGobiernoCategoriaRPT, error) {
	p.llamadas++
	return p.resultado, p.err
}
func (p *preparadorGobiernoRPTPrueba) PrepararAprobacionGobiernoCategoriaRPT(context.Context, ports.MaterialAvanceGobiernoCategoriaRPT) (ports.PreparacionGobiernoCategoriaRPT, error) {
	p.llamadas++
	return p.avance, p.err
}
func (p *preparadorGobiernoRPTPrueba) PrepararConfirmacionGobiernoCategoriaRPT(context.Context, ports.MaterialAvanceGobiernoCategoriaRPT) (ports.PreparacionGobiernoCategoriaRPT, error) {
	p.llamadas++
	return p.avance, p.err
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

func TestServicioGobiernoRPTPrevalidaCancelacionYCardinalidadAntesDeCopiar(t *testing.T) {
	_, _, cred, _ := entornoEmisionGobiernoRPT(t)
	b := borradorPublicarGobiernoRPTAplicacionPrueba(t, cred)
	for i := range 20_000 {
		b.Contenido.PreimagenesControl["categoria."+strconv.Itoa(i)] = domain.PreimagenControlGobiernoCategoriaRPT{}
	}
	preparador := &preparadorGobiernoRPTPrueba{}
	s := &ServicioGobiernoCategoriaRPT{preparador: preparador}
	for _, cancelado := range []bool{true, false} {
		ctx, cancelar := context.WithCancel(t.Context())
		if cancelado {
			cancelar()
		}
		o := OrdenProponerGobiernoCategoriaRPT{Credenciales: cred, Borrador: b}
		if _, err := s.Proponer(ctx, o); !errors.Is(err, ErrOrdenGobiernoCategoriaRPTInvalida) || preparador.llamadas != 0 {
			t.Fatalf("borrador excesivo o cancelado alcanzó preparación: cancelado=%v err=%v llamadas=%d", cancelado, err, preparador.llamadas)
		}
		if asignaciones := testing.AllocsPerRun(3, func() { _, _ = s.Proponer(ctx, o) }); asignaciones != 0 {
			t.Fatalf("borrador excesivo o cancelado se copió antes del rechazo: cancelado=%v asignaciones=%v", cancelado, asignaciones)
		}
		cancelar()
	}
}

func TestServicioGobiernoRPTRechazaMaterialPreparadoConContenidoAjeno(t *testing.T) {
	e, _, cred, _ := entornoEmisionGobiernoRPT(t)
	_ = e
	h := strings.Repeat("a", 64)
	b := borradorPublicarGobiernoRPTAplicacionPrueba(t, cred)
	contenido, _ := b.Contenido.PrepararBorradorParaEditor(cred.Actor.Principal.ID)
	preparado := ports.MaterialPropuestaGobiernoCategoriaRPT{
		PropuestaRef: b.PropuestaRef, ReciboRef: b.ReciboRef,
		Contenido: contenido, HuellaSHA256: h,
	}
	preparado.Contenido.PreimagenesHuellaSHA256 = h
	preparado.Contenido.FuenteRef = "resolucion-ajena"
	p := &preparadorGobiernoRPTPrueba{resultado: ports.PreparacionPropuestaGobiernoCategoriaRPT{Material: preparado}}
	s := &ServicioGobiernoCategoriaRPT{preparador: p}
	_, err := s.Proponer(t.Context(), OrdenProponerGobiernoCategoriaRPT{
		Credenciales: cred, Borrador: b,
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
			Revision: 2, Estado: domain.EstadoGobiernoCategoriaRPTAprobada,
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

type autorizadorGobiernoRPTCambiaRol struct {
	ports.AutorizadorGobiernoCategoriaRPT
	fuente *fuenteAutorizacionServicioPrueba
}

func (a autorizadorGobiernoRPTCambiaRol) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s domain.SolicitudAutorizacionLigadaV3, r domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, ports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	// Simula la reasignacion despues de la frontera HTTP, antes de V3.
	a.fuente.instantanea.VersionRol.RolID = "rol_ajeno"
	a.fuente.instantanea.AsignacionPerfil.VersionRolRef = a.fuente.instantanea.VersionRol.Referencia()
	a.fuente.instantanea.ControlVigenciaVersionRol.VersionRolRef = a.fuente.instantanea.VersionRol.Referencia()
	return a.AutorizadorGobiernoCategoriaRPT.EmitirMaterialAutorizacionAtestadaV3(ctx, s, r)
}

func TestGobiernoRPTDeniegaReasignacionDeRolEntreFronteraYV3(t *testing.T) {
	e, emisor, cred, p := entornoEmisionGobiernoRPT(t)
	preparador := &preparadorGobiernoRPTPrueba{avance: p}
	gestor := &gestorGobiernoRPTPrueba{}
	s, err := NuevoServicioGobiernoCategoriaRPT(preparador,
		autorizadorGobiernoRPTCambiaRol{emisor, e.fuente}, gestor,
		&relojAutorizacionServicioPrueba{ahora: e.ahora}, e.fuente.instantanea.VersionRol.Referencia())
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Aprobar(t.Context(), OrdenAvanzarGobiernoCategoriaRPT{Credenciales: cred, Material: materialAvanceGobiernoRPTPrueba()})
	if !errors.Is(err, ports.ErrGobiernoCategoriaRPTDenegado) || gestor.llamadas != 0 || e.concesiones.invocaciones != 1 {
		t.Fatalf("decision V3 de otro rol: err=%v efectos=%d concesiones=%d", err, gestor.llamadas, e.concesiones.invocaciones)
	}
}

type gestorGobiernoRPTPrueba struct {
	llamadas          int
	principal, perfil string
}

type gestorGobiernoRPTFalloPrueba struct {
	*gestorGobiernoRPTPrueba
	err      error
	cancelar context.CancelFunc
	llamadas int
}

func (g *gestorGobiernoRPTFalloPrueba) fallar() (ports.ResultadoGobiernoCategoriaRPT, error) {
	g.llamadas++
	if g.cancelar != nil {
		g.cancelar()
	}
	return ports.ResultadoGobiernoCategoriaRPT{}, g.err
}

func (g *gestorGobiernoRPTFalloPrueba) AprobarGobiernoCategoriaRPT(context.Context, ports.OrdenAvanceGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	return g.fallar()
}

func (g *gestorGobiernoRPTFalloPrueba) ConfirmarGobiernoCategoriaRPT(context.Context, ports.OrdenAvanceGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	return g.fallar()
}

func TestServicioGobiernoRPTSaneaErroresDelGestorAlAprobarYConfirmar(t *testing.T) {
	for _, accion := range []string{ports.AccionAprobarGobiernoCategoriaRPT, ports.AccionConfirmarGobiernoCategoriaRPT} {
		for _, caso := range []struct {
			nombre   string
			causa    error
			esperado error
			cancelar bool
		}{
			{"tecnico", errors.New("dni privado y traza interna"), ports.ErrGobiernoCategoriaRPTNoDisponible, false},
			{"nominal envuelto", fmt.Errorf("dato privado: %w", ports.ErrGobiernoCategoriaRPTConflicto), ports.ErrGobiernoCategoriaRPTConflicto, false},
			{"cancelacion devuelta", context.Canceled, ports.ErrGobiernoCategoriaRPTNoDisponible, false},
			{"contexto cancelado", errors.New("dato privado tras cancelar"), context.Canceled, true},
		} {
			t.Run(accion+"/"+caso.nombre, func(t *testing.T) {
				e, emisor, cred, preparacion := entornoEmisionGobiernoRPT(t)
				e.fuente.instantanea.VersionRol.Concesiones[0].Accion = accion
				preparacion.Accion = accion
				ctx, cancelar := context.WithCancel(t.Context())
				defer cancelar()
				gestor := &gestorGobiernoRPTFalloPrueba{gestorGobiernoRPTPrueba: &gestorGobiernoRPTPrueba{}, err: caso.causa}
				if caso.cancelar {
					gestor.cancelar = cancelar
				}
				s, err := NuevoServicioGobiernoCategoriaRPT(&preparadorGobiernoRPTPrueba{avance: preparacion},
					emisor, gestor, &relojAutorizacionServicioPrueba{ahora: e.ahora}, e.fuente.instantanea.VersionRol.Referencia())
				if err != nil {
					t.Fatal(err)
				}
				m := materialAvanceGobiernoRPTPrueba()
				if accion == ports.AccionConfirmarGobiernoCategoriaRPT {
					m.RevisionEsperada = 2
					_, err = s.Confirmar(ctx, OrdenAvanzarGobiernoCategoriaRPT{Credenciales: cred, Material: m})
				} else {
					_, err = s.Aprobar(ctx, OrdenAvanzarGobiernoCategoriaRPT{Credenciales: cred, Material: m})
				}
				if !errors.Is(err, caso.esperado) || err == caso.causa ||
					strings.Contains(err.Error(), "privado") || gestor.llamadas != 1 {
					t.Fatalf("error del gestor expuesto: accion=%s err=%v llamadas=%d", accion, err, gestor.llamadas)
				}
			})
		}
	}
}

func (g *gestorGobiernoRPTPrueba) resultado(s domain.SolicitudAutorizacionLigadaV3, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, propuesta, huella, recibo string, revision int64, estado string) (ports.ResultadoGobiernoCategoriaRPT, error) {
	g.llamadas++
	datos, err := s.Datos()
	if err != nil {
		return ports.ResultadoGobiernoCategoriaRPT{}, err
	}
	v, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		return ports.ResultadoGobiernoCategoriaRPT{}, err
	}
	g.principal, g.perfil = v.PrincipalID, v.PerfilActivoRef
	r := a.ResumenCapacidad()
	return ports.ResultadoGobiernoCategoriaRPT{PropuestaRef: propuesta, HuellaSHA256: huella, ReciboRef: recibo, Revision: revision, Estado: estado,
		Evidencia: ports.EvidenciaGobiernoCategoriaRPT{DecisionRef: r.DecisionRef(), EfectoRef: propuesta, HuellaEfectoSHA256: r.EfectoHuellaSHA256(),
			ConsumoHuellaSHA256: strings.Repeat("c", 64), AuditoriaRef: "auditoria:prueba", ConsumidaEn: r.EmitidaEn(), ConsumoNuevo: true}}, nil
}
func (g *gestorGobiernoRPTPrueba) ProponerGobiernoCategoriaRPT(_ context.Context, o ports.OrdenPropuestaGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	return g.resultado(o.Solicitud, o.Autorizacion, o.Material.PropuestaRef, o.Material.HuellaSHA256, o.Material.ReciboRef, 1, domain.EstadoGobiernoCategoriaRPTPropuesta)
}
func (g *gestorGobiernoRPTPrueba) AprobarGobiernoCategoriaRPT(_ context.Context, o ports.OrdenAvanceGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	estado := domain.EstadoGobiernoCategoriaRPTAprobada
	if o.Material.RevisionEsperada == 2 {
		estado = domain.EstadoGobiernoCategoriaRPTAprobada
	}
	return g.resultado(o.Solicitud, o.Autorizacion, o.Material.PropuestaRef, o.Material.HuellaSHA256, o.Material.ReciboRef, o.Material.RevisionEsperada+1, estado)
}
func (g *gestorGobiernoRPTPrueba) ConfirmarGobiernoCategoriaRPT(_ context.Context, o ports.OrdenAvanceGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error) {
	return g.resultado(o.Solicitud, o.Autorizacion, o.Material.PropuestaRef, o.Material.HuellaSHA256, o.Material.ReciboRef, 3, domain.EstadoGobiernoCategoriaRPTConfirmada)
}

func credencialesSegundoActorGobiernoRPT(t *testing.T, e *entornoAutorizacionSolicitudV3Prueba, cred CredencialesGobiernoCategoriaRPT) CredencialesGobiernoCategoriaRPT {
	t.Helper()
	i := cred.Actor.Instantanea
	i.CuentaRef = "cta_segunda0123456789abcdef"
	i.PersonaRef = "per_segunda0123456789abcdef"
	i.PerfilActivoRef = "prf_segunda0123456789abcdef"
	i.VinculoRef = "vca_segunda0123456789abcdef"
	cuenta := domain.CuentaAutenticadaContextoActor{CuentaRef: i.CuentaRef, Metodo: domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh}
	actor, err := domain.NuevoContextoActor(cuenta, i, cred.Actor.ResueltoEn)
	if err != nil {
		t.Fatal(err)
	}
	r := confirmacionRegistroContextoActorV2Prueba(t, actor, "oca_segunda0123456789abcdef")
	resultado := domain.ResultadoContextoActorRegistradoV2{RegistroContextoRef: r.RegistroContextoRef, Contexto: r.Contexto, RepresentacionCanonica: r.RepresentacionCanonica,
		HuellaSHA256: r.HuellaSHA256, ManifiestoProcedenciaCanonico: r.ManifiestoProcedenciaCanonico, ManifiestoProcedenciaHuellaSHA256: r.ManifiestoProcedenciaHuellaSHA256,
		AutoridadEfectiva: r.AutoridadEfectiva, ResueltoEnAutoritativo: r.ResueltoEnAutoritativo}
	v, err := cred.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	a := v.Autenticacion()
	a.CuentaRef = cuenta.CuentaRef
	a.CuentaOrdinariaRef = cuenta.CuentaRef
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(t.Context(), &revalidadorVinculoAplicacionAdversarial{resultado: a},
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: a.AutenticacionRef, SesionRef: a.SesionRef},
		resolutorContextoAutorizacionV3Prueba{resultado: resultado}, domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: i.PerfilActivoRef},
		&relojAutorizacionServicioPrueba{ahora: e.ahora})
	if err != nil {
		t.Fatal(err)
	}
	cred.Actor, cred.Vinculo, cred.ResultadoContexto = actor, vinculo, resultado
	e.fuente.instantanea.AsignacionPerfil.AsignacionID = "asignacion_segunda"
	e.fuente.instantanea.AsignacionPerfil.PrincipalID = actor.Principal.ID
	e.fuente.instantanea.AsignacionPerfil.PerfilActivoRef = actor.PerfilActivoRef
	return cred
}

func TestServicioGobiernoRPTDosActoresConPerfilesDistintosYRolComun(t *testing.T) {
	for _, segundo := range []bool{false, true} {
		for _, accion := range []string{ports.AccionProponerGobiernoCategoriaRPT, ports.AccionAprobarGobiernoCategoriaRPT, ports.AccionConfirmarGobiernoCategoriaRPT} {
			t.Run(accion+map[bool]string{false: "/actor_a", true: "/actor_b"}[segundo], func(t *testing.T) {
				e, emisor, cred, p := entornoEmisionGobiernoRPT(t)
				if segundo {
					cred = credencialesSegundoActorGobiernoRPT(t, e, cred)
				}
				e.fuente.instantanea.VersionRol.Concesiones[0].Accion = accion
				p.Accion = accion
				preparador := &preparadorGobiernoRPTPrueba{avance: p}
				gestor := &gestorGobiernoRPTPrueba{}
				s, err := NuevoServicioGobiernoCategoriaRPT(preparador, emisor, gestor, &relojAutorizacionServicioPrueba{ahora: e.ahora}, e.fuente.instantanea.VersionRol.Referencia())
				if err != nil {
					t.Fatal(err)
				}
				m := materialAvanceGobiernoRPTPrueba()
				switch accion {
				case ports.AccionProponerGobiernoCategoriaRPT:
					b := borradorPublicarGobiernoRPTAplicacionPrueba(t, cred)
					contenido, causa := b.Contenido.PrepararBorradorParaEditor(cred.Actor.Principal.ID)
					if causa != nil {
						t.Fatal(causa)
					}
					contenido.PreimagenesHuellaSHA256 = m.HuellaSHA256
					preparador.resultado = ports.PreparacionPropuestaGobiernoCategoriaRPT{Autorizable: p,
						Material: ports.MaterialPropuestaGobiernoCategoriaRPT{PropuestaRef: m.PropuestaRef, ReciboRef: m.ReciboRef, HuellaSHA256: m.HuellaSHA256, Contenido: contenido}}
					_, err = s.Proponer(t.Context(), OrdenProponerGobiernoCategoriaRPT{Credenciales: cred, Borrador: b})
				case ports.AccionAprobarGobiernoCategoriaRPT:
					_, err = s.Aprobar(t.Context(), OrdenAvanzarGobiernoCategoriaRPT{Credenciales: cred, Material: m})
				case ports.AccionConfirmarGobiernoCategoriaRPT:
					m.RevisionEsperada = 2
					_, err = s.Confirmar(t.Context(), OrdenAvanzarGobiernoCategoriaRPT{Credenciales: cred, Material: m})
				}
				if err != nil || gestor.llamadas != 1 || gestor.principal != cred.Actor.Principal.ID || gestor.perfil != cred.Actor.PerfilActivoRef || e.concesiones.invocaciones != 1 {
					t.Fatalf("acto individual: err=%v efectos=%d principal=%s perfil=%s concesiones=%d", err, gestor.llamadas, gestor.principal, gestor.perfil, e.concesiones.invocaciones)
				}
			})
		}
	}
}

func TestGobiernoRPTAplicacionDeniegaVinculoOrdinarioAjenoOVencido(t *testing.T) {
	for _, caso := range []string{"superficie ordinaria", "actor ajeno", "vinculo ajeno", "contexto vencido"} {
		t.Run(caso, func(t *testing.T) {
			e, emisor, cred, p := entornoEmisionGobiernoRPT(t)
			reloj := &relojAutorizacionServicioPrueba{ahora: e.ahora}
			s := &ServicioGobiernoCategoriaRPT{autorizador: emisor, reloj: reloj, versionRolRef: e.fuente.instantanea.VersionRol.Referencia()}
			switch caso {
			case "superficie ordinaria":
				v, _ := cred.Vinculo.Datos()
				a := v.Autenticacion()
				a.CuentaPrivilegiada = true
				a.Superficie = domain.SuperficieAutenticacionAdministracionPrivilegiadaV1
				a.CuentaOrdinariaRef = "cta_ordinaria0123456789abcdef"
				cred.Vinculo, _ = domain.CrearVinculoAutenticacionActorV2(t.Context(), &revalidadorVinculoAplicacionAdversarial{resultado: a}, domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: a.AutenticacionRef, SesionRef: a.SesionRef}, resolutorContextoAutorizacionV3Prueba{resultado: cred.ResultadoContexto}, domain.SolicitudContextoActor{Cuenta: domain.CuentaAutenticadaContextoActor{CuentaRef: a.CuentaRef, Metodo: a.MetodoObservado, Garantia: a.GarantiaObservada}, PerfilActivoRef: cred.Actor.PerfilActivoRef}, reloj)
			case "actor ajeno":
				otro := credencialesSegundoActorGobiernoRPT(t, e, cred)
				cred.Actor = otro.Actor
			case "vinculo ajeno":
				otro := credencialesSegundoActorGobiernoRPT(t, e, cred)
				cred.Vinculo = otro.Vinculo
			case "contexto vencido":
				reloj.ahora = e.ahora.Add(24 * time.Hour)
			}
			_, _, err := s.autorizar(t.Context(), cred, p, p.Accion, p.Recurso.Referencia, "catalogo.ejemplo", "bolsa", p.HuellaPropuesta)
			if !errors.Is(err, ports.ErrGobiernoCategoriaRPTDenegado) || e.concesiones.invocaciones != 0 {
				t.Fatalf("credencial denegada: err=%v concesiones=%d", err, e.concesiones.invocaciones)
			}
		})
	}
}

type emisorGobiernoRPTFalloPrueba struct {
	err      error
	cancelar context.CancelFunc
	llamadas int
}

func (e *emisorGobiernoRPTFalloPrueba) EmitirMaterialAutorizacionAtestadaV3(context.Context, domain.SolicitudAutorizacionLigadaV3, domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, ports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	if e.cancelar != nil {
		e.cancelar()
	}
	return domain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, e.err
}

func TestGobiernoRPTClasificaDenegacionExplicitaYFalloTecnicoDelEmisor(t *testing.T) {
	for _, caso := range []struct {
		nombre        string
		err, esperado error
		cancelar      bool
	}{
		{"denegacion explicita", ports.ErrDenegacionExplicitaAutorizacionLigadaV3, ports.ErrGobiernoCategoriaRPTDenegado, false},
		{"caida tecnica", errors.New("fallo privado del emisor"), ports.ErrGobiernoCategoriaRPTNoDisponible, false},
		{"exportador nulo", nil, ports.ErrGobiernoCategoriaRPTNoDisponible, false},
		{"cancelacion devuelta", context.Canceled, ports.ErrGobiernoCategoriaRPTNoDisponible, false},
		{"contexto cancelado", nil, context.Canceled, true},
		{"denegacion con contexto cancelado", ports.ErrDenegacionExplicitaAutorizacionLigadaV3, context.Canceled, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e, _, cred, p := entornoEmisionGobiernoRPT(t)
			ctx, cancelar := context.WithCancel(t.Context())
			defer cancelar()
			emisor := &emisorGobiernoRPTFalloPrueba{err: caso.err}
			if caso.cancelar {
				emisor.cancelar = cancelar
			}
			gestor := &gestorGobiernoRPTPrueba{}
			s, err := NuevoServicioGobiernoCategoriaRPT(&preparadorGobiernoRPTPrueba{avance: p}, emisor, gestor,
				&relojAutorizacionServicioPrueba{ahora: e.ahora}, e.fuente.instantanea.VersionRol.Referencia())
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.Aprobar(ctx, OrdenAvanzarGobiernoCategoriaRPT{Credenciales: cred, Material: materialAvanceGobiernoRPTPrueba()})
			if !errors.Is(err, caso.esperado) || strings.Contains(err.Error(), "fallo privado") || gestor.llamadas != 0 || emisor.llamadas != 1 {
				t.Fatalf("fallo del emisor: err=%v efectos=%d emision=%d", err, gestor.llamadas, emisor.llamadas)
			}
		})
	}
}

func borradorPublicarGobiernoRPTAplicacionPrueba(t *testing.T, cred CredencialesGobiernoCategoriaRPT) ports.BorradorPropuestaGobiernoCategoriaRPT {
	t.Helper()
	doc := domain.CatalogoConfigurable{ID: "catalogo.ejemplo", ModuloID: "bolsa", Version: 1, Revision: 1,
		Nombre: "Categorías sintéticas", FuenteRef: "fuente:prueba", MotivoCreacion: "motivo:prueba", Estado: domain.EstadoCatalogoPublicado,
		CreadoPor: cred.Actor.Principal.ID, CreadoEn: cred.Actor.ResueltoEn, PublicadoPor: "per_aprobadora0123456789abcdef", PublicadoEn: cred.Actor.ResueltoEn,
		AprobacionRef: "aprobacion:prevista", MotivoPublicacion: "motivo:prueba", Entradas: []domain.EntradaCatalogoConfigurable{{Clave: "categoria.ejemplo", Etiqueta: "Ejemplo", VigenteDesde: cred.Actor.ResueltoEn, Atributos: map[string]string{"estado": "habilitada"}}}}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	return ports.BorradorPropuestaGobiernoCategoriaRPT{PropuestaRef: "propuesta:ejemplo", ReciboRef: "recibo:ejemplo", Contenido: domain.ContenidoGobiernoCategoriaRPT{
		Accion: domain.AccionGobiernoCategoriaRPTPublicar, CatalogoID: doc.ID, ModuloID: doc.ModuloID, Version: doc.Version, DocumentoCanonico: &s,
		PreimagenesControl: map[string]domain.PreimagenControlGobiernoCategoriaRPT{}, MotivoRef: cred.Motivo.Referencia(), FuenteRef: doc.FuenteRef}}
}

func TestServicioGobiernoRPTUnaAprobacionYConfirmacionDesdeRevisionDos(t *testing.T) {
	for _, accion := range []string{ports.AccionAprobarGobiernoCategoriaRPT, ports.AccionConfirmarGobiernoCategoriaRPT} {
		for _, revision := range []int64{0, 1, 2, 3, 4} {
			e, emisor, cred, p := entornoEmisionGobiernoRPT(t)
			e.fuente.instantanea.VersionRol.Concesiones[0].Accion = accion
			p.Accion = accion
			preparador, gestor := &preparadorGobiernoRPTPrueba{avance: p}, &gestorGobiernoRPTPrueba{}
			s, err := NuevoServicioGobiernoCategoriaRPT(preparador, emisor, gestor, &relojAutorizacionServicioPrueba{ahora: e.ahora}, e.fuente.instantanea.VersionRol.Referencia())
			if err != nil {
				t.Fatal(err)
			}
			m := materialAvanceGobiernoRPTPrueba()
			m.RevisionEsperada = revision
			var r ports.ResultadoGobiernoCategoriaRPT
			if accion == ports.AccionAprobarGobiernoCategoriaRPT {
				r, err = s.Aprobar(t.Context(), OrdenAvanzarGobiernoCategoriaRPT{cred, m})
			} else {
				r, err = s.Confirmar(t.Context(), OrdenAvanzarGobiernoCategoriaRPT{cred, m})
			}
			valida := accion == ports.AccionAprobarGobiernoCategoriaRPT && revision == 1 || accion == ports.AccionConfirmarGobiernoCategoriaRPT && revision == 2
			if valida {
				if err != nil || r.Revision != revision+1 || gestor.llamadas != 1 {
					t.Fatalf("accion=%s revision=%d resultado=%+v err=%v", accion, revision, r, err)
				}
			} else if !errors.Is(err, ErrOrdenGobiernoCategoriaRPTInvalida) || preparador.llamadas != 0 || gestor.llamadas != 0 || e.concesiones.invocaciones != 0 {
				t.Fatalf("CAS ilegal llegó a dependencias: accion=%s revision=%d err=%v", accion, revision, err)
			}
		}
	}
}
