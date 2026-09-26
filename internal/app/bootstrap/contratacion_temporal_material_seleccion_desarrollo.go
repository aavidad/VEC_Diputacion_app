package bootstrap

import (
	"errors"
	"fmt"

	"vec-diputacion-granada/config"
)

// seleccionMaterialCTDesarrollo fija qué consumidores publica vec-server bajo
// la raíz de CT. Se calcula una vez desde la configuración y se traduce a
// descriptores con una función pura, de modo que las pruebas pueden componer
// la lista con todos los selectores encendidos y comprobar que cada
// audiencia es publicable por el gobierno de CT.
type seleccionMaterialCTDesarrollo struct {
	borradoresBolsa, miBolsa, portalCandidato                        bool
	dietas, cronos, documentos, cronosResolucion, cronosAvisos       bool
	fichaPropiaPersonal, firmaDocumento, seguimientoCese, personalB2 bool
	incorporacionAcreditada                                          bool
}

// seleccionMaterialCTDesarrolloDesdeConfig valida los selectores (un valor
// inválido o fuera de la doble llave detiene el arranque en lugar de
// interpretarse) y resuelve la selección.
func seleccionMaterialCTDesarrolloDesdeConfig(cfg config.Config) (seleccionMaterialCTDesarrollo, error) {
	var s seleccionMaterialCTDesarrollo
	if err := validarSelectoresDespliegueBolsaCT(cfg); err != nil {
		return s, err
	}
	firma, err := cfg.CTFirmaRegistroDesarrolloActivo()
	if err != nil {
		return s, err
	}
	personalB2, err := cfg.PersonalB2GobiernoDesarrolloActivo()
	if err != nil {
		return s, err
	}
	// Pedir el portal del candidato sin poder componer «Mi bolsa» (PostgreSQL
	// de llamamientos y material de identidad del candidato) no se ignora.
	if portal, _ := cfg.BolsaPortalCandidatoDesarrolloActivo(); portal && !debeComponerMiBolsaDesarrollo(cfg) {
		return s, fmt.Errorf("%w: falta Mi bolsa (PostgreSQL de llamamientos o identidad del candidato)",
			config.ErrConfiguracionBolsaPortalCandidatoActivacion)
	}
	s = seleccionMaterialCTDesarrollo{
		borradoresBolsa:         cfg.BolsaBorradoresEnabled,
		miBolsa:                 debeComponerMiBolsaDesarrollo(cfg),
		portalCandidato:         debeComponerPortalCandidatoDesarrollo(cfg),
		dietas:                  dietasBorradoresSolicitadas(cfg.DietasBorradoresEnabled),
		cronos:                  cronosEmpleadoSolicitado(cfg.CronosEmpleadoEnabled),
		documentos:              documentosSolicitados(cfg.DocumentosEnabled),
		cronosResolucion:        cronosResolucionSolicitada(cfg.CronosEmpleadoEnabled, cfg.CronosResolucionEnabled),
		cronosAvisos:            cronosNotificacionesSolicitadas(cfg.CronosEmpleadoEnabled, cfg.CronosNotificacionesEnabled),
		fichaPropiaPersonal:     personalEmpleadoSolicitado(cfg.PersonalEmpleadoEnabled),
		firmaDocumento:          firma,
		seguimientoCese:         seguimientoCeseSolicitado(cfg),
		incorporacionAcreditada: incorporacionAcreditadaSolicitada(cfg),
		personalB2:              personalB2,
	}
	return s, nil
}

// validarSelectoresDespliegueBolsaCT rechaza un valor inválido de los
// selectores del portal del candidato y del seguimiento de cese, y su
// encendido fuera de la doble llave de desarrollo. La raíz de vec-server la
// llama siempre, con o sin PostgreSQL de CT, para que un error tipográfico
// no pase en silencio.
func validarSelectoresDespliegueBolsaCT(cfg config.Config) error {
	if _, err := cfg.BolsaPortalCandidatoDesarrolloActivo(); err != nil {
		return err
	}
	if _, err := cfg.CTSeguimientoCeseDesarrolloActivo(); err != nil {
		return err
	}
	_, err := cfg.CTIncorporacionAcreditadaDesarrolloActivo()
	return err
}

// descriptoresMaterialSeleccionadosCTDesarrollo traduce la selección a la
// lista de descriptores del catálogo común de material.
func descriptoresMaterialSeleccionadosCTDesarrollo(s seleccionMaterialCTDesarrollo) []descriptorMaterialConsumidorV3Desarrollo {
	d := descriptoresMaterialAutorizacionContratacionTemporalDesarrollo()
	if s.borradoresBolsa {
		d = append(d, descriptoresMaterialBorradorLlamamientoBolsaDesarrollo()...)
	}
	if s.miBolsa {
		d = append(d, descriptorMaterialMiBolsaDesarrollo())
	}
	if s.portalCandidato {
		d = append(d, descriptoresMaterialPortalCandidatoDesarrollo()...)
		d = append(d, descriptoresMaterialContactoPropioDesarrollo()...)
	}
	if s.dietas {
		d = append(d, descriptoresMaterialDietasDesarrollo()...)
	}
	if s.cronos {
		d = append(d, descriptoresMaterialCronosDesarrollo()...)
	}
	if s.documentos {
		d = append(d, descriptoresMaterialDocumentosDesarrollo()...)
	}
	if s.cronosResolucion {
		d = append(d, descriptoresMaterialCronosResolucionDesarrollo()...)
	}
	if s.cronosAvisos {
		d = append(d, descriptoresMaterialCronosNotificacionesDesarrollo()...)
	}
	if s.fichaPropiaPersonal {
		d = append(d, descriptorMaterialFichaPropiaPersonalDesarrollo())
	}
	if s.firmaDocumento {
		d = append(d, descriptorMaterialFirmaDocumentoCTDesarrollo())
	}
	if s.seguimientoCese {
		d = append(d, descriptoresMaterialSeguimientoCeseDesarrollo()...)
	}
	if s.incorporacionAcreditada {
		d = append(d, descriptorMaterialConfirmacionGINPIXDesarrollo(), descriptorMaterialNoIncorporacionDesarrollo())
	}
	if s.personalB2 {
		d = append(d, descriptoresMaterialPersonalB2Desarrollo()...)
	}
	return d
}

// validarValorSelectoresDespliegueBolsaCT solo rechaza valores mal escritos.
// La usan las raíces que no componen estas capacidades (listener público,
// API pública), que comparten el entorno pero no la doble llave: allí un
// "true" no activa nada, pero un valor ilegible sigue siendo un error.
func validarValorSelectoresDespliegueBolsaCT(cfg config.Config) error {
	for _, err := range []error{
		func() error { _, err := cfg.BolsaPortalCandidatoDesarrolloActivo(); return err }(),
		func() error { _, err := cfg.CTSeguimientoCeseDesarrolloActivo(); return err }(),
		func() error { _, err := cfg.CTIncorporacionAcreditadaDesarrolloActivo(); return err }(),
	} {
		if errors.Is(err, config.ErrConfiguracionBolsaPortalCandidatoSelector) ||
			errors.Is(err, config.ErrConfiguracionCTSeguimientoCeseSelector) ||
			errors.Is(err, config.ErrConfiguracionCTIncorporacionAcreditadaSelector) {
			return err
		}
	}
	return nil
}
