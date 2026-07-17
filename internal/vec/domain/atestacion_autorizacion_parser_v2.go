package domain

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"time"
)

var (
	// ErrParseoAtestacionAutorizacionV2Invalido identifica un VEC-AD-2 que no
	// es completo, canonico o semanticamente coherente. Parsearlo no verifica
	// una firma y nunca concede autoridad.
	ErrParseoAtestacionAutorizacionV2Invalido = errors.New("vec: parseo no autoritativo VEC-AD-2 invalido")

	// ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida impide que
	// la proyeccion nominal pueda terminar accidentalmente en logs o codecs.
	ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida = errors.New("vec: serializacion de proyeccion no autoritativa VEC-AD-2 prohibida")
)

const representacionRedactadaProyeccionAtestacionAutorizacionV2 = "[PROYECCION-VEC-AD-2-NOMINAL-NO-AUTORITATIVA-REDACTADA]"

// datosDecisionAtestacionAutorizacionV2NoAutoritativos contiene, solo dentro
// del dominio, los 35 campos contractuales de DecisionAutorizacion. El vinculo
// se representa mediante sus 25 datos y nunca como la capacidad opaca viva.
// Este tipo no se exporta ni se devuelve al llamador.
type datosDecisionAtestacionAutorizacionV2NoAutoritativos struct {
	DecisionRef                           string
	Concedida                             bool
	Codigo                                string
	PrincipalID                           string
	PerfilActivoRef                       string
	Accion                                string
	RecursoRef                            string
	ModuloID                              string
	TipoRecurso                           string
	ContextoRecursoHuellaSHA256           string
	Finalidad                             string
	CorrelacionRef                        string
	EsquemaHuellaSolicitud                string
	SolicitudHuellaSHA256                 string
	EsquemaHuellaMotivo                   string
	MotivoHuellaSHA256                    string
	VinculoAutenticacionActor             DatosVinculoAutenticacionActorV1
	AsignacionRef                         string
	AsignacionHuellaSHA256                string
	VersionRolRef                         string
	VersionRolHuellaSHA256                string
	ControlVigenciaVersionRolRef          string
	ControlVigenciaVersionRolRevision     uint64
	ControlVigenciaVersionRolHuellaSHA256 string
	RevisionCatalogoPoliticas             uint64
	CatalogoPoliticasHuellaSHA256         string
	PoliticasEvaluadasRefs                []string
	PoliticasEvaluadasHuellasSHA256       map[string]string
	PoliticasRefs                         []string
	PoliticasHuellasSHA256                map[string]string
	GarantiaMinima                        AuthAssurance
	CamposPermitidos                      []string
	Obligaciones                          []string
	EmitidaEn                             time.Time
	ValidaHasta                           time.Time
}

type datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos struct {
	catalogoID           string
	catalogoVersion      uint64
	catalogoHuellaSHA256 string
	entradaClave         string
}

// ProyeccionAtestacionAutorizacionV2NoAutoritativa acredita unicamente que un
// buffer tiene la forma canonica VEC-AD-2. Sus campos son privados, no contiene
// DecisionAutorizacion ni VinculoAutenticacionActorV1 y no puede serializarse.
// Antes de utilizar sus compromisos, otra capa debe verificar el sobre, la
// procedencia, la vigencia, la revocacion y el consumo unico.
type ProyeccionAtestacionAutorizacionV2NoAutoritativa struct {
	cabecera CabeceraAtestacionAutorizacionV2
	datos    *datosDecisionAtestacionAutorizacionV2NoAutoritativos
	motivo   *datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos
}

