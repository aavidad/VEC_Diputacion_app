package ports

import (
	"context"
	"regexp"
	"strings"
	"time"
)

type claveResultadoBolsaRRHH struct{}
type peticionResultadoBolsaRRHH struct{ cursor string }

var patronCursorResultadoBolsaRRHH = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z#llamamiento:[0-9a-f]{64}$`)

func CursorResultadoBolsaRRHHValido(cursor string) bool {
	if !patronCursorResultadoBolsaRRHH.MatchString(cursor) {
		return false
	}
	instante, err := time.Parse("2006-01-02T15:04:05.000000Z", strings.SplitN(cursor, "#", 2)[0])
	return err == nil && !instante.IsZero()
}

// La vista JSON de la ficha pide el complemento Bolsa; una descarga de
// borrador conserva la lectura documental sin consultar participaciones.
func ConResultadoBolsaRRHH(ctx context.Context) context.Context {
	if ctx == nil {
		return nil
	}
	return context.WithValue(ctx, claveResultadoBolsaRRHH{}, peticionResultadoBolsaRRHH{})
}

func ResultadoBolsaRRHHSolicitado(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	_, ok := ctx.Value(claveResultadoBolsaRRHH{}).(peticionResultadoBolsaRRHH)
	return ok
}

func ConResultadoBolsaRRHHPagina(ctx context.Context, cursor string) context.Context {
	if ctx == nil || !CursorResultadoBolsaRRHHValido(cursor) {
		return nil
	}
	return context.WithValue(ctx, claveResultadoBolsaRRHH{}, peticionResultadoBolsaRRHH{cursor: cursor})
}

func CursorResultadoBolsaRRHH(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	peticion, ok := ctx.Value(claveResultadoBolsaRRHH{}).(peticionResultadoBolsaRRHH)
	if !ok {
		return ""
	}
	return peticion.cursor
}
