package domain

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var ErrVersionarRolBolsaInvalido = errors.New("admin_versionar_rol_bolsa_invalido")

const RolIDVersionarBolsaB1 = "tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo"

// SeleccionAsignacionVersionarRolBolsa liga un documento esperado con su
// puntero CAS. Es una preimagen sin autoridad; SQL comprueba la fila real bajo
// bloqueo antes de admitir el acto.
type SeleccionAsignacionVersionarRolBolsa struct {
	AsignacionRef string           `json:"asignacion_ref"`
	HuellaSHA256  string           `json:"huella_sha256"`
	Documento     AsignacionPerfil `json:"documento"`
}

func (s SeleccionAsignacionVersionarRolBolsa) ValidarPara(rolID string, versionMaxima int) error {
	h, err := s.Documento.HuellaSHA256()
	if err != nil || s.AsignacionRef != s.Documento.Referencia() || s.HuellaSHA256 != h ||
		s.Documento.Estado != EstadoAsignacionPerfilActiva ||
		!versionRolAnteriorBolsa(s.Documento.VersionRolRef, rolID, versionMaxima) {
		return ErrVersionarRolBolsaInvalido
	}
	return nil
}

func versionRolAnteriorBolsa(ref, rolID string, maxima int) bool {
	prefijo := "rol:" + rolID + ":v"
	if !strings.HasPrefix(ref, prefijo) {
		return false
	}
	version, err := strconv.Atoi(strings.TrimPrefix(ref, prefijo))
	return err == nil && version > 0 && version <= maxima
}

// PlanVersionarRolBolsa conserva la cabeza publicada y las asignaciones que
// cambiarán en el mismo acto. Las otras asignaciones no forman parte del plan.
type PlanVersionarRolBolsa struct {
	Operacion             OperacionGobiernoPerfil                `json:"operacion"`
	CatalogoRef           string                                 `json:"catalogo_ref"`
	CatalogoVersion       int                                    `json:"catalogo_version"`
	CatalogoHuellaSHA256  string                                 `json:"catalogo_huella_sha256"`
	VersionRolObjetivoRef string                                 `json:"version_rol_objetivo_ref"`
	Base                  PerfilPublicadoAdministracionV1        `json:"base"`
	DefinicionNueva       DefinicionVersionPerfilGobernado       `json:"definicion_nueva"`
	Seleccion             SeleccionAccionAdministracionV1        `json:"seleccion"`
	Asignaciones          []SeleccionAsignacionVersionarRolBolsa `json:"asignaciones"`
	Motivo                ReferenciaEntradaCatalogo              `json:"motivo"`
	ReferenciaActo        string                                 `json:"referencia_acto,omitempty"`
}

func (p PlanVersionarRolBolsa) HuellaSHA256() (string, error) {
	if p.Operacion != OperacionVersionarPerfilGobernado ||
		!textoAutorizacionSinComodinSeguro(p.CatalogoRef, 512, false) || p.CatalogoVersion < 1 ||
		!huellaSHA256AutorizacionV3NoNula(p.CatalogoHuellaSHA256) ||
		p.Base.Validar() != nil || p.Base.TipoPerfil != TipoPerfilAdministracionAdministrableV1 ||
		p.Base.Rol.RolID != RolIDVersionarBolsaB1 || p.Base.Rol.Estado != EstadoVersionRolPublicada ||
		p.Base.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		p.Base.Rol.Version == int(^uint(0)>>1) ||
		p.DefinicionNueva.RolID != p.Base.Rol.RolID || p.DefinicionNueva.Version != p.Base.Rol.Version+1 ||
		p.DefinicionNueva.Nombre != p.Base.Rol.Nombre || p.VersionRolObjetivoRef != p.DefinicionNueva.Referencia() ||
		len(p.DefinicionNueva.Concesiones) != len(p.Base.Rol.Concesiones)+1 ||
		len(p.DefinicionNueva.Concesiones) > maximoElementosAutorizacion ||
		len(p.Asignaciones) < 1 || len(p.Asignaciones) > 16 ||
		!textoAutorizacionSinComodinSeguro(p.Seleccion.EntradaRef, 512, false) ||
		p.Seleccion.EntradaVersion < 1 || !huellaSHA256AutorizacionV3NoNula(p.Seleccion.EntradaHuellaSHA256) ||
		!ReferenciaMotivoAutorizacionV2Valida(p.Motivo) || !ReferenciaActoAdministracionValida(p.ReferenciaActo) {
		return "", ErrVersionarRolBolsaInvalido
	}
	for i, anterior := range p.Base.Rol.Concesiones {
		if !reflect.DeepEqual(anterior, p.DefinicionNueva.Concesiones[i]) {
			return "", ErrVersionarRolBolsaInvalido
		}
		if anterior.ModuloID == "bolsa" && anterior.Accion == "bolsa.carga_convoca.confirmar" &&
			anterior.TipoRecurso == "carga_convoca" {
			return "", ErrVersionarRolBolsaInvalido
		}
	}
	if !concesionCargaConvocaB1(p.DefinicionNueva.Concesiones[len(p.DefinicionNueva.Concesiones)-1]) {
		return "", ErrVersionarRolBolsaInvalido
	}
	vistos := make(map[string]bool, len(p.Asignaciones))
	for _, a := range p.Asignaciones {
		if a.ValidarPara(p.Base.Rol.RolID, p.Base.Rol.Version) != nil || vistos[a.Documento.AsignacionID] {
			return "", ErrVersionarRolBolsaInvalido
		}
		vistos[a.Documento.AsignacionID] = true
	}
	return huellaAutorizacion(p)
}

