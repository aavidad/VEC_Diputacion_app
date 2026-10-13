package domain

import (
	"reflect"
	"time"
)

const RolIDPersonaMeritosCartera = "persona_meritos_cartera"

const (
	accionRolMeritosConsultar = "meritos.cartera.consultar_propia"
	accionRolMeritosRetirar   = "meritos.cartera.retirar_propio"
	accionRolMeritosDeclarar  = "meritos.hecho.declarar"
)

// La única ampliación de GobiernoRolNuevo permite publicar juntos los dos
// permisos de la cartera. El rol histórico de una concesión sigue usando su
// preparación original y la autoridad de AUT60.
func prepararPlanGobiernoRolMeritosCartera(c CatalogoAccionesAdministracionV1,
	s SolicitudPlanGobiernoPerfil, ahora time.Time, actor string) (PlanGobiernoPerfil, time.Time, error) {
	vacio := PlanGobiernoPerfil{}
	if c.Validar() != nil || !instanteAutorizacionCanonico(ahora) ||
		s.Operacion != OperacionCrearPerfilGobernado || s.Publicacion == nil ||
		s.Deshabilitacion != nil || (len(s.Publicacion.Selecciones) != 2 && len(s.Publicacion.Selecciones) != 3) ||
		len(s.Publicacion.RolPropuesto.Concesiones) != len(s.Publicacion.Selecciones) ||
		s.Publicacion.RolPropuesto.RolID != RolIDPersonaMeritosCartera ||
		!textoAutorizacionSinComodinSeguro(actor, 512, false) {
		return vacio, time.Time{}, ErrPlanGobiernoPerfilInvalido
	}
	pub := *s.Publicacion
	huella, err := c.HuellaSHA256()
	if err != nil || pub.CatalogoRef != c.Referencia || pub.CatalogoVersion != c.Version ||
		pub.CatalogoHuellaSHA256 != huella {
		return vacio, time.Time{}, ErrOrigenPerfilAdministracionNoCoincide
	}
	esperadas := []string{accionRolMeritosConsultar, accionRolMeritosRetirar}
	if len(pub.Selecciones) == 3 {
		esperadas = append(esperadas, accionRolMeritosDeclarar)
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
			!reflect.DeepEqual(entradas[i].DimensionesAmbito, []string{"persona_ref"}) ||
			!entradaYRolNuevoOrdinarios(*entradas[i], pub.RolPropuesto) ||
			!reflect.DeepEqual(entradas[i].Concesion, pub.RolPropuesto.Concesiones[i]) ||
			!concesionRolMeritosCartera(entradas[i].Concesion, esperadas[i]) {
			return vacio, time.Time{}, ErrPermisoPerfilAdministracionNoCoincide
		}
	}
	// Para una propuesta nueva rige ahora. En un replay con catálogo vencido
	// se reconstruye solo desde el intervalo publicado de las entradas.
	instante := ahora
	todasVigentes := true
	for _, entrada := range entradas {
		todasVigentes = todasVigentes && vigenteAccionesAdministracionEn(entrada.VigenteDesde, entrada.VigenteHasta, ahora)
	}
	if !todasVigentes || !vigenteAccionesAdministracionEn(c.VigenteDesde, c.VigenteHasta, ahora) ||
		!vigenteAccionesAdministracionEn(entradas[0].VigenteDesde, entradas[0].VigenteHasta, ahora) ||
		!vigenteAccionesAdministracionEn(entradas[1].VigenteDesde, entradas[1].VigenteHasta, ahora) {
		instante = c.VigenteDesde
		for _, e := range entradas {
			if e.VigenteDesde.After(instante) {
				instante = e.VigenteDesde
			}
		}
		if !vigenteAccionesAdministracionEn(c.VigenteDesde, c.VigenteHasta, instante) ||
			!vigenteAccionesAdministracionEn(entradas[0].VigenteDesde, entradas[0].VigenteHasta, instante) ||
			!vigenteAccionesAdministracionEn(entradas[1].VigenteDesde, entradas[1].VigenteHasta, instante) {
			return vacio, time.Time{}, ErrCatalogoAccionesAdministracionNoVigente
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

func concesionRolMeritosCartera(c ConcesionRol, accion string) bool {
	if c.Validar() != nil || c.ModuloID != "meritos" || c.GarantiaMinima != AuthAssuranceHigh ||
		c.TipoRecurso != "cartera_personal" && accion == accionRolMeritosConsultar ||
		c.TipoRecurso != "hecho" && accion == accionRolMeritosRetirar ||
		len(c.Finalidades) != 1 || (accion != accionRolMeritosDeclarar && c.Finalidades[0] != "gestion_cartera_propia") ||
		len(c.Obligaciones) != 1 || c.Obligaciones[0] != "auditar" {
		return false
	}
	if accion == accionRolMeritosDeclarar {
		return c.TipoRecurso == "hecho" && c.Finalidades[0] == "declaracion_hecho_propio" &&
			reflect.DeepEqual(c.CamposPermitidos, []string{"declarante_ref", "hecho", "recibo", "version"})
	}
	if accion == accionRolMeritosConsultar {
		campos := []string{"denominacion", "disponibilidad", "hecho_ref", "motivos_retirada", "retirada", "tipo", "total", "version_hecho"}
		return reflect.DeepEqual(c.CamposPermitidos, campos)
	}
	return reflect.DeepEqual(c.CamposPermitidos, []string{"disponibilidad", "recibo"})
}
