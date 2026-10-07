package administracion

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func shaGobiernoPrueba(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// materialGobiernoPlanPrueba arma un material de trece claves como el kit
// (vec-plan-firma-validar preparar) para la operación indicada.
func materialGobiernoPlanPrueba(t *testing.T, operacion, estado string) []byte {
	t.Helper()
	canon := []byte(`{"id":"ct.plan.firma.sintetico","version":1,"revision":2,"modulo_id":"contratacion_temporal","estado":"` + estado + `"}`)
	m := map[string]any{"esquema": "vec.catalogos.plan-firma.gobierno.v1", "operacion": operacion,
		"catalogo_id": "ct.plan.firma.sintetico", "version": 1, "revision_esperada": 2,
		"huella_esperada": strings.Repeat("b", 64), "clave_operacion": "clave-sintetica-0001",
		"catalogo_canonico_base64": base64.StdEncoding.EncodeToString(canon), "catalogo_sha256": shaGobiernoPrueba(canon),
		"traza_canonica_base64": "e30=", "traza_sha256": shaGobiernoPrueba([]byte("{}")),
		"evento_canonico_base64": "e30=", "evento_sha256": shaGobiernoPrueba([]byte("{}"))}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Reutiliza la sesión ADMIN del escenario de usuarios con la concesión del
// gobierno del plan (AUT51). No se invoca el PDP: todo caso debe cerrarse antes.
func escenarioEmisorGobiernoPlan(t *testing.T) (*EmisorGobiernoPlanFirma, context.Context, domain.ContextoActor,
	domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion, []byte, string) {
	t.Helper()
	u, actor, evidencia, snapshot, _ := escenarioSolicitudUsuarios(t)
	e, err := NuevoEmisorGobiernoPlanFirma(u.emisores[AudienciaUsuariosConsultarV3], u.motivos[AudienciaUsuariosConsultarV3], u.reloj)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.VersionRol.Concesiones = []domain.ConcesionRol{{Accion: "vec.catalogos.publicar", ModuloID: "contratacion_temporal",
		TipoRecurso: "catalogo_configurable", Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: domain.AuthAssuranceHigh}}
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
		t.Fatal("fixture_de_gobierno_invalida")
	}
	return e, ctx, actor, evidencia, snapshot, materialGobiernoPlanPrueba(t, "publicar", "publicado"), valor
}

// El escenario base pasa todas las precondiciones locales: sólo cambia una
// cosa en cada caso y la emisión se cierra antes del PDP.
func TestEmisorGobiernoPlanRechazaAntesDePDPEntradasFueraContrato(t *testing.T) {
	e, _, actor, _, snapshot, material, _ := escenarioEmisorGobiernoPlan(t)
	if !snapshotGobiernoPlanFirmaValidoParaMaterial(t, snapshot, actor, material, "vec.catalogos.publicar", e) {
		t.Fatal("el escenario base no pasa las precondiciones locales")
	}
	for _, caso := range []string{"otra_accion_concedida", "sin_concesion", "campos", "obligacion", "otro_modulo", "rol_sistemas",
		"vinculo_no_privilegiado", "correlacion_ajena", "sin_correlacion", "sin_material", "material_otro_esquema", "estado_incoherente",
		"asignacion_sin_unidad", "asignacion_dos_unidades"} {
		t.Run(caso, func(t *testing.T) {
			e, ctx, actor, evidencia, snapshot, material, correlacion := escenarioEmisorGobiernoPlan(t)
			switch caso {
			case "otra_accion_concedida":
				material = materialGobiernoPlanPrueba(t, "retirar", "retirado")
			case "sin_concesion":
				snapshot.VersionRol.Concesiones[0].Accion = "administracion.usuarios.consultar"
			case "campos":
				snapshot.VersionRol.Concesiones[0].CamposPermitidos = []string{"nombre"}
			case "obligacion":
				snapshot.VersionRol.Concesiones[0].Obligaciones = []string{"auditar"}
			case "otro_modulo":
				snapshot.VersionRol.Concesiones[0].ModuloID = "administracion"
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
				material = []byte(strings.Replace(string(material), "vec.catalogos.plan-firma.gobierno.v1", "vec.catalogos.otro.v1", 1))
			case "estado_incoherente":
				material = materialGobiernoPlanPrueba(t, "publicar", "borrador")
			case "asignacion_sin_unidad":
				snapshot.AsignacionPerfil.Ambitos = snapshot.AsignacionPerfil.Ambitos[:1]
			case "asignacion_dos_unidades":
				snapshot.AsignacionPerfil.Ambitos[1].Valores = append(snapshot.AsignacionPerfil.Ambitos[1].Valores, "unidad:otra")
			}
			salida, err := e.EmitirGobiernoPlanFirma(ctx, actor, evidencia, snapshot, material, correlacion)
			if err == nil || errors.Is(err, domain.ErrAutorizacionDenegada) || salida.Material.ValidarEstructura() == nil {
				t.Fatalf("%s abre el PDP o simula denegación: %v", caso, err)
			}
		})
	}
}

// snapshotGobiernoPlanFirmaValidoParaMaterial comprueba que el escenario base
// sí cumple las precondiciones (para que los casos negativos signifiquen algo).
func snapshotGobiernoPlanFirmaValidoParaMaterial(t *testing.T, s domain.InstantaneaAutorizacion, actor domain.ContextoActor,
	material []byte, accion string, e *EmisorGobiernoPlanFirma) bool {
	t.Helper()
	ambito, err := ambitoDePrueba(s)
	if err != nil {
		t.Fatal(err)
	}
	a, recurso, err := recursoDePrueba(material, ambito)
	return err == nil && a == accion && snapshotGobiernoPlanFirmaValido(s, actor, accion, recurso, e.reloj.Ahora())
}

func TestConstructorEmisorGobiernoPlanExigeEmisorMotivoYReloj(t *testing.T) {
	e, _, _, _, _, _, _ := escenarioEmisorGobiernoPlan(t)
	if x, err := NuevoEmisorGobiernoPlanFirma(nil, e.motivo, e.reloj); !errors.Is(err, ErrConfiguracion) || x != nil {
		t.Fatal("emisor_ausente_admitido")
	}
	if x, err := NuevoEmisorGobiernoPlanFirma(e.emisor, domain.ReferenciaEntradaCatalogo{}, e.reloj); !errors.Is(err, ErrConfiguracion) || x != nil {
		t.Fatal("motivo_inventado")
	}
	if x, err := NuevoEmisorGobiernoPlanFirma(e.emisor, e.motivo, nil); !errors.Is(err, ErrConfiguracion) || x != nil {
		t.Fatal("reloj_ausente_admitido")
	}
}
