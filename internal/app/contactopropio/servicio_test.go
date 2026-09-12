package contactopropio

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios"
	seguridad "vec-diputacion-granada/internal/vec/adapters/documentos/seguridad"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type relojContacto time.Time

func (r relojContacto) Ahora() time.Time { return time.Time(r) }

type fuenteAuditoriaPrueba struct {
	instantanea       domain.InstantaneaAutorizacion
	principal, perfil string
	llamadas          int
}

func (f *fuenteAuditoriaPrueba) ObtenerInstantaneaAutorizacion(_ context.Context, principal, perfil string) (domain.InstantaneaAutorizacion, error) {
	f.principal, f.perfil = principal, perfil
	f.llamadas++
	return f.instantanea, nil
}

func TestServicioDeniegaDependenciasAusentesYEntradasSinIdentidad(t *testing.T) {
	if s, err := NuevoServicio(Dependencias{}); s != nil || !errors.Is(err, ErrContactoPropioNoDisponible) {
		t.Fatal("dependencias vacías aceptadas")
	}
	var s *Servicio
	for _, entrada := range []struct {
		correo  string
		version uint64
	}{{"", 0}, {" usuario@example.invalid", 0}, {"usuario@example.invalid\r\n", 0}, {strings.Repeat("x", 255), 0}, {"usuario@example.invalid", 1<<53 - 1}} {
		if _, err := s.Guardar(context.Background(), entrada.correo, entrada.version); !errors.Is(err, ErrContactoPropioInvalido) {
			t.Fatal("entrada inválida aceptada")
		}
	}
	if _, err := s.Guardar(context.Background(), "usuario@example.invalid", 0); !errors.Is(err, ErrContactoPropioNoDisponible) {
		t.Fatal("servicio sin identidad aceptado")
	}
}

// Prueba del adaptador de auditoría con el HMAC existente real. Las instantáneas
// son datos sintéticos de esta unidad, no una concesión ni una prueba V3/PG.
func entornoAuditoriaPropia(t *testing.T) (preparadorAuditoria, domain.ContextoActor, *fuenteAuditoriaPrueba) {
	t.Helper()
	ahora := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	ref := func(prefijo, letra string) string { return prefijo + strings.Repeat(letra, 22) }
	cuenta := domain.CuentaAutenticadaContextoActor{CuentaRef: ref("cta_", "c"), Metodo: domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh}
	instantaneaActor := domain.InstantaneaContextoActor{VinculoRef: ref("vca_", "v"), VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, PersonaRef: ref("per_", "p"), PersonaVersion: 1, PerfilActivoRef: ref("prf_", "f"), PerfilVersion: 1, Estado: domain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := domain.NuevoContextoActor(cuenta, instantaneaActor, ahora)
	if err != nil {
		t.Fatal(err)
	}
	rol := domain.VersionRol{RolID: "contacto_sintetico", Version: 1, Nombre: "Contacto sintético", Estado: domain.EstadoVersionRolPublicada, Concesiones: []domain.ConcesionRol{{Accion: application.AccionAltaContactoUsuario, ModuloID: usuarios.ModuleID, TipoRecurso: "contacto_usuario", Finalidades: []string{FinalidadRegistro}, GarantiaMinima: domain.AuthAssuranceHigh}}, PublicadaPor: "autoridad-sintetica", PublicadaEn: ahora.Add(-2 * time.Hour)}
	huella, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	fuente := &fuenteAuditoriaPrueba{instantanea: domain.InstantaneaAutorizacion{AsignacionPerfil: domain.AsignacionPerfil{AsignacionID: "asignacion-sintetica", Version: 1, PerfilActivoRef: actor.PerfilActivoRef, PrincipalID: actor.Principal.ID, VersionRolRef: rol.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva, Ambitos: []domain.AmbitoPerfil{{Clave: "unidad", Valores: []string{"sintetica"}}}, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "autoridad-sintetica", EmitidaEn: ahora.Add(-2 * time.Hour)}, VersionRol: rol, ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: domain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "autoridad-sintetica", ActualizadoEn: rol.PublicadaEn}, RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella}}
	if err = fuente.instantanea.Validar(); err != nil {
		t.Fatal(err)
	}
	sellador, err := seguridad.NuevoSelladorHMAC("prueba_contacto", []byte(strings.Repeat("s", 32)))
	if err != nil {
		t.Fatal(err)
	}
	return preparadorAuditoria{fuente: fuente, seudonimizador: sellador, reloj: relojContacto(ahora), correlacion: "correlacion_sintetica", recurso: domain.RecursoAutorizable{Referencia: actor.PersonaRef, ModuloID: usuarios.ModuleID, Tipo: "contacto_usuario", Ambitos: map[string]string{"unidad": "sintetica"}}}, actor, fuente
}

func TestPreparadorContactoUsaHMACCentralYVersionRolExacta(t *testing.T) {
	p, actor, fuente := entornoAuditoriaPropia(t)
	audit, err := p.PrepararAuditoriaContactoUsuario(context.Background(), actor, application.AccionAltaContactoUsuario, usuarios.ModuleID, actor.PersonaRef, 1)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudSeudonimizarSujetoAlmacen(actor.Principal.ID, "bolsa_registro_accesos_t13")
	if err != nil {
		t.Fatal(err)
	}
	esperado, err := p.seudonimizador.SeudonimizarSujetoAlmacen(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if audit.ActorID != esperado || audit.ActorID == actor.Principal.ID || fuente.principal != actor.Principal.ID || fuente.perfil != actor.PerfilActivoRef || len(audit.ActorRoles) != 1 || audit.ActorRoles[0] != fuente.instantanea.VersionRol.Referencia() || audit.AuthorizationRef != "" || audit.Signature != "" || audit.Purpose != FinalidadRegistro || audit.SubjectRef != actor.PersonaRef || audit.CorrelationRef != p.correlacion {
		t.Fatal("auditoría no ligada a fuentes centrales")
	}
}

func TestPreparadorContactoDeniegaSujetoPerfilRevocacionYAmbitoAjenos(t *testing.T) {
	for _, nombre := range []string{"sujeto", "perfil", "principal", "revocacion", "ambito"} {
		t.Run(nombre, func(t *testing.T) {
			p, actor, fuente := entornoAuditoriaPropia(t)
			sujeto := actor.PersonaRef
			switch nombre {
			case "sujeto":
				sujeto = "per_" + strings.Repeat("z", 22)
			case "perfil":
				fuente.instantanea.AsignacionPerfil.PerfilActivoRef = "prf_" + strings.Repeat("z", 22)
			case "principal":
				fuente.instantanea.AsignacionPerfil.PrincipalID = "per_" + strings.Repeat("z", 22)
			case "revocacion":
				fuente.instantanea.AsignacionPerfil.VigenteHasta = p.reloj.Ahora()
			case "ambito":
				p.recurso.Ambitos["unidad"] = "ajena"
			}
			if _, err := p.PrepararAuditoriaContactoUsuario(context.Background(), actor, application.AccionAltaContactoUsuario, usuarios.ModuleID, sujeto, 1); !errors.Is(err, ErrContactoPropioNoDisponible) {
				t.Fatal("aceptó contexto ajeno/revocado")
			}
		})
	}
}
