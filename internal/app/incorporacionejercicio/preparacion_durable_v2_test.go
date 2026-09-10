package incorporacionejercicio

// Dobles sólo en fronteras de prueba. Se reutilizan los fixtures nominales
// existentes; esta prueba NO acredita PostgreSQL, permisos SQL o montaje HTTP.
import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
	hist "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/historiaincorporacion"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	pgpersonal "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
)

type detallePreparacionDoble func(context.Context, ct.SolicitudDetalleRRHH) (ct.DetalleExpedienteRRHH, error)

func (f detallePreparacionDoble) Consultar(c context.Context, s ct.SolicitudDetalleRRHH) (ct.DetalleExpedienteRRHH, error) {
	return f(c, s)
}

type planPreparacionDoble func(context.Context, string, string) (PlanPreparacionDurableV2, error)

func (f planPreparacionDoble) ResolverPlan(c context.Context, o, e string) (PlanPreparacionDurableV2, error) {
	return f(c, o, e)
}

type inicialPreparacionDoble func(context.Context, string, string, string) (dom.PublicacionDefinicionSeguimiento, dom.EstadoPersistidoSeguimiento, uint64, error)

func (f inicialPreparacionDoble) LeerPreparacionInicial(c context.Context, o, e, r string) (dom.PublicacionDefinicionSeguimiento, dom.EstadoPersistidoSeguimiento, uint64, error) {
	return f(c, o, e, r)
}

type localizadorCTPreparacionDoble func(context.Context, string, string, string) (hist.Selector, bool, error)

func (f localizadorCTPreparacionDoble) Localizar(c context.Context, o, e, s string) (hist.Selector, bool, error) {
	return f(c, o, e, s)
}

type localizadorPersonalPreparacionDoble func(context.Context, string, string, string) (pgpersonal.SolicitudLocalizadaAltaEjercicio, bool, error)

func (f localizadorPersonalPreparacionDoble) Localizar(c context.Context, o, e, s string) (pgpersonal.SolicitudLocalizadaAltaEjercicio, bool, error) {
	return f(c, o, e, s)
}

type restauradorPreparacionDoble func(context.Context, hist.Selector) (hist.Restauracion, error)

func (f restauradorPreparacionDoble) Restaurar(c context.Context, s hist.Selector) (hist.Restauracion, error) {
	return f(c, s)
}

type lectorPersonalPreparacionDoble func(context.Context, lector.Selector, string, ct.ContextoAutorizacionAltaV3) (lector.Resultado, error)

func (f lectorPersonalPreparacionDoble) Leer(c context.Context, s lector.Selector, u string, a ct.ContextoAutorizacionAltaV3) (lector.Resultado, error) {
	return f(c, s, u, a)
}

type casoPreparacionV2 struct {
	p          *PreparadorDurableV2
	app        *casoApp
	a          *autoridadEscenario
	plan       PlanPreparacionDurableV2
	detalle    ct.DetalleExpedienteRRHH
	pub        dom.PublicacionDefinicionSeguimiento
	estado     dom.EstadoPersistidoSeguimiento
	pasos      []string
	parcial    bool
	confirmada bool
	hook       func(string)
}

