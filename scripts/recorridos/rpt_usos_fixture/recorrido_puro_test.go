package rptusosfixture

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"maps"
	"strings"
	"testing"
	"time"

	cose "github.com/veraison/go-cose"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Todo el contexto, concesiones y registro de esta prueba son dobles en memoria.
// COSE, confianza, capacidad y exportación usan sus implementaciones nominales;
// no existe conexión SQL ni esta prueba acredita persistencia o un E2E positivo.
type relojPrueba struct{ ahora time.Time }

func (r relojPrueba) Ahora() time.Time { return r.ahora }

type revalidadorPrueba struct {
	resultado domain.AutenticacionRevalidadaV1
}

func (r revalidadorPrueba) RevalidarAutenticacionActorV1(context.Context, domain.SolicitudRevalidacionAutenticacionActorV1) (domain.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type resolutorPrueba struct {
	resultado domain.ResultadoContextoActorRegistradoV2
}

func (r resolutorPrueba) ResolverContextoActorRegistradoV2(context.Context, domain.SolicitudContextoActor) (domain.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type correlacionPrueba string

func (c correlacionPrueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return string(c), nil
}

type registroPrueba struct{ ahora time.Time }

func (r registroPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	return r.ahora, nil
}

type firmantePrueba struct {
	privada ed25519.PrivateKey
	ahora   time.Time
	fallar  bool
}

func (f *firmantePrueba) FirmarAtestacionAutorizacionV3(_ context.Context, s ports.SolicitudFirmaAtestacionAutorizacionV3) (ports.ResultadoFirmaAtestacionAutorizacionV3, error) {
	if f.fallar {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, ports.ErrFirmaAtestacionNoDisponible
	}
	payload, err := s.Mensaje()
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	cabecera, err := s.Cabecera()
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	aad, err := confianza.AADExternoAtestacionAutorizacionV3(cabecera.Audiencia)
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	msg := cose.NewSign1Message()
	msg.Headers.Protected.SetAlgorithm(cose.AlgorithmEdDSA)
	msg.Headers.Protected[cose.HeaderLabelKeyID] = []byte(cabecera.ClaveID)
	msg.Payload = payload
	signer, err := cose.NewSigner(cose.AlgorithmEdDSA, f.privada)
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	if err = msg.Sign(rand.Reader, aad, signer); err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	msg.Payload = nil
	msg.Headers.RawProtected = nil
	msg.Headers.RawUnprotected = nil
	sobre, err := msg.MarshalCBOR()
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	return ports.NuevoResultadoFirmaAtestacionAutorizacionV3(s, sobre, "ensayo:firma:cose", f.ahora)
}

type autorizadorPrueba struct {
	ahora       time.Time
	decisionRef string
}

func (a autorizadorPrueba) ExigirSolicitudLigadaV3(ctx context.Context, s domain.SolicitudAutorizacionLigadaV3, rc domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	var dc domain.DecisionAutorizacionLigadaV3
	var cc ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	d, err := s.Datos()
	if err != nil {
		return dc, cc, err
	}
	v, err := d.VinculoAutenticacionActor.Datos()
	if err != nil {
		return dc, cc, err
	}
	version := domain.VersionRol{RolID: "usos_categorias", Version: 1, Nombre: "Ensayo", Estado: domain.EstadoVersionRolPublicada,
		Concesiones:  []domain.ConcesionRol{{Accion: d.Accion, ModuloID: d.Recurso.ModuloID, TipoRecurso: d.Recurso.Tipo, Finalidades: []string{d.Finalidad}, GarantiaMinima: domain.AuthAssuranceHigh}},
		PublicadaPor: "responsable:ensayo", PublicadaEn: a.ahora.Add(-time.Hour)}
	ambitos := make([]domain.AmbitoPerfil, 0, len(d.Recurso.Ambitos))
	for k, val := range d.Recurso.Ambitos {
		ambitos = append(ambitos, domain.AmbitoPerfil{Clave: k, Valores: []string{val}})
	}
	hc, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return dc, cc, err
	}
	inst := domain.InstantaneaAutorizacion{
		AsignacionPerfil: domain.AsignacionPerfil{AsignacionID: "asignacion:ensayo", Version: 1, PerfilActivoRef: v.PerfilActivoRef, PrincipalID: v.PrincipalID,
			VersionRolRef: version.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva, Ambitos: ambitos, VigenteDesde: a.ahora.Add(-time.Hour), VigenteHasta: a.ahora.Add(time.Hour), EmitidaPor: "responsable:ensayo", EmitidaEn: a.ahora.Add(-time.Hour)},
		VersionRol: version, ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{VersionRolRef: version.Referencia(), Revision: 1, Estado: domain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: hc}
	ev, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(s, inst, a.decisionRef, a.ahora, a.ahora.Add(90*time.Second))
	if err != nil {
		return dc, cc, err
	}
	dc, err = domain.NuevaDecisionAutorizacionLigadaV3(s, ev)
	if err != nil {
		return dc, cc, err
	}
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, dc, d.ReferenciaMotivo, rc)
	if err != nil {
		return dc, cc, err
	}
	cc, err = ports.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, registroPrueba{a.ahora}, orden)
	return dc, cc, err
}

