package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/vec/domain"
)

type catalogoPerfilesLotePrueba map[string]domain.RolAdministrable

func (c catalogoPerfilesLotePrueba) ResolverRolAdministrable(_ context.Context, ref string) (domain.RolAdministrable, error) {
	rol, ok := c[ref]
	if !ok {
		return domain.RolAdministrable{}, domain.ErrActoAdministracionPerfilesInvalido
	}
	return rol, nil
}

type autoridadPerfilesLotePrueba struct {
	lotes      int
	ordinarios int
	err        error
	mutar      func(*domain.ReciboLoteAdministracionPerfiles)
	replay     *domain.ReciboLoteAdministracionPerfiles
}

func (a *autoridadPerfilesLotePrueba) AplicarActoOrdinario(context.Context, domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error) {
	a.ordinarios++
	return domain.ReciboAdministracionPerfiles{}, errors.New("no debe ejecutarse")
}
func (*autoridadPerfilesLotePrueba) ProponerActoSensible(context.Context, domain.SolicitudActoAdministracionPerfiles) (domain.PropuestaAdministracionPerfiles, error) {
	return domain.PropuestaAdministracionPerfiles{}, errors.New("no debe ejecutarse")
}
func (*autoridadPerfilesLotePrueba) CerrarPropuestaSensible(context.Context, domain.SolicitudCierrePropuestaAdministracionPerfiles) (domain.CierrePropuestaAdministracionPerfiles, error) {
	return domain.CierrePropuestaAdministracionPerfiles{}, errors.New("no debe ejecutarse")
}
func (a *autoridadPerfilesLotePrueba) AplicarLoteOrdinario(_ context.Context, s domain.SolicitudLoteAdministracionPerfiles) (domain.ReciboLoteAdministracionPerfiles, error) {
	a.lotes++
	if a.err != nil {
		return domain.ReciboLoteAdministracionPerfiles{}, a.err
	}
	if a.replay != nil {
		return *a.replay, nil
	}
	r := domain.ReciboLoteAdministracionPerfiles{OperacionRef: s.OperacionRef, ActoRef: s.OperacionRef,
		ReciboRef: "recibo_admin:" + strings.Repeat("b", 32), AuditoriaRef: "auditoria:lote:1", HuellaSolicitudSHA256: s.HuellaSolicitudSHA256,
		FuentesSHA256: strings.Repeat("a", 64),
		ConfirmadoEn:  s.Actor.ResueltoEn}
	for _, c := range s.Cambios {
		inicioDesde := c.Objetivo.VigenteDesde
		if c.InicioVigencia == domain.InicioVigenciaLoteInmediato {
			inicioDesde = r.ConfirmadoEn
		}
		r.Inicios = append(r.Inicios, domain.InicioEfectivoLoteAdministracion{
			Modo: c.InicioVigencia, VigenteDesde: inicioDesde})
		r.Cambios = append(r.Cambios, domain.ReciboAdministracionPerfiles{OperacionRef: s.OperacionRef, ActoRef: r.ActoRef, ReciboRef: r.ReciboRef,
			AuditoriaRef: r.AuditoriaRef, ActorPersonaRef: s.Actor.PersonaRef, PerfilActivoRef: s.Actor.PerfilActivoRef,
			AsignacionPerfilRef: s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), CorrelacionRef: s.CorrelacionRef,
			Motivo: s.Motivo, ReferenciaActo: s.ReferenciaActo, ObjetivoPersonaRef: c.Objetivo.PersonaRef, PerfilRef: c.Objetivo.PerfilRef, VinculoRef: c.Objetivo.VinculoRef,
			UnidadRef: c.Objetivo.UnidadRef, CentroRef: c.Objetivo.CentroRef, RolVersionRef: c.RolVersionRef,
			VigenteDesde: inicioDesde, VigenteHasta: c.Objetivo.VigenteHasta,
			HuellaAntesSHA256: c.Objetivo.HuellaSHA256, HuellaDespuesSHA256: strings.Repeat("d", 64),
			VersionPosterior: 1, EstadoPosterior: domain.EstadoVinculoContextoActorActivo, ConfirmadoEn: r.ConfirmadoEn})
	}
	if a.mutar != nil {
		a.mutar(&r)
	}
	return r, nil
}

