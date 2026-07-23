package confianzaatestacion

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// EmisorCapacidadesAtestacionAutorizacionV3 solo posee la clave HMAC de
// emision. No contiene raices COSE ni una credencial de consumidor SQL.
type EmisorCapacidadesAtestacionAutorizacionV3 struct {
	bloqueoSerializacionCapacidadV3
	clave    ClaveHMACCapacidadAtestacionV3
	reloj    ports.Reloj
	entropia io.Reader
}

func NuevoEmisorCapacidadesAtestacionAutorizacionV3(
	clave ClaveHMACCapacidadAtestacionV3,
	reloj ports.Reloj,
) (*EmisorCapacidadesAtestacionAutorizacionV3, error) {
	return nuevoEmisorCapacidadesAtestacionAutorizacionV3(
		clave,
		reloj,
		rand.Reader,
	)
}

func nuevoEmisorCapacidadesAtestacionAutorizacionV3(
	clave ClaveHMACCapacidadAtestacionV3,
	reloj ports.Reloj,
	entropia io.Reader,
) (*EmisorCapacidadesAtestacionAutorizacionV3, error) {
	clon, err := clonarClaveHMACCapacidadAtestacionV3(clave)
	if err != nil || clon.estado != EstadoClaveHMACCapacidadAtestacionV3Emision ||
		dependenciaConfianzaAtestacionNula(reloj) || entropia == nil {
		return nil, ErrConfiguracionCapacidadAtestacionV3Invalida
	}
	return &EmisorCapacidadesAtestacionAutorizacionV3{
		clave: clon, reloj: reloj, entropia: entropia,
	}, nil
}

