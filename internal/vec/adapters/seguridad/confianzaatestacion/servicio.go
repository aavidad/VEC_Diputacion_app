package confianzaatestacion

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/seguridad/verificacioncose"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const prefijoAADAtestacionAutorizacionV2 = "vec.confianza-atestacion-autorizacion.cose-sign1.v2\x00"

type raizVerificacionAtestacionV2 struct {
	verificador           *verificacioncose.VerificadorClave
	huellaClaveSPKISHA256 string
	audienciaDespliegue   string
	estado                EstadoClaveAtestacionAutorizacionV2
	validaDesde           time.Time
	validaHasta           time.Time
	revocadaEn            time.Time
}

// ServicioConfianzaAtestacionAutorizacionV2 contiene una copia defensiva de
// la lista positiva. Produce prueba criptografica y de configuracion, nunca
// una capacidad para mutar Bolsa.
type ServicioConfianzaAtestacionAutorizacionV2 struct {
	bloqueoSerializacion
	raices                    map[string]raizVerificacionAtestacionV2
	reloj                     ports.Reloj
	revisionConfiguracion     string
	huellaConfiguracionSHA256 string
	configuracionPublicadaEn  time.Time
	configuracionExpiraEn     time.Time
}

func NuevoServicioConfianzaAtestacionAutorizacionV2(
	configuracion ConfiguracionConfianzaAtestacionAutorizacionV2,
	reloj ports.Reloj,
) (*ServicioConfianzaAtestacionAutorizacionV2, error) {
	if configuracion.validar() != nil || dependenciaConfianzaAtestacionNula(reloj) {
		return nil, ErrConfiguracionConfianzaAtestacionV2Invalida
	}
	servicio := &ServicioConfianzaAtestacionAutorizacionV2{
		raices:                    make(map[string]raizVerificacionAtestacionV2, len(configuracion.raices)),
		reloj:                     reloj,
		revisionConfiguracion:     configuracion.revision,
		huellaConfiguracionSHA256: configuracion.huellaSHA256,
		configuracionPublicadaEn:  configuracion.publicadaEn,
		configuracionExpiraEn:     configuracion.expiraEn,
	}
	for _, raiz := range configuracion.raices {
		clon, err := clonarRaizPublicaAtestacionV2(raiz)
		if err != nil {
			return nil, ErrConfiguracionConfianzaAtestacionV2Invalida
		}
		verificador, err := verificacioncose.NuevoVerificadorClave(
			[]byte(clon.claveID),
			verificacioncose.AlgoritmoEdDSA,
			clon.clavePublica,
		)
		if err != nil {
			return nil, ErrConfiguracionConfianzaAtestacionV2Invalida
		}
		if _, existe := servicio.raices[clon.claveID]; existe {
			return nil, ErrConfiguracionConfianzaAtestacionV2Invalida
		}
		servicio.raices[clon.claveID] = raizVerificacionAtestacionV2{
			verificador:           verificador,
			huellaClaveSPKISHA256: clon.huellaClaveSPKISHA256,
			audienciaDespliegue:   clon.audienciaDespliegue,
			estado:                clon.estado,
			validaDesde:           clon.validaDesde,
			validaHasta:           clon.validaHasta,
			revocadaEn:            clon.revocadaEn,
		}
	}
	return servicio, nil
}