// ParsearMensajeAtestacionAutorizacionV2NoAutoritativo lee estrictamente los
// 35 campos de decision, los 25 del vinculo y las cuatro coordenadas del motivo.
// Los limites del mensaje, textos y colecciones se comprueban antes de reservar
// memoria. Al final se exige una reserializacion byte a byte identica.
func ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(
	contenido []byte,
) (ProyeccionAtestacionAutorizacionV2NoAutoritativa, error) {
	if len(contenido) == 0 || len(contenido) > TamanoMaximoMensajeAtestacionAutorizacionV2 ||
		!comprobarEsquemaDecisionAtestacionAutorizacionV2() ||
		!limitesEscritorAtestacionAutorizacionV2Compatibles(
			TamanoMaximoMensajeAtestacionAutorizacionV2,
			TamanoMaximoMensajeAtestacionAutorizacionV1,
		) {
		return ProyeccionAtestacionAutorizacionV2NoAutoritativa{}, errorParseoAtestacionAutorizacionV2()
	}

	cabeceraCruda, datos, motivo, err := parsearMensajeAtestacionSolicitudLigadaNoAutoritativo(
		contenido,
		EsquemaMensajeAtestacionAutorizacionV2,
	)
	cabecera := CabeceraAtestacionAutorizacionV2{
		FormatoVersion: cabeceraCruda.formatoVersion,
		Suite:          cabeceraCruda.suite,
		ClaveID:        cabeceraCruda.claveID,
		Audiencia:      cabeceraCruda.audiencia,
	}
	if err != nil || cabecera.Validar() != nil ||
		validarDatosAtestacionSolicitudLigadaNoAutoritativos(datos, motivo, true) != nil {
		return ProyeccionAtestacionAutorizacionV2NoAutoritativa{}, errorParseoAtestacionAutorizacionV2()
	}

	canonico, err := serializarMensajeAtestacionSolicitudLigadaNoAutoritativo(
		EsquemaMensajeAtestacionAutorizacionV2,
		cabecera.FormatoVersion,
		cabecera.Suite,
		cabecera.ClaveID,
		cabecera.Audiencia,
		datos,
		motivo,
		TamanoMaximoMensajeAtestacionAutorizacionV2,
	)
	if err != nil || !bytes.Equal(canonico, contenido) {
		return ProyeccionAtestacionAutorizacionV2NoAutoritativa{}, errorParseoAtestacionAutorizacionV2()
	}

	copiaDatos := clonarDatosAtestacionSolicitudLigadaNoAutoritativos(datos)
	copiaMotivo := motivo
	return ProyeccionAtestacionAutorizacionV2NoAutoritativa{
		cabecera: cabecera,
		datos:    &copiaDatos,
		motivo:   &copiaMotivo,
	}, nil
}

// Cabecera devuelve solo la seleccion nominal de suite, clave y audiencia. No
// afirma que esa configuracion haya sido aprobada ni que exista una firma.
func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) Cabecera() (
	CabeceraAtestacionAutorizacionV2,
	error,
) {
	if p.validar() != nil {
		return CabeceraAtestacionAutorizacionV2{}, errorParseoAtestacionAutorizacionV2()
	}
	return p.cabecera, nil
}

// DecisionRef devuelve el identificador opaco nominal, no una capacidad.
func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) DecisionRef() (string, error) {
	if p.validar() != nil {
		return "", errorParseoAtestacionAutorizacionV2()
	}
	return p.datos.DecisionRef, nil
}

// SolicitudHuellaSHA256 devuelve el compromiso nominal de la solicitud. No
// prueba por si mismo que la solicitud existiera o fuese evaluada por el PDP.
func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) SolicitudHuellaSHA256() (string, error) {
	if p.validar() != nil {
		return "", errorParseoAtestacionAutorizacionV2()
	}
	return p.datos.SolicitudHuellaSHA256, nil
}

// MotivoHuellaSHA256 devuelve solo el compromiso nominal de la referencia de
// motivo. Las cuatro coordenadas completas permanecen deliberadamente ocultas.
func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) MotivoHuellaSHA256() (string, error) {
	if p.validar() != nil {
		return "", errorParseoAtestacionAutorizacionV2()
	}
	return p.datos.MotivoHuellaSHA256, nil
}