// Emitir vuelve a cotejar la prueba VEC-AD-3 y deriva operacion y efecto de la
// solicitud firmada. Ninguno de esos valores se acepta como texto libre.
func (e *EmisorCapacidadesAtestacionAutorizacionV3) Emitir(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
	atestacion ports.AtestacionAutorizacionV3,
	prueba PruebaConfianzaAtestacionAutorizacionV3,
) (CapacidadBreveAtestacionAutorizacionV3, error) {
	vacia := CapacidadBreveAtestacionAutorizacionV3{}
	if ctx == nil || e == nil || dependenciaConfianzaAtestacionNula(e.reloj) ||
		e.entropia == nil {
		return vacia, ErrCapacidadAtestacionV3NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, errors.Join(ErrCapacidadAtestacionV3NoDisponible, err)
	}
	if prueba.ValidarPara(
		solicitud,
		decision,
		referenciaMotivo,
		resultadoContexto,
		atestacion,
	) != nil {
		return vacia, ErrCapacidadAtestacionV3NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, errors.Join(ErrCapacidadAtestacionV3NoDisponible, err)
	}
	emitidaEn := e.reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return vacia, errors.Join(ErrCapacidadAtestacionV3NoDisponible, err)
	}
	_, decisionValidaHasta, errVentana := decision.VentanaValidez()
	datosSolicitud, errSolicitud := solicitud.Datos()
	datosPrueba, errPrueba := prueba.Datos()
	if errVentana != nil || errSolicitud != nil || errPrueba != nil ||
		!instanteCanonicoConfianza(emitidaEn) ||
		!e.clave.validaParaEmitirEn(emitidaEn) ||
		emitidaEn.Before(datosPrueba.VerificadaEn) ||
		!emitidaEn.Before(decisionValidaHasta) ||
		!emitidaEn.Before(datosPrueba.ConfiguracionExpiraEn) ||
		!emitidaEn.Before(datosPrueba.RaizValidaHasta) {
		return vacia, ErrCapacidadAtestacionV3NoDisponible
	}
	// El perfil O2-04 cierra el conjunto de atributos del recurso. Por eso la
	// capacidad no inventa dos atributos ni acepta un efecto por canal lateral:
	// deriva la referencia y su huella de contexto del recurso completo que ya
	// forma parte del mensaje VEC-AD-3 firmado. El consumidor debe cotejar
	// exactamente ambos valores con la reserva que va a materializar.
	efectoRef := datosSolicitud.Recurso.Referencia
	huellaEfecto, errHuellaEfecto := datosSolicitud.Recurso.
		HuellaContextoAutorizacionSHA256()
	if !referenciaPruebaConfianzaValida(datosSolicitud.Accion) ||
		!referenciaPruebaConfianzaValida(efectoRef) ||
		errHuellaEfecto != nil ||
		!huellaSHA256ConfianzaValida(huellaEfecto) {
		return vacia, ErrCapacidadAtestacionV3NoDisponible
	}
	expiraEn := emitidaEn.Add(
		VigenciaMaximaCapacidadAtestacionAutorizacionV3,
	)
	for _, limite := range []time.Time{
		decisionValidaHasta,
		datosPrueba.ConfiguracionExpiraEn,
		datosPrueba.RaizValidaHasta,
		e.clave.validaHasta,
	} {
		if limite.Before(expiraEn) {
			expiraEn = limite
		}
	}
	if !expiraEn.After(emitidaEn) {
		return vacia, ErrCapacidadAtestacionV3NoDisponible
	}
	var nonce [sha256.Size]byte
	if _, err := io.ReadFull(e.entropia, nonce[:]); err != nil ||
		bytes.Equal(nonce[:], make([]byte, len(nonce))) {
		return vacia, ErrCapacidadAtestacionV3NoDisponible
	}
	if err := ctx.Err(); err != nil {
		clear(nonce[:])
		return vacia, errors.Join(ErrCapacidadAtestacionV3NoDisponible, err)
	}
	documento := capacidadAtestacionAutorizacionV3JSON{
		Esquema: esquemaCapacidadAtestacionAutorizacionV3,
		Version: versionCapacidadAtestacionAutorizacionV3,
		ClaveID: e.clave.claveID, ClaveVersion: e.clave.version,
		RevisionGobierno:     e.clave.revisionGobierno,
		HuellaGobiernoSHA256: e.clave.huellaGobiernoRef,
		EmisorID:             e.clave.emisorID, AudienciaConsumo: e.clave.audienciaConsumo,
		Nonce:                     hex.EncodeToString(nonce[:]),
		EmitidaEn:                 emitidaEn.Format(time.RFC3339Nano),
		ExpiraEn:                  expiraEn.Format(time.RFC3339Nano),
		DecisionRef:               datosPrueba.ReferenciaDecision,
		HuellaDecisionSHA256:      datosPrueba.HuellaDecisionSHA256,
		HuellaMotivoSHA256:        datosPrueba.HuellaMotivoSHA256,
		HuellaPayloadVECAD3SHA256: datosPrueba.HuellaMensajeSHA256,
		HuellaSobreCOSESHA256:     datosPrueba.HuellaSobreSHA256,
		HuellaPruebaSHA256:        datosPrueba.HuellaPruebaSHA256,
		ContextoRef:               datosPrueba.ReferenciaContexto,
		HuellaContextoSHA256:      datosPrueba.HuellaContextoSHA256,
		AudienciaDespliegue:       datosPrueba.AudienciaDespliegue,
		Operacion:                 datosSolicitud.Accion,
		EfectoRef:                 efectoRef,
		HuellaEfectoSHA256:        huellaEfecto,
		DecisionValidaHasta:       decisionValidaHasta.Format(time.RFC3339Nano),
		VerificadaEn:              datosPrueba.VerificadaEn.Format(time.RFC3339Nano),
		RevisionConfianza:         datosPrueba.RevisionConfiguracion,
		ConfiguracionSecuencia:    datosPrueba.SecuenciaConfiguracion,
		HuellaConfiguracionSHA256: datosPrueba.HuellaConfiguracionSHA256,
		ConfiguracionPublicadaEn:  datosPrueba.ConfiguracionPublicadaEn.Format(time.RFC3339Nano),
		ConfiguracionExpiraEn:     datosPrueba.ConfiguracionExpiraEn.Format(time.RFC3339Nano),
		RaizClaveID:               datosPrueba.ClaveID,
		RaizVersion:               datosPrueba.RaizVersion,
		HuellaRaizSPKISHA256:      datosPrueba.HuellaClaveSPKISHA256,
		RaizValidaDesde:           datosPrueba.RaizValidaDesde.Format(time.RFC3339Nano),
		RaizValidaHasta:           datosPrueba.RaizValidaHasta.Format(time.RFC3339Nano),
		Suite:                     datosPrueba.Suite,
	}
	clear(nonce[:])
	documento.MACSHA256 = calcularMACCapacidadAtestacionV3(
		documento,
		e.clave.material,
	)
	if err := ctx.Err(); err != nil {
		return vacia, errors.Join(ErrCapacidadAtestacionV3NoDisponible, err)
	}
	capacidad, err := nuevaCapacidadBreveAtestacionAutorizacionV3(documento)
	if err != nil {
		return vacia, ErrCapacidadAtestacionV3NoDisponible
	}
	return capacidad, nil
}

func preimagenMACCapacidadAtestacionV3(valores []string) []byte {
	var salida bytes.Buffer
	for _, valor := range valores {
		salida.WriteString(strconv.Itoa(len([]byte(valor))))
		salida.WriteByte(':')
		salida.WriteString(valor)
		salida.WriteByte('\n')
	}
	return salida.Bytes()
}

func calcularMACCapacidadAtestacionV3(
	documento capacidadAtestacionAutorizacionV3JSON,
	material []byte,
) string {
	mac := hmac.New(sha256.New, material)
	preimagen := preimagenMACCapacidadAtestacionV3(
		documento.valoresAutenticados(),
	)
	_, _ = mac.Write(preimagen)
	borrarBytesConfianzaAtestacion(preimagen)
	return hex.EncodeToString(mac.Sum(nil))
}

var _ ports.ExportadorCapacidadAtestacionAutorizacionV3 = CapacidadBreveAtestacionAutorizacionV3{}
