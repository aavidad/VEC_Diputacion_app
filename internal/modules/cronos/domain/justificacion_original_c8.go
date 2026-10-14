package domain

import (
	"math"
	"regexp"
)

// OriginalJustificacionC8V2 es la proyección atestada del original admitido
// por Documentos. Su recurso propietario es la solicitud de permiso Cronos;
// ninguna referencia externa hace las veces de expediente o custodia.
type OriginalJustificacionC8V2 struct {
	DocumentoID         string
	DocumentoVersion    uint64
	HuellaSHA256Bytes   string
	ReciboDocumentoRef  string
	RecursoRef          string
	PersonaTitularRef   string
	PersonaAportanteRef string
	CustodioID          string
	ModuloID            string
	TipoRecurso         string
	Finalidad           string
	EstadoAdmision      string
}

var reciboDocumentalC8 = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]{2,159}$`)

func (o OriginalJustificacionC8V2) ValidarPara(solicitudRef, personaRef string) error {
	if !SolicitudPermisoRefValida(solicitudRef) || !referenciaIdentidadMarcaje(personaRef, "per_") ||
		!RefDocumentoJustificacionValida(o.DocumentoID) || o.DocumentoVersion == 0 ||
		!HuellaEfectosValida(o.HuellaSHA256Bytes) || !reciboDocumentalC8.MatchString(o.ReciboDocumentoRef) ||
		o.RecursoRef != solicitudRef || o.PersonaTitularRef != personaRef || o.PersonaAportanteRef != personaRef ||
		!tecnicoJustificacion.MatchString(o.CustodioID) || o.ModuloID != "cronos" ||
		o.TipoRecurso != "justificacion" || o.Finalidad != "custodiar_justificacion_cronos" ||
		o.EstadoAdmision != "admitido" {
		return ErrJustificacionInvalida
	}
	return nil
}

// JustificacionOriginalC8V2 conserva la referencia exacta del original y el
// estado administrativo sin modificar el permiso ni sus recibos previos.
type JustificacionOriginalC8V2 struct {
	SolicitudRef string
	EmpleadoRef  string
	Original     OriginalJustificacionC8V2
	Version      int64
	Estado       EstadoJustificacion
	MotivoRef    string
}

func PrepararAnexoOriginalC8V2(solicitudRef, empleadoRef, personaRef string, original OriginalJustificacionC8V2,
	actual *JustificacionOriginalC8V2, versionEsperada int64) (JustificacionOriginalC8V2, error) {
	if original.ValidarPara(solicitudRef, personaRef) != nil || !referenciaIdentidadMarcaje(empleadoRef, "emp_") ||
		versionEsperada < 0 || versionEsperada == math.MaxInt64 {
		return JustificacionOriginalC8V2{}, ErrJustificacionInvalida
	}
	if actual == nil {
		if versionEsperada != 0 {
			return JustificacionOriginalC8V2{}, ErrJustificacionConflicto
		}
	} else if actual.SolicitudRef != solicitudRef || actual.EmpleadoRef != empleadoRef ||
		actual.Version != versionEsperada || (actual.Original.DocumentoID == original.DocumentoID &&
		actual.Original.DocumentoVersion == original.DocumentoVersion) {
		return JustificacionOriginalC8V2{}, ErrJustificacionConflicto
	}
	return JustificacionOriginalC8V2{SolicitudRef: solicitudRef, EmpleadoRef: empleadoRef,
		Original: original, Version: versionEsperada + 1, Estado: JustificacionPendiente}, nil
}

func PrepararRevisionOriginalC8V2(actual JustificacionOriginalC8V2, versionEsperada int64,
	decision EstadoJustificacion, motivoRef string) (JustificacionOriginalC8V2, error) {
	if !SolicitudPermisoRefValida(actual.SolicitudRef) || !referenciaIdentidadMarcaje(actual.EmpleadoRef, "emp_") ||
		actual.Original.ValidarPara(actual.SolicitudRef, actual.Original.PersonaTitularRef) != nil ||
		versionEsperada < 1 || versionEsperada == math.MaxInt64 || !referenciaMarcaje(motivoRef) ||
		(decision != JustificacionAceptada && decision != JustificacionRechazada) {
		return JustificacionOriginalC8V2{}, ErrJustificacionInvalida
	}
	if actual.Version != versionEsperada || actual.Estado != JustificacionPendiente || actual.MotivoRef != "" {
		return JustificacionOriginalC8V2{}, ErrJustificacionConflicto
	}
	actual.Version++
	actual.Estado = decision
	actual.MotivoRef = motivoRef
	return actual, nil
}