type emisorPrueba struct {
	t         *testing.T
	ahora     time.Time
	vinculo   domain.VinculoAutenticacionActorV2
	resultado domain.ResultadoContextoActorRegistradoV2
	motivo    domain.ReferenciaEntradaCatalogo
	firmante  *firmantePrueba
	atestador *application.ServicioAtestacionesAutorizacionV3
	confianza *confianza.ServicioConfianzaAtestacionAutorizacionV3
	capacidad *confianza.EmisorCapacidadesAtestacionAutorizacionV3
	llamadas  int
	alterar   func(domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3)
	ultimoS   domain.SolicitudAutorizacionLigadaV3
	ultimoA   ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func nuevoEmisorPrueba(t *testing.T, audiencia string) *emisorPrueba {
	return nuevoEmisorActorPrueba(t, audiencia, false)
}
func nuevoEmisorActorPrueba(t *testing.T, audiencia string, otroActor bool) *emisorPrueba {
	t.Helper()
	ahora := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl",
		Metodo:    domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh,
	}
	instantaneaActor := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 3,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: 4,
		PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 5,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	if otroActor {
		instantaneaActor.PersonaRef = "per_aaaaaaaaaaaaaaaaaaaaaa"
		instantaneaActor.VinculoRef = "vca_aaaaaaaaaaaaaaaaaaaaaa"
	}
	actor, err := domain.NuevoContextoActor(
		cuenta,
		instantaneaActor,
		ahora.Add(-2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	representacion, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := domain.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:          "prc_0123456789abcdefghijkl",
		ProcedenciaVersion:      1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad:    domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := domain.ManifiestoProcedenciaContextoActorV1{
		Esquema:           domain.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: domain.ProcedenciaCuentaContextoActorV1{
			CuentaRef: cuenta.CuentaRef, Version: instantaneaActor.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: domain.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantaneaActor.PersonaRef, Version: instantaneaActor.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: domain.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantaneaActor.PerfilActivoRef, Version: instantaneaActor.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: domain.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantaneaActor.VinculoRef, Version: instantaneaActor.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: make([]domain.ProcedenciaVinculoReferenciaContextoActorV1, 0),
	}
	canonManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
		canonManifiesto,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn",
		Contexto:            actor, RepresentacionCanonica: representacion,
		HuellaSHA256:                      huella,
		ManifiestoProcedenciaCanonico:     canonManifiesto,
		ManifiestoProcedenciaHuellaSHA256: huellaManifiesto,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef:               "ase_0123456789abcdefghijkl",
		SesionRef:                 "ses_0123456789abcdefghijkl",
		ControlSesionRef:          "cse_0123456789abcdefghijkl",
		ControlSesionRevision:     2,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:      domain.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
		SesionValidaHasta:            ahora.Add(20 * time.Minute),
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(
		context.Background(),
		revalidadorPrueba{autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		resolutorPrueba{resultado},
		domain.SolicitudContextoActor{
			Cuenta: cuenta, PerfilActivoRef: instantaneaActor.PerfilActivoRef,
		},
		relojPrueba{ahora},
	)
	if err != nil {
		t.Fatal(err)
	}

	motivo := domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_11111111111111111111111111111111"}
	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA("clave:ensayo:rpt", 1, publica, "vec:ensayo:rpt", confianza.EstadoClaveAtestacionAutorizacionV3Activa, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	config, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3("confianza:ensayo:rpt", 1, ahora.Add(-time.Minute), ahora.Add(time.Hour), raiz)
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(config, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	clave := make([]byte, 32)
	if _, err = rand.Read(clave); err != nil {
		t.Fatal(err)
	}
	hmac, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3("clave:capacidad:ensayo", 1, clave, "emisor:ensayo", audiencia, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	clear(clave)
	capacidad, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(hmac, relojPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	firmante := &firmantePrueba{privada: privada, ahora: ahora}
	atestador, err := application.NuevoServicioAtestacionesAutorizacionV3(domain.CabeceraAtestacionAutorizacionV3{FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: "clave:ensayo:rpt", Audiencia: "vec:ensayo:rpt"}, firmante)
	if err != nil {
		t.Fatal(err)
	}
	return &emisorPrueba{t: t, ahora: ahora, vinculo: vinculo, resultado: resultado, motivo: motivo, firmante: firmante, atestador: atestador, confianza: servicio, capacidad: capacidad}
}
func (e *emisorPrueba) Emitir(ctx context.Context, p ports.PreparacionAutorizacionUsoCategoriaRPT) (domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	var a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, correlacionPrueba(fmt.Sprintf("correlacion_%032x", e.llamadas)))
	if err != nil {
		return domain.SolicitudAutorizacionLigadaV3{}, a, err
	}
	s, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: e.vinculo, ReferenciaMotivo: e.motivo, Accion: p.Accion, Recurso: p.Recurso, Finalidad: p.Finalidad, Correlacion: correlacion})
	if err != nil {
		return s, a, err
	}
	emisor, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(autorizadorPrueba{e.ahora, fmt.Sprintf("decision:ensayo:%032x", e.llamadas)}, e.atestador, e.confianza, e.capacidad)
	if err != nil {
		return s, a, err
	}
	_, _, exportador, err := emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, e.resultado)
	if err != nil {
		return s, a, err
	}
	a, err = exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return s, a, err
	}
	if e.alterar != nil {
		returnS, returnA := e.alterar(s, a)
		e.ultimoS, e.ultimoA = s, a
		return returnS, returnA, nil
	}
	e.ultimoS, e.ultimoA = s, a
	return s, a, nil
}

// El preparador es un doble con huella opaca. No imita jsonb::text; la
// canonicalización real se comprueba únicamente en PostgreSQL por su autoridad.
type preparadorPrueba struct {
	llamadas int
	alterar  func(*ports.PreparacionAutorizacionUsoCategoriaRPT)
	err      error
}

func (p *preparadorPrueba) preparar(accion string, m ports.MaterialReservaUsoCategoriaRPT) (ports.PreparacionAutorizacionUsoCategoriaRPT, error) {
	p.llamadas++
	out := ports.PreparacionAutorizacionUsoCategoriaRPT{Accion: accion, Finalidad: finalidadUsos, AudienciaConsumo: audienciaUsos, Recurso: domain.RecursoAutorizable{Referencia: m.UsoRef, ModuloID: "catalogos_configurables", Tipo: "uso_categoria", Ambitos: map[string]string{"catalogo_id": m.Publicacion.CatalogoID, "modulo_id": "catalogos_configurables", "consumidor": m.Consumidor}, Atributos: map[string]string{"material_sha256": strings.Repeat("f", 64)}}}
	if p.alterar != nil {
		p.alterar(&out)
	}
	return out, p.err
}
func (p *preparadorPrueba) PrepararReservaUsoCategoriaRPT(_ context.Context, m ports.MaterialReservaUsoCategoriaRPT) (ports.PreparacionAutorizacionUsoCategoriaRPT, error) {
	return p.preparar(accionReserva, m)
}
func (p *preparadorPrueba) PrepararConfirmacionUsoCategoriaRPT(_ context.Context, m ports.MaterialTerminalUsoCategoriaRPT) (ports.PreparacionAutorizacionUsoCategoriaRPT, error) {
	return p.preparar(accionConfirmacion, m.Reserva)
}
func (p *preparadorPrueba) PrepararCancelacionUsoCategoriaRPT(_ context.Context, m ports.MaterialTerminalUsoCategoriaRPT) (ports.PreparacionAutorizacionUsoCategoriaRPT, error) {
	return p.preparar(accionCancelacion, m.Reserva)
}

type gestorPrueba struct {
	usos     map[string]ports.UsoCategoriaRPT
	historia int
	llamadas int
	err      error
	alterar  func(*ports.ResultadoUsoCategoriaRPT)
}

func (g *gestorPrueba) actuar(m ports.MaterialReservaUsoCategoriaRPT, t ports.MaterialTerminalUsoCategoriaRPT, estado string, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ResultadoUsoCategoriaRPT, error) {
	g.llamadas++
	if g.err != nil {
		return ports.ResultadoUsoCategoriaRPT{}, g.err
	}
	if g.usos == nil {
		g.usos = make(map[string]ports.UsoCategoriaRPT)
	}
	u, existe := g.usos[m.UsoRef]
	if !existe {
		u = ports.UsoCategoriaRPT{Consumidor: m.Consumidor, UsoRef: m.UsoRef, CategoriaID: m.CategoriaID, Publicacion: m.Publicacion, Estado: "reservado", Revision: 1, ReservaReciboRef: m.ReservaReciboRef, ReservadoEn: time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)}
		g.historia++
	}
	if estado != "reservado" && u.Estado == "reservado" {
		recibo := t.TerminalReciboRef
		instante := u.ReservadoEn.Add(time.Second)
		u.TerminalReciboRef = &recibo
		u.TerminalEn = &instante
		u.Revision = 2
		u.Estado = estado
		g.historia++
	}
	g.usos[m.UsoRef] = u
	resumen := a.ResumenCapacidad()
	out := ports.ResultadoUsoCategoriaRPT{Encontrado: true, Uso: &u, Evidencia: ports.EvidenciaLecturaRPT{DecisionRef: resumen.DecisionRef(), EfectoRef: resumen.EfectoRef(), HuellaEfectoSHA256: resumen.EfectoHuellaSHA256(), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: fmt.Sprintf("auditoria:%d", g.llamadas), ConsumidaEn: u.ReservadoEn, ConsumoNuevo: true}}
	if g.alterar != nil {
		g.alterar(&out)
	}
	return out, nil
}
func (g *gestorPrueba) ReservarUsoCategoriaRPT(_ context.Context, o ports.OrdenReservaUsoCategoriaRPT) (ports.ResultadoUsoCategoriaRPT, error) {
	return g.actuar(o.Material, ports.MaterialTerminalUsoCategoriaRPT{}, "reservado", o.Autorizacion)
}
func (g *gestorPrueba) ConfirmarUsoCategoriaRPT(_ context.Context, o ports.OrdenConfirmacionUsoCategoriaRPT) (ports.ResultadoUsoCategoriaRPT, error) {
	return g.actuar(o.Material.Reserva, o.Material, "confirmado", o.Autorizacion)
}
func (g *gestorPrueba) CancelarUsoCategoriaRPT(_ context.Context, o ports.OrdenCancelacionUsoCategoriaRPT) (ports.ResultadoUsoCategoriaRPT, error) {
	return g.actuar(o.Material.Reserva, o.Material, "cancelado", o.Autorizacion)
}

func entradaPrueba() Entrada {
	terminal := func(sufijo string) (ports.MaterialTerminalUsoCategoriaRPT, EvidenciaSintetica) {
		e := EvidenciaSintetica{Referencia: "fixture:rpt:terminal:" + sufijo, Rotulo: RotuloEntradaEnsayo, Bytes: []byte(RotuloEntradaEnsayo + "\n" + sufijo)}
		m := ports.MaterialTerminalUsoCategoriaRPT{Reserva: ports.MaterialReservaUsoCategoriaRPT{Consumidor: "contratacion_temporal", UsoRef: "uso:ensayo:" + sufijo, CategoriaID: "auxiliar", Publicacion: ports.ReferenciaPublicacionRPT{CatalogoID: "rpt", Version: 1, HuellaSHA256: strings.Repeat("b", 64)}, ReservaReciboRef: "recibo:reserva:" + sufijo}, TerminalReciboRef: "recibo:terminal:" + sufijo, EvidenciaRef: e.Referencia, EvidenciaSHA256: hash(e.Bytes)}
		return m, e
	}
	c, ec := terminal("confirmar")
	x, ex := terminal("cancelar")
	return Entrada{Confirmacion: c, Cancelacion: x, EvidenciaConfirmacion: ec, EvidenciaCancelacion: ex}
}

func TestRecorridoCOSERealYReplayConV3Nueva(t *testing.T) {
	p, e, g := &preparadorPrueba{}, nuevoEmisorPrueba(t, audienciaUsos), &gestorPrueba{}
	r, err := NuevoRecorrido(p, e, g)
	if err != nil {
		t.Fatal(err)
	}
	entrada := entradaPrueba()
	primero, err := r.Ejecutar(context.Background(), entrada)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := r.Ejecutar(context.Background(), entrada)
	if err != nil {
		t.Fatal(err)
	}
	if err = CotejarReplay(primero, segundo); err != nil {
		t.Fatal(err)
	}
	if p.llamadas != 8 || e.llamadas != 8 || g.llamadas != 8 || g.historia != 4 || len(g.usos) != 2 {
		t.Fatalf("conteos de coordinación inesperados: %d/%d/%d historia=%d", p.llamadas, e.llamadas, g.llamadas, g.historia)
	}
	if primero.Confirmacion.Uso.Estado != "confirmado" || primero.Cancelacion.Uso.Estado != "cancelado" {
		t.Fatal("terminales incorrectos")
	}
	// No hay documento de entrada en el resultado ni recuperación por otro POST.
	if bytes.Equal(e.ultimoA.SobreCOSESign1(), []byte{}) {
		t.Fatal("COSE vacío")
	}
}

func TestRecorridoFalloFirmaNoLlamaGestor(t *testing.T) {
	e := nuevoEmisorPrueba(t, audienciaUsos)
	e.firmante.fallar = true
	g := &gestorPrueba{}
	r, _ := NuevoRecorrido(&preparadorPrueba{}, e, g)
	_, err := r.Ejecutar(context.Background(), entradaPrueba())
	if err == nil || g.llamadas != 0 || g.historia != 0 {
		t.Fatal("fallo de firma no cerrado antes del gestor")
	}
}

func TestRecorridoRechazaPreparacionAlterada(t *testing.T) {
	casos := map[string]func(*ports.PreparacionAutorizacionUsoCategoriaRPT){
		"audiencia":      func(p *ports.PreparacionAutorizacionUsoCategoriaRPT) { p.AudienciaConsumo = "otra" },
		"accion":         func(p *ports.PreparacionAutorizacionUsoCategoriaRPT) { p.Accion = accionCancelacion },
		"ambito":         func(p *ports.PreparacionAutorizacionUsoCategoriaRPT) { p.Recurso.Ambitos["consumidor"] = "personal" },
		"atributo_extra": func(p *ports.PreparacionAutorizacionUsoCategoriaRPT) { p.Recurso.Atributos["extra"] = "valor" },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEmisorPrueba(t, audienciaUsos)
			g := &gestorPrueba{}
			p := &preparadorPrueba{alterar: cambiar}
			r, _ := NuevoRecorrido(p, e, g)
			_, err := r.Ejecutar(context.Background(), entradaPrueba())
			if err == nil || e.llamadas != 0 || g.llamadas != 0 {
				t.Fatal("preparación alterada llegó a emisión o efecto")
			}
		})
	}
}

