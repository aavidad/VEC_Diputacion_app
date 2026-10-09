package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"strings"

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
	cancelacion                                                      bool
	vinculoEmisionBolsa                                              bool
	exportacionServiciosPersonal                                     bool
	historiaServiciosPersonal                                        bool
	historiaRelacionesPersonal                                       bool
	incorporacionAcreditada, incorporacionB2                         bool
	reincorporacionTitular                                           bool
	politicaOfertas                                                  bool
	plantillasCatalogo                                               bool
	plantillasDocumental                                             bool
	ajustesReglasCT                                                  bool
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
	incorporacionB2, _, err := protocolosIncorporacionConfiguradosDesarrollo(cfg)
	if err != nil {
		return s, err
	}
	reincorporacion, err := selectorCapacidadRRHHDesarrollo(cfg, envCTReincorporacionTitularEnabled)
	if err != nil || (reincorporacion && (!seguimientoCeseSolicitado(cfg) || !cfg.BolsaBorradoresEnabled)) {
		return s, ErrActivacionDesarrolloInvalida
	}
	politicaOfertas, err := selectorCapacidadRRHHDesarrollo(cfg, envBolsaPoliticaOfertasEnabled)
	if err != nil || (politicaOfertas && !cfg.BolsaBorradoresEnabled) {
		return s, ErrActivacionDesarrolloInvalida
	}
	plantillasCatalogo, err := plantillasCatalogoCTDesarrolloSolicitado(cfg)
	if err != nil {
		return s, ErrActivacionDesarrolloInvalida
	}
	plantillasDocumental, err := plantillasDocumentalCTDesarrolloSolicitado(cfg)
	if err != nil {
		return s, ErrActivacionDesarrolloInvalida
	}
	ajustesReglasCT, err := ajustesReglasCTSolicitados(cfg)
	if err != nil {
		return s, ErrActivacionDesarrolloInvalida
	}
	// Pedir el portal del candidato sin poder componer «Mi bolsa» (PostgreSQL
	// de llamamientos y material de identidad del candidato) no se ignora.
	if portal, _ := cfg.BolsaPortalCandidatoDesarrolloActivo(); portal && !debeComponerMiBolsaDesarrollo(cfg) {
		return s, fmt.Errorf("%w: falta Mi bolsa (PostgreSQL de llamamientos o identidad del candidato)",
			config.ErrConfiguracionBolsaPortalCandidatoActivacion)
	}
	exportacionServicios, err := exportacionServiciosPersonalSolicitada(cfg)
	if err != nil {
		return s, err
	}
	historiaServicios, err := historiaServiciosPersonalSolicitada(cfg)
	if err != nil {
		return s, err
	}
	historiaRelaciones, err := historiaRelacionesPersonalSolicitada(cfg)
	if err != nil {
		return s, err
	}
	s = seleccionMaterialCTDesarrollo{
		borradoresBolsa:              cfg.BolsaBorradoresEnabled,
		miBolsa:                      debeComponerMiBolsaDesarrollo(cfg),
		portalCandidato:              debeComponerPortalCandidatoDesarrollo(cfg),
		dietas:                       dietasBorradoresSolicitadas(cfg.DietasBorradoresEnabled),
		cronos:                       cronosEmpleadoSolicitado(cfg.CronosEmpleadoEnabled),
		documentos:                   documentosSolicitados(cfg.DocumentosEnabled),
		cronosResolucion:             cronosResolucionSolicitada(cfg.CronosEmpleadoEnabled, cfg.CronosResolucionEnabled),
		cronosAvisos:                 cronosNotificacionesSolicitadas(cfg.CronosEmpleadoEnabled, cfg.CronosNotificacionesEnabled),
		fichaPropiaPersonal:          personalEmpleadoSolicitado(cfg.PersonalEmpleadoEnabled),
		exportacionServiciosPersonal: exportacionServicios,
		historiaServiciosPersonal:    historiaServicios,
		historiaRelacionesPersonal:   historiaRelaciones,
		firmaDocumento:               firma,
		seguimientoCese:              seguimientoCeseSolicitado(cfg),
		cancelacion:                  cancelacionCTSolicitada(cfg),
		personalB2:                   personalB2,
		incorporacionB2:              incorporacionB2,
		incorporacionAcreditada:      incorporacionAcreditadaSolicitada(cfg),
		reincorporacionTitular:       reincorporacion,
		politicaOfertas:              politicaOfertas,
		plantillasCatalogo:           plantillasCatalogo,
		plantillasDocumental:         plantillasDocumental,
		ajustesReglasCT:              ajustesReglasCT,
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
	if _, err := cfg.CTCancelacionDesarrolloActivo(); err != nil {
		return err
	}
	if _, err := selectorCapacidadRRHHDesarrollo(cfg, envBolsaCeseCTEnabled); err != nil {
		return err
	}
	if _, err := selectorCapacidadRRHHDesarrollo(cfg, envCTReincorporacionTitularEnabled); err != nil {
		return err
	}
	if _, err := selectorCapacidadRRHHDesarrollo(cfg, envBolsaPoliticaOfertasEnabled); err != nil {
		return err
	}
	if _, err := selectorCapacidadRRHHDesarrollo(cfg, envCTPlantillasGobiernoEnabled); err != nil {
		return err
	}
	if _, err := selectorCapacidadRRHHDesarrollo(cfg, envCTPlantillasDocumentalEnabled); err != nil {
		return err
	}
	if _, err := selectorCapacidadRRHHDesarrollo(cfg, envCTAjustesReglasEnabled); err != nil {
		return err
	}
	if _, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosPreferenciasDesarrollo); err != nil {
		return err
	}
	if _, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosCorreosDesarrollo); err != nil {
		return err
	}
	if _, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosImagenDesarrollo); err != nil {
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
	if s.reincorporacionTitular {
		d = append(d, descriptorMaterialConsultaReincorporacionTitularBolsaDesarrollo())
	}
	if s.politicaOfertas {
		d = append(d, descriptorMaterialPoliticaOfertasBolsaDesarrollo(), descriptorMaterialConsultaPoliticaOfertasBolsaDesarrollo())
	}
	if s.miBolsa {
		d = append(d, descriptorMaterialMiBolsaDesarrollo())
		d = append(d, descriptorMaterialHistorialMiBolsaDesarrollo())
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
	if s.historiaServiciosPersonal {
		d = append(d, descriptorMaterialHistoriaServiciosPersonal())
	}
	if s.historiaRelacionesPersonal {
		d = append(d, descriptorMaterialHistoriaRelacionesPersonal())
	}
	if s.exportacionServiciosPersonal {
		d = append(d, descriptorMaterialExportacionServiciosPersonal())
	}
	if s.firmaDocumento {
		d = append(d, descriptorMaterialFirmaDocumentoCTDesarrollo(), descriptorMaterialConsultaFirmasDocumentoCTDesarrollo())
	}
	if s.seguimientoCese {
		d = append(d, descriptoresMaterialSeguimientoCeseDesarrollo()...)
	}
	if s.reincorporacionTitular {
		d = append(d, descriptorMaterialReincorporacionTitularDesarrollo(), descriptorMaterialLecturaReincorporacionTitularDesarrollo())
	}
	if s.incorporacionAcreditada {
		d = append(d, descriptorMaterialConfirmacionGINPIXDesarrollo(), descriptorMaterialNoIncorporacionDesarrollo())
	}
	if s.personalB2 {
		d = append(d, descriptoresMaterialPersonalB2Desarrollo()...)
	}
	if s.incorporacionB2 {
		d = append(d, descriptoresMaterialIncorporacionB2()...)
	}
	if s.cancelacion {
		d = append(d, descriptoresMaterialCancelacionCTDesarrollo()...)
	}
	if s.vinculoEmisionBolsa {
		d = append(d, descriptorMaterialVinculoEmisionBolsaDesarrollo())
	}
	if s.plantillasCatalogo {
		d = append(d, descriptoresMaterialPlantillasCTDesarrollo()...)
	}
	if s.plantillasDocumental {
		d = append(d, descriptorMaterialPlantillasDocumentalCTDesarrollo())
	}
	if s.ajustesReglasCT {
		d = append(d, descriptorMaterialAjustesReglasCT())
	}
	return d
}

// validarValorSelectoresDespliegueBolsaCT solo rechaza valores mal escritos.
// La usan las raíces que no componen estas capacidades (listener público,
// API pública), que comparten el entorno pero no la doble llave: allí un
// "true" no activa nada, pero un valor ilegible sigue siendo un error.
func validarValorSelectoresDespliegueBolsaCT(cfg config.Config) error {
	// Estas dos capacidades se seleccionan desde el entorno. Las raíces
	// públicas no aplican la doble llave, pero sí rechazan valores ilegibles.
	for _, nombre := range []string{envCTPlantillasGobiernoEnabled, envCTPlantillasDocumentalEnabled, envUsuariosPreferenciasDesarrollo, envUsuariosCorreosDesarrollo, envUsuariosImagenDesarrollo} {
		switch strings.TrimSpace(os.Getenv(nombre)) {
		case "", "false", "true":
		default:
			return ErrActivacionDesarrolloInvalida
		}
	}
	for _, err := range []error{
		func() error { _, err := cfg.BolsaPortalCandidatoDesarrolloActivo(); return err }(),
		func() error { _, err := cfg.CTSeguimientoCeseDesarrolloActivo(); return err }(),
		func() error { _, err := cfg.CTCancelacionDesarrolloActivo(); return err }(),
		func() error { _, err := cfg.CTIncorporacionAcreditadaDesarrolloActivo(); return err }(),
	} {
		if errors.Is(err, config.ErrConfiguracionBolsaPortalCandidatoSelector) ||
			errors.Is(err, config.ErrConfiguracionCTSeguimientoCeseSelector) ||
			errors.Is(err, config.ErrConfiguracionCTCancelacionSelector) ||
			errors.Is(err, config.ErrConfiguracionCTIncorporacionAcreditadaSelector) {
			return err
		}
	}
	return nil
}
