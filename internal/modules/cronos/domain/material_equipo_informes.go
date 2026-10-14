package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrMaterialEquipoInformeInvalido = errors.New("cronos: material de equipo o informe inválido")

type TipoConsultaEquipo string

const (
	ConsultaPresenciaEquipo   TipoConsultaEquipo = "presencia"
	ConsultaIncidenciasEquipo TipoConsultaEquipo = "incidencias"
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
	Desde, Hasta                     string
	ZonaHoraria                      string
}

func (m MaterialConsultaEquipo) Canonico() ([]byte, error) {
	if m.Tipo != ConsultaPresenciaEquipo && m.Tipo != ConsultaIncidenciasEquipo ||
		!referenciaIdentidadMarcaje(m.ActorRef, "per_") ||
		!referenciaIdentidadMarcaje(m.PerfilRef, "prf_") ||
		!referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") ||
		!referenciaMarcaje(m.Foto.EquipoRef) || !referenciaMarcaje(m.Foto.UnidadRef) ||
		!referenciaMarcaje(m.Foto.OrganismoRef) || !referenciaMarcaje(m.Foto.VersionRef) ||
		!referenciaMarcaje(m.Foto.ReciboFuenteRef) || !referenciaMarcaje(m.Foto.FuenteRef) ||
		!fechaMaterialEquipoValida(m.Desde) || !fechaMaterialEquipoValida(m.Hasta) || m.Desde > m.Hasta ||
		m.ZonaHoraria != ZonaSaldoPeninsula && m.ZonaHoraria != ZonaSaldoCanarias {
		return nil, ErrMaterialEquipoInformeInvalido
	}
	return json.Marshal(map[string]string{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"equipo_ref": m.Foto.EquipoRef, "unidad_ref": m.Foto.UnidadRef,
		"organismo_ref": m.Foto.OrganismoRef, "equipo_version_ref": m.Foto.VersionRef,
		"recibo_fuente_ref": m.Foto.ReciboFuenteRef, "fuente_ref": m.Foto.FuenteRef,
		"desde": m.Desde, "hasta": m.Hasta, "vista": string(m.Tipo), "zona_horaria": m.ZonaHoraria,
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
	Tipo                             TipoInformeCronos
	Formato                          FormatoInformeCronos
	Desde, Hasta                     string
	PoliticaRef                      string
	PoliticaVersion                  int64
	PoliticaSHA256                   string
	ClaveOperacion                   string
}

func (m MaterialExportacionCronos) Canonico() ([]byte, error) {
	if !referenciaIdentidadMarcaje(m.ActorRef, "per_") ||
		!referenciaIdentidadMarcaje(m.PerfilRef, "prf_") ||
		!referenciaIdentidadMarcaje(m.EmpleadoRef, "emp_") ||
		(m.Tipo != InformeSaldo && m.Tipo != InformeMovimientos && m.Tipo != InformePermisos) ||
		(m.Formato != InformePDF && m.Formato != InformeCSV) ||
		!fechaMaterialEquipoValida(m.Desde) || !fechaMaterialEquipoValida(m.Hasta) || m.Desde > m.Hasta ||
		!referenciaMarcaje(m.PoliticaRef) || m.PoliticaVersion < 1 ||
		!HuellaEfectosValida(m.PoliticaSHA256) || !claveOperacionMarcaje(m.ClaveOperacion) {
		return nil, ErrMaterialEquipoInformeInvalido
	}
	base := map[string]any{
		"actor_ref": m.ActorRef, "perfil_ref": m.PerfilRef, "empleado_ref": m.EmpleadoRef,
		"tipo_informe": m.Tipo, "formato": m.Formato, "desde": m.Desde, "hasta": m.Hasta,
		"politica_ref": m.PoliticaRef, "politica_version": m.PoliticaVersion,
		"politica_sha256": m.PoliticaSHA256, "clave_operacion": m.ClaveOperacion,
	}
	if m.Foto != nil {
		if !referenciaMarcaje(m.Foto.EquipoRef) || !referenciaMarcaje(m.Foto.UnidadRef) ||
			!referenciaMarcaje(m.Foto.OrganismoRef) || !referenciaMarcaje(m.Foto.VersionRef) ||
			!referenciaMarcaje(m.Foto.ReciboFuenteRef) || !referenciaMarcaje(m.Foto.FuenteRef) {
			return nil, ErrMaterialEquipoInformeInvalido
		}
		base["equipo_ref"], base["unidad_ref"], base["organismo_ref"] = m.Foto.EquipoRef, m.Foto.UnidadRef, m.Foto.OrganismoRef
		base["equipo_version_ref"], base["recibo_fuente_ref"] = m.Foto.VersionRef, m.Foto.ReciboFuenteRef
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
			"politica_sha256": m.PoliticaSHA256,
		},
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
