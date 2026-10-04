package ensayofisicopg

import "context"

// DatosContextoFisico pertenece al manifiesto que la aplicación ya descifró,
// autenticó y contrastó con el artefacto. No constituye otra autorización.
type DatosContextoFisico struct {
	OperacionRef          string
	ConjuntoRef           string
	VentanaRef            string
	ManifiestoSHA256      string
	ArtefactoFisicoSHA256 string
	ImagenSHA256          string
}

// ContextoFisicoVerificado revalida el mismo vínculo antes de cada observación.
// Su implementación pertenece a la composición con CS03, nunca al navegador.
type ContextoFisicoVerificado interface {
	RevalidarContextoFisico(context.Context) (DatosContextoFisico, error)
}
type ProcedenciaFisica struct {
	OperacionRef          string
	ConjuntoRef           string
	VentanaRef            string
	ManifiestoSHA256      string
	ArtefactoFisicoSHA256 string
	ImagenSHA256          string
	SelloExclusion        string
}
