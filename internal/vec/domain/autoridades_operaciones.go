package domain

import (
	"bytes"
	"time"
)

func (f FuenteAutoridadVersionada) ActualizarBorrador(
	revisionEsperada uint64,
	contenido ContenidoFuenteAutoridad,
	actorRef string,
	motivoCodigo CodigoMotivoFuenteAutoridad,
	instante time.Time,
) (FuenteAutoridadVersionada, error) {
	canonico, huellaAnterior, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	if canonico.Estado != EstadoFuenteAutoridadBorrador {
		return FuenteAutoridadVersionada{}, ErrTransicionAutoridadInvalida
	}
	if revisionEsperada != canonico.Revision {
		return FuenteAutoridadVersionada{}, ErrRevisionAutoridadEnConflicto
	}
	if len(canonico.EdicionesBorrador) >= maximoEdicionesFuenteAutoridad {
		return FuenteAutoridadVersionada{}, ErrLimiteAutoridadAlcanzado
	}
	contenido, err = normalizarContenidoFuenteAutoridad(contenido)
	instante = normalizarInstanteFuenteAutoridad(instante)
	if err != nil || !referenciaPersonaFuenteAutoridadValida(actorRef) || !motivoCodigo.Valido() ||
		!instanteFuenteAutoridadCanonico(instante) || !instante.After(canonico.ultimaMutacionEn()) ||
		contenido.ConocidaEn.After(instante) {
		return FuenteAutoridadVersionada{}, ErrFuenteAutoridadInvalida
	}
	contenido, huellaNueva, err := prepararHuellaContenidoFuenteAutoridad(
		canonico.ID, canonico.Version, canonico.VersionAnterior, contenido,
	)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	if huellaNueva == huellaAnterior {
		return FuenteAutoridadVersionada{}, ErrFuenteAutoridadInvalida
	}
	edicion := EdicionBorradorFuenteAutoridad{
		RevisionAnterior: canonico.Revision, RevisionNueva: canonico.Revision + 1,
		ActorRef: actorRef, MotivoCodigo: motivoCodigo, RegistradaEn: instante,
		HuellaContenidoAnteriorSHA256: huellaAnterior, HuellaContenidoNuevaSHA256: huellaNueva,
		HuellaHistoriaAnteriorSHA256: canonico.huellaHistoriaActual(),
	}
	huellaHistoriaNueva, err := huellaHistoriaEdicionBorradorFuenteAutoridad(edicion)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	edicion.HuellaHistoriaNuevaSHA256 = huellaHistoriaNueva
	canonico.EdicionesBorrador = append(canonico.EdicionesBorrador, edicion)
	canonico.Contenido = contenido
	canonico.Revision++
	return canonico.ClonarCanonica()
}

func (f FuenteAutoridadVersionada) NuevaVersionV1(
	contenido ContenidoFuenteAutoridad,
	actorRef string,
	motivoCodigo CodigoMotivoFuenteAutoridad,
	instante time.Time,
) (FuenteAutoridadVersionada, error) {
	canonico, huellaContenido, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	instante = normalizarInstanteFuenteAutoridad(instante)
	if canonico.Estado == EstadoFuenteAutoridadBorrador || canonico.Version == ^uint64(0) ||
		!referenciaPersonaFuenteAutoridadValida(actorRef) ||
		!motivoCodigo.Valido() || !instanteFuenteAutoridadCanonico(instante) ||
		!instante.After(canonico.ultimaMutacionEn()) {
		return FuenteAutoridadVersionada{}, ErrTransicionAutoridadInvalida
	}
	contenidoNormalizado, err := normalizarContenidoFuenteAutoridad(contenido)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	_, bytesContenidoActual, err := prepararContenidoFuenteAutoridad(canonico.Contenido)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	_, bytesContenidoNuevo, err := prepararContenidoFuenteAutoridad(contenidoNormalizado)
	if err != nil || bytes.Equal(bytesContenidoActual, bytesContenidoNuevo) {
		return FuenteAutoridadVersionada{}, ErrFuenteAutoridadInvalida
	}
	bytesEstado, err := serializarEstadoPersistibleFuenteAutoridadV1(canonico)
	if err != nil {
		return FuenteAutoridadVersionada{}, ErrFuenteAutoridadInvalida
	}
	linaje := ReferenciaLinajeFuenteAutoridad{
		Fuente: ReferenciaFuenteAutoridad{
			FuenteID: canonico.ID, Version: canonico.Version, HuellaContenidoSHA256: huellaContenido,
		},
		Revision: canonico.Revision, Estado: canonico.Estado,
		HuellaHistoriaSHA256: canonico.huellaHistoriaActual(),
		HuellaEstadoSHA256:   huellaBytesFuenteAutoridad(bytesEstado),
	}
	return nuevaFuenteAutoridadBorradorVersionada(
		canonico.ID, canonico.Version+1, linaje, contenidoNormalizado, actorRef, motivoCodigo, instante,
	)
}

