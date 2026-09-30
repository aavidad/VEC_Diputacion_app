package bootstrap

import (
	"math"
	"strings"
	"testing"
	"time"

	usuarios "vec-diputacion-granada/internal/modules/usuarios/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func instantaneaPerfilUsuariosExternoPrueba(t *testing.T) core.InstantaneaAutorizacion {
	t.Helper()
	ahora := time.Date(2026, 9, 30, 1, 0, 0, 0, time.UTC)
	var concesiones []core.ConcesionRol
	poner := func(accion, tipo, finalidad string, campos []string) {
		concesiones = append(concesiones, core.ConcesionRol{Accion: accion, ModuloID: "usuarios", TipoRecurso: tipo,
			Finalidades: []string{finalidad}, CamposPermitidos: campos, GarantiaMinima: core.AuthAssuranceHigh})
	}
	poner(usuarios.AccionConsultarPreferencias, usuarios.TipoRecursoPreferencias, usuarios.FinalidadPreferenciasPropias, []string{"catalogo", "valores", "version"})
	poner(usuarios.AccionActualizarPreferencias, usuarios.TipoRecursoPreferencias, usuarios.FinalidadPreferenciasPropias, []string{"valores", "version"})
	for _, accion := range []string{usuarios.AccionConsultarImagen, usuarios.AccionActualizarImagen} {
		poner(accion, usuarios.TipoRecursoImagen, usuarios.FinalidadImagenPropia, usuarios.CamposPermitidosImagen(accion))
	}
	for _, accion := range []string{usuarios.AccionConsultarCorreos, usuarios.AccionAnadirCorreo, usuarios.AccionReenviarCorreo,
		usuarios.AccionVerificarCorreo, usuarios.AccionActivarCorreo, usuarios.AccionRetirarCorreo} {
		poner(accion, usuarios.TipoRecursoCorreos, usuarios.FinalidadCorreosPropios, usuarios.CamposPermitidosCorreos(accion))
	}
	rol := core.VersionRol{RolID: "candidato_usuarios_propios_desarrollo", Version: 1,
		Nombre: "areaPersonal.usuarios.rolPropio", Estado: core.EstadoVersionRolPublicada,
		Concesiones: concesiones, PublicadaPor: "seguridad:desarrollo:no-autoritativa", PublicadaEn: ahora}
	huella, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	return core.InstantaneaAutorizacion{
		VersionRol: rol,
		ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1,
			Estado: core.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: rol.PublicadaPor, ActualizadoEn: ahora},
		AsignacionPerfil: core.AsignacionPerfil{AsignacionID: "asg_usuarios_sintetica", Version: 1,
			PerfilActivoRef: "prf_usuarios_externo_sintetico", PrincipalID: "per_usuarios_sintetica",
			VersionRolRef: rol.Referencia(), Estado: core.EstadoAsignacionPerfilActiva,
			Ambitos:      []core.AmbitoPerfil{{Clave: "persona_ref", Valores: []string{"per_usuarios_sintetica"}}},
			VigenteDesde: ahora, VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "identidad:desarrollo:no-autoritativa", EmitidaEn: ahora},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella,
	}
}

func TestPerfilUsuariosExternoRechazaEstadosYCASNoCanonicos(t *testing.T) {
	i := instantaneaPerfilUsuariosExternoPrueba(t)
	for _, caso := range []string{"revocada", "retirado", "emisor ajeno", "cas negativo", "cas desbordado", "huella invalida", "revision desbordada"} {
		t.Run(caso, func(t *testing.T) {
			alterada := i
			version, revision := 0, uint64(0)
			ha, hc := "", ""
			switch caso {
			case "revocada":
				alterada.AsignacionPerfil.Estado = core.EstadoAsignacionPerfilRevocada
			case "retirado":
				alterada.ControlVigenciaVersionRol.Estado = core.EstadoControlVigenciaVersionRolRetirada
			case "emisor ajeno":
				alterada.VersionRol.PublicadaPor = "seguridad:ajena"
			case "cas negativo":
				version = -1
			case "cas desbordado":
				version = math.MaxInt
			case "revision desbordada":
				revision = math.MaxUint64
			case "huella invalida":
				version = 1
				ha = strings.Repeat("A", 64)
				alterada.AsignacionPerfil.Version = 2
			}
			if _, err := prepararPublicacionPerfilUsuariosExterno(alterada, "cta_usuarios_sintetica", "aprobacion:sintetica", revision, hc, version, ha); err == nil {
				t.Fatal("publicación inválida preparada")
			}
		})
	}
}

func TestPerfilUsuariosExternoExigeDiezAccionesExactasYCAS(t *testing.T) {
	i := instantaneaPerfilUsuariosExternoPrueba(t)
	plan, err := prepararPublicacionPerfilUsuariosExterno(i, "cta_usuarios_sintetica", "aprobacion:sintetica", 0, "", 0, "")
	if err != nil || plan.HuellaRol == "" || plan.HuellaControl == "" || plan.HuellaAsignacion == "" {
		t.Fatalf("plan válido rechazado: %v", err)
	}
	alterada := i
	alterada.VersionRol.Concesiones = append([]core.ConcesionRol(nil), i.VersionRol.Concesiones...)
	alterada.VersionRol.Concesiones[0].ModuloID = "bolsa"
	if _, err := prepararPublicacionPerfilUsuariosExterno(alterada, "cta_usuarios_sintetica", "aprobacion:sintetica", 0, "", 0, ""); err == nil {
		t.Fatal("se admitió concesión de Bolsa")
	}
	alterada = i
	alterada.AsignacionPerfil.Ambitos = []core.AmbitoPerfil{{Clave: "persona_ref", Valores: []string{"per_ajena_sintetica"}}}
	if _, err := prepararPublicacionPerfilUsuariosExterno(alterada, "cta_usuarios_sintetica", "aprobacion:sintetica", 0, "", 0, ""); err == nil {
		t.Fatal("se admitió persona ajena")
	}
	if _, err := prepararPublicacionPerfilUsuariosExterno(i, "cta_usuarios_sintetica", "aprobacion:sintetica", 1, "", 0, ""); err == nil {
		t.Fatal("se admitió CAS sin huella previa")
	}
}
