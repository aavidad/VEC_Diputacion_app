package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// Doble privado. No implementa publicación PostgreSQL ni autoridad real.
type autoridadGobiernoPerfilPrueba struct {
	*autoridadPerfilesLotePrueba
	catalogo                      domain.CatalogoAccionesAdministracionV1
	ahora                         time.Time
	lecturas, propuestas, cierres int
	cierre                        domain.CierreGobiernoPerfil
}

func (a *autoridadGobiernoPerfilPrueba) ResolverCatalogoGobiernoPerfil(context.Context, domain.SolicitudPropuestaGobiernoPerfil) (domain.CatalogoAccionesAdministracionV1, error) {
	a.lecturas++
	return a.catalogo, nil
}
func (a *autoridadGobiernoPerfilPrueba) ProponerGobiernoPerfil(_ context.Context, o domain.OrdenPropuestaGobiernoPerfil) (domain.PropuestaGobiernoPerfil, error) {
	a.propuestas++
	h, err := o.Material.HuellaSHA256()
	if err != nil {
		return domain.PropuestaGobiernoPerfil{}, err
	}
	return domain.PropuestaGobiernoPerfil{Material: o.Material, HuellaSHA256: h, CaducaEn: a.ahora.Add(time.Hour)}, nil
}
func (a *autoridadGobiernoPerfilPrueba) CerrarGobiernoPerfil(context.Context, domain.SolicitudCierreGobiernoPerfil) (domain.CierreGobiernoPerfil, error) {
	a.cierres++
	return a.cierre, nil
}

func gobiernoPerfilAplicacionPrueba(t *testing.T) (*ServicioAdministracionPerfiles, domain.SolicitudPropuestaGobiernoPerfil, *autoridadGobiernoPerfilPrueba, catalogoPerfilesLotePrueba) {
	t.Helper()
	servicio, lote, anterior, catalogoAdmin := lotePerfilesAplicacionPrueba(t)
	fecha := servicio.reloj.Ahora()
	concesion := domain.ConcesionRol{Accion: "sintetico.leer", ModuloID: "sintetico", TipoRecurso: "expediente", Finalidades: []string{"revision"},
		GarantiaMinima: domain.AuthAssuranceHigh, CamposPermitidos: []string{"estado"}, Obligaciones: []string{"auditar"}}
	c := domain.CatalogoAccionesAdministracionV1{Referencia: "catalogo:sintetico", Version: 1, FuenteRef: "fuente:sintetica", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("a", 64),
		VigenteDesde: fecha.Add(-time.Hour), VigenteHasta: fecha.Add(4 * time.Hour),
		Entradas: []domain.EntradaAccionAdministracionV1{{Referencia: "entrada:sintetica", Version: 1, FuenteRef: "fuente:modulo", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("b", 64),
			Concesion: concesion, DimensionesAmbito: []string{"unidad"}, ClaseControl: "consulta_auditada", VigenteDesde: fecha.Add(-time.Hour), VigenteHasta: fecha.Add(4 * time.Hour)}},
		Perfiles: []domain.PerfilPublicadoAdministracionV1{{Rol: lote.InstantaneaAutorizacion.VersionRol,
			ControlVigencia: lote.InstantaneaAutorizacion.ControlVigenciaVersionRol, TipoPerfil: domain.TipoPerfilAdministracionFijoSistemaV1}}}
	hc, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	he, err := c.Entradas[0].HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	pub := domain.PropuestaPerfilAdministracionV1{CatalogoRef: c.Referencia, CatalogoVersion: 1, CatalogoHuellaSHA256: hc,
		RolPropuesto: domain.VersionRol{RolID: "nuevo_sintetico", Version: 1, Nombre: "nuevo_sintetico", Estado: domain.EstadoVersionRolPublicada,
			Concesiones: []domain.ConcesionRol{concesion}, PublicadaPor: "actor:archivo", PublicadaEn: fecha},
		Selecciones: []domain.SeleccionAccionAdministracionV1{{EntradaRef: c.Entradas[0].Referencia, EntradaVersion: 1, EntradaHuellaSHA256: he}}}
	intencion := domain.SolicitudPlanGobiernoPerfil{Operacion: domain.OperacionCrearPerfilGobernado, Publicacion: &pub, Motivo: lote.Motivo}
	plan, err := PrepararPlanGobiernoPerfilOffline(c, intencion, fecha)
	if err != nil {
		t.Fatal(err)
	}
	h, err := plan.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	a := &autoridadGobiernoPerfilPrueba{autoridadPerfilesLotePrueba: anterior, catalogo: c, ahora: fecha}
	servicio.actos = a
	s := domain.SolicitudPropuestaGobiernoPerfil{OperacionRef: "propuesta_admin:" + strings.Repeat("a", 32),
		Actor: lote.Actor, Evidencia: lote.Evidencia, InstantaneaAutorizacion: lote.InstantaneaAutorizacion,
		Intencion: intencion, HuellaPlanEsperada: h, CorrelacionRef: lote.CorrelacionRef}
	return servicio, s, a, catalogoAdmin
}

