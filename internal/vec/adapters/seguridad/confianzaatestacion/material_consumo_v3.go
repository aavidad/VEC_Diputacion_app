package confianzaatestacion

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"hash"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrMaterialConsumoAutorizacionAtestadaV3Invalido = errors.New(
	"vec: material de consumo de autorizacion atestada V3 invalido",
)

// MaterialConsumoAutorizacionAtestadaV3 conserva un único conjunto coherente
// de pruebas VEC-AD-3. No verifica gobierno vivo ni concede autoridad; esa
// responsabilidad permanece en el consumidor PostgreSQL transaccional.
type MaterialConsumoAutorizacionAtestadaV3 struct {
	bloqueoSerializacionV3
	entradas entradasMaterialConsumoAutorizacionAtestadaV3
	huella   [sha256.Size]byte
}

type entradasMaterialConsumoAutorizacionAtestadaV3 struct {
	capacidadCanonica     []byte
	resumenCapacidad      ports.ResumenCapacidadAtestacionAutorizacionV3
	decisionCanonica      []byte
	motivoCanonico        []byte
	contextoActorCanonico []byte
	personaVersion        uint64
	perfilVersion         uint64
	payloadVECAD3         []byte
	sobreCOSESign1        []byte
	evidenciaVerificacion []byte
	raizPublicaSPKI       []byte
}

