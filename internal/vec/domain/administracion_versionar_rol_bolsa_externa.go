package domain

import (
	"errors"
	"reflect"
	"slices"
	"time"
)

var ErrVersionarRolBolsaExternaInvalido = errors.New("admin_versionar_rol_bolsa_externa_invalido")

// El orden de las dos concesiones pertenece al plan firmado por dos ADMIN.
const RolIDVersionarBolsaExterna = "candidato_bolsa_portal_historial_propio_desarrollo"

var accionesVersionarRolBolsaExterna = [...]string{
	"bolsa.preferencias_bolsas.consultar_propias",
	"bolsa.preferencias_bolsas.ordenar_propias",
}

var accionesInscripcionBolsaExterna = [...]string{
	"bolsa.inscripcion.convocatorias.listar", "bolsa.inscripcion.convocatoria.consultar",
	"bolsa.inscripcion.propias.listar", "bolsa.inscripcion.propia.consultar", "bolsa.inscripcion.presentar",
}

var accionesAlegacionesBolsaExterna = [...]string{
	"bolsa.alegacion.propias.consultar", "bolsa.inscripcion.alegacion.presentar",
}

func accionesDeltaBolsaExterna(rol VersionRol, selecciones int) []string {
	switch estadoRolBolsaExterna(rol) {
	case "base":
		if selecciones == 2 {
			return accionesVersionarRolBolsaExterna[:]
		}
		if selecciones == 5 {
			return accionesInscripcionBolsaExterna[:]
		}
	case "i":
		if selecciones == 2 {
			return accionesVersionarRolBolsaExterna[:]
		}
	case "p8", "p10":
		if selecciones == 5 {
			return accionesInscripcionBolsaExterna[:]
		}
	case "pi8", "pi10", "ip8", "ip10":
		if selecciones == 2 {
			return accionesAlegacionesBolsaExterna[:]
		}
	}
	return nil
}

// El mismo conjunto finito de cabezas que AUT104; ningún tamaño por sí solo acredita una historia.
func estadoRolBolsaExterna(rol VersionRol) string {
	const baseLen = 7
	if rol.RolID != RolIDVersionarBolsaExterna || len(rol.Concesiones) < baseLen ||
		!proyeccionInscripcionUnicaBolsaExterna(rol.Concesiones) {
		return ""
	}
	base := rol
	base.Version, base.Concesiones = 1, rol.Concesiones[:baseLen]
	// Como AUT104, las siete concesiones de la v1 forman un conjunto exacto:
	// la v1 publicada empieza por participaciones_propias.consultar. Lo
	// añadido después sí conserva el orden de las altas gobernadas.
	if !baseVersionarRolBolsaExterna(base) {
		return ""
	}
	resto := rol.Concesiones[baseLen:]
	if len(resto) == 0 && rol.Version == 1 {
		return "base"
	}
	if tramosRolBolsaExterna(resto, accionesInscripcionBolsaExterna[:]) && rol.Version == 2 {
		return "i"
	}
	for _, orden := range []struct {
		nombre  string
		tramos  [][]string
		version int
	}{
		{"p", [][]string{accionesVersionarRolBolsaExterna[:]}, 2},
		{"pi", [][]string{accionesVersionarRolBolsaExterna[:], accionesInscripcionBolsaExterna[:]}, 3},
		{"ip", [][]string{accionesInscripcionBolsaExterna[:], accionesVersionarRolBolsaExterna[:]}, 3},
		{"pia", [][]string{accionesVersionarRolBolsaExterna[:], accionesInscripcionBolsaExterna[:], accionesAlegacionesBolsaExterna[:]}, 4},
		{"ipa", [][]string{accionesInscripcionBolsaExterna[:], accionesVersionarRolBolsaExterna[:], accionesAlegacionesBolsaExterna[:]}, 4},
	} {
		if !tramosRolBolsaExterna(resto, orden.tramos...) {
			continue
		}
		indice := 0
		for _, c := range rol.Concesiones {
			if c.Accion == accionesVersionarRolBolsaExterna[0] {
				break
			}
			indice++
		}
		campos := rol.Concesiones[indice].CamposPermitidos
		if slices.Equal(campos, camposPreferenciasBolsaExterna(accionesVersionarRolBolsaExterna[0])) && rol.Version == orden.version {
			return orden.nombre + "8"
		}
		if slices.Equal(campos, camposPreferenciasModalidadesBolsaExterna()) &&
			(rol.Version == orden.version || rol.Version == orden.version+1) {
			return orden.nombre + "10"
		}
	}
	return ""
}

