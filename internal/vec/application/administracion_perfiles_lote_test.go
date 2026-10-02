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
	r := domain.ReciboLoteAdministracionPerfiles{OperacionRef: s.OperacionRef, ActoRef: s.OperacionRef,
		ReciboRef: "recibo_admin:" + strings.Repeat("b", 32), AuditoriaRef: "auditoria:lote:1", HuellaSolicitudSHA256: s.HuellaSolicitudSHA256,
		ConfirmadoEn: s.Cambios[0].Objetivo.VigenteDesde}
	for _, c := range s.Cambios {
		r.Cambios = append(r.Cambios, domain.ReciboAdministracionPerfiles{OperacionRef: s.OperacionRef, ActoRef: r.ActoRef, ReciboRef: r.ReciboRef,
			AuditoriaRef: r.AuditoriaRef, ActorPersonaRef: s.Actor.PersonaRef, PerfilActivoRef: s.Actor.PerfilActivoRef,
			AsignacionPerfilRef: s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), CorrelacionRef: s.CorrelacionRef,
			Motivo: s.Motivo, ReferenciaActo: s.ReferenciaActo, ObjetivoPersonaRef: c.Objetivo.PersonaRef, PerfilRef: c.Objetivo.PerfilRef, VinculoRef: c.Objetivo.VinculoRef,
			UnidadRef: c.Objetivo.UnidadRef, CentroRef: c.Objetivo.CentroRef, RolVersionRef: c.RolVersionRef,
			VigenteDesde: c.Objetivo.VigenteDesde, VigenteHasta: c.Objetivo.VigenteHasta,
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
	datos, err := e.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	s := domain.SolicitudLoteAdministracionPerfiles{OperacionRef: "acto_admin:" + strings.Repeat("a", 32), Actor: e.resultado.Contexto,
		Evidencia:               domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: e.resultado, Vinculo: datos.VinculoAutenticacionActor},
		InstantaneaAutorizacion: e.instantanea, Motivo: datos.ReferenciaMotivo, CorrelacionRef: "correlacion_" + strings.Repeat("e", 32)}
	for i, letra := range []string{"x", "y"} {
		s.Cambios = append(s.Cambios, domain.CambioPerfilAdministracion{Operacion: domain.OperacionOtorgarPerfil,
			RolVersionRef: []string{"rol:tramitacion_bolsa:v1", "rol:gestor_cronos:v2"}[i],
			Objetivo: domain.PreimagenAdministracionPerfiles{CuentaRef: "cta_" + strings.Repeat("b", 24), CuentaVersion: 2,
				PersonaRef: "per_" + strings.Repeat("d", 24), PersonaVersion: 3, PerfilRef: "prf_" + strings.Repeat(letra, 24), VinculoRef: "vca_" + strings.Repeat(letra, 24),
				UnidadRef: "unidad:prueba", HuellaSHA256: strings.Repeat("f", 64), ProcedenciaRef: "procedencia:prueba:1", ProcedenciaVersion: 1,
				ProcedenciaHuellaSHA256: strings.Repeat("e", 64), VigenteDesde: e.ahora, VigenteHasta: e.ahora.Add(time.Hour)}})
	}
	sellarLotePerfilesPrueba(t, &s)
	huella, err := e.instantanea.VersionRol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	c := catalogoPerfilesLotePrueba{e.instantanea.VersionRol.Referencia(): {VersionRef: e.instantanea.VersionRol.Referencia(), Clase: domain.ClaseControlPerfilAdministrador,
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
func TestAdministracionPerfilesLoteDeniegaAntesDelPuerto(t *testing.T) {
	for _, caso := range []string{"autoasignacion", "sistemas", "duplicado", "huella", "CAS", "sensible", "evidencia", "correlacion", "vigencia"} {
		t.Run(caso, func(t *testing.T) {
			servicio, s, a, c := lotePerfilesAplicacionPrueba(t)
			switch caso {
			case "autoasignacion":
				for i := range s.Cambios {
					s.Cambios[i].Objetivo.PersonaRef = s.Actor.PersonaRef
				}
			case "sistemas":
				r := c[s.InstantaneaAutorizacion.VersionRol.Referencia()]
				r.Clase = domain.ClaseControlPerfilOrdinario
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
				c[r.VersionRef] = r
			case "evidencia":
				s.Evidencia = domain.EvidenciaSesionAdministracionPerfiles{}
			case "correlacion":
				s.CorrelacionRef = ""
			case "vigencia":
				s.Cambios[0].Objetivo.VigenteDesde = time.Time{}
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
	for _, caso := range []string{"actor", "asignacion", "correlacion", "preimagen", "perfil", "version", "fecha", "motivo", "parcial"} {
		t.Run(caso, func(t *testing.T) {
			servicio, s, a, _ := lotePerfilesAplicacionPrueba(t)
			a.mutar = func(r *domain.ReciboLoteAdministracionPerfiles) {
				switch caso {
				case "actor":
					r.Cambios[0].ActorPersonaRef = s.Cambios[0].Objetivo.PersonaRef
				case "asignacion":
					r.Cambios[0].AsignacionPerfilRef = "asignacion:otra:v1"
				case "correlacion":
					r.Cambios[0].CorrelacionRef = "correlacion_" + strings.Repeat("f", 32)
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
