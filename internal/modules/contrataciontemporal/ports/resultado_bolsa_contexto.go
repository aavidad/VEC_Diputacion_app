package ports

import "context"

type claveResultadoBolsaRRHH struct{}

// La vista JSON de la ficha pide el complemento Bolsa; una descarga de
// borrador conserva la lectura documental sin consultar participaciones.
func ConResultadoBolsaRRHH(ctx context.Context) context.Context {
	if ctx == nil {
		return nil
	}
	return context.WithValue(ctx, claveResultadoBolsaRRHH{}, true)
}

func ResultadoBolsaRRHHSolicitado(ctx context.Context) bool {
	return ctx != nil && ctx.Value(claveResultadoBolsaRRHH{}) == true
}
