package domain

import (
	"errors"
	"strings"
	"time"
)

var ErrPlanVersionInscripcionInvalido = errors.New("admin_plan_version_inscripcion_invalido")

type PerfilObjetivoVersionInscripcion string

const (
	PerfilVersionInscripcionExterno  PerfilObjetivoVersionInscripcion = "externo"
	PerfilVersionInscripcionEmpleado PerfilObjetivoVersionInscripcion = "empleado"
	PerfilVersionInscripcionRRHH     PerfilObjetivoVersionInscripcion = "rrhh"
)

const (
	RolInscripcionExterno  = "candidato_bolsa_portal_historial_propio_desarrollo"
	RolInscripcionEmpleado = "empleado_bolsa_inscripcion_desarrollo"
	RolInscripcionRRHH     = "tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo"
)

// PreimagenAsignacionInscripcion nombra una versión histórica exacta. El
// puntero actual se comprueba de nuevo en la transacción de publicación.
type PreimagenAsignacionInscripcion struct {
	AsignacionRef string           `json:"asignacion_ref"`
	HuellaSHA256  string           `json:"huella_sha256"`
	Documento     AsignacionPerfil `json:"documento"`
}

type ProyeccionEmpleadoInscripcion struct {
	ProyeccionRef           string `json:"proyeccion_ref"`
	Version                 uint64 `json:"version"`
	ProcedenciaRef          string `json:"procedencia_ref"`
	ProcedenciaVersion      uint64 `json:"procedencia_version"`
	ProcedenciaHuellaSHA256 string `json:"procedencia_huella_sha256"`
}

