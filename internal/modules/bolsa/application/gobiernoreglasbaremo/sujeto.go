package gobiernoreglasbaremo

import "strings"

const prefijoSujetoSeudonimoHMAC = "hmac-sha256:reglas_baremo_v2:"

// SujetoSeudonimoHMAC es un compromiso opaco sobre el sujeto del efecto. No
// es autoridad ni identifica al actor. Un futuro puerto confiable debe
// acunarlo y cotejarlo, con la separacion de dominio fija de este modulo,
// sobre tenant+convocatoria+expediente+contenido; un handler nunca debe
// calcularlo ni decidir su valor.
type SujetoSeudonimoHMAC struct {
	bloqueoSerializacion
	valor string
}

// RestaurarSujetoSeudonimoHMAC valida la representacion recibida de la
// frontera confiable. No calcula ni acredita el HMAC.
func RestaurarSujetoSeudonimoHMAC(valor string) (SujetoSeudonimoHMAC, error) {
	sujeto := SujetoSeudonimoHMAC{valor: valor}
	if sujeto.validar() != nil {
		return SujetoSeudonimoHMAC{}, ErrSujetoSeudonimoHMACInvalido
	}
	return sujeto, nil
}

// ValorCanonico permite al adaptador confiable cotejar o persistir el
// compromiso por una via explicita; los codecs y registros siguen cerrados.
func (s SujetoSeudonimoHMAC) ValorCanonico() (string, error) {
	if s.validar() != nil {
		return "", ErrSujetoSeudonimoHMACInvalido
	}
	return s.valor, nil
}

func (s SujetoSeudonimoHMAC) validar() error {
	if !sujetoSeudonimoHMACValido(s.valor) {
		return ErrSujetoSeudonimoHMACInvalido
	}
	return nil
}

func sujetoSeudonimoHMACValido(valor string) bool {
	if !strings.HasPrefix(valor, prefijoSujetoSeudonimoHMAC) {
		return false
	}
	huella := valor[len(prefijoSujetoSeudonimoHMAC):]
	return huellaSHA256Valida(huella)
}