func nuevoCasoPreparacionV2(t *testing.T) *casoPreparacionV2 {
	t.Helper()
	c := &casoPreparacionV2{app: nuevoCasoApp(t), a: autoridadEntorno(t)}
	x := c.app.p.p
	c.a.a.peticion.PreparacionCT = x.Preparacion
	c.plan = PlanPreparacionDurableV2{OrganizacionRef: x.Preparacion.OrganizacionRef, UnidadRef: x.Preparacion.UnidadRef,
		SolicitudPersonal: x.SolicitudPersonal, FuentePersonal: c.app.s.c.TernaPersonal,
		SeguimientoRef: registroV2Ref("seguimiento:ejercicio:001"), RelacionRef: registroV2Ref("relacion:app"), VersionExpedienteRaiz: 8,
		Periodo: dom.IntervaloSeguimiento{Desde: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC), Hasta: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)}, MotivoClave: x.MotivoClave, Documentos: x.Documentos, MotivoV3: x.MotivoV3}
	b, err := os.ReadFile("../../modules/contrataciontemporal/adapters/seguimientoejercicio/testdata/definicion-ejercicio.json")
	registroV2Exigir(t, err)
	var archivo struct {
		Publicacion dom.PublicacionDefinicionSeguimiento `json:"publicacion"`
	}
	registroV2Exigir(t, json.Unmarshal(b, &archivo))
	def, err := dom.RestaurarDefinicionSeguimiento(archivo.Publicacion)
	registroV2Exigir(t, err)
	c.pub, c.plan.Definicion = def.Publicacion(), def.Referencia()
	s, err := dom.NuevoSeguimiento(def, dom.AltaSeguimiento{Referencia: c.plan.SeguimientoRef, OrganizacionRef: c.plan.OrganizacionRef,
		ExpedienteRef: x.SolicitudPersonal.ExpedienteRef, RelacionRef: c.plan.RelacionRef, PeriodoPrevisto: c.plan.Periodo, CreadoEn: c.a.ahora.Add(-time.Hour)})
	registroV2Exigir(t, err)
	c.estado = s.Estado()
	c.detalle = detallePreparacionPrueba(c.plan, c.a.ahora)
	sol, err := ct.NuevaSolicitudDetalleRRHH(x.SolicitudPersonal.ExpedienteRef, 0)
	registroV2Exigir(t, err)
	registroV2Exigir(t, c.detalle.ValidarContenidoPublicablePara(sol))
	c.p, err = NuevoPreparadorDurableV2(ConfiguracionPreparacionDurableV2{
		Autoridad: c.a.a, Reloj: c.a.reloj, FuentePersonal: c.app.s.c.FuentePersonal, TernaPersonal: c.plan.FuentePersonal,
		Detalle: detallePreparacionDoble(func(_ context.Context, sol ct.SolicitudDetalleRRHH) (ct.DetalleExpedienteRRHH, error) {
			c.paso("detalle")
			if sol.ExpedienteRef() != c.plan.SolicitudPersonal.ExpedienteRef {
				t.Fatal("selector detalle")
			}
			return c.detalle.Clonar(), nil
		}),
		Planes: planPreparacionDoble(func(_ context.Context, o, e string) (PlanPreparacionDurableV2, error) {
			c.paso("plan")
			if o != c.plan.OrganizacionRef || e != c.plan.SolicitudPersonal.ExpedienteRef {
				t.Fatal("selector plan")
			}
			return c.plan.Copia(), nil
		}),
		Inicial: inicialPreparacionDoble(func(_ context.Context, o, e, r string) (dom.PublicacionDefinicionSeguimiento, dom.EstadoPersistidoSeguimiento, uint64, error) {
			c.paso("inicial")
			if o != c.plan.OrganizacionRef || e != c.plan.SolicitudPersonal.ExpedienteRef || r != c.plan.RelacionRef {
				t.Fatal("selector raíz")
			}
			return c.pub, c.estado, c.plan.VersionExpedienteRaiz, nil
		}),
		LocalizadorCT: localizadorCTPreparacionDoble(func(_ context.Context, o, e, s string) (hist.Selector, bool, error) {
			c.paso("ct81")
			c.selector(t, o, e, s)
			if !c.confirmada {
				return hist.Selector{}, false, nil
			}
			r := c.app.ctTX.persistido.Recibo
			return hist.Selector{ReciboRef: r.Transicion.ReciboRef, MaterialSHA256: r.MaterialOriginalSHA256, IntencionSHA256: r.IntencionSHA256}, true, nil
		}),
		Restaurador: restauradorPreparacionDoble(func(_ context.Context, s hist.Selector) (hist.Restauracion, error) {
			c.paso("restaurar")
			if !c.confirmada || s.ReciboRef != c.app.ctTX.persistido.Recibo.Transicion.ReciboRef {
				t.Fatal("restauración sin selector")
			}
			return hist.Restauracion{OrdenOriginal: c.app.ctTX.original, Historia: c.app.ctTX.persistido.Historia}, nil
		}),
		LocalizadorPersonal: localizadorPersonalPreparacionDoble(func(_ context.Context, o, e, s string) (pgpersonal.SolicitudLocalizadaAltaEjercicio, bool, error) {
			c.paso("personal6")
			c.selector(t, o, e, s)
			if !c.parcial {
				return pgpersonal.SolicitudLocalizadaAltaEjercicio{}, false, nil
			}
			r := c.app.alta.registro(t)
			return pgpersonal.SolicitudLocalizadaAltaEjercicio{Solicitud: r.Solicitud, Selector: lector.Selector{OrganizacionRef: o, ExpedienteRef: e, SolicitudRef: s, VersionExpediente: r.Solicitud.VersionExpediente,
				ResultadoRef: r.Resultado.ResultadoRef, ReciboRef: r.Resultado.ReciboRef, RelacionRef: r.Resultado.RelacionRef, OcupacionRef: r.Resultado.OcupacionRef, MaterialSHA256: r.MaterialSHA256}}, true, nil
		}),
		LectorPersonal: lectorPersonalPreparacionDoble(func(_ context.Context, s lector.Selector, u string, a ct.ContextoAutorizacionAltaV3) (lector.Resultado, error) {
			c.paso("personal5")
			if !c.parcial || u != c.plan.UnidadRef || !mismoContexto(a, c.a.a.contexto) || s.RelacionRef != c.plan.RelacionRef {
				t.Fatal("lector no ligado")
			}
			return lector.Resultado{Registro: c.app.alta.registro(t), DecisionLecturaRef: "decision:lectura:actual", AuditoriaLecturaRef: "auditoria:lectura:actual", ConsumoHuellaSHA256: strings.Repeat("a", 64), LeidaEn: c.a.ahora}, nil
		}),
	})
	registroV2Exigir(t, err)
	return c
}

