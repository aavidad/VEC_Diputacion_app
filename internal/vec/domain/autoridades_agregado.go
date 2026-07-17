package domain

import "time"

type TransicionFuenteAutoridad struct {
	Secuencia                    uint64                       `json:"secuencia"`
	EstadoAnterior               EstadoFuenteAutoridad        `json:"estado_anterior"`
	EstadoNuevo                  EstadoFuenteAutoridad        `json:"estado_nuevo"`
	ActorRef                     string                       `json:"actor_ref"`
	MotivoCodigo                 CodigoMotivoFuenteAutoridad  `json:"motivo_codigo"`
	SolicitudRef                 string                       `json:"solicitud_ref"`
	PreparadaEn                  time.Time                    `json:"preparada_en"`
	ExpiraEn                     time.Time                    `json:"expira_en"`
	RegistradaEn                 time.Time                    `json:"registrada_en"`
	Evidencia                    EvidenciaActoFuenteAutoridad `json:"evidencia"`
	HuellaHistoriaAnteriorSHA256 string                       `json:"huella_historia_anterior_sha256"`
	HuellaHistoriaNuevaSHA256    string                       `json:"huella_historia_nueva_sha256"`
}

// EdicionBorradorFuenteAutoridad conserva todos los actores que alteraron el
// borrador y encadena la huella anterior con la nueva.
type EdicionBorradorFuenteAutoridad struct {
	RevisionAnterior              uint64                      `json:"revision_anterior"`
	RevisionNueva                 uint64                      `json:"revision_nueva"`
	ActorRef                      string                      `json:"actor_ref"`
	MotivoCodigo                  CodigoMotivoFuenteAutoridad `json:"motivo_codigo"`
	RegistradaEn                  time.Time                   `json:"registrada_en"`
	HuellaContenidoAnteriorSHA256 string                      `json:"huella_contenido_anterior_sha256"`
	HuellaContenidoNuevaSHA256    string                      `json:"huella_contenido_nueva_sha256"`
	HuellaHistoriaAnteriorSHA256  string                      `json:"huella_historia_anterior_sha256"`
	HuellaHistoriaNuevaSHA256     string                      `json:"huella_historia_nueva_sha256"`
}

// FuenteAutoridadVersionada registra autoridad documental, no una regla de
// negocio. Las revisiones anteriores viven en el repositorio append-only.
type FuenteAutoridadVersionada struct {
	ID                           string                           `json:"id"`
	Version                      uint64                           `json:"version"`
	Revision                     uint64                           `json:"revision"`
	VersionAnterior              ReferenciaLinajeFuenteAutoridad  `json:"version_anterior,omitempty"`
	Contenido                    ContenidoFuenteAutoridad         `json:"contenido"`
	HuellaContenidoInicialSHA256 string                           `json:"huella_contenido_inicial_sha256"`
	HuellaHistoriaInicialSHA256  string                           `json:"huella_historia_inicial_sha256"`
	Estado                       EstadoFuenteAutoridad            `json:"estado"`
	CreadaPor                    string                           `json:"creada_por"`
	CreadaEn                     time.Time                        `json:"creada_en"`
	MotivoCreacionCodigo         CodigoMotivoFuenteAutoridad      `json:"motivo_creacion_codigo"`
	EdicionesBorrador            []EdicionBorradorFuenteAutoridad `json:"ediciones_borrador,omitempty"`
	Transiciones                 []TransicionFuenteAutoridad      `json:"transiciones,omitempty"`
}

type DatosAltaFuenteAutoridadV1 struct {
	ID                   string
	Contenido            ContenidoFuenteAutoridad
	CreadaPor            string
	CreadaEn             time.Time
	MotivoCreacionCodigo CodigoMotivoFuenteAutoridad
}

func NuevaFuenteAutoridadBorradorV1(datos DatosAltaFuenteAutoridadV1) (FuenteAutoridadVersionada, error) {
	return nuevaFuenteAutoridadBorradorVersionada(
		datos.ID, 1, ReferenciaLinajeFuenteAutoridad{}, datos.Contenido, datos.CreadaPor,
		datos.MotivoCreacionCodigo, datos.CreadaEn,
	)
}

