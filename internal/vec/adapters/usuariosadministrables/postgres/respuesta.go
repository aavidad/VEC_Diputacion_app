package postgres

import (
	"bytes"
	"encoding/json"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

var (
	huella64            = regexp.MustCompile(`^[0-9a-f]{64}$`)
	referenciaAuditoria = regexp.MustCompile(`^aud_v3_[0-9a-f]{32}$`)
	referenciaPerfil    = regexp.MustCompile(`^prf_[A-Za-z0-9_-]{22,124}$`)
)

const versionMaxima = uint64(1<<53 - 1)

func objetoCerrado(bruto []byte, claves ...string) (map[string]json.RawMessage, error) {
	var x map[string]json.RawMessage
	if len(bruto) == 0 || json.Unmarshal(bruto, &x) != nil || len(x) != len(claves) {
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	for _, k := range claves {
		if _, ok := x[k]; !ok {
			return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
		}
	}
	return x, nil
}

func texto(campo json.RawMessage) (string, bool) {
	var x string
	return x, json.Unmarshal(campo, &x) == nil
}

func numero(campo json.RawMessage) (uint64, bool) {
	var x uint64
	return x, json.Unmarshal(campo, &x) == nil && x > 0 && x <= versionMaxima
}

func fecha(campo json.RawMessage) (time.Time, bool) {
	var x time.Time
	return x, json.Unmarshal(campo, &x) == nil && !x.IsZero() && x.Year() >= 1 && x.Year() <= 9999
}

func perfilRespuesta(bruto []byte) (ports.PerfilUsuarioAdministrable, error) {
	var p ports.PerfilUsuarioAdministrable
	x, err := objetoCerrado(bruto, "perfil_ref", "rol_version_ref", "version", "estado", "vigente_desde", "vigente_hasta")
	if err != nil {
		return p, err
	}
	p.PerfilRef, _ = texto(x["perfil_ref"])
	p.RolVersionRef, _ = texto(x["rol_version_ref"])
	p.Version, _ = numero(x["version"])
	p.Estado, _ = texto(x["estado"])
	p.VigenteDesde, _ = fecha(x["vigente_desde"])
	p.VigenteHasta, _ = fecha(x["vigente_hasta"])
	if !referenciaPerfil.MatchString(p.PerfilRef) || !referenciaRol.MatchString(p.RolVersionRef) || p.Version == 0 ||
		p.Estado != "vigente" && p.Estado != "caducado" && p.Estado != "revocado" && p.Estado != "pendiente" ||
		p.VigenteDesde.IsZero() || p.VigenteHasta.IsZero() || !p.VigenteHasta.After(p.VigenteDesde) {
		return ports.PerfilUsuarioAdministrable{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return p, nil
}

func personaRespuesta(bruto []byte, unidad string) (ports.UsuarioAdministrable, error) {
	var p ports.UsuarioAdministrable
	x, err := objetoCerrado(bruto, "persona_ref", "unidad_ref", "denominacion_version", "perfiles")
	if err != nil {
		return p, err
	}
	p.PersonaRef, _ = texto(x["persona_ref"])
	p.UnidadRef, _ = texto(x["unidad_ref"])
	if !referenciaPersona.MatchString(p.PersonaRef) || len(p.PersonaRef) > 128 || p.UnidadRef != unidad {
		return ports.UsuarioAdministrable{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	if !bytes.Equal(x["denominacion_version"], []byte("null")) {
		v, ok := numero(x["denominacion_version"])
		if !ok {
			return ports.UsuarioAdministrable{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
		}
		p.DenominacionVersion = &v
	}
	var perfiles []json.RawMessage
	if json.Unmarshal(x["perfiles"], &perfiles) != nil || len(perfiles) == 0 {
		return ports.UsuarioAdministrable{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	p.Perfiles = make([]ports.PerfilUsuarioAdministrable, 0, len(perfiles))
	previo := ""
	for _, brutoPerfil := range perfiles {
		perfil, err := perfilRespuesta(brutoPerfil)
		if err != nil || previo >= perfil.PerfilRef {
			return ports.UsuarioAdministrable{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
		}
		p.Perfiles = append(p.Perfiles, perfil)
		previo = perfil.PerfilRef
	}
	return p, nil
}

func consumoRespuesta(bruto []byte, decision, efecto, huella string) error {
	x, err := objetoCerrado(bruto, "decision_ref", "efecto_ref", "huella_efecto_sha256", "consumo_huella_sha256", "auditoria_ref", "consumida_en", "consumo_nuevo")
	if err != nil {
		return err
	}
	a, _ := texto(x["decision_ref"])
	b, _ := texto(x["efecto_ref"])
	c, _ := texto(x["huella_efecto_sha256"])
	d, _ := texto(x["consumo_huella_sha256"])
	e, _ := texto(x["auditoria_ref"])
	_, ok := fecha(x["consumida_en"])
	if a != decision || b != efecto || c != huella || !huella64.MatchString(d) || !referenciaAuditoria.MatchString(e) || !ok || !bytes.Equal(x["consumo_nuevo"], []byte("true")) {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return nil
}

func listaRespuesta(bruto []byte, a ambito, f ports.FiltrosUsuariosAdministrables, decision, efecto, huella string) (ports.PaginaUsuariosAdministrables, error) {
	var pagina ports.PaginaUsuariosAdministrables
	x, err := objetoCerrado(bruto, "datos", "consumo")
	if err != nil || consumoRespuesta(x["consumo"], decision, efecto, huella) != nil {
		return pagina, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	datos, err := objetoCerrado(x["datos"], "personas", "siguiente_cursor")
	if err != nil {
		return pagina, err
	}
	var personas []json.RawMessage
	if json.Unmarshal(datos["personas"], &personas) != nil || personas == nil || len(personas) > limitePersonas {
		return pagina, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	pagina.Personas = make([]ports.UsuarioAdministrable, 0, len(personas))
	previo := ""
	for _, brutoPersona := range personas {
		p, err := personaRespuesta(brutoPersona, a.UnidadRef)
		if err != nil || previo >= p.PersonaRef {
			return ports.PaginaUsuariosAdministrables{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
		}
		pagina.Personas = append(pagina.Personas, p)
		previo = p.PersonaRef
	}
	var ok bool
	pagina.SiguienteCursor, ok = texto(datos["siguiente_cursor"])
	if !ok {
		return ports.PaginaUsuariosAdministrables{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	conjunto, err := conjuntoUsuarios(a)
	if err != nil || !cursorRespuestaValido(pagina.SiguienteCursor, conjunto, f, previo) || pagina.SiguienteCursor != "" && (len(personas) != limitePersonas || pagina.SiguienteCursor == f.Cursor) {
		return ports.PaginaUsuariosAdministrables{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return pagina, nil
}

func fichaRespuesta(bruto []byte, a ambito, persona, decision, efecto, huella string) (*ports.UsuarioAdministrable, error) {
	x, err := objetoCerrado(bruto, "datos", "consumo")
	if err != nil || consumoRespuesta(x["consumo"], decision, efecto, huella) != nil {
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	if bytes.Equal(x["datos"], []byte("null")) {
		return nil, nil
	}
	p, err := personaRespuesta(x["datos"], a.UnidadRef)
	if err != nil || p.PersonaRef != persona {
		return nil, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return &p, nil
}