func TestRecorridoDeniegaMezclaSolicitudMaterial(t *testing.T) {
	casos := map[string]func(*domain.DatosSolicitudAutorizacionLigadaV3){
		"material": func(d *domain.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos = maps.Clone(d.Recurso.Atributos)
			d.Recurso.Atributos["material_sha256"] = strings.Repeat("e", 64)
		},
		"motivo": func(d *domain.DatosSolicitudAutorizacionLigadaV3) { d.ReferenciaMotivo.CatalogoVersion++ },
		"correlacion": func(d *domain.DatosSolicitudAutorizacionLigadaV3) {
			c, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), correlacionPrueba("correlacion_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
			if err != nil {
				t.Fatal(err)
			}
			d.Correlacion = c
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEmisorPrueba(t, audienciaUsos)
			g := &gestorPrueba{}
			e.alterar = func(s domain.SolicitudAutorizacionLigadaV3, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
				d, err := s.Datos()
				if err != nil {
					t.Fatal(err)
				}
				cambiar(&d)
				s, err = domain.NuevaSolicitudAutorizacionLigadaV3(d)
				if err != nil {
					t.Fatal(err)
				}
				return s, a
			}
			r, _ := NuevoRecorrido(&preparadorPrueba{}, e, g)
			_, err := r.Ejecutar(context.Background(), entradaPrueba())
			if !errors.Is(err, ports.ErrUsoCategoriaRPTDenegado) || g.llamadas != 0 {
				t.Fatal("mezcla admitida")
			}
		})
	}
}

