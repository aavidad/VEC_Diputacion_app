package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type llamadasPersonaB2Prueba struct{ actor, consultas int }

func escenarioPersonaB2Capturada(t *testing.T) (*FuentesContextoIncorporacionB2, *fuentesContextoB2Prueba, inc.ContratoPlanNominal, *autoridadIncorporacionPersonalB2, context.Context, *llamadasPersonaB2Prueba) {
	t.Helper()
	f, d, contrato := escenarioFuentesContextoB2(t)
	a, ctx, _ := escenarioCapturaAsignacionB2Prueba(t)
	llamadas := &llamadasPersonaB2Prueba{}
	f.c.ActorBolsa = func(ctx context.Context) (bp.ActorConfiablePersonaAceptacionCT, error) {
		llamadas.actor++
		actual, err := a.contexto(ctx, bp.AccionConsultaPersonaAceptacionCT)
		return bp.ActorConfiablePersonaAceptacionCT{Vinculo: actual.Vinculo, Resultado: actual.Resultado}, err
	}
	base := d.bolsa
	d.bolsa = func(ctx context.Context, q bp.SolicitudConsultaPersonaAceptacionCT) (bp.ResultadoConsultaPersonaAceptacionCT, error) {
		llamadas.consultas++
		r, err := base(ctx, q)
		if r.Vinculo != nil {
			r.Vinculo.Ref, r.Vinculo.Version, r.Vinculo.Poblacion = "vinculo:persona", 1, "externa"
			r.Vinculo.VigenteHasta = a.reloj.Ahora().Add(time.Hour)
		}
		return r, err
	}
	return f, d, contrato, a, ctx, llamadas
}

