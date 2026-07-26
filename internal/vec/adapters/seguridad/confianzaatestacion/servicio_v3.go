package confianzaatestacion

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/seguridad/verificacioncose"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const prefijoAADAtestacionAutorizacionV3 = "vec.confianza-atestacion-autorizacion.cose-sign1.v3\x00"

type raizVerificacionAtestacionV3 struct {
	verificador           *verificacioncose.VerificadorClave
	version               uint64
	huellaClaveSPKISHA256 string
	audienciaDespliegue   string
	estado                EstadoClaveAtestacionAutorizacionV3
	validaDesde           time.Time
	validaHasta           time.Time
	revocadaEn            time.Time
}

// ServicioConfianzaAtestacionAutorizacionV3 solo posee claves publicas. No
// contiene la clave HMAC del emisor de capacidades ni credenciales SQL.
type ServicioConfianzaAtestacionAutorizacionV3 struct {
	bloqueoSerializacionV3
	raices                    map[string]raizVerificacionAtestacionV3
	reloj                     ports.Reloj
	revisionConfiguracion     string
	secuenciaConfiguracion    uint64
	huellaConfiguracionSHA256 string
	configuracionPublicadaEn  time.Time
	configuracionExpiraEn     time.Time
}

func NuevoServicioConfianzaAtestacionAutorizacionV3(
	configuracion ConfiguracionConfianzaAtestacionAutorizacionV3,
	reloj ports.Reloj,
) (*ServicioConfianzaAtestacionAutorizacionV3, error) {
	if configuracion.validar() != nil ||
		dependenciaConfianzaAtestacionNula(reloj) {
		return nil, ErrConfiguracionConfianzaAtestacionV3Invalida
	}
	servicio := &ServicioConfianzaAtestacionAutorizacionV3{
		raices: make(map[string]raizVerificacionAtestacionV3, len(configuracion.raices)),
		reloj:  reloj, revisionConfiguracion: configuracion.revision,
		secuenciaConfiguracion:    configuracion.secuencia,
		huellaConfiguracionSHA256: configuracion.huellaSHA256,
		configuracionPublicadaEn:  configuracion.publicadaEn,
		configuracionExpiraEn:     configuracion.expiraEn,
	}
	for _, raiz := range configuracion.raices {
		clon, err := clonarRaizPublicaAtestacionV3(raiz)
		if err != nil {
			return nil, ErrConfiguracionConfianzaAtestacionV3Invalida
		}
		verificador, err := verificacioncose.NuevoVerificadorClave(
			[]byte(clon.claveID),
			verificacioncose.AlgoritmoEdDSA,
			clon.clavePublica,
		)
		if err != nil {
			return nil, ErrConfiguracionConfianzaAtestacionV3Invalida
		}
		if _, existe := servicio.raices[clon.claveID]; existe {
			return nil, ErrConfiguracionConfianzaAtestacionV3Invalida
		}
		servicio.raices[clon.claveID] = raizVerificacionAtestacionV3{
			verificador:           verificador,
			version:               clon.version,
			huellaClaveSPKISHA256: clon.huellaClaveSPKISHA256,
			audienciaDespliegue:   clon.audienciaDespliegue,
			estado:                clon.estado, validaDesde: clon.validaDesde,
			validaHasta: clon.validaHasta, revocadaEn: clon.revocadaEn,
		}
	}
	return servicio, nil
}