func tramosRolBolsaExterna(concesiones []ConcesionRol, tramos ...[]string) bool {
	indice := 0
	for _, tramo := range tramos {
		for _, accion := range tramo {
			if indice >= len(concesiones) || !concesionDeltaBolsaExterna(concesiones[indice], accion) {
				return false
			}
			indice++
		}
	}
	return indice == len(concesiones)
}

var accionesBaseVersionarRolBolsaExterna = [...]string{
	"bolsa.historial_propio.consultar",
	"bolsa.participaciones_propias.consultar",
	"bolsa.participaciones_propias.solicitar_pausa",
	"bolsa.participaciones_propias.solicitar_reactivacion",
	"bolsa.participaciones_propias.responder_llamamiento",
	"bolsa.participaciones_propias.manifestar_disposicion",
	"bolsa.participaciones_propias.confirmar_contacto",
}

func baseVersionarRolBolsaExterna(rol VersionRol) bool {
	if rol.RolID != RolIDVersionarBolsaExterna || rol.Version != 1 ||
		len(rol.Concesiones) != len(accionesBaseVersionarRolBolsaExterna) {
		return false
	}
	vistos := make(map[string]bool, len(rol.Concesiones))
	for _, c := range rol.Concesiones {
		if c.Validar() != nil || c.ModuloID != "bolsa" || c.GarantiaMinima != AuthAssuranceHigh ||
			!slices.Contains(accionesBaseVersionarRolBolsaExterna[:], c.Accion) || vistos[c.Accion] ||
			len(c.Obligaciones) != 0 {
			return false
		}
		recurso, finalidad := "participaciones_candidato", "gestion_participaciones_propias"
		var campos []string
		switch c.Accion {
		case "bolsa.historial_propio.consultar":
			finalidad = "consulta_historial_propio"
			campos = []string{"contratos_propios", "llamamientos_propios", "renuncias_propias"}
		case "bolsa.participaciones_propias.consultar":
			finalidad = "consulta_participaciones_propias"
			campos = []string{"participaciones_candidato_minimizadas"}
		case "bolsa.participaciones_propias.manifestar_disposicion":
			recurso = "oferta_bolsa"
		}
		if c.TipoRecurso != recurso || !slices.Equal(c.Finalidades, []string{finalidad}) ||
			!slices.Equal(c.CamposPermitidos, campos) {
			return false
		}
		vistos[c.Accion] = true
	}
	return true
}

// IntencionVersionarRolBolsaExterna sólo identifica la cabeza, el catálogo y
// preimágenes esperadas. No acredita publicación, identidad ni vigencia SQL.
type IntencionVersionarRolBolsaExterna struct {
	CatalogoRef          string
	CatalogoVersion      int
	CatalogoHuellaSHA256 string
	BaseRef              string
	BaseHuellaSHA256     string
	ControlRevision      uint64
	ControlHuellaSHA256  string
	Selecciones          []SeleccionAccionAdministracionV1
	Asignaciones         []SeleccionAsignacionVersionarRolBolsa
	Motivo               ReferenciaEntradaCatalogo
	ReferenciaActo       string
}