func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) validar() error {
	if p.datos == nil || p.motivo == nil || p.cabecera.Validar() != nil ||
		validarDatosAtestacionSolicitudLigadaNoAutoritativos(*p.datos, *p.motivo, true) != nil {
		return errorParseoAtestacionAutorizacionV2()
	}
	return nil
}

type cabeceraAtestacionSolicitudLigadaNoAutoritativa struct {
	formatoVersion uint16
	suite          string
	claveID        string
	audiencia      string
}

func parsearMensajeAtestacionSolicitudLigadaNoAutoritativo(
	contenido []byte,
	esquema string,
) (
	cabeceraAtestacionSolicitudLigadaNoAutoritativa,
	datosDecisionAtestacionAutorizacionV2NoAutoritativos,
	datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos,
	error,
) {
	// El trailer es redundante a proposito: rechazarlo antes de recorrer hasta
	// 512 KiB evita trabajo y reservas sobre un mensaje obviamente truncado o
	// concatenado. Se vuelve a comprobar al final para conservar defensa en
	// profundidad frente a cambios futuros del lector.
	if len(contenido) < 8 ||
		binary.BigEndian.Uint64(contenido[len(contenido)-8:]) != uint64(len(contenido)) {
		return cabeceraAtestacionSolicitudLigadaNoAutoritativa{},
			datosDecisionAtestacionAutorizacionV2NoAutoritativos{},
			datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos{},
			errorParseoAtestacionAutorizacionV2()
	}
	lector := lectorHistoricoAtestacionAutorizacionV1{contenido: contenido}
	lector.exigirBytes([]byte(esquema))
	lector.exigirByte(0)
	cabecera := cabeceraAtestacionSolicitudLigadaNoAutoritativa{
		formatoVersion: lector.leerUint16(),
		suite:          lector.leerTexto(128),
		claveID:        lector.leerTexto(512),
		audiencia:      lector.leerTexto(512),
	}
	datos := datosDecisionAtestacionAutorizacionV2NoAutoritativos{
		DecisionRef:                           lector.leerTexto(512),
		Concedida:                             lector.leerBooleano(),
		Codigo:                                lector.leerTexto(128),
		PrincipalID:                           lector.leerTexto(512),
		PerfilActivoRef:                       lector.leerTexto(512),
		Accion:                                lector.leerTexto(256),
		RecursoRef:                            lector.leerTexto(512),
		ModuloID:                              lector.leerTexto(128),
		TipoRecurso:                           lector.leerTexto(128),
		ContextoRecursoHuellaSHA256:           lector.leerTexto(64),
		Finalidad:                             lector.leerTexto(512),
		CorrelacionRef:                        lector.leerTexto(512),
		EsquemaHuellaSolicitud:                lector.leerTexto(128),
		SolicitudHuellaSHA256:                 lector.leerTexto(64),
		EsquemaHuellaMotivo:                   lector.leerTexto(128),
		MotivoHuellaSHA256:                    lector.leerTexto(64),
		VinculoAutenticacionActor:             leerDatosVinculoAutenticacionActorV1Historico(&lector),
		AsignacionRef:                         lector.leerTexto(512),
		AsignacionHuellaSHA256:                lector.leerTexto(64),
		VersionRolRef:                         lector.leerTexto(512),
		VersionRolHuellaSHA256:                lector.leerTexto(64),
		ControlVigenciaVersionRolRef:          lector.leerTexto(512),
		ControlVigenciaVersionRolRevision:     lector.leerUint64(),
		ControlVigenciaVersionRolHuellaSHA256: lector.leerTexto(64),
		RevisionCatalogoPoliticas:             lector.leerUint64(),
		CatalogoPoliticasHuellaSHA256:         lector.leerTexto(64),
		PoliticasEvaluadasRefs:                leerListaAtestacionSolicitudLigadaAcotada(&lector),
		PoliticasEvaluadasHuellasSHA256:       leerMapaAtestacionSolicitudLigadaAcotado(&lector),
		PoliticasRefs:                         leerListaAtestacionSolicitudLigadaAcotada(&lector),
		PoliticasHuellasSHA256:                leerMapaAtestacionSolicitudLigadaAcotado(&lector),
		GarantiaMinima:                        AuthAssurance(lector.leerTexto(32)),
		CamposPermitidos:                      leerListaAtestacionSolicitudLigadaAcotada(&lector),
		Obligaciones:                          leerListaAtestacionSolicitudLigadaAcotada(&lector),
		EmitidaEn:                             lector.leerInstante(),
		ValidaHasta:                           lector.leerInstante(),
	}
	motivo := datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos{
		catalogoID:           lector.leerTexto(512),
		catalogoVersion:      lector.leerUint64(),
		catalogoHuellaSHA256: lector.leerTexto(64),
		entradaClave:         lector.leerTexto(64),
	}
	longitud := lector.leerUint64()
	if lector.err != nil || lector.posicion != len(contenido) || longitud != uint64(len(contenido)) {
		return cabeceraAtestacionSolicitudLigadaNoAutoritativa{},
			datosDecisionAtestacionAutorizacionV2NoAutoritativos{},
			datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos{},
			errorParseoAtestacionAutorizacionV2()
	}
	return cabecera, datos, motivo, nil
}

