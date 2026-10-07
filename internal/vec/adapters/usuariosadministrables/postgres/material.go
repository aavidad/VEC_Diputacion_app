package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	accionListar    = "administracion.usuarios.listar"
	accionConsultar = "administracion.usuarios.consultar"
	audienciaListar = "vec.admin.usuarios.listar.v1"
	audienciaFicha  = "vec.admin.usuarios.consultar.v1"
	limitePersonas  = 50
)

var (
	referenciaAmbito  = regexp.MustCompile(`^[A-Za-z0-9_:-]{3,128}$`)
	referenciaPersona = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,124}$`)
	referenciaRol     = regexp.MustCompile(`^rol:[A-Za-z0-9_-]+:v[1-9][0-9]*$`)
	ligaduraCursor    = regexp.MustCompile(`^usuarios:([0-9a-f]{64}):(per_[A-Za-z0-9_-]{22,124})$`)
)

type ambito struct {
	OrganizacionRef string
	UnidadRef       string
}

func (a ambito) validar() error {
	if !referenciaAmbito.MatchString(a.OrganizacionRef) || !referenciaAmbito.MatchString(a.UnidadRef) {
		return ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return nil
}

type parAmbito struct {
	OrganizacionRef string `json:"organizacion_ref"`
	UnidadRef       string `json:"unidad_ref"`
}

type filtrosMaterial struct {
	PerfilRef string `json:"perfil_ref"`
	UnidadRef string `json:"unidad_ref"`
	Estado    string `json:"estado"`
}

type materialLista struct {
	Esquema         string          `json:"esquema"`
	OrganizacionRef string          `json:"organizacion_ref"`
	UnidadRef       string          `json:"unidad_ref"`
	ConjuntoRef     string          `json:"conjunto_ref"`
	Filtros         filtrosMaterial `json:"filtros"`
	Cursor          string          `json:"cursor"`
	Limite          int             `json:"limite"`
}

type materialFicha struct {
	Esquema         string `json:"esquema"`
	OrganizacionRef string `json:"organizacion_ref"`
	UnidadRef       string `json:"unidad_ref"`
	ConjuntoRef     string `json:"conjunto_ref"`
	PersonaRef      string `json:"persona_ref"`
}

type peticion struct {
	material    []byte
	recurso     domain.RecursoAutorizable
	accion      string
	audiencia   string
	correlacion string
	decisionRef string
	contextoSHA string
	ambito      ambito
	filtros     ports.FiltrosUsuariosAdministrables
}

func conjuntoUsuarios(a ambito) (string, error) {
	if err := a.validar(); err != nil {
		return "", err
	}
	b, err := json.Marshal(parAmbito{a.OrganizacionRef, a.UnidadRef})
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(append([]byte("vec.admin.conjunto-usuarios.v1\n"), b...))
	return "conjunto_admin:" + hex.EncodeToString(h[:16]), nil
}

func cursorUsuarios(conjunto string, f filtrosMaterial, persona string) (string, error) {
	if !referenciaPersona.MatchString(persona) {
		return "", domain.ErrAutorizacionDenegada
	}
	b, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	material := "vec.admin.cursor-usuarios.v1\n" + conjunto + "\n" + string(b)
	h := sha256.Sum256([]byte(material))
	return "usuarios:" + hex.EncodeToString(h[:]) + ":" + persona, nil
}

func validarFiltros(f ports.FiltrosUsuariosAdministrables) error {
	if f.PerfilRef != "" && (len(f.PerfilRef) > 128 || !referenciaRol.MatchString(f.PerfilRef)) ||
		f.UnidadRef != "" && !referenciaAmbito.MatchString(f.UnidadRef) ||
		f.Estado != "" && f.Estado != "vigente" && f.Estado != "caducado" || len(f.Cursor) > 256 {
		return domain.ErrAutorizacionDenegada
	}
	return nil
}

func materialListar(a ambito, f ports.FiltrosUsuariosAdministrables) (peticion, error) {
	if err := validarFiltros(f); err != nil {
		return peticion{}, err
	}
	conjunto, err := conjuntoUsuarios(a)
	if err != nil {
		return peticion{}, err
	}
	fm := filtrosMaterial{f.PerfilRef, f.UnidadRef, f.Estado}
	if f.Cursor != "" {
		p := ligaduraCursor.FindStringSubmatch(f.Cursor)
		if len(p) != 3 {
			return peticion{}, domain.ErrAutorizacionDenegada
		}
		esperado, err := cursorUsuarios(conjunto, fm, p[2])
		if err != nil || f.Cursor != esperado {
			return peticion{}, domain.ErrAutorizacionDenegada
		}
	}
	b, err := json.Marshal(materialLista{"vec.admin.usuarios.listar.v1", a.OrganizacionRef, a.UnidadRef, conjunto, fm, f.Cursor, limitePersonas})
	if err != nil {
		return peticion{}, err
	}
	p, err := recursoPeticion(b, accionListar, audienciaListar, "conjunto_usuarios", conjunto, a)
	p.filtros = f
	return p, err
}

func materialConsultar(a ambito, persona string) (peticion, error) {
	if !referenciaPersona.MatchString(persona) || len(persona) > 128 {
		return peticion{}, domain.ErrAutorizacionDenegada
	}
	conjunto, err := conjuntoUsuarios(a)
	if err != nil {
		return peticion{}, err
	}
	b, err := json.Marshal(materialFicha{"vec.admin.usuarios.consultar.v1", a.OrganizacionRef, a.UnidadRef, conjunto, persona})
	if err != nil {
		return peticion{}, err
	}
	return recursoPeticion(b, accionConsultar, audienciaFicha, "persona_administrable", persona, a)
}

func recursoPeticion(material []byte, accion, audiencia, tipo, referencia string, a ambito) (peticion, error) {
	h := sha256.Sum256(material)
	r := domain.RecursoAutorizable{Referencia: referencia, ModuloID: "administracion", Tipo: tipo,
		Ambitos:   map[string]string{"organizacion_ref": a.OrganizacionRef, "unidad_ref": a.UnidadRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if err := r.Validar(); err != nil {
		return peticion{}, errors.Join(ports.ErrLecturaUsuariosAdministrablesNoDisponible, err)
	}
	return peticion{material: material, recurso: r, accion: accion, audiencia: audiencia, ambito: a}, nil
}

func cursorRespuestaValido(cursor, conjunto string, f ports.FiltrosUsuariosAdministrables, ultima string) bool {
	if cursor == "" {
		return true
	}
	if ultima == "" {
		return false
	}
	esperado, err := cursorUsuarios(conjunto, filtrosMaterial{f.PerfilRef, f.UnidadRef, f.Estado}, ultima)
	return err == nil && cursor == esperado
}