func (c *casoPreparacionV2) paso(p string) {
	c.pasos = append(c.pasos, p)
	if c.hook != nil {
		c.hook(p)
	}
}
func (c *casoPreparacionV2) selector(t *testing.T, o, e, s string) {
	t.Helper()
	if o != c.plan.OrganizacionRef || e != c.plan.SolicitudPersonal.ExpedienteRef || s != c.plan.SolicitudPersonal.SolicitudRef {
		t.Fatal("selector cruzado")
	}
}
func detallePreparacionPrueba(p PlanPreparacionDurableV2, t time.Time) ct.DetalleExpedienteRRHH {
	t = t.Add(-time.Hour)
	d := ct.DetalleExpedienteRRHH{Resumen: ct.ResumenExpedienteRRHH{ExpedienteRef: p.SolicitudPersonal.ExpedienteRef, OrganizacionRef: p.OrganizacionRef, NumeroVisible: "2026/CT-000001", Version: 8,
		FlujoRef: registroV2Ref("flujo"), FlujoVersion: 1, FlujoHuella: strings.Repeat("a", 64), FaseClave: "nombramiento", EstadoClave: dom.EstadoEnCurso,
		CentroRef: "centro:ejercicio:0001", CategoriaRef: "categoria:ejercicio:1", UnidadRef: p.UnidadRef, CreadoEn: t, ActualizadoEn: t.Add(7 * time.Minute)},
		Solicitud: ct.SolicitudOperativaRRHH{GrupoSubgrupo: "C1", MotivoClave: "ejercicio", PeriodoInicio: p.Periodo.Desde, PeriodoFin: p.Periodo.Hasta},
		Analisis:  &ct.AnalisisOperativoRRHH{ModalidadClave: "ejercicio", CategoriaRef: "categoria:ejercicio:1", CausaClave: "ejercicio", PeriodoInicio: p.Periodo.Desde, PeriodoFin: p.Periodo.Hasta, PorcentajeJornada: 10000, ResultadoRC: dom.RCNoRequerida},
		Cobertura: &ct.CoberturaOperativaRRHH{ViaClave: "ejercicio", DecisionGobernada: true}, Asignacion: &ct.AsignacionOperativaRRHH{UnidadRef: p.UnidadRef, AsignadaEn: t}}
	d.Resumen.ModalidadClave = "ejercicio"
	for i := uint64(1); i <= 8; i++ {
		h := ct.HitoExpedienteRRHH{Secuencia: i, VersionExpediente: i, AccionClave: "ejercicio", RealizadaEn: t.Add(time.Duration(i-1) * time.Minute), FaseOrigen: "nombramiento", FaseDestino: "nombramiento", EstadoOrigen: dom.EstadoEnCurso, EstadoDestino: dom.EstadoEnCurso}
		if i == 1 {
			h.FaseOrigen = ""
			h.EstadoOrigen = dom.EstadoPendiente
		}
		d.Hitos = append(d.Hitos, h)
	}
	return d
}

