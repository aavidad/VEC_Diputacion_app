package gobiernoconvocatorias

import "errors"

var (
	ErrRotacionIdempotenciaInvalida = errors.New("gobierno convocatorias: rotacion de idempotencia invalida")
	ErrConsultaIdempotenciaAmbigua  = errors.New("gobierno convocatorias: consulta idempotente ambigua")
)

// IdentidadOperacionDerivada empareja L y F de una misma generacion. El orden
// del conjunto determina cual es primaria; no se aceptan banderas declaradas.
type IdentidadOperacionDerivada struct {
	bloqueoSerializacionDiario
	Localizador     LocalizadorOperacion
	HuellaSolicitud HuellaSolicitud
}

func NuevaIdentidadOperacionDerivada(
	localizador LocalizadorOperacion,
	huella HuellaSolicitud,
) (IdentidadOperacionDerivada, error) {
	resultado := IdentidadOperacionDerivada{Localizador: localizador, HuellaSolicitud: huella}
	if !resultado.valida() {
		return IdentidadOperacionDerivada{}, ErrRotacionIdempotenciaInvalida
	}
	return resultado, nil
}

func (i IdentidadOperacionDerivada) valida() bool {
	return i.Localizador.Valido() && i.HuellaSolicitud.Valida() &&
		i.Localizador.hmac.versionEsquema == i.HuellaSolicitud.hmac.versionEsquema &&
		i.Localizador.hmac.clave.generacionClave == i.HuellaSolicitud.hmac.clave.generacionClave
}

func (i IdentidadOperacionDerivada) generacion() uint32 {
	if !i.valida() {
		return 0
	}
	return i.Localizador.hmac.clave.generacionClave
}

// ConjuntoIdentidadesOperacion contiene la generacion primaria primero y un
// maximo acotado de generaciones historicas, en orden estrictamente decreciente.
type ConjuntoIdentidadesOperacion struct {
	bloqueoSerializacionDiario
	identidades []IdentidadOperacionDerivada
}

func NuevoConjuntoIdentidadesOperacion(
	identidades ...IdentidadOperacionDerivada,
) (ConjuntoIdentidadesOperacion, error) {
	resultado := ConjuntoIdentidadesOperacion{
		identidades: append([]IdentidadOperacionDerivada(nil), identidades...),
	}
	if !resultado.valido() {
		return ConjuntoIdentidadesOperacion{}, ErrRotacionIdempotenciaInvalida
	}
	return resultado, nil
}

func (c ConjuntoIdentidadesOperacion) valido() bool {
	if len(c.identidades) == 0 || len(c.identidades) > maximoIdentidadesRotacionBorrador {
		return false
	}
	for indice, identidad := range c.identidades {
		if !identidad.valida() {
			return false
		}
		if indice > 0 && c.identidades[indice-1].generacion() <= identidad.generacion() {
			return false
		}
		for anterior := 0; anterior < indice; anterior++ {
			otra := c.identidades[anterior]
			if identidad.Localizador.CoincideExactamente(otra.Localizador) ||
				identidad.HuellaSolicitud.CoincideExactamente(otra.HuellaSolicitud) {
				return false
			}
		}
	}
	return true
}

func (c ConjuntoIdentidadesOperacion) primaria() (ProyeccionIdentidadOperacion, error) {
	proyecciones, err := c.proyecciones()
	if err != nil {
		return ProyeccionIdentidadOperacion{}, err
	}
	return proyecciones[0], nil
}

func (c ConjuntoIdentidadesOperacion) proyecciones() ([]ProyeccionIdentidadOperacion, error) {
	if !c.valido() {
		return nil, ErrRotacionIdempotenciaInvalida
	}
	resultado := make([]ProyeccionIdentidadOperacion, 0, len(c.identidades))
	for _, identidad := range c.identidades {
		proyeccion, err := nuevaProyeccionIdentidadOperacion(
			identidad.Localizador, identidad.HuellaSolicitud,
		)
		if err != nil {
			return nil, ErrRotacionIdempotenciaInvalida
		}
		resultado = append(resultado, proyeccion)
	}
	return resultado, nil
}

func identidadIncluidaExactamente(
	identidad ProyeccionIdentidadOperacion,
	candidatas []ProyeccionIdentidadOperacion,
) bool {
	coincidencias := 0
	for _, candidata := range candidatas {
		if identidadesProyectadasCoinciden(identidad, candidata) {
			coincidencias++
		}
	}
	return coincidencias == 1
}

func identidadesProyectadasCoinciden(a, b ProyeccionIdentidadOperacion) bool {
	return proyeccionesHMACCoinciden(a.Localizador, b.Localizador, dominioClaveHMACLocalizador) &&
		proyeccionesHMACCoinciden(a.HuellaSolicitud, b.HuellaSolicitud, dominioClaveHMACHuellaSolicitud)
}

func proyeccionesHMACCoinciden(
	a, b ProyeccionHMACDiario,
	dominio dominioClaveHMAC,
) bool {
	convertir := func(p ProyeccionHMACDiario) (hmacNominalIdempotencia, error) {
		if dominio == dominioClaveHMACLocalizador && !p.valida("localizador") ||
			dominio == dominioClaveHMACHuellaSolicitud && !p.valida("huella_solicitud") {
			return hmacNominalIdempotencia{}, ErrHMACIdempotenciaInvalido
		}
		var referencia ReferenciaClaveHMAC
		var err error
		if dominio == dominioClaveHMACLocalizador {
			referencia, err = NuevaReferenciaClaveHMACLocalizador(p.ClaveRef, p.GeneracionClave)
		} else {
			referencia, err = NuevaReferenciaClaveHMACHuellaSolicitud(p.ClaveRef, p.GeneracionClave)
		}
		if err != nil {
			return hmacNominalIdempotencia{}, err
		}
		return nuevoHMACNominalIdempotencia(p.VersionEsquema, referencia, dominio, p.ValorHMACSHA256)
	}
	primero, errA := convertir(a)
	segundo, errB := convertir(b)
	return errA == nil && errB == nil && primero.coincide(segundo)
}