func TestFuentesContextoB2PersonaUnaConsultaPorPeticionYActorPorUso(t *testing.T) {
	f, _, c, _, ctx, llamadas := escenarioPersonaB2Capturada(t)
	inicial := &fuentesIncorporacionPersonalB2{externas: f}
	if _, err := inicial.resolverPersonaInicialB2(ctx, c); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := f.LeerPersonaSeleccionada(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	if llamadas.consultas != 1 || llamadas.actor != 3 {
		t.Fatalf("lecturas/actores de una petición: %+v", llamadas)
	}
	ctx = context.WithValue(ctx, claveContextosNominalesIncorporacion{}, &capturaContextosNominalesIncorporacion{})
	if _, err := f.LeerPersonaSeleccionada(ctx, c); err != nil || llamadas.consultas != 2 || llamadas.actor != 4 {
		t.Fatalf("otra petición reutilizó persona/actor: %+v err=%v", llamadas, err)
	}
}

func TestFuentesContextoB2CapturaPersonaPorSelectorCompletoYFuente(t *testing.T) {
	for nombre, cambiar := range map[string]func(*dom.SelectorBolsaPlanB2){
		"unidad":            func(x *dom.SelectorBolsaPlanB2) { x.UnidadRef += ":otra" },
		"categoria":         func(x *dom.SelectorBolsaPlanB2) { x.CategoriaRef += ":otra" },
		"necesidad":         func(x *dom.SelectorBolsaPlanB2) { x.NecesidadRef += ":otra" },
		"aceptacion":        func(x *dom.SelectorBolsaPlanB2) { x.AceptacionOperacionRef += ":otra" },
		"huella_aceptacion": func(x *dom.SelectorBolsaPlanB2) { x.AceptacionRegistroSHA256 = strings.Repeat("a", 64) },
		"apertura":          func(x *dom.SelectorBolsaPlanB2) { x.AperturaOperacionRef += ":otra" },
		"huella_apertura":   func(x *dom.SelectorBolsaPlanB2) { x.AperturaRegistroSHA256 = strings.Repeat("a", 64) },
		"llamamiento":       func(x *dom.SelectorBolsaPlanB2) { x.LlamamientoRef += ":otro" },
		"propuesta":         func(x *dom.SelectorBolsaPlanB2) { x.PropuestaRef += ":otra" },
	} {
		t.Run(nombre, func(t *testing.T) {
			f, _, c, _, ctx, llamadas := escenarioPersonaB2Capturada(t)
			x := c.SelectorBolsa
			if _, err := f.consultarPersonaBolsaB2(ctx, x); err != nil {
				t.Fatal(err)
			}
			cambiar(&x)
			if _, err := f.consultarPersonaBolsaB2(ctx, x); err != nil {
				t.Fatal(err)
			}
			if _, err := f.consultarPersonaBolsaB2(ctx, c.SelectorBolsa); err != nil || llamadas.consultas != 2 {
				t.Fatalf("selector cambió o se perdió captura: consultas=%d err=%v", llamadas.consultas, err)
			}
			otra := *f
			if _, err := otra.consultarPersonaBolsaB2(ctx, c.SelectorBolsa); err != nil || llamadas.consultas != 3 {
				t.Fatalf("otra fuente reutilizó lectura: consultas=%d err=%v", llamadas.consultas, err)
			}
		})
	}
}

func TestFuentesContextoB2CapturaPersonaIndependienteDeDTO(t *testing.T) {
	f, _, c, _, ctx, llamadas := escenarioPersonaB2Capturada(t)
	r, err := f.consultarPersonaBolsaB2(ctx, c.SelectorBolsa)
	if err != nil {
		t.Fatal(err)
	}
	persona, aceptacion, vinculo, evidencia := *r.Persona, *r.Aceptacion, *r.Vinculo, r.Evidencia
	for range 2 {
		r.Persona.Ref, r.Aceptacion.ReciboRef, r.Vinculo.ProcedenciaRef = "persona:ajena", "recibo:ajeno", "fuente:ajena"
		r.Evidencia.AuditoriaRef = "auditoria:ajena"
		r, err = f.consultarPersonaBolsaB2(ctx, c.SelectorBolsa)
		if err != nil || *r.Persona != persona || *r.Aceptacion != aceptacion || *r.Vinculo != vinculo || r.Evidencia != evidencia {
			t.Fatalf("mutación del DTO alteró captura: %+v err=%v", r, err)
		}
	}
	if llamadas.consultas != 1 {
		t.Fatalf("se releyó para evitar alias: %d", llamadas.consultas)
	}
}

func TestFuentesContextoB2PersonaVencidaSeRelee(t *testing.T) {
	f, d, c, a, ctx, llamadas := escenarioPersonaB2Capturada(t)
	base := d.bolsa
	var vence time.Time
	d.bolsa = func(ctx context.Context, q bp.SolicitudConsultaPersonaAceptacionCT) (bp.ResultadoConsultaPersonaAceptacionCT, error) {
		r, err := base(ctx, q)
		if llamadas.consultas == 1 {
			vence = a.reloj.Ahora().Add(50 * time.Millisecond)
			r.Vinculo.VigenteHasta = vence
		}
		return r, err
	}
	if _, err := f.LeerPersonaSeleccionada(ctx, c); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Until(vence) + time.Millisecond)
	if _, err := f.LeerPersonaSeleccionada(ctx, c); err != nil || llamadas.consultas != 2 {
		t.Fatalf("vínculo vencido reutilizado: consultas=%d err=%v", llamadas.consultas, err)
	}
}

func TestFuentesContextoB2PersonaNoCapturaErroresNiPendientes(t *testing.T) {
	for _, caso := range []string{"denegada", "caida", "pendiente", "no_encontrada", "incompleta", "selector_ajeno"} {
		t.Run(caso, func(t *testing.T) {
			f, d, c, _, ctx, llamadas := escenarioPersonaB2Capturada(t)
			base := d.bolsa
			d.bolsa = func(ctx context.Context, q bp.SolicitudConsultaPersonaAceptacionCT) (bp.ResultadoConsultaPersonaAceptacionCT, error) {
				r, err := base(ctx, q)
				if llamadas.consultas != 1 {
					return r, err
				}
				switch caso {
				case "denegada":
					return bp.ResultadoConsultaPersonaAceptacionCT{}, bp.ErrConsultaPersonaAceptacionCTDenegada
				case "caida":
					return bp.ResultadoConsultaPersonaAceptacionCT{}, bp.ErrConsultaPersonaAceptacionCTNoDisponible
				case "pendiente", "no_encontrada":
					r = bp.ResultadoConsultaPersonaAceptacionCT{Estado: caso}
				case "incompleta":
					r.Persona = nil
				case "selector_ajeno":
					r.Aceptacion.OperacionRef = "operacion:ajena"
				}
				return r, err
			}
			_, err := f.consultarPersonaBolsaB2(ctx, c.SelectorBolsa)
			if caso == "denegada" && !errors.Is(err, bp.ErrConsultaPersonaAceptacionCTDenegada) || caso == "caida" && !errors.Is(err, bp.ErrConsultaPersonaAceptacionCTNoDisponible) {
				t.Fatalf("error de fuente ocultado: %v", err)
			}
			for range 2 {
				if _, err := f.consultarPersonaBolsaB2(ctx, c.SelectorBolsa); err != nil {
					t.Fatal(err)
				}
			}
			if llamadas.consultas != 2 {
				t.Fatalf("se capturó negativa o no se conservó positivo: %d", llamadas.consultas)
			}
		})
	}
}