// Verificar comprueba VEC-AD-3 exacto y devuelve una prueba nominal. No acuña
// capacidades ni acepta atestaciones V1/V2.
func (s *ServicioConfianzaAtestacionAutorizacionV3) Verificar(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
	atestacion ports.AtestacionAutorizacionV3,
) (PruebaConfianzaAtestacionAutorizacionV3, error) {
	vacia := PruebaConfianzaAtestacionAutorizacionV3{}
	if ctx == nil || s == nil ||
		dependenciaConfianzaAtestacionNula(s.reloj) || len(s.raices) == 0 {
		return vacia, ErrVerificacionConfianzaAtestacionV3Fallida
	}
	if err := ctx.Err(); err != nil {
		return vacia, errors.Join(ErrVerificacionConfianzaAtestacionV3Fallida, err)
	}
	instante := s.reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return vacia, errors.Join(ErrVerificacionConfianzaAtestacionV3Fallida, err)
	}
	datosSolicitud, errSolicitud := solicitud.Datos()
	emitidaEn, validaHasta, errVentana := decision.VentanaValidez()
	concedida, codigo, errResultado := decision.Resultado()
	if !instanteCanonicoConfianza(instante) ||
		instante.Before(s.configuracionPublicadaEn) ||
		!instante.Before(s.configuracionExpiraEn) ||
		errSolicitud != nil || errVentana != nil || errResultado != nil ||
		!concedida || codigo != "concedida" ||
		decision.ValidarPara(solicitud) != nil ||
		resultadoContexto.Validar() != nil ||
		datosSolicitud.ReferenciaMotivo != referenciaMotivo ||
		datosSolicitud.VinculoAutenticacionActor.ValidarPara(resultadoContexto) != nil ||
		!datosSolicitud.VinculoAutenticacionActor.VigenteEn(
			instante,
			resultadoContexto,
		) ||
		instante.Before(emitidaEn) || !instante.Before(validaHasta) {
		return vacia, ErrVerificacionConfianzaAtestacionV3Fallida
	}
	solicitudAtestada, err := atestacion.Solicitud()
	if err != nil {
		return vacia, ErrVerificacionConfianzaAtestacionV3Fallida
	}
	cabecera, err := solicitudAtestada.Cabecera()
	if err != nil ||
		cabecera.FormatoVersion != domain.VersionFormatoAtestacionAutorizacionV3 ||
		cabecera.Suite != SuiteAtestacionAutorizacionV3COSEEdDSA {
		return vacia, ErrVerificacionConfianzaAtestacionV3Fallida
	}
	esperada, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV3(
		cabecera, decision, referenciaMotivo, resultadoContexto,
	)
	if err != nil || atestacion.ValidarPara(esperada) != nil {
		return vacia, ErrVerificacionConfianzaAtestacionV3Fallida
	}
	raiz, existe := s.raices[cabecera.ClaveID]
	if !existe || raiz.verificador == nil ||
		raiz.estado != EstadoClaveAtestacionAutorizacionV3Activa ||
		!raiz.revocadaEn.IsZero() ||
		cabecera.Audiencia != raiz.audienciaDespliegue ||
		instante.Before(raiz.validaDesde) ||
		!instante.Before(raiz.validaHasta) {
		return vacia, ErrVerificacionConfianzaAtestacionV3Fallida
	}
	mensaje, err := esperada.Mensaje()
	if err != nil {
		return vacia, ErrVerificacionConfianzaAtestacionV3Fallida
	}
	defer borrarBytesConfianzaAtestacion(mensaje)
	resultadoFirma, err := atestacion.Resultado()
	if err != nil {
		return vacia, ErrVerificacionConfianzaAtestacionV3Fallida
	}
	sobre, err := resultadoFirma.Firma()
	if err != nil {
		return vacia, ErrVerificacionConfianzaAtestacionV3Fallida
	}
	defer borrarBytesConfianzaAtestacion(sobre)
	inspeccion, err := verificacioncose.InspeccionarSobreSign1(sobre, len(sobre))
	if err != nil {
		return vacia, ErrVerificacionConfianzaAtestacionV3Fallida
	}
	algoritmo, errAlgoritmo := inspeccion.Algoritmo()
	claveID, errClave := inspeccion.ClaveID()
	aad, errAAD := AADExternoAtestacionAutorizacionV3(cabecera.Audiencia)
	if errAlgoritmo != nil || errClave != nil || errAAD != nil ||
		algoritmo != verificacioncose.AlgoritmoEdDSA ||
		string(claveID) != cabecera.ClaveID ||
		raiz.verificador.VerificarPayloadSeparado(inspeccion, mensaje, aad) != nil {
		return vacia, ErrVerificacionConfianzaAtestacionV3Fallida
	}
	if err := ctx.Err(); err != nil {
		return vacia, errors.Join(ErrVerificacionConfianzaAtestacionV3Fallida, err)
	}
	huellaDecision, errDecision := domain.HuellaSHA256DecisionAutorizacionV3(decision)
	huellaMotivo, errMotivo := domain.HuellaSHA256MotivoAutorizacionV2(referenciaMotivo)
	if errDecision != nil || errMotivo != nil {
		return vacia, ErrVerificacionConfianzaAtestacionV3Fallida
	}
	prueba, err := nuevaPruebaConfianzaAtestacionAutorizacionV3(
		datosPruebaConfianzaAtestacionAutorizacionV3{
			ReferenciaDecision:    solicitudAtestadaReferenciaDecision(solicitudAtestada),
			HuellaDecisionSHA256:  huellaDecision,
			HuellaMotivoSHA256:    huellaMotivo,
			ReferenciaContexto:    resultadoContexto.RegistroContextoRef,
			HuellaContextoSHA256:  resultadoContexto.HuellaSHA256,
			HuellaMensajeSHA256:   huellaBytesConfianzaAtestacion(mensaje),
			HuellaSobreSHA256:     huellaBytesConfianzaAtestacion(sobre),
			ClaveID:               cabecera.ClaveID,
			RaizVersion:           raiz.version,
			HuellaClaveSPKISHA256: raiz.huellaClaveSPKISHA256,
			AlgoritmoCOSE:         AlgoritmoCOSEAtestacionAutorizacionV3EdDSA,
			Suite:                 cabecera.Suite, AudienciaDespliegue: cabecera.Audiencia,
			EstadoClave: raiz.estado, VerificadaEn: instante,
			RaizValidaDesde: raiz.validaDesde, RaizValidaHasta: raiz.validaHasta,
			RevisionConfiguracion:     s.revisionConfiguracion,
			SecuenciaConfiguracion:    s.secuenciaConfiguracion,
			HuellaConfiguracionSHA256: s.huellaConfiguracionSHA256,
			ConfiguracionPublicadaEn:  s.configuracionPublicadaEn,
			ConfiguracionExpiraEn:     s.configuracionExpiraEn,
		},
	)
	if err != nil ||
		prueba.ValidarPara(
			solicitud,
			decision,
			referenciaMotivo,
			resultadoContexto,
			atestacion,
		) != nil {
		return vacia, ErrVerificacionConfianzaAtestacionV3Fallida
	}
	return prueba, nil
}

func solicitudAtestadaReferenciaDecision(
	solicitud ports.SolicitudFirmaAtestacionAutorizacionV3,
) string {
	referencia, err := solicitud.ReferenciaDecision()
	if err != nil {
		return ""
	}
	return referencia
}

func AADExternoAtestacionAutorizacionV3(
	audienciaDespliegue string,
) ([]byte, error) {
	cabecera := domain.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite:          SuiteAtestacionAutorizacionV3COSEEdDSA,
		ClaveID:        "clave-validacion-audiencia-v3",
		Audiencia:      audienciaDespliegue,
	}
	if cabecera.Validar() != nil {
		return nil, ErrConfiguracionConfianzaAtestacionV3Invalida
	}
	return []byte(prefijoAADAtestacionAutorizacionV3 + audienciaDespliegue), nil
}
