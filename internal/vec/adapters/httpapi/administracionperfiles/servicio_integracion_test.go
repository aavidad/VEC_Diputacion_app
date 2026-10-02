package administracionperfiles

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type catalogoFocal struct{ rol domain.RolAdministrable }

func (c catalogoFocal) ResolverRolAdministrable(context.Context, string) (domain.RolAdministrable, error) {
	return c.rol, nil
}

type relojFocal struct{ ahora time.Time }

func (r relojFocal) Ahora() time.Time { return r.ahora }

type autoridadFocal struct {
	llamada    *domain.SolicitudActoAdministracionPerfiles
	alteracion bool
}

func (a *autoridadFocal) AplicarActoOrdinario(_ context.Context, s domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error) {
	a.llamada = &s
	r := domain.ReciboAdministracionPerfiles{OperacionRef: s.OperacionRef, ActoRef: s.OperacionRef, ReciboRef: "recibo_admin:" + strings.Repeat("a", 32), AuditoriaRef: "auditoria:prueba", ObjetivoPersonaRef: s.Objetivo.PersonaRef, PerfilRef: s.Objetivo.PerfilRef, VinculoRef: s.Objetivo.VinculoRef, VersionPosterior: 1, EstadoPosterior: domain.EstadoVinculoContextoActorActivo, HuellaAntesSHA256: s.Objetivo.HuellaSHA256, HuellaDespuesSHA256: strings.Repeat("f", 64), ConfirmadoEn: s.Actor.ResueltoEn, UnidadRef: s.Objetivo.UnidadRef, ReferenciaActo: s.ReferenciaActo}
	if a.alteracion {
		r.UnidadRef = "unidad:otra"
	}
	return r, nil
}
func (a *autoridadFocal) ProponerActoSensible(context.Context, domain.SolicitudActoAdministracionPerfiles) (ports.PropuestaAdministracionPerfiles, error) {
	return ports.PropuestaAdministracionPerfiles{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func (a *autoridadFocal) CerrarPropuestaSensible(context.Context, domain.SolicitudCierrePropuestaAdministracionPerfiles) (ports.CierrePropuestaAdministracionPerfiles, error) {
	return ports.CierrePropuestaAdministracionPerfiles{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}

func TestActoHTTPConservaUnidadYReferenciaOpcional(t *testing.T) {
	for _, caso := range []struct {
		nombre, unidad, referencia string
		alteracion                 bool
		estado                     int
		efecto                     bool
	}{
		{"sin acto legal", "unidad:prueba", "", false, http.StatusOK, true},
		{"con acto legal", "unidad:prueba", "Resolución 2026/123", false, http.StatusOK, true},
		{"sin ámbito obligatorio", "", "", false, http.StatusBadRequest, false},
		{"recibo de otra unidad", "unidad:prueba", "", true, http.StatusServiceUnavailable, true},
		{"referencia multilinea", "unidad:prueba", "acto\n123", false, http.StatusBadRequest, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			sesion := sesionADMINPrueba(t)
			rol := domain.RolAdministrable{VersionRef: "rol:dietas_liquidacion_rrhh:v1", Clase: domain.ClaseControlPerfilOrdinario, HuellaSHA256: strings.Repeat("a", 64), VigenteDesde: sesion.Actor.ResueltoEn.Add(-time.Hour), VigenteHasta: sesion.Actor.ResueltoEn.Add(time.Hour), UnidadRequerida: true}
			autoridad := &autoridadFocal{alteracion: caso.alteracion}
			servicio, err := application.NuevoServicioAdministracionPerfiles(catalogoFocal{rol}, autoridad, relojFocal{sesion.Actor.ResueltoEn})
			if err != nil {
				t.Fatal(err)
			}
			h, err := NuevoHandler("https://admin.example.test", &sesionPrueba{resultado: sesion}, &lecturasPrueba{}, catalogoFocal{rol}, servicio, &auditorPrueba{})
			if err != nil {
				t.Fatal(err)
			}
			dto := SolicitudActo{OperacionRef: "acto_admin:" + strings.Repeat("b", 32), Operacion: "otorgar", RolVersionRef: rol.VersionRef, ReferenciaActo: caso.referencia,
				Objetivo: Objetivo{UnidadRef: caso.unidad, CuentaRef: "cta_" + strings.Repeat("f", 22), CuentaVersion: 1, PersonaRef: "per_" + strings.Repeat("f", 22), PersonaVersion: 1, PerfilRef: "prf_" + strings.Repeat("f", 22), VinculoRef: "vca_" + strings.Repeat("f", 22), HuellaSHA256: strings.Repeat("c", 64), ProcedenciaRef: "procedencia:maestra:prueba", ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("d", 64), VigenteHasta: sesion.Actor.ResueltoEn.Add(time.Hour)},
				Motivo:   Motivo{CatalogoID: "motivos_admin", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("e", 64), EntradaClave: "provision"}}
			cuerpo, err := json.Marshal(dto)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionADMIN(http.MethodPost, PrefijoV1+"/actos-ordinarios", string(cuerpo)))
			if w.Code != caso.estado {
				t.Fatalf("estado=%d esperado=%d cuerpo=%s", w.Code, caso.estado, w.Body.String())
			}
			if (autoridad.llamada != nil) != caso.efecto {
				t.Fatal("efecto de autoridad inesperado")
			}
			if autoridad.llamada != nil && (autoridad.llamada.ReferenciaActo != caso.referencia ||
				autoridad.llamada.Objetivo.UnidadRef != caso.unidad ||
				autoridad.llamada.Evidencia.ValidarPara(sesion.Actor) != nil) {
				t.Fatal("se perdieron unidad, acto o evidencia de sesión")
			}
		})
	}
}
