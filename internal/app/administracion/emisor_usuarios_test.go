package administracion

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	formato "vec-diputacion-granada/internal/vec/adapters/usuariosadministrables/postgres"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type autenticacionUsuariosPrueba struct {
	valor domain.AutenticacionRevalidadaV1
}

func (r autenticacionUsuariosPrueba) RevalidarAutenticacionActorV1(context.Context, domain.SolicitudRevalidacionAutenticacionActorV1) (domain.AutenticacionRevalidadaV1, error) {
	return r.valor, nil
}

type contextoUsuariosPrueba struct {
	valor domain.ResultadoContextoActorRegistradoV2
}

func (r contextoUsuariosPrueba) ResolverContextoActorRegistradoV2(context.Context, domain.SolicitudContextoActor) (domain.ResultadoContextoActorRegistradoV2, error) {
	return r.valor, nil
}

// Fixture sólo de construcción de solicitud: no se invoca el PDP ni se
// fabrica una confirmación SQL favorable, firma, capacidad o recibo.
func escenarioSolicitudUsuarios(t *testing.T) (*EmisorUsuarios, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion, ports.EmisionUsuariosAdministrables) {
	t.Helper()
	cfg, deps := escenarioConfianzaUsuariosPrueba(t)
	cadena, err := NuevaConfianzaUsuariosV3(cfg, deps)
	if err != nil {
		t.Fatal(err)
	}
	motivos := map[string]domain.ReferenciaEntradaCatalogo{}
	for _, audiencia := range []string{AudienciaUsuariosListarV3, AudienciaUsuariosConsultarV3} {
		motivos[audiencia] = domain.ReferenciaEntradaCatalogo{CatalogoID: deps.CatalogoMotivosID, CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_" + strings.Repeat("f", 32)}
	}
	e, err := NuevoEmisorUsuarios(cadena.Emisores, motivos, deps.Reloj)
	if err != nil {
		t.Fatal(err)
	}
	ahora := deps.Reloj.Ahora()
	resultado, v, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_"+strings.Repeat("a", 22), "prf_"+strings.Repeat("b", 22), domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := v.Datos()
	if err != nil {
		t.Fatal(err)
	}
	a := datos.Autenticacion()
	a.CuentaOrdinariaRef = "cta_" + strings.Repeat("f", 24)
	a.CuentaPrivilegiada = true
	a.Superficie = domain.SuperficieAutenticacionAdministracionPrivilegiadaV1
	v, resultado, err = domain.CrearVinculoAutenticacionActorV2ConResultado(context.Background(), autenticacionUsuariosPrueba{a}, domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: a.AutenticacionRef, SesionRef: a.SesionRef}, contextoUsuariosPrueba{resultado}, domain.SolicitudContextoActor{Cuenta: domain.CuentaAutenticadaContextoActor{CuentaRef: resultado.Contexto.Instantanea.CuentaRef, Metodo: domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh}, PerfilActivoRef: resultado.Contexto.PerfilActivoRef}, deps.Reloj)
	if err != nil {
		t.Fatal(err)
	}
	actor := resultado.Contexto
	evidencia := domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: v}
	material := []byte(`{"esquema":"vec.admin.usuarios.listar.v1","organizacion_ref":"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","unidad_ref":"unidad_admin_sintetica","conjunto_ref":"conjunto_admin:dfa8fa3eef2981ce04ad7dcccbee704d","filtros":{"perfil_ref":"","unidad_ref":"","estado":""},"cursor":"","limite":50}`)
	recurso := domain.RecursoAutorizable{Referencia: "conjunto_admin:dfa8fa3eef2981ce04ad7dcccbee704d", ModuloID: "administracion", Tipo: "conjunto_usuarios", Ambitos: map[string]string{"organizacion_ref": "org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "unidad_ref": "unidad_admin_sintetica"}, Atributos: map[string]string{"material_sha256": "eb9b4ee7842f339489059734f0613261f5a4464555226c2bc5dc6111cbdf6776"}}
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	entrada := ports.EmisionUsuariosAdministrables{Material: material, Recurso: recurso, Accion: "administracion.usuarios.listar", Audiencia: AudienciaUsuariosListarV3, Correlacion: correlacion}
	publicado := ahora.Add(-time.Minute)
	rol := domain.VersionRol{RolID: "administracion_perfiles", Version: 5, Nombre: "Perfil de ensayo", Estado: domain.EstadoVersionRolPublicada, PublicadaPor: "ensayo:usuarios", PublicadaEn: publicado, Concesiones: []domain.ConcesionRol{{Accion: entrada.Accion, ModuloID: "administracion", TipoRecurso: recurso.Tipo, Finalidades: []string{"gestion_usuarios"}, GarantiaMinima: domain.AuthAssuranceHigh, CamposPermitidos: camposUsuarios(entrada.Audiencia), Obligaciones: []string{"auditar"}}}}
	asig := domain.AsignacionPerfil{AsignacionID: "ensayo:usuarios:asignacion", Version: 2, PerfilActivoRef: actor.PerfilActivoRef, PrincipalID: actor.PersonaRef, VersionRolRef: rol.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva, Ambitos: []domain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{recurso.Ambitos["organizacion_ref"]}}, {Clave: "unidad_ref", Valores: []string{recurso.Ambitos["unidad_ref"]}}}, VigenteDesde: publicado, VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "ensayo:usuarios", EmitidaEn: publicado}
	hash, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := domain.InstantaneaAutorizacion{AsignacionPerfil: asig, VersionRol: rol, ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: domain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "ensayo:usuarios", ActualizadoEn: publicado}, RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: hash}
	if snapshot.Validar() != nil || evidencia.ValidarEn(actor, ahora) != nil {
		t.Fatal("fixture_de_solicitud_invalida")
	}
	return e, actor, evidencia, snapshot, entrada
}

