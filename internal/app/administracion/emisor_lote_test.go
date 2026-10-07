package administracion

import (
	"context"
	"errors"
	"strings"
	"testing"

	lote "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Reutiliza la sesión ADMIN del escenario de usuarios y cambia la concesión a
// la del lote. No se invoca el PDP: todos los casos deben cerrarse antes.
func escenarioEmisorLote(t *testing.T) (*EmisorLote, context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion, domain.RecursoAutorizable, lote.Efecto) {
	t.Helper()
	u, actor, evidencia, snapshot, _ := escenarioSolicitudUsuarios(t)
	e, err := NuevoEmisorLote(u.emisores[AudienciaUsuariosConsultarV3], u.motivos[AudienciaUsuariosConsultarV3], u.reloj)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.VersionRol.Concesiones = []domain.ConcesionRol{{Accion: AccionLoteOrdinarioV3, ModuloID: "administracion", TipoRecurso: "persona",
		Finalidades: []string{finalidadLoteOrdinario}, GarantiaMinima: domain.AuthAssuranceHigh, Obligaciones: []string{"auditar"}}}
	snapshot.AsignacionPerfil.VersionRolRef = snapshot.VersionRol.Referencia()
	snapshot.ControlVigenciaVersionRol.VersionRolRef = snapshot.VersionRol.Referencia()
	org := snapshot.AsignacionPerfil.Ambitos[0].Valores[0]
	unidad := snapshot.AsignacionPerfil.Ambitos[1].Valores[0]
	material := []byte(`{"Esquema":"administracion_perfiles_lote:v3"}`)
	recurso := domain.RecursoAutorizable{Referencia: "per_" + strings.Repeat("c", 22), ModuloID: "administracion", Tipo: "persona",
		Ambitos: map[string]string{"organizacion_ref": org, "unidad_ref": unidad}, Atributos: map[string]string{"solicitud_sha256": huellaMaterialLote(material)}}
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	valor, err := correlacion.ValorCanonico()
	if err != nil {
		t.Fatal(err)
	}
	efecto := lote.Efecto{Accion: AccionLoteOrdinarioV3, Audiencia: AudienciaLoteOrdinarioV3, Referencia: recurso.Referencia,
		Material: material, CorrelacionAccesoRef: valor}
	if snapshot.Validar() != nil || !snapshotLoteValido(snapshot, actor, recurso, e.reloj.Ahora()) {
		t.Fatal("fixture_de_lote_invalida")
	}
	return e, ctx, actor, evidencia, snapshot, recurso, efecto
}

func TestEmisorLoteRechazaAntesDePDPEntradasFueraContrato(t *testing.T) {
	for _, caso := range []string{"autoasignacion", "accion", "audiencia", "tipo", "efecto_ajeno", "sin_concesion", "campos", "obligacion",
		"ambito_ajeno", "rol_sistemas", "vinculo_no_privilegiado", "correlacion_ajena", "sin_correlacion", "sin_material", "solicitud_ajena", "atributo_extra"} {
		t.Run(caso, func(t *testing.T) {
			e, ctx, actor, evidencia, snapshot, recurso, efecto := escenarioEmisorLote(t)
			switch caso {
			case "autoasignacion":
				recurso.Referencia, efecto.Referencia = actor.PersonaRef, actor.PersonaRef
			case "accion":
				efecto.Accion = "administracion.perfiles.otorgar"
			case "audiencia":
				efecto.Audiencia = AudienciaUsuariosConsultarV3
			case "tipo":
				recurso.Tipo = "persona_administrable"
			case "efecto_ajeno":
				efecto.Referencia = "per_" + strings.Repeat("e", 22)
			case "sin_concesion":
				snapshot.VersionRol.Concesiones[0].Accion = "administracion.usuarios.consultar"
			case "campos":
				snapshot.VersionRol.Concesiones[0].CamposPermitidos = []string{"nombre"}
			case "obligacion":
				snapshot.VersionRol.Concesiones[0].Obligaciones = nil
			case "ambito_ajeno":
				recurso.Ambitos["unidad_ref"] = "unidad:ajena"
			case "rol_sistemas":
				snapshot.VersionRol.RolID = "administracion_sistemas"
			case "vinculo_no_privilegiado":
				evidencia.Vinculo = domain.VinculoAutenticacionActorV2{}
			case "correlacion_ajena":
				efecto.CorrelacionAccesoRef = "correlacion_" + strings.Repeat("0", 32)
			case "sin_correlacion":
				ctx = context.Background()
			case "sin_material":
				efecto.Material = nil
			case "solicitud_ajena":
				efecto.Material = []byte(`{"Esquema":"otro"}`)
			case "atributo_extra":
				recurso.Atributos["persona_ref"] = recurso.Referencia
			}
			salida, err := e.EmitirLoteOrdinario(ctx, actor, evidencia, snapshot, recurso, efecto)
			if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) || errors.Is(err, domain.ErrAutorizacionDenegada) || salida.ValidarEstructura() == nil {
				t.Fatalf("%s abre el PDP o simula denegación: %v", caso, err)
			}
		})
	}
}

func TestConstructorEmisorLoteExigeEmisorMotivoYReloj(t *testing.T) {
	e, _, _, _, _, _, _ := escenarioEmisorLote(t)
	if x, err := NuevoEmisorLote(nil, e.motivo, e.reloj); !errors.Is(err, ErrConfiguracion) || x != nil {
		t.Fatal("emisor_ausente_admitido")
	}
	if x, err := NuevoEmisorLote(e.emisor, domain.ReferenciaEntradaCatalogo{}, e.reloj); !errors.Is(err, ErrConfiguracion) || x != nil {
		t.Fatal("motivo_inventado")
	}
	if x, err := NuevoEmisorLote(e.emisor, e.motivo, nil); !errors.Is(err, ErrConfiguracion) || x != nil {
		t.Fatal("reloj_ausente_admitido")
	}
}
