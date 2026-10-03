package ports

import "context"

// El perfil operacional procede del contexto autenticado del registrador.
// La competencia del cargo firmante se acredita por una autoridad separada.
type FuentePerfilActivoOperadorFirmaV2 interface {
	ObtenerPerfilActivoOperadorFirmaV2(context.Context) (string, error)
}