func TestRecorridoDeniegaAudienciaMaterial(t *testing.T) {
	e := nuevoEmisorPrueba(t, "audiencia:ajena")
	g := &gestorPrueba{}
	r, _ := NuevoRecorrido(&preparadorPrueba{}, e, g)
	_, err := r.Ejecutar(context.Background(), entradaPrueba())
	if !errors.Is(err, ports.ErrUsoCategoriaRPTDenegado) || g.llamadas != 0 {
		t.Fatal("audiencia ajena admitida")
	}
}

func TestRecorridoCapacidadUsadaDeNuevo(t *testing.T) {
	e := nuevoEmisorPrueba(t, audienciaUsos)
	g := &gestorPrueba{}
	r, _ := NuevoRecorrido(&preparadorPrueba{}, e, g)
	m := entradaPrueba().Confirmacion.Reserva
	if _, err := r.Reserva(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	previaS, previaA := e.ultimoS, e.ultimoA
	e.alterar = func(domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
		return previaS, previaA
	}
	_, err := r.Reserva(context.Background(), m)
	if !errors.Is(err, ports.ErrUsoCategoriaRPTDenegado) || g.llamadas != 1 {
		t.Fatal("capacidad repetida llegó al gestor")
	}
}

func TestRecorridoEvidenciaInvalidaNoEscribe(t *testing.T) {
	casos := map[string]func(*Entrada){
		"bytes":          func(e *Entrada) { e.EvidenciaCancelacion.Bytes = []byte(RotuloEntradaEnsayo + "\ncambiado") },
		"rotulo":         func(e *Entrada) { e.EvidenciaConfirmacion.Rotulo = "resolucion" },
		"referencia":     func(e *Entrada) { e.Confirmacion.EvidenciaRef = "efecto:ct:real" },
		"uso_repetido":   func(e *Entrada) { e.Cancelacion.Reserva.UsoRef = e.Confirmacion.Reserva.UsoRef },
		"recibo_cruzado": func(e *Entrada) { e.Cancelacion.TerminalReciboRef = e.Confirmacion.Reserva.ReservaReciboRef },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			p := &preparadorPrueba{}
			e := nuevoEmisorPrueba(t, audienciaUsos)
			g := &gestorPrueba{}
			r, _ := NuevoRecorrido(p, e, g)
			entrada := entradaPrueba()
			cambiar(&entrada)
			_, err := r.Ejecutar(context.Background(), entrada)
			if !errors.Is(err, ports.ErrUsoCategoriaRPTInvalido) || p.llamadas != 0 || e.llamadas != 0 || g.llamadas != 0 {
				t.Fatal("evidencia no valida inició el recorrido")
			}
		})
	}
}

