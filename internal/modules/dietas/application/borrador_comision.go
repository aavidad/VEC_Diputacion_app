package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrComposicionBorradorInvalida = errors.New("dietas: composicion de borrador invalida")
var referenciaComision = regexp.MustCompile(`^dco_[A-Za-z0-9_-]{22,128}$`)

const (
	AccionCrearBorradorPropio        = "dietas.borrador.propio.crear"
	AccionConsultarBorradorPropio    = "dietas.borrador.propio.consultar"
	FinalidadCrearBorradorPropio     = "crear_borrador_propio"
	FinalidadConsultarBorradorPropio = "consultar_borrador_propio"
	RecursoMisBorradores             = "dietas:borradores:propios"
)

type CasoUsoBorradorComision interface {
	CrearPropio(context.Context, dietasports.IdentidadEfectivaBorrador, dietasports.SolicitudCrearBorradorPropio) (dietasports.ResultadoBorradorComision, error)
	ObtenerPropio(context.Context, dietasports.IdentidadEfectivaBorrador, string) (dietasports.ResultadoBorradorComision, error)
	ListarPropios(context.Context, dietasports.IdentidadEfectivaBorrador, dietasports.ConsultaBorradoresPropios) (dietasports.PaginaBorradoresPropios, error)
}

type ServicioBorradorComision struct {
	repositorio dietasports.RepositorioBorradorComision
}

func NuevoServicioBorradorComision(repositorio dietasports.RepositorioBorradorComision) (*ServicioBorradorComision, error) {
	if interfazNula(repositorio) {
		return nil, ErrComposicionBorradorInvalida
	}
	return &ServicioBorradorComision{repositorio: repositorio}, nil
}

func (s *ServicioBorradorComision) CrearPropio(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, solicitud dietasports.SolicitudCrearBorradorPropio) (dietasports.ResultadoBorradorComision, error) {
	if ctx == nil || !servicioValido(s) {
		return dietasports.ResultadoBorradorComision{}, domain.ErrComisionBorradorInvalida
	}
	if err := ctx.Err(); err != nil {
		return dietasports.ResultadoBorradorComision{}, err
	}
	if err := identidadValida(identidad, AccionCrearBorradorPropio, RecursoMisBorradores, FinalidadCrearBorradorPropio); err != nil {
		return dietasports.ResultadoBorradorComision{}, err
	}
	solicitud, err := normalizarSolicitudCrear(solicitud)
	if err != nil {
		return dietasports.ResultadoBorradorComision{}, err
	}
	if solicitud.RelacionRef != "" && solicitud.RelacionRef != identidad.Relacion.RelacionRef {
		return dietasports.ResultadoBorradorComision{}, dietasports.ErrRelacionNoValida
	}
	if identidad.Autorizacion.Revalidacion.FechaReferencia != solicitud.FechaInicio {
		return dietasports.ResultadoBorradorComision{}, dietasports.ErrAccesoBorradorDenegado
	}
	return s.repositorio.CrearORecuperar(ctx, identidad, solicitud)
}

func (s *ServicioBorradorComision) RecuperarPorClave(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, solicitud dietasports.SolicitudCrearBorradorPropio) (dietasports.ResultadoBorradorComision, bool, error) {
	var cero dietasports.ResultadoBorradorComision
	if ctx == nil || !servicioValido(s) || solicitud.Calculo != nil {
		return cero, false, domain.ErrComisionBorradorInvalida
	}
	if err := identidadValida(identidad, AccionCrearBorradorPropio, RecursoMisBorradores, FinalidadCrearBorradorPropio); err != nil {
		return cero, false, err
	}
	normalizada, err := normalizarSolicitudCrear(solicitud)
	if err != nil || (normalizada.RelacionRef != "" && normalizada.RelacionRef != identidad.Relacion.RelacionRef) || identidad.Autorizacion.Revalidacion.FechaReferencia != normalizada.FechaInicio {
		return cero, false, dietasports.ErrAccesoBorradorDenegado
	}
	recuperador, ok := s.repositorio.(interface {
		RecuperarPorClave(context.Context, dietasports.IdentidadEfectivaBorrador, dietasports.SolicitudCrearBorradorPropio) (dietasports.ResultadoBorradorComision, bool, error)
	})
	if !ok {
		return cero, false, dietasports.ErrBorradorNoDisponible
	}
	return recuperador.RecuperarPorClave(ctx, identidad, normalizada)
}

