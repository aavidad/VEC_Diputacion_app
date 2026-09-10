package domain

import (
	"bytes"
	"errors"
	"time"
)

const (
	TransicionCerrarAdministrativamenteSinCese  ClaveCatalogo = "cerrar_administrativamente_sin_cese"
	EstadoCerradoAdministrativamenteSeguimiento ClaveCatalogo = "cerrado_administrativamente"
)

var ErrContinuacionSeguimientoInvalida = errors.New("contratacion temporal: continuacion de seguimiento invalida")

// ContinuacionSeguimiento liga una unica adopcion y su cierre administrativo al
// estado previo exacto. No sustituye la raiz ni acredita permisos o publicacion.
// Las publicaciones originales se conservan por separado y se exigen al validar.
type ContinuacionSeguimiento struct {
	SeguimientoRef                string                          `json:"seguimiento_ref"`
	OrganizacionRef               string                          `json:"organizacion_ref"`
	ExpedienteRef                 string                          `json:"expediente_ref"`
	RelacionRef                   string                          `json:"relacion_ref"`
	DefinicionOriginal            ReferenciaDefinicionSeguimiento `json:"definicion_original"`
	DefinicionSucesora            ReferenciaDefinicionSeguimiento `json:"definicion_sucesora"`
	HuellaRaizSHA256              string                          `json:"huella_raiz_sha256"`
	VersionAnterior               uint64                          `json:"version_anterior"`
	HuellaEstadoAnteriorSHA256    string                          `json:"huella_estado_anterior_sha256"`
	HuellaActuacionAnteriorSHA256 string                          `json:"huella_actuacion_anterior_sha256"`
	PrimeraSecuencia              uint64                          `json:"primera_secuencia"`
	ActuacionRef                  string                          `json:"actuacion_ref"`
	ReciboRef                     string                          `json:"recibo_ref"`
	CorrelacionRef                string                          `json:"correlacion_ref"`
	RegistradaEn                  time.Time                       `json:"registrada_en"`
	HuellaPeticionSHA256          string                          `json:"huella_peticion_sha256"`
}

// NuevaContinuacionSeguimiento prepara material de integridad; el adaptador debe
// acreditar la publicacion y la autorizacion dentro de la transaccion de cierre.
func NuevaContinuacionSeguimiento(
	original, sucesora DefinicionSeguimiento,
	seguimiento Seguimiento,
	datos DatosTransicionSeguimiento,
) (ContinuacionSeguimiento, error) {
	cero := ContinuacionSeguimiento{}
	validado, err := RehidratarSeguimiento(original, seguimiento.Estado())
	if err != nil {
		return cero, ErrContinuacionSeguimientoInvalida
	}
	normalizados, err := normalizarDatosTransicionSeguimiento(datos)
	if err != nil || validarSucesoraCierreSeguimiento(original, sucesora) != nil {
		return cero, ErrContinuacionSeguimientoInvalida
	}
	estado := validado.estado
	if estado.Version == 0 || len(estado.Actuaciones) >= maximoActuacionesSeguimiento ||
		estado.EstadoActual != "vigente" || len(estado.PeriodosResultantes) == 0 || estado.CeseEfectivo != nil ||
		normalizados.TransicionClave != TransicionCerrarAdministrativamenteSinCese ||
		normalizados.Periodo != nil || normalizados.Calendario != nil || normalizados.RectificaActuacionRef != "" ||
		!normalizados.EfectivoEn.Equal(normalizados.RegistradaEn) ||
		normalizados.RegistradaEn.Before(estado.ActualizadoEn) ||
		normalizados.RegistradaEn.Before(sucesora.publicacion.PublicadoEn) ||
		!sucesora.publicacion.Vigencia.contiene(normalizados.RegistradaEn) {
		return cero, ErrContinuacionSeguimientoInvalida
	}
	transicion, _ := sucesora.transicion(TransicionCerrarAdministrativamenteSinCese)
	if validarRequisitosTransicion(transicion, normalizados, estado.Actuaciones, nil) != nil {
		return cero, ErrContinuacionSeguimientoInvalida
	}
	for _, a := range estado.Actuaciones {
		if a.ActuacionRef == normalizados.ActuacionRef || a.ReciboRef == normalizados.ReciboRef {
			return cero, ErrActuacionSeguimientoEnConflicto
		}
	}
	canon, err := SerializarEstadoSeguimientoCanonico(original, estado)
	if err != nil {
		return cero, ErrContinuacionSeguimientoInvalida
	}
	huellaPeticion, err := calcularHuellaPeticionSeguimiento(normalizados)
	if err != nil {
		return cero, ErrContinuacionSeguimientoInvalida
	}
	return ContinuacionSeguimiento{
		SeguimientoRef: estado.Referencia, OrganizacionRef: estado.OrganizacionRef,
		ExpedienteRef: estado.ExpedienteRef, RelacionRef: estado.RelacionRef,
		DefinicionOriginal: original.Referencia(), DefinicionSucesora: sucesora.Referencia(),
		HuellaRaizSHA256: estado.HuellaRaizSHA256, VersionAnterior: estado.Version,
		HuellaEstadoAnteriorSHA256:    resumenSeguimiento(canon),
		HuellaActuacionAnteriorSHA256: huellaAnteriorSeguimiento(estado), PrimeraSecuencia: estado.Version + 1,
		ActuacionRef: normalizados.ActuacionRef, ReciboRef: normalizados.ReciboRef,
		CorrelacionRef: normalizados.CorrelacionRef, RegistradaEn: normalizados.RegistradaEn,
		HuellaPeticionSHA256: huellaPeticion,
	}, nil
}

