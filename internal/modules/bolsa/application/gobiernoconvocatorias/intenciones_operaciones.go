package gobiernoconvocatorias

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	esquemaIntencionBorradorV2         = "bolsa.convocatoria.borrador.intencion.v2"
	dominioIdempotenciaBorradorV2      = "bolsa.convocatoria.borrador.idempotencia.v2"
	separadorLocalizadorBorradorV2     = "bolsa.convocatoria.borrador.localizador.v2"
	separadorHuellaSolicitudBorradorV2 = "bolsa.convocatoria.borrador.huella.v2"
	maximoIdentidadesRotacionBorrador  = 4
)

var (
	ErrIntencionBorradorInvalida = errors.New("gobierno convocatorias: intencion semantica de borrador invalida")
	ErrOrdenBorradorInvalida     = errors.New("gobierno convocatorias: orden de borrador invalida")
)

// SelectorPlantillaBorrador identifica una version publicada. El resolvedor
// confiable incorpora su huella antes de derivar F.
type SelectorPlantillaBorrador struct {
	ID                    string
	Version               int
	HuellaContenidoSHA256 string
}

type PlantillaBorradorResuelta struct {
	bloqueoSerializacionDiario
	Referencia    dominiobolsa.ReferenciaConfiguracionConvocatoria
	Configuracion dominiobolsa.ConfiguracionFijadaConvocatoria
}

// PreparacionAltaBorrador se obtiene solo cuando la consulta idempotente no ha
// encontrado un resultado recuperable. ID, flujo y ambito no forman parte de F.
type PreparacionAltaBorrador struct {
	bloqueoSerializacionDiario
	Plantilla          PlantillaBorradorResuelta
	ID                 string
	InstanciaFlujoRef  string
	AmbitoOrganizativo dominiobolsa.AmbitoOrganizativoConvocatoria
}

type OrdenCrearBorrador struct {
	bloqueoSerializacionDiario
	ClaveCliente              ClaveClienteIdempotenciaConvocatoria
	Actor                     dominiovec.ContextoActor
	VinculoAutenticacionActor dominiovec.VinculoAutenticacionActorV1
	Plantilla                 SelectorPlantillaBorrador
	CodigoVersionPublica      string
	Contenido                 dominiobolsa.ContenidoPublicableConvocatoria
	ExpedienteRef             string
	MotivoCatalogo            dominiovec.ReferenciaEntradaCatalogo
	CorrelacionRef            string
}

type OrdenActualizarBorrador struct {
	bloqueoSerializacionDiario
	ClaveCliente              ClaveClienteIdempotenciaConvocatoria
	Actor                     dominiovec.ContextoActor
	VinculoAutenticacionActor dominiovec.VinculoAutenticacionActorV1
	Esperada                  puertosbolsa.ReferenciaEstadoVersionConvocatoria
	Contenido                 dominiobolsa.ContenidoPublicableConvocatoria
	MotivoCatalogo            dominiovec.ReferenciaEntradaCatalogo
	CorrelacionRef            string
}

type datosIntencionBorrador struct {
	Esquema              string
	Accion               string
	Plantilla            *dominiobolsa.ReferenciaConfiguracionConvocatoria
	CodigoVersionPublica string
	ExpedienteRef        string
	Esperada             *puertosbolsa.ReferenciaEstadoVersionConvocatoria
	Contenido            dominiobolsa.ContenidoPublicableConvocatoria
	MotivoCatalogo       dominiovec.ReferenciaEntradaCatalogo
	representacion       []byte
}

// IntencionBorradorCanonica es la solicitud estable autenticada por F. Excluye
// reloj, correlacion, IDs generados, agregado resultante, PDP y HMAC del motivo.
type IntencionBorradorCanonica struct {
	bloqueoSerializacionDiario
	datos *datosIntencionBorrador
}

func nuevaIntencionAltaBorradorCanonica(
	plantilla dominiobolsa.ReferenciaConfiguracionConvocatoria,
	codigoVersionPublica, expedienteRef string,
	contenido dominiobolsa.ContenidoPublicableConvocatoria,
	motivo dominiovec.ReferenciaEntradaCatalogo,
) (IntencionBorradorCanonica, error) {
	return nuevaIntencionBorradorCanonica(datosIntencionBorrador{
		Esquema:   esquemaIntencionBorradorV2,
		Accion:    puertosbolsa.AccionCrearBorradorConvocatoria,
		Plantilla: &plantilla, CodigoVersionPublica: codigoVersionPublica,
		ExpedienteRef: expedienteRef, Contenido: contenido, MotivoCatalogo: motivo,
	})
}

