package ports

import (
	"errors"
	"testing"
)

func TestRecuperacionHistoricaNoCruzaRevisionesPAdESConLaMismaFirma(t *testing.T) {
	contenido := contenidoDecisionValidoPrueba(t)
	politica := politicaFirmaValidaPrueba()
	revisionB := artefactoFirmaValidoPrueba(t, contenido, politica, "inicial")
	sello := selloValidoPrueba(revisionB, politica)
	revisionT := sello.ArtefactoSellado
	revisionLTA := revisionT
	revisionLTA.DocumentoFirmadoRef += ":pades-lta"
	revisionLTA.HuellaDocumentoSHA256 = huellaPruebaPuertos("f")

	casos := []struct {
		nombre   string
		revision ArtefactoFirma
	}{
		{nombre: "PAdES-T", revision: revisionT},
		{nombre: "PAdES-LTA", revision: revisionLTA},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			if err := caso.revision.Validar(); err != nil {
				t.Fatalf("precondicion: revision %s invalida: %v", caso.nombre, err)
			}
			if caso.revision.FirmaRef != revisionB.FirmaRef ||
				caso.revision.HuellaFirmaSHA256 != revisionB.HuellaFirmaSHA256 {
				t.Fatalf("precondicion: la revision %s no conserva la identidad criptografica base", caso.nombre)
			}
			solicitudB := SolicitudRecuperarArtefactoFirma{
				Contexto: contextoFirmaValido(
					AccionRecuperarArtefactoFirmaBaremacion,
					revisionB.FirmaRef,
				),
				FirmaRef:              revisionB.FirmaRef,
				HuellaFirmaSHA256:     revisionB.HuellaFirmaSHA256,
				DocumentoFirmadoRef:   revisionB.DocumentoFirmadoRef,
				HuellaDocumentoSHA256: revisionB.HuellaDocumentoSHA256,
			}
			if err := solicitudB.Validar(); err != nil {
				t.Fatalf("precondicion: solicitud B exacta invalida: %v", err)
			}
			if err := caso.revision.ValidarRecuperacion(solicitudB); !errors.Is(err, ErrEvidenciaFirmaNoEncontrada) {
				t.Fatalf("la solicitud B recupero indebidamente la revision %s: %v", caso.nombre, err)
			}
		})
	}
}

func TestSolicitudRecuperarArtefactoAutorizaFirmaYSeleccionaRevisionExacta(t *testing.T) {
	contenido := contenidoDecisionValidoPrueba(t)
	politica := politicaFirmaValidaPrueba()
	artefacto := artefactoFirmaValidoPrueba(t, contenido, politica, "selector-exacto")
	solicitud := SolicitudRecuperarArtefactoFirma{
		Contexto: contextoFirmaValido(AccionRecuperarArtefactoFirmaBaremacion, artefacto.FirmaRef),
		FirmaRef: artefacto.FirmaRef, HuellaFirmaSHA256: artefacto.HuellaFirmaSHA256,
		DocumentoFirmadoRef:   artefacto.DocumentoFirmadoRef,
		HuellaDocumentoSHA256: artefacto.HuellaDocumentoSHA256,
	}
	if err := artefacto.ValidarRecuperacion(solicitud); err != nil {
		t.Fatalf("la firma autorizada no recupero su revision exacta: %v", err)
	}
	solicitud.Contexto = contextoFirmaValido(
		AccionRecuperarArtefactoFirmaBaremacion, artefacto.DocumentoFirmadoRef,
	)
	if err := solicitud.Validar(); !errors.Is(err, ErrSolicitudBaremacionInvalida) {
		t.Fatalf("se autorizo la recuperacion contra el documento en vez de FirmaRef: %v", err)
	}
}
