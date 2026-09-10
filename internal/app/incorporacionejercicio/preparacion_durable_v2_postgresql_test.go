package incorporacionejercicio

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"reflect"
	"strings"
	"testing"
	"time"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	pgpersonal "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
)

type relojSinIOPreparacionV2 struct{}

func (*relojSinIOPreparacionV2) Ahora() time.Time { panic("la fábrica no debe consultar el reloj") }

func configuracionPGPreparacionPrueba(t *testing.T, plan PlanPreparacionDurableV2) ConfiguracionPreparacionDurableV2PostgreSQL {
	t.Helper()
	b, err := json.Marshal(struct {
		Esquema    string                     `json:"esquema"`
		Referencia string                     `json:"referencia"`
		Version    uint64                     `json:"version"`
		Planes     []PlanPreparacionDurableV2 `json:"planes"`
	}{"vec.contratacion-temporal.planes-incorporacion.v2", "planes:ensamblaje:v2", 2, []PlanPreparacionDurableV2{plan}})
	registroV2Exigir(t, err)
	// Pools inertes exclusivamente para construcción sin IO. No se invocan
	// lectores ni Close sobre ellos; tampoco acreditan configuración/roles.
	return ConfiguracionPreparacionDurableV2PostgreSQL{Autoridad: &AutoridadAplicacion{}, Detalle: &appct.ServicioConsultaDetalleRRHH{}, Reloj: &relojSinIOPreparacionV2{},
		Planes: b, TernaPlanes: ct.ReferenciaVersionadaPersonalRPT{Referencia: "planes:ensamblaje:v2", Version: 2, HuellaSHA256: registroV2Hash(b)},
		Pools: PoolsPreparacionDurableV2{InicialCT: new(pgxpool.Pool), LocalizadorCT: new(pgxpool.Pool), LocalizadorPersonal: new(pgxpool.Pool), LecturaPersonal: new(pgxpool.Pool),
			Historia: pgct.PoolsHistoriaIncorporacionV2{RegistroCT: new(pgxpool.Pool), Autenticacion: new(pgxpool.Pool), Contexto: new(pgxpool.Pool), Evaluacion: new(pgxpool.Pool), Concesion: new(pgxpool.Pool)}}}
}

func TestPreparadorDurableV2PostgreSQLFabricaSinIO(t *testing.T) {
	c := nuevoCasoPreparacionV2(t)
	config := configuracionPGPreparacionPrueba(t, c.plan)
	p, err := NuevoPreparadorDurableV2PostgreSQL(config)
	registroV2Exigir(t, err)
	clear(config.Planes)
	plan, err := p.c.Planes.ResolverPlan(context.Background(), c.plan.OrganizacionRef, c.plan.SolicitudPersonal.ExpedienteRef)
	registroV2Exigir(t, err)
	if !reflect.DeepEqual(plan, c.plan) {
		t.Fatal("plan modificado por el ensamblaje")
	}
	for _, caso := range []string{"pool_nulo", "personal_ct_compartido", "historia_viva_compartida", "autoridad", "detalle", "sello", "reloj"} {
		t.Run(caso, func(t *testing.T) {
			x := configuracionPGPreparacionPrueba(t, c.plan)
			switch caso {
			case "pool_nulo":
				x.Pools.LocalizadorPersonal = nil
			case "personal_ct_compartido":
				x.Pools.LecturaPersonal = x.Pools.InicialCT
			case "historia_viva_compartida":
				x.Pools.Historia.RegistroCT = x.Pools.LocalizadorCT
			case "autoridad":
				x.Autoridad = nil
			case "detalle":
				x.Detalle = nil
			case "sello":
				x.Planes[0] = 'x'
			case "reloj":
				var nulo *relojSinIOPreparacionV2
				x.Reloj = nulo
			}
			p, err := NuevoPreparadorDurableV2PostgreSQL(x)
			if p != nil || err != ct.ErrComposicionIncorporacionAplicacion {
				t.Fatal("composición inválida aceptada", err)
			}
		})
	}
}

