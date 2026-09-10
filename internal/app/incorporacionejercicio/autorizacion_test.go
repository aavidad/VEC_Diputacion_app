package incorporacionejercicio

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	cose "github.com/veraison/go-cose"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	cd "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pa "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	fuente "vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	pl "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Gobierno, registro y firmante de prueba son explícitos; PDP, COSE, verificador,
// emisor, órdenes y confirmaciones se ejecutan por sus APIs nominales reales.
type autoridadStoreDoble struct {
	snapshot  core.InstantaneaAutorizacion
	ahora     time.Time
	registros int
	mu        sync.Mutex
	fallo     error
	despues   func()
}

func (d *autoridadStoreDoble) ObtenerInstantaneaAutorizacion(_ context.Context, principal, perfil string) (core.InstantaneaAutorizacion, error) {
	if principal != d.snapshot.AsignacionPerfil.PrincipalID || perfil != d.snapshot.AsignacionPerfil.PerfilActivoRef {
		return core.InstantaneaAutorizacion{}, ErrAutoridadAplicacion
	}
	return d.snapshot, d.fallo
}
func (d *autoridadStoreDoble) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, vp.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.registros++
	if d.despues != nil {
		d.despues()
	}
	return d.ahora, d.fallo
}
func (d *autoridadStoreDoble) RegistrarDenegacionAutorizacionLigadaV3(context.Context, vp.OrdenRegistroDenegacionAutorizacionLigadaV3) error {
	return nil
}

type autoridadMotivosDoble struct {
	motivo core.ReferenciaEntradaCatalogo
}

func (d autoridadMotivosDoble) ValidarReferenciaMotivoAutorizacionV2(_ context.Context, m core.ReferenciaEntradaCatalogo, _ time.Time) error {
	if m != d.motivo {
		return ErrAutoridadAplicacion
	}
	return nil
}

type autoridadGeneradorDoble struct{ n atomic.Uint64 }

func (d *autoridadGeneradorDoble) NuevaReferenciaDecisionAutorizacion() (string, error) {
	return fmt.Sprintf("dec_%032x", d.n.Add(1)), nil
}
func (d *autoridadGeneradorDoble) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return fmt.Sprintf("correlacion_%032x", d.n.Add(1)), nil
}

type autoridadFirmanteDoble struct {
	privada ed25519.PrivateKey
	ahora   time.Time
	fallo   error
	despues func()
}

func (f *autoridadFirmanteDoble) FirmarAtestacionAutorizacionV3(_ context.Context, s vp.SolicitudFirmaAtestacionAutorizacionV3) (vp.ResultadoFirmaAtestacionAutorizacionV3, error) {
	var cero vp.ResultadoFirmaAtestacionAutorizacionV3
	if f.fallo != nil {
		return cero, f.fallo
	}
	b, e := s.Mensaje()
	if e != nil {
		return cero, e
	}
	defer clear(b)
	aad, e := confianza.AADExternoAtestacionAutorizacionV3("vec/prueba/autoridad")
	if e != nil {
		return cero, e
	}
	m := cose.NewSign1Message()
	m.Headers.Protected.SetAlgorithm(cose.AlgorithmEdDSA)
	m.Headers.Protected[cose.HeaderLabelKeyID] = []byte("clave:prueba:autoridad")
	m.Payload = b
	firm, e := cose.NewSigner(cose.AlgorithmEdDSA, f.privada)
	if e != nil {
		return cero, e
	}
	if e = m.Sign(rand.Reader, aad, firm); e != nil {
		return cero, e
	}
	m.Payload = nil
	m.Headers.RawProtected = nil
	m.Headers.RawUnprotected = nil
	sobre, e := m.MarshalCBOR()
	if e != nil {
		return cero, e
	}
	defer clear(sobre)
	if f.despues != nil {
		f.despues()
	}
	return vp.NuevoResultadoFirmaAtestacionAutorizacionV3(s, sobre, "evidencia:prueba:autoridad", f.ahora)
}