// PrepararSolicitudTransicionV1 devuelve una solicitud y sus bytes canónicos
// firmables. El adaptador no construye JSON ni repite los parámetros al
// aplicar el acto.
func (f FuenteAutoridadVersionada) PrepararSolicitudTransicionV1(
	datos DatosPreparacionTransicionFuenteAutoridadV1,
) (SolicitudTransicionFuenteAutoridadV1, error) {
	canonico, huellaContenido, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil {
		return SolicitudTransicionFuenteAutoridadV1{}, err
	}
	datos.PreparadaEn = normalizarInstanteFuenteAutoridad(datos.PreparadaEn)
	datos.ExpiraEn = normalizarInstanteFuenteAutoridad(datos.ExpiraEn)
	compromiso, err := canonico.prepararCompromisoCanonico(
		huellaContenido, datos,
	)
	if err != nil {
		return SolicitudTransicionFuenteAutoridadV1{}, err
	}
	return nuevaSolicitudTransicionFuenteAutoridadV1(compromiso)
}

func (f FuenteAutoridadVersionada) AplicarTransicionV1(
	solicitud SolicitudTransicionFuenteAutoridadV1,
	evidencia EvidenciaActoFuenteAutoridad,
	registradaEn time.Time,
) (FuenteAutoridadVersionada, error) {
	canonico, huellaContenido, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	compromiso, err := solicitud.Compromiso()
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	esperado, err := canonico.prepararCompromisoCanonico(huellaContenido, DatosPreparacionTransicionFuenteAutoridadV1{
		EstadoNuevo: compromiso.EstadoNuevo, ActorRef: compromiso.ActorRef,
		MotivoCodigo: compromiso.MotivoCodigo, SolicitudRef: compromiso.SolicitudRef,
		PreparadaEn: compromiso.PreparadaEn, ExpiraEn: compromiso.ExpiraEn,
	})
	if err != nil || compromiso != esperado {
		return FuenteAutoridadVersionada{}, ErrSolicitudAutoridadObsoleta
	}
	bytesEsperados, err := esperado.BytesCanonicos()
	bytesSolicitud, errSolicitud := solicitud.BytesCanonicos()
	if err != nil || errSolicitud != nil || !bytes.Equal(bytesEsperados, bytesSolicitud) {
		return FuenteAutoridadVersionada{}, ErrTransicionAutoridadInvalida
	}
	evidenciaCanonica, err := evidencia.ClonarCanonica()
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	if validarEvidenciaTransicionFuenteAutoridad(evidenciaCanonica, compromiso) != nil {
		return FuenteAutoridadVersionada{}, ErrEvidenciaActoAutoridadInvalida
	}
	registradaEn = normalizarInstanteFuenteAutoridad(registradaEn)
	if !instanteFuenteAutoridadCanonico(registradaEn) || registradaEn.Before(evidenciaCanonica.ComprobadaEn) ||
		!registradaEn.Before(compromiso.ExpiraEn) || !registradaEn.After(canonico.ultimaMutacionEn()) {
		if !registradaEn.Before(compromiso.ExpiraEn) {
			return FuenteAutoridadVersionada{}, ErrSolicitudAutoridadExpirada
		}
		return FuenteAutoridadVersionada{}, ErrTransicionAutoridadInvalida
	}
	transicion := TransicionFuenteAutoridad{
		Secuencia: uint64(len(canonico.Transiciones) + 1), EstadoAnterior: canonico.Estado,
		EstadoNuevo: compromiso.EstadoNuevo, ActorRef: compromiso.ActorRef,
		MotivoCodigo: compromiso.MotivoCodigo, SolicitudRef: compromiso.SolicitudRef,
		PreparadaEn: compromiso.PreparadaEn, ExpiraEn: compromiso.ExpiraEn, RegistradaEn: registradaEn,
		Evidencia: evidenciaCanonica, HuellaHistoriaAnteriorSHA256: canonico.huellaHistoriaActual(),
	}
	huellaHistoriaNueva, err := huellaHistoriaTransicionFuenteAutoridad(transicion, compromiso)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	transicion.HuellaHistoriaNuevaSHA256 = huellaHistoriaNueva
	canonico.Estado = compromiso.EstadoNuevo
	canonico.Revision++
	canonico.Transiciones = append(canonico.Transiciones, transicion)
	return canonico.ClonarCanonica()
}

