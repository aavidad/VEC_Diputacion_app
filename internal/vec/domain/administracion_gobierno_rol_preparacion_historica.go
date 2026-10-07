package domain

import "time"

// PrepararPlanGobiernoRolNuevoDesdeCatalogo usa siempre el catálogo exacto
// leído de AUT58. Si el descriptor ya caducó, prepara con su instante
// publicado original para que AUT60 pueda reconocer un replay histórico.
// Esto no autoriza una primera propuesta caducada: AUT60 exige cabeza y
// vigencia ACTUALES al escribir por primera vez. La fecha aportada en el DTO
// del solicitante jamás decide cuál de los dos casos se ejecuta.
func PrepararPlanGobiernoRolNuevoDesdeCatalogo(c CatalogoAccionesAdministracionV1,
	s SolicitudPlanGobiernoPerfil, ahora time.Time, actorPersonaRef string) (PlanGobiernoPerfil, time.Time, error) {
	var vacio PlanGobiernoPerfil
	if c.Validar() != nil || !instanteAutorizacionCanonico(ahora) ||
		s.Operacion != OperacionCrearPerfilGobernado || s.Publicacion == nil ||
		s.Deshabilitacion != nil || len(s.Publicacion.Selecciones) != 1 ||
		len(s.Publicacion.RolPropuesto.Concesiones) != 1 ||
		!textoAutorizacionSinComodinSeguro(actorPersonaRef, 512, false) {
		return vacio, time.Time{}, ErrPlanGobiernoPerfilInvalido
	}
	pub := *s.Publicacion
	huella, err := c.HuellaSHA256()
	if err != nil || pub.CatalogoRef != c.Referencia || pub.CatalogoVersion != c.Version ||
		pub.CatalogoHuellaSHA256 != huella {
		return vacio, time.Time{}, ErrOrigenPerfilAdministracionNoCoincide
	}
	seleccion := pub.Selecciones[0]
	var entrada *EntradaAccionAdministracionV1
	for i := range c.Entradas {
		e := &c.Entradas[i]
		he, eErr := e.HuellaSHA256()
		if eErr == nil && e.Referencia == seleccion.EntradaRef && e.Version == seleccion.EntradaVersion &&
			he == seleccion.EntradaHuellaSHA256 {
			entrada = e
			break
		}
	}
	if entrada == nil {
		return vacio, time.Time{}, ErrPermisoPerfilAdministracionNoCoincide
	}
	instante := ahora
	if !vigenteAccionesAdministracionEn(c.VigenteDesde, c.VigenteHasta, ahora) ||
		!vigenteAccionesAdministracionEn(entrada.VigenteDesde, entrada.VigenteHasta, ahora) {
		instante = entrada.VigenteDesde
	}
	// Autor/fecha del documento de preparación son marcadores técnicos para
	// el validador común, no metadatos efectivos de publicación.
	pub.RolPropuesto.PublicadaPor = actorPersonaRef
	pub.RolPropuesto.PublicadaEn = instante
	s.Publicacion = &pub
	plan, err := PrepararPlanGobiernoPerfil(c, s, instante)
	if err != nil || plan.Operacion != OperacionCrearPerfilGobernado || plan.Base != nil ||
		plan.DefinicionNueva == nil || len(plan.DefinicionNueva.Concesiones) != 1 {
		return vacio, time.Time{}, ErrPlanGobiernoPerfilInvalido
	}
	return plan, instante, nil
}
