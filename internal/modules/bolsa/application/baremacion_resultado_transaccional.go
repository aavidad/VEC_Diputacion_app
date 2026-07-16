package application

import (
	"errors"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// clasificarDesenlaceConfirmacionBaremacion se ejecuta exclusivamente despues
// de cruzar la frontera que puede haber enviado COMMIT. Solo una prueba
// autenticada y coherente de no aplicacion evita el estado indeterminado; esa
// prueba no concede por si misma autoridad para abandonar ni repetir.
//
// BLOQUEANTE PRODUCTIVO: la solicitud V1 aun no porta el identificador opaco
// previo al COMMIT, por lo que esta capa no puede cotejar que la prueba tipada
// corresponde a esta invocacion exacta. El contrato V2 debe hacer ese vinculo
// obligatorio; no se fabrica aqui una identidad que el repositorio no aporto.
func clasificarDesenlaceConfirmacionBaremacion(
	resultado puertosbolsa.ResultadoConfirmarCambioBaremacion,
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
	errInvocacion error,
) error {
	solicitudValida := solicitud.Validar() == nil
	errValidacion := resultado.ValidarPara(solicitud)
	if errInvocacion == nil && errValidacion == nil {
		return nil
	}

	tipado, pruebaUnica, contieneTipado := extraerResultadoTransaccionalUnicoBaremacion(errInvocacion)
	noAplicadaAcreditada := solicitudValida && errValidacion != nil && pruebaUnica &&
		tipado.NoAplicadaVerificada()
	if noAplicadaAcreditada {
		// Solo se propaga el contrato tipado y expurgado. Los envoltorios o
		// hermanos tecnicos deben auditarse dentro del adaptador.
		return errors.Join(ErrResultadoBaremacionNoConfiable, tipado)
	}

	var causa error
	switch {
	case errInvocacion == nil:
		causa = errValidacion
	case pruebaUnica && tipado.NoAplicadaVerificada():
		// Si existe tambien un resultado ordinario valido o la solicitud no es
		// valida, las dos afirmaciones se contradicen y ninguna se propaga.
		causa = puertosbolsa.ErrResultadoTransaccionalBaremacionInvalido
	case pruebaUnica:
		// El tipo contractual tiene representaciones seguras y conserva el
		// identificador opaco sin arrastrar hermanos tecnicos del adaptador.
		causa = tipado
	case contieneTipado:
		// Typed nil, prueba invalida, varios identificadores o ramas hermanas:
		// nunca se elige arbitrariamente uno de ellos.
		causa = puertosbolsa.ErrResultadoTransaccionalBaremacionInvalido
	default:
		// Las causas genericas se auditan y expurgan en el adaptador. Cruzar esta
		// frontera solo conserva clasificaciones estables, nunca texto tecnico.
		causa = nil
	}
	return errors.Join(
		ErrResultadoBaremacionNoConfiable,
		puertosbolsa.ErrResultadoTransaccionalBaremacionIndeterminado,
		puertosbolsa.ErrReconciliacionTransaccionalBaremacionRequerida,
		causa,
	)
}

// extraerResultadoTransaccionalUnicoBaremacion no confia en implementaciones
// personalizadas de errors.As: exige un unico nodo concreto dentro de un grafo
// acotado. Dos pruebas, aunque sean validas por separado, son ambiguas.
func extraerResultadoTransaccionalUnicoBaremacion(
	err error,
) (
	resultado *puertosbolsa.ErrorResultadoTransaccionalBaremacion,
	unico bool,
	contieneTipado bool,
) {
	const (
		profundidadMaxima = 32
		nodosMaximos      = 128
		hijosMaximos      = 32
	)
	estructuraValida := true
	abortado := false
	nodos := 0
	defer func() {
		if recover() != nil {
			resultado, unico, contieneTipado = nil, false, false
		}
	}()
	var recorrer func(error, int)
	recorrer = func(actual error, profundidad int) {
		if abortado || actual == nil {
			return
		}
		nodos++
		if profundidad > profundidadMaxima || nodos > nodosMaximos {
			estructuraValida = false
			abortado = true
			return
		}
		if tipado, esTipado := actual.(*puertosbolsa.ErrorResultadoTransaccionalBaremacion); esTipado {
			contieneTipado = true
			if tipado == nil || tipado.Validar() != nil {
				estructuraValida = false
				return
			}
			if resultado != nil {
				estructuraValida = false
			} else {
				resultado = tipado
			}
		}
		if multiple, ok := actual.(interface{ Unwrap() []error }); ok {
			hijos := multiple.Unwrap()
			if len(hijos) > hijosMaximos {
				estructuraValida = false
				abortado = true
				return
			}
			if len(hijos) != 1 {
				estructuraValida = false
			}
			for _, hijo := range hijos {
				recorrer(hijo, profundidad+1)
			}
			return
		}
		if simple, ok := actual.(interface{ Unwrap() error }); ok {
			recorrer(simple.Unwrap(), profundidad+1)
		}
	}
	recorrer(err, 0)
	return resultado, !abortado && estructuraValida && resultado != nil, contieneTipado
}