func (s *ServicioBorradorComision) ObtenerPropio(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, referencia string) (dietasports.ResultadoBorradorComision, error) {
	if ctx == nil || !servicioValido(s) || !referenciaComision.MatchString(referencia) {
		return dietasports.ResultadoBorradorComision{}, dietasports.ErrComisionNoEncontrada
	}
	if err := ctx.Err(); err != nil {
		return dietasports.ResultadoBorradorComision{}, err
	}
	// No revelar una referencia ajena, pero sí mantener denegación inequívoca
	// cuando la frontera no pudo acreditar la identidad.
	if err := identidadValida(identidad, AccionConsultarBorradorPropio, referencia, FinalidadConsultarBorradorPropio); err != nil {
		return dietasports.ResultadoBorradorComision{}, err
	}
	return s.repositorio.ObtenerPropio(ctx, identidad, referencia)
}

func (s *ServicioBorradorComision) ListarPropios(ctx context.Context, identidad dietasports.IdentidadEfectivaBorrador, consulta dietasports.ConsultaBorradoresPropios) (dietasports.PaginaBorradoresPropios, error) {
	if ctx == nil || !servicioValido(s) {
		return dietasports.PaginaBorradoresPropios{}, dietasports.ErrAccesoBorradorDenegado
	}
	if err := ctx.Err(); err != nil {
		return dietasports.PaginaBorradoresPropios{}, err
	}
	if err := identidadValida(identidad, AccionConsultarBorradorPropio, RecursoMisBorradores, FinalidadConsultarBorradorPropio); err != nil {
		return dietasports.PaginaBorradoresPropios{}, err
	}
	if consulta.Limite == 0 {
		consulta.Limite = 20
	}
	if consulta.Limite < 1 || consulta.Limite > 50 || (consulta.Cursor != "" && !referenciaComision.MatchString(consulta.Cursor)) {
		return dietasports.PaginaBorradoresPropios{}, domain.ErrComisionBorradorInvalida
	}
	return s.repositorio.ListarPropios(ctx, identidad, consulta)
}

func identidadValida(identidad dietasports.IdentidadEfectivaBorrador, accion, recurso, finalidad string) error {
	if identidad.ContextoRegistrado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.ContextoRegistrado) != nil {
		return dietasports.ErrAccesoBorradorDenegado
	}
	actor := identidad.ContextoRegistrado.Contexto
	if actor.Validar() != nil || !relacionAcreditada(identidad.Relacion) || identidad.Relacion.PersonaRef != actor.PersonaRef {
		return dietasports.ErrAccesoBorradorDenegado
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 || empleados[0] != identidad.Relacion.EmpleadoRef {
		return dietasports.ErrRelacionNoValida
	}
	a := identidad.Autorizacion
	if a.Material.ValidarEstructura() != nil || a.Accion != accion || a.RecursoRef != recurso || a.Finalidad != finalidad ||
		a.Revalidacion.RelacionRef != identidad.Relacion.RelacionRef || a.Revalidacion.PersonaRef != identidad.Relacion.PersonaRef ||
		a.Revalidacion.EmpleadoRef != identidad.Relacion.EmpleadoRef || a.Revalidacion.UnidadRef != identidad.Relacion.UnidadRef ||
		a.Revalidacion.VigenteDesde != identidad.Relacion.VigenteDesde || a.Revalidacion.VigenteHasta != identidad.Relacion.VigenteHasta || !fechaCivilCanonica(a.Revalidacion.FechaReferencia) ||
		a.Revalidacion.Version != identidad.Relacion.Version || a.Revalidacion.ProcedenciaActoRef != identidad.Relacion.ProcedenciaActoRef ||
		a.Revalidacion.FuenteRef != identidad.Relacion.FuenteRef || a.Revalidacion.FuenteVersion != identidad.Relacion.FuenteVersion {
		return dietasports.ErrAccesoBorradorDenegado
	}
	return nil
}

func relacionAcreditada(r dietasports.RelacionServicioAcreditada) bool {
	return r.RelacionRef != "" && r.PersonaRef != "" && r.EmpleadoRef != "" && r.UnidadRef != "" &&
		fechaCivilCanonica(r.VigenteDesde) && (r.VigenteHasta == "" || fechaCivilCanonica(r.VigenteHasta)) &&
		(r.VigenteHasta == "" || r.VigenteDesde < r.VigenteHasta) && r.Version > 0 &&
		r.ProcedenciaActoRef != "" && r.FuenteRef != "" && r.FuenteVersion > 0
}

