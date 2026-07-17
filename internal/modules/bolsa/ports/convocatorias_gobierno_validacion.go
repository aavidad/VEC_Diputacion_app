package ports

import (
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

func estadoVersionConvocatoria(
	version dominiobolsa.VersionConvocatoriaGobernada,
) (ReferenciaEstadoVersionConvocatoria, error) {
	return EstadoVersionConvocatoria(version)
}

func clonarEstadoVersion(
	origen *ReferenciaEstadoVersionConvocatoria,
) *ReferenciaEstadoVersionConvocatoria {
	if origen == nil {
		return nil
	}
	copia := *origen
	return &copia
}

func materialesIntencionConvocatoriaIguales(
	primero, segundo MaterialIntencionGobiernoConvocatoria,
) bool {
	huellaPrimera, errPrimera := primero.HuellaSHA256()
	huellaSegunda, errSegunda := segundo.HuellaSHA256()
	return errPrimera == nil && errSegunda == nil && huellaPrimera == huellaSegunda
}

func predecesoraExactaParaBorradorSucesor(
	sucesora, predecesora dominiobolsa.VersionConvocatoriaGobernada,
) bool {
	estadoPredecesoraValido := predecesora.EstadoGobierno == dominiobolsa.EstadoGobiernoConvocatoriaPublicada ||
		predecesora.EstadoGobierno == dominiobolsa.EstadoGobiernoConvocatoriaRetirada
	return estadoPredecesoraValido && relacionVersionesSucesorasExacta(sucesora, predecesora)
}

func relacionVersionesSucesorasExacta(
	sucesora, predecesora dominiobolsa.VersionConvocatoriaGobernada,
) bool {
	noAntes := predecesora.PublicadaEn
	if predecesora.EstadoGobierno == dominiobolsa.EstadoGobiernoConvocatoriaRetirada {
		noAntes = predecesora.RetiradaEn
	}
	return predecesora.Validar() == nil &&
		sucesora.ID == predecesora.ID && sucesora.Secuencia == predecesora.Secuencia+1 &&
		sucesora.CodigoVersionPublica != predecesora.CodigoVersionPublica &&
		sucesora.VersionAnteriorRef == predecesora.Referencia() &&
		sucesora.Contenido.IdentificadorPublico == predecesora.Contenido.IdentificadorPublico &&
		sucesora.InstanciaFlujoRef == predecesora.InstanciaFlujoRef &&
		sucesora.AmbitoOrganizativo == predecesora.AmbitoOrganizativo &&
		sucesora.Configuracion.FlujoProceso == predecesora.Configuracion.FlujoProceso &&
		!sucesora.CreadaEn.Before(noAntes)
}

func parejaPublicacionSucesoraExacta(
	publicada, predecesora dominiobolsa.VersionConvocatoriaGobernada,
) bool {
	if !relacionVersionesSucesorasExacta(publicada, predecesora) {
		return false
	}
	switch predecesora.EstadoGobierno {
	case dominiobolsa.EstadoGobiernoConvocatoriaSustituida:
		return predecesora.SustituidaPorRef == publicada.Referencia() &&
			predecesora.SustituidaPor == publicada.PublicadaPor &&
			predecesora.SustituidaEn.Equal(publicada.PublicadaEn)
	case dominiobolsa.EstadoGobiernoConvocatoriaRetirada:
		return true
	default:
		return false
	}
}

func atestacionVerificacionVigenteEn(emitidaEn, validaHasta, instante time.Time) bool {
	return instanteGobiernoConvocatoriaCanonico(instante) && !instante.Before(emitidaEn) &&
		instante.Before(validaHasta)
}

func referenciaGobiernoConvocatoriaValida(valor string) bool {
	if valor == "" || len(valor) > 512 || valor != strings.TrimSpace(valor) ||
		!utf8.ValidString(valor) || strings.ContainsRune(valor, '*') {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.IsSpace(caracter) ||
			unicode.Is(unicode.Bidi_Control, caracter) || caracter == unicode.ReplacementChar {
			return false
		}
	}
	return true
}

func referenciaVersionGobernadaConvocatoriaValida(valor string) bool {
	indice := strings.LastIndexByte(valor, '#')
	if indice < 1 || indice == len(valor)-1 || !referenciaGobiernoConvocatoriaValida(valor[:indice]) {
		return false
	}
	secuencia, err := strconv.Atoi(valor[indice+1:])
	return err == nil && secuencia > 0 && strconv.Itoa(secuencia) == valor[indice+1:]
}

func huellaGobiernoConvocatoriaValida(valor string) bool     { return huellaSHA256Valida(valor) }
func huellaHMACGobiernoConvocatoriaValida(valor string) bool { return huellaHMACSHA256Valida(valor) }

func instanteGobiernoConvocatoriaCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC && instante.Nanosecond()%1_000 == 0
}

func claveIdempotenciaConvocatoriaValida(valor string) bool {
	return len(valor) >= 32 && len(valor) <= 128 && referenciaGobiernoConvocatoriaValida(valor)
}

func numeroDecimalConvocatoria(valor int) string { return strconv.Itoa(valor) }
