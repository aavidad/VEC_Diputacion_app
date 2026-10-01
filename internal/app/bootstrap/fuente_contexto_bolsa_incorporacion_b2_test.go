package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

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
