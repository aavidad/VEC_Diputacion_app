package bootstrap

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	appvec "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func TestNuevoServicioAnotacionAdministrativaDesarrolloFallaCerradoSinFronteras(t *testing.T) {
	if servicio, err := NuevoServicioAnotacionAdministrativaDesarrollo(ConfiguracionAnotacionAdministrativaDesarrollo{}); servicio != nil || err == nil {
		t.Fatalf("configuracion incompleta aceptada: servicio=%#v err=%v", servicio, err)
	}
}

// Todas las fuentes de este ensayo son dobles privados; no abre PostgreSQL ni
// publica roles. Acredita la frontera y el montaje, no el recorrido durable.
func configuracionContinuidadNominalPrueba(t *testing.T) (archivoOperacionContinuidadNominal, ReferenciasCTIncorporacionDesarrollo, time.Time) {
	t.Helper()
	alta, _, _ := escenarioConsultasRRHHDesarrolloPrueba(t)
	v, _ := alta.soporte.contexto.Vinculo.Datos()
	refs := ReferenciasCTIncorporacionDesarrollo{PrincipalV3Ref: v.PrincipalID, PerfilV3Ref: v.PerfilActivoRef, OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, UnidadRef: "ref:" + strings.Repeat("a", 64), ActorRef: "ref:" + strings.Repeat("b", 64)}
	ahora := alta.soporte.reloj.Ahora()
	i, e := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(refs.PrincipalV3Ref, refs.PerfilV3Ref, ahora, "anotacion_nominal_prueba", "Anotación de prueba", "anotacion_nominal_prueba", []core.ConcesionRol{{Accion: string(dom.AccionRegistrarAnotacionAdministrativa), ModuloID: ct.ModuloContratacion, TipoRecurso: ct.TipoRecursoAnotacionAdministrativa, Finalidades: []string{ct.FinalidadRegistrarAnotacionAdministrativa}, GarantiaMinima: core.AuthAssuranceHigh}}, []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{refs.OrganizacionRef}}, {Clave: "expediente_ref", Valores: []string{"expediente:prueba"}}, {Clave: "fase_previa", Valores: []string{"seguimiento"}}, {Clave: "estado_previo", Valores: []string{"en_curso"}}})
	if e != nil {
		t.Fatal(e)
	}
	return archivoOperacionContinuidadNominal{Instantanea: i, Motivo: alta.soporte.motivoDetalleRRHH}, refs, ahora
}
func TestContinuidadNominalConfiguracionExigePermisoPropioYAmbitos(t *testing.T) {
	c, refs, ahora := configuracionContinuidadNominalPrueba(t)
	validar := func(c archivoOperacionContinuidadNominal) error {
		return validarOperacionContinuidadNominal(c, refs, string(dom.AccionRegistrarAnotacionAdministrativa), ct.FinalidadRegistrarAnotacionAdministrativa, ct.TipoRecursoAnotacionAdministrativa, ahora)
	}
	if e := validar(c); e != nil {
		t.Fatal(e)
	}
	for nombre, alterar := range map[string]func(*archivoOperacionContinuidadNominal){
		"rol lectura": func(c *archivoOperacionContinuidadNominal) {
			c.Instantanea.VersionRol.Concesiones[0].Accion = ct.AccionConsultarDetalleRRHH
		},
		"otra finalidad": func(c *archivoOperacionContinuidadNominal) {
			c.Instantanea.VersionRol.Concesiones[0].Finalidades = []string{ct.FinalidadConsultarDetalleRRHH}
		},
		"permiso adicional": func(c *archivoOperacionContinuidadNominal) {
			p := c.Instantanea.VersionRol.Concesiones[0]
			p.Accion = ct.AccionConsultarDetalleRRHH
			c.Instantanea.VersionRol.Concesiones = append(c.Instantanea.VersionRol.Concesiones, p)
		},
		"otro actor": func(c *archivoOperacionContinuidadNominal) {
			c.Instantanea.AsignacionPerfil.PrincipalID = "actor:ajeno"
		},
		"otra organizacion": func(c *archivoOperacionContinuidadNominal) {
			c.Instantanea.AsignacionPerfil.Ambitos[0].Valores = []string{"organizacion:ajena"}
		},
		"sin expediente": func(c *archivoOperacionContinuidadNominal) {
			c.Instantanea.AsignacionPerfil.Ambitos = append(c.Instantanea.AsignacionPerfil.Ambitos[:1], c.Instantanea.AsignacionPerfil.Ambitos[2:]...)
		},
		"vencida": func(c *archivoOperacionContinuidadNominal) { c.Instantanea.AsignacionPerfil.VigenteHasta = ahora },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := c
			alterada.Instantanea = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(c.Instantanea)
			alterar(&alterada)
			if validar(alterada) == nil {
				t.Fatal("configuración ajena admitida")
			}
		})
	}
}

type fuenteContinuidadNominalPrueba struct {
	instantanea core.InstantaneaAutorizacion
	llamadas    int
}