type autoridadEscenario struct {
	a        *AutoridadAplicacion
	ahora    time.Time
	reloj    autoridadRelojDoble
	fuente   autoridadFuenteDoble
	reval    autoridadRevalidadorDoble
	gen      *autoridadGeneradorDoble
	store    *autoridadStoreDoble
	firmante *autoridadFirmanteDoble
}

func autoridadExigir(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal("fixture nominal no construida")
	}
}
func autoridadEntorno(t *testing.T, operaciones ...autoridadOperacion) *autoridadEscenario {
	t.Helper()
	e := &autoridadEscenario{ahora: time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC), gen: &autoridadGeneradorDoble{}}
	e.reloj = autoridadRelojDoble{instante: e.ahora}
	marcaPerfil := "a"
	if len(operaciones) > 0 && operaciones[0] == autoridadAlta {
		marcaPerfil = "b"
	}
	c, aut, sc := autoridadFixtureContexto(t, e.ahora, "a", marcaPerfil)
	e.reval = autoridadRevalidadorDoble{resultado: aut}
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 2, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_11111111111111111111111111111111"}
	e.fuente = autoridadFuenteDoble{p: PeticionAutoridad{Autenticacion: core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: aut.AutenticacionRef, SesionRef: aut.SesionRef}, Contexto: sc, PreparacionCT: ct.PreparacionSeguimientoConfirmacionIncorporacion{OrganizacionRef: "organizacion:ejercicio:personal", UnidadRef: "ref:" + strings.Repeat("e", 64), ActorRef: "actor:ejercicio:rrhh", CorrelacionRef: "correlacion:ejercicio:ct"}, MotivoAlta: motivo, MotivoLectura: motivo}}
	v, _ := c.Vinculo.Datos()
	rol := core.VersionRol{RolID: "rrhh_ejercicio", Version: 1, Nombre: "Prueba no productiva", Estado: core.EstadoVersionRolPublicada, PublicadaPor: "seguridad:ejercicio", PublicadaEn: e.ahora.Add(-time.Hour)}
	for _, op := range []autoridadOperacion{autoridadAlta, autoridadLectura, autoridadCT} {
		k, _ := autoridadContratoPara(op)
		rol.Concesiones = append(rol.Concesiones, core.ConcesionRol{Accion: k.accion, ModuloID: k.modulo, TipoRecurso: k.tipo, Finalidades: []string{k.finalidad}, GarantiaMinima: core.AuthAssuranceHigh})
	}
	h, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
	autoridadExigir(t, err)
	snap := core.InstantaneaAutorizacion{VersionRol: rol, AsignacionPerfil: core.AsignacionPerfil{AsignacionID: "asignacion:ejercicio", Version: 1, PerfilActivoRef: v.PerfilActivoRef, PrincipalID: v.PrincipalID, VersionRolRef: rol.Referencia(), Estado: core.EstadoAsignacionPerfilActiva, Ambitos: []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{e.fuente.p.PreparacionCT.OrganizacionRef}}, {Clave: "unidad_ref", Valores: []string{e.fuente.p.PreparacionCT.UnidadRef}}, {Clave: "centro_ref", Valores: []string{"centro:ejercicio:0001"}}}, EmitidaPor: "identidad:ejercicio", EmitidaEn: e.ahora.Add(-2 * time.Hour), VigenteDesde: e.ahora.Add(-time.Hour), VigenteHasta: e.ahora.Add(time.Hour)}, ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: core.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: rol.PublicadaPor, ActualizadoEn: rol.PublicadaEn}, RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: h}
	if len(operaciones) > 0 && operaciones[0] == autoridadAlta {
		snap.AsignacionPerfil.Ambitos = append(snap.AsignacionPerfil.Ambitos[:1], snap.AsignacionPerfil.Ambitos[2])
	} else {
		snap.AsignacionPerfil.Ambitos = snap.AsignacionPerfil.Ambitos[:2]
	}
	autoridadExigir(t, snap.Validar())
	e.store = &autoridadStoreDoble{snapshot: snap, ahora: e.ahora}
	servicio, err := app.NuevoServicioAutorizacionSolicitudLigadaV3(e.store, e.store, e.store, autoridadMotivosDoble{motivo}, e.reloj, e.gen, app.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	autoridadExigir(t, err)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	autoridadExigir(t, err)
	t.Cleanup(func() { clear(priv) })
	e.firmante = &autoridadFirmanteDoble{privada: priv, ahora: e.ahora}
	at, err := app.NuevoServicioAtestacionesAutorizacionV3(core.CabeceraAtestacionAutorizacionV3{FormatoVersion: core.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: "clave:prueba:autoridad", Audiencia: "vec/prueba/autoridad"}, e.firmante)
	autoridadExigir(t, err)
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA("clave:prueba:autoridad", 1, pub, "vec/prueba/autoridad", confianza.EstadoClaveAtestacionAutorizacionV3Activa, e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{})
	autoridadExigir(t, err)
	config, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3("confianza:ejercicio", 1, e.ahora.Add(-time.Minute), e.ahora.Add(time.Hour), raiz)
	autoridadExigir(t, err)
	ver, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(config, e.reloj)
	autoridadExigir(t, err)
	emisiones := EmisionesAutoridad{}
	for i, p := range []*EmisionAutoridad{&emisiones.Alta, &emisiones.Lectura, &emisiones.CT} {
		k, _ := autoridadContratoPara(autoridadOperacion(i + 1))
		secreto := make([]byte, 32)
		_, err = rand.Read(secreto)
		autoridadExigir(t, err)
		clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(fmt.Sprintf("clave:ejercicio:%d", i), 1, secreto, "emisor:ejercicio", k.audiencia, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("7", 64))
		clear(secreto)
		autoridadExigir(t, err)
		em, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, e.reloj)
		autoridadExigir(t, err)
		*p = EmisionAutoridad{Emisor: em, Raiz: raiz}
	}
	cadena, err := NuevaCadenaAutorizacionAplicacion(servicio, at, ver, emisiones)
	autoridadExigir(t, err)
	e.a, err = NuevaAutoridadAplicacion(context.Background(), e.fuente, e.reval, autoridadContextoDoble{c.Resultado}, cadena, e.gen, e.reloj)
	autoridadExigir(t, err)
	return e
}
func autoridadMaterialAlta(t *testing.T, e *autoridadEscenario) pa.MaterialAlta {
	t.Helper()
	s := ct.SolicitudAltaPersonalRPT{Esquema: ct.EsquemaAltaPersonalRPT, ContratoVersion: 1, SolicitudRef: "solicitud:ejercicio:0001", ExpedienteRef: "expediente:ejercicio:ct:0001", VersionExpediente: 7, CapacidadRef: "capacidad:ejercicio:0001", CorrelacionRef: "correlacion:ejercicio:0001", IdempotenciaRef: "idempotencia:ejercicio:0001", FuenteRPT: ct.ReferenciaVersionadaPersonalRPT{Referencia: "rpt:ejercicio:20260908", Version: 3, HuellaSHA256: "a81229be023f01b99c16b76c9d4f6283dc6efaf5177340b063d36ab30dc654ac"}, PuestoRef: "puesto:ejercicio:0001", PlazaRef: "plaza:ejercicio:0001"}
	b, err := os.ReadFile("../../modules/personal/adapters/fuenteejercicio/testdata/fuente-ejercicio.json")
	autoridadExigir(t, err)
	terna := fuente.TernaEsperada{Referencia: "fuente:personal:ejercicio:20260908", Version: 1, HuellaSHA256: "c8f1d4a18721e50bac27cbc803d226b644b910b1a7f5523d8935d3d4b47d320f"}
	f, err := fuente.NuevaFuenteEjercicio(b, terna)
	autoridadExigir(t, err)
	v, err := f.Resolver(context.Background(), s)
	autoridadExigir(t, err)
	actor, _ := e.a.contexto.Vinculo.Datos()
	m := pa.MaterialAlta{Preparacion: pa.PreparacionAlta{Solicitud: s, Fuente: terna, Vinculo: v}, OrganizacionRef: e.fuente.p.PreparacionCT.OrganizacionRef, ActorRef: actor.PrincipalID, PerfilRef: actor.PerfilActivoRef}
	autoridadExigir(t, m.Validar())
	return m
}
func autoridadSelector(e *autoridadEscenario) pl.Selector {
	return pl.Selector{OrganizacionRef: e.fuente.p.PreparacionCT.OrganizacionRef, SolicitudRef: "solicitud:ejercicio:0001", ExpedienteRef: "expediente:ejercicio:ct:0001", VersionExpediente: 7, ResultadoRef: "resultado:ejercicio:0001", ReciboRef: "recibo:ejercicio:0001", RelacionRef: "relacion:ejercicio:0001", OcupacionRef: "ocupacion:ejercicio:0001", MaterialSHA256: strings.Repeat("a", 64)}
}
func autoridadMaterialCT(t *testing.T, e *autoridadEscenario, original ...pa.MaterialAlta) ct.DatosMaterialConfirmacionIncorporacionV2 {
	m := autoridadMaterialAlta(t, e)
	if len(original) > 0 {
		m = original[0]
	}
	s := m.Preparacion.Solicitud
	h, err := m.HuellaSHA256()
	autoridadExigir(t, err)
	hs, err := s.HuellaSHA256()
	autoridadExigir(t, err)
	b, err := json.Marshal(struct {
		Esquema  string
		Material pa.MaterialAlta
	}{pa.EsquemaMaterialAlta, m})
	autoridadExigir(t, err)
	r := ct.ResultadoAltaPersonalRPT{Esquema: ct.EsquemaAltaPersonalRPT, ContratoVersion: 1, ResultadoRef: "resultado:ejercicio:0001", ReciboRef: "recibo:ejercicio:0001", SolicitudRef: s.SolicitudRef, CorrelacionRef: s.CorrelacionRef, IdempotenciaRef: s.IdempotenciaRef, HuellaSolicitudSHA256: hs, Estado: ct.AltaPersonalRPTConfirmada, RelacionRef: "relacion:ejercicio:0001", OcupacionRef: "ocupacion:ejercicio:0001"}
	cor, err := core.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), e.gen)
	autoridadExigir(t, err)
	return ct.DatosMaterialConfirmacionIncorporacionV2{Confirmacion: ct.DatosConfirmacionIncorporacion{SolicitudPersonal: s, ResultadoPersonal: r, VersionSeguimientoEsperada: 0, PeriodoIncorporacion: cd.IntervaloSeguimiento{Desde: e.ahora.Add(time.Hour), Hasta: e.ahora.Add(25 * time.Hour)}, MotivoClave: "revision_ejercicio", Documentos: []cd.DocumentoSeguimiento{{TipoClave: "resolucion", Referencia: "documento:ejercicio:resolucion"}}}, VersionActualExpediente: 7, Preparacion: e.fuente.p.PreparacionCT, SolicitudContexto: e.a.solicitudContexto(), Contexto: e.a.contexto, MotivoV3: e.fuente.p.MotivoAlta, CorrelacionV3: cor, Personal: ct.RegistroPersonalEjercicio{Solicitud: s, Resultado: r, MaterialCanonico: b, MaterialSHA256: h, RegistradoEn: e.ahora.Add(-time.Hour), DecisionOriginalRef: "decision:ejercicio:personal", AuditoriaRef: "auditoria:ejercicio:personal", OutboxRef: "outbox:ejercicio:personal", EjercicioSintetico: true}, EjercicioSintetico: true}
}
func TestAutoridadAplicacionNominalIndependiente(t *testing.T) {
	eAlta := autoridadEntorno(t, autoridadAlta)
	e := autoridadEntorno(t)
	ctx := context.Background()
	e.gen.n.Store(100)
	m := autoridadMaterialAlta(t, eAlta)
	au, err := eAlta.a.ResolverAutoridad(ctx, m.Preparacion)
	autoridadExigir(t, err)
	if au.OrganizacionRef != m.OrganizacionRef {
		t.Fatal("organizacion")
	}
	alta, err := eAlta.a.AutorizarAlta(ctx, m)
	autoridadExigir(t, err)
	lectura, err := pl.NuevoMaterialV2(autoridadSelector(e), e.fuente.p.PreparacionCT.UnidadRef, e.a.contexto, e.ahora)
	autoridadExigir(t, err)
	pre, err := e.a.AutorizarLecturaIncorporacionV2(ctx, lectura)
	autoridadExigir(t, err)
	datos := autoridadMaterialCT(t, e, m)
	ctm, err := ct.NuevoMaterialConfirmacionIncorporacionV2(datos, e.ahora)
	autoridadExigir(t, err)
	cta, err := e.a.AutorizarConfirmacionIncorporacion(ctx, ctm)
	autoridadExigir(t, err)
	autoridadExigir(t, cta.ValidarPara(ctm, e.ahora))
	commit, err := e.a.AutorizarLecturaIncorporacionV2(ctx, lectura)
	autoridadExigir(t, err)
	vistos := map[string]bool{}
	for i, x := range []vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{alta.Exportacion, pre.Exportacion, cta.Exportacion, commit.Exportacion} {
		r := x.ResumenCapacidad()
		if vistos[r.DecisionRef()] {
			t.Fatal("permiso reutilizado")
		}
		vistos[r.DecisionRef()] = true
		esperado := e.a.contexto.Resultado.RegistroContextoRef
		if i == 0 {
			esperado = eAlta.a.contexto.Resultado.RegistroContextoRef
		}
		if r.ContextoRef() != esperado {
			t.Fatal("contexto")
		}
	}
	if e.store.registros != 3 || eAlta.store.registros != 1 {
		t.Fatal("concesiones")
	}
	r, _ := pa.RecursoAltaEjercicio(m)
	k, _ := autoridadContratoPara(autoridadAlta)
	autoridadExigir(t, autoridadCotejarExportacion(ct.AutorizacionConfirmacionIncorporacionV2(alta), eAlta.a.contexto, r, k, e.ahora))
	c := alta.Exportacion.DecisionCanonica()
	c[0] ^= 1
	if alta.Exportacion.ValidarEstructura() != nil {
		t.Fatal("alias")
	}
}
func TestAutoridadAplicacionDenegaciones(t *testing.T) {
	for _, caso := range []string{"rol_sin_permiso", "motivo_ajeno", "audiencia_cruzada", "registro_falla", "firma_falla", "cancel_registro", "cancel_firma", "contexto_mutado"} {
		t.Run(caso, func(t *testing.T) {
			e := autoridadEntorno(t, autoridadAlta)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			m := autoridadMaterialAlta(t, e)
			switch caso {
			case "rol_sin_permiso":
				e.store.snapshot.VersionRol.Concesiones = e.store.snapshot.VersionRol.Concesiones[1:]
			case "motivo_ajeno":
				e.a.peticion.MotivoAlta.EntradaClave = "motivo_22222222222222222222222222222222"
			case "audiencia_cruzada":
				e.a.cadena.emisiones.Alta = e.a.cadena.emisiones.Lectura
			case "registro_falla":
				e.store.fallo = errors.New("dato privado no visible")
			case "firma_falla":
				e.firmante.fallo = errors.New("dato privado no visible")
			case "cancel_registro":
				e.store.despues = cancel
			case "cancel_firma":
				e.firmante.despues = cancel
			case "contexto_mutado":
				e.a.contexto.Resultado.RepresentacionCanonica[0] ^= 1
			}
			x, err := e.a.AutorizarAlta(ctx, m)
			if err == nil || !reflect.DeepEqual(x, pa.AutorizacionAlta{}) || strings.Contains(err.Error(), "privado") {
				t.Fatal("no falla cerrado")
			}
			if strings.HasPrefix(caso, "cancel_") && !errors.Is(err, context.Canceled) {
				t.Fatal("cancel perdida")
			}
		})
	}
}
func TestAutoridadAplicacionConcurrente(t *testing.T) {
	e := autoridadEntorno(t)
	m, err := pl.NuevoMaterialV2(autoridadSelector(e), e.fuente.p.PreparacionCT.UnidadRef, e.a.contexto, e.ahora)
	autoridadExigir(t, err)
	ch := make(chan string, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a, err := e.a.AutorizarLecturaIncorporacionV2(context.Background(), m)
			if err != nil {
				ch <- ""
				return
			}
			ch <- a.Exportacion.ResumenCapacidad().DecisionRef()
		}()
	}
	wg.Wait()
	close(ch)
	ids := map[string]bool{}
	for id := range ch {
		if id == "" || ids[id] {
			t.Fatal("concurrencia")
		}
		ids[id] = true
	}
	if e.store.registros != 8 {
		t.Fatal("registros")
	}
}

