package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"vec-diputacion-granada/config"
)

var ErrCustodiaImportacionConvocaNoDisponible = errors.New("bootstrap: custodia de importacion Convoca no disponible")

func custodiarImportacionConvocaDesarrollo(cfg config.Config, contenido []byte) (string, error) {
	if !cfg.DevelopmentEnabledByDoubleKey() || len(contenido) == 0 || len(contenido) > 16<<20 {
		return "", ErrCustodiaImportacionConvocaNoDisponible
	}
	s := sha256.Sum256(contenido)
	h := hex.EncodeToString(s[:])
	d := filepath.Join(cfg.DevelopmentMaterialDir, "importaciones")
	if err := os.MkdirAll(d, 0700); err != nil {
		return "", ErrCustodiaImportacionConvocaNoDisponible
	}
	p := filepath.Join(d, h+".xls")
	if err := guardarCustodiaNuevaImportacionConvoca(p, contenido); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return "", ErrCustodiaImportacionConvocaNoDisponible
		}
		if err := verificarCustodiaExistenteImportacionConvoca(p, s); err != nil {
			return "", ErrCustodiaImportacionConvocaNoDisponible
		}
	}
	return "fichero:sha256:" + h, nil
}

func guardarCustodiaNuevaImportacionConvoca(ruta string, contenido []byte) error {
	fichero, err := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0400)
	if err != nil {
		return err
	}
	if _, err = fichero.Write(contenido); err == nil {
		err = fichero.Sync()
	}
	cierre := fichero.Close()
	if err != nil {
		_ = os.Remove(ruta)
		return err
	}
	return cierre
}

func verificarCustodiaExistenteImportacionConvoca(ruta string, esperada [sha256.Size]byte) error {
	info, err := os.Lstat(ruta)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0400 || info.Size() < 1 || info.Size() > 16<<20 {
		return ErrCustodiaImportacionConvocaNoDisponible
	}
	existente, err := os.ReadFile(ruta)
	if err != nil {
		return err
	}
	defer borrarBytes(existente)
	if sha256.Sum256(existente) != esperada {
		return ErrCustodiaImportacionConvocaNoDisponible
	}
	return nil
}