func nuevaFuenteAutoridadBorradorVersionada(
	id string,
	version uint64,
	versionAnterior ReferenciaLinajeFuenteAutoridad,
	contenido ContenidoFuenteAutoridad,
	actorRef string,
	motivoCodigo CodigoMotivoFuenteAutoridad,
	registradaEn time.Time,
) (FuenteAutoridadVersionada, error) {
	registradaEn = normalizarInstanteFuenteAutoridad(registradaEn)
	contenido, err := normalizarContenidoFuenteAutoridad(contenido)
	if err != nil || !referenciaPersonaFuenteAutoridadValida(actorRef) || !motivoCodigo.Valido() ||
		!instanteFuenteAutoridadCanonico(registradaEn) || contenido.ConocidaEn.After(registradaEn) {
		return FuenteAutoridadVersionada{}, ErrFuenteAutoridadInvalida
	}
	contenido, huellaInicial, err := prepararHuellaContenidoFuenteAutoridad(
		id, version, versionAnterior, contenido,
	)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	huellaHistoriaInicial, err := huellaHistoriaInicialFuenteAutoridad(
		id, version, versionAnterior, huellaInicial, actorRef, motivoCodigo, registradaEn,
	)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	fuente := FuenteAutoridadVersionada{
		ID: id, Version: version, Revision: 1, VersionAnterior: versionAnterior,
		Contenido: contenido, HuellaContenidoInicialSHA256: huellaInicial,
		HuellaHistoriaInicialSHA256: huellaHistoriaInicial,
		Estado:                      EstadoFuenteAutoridadBorrador, CreadaPor: actorRef, CreadaEn: registradaEn,
		MotivoCreacionCodigo: motivoCodigo,
	}
	return fuente.ClonarCanonica()
}

func (f FuenteAutoridadVersionada) ReferenciaExacta() (ReferenciaFuenteAutoridad, error) {
	_, huella, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil {
		return ReferenciaFuenteAutoridad{}, err
	}
	referencia := ReferenciaFuenteAutoridad{FuenteID: f.ID, Version: f.Version, HuellaContenidoSHA256: huella}
	if err := referencia.Validar(); err != nil {
		return ReferenciaFuenteAutoridad{}, err
	}
	return referencia, nil
}

func (f FuenteAutoridadVersionada) ReferenciaLinajeExacta() (ReferenciaLinajeFuenteAutoridad, error) {
	canonica, huellaContenido, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil || canonica.Estado == EstadoFuenteAutoridadBorrador {
		return ReferenciaLinajeFuenteAutoridad{}, ErrReferenciaAutoridadInvalida
	}
	bytesEstado, err := serializarEstadoPersistibleFuenteAutoridadV1(canonica)
	if err != nil {
		return ReferenciaLinajeFuenteAutoridad{}, ErrReferenciaAutoridadInvalida
	}
	referencia := ReferenciaLinajeFuenteAutoridad{
		Fuente: ReferenciaFuenteAutoridad{
			FuenteID: canonica.ID, Version: canonica.Version, HuellaContenidoSHA256: huellaContenido,
		},
		Revision: canonica.Revision, Estado: canonica.Estado,
		HuellaHistoriaSHA256: canonica.huellaHistoriaActual(),
		HuellaEstadoSHA256:   huellaBytesFuenteAutoridad(bytesEstado),
	}
	if err := referencia.Validar(); err != nil {
		return ReferenciaLinajeFuenteAutoridad{}, err
	}
	return referencia, nil
}

// Citar solo expone preceptos que existen en una version ya publicada. Una
// suspension o derogacion no borra la cita historica; un borrador nunca es
// fuente citable.
func (f FuenteAutoridadVersionada) Citar(preceptos ...string) (CitaFuenteAutoridad, error) {
	canonico, huella, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil || canonico.Estado == EstadoFuenteAutoridadBorrador {
		return CitaFuenteAutoridad{}, ErrReferenciaAutoridadInvalida
	}
	disponibles := make(map[string]struct{}, len(canonico.Contenido.Preceptos))
	for _, precepto := range canonico.Contenido.Preceptos {
		disponibles[precepto.Clave] = struct{}{}
	}
	for _, precepto := range preceptos {
		if _, existe := disponibles[precepto]; !existe {
			return CitaFuenteAutoridad{}, ErrReferenciaAutoridadInvalida
		}
	}
	cita := CitaFuenteAutoridad{
		Fuente: ReferenciaFuenteAutoridad{
			FuenteID: canonico.ID, Version: canonico.Version, HuellaContenidoSHA256: huella,
		},
		Preceptos: append([]string(nil), preceptos...),
	}
	return cita.ClonarCanonica()
}

func (f FuenteAutoridadVersionada) Validar() error {
	_, _, err := f.prepararCanonicaSinEstadoPersistible()
	return err
}

func (f FuenteAutoridadVersionada) ClonarCanonica() (FuenteAutoridadVersionada, error) {
	clon, _, err := f.prepararCanonicaSinEstadoPersistible()
	return clon, err
}

func (f FuenteAutoridadVersionada) HuellaContenidoSHA256() (string, error) {
	_, huella, err := f.prepararCanonicaSinEstadoPersistible()
	return huella, err
}

func (f FuenteAutoridadVersionada) HuellaEstadoSHA256() (string, error) {
	_, _, estadoCanonico, err := f.prepararCanonica()
	if err != nil {
		return "", err
	}
	return huellaBytesFuenteAutoridad(estadoCanonico), nil
}