func ValidarContinuacionSeguimiento(
	original, sucesora DefinicionSeguimiento,
	seguimiento Seguimiento,
	continuacion ContinuacionSeguimiento,
	datos DatosTransicionSeguimiento,
) error {
	esperada, err := NuevaContinuacionSeguimiento(original, sucesora, seguimiento, datos)
	if err != nil || continuacion != esperada {
		return ErrContinuacionSeguimientoInvalida
	}
	return nil
}

// AplicarCierreConContinuacion anade una actuacion bajo la sucesora sin tocar
// periodos, cese, raiz, definicion fundacional ni actuaciones anteriores. Solo
// acepta un estado V1; un replay confirmado se recupera en el puerto transaccional.
func AplicarCierreConContinuacion(
	seguimiento Seguimiento,
	original, sucesora DefinicionSeguimiento,
	continuacion ContinuacionSeguimiento,
	versionEsperada uint64,
	datos DatosTransicionSeguimiento,
) (Seguimiento, error) {
	validado, err := RehidratarSeguimiento(original, seguimiento.Estado())
	if err != nil {
		return Seguimiento{}, ErrContinuacionSeguimientoInvalida
	}
	if versionEsperada != validado.Version() {
		return Seguimiento{}, ErrVersionEnConflicto
	}
	if err := ValidarContinuacionSeguimiento(original, sucesora, validado, continuacion, datos); err != nil {
		return Seguimiento{}, err
	}
	return aplicarCierreContinuacionValidada(validado, sucesora, continuacion, datos)
}

func aplicarCierreContinuacionValidada(
	seguimiento Seguimiento,
	sucesora DefinicionSeguimiento,
	continuacion ContinuacionSeguimiento,
	datos DatosTransicionSeguimiento,
) (Seguimiento, error) {
	normalizados, err := normalizarDatosTransicionSeguimiento(datos)
	if err != nil {
		return Seguimiento{}, ErrContinuacionSeguimientoInvalida
	}
	transicion, _ := sucesora.transicion(TransicionCerrarAdministrativamenteSinCese)
	siguiente, err := seguimiento.aplicarTransicionValidada(
		sucesora.Referencia(), transicion, normalizados, continuacion.HuellaPeticionSHA256, nil,
	)
	if err != nil {
		return Seguimiento{}, ErrContinuacionSeguimientoInvalida
	}
	siguiente.estado.Continuacion = &continuacion
	return siguiente, nil
}

// RehidratarSeguimientoConContinuacion valida el prefijo con la publicacion
// fundacional y exactamente una actuacion posterior con la sucesora. No admite
// adopciones vacias, otra continuacion, reapertura ni ampliacion de este contrato.
func RehidratarSeguimientoConContinuacion(
	original, sucesora DefinicionSeguimiento,
	estado EstadoPersistidoSeguimiento,
) (Seguimiento, error) {
	if estado.Continuacion == nil || len(estado.Actuaciones) < 2 ||
		len(estado.Actuaciones) > maximoActuacionesSeguimiento ||
		len(estado.PeriodosResultantes) > len(estado.Actuaciones) ||
		estado.Version != uint64(len(estado.Actuaciones)) ||
		estado.Continuacion.VersionAnterior != estado.Version-1 ||
		estado.Continuacion.PrimeraSecuencia != estado.Version {
		return Seguimiento{}, ErrContinuacionSeguimientoInvalida
	}
	// El cierre no altera periodos/cese. Al retirar solo su ultima actuacion se
	// recupera el estado anterior, cuya raiz, canon y cadena se verifican con V1.
	prefijo := estado.clonar()
	continuacion := *prefijo.Continuacion
	prefijo.Continuacion = nil
	ultima := prefijo.Actuaciones[len(prefijo.Actuaciones)-1]
	prefijo.Actuaciones = prefijo.Actuaciones[:len(prefijo.Actuaciones)-1]
	prefijo.Version--
	anterior := prefijo.Actuaciones[len(prefijo.Actuaciones)-1]
	prefijo.EstadoActual = anterior.EstadoDestino
	prefijo.ActualizadoEn = anterior.RegistradaEn
	validado, err := RehidratarSeguimiento(original, prefijo)
	if err != nil || ValidarContinuacionSeguimiento(original, sucesora, validado, continuacion, ultima.datos()) != nil {
		return Seguimiento{}, ErrContinuacionSeguimientoInvalida
	}
	siguiente, err := aplicarCierreContinuacionValidada(validado, sucesora, continuacion, ultima.datos())
	if err != nil {
		return Seguimiento{}, ErrContinuacionSeguimientoInvalida
	}
	generado, errGenerado := materialCanonicoEstadoSeguimientoContinuado(siguiente.estado)
	guardado, errGuardado := materialCanonicoEstadoSeguimientoContinuado(estado)
	if errGenerado != nil || errGuardado != nil || !bytes.Equal(generado, guardado) {
		return Seguimiento{}, ErrContinuacionSeguimientoInvalida
	}
	return siguiente, nil
}