func TestRecorridoRechazaReciboVersionYConsumoAlterados(t *testing.T) {
	casos := map[string]func(*ports.ResultadoUsoCategoriaRPT){
		"version":       func(o *ports.ResultadoUsoCategoriaRPT) { o.Uso.Revision++ },
		"recibo":        func(o *ports.ResultadoUsoCategoriaRPT) { o.Uso.ReservaReciboRef = "recibo:ajeno" },
		"consumo_viejo": func(o *ports.ResultadoUsoCategoriaRPT) { o.Evidencia.ConsumoNuevo = false },
		"decision":      func(o *ports.ResultadoUsoCategoriaRPT) { o.Evidencia.DecisionRef = "decision:ajena" },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEmisorPrueba(t, audienciaUsos)
			g := &gestorPrueba{alterar: cambiar}
			r, _ := NuevoRecorrido(&preparadorPrueba{}, e, g)
			out, err := r.Ejecutar(context.Background(), entradaPrueba())
			if !errors.Is(err, ports.ErrUsoCategoriaRPTNoConfiable) || g.llamadas != 1 || out.ReservaConfirmacion.Uso == nil {
				t.Fatal("recibo no confiable no detenido/conservado")
			}
		})
	}
}

func TestRecorridoDeniegaDependenciasYContexto(t *testing.T) {
	var p *preparadorPrueba
	if _, err := NuevoRecorrido(p, (*emisorPrueba)(nil), (*gestorPrueba)(nil)); err == nil {
		t.Fatal("dependencias nulas")
	}
	e := nuevoEmisorPrueba(t, audienciaUsos)
	g := &gestorPrueba{}
	r, _ := NuevoRecorrido(&preparadorPrueba{}, e, g)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	for _, c := range []context.Context{nil, ctx} {
		if _, err := r.Ejecutar(c, entradaPrueba()); err == nil {
			t.Fatal("contexto inválido")
		}
	}
	if g.llamadas != 0 || e.llamadas != 0 {
		t.Fatal("contexto inválido produjo efecto")
	}
}