func TestFuentesContextoB2PersonaRevalidaActorYCancelaAntesDeConsulta(t *testing.T) {
	f, _, c, _, ctx, llamadas := escenarioPersonaB2Capturada(t)
	if _, err := f.LeerPersonaSeleccionada(ctx, c); err != nil {
		t.Fatal(err)
	}
	f.c.ActorBolsa = func(context.Context) (bp.ActorConfiablePersonaAceptacionCT, error) {
		return bp.ActorConfiablePersonaAceptacionCT{}, ct.ErrAutorizacionDenegada
	}
	if _, err := f.LeerPersonaSeleccionada(ctx, c); !errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) || llamadas.consultas != 1 {
		t.Fatalf("actor denegado recuperó persona capturada: consultas=%d err=%v", llamadas.consultas, err)
	}
	f, _, c, _, ctx, llamadas = escenarioPersonaB2Capturada(t)
	ctx, cancelar := context.WithCancel(ctx)
	baseActor := f.c.ActorBolsa
	f.c.ActorBolsa = func(ctx context.Context) (bp.ActorConfiablePersonaAceptacionCT, error) {
		actor, err := baseActor(ctx)
		cancelar()
		return actor, err
	}
	if _, err := f.LeerPersonaSeleccionada(ctx, c); !errors.Is(err, context.Canceled) || llamadas.consultas != 0 {
		t.Fatalf("cancelación posterior al actor consultó Bolsa: consultas=%d err=%v", llamadas.consultas, err)
	}
}

func TestFuentesContextoB2PersonaRevocadaEnOtraPeticionSeDeniega(t *testing.T) {
	f, _, c, a, ctx, llamadas := escenarioPersonaB2Capturada(t)
	if _, err := f.LeerPersonaSeleccionada(ctx, c); err != nil {
		t.Fatal(err)
	}
	a.perfiles.soporte.mu.Lock()
	lector := a.perfiles.soporte.autoridadAsignaciones.(*lectorAsignacionB2ContadoPrueba)
	a.perfiles.soporte.mu.Unlock()
	perfil := a.perfiles.b2[bp.AccionConsultaPersonaAceptacionCT]
	publicada := lector.base.publicadas[perfil.perfilRef()]
	publicada.instantanea.AsignacionPerfil.Estado = core.EstadoAsignacionPerfilRevocada
	lector.base.publicadas[perfil.perfilRef()] = publicada
	ctx = context.WithValue(ctx, claveContextosNominalesIncorporacion{}, &capturaContextosNominalesIncorporacion{})
	if _, err := f.LeerPersonaSeleccionada(ctx, c); !errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) || llamadas.consultas != 1 {
		t.Fatalf("otra petición reutilizó dato con perfil revocado: consultas=%d err=%v", llamadas.consultas, err)
	}
}