func lotePerfilesAplicacionPrueba(t *testing.T) (*ServicioAdministracionPerfiles, domain.SolicitudLoteAdministracionPerfiles, *autoridadPerfilesLotePrueba, catalogoPerfilesLotePrueba) {
	t.Helper()
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	e.instantanea.VersionRol.RolID = "administracion_perfiles"
	e.instantanea.AsignacionPerfil.VersionRolRef = e.instantanea.VersionRol.Referencia()
	e.instantanea.ControlVigenciaVersionRol.VersionRolRef = e.instantanea.VersionRol.Referencia()
	datos, err := e.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	s := domain.SolicitudLoteAdministracionPerfiles{OperacionRef: "acto_admin:" + strings.Repeat("a", 32), OrganizacionRef: "org_prueba", Actor: e.resultado.Contexto,
		Evidencia:               domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: e.resultado, Vinculo: datos.VinculoAutenticacionActor},
		InstantaneaAutorizacion: e.instantanea, Motivo: datos.ReferenciaMotivo, CorrelacionRef: "correlacion_" + strings.Repeat("e", 32)}
	for i, letra := range []string{"x", "y"} {
		s.Cambios = append(s.Cambios, domain.CambioPerfilAdministracion{Operacion: domain.OperacionOtorgarPerfil,
			InicioVigencia: domain.InicioVigenciaLoteProgramado,
			RolVersionRef:  []string{"rol:tramitacion_bolsa:v1", "rol:gestor_cronos:v2"}[i],
			Objetivo: domain.PreimagenAdministracionPerfiles{CuentaRef: "cta_" + strings.Repeat("b", 24), CuentaVersion: 2,
				PersonaRef: "per_" + strings.Repeat("d", 24), PersonaVersion: 3, PerfilRef: "prf_" + strings.Repeat(letra, 24), VinculoRef: "vca_" + strings.Repeat(letra, 24),
				UnidadRef: "unidad:prueba", HuellaSHA256: strings.Repeat("f", 64), ProcedenciaRef: "procedencia:prueba:1", ProcedenciaVersion: 1,
				ProcedenciaHuellaSHA256: strings.Repeat("e", 64), VigenteDesde: e.ahora.Add(time.Minute), VigenteHasta: e.ahora.Add(time.Hour)}})
	}
	sellarLotePerfilesPrueba(t, &s)
	huella, err := e.instantanea.VersionRol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	c := catalogoPerfilesLotePrueba{e.instantanea.VersionRol.Referencia(): {VersionRef: e.instantanea.VersionRol.Referencia(), Clase: domain.ClaseControlPerfilAdministrador, CategoriaAdmin: "aplicacion",
		HuellaSHA256: huella, VigenteDesde: e.ahora.Add(-time.Hour), VigenteHasta: e.ahora.Add(2 * time.Hour)}}
	for _, cambio := range s.Cambios {
		c[cambio.RolVersionRef] = domain.RolAdministrable{VersionRef: cambio.RolVersionRef, Clase: domain.ClaseControlPerfilOrdinario,
			HuellaSHA256: strings.Repeat("1", 64), VigenteDesde: e.ahora.Add(-time.Hour), VigenteHasta: e.ahora.Add(2 * time.Hour)}
	}
	a := &autoridadPerfilesLotePrueba{}
	servicio, err := NuevoServicioAdministracionPerfiles(c, a, &relojAutorizacionServicioPrueba{ahora: e.ahora})
	if err != nil {
		t.Fatal(err)
	}
	return servicio, s, a, c
}
func sellarLotePerfilesPrueba(t *testing.T, s *domain.SolicitudLoteAdministracionPerfiles) {
	t.Helper()
	_, h, err := s.CanonicoYHuella()
	if err != nil {
		t.Fatal(err)
	}
	s.HuellaSolicitudSHA256 = h
}
func TestAdministracionPerfilesLoteUsaUnaSolaOrden(t *testing.T) {
	servicio, s, a, _ := lotePerfilesAplicacionPrueba(t)
	r, err := servicio.AplicarLoteOrdinario(context.Background(), s)
	if err != nil || len(r.Cambios) != 2 || a.lotes != 1 || a.ordinarios != 0 {
		t.Fatalf("lote no indivisible: %v llamadas=%d/%d", err, a.lotes, a.ordinarios)
	}
}

