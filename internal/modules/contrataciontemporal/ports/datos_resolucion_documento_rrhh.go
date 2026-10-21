package ports

import "context"

// FuenteResolucionDocumentoRRHH recupera el acto por la lectura nominal de CT.
// Un resultado nulo significa que todavía no consta resolución registrada.
type FuenteResolucionDocumentoRRHH func(context.Context, string) (*ResultadoResolucionFormalizacion, error)

// DatosResolucionDocumentoRRHH conserva el compromiso del acto recuperado.
// Antes de registrar el acto no hay una huella confirmada que cotejar.
type DatosResolucionDocumentoRRHH struct {
	Solicitud      SolicitudResolucionFormalizacion
	OriginalSHA256 string
}

type claveResolucionDocumentoRRHH struct{}

// ConResolucionDocumentoRRHH lleva al renderizado previo los mismos datos
// validados que registrará la transacción. No concede acceso ni acredita el acto.
func ConResolucionDocumentoRRHH(ctx context.Context, s SolicitudResolucionFormalizacion) (context.Context, error) {
	if ctx == nil || s.Validar() != nil {
		return nil, ErrSolicitudResolucionFormalizacionInvalida
	}
	return context.WithValue(ctx, claveResolucionDocumentoRRHH{}, s), nil
}

// ResolucionParaDocumentoRRHH usa la solicitud en curso o recupera el recibo
// confirmado. Nunca deriva el número de la secuencia ni del expediente.
func ResolucionParaDocumentoRRHH(ctx context.Context, expediente string, fuente FuenteResolucionDocumentoRRHH) (*DatosResolucionDocumentoRRHH, error) {
	if ctx == nil {
		return nil, ErrSolicitudResolucionFormalizacionInvalida
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var huella string
	s, presente := ctx.Value(claveResolucionDocumentoRRHH{}).(SolicitudResolucionFormalizacion)
	if !presente && fuente != nil {
		r, err := fuente(ctx, expediente)
		if err != nil {
			return nil, err
		}
		if r == nil {
			return nil, nil
		}
		if r.ValidarPara(r.Solicitud) != nil {
			return nil, ErrResultadoResolucionFormalizacionNoConfiable
		}
		s, presente, huella = r.Solicitud, true, r.DocumentoSHA256
	}
	if !presente {
		return nil, nil
	}
	if s.Validar() != nil || s.ExpedienteRef != expediente {
		return nil, ErrResultadoResolucionFormalizacionNoConfiable
	}
	return &DatosResolucionDocumentoRRHH{Solicitud: s, OriginalSHA256: huella}, ctx.Err()
}
