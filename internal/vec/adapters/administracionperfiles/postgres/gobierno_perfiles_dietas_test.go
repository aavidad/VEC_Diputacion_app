package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteGobiernoDietasPrueba struct {
	catalogo domain.CatalogoAccionesAdministracionV1
	llamadas *int
}

func (f fuenteGobiernoDietasPrueba) ObtenerCatalogoAccionesAdministracionV1(
	_ context.Context, ref string, version int, huella string,
) (domain.CatalogoAccionesAdministracionV1, error) {
	if f.llamadas != nil {
		*f.llamadas++
	}
	h, err := f.catalogo.HuellaSHA256()
	if err != nil || ref != f.catalogo.Referencia || version != f.catalogo.Version || huella != h {
		return domain.CatalogoAccionesAdministracionV1{}, domain.ErrOrigenPerfilAdministracionNoCoincide
	}
	return f.catalogo, nil
}

func escenarioGobiernoDietasAdaptador(t *testing.T, rolID string) (
	domain.CatalogoAccionesAdministracionV1, domain.OrdenPropuestaGobiernoPerfil,
) {
	t.Helper()
	canon, _ := catalogoAccionesPublicadoPrueba(t)
	var c domain.CatalogoAccionesAdministracionV1
	if err := json.Unmarshal(canon, &c); err != nil {
		t.Fatal(err)
	}
	type permiso struct {
		accion, modulo, recurso, finalidad string
		campos, dimensiones                []string
	}
	comunes := []string{"comision.calculo", "comision.codigos_ruta", "comision.estado", "comision.fecha_fin",
		"comision.fecha_inicio", "comision.motivo", "comision.referencia", "comision.relacion_ref"}
	permisos := []permiso{
		{"dietas.ruta.catalogo.consultar", "dietas", "catalogo_rutas_dietas", "consultar_itinerario_dietas", nil, []string{"ambito_ref"}},
		{"dietas.ruta.calculo.solicitar", "dietas", "calculo_rutas_dietas", "consultar_itinerario_dietas", nil, []string{"ambito_ref"}},
	}
	if rolID == domain.RolIDPersonaDietasComisiones {
		permisos = []permiso{
			{"personal.relacion.propia.consultar_dietas", "personal", "relacion_empleado_dietas", "preparar_borrador_dietas",
				[]string{"desde", "empleado_ref", "estado", "fuente_ref", "fuente_version", "hasta", "persona_ref", "procedencia_acto_ref", "relacion_ref", "unidad_ref", "version"}, []string{"empleado_ref", "persona_ref"}},
			{"dietas.borrador.propio.crear", "dietas", "comision_borrador", "crear_borrador_propio",
				append(append([]string(nil), comunes...), "recibo.referencia", "recibo.registrado_en", "recibo.repeticion", "recibo.version"), []string{"empleado_ref", "persona_ref"}},
			{"dietas.borrador.propio.consultar", "dietas", "comision_borrador", "consultar_borrador_propio",
				append(append([]string(nil), comunes...), "items.comision.calculo", "items.comision.codigos_ruta", "items.comision.estado",
					"items.comision.fecha_fin", "items.comision.fecha_inicio", "items.comision.motivo", "items.comision.referencia",
					"items.comision.relacion_ref", "items.recibo.referencia", "items.recibo.registrado_en", "items.recibo.repeticion",
					"items.recibo.version", "recibo.referencia", "recibo.registrado_en", "recibo.repeticion", "recibo.version", "siguiente_cursor"),
				[]string{"empleado_ref", "persona_ref"}},
		}
	}
	c.Entradas = make([]domain.EntradaAccionAdministracionV1, len(permisos))
	selecciones := make([]domain.SeleccionAccionAdministracionV1, len(permisos))
	concesiones := make([]domain.ConcesionRol, len(permisos))
	for i, p := range permisos {
		e := domain.EntradaAccionAdministracionV1{
			Referencia: "entrada:dietas:" + p.accion, Version: 1,
			FuenteRef: "fuente:dietas", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("b", 64),
			Concesion: domain.ConcesionRol{Accion: p.accion, ModuloID: p.modulo, TipoRecurso: p.recurso,
				Finalidades: []string{p.finalidad}, GarantiaMinima: domain.AuthAssuranceHigh, CamposPermitidos: p.campos},
			DimensionesAmbito: p.dimensiones, ClaseControl: string(domain.ClaseControlPerfilOrdinario),
			VigenteDesde: c.VigenteDesde, VigenteHasta: c.VigenteHasta,
		}
		c.Entradas[i], concesiones[i] = e, e.Concesion
		h, err := e.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		selecciones[i] = domain.SeleccionAccionAdministracionV1{EntradaRef: e.Referencia,
			EntradaVersion: e.Version, EntradaHuellaSHA256: h}
	}
	h, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	base, _, _, _ := contratoV2Prueba(t)
	base.InstantaneaAutorizacion.AsignacionPerfil.Ambitos = []domain.AmbitoPerfil{
		{Clave: "organizacion_ref", Valores: []string{"organizacion:prueba"}},
		{Clave: "unidad_ref", Valores: []string{"unidad:prueba"}},
	}
	motivo := domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("c", 64), EntradaClave: "motivo_" + strings.Repeat("a", 32)}
	intencion := domain.SolicitudPlanGobiernoPerfil{Operacion: domain.OperacionCrearPerfilGobernado,
		Publicacion: &domain.PropuestaPerfilAdministracionV1{CatalogoRef: c.Referencia, CatalogoVersion: c.Version,
			CatalogoHuellaSHA256: h, RolPropuesto: domain.VersionRol{RolID: rolID, Version: 1, Estado: domain.EstadoVersionRolPublicada,
				Nombre: "Dietas", Concesiones: concesiones}, Selecciones: selecciones},
		Motivo: motivo, ReferenciaActo: "resolucion:dietas:prueba"}
	plan, _, err := domain.PrepararPlanGobiernoRolNuevoDesdeCatalogo(c, intencion,
		c.VigenteDesde.Add(time.Hour), base.Actor.PersonaRef)
	if err != nil {
		pub := *intencion.Publicacion
		pub.RolPropuesto.PublicadaPor = base.Actor.PersonaRef
		pub.RolPropuesto.PublicadaEn = c.VigenteDesde.Add(time.Hour)
		_, comprobacion := domain.ComprobarPropuestaPerfilAdministracionV1(c, pub, c.VigenteDesde.Add(time.Hour))
		t.Fatalf("plan nominal: %v; catálogo=%v; rol=%v; comprobación=%v", err, c.Validar(), pub.RolPropuesto.Validar(), comprobacion)
	}
	hPlan, err := plan.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	s := domain.SolicitudPropuestaGobiernoPerfil{OperacionRef: "propuesta_admin:" + strings.Repeat("1", 32),
		Actor: base.Actor, Evidencia: base.Evidencia, InstantaneaAutorizacion: base.InstantaneaAutorizacion,
		Intencion: intencion, HuellaPlanEsperada: hPlan, CorrelacionRef: "correlacion_" + strings.Repeat("5", 32)}
	m := domain.MaterialPropuestaGobiernoPerfil{OperacionRef: s.OperacionRef,
		ProponentePersonaRef: s.Actor.PersonaRef, PerfilActivoRef: s.Actor.PerfilActivoRef,
		AsignacionPerfilRef: s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), Plan: plan}
	o := domain.OrdenPropuestaGobiernoPerfil{Solicitud: s, Material: m}
	if err := o.Validar(); err != nil {
		t.Fatalf("orden nominal: %v", err)
	}
	return c, o
}

