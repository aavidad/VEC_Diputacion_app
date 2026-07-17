package calculoexperienciaoficial

import (
	"fmt"
	"io"
	"log/slog"
)

const (
	textoClaveOculta     = "[CLAVE-EFECTO-CALCULO-EXPERIENCIA-OFICIAL-OCULTA]"
	textoIntencionOculta = "[INTENCION-CALCULO-EXPERIENCIA-OFICIAL-OCULTA]"
	textoReciboOculto    = "[RECIBO-CALCULO-EXPERIENCIA-OFICIAL-OCULTO]"
)

func (ClaveEfectoV1) String() string { return textoClaveOculta }
func (c ClaveEfectoV1) GoString() string {
	return c.String()
}
func (c ClaveEfectoV1) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (c ClaveEfectoV1) LogValue() slog.Value { return slog.StringValue(c.String()) }

func (IntencionResultadoV1) String() string { return textoIntencionOculta }
func (i IntencionResultadoV1) GoString() string {
	return i.String()
}
func (i IntencionResultadoV1) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, i.String())
}
func (i IntencionResultadoV1) LogValue() slog.Value { return slog.StringValue(i.String()) }

func (ReciboV1) String() string { return textoReciboOculto }
func (r ReciboV1) GoString() string {
	return r.String()
}
func (r ReciboV1) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ReciboV1) LogValue() slog.Value { return slog.StringValue(r.String()) }