func TestRecorridoDeniegaActorMezclado(t *testing.T) {
	e := nuevoEmisorPrueba(t, audienciaUsos)
	otro := nuevoEmisorActorPrueba(t, audienciaUsos, true)
	e.alterar = func(s domain.SolicitudAutorizacionLigadaV3, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
		d, err := s.Datos()
		if err != nil {
			t.Fatal(err)
		}
		d.VinculoAutenticacionActor = otro.vinculo
		s, err = domain.NuevaSolicitudAutorizacionLigadaV3(d)
		if err != nil {
			t.Fatal(err)
		}
		return s, a
	}
	g := &gestorPrueba{}
	r, _ := NuevoRecorrido(&preparadorPrueba{}, e, g)
	_, err := r.Ejecutar(context.Background(), entradaPrueba())
	if !errors.Is(err, ports.ErrUsoCategoriaRPTDenegado) || g.llamadas != 0 {
		t.Fatal("actor ajeno llegó al gestor")
	}
}

func TestRecorridoConservaRecibosPreviosAlFalloTerminal(t *testing.T) {
	e := nuevoEmisorPrueba(t, audienciaUsos)
	g := &gestorPrueba{}
	r, _ := NuevoRecorrido(&preparadorPrueba{}, e, g)
	e.alterar = func(s domain.SolicitudAutorizacionLigadaV3, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
		e.firmante.fallar = true
		return s, a
	}
	out, err := r.Ejecutar(context.Background(), entradaPrueba())
	if err == nil || out.ReservaConfirmacion.Uso == nil || out.Confirmacion.Uso != nil || g.historia != 1 {
		t.Fatal("recibo previo perdido o terminal escrito tras fallo de firma")
	}
}