func TestPreparadorDurableV2PlanRealEnsamblado(t *testing.T) {
	c := nuevoCasoPreparacionV2(t)
	x := configuracionPGPreparacionPrueba(t, c.plan)
	x.Autoridad = c.a.a
	x.Reloj = c.a.reloj
	x.FuentePersonal = c.app.s.c.FuentePersonal
	x.TernaPersonal = c.plan.FuentePersonal
	p, err := NuevoPreparadorDurableV2PostgreSQL(x)
	registroV2Exigir(t, err)
	// Se conserva la carga REAL del plan y fuente Personal del ensamblaje.
	// Sólo se sustituyen fronteras IO por los dobles existentes de este paquete.
	p.c.Detalle = c.p.c.Detalle
	p.c.Inicial = c.p.c.Inicial
	p.c.LocalizadorCT = c.p.c.LocalizadorCT
	p.c.Restaurador = c.p.c.Restaurador
	p.c.LocalizadorPersonal = c.p.c.LocalizadorPersonal
	p.c.LectorPersonal = c.p.c.LectorPersonal
	for i := 0; i < 2; i++ {
		r, err := p.Preparar(context.Background(), c.app.i)
		registroV2Exigir(t, err)
		if r.SolicitudPersonal != c.plan.SolicitudPersonal || r.VersionActualExpediente != 8 || r.VersionSeguimientoEsperada != 0 {
			t.Fatal("plan/reintento alterado")
		}
	}
	if c.app.alta.n != 0 || c.app.ctTX.llamadas != 0 {
		t.Fatal("preparación con efectos")
	}
}