func fechaCivilCanonica(valor string) bool {
	fecha, err := time.Parse("2006-01-02", valor)
	return err == nil && fecha.Format("2006-01-02") == valor
}

func servicioValido(s *ServicioBorradorComision) bool {
	return s != nil && !interfazNula(s.repositorio)
}
func interfazNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return (r.Kind() == reflect.Ptr || r.Kind() == reflect.Interface || r.Kind() == reflect.Func || r.Kind() == reflect.Map || r.Kind() == reflect.Slice) && r.IsNil()
}

var _ CasoUsoBorradorComision = (*ServicioBorradorComision)(nil)

type materialOperacionBorradorV1 struct {
	Esquema         string                       `json:"esquema"`
	Operacion       string                       `json:"operacion"`
	RecursoRef      string                       `json:"recurso_ref"`
	Identidad       identidadOperacionBorradorV1 `json:"identidad"`
	HuellaSemantica string                       `json:"huella_semantica,omitempty"`
	Comando         *crearOperacionBorradorV1    `json:"comando,omitempty"`
	Consulta        *listaOperacionBorradorV1    `json:"consulta,omitempty"`
	Referencia      string                       `json:"referencia,omitempty"`
}
type identidadOperacionBorradorV1 struct {
	ActorRef           string `json:"actor_ref"`
	PerfilRef          string `json:"perfil_ref"`
	PersonaRef         string `json:"persona_ref"`
	EmpleadoRef        string `json:"empleado_ref"`
	RelacionRef        string `json:"relacion_ref"`
	UnidadRef          string `json:"unidad_ref"`
	RelacionVersion    int64  `json:"relacion_version"`
	VigenteDesde       string `json:"vigente_desde"`
	VigenteHasta       string `json:"vigente_hasta"`
	FechaReferencia    string `json:"fecha_referencia"`
	ProcedenciaActoRef string `json:"procedencia_acto_ref"`
	FuenteRef          string `json:"fuente_ref"`
	FuenteVersion      int64  `json:"fuente_version"`
	ContextoActorRef   string `json:"contexto_actor_ref"`
	ContextoVersion    uint64 `json:"contexto_version"`
	CuentaRef          string `json:"cuenta_ref"`
	CuentaVersion      uint64 `json:"cuenta_version"`
	PersonaVersion     uint64 `json:"persona_version"`
	PerfilVersion      uint64 `json:"perfil_version"`
}
type crearOperacionBorradorV1 struct {
	ClaveIdempotencia string                  `json:"clave_idempotencia"`
	FechaInicio       string                  `json:"fecha_inicio"`
	FechaFin          string                  `json:"fecha_fin"`
	Motivo            string                  `json:"motivo"`
	CodigosRuta       []string                `json:"codigos_ruta"`
	HoraInicio        string                  `json:"hora_inicio,omitempty"`
	HoraFin           string                  `json:"hora_fin,omitempty"`
	Calculo           *domain.CalculoComision `json:"calculo,omitempty"`
}
type listaOperacionBorradorV1 struct {
	Cursor string `json:"cursor"`
	Limite int    `json:"limite"`
}

type replaySemanticoCrearBorradorV1 struct {
	PersonaRef         string   `json:"persona_ref"`
	EmpleadoRef        string   `json:"empleado_ref"`
	RelacionRef        string   `json:"relacion_ref"`
	UnidadRef          string   `json:"unidad_ref"`
	RelacionVersion    int64    `json:"relacion_version"`
	VigenteDesde       string   `json:"vigente_desde"`
	VigenteHasta       string   `json:"vigente_hasta"`
	ProcedenciaActoRef string   `json:"procedencia_acto_ref"`
	FuenteRef          string   `json:"fuente_ref"`
	FuenteVersion      int64    `json:"fuente_version"`
	ClaveIdempotencia  string   `json:"clave_idempotencia"`
	FechaInicio        string   `json:"fecha_inicio"`
	FechaFin           string   `json:"fecha_fin"`
	Motivo             string   `json:"motivo"`
	CodigosRuta        []string `json:"codigos_ruta"`
	HoraInicio         string   `json:"hora_inicio,omitempty"`
	HoraFin            string   `json:"hora_fin,omitempty"`
}

