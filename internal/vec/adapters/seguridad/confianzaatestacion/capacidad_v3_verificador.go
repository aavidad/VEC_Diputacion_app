package confianzaatestacion

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"errors"

	"vec-diputacion-granada/internal/vec/ports"
)

type identificadorClaveCapacidadV3 struct {
	claveID string
	version uint64
}

// VerificadorCapacidadesAtestacionAutorizacionV3 representa al consumidor
// independiente. No contiene raices COSE y no puede emitir porque no expone
// ningun metodo de acuñacion.
type VerificadorCapacidadesAtestacionAutorizacionV3 struct {
	bloqueoSerializacionCapacidadV3
	claves map[identificadorClaveCapacidadV3]ClaveHMACCapacidadAtestacionV3
	reloj  ports.Reloj
}

func NuevoVerificadorCapacidadesAtestacionAutorizacionV3(
	reloj ports.Reloj,
	claves ...ClaveHMACCapacidadAtestacionV3,
) (*VerificadorCapacidadesAtestacionAutorizacionV3, error) {
	if dependenciaConfianzaAtestacionNula(reloj) ||
		len(claves) == 0 || len(claves) > 4 {
		return nil, ErrConfiguracionCapacidadAtestacionV3Invalida
	}
	verificador := &VerificadorCapacidadesAtestacionAutorizacionV3{
		claves: make(
			map[identificadorClaveCapacidadV3]ClaveHMACCapacidadAtestacionV3,
			len(claves),
		),
		reloj: reloj,
	}
	audiencia := ""
	for _, clave := range claves {
		clon, err := clonarClaveHMACCapacidadAtestacionV3(clave)
		if err != nil {
			return nil, ErrConfiguracionCapacidadAtestacionV3Invalida
		}
		if audiencia == "" {
			audiencia = clon.audienciaConsumo
		}
		if clon.audienciaConsumo != audiencia {
			return nil, ErrConfiguracionCapacidadAtestacionV3Invalida
		}
		identificador := identificadorClaveCapacidadV3{
			claveID: clon.claveID,
			version: clon.version,
		}
		if _, existe := verificador.claves[identificador]; existe {
			return nil, ErrConfiguracionCapacidadAtestacionV3Invalida
		}
		verificador.claves[identificador] = clon
	}
	return verificador, nil
}

func (v *VerificadorCapacidadesAtestacionAutorizacionV3) Verificar(
	ctx context.Context,
	capacidad CapacidadBreveAtestacionAutorizacionV3,
) error {
	contenido, err := capacidad.ExportacionCanonicaParaConsumidor()
	if err != nil {
		return ErrCapacidadAtestacionV3Invalida
	}
	defer borrarBytesConfianzaAtestacion(contenido)
	return v.VerificarExportacionCanonica(ctx, contenido)
}

// VerificarExportacionCanonica prueba el salto de proceso: interpreta solo la
// exportacion controlada y coteja MAC con su propia copia gobernada de claves.
func (v *VerificadorCapacidadesAtestacionAutorizacionV3) VerificarExportacionCanonica(
	ctx context.Context,
	contenido []byte,
) error {
	if ctx == nil || v == nil ||
		dependenciaConfianzaAtestacionNula(v.reloj) || len(v.claves) == 0 {
		return ErrCapacidadAtestacionV3Invalida
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(ErrCapacidadAtestacionV3Invalida, err)
	}
	documento, err := interpretarExportacionCapacidadV3(contenido)
	if err != nil {
		return ErrCapacidadAtestacionV3Invalida
	}
	instante := v.reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return errors.Join(ErrCapacidadAtestacionV3Invalida, err)
	}
	emitidaEn, errEmitida := parsearInstanteCapacidadV3(documento.EmitidaEn)
	expiraEn, errExpira := parsearInstanteCapacidadV3(documento.ExpiraEn)
	clave, existe := v.claves[identificadorClaveCapacidadV3{
		claveID: documento.ClaveID,
		version: documento.ClaveVersion,
	}]
	if errEmitida != nil || errExpira != nil || !existe ||
		!instanteCanonicoConfianza(instante) ||
		instante.Before(emitidaEn) || !instante.Before(expiraEn) ||
		!clave.validaParaVerificarEn(instante) ||
		documento.RevisionGobierno != clave.revisionGobierno ||
		documento.HuellaGobiernoSHA256 != clave.huellaGobiernoRef ||
		documento.EmisorID != clave.emisorID ||
		documento.AudienciaConsumo != clave.audienciaConsumo ||
		emitidaEn.Before(clave.validaDesde) ||
		expiraEn.After(clave.validaHasta) {
		return ErrCapacidadAtestacionV3Invalida
	}
	calculada := calcularMACCapacidadAtestacionV3(documento, clave.material)
	esperada, errEsperada := hex.DecodeString(documento.MACSHA256)
	obtenida, errObtenida := hex.DecodeString(calculada)
	if errEsperada != nil || errObtenida != nil ||
		subtle.ConstantTimeCompare(obtenida, esperada) != 1 {
		return ErrCapacidadAtestacionV3Invalida
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(ErrCapacidadAtestacionV3Invalida, err)
	}
	return nil
}