func TestAdministracionPerfilesLoteInmediatoUsaInstantePrivadoDelRecibo(t *testing.T) {
	servicio, s, a, _ := lotePerfilesAplicacionPrueba(t)
	s.Cambios[0].InicioVigencia = domain.InicioVigenciaLoteInmediato
	s.Cambios[0].Objetivo.VigenteDesde = time.Time{}
	sellarLotePerfilesPrueba(t, &s)
	r, err := servicio.AplicarLoteOrdinario(context.Background(), s)
	if err != nil || a.lotes != 1 || r.Inicios[0].Modo != domain.InicioVigenciaLoteInmediato ||
		!r.Inicios[0].VigenteDesde.Equal(r.ConfirmadoEn) ||
		!r.Cambios[0].VigenteDesde.Equal(r.ConfirmadoEn) {
		t.Fatalf("inicio_inmediato_no_privado: %v", err)
	}
}
func TestAdministracionPerfilesLoteDeniegaAntesDelPuerto(t *testing.T) {
	for _, caso := range []string{"autoasignacion", "sistemas", "duplicado", "huella", "CAS", "sensible", "evidencia", "correlacion", "vigencia", "programado_pasado", "hasta_pasado"} {
		t.Run(caso, func(t *testing.T) {
			servicio, s, a, c := lotePerfilesAplicacionPrueba(t)
			switch caso {
			case "autoasignacion":
				for i := range s.Cambios {
					s.Cambios[i].Objetivo.PersonaRef = s.Actor.PersonaRef
				}
			case "sistemas":
				r := c[s.InstantaneaAutorizacion.VersionRol.Referencia()]
				r.CategoriaAdmin = "sistemas"
				c[r.VersionRef] = r
			case "duplicado":
				s.Cambios[1] = s.Cambios[0]
			case "huella":
				s.Cambios[1].Objetivo.UnidadRef = "unidad:otra"
			case "CAS":
				s.Cambios[1].Objetivo.PersonaVersion++
			case "sensible":
				r := c[s.Cambios[1].RolVersionRef]
				r.Clase = domain.ClaseControlPerfilAdministrador
				r.CategoriaAdmin = "aplicacion"
				c[r.VersionRef] = r
			case "evidencia":
				s.Evidencia = domain.EvidenciaSesionAdministracionPerfiles{}
			case "correlacion":
				s.CorrelacionRef = ""
			case "vigencia":
				s.Cambios[0].Objetivo.VigenteDesde = time.Time{}
			case "programado_pasado", "hasta_pasado":
				for i := range s.Cambios {
					if s.Cambios[i].Operacion != domain.OperacionOtorgarPerfil {
						continue
					}
					if caso == "programado_pasado" {
						// El fixture programa a ahora+1min y el rol rige desde ahora-1h:
						// ahora-1min está dentro del rol pero ya no es futuro.
						s.Cambios[i].InicioVigencia = domain.InicioVigenciaLoteProgramado
						s.Cambios[i].Objetivo.VigenteDesde = s.Cambios[i].Objetivo.VigenteDesde.Add(-2 * time.Minute)
					} else {
						s.Cambios[i].InicioVigencia = domain.InicioVigenciaLoteInmediato
						s.Cambios[i].Objetivo.VigenteDesde = time.Time{}
						s.Cambios[i].Objetivo.VigenteHasta = time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
					}
				}
				sellarLotePerfilesPrueba(t, &s)
			}
			_, err := servicio.AplicarLoteOrdinario(context.Background(), s)
			if err == nil || a.lotes != 0 || a.ordinarios != 0 {
				t.Fatalf("%s no denegado: %v", caso, err)
			}
		})
	}
}
func TestAdministracionPerfilesLoteFalloDurableNoDaRecibo(t *testing.T) {
	servicio, s, a, _ := lotePerfilesAplicacionPrueba(t)
	fallo := errors.New("rollback por CAS")
	a.err = fallo
	r, err := servicio.AplicarLoteOrdinario(context.Background(), s)
	if !errors.Is(err, fallo) || r.ReciboRef != "" || a.lotes != 1 || a.ordinarios != 0 {
		t.Fatal("fallo durable presentado como éxito")
	}
}
func TestAdministracionPerfilesLoteNoAceptaReciboAjeno(t *testing.T) {
	for _, caso := range []string{"actor", "asignacion", "correlacion", "correlacion_dispar", "preimagen", "perfil", "version", "fecha", "motivo", "parcial"} {
		t.Run(caso, func(t *testing.T) {
			servicio, s, a, _ := lotePerfilesAplicacionPrueba(t)
			a.mutar = func(r *domain.ReciboLoteAdministracionPerfiles) {
				switch caso {
				case "actor":
					r.Cambios[0].ActorPersonaRef = s.Cambios[0].Objetivo.PersonaRef
				case "asignacion":
					r.Cambios[0].AsignacionPerfilRef = "asignacion:otra:v1"
				case "correlacion":
					r.Cambios[0].CorrelacionRef = ""
				case "correlacion_dispar":
					r.Cambios[1].CorrelacionRef = "correlacion_" + strings.Repeat("9", 32)
				case "preimagen":
					r.Cambios[0].HuellaAntesSHA256 = strings.Repeat("0", 64)
				case "perfil":
					r.Cambios[0].PerfilRef = r.Cambios[1].PerfilRef
				case "version":
					r.Cambios[0].VersionPosterior = 2
				case "fecha":
					r.Cambios[0].ConfirmadoEn = r.ConfirmadoEn.Add(time.Second)
				case "motivo":
					r.Cambios[0].Motivo.EntradaClave = "otro"
				case "parcial":
					r.Cambios = r.Cambios[:1]
				}
			}
			r, err := servicio.AplicarLoteOrdinario(context.Background(), s)
			if err == nil || r.ReciboRef != "" || a.lotes != 1 || a.ordinarios != 0 {
				t.Fatalf("recibo %s aceptado", caso)
			}
		})
	}
}