func TestPreparadorDurableV2ParcialConLectorNominal(t *testing.T) {
	for _, caso := range []string{"original", "denegada", "registro_cruzado"} {
		t.Run(caso, func(t *testing.T) {
			c := nuevoCasoPreparacionV2(t)
			// Fijar los dos ámbitos del DOBLE a las referencias opacas de la
			// raíz de prueba. No cambia roles ni el proveedor nominal real.
			c.a.fuente.p.PreparacionCT = c.a.a.peticion.PreparacionCT
			for i := range c.a.store.snapshot.AsignacionPerfil.Ambitos {
				a := &c.a.store.snapshot.AsignacionPerfil.Ambitos[i]
				if a.Clave == "organizacion_ref" {
					a.Valores = []string{c.plan.OrganizacionRef}
				}
				if a.Clave == "unidad_ref" {
					a.Valores = []string{c.plan.UnidadRef}
				}
			}
			registroV2Exigir(t, c.a.store.snapshot.Validar())
			material := autoridadMaterialAlta(t, c.a)
			material.Preparacion.Solicitud = c.plan.SolicitudPersonal
			material.Preparacion.Fuente = c.plan.FuentePersonal
			datos := autoridadMaterialCT(t, c.a, material)
			original := datos.Personal
			original.Resultado.RelacionRef = c.plan.RelacionRef
			registroV2Exigir(t, original.ValidarEstructuraPara(c.plan.SolicitudPersonal, c.a.ahora))
			def, err := dom.RestaurarDefinicionSeguimiento(c.pub)
			registroV2Exigir(t, err)
			s, err := dom.NuevoSeguimiento(def, dom.AltaSeguimiento{Referencia: c.plan.SeguimientoRef,
				OrganizacionRef: c.plan.OrganizacionRef, ExpedienteRef: c.plan.SolicitudPersonal.ExpedienteRef,
				RelacionRef: c.plan.RelacionRef, PeriodoPrevisto: c.plan.Periodo, CreadoEn: c.a.ahora.Add(-time.Hour)})
			registroV2Exigir(t, err)
			c.estado = s.Estado()
			c.detalle = detallePreparacionPrueba(c.plan, c.a.ahora)
			sel := lector.Selector{OrganizacionRef: c.plan.OrganizacionRef, ExpedienteRef: original.Solicitud.ExpedienteRef,
				SolicitudRef: original.Solicitud.SolicitudRef, VersionExpediente: original.Solicitud.VersionExpediente,
				ResultadoRef: original.Resultado.ResultadoRef, ReciboRef: original.Resultado.ReciboRef,
				RelacionRef: original.Resultado.RelacionRef, OcupacionRef: original.Resultado.OcupacionRef, MaterialSHA256: original.MaterialSHA256}
			c.p.c.LocalizadorPersonal = localizadorPersonalPreparacionDoble(func(_ context.Context, o, e, solicitud string) (pgpersonal.SolicitudLocalizadaAltaEjercicio, bool, error) {
				c.paso("personal6")
				c.selector(t, o, e, solicitud)
				return pgpersonal.SolicitudLocalizadaAltaEjercicio{Solicitud: original.Solicitud, Selector: sel}, true, nil
			})
			decisiones := map[string]bool{}
			lecturas := 0
			consumidor, err := lector.NuevoV2(c.a.a, txLecturaApp(func(ctx context.Context, recibido lector.Selector, orden lector.OrdenV2) (lector.Resultado, error) {
				lecturas++
				c.paso("personal5_nominal")
				registroV2Exigir(t, orden.ValidarEn(c.a.ahora))
				x := orden.Exportacion().ResumenCapacidad()
				if recibido != sel || orden.Selector() != sel || orden.UnidadRef() != c.plan.UnidadRef || x.AudienciaConsumo() != lector.AudienciaV2 || x.Operacion() != lector.Accion ||
					x.ContextoRef() != c.a.a.contexto.Resultado.RegistroContextoRef || decisiones[x.DecisionRef()] || x.DecisionRef() == original.DecisionOriginalRef {
					t.Fatal("lector sin permiso propio fresco/exacto")
				}
				decisiones[x.DecisionRef()] = true
				r := cloneRegistro(original)
				if caso == "registro_cruzado" {
					r.Resultado.ReciboRef = "recibo:ajeno"
				}
				return lector.Resultado{Registro: r, DecisionLecturaRef: x.DecisionRef(), AuditoriaLecturaRef: "auditoria:lectura:nominal",
					ConsumoHuellaSHA256: strings.Repeat("a", 64), LeidaEn: c.a.ahora}, nil
			}), c.a.reloj)
			registroV2Exigir(t, err)
			c.p.c.LectorPersonal = consumidor
			if caso == "denegada" {
				c.a.store.fallo = errors.New("indisponibilidad de la frontera nominal de prueba")
			}
			var primera ct.ProyeccionIncorporacionAplicacionV2
			for i := 0; i < 2; i++ {
				r, err := c.p.Consultar(context.Background(), c.plan.SolicitudPersonal.ExpedienteRef)
				if caso != "original" {
					if !errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) || !reflect.DeepEqual(r, ct.ProyeccionIncorporacionAplicacionV2{}) {
						t.Fatal("lectura fallida divulgada", err)
					}
					break
				}
				registroV2Exigir(t, err)
				if r.Recibo != nil || r.Preparacion == nil || r.Preparacion.SolicitudPersonalRef != original.Solicitud.SolicitudRef || r.Preparacion.VersionSolicitudPersonal != 7 || r.VersionActualExpediente != 8 {
					t.Fatal("parcial convertido en alta/incorporación")
				}
				if i == 0 {
					primera = r.Copia()
				} else if !reflect.DeepEqual(r, primera) {
					t.Fatal("preparación cambió al renovar permiso")
				}
			}
			if caso == "denegada" && (lecturas != 0 || c.a.store.registros != 0) {
				t.Fatal("se leyó Personal sin autorización")
			}
			if caso == "original" && (lecturas != 2 || len(decisiones) != 2 || c.a.store.registros != 2) {
				t.Fatal("no se renovó el permiso propio por consulta")
			}
			if c.app.alta.n != 0 || c.app.ctTX.llamadas != 0 {
				t.Fatal("preparar ejecutó un alta o confirmación")
			}
		})
	}
}