// El reloj de aplicación se cancela/retrocede exactamente en la última lectura
// observada de una emisión nominal; no basta cancelar antes de las dependencias.
type autoridadRelojSecuencia struct {
	ahora        time.Time
	llamadas, en int
	cancel       context.CancelFunc
	retroceso    bool
}

func (r *autoridadRelojSecuencia) Ahora() time.Time {
	r.llamadas++
	if r.en == r.llamadas {
		if r.cancel != nil {
			r.cancel()
		}
		if r.retroceso {
			return r.ahora.Add(-time.Microsecond)
		}
	}
	return r.ahora
}
func TestAutoridadAplicacionUltimaFrontera(t *testing.T) {
	base := autoridadEntorno(t)
	d := autoridadMaterialCT(t, base)
	m, e := ct.NuevoMaterialConfirmacionIncorporacionV2(d, base.ahora)
	autoridadExigir(t, e)
	reloj := &autoridadRelojSecuencia{ahora: base.ahora}
	base.a.reloj = reloj
	_, e = base.a.AutorizarConfirmacionIncorporacion(context.Background(), m)
	autoridadExigir(t, e)
	ultimo := reloj.llamadas
	for _, caso := range []string{"cancelacion", "retroceso"} {
		t.Run(caso, func(t *testing.T) {
			x := autoridadEntorno(t)
			d := autoridadMaterialCT(t, x)
			m, e := ct.NuevoMaterialConfirmacionIncorporacionV2(d, x.ahora)
			autoridadExigir(t, e)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r := &autoridadRelojSecuencia{ahora: x.ahora, en: ultimo, retroceso: caso == "retroceso"}
			if caso == "cancelacion" {
				r.cancel = cancel
			}
			x.a.reloj = r
			a, e := x.a.AutorizarConfirmacionIncorporacion(ctx, m)
			if e == nil || !reflect.DeepEqual(a, ct.AutorizacionConfirmacionIncorporacionV2{}) || r.llamadas != ultimo {
				t.Fatal("ultima frontera")
			}
			if caso == "cancelacion" && !errors.Is(e, context.Canceled) {
				t.Fatal("cancel")
			}
		})
	}
}
func TestAutoridadAplicacionCotejoExportacion(t *testing.T) {
	e := autoridadEntorno(t)
	d := autoridadMaterialCT(t, e)
	m, err := ct.NuevoMaterialConfirmacionIncorporacionV2(d, e.ahora)
	autoridadExigir(t, err)
	a, err := e.a.AutorizarConfirmacionIncorporacion(context.Background(), m)
	autoridadExigir(t, err)
	r, err := ct.RecursoConfirmacionIncorporacionV2(m)
	autoridadExigir(t, err)
	k, _ := autoridadContratoPara(autoridadCT)
	for _, caso := range []string{"ref", "modulo", "tipo", "ambito", "atributo", "audiencia", "accion", "finalidad", "decision", "confirmacion", "contexto", "tiempo", "copias"} {
		t.Run(caso, func(t *testing.T) {
			rr, err := ct.RecursoConfirmacionIncorporacionV2(m)
			autoridadExigir(t, err)
			kk := k
			cc, _ := e.a.ContextoAutoridad()
			aa := a
			ahora := e.ahora
			switch caso {
			case "ref":
				rr.Referencia += "x"
			case "modulo":
				rr.ModuloID = "personal"
			case "tipo":
				rr.Tipo = pl.TipoRecursoV2
			case "ambito":
				rr.Ambitos["organizacion_ref"] += "x"
			case "atributo":
				rr.Atributos["adicional"] = "x"
			case "audiencia":
				kk.audiencia = pl.AudienciaV2
			case "accion":
				kk.accion = pl.Accion
			case "finalidad":
				kk.finalidad = pl.Finalidad
			case "decision":
				aa.Decision = core.DecisionAutorizacionLigadaV3{}
			case "confirmacion":
				aa.Confirmacion = vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
			case "contexto":
				cc.Resultado.RegistroContextoRef += "x"
			case "tiempo":
				ahora = ahora.Add(time.Hour)
			case "copias":
				bytes := aa.Exportacion.ContextoActorCanonico()
				bytes[0] ^= 1
				autoridadExigir(t, autoridadCotejarExportacion(aa, cc, r, kk, ahora))
				return
			}
			if autoridadCotejarExportacion(aa, cc, rr, kk, ahora) == nil {
				t.Fatal("cruce admitido")
			}
		})
	}
}

