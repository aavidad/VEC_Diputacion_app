package domain

import (
	"reflect"
	"time"
)

const (
	RolIDPersonaDietasComisiones = "persona_dietas_comisiones"
	RolIDPersonaDietasRutas      = "persona_dietas_rutas"
)

var accionesRolDietasComisiones = []string{
	"personal.relacion.propia.consultar_dietas",
	"dietas.borrador.propio.crear",
	"dietas.borrador.propio.consultar",
}

var accionesRolDietasRutas = []string{
	"dietas.ruta.catalogo.consultar",
	"dietas.ruta.calculo.solicitar",
}

var camposDietasRelacion = []string{
	"desde", "empleado_ref", "estado", "fuente_ref", "fuente_version", "hasta",
	"persona_ref", "procedencia_acto_ref", "relacion_ref", "unidad_ref", "version",
}

var camposDietasBorrador = []string{
	"comision.calculo", "comision.codigos_ruta", "comision.estado", "comision.fecha_fin",
	"comision.fecha_inicio", "comision.motivo", "comision.referencia", "comision.relacion_ref",
	"recibo.referencia", "recibo.registrado_en", "recibo.repeticion", "recibo.version",
}

var camposDietasConsulta = []string{
	"comision.calculo", "comision.codigos_ruta", "comision.estado", "comision.fecha_fin",
	"comision.fecha_inicio", "comision.motivo", "comision.referencia", "comision.relacion_ref",
	"items.comision.calculo", "items.comision.codigos_ruta", "items.comision.estado",
	"items.comision.fecha_fin", "items.comision.fecha_inicio", "items.comision.motivo",
	"items.comision.referencia", "items.comision.relacion_ref", "items.recibo.referencia",
	"items.recibo.registrado_en", "items.recibo.repeticion", "items.recibo.version",
	"recibo.referencia", "recibo.registrado_en", "recibo.repeticion", "recibo.version",
	"siguiente_cursor",
}

// Esta excepción admite únicamente los dos contratos publicados de Dietas.
// El catálogo AUT58 aporta las entradas, versiones y huellas; aquí se coteja
// además su contenido nominal antes de usar el validador común.
func prepararPlanGobiernoRolDietas(c CatalogoAccionesAdministracionV1,
	s SolicitudPlanGobiernoPerfil, ahora time.Time, actor string) (PlanGobiernoPerfil, time.Time, error) {
	vacio := PlanGobiernoPerfil{}
	if s.Publicacion == nil {
		return vacio, time.Time{}, ErrPlanGobiernoPerfilInvalido
	}
	rolID := s.Publicacion.RolPropuesto.RolID
	var esperadas []string
	switch rolID {
	case RolIDPersonaDietasComisiones:
		esperadas = accionesRolDietasComisiones
	case RolIDPersonaDietasRutas:
		esperadas = accionesRolDietasRutas
	default:
		return vacio, time.Time{}, ErrPlanGobiernoPerfilInvalido
	}
	if c.Validar() != nil || !instanteAutorizacionCanonico(ahora) ||
		s.Operacion != OperacionCrearPerfilGobernado || s.Deshabilitacion != nil ||
		len(s.Publicacion.Selecciones) != len(esperadas) ||
		len(s.Publicacion.RolPropuesto.Concesiones) != len(esperadas) ||
		!textoAutorizacionSinComodinSeguro(actor, 512, false) {
		return vacio, time.Time{}, ErrPlanGobiernoPerfilInvalido
	}
	pub := *s.Publicacion
	huella, err := c.HuellaSHA256()
	if err != nil || pub.CatalogoRef != c.Referencia || pub.CatalogoVersion != c.Version ||
		pub.CatalogoHuellaSHA256 != huella {
		return vacio, time.Time{}, ErrOrigenPerfilAdministracionNoCoincide
	}
	entradas := make([]*EntradaAccionAdministracionV1, len(esperadas))
	for i, seleccion := range pub.Selecciones {
		for j := range c.Entradas {
			e := &c.Entradas[j]
			he, eErr := e.HuellaSHA256()
			if eErr == nil && e.Referencia == seleccion.EntradaRef &&
				e.Version == seleccion.EntradaVersion && he == seleccion.EntradaHuellaSHA256 {
				entradas[i] = e
				break
			}
		}
		if entradas[i] == nil || entradas[i].Concesion.Accion != esperadas[i] ||
			!entradaYRolNuevoOrdinarios(*entradas[i], pub.RolPropuesto) ||
			!concesionesPerfilAdministracionIguales(entradas[i].Concesion, pub.RolPropuesto.Concesiones[i]) ||
			!concesionRolDietasExacta(rolID, i, entradas[i].Concesion) ||
			!dimensionesRolDietasExactas(rolID, entradas[i].DimensionesAmbito) {
			return vacio, time.Time{}, ErrPermisoPerfilAdministracionNoCoincide
		}
	}
	// AUT60 distingue la primera publicación, que exige vigencia actual, del
	// replay histórico. El DTO no puede escoger el instante de preparación.
	instante := ahora
	todasVigentes := vigenteAccionesAdministracionEn(c.VigenteDesde, c.VigenteHasta, ahora)
	for _, e := range entradas {
		todasVigentes = todasVigentes && vigenteAccionesAdministracionEn(e.VigenteDesde, e.VigenteHasta, ahora)
	}
	if !todasVigentes {
		instante = c.VigenteDesde
		for _, e := range entradas {
			if e.VigenteDesde.After(instante) {
				instante = e.VigenteDesde
			}
		}
		if !vigenteAccionesAdministracionEn(c.VigenteDesde, c.VigenteHasta, instante) {
			return vacio, time.Time{}, ErrCatalogoAccionesAdministracionNoVigente
		}
		for _, e := range entradas {
			if !vigenteAccionesAdministracionEn(e.VigenteDesde, e.VigenteHasta, instante) {
				return vacio, time.Time{}, ErrCatalogoAccionesAdministracionNoVigente
			}
		}
	}
	pub.RolPropuesto.PublicadaPor = actor
	pub.RolPropuesto.PublicadaEn = instante
	s.Publicacion = &pub
	plan, err := PrepararPlanGobiernoPerfil(c, s, instante)
	if err != nil || plan.Operacion != OperacionCrearPerfilGobernado || plan.Base != nil ||
		plan.DefinicionNueva == nil || len(plan.DefinicionNueva.Concesiones) != len(esperadas) {
		return vacio, time.Time{}, ErrPlanGobiernoPerfilInvalido
	}
	return plan, instante, nil
}

