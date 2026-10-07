package validadorautofirma

import (
	"context"

	"vec-diputacion-granada/internal/vec/documentos/ports"
)

var _ ports.VerificadorFirmasDocumento = (*Cliente)(nil)

// VerificarFirmas conserva TODAS las revisiones del dictamen, incluidos sellos
// de documento. Nunca convierte la TSA en el firmante del tramite.
func (c *Cliente) VerificarFirmas(ctx context.Context, s ports.SolicitudVerificacionFirma) (ports.VerificacionFirmasDocumento, error) {
	d, fallo, err := c.verificarDictamenV2(ctx, s)
	if err != nil {
		return ports.VerificacionFirmasDocumento{}, err
	}
	if fallo != "" {
		return ports.VerificacionFirmasDocumento{
			Estado: ports.EstadoVerificacionIndeterminada, Motivo: fallo,
			HuellaOriginalSHA256: s.HuellaOriginalSHA256,
			HuellaFirmadoSHA256:  huellaBytesV2(s.ContenidoFirmado),
		}, nil
	}
	r := ports.VerificacionFirmasDocumento{
		Estado: ports.EstadoVerificacionFirma(d.Estado), Motivo: motivoPuertoV2(d),
		Formato: d.Formato, VinculoOriginal: d.VinculoOriginal.Estado == "acreditado",
		HuellaOriginalSHA256: d.HuellaOriginalSHA256, HuellaFirmadoSHA256: d.HuellaFirmadoSHA256,
		ComprobadoEn: d.ComprobadoEn,
		CambiosPosteriores: ports.CambioPosteriorFirmaPDF{
			Estado: d.CambiosPosteriores.Estado, Detalle: d.CambiosPosteriores.Detalle,
		},
	}
	if d.CambiosPosteriores.BytesNoFirmados != nil {
		n := *d.CambiosPosteriores.BytesNoFirmados
		if n < 0 {
			return falloMultipleV2(s), nil
		}
		b := uint64(n)
		r.CambiosPosteriores.BytesNoFirmados = &b
	}
	for _, f := range d.Firmas {
		if f.RevisionLongitud < 0 {
			return falloMultipleV2(s), nil
		}
		p := ports.FirmaPDFVerificada{
			Orden: f.Orden, RevisionLongitud: uint64(f.RevisionLongitud),
			RevisionHuellaSHA256:            f.RevisionHuellaSHA256,
			ContenidoFirmadoHuellaSHA256:    f.ContenidoFirmadoHuellaSHA256,
			CubreDocumentoCompletoHastaAqui: f.CubreDocumentoCompletoHastaAqui,
			CertificadoHuellaSHA256:         f.CertificadoHuellaSHA256,
			IntegridadEstado:                f.Integridad.Estado, CadenaEstado: f.Cadena.Estado,
			CertificadoEstado: f.Certificado.Estado, RevocacionEstado: f.Revocacion.Estado,
			SelloTiempoEstado: f.SelloTiempo.Estado, TipoFirma: f.TipoFirma, NivelDocMDP: f.NivelDocMDP,
			CambiosDesdeAnterior: ports.CambioFirmaPDF{Estado: f.CambiosDesdeAnterior.Estado, Detalle: f.CambiosDesdeAnterior.Detalle},
		}
		// Las conversiones son seguras tras limites enteros y de segmentos.
		for i, b := range f.ByteRange {
			if b < 0 {
				return falloMultipleV2(s), nil
			}
			p.ByteRange[i] = uint64(b)
		}
		if f.CertificadoHuellaSHA256 != "" {
			p.FirmanteRef = "ref:" + f.CertificadoHuellaSHA256
		}
		r.Firmas = append(r.Firmas, p)
	}
	return r, nil
}

func falloMultipleV2(s ports.SolicitudVerificacionFirma) ports.VerificacionFirmasDocumento {
	return ports.VerificacionFirmasDocumento{
		Estado: ports.EstadoVerificacionIndeterminada, Motivo: ports.MotivoRespuestaNoInterpretable,
		HuellaOriginalSHA256: s.HuellaOriginalSHA256, HuellaFirmadoSHA256: huellaBytesV2(s.ContenidoFirmado),
	}
}

// motivoPuertoV2 mantiene el estado declarado. Los motivos nuevos de GrxFirma
// se expresan con el motivo neutral de integridad y sus cambios por separado.
func motivoPuertoV2(d dictamenV2) ports.MotivoVerificacionFirma {
	if m := motivosDictamen[d.Motivo]; m != "" {
		return m
	}
	if d.Estado == "no_valida" {
		return ports.MotivoIntegridadNoValida
	}
	return ports.MotivoIntegridadParcial
}