// Estas dos lecturas comprueban tanto el maximo contractual como el minimo de
// bytes aun disponibles antes de make. Un conteo hostil en un mensaje truncado
// no provoca una reserva proporcional al valor declarado.
func leerListaAtestacionSolicitudLigadaAcotada(
	lector *lectorHistoricoAtestacionAutorizacionV1,
) []string {
	cantidad := lector.leerUint32()
	if lector.err != nil || cantidad > maximoElementosAutorizacion ||
		uint64(cantidad)*4 > uint64(len(lector.contenido)-lector.posicion) {
		lector.err = errorParseoAtestacionAutorizacionV2()
		return nil
	}
	valores := make([]string, 0, int(cantidad))
	for indice := uint32(0); indice < cantidad; indice++ {
		valor := lector.leerTexto(512)
		if lector.err != nil || (len(valores) > 0 && bytes.Compare(
			[]byte(valores[len(valores)-1]), []byte(valor),
		) >= 0) {
			lector.err = errorParseoAtestacionAutorizacionV2()
			return nil
		}
		valores = append(valores, valor)
	}
	return valores
}

func leerMapaAtestacionSolicitudLigadaAcotado(
	lector *lectorHistoricoAtestacionAutorizacionV1,
) map[string]string {
	cantidad := lector.leerUint32()
	if lector.err != nil || cantidad > maximoElementosAutorizacion ||
		uint64(cantidad)*8 > uint64(len(lector.contenido)-lector.posicion) {
		lector.err = errorParseoAtestacionAutorizacionV2()
		return nil
	}
	valores := make(map[string]string, int(cantidad))
	anterior := ""
	for indice := uint32(0); indice < cantidad; indice++ {
		clave := lector.leerTexto(512)
		valor := lector.leerTexto(64)
		if lector.err != nil || (indice > 0 && bytes.Compare([]byte(anterior), []byte(clave)) >= 0) {
			lector.err = errorParseoAtestacionAutorizacionV2()
			return nil
		}
		if _, repetida := valores[clave]; repetida {
			lector.err = errorParseoAtestacionAutorizacionV2()
			return nil
		}
		valores[clave] = valor
		anterior = clave
	}
	return valores
}