// PlanVersionarRolBolsaExterna conserva la cabeza v1 y las preimágenes CAS
// de asignaciones externas. El SQL coteja la fuente antes de publicar v2.
type PlanVersionarRolBolsaExterna struct {
	Operacion             OperacionGobiernoPerfil                `json:"operacion"`
	CatalogoRef           string                                 `json:"catalogo_ref"`
	CatalogoVersion       int                                    `json:"catalogo_version"`
	CatalogoHuellaSHA256  string                                 `json:"catalogo_huella_sha256"`
	VersionRolObjetivoRef string                                 `json:"version_rol_objetivo_ref"`
	Base                  PerfilPublicadoAdministracionV1        `json:"base"`
	DefinicionNueva       DefinicionVersionPerfilGobernado       `json:"definicion_nueva"`
	Selecciones           []SeleccionAccionAdministracionV1      `json:"selecciones"`
	Asignaciones          []SeleccionAsignacionVersionarRolBolsa `json:"asignaciones"`
	Motivo                ReferenciaEntradaCatalogo              `json:"motivo"`
	ReferenciaActo        string                                 `json:"referencia_acto,omitempty"`
}

func (p PlanVersionarRolBolsaExterna) HuellaSHA256() (string, error) {
	acciones := accionesDeltaBolsaExterna(p.Base.Rol, len(p.Selecciones))
	estado := estadoRolBolsaExterna(p.Base.Rol)
	reemplazo := slices.Contains([]string{"p8", "pi8", "ip8", "pia8", "ipa8"}, estado) &&
		seleccionModalidadesBolsaExterna(p.Selecciones) &&
		sustitucionModalidadesBolsaExterna(p.Base.Rol.Concesiones, p.DefinicionNueva.Concesiones)
	if p.Operacion != OperacionVersionarPerfilGobernado ||
		!textoAutorizacionSinComodinSeguro(p.CatalogoRef, 512, false) || p.CatalogoVersion < 1 ||
		!huellaSHA256AutorizacionV3NoNula(p.CatalogoHuellaSHA256) ||
		p.Base.Validar() != nil || p.Base.TipoPerfil != TipoPerfilAdministracionAdministrableV1 ||
		(len(acciones) == 0 && !reemplazo) || p.Base.Rol.Estado != EstadoVersionRolPublicada ||
		p.Base.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		p.DefinicionNueva.RolID != p.Base.Rol.RolID || p.DefinicionNueva.Version != p.Base.Rol.Version+1 ||
		p.DefinicionNueva.Nombre != p.Base.Rol.Nombre || p.VersionRolObjetivoRef != p.DefinicionNueva.Referencia() ||
		(!reemplazo && len(p.DefinicionNueva.Concesiones) != len(p.Base.Rol.Concesiones)+len(acciones)) ||
		len(p.DefinicionNueva.Concesiones) > maximoElementosAutorizacion ||
		(!reemplazo && len(p.Selecciones) != len(acciones)) ||
		len(p.Asignaciones) < 1 || len(p.Asignaciones) > 16 ||
		!ReferenciaMotivoAutorizacionV2Valida(p.Motivo) || !ReferenciaActoAdministracionValida(p.ReferenciaActo) ||
		!proyeccionInscripcionUnicaBolsaExterna(p.DefinicionNueva.Concesiones) {
		return "", ErrVersionarRolBolsaExternaInvalido
	}
	if !reemplazo {
		for i, anterior := range p.Base.Rol.Concesiones {
			if !reflect.DeepEqual(anterior, p.DefinicionNueva.Concesiones[i]) ||
				slices.Contains(acciones, anterior.Accion) {
				return "", ErrVersionarRolBolsaExternaInvalido
			}
		}
	}
	if reemplazo {
		acciones = accionesVersionarRolBolsaExterna[:]
	}
	for i, accion := range acciones {
		s := p.Selecciones[i]
		if !textoAutorizacionSinComodinSeguro(s.EntradaRef, 512, false) || s.EntradaVersion < 1 ||
			!huellaSHA256AutorizacionV3NoNula(s.EntradaHuellaSHA256) ||
			(!reemplazo && !concesionDeltaBolsaExterna(p.DefinicionNueva.Concesiones[len(p.Base.Rol.Concesiones)+i], accion)) {
			return "", ErrVersionarRolBolsaExternaInvalido
		}
	}
	vistas := make(map[string]bool, len(p.Asignaciones))
	for _, a := range p.Asignaciones {
		if a.ValidarPara(p.Base.Rol.RolID, p.Base.Rol.Version) != nil ||
			!ambitosCandidatoBolsaExterna(a.Documento.Ambitos) || vistas[a.Documento.AsignacionID] {
			return "", ErrVersionarRolBolsaExternaInvalido
		}
		vistas[a.Documento.AsignacionID] = true
	}
	return huellaAutorizacion(p)
}

