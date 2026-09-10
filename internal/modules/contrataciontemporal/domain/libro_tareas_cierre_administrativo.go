package domain

import "errors"

const AmbitoLibroTareasCierreAdministrativoEjercicio = "cierre_administrativo_ejercicio_incorporacion_y_primera_anotacion"

type ClaveTareaCierreAdministrativoEjercicio string

const (
	TareaIncorporacionOriginalAcreditada ClaveTareaCierreAdministrativoEjercicio = "incorporacion_original_acreditada"
	TareaPrimeraAnotacionAcreditada      ClaveTareaCierreAdministrativoEjercicio = "primera_anotacion_administrativa_acreditada"
)

type TipoEvidenciaTareaCierreAdministrativoEjercicio string

const (
	EvidenciaIncorporacionOriginal TipoEvidenciaTareaCierreAdministrativoEjercicio = "incorporacion_original"
	EvidenciaPrimeraAnotacion      TipoEvidenciaTareaCierreAdministrativoEjercicio = "primera_anotacion_administrativa"
)

var ErrLibroTareasCierreAdministrativoEjercicioInvalido = errors.New("contratacion temporal: libro de tareas de cierre administrativo del ejercicio invalido")

type TareaLibroCierreAdministrativoEjercicio struct {
	Clave     ClaveTareaCierreAdministrativoEjercicio
	Evidencia TipoEvidenciaTareaCierreAdministrativoEjercicio
}

type PublicacionLibroTareasCierreAdministrativoEjercicio struct {
	Referencia string
	Version    uint64
	Ambito     string
	Tareas     []TareaLibroCierreAdministrativoEjercicio
}

// PublicarLibroTareasCierreAdministrativoEjercicio admite solo el conjunto técnico del ejercicio.
// No representa obligaciones jurídicas ni una interfaz de usuario.
func PublicarLibroTareasCierreAdministrativoEjercicio(p PublicacionLibroTareasCierreAdministrativoEjercicio) (PublicacionLibroTareasCierreAdministrativoEjercicio, error) {
	if p.Validar() != nil {
		return PublicacionLibroTareasCierreAdministrativoEjercicio{}, ErrLibroTareasCierreAdministrativoEjercicioInvalido
	}
	p.Tareas = append([]TareaLibroCierreAdministrativoEjercicio(nil), p.Tareas...)
	return p, nil
}

func (p PublicacionLibroTareasCierreAdministrativoEjercicio) Validar() error {
	if !referenciaValida(p.Referencia) || p.Version == 0 || p.Ambito != AmbitoLibroTareasCierreAdministrativoEjercicio || len(p.Tareas) != 2 {
		return ErrLibroTareasCierreAdministrativoEjercicioInvalido
	}
	esperadas := []TareaLibroCierreAdministrativoEjercicio{{TareaIncorporacionOriginalAcreditada, EvidenciaIncorporacionOriginal}, {TareaPrimeraAnotacionAcreditada, EvidenciaPrimeraAnotacion}}
	for i, esperada := range esperadas {
		if p.Tareas[i] != esperada {
			return ErrLibroTareasCierreAdministrativoEjercicioInvalido
		}
	}
	return nil
}

type EvidenciaTareaCierreAdministrativoEjercicio struct {
	Tipo                        TipoEvidenciaTareaCierreAdministrativoEjercicio
	Referencia                  string
	OrganizacionRef             string
	ExpedienteRef               string
	SeguimientoRef              string
	VersionSeguimientoOriginal  uint64
	HuellaRaizSeguimientoSHA256 string
	VersionEvidencia            uint64
	HuellaEvidenciaSHA256       string
}

func (e EvidenciaTareaCierreAdministrativoEjercicio) validar() error {
	if (e.Tipo != EvidenciaIncorporacionOriginal && e.Tipo != EvidenciaPrimeraAnotacion) || !referenciaValida(e.Referencia) || !referenciaValida(e.OrganizacionRef) || !referenciaValida(e.ExpedienteRef) || !referenciaValida(e.SeguimientoRef) || e.VersionSeguimientoOriginal != 1 || !huellaValida(e.HuellaRaizSeguimientoSHA256) || e.VersionEvidencia == 0 || !huellaValida(e.HuellaEvidenciaSHA256) {
		return ErrLibroTareasCierreAdministrativoEjercicioInvalido
	}
	return nil
}

