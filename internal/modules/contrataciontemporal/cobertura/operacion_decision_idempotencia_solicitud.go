package cobertura

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// SolicitudConsultarOperacionDecisionCoberturaConfirmada encapsula la
// identidad funcional y sus generaciones HMAC. Para buscar, el adaptador solo
// necesita AmbitosIdempotencia; el cotejo del recibo se repite en este tipo.
type SolicitudConsultarOperacionDecisionCoberturaConfirmada struct {
	bloqueoSerializacionOperacionDecisionCobertura
	identidad *DatosIdentidadOperacionDecisionCobertura
	sellos    SellosOperacionDecisionCobertura
}

func NuevaSolicitudConsultarOperacionDecisionCoberturaConfirmada(
	identidad DatosIdentidadOperacionDecisionCobertura,
	sellos SellosOperacionDecisionCobertura,
) (SolicitudConsultarOperacionDecisionCoberturaConfirmada, error) {
	solicitud := SolicitudConsultarOperacionDecisionCoberturaConfirmada{
		identidad: &identidad,
		sellos:    sellos,
	}
	if solicitud.Validar() != nil {
		return SolicitudConsultarOperacionDecisionCoberturaConfirmada{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return solicitud, nil
}

func (s SolicitudConsultarOperacionDecisionCoberturaConfirmada) Validar() error {
	if s.identidad == nil || s.identidad.Validar() != nil ||
		s.sellos.Validar() != nil {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return nil
}

func (s SolicitudConsultarOperacionDecisionCoberturaConfirmada) identidadInterna() (
	DatosIdentidadOperacionDecisionCobertura,
	error,
) {
	if s.Validar() != nil {
		return DatosIdentidadOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return *s.identidad, nil
}

func (s SolicitudConsultarOperacionDecisionCoberturaConfirmada) coordenadas() (
	string,
	string,
	uint64,
	error,
) {
	identidad, err := s.identidadInterna()
	if err != nil {
		return "", "", 0, err
	}
	return identidad.organizacionRef, identidad.expedienteRef,
		identidad.versionExpediente, nil
}

// AmbitosIdempotencia devuelve una copia de los sellos de búsqueda. No revela
// semántica, clave de reintento, actor, perfil ni token propietario.
func (s SolicitudConsultarOperacionDecisionCoberturaConfirmada) AmbitosIdempotencia() (
	ports.ColeccionSellosHMAC,
	error,
) {
	if s.Validar() != nil {
		return ports.ColeccionSellosHMAC{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	datos, err := s.sellos.AmbitosIdempotenciaHMAC.Datos()
	if err != nil {
		return ports.ColeccionSellosHMAC{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	retenidos := make([]string, 0, len(datos.Retenidos))
	for _, retenido := range datos.Retenidos {
		retenidos = append(retenidos, retenido.Valor)
	}
	return ports.NuevaColeccionSellosHMAC(datos.Activo.Valor, retenidos)
}

func (s SolicitudConsultarOperacionDecisionCoberturaConfirmada) contienePar(
	ambito string,
	semantica string,
) bool {
	return s.Validar() == nil && s.sellos.contienePar(ambito, semantica)
}

func (s SolicitudConsultarOperacionDecisionCoberturaConfirmada) parActivo() (
	string,
	string,
	error,
) {
	if s.Validar() != nil {
		return "", "", ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return s.sellos.parActivo()
}

// DatosSolicitudReservarOperacionDecisionCobertura es la vista defensiva para
// persistencia. Solo incluye la huella del token; el secreto nunca cruza esta
// frontera ni debe guardarse.
type DatosSolicitudReservarOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	OrganizacionRef        string
	ExpedienteRef          string
	VersionExpediente      uint64
	AmbitoIdempotenciaHMAC string
	HuellaSemanticaHMAC    string
	TokenPropietarioSHA256 string
}

// SolicitudReservarOperacionDecisionCobertura liga la petición exacta con un
// token CSPRNG. Reapropiar exige un token nuevo y un cercado superior; el lease
// por sí solo nunca demuestra propiedad.
type SolicitudReservarOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	consulta SolicitudConsultarOperacionDecisionCoberturaConfirmada
	token    TokenPropietarioOperacionDecisionCobertura
}

func NuevaSolicitudReservarOperacionDecisionCobertura(
	consulta SolicitudConsultarOperacionDecisionCoberturaConfirmada,
	token TokenPropietarioOperacionDecisionCobertura,
) (SolicitudReservarOperacionDecisionCobertura, error) {
	solicitud := SolicitudReservarOperacionDecisionCobertura{
		consulta: consulta,
		token:    token,
	}
	if solicitud.Validar() != nil {
		return SolicitudReservarOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return solicitud, nil
}

func (s SolicitudReservarOperacionDecisionCobertura) Validar() error {
	if s.consulta.Validar() != nil {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	if _, err := s.token.HuellaSHA256(); err != nil {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return nil
}

func (s SolicitudReservarOperacionDecisionCobertura) Datos() (
	DatosSolicitudReservarOperacionDecisionCobertura,
	error,
) {
	if s.Validar() != nil {
		return DatosSolicitudReservarOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	organizacionRef, expedienteRef, versionExpediente, err :=
		s.consulta.coordenadas()
	ambito, semantica, errSellos := s.consulta.parActivo()
	huellaToken, errToken := s.token.HuellaSHA256()
	if err != nil || errSellos != nil || errToken != nil {
		return DatosSolicitudReservarOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return DatosSolicitudReservarOperacionDecisionCobertura{
		OrganizacionRef: organizacionRef, ExpedienteRef: expedienteRef,
		VersionExpediente:      versionExpediente,
		AmbitoIdempotenciaHMAC: ambito,
		HuellaSemanticaHMAC:    semantica,
		TokenPropietarioSHA256: huellaToken,
	}, nil
}

func (s SolicitudReservarOperacionDecisionCobertura) AmbitosIdempotencia() (
	ports.ColeccionSellosHMAC,
	error,
) {
	if s.Validar() != nil {
		return ports.ColeccionSellosHMAC{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return s.consulta.AmbitosIdempotencia()
}

// ConsultaConfirmada devuelve la solicitud opaca ligada a esta reserva para
// reconstruir un replay terminal que aparezca durante la carrera de reserva.
// No expone el token propietario ni añade una vista de datos nueva.
func (s SolicitudReservarOperacionDecisionCobertura) ConsultaConfirmada() (
	SolicitudConsultarOperacionDecisionCoberturaConfirmada,
	error,
) {
	return s.consultaConfirmada()
}

// CoincideParPersistido permite al adaptador distinguir replay, colisión y
// rotación sin recibir la colección de huellas semánticas. La comparación
// subyacente es constante y exige la misma generación en ambos sellos.
func (s SolicitudReservarOperacionDecisionCobertura) CoincideParPersistido(
	ambitoHMAC string,
	semanticaHMAC string,
) bool {
	return s.Validar() == nil &&
		s.consulta.contienePar(ambitoHMAC, semanticaHMAC)
}

func (s SolicitudReservarOperacionDecisionCobertura) consultaConfirmada() (
	SolicitudConsultarOperacionDecisionCoberturaConfirmada,
	error,
) {
	if s.Validar() != nil {
		return SolicitudConsultarOperacionDecisionCoberturaConfirmada{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return s.consulta, nil
}

func (s SolicitudReservarOperacionDecisionCobertura) tokenCoincide(
	huella string,
) bool {
	return s.Validar() == nil && s.token.CoincideConHuellaSHA256(huella)
}