func ambitosCandidatoBolsaExterna(ambitos []AmbitoPerfil) bool {
	return len(ambitos) == 1 && ambitos[0].Clave == "candidato_ref" &&
		len(ambitos[0].Valores) == 1 && ambitos[0].Validar() == nil
}

func dimensionesCandidatoBolsaExterna(dimensiones []string) bool {
	return len(dimensiones) == 1 && dimensiones[0] == "candidato_ref"
}

func camposPreferenciasBolsaExterna(accion string) []string {
	if accion == accionesVersionarRolBolsaExterna[0] {
		return []string{"version", "orden_guardado", "bolsas_actuales", "bolsas_nuevas",
			"bolsas_retiradas", "revision_requerida", "recibo_ref", "registrado_en"}
	}
	return nil
}

func camposPreferenciasModalidadesBolsaExterna() []string {
	return []string{"version", "orden_guardado", "bolsas_actuales", "bolsas_nuevas",
		"bolsas_retiradas", "revision_requerida", "modalidades_guardadas",
		"modalidades_disponibles", "recibo_ref", "registrado_en"}
}

func indiceConsultaPreferenciasBolsaExterna(concesiones []ConcesionRol) int {
	for j, c := range concesiones {
		if c.Accion == accionesVersionarRolBolsaExterna[0] {
			return j
		}
	}
	return -1
}

func sustitucionModalidadesBolsaExterna(base, nueva []ConcesionRol) bool {
	if len(base) != len(nueva) {
		return false
	}
	indice := indiceConsultaPreferenciasBolsaExterna(base)
	if indice < 0 || !slices.Equal(base[indice].CamposPermitidos, camposPreferenciasBolsaExterna(accionesVersionarRolBolsaExterna[0])) {
		return false
	}
	for j := range base {
		if j == indice {
			esperada := base[j]
			esperada.CamposPermitidos = camposPreferenciasModalidadesBolsaExterna()
			if !reflect.DeepEqual(esperada, nueva[j]) {
				return false
			}
		} else if !reflect.DeepEqual(base[j], nueva[j]) {
			return false
		}
	}
	return true
}

func seleccionModalidadesBolsaExterna(selecciones []SeleccionAccionAdministracionV1) bool {
	return len(selecciones) == 2 &&
		selecciones[0].EntradaRef == "accion:"+accionesVersionarRolBolsaExterna[0] &&
		selecciones[1].EntradaRef == "accion:"+accionesVersionarRolBolsaExterna[1] &&
		selecciones[0].EntradaVersion == 2 && selecciones[1].EntradaVersion == 2
}

func concesionVersionarRolBolsaExterna(c ConcesionRol, accion string) bool {
	if c.Validar() != nil || c.ModuloID != "bolsa" || c.Accion != accion ||
		c.TipoRecurso != "participaciones_candidato" || c.GarantiaMinima != AuthAssuranceHigh ||
		!slices.Equal(c.Finalidades, []string{"gestion_preferencias_bolsas_propias"}) ||
		!(slices.Equal(c.CamposPermitidos, camposPreferenciasBolsaExterna(accion)) ||
			(accion == accionesVersionarRolBolsaExterna[0] && slices.Equal(c.CamposPermitidos, camposPreferenciasModalidadesBolsaExterna()))) {
		return false
	}
	switch accion {
	case accionesVersionarRolBolsaExterna[0]:
		return len(c.Obligaciones) == 0 || slices.Equal(c.Obligaciones, []string{"auditar"})
	case accionesVersionarRolBolsaExterna[1]:
		return len(c.Obligaciones) == 0
	default:
		return false
	}
}