type EstadoTareaCierreAdministrativoEjercicio struct {
	Clave     ClaveTareaCierreAdministrativoEjercicio
	Pendiente bool
	Evidencia EvidenciaTareaCierreAdministrativoEjercicio
}

type SnapshotTareasCierreAdministrativoEjercicio struct {
	Libro           PublicacionLibroTareasCierreAdministrativoEjercicio
	OrganizacionRef string
	ExpedienteRef   string
	SeguimientoRef  string
	Estados         []EstadoTareaCierreAdministrativoEjercicio
}

// NuevoSnapshotTareasCierreAdministrativoEjercicio solo produce un conjunto completo
// tras acreditar una evidencia única y coherente por cada tarea publicada.
func NuevoSnapshotTareasCierreAdministrativoEjercicio(s SnapshotTareasCierreAdministrativoEjercicio) (SnapshotTareasCierreAdministrativoEjercicio, error) {
	if s.validar() != nil {
		return SnapshotTareasCierreAdministrativoEjercicio{}, ErrLibroTareasCierreAdministrativoEjercicioInvalido
	}
	s.Libro.Tareas = append([]TareaLibroCierreAdministrativoEjercicio(nil), s.Libro.Tareas...)
	s.Estados = append([]EstadoTareaCierreAdministrativoEjercicio(nil), s.Estados...)
	return s, nil
}

func (s SnapshotTareasCierreAdministrativoEjercicio) validar() error {
	if s.Libro.Validar() != nil || !referenciaValida(s.OrganizacionRef) || !referenciaValida(s.ExpedienteRef) || !referenciaValida(s.SeguimientoRef) || len(s.Estados) != len(s.Libro.Tareas) {
		return ErrLibroTareasCierreAdministrativoEjercicioInvalido
	}
	vistos := map[ClaveTareaCierreAdministrativoEjercicio]struct{}{}
	refs := map[string]struct{}{}
	huellas := map[string]struct{}{}
	raiz := ""
	for i, tarea := range s.Libro.Tareas {
		estado := s.Estados[i]
		if estado.Clave != tarea.Clave || estado.Evidencia.Tipo != tarea.Evidencia || estado.Evidencia.validar() != nil || estado.Evidencia.OrganizacionRef != s.OrganizacionRef || estado.Evidencia.ExpedienteRef != s.ExpedienteRef || estado.Evidencia.SeguimientoRef != s.SeguimientoRef {
			return ErrLibroTareasCierreAdministrativoEjercicioInvalido
		}
		if _, ok := vistos[estado.Clave]; ok {
			return ErrLibroTareasCierreAdministrativoEjercicioInvalido
		}
		if _, ok := refs[estado.Evidencia.Referencia]; ok {
			return ErrLibroTareasCierreAdministrativoEjercicioInvalido
		}
		if _, ok := huellas[estado.Evidencia.HuellaEvidenciaSHA256]; ok {
			return ErrLibroTareasCierreAdministrativoEjercicioInvalido
		}
		vistos[estado.Clave] = struct{}{}
		refs[estado.Evidencia.Referencia] = struct{}{}
		huellas[estado.Evidencia.HuellaEvidenciaSHA256] = struct{}{}
		if raiz == "" {
			raiz = estado.Evidencia.HuellaRaizSeguimientoSHA256
		} else if raiz != estado.Evidencia.HuellaRaizSeguimientoSHA256 {
			return ErrLibroTareasCierreAdministrativoEjercicioInvalido
		}
	}
	return nil
}
func (s SnapshotTareasCierreAdministrativoEjercicio) Total() uint32 { return uint32(len(s.Estados)) }
func (s SnapshotTareasCierreAdministrativoEjercicio) Pendientes() uint32 {
	var n uint32
	for _, e := range s.Estados {
		if e.Pendiente {
			n++
		}
	}
	return n
}