// ConstruirEfectoAutorizacionBorrador liga actor registrado, relación
// acreditada, sello de Personal y solicitud ya normalizada. La huella del
// recurso es la canónica VEC de ámbitos+atributos, no el SHA bruto material.
func ConstruirEfectoAutorizacionBorrador(contexto vecdomain.ResultadoContextoActorRegistradoV2, relacion dietasports.RelacionServicioAcreditada, sello dietasports.RevalidacionRelacionPersonal, solicitud dietasports.SolicitudOperacionBorrador) (dietasports.EfectoAutorizacionBorrador, error) {
	if contexto.Validar() != nil || validarSolicitudOperacion(solicitud) != nil || !relacionSelloValido(relacion, sello) {
		return dietasports.EfectoAutorizacionBorrador{}, dietasports.ErrEfectoAutorizacionBorradorInvalido
	}
	return construirEfectoAutorizacionBorradorActor(contexto.Contexto, relacion, sello, solicitud)
}

// construirEfectoAutorizacionBorradorActor conserva la preimagen aislada para
// pruebas del contrato. La API pública exige además el recibo ContextoActor V2.
func construirEfectoAutorizacionBorradorActor(actor vecdomain.ContextoActor, relacion dietasports.RelacionServicioAcreditada, sello dietasports.RevalidacionRelacionPersonal, solicitud dietasports.SolicitudOperacionBorrador) (dietasports.EfectoAutorizacionBorrador, error) {
	if actor.Validar() != nil || validarSolicitudOperacion(solicitud) != nil || !relacionSelloValido(relacion, sello) {
		return dietasports.EfectoAutorizacionBorrador{}, dietasports.ErrEfectoAutorizacionBorradorInvalido
	}
	if solicitud.RelacionRef != "" && solicitud.RelacionRef != relacion.RelacionRef {
		return dietasports.EfectoAutorizacionBorrador{}, dietasports.ErrEfectoAutorizacionBorradorInvalido
	}
	if solicitud.Operacion == dietasports.OperacionCrearBorrador && sello.FechaReferencia != solicitud.Crear.FechaInicio {
		return dietasports.EfectoAutorizacionBorrador{}, dietasports.ErrEfectoAutorizacionBorradorInvalido
	}
	empleados, err := actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || actor.PersonaRef != relacion.PersonaRef || len(empleados) != 1 || empleados[0] != relacion.EmpleadoRef {
		return dietasports.EfectoAutorizacionBorrador{}, dietasports.ErrEfectoAutorizacionBorradorInvalido
	}
	instantanea := actor.Instantanea
	if instantanea.CuentaVersion == 0 || instantanea.PersonaVersion == 0 || instantanea.PerfilVersion == 0 || instantanea.VinculoVersion == 0 {
		return dietasports.EfectoAutorizacionBorrador{}, dietasports.ErrEfectoAutorizacionBorradorInvalido
	}
	id := identidadOperacionBorradorV1{
		ActorRef: actor.Principal.ID, PerfilRef: actor.PerfilActivoRef,
		PersonaRef: relacion.PersonaRef, EmpleadoRef: relacion.EmpleadoRef,
		RelacionRef: relacion.RelacionRef, UnidadRef: relacion.UnidadRef,
		RelacionVersion: relacion.Version, VigenteDesde: relacion.VigenteDesde,
		VigenteHasta: relacion.VigenteHasta, FechaReferencia: sello.FechaReferencia,
		ProcedenciaActoRef: relacion.ProcedenciaActoRef, FuenteRef: relacion.FuenteRef,
		FuenteVersion: relacion.FuenteVersion, ContextoActorRef: instantanea.VinculoRef,
		ContextoVersion: instantanea.VinculoVersion, CuentaRef: instantanea.CuentaRef,
		CuentaVersion: instantanea.CuentaVersion, PersonaVersion: instantanea.PersonaVersion,
		PerfilVersion: instantanea.PerfilVersion,
	}
	m := materialOperacionBorradorV1{Esquema: dietasports.EsquemaEfectoAutorizacionBorradorV1, Identidad: id}
	var operacion string
	switch {
	case solicitud.Operacion == dietasports.OperacionCrearBorrador:
		operacion = "crear"
		m.RecursoRef = "dietas:borradores:propios"
		m.Comando = &crearOperacionBorradorV1{solicitud.Crear.ClaveIdempotencia, solicitud.Crear.FechaInicio, solicitud.Crear.FechaFin, solicitud.Crear.Motivo, append([]string{}, solicitud.Crear.CodigosRuta...), solicitud.Crear.HoraInicio, solicitud.Crear.HoraFin, solicitud.Crear.Calculo}
		m.HuellaSemantica, err = huellaSemanticaCrearBorrador(relacion, solicitud.Crear)
		if err != nil {
			return dietasports.EfectoAutorizacionBorrador{}, dietasports.ErrEfectoAutorizacionBorradorInvalido
		}
	case solicitud.Referencia != "":
		operacion = "detalle"
		m.RecursoRef = solicitud.Referencia
		m.Referencia = solicitud.Referencia
	case solicitud.Operacion == dietasports.OperacionConsultarBorrador:
		operacion = "lista"
		m.RecursoRef = "dietas:borradores:propios"
		m.Consulta = &listaOperacionBorradorV1{solicitud.Consulta.Cursor, solicitud.Consulta.Limite}
	default:
		return dietasports.EfectoAutorizacionBorrador{}, dietasports.ErrEfectoAutorizacionBorradorInvalido
	}
	m.Operacion = operacion
	material, err := json.Marshal(m)
	if err != nil {
		return dietasports.EfectoAutorizacionBorrador{}, dietasports.ErrEfectoAutorizacionBorradorInvalido
	}
	h := sha256.Sum256(material)
	atributos := map[string]string{
		"operacion": operacion, "recurso_ref": m.RecursoRef, "material_sha256": hex.EncodeToString(h[:]),
		"fecha_referencia": sello.FechaReferencia, "relacion_version": strconv.FormatInt(id.RelacionVersion, 10),
		"vigente_desde": relacion.VigenteDesde, "procedencia_acto_ref": relacion.ProcedenciaActoRef,
		"fuente_ref": relacion.FuenteRef, "fuente_version": strconv.FormatInt(id.FuenteVersion, 10),
		"contexto_actor_ref": id.ContextoActorRef, "contexto_version": strconv.FormatUint(id.ContextoVersion, 10),
		"cuenta_ref": id.CuentaRef, "cuenta_version": strconv.FormatUint(id.CuentaVersion, 10),
		"persona_version": strconv.FormatUint(id.PersonaVersion, 10), "perfil_version": strconv.FormatUint(id.PerfilVersion, 10),
	}
	// RecursoAutorizable no admite cadenas vacías; `sin_fin` es el centinela
	// técnico cerrado de una relación abierta. El material conserva la cadena
	// vacía original, y el consumidor SQL debe reconstruir este mismo atributo.
	atributos["vigente_hasta"] = relacion.VigenteHasta
	if relacion.VigenteHasta == "" {
		atributos["vigente_hasta"] = "sin_fin"
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: m.RecursoRef, ModuloID: dietasports.ModuloDietas, Tipo: dietasports.TipoRecursoComisionBorrador, Ambitos: map[string]string{"persona_ref": relacion.PersonaRef, "empleado_ref": relacion.EmpleadoRef, "relacion_ref": relacion.RelacionRef, "unidad_ref": relacion.UnidadRef}, Atributos: atributos}
	if recurso.Validar() != nil {
		return dietasports.EfectoAutorizacionBorrador{}, dietasports.ErrEfectoAutorizacionBorradorInvalido
	}
	return dietasports.EfectoAutorizacionBorrador{Material: append([]byte(nil), material...), Recurso: recurso}, nil
}

