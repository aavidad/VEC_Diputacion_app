// Package adminselector traduce HTTP para seleccionar un único perfil ADMIN.
// La autoridad central conserva asignaciones, versión y auditoría.
package adminselector

import (
	"context"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
)

const (
	PrefijoV1     = "/api/admin/seleccion-perfil/v1"
	RutaPropios   = PrefijoV1 + "/propios"
	RutaSeleccion = PrefijoV1 + "/seleccion"
)

// FuenteObservacion solo se conecta a la frontera ADMIN del servidor. Debe
// verificar mTLS directo, certificado, CA, revocación, host y política de canal
// y cuenta para esta petición, antes de resolver un perfil activo. No depende
// del proveedor de Aplicación ni reconstruye identidad desde cabeceras libres.
type FuenteObservacion interface {
	ObservarADMIN(context.Context, *http.Request) (adminperfiles.ObservacionADMIN, error)
}

// Seleccionador es la autoridad durable: revalida cuenta administrativa,
// persona y asignación vigente, aplica CAS y registra selección o denegación
// en la misma transacción. ListarPropios audita la lectura y solo proyecta
// asignaciones de la persona y cuenta observadas. PerfilRef es una solicitud,
// nunca una concesión. Ningún doble sintético se monta en producción.
type Seleccionador interface {
	ListarPropiosADMIN(context.Context, adminperfiles.ObservacionADMIN) (adminperfiles.PerfilesPropios, error)
	SeleccionarPerfilADMIN(context.Context, adminperfiles.ObservacionADMIN, string, uint64) (adminperfiles.SeleccionPerfil, error)
}

type PerfilPropio struct {
	PerfilRef      string `json:"perfil_ref"`
	RolVersionRef  string `json:"rol_version_ref"`
	ClaveI18N      string `json:"clave_i18n"`
	CategoriaADMIN string `json:"categoria_admin"`
}

type PerfilesPropios struct {
	Revision        uint64         `json:"revision"`
	PerfilActivoRef string         `json:"perfil_activo_ref,omitempty"`
	Perfiles        []PerfilPropio `json:"perfiles"`
}

type Seleccion struct {
	PerfilActivoRef string    `json:"perfil_activo_ref"`
	Revision        uint64    `json:"revision"`
	AuditoriaRef    string    `json:"auditoria_ref"`
	SeleccionadaEn  time.Time `json:"seleccionada_en"`
}