func (f *fuenteContinuidadNominalPrueba) ObtenerInstantaneaAutorizacion(context.Context, string, string) (core.InstantaneaAutorizacion, error) {
	f.llamadas++
	return f.instantanea, nil
}
func TestContinuidadNominalFuenteDetectaCarreraDeRol(t *testing.T) {
	c, refs, _ := configuracionContinuidadNominalPrueba(t)
	origen := &fuenteContinuidadNominalPrueba{instantanea: c.Instantanea}
	f := &fuenteRolContinuidadNominal{origen, refs, c.Instantanea.VersionRol, c.Instantanea.AsignacionPerfil}
	if _, e := f.ObtenerInstantaneaAutorizacion(context.Background(), refs.PrincipalV3Ref, refs.PerfilV3Ref); e != nil {
		t.Fatal(e)
	}
	origen.instantanea = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(c.Instantanea)
	origen.instantanea.VersionRol.Concesiones[0].Accion = ct.AccionConsultarDetalleRRHH
	if _, e := f.ObtenerInstantaneaAutorizacion(context.Background(), refs.PrincipalV3Ref, refs.PerfilV3Ref); e == nil {
		t.Fatal("rol de lectura sobrevenido admitido")
	}
	llamadas := origen.llamadas
	if _, e := f.ObtenerInstantaneaAutorizacion(context.Background(), "actor:ajeno", refs.PerfilV3Ref); e == nil || origen.llamadas != llamadas {
		t.Fatal("consultó o aceptó identidad ajena")
	}
}
func TestContinuidadNominalMontajeRealDeniegaSinCanal(t *testing.T) {
	alta, consultas, _ := escenarioConsultasRRHHDesarrolloPrueba(t)
	c := &continuidadNominalDesarrollo{pool: new(pgxpool.Pool), detalle: new(appct.ServicioConsultaDetalleRRHH), anotacion: &autoridadContinuidadNominal{soporte: alta.soporte, consultas: consultas, reloj: alta.soporte.reloj}, cierre: &autoridadContinuidadNominal{soporte: alta.soporte, consultas: consultas, reloj: alta.soporte.reloj}}
	rutas, e := c.rutas(nuevoDerivadorIdempotenciaPrueba(t, 2, 1))
	if e != nil {
		t.Fatal(e)
	}
	if len(rutas) != 4 {
		t.Fatalf("rutas nominales=%d", len(rutas))
	}
	for _, ruta := range rutas {
		metodo, cuerpo := http.MethodPost, `{}`
		if ruta.Ruta == httpct.RutaRecuperacionAnotacionesAdministrativas || ruta.Ruta == httpct.RutaPreparacionCierreSinCese {
			metodo = http.MethodGet
		}
		r := httptest.NewRequest(metodo, ruta.Ruta, strings.NewReader(cuerpo))
		w := httptest.NewRecorder()
		if !esRutaContratacionTemporalDesarrollo(r) {
			t.Fatal("ruta fuera de frontera mTLS")
		}
		ruta.Manejador.ServeHTTP(w, r)
		if w.Code < 400 {
			t.Fatalf("ruta sin canal aceptada: %s %d", ruta.Ruta, w.Code)
		}
	}
}
func TestContinuidadNominalCanalConservaIdentidadYSeparaOperacion(t *testing.T) {
	alta, consultas, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	v, _ := alta.soporte.contexto.Vinculo.Datos()
	refs := ReferenciasCTIncorporacionDesarrollo{PrincipalV3Ref: v.PrincipalID, PerfilV3Ref: v.PerfilActivoRef}
	a := &autoridadContinuidadNominal{soporte: alta.soporte, consultas: consultas, referencias: refs, accion: string(dom.AccionRegistrarAnotacionAdministrativa), reloj: alta.soporte.reloj}
	ctx := contextoRutaConsultasRRHHDesarrolloPrueba(alta.soporte, principal, httpct.RutaRecuperacionAnotacionesAdministrativas)
	hijo, e := contextoDetalleIncorporacionV2Desarrollo(ctx, alta.soporte)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.contexto(hijo); e == nil {
		t.Fatal("contexto sin operación nominal admitido")
	}
	hijo = context.WithValue(hijo, claveRutaContinuidadNominal{}, httpct.RutaRecuperacionAnotacionesAdministrativas)
	c, e := a.contexto(hijo)
	if e != nil {
		t.Fatal(e)
	}
	actual, _ := c.Vinculo.Datos()
	if actual.PrincipalID != v.PrincipalID || actual.PerfilActivoRef != v.PerfilActivoRef {
		t.Fatal("identidad sustituida")
	}
	f := &fuenteAutoridadIncorporacionV2Desarrollo{soporte: alta.soporte, consultas: consultas, referencias: refs}
	if _, e := f.PeticionVerificada(hijo); e == nil {
		t.Fatal("contexto continuidad habilitó incorporación")
	}
	a.accion = ct.AccionAutorizacionCerrarAdministrativamente
	if _, e = a.contexto(hijo); e == nil {
		t.Fatal("GET anotación habilitó cierre")
	}
}

