package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"

	"vec-diputacion-granada/internal/shared/baremacion"
	vec "vec-diputacion-granada/internal/vec/domain"
)

var ErrFirmantesMerito = errors.New("meritos.error.firmantes_invalidos")

// Clase y efecto clasifican firmas; ninguno concede competencia por si mismo.
type DefinicionRolFirmante struct {
	Clave  string `json:"clave"`
	Clase  string `json:"clase"`
	Efecto string `json:"efecto"`
}

type CatalogoRolesFirmantes struct {
	Referencia   string                  `json:"referencia"`
	Version      uint64                  `json:"version"`
	HuellaSHA256 string                  `json:"huella_sha256"`
	Roles        []DefinicionRolFirmante `json:"roles"`
}

func (c CatalogoRolesFirmantes) Validar() error {
	if !ReferenciaValida(c.Referencia) || c.Version == 0 || !huellaFirmantesValida(c.HuellaSHA256) || len(c.Roles) == 0 || len(c.Roles) > 256 {
		return ErrFirmantesMerito
	}
	vistos := map[string]bool{}
	for _, r := range c.Roles {
		if !ReferenciaValida(r.Clave) || vistos[r.Clave] || !claseFirmanteValida(r.Clase) || !efectoFirmanteValido(r.Efecto) {
			return ErrFirmantesMerito
		}
		if r.Clase == "tecnica" && r.Efecto != "constancia_documental" ||
			r.Clase == "organo" && r.Efecto != "decision_segun_bases" ||
			r.Clase == "observacion" && r.Efecto != "sin_efecto_decisorio" {
			return ErrFirmantesMerito
		}
		vistos[r.Clave] = true
	}
	return nil
}

func claseFirmanteValida(clase string) bool {
	return clase == "tecnica" || clase == "organo" || clase == "observacion"
}

func efectoFirmanteValido(efecto string) bool {
	return efecto == "constancia_documental" || efecto == "decision_segun_bases" || efecto == "sin_efecto_decisorio"
}

// MaterialDecisionFirmantes identifica el contenido preparado por RRHH. La
// huella incluye motivo, puntos y evidencias exactas en su preimagen autoritativa.
// Este valor no evalua el baremo ni cambia un Hecho.
type MaterialDecisionFirmantes struct {
	ConvocatoriaRef   string                    `json:"convocatoria_ref"`
	HechoRef          string                    `json:"hecho_ref"`
	HechoVersion      int                       `json:"hecho_version"`
	DecisionRef       string                    `json:"decision_ref"`
	DecisionVersion   uint64                    `json:"decision_version"`
	ContenidoSHA256   string                    `json:"contenido_sha256"`
	Evidencias        []vec.ReferenciaDocumento `json:"evidencias"`
	EvidenciasSHA256  string                    `json:"evidencias_sha256"`
	MotivoRef         string                    `json:"motivo_ref"`
	PuntosMicropuntos string                    `json:"puntos_micropuntos"`
}

func (m MaterialDecisionFirmantes) Validar() error {
	if (m.ConvocatoriaRef != "" && !ReferenciaValida(m.ConvocatoriaRef)) || !ReferenciaValida(m.HechoRef) || m.HechoVersion < 1 || !ReferenciaValida(m.DecisionRef) || m.DecisionVersion == 0 ||
		!huellaFirmantesValida(m.ContenidoSHA256) || !huellaFirmantesValida(m.EvidenciasSHA256) ||
		!ReferenciaValida(m.MotivoRef) ||
		len(m.Evidencias) == 0 || len(m.Evidencias) > 32 {
		return ErrFirmantesMerito
	}
	if err := validarPuntosCanonicosFirmantes(m.PuntosMicropuntos); err != nil {
		return err
	}
	for i, e := range m.Evidencias {
		if e.Validar() != nil || !ReferenciaValida(e.ID) || i > 0 && (m.Evidencias[i-1].ID > e.ID ||
			m.Evidencias[i-1].ID == e.ID && m.Evidencias[i-1].Version >= e.Version) {
			return ErrFirmantesMerito
		}
	}
	b, err := json.Marshal(m.Evidencias)
	if err != nil {
		return ErrFirmantesMerito
	}
	h := sha256.Sum256(b)
	if m.EvidenciasSHA256 != hex.EncodeToString(h[:]) {
		return ErrFirmantesMerito
	}
	huella, err := HuellaContenidoDecisionFirmantes(m)
	if err != nil || huella != m.ContenidoSHA256 {
		return ErrFirmantesMerito
	}
	return nil
}