func (f FuenteAutoridadVersionada) prepararCompromisoCanonico(
	huellaContenido string,
	datos DatosPreparacionTransicionFuenteAutoridadV1,
) (CompromisoTransicionFuenteAutoridadV1, error) {
	if len(f.Transiciones) >= maximoTransicionesFuenteAutoridad ||
		!transicionPermitidaFuenteAutoridad(f.Estado, datos.EstadoNuevo) ||
		!referenciaPersonaFuenteAutoridadValida(datos.ActorRef) || !datos.MotivoCodigo.Valido() ||
		!referenciaFuenteAutoridadValida(datos.SolicitudRef) ||
		!instanteFuenteAutoridadCanonico(datos.PreparadaEn) || !datos.PreparadaEn.After(f.ultimaMutacionEn()) ||
		!instanteFuenteAutoridadCanonico(datos.ExpiraEn) || !datos.ExpiraEn.After(datos.PreparadaEn) {
		return CompromisoTransicionFuenteAutoridadV1{}, ErrTransicionAutoridadInvalida
	}
	for _, transicion := range f.Transiciones {
		if transicion.SolicitudRef == datos.SolicitudRef {
			return CompromisoTransicionFuenteAutoridadV1{}, ErrTransicionAutoridadInvalida
		}
	}
	if f.Estado == EstadoFuenteAutoridadBorrador {
		if datos.ActorRef == f.CreadaPor || f.fueEditorBorrador(datos.ActorRef) {
			return CompromisoTransicionFuenteAutoridadV1{}, ErrTransicionAutoridadInvalida
		}
	}
	return construirCompromisoTransicionFuenteAutoridad(
		f.ID, f.Version, huellaContenido, f.Revision, uint64(len(f.Transiciones)+1),
		f.Estado, datos.EstadoNuevo, datos.ActorRef, datos.MotivoCodigo, f.huellaHistoriaActual(),
		datos.SolicitudRef, datos.PreparadaEn, datos.ExpiraEn,
	)
}

func construirCompromisoTransicionFuenteAutoridad(
	ID string,
	version uint64,
	huellaContenido string,
	revisionPrevia, secuencia uint64,
	estadoAnterior, estadoNuevo EstadoFuenteAutoridad,
	actorRef string,
	motivoCodigo CodigoMotivoFuenteAutoridad,
	huellaHistoriaPrevia string,
	solicitudRef string,
	preparadaEn, expiraEn time.Time,
) (CompromisoTransicionFuenteAutoridadV1, error) {
	accion, valida := accionActoParaTransicionFuenteAutoridad(estadoAnterior, estadoNuevo)
	if !valida {
		return CompromisoTransicionFuenteAutoridadV1{}, ErrTransicionAutoridadInvalida
	}
	compromiso := CompromisoTransicionFuenteAutoridadV1{
		Esquema: esquemaCompromisoTransicionFuenteAutoridadV1, SolicitudRef: solicitudRef,
		Fuente: ReferenciaFuenteAutoridad{
			FuenteID: ID, Version: version, HuellaContenidoSHA256: huellaContenido,
		},
		RevisionPrevia: revisionPrevia, Secuencia: secuencia,
		EstadoAnterior: estadoAnterior, EstadoNuevo: estadoNuevo, Accion: accion,
		ActorRef: actorRef, MotivoCodigo: motivoCodigo,
		HuellaHistoriaPreviaSHA256: huellaHistoriaPrevia, PreparadaEn: preparadaEn, ExpiraEn: expiraEn,
	}
	if err := compromiso.Validar(); err != nil {
		return CompromisoTransicionFuenteAutoridadV1{}, err
	}
	return compromiso, nil
}