func nuevaIntencionActualizacionBorradorCanonica(
	esperada puertosbolsa.ReferenciaEstadoVersionConvocatoria,
	contenido dominiobolsa.ContenidoPublicableConvocatoria,
	motivo dominiovec.ReferenciaEntradaCatalogo,
) (IntencionBorradorCanonica, error) {
	return nuevaIntencionBorradorCanonica(datosIntencionBorrador{
		Esquema:  esquemaIntencionBorradorV2,
		Accion:   puertosbolsa.AccionActualizarBorradorConvocatoria,
		Esperada: &esperada, Contenido: contenido, MotivoCatalogo: motivo,
	})
}

func nuevaIntencionBorradorCanonica(datos datosIntencionBorrador) (IntencionBorradorCanonica, error) {
	contenido, err := datos.Contenido.ClonarCanonico()
	if err != nil || datos.Esquema != esquemaIntencionBorradorV2 || datos.MotivoCatalogo.Validar() != nil {
		return IntencionBorradorCanonica{}, ErrIntencionBorradorInvalida
	}
	datos.Contenido = contenido
	alta := datos.Accion == puertosbolsa.AccionCrearBorradorConvocatoria
	actualizacion := datos.Accion == puertosbolsa.AccionActualizarBorradorConvocatoria
	if !alta && !actualizacion ||
		alta && (datos.Plantilla == nil || datos.Plantilla.Validar() != nil ||
			datos.CodigoVersionPublica == "" || !referenciaProyeccionValida(datos.ExpedienteRef) || datos.Esperada != nil) ||
		actualizacion && (datos.Esperada == nil || datos.Esperada.Validar() != nil || datos.Plantilla != nil ||
			datos.CodigoVersionPublica != "" || datos.ExpedienteRef != "") {
		return IntencionBorradorCanonica{}, ErrIntencionBorradorInvalida
	}
	representacion, err := representarIntencionBorrador(datos)
	if err != nil {
		return IntencionBorradorCanonica{}, ErrIntencionBorradorInvalida
	}
	datos.representacion = representacion
	return IntencionBorradorCanonica{datos: &datos}, nil
}

func representarIntencionBorrador(datos datosIntencionBorrador) ([]byte, error) {
	return json.Marshal(struct {
		Esquema              string                                            `json:"esquema"`
		Accion               string                                            `json:"accion"`
		Plantilla            *dominiobolsa.ReferenciaConfiguracionConvocatoria `json:"plantilla,omitempty"`
		CodigoVersionPublica string                                            `json:"codigo_version_publica,omitempty"`
		ExpedienteRef        string                                            `json:"expediente_ref,omitempty"`
		Esperada             *puertosbolsa.ReferenciaEstadoVersionConvocatoria `json:"esperada,omitempty"`
		Contenido            dominiobolsa.ContenidoPublicableConvocatoria      `json:"contenido"`
		MotivoCatalogo       dominiovec.ReferenciaEntradaCatalogo              `json:"motivo_catalogo"`
	}{datos.Esquema, datos.Accion, datos.Plantilla, datos.CodigoVersionPublica,
		datos.ExpedienteRef, datos.Esperada, datos.Contenido, datos.MotivoCatalogo})
}

func (i IntencionBorradorCanonica) valida() bool {
	if i.datos == nil || i.datos.Esquema != esquemaIntencionBorradorV2 || len(i.datos.representacion) == 0 {
		return false
	}
	regenerada, err := representarIntencionBorrador(*i.datos)
	return err == nil && bytes.Equal(regenerada, i.datos.representacion)
}

func (i IntencionBorradorCanonica) accion() string {
	if !i.valida() {
		return ""
	}
	return i.datos.Accion
}