// HuellaContenidoDecisionFirmantes liga los campos que RRHH edita, incluida
// la motivacion y los puntos declarados. No acredita que RRHH tenga permiso.
func HuellaContenidoDecisionFirmantes(m MaterialDecisionFirmantes) (string, error) {
	b, err := json.Marshal(struct {
		ConvocatoriaRef   string                    `json:"convocatoria_ref"`
		HechoRef          string                    `json:"hecho_ref"`
		HechoVersion      int                       `json:"hecho_version"`
		DecisionRef       string                    `json:"decision_ref"`
		DecisionVersion   uint64                    `json:"decision_version"`
		Evidencias        []vec.ReferenciaDocumento `json:"evidencias"`
		EvidenciasSHA256  string                    `json:"evidencias_sha256"`
		MotivoRef         string                    `json:"motivo_ref"`
		PuntosMicropuntos string                    `json:"puntos_micropuntos"`
	}{m.ConvocatoriaRef, m.HechoRef, m.HechoVersion, m.DecisionRef, m.DecisionVersion,
		m.Evidencias, m.EvidenciasSHA256, m.MotivoRef, m.PuntosMicropuntos})
	if err != nil {
		return "", ErrFirmantesMerito
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func validarPuntosCanonicosFirmantes(s string) error {
	if len(s) == 0 || len(s) > 16 || len(s) > 1 && s[0] == '0' {
		return ErrFirmantesMerito
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return ErrFirmantesMerito
		}
	}
	valor, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return errors.Join(ErrFirmantesMerito, err)
	}
	_, err = baremacion.PuntosDesdeMicropuntos(valor)
	if err != nil {
		return errors.Join(ErrFirmantesMerito, err)
	}
	return nil
}

func huellaFirmantesValida(s string) bool {
	if len(s) != 64 || s == "0000000000000000000000000000000000000000000000000000000000000000" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return false
		}
	}
	return true
}

// FirmanteDesignado conserva una designacion individual y su orden. Sustituye
// apunta a la designacion anterior, nunca a una afiliacion o rol generico.
type FirmanteDesignado struct {
	PersonaRef         string `json:"persona_ref"`
	RolClave           string `json:"rol_clave"`
	OrganoRef          string `json:"organo_ref"`
	DesignacionRef     string `json:"designacion_ref"`
	DesignacionVersion uint64 `json:"designacion_version"`
	Orden              uint32 `json:"orden"`
	Obligatorio        bool   `json:"obligatorio"`
	SustituyeRef       string `json:"sustituye_ref,omitempty"`
}

// FirmaConstatada solo sirve para ensayar la evolucion pura del roster. El
// caso de uso futuro exigira el dictamen GrxFirma y la custodia antes de crearla.
type FirmaConstatada struct {
	DesignacionRef string `json:"designacion_ref"`
	PersonaRef     string `json:"persona_ref"`
	DocumentoRef   string `json:"documento_ref"`
	Revision       uint64 `json:"revision"`
	HuellaSHA256   string `json:"huella_sha256"`
	Sentido        string `json:"sentido"`
}

type PlanFirmantesMerito struct {
	Material           MaterialDecisionFirmantes `json:"material"`
	CatalogoRef        string                    `json:"catalogo_ref"`
	CatalogoVersion    uint64                    `json:"catalogo_version"`
	CatalogoSHA256     string                    `json:"catalogo_sha256"`
	VersionRoster      uint64                    `json:"version_roster"`
	HuellaRosterSHA256 string                    `json:"huella_roster_sha256"`
	Estado             string                    `json:"estado"`
	Firmantes          []FirmanteDesignado       `json:"firmantes"`
	Firmas             []FirmaConstatada         `json:"firmas"`
}