func validarDatosAtestacionSolicitudLigadaNoAutoritativos(
	d datosDecisionAtestacionAutorizacionV2NoAutoritativos,
	m datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos,
	concesion bool,
) error {
	v := d.VinculoAutenticacionActor
	if v.Validar() != nil ||
		!textoAutorizacionSinComodinSeguro(d.DecisionRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.Codigo, 128, false) ||
		!textoAutorizacionSinComodinSeguro(d.PrincipalID, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.PerfilActivoRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.Accion, 256, false) ||
		!textoAutorizacionSinComodinSeguro(d.RecursoRef, 512, false) ||
		!textoAutorizacionSinComodinSeguro(d.ModuloID, 128, false) ||
		!textoAutorizacionSinComodinSeguro(d.TipoRecurso, 128, false) ||
		!huellaSHA256AutorizacionValida(d.ContextoRecursoHuellaSHA256) ||
		!textoAutorizacionSinComodinSeguro(d.Finalidad, 512, false) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(d.CorrelacionRef) ||
		d.EsquemaHuellaSolicitud != EsquemaHuellaSolicitudAutorizacionV2 ||
		!huellaSHA256AutorizacionValida(d.SolicitudHuellaSHA256) ||
		d.SolicitudHuellaSHA256 == huellaSHA256Nula ||
		d.EsquemaHuellaMotivo != EsquemaHuellaMotivoAutorizacionV2 ||
		!huellaSHA256AutorizacionValida(d.MotivoHuellaSHA256) ||
		d.MotivoHuellaSHA256 == huellaSHA256Nula ||
		d.PrincipalID != v.PrincipalID || d.PerfilActivoRef != v.PerfilActivoRef ||
		!textoAutorizacionSinComodinSeguro(d.AsignacionRef, 512, false) ||
		!huellaSHA256AutorizacionValida(d.AsignacionHuellaSHA256) ||
		!textoAutorizacionSinComodinSeguro(d.VersionRolRef, 512, false) ||
		!huellaSHA256AutorizacionValida(d.VersionRolHuellaSHA256) ||
		d.ControlVigenciaVersionRolRef != d.VersionRolRef ||
		d.ControlVigenciaVersionRolRevision == 0 ||
		!huellaSHA256AutorizacionValida(d.ControlVigenciaVersionRolHuellaSHA256) ||
		d.RevisionCatalogoPoliticas == 0 ||
		!huellaSHA256AutorizacionValida(d.CatalogoPoliticasHuellaSHA256) ||
		!listaAutorizacionValida(d.PoliticasEvaluadasRefs, false, false) ||
		!listaAutorizacionValida(d.PoliticasRefs, false, false) ||
		!listaAutorizacionValida(d.CamposPermitidos, false, false) ||
		!listaAutorizacionValida(d.Obligaciones, false, false) ||
		!listaAtestacionAutorizacionCanonica(d.PoliticasEvaluadasRefs) ||
		!listaAtestacionAutorizacionCanonica(d.PoliticasRefs) ||
		!listaAtestacionAutorizacionCanonica(d.CamposPermitidos) ||
		!listaAtestacionAutorizacionCanonica(d.Obligaciones) ||
		!instanteAutorizacionCanonico(d.EmitidaEn) || !instanteAutorizacionCanonico(d.ValidaHasta) ||
		!d.ValidaHasta.After(d.EmitidaEn) || d.EmitidaEn.Before(v.SesionRevalidadaEn) ||
		d.ValidaHasta.After(v.SesionValidaHasta) ||
		d.ValidaHasta.Sub(d.EmitidaEn) > VigenciaMaximaDecisionAutorizacion {
		return errorParseoAtestacionAutorizacionV2()
	}
	if concesion {
		if !d.Concedida || d.Codigo != "concedida" || !d.GarantiaMinima.Valida() ||
			!CumpleGarantiaAutenticacion(v.GarantiaObservada, d.GarantiaMinima) {
			return errorParseoAtestacionAutorizacionV2()
		}
	} else if d.Concedida || d.Codigo == "concedida" ||
		(d.GarantiaMinima != "" && !d.GarantiaMinima.Valida()) {
		return errorParseoAtestacionAutorizacionV2()
	}

	huellaCatalogo, err := huellaCatalogoPoliticasDesdeEvidencias(
		d.PoliticasEvaluadasRefs,
		d.PoliticasEvaluadasHuellasSHA256,
	)
	if err != nil || huellaCatalogo != d.CatalogoPoliticasHuellaSHA256 ||
		len(d.PoliticasHuellasSHA256) != len(d.PoliticasRefs) {
		return errorParseoAtestacionAutorizacionV2()
	}
	for _, referencia := range d.PoliticasRefs {
		huellaAplicada, aplicada := d.PoliticasHuellasSHA256[referencia]
		huellaEvaluada, evaluada := d.PoliticasEvaluadasHuellasSHA256[referencia]
		if !aplicada || !evaluada || !huellaSHA256AutorizacionValida(huellaAplicada) ||
			huellaAplicada != huellaEvaluada {
			return errorParseoAtestacionAutorizacionV2()
		}
	}

	referenciaMotivo, valida := referenciaMotivoAtestacionSolicitudLigadaDesdeDatos(m)
	if !valida {
		return errorParseoAtestacionAutorizacionV2()
	}
	huellaMotivo, err := HuellaSHA256MotivoAutorizacionV2(referenciaMotivo)
	if err != nil || huellaMotivo != d.MotivoHuellaSHA256 {
		return errorParseoAtestacionAutorizacionV2()
	}
	return nil
}