func camposInscripcionBolsaExterna(accion string) []string {
	requisito := []string{"codigo", "descripcion", "estado", "hito_cumplimiento", "hito_etiqueta",
		"hito_fecha", "motivo_codigo", "motivo_etiqueta", "obligatorio", "procedencia_ref"}
	convocatoria := []string{"catalogo_version", "categorias[].categoria", "categorias[].categoria_ref",
		"convocatoria_ref", "estado_solicitud_propia", "impedimento_etiqueta", "numero_categorias",
		"plazo_fin", "plazo_inicio", "puede_iniciar", "requisitos_resumen", "solicitud_ref", "titulo"}
	for _, hoja := range requisito {
		convocatoria = append(convocatoria, "requisitos[]."+hoja)
	}
	solicitud := []string{"bases_ref", "bolsa_ref", "catalogo_version", "categoria", "categoria_ref",
		"convocatoria_ref", "convocatoria_titulo", "decidida_en", "decision_ref", "declaracion_ref",
		"estado", "motivo_codigo", "motivo_etiqueta", "participacion_ref", "plazo_fin", "plazo_inicio",
		"recibo_ref", "registrada_en", "solicitud_ref", "version"}
	for _, hoja := range requisito {
		solicitud = append(solicitud, "requisitos[]."+hoja)
	}
	var campos []string
	switch accion {
	case accionesInscripcionBolsaExterna[0]:
		campos = []string{"cursor_siguiente", "total"}
		for _, hoja := range convocatoria {
			campos = append(campos, "convocatorias[]."+hoja)
		}
	case accionesInscripcionBolsaExterna[1]:
		campos = convocatoria
	case accionesInscripcionBolsaExterna[2]:
		campos = []string{"cursor_siguiente", "total"}
		for _, hoja := range solicitud {
			campos = append(campos, "solicitudes[]."+hoja)
		}
	case accionesInscripcionBolsaExterna[3]:
		campos = solicitud
	case accionesInscripcionBolsaExterna[4]:
		return nil
	default:
		return nil
	}
	slices.Sort(campos)
	return campos
}

// camposInscripcionBolsaExternaV2 es la proyección de la fuente v2 de
// inscripción: la lista y la ficha de convocatorias añaden la modalidad
// (F1-40) y las solicitudes propias por categoría (F1-41). Las demás
// acciones conservan los campos de la v1. AUT105 fija las mismas hojas.
func camposInscripcionBolsaExternaV2(accion string) []string {
	campos := camposInscripcionBolsaExterna(accion)
	var prefijo string
	switch accion {
	case accionesInscripcionBolsaExterna[0]:
		prefijo = "convocatorias[]."
	case accionesInscripcionBolsaExterna[1]:
	default:
		return campos
	}
	for _, hoja := range []string{"modalidad_inscripcion", "solicitudes_propias[].categoria_ref",
		"solicitudes_propias[].estado", "solicitudes_propias[].solicitud_ref"} {
		campos = append(campos, prefijo+hoja)
	}
	slices.Sort(campos)
	return campos
}

// proyeccionInscripcionUnicaBolsaExterna exige que la lista y la ficha de
// convocatorias salgan de la misma versión de la fuente de inscripción: las
// dos con los campos de la v1 o las dos con los de la v2. Las otras tres
// acciones de inscripción tienen los mismos campos en ambas versiones. AUT105
// aplica la misma regla al publicar.
func proyeccionInscripcionUnicaBolsaExterna(concesiones []ConcesionRol) bool {
	v2 := map[bool]bool{}
	for _, c := range concesiones {
		if c.Accion == accionesInscripcionBolsaExterna[0] || c.Accion == accionesInscripcionBolsaExterna[1] {
			v2[slices.Equal(c.CamposPermitidos, camposInscripcionBolsaExternaV2(c.Accion))] = true
		}
	}
	return len(v2) < 2
}