func TestGobiernoPerfilNominalUsaCatalogoCentralYDeniegaSinCapacidad(t *testing.T) {
	servicio, s, a, _ := gobiernoPerfilAplicacionPrueba(t)
	p, err := servicio.ProponerGobiernoPerfil(context.Background(), s)
	if err != nil || p.Material.Plan.DefinicionNueva == nil || a.lecturas != 1 || a.propuestas != 1 || a.ordinarios != 0 {
		t.Fatalf("propuesta nominal: %v", err)
	}
	servicio.actos = a.autoridadPerfilesLotePrueba
	if p, err := servicio.ProponerGobiernoPerfil(context.Background(), s); err == nil || p.HuellaSHA256 != "" || a.ordinarios != 0 {
		t.Fatal("autoridad anterior publica por fallback")
	}
}

func TestGobiernoPerfilNominalDeniegaDatosDePermisoYFuenteDivergente(t *testing.T) {
	for _, caso := range []string{"sistemas", "evidencia", "material", "catalogo", "huella"} {
		t.Run(caso, func(t *testing.T) {
			servicio, s, a, catalogoAdmin := gobiernoPerfilAplicacionPrueba(t)
			switch caso {
			case "sistemas":
				r := catalogoAdmin[s.InstantaneaAutorizacion.VersionRol.Referencia()]
				r.CategoriaAdmin = "sistemas"
				catalogoAdmin[r.VersionRef] = r
			case "evidencia":
				s.Evidencia = domain.EvidenciaSesionAdministracionPerfiles{}
			case "material":
				s.Intencion.Publicacion.RolPropuesto.Nombre = "cambiado"
			case "catalogo":
				a.catalogo.FuenteVersion++
			case "huella":
				s.HuellaPlanEsperada = strings.Repeat("f", 64)
			}
			if p, err := servicio.ProponerGobiernoPerfil(context.Background(), s); err == nil || p.HuellaSHA256 != "" || a.propuestas != 0 {
				t.Fatal("datos/fuente alterados alcanzaron propuesta")
			}
		})
	}
}

func cierreGobiernoPerfilAplicacionPrueba(t *testing.T) (*ServicioAdministracionPerfiles, domain.SolicitudCierreGobiernoPerfil, *autoridadGobiernoPerfilPrueba) {
	t.Helper()
	servicio, propuesta, a, _ := gobiernoPerfilAplicacionPrueba(t)
	plan, err := PrepararPlanGobiernoPerfilOffline(a.catalogo, propuesta.Intencion, a.ahora)
	if err != nil {
		t.Fatal(err)
	}
	m := domain.MaterialPropuestaGobiernoPerfil{OperacionRef: propuesta.OperacionRef,
		ProponentePersonaRef: "per_" + strings.Repeat("z", 24), PerfilActivoRef: "prf_" + strings.Repeat("z", 24),
		AsignacionPerfilRef: "asignacion:proponente_sintetico:v1", Plan: plan}
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	s := domain.SolicitudCierreGobiernoPerfil{OperacionRef: "cierre_admin:" + strings.Repeat("b", 32), PropuestaRef: m.OperacionRef, PropuestaHuellaSHA256: h,
		ProponentePersonaRef: m.ProponentePersonaRef, VersionRolObjetivoRef: plan.VersionRolObjetivoRef,
		Aprobador: propuesta.Actor, Evidencia: propuesta.Evidencia, InstantaneaAutorizacion: propuesta.InstantaneaAutorizacion,
		Decision: domain.DecisionAprobarPropuestaPerfil, Motivo: propuesta.Intencion.Motivo, CorrelacionRef: propuesta.CorrelacionRef}
	d := plan.DefinicionNueva
	rol := domain.VersionRol{RolID: d.RolID, Version: d.Version, Nombre: d.Nombre, Concesiones: d.Concesiones,
		Estado: domain.EstadoVersionRolPublicada, PublicadaPor: s.Aprobador.PersonaRef, PublicadaEn: a.ahora}
	a.cierre = domain.CierreGobiernoPerfil{OperacionRef: s.OperacionRef, Material: m, PropuestaHuellaSHA256: h, Decision: s.Decision,
		ConfirmadoEn: a.ahora, AuditoriaAccesoRef: "auditoria:acceso:actual", Recibo: &domain.ReciboGobiernoPerfil{
			ActoRef: "acto_admin:" + strings.Repeat("c", 32), ReciboRef: "recibo_admin:" + strings.Repeat("d", 32),
			ActorPersonaRef: s.Aprobador.PersonaRef, PerfilActivoRef: s.Aprobador.PerfilActivoRef,
			AsignacionPerfilRef: s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), CorrelacionRef: s.CorrelacionRef,
			Motivo: s.Motivo, AuditoriaRef: "auditoria:efecto:original", VersionRol: rol,
			ControlPosterior: domain.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: domain.EstadoControlVigenciaVersionRolHabilitada,
				ActualizadoPor: s.Aprobador.PersonaRef, ActualizadoEn: a.ahora}}}
	return servicio, s, a
}