func TestPreparadorDurableV2InicialVersionesYRelectura(t *testing.T) {
	c := nuevoCasoPreparacionV2(t)
	ctx := context.Background()
	p, err := c.p.Consultar(ctx, c.plan.SolicitudPersonal.ExpedienteRef)
	registroV2Exigir(t, err)
	if p.Recibo != nil || p.Preparacion == nil || p.Preparacion.VersionSolicitudPersonal != 7 || p.VersionActualExpediente != 8 || p.Preparacion.VersionSeguimientoEsperada != 0 {
		t.Fatal("versiones mezcladas")
	}
	if !reflect.DeepEqual(c.pasos, []string{"detalle", "plan", "ct81", "personal6", "inicial"}) {
		t.Fatal(c.pasos)
	}
	x, err := c.p.Preparar(ctx, c.app.i)
	registroV2Exigir(t, err)
	if x.SolicitudPersonal != c.plan.SolicitudPersonal || x.VersionActualExpediente != 8 || x.VersionSeguimientoEsperada != 0 {
		t.Fatal("intención cambiada")
	}
	x.Documentos[0].Referencia = "mutada"
	nuevo, err := NuevoPreparadorDurableV2(c.p.c)
	registroV2Exigir(t, err)
	y, err := nuevo.Consultar(ctx, c.plan.SolicitudPersonal.ExpedienteRef)
	registroV2Exigir(t, err)
	if !reflect.DeepEqual(p, y) || c.app.alta.n != 0 || c.app.ctTX.llamadas != 0 || c.a.store.registros != 0 {
		t.Fatal("relectura o efectos")
	}
}

func TestPreparadorDurableV2OriginalYParcial(t *testing.T) {
	for _, confirmada := range []bool{false, true} {
		t.Run(map[bool]string{false: "parcial", true: "confirmada"}[confirmada], func(t *testing.T) {
			c := nuevoCasoPreparacionV2(t)
			// Construye sólo el fixture original por el servicio nominal existente.
			original, err := c.app.s.Confirmar(context.Background(), c.app.i)
			registroV2Exigir(t, err)
			c.parcial, c.confirmada = true, confirmada
			if confirmada {
				c.p.c.FuentePersonal = nil
				c.estado = dom.EstadoPersistidoSeguimiento{}
			}
			prevAlta, prevCT := c.app.alta.n, c.app.ctTX.llamadas
			for i := 0; i < 2; i++ {
				nuevo, err := NuevoPreparadorDurableV2(c.p.c)
				registroV2Exigir(t, err)
				r, err := nuevo.Consultar(context.Background(), c.plan.SolicitudPersonal.ExpedienteRef)
				registroV2Exigir(t, err)
				if confirmada {
					if r.Recibo == nil || !reflect.DeepEqual(*r.Recibo, original) || r.Preparacion != nil {
						t.Fatal("original reetiquetado")
					}
				} else if r.Recibo != nil || r.Preparacion.SolicitudPersonalRef != original.SolicitudPersonalRef {
					t.Fatal("parcial inventado")
				}
			}
			if c.app.alta.n != prevAlta || c.app.ctTX.llamadas != prevCT {
				t.Fatal("GET escribe")
			}
			if confirmada && !reflect.DeepEqual(c.pasos, []string{"detalle", "plan", "ct81", "restaurar", "detalle", "plan", "ct81", "restaurar"}) {
				t.Fatal(c.pasos)
			}
		})
	}
}

func TestPreparadorDurableV2Fronteras(t *testing.T) {
	for _, caso := range []string{"denegada", "ausencia_plan", "conflicto", "expediente", "solicitud", "version", "raiz", "definicion", "estado", "motivo", "documento", "periodo", "fuente"} {
		t.Run(caso, func(t *testing.T) {
			c := nuevoCasoPreparacionV2(t)
			i := c.app.i.Copia()
			esperado := ct.ErrConflictoIncorporacionAplicacion
			switch caso {
			case "denegada":
				c.p.c.Detalle = detallePreparacionDoble(func(context.Context, ct.SolicitudDetalleRRHH) (ct.DetalleExpedienteRRHH, error) {
					return ct.DetalleExpedienteRRHH{}, appct.ErrConsultaRRHHNoObservable
				})
				esperado = ct.ErrDenegadaIncorporacionAplicacion
			case "ausencia_plan":
				c.p.c.Planes = planPreparacionDoble(func(context.Context, string, string) (PlanPreparacionDurableV2, error) {
					return PlanPreparacionDurableV2{}, ct.ErrComposicionIncorporacionAplicacion
				})
				esperado = ct.ErrComposicionIncorporacionAplicacion
			case "conflicto":
				c.p.c.LocalizadorCT = localizadorCTPreparacionDoble(func(context.Context, string, string, string) (hist.Selector, bool, error) {
					return hist.Selector{}, false, ct.ErrConflictoIncorporacionAplicacion
				})
			case "expediente":
				c.detalle.Resumen.OrganizacionRef = "organizacion:ajena"
			case "solicitud":
				i.SolicitudPersonalRef = "solicitud:ajena"
			case "version":
				i.VersionActualExpedienteObservada = 7
			case "raiz":
				c.plan.SeguimientoRef = registroV2Ref("otra raiz")
			case "definicion":
				c.plan.Definicion.Version++
			case "estado":
				c.estado.HuellaRaizSHA256 = strings.Repeat("f", 64)
				esperado = ct.ErrComposicionIncorporacionAplicacion
			case "motivo":
				c.plan.MotivoClave = "otro_motivo"
			case "documento":
				c.plan.Documentos = nil
			case "periodo":
				c.plan.Periodo.Hasta = c.plan.Periodo.Hasta.Add(time.Hour)
			case "fuente":
				c.p.c.FuentePersonal = []byte("no disponible")
				esperado = ct.ErrComposicionIncorporacionAplicacion
			}
			r, err := c.p.Preparar(context.Background(), i)
			if !errors.Is(err, esperado) || !reflect.DeepEqual(r, ct.PreparacionIncorporacionAplicacionV2{}) {
				t.Fatalf("%s: %v", caso, err)
			}
			if caso == "denegada" && len(c.pasos) != 0 {
				t.Fatal("leyó sin autorización")
			}
			if c.app.alta.n != 0 || c.app.ctTX.llamadas != 0 {
				t.Fatal("efecto")
			}
		})
	}
}