func camposAlegacionesBolsaExterna() []string {
	campos := []string{"total", "cursor_siguiente", "objetos[].tipo", "objetos[].solicitud_ref",
		"objetos[].participacion_ref", "objetos[].acto_ref", "objetos[].version_acto", "objetos[].titulo",
		"objetos[].estado", "objetos[].ocurrido_en", "objetos[].convocatoria_ref", "objetos[].bolsa_ref"}
	for _, hoja := range []string{"alegacion_ref", "objeto.tipo", "objeto.solicitud_ref", "objeto.participacion_ref",
		"objeto.acto_ref", "objeto.version_acto", "fundamento", "estado", "presentada_en", "recibo_ref", "auditoria_ref"} {
		campos = append(campos, "alegaciones[]."+hoja)
	}
	slices.Sort(campos)
	return campos
}

func concesionDeltaBolsaExterna(c ConcesionRol, accion string) bool {
	if slices.Contains(accionesAlegacionesBolsaExterna[:], accion) {
		if c.Validar() != nil || c.ModuloID != "bolsa" || c.Accion != accion ||
			c.GarantiaMinima != AuthAssuranceHigh || len(c.Obligaciones) != 0 {
			return false
		}
		if accion == accionesAlegacionesBolsaExterna[0] {
			return c.TipoRecurso == "alegaciones_propias" &&
				slices.Equal(c.Finalidades, []string{"consulta_alegaciones_propias"}) &&
				slices.Equal(c.CamposPermitidos, camposAlegacionesBolsaExterna())
		}
		return c.TipoRecurso == "alegacion_bolsa" &&
			slices.Equal(c.Finalidades, []string{"presentacion_alegacion_propia"}) && len(c.CamposPermitidos) == 0
	}
	if !slices.Contains(accionesInscripcionBolsaExterna[:], accion) {
		return concesionVersionarRolBolsaExterna(c, accion)
	}
	if c.Validar() != nil || c.ModuloID != "bolsa" || c.Accion != accion ||
		c.GarantiaMinima != AuthAssuranceHigh || len(c.Obligaciones) != 0 ||
		!(slices.Equal(c.CamposPermitidos, camposInscripcionBolsaExterna(accion)) ||
			slices.Equal(c.CamposPermitidos, camposInscripcionBolsaExternaV2(accion))) {
		return false
	}
	switch accion {
	case accionesInscripcionBolsaExterna[0], accionesInscripcionBolsaExterna[1]:
		return c.TipoRecurso == "convocatoria_inscripcion" && slices.Equal(c.Finalidades, []string{"consulta_convocatoria_abierta"})
	case accionesInscripcionBolsaExterna[2], accionesInscripcionBolsaExterna[3]:
		return c.TipoRecurso == "solicitud_inscripcion" && slices.Equal(c.Finalidades, []string{"consulta_inscripcion_propia"})
	case accionesInscripcionBolsaExterna[4]:
		return c.TipoRecurso == "inscripcion_convocatoria" && slices.Equal(c.Finalidades, []string{"presentar_inscripcion"})
	}
	return false
}

func PrepararPlanVersionarRolBolsaExterna(c CatalogoAccionesAdministracionV1,
	base PerfilPublicadoAdministracionV1, i IntencionVersionarRolBolsaExterna, ahora time.Time) (PlanVersionarRolBolsaExterna, error) {
	return prepararPlanVersionarRolBolsaExterna(c, base, i, ahora, true)
}