// Registro/fuente de ensayo: el PDP y la decisión V3 sí son los reales del
// dominio. Este doble no acredita COMMIT ni emite material consumible en PG.
type registroContinuidadPDPPrueba struct {
	publicador  *publicadorPermisoIncorporacionPrueba
	motivo      core.ReferenciaEntradaCatalogo
	ahora       time.Time
	concesiones int
}

func (r *registroContinuidadPDPPrueba) ObtenerInstantaneaAutorizacion(context.Context, string, string) (core.InstantaneaAutorizacion, error) {
	return r.publicador.ultima, nil
}
func (r *registroContinuidadPDPPrueba) ValidarReferenciaMotivoAutorizacionV2(_ context.Context, m core.ReferenciaEntradaCatalogo, _ time.Time) error {
	if m != r.motivo {
		return ct.ErrAutorizacionDenegada
	}
	return nil
}
func (r *registroContinuidadPDPPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(_ context.Context, o vp.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	d, e := o.Datos()
	if e != nil || d.Decision.ValidarPara(d.Solicitud) != nil {
		return time.Time{}, ct.ErrAutorizacionDenegada
	}
	r.concesiones++
	desde, _, err := d.Decision.VentanaValidez()
	return desde, err
}
func (r *registroContinuidadPDPPrueba) RegistrarDenegacionAutorizacionLigadaV3(context.Context, vp.OrdenRegistroDenegacionAutorizacionLigadaV3) error {
	return nil
}
func TestContinuidadNominalPermisoDedicadoLlegaAPDPReal(t *testing.T) {
	base, ctx, datos, publicador := escenarioPermisoIncorporacionPrueba(t)
	c, _, ahora := configuracionContinuidadNominalPrueba(t)
	c.Motivo = base.motivoDetalle
	c.Instantanea.AsignacionPerfil.PrincipalID = base.referencias.PrincipalV3Ref
	c.Instantanea.AsignacionPerfil.PerfilActivoRef = base.referencias.PerfilV3Ref
	exp := datos.Recurso.Referencia
	for n := range c.Instantanea.AsignacionPerfil.Ambitos {
		if c.Instantanea.AsignacionPerfil.Ambitos[n].Clave == "expediente_ref" {
			c.Instantanea.AsignacionPerfil.Ambitos[n].Valores = []string{exp}
		}
	}
	registro := &registroContinuidadPDPPrueba{publicador: publicador, motivo: c.Motivo, ahora: ahora}
	fuente := &fuenteRolContinuidadNominal{registro, base.referencias, c.Instantanea.VersionRol, c.Instantanea.AsignacionPerfil}
	pdp, e := appvec.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registro, registro, registro, base.reloj, seg.GeneradorReferenciasCriptograficas{}, appvec.ConfiguracionServicioAutorizacion{})
	if e != nil {
		t.Fatal(e)
	}
	a := &autoridadContinuidadNominal{soporte: base.soporte, consultas: base.consultas, referencias: base.referencias, planes: base.planes, configuracion: c, accion: string(dom.AccionRegistrarAnotacionAdministrativa), finalidad: ct.FinalidadRegistrarAnotacionAdministrativa, tipo: ct.TipoRecursoAnotacionAdministrativa, fuente: fuente, pdp: pdp, reloj: base.reloj}
	ctx = context.WithValue(ctx, claveRutaContinuidadNominal{}, httpct.RutaAnotacionesAdministrativas)
	h, e := c.Instantanea.VersionRol.HuellaSHA256()
	if e != nil {
		t.Fatal(e)
	}
	datos.Accion = a.accion
	datos.Finalidad = a.finalidad
	datos.Recurso = core.RecursoAutorizable{Referencia: exp, ModuloID: ct.ModuloContratacion, Tipo: a.tipo, Ambitos: map[string]string{"organizacion_ref": a.referencias.OrganizacionRef, "expediente_ref": exp, "fase_previa": "seguimiento", "estado_previo": "en_curso"}, Atributos: map[string]string{"politica_ref": c.Instantanea.VersionRol.RolID, "politica_version": strconv.Itoa(c.Instantanea.VersionRol.Version), "politica_huella_sha256": h}}
	solicitud, e := core.NuevaSolicitudAutorizacionLigadaV3(datos)
	if e != nil {
		t.Fatal(e)
	}
	d, confirmacion, e := a.ExigirSolicitudLigadaV3(ctx, solicitud, base.soporte.contexto.Resultado)
	if e != nil {
		t.Fatal(e)
	}
	concedida, _, e := d.Resultado()
	if e != nil || !concedida || registro.concesiones != 1 {
		t.Fatal("permiso nominal no llegó a registro")
	}
	orden, e := vp.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, d, c.Motivo, base.soporte.contexto.Resultado)
	if e != nil || confirmacion.ValidarPara(orden) != nil {
		t.Fatal("confirmación no liga la solicitud nominal")
	}
	if strings.Join(publicador.orden, ",") != "preparar,publicar" {
		t.Fatal("publicación nominal ausente o repetida")
	}
}