func concesionCargaConvocaB1(c ConcesionRol) bool {
	return c.Validar() == nil && c.ModuloID == "bolsa" && c.Accion == "bolsa.carga_convoca.confirmar" &&
		c.TipoRecurso == "carga_convoca" && c.GarantiaMinima == AuthAssuranceHigh &&
		len(c.Finalidades) == 1 && c.Finalidades[0] == "carga_bolsa_convoca" &&
		len(c.CamposPermitidos) == 0 && len(c.Obligaciones) == 0
}

// IntencionVersionarRolBolsa señala preimágenes esperadas. Sus asignaciones
// pueden proceder del cliente y NO acreditan su vigencia: AUT63 coteja cada
// documento y huella contra la autoridad central bajo bloqueo. La definición
// de la nueva concesión procede exclusivamente del catálogo gobernado.
type IntencionVersionarRolBolsa struct {
	CatalogoRef          string
	CatalogoVersion      int
	CatalogoHuellaSHA256 string
	BaseRef              string
	BaseHuellaSHA256     string
	ControlRevision      uint64
	ControlHuellaSHA256  string
	Seleccion            SeleccionAccionAdministracionV1
	Asignaciones         []SeleccionAsignacionVersionarRolBolsa
	Motivo               ReferenciaEntradaCatalogo
	ReferenciaActo       string
}