func dimensionesRolDietasExactas(rolID string, dimensiones []string) bool {
	if rolID == RolIDPersonaDietasComisiones {
		return reflect.DeepEqual(dimensiones, []string{"empleado_ref", "persona_ref"})
	}
	return rolID == RolIDPersonaDietasRutas && reflect.DeepEqual(dimensiones, []string{"ambito_ref"})
}

func concesionRolDietasExacta(rolID string, i int, c ConcesionRol) bool {
	if c.Validar() != nil || c.GarantiaMinima != AuthAssuranceHigh || len(c.Obligaciones) != 0 ||
		len(c.Finalidades) != 1 {
		return false
	}
	if rolID == RolIDPersonaDietasComisiones {
		switch i {
		case 0:
			return c.Accion == accionesRolDietasComisiones[0] && c.ModuloID == "personal" &&
				c.TipoRecurso == "relacion_empleado_dietas" && c.Finalidades[0] == "preparar_borrador_dietas" &&
				reflect.DeepEqual(c.CamposPermitidos, camposDietasRelacion)
		case 1:
			return c.Accion == accionesRolDietasComisiones[1] && c.ModuloID == "dietas" &&
				c.TipoRecurso == "comision_borrador" && c.Finalidades[0] == "crear_borrador_propio" &&
				reflect.DeepEqual(c.CamposPermitidos, camposDietasBorrador)
		case 2:
			return c.Accion == accionesRolDietasComisiones[2] && c.ModuloID == "dietas" &&
				c.TipoRecurso == "comision_borrador" && c.Finalidades[0] == "consultar_borrador_propio" &&
				reflect.DeepEqual(c.CamposPermitidos, camposDietasConsulta)
		}
	}
	if rolID == RolIDPersonaDietasRutas && c.ModuloID == "dietas" &&
		c.Finalidades[0] == "consultar_itinerario_dietas" && len(c.CamposPermitidos) == 0 {
		switch i {
		case 0:
			return c.Accion == accionesRolDietasRutas[0] && c.TipoRecurso == "catalogo_rutas_dietas"
		case 1:
			return c.Accion == accionesRolDietasRutas[1] && c.TipoRecurso == "calculo_rutas_dietas"
		}
	}
	return false
}

func planRolDietasExacto(p PlanGobiernoPerfil) bool {
	if p.Operacion != OperacionCrearPerfilGobernado || p.Base != nil || p.DefinicionNueva == nil ||
		p.DefinicionNueva.Version != 1 {
		return false
	}
	rolID := p.DefinicionNueva.RolID
	var esperadas []string
	switch rolID {
	case RolIDPersonaDietasComisiones:
		esperadas = accionesRolDietasComisiones
	case RolIDPersonaDietasRutas:
		esperadas = accionesRolDietasRutas
	default:
		return false
	}
	if len(p.DefinicionNueva.Concesiones) != len(esperadas) || len(p.Selecciones) != len(esperadas) ||
		p.VersionRolObjetivoRef != "rol:"+rolID+":v1" {
		return false
	}
	for i, concesion := range p.DefinicionNueva.Concesiones {
		s := p.Selecciones[i]
		if !concesionRolDietasExacta(rolID, i, concesion) ||
			!textoAutorizacionSinComodinSeguro(s.EntradaRef, 512, false) || s.EntradaVersion < 1 ||
			!huellaSHA256AutorizacionV3NoNula(s.EntradaHuellaSHA256) {
			return false
		}
		for j := 0; j < i; j++ {
			if p.Selecciones[i].EntradaRef == p.Selecciones[j].EntradaRef {
				return false
			}
		}
	}
	return true
}
