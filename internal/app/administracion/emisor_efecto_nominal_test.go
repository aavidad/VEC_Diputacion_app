package administracion

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	efecto "vec-diputacion-granada/internal/vec/adapters/postgres/efectonominaladmin"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// materialCargoPrueba arma un material de Personal28 con los ámbitos de la
// asignación del escenario de usuarios (organización y unidad sintéticas).
func materialCargoPrueba(t *testing.T, operacion, objeto, unidad string) []byte {
	t.Helper()
	m := map[string]any{"esquema": "vec.personal.cargo-competencial.publicacion.v1", "operacion": operacion,
		"clave_idempotencia": strings.Repeat("a", 32), "objeto_ref": objeto,
		"organizacion_ref": "org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "unidad_ref": unidad,
		"version_esperada": 0, "huella_esperada": nil, "datos": map[string]any{"version": 1}}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

const objetoCargoPrueba = "car_" + "AAAAAAAAAAAAAAAAAAAAAAAA"

// Reutiliza la sesión ADMIN del escenario de usuarios con la concesión de
// publicación de cargos de Rol7. No se invoca el PDP: todo caso se cierra antes.
func escenarioEmisorCargo(t *testing.T) (*EmisorEfectoNominalADMIN, context.Context, domain.ContextoActor,
	domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion, []byte, string) {
	t.Helper()
	u, actor, evidencia, snapshot, _ := escenarioSolicitudUsuarios(t)
	e, err := NuevoEmisorEfectoNominalADMIN(u.emisores[AudienciaUsuariosConsultarV3], u.motivos[AudienciaUsuariosConsultarV3], u.reloj,
		personalpg.ContratoPublicacionCargoCompetencial())
	if err != nil {
		t.Fatal(err)
	}
	snapshot.VersionRol.Concesiones = []domain.ConcesionRol{{Accion: "personal.cargo_competencial.publicar", ModuloID: "personal",
		TipoRecurso: "cargo_competencial", Finalidades: []string{"administrar_cargos_competenciales"}, GarantiaMinima: domain.AuthAssuranceHigh,
		CamposPermitidos: []string{"cargo", "enlace", "huella_sha256", "recibo", "version"}}}
	snapshot.AsignacionPerfil.VersionRolRef = snapshot.VersionRol.Referencia()
	snapshot.ControlVigenciaVersionRol.VersionRolRef = snapshot.VersionRol.Referencia()
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
	if snapshot.Validar() != nil {
		t.Fatal("fixture_de_cargos_invalida")
	}
	return e, ctx, actor, evidencia, snapshot, materialCargoPrueba(t, "cargo", objetoCargoPrueba, "unidad_admin_sintetica"), valor
}

// El escenario base pasa todas las precondiciones locales; cada caso cambia
// una sola cosa y la emisión se cierra antes del PDP.
func TestEmisorEfectoNominalRechazaAntesDePDPEntradasFueraContrato(t *testing.T) {
	e, _, actor, _, snapshot, material, _ := escenarioEmisorCargo(t)
	accion, recurso, err := personalpg.RecursoPublicacionCargoCompetencial(material, snapshot.AsignacionPerfil)
	if err != nil || !e.snapshotValido(snapshot, actor, accion, recurso, e.reloj.Ahora()) {
		t.Fatal("el escenario base no pasa las precondiciones locales")
	}
	for _, caso := range []string{"sin_concesion", "campos_distintos", "sin_campos", "obligacion", "otra_finalidad", "otro_modulo",
		"rol_sistemas", "vinculo_no_privilegiado", "correlacion_ajena", "sin_correlacion", "sin_material", "material_otro_esquema",
		"objeto_de_enlace_en_cargo"} {
		t.Run(caso, func(t *testing.T) {
			e, ctx, actor, evidencia, snapshot, material, correlacion := escenarioEmisorCargo(t)
			c := &snapshot.VersionRol.Concesiones[0]
			switch caso {
			case "sin_concesion":
				c.Accion = "administracion.usuarios.consultar"
			case "campos_distintos":
				c.CamposPermitidos = []string{"cargo", "version"}
			case "sin_campos":
				c.CamposPermitidos = nil
			case "obligacion":
				c.Obligaciones = []string{"auditar"}
			case "otra_finalidad":
				c.Finalidades = []string{"gestion_perfiles"}
			case "otro_modulo":
				c.ModuloID = "administracion"
			case "rol_sistemas":
				snapshot.VersionRol.RolID = "administracion_sistemas"
			case "vinculo_no_privilegiado":
				evidencia.Vinculo = domain.VinculoAutenticacionActorV2{}
			case "correlacion_ajena":
				correlacion = "correlacion_" + strings.Repeat("0", 32)
			case "sin_correlacion":
				ctx = context.Background()
			case "sin_material":
				material = nil
			case "material_otro_esquema":
				material = []byte(strings.Replace(string(material), "cargo-competencial.publicacion.v1", "cargo-competencial.otra.v1", 1))
			case "objeto_de_enlace_en_cargo":
				material = materialCargoPrueba(t, "cargo", "enc_"+strings.Repeat("A", 24), "unidad_admin_sintetica")
			}
			salida, err := e.Emitir(ctx, actor, evidencia, snapshot, material, correlacion)
			if err == nil || errors.Is(err, domain.ErrAutorizacionDenegada) || salida.Material.ValidarEstructura() == nil {
				t.Fatalf("%s abre el PDP o simula denegación: %v", caso, err)
			}
		})
	}
}

func TestConstructorEmisorEfectoNominalExigeEmisorMotivoRelojYContrato(t *testing.T) {
	e, _, _, _, _, _, _ := escenarioEmisorCargo(t)
	contrato := personalpg.ContratoPublicacionCargoCompetencial()
	if x, err := NuevoEmisorEfectoNominalADMIN(nil, e.motivo, e.reloj, contrato); !errors.Is(err, ErrConfiguracion) || x != nil {
		t.Fatal("emisor_ausente_admitido")
	}
	if x, err := NuevoEmisorEfectoNominalADMIN(e.emisor, domain.ReferenciaEntradaCatalogo{}, e.reloj, contrato); !errors.Is(err, ErrConfiguracion) || x != nil {
		t.Fatal("motivo_inventado")
	}
	if x, err := NuevoEmisorEfectoNominalADMIN(e.emisor, e.motivo, nil, contrato); !errors.Is(err, ErrConfiguracion) || x != nil {
		t.Fatal("reloj_ausente_admitido")
	}
	for nombre, cambiar := range map[string]func(*efecto.Contrato){
		"sin_recurso":        func(c *efecto.Contrato) { c.Recurso = nil },
		"campos_sin_orden":   func(c *efecto.Contrato) { c.Campos = []string{"version", "cargo"} },
		"sentencia_libre":    func(c *efecto.Contrato) { c.Sentencia = "SELECT 1" },
		"sin_audiencia":      func(c *efecto.Contrato) { c.Audiencia = "" },
		"sin_acciones":       func(c *efecto.Contrato) { c.Acciones = nil },
		"prefijo_incorrecto": func(c *efecto.Contrato) { c.PrefijoIntento = "Con Espacios" },
	} {
		c := personalpg.ContratoPublicacionCargoCompetencial()
		cambiar(&c)
		if x, err := NuevoEmisorEfectoNominalADMIN(e.emisor, e.motivo, e.reloj, c); !errors.Is(err, ErrConfiguracion) || x != nil {
			t.Fatalf("%s admitido", nombre)
		}
	}
}

// La confianza de un efecto sólo admite audiencias conocidas.
func TestConfianzaEfectoNominalSoloAudienciasConocidas(t *testing.T) {
	if _, err := NuevaConfianzaEfectoNominalV3(ConfiguracionConfianzaPerfilesV3{}, DependenciasConfianzaPerfilesV3{}, AudienciaUsuariosListarV3); !errors.Is(err, ErrConfiguracion) {
		t.Fatal("audiencia ajena admitida")
	}
	if personalpg.AudienciaPublicarCargoCompetencial != AudienciaCargoCompetencialV3 {
		t.Fatal("la audiencia de Personal no es la de vec-admin")
	}
}

// Un material de otra unidad u organización que la asignación vigente no
// cubre es una denegación explícita, sin llegar al PDP.
func TestEmisorEfectoNominalDeniegaAmbitoNoCubierto(t *testing.T) {
	for _, material := range [][]byte{
		materialCargoPrueba(t, "cargo", objetoCargoPrueba, "unidad_ajena_sintetica"),
		[]byte(strings.Replace(string(materialCargoPrueba(t, "cargo", objetoCargoPrueba, "unidad_admin_sintetica")),
			"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "org_cccccccccccccccccccccccccccccccc", 1)),
	} {
		e, ctx, actor, evidencia, snapshot, _, correlacion := escenarioEmisorCargo(t)
		salida, err := e.Emitir(ctx, actor, evidencia, snapshot, material, correlacion)
		if !errors.Is(err, domain.ErrAutorizacionDenegada) || salida.Material.ValidarEstructura() == nil {
			t.Fatalf("ámbito no cubierto sin denegación explícita: %v", err)
		}
	}
	// Una asignación sin unidad tampoco cubre un material con unidad.
	e, ctx, actor, evidencia, snapshot, material, correlacion := escenarioEmisorCargo(t)
	snapshot.AsignacionPerfil.Ambitos = snapshot.AsignacionPerfil.Ambitos[:1]
	if _, err := e.Emitir(ctx, actor, evidencia, snapshot, material, correlacion); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("asignación sin unidad sin denegación explícita: %v", err)
	}
}