func validarEvidenciaTransicionFuenteAutoridad(
	evidencia EvidenciaActoFuenteAutoridad,
	compromiso CompromisoTransicionFuenteAutoridadV1,
) error {
	huellaCompromiso, err := compromiso.HuellaSHA256()
	huellaMensaje, errMensaje := mensajeAtestacionActoFuenteAutoridad(compromiso, evidencia).HuellaSHA256()
	if err != nil || evidencia.Validar() != nil || evidencia.Accion != compromiso.Accion ||
		evidencia.FuenteID != compromiso.Fuente.FuenteID || evidencia.FuenteVersion != compromiso.Fuente.Version ||
		evidencia.HuellaContenidoSHA256 != compromiso.Fuente.HuellaContenidoSHA256 ||
		evidencia.HuellaCompromisoSHA256 != huellaCompromiso || errMensaje != nil ||
		evidencia.HuellaMensajeAtestadoSHA256 != huellaMensaje ||
		evidencia.ComprobadaEn.Before(compromiso.PreparadaEn) ||
		!evidencia.ComprobadaEn.Before(compromiso.ExpiraEn) {
		return ErrEvidenciaActoAutoridadInvalida
	}
	return nil
}

func (f FuenteAutoridadVersionada) ultimaMutacionEn() time.Time {
	if len(f.Transiciones) != 0 {
		return f.Transiciones[len(f.Transiciones)-1].RegistradaEn
	}
	if len(f.EdicionesBorrador) != 0 {
		return f.EdicionesBorrador[len(f.EdicionesBorrador)-1].RegistradaEn
	}
	return f.CreadaEn
}

func (f FuenteAutoridadVersionada) huellaHistoriaActual() string {
	if len(f.Transiciones) != 0 {
		return f.Transiciones[len(f.Transiciones)-1].HuellaHistoriaNuevaSHA256
	}
	if len(f.EdicionesBorrador) != 0 {
		return f.EdicionesBorrador[len(f.EdicionesBorrador)-1].HuellaHistoriaNuevaSHA256
	}
	return f.HuellaHistoriaInicialSHA256
}

func (f FuenteAutoridadVersionada) fueEditorBorrador(actorRef string) bool {
	for _, edicion := range f.EdicionesBorrador {
		if edicion.ActorRef == actorRef {
			return true
		}
	}
	return false
}

func huellaHistoriaInicialFuenteAutoridad(
	id string,
	version uint64,
	versionAnterior ReferenciaLinajeFuenteAutoridad,
	huellaContenidoInicial, actorRef string,
	motivoCodigo CodigoMotivoFuenteAutoridad,
	registradaEn time.Time,
) (string, error) {
	if !esClaveDocumentalCanonica(id) || version < 1 || !esSHA256Autoridad(huellaContenidoInicial) ||
		!referenciaPersonaFuenteAutoridadValida(actorRef) || !motivoCodigo.Valido() ||
		!instanteFuenteAutoridadCanonico(registradaEn) {
		return "", ErrFuenteAutoridadInvalida
	}
	return huellaValorFuenteAutoridad(historiaInicialPersistibleAutoridadV1{
		Esquema: "vec.fuente_autoridad.historia_inicial.v1", ID: id, Version: uint64(version),
		VersionAnteriorFuenteID:       versionAnterior.Fuente.FuenteID,
		VersionAnteriorNumero:         versionAnterior.Fuente.Version,
		VersionAnteriorHuellaSHA256:   versionAnterior.Fuente.HuellaContenidoSHA256,
		VersionAnteriorRevision:       versionAnterior.Revision,
		VersionAnteriorEstado:         string(versionAnterior.Estado),
		VersionAnteriorHistoriaSHA256: versionAnterior.HuellaHistoriaSHA256,
		VersionAnteriorEstadoSHA256:   versionAnterior.HuellaEstadoSHA256,
		HuellaContenidoInicialSHA256:  huellaContenidoInicial, ActorRef: actorRef,
		MotivoCodigo: string(motivoCodigo), RegistradaEn: textoInstantePersistibleAutoridadV1(registradaEn),
	}, maximoBytesSobreContenido)
}