func TestFuentesContextoB2PersonaConcurrentePuedeAutorizarSinBloqueoNominal(t *testing.T) {
	f, d, c, a, ctx, llamadas := escenarioPersonaB2Capturada(t)
	ctx, cancelar := context.WithTimeout(ctx, 2*time.Second)
	defer cancelar()
	base := d.bolsa
	d.bolsa = func(ctx context.Context, q bp.SolicitudConsultaPersonaAceptacionCT) (bp.ResultadoConsultaPersonaAceptacionCT, error) {
		// El proveedor real solicita V3 y vuelve a resolver este perfil.
		if _, err := a.contexto(ctx, bp.AccionConsultaPersonaAceptacionCT); err != nil {
			return bp.ResultadoConsultaPersonaAceptacionCT{}, err
		}
		return base(ctx, q)
	}
	var grupo sync.WaitGroup
	errores := make(chan error, 8)
	for range 8 {
		grupo.Go(func() { _, err := f.LeerPersonaSeleccionada(ctx, c); errores <- err })
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatal(err)
		}
	}
	if llamadas.consultas != 1 || llamadas.actor != 8 {
		t.Fatalf("captura concurrente/actor: %+v", llamadas)
	}
}

type contextoEsperaPersonaB2Prueba struct {
	context.Context
	entrada chan struct{}
	una     sync.Once
}

func (c *contextoEsperaPersonaB2Prueba) Value(clave any) any {
	if _, ok := clave.(claveContextosNominalesIncorporacion); ok {
		c.una.Do(func() { close(c.entrada) })
	}
	return c.Context.Value(clave)
}

func TestFuentesContextoB2PersonaCanceladaMientrasEsperaNoConsultaNiRecuperaCache(t *testing.T) {
	f, _, c, _, ctx, llamadas := escenarioPersonaB2Capturada(t)
	if _, err := f.LeerPersonaSeleccionada(ctx, c); err != nil {
		t.Fatal(err)
	}
	holder := ctx.Value(claveContextosNominalesIncorporacion{}).(*capturaContextosNominalesIncorporacion)
	holder.muPersonasB2 <- struct{}{}
	defer func() { <-holder.muPersonasB2 }()
	cancelable, cancelar := context.WithCancel(ctx)
	defer cancelar()
	espera := &contextoEsperaPersonaB2Prueba{Context: cancelable, entrada: make(chan struct{})}
	termino := make(chan error, 1)
	go func() { _, err := f.LeerPersonaSeleccionada(espera, c); termino <- err }()
	<-espera.entrada
	cancelar()
	select {
	case err := <-termino:
		if !errors.Is(err, context.Canceled) || llamadas.consultas != 1 || llamadas.actor != 1 {
			t.Fatalf("cancelada consultó fuente/actor o recibió caché: %+v err=%v", llamadas, err)
		}
	case <-time.After(time.Second):
		t.Fatal("la cancelación esperó al lector que ocupaba el turno")
	}
}

func TestFuentesContextoB2PersonaVencidaMientrasEsperaSeRelee(t *testing.T) {
	f, d, c, a, ctx, llamadas := escenarioPersonaB2Capturada(t)
	base := d.bolsa
	var vence time.Time
	d.bolsa = func(ctx context.Context, q bp.SolicitudConsultaPersonaAceptacionCT) (bp.ResultadoConsultaPersonaAceptacionCT, error) {
		r, err := base(ctx, q)
		if llamadas.consultas == 1 {
			vence = a.reloj.Ahora().Add(50 * time.Millisecond)
			r.Vinculo.VigenteHasta = vence
		}
		return r, err
	}
	if _, err := f.LeerPersonaSeleccionada(ctx, c); err != nil {
		t.Fatal(err)
	}
	holder := ctx.Value(claveContextosNominalesIncorporacion{}).(*capturaContextosNominalesIncorporacion)
	holder.muPersonasB2 <- struct{}{}
	espera := &contextoEsperaPersonaB2Prueba{Context: ctx, entrada: make(chan struct{})}
	termino := make(chan error, 1)
	go func() { _, err := f.LeerPersonaSeleccionada(espera, c); termino <- err }()
	<-espera.entrada
	time.Sleep(time.Until(vence) + time.Millisecond)
	<-holder.muPersonasB2
	select {
	case err := <-termino:
		if err != nil || llamadas.consultas != 2 || llamadas.actor != 2 {
			t.Fatalf("vínculo vencido durante espera no se releyó: %+v err=%v", llamadas, err)
		}
	case <-time.After(time.Second):
		t.Fatal("la lectura no terminó tras liberar el turno")
	}
}