// Verificar comprueba la decision esperada, el motivo, la configuracion
// vigente y un COSE_Sign1 con payload separado. La fecha informativa devuelta
// por el firmante no se usa como reloj de seguridad.
func (s *ServicioConfianzaAtestacionAutorizacionV2) Verificar(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
	atestacion ports.AtestacionAutorizacionV2,
) (PruebaConfianzaAtestacionAutorizacionV2, error) {
	if ctx == nil || s == nil || dependenciaConfianzaAtestacionNula(s.reloj) || len(s.raices) == 0 {
		return PruebaConfianzaAtestacionAutorizacionV2{}, ErrVerificacionConfianzaAtestacionV2Fallida
	}
	if err := ctx.Err(); err != nil {
		return PruebaConfianzaAtestacionAutorizacionV2{}, errors.Join(
			ErrVerificacionConfianzaAtestacionV2Fallida,
			err,
		)
	}
	instante := s.reloj.Ahora()
	if !instanteCanonicoConfianza(instante) ||
		instante.Before(s.configuracionPublicadaEn) ||
		!instante.Before(s.configuracionExpiraEn) ||
		decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2() != nil ||
		instante.Before(decision.EmitidaEn) || !instante.Before(decision.ValidaHasta) {
		return PruebaConfianzaAtestacionAutorizacionV2{}, ErrVerificacionConfianzaAtestacionV2Fallida
	}

	solicitudAtestada, err := atestacion.Solicitud()
	if err != nil {
		return PruebaConfianzaAtestacionAutorizacionV2{}, ErrVerificacionConfianzaAtestacionV2Fallida
	}
	cabecera, err := solicitudAtestada.Cabecera()
	if err != nil || cabecera.FormatoVersion != domain.VersionFormatoAtestacionAutorizacionV2 ||
		cabecera.Suite != SuiteAtestacionAutorizacionV2COSEEdDSA {
		return PruebaConfianzaAtestacionAutorizacionV2{}, ErrVerificacionConfianzaAtestacionV2Fallida
	}
	solicitudEsperada, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV2(
		cabecera,
		decision,
		referenciaMotivo,
	)
	if err != nil || atestacion.ValidarPara(solicitudEsperada) != nil {
		return PruebaConfianzaAtestacionAutorizacionV2{}, ErrVerificacionConfianzaAtestacionV2Fallida
	}
	raiz, existe := s.raices[cabecera.ClaveID]
	if !existe || raiz.verificador == nil ||
		raiz.estado != EstadoClaveAtestacionAutorizacionV2Activa ||
		!raiz.revocadaEn.IsZero() || cabecera.Audiencia != raiz.audienciaDespliegue ||
		instante.Before(raiz.validaDesde) || !instante.Before(raiz.validaHasta) {
		return PruebaConfianzaAtestacionAutorizacionV2{}, ErrVerificacionConfianzaAtestacionV2Fallida
	}

	mensaje, err := solicitudEsperada.Mensaje()
	if err != nil {
		return PruebaConfianzaAtestacionAutorizacionV2{}, ErrVerificacionConfianzaAtestacionV2Fallida
	}
	defer borrarBytesConfianzaAtestacion(mensaje)
	resultado, err := atestacion.Resultado()
	if err != nil {
		return PruebaConfianzaAtestacionAutorizacionV2{}, ErrVerificacionConfianzaAtestacionV2Fallida
	}
	sobre, err := resultado.Firma()
	if err != nil {
		return PruebaConfianzaAtestacionAutorizacionV2{}, ErrVerificacionConfianzaAtestacionV2Fallida
	}
	defer borrarBytesConfianzaAtestacion(sobre)
	inspeccion, err := verificacioncose.InspeccionarSobreSign1(sobre, len(sobre))
	if err != nil {
		return PruebaConfianzaAtestacionAutorizacionV2{}, ErrVerificacionConfianzaAtestacionV2Fallida
	}
	algoritmo, errAlgoritmo := inspeccion.Algoritmo()
	claveID, errClave := inspeccion.ClaveID()
	aad, errAAD := AADExternoAtestacionAutorizacionV2(cabecera.Audiencia)
	if errAlgoritmo != nil || errClave != nil || errAAD != nil ||
		algoritmo != verificacioncose.AlgoritmoEdDSA || string(claveID) != cabecera.ClaveID ||
		raiz.verificador.VerificarPayloadSeparado(inspeccion, mensaje, aad) != nil {
		return PruebaConfianzaAtestacionAutorizacionV2{}, ErrVerificacionConfianzaAtestacionV2Fallida
	}
	if err := ctx.Err(); err != nil {
		return PruebaConfianzaAtestacionAutorizacionV2{}, errors.Join(
			ErrVerificacionConfianzaAtestacionV2Fallida,
			err,
		)
	}
	prueba, err := nuevaPruebaConfianzaAtestacionAutorizacionV2(
		datosPruebaConfianzaAtestacionAutorizacionV2{
			ReferenciaDecision:          decision.DecisionRef,
			HuellaSolicitudLigadaSHA256: decision.SolicitudHuellaSHA256,
			HuellaMotivoCatalogoSHA256:  decision.MotivoHuellaSHA256,
			HuellaMensajeSHA256:         huellaBytesConfianzaAtestacion(mensaje),
			HuellaSobreSHA256:           huellaBytesConfianzaAtestacion(sobre),
			ClaveID:                     cabecera.ClaveID,
			HuellaClaveSPKISHA256:       raiz.huellaClaveSPKISHA256,
			AlgoritmoCOSE:               AlgoritmoCOSEAtestacionAutorizacionV2EdDSA,
			Suite:                       cabecera.Suite,
			AudienciaDespliegue:         cabecera.Audiencia,
			EstadoClave:                 raiz.estado,
			VerificadaEn:                instante,
			RaizValidaDesde:             raiz.validaDesde,
			RaizValidaHasta:             raiz.validaHasta,
			RevisionConfiguracion:       s.revisionConfiguracion,
			HuellaConfiguracionSHA256:   s.huellaConfiguracionSHA256,
			ConfiguracionPublicadaEn:    s.configuracionPublicadaEn,
			ConfiguracionExpiraEn:       s.configuracionExpiraEn,
		},
	)
	if err != nil || prueba.ValidarPara(decision, referenciaMotivo, atestacion) != nil {
		return PruebaConfianzaAtestacionAutorizacionV2{}, ErrVerificacionConfianzaAtestacionV2Fallida
	}
	return prueba, nil
}

// AADExternoAtestacionAutorizacionV2 publica la vinculacion exacta que debe
// emplear el conector HSM/KMS al producir el COSE_Sign1 con payload separado.
func AADExternoAtestacionAutorizacionV2(audienciaDespliegue string) ([]byte, error) {
	if !audienciaDespliegueAtestacionV2Valida(audienciaDespliegue) {
		return nil, ErrConfiguracionConfianzaAtestacionV2Invalida
	}
	return []byte(prefijoAADAtestacionAutorizacionV2 + audienciaDespliegue), nil
}

func dependenciaConfianzaAtestacionNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func borrarBytesConfianzaAtestacion(datos []byte) {
	for indice := range datos {
		datos[indice] = 0
	}
}