func TestGobiernoPerfilCierrePublicaMetadatosNominalesYReplayNoLosReescribe(t *testing.T) {
	servicio, s, a := cierreGobiernoPerfilAplicacionPrueba(t)
	c, err := servicio.CerrarGobiernoPerfil(context.Background(), s)
	if err != nil || c.Recibo == nil || c.Recibo.VersionRol.PublicadaPor != s.Aprobador.PersonaRef {
		t.Fatalf("cierre: %v", err)
	}
	s.CorrelacionRef = "correlacion_" + strings.Repeat("f", 32)
	s.InstantaneaAutorizacion.RevisionCatalogoPoliticas++
	r, err := servicio.CerrarGobiernoPerfil(context.Background(), s)
	if err != nil || r.Recibo.CorrelacionRef != c.Recibo.CorrelacionRef || r.Recibo.ReciboRef != c.Recibo.ReciboRef || a.cierres != 2 {
		t.Fatal("replay altera recibo/autor/correlación originales")
	}
}

func TestGobiernoPerfilCierreRechazaReciboAjenoYFaltaDobleControl(t *testing.T) {
	for _, caso := range []string{"misma_persona", "recurso", "asignacion", "autor_archivo", "fecha_archivo", "control", "concesion", "evidencia", "rechazo_con_recibo"} {
		t.Run(caso, func(t *testing.T) {
			servicio, s, a := cierreGobiernoPerfilAplicacionPrueba(t)
			switch caso {
			case "misma_persona":
				s.ProponentePersonaRef = s.Aprobador.PersonaRef
			case "recurso":
				s.VersionRolObjetivoRef = "rol:otro:v1"
			case "asignacion":
				a.cierre.Recibo.AsignacionPerfilRef = "asignacion:otra:v1"
			case "autor_archivo":
				a.cierre.Recibo.VersionRol.PublicadaPor = "actor:archivo"
			case "fecha_archivo":
				a.cierre.Recibo.VersionRol.PublicadaEn = a.ahora.Add(-time.Second)
			case "control":
				a.cierre.Recibo.ControlPosterior.Revision++
			case "concesion":
				a.cierre.Recibo.VersionRol.Concesiones[0].Accion = "sintetico.modificar"
			case "evidencia":
				s.Evidencia = domain.EvidenciaSesionAdministracionPerfiles{}
			case "rechazo_con_recibo":
				s.Decision, a.cierre.Decision = domain.DecisionRechazarPropuestaPerfil, domain.DecisionRechazarPropuestaPerfil
			}
			if c, err := servicio.CerrarGobiernoPerfil(context.Background(), s); err == nil || c.Recibo != nil {
				t.Fatal("recibo ajeno o cierre no autorizado aceptado")
			}
		})
	}
}