func TestAdministracionPerfilesLoteReplayRevalidaAccesoYConservaRecibo(t *testing.T) {
	servicio, original, autoridad, _ := lotePerfilesAplicacionPrueba(t)
	recibo, err := servicio.AplicarLoteOrdinario(context.Background(), original)
	if err != nil {
		t.Fatal(err)
	}
	autoridad.replay = &recibo
	reintento := original
	reintento.CorrelacionRef = "correlacion_" + strings.Repeat("9", 32)
	reintento.InstantaneaAutorizacion.RevisionCatalogoPoliticas++
	datos, err := original.Evidencia.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	autenticacion := datos.Autenticacion()
	autenticacion.ControlSesionRevision++
	autenticacion.ControlSesionHuellaSHA256 = strings.Repeat("9", 64)
	autenticacion.SesionRevalidadaEn = autenticacion.SesionRevalidadaEn.Add(time.Minute)
	vinculo, resultado, err := domain.CrearVinculoAutenticacionActorV2ConResultado(context.Background(),
		&revalidadorVinculoAplicacionAdversarial{resultado: autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: datos.AutenticacionRef, SesionRef: datos.SesionRef},
		resolutorContextoAutorizacionV3Prueba{resultado: original.Evidencia.ResultadoContexto}, solicitudServicioContextoActorPrueba(), servicio.reloj)
	if err != nil || vinculo.CoincideExactamenteCon(original.Evidencia.Vinculo) {
		t.Fatalf("evidencia renovada: %v", err)
	}
	reintento.Evidencia = domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: vinculo}
	sellarLotePerfilesPrueba(t, &reintento)
	if reintento.HuellaSolicitudSHA256 != original.HuellaSolicitudSHA256 {
		t.Fatal("un nuevo acceso cambió la huella del efecto")
	}
	recuperado, err := servicio.AplicarLoteOrdinario(context.Background(), reintento)
	if err != nil || recuperado.ReciboRef != recibo.ReciboRef ||
		!recuperado.ConfirmadoEn.Equal(recibo.ConfirmadoEn) || recuperado.Cambios[0] != recibo.Cambios[0] ||
		autoridad.lotes != 2 || autoridad.ordinarios != 0 {
		t.Fatalf("replay alteró el recibo original: %v", err)
	}
	for _, caso := range []string{"evidencia", "asignacion_caducada", "rol_retirado"} {
		t.Run(caso, func(t *testing.T) {
			invalida := reintento
			switch caso {
			case "evidencia":
				invalida.Evidencia = domain.EvidenciaSesionAdministracionPerfiles{}
			case "asignacion_caducada":
				invalida.InstantaneaAutorizacion.AsignacionPerfil.VigenteHasta = servicio.reloj.Ahora()
			case "rol_retirado":
				invalida.InstantaneaAutorizacion.ControlVigenciaVersionRol.Estado = domain.EstadoControlVigenciaVersionRolRetirada
			}
			if _, err := servicio.AplicarLoteOrdinario(context.Background(), invalida); err == nil || autoridad.lotes != 2 {
				t.Fatal("replay aceptó acceso sin autoridad vigente")
			}
		})
	}
}