func huellaSemanticaCrearBorrador(relacion dietasports.RelacionServicioAcreditada, solicitud dietasports.SolicitudCrearBorradorPropio) (string, error) {
	canon := replaySemanticoCrearBorradorV1{
		PersonaRef: relacion.PersonaRef, EmpleadoRef: relacion.EmpleadoRef, RelacionRef: relacion.RelacionRef,
		UnidadRef: relacion.UnidadRef, RelacionVersion: relacion.Version,
		VigenteDesde: relacion.VigenteDesde, VigenteHasta: relacion.VigenteHasta,
		ProcedenciaActoRef: relacion.ProcedenciaActoRef, FuenteRef: relacion.FuenteRef, FuenteVersion: relacion.FuenteVersion,
		ClaveIdempotencia: solicitud.ClaveIdempotencia, FechaInicio: solicitud.FechaInicio, FechaFin: solicitud.FechaFin,
		Motivo: solicitud.Motivo, CodigosRuta: append([]string{}, solicitud.CodigosRuta...), HoraInicio: solicitud.HoraInicio, HoraFin: solicitud.HoraFin,
	}
	bytes, err := json.Marshal(canon)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(bytes)
	return hex.EncodeToString(suma[:]), nil
}

func relacionSelloValido(r dietasports.RelacionServicioAcreditada, s dietasports.RevalidacionRelacionPersonal) bool {
	if !referenciaDietas(r.RelacionRef, "rel_") || !referenciaDietas(r.PersonaRef, "per_") || !referenciaDietas(r.EmpleadoRef, "emp_") || r.UnidadRef == "" ||
		!fechaCivilSelloValida(r.VigenteDesde) || (r.VigenteHasta != "" && (!fechaCivilSelloValida(r.VigenteHasta) || r.VigenteDesde >= r.VigenteHasta)) ||
		r.Version <= 0 || r.FuenteVersion <= 0 || r.ProcedenciaActoRef == "" || r.FuenteRef == "" || !fechaCivilSelloValida(s.FechaReferencia) || s.Version <= 0 || s.FuenteVersion <= 0 {
		return false
	}
	return s.RelacionRef == r.RelacionRef && s.PersonaRef == r.PersonaRef && s.EmpleadoRef == r.EmpleadoRef && s.UnidadRef == r.UnidadRef && s.VigenteDesde == r.VigenteDesde && s.VigenteHasta == r.VigenteHasta && s.Version == r.Version && s.ProcedenciaActoRef == r.ProcedenciaActoRef && s.FuenteRef == r.FuenteRef && s.FuenteVersion == r.FuenteVersion
}

