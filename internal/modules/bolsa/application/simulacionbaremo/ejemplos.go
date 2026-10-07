package simulacionbaremo

import _ "embed"

// Los ejemplos son totalmente sintéticos. Se embeben para que un adaptador
// local pueda simular sin admitir rutas, ficheros o méritos de personas.
//
//go:embed testdata/meritos_reglas_a.json
var ejemploReglasMeritos []byte

//go:embed testdata/meritos_entrada.json
var ejemploEntradaMeritos []byte

//go:embed testdata/reglas_a.json
var ejemploReglasExperiencia []byte

//go:embed testdata/entrada.json
var ejemploEntradaExperiencia []byte

// DatosEjemploMeritos devuelve copias independientes de las dos instantáneas.
func DatosEjemploMeritos() (reglas, entrada []byte) {
	return append([]byte(nil), ejemploReglasMeritos...), append([]byte(nil), ejemploEntradaMeritos...)
}
func DatosEjemploExperiencia() (reglas, entrada []byte) {
	return append([]byte(nil), ejemploReglasExperiencia...), append([]byte(nil), ejemploEntradaExperiencia...)
}
