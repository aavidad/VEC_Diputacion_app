package administracionperfiles

import (
	"regexp"

	"vec-diputacion-granada/internal/vec/ports"
)

var organizacionPrivadaLote = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,127}$`)

// NuevoHandlerLoteOrdinario fija la organización fuera de la petición. Los
// demás métodos del servicio conservan sus contratos y permisos propios.
func NuevoHandlerLoteOrdinario(organizacion, origen string, sesiones ResolvedorSesion,
	lecturas FuenteLecturas, catalogo ports.CatalogoRolesAdministrables,
	actos ServicioActos, auditor AuditorFrontera) (*Handler, error) {
	if !organizacionPrivadaLote.MatchString(organizacion) {
		return nil, ErrConfiguracionIncompleta
	}
	h, err := NuevoHandler(origen, sesiones, lecturas, catalogo, actos, auditor)
	if err != nil {
		return nil, err
	}
	h.organizacionLote = organizacion
	return h, nil
}
