package internactproveedores

import (
	"context"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type destinoIntentosOHPrueba struct {
	ordenes          []vecports.OrdenIntentoAuditoria
	fallo, preflight error
	acuseInvalido    bool
	antes            func()
}

func TestIntentosOHOrganismosValidosParaAmbosContratos(t *testing.T) {
	for _, org := range []string{"dipgra", "httpinterno"} {
		t.Run(org, func(t *testing.T) {
			i := contextoOHPrueba(t, time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC))
			f := &fuenteOHPrueba{valor: i, organismo: org}
			d := &destinoIntentosOHPrueba{}
			r, _ := NuevoRegistroIntentosOrganizacionHistorica(DependenciasIntentosOrganizacionHistorica{d, configIntentosOHPrueba()})
			m := consultaOHPrueba(t, f, org, "")
			if m.Recurso().Referencia != org {
				t.Fatal("recurso V3 modificado")
			}
			c, _ := NuevaConsultaOrganizacionHistoricaConIntentos(consultorIntentosOHPrueba(func(ctx context.Context, _ personal.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
				return personalports.ResultadoConsultaOrganizacionHistorica{}, r.RegistrarIntentoConsultaOrganizacionHistorica(ctx, personalports.IntentoConsultaOrganizacionHistorica{Motivo: "denegado"})
			}), f, r)
			ctx, _ := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
			if _, err := c.Consultar(ctx, m.Solicitud()); err != nil || len(d.ordenes) != 1 {
				t.Fatal("organismo válido no auditable")
			}
			got, _ := d.ordenes[0].Datos()
			if got.Datos.RecursoRef != "personal:organizacion_historica:"+org {
				t.Fatal("recurso común divergente")
			}
		})
	}
}

type fuenteOHCaducidadPrueba struct {
	*fuenteOHPrueba
	reloj *relojB2
}

func (f fuenteOHCaducidadPrueba) ContextoVinculadoOrganizacionHistorica(ctx context.Context) (ct.ContextoAutorizacionAltaV3, string, string, error) {
	if !f.valor.Vinculo.VigenteEn(f.reloj.Ahora(), f.valor.Resultado) {
		return ct.ContextoAutorizacionAltaV3{}, "", "", ErrOrganizacionHistoricaV3NoDisponible
	}
	return f.fuenteOHPrueba.ContextoVinculadoOrganizacionHistorica(ctx)
}

type autoridadOHCruceCaducidadPrueba struct{ fuente fuenteOHCaducidadPrueba }

func (a autoridadOHCruceCaducidadPrueba) ResolverContextoOrganizacionHistorica(ctx context.Context) (core.ContextoActor, string, string, error) {
	r, org, unidad, err := a.fuente.ContextoVinculadoOrganizacionHistorica(ctx)
	if err != nil {
		return core.ContextoActor{}, "", "", err
	}
	datos, _ := r.Vinculo.Datos()
	a.fuente.reloj.ahora = datos.SesionValidaHasta
	return r.Resultado.Contexto, org, unidad, nil
}

type repositorioOHSinLecturaPrueba struct{ n int }

func (r *repositorioOHSinLecturaPrueba) ConsultarOrganizacionHistorica(context.Context, personalports.OrdenConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
	r.n++
	return personalports.ResultadoConsultaOrganizacionHistorica{}, errors.New("lectura prohibida")
}

type auditorFronteraOHCaducidadPrueba struct{ n int }

func (a *auditorFronteraOHCaducidadPrueba) RegistrarDenegacionOrganizacionHistorica(context.Context, httpapi.DenegacionOrganizacionHistorica) error {
	a.n++
	return nil
}
func TestIntentosOHVinculoVenceEntreHTTPYCaptura(t *testing.T) {
	p, f, pdp, e, _ := escenarioOHPrueba(t)
	fuente := fuenteOHCaducidadPrueba{f, e.reloj}
	p.fuente = fuente
	original, err := f.valor.Resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	d := &destinoIntentosOHPrueba{}
	r, _ := NuevoRegistroIntentosOrganizacionHistorica(DependenciasIntentosOrganizacionHistorica{d, configIntentosOHPrueba()})
	repo := &repositorioOHSinLecturaPrueba{}
	s, err := personalapp.NuevoServicioConsultaOrganizacionHistorica(p, repo, r)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NuevaConsultaOrganizacionHistoricaConIntentos(s, fuente, r)
	if err != nil {
		t.Fatal(err)
	}
	auditor := &auditorFronteraOHCaducidadPrueba{}
	h, err := httpapi.NewHandlerOrganizacionHistoricaPersonal(autoridadOHCruceCaducidadPrueba{fuente}, c, auditor)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
	request := httptest.NewRequest("GET", httpapi.RutaOrganizacionHistoricaPersonal+"?vigente_en=2026-09-25&conocido_en=2026-09-25T00:00:00.000000Z", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, request)
	if w.Code != 503 || len(d.ordenes) != 1 || repo.n != 0 || len(pdp.solicitudes) != 0 || auditor.n != 0 {
		t.Fatalf("caducidad: HTTP=%d intentos=%d lecturas=%d PDP=%d frontera=%d", w.Code, len(d.ordenes), repo.n, len(pdp.solicitudes), auditor.n)
	}
	got, _ := d.ordenes[0].Datos()
	if got.Datos.Resultado != core.ResultadoIntentoAuditoriaError || !reflect.DeepEqual(got.ResultadoContexto, original) {
		t.Fatal("error no conserva identidad original")
	}
}