func TestGobiernoRolDietasAdaptadorAdmitePlanesExactosYRechazaCompuestosAjenos(t *testing.T) {
	for _, rolID := range []string{domain.RolIDPersonaDietasComisiones, domain.RolIDPersonaDietasRutas} {
		t.Run(rolID, func(t *testing.T) {
			c, orden := escenarioGobiernoDietasAdaptador(t, rolID)
			autoridad := &AutoridadGobiernoRolNuevo{pool: &poolGobiernoReferenciaPrueba{},
				fuente: fuenteGobiernoDietasPrueba{catalogo: c}, emisor: emisorGobiernoReferenciaPrueba{t: t},
				reloj: relojFijo(c.VigenteDesde.Add(time.Hour))}
			if _, err := autoridad.ResolverCatalogoGobiernoPerfil(context.Background(), orden.Solicitud); err != nil {
				t.Fatalf("resolución del catálogo: %v", err)
			}
			e, err := materialPropuestaGobiernoRol(orden)
			if err != nil {
				t.Fatalf("material de propuesta: %v", err)
			}
			r, err := RecursoGobiernoRolNuevo(e, orden.Solicitud.InstantaneaAutorizacion.AsignacionPerfil)
			if err != nil || r.Referencia != orden.Material.Plan.VersionRolObjetivoRef {
				t.Fatalf("recurso propuesto: %+v, %v", r, err)
			}
			for _, mutar := range []struct {
				nombre  string
				cambiar func(*domain.PlanGobiernoPerfil)
			}{
				{"rol compuesto ajeno", func(p *domain.PlanGobiernoPerfil) {
					p.DefinicionNueva.RolID = "otro_rol"
					p.VersionRolObjetivoRef = "rol:otro_rol:v1"
				}},
				{"concesión cambiada", func(p *domain.PlanGobiernoPerfil) {
					p.DefinicionNueva.Concesiones[0].Finalidades = []string{"finalidad_ajena"}
				}},
				{"concesión de otro módulo", func(p *domain.PlanGobiernoPerfil) {
					p.DefinicionNueva.Concesiones[0].ModuloID = "bolsa"
				}},
				{"Dietas con una sola concesión", func(p *domain.PlanGobiernoPerfil) {
					p.DefinicionNueva.Concesiones = p.DefinicionNueva.Concesiones[:1]
					p.Selecciones = p.Selecciones[:1]
				}},
			} {
				t.Run(mutar.nombre, func(t *testing.T) {
					otro := orden
					otro.Material.Plan = orden.Material.Plan.Copia()
					mutar.cambiar(&otro.Material.Plan)
					h, err := otro.Material.Plan.HuellaSHA256()
					if err != nil {
						t.Fatal(err)
					}
					otro.Solicitud.HuellaPlanEsperada = h
					if _, err := materialPropuestaGobiernoRol(otro); !errors.Is(err, domain.ErrPlanGobiernoPerfilInvalido) {
						t.Fatalf("material ajeno llegó al emisor: %v", err)
					}
					materialCanon, err := json.Marshal(otro.Material)
					if err != nil {
						t.Fatal(err)
					}
					materialSHA, err := otro.Material.HuellaSHA256()
					if err != nil {
						t.Fatal(err)
					}
					b, err := json.Marshal(propuestaGobiernoRolEnvelope{Esquema: "administracion_gobierno_rol_nuevo_propuesta_v1",
						MaterialCanon: string(materialCanon), MaterialSHA256: materialSHA, PlanSHA256: h,
						CorrelacionRef: e.CorrelacionAccesoRef})
					if err != nil {
						t.Fatal(err)
					}
					eAjeno := e
					eAjeno.Material = b
					eAjeno.Referencia = otro.Material.Plan.VersionRolObjetivoRef
					if _, err := RecursoGobiernoRolNuevo(eAjeno, orden.Solicitud.InstantaneaAutorizacion.AsignacionPerfil); !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
						t.Fatalf("recurso ajeno llegó a la decisión: %v", err)
					}
				})
			}
		})
	}
}

