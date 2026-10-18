package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"time"

	"vec-diputacion-granada/config"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vecapp "vec-diputacion-granada/internal/vec/application"
)

const envCTFirmaVecR5Enabled = "VEC_CT_FIRMA_VEC_R5_ENABLED"

var errMontajeFirmaVecR5 = errors.New("contratacion temporal: montaje nominal firma VEC R5 no disponible")

type configuracionPlanFirmaCT struct {
	Archivo      string `json:"archivo"`
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type configuracionAdmisionFirmaCT struct {
	PerfilRef       string `json:"perfil_ref"`
	Referencia      string `json:"referencia"`
	Version         uint64 `json:"version"`
	HuellaSHA256    string `json:"huella_sha256"`
	RolVersionRef   string `json:"rol_version_ref"`
	RolSHA256       string `json:"rol_sha256"`
	ControlRevision uint64 `json:"control_revision"`
	ControlSHA256   string `json:"control_sha256"`
}

type configuracionFirmasCT struct {
	configuracionSesionFirmanteV2
	Plan       configuracionPlanFirmaCT       `json:"plan"`
	Admisiones []configuracionAdmisionFirmaCT `json:"admisiones"`
}

func leerConfiguracionFirmaVecR5(cfg config.Config) (configuracionFirmasCT, bool, error) {
	var c configuracionFirmasCT
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envCTFirmaVecR5Enabled)
	if err != nil || !activo {
		return c, activo, err
	}
	contenido, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "firma-vec.json"), 64<<10)
	if err != nil {
		return c, true, marcarFalloComponenteArranque("ct_firma_vec_configuracion", &falloFuenteConfiguracionArranque{causa: err})
	}
	defer borrarBytes(contenido)
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.DisallowUnknownFields()
	if validarClavesJSONUnicas(contenido) != nil || d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF ||
		c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa || c.PoliticaAutenticacionRef == "" || !huellaSHA256ValidaContratacionTemporalDesarrollo(c.PoliticaAutenticacionSHA256) ||
		!filepath.IsAbs(c.CRLPath) || filepath.Clean(c.CRLPath) != c.CRLPath || len(c.Firmantes) == 0 || len(c.Firmantes) > 64 ||
		!filepath.IsAbs(c.Plan.Archivo) || filepath.Clean(c.Plan.Archivo) != c.Plan.Archivo ||
		!(ctdomain.VersionPlanFirmaV2{Referencia: c.Plan.Referencia, Version: c.Plan.Version, HuellaSHA256: c.Plan.HuellaSHA256}).Valida() || len(c.Admisiones) != len(c.Firmantes) {
		return configuracionFirmasCT{}, true, marcarFalloComponenteArranque("ct_firma_vec_configuracion", &falloFuenteConfiguracionArranque{causa: errMontajeFirmaVecR5})
	}
	retirada, err := time.Parse(time.RFC3339, c.RetiradaEn)
	if err != nil || retirada.Location() != time.UTC || retirada.Format(time.RFC3339) != c.RetiradaEn || !time.Now().Before(retirada) {
		return configuracionFirmasCT{}, true, marcarFalloComponenteArranque("ct_firma_vec_retirada", &falloFuenteConfiguracionArranque{causa: errMontajeFirmaVecR5})
	}
	porPerfil := make(map[string]bool, len(c.Firmantes))
	slots := make(map[string]bool, len(c.Firmantes)*4)
	for _, s := range c.Firmantes {
		if !perfilActivoSeguridadComunValido(s.PerfilRef) || porPerfil[s.PerfilRef] {
			return configuracionFirmasCT{}, true, errMontajeFirmaVecR5
		}
		for _, perfil := range []string{s.PerfilRef, s.PerfilConsultaCTRef, s.PerfilConsultaCircuitoRef, s.PerfilDocumentosRef} {
			if !perfilActivoSeguridadComunValido(perfil) || slots[perfil] {
				return configuracionFirmasCT{}, true, errMontajeFirmaVecR5
			}
			slots[perfil] = true
		}
		porPerfil[s.PerfilRef] = true
	}
	for _, a := range c.Admisiones {
		if !porPerfil[a.PerfilRef] {
			return configuracionFirmasCT{}, true, errMontajeFirmaVecR5
		}
		delete(porPerfil, a.PerfilRef)
		if _, err := vecapp.NuevaAdmisionGarantiaFirmaVecDesarrollo(vecapp.ConfiguracionAdmisionGarantiaFirmaVecDesarrollo{EntornoConfiable: vecapp.EntornoAdmisionFirmaDesarrollo,
			Politica: politicaAdmisionFirmaCT(c, a, retirada), Reloj: relojContratacionTemporalDesarrollo{}}); err != nil {
			return configuracionFirmasCT{}, true, err
		}
	}
	return c, true, nil
}

func politicaAdmisionFirmaCT(c configuracionFirmasCT, a configuracionAdmisionFirmaCT, retirada time.Time) vecapp.PoliticaPrivadaAdmisionFirmaDesarrollo {
	return vecapp.PoliticaPrivadaAdmisionFirmaDesarrollo{Referencia: a.Referencia, Version: a.Version, HuellaSHA256: a.HuellaSHA256, RetiradaEn: retirada,
		PoliticaAutenticacionRef: c.PoliticaAutenticacionRef, PoliticaAutenticacionSHA256: c.PoliticaAutenticacionSHA256,
		RolVersionRef: a.RolVersionRef, RolSHA256: a.RolSHA256, ControlRevision: a.ControlRevision, ControlSHA256: a.ControlSHA256}
}
