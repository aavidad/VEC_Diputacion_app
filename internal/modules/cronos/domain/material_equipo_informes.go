package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrMaterialEquipoInformeInvalido = errors.New("cronos: material de equipo o informe inválido")

type TipoConsultaEquipo string

const (
	ConsultaPresenciaEquipo   TipoConsultaEquipo = "presencia"
	ConsultaIncidenciasEquipo TipoConsultaEquipo = "incidencias"
)

type AlcanceConsultaEquipo string

const (
	AlcanceJefatura AlcanceConsultaEquipo = "jefatura"
	AlcanceRRHH     AlcanceConsultaEquipo = "rrhh"
)

// FotoEquipoAutoritativa procede de Personal/Organización y de su recibo
// nominal. El navegador no elige miembros, versión ni ámbito.
type FotoEquipoAutoritativa struct {
	EquipoRef, UnidadRef, OrganismoRef, VersionRef string
	ReciboFuenteRef, FuenteRef                     string
	Desde, HastaExclusivo                          string
}

// MaterialConsultaEquipo liga actor, ámbito histórico, corte y período antes
// de pedir la decisión V3. No incluye una lista de personas del navegador.
type MaterialConsultaEquipo struct {
	ActorRef, PerfilRef, EmpleadoRef string
	Foto                             FotoEquipoAutoritativa
	Tipo                             TipoConsultaEquipo
	Alcance                          AlcanceConsultaEquipo
	Desde, Hasta                     string
	ZonaHoraria                      string
	Cursor                           string
}

func (m MaterialConsultaEquipo) Canonico() ([]byte, error) {
	if m.Tipo != ConsultaPresenciaEquipo && m.Tipo != ConsultaIncidenciasEquipo ||
		(m.Alcance != AlcanceJefatura && m.Alcance != AlcanceRRHH) ||
		!referenciaIdentidadMarcaje(m.ActorRef, "per_") ||
		!referenciaIdentidadMarcaje(m.PerfilRef, "prf_") ||
		!referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") ||
		!referenciaMarcaje(m.Foto.EquipoRef) || !referenciaMarcaje(m.Foto.UnidadRef) ||
		!referenciaMarcaje(m.Foto.OrganismoRef) || !referenciaMarcaje(m.Foto.VersionRef) ||
		!referenciaMarcaje(m.Foto.ReciboFuenteRef) || !referenciaMarcaje(m.Foto.FuenteRef) ||
		!fechaMaterialEquipoValida(m.Foto.Desde) || !fechaMaterialEquipoValida(m.Foto.HastaExclusivo) ||
		m.Foto.Desde >= m.Foto.HastaExclusivo ||
		!fechaMaterialEquipoValida(m.Desde) || !fechaMaterialEquipoValida(m.Hasta) || m.Desde > m.Hasta ||
		m.Desde < m.Foto.Desde || m.Hasta >= m.Foto.HastaExclusivo ||
		(m.ZonaHoraria != ZonaSaldoPeninsula && m.ZonaHoraria != ZonaSaldoCanarias) ||
		!cursorEquipoValido(m.Cursor) {
		return nil, ErrMaterialEquipoInformeInvalido
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"equipo_ref": m.Foto.EquipoRef, "unidad_ref": m.Foto.UnidadRef,
		"organismo_ref": m.Foto.OrganismoRef, "equipo_version_ref": m.Foto.VersionRef,
		"recibo_fuente_ref": m.Foto.ReciboFuenteRef, "fuente_ref": m.Foto.FuenteRef,
		"desde": m.Desde, "hasta": m.Hasta, "vista": string(m.Tipo), "zona_horaria": m.ZonaHoraria,
		"alcance": string(m.Alcance), "cursor": m.Cursor,
	})
}

func (m MaterialConsultaEquipo) Recurso() (vecdomain.RecursoAutorizable, error) {
	canon, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(canon)
	r := vecdomain.RecursoAutorizable{
		Referencia: m.Foto.EquipoRef, ModuloID: "cronos", Tipo: string(m.Tipo) + "_equipo",
		Ambitos: map[string]string{
			"equipo_ref": m.Foto.EquipoRef, "unidad_ref": m.Foto.UnidadRef,
			"organismo_ref": m.Foto.OrganismoRef,
		},
		Atributos: map[string]string{
			"material_sha256": hex.EncodeToString(h[:]), "equipo_version_ref": m.Foto.VersionRef,
			"desde": m.Desde, "hasta": m.Hasta, "vista": string(m.Tipo),
			"alcance": string(m.Alcance), "cursor": cursorRecursoEquipo(m.Cursor),
		},
	}
	if _, err := r.HuellaContextoAutorizacionSHA256(); err != nil {
		return vecdomain.RecursoAutorizable{}, ErrMaterialEquipoInformeInvalido
	}
	return r, nil
}