func TestAdministracionPerfilesLoteHuellaConservaTodoElMaterialDelEfecto(t *testing.T) {
	for _, caso := range []string{"destinataria", "CAS", "motivo", "ambito", "organizacion", "modo_inicio", "fecha_inicio", "orden", "asignacion"} {
		t.Run(caso, func(t *testing.T) {
			_, solicitud, _, _ := lotePerfilesAplicacionPrueba(t)
			original := solicitud.HuellaSolicitudSHA256
			switch caso {
			case "destinataria":
				for i := range solicitud.Cambios {
					solicitud.Cambios[i].Objetivo.PersonaRef = "per_" + strings.Repeat("z", 24)
				}
			case "CAS":
				for i := range solicitud.Cambios {
					solicitud.Cambios[i].Objetivo.PersonaVersion++
				}
			case "motivo":
				solicitud.Motivo.EntradaClave = "motivo_" + strings.Repeat("9", 32)
			case "ambito":
				for i := range solicitud.Cambios {
					solicitud.Cambios[i].Objetivo.UnidadRef = "unidad:otra"
				}
			case "organizacion":
				solicitud.OrganizacionRef = "org_otra"
			case "modo_inicio":
				solicitud.Cambios[0].InicioVigencia = domain.InicioVigenciaLoteInmediato
				solicitud.Cambios[0].Objetivo.VigenteDesde = time.Time{}
			case "fecha_inicio":
				solicitud.Cambios[0].Objetivo.VigenteDesde = solicitud.Cambios[0].Objetivo.VigenteDesde.Add(time.Second)
			case "orden":
				solicitud.Cambios[0], solicitud.Cambios[1] = solicitud.Cambios[1], solicitud.Cambios[0]
			case "asignacion":
				solicitud.InstantaneaAutorizacion.AsignacionPerfil.Version++
			}
			_, nueva, err := solicitud.CanonicoYHuella()
			if err != nil || nueva == original {
				t.Fatalf("material distinto reutilizó la huella: %v", err)
			}
		})
	}
}

func TestAdministracionPerfilesNoAdmiteOtroRolAunqueCatalogoLoClasifiqueAplicacion(t *testing.T) {
	servicio, solicitud, autoridad, catalogo := lotePerfilesAplicacionPrueba(t)
	original := solicitud.InstantaneaAutorizacion.VersionRol.Referencia()
	rol := catalogo[original]
	solicitud.InstantaneaAutorizacion.VersionRol.RolID = "administrador_aplicacion_prueba"
	versionRef := solicitud.InstantaneaAutorizacion.VersionRol.Referencia()
	solicitud.InstantaneaAutorizacion.AsignacionPerfil.VersionRolRef = versionRef
	solicitud.InstantaneaAutorizacion.ControlVigenciaVersionRol.VersionRolRef = versionRef
	rol.VersionRef = versionRef
	var err error
	rol.HuellaSHA256, err = solicitud.InstantaneaAutorizacion.VersionRol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	catalogo[versionRef] = rol
	sellarLotePerfilesPrueba(t, &solicitud)
	if solicitud.Validar() != nil || rol.ValidarEn(servicio.reloj.Ahora()) != nil {
		t.Fatal("fixture nominal alternativo inválido")
	}
	if _, err := servicio.AplicarLoteOrdinario(context.Background(), solicitud); err == nil || autoridad.lotes != 0 || autoridad.ordinarios != 0 {
		t.Fatal("otro rol de categoría Aplicación recibió autoridad nominal de perfiles")
	}
}

// Un recibo cuyo inicio efectivo inmediato no queda antes del fin de la
// vigencia no acredita el alta, aunque el resto coincida con la solicitud.
func TestAdministracionPerfilesLoteReciboConFinNoPosteriorAlInicio(t *testing.T) {
	servicio, s, a, _ := lotePerfilesAplicacionPrueba(t)
	for i := range s.Cambios {
		if s.Cambios[i].Operacion == domain.OperacionOtorgarPerfil {
			s.Cambios[i].InicioVigencia = domain.InicioVigenciaLoteInmediato
			s.Cambios[i].Objetivo.VigenteDesde = time.Time{}
		}
	}
	sellarLotePerfilesPrueba(t, &s)
	a.mutar = func(r *domain.ReciboLoteAdministracionPerfiles) {
		fin := s.Cambios[0].Objetivo.VigenteHasta
		r.ConfirmadoEn = fin
		for i := range r.Cambios {
			r.Cambios[i].ConfirmadoEn = fin
			if r.Inicios[i].Modo == domain.InicioVigenciaLoteInmediato {
				r.Inicios[i].VigenteDesde, r.Cambios[i].VigenteDesde = fin, fin
			}
		}
	}
	if _, err := servicio.AplicarLoteOrdinario(context.Background(), s); err == nil || a.lotes != 1 {
		t.Fatalf("recibo con fin no posterior al inicio aceptado: %v", err)
	}
}