// ReconstruirPlanVersionarRolBolsaExterna coteja el catálogo histórico exacto
// para recuperar una propuesta. AUT77 exige vigencia al crearla por primera vez
// y vuelve a comprobar la autorización actual incluso en un replay.
func ReconstruirPlanVersionarRolBolsaExterna(c CatalogoAccionesAdministracionV1,
	base PerfilPublicadoAdministracionV1, i IntencionVersionarRolBolsaExterna, ahora time.Time) (PlanVersionarRolBolsaExterna, error) {
	return prepararPlanVersionarRolBolsaExterna(c, base, i, ahora, false)
}

func prepararPlanVersionarRolBolsaExterna(c CatalogoAccionesAdministracionV1,
	base PerfilPublicadoAdministracionV1, i IntencionVersionarRolBolsaExterna, ahora time.Time,
	exigirVigencia bool) (PlanVersionarRolBolsaExterna, error) {
	var vacio PlanVersionarRolBolsaExterna
	estado := estadoRolBolsaExterna(base.Rol)
	reemplazo := slices.Contains([]string{"p8", "pi8", "ip8", "pia8", "ipa8"}, estado) &&
		seleccionModalidadesBolsaExterna(i.Selecciones)
	acciones := accionesDeltaBolsaExterna(base.Rol, len(i.Selecciones))
	if reemplazo {
		acciones = accionesVersionarRolBolsaExterna[:]
	}
	hc, err := c.HuellaSHA256()
	if err != nil || !instanteAutorizacionCanonico(ahora) ||
		(exigirVigencia && !vigenteAccionesAdministracionEn(c.VigenteDesde, c.VigenteHasta, ahora)) ||
		c.Referencia != i.CatalogoRef || c.Version != i.CatalogoVersion || hc != i.CatalogoHuellaSHA256 ||
		(len(i.Selecciones) != len(accionesVersionarRolBolsaExterna) &&
			len(i.Selecciones) != len(accionesInscripcionBolsaExterna)) ||
		len(i.Asignaciones) < 1 || len(i.Asignaciones) > 16 {
		return vacio, ErrVersionarRolBolsaExternaInvalido
	}
	if base.Validar() != nil || base.TipoPerfil != TipoPerfilAdministracionAdministrableV1 ||
		base.Rol.Estado != EstadoVersionRolPublicada || base.ControlVigencia.Estado != EstadoControlVigenciaVersionRolHabilitada ||
		base.ControlVigencia.ActualizadoEn.After(ahora) ||
		len(acciones) == 0 {
		return vacio, ErrVersionarRolBolsaExternaInvalido
	}
	hr, er := base.Rol.HuellaSHA256()
	hcontrol, ec := base.ControlVigencia.HuellaSHA256()
	if er != nil || ec != nil || i.BaseRef != base.Rol.Referencia() || i.BaseHuellaSHA256 != hr ||
		i.ControlRevision != base.ControlVigencia.Revision || i.ControlHuellaSHA256 != hcontrol ||
		len(base.Rol.Concesiones)+len(i.Selecciones) > maximoElementosAutorizacion && !reemplazo {
		return vacio, ErrVersionarRolBolsaExternaInvalido
	}
	if !reemplazo {
		for _, anterior := range base.Rol.Concesiones {
			if slices.Contains(acciones, anterior.Accion) {
				return vacio, ErrVersionarRolBolsaExternaInvalido
			}
		}
	}
	concesiones := copiarConcesionesVersionarBolsa(base.Rol.Concesiones)
	for j, accion := range acciones {
		s := i.Selecciones[j]
		var entrada *EntradaAccionAdministracionV1
		for k := range c.Entradas {
			e := &c.Entradas[k]
			h, eErr := e.HuellaSHA256()
			if eErr == nil && e.Referencia == s.EntradaRef && e.Version == s.EntradaVersion && h == s.EntradaHuellaSHA256 {
				entrada = e
				break
			}
		}
		if entrada == nil || entrada.ClaseControl != string(ClaseControlPerfilOrdinario) ||
			(exigirVigencia && !vigenteAccionesAdministracionEn(entrada.VigenteDesde, entrada.VigenteHasta, ahora)) ||
			!dimensionesCandidatoBolsaExterna(entrada.DimensionesAmbito) ||
			!concesionDeltaBolsaExterna(entrada.Concesion, accion) {
			return vacio, ErrVersionarRolBolsaExternaInvalido
		}
		if reemplazo {
			indice := slices.IndexFunc(concesiones, func(con ConcesionRol) bool { return con.Accion == accion })
			if indice < 0 || entrada.Version != 2 || entrada.FuenteVersion != 2 ||
				entrada.FuenteRef != "fuente:bolsa:preferencias_bolsas_propias" {
				return vacio, ErrVersionarRolBolsaExternaInvalido
			}
			esperada := concesiones[indice]
			if j == 0 {
				esperada.CamposPermitidos = camposPreferenciasModalidadesBolsaExterna()
			}
			if !reflect.DeepEqual(esperada, entrada.Concesion) {
				return vacio, ErrVersionarRolBolsaExternaInvalido
			}
			if j == 0 {
				concesiones[indice].CamposPermitidos = camposPreferenciasModalidadesBolsaExterna()
			}
		} else {
			concesiones = append(concesiones, copiarConcesionesVersionarBolsa([]ConcesionRol{entrada.Concesion})[0])
		}
	}
	asignaciones := make([]SeleccionAsignacionVersionarRolBolsa, len(i.Asignaciones))
	for j, a := range i.Asignaciones {
		if a.ValidarPara(base.Rol.RolID, base.Rol.Version) != nil || !ambitosCandidatoBolsaExterna(a.Documento.Ambitos) {
			return vacio, ErrVersionarRolBolsaExternaInvalido
		}
		asignaciones[j] = SeleccionAsignacionVersionarRolBolsa{AsignacionRef: a.AsignacionRef,
			HuellaSHA256: a.HuellaSHA256, Documento: copiarAsignacionVersionarBolsa(a.Documento)}
	}
	baseCopia := base
	baseCopia.Rol.Concesiones = copiarConcesionesVersionarBolsa(base.Rol.Concesiones)
	plan := PlanVersionarRolBolsaExterna{
		Operacion: OperacionVersionarPerfilGobernado, CatalogoRef: c.Referencia, CatalogoVersion: c.Version,
		CatalogoHuellaSHA256:  hc,
		VersionRolObjetivoRef: (VersionRol{RolID: base.Rol.RolID, Version: base.Rol.Version + 1}).Referencia(),
		Base:                  baseCopia, DefinicionNueva: DefinicionVersionPerfilGobernado{RolID: base.Rol.RolID,
			Version: base.Rol.Version + 1, Nombre: base.Rol.Nombre, Concesiones: concesiones},
		Selecciones:  append([]SeleccionAccionAdministracionV1(nil), i.Selecciones...),
		Asignaciones: asignaciones, Motivo: i.Motivo, ReferenciaActo: i.ReferenciaActo,
	}
	if _, err := plan.HuellaSHA256(); err != nil {
		return vacio, err
	}
	return plan, nil
}

func (p PlanVersionarRolBolsaExterna) Copia() PlanVersionarRolBolsaExterna {
	p.Base.Rol.Concesiones = copiarConcesionesVersionarBolsa(p.Base.Rol.Concesiones)
	p.DefinicionNueva.Concesiones = copiarConcesionesVersionarBolsa(p.DefinicionNueva.Concesiones)
	p.Selecciones = append([]SeleccionAccionAdministracionV1(nil), p.Selecciones...)
	p.Asignaciones = append([]SeleccionAsignacionVersionarRolBolsa(nil), p.Asignaciones...)
	for j := range p.Asignaciones {
		p.Asignaciones[j].Documento = copiarAsignacionVersionarBolsa(p.Asignaciones[j].Documento)
	}
	return p
}