// FuenteUnidadInscripcion identifica la revisión de Personal que prueba la
// relación entre la organización del ADMIN y la unidad del AID RRHH.
type FuenteUnidadInscripcion struct {
	Referencia   string `json:"referencia"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

// El almacén externo tiene puntero e historia propios. Una fila de la tabla
// normal no acredita por sí sola el perfil externo de AUT16.
type CambioAsignacionInscripcion struct {
	Almacen            string                          `json:"almacen"`
	Modo               string                          `json:"modo"`
	AsignacionID       string                          `json:"asignacion_id"`
	PrincipalID        string                          `json:"principal_id"`
	PerfilActivoRef    string                          `json:"perfil_activo_ref"`
	Ambitos            []AmbitoPerfil                  `json:"ambitos"`
	VigenteDesde       time.Time                       `json:"vigente_desde"`
	VigenteHasta       time.Time                       `json:"vigente_hasta"`
	CandidatoRef       string                          `json:"candidato_ref,omitempty"`
	EmpleadoRef        string                          `json:"empleado_ref,omitempty"`
	ProyeccionEmpleado *ProyeccionEmpleadoInscripcion  `json:"proyeccion_empleado,omitempty"`
	FuenteUnidad       *FuenteUnidadInscripcion        `json:"fuente_unidad,omitempty"`
	Anterior           *PreimagenAsignacionInscripcion `json:"anterior,omitempty"`
}

// PlanVersionInscripcion selecciona cinco entradas nuevas del catálogo
// central. El material no concede permisos: AUT68 coteja fuente, dos ADMIN,
// versión, asignación y procedencia en la misma transacción que el efecto.
type PlanVersionInscripcion struct {
	Operacion             OperacionGobiernoPerfil           `json:"operacion"`
	PerfilObjetivo        PerfilObjetivoVersionInscripcion  `json:"perfil_objetivo"`
	CatalogoRef           string                            `json:"catalogo_ref"`
	CatalogoVersion       int                               `json:"catalogo_version"`
	CatalogoHuellaSHA256  string                            `json:"catalogo_huella_sha256"`
	VersionRolObjetivoRef string                            `json:"version_rol_objetivo_ref"`
	Base                  *PerfilPublicadoAdministracionV1  `json:"base,omitempty"`
	DefinicionNueva       DefinicionVersionPerfilGobernado  `json:"definicion_nueva"`
	Selecciones           []SeleccionAccionAdministracionV1 `json:"selecciones"`
	Asignaciones          []CambioAsignacionInscripcion     `json:"asignaciones"`
	Motivo                ReferenciaEntradaCatalogo         `json:"motivo"`
	ReferenciaActo        string                            `json:"referencia_acto,omitempty"`
}

func (p PlanVersionInscripcion) HuellaSHA256() (string, error) {
	if p.ValidarEstructura() != nil {
		return "", ErrPlanVersionInscripcionInvalido
	}
	return huellaAutorizacion(p)
}

func (p PlanVersionInscripcion) ValidarEstructura() error {
	rolID, acciones, ok := contratoPerfilInscripcion(p.PerfilObjetivo)
	if !ok || !textoAutorizacionSinComodinSeguro(p.CatalogoRef, 512, false) ||
		p.CatalogoVersion < 1 || !huellaSHA256AutorizacionV3NoNula(p.CatalogoHuellaSHA256) ||
		!ReferenciaMotivoAutorizacionV2Valida(p.Motivo) ||
		!ReferenciaActoAdministracionValida(p.ReferenciaActo) ||
		p.DefinicionNueva.RolID != rolID || p.DefinicionNueva.Version < 1 ||
		!textoAutorizacionSeguro(p.DefinicionNueva.Nombre, 512, true) ||
		p.VersionRolObjetivoRef != p.DefinicionNueva.Referencia() ||
		len(p.Selecciones) != len(acciones) || len(p.Asignaciones) < 1 || len(p.Asignaciones) > 16 {
		return ErrPlanVersionInscripcionInvalido
	}
	if p.PerfilObjetivo == PerfilVersionInscripcionEmpleado {
		if p.Operacion != OperacionCrearPerfilGobernado || p.Base != nil ||
			p.DefinicionNueva.Version != 1 || len(p.Asignaciones) != 1 ||
			len(p.DefinicionNueva.Concesiones) != len(acciones) {
			return ErrPlanVersionInscripcionInvalido
		}
	} else if p.Operacion != OperacionVersionarPerfilGobernado || p.Base == nil ||
		p.Base.Validar() != nil || p.Base.Rol.RolID != rolID ||
		p.Base.Rol.Estado != EstadoVersionRolPublicada ||
		p.Base.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		p.DefinicionNueva.Version != p.Base.Rol.Version+1 ||
		len(p.DefinicionNueva.Concesiones) != len(p.Base.Rol.Concesiones)+len(acciones) ||
		(p.PerfilObjetivo == PerfilVersionInscripcionExterno && len(p.Asignaciones) != 1) {
		return ErrPlanVersionInscripcionInvalido
	}
	if p.Base != nil {
		for i, anterior := range p.Base.Rol.Concesiones {
			if !concesionesPerfilAdministracionIguales(anterior, p.DefinicionNueva.Concesiones[i]) {
				return ErrPlanVersionInscripcionInvalido
			}
		}
	}
	concesionesVistas := make(map[string]struct{}, len(p.DefinicionNueva.Concesiones))
	for _, concesion := range p.DefinicionNueva.Concesiones {
		if concesion.Validar() != nil {
			return ErrPlanVersionInscripcionInvalido
		}
		clave := concesion.ModuloID + "\x00" + concesion.Accion + "\x00" + concesion.TipoRecurso
		if _, repetida := concesionesVistas[clave]; repetida {
			return ErrPlanVersionInscripcionInvalido
		}
		concesionesVistas[clave] = struct{}{}
	}
	indice := len(p.DefinicionNueva.Concesiones) - len(acciones)
	selecciones := make(map[string]struct{}, len(acciones))
	for i, esperado := range acciones {
		concesion := p.DefinicionNueva.Concesiones[indice+i]
		seleccion := p.Selecciones[i]
		if concesion.Validar() != nil || concesion.ModuloID != "bolsa" ||
			concesion.Accion != esperado || !concesionInscripcionExacta(concesion, p.PerfilObjetivo) ||
			!textoAutorizacionSinComodinSeguro(seleccion.EntradaRef, 512, false) ||
			seleccion.EntradaVersion < 1 ||
			!huellaSHA256AutorizacionV3NoNula(seleccion.EntradaHuellaSHA256) {
			return ErrPlanVersionInscripcionInvalido
		}
		if _, repetida := selecciones[seleccion.EntradaRef]; repetida {
			return ErrPlanVersionInscripcionInvalido
		}
		selecciones[seleccion.EntradaRef] = struct{}{}
	}
	vistas := make(map[string]struct{}, len(p.Asignaciones))
	for _, cambio := range p.Asignaciones {
		if !cambioAsignacionInscripcionValido(cambio, p.PerfilObjetivo) {
			return ErrPlanVersionInscripcionInvalido
		}
		if _, repetida := vistas[cambio.AsignacionID]; repetida {
			return ErrPlanVersionInscripcionInvalido
		}
		vistas[cambio.AsignacionID] = struct{}{}
	}
	return nil
}

// ValidarContraCatalogo sólo prepara una propuesta. La fuente sigue siendo
// autoridad de lectura; publicar exige que AUT68 vuelva a resolverla y que
// consuma las dos decisiones ADMIN V3 dentro de la transacción del efecto.
func (p PlanVersionInscripcion) ValidarContraCatalogo(c CatalogoAccionesAdministracionV1, ahora time.Time) error {
	if p.ValidarEstructura() != nil || c.Validar() != nil || !instanteAutorizacionCanonico(ahora) ||
		!vigenteAccionesAdministracionEn(c.VigenteDesde, c.VigenteHasta, ahora) {
		return ErrPlanVersionInscripcionInvalido
	}
	huella, err := c.HuellaSHA256()
	if err != nil || p.CatalogoRef != c.Referencia || p.CatalogoVersion != c.Version ||
		p.CatalogoHuellaSHA256 != huella {
		return ErrPlanVersionInscripcionInvalido
	}
	for _, cambio := range p.Asignaciones {
		if ahora.Before(cambio.VigenteDesde) || !ahora.Before(cambio.VigenteHasta) {
			return ErrPlanVersionInscripcionInvalido
		}
	}
	if p.Base != nil {
		baseEncontrada := false
		for _, publicado := range c.Perfiles {
			if publicado.Rol.RolID != p.Base.Rol.RolID {
				continue
			}
			hr, er := publicado.Rol.HuellaSHA256()
			hb, eb := p.Base.Rol.HuellaSHA256()
			hc, ec := publicado.ControlVigencia.HuellaSHA256()
			hbc, ebc := p.Base.ControlVigencia.HuellaSHA256()
			if er != nil || eb != nil || ec != nil || ebc != nil || hr != hb || hc != hbc ||
				publicado.TipoPerfil != p.Base.TipoPerfil {
				return ErrPlanVersionInscripcionInvalido
			}
			baseEncontrada = true
		}
		if !baseEncontrada {
			return ErrPlanVersionInscripcionInvalido
		}
	} else {
		for _, publicado := range c.Perfiles {
			if publicado.Rol.RolID == p.DefinicionNueva.RolID {
				return ErrPlanVersionInscripcionInvalido
			}
		}
	}
	_, acciones, _ := contratoPerfilInscripcion(p.PerfilObjetivo)
	indice := len(p.DefinicionNueva.Concesiones) - len(acciones)
	for i, seleccion := range p.Selecciones {
		coincidencias := 0
		for _, entrada := range c.Entradas {
			if entrada.Referencia != seleccion.EntradaRef {
				continue
			}
			he, er := entrada.HuellaSHA256()
			if er != nil || entrada.Version != seleccion.EntradaVersion || he != seleccion.EntradaHuellaSHA256 ||
				entrada.ClaseControl != "ordinario" ||
				!vigenteAccionesAdministracionEn(entrada.VigenteDesde, entrada.VigenteHasta, ahora) ||
				!concesionesPerfilAdministracionIguales(entrada.Concesion, p.DefinicionNueva.Concesiones[indice+i]) {
				return ErrPlanVersionInscripcionInvalido
			}
			if p.PerfilObjetivo == PerfilVersionInscripcionEmpleado &&
				!existeParExternoInscripcion(c.Entradas, entrada.Concesion) {
				return ErrPlanVersionInscripcionInvalido
			}
			for _, cambio := range p.Asignaciones {
				if !dimensionesInscripcionIguales(entrada.DimensionesAmbito, cambio.Ambitos) {
					return ErrPlanVersionInscripcionInvalido
				}
			}
			coincidencias++
		}
		if coincidencias != 1 {
			return ErrPlanVersionInscripcionInvalido
		}
	}
	return nil
}

func dimensionesInscripcionIguales(esperadas []string, asignadas []AmbitoPerfil) bool {
	if len(esperadas) != len(asignadas) {
		return false
	}
	vistas := make(map[string]struct{}, len(esperadas))
	for _, clave := range esperadas {
		vistas[clave] = struct{}{}
	}
	for _, ambito := range asignadas {
		if _, existe := vistas[ambito.Clave]; !existe {
			return false
		}
		delete(vistas, ambito.Clave)
	}
	return len(vistas) == 0
}

func contratoPerfilInscripcion(perfil PerfilObjetivoVersionInscripcion) (string, []string, bool) {
	propias := []string{
		"bolsa.inscripcion.convocatorias.listar", "bolsa.inscripcion.convocatoria.consultar",
		"bolsa.inscripcion.propias.listar", "bolsa.inscripcion.propia.consultar",
		"bolsa.inscripcion.presentar",
	}
	switch perfil {
	case PerfilVersionInscripcionExterno:
		return RolInscripcionExterno, propias, true
	case PerfilVersionInscripcionEmpleado:
		return RolInscripcionEmpleado, propias, true
	case PerfilVersionInscripcionRRHH:
		return RolInscripcionRRHH, []string{
			"bolsa.inscripcion.rrhh.convocatorias.listar",
			"bolsa.inscripcion.rrhh.listar", "bolsa.inscripcion.rrhh.consultar",
			"bolsa.inscripcion.rrhh.motivos", "bolsa.inscripcion.rrhh.decidir",
			"bolsa.inscripcion.rrhh.incorporar",
		}, true
	}
	return "", nil, false
}

func concesionInscripcionExacta(c ConcesionRol, perfil PerfilObjetivoVersionInscripcion) bool {
	var finalidad, tipo string
	switch c.Accion {
	case "bolsa.inscripcion.rrhh.convocatorias.listar":
		finalidad, tipo = "consulta_convocatorias_gestion_rrhh", "conjunto_gestion_inscripcion"
	case "bolsa.inscripcion.convocatorias.listar", "bolsa.inscripcion.convocatoria.consultar":
		finalidad, tipo = "consulta_convocatoria_abierta", "convocatoria_inscripcion"
	case "bolsa.inscripcion.propias.listar", "bolsa.inscripcion.propia.consultar":
		finalidad, tipo = "consulta_inscripcion_propia", "solicitud_inscripcion"
	case "bolsa.inscripcion.rrhh.listar", "bolsa.inscripcion.rrhh.consultar":
		finalidad, tipo = "consulta_inscripcion_rrhh", "solicitud_inscripcion"
	case "bolsa.inscripcion.rrhh.motivos":
		finalidad, tipo = "consulta_motivos_inscripcion_rrhh", "motivos_inscripcion"
	case "bolsa.inscripcion.presentar":
		finalidad, tipo = "presentar_inscripcion", "inscripcion_convocatoria"
	case "bolsa.inscripcion.rrhh.decidir":
		finalidad, tipo = "revisar_inscripcion", "solicitud_inscripcion"
	case "bolsa.inscripcion.rrhh.incorporar":
		finalidad, tipo = "incorporar_inscripcion", "solicitud_inscripcion"
	default:
		return false
	}
	if perfil == PerfilVersionInscripcionEmpleado {
		tipo += "_empleado"
	}
	if c.GarantiaMinima != AuthAssuranceHigh || len(c.Finalidades) != 1 ||
		c.Finalidades[0] != finalidad || strings.Contains(c.TipoRecurso, "*") {
		return false
	}
	return c.TipoRecurso == tipo
}

func existeParExternoInscripcion(entradas []EntradaAccionAdministracionV1, empleado ConcesionRol) bool {
	base, ok := strings.CutSuffix(empleado.TipoRecurso, "_empleado")
	if !ok || base == "" {
		return false
	}
	for _, entrada := range entradas {
		externo := entrada.Concesion
		if externo.Accion != empleado.Accion || externo.ModuloID != "bolsa" || externo.TipoRecurso != base ||
			!concesionInscripcionExacta(externo, PerfilVersionInscripcionExterno) ||
			len(entrada.DimensionesAmbito) != 1 || entrada.DimensionesAmbito[0] != "candidato_ref" {
			continue
		}
		return listasPerfilAdministracionIguales(externo.Finalidades, empleado.Finalidades) &&
			listasPerfilAdministracionIguales(externo.CamposPermitidos, empleado.CamposPermitidos) &&
			listasPerfilAdministracionIguales(externo.Obligaciones, empleado.Obligaciones) &&
			externo.GarantiaMinima == empleado.GarantiaMinima
	}
	return false
}

func cambioAsignacionInscripcionValido(c CambioAsignacionInscripcion, perfil PerfilObjetivoVersionInscripcion) bool {
	if !textoAutorizacionSinComodinSeguro(c.AsignacionID, 512, false) ||
		!referenciaOpacaAdministracionPerfiles(c.PrincipalID, "per_") ||
		!referenciaOpacaAdministracionPerfiles(c.PerfilActivoRef, "prf_") ||
		!instanteAutorizacionCanonico(c.VigenteDesde) || !instanteAutorizacionCanonico(c.VigenteHasta) ||
		!c.VigenteHasta.After(c.VigenteDesde) || len(c.Ambitos) < 1 || len(c.Ambitos) > maximoElementosAutorizacion {
		return false
	}
	claves := make(map[string]struct{}, len(c.Ambitos))
	for _, ambito := range c.Ambitos {
		if ambito.Validar() != nil {
			return false
		}
		if _, duplicada := claves[ambito.Clave]; duplicada {
			return false
		}
		claves[ambito.Clave] = struct{}{}
	}
	switch perfil {
	case PerfilVersionInscripcionExterno:
		if c.Almacen != "externo" || c.Modo != "alta" || c.Anterior != nil ||
			!referenciaOpacaAdministracionPerfiles(c.CandidatoRef, "can_") ||
			c.EmpleadoRef != "" || c.ProyeccionEmpleado != nil || c.FuenteUnidad != nil || len(c.Ambitos) != 1 ||
			c.Ambitos[0].Clave != "candidato_ref" || len(c.Ambitos[0].Valores) != 1 ||
			c.Ambitos[0].Valores[0] != c.CandidatoRef {
			return false
		}
	case PerfilVersionInscripcionEmpleado:
		if c.Almacen != "normal" || c.Modo != "alta" || c.Anterior != nil || c.CandidatoRef != "" ||
			c.FuenteUnidad != nil ||
			!referenciaOpacaAdministracionPerfiles(c.EmpleadoRef, "emp_") ||
			c.ProyeccionEmpleado == nil ||
			!referenciaOpacaAdministracionPerfiles(c.ProyeccionEmpleado.ProyeccionRef, "pep_") ||
			c.ProyeccionEmpleado.Version == 0 ||
			!referenciaOpacaAdministracionPerfiles(c.ProyeccionEmpleado.ProcedenciaRef, "prc_") ||
			c.ProyeccionEmpleado.ProcedenciaVersion == 0 ||
			!huellaSHA256AutorizacionV3NoNula(c.ProyeccionEmpleado.ProcedenciaHuellaSHA256) ||
			len(c.Ambitos) != 1 || c.Ambitos[0].Clave != "empleado_ref" ||
			len(c.Ambitos[0].Valores) != 1 || c.Ambitos[0].Valores[0] != c.EmpleadoRef {
			return false
		}
	case PerfilVersionInscripcionRRHH:
		var huellaAnterior string
		if c.Anterior != nil {
			huellaAnterior, _ = c.Anterior.Documento.HuellaSHA256()
		}
		if c.Almacen != "normal" || c.Modo != "avance" || c.Anterior == nil ||
			c.CandidatoRef != "" || c.EmpleadoRef != "" || c.ProyeccionEmpleado != nil ||
			c.FuenteUnidad == nil || !fuenteUnidadInscripcionValida(*c.FuenteUnidad) ||
			c.Anterior.Documento.Validar() != nil || c.Anterior.Documento.Estado != EstadoAsignacionPerfilActiva ||
			c.Anterior.AsignacionRef != c.Anterior.Documento.Referencia() ||
			c.Anterior.Documento.AsignacionID != c.AsignacionID ||
			c.Anterior.Documento.PrincipalID != c.PrincipalID ||
			c.Anterior.Documento.PerfilActivoRef != c.PerfilActivoRef ||
			!c.VigenteDesde.Equal(c.Anterior.Documento.VigenteDesde) ||
			!c.VigenteHasta.Equal(c.Anterior.Documento.VigenteHasta) ||
			!huellaSHA256AutorizacionV3NoNula(c.Anterior.HuellaSHA256) ||
			c.Anterior.HuellaSHA256 != huellaAnterior ||
			!ambitosInscripcionIguales(c.Ambitos, c.Anterior.Documento.Ambitos) {
			return false
		}
	default:
		return false
	}
	return true
}

func fuenteUnidadInscripcionValida(f FuenteUnidadInscripcion) bool {
	if len(f.Referencia) < 3 || len(f.Referencia) > 160 || f.Referencia[0] < 'a' || f.Referencia[0] > 'z' ||
		f.Version < 1 || f.Version > 2147483647 || !huellaSHA256AutorizacionV3NoNula(f.HuellaSHA256) {
		return false
	}
	for _, c := range f.Referencia[1:] {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' && c != ':' && c != '-' {
			return false
		}
	}
	return true
}

func ambitosInscripcionIguales(a, b []AmbitoPerfil) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Clave != b[i].Clave || !listasPerfilAdministracionIguales(a[i].Valores, b[i].Valores) {
			return false
		}
	}
	return true
}