func PrepararPlanVersionarRolBolsa(c CatalogoAccionesAdministracionV1,
	intencion IntencionVersionarRolBolsa, ahora time.Time) (PlanVersionarRolBolsa, error) {
	var vacio PlanVersionarRolBolsa
	hc, err := c.HuellaSHA256()
	if err != nil || !instanteAutorizacionCanonico(ahora) ||
		c.Referencia != intencion.CatalogoRef || c.Version != intencion.CatalogoVersion ||
		hc != intencion.CatalogoHuellaSHA256 ||
		len(intencion.Asignaciones) < 1 || len(intencion.Asignaciones) > 16 {
		return vacio, ErrVersionarRolBolsaInvalido
	}
	var base *PerfilPublicadoAdministracionV1
	for i := range c.Perfiles {
		if c.Perfiles[i].Rol.RolID == RolIDVersionarBolsaB1 {
			base = &c.Perfiles[i]
			break
		}
	}
	if base == nil || base.TipoPerfil != TipoPerfilAdministracionAdministrableV1 ||
		base.Rol.Estado != EstadoVersionRolPublicada || base.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		base.ControlVigencia.ActualizadoEn.After(ahora) || base.Rol.Version == int(^uint(0)>>1) {
		return vacio, ErrVersionarRolBolsaInvalido
	}
	hr, er := base.Rol.HuellaSHA256()
	hcontrol, ec := base.ControlVigencia.HuellaSHA256()
	if er != nil || ec != nil || intencion.BaseRef != base.Rol.Referencia() || intencion.BaseHuellaSHA256 != hr ||
		intencion.ControlRevision != base.ControlVigencia.Revision || intencion.ControlHuellaSHA256 != hcontrol {
		return vacio, ErrVersionarRolBolsaInvalido
	}
	var entrada *EntradaAccionAdministracionV1
	for i := range c.Entradas {
		e := &c.Entradas[i]
		h, eErr := e.HuellaSHA256()
		if eErr == nil && e.Referencia == intencion.Seleccion.EntradaRef &&
			e.Version == intencion.Seleccion.EntradaVersion && h == intencion.Seleccion.EntradaHuellaSHA256 {
			entrada = e
			break
		}
	}
	if entrada == nil || entrada.ClaseControl != string(ClaseControlPerfilOrdinario) ||
		!concesionCargaConvocaB1(entrada.Concesion) {
		return vacio, ErrVersionarRolBolsaInvalido
	}
	for _, anterior := range base.Rol.Concesiones {
		if anterior.ModuloID == entrada.Concesion.ModuloID && anterior.Accion == entrada.Concesion.Accion &&
			anterior.TipoRecurso == entrada.Concesion.TipoRecurso {
			return vacio, ErrVersionarRolBolsaInvalido
		}
	}
	asignaciones := make([]SeleccionAsignacionVersionarRolBolsa, len(intencion.Asignaciones))
	for i, a := range intencion.Asignaciones {
		if a.ValidarPara(base.Rol.RolID, base.Rol.Version) != nil {
			return vacio, ErrVersionarRolBolsaInvalido
		}
		asignaciones[i] = SeleccionAsignacionVersionarRolBolsa{a.AsignacionRef, a.HuellaSHA256,
			copiarAsignacionVersionarBolsa(a.Documento)}
	}
	baseCopia := *base
	baseCopia.Rol.Concesiones = copiarConcesionesVersionarBolsa(base.Rol.Concesiones)
	concesiones := copiarConcesionesVersionarBolsa(base.Rol.Concesiones)
	concesiones = append(concesiones, copiarConcesionesVersionarBolsa([]ConcesionRol{entrada.Concesion})[0])
	plan := PlanVersionarRolBolsa{Operacion: OperacionVersionarPerfilGobernado,
		CatalogoRef: c.Referencia, CatalogoVersion: c.Version, CatalogoHuellaSHA256: hc,
		VersionRolObjetivoRef: (VersionRol{RolID: base.Rol.RolID, Version: base.Rol.Version + 1}).Referencia(),
		Base:                  baseCopia, DefinicionNueva: DefinicionVersionPerfilGobernado{RolID: base.Rol.RolID,
			Version: base.Rol.Version + 1, Nombre: base.Rol.Nombre, Concesiones: concesiones},
		Seleccion: intencion.Seleccion, Motivo: intencion.Motivo, ReferenciaActo: intencion.ReferenciaActo,
		Asignaciones: asignaciones}
	if _, err := plan.HuellaSHA256(); err != nil {
		return vacio, err
	}
	return plan, nil
}

func copiarAsignacionVersionarBolsa(a AsignacionPerfil) AsignacionPerfil {
	a.Ambitos = append([]AmbitoPerfil(nil), a.Ambitos...)
	for i := range a.Ambitos {
		a.Ambitos[i].Valores = append([]string(nil), a.Ambitos[i].Valores...)
	}
	return a
}

func copiarConcesionesVersionarBolsa(original []ConcesionRol) []ConcesionRol {
	copia := make([]ConcesionRol, len(original))
	copy(copia, original)
	for i := range copia {
		copia[i].Finalidades = copiarListaVersionarBolsa(original[i].Finalidades)
		copia[i].CamposPermitidos = copiarListaVersionarBolsa(original[i].CamposPermitidos)
		copia[i].Obligaciones = copiarListaVersionarBolsa(original[i].Obligaciones)
	}
	return copia
}

func copiarListaVersionarBolsa(original []string) []string {
	if original == nil {
		return nil
	}
	copia := make([]string, len(original))
	copy(copia, original)
	return copia
}

func (p PlanVersionarRolBolsa) Copia() PlanVersionarRolBolsa {
	p.Base.Rol.Concesiones = copiarConcesionesVersionarBolsa(p.Base.Rol.Concesiones)
	p.DefinicionNueva.Concesiones = copiarConcesionesVersionarBolsa(p.DefinicionNueva.Concesiones)
	p.Asignaciones = append([]SeleccionAsignacionVersionarRolBolsa(nil), p.Asignaciones...)
	for i := range p.Asignaciones {
		p.Asignaciones[i].Documento = copiarAsignacionVersionarBolsa(p.Asignaciones[i].Documento)
	}
	return p
}