// NuevoMaterialConsumoAutorizacionAtestadaV3 solo compone nominales ya
// validados. No admite un constructor alternativo desde las diez entradas
// crudas que un módulo funcional pudiera fabricar.
func NuevoMaterialConsumoAutorizacionAtestadaV3(
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	motivo domain.ReferenciaEntradaCatalogo,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
	atestacion ports.AtestacionAutorizacionV3,
	prueba PruebaConfianzaAtestacionAutorizacionV3,
	capacidad CapacidadBreveAtestacionAutorizacionV3,
	raiz RaizPublicaAtestacionAutorizacionV3,
) (MaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := MaterialConsumoAutorizacionAtestadaV3{}
	if decision.ValidarPara(solicitud) != nil ||
		resultadoContexto.Validar() != nil ||
		prueba.ValidarPara(
			solicitud, decision, motivo, resultadoContexto, atestacion,
		) != nil ||
		capacidad.validar() != nil || raiz.validar() != nil {
		return vacio, ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	datosSolicitud, errSolicitud := solicitud.Datos()
	datosPrueba, errPrueba := prueba.Datos()
	if errSolicitud != nil || errPrueba != nil ||
		datosSolicitud.ReferenciaMotivo != motivo {
		return vacio, ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	solicitudFirma, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV3(
		domain.CabeceraAtestacionAutorizacionV3{
			FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
			Suite:          datosPrueba.Suite,
			ClaveID:        datosPrueba.ClaveID,
			Audiencia:      datosPrueba.AudienciaDespliegue,
		},
		decision,
		motivo,
		resultadoContexto,
	)
	if err != nil || atestacion.ValidarPara(solicitudFirma) != nil {
		return vacio, ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	entradas, err := extraerEntradasMaterialConsumoV3(
		decision,
		motivo,
		resultadoContexto,
		solicitudFirma,
		atestacion,
		prueba,
		capacidad,
		raiz,
	)
	if err != nil || cruzarMaterialConsumoV3(
		entradas,
		datosSolicitud,
		decision,
		datosPrueba,
		capacidad,
		raiz,
	) != nil {
		return vacio, ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	huella := calcularHuellaEntradasMaterialConsumoV3(entradas)
	material := MaterialConsumoAutorizacionAtestadaV3{
		entradas: clonarEntradasMaterialConsumoV3(entradas),
		huella:   huella,
	}
	if material.validar() != nil {
		return vacio, ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	return material, nil
}

func extraerEntradasMaterialConsumoV3(
	decision domain.DecisionAutorizacionLigadaV3,
	motivo domain.ReferenciaEntradaCatalogo,
	resultado domain.ResultadoContextoActorRegistradoV2,
	solicitudFirma ports.SolicitudFirmaAtestacionAutorizacionV3,
	atestacion ports.AtestacionAutorizacionV3,
	prueba PruebaConfianzaAtestacionAutorizacionV3,
	capacidad CapacidadBreveAtestacionAutorizacionV3,
	raiz RaizPublicaAtestacionAutorizacionV3,
) (entradasMaterialConsumoAutorizacionAtestadaV3, error) {
	vacias := entradasMaterialConsumoAutorizacionAtestadaV3{}
	capacidadCanonica, errCapacidad := capacidad.ExportacionCanonicaParaConsumidor()
	resumenCapacidad, errResumen := capacidad.ResumenParaConsumidor()
	decisionCanonica, errDecision := domain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	motivoCanonico, errMotivo := domain.RepresentacionCanonicaMotivoAutorizacionV2(motivo)
	payload, errPayload := solicitudFirma.Mensaje()
	resultadoFirma, errResultado := atestacion.Resultado()
	if errCapacidad != nil || errResumen != nil || errDecision != nil || errMotivo != nil ||
		errPayload != nil || errResultado != nil {
		return vacias, ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	sobre, errSobre := resultadoFirma.Firma()
	evidencia, errEvidencia := prueba.ExportacionCanonicaParaConsumidor()
	spki, errSPKI := x509.MarshalPKIXPublicKey(raiz.clavePublica)
	if errSobre != nil || errEvidencia != nil || errSPKI != nil {
		return vacias, ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	return entradasMaterialConsumoAutorizacionAtestadaV3{
		capacidadCanonica:     capacidadCanonica,
		resumenCapacidad:      resumenCapacidad,
		decisionCanonica:      decisionCanonica,
		motivoCanonico:        motivoCanonico,
		contextoActorCanonico: append([]byte(nil), resultado.RepresentacionCanonica...),
		personaVersion:        resultado.Contexto.Instantanea.PersonaVersion,
		perfilVersion:         resultado.Contexto.Instantanea.PerfilVersion,
		payloadVECAD3:         payload,
		sobreCOSESign1:        sobre,
		evidenciaVerificacion: evidencia,
		raizPublicaSPKI:       spki,
	}, nil
}

func cruzarMaterialConsumoV3(
	entradas entradasMaterialConsumoAutorizacionAtestadaV3,
	datosSolicitud domain.DatosSolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	datosPrueba DatosPruebaConfianzaAtestacionAutorizacionV3,
	capacidad CapacidadBreveAtestacionAutorizacionV3,
	raiz RaizPublicaAtestacionAutorizacionV3,
) error {
	if validarEstructuraEntradasMaterialConsumoV3(entradas) != nil {
		return ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	documento, err := interpretarExportacionCapacidadV3(entradas.capacidadCanonica)
	_, decisionValidaHasta, errVentana := decision.VentanaValidez()
	huellaEfecto, errEfecto := datosSolicitud.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || errVentana != nil || errEfecto != nil {
		return ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	if !huellasMaterialConsumoV3Coinciden(documento.HuellaDecisionSHA256, entradas.decisionCanonica) ||
		!huellasMaterialConsumoV3Coinciden(documento.HuellaMotivoSHA256, entradas.motivoCanonico) ||
		!huellasMaterialConsumoV3Coinciden(documento.HuellaContextoSHA256, entradas.contextoActorCanonico) ||
		!huellasMaterialConsumoV3Coinciden(documento.HuellaPayloadVECAD3SHA256, entradas.payloadVECAD3) ||
		!huellasMaterialConsumoV3Coinciden(documento.HuellaSobreCOSESHA256, entradas.sobreCOSESign1) ||
		!huellasMaterialConsumoV3Coinciden(documento.HuellaPruebaSHA256, entradas.evidenciaVerificacion) ||
		!huellasMaterialConsumoV3Coinciden(documento.HuellaRaizSPKISHA256, entradas.raizPublicaSPKI) {
		return ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	if documento.DecisionRef != datosPrueba.ReferenciaDecision ||
		documento.HuellaDecisionSHA256 != datosPrueba.HuellaDecisionSHA256 ||
		documento.HuellaMotivoSHA256 != datosPrueba.HuellaMotivoSHA256 ||
		documento.ContextoRef != datosPrueba.ReferenciaContexto ||
		documento.HuellaContextoSHA256 != datosPrueba.HuellaContextoSHA256 ||
		documento.HuellaPayloadVECAD3SHA256 != datosPrueba.HuellaMensajeSHA256 ||
		documento.HuellaSobreCOSESHA256 != datosPrueba.HuellaSobreSHA256 ||
		documento.HuellaPruebaSHA256 != datosPrueba.HuellaPruebaSHA256 ||
		documento.Operacion != datosSolicitud.Accion ||
		documento.EfectoRef != datosSolicitud.Recurso.Referencia ||
		documento.HuellaEfectoSHA256 != huellaEfecto ||
		documento.RaizClaveID != raiz.claveID ||
		documento.RaizVersion != raiz.version ||
		documento.HuellaRaizSPKISHA256 != raiz.huellaClaveSPKISHA256 ||
		documento.AudienciaDespliegue != raiz.audienciaDespliegue ||
		documento.Suite != SuiteAtestacionAutorizacionV3COSEEdDSA {
		return ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	if entradas.resumenCapacidad.DecisionRef() != documento.DecisionRef ||
		entradas.resumenCapacidad.DecisionHuellaSHA256() != documento.HuellaDecisionSHA256 ||
		entradas.resumenCapacidad.MotivoHuellaSHA256() != documento.HuellaMotivoSHA256 ||
		entradas.resumenCapacidad.ContextoRef() != documento.ContextoRef ||
		entradas.resumenCapacidad.ContextoHuellaSHA256() != documento.HuellaContextoSHA256 ||
		entradas.resumenCapacidad.Operacion() != documento.Operacion ||
		entradas.resumenCapacidad.EfectoRef() != documento.EfectoRef ||
		entradas.resumenCapacidad.EfectoHuellaSHA256() != documento.HuellaEfectoSHA256 ||
		entradas.resumenCapacidad.AudienciaConsumo() != documento.AudienciaConsumo {
		return ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	emitidaEn, errEmitida := parsearInstanteCapacidadV3(documento.EmitidaEn)
	expiraEn, errExpira := parsearInstanteCapacidadV3(documento.ExpiraEn)
	if errEmitida != nil || errExpira != nil ||
		!entradas.resumenCapacidad.EmitidaEn().Equal(emitidaEn) ||
		!entradas.resumenCapacidad.ExpiraEn().Equal(expiraEn) {
		return ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	return cruzarVentanasYGobiernoMaterialConsumoV3(
		documento, datosPrueba, decisionValidaHasta, raiz,
	)
}

func cruzarVentanasYGobiernoMaterialConsumoV3(
	documento capacidadAtestacionAutorizacionV3JSON,
	prueba DatosPruebaConfianzaAtestacionAutorizacionV3,
	decisionValidaHasta time.Time,
	raiz RaizPublicaAtestacionAutorizacionV3,
) error {
	verificadaEn, errVerificada := parsearInstanteCapacidadV3(documento.VerificadaEn)
	configPublicada, errPublicada := parsearInstanteCapacidadV3(documento.ConfiguracionPublicadaEn)
	configExpira, errConfigExpira := parsearInstanteCapacidadV3(documento.ConfiguracionExpiraEn)
	raizDesde, errRaizDesde := parsearInstanteCapacidadV3(documento.RaizValidaDesde)
	raizHasta, errRaizHasta := parsearInstanteCapacidadV3(documento.RaizValidaHasta)
	decisionHasta, errDecisionHasta := parsearInstanteCapacidadV3(documento.DecisionValidaHasta)
	if errVerificada != nil || errPublicada != nil || errConfigExpira != nil ||
		errRaizDesde != nil || errRaizHasta != nil || errDecisionHasta != nil ||
		!decisionHasta.Equal(decisionValidaHasta) ||
		!verificadaEn.Equal(prueba.VerificadaEn) ||
		documento.RevisionConfianza != prueba.RevisionConfiguracion ||
		documento.ConfiguracionSecuencia != prueba.SecuenciaConfiguracion ||
		documento.HuellaConfiguracionSHA256 != prueba.HuellaConfiguracionSHA256 ||
		!configPublicada.Equal(prueba.ConfiguracionPublicadaEn) ||
		!configExpira.Equal(prueba.ConfiguracionExpiraEn) ||
		documento.RaizClaveID != prueba.ClaveID ||
		documento.RaizVersion != prueba.RaizVersion ||
		documento.HuellaRaizSPKISHA256 != prueba.HuellaClaveSPKISHA256 ||
		!raizDesde.Equal(prueba.RaizValidaDesde) ||
		!raizHasta.Equal(prueba.RaizValidaHasta) ||
		!raizDesde.Equal(raiz.validaDesde) ||
		!raizHasta.Equal(raiz.validaHasta) ||
		prueba.EstadoClave != raiz.estado ||
		prueba.AudienciaDespliegue != raiz.audienciaDespliegue ||
		prueba.Suite != SuiteAtestacionAutorizacionV3COSEEdDSA {
		return ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	return nil
}

func (m MaterialConsumoAutorizacionAtestadaV3) validar() error {
	if validarEstructuraEntradasMaterialConsumoV3(m.entradas) != nil {
		return ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	actual := calcularHuellaEntradasMaterialConsumoV3(m.entradas)
	if subtle.ConstantTimeCompare(actual[:], m.huella[:]) != 1 {
		return ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	return nil
}

func (m MaterialConsumoAutorizacionAtestadaV3) ExportarMaterialParaConsumidor() (
	ports.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	error,
) {
	if m.validar() != nil {
		return ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	e := m.entradas
	exportacion, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		e.capacidadCanonica,
		e.resumenCapacidad,
		e.decisionCanonica,
		e.motivoCanonico,
		e.contextoActorCanonico,
		e.personaVersion,
		e.perfilVersion,
		e.payloadVECAD3,
		e.sobreCOSESign1,
		e.evidenciaVerificacion,
		e.raizPublicaSPKI,
	)
	if err != nil {
		return ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{},
			ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	return exportacion, nil
}

func validarEstructuraEntradasMaterialConsumoV3(
	e entradasMaterialConsumoAutorizacionAtestadaV3,
) error {
	enRango := func(valor []byte, minimo, maximo int) bool {
		return len(valor) >= minimo && len(valor) <= maximo
	}
	if !enRango(e.capacidadCanonica, ports.TamanoMinimoCapacidadCanonicaV3, ports.TamanoMaximoCapacidadCanonicaV3) ||
		e.resumenCapacidad.ValidarEstructura() != nil ||
		!enRango(e.decisionCanonica, 1, ports.TamanoMaximoDecisionCanonicaV3) ||
		!enRango(e.motivoCanonico, 1, ports.TamanoMaximoMotivoCanonicoV3) ||
		!enRango(e.contextoActorCanonico, 1, ports.TamanoMaximoContextoActorCanonicoV3) ||
		e.personaVersion == 0 || e.personaVersion > ports.VersionMaximaExactaMaterialConsumoV3 ||
		e.perfilVersion == 0 || e.perfilVersion > ports.VersionMaximaExactaMaterialConsumoV3 ||
		!enRango(e.payloadVECAD3, 1, ports.TamanoMaximoPayloadVECAD3) ||
		!enRango(e.sobreCOSESign1, 1, ports.TamanoMaximoSobreCOSESign1V3) ||
		!enRango(e.evidenciaVerificacion, 1, ports.TamanoMaximoEvidenciaVerificacionV3) ||
		len(e.raizPublicaSPKI) != ports.TamanoRaizPublicaSPKIEd25519V3 {
		return ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	clave, err := x509.ParsePKIXPublicKey(e.raizPublicaSPKI)
	publica, correcta := clave.(ed25519.PublicKey)
	if err != nil || !correcta || len(publica) != ed25519.PublicKeySize {
		return ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	return nil
}

func huellasMaterialConsumoV3Coinciden(esperada string, contenido []byte) bool {
	esperadaBytes, err := hexDecodificarHuellaConfianza(esperada)
	if err != nil {
		return false
	}
	actual := sha256.Sum256(contenido)
	return subtle.ConstantTimeCompare(actual[:], esperadaBytes) == 1
}

func hexDecodificarHuellaConfianza(valor string) ([]byte, error) {
	if !huellaSHA256ConfianzaValida(valor) {
		return nil, ErrMaterialConsumoAutorizacionAtestadaV3Invalido
	}
	destino := make([]byte, sha256.Size)
	for indice := 0; indice < sha256.Size; indice++ {
		alto, okAlto := nibbleHexadecimalMaterialV3(valor[indice*2])
		bajo, okBajo := nibbleHexadecimalMaterialV3(valor[indice*2+1])
		if !okAlto || !okBajo {
			return nil, ErrMaterialConsumoAutorizacionAtestadaV3Invalido
		}
		destino[indice] = alto<<4 | bajo
	}
	return destino, nil
}

func nibbleHexadecimalMaterialV3(valor byte) (byte, bool) {
	switch {
	case valor >= '0' && valor <= '9':
		return valor - '0', true
	case valor >= 'a' && valor <= 'f':
		return valor - 'a' + 10, true
	default:
		return 0, false
	}
}

func calcularHuellaEntradasMaterialConsumoV3(
	e entradasMaterialConsumoAutorizacionAtestadaV3,
) [sha256.Size]byte {
	calculador := sha256.New()
	for _, contenido := range [][]byte{
		e.capacidadCanonica,
		[]byte(e.resumenCapacidad.DecisionRef()),
		[]byte(e.resumenCapacidad.DecisionHuellaSHA256()),
		[]byte(e.resumenCapacidad.MotivoHuellaSHA256()),
		[]byte(e.resumenCapacidad.ContextoRef()),
		[]byte(e.resumenCapacidad.ContextoHuellaSHA256()),
		[]byte(e.resumenCapacidad.Operacion()),
		[]byte(e.resumenCapacidad.EfectoRef()),
		[]byte(e.resumenCapacidad.EfectoHuellaSHA256()),
		[]byte(e.resumenCapacidad.AudienciaConsumo()),
		[]byte(e.resumenCapacidad.EmitidaEn().Format(time.RFC3339Nano)),
		[]byte(e.resumenCapacidad.ExpiraEn().Format(time.RFC3339Nano)),
		e.decisionCanonica,
		e.motivoCanonico,
		e.contextoActorCanonico,
	} {
		escribirBloqueHuellaMaterialConsumoV3(calculador, contenido)
	}
	var version [8]byte
	binary.BigEndian.PutUint64(version[:], e.personaVersion)
	escribirBloqueHuellaMaterialConsumoV3(calculador, version[:])
	binary.BigEndian.PutUint64(version[:], e.perfilVersion)
	escribirBloqueHuellaMaterialConsumoV3(calculador, version[:])
	for _, contenido := range [][]byte{
		e.payloadVECAD3, e.sobreCOSESign1, e.evidenciaVerificacion,
		e.raizPublicaSPKI,
	} {
		escribirBloqueHuellaMaterialConsumoV3(calculador, contenido)
	}
	var resultado [sha256.Size]byte
	copy(resultado[:], calculador.Sum(nil))
	return resultado
}

func escribirBloqueHuellaMaterialConsumoV3(calculador hash.Hash, contenido []byte) {
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(len(contenido)))
	_, _ = calculador.Write(longitud[:])
	_, _ = calculador.Write(contenido)
}

func clonarEntradasMaterialConsumoV3(
	e entradasMaterialConsumoAutorizacionAtestadaV3,
) entradasMaterialConsumoAutorizacionAtestadaV3 {
	e.capacidadCanonica = bytes.Clone(e.capacidadCanonica)
	e.decisionCanonica = bytes.Clone(e.decisionCanonica)
	e.motivoCanonico = bytes.Clone(e.motivoCanonico)
	e.contextoActorCanonico = bytes.Clone(e.contextoActorCanonico)
	e.payloadVECAD3 = bytes.Clone(e.payloadVECAD3)
	e.sobreCOSESign1 = bytes.Clone(e.sobreCOSESign1)
	e.evidenciaVerificacion = bytes.Clone(e.evidenciaVerificacion)
	e.raizPublicaSPKI = bytes.Clone(e.raizPublicaSPKI)
	return e
}

var _ ports.ExportadorMaterialConsumoAutorizacionAtestadaV3 = MaterialConsumoAutorizacionAtestadaV3{}
