package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
)

const maxConfiguracionPreparacion = 16 << 10

// Solo indica cómo empaquetar material suministrado. Los actores, fechas y
// decisiones siguen procediendo de los tres ficheros canónicos originales.
type configuracionPreparacion struct {
	Operacion        string  `json:"operacion"`
	RevisionEsperada int     `json:"revision_esperada"`
	HuellaEsperada   *string `json:"huella_esperada"`
	ClaveOperacion   string  `json:"clave_operacion"`
}

func ejecutarPreparacion(args []string, salida io.Writer) int {
	if len(args) != 5 {
		registrarRechazo(errMaterial, "argumentos")
		return 2
	}
	limites := []int{maxConfiguracionPreparacion, 2 << 20, 65536, 65536}
	bloques := make([][]byte, len(limites))
	for i, limite := range limites {
		b, err := leerEntradaPreparacion(args[i], limite)
		if err != nil {
			return rechazar(err, "entrada")
		}
		bloques[i] = b
	}
	b, r, err := prepararMaterial(bloques[0], bloques[1], bloques[2], bloques[3])
	if err != nil {
		return rechazar(err, "validacion")
	}
	r.Estado = "preparado_sin_autorizacion"
	if err := conservarPreparacion(args[4], b, r, salida); err != nil {
		return rechazar(err, "salida")
	}
	return 0
}

func prepararMaterial(config, catalogo, traza, evento []byte) ([]byte, resumen, error) {
	var cero resumen
	if len(config) > maxConfiguracionPreparacion || len(catalogo) > 2<<20 || len(traza) > 65536 || len(evento) > 65536 || jsonUnico(config) != nil {
		return nil, cero, errMaterial
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(config, &raw) != nil || len(raw) != 4 {
		return nil, cero, errMaterial
	}
	for _, clave := range []string{"operacion", "revision_esperada", "huella_esperada", "clave_operacion"} {
		v := bytes.TrimSpace(raw[clave])
		if len(v) == 0 {
			return nil, cero, errMaterial
		}
		if clave == "huella_esperada" && bytes.Equal(v, []byte("null")) {
			continue
		}
		if clave == "revision_esperada" {
			if v[0] < '0' || v[0] > '9' {
				return nil, cero, errMaterial
			}
		} else if v[0] != '"' {
			return nil, cero, errMaterial
		}
	}
	var cfg configuracionPreparacion
	if decodificarEstricto(config, &cfg) != nil {
		return nil, cero, errMaterial
	}
	// Se extraen identificador y versión para la envoltura; no se vuelve a
	// serializar ningún bloque. validar coteja después todos los campos.
	var cabecera struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	if json.Unmarshal(catalogo, &cabecera) != nil {
		return nil, cero, errMaterial
	}
	m := material{
		Esquema: "vec.catalogos.plan-firma.gobierno.v1", Operacion: cfg.Operacion,
		CatalogoID: cabecera.ID, Version: cabecera.Version, RevisionEsperada: cfg.RevisionEsperada,
		HuellaEsperada: cfg.HuellaEsperada, ClaveOperacion: cfg.ClaveOperacion,
		CatalogoBase64: base64.StdEncoding.EncodeToString(catalogo), CatalogoSHA: huella(catalogo),
		TrazaBase64: base64.StdEncoding.EncodeToString(traza), TrazaSHA: huella(traza),
		EventoBase64: base64.StdEncoding.EncodeToString(evento), EventoSHA: huella(evento),
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, cero, err
	}
	r, err := validar(b, huella(b))
	if err != nil {
		return nil, cero, err
	}
	return b, r, nil
}