// PrepararPlanFirmantesMerito congela la lista exacta para la version de
// decision. Cada rol debe estar en el catalogo publicado recibido.
func PrepararPlanFirmantesMerito(m MaterialDecisionFirmantes, c CatalogoRolesFirmantes, version uint64, firmantes []FirmanteDesignado) (PlanFirmantesMerito, error) {
	if m.Validar() != nil || c.Validar() != nil || version == 0 || len(firmantes) == 0 || len(firmantes) > 4096 {
		return PlanFirmantesMerito{}, ErrFirmantesMerito
	}
	roles := map[string]DefinicionRolFirmante{}
	for _, r := range c.Roles {
		roles[r.Clave] = r
	}
	ordenados := append([]FirmanteDesignado(nil), firmantes...)
	sort.Slice(ordenados, func(i, j int) bool { return ordenados[i].Orden < ordenados[j].Orden })
	designaciones, personas := map[string]bool{}, map[string]bool{}
	obligatorios, tecnicosObligatorios := 0, 0
	for i, f := range ordenados {
		if !ReferenciaValida(f.PersonaRef) || !ReferenciaValida(f.OrganoRef) || !ReferenciaValida(f.DesignacionRef) ||
			f.DesignacionVersion == 0 || f.Orden != uint32(i+1) || designaciones[f.DesignacionRef] || personas[f.PersonaRef] ||
			roles[f.RolClave].Clave == "" || f.SustituyeRef != "" && !ReferenciaValida(f.SustituyeRef) {
			return PlanFirmantesMerito{}, ErrFirmantesMerito
		}
		if f.Obligatorio {
			obligatorios++
			if roles[f.RolClave].Clase == "tecnica" {
				tecnicosObligatorios++
			}
		}
		designaciones[f.DesignacionRef], personas[f.PersonaRef] = true, true
	}
	if obligatorios == 0 || tecnicosObligatorios == 0 {
		return PlanFirmantesMerito{}, ErrFirmantesMerito
	}
	material := struct {
		Decision        MaterialDecisionFirmantes `json:"decision"`
		Catalogo        string                    `json:"catalogo"`
		CatalogoVersion uint64                    `json:"catalogo_version"`
		CatalogoSHA256  string                    `json:"catalogo_sha256"`
		Version         uint64                    `json:"version"`
		Firmantes       []FirmanteDesignado       `json:"firmantes"`
	}{m, c.Referencia, c.Version, c.HuellaSHA256, version, ordenados}
	b, err := json.Marshal(material)
	if err != nil {
		return PlanFirmantesMerito{}, ErrFirmantesMerito
	}
	h := sha256.Sum256(b)
	return PlanFirmantesMerito{Material: m, CatalogoRef: c.Referencia, CatalogoVersion: c.Version,
		CatalogoSHA256: c.HuellaSHA256, VersionRoster: version, HuellaRosterSHA256: hex.EncodeToString(h[:]),
		Estado: "preparado_congelado", Firmantes: ordenados, Firmas: []FirmaConstatada{}}, nil
}

// AnotarFirmaConstatada no verifica criptografia ni concede eficacia; el caso
// de uso de firma debe aportar una constancia ya verificada por Documentos.
func (p PlanFirmantesMerito) anotarFirmaConstatada(f FirmaConstatada) (PlanFirmantesMerito, error) {
	if p.Material.Validar() != nil || p.VersionRoster == 0 ||
		(p.Estado != "preparado_congelado" && p.Estado != "en_firma") ||
		!ReferenciaValida(f.DesignacionRef) || !ReferenciaValida(f.PersonaRef) || !ReferenciaValida(f.DocumentoRef) ||
		f.Revision == 0 || !huellaFirmantesValida(f.HuellaSHA256) || !huellaFirmantesValida(p.HuellaRosterSHA256) ||
		f.Sentido != "conformidad" && f.Sentido != "disconformidad" {
		return PlanFirmantesMerito{}, ErrFirmantesMerito
	}
	valido := false
	orden := uint32(0)
	for _, d := range p.Firmantes {
		if d.DesignacionRef == f.DesignacionRef && d.PersonaRef == f.PersonaRef {
			valido = true
			orden = d.Orden
		}
	}
	ultimoOrden := uint32(0)
	for _, anterior := range p.Firmas {
		if anterior.DesignacionRef == f.DesignacionRef {
			return PlanFirmantesMerito{}, ErrFirmantesMerito
		}
		for _, d := range p.Firmantes {
			if d.DesignacionRef == anterior.DesignacionRef && d.Orden > ultimoOrden {
				ultimoOrden = d.Orden
			}
		}
	}
	if !valido || orden <= ultimoOrden {
		return PlanFirmantesMerito{}, ErrFirmantesMerito
	}
	for _, d := range p.Firmantes {
		if d.Obligatorio && d.Orden < orden && d.Orden > ultimoOrden {
			return PlanFirmantesMerito{}, ErrFirmantesMerito
		}
	}
	p.Firmantes = append([]FirmanteDesignado(nil), p.Firmantes...)
	p.Firmas = append(append([]FirmaConstatada(nil), p.Firmas...), f)
	p.Estado = "en_firma"
	return p, nil
}