func (i IntencionBorradorCanonica) coincideEjecucion(
	version dominiobolsa.VersionConvocatoriaGobernada,
	material puertosbolsa.MaterialIntencionGobiernoConvocatoria,
	plantilla *PlantillaBorradorResuelta,
) bool {
	if !i.valida() || version.Validar() != nil || material.Validar() != nil ||
		material.Accion != i.datos.Accion || material.EstadoPrincipalNuevo.Referencia != version.Referencia() {
		return false
	}
	contenido, err := version.Contenido.ClonarCanonico()
	if err != nil || !reflect.DeepEqual(contenido, i.datos.Contenido) {
		return false
	}
	switch i.datos.Accion {
	case puertosbolsa.AccionCrearBorradorConvocatoria:
		return plantilla != nil && i.datos.Plantilla != nil &&
			plantilla.Referencia == *i.datos.Plantilla &&
			version.Configuracion.Plantilla == plantilla.Referencia &&
			reflect.DeepEqual(plantilla.Configuracion, version.Configuracion) &&
			version.CodigoVersionPublica == i.datos.CodigoVersionPublica &&
			version.ExpedienteRef == i.datos.ExpedienteRef &&
			version.MotivoCreacion == i.datos.MotivoCatalogo.Referencia() &&
			material.EstadoPrincipalEsperado == nil && material.EstadoRelacionadoEsperado == nil &&
			material.EstadoRelacionadoNuevo == nil
	case puertosbolsa.AccionActualizarBorradorConvocatoria:
		return plantilla == nil && i.datos.Esperada != nil &&
			material.EstadoPrincipalEsperado != nil && *material.EstadoPrincipalEsperado == *i.datos.Esperada &&
			version.MotivoModificacion == i.datos.MotivoCatalogo.Referencia() &&
			material.EstadoRelacionadoEsperado == nil && material.EstadoRelacionadoNuevo == nil
	default:
		return false
	}
}

type ambitoIdempotenciaBorrador struct {
	personaRef, perfilActivoRef, accion string
}

type SolicitudDerivacionIdempotencia struct {
	bloqueoSerializacionDiario
	clave     ClaveClienteIdempotenciaConvocatoria
	intencion IntencionBorradorCanonica
	ambito    ambitoIdempotenciaBorrador
}

func nuevaSolicitudDerivacionIdempotencia(
	clave ClaveClienteIdempotenciaConvocatoria,
	intencion IntencionBorradorCanonica,
	actor dominiovec.ContextoActor,
) (SolicitudDerivacionIdempotencia, error) {
	ambito := ambitoIdempotenciaBorrador{
		personaRef: actor.PersonaRef, perfilActivoRef: actor.PerfilActivoRef, accion: intencion.accion(),
	}
	if !clave.Valida() || !intencion.valida() ||
		ambito.personaRef == "" || ambito.perfilActivoRef == "" || ambito.accion == "" {
		return SolicitudDerivacionIdempotencia{}, ErrIntencionBorradorInvalida
	}
	if _, err := actor.Clonar(); err != nil {
		return SolicitudDerivacionIdempotencia{}, ErrIntencionBorradorInvalida
	}
	return SolicitudDerivacionIdempotencia{clave: clave, intencion: intencion, ambito: ambito}, nil
}

// MaterialParaConectorConfiable expone sólo copias efimeras al derivador HMAC.
// El conector calcula L y F con separadores distintos sobre persona+perfil+
// dominio+accion. Nunca registra ni persiste estas preimagenes en claro.
func (s SolicitudDerivacionIdempotencia) MaterialParaConectorConfiable() (
	[]byte,
	[]byte,
	error,
) {
	if !s.clave.Valida() || !s.intencion.valida() {
		return nil, nil, ErrIntencionBorradorInvalida
	}
	ambito := representarAmbitoIdempotencia(s.ambito)
	if len(ambito) == 0 {
		return nil, nil, ErrIntencionBorradorInvalida
	}
	return representarPreimagenIdempotencia(
			separadorLocalizadorBorradorV2, ambito, s.clave.valor[:],
		), representarPreimagenIdempotencia(
			separadorHuellaSolicitudBorradorV2, ambito, s.intencion.datos.representacion,
		), nil
}

func representarPreimagenIdempotencia(separador string, ambito, carga []byte) []byte {
	partes := [][]byte{[]byte(separador), ambito, carga}
	total := 0
	for _, parte := range partes {
		total += 4 + len(parte)
	}
	resultado := make([]byte, 0, total)
	for _, parte := range partes {
		var longitud [4]byte
		binary.BigEndian.PutUint32(longitud[:], uint32(len(parte)))
		resultado = append(resultado, longitud[:]...)
		resultado = append(resultado, parte...)
	}
	return resultado
}

func representarAmbitoIdempotencia(a ambitoIdempotenciaBorrador) []byte {
	if a.personaRef == "" || a.perfilActivoRef == "" || a.accion == "" {
		return nil
	}
	partes := []string{dominioIdempotenciaBorradorV2, a.accion, a.personaRef, a.perfilActivoRef}
	total := 0
	for _, parte := range partes {
		total += 4 + len(parte)
	}
	resultado := make([]byte, 0, total)
	for _, parte := range partes {
		var longitud [4]byte
		binary.BigEndian.PutUint32(longitud[:], uint32(len(parte)))
		resultado = append(resultado, longitud[:]...)
		resultado = append(resultado, parte...)
	}
	return resultado
}

func instanteOperacionCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC && instante.Nanosecond()%1_000 == 0
}