type fuentesContextoB2Prueba struct {
	bolsa       func(context.Context, bp.SolicitudConsultaPersonaAceptacionCT) (bp.ResultadoConsultaPersonaAceptacionCT, error)
	vinculo     func(context.Context, ct.ConsultaVinculoCategoriaRPT) (ct.LecturaVinculoCategoriaRPT, error)
	autorizar   func(context.Context, dom.PublicacionCategoriaRPT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	publicacion func(context.Context, dom.PublicacionCategoriaRPT, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (dom.PublicacionCategoriaRPT, error)
}

func (d *fuentesContextoB2Prueba) ConsultarPersonaAceptacionCT(c context.Context, q bp.SolicitudConsultaPersonaAceptacionCT) (bp.ResultadoConsultaPersonaAceptacionCT, error) {
	return d.bolsa(c, q)
}
func (d *fuentesContextoB2Prueba) Consultar(c context.Context, q ct.ConsultaVinculoCategoriaRPT) (ct.LecturaVinculoCategoriaRPT, error) {
	return d.vinculo(c, q)
}
func (d *fuentesContextoB2Prueba) LecturaRPT(c context.Context, p dom.PublicacionCategoriaRPT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return d.autorizar(c, p)
}
func (d *fuentesContextoB2Prueba) ConsultarPublicacionCategoriaRPT(c context.Context, p dom.PublicacionCategoriaRPT, a vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (dom.PublicacionCategoriaRPT, error) {
	return d.publicacion(c, p, a)
}

func escenarioFuentesContextoB2(t *testing.T) (*FuentesContextoIncorporacionB2, *fuentesContextoB2Prueba, inc.ContratoPlanNominal) {
	t.Helper()
	c := inc.ContratoPlanNominal{
		Protocolo: inc.ProtocoloPersonalB2V1, OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:prueba",
		VersionExpediente: 7, CategoriaRef: "categoria_prueba", AceptacionRef: "aceptacion:ct", LlamamientoRef: "llamamiento:prueba",
		SeleccionReciboRef: "recibo:aceptacion_bolsa", VinculoRevision: 1, VinculoReciboRef: "recibo:vinculo_ct",
		PuestoRef: "puesto:seleccion_rrhh", PlazaRef: "plaza:seleccion_rrhh",
		FuenteRPT: ct.ReferenciaVersionadaPersonalRPT{Referencia: "fuente:rpt_estructural", Version: 3, HuellaSHA256: strings.Repeat("a", 64)},
		DatosPersonal: inc.DatosActosPersonalB2{CatalogoRPTID: "categorias_rpt", ModuloRPTID: "vec.module.puesto_trabajo",
			CatalogoRPTVersion: 2, CatalogoRPTHuellaSHA256: strings.Repeat("b", 64), CategoriaID: "categoria_prueba"},
		SelectorBolsa: dom.SelectorBolsaPlanB2{UnidadRef: "unidad:prueba", CategoriaRef: "categoria_prueba", NecesidadRef: "necesidad:prueba",
			AceptacionOperacionRef: "operacion:aceptacion", AceptacionRegistroSHA256: strings.Repeat("c", 64),
			AperturaOperacionRef: "operacion:apertura", AperturaRegistroSHA256: strings.Repeat("d", 64),
			LlamamientoRef: "llamamiento:prueba", PropuestaRef: "propuesta:bolsa"},
	}
	d := &fuentesContextoB2Prueba{}
	d.bolsa = func(_ context.Context, q bp.SolicitudConsultaPersonaAceptacionCT) (bp.ResultadoConsultaPersonaAceptacionCT, error) {
		x := q.Selector
		return bp.ResultadoConsultaPersonaAceptacionCT{Estado: "acreditado",
			Aceptacion: &bp.AceptacionPersonaCT{OperacionRef: x.AceptacionOperacionRef, ReciboRef: c.SeleccionReciboRef,
				RegistroSHA256: x.AceptacionRegistroSHA256, AperturaOperacionRef: x.AperturaOperacionRef,
				AperturaRegistroSHA256: x.AperturaRegistroSHA256, LlamamientoRef: x.LlamamientoRef},
			Persona: &bp.PersonaAceptadaCT{Ref: "per_persona_seleccionada_abcdefghijkl", Version: 4},
			Vinculo: &bp.VinculoPersonaAceptadaCT{ProcedenciaRef: "fuente:persona", ProcedenciaVersion: 5, ProcedenciaSHA256: strings.Repeat("e", 64)},
		}, nil
	}
	d.vinculo = func(_ context.Context, q ct.ConsultaVinculoCategoriaRPT) (ct.LecturaVinculoCategoriaRPT, error) {
		return ct.LecturaVinculoCategoriaRPT{OrganizacionRef: q.OrganizacionRef, ExpedienteRef: q.ExpedienteRef,
			Analisis: ct.AnclajeAnalisisCategoriaRPT{VersionExpediente: c.VersionExpediente, CategoriaRef: c.CategoriaRef},
			Vinculo: &ct.EstadoVinculoCategoriaRPT{Revision: c.VinculoRevision, ReciboRef: c.VinculoReciboRef,
				CatalogoID: c.DatosPersonal.CatalogoRPTID, ModuloID: c.DatosPersonal.ModuloRPTID,
				CatalogoVersion: c.DatosPersonal.CatalogoRPTVersion, CatalogoHuellaSHA256: c.DatosPersonal.CatalogoRPTHuellaSHA256,
				CategoriaID: c.DatosPersonal.CategoriaID, Prospectivo: true}}, nil
	}
	d.autorizar = func(context.Context, dom.PublicacionCategoriaRPT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
	}
	d.publicacion = func(_ context.Context, p dom.PublicacionCategoriaRPT, _ vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (dom.PublicacionCategoriaRPT, error) {
		return p, nil
	}
	f, err := NuevasFuentesContextoIncorporacionB2(ConfiguracionFuentesContextoIncorporacionB2{
		Bolsa: d, Vinculos: d, Publicacion: d, AutoridadRPT: d,
		ActorBolsa: func(context.Context) (bp.ActorConfiablePersonaAceptacionCT, error) {
			return bp.ActorConfiablePersonaAceptacionCT{}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return f, d, c
}

func TestFuentesContextoB2BolsaTraduceSelectorYRevalidaActor(t *testing.T) {
	f, d, c := escenarioFuentesContextoB2(t)
	base := d.bolsa
	llamadas := 0
	f.c.ActorBolsa = func(context.Context) (bp.ActorConfiablePersonaAceptacionCT, error) {
		llamadas++
		return bp.ActorConfiablePersonaAceptacionCT{Resultado: core.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "contexto:actor_actual"}}, nil
	}
	d.bolsa = func(ctx context.Context, q bp.SolicitudConsultaPersonaAceptacionCT) (bp.ResultadoConsultaPersonaAceptacionCT, error) {
		if q.ActorConfiable.Resultado.RegistroContextoRef != "contexto:actor_actual" {
			t.Fatal("actor no procede de la petición")
		}
		x := c.SelectorBolsa
		esperado := bp.SelectorPersonaAceptacionCT{UnidadRef: x.UnidadRef, CategoriaRef: x.CategoriaRef, NecesidadRef: x.NecesidadRef,
			AceptacionOperacionRef: x.AceptacionOperacionRef, AceptacionRegistroSHA256: x.AceptacionRegistroSHA256,
			AperturaOperacionRef: x.AperturaOperacionRef, AperturaRegistroSHA256: x.AperturaRegistroSHA256,
			LlamamientoRef: x.LlamamientoRef, PropuestaRef: x.PropuestaRef}
		if q.Selector != esperado {
			t.Fatal("selector modificado")
		}
		return base(ctx, q)
	}
	for range 2 {
		p, err := f.LeerPersonaSeleccionada(context.Background(), c)
		if err != nil || p.PersonaRef != "per_persona_seleccionada_abcdefghijkl" || p.PersonaVersion != 4 ||
			p.ReciboRef != c.SeleccionReciboRef || p.FuenteRef != "fuente:persona" || p.FuenteVersion != 5 ||
			p.FuenteSHA256 != strings.Repeat("e", 64) || p.SeleccionRef != "" || p.VersionSeleccion != 0 {
			t.Fatalf("persona o procedencia no conservadas: %+v %v", p, err)
		}
	}
	if llamadas != 2 {
		t.Fatal("actor almacenado entre peticiones")
	}
}

func TestFuentesContextoB2RPTConservaPublicacionYPuestoElegido(t *testing.T) {
	f, d, c := escenarioFuentesContextoB2(t)
	secuencia := []string{}
	autorizar, publicar := d.autorizar, d.publicacion
	d.autorizar = func(ctx context.Context, p dom.PublicacionCategoriaRPT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		secuencia = append(secuencia, "autorizar")
		return autorizar(ctx, p)
	}
	d.publicacion = func(ctx context.Context, p dom.PublicacionCategoriaRPT, a vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (dom.PublicacionCategoriaRPT, error) {
		secuencia = append(secuencia, "consultar")
		return publicar(ctx, p, a)
	}
	p, err := f.LeerPuestoRPT(context.Background(), c)
	if err != nil || p.Fuente != c.FuenteRPT || p.PuestoRef != c.PuestoRef || p.PlazaRef != c.PlazaRef ||
		p.VinculoReciboRef != c.VinculoReciboRef || !p.Prospectivo || p.AcreditaProcedenciaHistorica ||
		!reflect.DeepEqual(secuencia, []string{"autorizar", "consultar"}) {
		t.Fatalf("RPT: %+v %v %v", p, err, secuencia)
	}
}

func TestFuentesContextoB2DetieneMezclasAntesDeConsultarPublicacion(t *testing.T) {
	for _, caso := range []string{"revision", "recibo", "categoria", "version", "huella", "historico", "expediente"} {
		t.Run(caso, func(t *testing.T) {
			f, d, c := escenarioFuentesContextoB2(t)
			base := d.vinculo
			d.vinculo = func(ctx context.Context, q ct.ConsultaVinculoCategoriaRPT) (ct.LecturaVinculoCategoriaRPT, error) {
				l, _ := base(ctx, q)
				switch caso {
				case "revision":
					l.Vinculo.Revision++
				case "recibo":
					l.Vinculo.ReciboRef = "recibo:otro"
				case "categoria":
					l.Vinculo.CategoriaID = "categoria_otra"
				case "version":
					l.Vinculo.CatalogoVersion++
				case "huella":
					l.Vinculo.CatalogoHuellaSHA256 = strings.Repeat("f", 64)
				case "historico":
					l.Vinculo.AcreditaProcedenciaHistorica = true
				case "expediente":
					l.ExpedienteRef = "expediente:otro"
				}
				return l, nil
			}
			d.autorizar = func(context.Context, dom.PublicacionCategoriaRPT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
				t.Fatal("autorizó publicación después de detectar otra fuente")
				return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
			}
			if p, err := f.LeerPuestoRPT(context.Background(), c); !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || p != (inc.PuestoRPTNominal{}) {
				t.Fatalf("mezcla admitida: %+v %v", p, err)
			}
		})
	}
}

func TestFuentesContextoB2EstadosPendientesYRespuestaDivergente(t *testing.T) {
	f, d, c := escenarioFuentesContextoB2(t)
	base := d.bolsa
	for _, estado := range []string{"pendiente", "no_encontrada"} {
		d.bolsa = func(context.Context, bp.SolicitudConsultaPersonaAceptacionCT) (bp.ResultadoConsultaPersonaAceptacionCT, error) {
			return bp.ResultadoConsultaPersonaAceptacionCT{Estado: estado}, nil
		}
		if _, err := f.LeerPersonaSeleccionada(context.Background(), c); !errors.Is(err, ct.ErrPreparacionIncorporacionPendiente) {
			t.Fatal(err)
		}
	}
	d.bolsa = func(ctx context.Context, q bp.SolicitudConsultaPersonaAceptacionCT) (bp.ResultadoConsultaPersonaAceptacionCT, error) {
		r, _ := base(ctx, q)
		r.Aceptacion.ReciboRef = "recibo:otra_aceptacion"
		return r, nil
	}
	if _, err := f.LeerPersonaSeleccionada(context.Background(), c); !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) {
		t.Fatal(err)
	}
	d.publicacion = func(_ context.Context, p dom.PublicacionCategoriaRPT, _ vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (dom.PublicacionCategoriaRPT, error) {
		p.CatalogoVersion++
		return p, nil
	}
	if _, err := f.LeerPuestoRPT(context.Background(), c); !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) {
		t.Fatal(err)
	}
}

func TestFuentesContextoB2ErroresNoConfundenDenegacionYCaida(t *testing.T) {
	for _, caso := range []struct{ entrada, esperado error }{
		{bp.ErrConsultaPersonaAceptacionCTDenegada, ct.ErrDenegadaIncorporacionAplicacion},
		{ct.ErrVinculoCategoriaRPTDenegado, ct.ErrDenegadaIncorporacionAplicacion},
		{vp.ErrLecturaRPTDenegada, ct.ErrDenegadaIncorporacionAplicacion},
		{errors.Join(core.ErrAutorizacionDenegada, vp.ErrRegistroDecisionNoDisponible), ct.ErrComposicionIncorporacionAplicacion},
		{errors.New("detalle privado de dependencia"), ct.ErrComposicionIncorporacionAplicacion},
		{ct.ErrVinculoCategoriaRPTNoEncontrado, ct.ErrPreparacionIncorporacionPendiente},
		{context.Canceled, context.Canceled}, {context.DeadlineExceeded, context.DeadlineExceeded},
	} {
		if e := errorFuenteContextoIncorporacionB2(context.Background(), caso.entrada); e != caso.esperado {
			t.Fatalf("%v: %v", caso.entrada, e)
		}
	}
}

func TestFuentesContextoB2CancelacionDetieneLecturas(t *testing.T) {
	f, d, c := escenarioFuentesContextoB2(t)
	ctx, cancelar := context.WithCancel(context.Background())
	f.c.ActorBolsa = func(context.Context) (bp.ActorConfiablePersonaAceptacionCT, error) {
		cancelar()
		return bp.ActorConfiablePersonaAceptacionCT{}, nil
	}
	d.bolsa = func(context.Context, bp.SolicitudConsultaPersonaAceptacionCT) (bp.ResultadoConsultaPersonaAceptacionCT, error) {
		t.Fatal("Bolsa tras cancelación")
		return bp.ResultadoConsultaPersonaAceptacionCT{}, nil
	}
	if _, e := f.LeerPersonaSeleccionada(ctx, c); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	ctx, cancelar = context.WithCancel(context.Background())
	d.autorizar = func(context.Context, dom.PublicacionCategoriaRPT) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		cancelar()
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
	}
	d.publicacion = func(context.Context, dom.PublicacionCategoriaRPT, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (dom.PublicacionCategoriaRPT, error) {
		t.Fatal("RPT tras cancelación")
		return dom.PublicacionCategoriaRPT{}, nil
	}
	if _, e := f.LeerPuestoRPT(ctx, c); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}

func TestFuentesContextoB2ConstructorRechazaDependenciasNil(t *testing.T) {
	f, _, _ := escenarioFuentesContextoB2(t)
	for _, alterar := range []func(*ConfiguracionFuentesContextoIncorporacionB2){
		func(c *ConfiguracionFuentesContextoIncorporacionB2) { c.Bolsa = (*fuentesContextoB2Prueba)(nil) },
		func(c *ConfiguracionFuentesContextoIncorporacionB2) { c.ActorBolsa = nil },
		func(c *ConfiguracionFuentesContextoIncorporacionB2) { c.Vinculos = nil },
		func(c *ConfiguracionFuentesContextoIncorporacionB2) { c.Publicacion = nil },
		func(c *ConfiguracionFuentesContextoIncorporacionB2) { c.AutoridadRPT = nil },
	} {
		c := f.c
		alterar(&c)
		if _, e := NuevasFuentesContextoIncorporacionB2(c); !errors.Is(e, ct.ErrComposicionIncorporacionAplicacion) {
			t.Fatal(e)
		}
	}
}
