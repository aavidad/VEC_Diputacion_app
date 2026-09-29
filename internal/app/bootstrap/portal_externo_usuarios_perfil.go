package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"

	usuarios "vec-diputacion-granada/internal/modules/usuarios/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

var errPerfilUsuariosExterno = errors.New("bootstrap: perfil externo de Usuarios no disponible")

// publicacionPerfilUsuariosExterno contiene únicamente material para una
// operación administrativa explícita. Construirlo no publica ningún permiso.
type publicacionPerfilUsuariosExterno struct {
	CuentaRef, AprobacionRef                   string
	Rol, Control, Asignacion                   []byte
	HuellaRol, HuellaControl, HuellaAsignacion string
	RevisionEsperada                           uint64
	HuellaControlEsperada                      string
	VersionEsperada                            int
	HuellaAsignacionEsperada                   string
}

func prepararPublicacionPerfilUsuariosExterno(i core.InstantaneaAutorizacion, cuentaRef, aprobacionRef string,
	revisionEsperada uint64, huellaControlEsperada string, versionEsperada int, huellaAsignacionEsperada string,
) (publicacionPerfilUsuariosExterno, error) {
	vacia := publicacionPerfilUsuariosExterno{}
	if cuentaRef == "" || aprobacionRef == "" || i.Validar() != nil ||
		i.VersionRol.RolID != "candidato_usuarios_propios_desarrollo" ||
		i.VersionRol.Nombre != "areaPersonal.usuarios.rolPropio" ||
		i.VersionRol.Estado != core.EstadoVersionRolPublicada ||
		i.AsignacionPerfil.VersionRolRef != i.VersionRol.Referencia() ||
		i.AsignacionPerfil.Version != versionEsperada+1 ||
		i.ControlVigenciaVersionRol.Revision != revisionEsperada+1 ||
		len(i.AsignacionPerfil.Ambitos) != 1 ||
		i.AsignacionPerfil.Ambitos[0].Clave != "persona_ref" ||
		!reflect.DeepEqual(i.AsignacionPerfil.Ambitos[0].Valores, []string{i.AsignacionPerfil.PrincipalID}) ||
		(revisionEsperada == 0) != (huellaControlEsperada == "") ||
		(versionEsperada == 0) != (huellaAsignacionEsperada == "") ||
		!concesionesPerfilUsuariosExternoExactas(i.VersionRol.Concesiones) {
		return vacia, errPerfilUsuariosExterno
	}
	rol, e1 := json.Marshal(i.VersionRol)
	control, e2 := json.Marshal(i.ControlVigenciaVersionRol)
	asignacion, e3 := json.Marshal(i.AsignacionPerfil)
	if e1 != nil || e2 != nil || e3 != nil {
		return vacia, errPerfilUsuariosExterno
	}
	huella := func(b []byte) string { suma := sha256.Sum256(b); return hex.EncodeToString(suma[:]) }
	return publicacionPerfilUsuariosExterno{
		CuentaRef: cuentaRef, AprobacionRef: aprobacionRef,
		Rol: rol, Control: control, Asignacion: asignacion,
		HuellaRol: huella(rol), HuellaControl: huella(control), HuellaAsignacion: huella(asignacion),
		RevisionEsperada: revisionEsperada, HuellaControlEsperada: huellaControlEsperada,
		VersionEsperada: versionEsperada, HuellaAsignacionEsperada: huellaAsignacionEsperada,
	}, nil
}

func concesionesPerfilUsuariosExternoExactas(recibidas []core.ConcesionRol) bool {
	esperadas := map[string]core.ConcesionRol{}
	annadir := func(accion, tipo, finalidad string, campos []string) {
		esperadas[accion] = core.ConcesionRol{Accion: accion, ModuloID: "usuarios", TipoRecurso: tipo,
			Finalidades: []string{finalidad}, CamposPermitidos: campos, GarantiaMinima: core.AuthAssuranceHigh}
	}
	annadir(usuarios.AccionConsultarPreferencias, usuarios.TipoRecursoPreferencias, usuarios.FinalidadPreferenciasPropias, []string{"catalogo", "valores", "version"})
	annadir(usuarios.AccionActualizarPreferencias, usuarios.TipoRecursoPreferencias, usuarios.FinalidadPreferenciasPropias, []string{"valores", "version"})
	for _, accion := range []string{usuarios.AccionConsultarImagen, usuarios.AccionActualizarImagen} {
		annadir(accion, usuarios.TipoRecursoImagen, usuarios.FinalidadImagenPropia, usuarios.CamposPermitidosImagen(accion))
	}
	for _, accion := range []string{usuarios.AccionConsultarCorreos, usuarios.AccionAnadirCorreo, usuarios.AccionReenviarCorreo,
		usuarios.AccionVerificarCorreo, usuarios.AccionActivarCorreo, usuarios.AccionRetirarCorreo} {
		annadir(accion, usuarios.TipoRecursoCorreos, usuarios.FinalidadCorreosPropios, usuarios.CamposPermitidosCorreos(accion))
	}
	if len(recibidas) != len(esperadas) {
		return false
	}
	for _, c := range recibidas {
		esperada, ok := esperadas[c.Accion]
		if !ok || !reflect.DeepEqual(c, esperada) {
			return false
		}
		delete(esperadas, c.Accion)
	}
	return len(esperadas) == 0
}