func TestSolicitudUsuariosLigaV2OriginalYContextoRecurso(t *testing.T) {
	e, actor, evidencia, snapshot, entrada := escenarioSolicitudUsuarios(t)
	solicitud, resultado, copia, err := e.solicitud(context.Background(), actor, evidencia, snapshot, entrada)
	if err != nil {
		v, _ := evidencia.Vinculo.Datos()
		_, fmtErr := formato.ValidarEmisionUsuariosAdministrables(entrada)
		t.Fatalf("solicitud: actor=%v evidencia=%v vigencia=%v formato=%v snapshot=%v superficie=%v privada=%v garantia=%v", actor.Validar(), evidencia.ValidarEn(actor, e.reloj.Ahora()), actor.Instantanea.VigenteEn(e.reloj.Ahora()), fmtErr, snapshotUsuariosValido(snapshot, actor, entrada, e.reloj.Ahora()), v.Superficie, v.CuentaPrivilegiada, v.GarantiaObservada)
	}
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	esperado, err := entrada.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := datos.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || huella != esperado || huella == entrada.Recurso.Atributos["material_sha256"] || !datos.VinculoAutenticacionActor.CoincideExactamenteCon(evidencia.Vinculo) || resultado.RegistroContextoRef != evidencia.ResultadoContexto.RegistroContextoRef || datos.Finalidad != "gestion_usuarios" || datos.Accion != entrada.Accion {
		t.Fatal("ligadura_de_solicitud_divergente")
	}
	resultado.RepresentacionCanonica[0] ^= 1
	copia.Material[0] ^= 1
	copia.Recurso.Ambitos["unidad_ref"] = "unidad:alterada"
	if evidencia.ResultadoContexto.Validar() != nil || entrada.Material[0] != '{' || entrada.Recurso.Ambitos["unidad_ref"] != "unidad_admin_sintetica" {
		t.Fatal("evidencia_o_material_original_alterados")
	}
}

func TestEmisorUsuariosRechazaAntesDePDPEntradasFueraContrato(t *testing.T) {
	for _, caso := range []string{"evidencia_alterada", "scope", "campos", "rol_sistemas", "vinculo_no_privilegiado", "material_ajeno"} {
		t.Run(caso, func(t *testing.T) {
			e, actor, evidencia, snapshot, entrada := escenarioSolicitudUsuarios(t)
			switch caso {
			case "evidencia_alterada":
				evidencia.ResultadoContexto.RepresentacionCanonica = append([]byte(nil), evidencia.ResultadoContexto.RepresentacionCanonica...)
				evidencia.ResultadoContexto.RepresentacionCanonica[0] ^= 1
			case "scope":
				snapshot.AsignacionPerfil.Ambitos[1].Valores = []string{"unidad:ajena"}
			case "campos":
				snapshot.VersionRol.Concesiones[0].CamposPermitidos = append(snapshot.VersionRol.Concesiones[0].CamposPermitidos, "nombre")
			case "rol_sistemas":
				snapshot.VersionRol.RolID = "administracion_sistemas"
			case "vinculo_no_privilegiado":
				evidencia.Vinculo = domain.VinculoAutenticacionActorV2{}
			case "material_ajeno":
				entrada.Material = append(entrada.Material, ' ')
			}
			salida, err := e.EmitirLecturaUsuariosAdministrables(context.Background(), actor, evidencia, snapshot, entrada)
			if !errors.Is(err, ports.ErrLecturaUsuariosAdministrablesNoDisponible) || errors.Is(err, domain.ErrAutorizacionDenegada) || salida.ValidarEstructura() == nil {
				t.Fatal("entrada_abre_PDP_o_simula_denegacion")
			}
		})
	}
}

func TestConstructorEmisorUsuariosNoAdmiteOtrosConjuntos(t *testing.T) {
	e, _, _, _, _ := escenarioSolicitudUsuarios(t)
	otros := map[string]*confianza.EmisorMaterialAutorizacionAtestadaV3{AudienciaPerfilesCapacidadesV3: e.emisores[AudienciaUsuariosListarV3], AudienciaUsuariosConsultarV3: e.emisores[AudienciaUsuariosConsultarV3]}
	if emisor, err := NuevoEmisorUsuarios(otros, e.motivos, e.reloj); !errors.Is(err, ErrConfiguracion) || emisor != nil {
		t.Fatal("audiencia_legacy_admitida")
	}
	if emisor, err := NuevoEmisorUsuarios(e.emisores, nil, e.reloj); !errors.Is(err, ErrConfiguracion) || emisor != nil {
		t.Fatal("motivo_inventado")
	}
}