func TestAutoridadAplicacionAmbitosNoIntercambiables(t *testing.T) {
	alta := autoridadEntorno(t, autoridadAlta)
	lectura := autoridadEntorno(t)
	c, _ := alta.a.ContextoAutoridad()
	m, e := pl.NuevoMaterialV2(autoridadSelector(alta), alta.fuente.p.PreparacionCT.UnidadRef, c, alta.ahora)
	autoridadExigir(t, e)
	x, e := alta.a.AutorizarLecturaIncorporacionV2(context.Background(), m)
	if e == nil || !reflect.DeepEqual(x, pl.AutorizacionV2{}) || alta.store.registros != 0 {
		t.Fatal("alta no concede lectura")
	}
	a, e := lectura.a.AutorizarAlta(context.Background(), autoridadMaterialAlta(t, lectura))
	if e == nil || !reflect.DeepEqual(a, pa.AutorizacionAlta{}) || lectura.store.registros != 0 {
		t.Fatal("lectura no concede alta")
	}
}

// El reloj no puede retroceder entre la validación pública del material y la
// primera dependencia de la cadena, aunque siga posterior a creadaEn.
func TestAutoridadAplicacionEntradaMonotona(t *testing.T) {
	e := autoridadEntorno(t, autoridadAlta)
	m := autoridadMaterialAlta(t, e)
	n := 0
	e.a.reloj = autoridadRelojFuncion(func() time.Time {
		n++
		if n == 1 {
			return e.ahora.Add(time.Second)
		}
		return e.ahora
	})
	a, err := e.a.AutorizarAlta(context.Background(), m)
	if err == nil || !reflect.DeepEqual(a, pa.AutorizacionAlta{}) || e.store.registros != 0 {
		t.Fatal("retroceso entre fronteras")
	}
}

type autoridadRelojFuncion func() time.Time

func (f autoridadRelojFuncion) Ahora() time.Time { return f() }