type TipoInformeCronos string

const (
	InformeSaldo       TipoInformeCronos = "saldo"
	InformeMovimientos TipoInformeCronos = "movimientos"
	InformePermisos    TipoInformeCronos = "permisos"
)

type FormatoInformeCronos string

const (
	InformePDF FormatoInformeCronos = "pdf"
	InformeCSV FormatoInformeCronos = "csv"
)

// MaterialExportacionCronos exige política publicada de campos y tipo. El
// mismo permiso de lectura nunca satisface esta acción de exportación.
type MaterialExportacionCronos struct {
	ActorRef, PerfilRef, EmpleadoRef string
	Foto                             *FotoEquipoAutoritativa // nil: informe propio
	Alcance                          string                  // empleado, jefatura o rrhh; lo fija el servidor
	Tipo                             TipoInformeCronos
	Formato                          FormatoInformeCronos
	Desde, Hasta                     string
	ZonaHoraria                      string
	Idioma                           string
	PoliticaRef                      string
	PoliticaVersion                  int64
	PoliticaSHA256                   string
	ClaveOperacion                   string
	Fase                             string
	PreparacionRef                   string
	FuenteSHA256                     string
	DocumentoSHA256                  string
	Tamano                           int64
}

func (m MaterialExportacionCronos) Canonico() ([]byte, error) {
	if !referenciaIdentidadMarcaje(m.ActorRef, "per_") ||
		!referenciaIdentidadMarcaje(m.PerfilRef, "prf_") ||
		!referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") ||
		(m.Tipo != InformeSaldo && m.Tipo != InformeMovimientos && m.Tipo != InformePermisos) ||
		(m.Formato != InformePDF && m.Formato != InformeCSV) ||
		!fechaMaterialEquipoValida(m.Desde) || !fechaMaterialEquipoValida(m.Hasta) || m.Desde > m.Hasta ||
		(m.ZonaHoraria != ZonaSaldoPeninsula && m.ZonaHoraria != ZonaSaldoCanarias) ||
		!idiomaMaterialEquipoValido(m.Idioma) ||
		!referenciaMarcaje(m.PoliticaRef) || m.PoliticaVersion < 1 ||
		!HuellaEfectosValida(m.PoliticaSHA256) || !claveOperacionMarcaje(m.ClaveOperacion) {
		return nil, ErrMaterialEquipoInformeInvalido
	}
	if (m.Fase != "preparacion" && m.Fase != "confirmacion") ||
		(m.Fase == "preparacion" && (m.PreparacionRef != "" || m.FuenteSHA256 != "" || m.DocumentoSHA256 != "" || m.Tamano != 0)) ||
		(m.Fase == "confirmacion" && (!referenciaMarcaje(m.PreparacionRef) ||
			!HuellaEfectosValida(m.FuenteSHA256) || !HuellaEfectosValida(m.DocumentoSHA256) ||
			m.Tamano < 1 || m.Tamano > 10*1024*1024)) {
		return nil, ErrMaterialEquipoInformeInvalido
	}
	if (m.Foto == nil && m.Alcance != "empleado") ||
		(m.Foto != nil && m.Alcance != string(AlcanceJefatura) && m.Alcance != string(AlcanceRRHH)) {
		return nil, ErrMaterialEquipoInformeInvalido
	}
	base := map[string]any{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"alcance":      m.Alcance,
		"tipo_informe": m.Tipo, "formato": m.Formato, "desde": m.Desde, "hasta": m.Hasta,
		"zona_horaria": m.ZonaHoraria,
		"idioma":       m.Idioma,
		"politica_ref": m.PoliticaRef, "politica_version": m.PoliticaVersion,
		"politica_sha256": m.PoliticaSHA256, "clave_operacion": m.ClaveOperacion,
		"fase": m.Fase,
	}
	if m.Fase == "confirmacion" {
		base["preparacion_ref"] = m.PreparacionRef
		base["fuente_sha256"] = m.FuenteSHA256
		base["documento_sha256"] = m.DocumentoSHA256
		base["tamano"] = m.Tamano
	}
	if m.Foto != nil {
		if !referenciaMarcaje(m.Foto.EquipoRef) || !referenciaMarcaje(m.Foto.UnidadRef) ||
			!referenciaMarcaje(m.Foto.OrganismoRef) || !referenciaMarcaje(m.Foto.VersionRef) ||
			!referenciaMarcaje(m.Foto.ReciboFuenteRef) || !referenciaMarcaje(m.Foto.FuenteRef) ||
			!fechaMaterialEquipoValida(m.Foto.Desde) || !fechaMaterialEquipoValida(m.Foto.HastaExclusivo) ||
			m.Foto.Desde >= m.Foto.HastaExclusivo ||
			m.Desde < m.Foto.Desde || m.Hasta >= m.Foto.HastaExclusivo {
			return nil, ErrMaterialEquipoInformeInvalido
		}
		base["equipo_ref"], base["unidad_ref"], base["organismo_ref"] = m.Foto.EquipoRef, m.Foto.UnidadRef, m.Foto.OrganismoRef
		base["equipo_version_ref"], base["recibo_fuente_ref"] = m.Foto.VersionRef, m.Foto.ReciboFuenteRef
		base["fuente_ref"] = m.Foto.FuenteRef
	}
	return json.Marshal(base)
}

