package config

import _ "embed"

// EnvBolsaAvisoOfertaEnlace selecciona el enlace HTTPS a Mi Bolsa que usa el
// aviso. Ausente o inválido deja la intención pendiente y el portal abierto.
const EnvBolsaAvisoOfertaEnlace = "VEC_BOLSA_AVISO_OFERTA_ENLACE"

//go:embed bolsa_aviso_oferta_v1.json
var catalogoAvisoOferta []byte

func CatalogoAvisoOfertaBolsa() []byte { return append([]byte(nil), catalogoAvisoOferta...) }