func fechaCivilSelloValida(valor string) bool {
	fecha, err := time.Parse("2006-01-02", valor)
	return err == nil && fecha.Format("2006-01-02") == valor
}

func referenciaDietas(valor, prefijo string) bool {
	if len(valor) < len(prefijo)+22 || len(valor) > len(prefijo)+128 || len(valor) <= len(prefijo) || valor[:len(prefijo)] != prefijo {
		return false
	}
	for _, r := range valor[len(prefijo):] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

var claveIdempotenciaDietas = regexp.MustCompile(`^[A-Za-z0-9_-]{16,128}$`)
var codigoRutaDietas = regexp.MustCompile(`^[A-Za-z0-9:_-]{1,64}$`)

func normalizarSolicitudCrear(s dietasports.SolicitudCrearBorradorPropio) (dietasports.SolicitudCrearBorradorPropio, error) {
	s.CodigosRuta = append([]string{}, s.CodigosRuta...)
	vistos := map[string]bool{}
	for _, codigo := range s.CodigosRuta {
		if vistos[codigo] {
			return s, domain.ErrComisionBorradorInvalida
		}
		vistos[codigo] = true
	}
	return s, validarSolicitudCrear(s)
}
func validarSolicitudCrear(s dietasports.SolicitudCrearBorradorPropio) error {
	i, e1 := time.Parse("2006-01-02", s.FechaInicio)
	f, e2 := time.Parse("2006-01-02", s.FechaFin)
	if !claveIdempotenciaDietas.MatchString(s.ClaveIdempotencia) || e1 != nil || e2 != nil || i.After(f) || len(s.Motivo) < 3 || len(s.Motivo) > 600 || len(s.CodigosRuta) > 16 || (s.RelacionRef != "" && !referenciaDietas(s.RelacionRef, "rel_")) || (s.HoraInicio != "" && !horaSolicitudValida(s.HoraInicio)) || (s.HoraFin != "" && !horaSolicitudValida(s.HoraFin)) || (s.Calculo != nil && (s.Calculo.Validar(s.CodigosRuta) != nil || s.HoraInicio != s.Calculo.HoraInicio || s.HoraFin != s.Calculo.HoraFin)) {
		return domain.ErrComisionBorradorInvalida
	}
	for _, r := range s.Motivo {
		if r < 0x20 || r == 0x7f {
			return domain.ErrComisionBorradorInvalida
		}
	}
	for _, c := range s.CodigosRuta {
		if !codigoRutaDietas.MatchString(c) {
			return domain.ErrComisionBorradorInvalida
		}
	}
	return nil
}
func NuevaSolicitudOperacionCrearBorrador(s dietasports.SolicitudCrearBorradorPropio) (dietasports.SolicitudOperacionBorrador, error) {
	n, e := normalizarSolicitudCrear(s)
	if e != nil {
		return dietasports.SolicitudOperacionBorrador{}, e
	}
	return dietasports.SolicitudOperacionBorrador{Operacion: dietasports.OperacionCrearBorrador, Crear: n, RelacionRef: n.RelacionRef}, nil
}
func NuevaSolicitudOperacionListarBorrador(c dietasports.ConsultaBorradoresPropios, rel string) (dietasports.SolicitudOperacionBorrador, error) {
	if c.Limite == 0 {
		c.Limite = 20
	}
	if c.Limite < 1 || c.Limite > 50 || (c.Cursor != "" && !referenciaDietas(c.Cursor, "dco_")) || (rel != "" && !referenciaDietas(rel, "rel_")) {
		return dietasports.SolicitudOperacionBorrador{}, domain.ErrComisionBorradorInvalida
	}
	return dietasports.SolicitudOperacionBorrador{Operacion: dietasports.OperacionConsultarBorrador, Consulta: c, RelacionRef: rel}, nil
}
func NuevaSolicitudOperacionObtenerBorrador(ref, rel string) (dietasports.SolicitudOperacionBorrador, error) {
	if !referenciaDietas(ref, "dco_") || (rel != "" && !referenciaDietas(rel, "rel_")) {
		return dietasports.SolicitudOperacionBorrador{}, domain.ErrComisionBorradorInvalida
	}
	return dietasports.SolicitudOperacionBorrador{Operacion: dietasports.OperacionConsultarBorrador, Referencia: ref, RelacionRef: rel}, nil
}
func validarSolicitudOperacion(s dietasports.SolicitudOperacionBorrador) error {
	if s.Operacion == dietasports.OperacionCrearBorrador {
		n, e := normalizarSolicitudCrear(s.Crear)
		if e != nil || s.RelacionRef != n.RelacionRef || s.Referencia != "" || s.Consulta != (dietasports.ConsultaBorradoresPropios{}) || !solicitudesCrearIguales(s.Crear, n) {
			return domain.ErrComisionBorradorInvalida
		}
		return nil
	}
	if s.Operacion != dietasports.OperacionConsultarBorrador || (s.RelacionRef != "" && !referenciaDietas(s.RelacionRef, "rel_")) || !solicitudCrearVacia(s.Crear) {
		return domain.ErrComisionBorradorInvalida
	}
	if s.Referencia != "" {
		if !referenciaDietas(s.Referencia, "dco_") || s.Consulta != (dietasports.ConsultaBorradoresPropios{}) {
			return domain.ErrComisionBorradorInvalida
		}
		return nil
	}
	if s.Consulta.Limite < 1 || s.Consulta.Limite > 50 || (s.Consulta.Cursor != "" && !referenciaDietas(s.Consulta.Cursor, "dco_")) {
		return domain.ErrComisionBorradorInvalida
	}
	return nil
}

func solicitudesCrearIguales(a, b dietasports.SolicitudCrearBorradorPropio) bool {
	if a.ClaveIdempotencia != b.ClaveIdempotencia || a.FechaInicio != b.FechaInicio || a.FechaFin != b.FechaFin || a.HoraInicio != b.HoraInicio || a.HoraFin != b.HoraFin || a.Motivo != b.Motivo || a.RelacionRef != b.RelacionRef || !reflect.DeepEqual(a.Calculo, b.Calculo) || len(a.CodigosRuta) != len(b.CodigosRuta) {
		return false
	}
	for i := range a.CodigosRuta {
		if a.CodigosRuta[i] != b.CodigosRuta[i] {
			return false
		}
	}
	return true
}
func solicitudCrearVacia(s dietasports.SolicitudCrearBorradorPropio) bool {
	return s.ClaveIdempotencia == "" && s.FechaInicio == "" && s.FechaFin == "" && s.HoraInicio == "" && s.HoraFin == "" && s.Calculo == nil && s.Motivo == "" && s.RelacionRef == "" && len(s.CodigosRuta) == 0
}

func horaSolicitudValida(s string) bool {
	return len(s) == 5 && s[2] == ':' && s[:2] >= "00" && s[:2] <= "23" && s[3:] >= "00" && s[3:] <= "59"
}