func (m MaterialExportacionCronos) Recurso() (vecdomain.RecursoAutorizable, error) {
	canon, err := m.Canonico()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(canon)
	ambitos := map[string]string{"empleado_ref": m.EmpleadoRef}
	ref, tipo := m.EmpleadoRef, "informe_propio"
	if m.Foto != nil {
		ambitos = map[string]string{"equipo_ref": m.Foto.EquipoRef, "unidad_ref": m.Foto.UnidadRef, "organismo_ref": m.Foto.OrganismoRef}
		ref, tipo = m.Foto.EquipoRef, "informe_equipo"
	}
	r := vecdomain.RecursoAutorizable{
		Referencia: ref, ModuloID: "cronos", Tipo: tipo, Ambitos: ambitos,
		Atributos: map[string]string{
			"material_sha256": hex.EncodeToString(h[:]), "tipo_informe": string(m.Tipo),
			"formato": string(m.Formato), "desde": m.Desde, "hasta": m.Hasta,
			"zona_horaria":    m.ZonaHoraria,
			"idioma":          m.Idioma,
			"politica_sha256": m.PoliticaSHA256, "alcance": m.Alcance,
			"fase": m.Fase,
		},
	}
	if m.Fase == "confirmacion" {
		r.Atributos["preparacion_ref"] = m.PreparacionRef
		r.Atributos["fuente_sha256"] = m.FuenteSHA256
		r.Atributos["documento_sha256"] = m.DocumentoSHA256
		r.Atributos["tamano"] = strconv.FormatInt(m.Tamano, 10)
	}
	if m.Foto != nil {
		r.Atributos["equipo_version_ref"] = m.Foto.VersionRef
	}
	if _, err := r.HuellaContextoAutorizacionSHA256(); err != nil {
		return vecdomain.RecursoAutorizable{}, ErrMaterialEquipoInformeInvalido
	}
	return r, nil
}

func fechaMaterialEquipoValida(v string) bool {
	if len(v) != 10 {
		return false
	}
	f, err := time.Parse("2006-01-02", v)
	return err == nil && f.Format("2006-01-02") == v
}

func cursorEquipoValido(v string) bool {
	if v == "" {
		return true
	}
	for i := 0; i < len(v); i++ {
		if v[i] == '|' {
			return referenciaIdentidadMarcaje(v[:i], "emp_") && fechaMaterialEquipoValida(v[i+1:])
		}
	}
	return false
}

func cursorRecursoEquipo(v string) string {
	if v == "" {
		return "inicio"
	}
	return v
}

func idiomaMaterialEquipoValido(v string) bool {
	if len(v) < 2 || len(v) > 17 {
		return false
	}
	partes := 0
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c == '-' {
			if i == 0 || i == len(v)-1 || partes > 0 {
				return false
			}
			partes++
			continue
		}
		if (c < 'a' || c > 'z') && (partes == 0 || (c < 'A' || c > 'Z') && (c < '0' || c > '9')) {
			return false
		}
	}
	return true
}
