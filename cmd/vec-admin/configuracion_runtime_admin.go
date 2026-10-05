package main

import (
	"encoding/hex"
	"strings"
)

// El runtime nominal tiene configuración propia. El formato base conserva sus
// campos y una fuente ausente nunca activa el constructor heredado.
type configuracionRuntimeADMIN struct {
	Version                      uint64 `json:"version"`
	PoolContexto                 string `json:"pool_contexto"`
	FuenteIdentificadoresArchivo string `json:"fuente_identificadores_archivo"`
	FuenteIdentificadoresSHA256  string `json:"fuente_identificadores_sha256"`
	ProcesoContexto              string `json:"proceso_contexto"`
}

func cargarConfiguracionRuntimeADMIN(ruta string, base configuracionPerfilesPrivada) (configuracionRuntimeADMIN, error) {
	b, err := leerArchivoPrivadoPerfiles(ruta)
	if err != nil {
		return configuracionRuntimeADMIN{}, errConfiguracionPrivadaPerfiles
	}
	defer clear(b)
	var c configuracionRuntimeADMIN
	if decodificarConfiguracionPrivada(b, &c) != nil || validarConfiguracionRuntimeADMIN(c, base) != nil {
		return configuracionRuntimeADMIN{}, errConfiguracionPrivadaPerfiles
	}
	return c, nil
}

func validarConfiguracionRuntimeADMIN(c configuracionRuntimeADMIN, base configuracionPerfilesPrivada) error {
	b, err := hex.DecodeString(c.FuenteIdentificadoresSHA256)
	if c.Version != 1 || !rutaPrivadaPerfilesValida(c.PoolContexto) || !rutaPrivadaPerfilesValida(c.FuenteIdentificadoresArchivo) ||
		c.PoolContexto == c.FuenteIdentificadoresArchivo || !procesoUsuarios.MatchString(c.ProcesoContexto) ||
		err != nil || len(b) != 32 || c.FuenteIdentificadoresSHA256 != strings.ToLower(c.FuenteIdentificadoresSHA256) || strings.Trim(c.FuenteIdentificadoresSHA256, "0") == "" {
		return errConfiguracionPrivadaPerfiles
	}
	for _, ruta := range []string{base.Pools.FuenteAutorizacion, base.Pools.RegistroAutorizacion, base.Pools.Motivos,
		base.Pools.RegistroSesiones, base.Pools.RevalidacionSesiones, base.Pools.CuentasADMIN, base.Pools.ActosADMIN,
		base.Pools.AuditoriaFrontera, base.Firmante.ClavePrivadaArchivo, base.Identidad.RutaConfiguracionHMAC} {
		if c.PoolContexto == ruta || c.FuenteIdentificadoresArchivo == ruta {
			return errConfiguracionPrivadaPerfiles
		}
	}
	return nil
}