// TieneDisconformidad informa sin resolver ni publicar ningun acto.
func (p PlanFirmantesMerito) TieneDisconformidad() bool {
	for _, f := range p.Firmas {
		if f.Sentido == "disconformidad" {
			return true
		}
	}
	return false
}

// NuevaVersionContenido deja las firmas previas en la version historica del
// repositorio y crea una nueva lista congelada sin firmas eficaces. Cambiar
// contenido, motivo o puntos exige huella nueva y version sucesora.
func (p PlanFirmantesMerito) NuevaVersionContenido(m MaterialDecisionFirmantes, c CatalogoRolesFirmantes, firmantes []FirmanteDesignado) (PlanFirmantesMerito, error) {
	if p.Material.Validar() != nil || p.VersionRoster == 0 || !huellaFirmantesValida(p.HuellaRosterSHA256) {
		return PlanFirmantesMerito{}, ErrFirmantesMerito
	}
	anterior := p.Material
	anterior.DecisionVersion = m.DecisionVersion
	if m.DecisionRef != p.Material.DecisionRef || m.HechoRef != p.Material.HechoRef ||
		m.ConvocatoriaRef != p.Material.ConvocatoriaRef || m.DecisionVersion != p.Material.DecisionVersion+1 ||
		materialesFirmantesIguales(anterior, m) {
		return PlanFirmantesMerito{}, ErrFirmantesMerito
	}
	nuevo, err := PrepararPlanFirmantesMerito(m, c, p.VersionRoster+1, firmantes)
	if err != nil {
		return PlanFirmantesMerito{}, err
	}
	return nuevo, nil
}

func materialesFirmantesIguales(a, b MaterialDecisionFirmantes) bool {
	x, ex := json.Marshal(a)
	y, ey := json.Marshal(b)
	return ex == nil && ey == nil && string(x) == string(y)
}

// SustituirFirmante crea version nueva de roster, incluso si el contenido no
// cambio. El sustituto requiere nueva designacion del mismo rol y organo; la
// designacion anterior no puede seguir en la lista vigente.
func (p PlanFirmantesMerito) SustituirFirmante(c CatalogoRolesFirmantes, anteriorRef string, sustituto FirmanteDesignado) (PlanFirmantesMerito, error) {
	if p.Material.Validar() != nil || p.VersionRoster == 0 || !huellaFirmantesValida(p.HuellaRosterSHA256) ||
		!ReferenciaValida(anteriorRef) || sustituto.SustituyeRef != anteriorRef || sustituto.DesignacionRef == anteriorRef || p.CatalogoRef != c.Referencia ||
		p.CatalogoVersion != c.Version || p.CatalogoSHA256 != c.HuellaSHA256 {
		return PlanFirmantesMerito{}, ErrFirmantesMerito
	}
	nuevos := append([]FirmanteDesignado(nil), p.Firmantes...)
	indice := -1
	for i, f := range nuevos {
		if f.DesignacionRef == anteriorRef {
			indice = i
			if sustituto.PersonaRef == f.PersonaRef || sustituto.RolClave != f.RolClave || sustituto.OrganoRef != f.OrganoRef ||
				sustituto.Orden != f.Orden || sustituto.Obligatorio != f.Obligatorio {
				return PlanFirmantesMerito{}, ErrFirmantesMerito
			}
		}
	}
	if indice < 0 {
		return PlanFirmantesMerito{}, ErrFirmantesMerito
	}
	nuevos[indice] = sustituto
	return PrepararPlanFirmantesMerito(p.Material, c, p.VersionRoster+1, nuevos)
}
