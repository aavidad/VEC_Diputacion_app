package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	core "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionExportacionServiciosPropios      = "personal.registro_empleado.ficha_propia.servicios.exportar"
	AudienciaExportacionServiciosPropios   = "vec_personal.registro_empleado.ficha_propia.servicios.exportar.v1"
	FinalidadExportacionServiciosPropios   = "exportar_servicios_propios"
	TipoRecursoExportacionServiciosPropios = "exportacion_servicios_propios"
	LimiteBytesExportacionServiciosPropios = 1 << 20
)

var (
	ErrExportacionServiciosPropiosInvalida     = errors.New("personal: exportacion servicios propios invalida")
	ErrExportacionServiciosPropiosDenegada     = errors.New("personal: exportacion servicios propios denegada")
	ErrExportacionServiciosPropiosNoDisponible = errors.New("personal: exportacion servicios propios no disponible")
	idiomaExportacionServicios                 = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8}){0,3}$`)
	referenciaFormatoExportacionServicios      = regexp.MustCompile(`^[a-z][a-z0-9._:-]{1,127}$`)
	reciboExportacionServicios                 = regexp.MustCompile(`^fichapropia:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	archivoExportacionServicios                = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,79}\.csv$`)
	huellaExportacionServicios                 = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// El formato procede de un catálogo del servidor. La petición sólo elige un
// idioma que ese catálogo tenga disponible; no aporta cabeceras ni etiquetas.
type DatosFormatoExportacionServiciosPropios struct {
	Referencia     string
	Version        uint64
	Idioma         string
	CatalogoSHA256 string
	NombreArchivo  string
	Cabeceras      []string
	Estados        map[string]string
}
type FormatoExportacionServiciosPropios struct {
	datos DatosFormatoExportacionServiciosPropios
}

func NuevoFormatoExportacionServiciosPropios(d DatosFormatoExportacionServiciosPropios) (FormatoExportacionServiciosPropios, error) {
	if !referenciaFormatoExportacionServicios.MatchString(d.Referencia) || d.Version == 0 || d.Version > 9007199254740991 || !idiomaExportacionServicios.MatchString(d.Idioma) || !huellaExportacionServicios.MatchString(d.CatalogoSHA256) || !archivoExportacionServicios.MatchString(d.NombreArchivo) || len(d.Cabeceras) != 5 || len(d.Estados) != 3 {
		return FormatoExportacionServiciosPropios{}, ErrExportacionServiciosPropiosInvalida
	}
	for _, v := range d.Cabeceras {
		if !textoFormatoExportacionValido(v) {
			return FormatoExportacionServiciosPropios{}, ErrExportacionServiciosPropiosInvalida
		}
	}
	for _, estado := range []string{"declarado", "comprobado", "reconocido"} {
		if !textoFormatoExportacionValido(d.Estados[estado]) {
			return FormatoExportacionServiciosPropios{}, ErrExportacionServiciosPropiosInvalida
		}
	}
	d.Cabeceras = append([]string(nil), d.Cabeceras...)
	d.Estados = copiarMapaRelacion(d.Estados)
	return FormatoExportacionServiciosPropios{d}, nil
}
func textoFormatoExportacionValido(v string) bool {
	return v != "" && utf8.ValidString(v) && len(v) <= 160 && !strings.ContainsAny(v, "\x00\r\n")
}
func (f FormatoExportacionServiciosPropios) Datos() DatosFormatoExportacionServiciosPropios {
	d := f.datos
	d.Cabeceras = append([]string(nil), d.Cabeceras...)
	d.Estados = copiarMapaRelacion(d.Estados)
	return d
}
func (f FormatoExportacionServiciosPropios) Validar() error {
	_, e := NuevoFormatoExportacionServiciosPropios(f.Datos())
	return e
}

type SolicitudExportacionServiciosPropios struct {
	Actor     core.ContextoActor
	ReciboRef string
	Corte     CorteEmpleadoB2
	Idioma    string
}
type MaterialExportacionServiciosPropios struct {
	solicitud SolicitudExportacionServiciosPropios
	formato   FormatoExportacionServiciosPropios
	empleado  string
	canonico  []byte
	recurso   core.RecursoAutorizable
}

func NuevoMaterialExportacionServiciosPropios(s SolicitudExportacionServiciosPropios, f FormatoExportacionServiciosPropios) (MaterialExportacionServiciosPropios, error) {
	var cero MaterialExportacionServiciosPropios
	if !reciboExportacionServicios.MatchString(s.ReciboRef) || s.Corte.Validar() != nil || s.Actor.Validar() != nil || f.Validar() != nil || s.Idioma != f.datos.Idioma {
		return cero, ErrExportacionServiciosPropiosInvalida
	}
	base, err := NuevoMaterialFichaPropia(SolicitudFichaPropia{Actor: s.Actor, Corte: s.Corte})
	if err != nil {
		if errors.Is(err, ErrFichaPropiaSinEmpleado) || errors.Is(err, ErrFichaPropiaAmbigua) {
			return cero, ErrExportacionServiciosPropiosDenegada
		}
		return cero, ErrExportacionServiciosPropiosInvalida
	}
	actor := base.Actor()
	s.Actor = actor
	d := f.Datos()
	contenido := struct {
		Esquema          string `json:"esquema"`
		EmpleadoRef      string `json:"empleado_ref"`
		ReciboRef        string `json:"recibo_ref"`
		VigenteEn        string `json:"vigente_en"`
		ConocidoEn       string `json:"conocido_en"`
		Idioma           string `json:"idioma"`
		FormatoRef       string `json:"formato_ref"`
		FormatoVersion   uint64 `json:"formato_version"`
		CatalogoSHA256   string `json:"catalogo_sha256"`
		ActorRef         string `json:"actor_ref"`
		ContextoActorRef string `json:"contexto_actor_ref"`
		ContextoVersion  uint64 `json:"contexto_version"`
		CuentaRef        string `json:"cuenta_ref"`
		CuentaVersion    uint64 `json:"cuenta_version"`
		PerfilRef        string `json:"perfil_ref"`
		PerfilVersion    uint64 `json:"perfil_version"`
		PersonaRef       string `json:"persona_ref"`
		PersonaVersion   uint64 `json:"persona_version"`
	}{"vec.personal.servicios-propios.exportacion.v1", base.EmpleadoRef(), s.ReciboRef, s.Corte.VigenteEn.Texto(), s.Corte.ConocidoEn.UTC().Format("2006-01-02T15:04:05.000000Z"), s.Idioma, d.Referencia, d.Version, d.CatalogoSHA256, actor.Principal.ID, actor.Instantanea.VinculoRef, actor.Instantanea.VinculoVersion, actor.Instantanea.CuentaRef, actor.Instantanea.CuentaVersion, actor.PerfilActivoRef, actor.Instantanea.PerfilVersion, actor.PersonaRef, actor.Instantanea.PersonaVersion}
	canon, err := json.Marshal(contenido)
	if err != nil {
		return cero, ErrExportacionServiciosPropiosInvalida
	}
	h := sha256.Sum256(canon)
	recurso := core.RecursoAutorizable{Referencia: base.EmpleadoRef(), ModuloID: "personal", Tipo: TipoRecursoExportacionServiciosPropios, Ambitos: map[string]string{"empleado_ref": base.EmpleadoRef()}, Atributos: map[string]string{"operacion": "servicios_propios_exportar", "recibo_ref": s.ReciboRef, "vigente_en": contenido.VigenteEn, "conocido_en": contenido.ConocidoEn, "idioma": s.Idioma, "formato_ref": d.Referencia, "formato_version": jsonNumeroFormato(d.Version), "catalogo_sha256": d.CatalogoSHA256, "material_sha256": hex.EncodeToString(h[:])}}
	if _, err = recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return cero, ErrExportacionServiciosPropiosInvalida
	}
	return MaterialExportacionServiciosPropios{s, f, base.EmpleadoRef(), canon, recurso}, nil
}
func jsonNumeroFormato(v uint64) string { b, _ := json.Marshal(v); return string(b) }
func (m MaterialExportacionServiciosPropios) Solicitud() SolicitudExportacionServiciosPropios {
	s := m.solicitud
	s.Actor, _ = s.Actor.Clonar()
	return s
}
func (m MaterialExportacionServiciosPropios) Actor() core.ContextoActor { return m.Solicitud().Actor }
func (m MaterialExportacionServiciosPropios) Corte() CorteEmpleadoB2    { return m.solicitud.Corte }
func (m MaterialExportacionServiciosPropios) EmpleadoRef() string       { return m.empleado }
func (m MaterialExportacionServiciosPropios) Formato() FormatoExportacionServiciosPropios {
	f, _ := NuevoFormatoExportacionServiciosPropios(m.formato.Datos())
	return f
}
func (m MaterialExportacionServiciosPropios) Canonico() []byte {
	return append([]byte(nil), m.canonico...)
}
func (m MaterialExportacionServiciosPropios) Recurso() core.RecursoAutorizable {
	r := m.recurso
	r.Ambitos = copiarMapaRelacion(r.Ambitos)
	r.Atributos = copiarMapaRelacion(r.Atributos)
	return r
}
func (m MaterialExportacionServiciosPropios) HuellaSHA256() (string, error) {
	return m.recurso.HuellaContextoAutorizacionSHA256()
}
func (m MaterialExportacionServiciosPropios) ValidarServicios(c CorteEmpleadoB2, s []ServicioFichaPropia) error {
	if s == nil || len(s) > LimiteFilasFichaPropia || !corteRegistroB2Igual(c, m.Corte()) {
		return ErrExportacionServiciosPropiosInvalida
	}
	f := FichaPropia{Corte: c, Relaciones: []RelacionFichaPropia{}, Servicios: s}
	base, err := NuevoMaterialFichaPropia(SolicitudFichaPropia{Actor: m.Actor(), Corte: m.Corte()})
	if err != nil || f.ValidarPara(base) != nil {
		return ErrExportacionServiciosPropiosInvalida
	}
	return nil
}