func TestPreparadorDurableV2Cancelacion(t *testing.T) {
	for _, paso := range []string{"detalle", "plan", "ct81", "personal6", "inicial", "ultimo_reloj"} {
		t.Run(paso, func(t *testing.T) {
			c := nuevoCasoPreparacionV2(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if paso == "ultimo_reloj" {
				llamadas := 0
				c.p.c.Reloj = autoridadRelojDoble{instante: c.a.ahora, despues: func() {
					llamadas++
					if llamadas == 2 {
						cancel()
					}
				}}
			} else {
				c.hook = func(p string) {
					if p == paso {
						cancel()
					}
				}
			}
			r, err := c.p.Consultar(ctx, c.plan.SolicitudPersonal.ExpedienteRef)
			if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(r, ct.ProyeccionIncorporacionAplicacionV2{}) {
				t.Fatal("cancelación ignorada", err)
			}
		})
	}
}

func TestPreparadorDurableV2RecuperacionCruces(t *testing.T) {
	for _, caso := range []string{"selector", "historia", "version_actual", "cancelacion"} {
		t.Run(caso, func(t *testing.T) {
			c := nuevoCasoPreparacionV2(t)
			_, err := c.app.s.Confirmar(context.Background(), c.app.i)
			registroV2Exigir(t, err)
			c.confirmada = true
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch caso {
			case "selector":
				base := c.p.c.LocalizadorCT
				c.p.c.LocalizadorCT = localizadorCTPreparacionDoble(func(ctx context.Context, o, e, s string) (hist.Selector, bool, error) {
					sel, ok, err := base.Localizar(ctx, o, e, s)
					sel.IntencionSHA256 = strings.Repeat("b", 64)
					return sel, ok, err
				})
			case "historia":
				c.p.c.Restaurador = restauradorPreparacionDoble(func(context.Context, hist.Selector) (hist.Restauracion, error) { return hist.Restauracion{}, nil })
			case "version_actual":
				c.detalle.Resumen.Version = 9
				h := c.detalle.Hitos[7]
				h.Secuencia = 9
				h.VersionExpediente = 9
				c.detalle.Hitos = append(c.detalle.Hitos, h)
			case "cancelacion":
				c.hook = func(p string) {
					if p == "restaurar" {
						cancel()
					}
				}
			}
			r, err := c.p.Consultar(ctx, c.plan.SolicitudPersonal.ExpedienteRef)
			if caso == "version_actual" {
				registroV2Exigir(t, err)
				if r.VersionActualExpediente != 9 || r.Recibo == nil || r.Recibo.VersionActualExpediente != 8 {
					t.Fatal("historia actualizada")
				}
				i := c.app.i.Copia()
				i.VersionActualExpedienteObservada = 9
				_, err = c.p.Preparar(ctx, i)
				if !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) {
					t.Fatal("intención original rehecha", err)
				}
				return
			}
			if err == nil || !reflect.DeepEqual(r, ct.ProyeccionIncorporacionAplicacionV2{}) {
				t.Fatal("original cruzado")
			}
			if caso == "cancelacion" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}