func referenciaMotivoAtestacionSolicitudLigadaDesdeDatos(
	m datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos,
) (ReferenciaEntradaCatalogo, bool) {
	if m.catalogoVersion == 0 || m.catalogoVersion > math.MaxInt32 {
		return ReferenciaEntradaCatalogo{}, false
	}
	referencia := ReferenciaEntradaCatalogo{
		CatalogoID:           m.catalogoID,
		CatalogoVersion:      int(m.catalogoVersion),
		CatalogoHuellaSHA256: m.catalogoHuellaSHA256,
		EntradaClave:         m.entradaClave,
	}
	return referencia, ReferenciaMotivoAutorizacionV2Valida(referencia)
}

func serializarMensajeAtestacionSolicitudLigadaNoAutoritativo(
	esquema string,
	formatoVersion uint16,
	suite string,
	claveID string,
	audiencia string,
	d datosDecisionAtestacionAutorizacionV2NoAutoritativos,
	m datosReferenciaMotivoAtestacionAutorizacionV2NoAutoritativos,
	limite int,
) ([]byte, error) {
	escritor := nuevoEscritorAtestacionAutorizacionV1()
	escritor.escribirBytes([]byte(esquema))
	escritor.escribirByte(0)
	escritor.escribirUint16(formatoVersion)
	escritor.escribirTexto(suite)
	escritor.escribirTexto(claveID)
	escritor.escribirTexto(audiencia)
	escritor.escribirTexto(d.DecisionRef)
	escritor.escribirBooleano(d.Concedida)
	escritor.escribirTexto(d.Codigo)
	escritor.escribirTexto(d.PrincipalID)
	escritor.escribirTexto(d.PerfilActivoRef)
	escritor.escribirTexto(d.Accion)
	escritor.escribirTexto(d.RecursoRef)
	escritor.escribirTexto(d.ModuloID)
	escritor.escribirTexto(d.TipoRecurso)
	escritor.escribirTexto(d.ContextoRecursoHuellaSHA256)
	escritor.escribirTexto(d.Finalidad)
	escritor.escribirTexto(d.CorrelacionRef)
	escritor.escribirTexto(d.EsquemaHuellaSolicitud)
	escritor.escribirTexto(d.SolicitudHuellaSHA256)
	escritor.escribirTexto(d.EsquemaHuellaMotivo)
	escritor.escribirTexto(d.MotivoHuellaSHA256)
	escribirDatosVinculoAutenticacionActorV1Historico(escritor, d.VinculoAutenticacionActor)
	escritor.escribirTexto(d.AsignacionRef)
	escritor.escribirTexto(d.AsignacionHuellaSHA256)
	escritor.escribirTexto(d.VersionRolRef)
	escritor.escribirTexto(d.VersionRolHuellaSHA256)
	escritor.escribirTexto(d.ControlVigenciaVersionRolRef)
	escritor.escribirUint64(d.ControlVigenciaVersionRolRevision)
	escritor.escribirTexto(d.ControlVigenciaVersionRolHuellaSHA256)
	escritor.escribirUint64(d.RevisionCatalogoPoliticas)
	escritor.escribirTexto(d.CatalogoPoliticasHuellaSHA256)
	escritor.escribirLista(d.PoliticasEvaluadasRefs)
	escritor.escribirMapa(d.PoliticasEvaluadasHuellasSHA256)
	escritor.escribirLista(d.PoliticasRefs)
	escritor.escribirMapa(d.PoliticasHuellasSHA256)
	escritor.escribirTexto(string(d.GarantiaMinima))
	escritor.escribirLista(d.CamposPermitidos)
	escritor.escribirLista(d.Obligaciones)
	escritor.escribirInstante(d.EmitidaEn)
	escritor.escribirInstante(d.ValidaHasta)
	escritor.escribirTexto(m.catalogoID)
	escritor.escribirUint64(m.catalogoVersion)
	escritor.escribirTexto(m.catalogoHuellaSHA256)
	escritor.escribirTexto(m.entradaClave)
	if escritor.err != nil || limite < 8 || escritor.buffer.Len() > limite-8 {
		return nil, errorParseoAtestacionAutorizacionV2()
	}
	longitud := uint64(escritor.buffer.Len() + 8)
	escritor.escribirUint64(longitud)
	if escritor.err != nil || escritor.buffer.Len() != int(longitud) || escritor.buffer.Len() > limite {
		return nil, errorParseoAtestacionAutorizacionV2()
	}
	return append([]byte(nil), escritor.buffer.Bytes()...), nil
}

