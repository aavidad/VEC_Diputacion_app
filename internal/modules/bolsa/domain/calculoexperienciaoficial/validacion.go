package calculoexperienciaoficial

import (
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

const prefijoSujetoPseudonimizadoHMACV1 = "hmac-sha256:"

func validarReferencia(referencia ReferenciaExactaV1, campo string) error {
	if !referenciaOpacaValida(referencia.Referencia) || referencia.Version == 0 ||
		referencia.Version > maximoVersionV1 || !huellaSHA256Valida(referencia.HuellaSHA256) {
		return nuevoError(campo, CodigoValorNoCanonico)
	}
	return nil
}

func validarReglas(reglas VinculoReglasV1) error {
	if err := validarReferencia(reglas.Contenido, "clave.reglas.contenido"); err != nil {
		return err
	}
	if reglas.Revision == 0 || reglas.Revision > maximoVersionV1 ||
		!huellaSHA256Valida(reglas.HuellaEstadoSHA256) {
		return nuevoError("clave.reglas", CodigoValorNoCanonico)
	}
	return nil
}

func validarEntrada(entrada VinculoEntradaV1) error {
	if err := validarReferencia(entrada.Instantanea, "clave.entrada.instantanea"); err != nil {
		return err
	}
	if !huellaSHA256Valida(entrada.HuellaContenidoSHA256) {
		return nuevoError("clave.entrada", CodigoValorNoCanonico)
	}
	return nil
}

func validarMotor(motor VinculoMotorV1) error {
	if !tokenTecnicoValido(motor.Contrato, 128) || motor.Version == 0 ||
		motor.Version > maximoVersionV1 || !huellaSHA256Valida(motor.HuellaContratoSHA256) {
		return nuevoError("clave.motor", CodigoValorNoCanonico)
	}
	return nil
}

func validarCausa(causa CausaGobernadaV1) error {
	if err := validarReferencia(causa.Catalogo, "clave.causa.catalogo"); err != nil {
		return err
	}
	if !claveCatalogadaValida(causa.Clave) {
		return nuevoError("clave.causa.clave", CodigoValorNoCanonico)
	}
	return nil
}

func validarPredecesor(predecesor VinculoPredecesorV1) error {
	if !referenciaOpacaValida(predecesor.ReferenciaRecibo) ||
		!huellaSHA256Valida(predecesor.HuellaReciboSHA256) {
		return nuevoError("clave.predecesor", CodigoValorNoCanonico)
	}
	return nil
}

func validarDatosClave(datos DatosClaveEfectoV1) error {
	if err := validarReferencia(datos.SujetoPseudonimizado, "clave.sujeto_pseudonimizado"); err != nil {
		return err
	}
	if !referenciaSujetoPseudonimizadoHMACValida(datos.SujetoPseudonimizado.Referencia) {
		return nuevoError("clave.sujeto_pseudonimizado", CodigoValorNoCanonico)
	}
	if err := validarReferencia(datos.Convocatoria, "clave.convocatoria"); err != nil {
		return err
	}
	if err := validarReglas(datos.Reglas); err != nil {
		return err
	}
	if err := validarEntrada(datos.Entrada); err != nil {
		return err
	}
	if err := validarMotor(datos.Motor); err != nil {
		return err
	}
	if !huellaSHA256Valida(datos.HuellaPlanSHA256) {
		return nuevoError("clave.huella_plan_sha256", CodigoValorNoCanonico)
	}
	if err := validarCausa(datos.Causa); err != nil {
		return err
	}
	if !referenciasClaveDistintas(datos) {
		return nuevoError("clave.referencias", CodigoValorInvalido)
	}
	switch datos.Tipo {
	case EfectoCalculoInicial:
		if datos.Predecesor != nil {
			return nuevoError("clave.predecesor", CodigoEstadoIncompatible)
		}
	case EfectoRectificacion:
		if datos.Predecesor == nil {
			return nuevoError("clave.predecesor", CodigoEstadoIncompatible)
		}
		if err := validarPredecesor(*datos.Predecesor); err != nil {
			return err
		}
	default:
		return nuevoError("clave.tipo", CodigoValorNoCanonico)
	}
	return nil
}

// referenciaSujetoPseudonimizadoHMACValida aplica la misma gramatica cerrada
// del seudonimo HMAC de baremacion sin acoplar el dominio a sus puertos.
func referenciaSujetoPseudonimizadoHMACValida(valor string) bool {
	if !strings.HasPrefix(valor, prefijoSujetoPseudonimizadoHMACV1) {
		return false
	}
	partes := strings.Split(strings.TrimPrefix(valor, prefijoSujetoPseudonimizadoHMACV1), ":")
	return len(partes) == 2 && claveCatalogadaValida(partes[0]) &&
		huellaSHA256Valida(partes[1])
}

func referenciasClaveDistintas(datos DatosClaveEfectoV1) bool {
	vistas := []string{
		datos.SujetoPseudonimizado.Referencia,
		datos.Convocatoria.Referencia,
		datos.Reglas.Contenido.Referencia,
		datos.Entrada.Instantanea.Referencia,
		datos.Causa.Catalogo.Referencia,
	}
	if datos.Predecesor != nil {
		vistas = append(vistas, datos.Predecesor.ReferenciaRecibo)
	}
	vistos := make(map[string]struct{}, len(vistas))
	for _, vista := range vistas {
		if _, existe := vistos[vista]; existe {
			return false
		}
		vistos[vista] = struct{}{}
	}
	return true
}

func estadoYFaseValidos(estado EstadoResultadoV1, fase FaseResultadoV1) bool {
	if estado == ResultadoCompletado {
		return fase == FaseCompletado
	}
	if estado != ResultadoBloqueado {
		return false
	}
	return fase == FaseSeleccion || fase == FaseIntervalos || fase == FasePuntuacion
}

func huellaSHA256Valida(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	bytes, err := hex.DecodeString(valor)
	if err != nil || len(bytes) != 32 {
		return false
	}
	for _, caracter := range valor {
		if caracter >= 'A' && caracter <= 'F' {
			return false
		}
	}
	return true
}

func referenciaOpacaValida(valor string) bool {
	if len(valor) == 0 || len(valor) > 512 || !utf8.ValidString(valor) ||
		!caracterAlfanumericoASCII(rune(valor[0])) {
		return false
	}
	for _, caracter := range valor {
		if !caracterReferenciaValido(caracter) {
			return false
		}
	}
	return true
}

func caracterReferenciaValido(caracter rune) bool {
	return caracterAlfanumericoASCII(caracter) ||
		caracter == ':' || caracter == '/' || caracter == '#' || caracter == '-' ||
		caracter == '_' || caracter == '.'
}

func caracterAlfanumericoASCII(caracter rune) bool {
	return caracter >= 'a' && caracter <= 'z' || caracter >= 'A' && caracter <= 'Z' ||
		caracter >= '0' && caracter <= '9'
}

func claveCatalogadaValida(valor string) bool {
	if len(valor) == 0 || len(valor) > 128 || valor[0] < 'a' || valor[0] > 'z' {
		return false
	}
	for _, caracter := range valor {
		if !(caracter >= 'a' && caracter <= 'z') && !(caracter >= '0' && caracter <= '9') &&
			caracter != '.' && caracter != '-' && caracter != '_' {
			return false
		}
	}
	return true
}

func tokenTecnicoValido(valor string, maximo int) bool {
	if len(valor) == 0 || len(valor) > maximo || !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if !(caracter >= 'a' && caracter <= 'z') && !(caracter >= '0' && caracter <= '9') &&
			caracter != '.' && caracter != ':' && caracter != '-' && caracter != '_' {
			return false
		}
	}
	return true
}