// MarshalJSON impide persistir por accidente la estructura viva. Todo JSON
// del agregado cruza el contrato congelado EstadoPersistibleV1.
func (f FuenteAutoridadVersionada) MarshalJSON() ([]byte, error) {
	return f.EstadoPersistibleV1()
}

// UnmarshalJSON impide que un adaptador eluda por accidente la rehidratacion
// estricta V1 mediante encoding/json. Solo se acepta el estado canonico exacto.
func (f *FuenteAutoridadVersionada) UnmarshalJSON(datos []byte) error {
	if f == nil {
		return ErrEstadoPersistibleFuenteAutoridadInvalido
	}
	rehidratada, err := RehidratarFuenteAutoridadV1(datos)
	if err != nil {
		return err
	}
	*f = rehidratada
	return nil
}

func (f FuenteAutoridadVersionada) prepararCanonica() (FuenteAutoridadVersionada, string, []byte, error) {
	return f.prepararCanonicaSegun(true)
}

func (f FuenteAutoridadVersionada) prepararCanonicaSinEstadoPersistible() (
	FuenteAutoridadVersionada,
	string,
	error,
) {
	canonica, huellaContenido, _, err := f.prepararCanonicaSegun(false)
	return canonica, huellaContenido, err
}

func (f FuenteAutoridadVersionada) prepararCanonicaSegun(
	incluirEstadoPersistible bool,
) (FuenteAutoridadVersionada, string, []byte, error) {
	contenido, huellaContenido, err := prepararHuellaContenidoFuenteAutoridad(
		f.ID, f.Version, f.VersionAnterior, f.Contenido,
	)
	huellaHistoriaInicial, errHistoria := huellaHistoriaInicialFuenteAutoridad(
		f.ID, f.Version, f.VersionAnterior, f.HuellaContenidoInicialSHA256,
		f.CreadaPor, f.MotivoCreacionCodigo, f.CreadaEn,
	)
	if err != nil || f.Revision == 0 || !f.Estado.Valido() ||
		!esSHA256Autoridad(f.HuellaContenidoInicialSHA256) ||
		errHistoria != nil || f.HuellaHistoriaInicialSHA256 != huellaHistoriaInicial ||
		!referenciaPersonaFuenteAutoridadValida(f.CreadaPor) ||
		!instanteFuenteAutoridadCanonico(f.CreadaEn) || f.Contenido.ConocidaEn.After(f.CreadaEn) ||
		!f.MotivoCreacionCodigo.Valido() || len(f.EdicionesBorrador) > maximoEdicionesFuenteAutoridad ||
		len(f.Transiciones) > maximoTransicionesFuenteAutoridad ||
		f.Revision != uint64(1+len(f.EdicionesBorrador)+len(f.Transiciones)) {
		return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
	}

	clon := f
	clon.Contenido = contenido
	clon.EdicionesBorrador = append([]EdicionBorradorFuenteAutoridad(nil), f.EdicionesBorrador...)
	clon.Transiciones = make([]TransicionFuenteAutoridad, len(f.Transiciones))

	huellaEncadenada := f.HuellaContenidoInicialSHA256
	huellaHistoria := f.HuellaHistoriaInicialSHA256
	ultimoRegistro := f.CreadaEn
	editores := make(map[string]struct{}, len(f.EdicionesBorrador))
	for indice, edicion := range f.EdicionesBorrador {
		revisionAnterior := uint64(indice + 1)
		if edicion.RevisionAnterior != revisionAnterior || edicion.RevisionNueva != revisionAnterior+1 ||
			!referenciaPersonaFuenteAutoridadValida(edicion.ActorRef) || !edicion.MotivoCodigo.Valido() ||
			!instanteFuenteAutoridadCanonico(edicion.RegistradaEn) || !edicion.RegistradaEn.After(ultimoRegistro) ||
			!esSHA256Autoridad(edicion.HuellaContenidoAnteriorSHA256) ||
			!esSHA256Autoridad(edicion.HuellaContenidoNuevaSHA256) ||
			!esSHA256Autoridad(edicion.HuellaHistoriaAnteriorSHA256) ||
			!esSHA256Autoridad(edicion.HuellaHistoriaNuevaSHA256) ||
			edicion.HuellaContenidoAnteriorSHA256 != huellaEncadenada ||
			edicion.HuellaHistoriaAnteriorSHA256 != huellaHistoria {
			return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
		}
		huellaHistoriaNueva, err := huellaHistoriaEdicionBorradorFuenteAutoridad(edicion)
		if err != nil || edicion.HuellaHistoriaNuevaSHA256 != huellaHistoriaNueva {
			return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
		}
		huellaEncadenada = edicion.HuellaContenidoNuevaSHA256
		huellaHistoria = huellaHistoriaNueva
		ultimoRegistro = edicion.RegistradaEn
		editores[edicion.ActorRef] = struct{}{}
	}
	if huellaEncadenada != huellaContenido {
		return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
	}

	if (f.Estado == EstadoFuenteAutoridadBorrador) != (len(f.Transiciones) == 0) {
		return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
	}
	estado := EstadoFuenteAutoridadBorrador
	referenciasSolicitud := make(map[string]struct{}, len(f.Transiciones))
	referenciasEvidencia := make(map[string]struct{}, len(f.Transiciones))
	referenciasAtestacion := make(map[string]struct{}, len(f.Transiciones))
	for indice, transicion := range f.Transiciones {
		secuencia := uint64(indice + 1)
		if transicion.Secuencia != secuencia || transicion.EstadoAnterior != estado ||
			!transicionPermitidaFuenteAutoridad(transicion.EstadoAnterior, transicion.EstadoNuevo) ||
			!referenciaPersonaFuenteAutoridadValida(transicion.ActorRef) || !transicion.MotivoCodigo.Valido() ||
			!referenciaFuenteAutoridadValida(transicion.SolicitudRef) ||
			!instanteFuenteAutoridadCanonico(transicion.PreparadaEn) || !transicion.PreparadaEn.After(ultimoRegistro) ||
			!instanteFuenteAutoridadCanonico(transicion.ExpiraEn) || !transicion.ExpiraEn.After(transicion.PreparadaEn) ||
			!instanteFuenteAutoridadCanonico(transicion.RegistradaEn) || !transicion.RegistradaEn.After(ultimoRegistro) ||
			transicion.RegistradaEn.Before(transicion.Evidencia.ComprobadaEn) ||
			!transicion.RegistradaEn.Before(transicion.ExpiraEn) ||
			!esSHA256Autoridad(transicion.HuellaHistoriaAnteriorSHA256) ||
			!esSHA256Autoridad(transicion.HuellaHistoriaNuevaSHA256) ||
			transicion.HuellaHistoriaAnteriorSHA256 != huellaHistoria {
			return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
		}
		if indice == 0 {
			if transicion.ActorRef == f.CreadaPor {
				return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
			}
			if _, edito := editores[transicion.ActorRef]; edito {
				return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
			}
		}
		compromiso, err := construirCompromisoTransicionFuenteAutoridad(
			f.ID, f.Version, huellaContenido,
			uint64(1+len(f.EdicionesBorrador)+indice), secuencia,
			transicion.EstadoAnterior, transicion.EstadoNuevo, transicion.ActorRef,
			transicion.MotivoCodigo, huellaHistoria, transicion.SolicitudRef,
			transicion.PreparadaEn, transicion.ExpiraEn,
		)
		if err != nil || validarEvidenciaTransicionFuenteAutoridad(transicion.Evidencia, compromiso) != nil ||
			!registrarReferenciaFuenteAutoridadUnica(referenciasSolicitud, transicion.SolicitudRef) ||
			!registrarReferenciasEvidenciaFuenteAutoridad(
				transicion.Evidencia, referenciasEvidencia, referenciasAtestacion,
			) {
			return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
		}
		huellaHistoriaNueva, err := huellaHistoriaTransicionFuenteAutoridad(transicion, compromiso)
		if err != nil || transicion.HuellaHistoriaNuevaSHA256 != huellaHistoriaNueva {
			return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
		}
		evidencia, err := transicion.Evidencia.ClonarCanonica()
		if err != nil {
			return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
		}
		clon.Transiciones[indice] = transicion
		clon.Transiciones[indice].Evidencia = evidencia
		estado = transicion.EstadoNuevo
		huellaHistoria = huellaHistoriaNueva
		ultimoRegistro = transicion.RegistradaEn
	}
	if estado != f.Estado {
		return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
	}

	if !incluirEstadoPersistible {
		return clon, huellaContenido, nil, nil
	}
	estadoCanonico, err := serializarEstadoPersistibleFuenteAutoridadV1(clon)
	if err != nil {
		return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
	}
	return clon, huellaContenido, estadoCanonico, nil
}

func registrarReferenciasEvidenciaFuenteAutoridad(
	evidencia EvidenciaActoFuenteAutoridad,
	evidencias, atestaciones map[string]struct{},
) bool {
	if !registrarReferenciaFuenteAutoridadUnica(evidencias, evidencia.EvidenciaRef) ||
		!registrarReferenciaFuenteAutoridadUnica(atestaciones, evidencia.AtestacionRef) {
		return false
	}
	return true
}

func registrarReferenciaFuenteAutoridadUnica(vistas map[string]struct{}, referencia string) bool {
	if _, repetida := vistas[referencia]; repetida {
		return false
	}
	vistas[referencia] = struct{}{}
	return true
}