func clonarDatosAtestacionSolicitudLigadaNoAutoritativos(
	d datosDecisionAtestacionAutorizacionV2NoAutoritativos,
) datosDecisionAtestacionAutorizacionV2NoAutoritativos {
	copia := d
	copia.PoliticasEvaluadasRefs = append([]string(nil), d.PoliticasEvaluadasRefs...)
	copia.PoliticasRefs = append([]string(nil), d.PoliticasRefs...)
	copia.CamposPermitidos = append([]string(nil), d.CamposPermitidos...)
	copia.Obligaciones = append([]string(nil), d.Obligaciones...)
	copia.PoliticasEvaluadasHuellasSHA256 = clonarMapaTextoHistoricoAtestacionAutorizacionV1(
		d.PoliticasEvaluadasHuellasSHA256,
	)
	copia.PoliticasHuellasSHA256 = clonarMapaTextoHistoricoAtestacionAutorizacionV1(
		d.PoliticasHuellasSHA256,
	)
	return copia
}

func errorParseoAtestacionAutorizacionV2() error {
	return errors.Join(ErrParseoAtestacionAutorizacionV2Invalido, ErrMensajeAtestacionAutorizacionInvalido)
}

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) String() string {
	return representacionRedactadaProyeccionAtestacionAutorizacionV2
}

func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) GoString() string { return p.String() }

func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}

func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) LogValue() slog.Value {
	return slog.StringValue(p.String())
}

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) UnmarshalJSON([]byte) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) UnmarshalText([]byte) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) UnmarshalBinary([]byte) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) GobDecode([]byte) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) UnmarshalCBOR([]byte) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) MarshalYAML() (any, error) {
	return nil, ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida
}