func (d *destinoIntentosOHPrueba) PreflightIntentoAuditoria(context.Context) error {
	return d.preflight
}
func (d *destinoIntentosOHPrueba) AppendIntentoAuditoria(ctx context.Context, o vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	if ctx.Err() != nil {
		return vecports.AcuseIntentoAuditoria{}, ctx.Err()
	}
	if d.antes != nil {
		d.antes()
	}
	d.ordenes = append(d.ordenes, o)
	datos, err := o.Datos()
	if err != nil {
		return vecports.AcuseIntentoAuditoria{}, err
	}
	acuse := vecports.AcuseIntentoAuditoria{AuditoriaRef: "aud_oh_prueba", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: datos.Datos.CorrelacionRef, RegistradaEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)}
	if d.acuseInvalido {
		acuse.CorrelacionRef = "otra"
	}
	return acuse, d.fallo
}
func configIntentosOHPrueba() ConfiguracionIntentosOrganizacionHistorica {
	m := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos.oh", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"}
	return ConfiguracionIntentosOrganizacionHistorica{Proceso: "vec-interno", Canal: string(core.SuperficieAutenticacionInternaCorporativaV1), MotivoDenegado: m, MotivoEntradaInvalida: m, MotivoNoDisponible: m}
}

type consultorIntentosOHPrueba func(context.Context, personal.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error)

func (f consultorIntentosOHPrueba) Consultar(ctx context.Context, s personal.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
	return f(ctx, s)
}