// La sucesora es una extension nominal: conserva exactamente las entradas ya
// publicadas y anade solo el estado final y la transicion administrativa. Una
// nueva version numerica no autoriza reinterpretar transiciones historicas.
func validarSucesoraCierreSeguimiento(original, sucesora DefinicionSeguimiento) error {
	if original.Validar() != nil || sucesora.Validar() != nil {
		return ErrContinuacionSeguimientoInvalida
	}
	a, b := original.publicacion, sucesora.publicacion
	if a.Version == ^uint64(0) || b.Referencia != a.Referencia || b.Version != a.Version+1 ||
		!b.PublicadoEn.After(a.PublicadoEn) || b.Vigencia.Desde.Before(b.PublicadoEn) ||
		b.EstadoInicial != a.EstadoInicial || b.ProhibeCiclosSilenciosos != a.ProhibeCiclosSilenciosos ||
		len(b.Estados) != len(a.Estados)+1 || len(b.Transiciones) != len(a.Transiciones)+1 {
		return ErrContinuacionSeguimientoInvalida
	}
	estados := make(map[ClaveCatalogo]bool, len(b.Estados))
	for _, e := range b.Estados {
		estados[e.Clave] = e.Final
	}
	for _, e := range a.Estados {
		final, existe := estados[e.Clave]
		if !existe || final != e.Final || e.Clave == EstadoCerradoAdministrativamenteSeguimiento {
			return ErrContinuacionSeguimientoInvalida
		}
	}
	finalVigente, vigenteExiste := estados["vigente"]
	finalCierre, cierreExiste := estados[EstadoCerradoAdministrativamenteSeguimiento]
	if !vigenteExiste || finalVigente || !cierreExiste || !finalCierre {
		return ErrContinuacionSeguimientoInvalida
	}
	for _, motivo := range a.Motivos {
		if !contieneClaveSeguimiento(b.Motivos, motivo) {
			return ErrContinuacionSeguimientoInvalida
		}
	}
	for _, anterior := range a.Transiciones {
		nueva, existe := sucesora.transicion(anterior.Clave)
		if !existe || anterior.Clave == TransicionCerrarAdministrativamenteSinCese ||
			!transicionesContinuacionIguales(anterior, nueva) {
			return ErrContinuacionSeguimientoInvalida
		}
	}
	cierre, existe := sucesora.transicion(TransicionCerrarAdministrativamenteSinCese)
	if !existe || cierre.Origen != "vigente" || cierre.Destino != EstadoCerradoAdministrativamenteSeguimiento ||
		cierre.Clase != TransicionOrdinaria || cierre.EfectoPeriodo != EfectoPeriodoNinguno ||
		!cierre.MotivoObligatorio || cierre.RequierePeriodo || cierre.Calendario != nil || cierre.ExigeActorDistinto {
		return ErrContinuacionSeguimientoInvalida
	}
	return nil
}

func transicionesContinuacionIguales(a, b TransicionDefinidaSeguimiento) bool {
	var izquierda, derecha bytes.Buffer
	ei := nuevoEscritorCanonSeguimiento(&izquierda, "comparacion-transicion")
	ed := nuevoEscritorCanonSeguimiento(&derecha, "comparacion-transicion")
	ei.transicionDefinida(a)
	ed.transicionDefinida(b)
	return ei.err == nil && ed.err == nil && bytes.Equal(izquierda.Bytes(), derecha.Bytes())
}
