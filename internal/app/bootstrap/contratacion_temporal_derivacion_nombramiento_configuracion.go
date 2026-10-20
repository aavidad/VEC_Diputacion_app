package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const archivoActivacionNombramientoCTDesarrollo = "identidad/nombramientos-ct.json"
const esquemaActivacionNombramientoCTDesarrollo = "vec.ct.nombramientos.activacion.v1"

type archivoActivacionNombramientoCT struct {
	Esquema                    string                   `json:"esquema"`
	Derivacion                 perfilActivacionOfertaCT `json:"derivacion"`
	FuenteAutorizacionDSNFile  string                   `json:"fuente_autorizacion_dsn_file"`
	MotivosAutorizacionDSNFile string                   `json:"motivos_autorizacion_dsn_file"`
}

func leerActivacionNombramientoCTDesarrollo(cfg config.Config) (archivoActivacionNombramientoCT, bool, error) {
	var vacia archivoActivacionNombramientoCT
	if !cfg.DevelopmentEnabledByDoubleKey() || !filepath.IsAbs(cfg.DevelopmentMaterialDir) ||
		validarArbolMaterialDesarrollo(cfg.DevelopmentMaterialDir) != nil {
		return vacia, false, errActivacionOfertaCTDesarrollo
	}
	raiz, err := os.OpenRoot(cfg.DevelopmentMaterialDir)
	if err != nil {
		return vacia, false, errActivacionOfertaCTDesarrollo
	}
	defer raiz.Close()
	if _, err := raiz.Lstat(archivoActivacionNombramientoCTDesarrollo); errors.Is(err, os.ErrNotExist) {
		return vacia, false, nil
	} else if err != nil {
		return vacia, false, errActivacionOfertaCTDesarrollo
	}
	b, err := leerArchivoIncorporacionV2(raiz, archivoActivacionNombramientoCTDesarrollo, 8<<10)
	if err != nil || validarClavesJSONUnicas(b) != nil {
		return vacia, true, errActivacionOfertaCTDesarrollo
	}
	defer borrarBytes(b)
	var c archivoActivacionNombramientoCT
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF ||
		c.Esquema != esquemaActivacionNombramientoCTDesarrollo ||
		!perfilActivacionOfertaCTValido(c.Derivacion) ||
		!filepath.IsLocal(c.FuenteAutorizacionDSNFile) ||
		!filepath.IsLocal(c.MotivosAutorizacionDSNFile) ||
		c.FuenteAutorizacionDSNFile == c.MotivosAutorizacionDSNFile {
		return vacia, true, errActivacionOfertaCTDesarrollo
	}
	return c, true, nil
}

func descriptorMaterialDerivacionNombramientoCTDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        ports.AudienciaDerivacionNombramiento,
		Dominio:          "vec.ct.nombramiento-derivado.capacidad-v3",
		Prefijo:          "clave:capacidad:ct:nombramiento-derivado:",
		ProveedorNominal: proveedorMaterialContratacionTemporal,
	}
}
