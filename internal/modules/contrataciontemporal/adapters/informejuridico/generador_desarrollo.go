package informejuridico

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const rotuloDocumentoDesarrollo = "DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA"

// GeneradorDesarrollo materializa un artefacto de texto determinista para el
// recorrido sintetico. Implementa el mismo puerto que usara el servicio
// documental comun cuando su composicion completa este disponible.
type GeneradorDesarrollo struct{}

func (GeneradorDesarrollo) GenerarDocumentoInformeJuridico(
	ctx context.Context,
	solicitud ports.SolicitudGenerarDocumentoInformeJuridico,
) (ports.DocumentoInformeJuridico, error) {
	if ctx == nil || solicitud.Borrador.Validar() != nil {
		return ports.DocumentoInformeJuridico{}, ErrPaqueteDatosInformeJuridicoInvalido
	}
	if err := ctx.Err(); err != nil {
		return ports.DocumentoInformeJuridico{}, err
	}
	paquete, err := GenerarPaqueteDatos(solicitud.Borrador)
	if err != nil {
		return ports.DocumentoInformeJuridico{}, err
	}
	contenidoJSON, err := paquete.ContenidoJSON()
	if err != nil {
		return ports.DocumentoInformeJuridico{}, err
	}
	contenido := fmt.Sprintf(
		"%s\n\nINFORME JURIDICO PROVISIONAL\nPendiente de revision y firma.\n\nDatos juridicos canónicos:\n%s\n",
		rotuloDocumentoDesarrollo,
		contenidoJSON,
	)
	suma := sha256.Sum256([]byte(contenido))
	huellaPaquete, err := paquete.HuellaSHA256()
	if err != nil {
		return ports.DocumentoInformeJuridico{}, err
	}
	resultado := ports.DocumentoInformeJuridico{
		DocumentoRef:          solicitud.DocumentoRef,
		VersionDocumento:      1,
		Formato:               ports.FormatoInformeJuridicoDesarrollo,
		Nombre:                "informe-juridico-desarrollo.txt",
		HuellaDocumentoSHA256: hex.EncodeToString(suma[:]),
		HuellaPaqueteSHA256:   huellaPaquete,
		ContenidoDesarrollo:   contenido,
	}
	if resultado.ValidarPara(solicitud) != nil {
		return ports.DocumentoInformeJuridico{}, ErrPaqueteDatosInformeJuridicoInvalido
	}
	return resultado, nil
}
