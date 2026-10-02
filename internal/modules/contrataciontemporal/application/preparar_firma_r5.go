package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

// preparacionFirmaR5 reúne solo hechos obtenidos de fuentes confiables. La
// historia y la política se leen después, ligadas al principal acreditado.
type preparacionFirmaR5 struct {
	original    ports.OriginalFirmaAutorizado
	dictamen    docports.VerificacionFirmaMotivada
	competencia ports.EvidenciaCompetenciaFirmante
}

func prepararFirmaR5(
	ctx context.Context, base *ServicioFirmaDocumento, fuente ports.FuenteCompetenciaFirmante,
	org, exp, documento, originalRef string, originalVersion uint64,
	pdf []byte, circuito domain.CircuitoFirma, paso domain.PasoCircuitoFirma,
) (preparacionFirmaR5, error) {
	var cero preparacionFirmaR5
	original, err := obtenerOriginalFirmaAutorizado(ctx, base.original, ports.SolicitudOriginalFirma{
		OrganizacionRef: org, ExpedienteRef: exp, Documento: documento,
		OriginalRef: originalRef, OriginalVersion: originalVersion,
	})
	if err != nil {
		return cero, err
	}
	peticion := docports.SolicitudVerificacionFirma{
		DocumentoID: originalRef, Version: originalVersion, FormatoEsperado: "PAdES",
		HuellaOriginalSHA256: original.HuellaSHA256, ContenidoOriginal: original.Contenido,
		ContenidoFirmado: pdf,
	}
	dictamen, err := base.verificador.VerificarMotivado(ctx, peticion)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, DictamenRechazado{Estado: docports.EstadoVerificacionIndeterminada, Motivo: docports.MotivoRespuestaNoInterpretable}
	}
	if dictamen.ValidarContra(peticion) != nil || dictamen.Motivo != docports.MotivoFirmaVerificada {
		motivo := dictamen.Motivo
		if motivo.EstadoAsociado() == "" || motivo == docports.MotivoFirmaVerificada {
			motivo = docports.MotivoRespuestaNoInterpretable
		}
		return cero, DictamenRechazado{Estado: dictamen.Resultado.Estado, Motivo: motivo}
	}
	r := dictamen.Resultado
	q := ports.SolicitudCompetenciaFirmante{
		OrganizacionRef: org, ExpedienteRef: exp, Documento: documento,
		CatalogoRef: circuito.CatalogoRef, CatalogoHuella: circuito.HuellaCatalogo,
		PasoRef: paso.Referencia, PasoOrden: paso.Orden, CargoFirmante: paso.Cargo,
		PerfilFirmanteRef: paso.PerfilRef, FirmanteRef: r.FirmanteRef, CertificadoHuella: r.CertificadoHuellaSHA256,
	}
	evidencia, err := fuente.AcreditarCompetenciaFirmante(ctx, q)
	if err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		if errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) {
			return cero, ports.ErrCompetenciaFirmanteNoAcreditada
		}
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	if evidencia.Solicitud != q || !competenciaTemporalValida(evidencia) ||
		evidencia.CargoFirmante != paso.Cargo || evidencia.PerfilFirmanteRef != paso.PerfilRef {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	return preparacionFirmaR5{original: original, dictamen: dictamen, competencia: evidencia}, nil
}