func TestGobiernoPerfilRetiradaNominalConservaVersionRolYAvanzaSoloControl(t *testing.T) {
	servicio, s, a := cierreGobiernoPerfilAplicacionPrueba(t)
	r := a.cierre.Recibo
	base := domain.PerfilPublicadoAdministracionV1{Rol: r.VersionRol, ControlVigencia: r.ControlPosterior, TipoPerfil: domain.TipoPerfilAdministracionAdministrableV1}
	a.cierre.Material.Plan.Operacion = domain.OperacionDeshabilitarVersionPerfil
	a.cierre.Material.Plan.Base, a.cierre.Material.Plan.DefinicionNueva, a.cierre.Material.Plan.Selecciones = &base, nil, nil
	h, err := a.cierre.Material.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	s.PropuestaHuellaSHA256, a.cierre.PropuestaHuellaSHA256 = h, h
	r.ControlPosterior.Revision, r.ControlPosterior.Estado = base.ControlVigencia.Revision+1, domain.EstadoControlVigenciaVersionRolRetirada
	r.ControlPosterior.ActoRef, r.ControlPosterior.MotivoCodigo = r.ActoRef, s.Motivo.EntradaClave
	cierre, err := servicio.CerrarGobiernoPerfil(context.Background(), s)
	if err != nil || cierre.Recibo.ControlPosterior.Revision != 2 || base.ControlVigencia.Estado != domain.EstadoControlVigenciaVersionRolHabilitada {
		t.Fatalf("retirada altera base o no avanza control: %v", err)
	}
	r.ControlPosterior.MotivoCodigo = "otro_codigo"
	if c, err := servicio.CerrarGobiernoPerfil(context.Background(), s); err == nil || c.Recibo != nil {
		t.Fatal("control de retirada conserva motivo ajeno")
	}
	r.ControlPosterior.MotivoCodigo = s.Motivo.EntradaClave
	// El documento del rol anterior no puede reescribirse durante la retirada.
	r.VersionRol.Nombre = "otra_definicion"
	if c, err := servicio.CerrarGobiernoPerfil(context.Background(), s); err == nil || c.Recibo != nil {
		t.Fatal("retirada reescribe VersionRol")
	}
}

func TestGobiernoPerfilRechazaConfirmacionAnteriorALaPreimagen(t *testing.T) {
	for _, operacion := range []domain.OperacionGobiernoPerfil{domain.OperacionVersionarPerfilGobernado, domain.OperacionDeshabilitarVersionPerfil} {
		t.Run(string(operacion), func(t *testing.T) {
			servicio, s, a := cierreGobiernoPerfilAplicacionPrueba(t)
			r := a.cierre.Recibo
			base := domain.PerfilPublicadoAdministracionV1{Rol: r.VersionRol, ControlVigencia: r.ControlPosterior, TipoPerfil: domain.TipoPerfilAdministracionAdministrableV1}
			a.cierre.Material.Plan.Operacion, a.cierre.Material.Plan.Base = operacion, &base
			if operacion == domain.OperacionVersionarPerfilGobernado {
				a.cierre.Material.Plan.DefinicionNueva.Version = 2
				r.VersionRol.Version = 2
				r.ControlPosterior.VersionRolRef = r.VersionRol.Referencia()
				a.cierre.Material.Plan.VersionRolObjetivoRef = r.VersionRol.Referencia()
				s.VersionRolObjetivoRef = r.VersionRol.Referencia()
			} else {
				a.cierre.Material.Plan.DefinicionNueva, a.cierre.Material.Plan.Selecciones = nil, nil
				r.ControlPosterior.Revision, r.ControlPosterior.Estado = 2, domain.EstadoControlVigenciaVersionRolRetirada
				r.ControlPosterior.ActoRef, r.ControlPosterior.MotivoCodigo = r.ActoRef, s.Motivo.EntradaClave
			}
			h, err := a.cierre.Material.HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			s.PropuestaHuellaSHA256, a.cierre.PropuestaHuellaSHA256 = h, h
			if c, err := servicio.CerrarGobiernoPerfil(context.Background(), s); err != nil || c.Recibo == nil {
				t.Fatalf("cierre correcto rechazado: %v", err)
			}
			s.CorrelacionRef = "correlacion_" + strings.Repeat("f", 32)
			if c, err := servicio.CerrarGobiernoPerfil(context.Background(), s); err != nil || c.Recibo.CorrelacionRef == s.CorrelacionRef {
				t.Fatal("replay no conserva la correlación original")
			}
			a.cierre.ConfirmadoEn = a.ahora.Add(-time.Microsecond)
			r.ControlPosterior.ActualizadoEn = a.cierre.ConfirmadoEn
			if operacion == domain.OperacionVersionarPerfilGobernado {
				r.VersionRol.PublicadaEn = a.cierre.ConfirmadoEn
			}
			if c, err := servicio.CerrarGobiernoPerfil(context.Background(), s); err == nil || c.Recibo != nil {
				t.Fatal("recibo retrocede respecto a su preimagen")
			}
		})
	}
}