func huellaHistoriaEdicionBorradorFuenteAutoridad(edicion EdicionBorradorFuenteAutoridad) (string, error) {
	if edicion.RevisionAnterior == 0 || edicion.RevisionNueva != edicion.RevisionAnterior+1 ||
		!referenciaPersonaFuenteAutoridadValida(edicion.ActorRef) || !edicion.MotivoCodigo.Valido() ||
		!instanteFuenteAutoridadCanonico(edicion.RegistradaEn) ||
		!esSHA256Autoridad(edicion.HuellaContenidoAnteriorSHA256) ||
		!esSHA256Autoridad(edicion.HuellaContenidoNuevaSHA256) ||
		!esSHA256Autoridad(edicion.HuellaHistoriaAnteriorSHA256) {
		return "", ErrFuenteAutoridadInvalida
	}
	return huellaValorFuenteAutoridad(historiaEdicionPersistibleAutoridadV1{
		Esquema:          "vec.fuente_autoridad.historia_edicion.v1",
		RevisionAnterior: edicion.RevisionAnterior, RevisionNueva: edicion.RevisionNueva,
		ActorRef: edicion.ActorRef, MotivoCodigo: string(edicion.MotivoCodigo),
		RegistradaEn:                  textoInstantePersistibleAutoridadV1(edicion.RegistradaEn),
		HuellaContenidoAnteriorSHA256: edicion.HuellaContenidoAnteriorSHA256,
		HuellaContenidoNuevaSHA256:    edicion.HuellaContenidoNuevaSHA256,
		HuellaHistoriaAnteriorSHA256:  edicion.HuellaHistoriaAnteriorSHA256,
	}, maximoBytesSobreContenido)
}

func huellaHistoriaTransicionFuenteAutoridad(
	transicion TransicionFuenteAutoridad,
	compromiso CompromisoTransicionFuenteAutoridadV1,
) (string, error) {
	huellaCompromiso, err := compromiso.HuellaSHA256()
	if err != nil || transicion.HuellaHistoriaAnteriorSHA256 != compromiso.HuellaHistoriaPreviaSHA256 ||
		transicion.Evidencia.HuellaCompromisoSHA256 != huellaCompromiso {
		return "", ErrFuenteAutoridadInvalida
	}
	return huellaValorFuenteAutoridad(historiaTransicionPersistibleAutoridadV1{
		Esquema:                      "vec.fuente_autoridad.historia_transicion.v1",
		HuellaHistoriaAnteriorSHA256: transicion.HuellaHistoriaAnteriorSHA256,
		HuellaCompromisoSHA256:       huellaCompromiso, EvidenciaRef: transicion.Evidencia.EvidenciaRef,
		HuellaMensajeAtestadoSHA256: transicion.Evidencia.HuellaMensajeAtestadoSHA256,
		AtestacionRef:               transicion.Evidencia.AtestacionRef,
		HuellaAtestacionSHA256:      transicion.Evidencia.HuellaAtestacionSHA256,
		FirmaAtestacionRef:          transicion.Evidencia.FirmaAtestacionRef,
		RegistradaEn:                textoInstantePersistibleAutoridadV1(transicion.RegistradaEn),
	}, maximoBytesSobreContenido)
}

func transicionPermitidaFuenteAutoridad(anterior, nueva EstadoFuenteAutoridad) bool {
	switch anterior {
	case EstadoFuenteAutoridadBorrador:
		return nueva == EstadoFuenteAutoridadPublicada
	case EstadoFuenteAutoridadPublicada:
		return nueva == EstadoFuenteAutoridadSuspendida || nueva == EstadoFuenteAutoridadDerogada
	case EstadoFuenteAutoridadSuspendida:
		return nueva == EstadoFuenteAutoridadPublicada || nueva == EstadoFuenteAutoridadDerogada
	default:
		return false
	}
}

func accionActoParaTransicionFuenteAutoridad(anterior, nueva EstadoFuenteAutoridad) (AccionActoFuenteAutoridad, bool) {
	switch {
	case anterior == EstadoFuenteAutoridadBorrador && nueva == EstadoFuenteAutoridadPublicada:
		return AccionActoPublicarFuenteAutoridad, true
	case anterior == EstadoFuenteAutoridadPublicada && nueva == EstadoFuenteAutoridadSuspendida:
		return AccionActoSuspenderFuenteAutoridad, true
	case anterior == EstadoFuenteAutoridadSuspendida && nueva == EstadoFuenteAutoridadPublicada:
		return AccionActoLevantarSuspensionFuenteAutoridad, true
	case (anterior == EstadoFuenteAutoridadPublicada || anterior == EstadoFuenteAutoridadSuspendida) && nueva == EstadoFuenteAutoridadDerogada:
		return AccionActoDerogarFuenteAutoridad, true
	default:
		return "", false
	}
}