func TestCotejarReplayDeniegaCambioReciboFechaYVersion(t *testing.T) {
	casos := map[string]func(*Resultado){
		"version":         func(r *Resultado) { r.Confirmacion.Uso.Revision++ },
		"fecha":           func(r *Resultado) { r.Cancelacion.Uso.ReservadoEn = r.Cancelacion.Uso.ReservadoEn.Add(time.Second) },
		"recibo_terminal": func(r *Resultado) { s := "recibo:alterado"; r.Cancelacion.Uso.TerminalReciboRef = &s },
		"decision_vieja":  func(r *Resultado) { r.Cancelacion.Evidencia.DecisionRef = r.Confirmacion.Evidencia.DecisionRef },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEmisorPrueba(t, audienciaUsos)
			g := &gestorPrueba{}
			r, _ := NuevoRecorrido(&preparadorPrueba{}, e, g)
			primero, err := r.Ejecutar(context.Background(), entradaPrueba())
			if err != nil {
				t.Fatal(err)
			}
			segundo, err := r.Ejecutar(context.Background(), entradaPrueba())
			if err != nil {
				t.Fatal(err)
			}
			alterar(&segundo)
			if !errors.Is(CotejarReplay(primero, segundo), ports.ErrUsoCategoriaRPTNoConfiable) {
				t.Fatal("replay cambiado admitido")
			}
		})
	}
}