// Ataque en la entrada del adaptador: un rol compuesto ajeno a Dietas no llega
// a leer el catálogo, y un rol de Dietas con una concesión de otro módulo,
// aunque el catálogo la publique con huellas coherentes, no obtiene plan.
func TestGobiernoRolDietasAdaptadorResolverRechazaIntencionesAjenas(t *testing.T) {
	for _, rolID := range []string{domain.RolIDPersonaDietasComisiones, domain.RolIDPersonaDietasRutas} {
		t.Run(rolID+"/rol_compuesto_ajeno", func(t *testing.T) {
			c, orden := escenarioGobiernoDietasAdaptador(t, rolID)
			llamadas := 0
			autoridad := &AutoridadGobiernoRolNuevo{pool: &poolGobiernoReferenciaPrueba{},
				fuente: fuenteGobiernoDietasPrueba{catalogo: c, llamadas: &llamadas}, emisor: emisorGobiernoReferenciaPrueba{t: t},
				reloj: relojFijo(c.VigenteDesde.Add(time.Hour))}
			s := orden.Solicitud
			pub := *s.Intencion.Publicacion
			pub.RolPropuesto.RolID = "otro_rol"
			s.Intencion.Publicacion = &pub
			if _, err := autoridad.ResolverCatalogoGobiernoPerfil(context.Background(), s); !errors.Is(err, domain.ErrPlanGobiernoPerfilInvalido) || llamadas != 0 {
				t.Fatalf("rol compuesto ajeno: error=%v lecturas de catálogo=%d", err, llamadas)
			}
		})
		t.Run(rolID+"/concesion_de_otro_modulo", func(t *testing.T) {
			c, orden := escenarioGobiernoDietasAdaptador(t, rolID)
			c.Entradas[0].Concesion.ModuloID = "bolsa"
			s := orden.Solicitud
			pub := *s.Intencion.Publicacion
			pub.RolPropuesto.Concesiones = append([]domain.ConcesionRol(nil), pub.RolPropuesto.Concesiones...)
			pub.RolPropuesto.Concesiones[0] = c.Entradas[0].Concesion
			pub.Selecciones = append([]domain.SeleccionAccionAdministracionV1(nil), pub.Selecciones...)
			he, err := c.Entradas[0].HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			pub.Selecciones[0].EntradaHuellaSHA256 = he
			hc, err := c.HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			pub.CatalogoHuellaSHA256 = hc
			s.Intencion.Publicacion = &pub
			llamadas := 0
			autoridad := &AutoridadGobiernoRolNuevo{pool: &poolGobiernoReferenciaPrueba{},
				fuente: fuenteGobiernoDietasPrueba{catalogo: c, llamadas: &llamadas}, emisor: emisorGobiernoReferenciaPrueba{t: t},
				reloj: relojFijo(c.VigenteDesde.Add(time.Hour))}
			if _, err := autoridad.ResolverCatalogoGobiernoPerfil(context.Background(), s); !errors.Is(err, domain.ErrPlanGobiernoPerfilInvalido) || llamadas != 1 {
				t.Fatalf("concesión de otro módulo: error=%v lecturas de catálogo=%d", err, llamadas)
			}
		})
	}
}
