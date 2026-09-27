package auditoria

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type revalidadorAuditoriaHTTPPrueba struct {
	autenticacion domain.AutenticacionRevalidadaV1
}

func (r revalidadorAuditoriaHTTPPrueba) RevalidarAutenticacionActorV1(context.Context, domain.SolicitudRevalidacionAutenticacionActorV1) (domain.AutenticacionRevalidadaV1, error) {
	return r.autenticacion, nil
}

type resolutorAuditoriaHTTPPrueba struct {
	resultado domain.ResultadoContextoActorRegistradoV2
}

func (r resolutorAuditoriaHTTPPrueba) ResolverContextoActorRegistradoV2(context.Context, domain.SolicitudContextoActor) (domain.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type relojAuditoriaHTTPPrueba struct{ instante time.Time }

func (r relojAuditoriaHTTPPrueba) Ahora() time.Time { return r.instante }

type generadorCorrelacionAuditoriaHTTPPrueba struct{}

func (generadorCorrelacionAuditoriaHTTPPrueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return "correlacion_11111111111111111111111111111111", nil
}

type identidadAuditoriaHTTPPrueba struct{ resultado IdentidadResuelta }

func (i identidadAuditoriaHTTPPrueba) ResolverIdentidadConsulta(context.Context, *http.Request) (IdentidadResuelta, error) {
	return i.resultado, nil
}

type opcionesAuditoriaHTTPPrueba struct{ llamadas int }

func (o *opcionesAuditoriaHTTPPrueba) Actuales(context.Context) (Opciones, error) {
	o.llamadas++
	return Opciones{FinalidadRef: "auditoria_rrhh", MotivoRef: "motivos:1:motivo_11111111111111111111111111111111", PermisoRequerido: AccionConsultar}, nil
}

func identidadVigenteAuditoriaHTTPPrueba(t *testing.T, ahora time.Time) IdentidadResuelta {
	t.Helper()
	ahora = ahora.UTC().Truncate(time.Microsecond)
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl",
		Metodo:    domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh,
	}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 3,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: 4,
		PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 5,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	canonActor, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huellaActor, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := domain.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef: "prc_0123456789abcdefghijkl", ProcedenciaVersion: 1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad:    domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := domain.ManifiestoProcedenciaContextoActorV1{
		Esquema:           domain.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: domain.ProcedenciaCuentaContextoActorV1{
			CuentaRef: cuenta.CuentaRef, Version: instantanea.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: domain.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantanea.PersonaRef, Version: instantanea.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: domain.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantanea.PerfilActivoRef, Version: instantanea.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: domain.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantanea.VinculoRef, Version: instantanea.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []domain.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	canonManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(canonManifiesto)
	if err != nil {
		t.Fatal(err)
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn",
		Contexto:            actor, RepresentacionCanonica: canonActor, HuellaSHA256: huellaActor,
		ManifiestoProcedenciaCanonico:     canonManifiesto,
		ManifiestoProcedenciaHuellaSHA256: huellaManifiesto,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef: "aut_0123456789abcdefghijkl", AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef: "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 2,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:      domain.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
		SesionValidaHasta:            ahora.Add(30 * time.Minute),
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(context.Background(),
		revalidadorAuditoriaHTTPPrueba{autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef},
		resolutorAuditoriaHTTPPrueba{resultado},
		domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: instantanea.PerfilActivoRef},
		relojAuditoriaHTTPPrueba{ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), generadorCorrelacionAuditoriaHTTPPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	return IdentidadResuelta{Vinculo: vinculo, Resultado: resultado, Correlacion: correlacion}
}

func TestOpcionesAuditoriaAdmiteInstanteRealSubmicrosegundo(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	identidad := identidadVigenteAuditoriaHTTPPrueba(t, ahora)
	opciones := &opcionesAuditoriaHTTPPrueba{}
	h, err := NuevoManejador(&Servicio{}, opciones, identidadAuditoriaHTTPPrueba{identidad})
	if err != nil {
		t.Fatal(err)
	}
	h.ahora = func() time.Time { return time.Now().UTC().Truncate(time.Microsecond).Add(125 * time.Nanosecond) }
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaOpciones, nil))
	if w.Code != http.StatusOK || opciones.llamadas != 1 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("GET opciones status=%d llamadas=%d body=%s", w.Code, opciones.llamadas, w.Body.String())
	}
}

func TestOpcionesAuditoriaDeniegaIdentidadCaducadaConInstanteCanonico(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	identidad := identidadVigenteAuditoriaHTTPPrueba(t, ahora)
	opciones := &opcionesAuditoriaHTTPPrueba{}
	h, err := NuevoManejador(&Servicio{}, opciones, identidadAuditoriaHTTPPrueba{identidad})
	if err != nil {
		t.Fatal(err)
	}
	h.ahora = func() time.Time { return time.Now().Add(2 * time.Hour) }
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaOpciones, nil))
	if w.Code != http.StatusForbidden || opciones.llamadas != 0 {
		t.Fatalf("GET caducado status=%d llamadas=%d body=%s", w.Code, opciones.llamadas, w.Body.String())
	}
}