func TestIntentosOHCapturaOriginalYRecuperaMismaOrden(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	identidad := contextoOHPrueba(t, ahora)
	f := &fuenteOHPrueba{valor: identidad, organismo: "organizacion:prueba", unidad: "unidad:prueba"}
	d := &destinoIntentosOHPrueba{}
	r, err := NuevoRegistroIntentosOrganizacionHistorica(DependenciasIntentosOrganizacionHistorica{d, configIntentosOHPrueba()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ref, _ := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	correlacion, _ := ref.ValorCanonico()
	var capturado context.Context
	c, err := NuevaConsultaOrganizacionHistoricaConIntentos(consultorIntentosOHPrueba(func(ctx context.Context, _ personal.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
		capturado = ctx
		// Una resolución posterior no puede sustituir el sujeto histórico.
		f.valor.Resultado.RepresentacionCanonica[0] = '!'
		if err := r.RegistrarIntentoConsultaOrganizacionHistorica(ctx, personalports.IntentoConsultaOrganizacionHistorica{Motivo: "denegado"}); err != nil {
			t.Fatal(err)
		}
		return personalports.ResultadoConsultaOrganizacionHistorica{}, personal.ErrConsultaOrganizacionHistoricaDenegada
	}), f, r)
	if err != nil {
		t.Fatal(err)
	}
	s := personal.SolicitudConsultaOrganizacionHistorica{Actor: identidad.Resultado.Contexto, Selector: personal.SelectorOrganizacionHistorica{OrganismoRef: f.organismo, UnidadClave: f.unidad}}
	if _, err := c.Consultar(ctx, s); !errors.Is(err, personal.ErrConsultaOrganizacionHistoricaDenegada) {
		t.Fatal(err)
	}
	if f.llamadas != 1 || len(d.ordenes) != 1 {
		t.Fatal("resolución o intento duplicados")
	}
	datos, err := d.ordenes[0].Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datos.Datos.CorrelacionRef != correlacion || datos.Datos.RecursoRef != "personal:organizacion_historica:"+s.Selector.OrganismoRef || !reflect.DeepEqual(datos.ResultadoContexto.Contexto, s.Actor) || datos.Datos.Resultado != core.ResultadoIntentoAuditoriaDenegado {
		t.Fatal("identidad, recurso o correlación alterados")
	}
	if err := r.RegistrarIntentoConsultaOrganizacionHistorica(capturado, personalports.IntentoConsultaOrganizacionHistorica{Motivo: "denegado"}); err != nil || len(d.ordenes) != 1 {
		t.Fatal("acuse confirmado se volvió a escribir")
	}
	if err := r.RegistrarIntentoConsultaOrganizacionHistorica(capturado, personalports.IntentoConsultaOrganizacionHistorica{Motivo: "no_disponible"}); err == nil {
		t.Fatal("misma referencia con material distinto")
	}
}

func TestIntentosOHExigeIdentidadYAcuseValido(t *testing.T) {
	for _, invalidar := range []func(*destinoIntentosOHPrueba){func(d *destinoIntentosOHPrueba) { d.fallo = errors.New("destino privado") }, func(d *destinoIntentosOHPrueba) { d.acuseInvalido = true }} {
		i := contextoOHPrueba(t, time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC))
		f := &fuenteOHPrueba{valor: i, organismo: "organizacion:prueba"}
		d := &destinoIntentosOHPrueba{}
		invalidar(d)
		r, _ := NuevoRegistroIntentosOrganizacionHistorica(DependenciasIntentosOrganizacionHistorica{d, configIntentosOHPrueba()})
		c, _ := NuevaConsultaOrganizacionHistoricaConIntentos(consultorIntentosOHPrueba(func(ctx context.Context, _ personal.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
			return personalports.ResultadoConsultaOrganizacionHistorica{}, r.RegistrarIntentoConsultaOrganizacionHistorica(ctx, personalports.IntentoConsultaOrganizacionHistorica{Motivo: "no_disponible"})
		}), f, r)
		ctx, _ := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
		if _, err := c.Consultar(ctx, personal.SolicitudConsultaOrganizacionHistorica{Actor: i.Resultado.Contexto, Selector: personal.SelectorOrganizacionHistorica{OrganismoRef: f.organismo}}); !errors.Is(err, personal.ErrOrganizacionHistoricaNoDisponible) || len(d.ordenes) != 2 {
			t.Fatal("fallo/acuse inválido confirmado")
		}
		a, _ := d.ordenes[0].Datos()
		b, _ := d.ordenes[1].Datos()
		if a.IntentoRef != b.IntentoRef || a.Datos != b.Datos {
			t.Fatal("reintento recreó orden")
		}
	}
	d := &destinoIntentosOHPrueba{}
	r, _ := NuevoRegistroIntentosOrganizacionHistorica(DependenciasIntentosOrganizacionHistorica{d, configIntentosOHPrueba()})
	if r.VerificarRegistroConsultaOrganizacionHistorica(context.Background()) == nil {
		t.Fatal("sin captura")
	}
	if r.RegistrarIntentoConsultaOrganizacionHistorica(context.Background(), personalports.IntentoConsultaOrganizacionHistorica{Motivo: "denegado"}) == nil || len(d.ordenes) != 0 {
		t.Fatal("actor supuesto")
	}
	if _, err := NuevoRegistroIntentosOrganizacionHistorica(DependenciasIntentosOrganizacionHistorica{}); err == nil {
		t.Fatal("sin destino")
	}
	cfg := configIntentosOHPrueba()
	cfg.Canal = "otro"
	if _, err := NuevoRegistroIntentosOrganizacionHistorica(DependenciasIntentosOrganizacionHistorica{d, cfg}); err == nil {
		t.Fatal("canal supuesto")
	}
}

func TestProveedorOHReutilizaCapturaYCorrelacion(t *testing.T) {
	p, f, pdp, _, _ := escenarioOHPrueba(t)
	d := &destinoIntentosOHPrueba{}
	r, _ := NuevoRegistroIntentosOrganizacionHistorica(DependenciasIntentosOrganizacionHistorica{d, configIntentosOHPrueba()})
	m := consultaOHPrueba(t, f, f.organismo, f.unidad)
	ctx, _ := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
	corr, _ := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	c, _ := NuevaConsultaOrganizacionHistoricaConIntentos(consultorIntentosOHPrueba(func(ctx context.Context, _ personal.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
		f.err = errors.New("resolución posterior prohibida")
		_, err := p.AutorizarConsultaOrganizacionHistorica(ctx, m)
		return personalports.ResultadoConsultaOrganizacionHistorica{}, err
	}), f, r)
	if _, err := c.Consultar(ctx, m.Solicitud()); err != nil {
		t.Fatal(err)
	}
	if f.llamadas != 1 {
		t.Fatal("segunda fuente")
	}
	if len(pdp.solicitudes) != 1 {
		t.Fatal("sin solicitud V3")
	}
	datos, err := pdp.solicitudes[0].Datos()
	if err != nil {
		t.Fatal(err)
	}
	got, _ := datos.Correlacion.ValorCanonico()
	want, _ := corr.ValorCanonico()
	if got != want {
		t.Fatal("correlación V3 distinta de auditoría")
	}
}
