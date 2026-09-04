package postgres

import (
	"crypto/hmac"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func nuevaCabeceraDecisionCoberturaO404E(
	d cobertura.DatosCabeceraSesionTCBOperacionDecisionCobertura,
) (cabeceraDecisionCoberturaO404E, error) {
	if d.Esquema == "" || d.HuellaOrdenSHA256 == "" ||
		d.OrganizacionRef == "" || d.ExpedienteRef == "" ||
		d.VersionExpediente == 0 || d.ReservaRef == "" ||
		d.ReciboRef == "" || d.AuditoriaRef == "" ||
		d.CorrelacionVECRef == "" || d.DecisionVECRef == "" ||
		d.RevisionCercado == 0 || d.ValidaHastaOrden.IsZero() ||
		(d.Rama ==
			cobertura.RamaSesionTCBOperacionDecisionCoberturaConcedida &&
			d.NumeroConsumosC1 == 0) ||
		(d.Rama ==
			cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada &&
			d.NumeroConsumosC1 != 0) {
		return cabeceraDecisionCoberturaO404E{},
			errSesionDecisionCoberturaO404EInvalida
	}
	return cabeceraDecisionCoberturaO404E{
		Esquema: d.Esquema, HuellaOrdenSHA256: d.HuellaOrdenSHA256,
		OrganizacionRef: d.OrganizacionRef, ExpedienteRef: d.ExpedienteRef,
		VersionExpediente: d.VersionExpediente, ReservaRef: d.ReservaRef,
		ReciboRef: d.ReciboRef, ActuacionRef: d.ActuacionRef,
		AuditoriaRef: d.AuditoriaRef, EventoRef: d.EventoRef,
		CorrelacionVECRef: d.CorrelacionVECRef,
		DecisionVECRef:    d.DecisionVECRef, AnalisisRef: d.AnalisisRef,
		AnalisisHuellaSHA256:    d.AnalisisHuellaSHA256,
		TokenPropietarioSHA256:  d.TokenPropietarioSHA256,
		AmbitoIdempotenciaHMAC:  d.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:     d.HuellaSemanticaHMAC,
		RevisionCercadoAnterior: d.RevisionCercadoAnterior,
		RevisionCercado:         d.RevisionCercado, ObservadaEnDB: d.ObservadaEnDB,
		PropiedadHasta: d.PropiedadHasta, ValidaHastaOrden: d.ValidaHastaOrden,
		PreparacionC1Ref:             d.PreparacionC1Ref,
		PreparacionC1HuellaSHA256:    d.PreparacionC1HuellaSHA256,
		PreparacionC1PreparadaEn:     d.PreparacionC1PreparadaEn,
		PreparacionC1ValidaHasta:     d.PreparacionC1ValidaHasta,
		NumeroConsumosC1:             d.NumeroConsumosC1,
		HuellaOrdenesConsumoC1SHA256: d.HuellaOrdenesConsumoC1SHA256,
	}, nil
}

func nuevoGobiernoDecisionCoberturaO404E(
	c cabeceraDecisionCoberturaO404E,
	d cobertura.DatosGobiernoSesionTCBOperacionDecisionCobertura,
) (gobiernoDecisionCoberturaO404E, error) {
	if d.Catalogo.Referencia == "" || d.Politica.Referencia == "" ||
		d.PoliticaActuacion.Referencia == "" ||
		d.Politica.OrganizacionRef != c.OrganizacionRef ||
		d.PoliticaActuacion.OrganizacionRef != c.OrganizacionRef ||
		d.EvaluadaEn.IsZero() || d.ValidaHasta.IsZero() ||
		!d.ValidaHasta.After(d.EvaluadaEn) {
		return gobiernoDecisionCoberturaO404E{},
			errSesionDecisionCoberturaO404EInvalida
	}
	return gobiernoDecisionCoberturaO404E{
		Catalogo: d.Catalogo, Politica: d.Politica,
		PoliticaActuacion: d.PoliticaActuacion, Accion: d.Accion,
		FinalidadCTClave: d.FinalidadCTClave, FinalidadCTRef: d.FinalidadCTRef,
		FinalidadVEC: d.FinalidadVEC, UnidadEjecutoraRef: d.UnidadEjecutoraRef,
		FaseDestino: d.FaseDestino, EstadoDestino: d.EstadoDestino,
		MotivoAutorizacion: d.MotivoAutorizacion,
		EvaluadaEn:         d.EvaluadaEn, ValidaHasta: d.ValidaHasta,
	}, nil
}

func nuevaDecisionVECDecisionCoberturaO404E(
	c cabeceraDecisionCoberturaO404E,
	d cobertura.DatosDecisionVECSesionTCBOperacionDecisionCobertura,
) (decisionVECDecisionCoberturaO404E, error) {
	solicitud, errSolicitud := d.Orden.Solicitud.Datos()
	decisionCanonica, errDecision :=
		dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(
			d.Orden.Decision,
		)
	motivoCanonico, errMotivo :=
		dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(
			d.Orden.ReferenciaMotivo,
		)
	if errSolicitud != nil || errDecision != nil || errMotivo != nil ||
		d.Orden.ResultadoContexto.Validar() != nil {
		borrarBytes(decisionCanonica)
		borrarBytes(motivoCanonico)
		return decisionVECDecisionCoberturaO404E{},
			errSesionDecisionCoberturaO404EInvalida
	}
	correlacion, errCorrelacion := solicitud.Correlacion.ValorCanonico()
	huellaRecurso, errRecurso :=
		solicitud.Recurso.HuellaContextoAutorizacionSHA256()
	instantanea := d.Orden.ResultadoContexto.Contexto.Instantanea
	if errCorrelacion != nil || errRecurso != nil ||
		d.Resumen.Concedida != d.Concedida ||
		!igualesDecisionCoberturaO404E(d.Resumen.DecisionRef, c.DecisionVECRef) ||
		!igualesDecisionCoberturaO404E(correlacion, c.CorrelacionVECRef) {
		borrarBytes(decisionCanonica)
		borrarBytes(motivoCanonico)
		return decisionVECDecisionCoberturaO404E{},
			errSesionDecisionCoberturaO404EInvalida
	}
	resultado := decisionVECDecisionCoberturaO404E{
		DecisionCanonica:     decisionCanonica,
		MotivoCanonico:       motivoCanonico,
		PersonaVersion:       instantanea.PersonaVersion,
		PerfilVersion:        instantanea.PerfilVersion,
		DecisionRef:          d.Resumen.DecisionRef,
		DecisionHuellaSHA256: d.Resumen.DecisionHuellaSHA256,
		CodigoProbatorio:     d.Resumen.CodigoProbatorio,
		Concedida:            d.Resumen.Concedida, EmitidaEn: d.Resumen.EmitidaEn,
		ValidaHasta:                 d.Resumen.ValidaHasta,
		PrincipalID:                 instantanea.PersonaRef,
		PerfilActivoRef:             instantanea.PerfilActivoRef,
		Accion:                      solicitud.Accion,
		ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad:                   solicitud.Finalidad, CorrelacionRef: correlacion,
	}
	copiarRecursoDecisionVECDecisionCoberturaO404E(
		&resultado,
		solicitud.Recurso,
	)
	return resultado, nil
}

func nuevoConsumoC1DecisionCoberturaO404E(
	c cabeceraDecisionCoberturaO404E,
	d cobertura.DatosConsumoC1SesionTCBOperacionDecisionCobertura,
) (consumoC1DecisionCoberturaO404E, error) {
	pruebas, err := d.PruebasCanonicas.Datos()
	if err != nil {
		return consumoC1DecisionCoberturaO404E{},
			errSesionDecisionCoberturaO404EInvalida
	}
	r := d.Resumen
	if r.OrganizacionRef != c.OrganizacionRef ||
		r.ExpedienteRef != c.ExpedienteRef ||
		r.VersionExpediente != c.VersionExpediente ||
		r.PeticionRef != d.Orden.PeticionRef ||
		r.HuellaPeticionSHA256 != d.Orden.HuellaPeticionSHA256 ||
		r.HuellaResultadoSHA256 != d.Orden.HuellaResultadoSHA256 ||
		r.HuellaRespuestaSHA256 != d.Orden.HuellaRespuestaSHA256 ||
		r.AutoridadRef != d.Orden.AutoridadRef ||
		r.Generacion != d.Orden.Generacion ||
		r.ReciboRespuestaRef != d.Orden.ReciboRespuestaRef {
		limpiarPruebasCanonicasDecisionCoberturaO404E(&pruebas)
		return consumoC1DecisionCoberturaO404E{},
			errSesionDecisionCoberturaO404EInvalida
	}
	return consumoC1DecisionCoberturaO404E{
		Posicion: d.Posicion, Total: d.Total,
		PeticionRef: r.PeticionRef, OrganizacionRef: r.OrganizacionRef,
		ExpedienteRef: r.ExpedienteRef, VersionExpediente: r.VersionExpediente,
		CatalogoRef: r.Catalogo.Referencia, CatalogoVersion: r.Catalogo.Version,
		CatalogoHuellaSHA256: r.Catalogo.HuellaSHA256, ViaClave: r.ViaClave,
		ComprobacionClave:      r.Comprobacion.Clave,
		ComprobacionResultado:  r.Comprobacion.Resultado,
		ComprobacionFuenteRef:  r.Comprobacion.FuenteRef,
		ComprobacionReciboRef:  r.Comprobacion.ReciboRef,
		ComprobacionEvaluadaEn: r.Comprobacion.EvaluadaEn,
		OrdenComprobacion:      r.OrdenComprobacion,
		Obligatoria:            r.ComprobacionObligatoria,
		ProcedenciaClave:       r.ProcedenciaClave,
		DefinicionFuenteRef:    r.DefinicionFuenteRef, CategoriaRef: r.CategoriaRef,
		Periodo: r.Periodo, SolicitadaEn: r.SolicitadaEn,
		EmitidaEn: r.EmitidaEn, ValidaHasta: r.ValidaHasta,
		HuellaPeticionSHA256:  r.HuellaPeticionSHA256,
		HuellaResultadoSHA256: r.HuellaResultadoSHA256,
		HuellaRespuestaSHA256: r.HuellaRespuestaSHA256,
		AutoridadRef:          r.AutoridadRef, Generacion: r.Generacion,
		ReciboRespuestaRef:    r.ReciboRespuestaRef,
		VerificadorRef:        r.VerificadorRef,
		PublicadorCatalogoRef: r.PublicadorCatalogoRef,
		Pruebas: pruebasCanonicasC1DecisionCoberturaO404E{
			Peticion:        pruebas.Peticion,
			Resultado:       pruebas.Resultado,
			Atestacion:      pruebas.Atestacion,
			ConfirmacionTCB: pruebas.ConfirmacionTCB,
			Catalogo:        pruebas.Catalogo,
			Verificador:     pruebas.Verificador,
			Resumen:         pruebas.Resumen,
		},
	}, nil
}

func nuevaConcesionDecisionCoberturaO404E(
	c cabeceraDecisionCoberturaO404E,
	d cobertura.DatosEfectoConcedidoSesionTCBOperacionDecisionCobertura,
) (concesionDecisionCoberturaO404E, error) {
	if d.AgregadoAnterior.Validar() != nil ||
		d.AgregadoSiguiente.Validar() != nil ||
		d.AgregadoAnterior.OrganizacionRef != c.OrganizacionRef ||
		d.AgregadoAnterior.Referencia != c.ExpedienteRef ||
		d.AgregadoAnterior.Version != c.VersionExpediente ||
		d.AgregadoSiguiente.OrganizacionRef != c.OrganizacionRef ||
		d.AgregadoSiguiente.Referencia != c.ExpedienteRef ||
		d.AgregadoSiguiente.Version != c.VersionExpediente+1 ||
		d.Propuesta.OrganizacionRef != c.OrganizacionRef ||
		d.Propuesta.ExpedienteRef != c.ExpedienteRef ||
		d.Propuesta.VersionExpediente != c.VersionExpediente ||
		d.EfectoEn.IsZero() || d.ValidaHasta.IsZero() ||
		!d.ValidaHasta.After(d.EfectoEn) {
		return concesionDecisionCoberturaO404E{},
			errSesionDecisionCoberturaO404EInvalida
	}
	return concesionDecisionCoberturaO404E{
		AgregadoAnterior:  d.AgregadoAnterior,
		AgregadoSiguiente: d.AgregadoSiguiente,
		Propuesta:         nuevaPublicacionPropuestaDecisionCoberturaO404E(d.Propuesta),
		MotivoFuncional:   d.MotivoFuncional,
		EfectoEn:          d.EfectoEn, ValidaHasta: d.ValidaHasta,
	}, nil
}

func nuevaPublicacionPropuestaDecisionCoberturaO404E(
	p domain.PublicacionPropuestaDecisionCobertura,
) publicacionPropuestaDecisionCoberturaO404E {
	return publicacionPropuestaDecisionCoberturaO404E{
		Referencia: p.Referencia, HuellaSHA256: p.HuellaSHA256, Canon: p.Canon,
		OrganizacionRef: p.OrganizacionRef, ExpedienteRef: p.ExpedienteRef,
		VersionExpediente: p.VersionExpediente, AnalisisRef: p.AnalisisRef,
		AnalisisHuellaSHA256:              p.AnalisisHuellaSHA256,
		PreparacionEvidenciasRef:          p.PreparacionEvidenciasRef,
		PreparacionEvidenciasHuellaSHA256: p.PreparacionEvidenciasHuellaSHA256,
		Catalogo:                          p.Catalogo, Politica: p.Politica,
		FinalidadClave: p.FinalidadClave, FinalidadRef: p.FinalidadRef,
		CategoriaRef: p.CategoriaRef, Periodo: p.Periodo,
		GeneradaEn: p.GeneradaEn, ValidaHasta: p.ValidaHasta,
		Estado: p.Estado, ViaPropuesta: p.ViaPropuesta,
		Resultados: p.Resultados, Evaluaciones: p.Evaluaciones,
	}
}

func nuevaDenegacionDecisionCoberturaO404E(
	c cabeceraDecisionCoberturaO404E,
	d cobertura.DatosTerminalDenegadoSesionTCBOperacionDecisionCobertura,
) (denegacionDecisionCoberturaO404E, error) {
	huellaRecurso, err := d.RecursoVEC.HuellaContextoAutorizacionSHA256()
	if err != nil || d.OrganizacionRef != c.OrganizacionRef ||
		d.ExpedienteRef != c.ExpedienteRef ||
		d.VersionExpediente != c.VersionExpediente ||
		d.ReservaRef != c.ReservaRef || d.ReciboRef != c.ReciboRef ||
		d.AuditoriaRef != c.AuditoriaRef ||
		d.CorrelacionVECRef != c.CorrelacionVECRef ||
		d.DecisionVECRef != c.DecisionVECRef ||
		d.RevisionCercado != c.RevisionCercado {
		return denegacionDecisionCoberturaO404E{},
			errSesionDecisionCoberturaO404EInvalida
	}
	return denegacionDecisionCoberturaO404E{
		OrganizacionRef: d.OrganizacionRef, ExpedienteRef: d.ExpedienteRef,
		VersionExpediente: d.VersionExpediente, ReservaRef: d.ReservaRef,
		ReciboRef: d.ReciboRef, AuditoriaRef: d.AuditoriaRef,
		CorrelacionVECRef: d.CorrelacionVECRef,
		DecisionVECRef:    d.DecisionVECRef, RevisionCercado: d.RevisionCercado,
		RecursoRef:    d.RecursoVEC.Referencia,
		RecursoModulo: d.RecursoVEC.ModuloID, RecursoTipo: d.RecursoVEC.Tipo,
		Ambitos:             clonarMapaDecisionCoberturaO404E(d.RecursoVEC.Ambitos),
		Atributos:           clonarMapaDecisionCoberturaO404E(d.RecursoVEC.Atributos),
		RecursoHuellaSHA256: huellaRecurso, ActorRef: d.ActorRef,
		PerfilRef: d.PerfilRef, AccionVEC: d.AccionVEC,
		FinalidadVEC: d.FinalidadVEC, MotivoVEC: d.MotivoVEC,
		LimitePreparacion: d.LimitePreparacion, ValidaHasta: d.ValidaHasta,
		PruebaHuellaSHA256: d.PruebaHuellaSHA256,
	}, nil
}

func clonarMapaDecisionCoberturaO404E(
	origen map[string]string,
) map[string]string {
	if origen == nil {
		return nil
	}
	copia := make(map[string]string, len(origen))
	for clave, valor := range origen {
		copia[clave] = valor
	}
	return copia
}

func copiarRecursoDecisionVECDecisionCoberturaO404E(
	destino *decisionVECDecisionCoberturaO404E,
	recurso dominiovec.RecursoAutorizable,
) {
	if destino == nil {
		return
	}
	destino.RecursoRef = recurso.Referencia
	destino.RecursoModulo = recurso.ModuloID
	destino.RecursoTipo = recurso.Tipo
	destino.Ambitos = clonarMapaDecisionCoberturaO404E(recurso.Ambitos)
	destino.Atributos = clonarMapaDecisionCoberturaO404E(recurso.Atributos)
}

func validarRecursoDenegacionDecisionCoberturaO404E(
	d denegacionDecisionCoberturaO404E,
) bool {
	// Reutilizar la validación nominal del dominio mantiene cerrados los
	// límites de 512 entradas y sus reglas de texto, incluido el rechazo de
	// comodines en ámbitos. La huella liga además cada par clave-valor.
	recurso := dominiovec.RecursoAutorizable{
		Referencia: d.RecursoRef,
		ModuloID:   d.RecursoModulo,
		Tipo:       d.RecursoTipo,
		Ambitos:    clonarMapaDecisionCoberturaO404E(d.Ambitos),
		Atributos:  clonarMapaDecisionCoberturaO404E(d.Atributos),
	}
	if err := recurso.Validar(); err != nil {
		return false
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	return err == nil &&
		huellaSHA256DecisionCoberturaO404EValida(d.RecursoHuellaSHA256) &&
		igualesDecisionCoberturaO404E(huella, d.RecursoHuellaSHA256)
}

func validarRecursoDecisionVECDecisionCoberturaO404E(
	d decisionVECDecisionCoberturaO404E,
) bool {
	recurso := dominiovec.RecursoAutorizable{
		Referencia: d.RecursoRef,
		ModuloID:   d.RecursoModulo,
		Tipo:       d.RecursoTipo,
		Ambitos:    clonarMapaDecisionCoberturaO404E(d.Ambitos),
		Atributos:  clonarMapaDecisionCoberturaO404E(d.Atributos),
	}
	if err := recurso.Validar(); err != nil {
		return false
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	return err == nil &&
		huellaSHA256DecisionCoberturaO404EValida(
			d.ContextoRecursoHuellaSHA256,
		) &&
		igualesDecisionCoberturaO404E(
			huella,
			d.ContextoRecursoHuellaSHA256,
		)
}

func recursosDecisionVECYDenegacionDecisionCoberturaO404EIguales(
	v decisionVECDecisionCoberturaO404E,
	d denegacionDecisionCoberturaO404E,
) bool {
	return igualesDecisionCoberturaO404E(v.RecursoRef, d.RecursoRef) &&
		igualesDecisionCoberturaO404E(v.RecursoModulo, d.RecursoModulo) &&
		igualesDecisionCoberturaO404E(v.RecursoTipo, d.RecursoTipo) &&
		igualesMapasDecisionCoberturaO404E(v.Ambitos, d.Ambitos) &&
		igualesMapasDecisionCoberturaO404E(v.Atributos, d.Atributos) &&
		igualesDecisionCoberturaO404E(
			v.ContextoRecursoHuellaSHA256,
			d.RecursoHuellaSHA256,
		)
}

func igualesMapasDecisionCoberturaO404E(
	a map[string]string,
	b map[string]string,
) bool {
	if len(a) != len(b) {
		return false
	}
	for clave, valor := range a {
		if otro, existe := b[clave]; !existe ||
			!igualesDecisionCoberturaO404E(valor, otro) {
			return false
		}
	}
	return true
}

func codificarCargaConfirmarDecisionCoberturaO404E(
	carga cargaConfirmarDecisionCoberturaO404E,
) ([]byte, error) {
	if carga.Esquema != esquemaCargaDecisionCoberturaO404E ||
		!validarRecursoDecisionVECDecisionCoberturaO404E(
			carga.DecisionVEC,
		) ||
		(carga.Gobierno == nil) !=
			(carga.Rama ==
				cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada) ||
		(carga.Concesion == nil) !=
			(carga.Rama ==
				cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada) ||
		(carga.Denegacion == nil) !=
			(carga.Rama ==
				cobertura.RamaSesionTCBOperacionDecisionCoberturaConcedida) ||
		(carga.Denegacion != nil &&
			!validarRecursoDenegacionDecisionCoberturaO404E(
				*carga.Denegacion,
			)) ||
		(carga.Denegacion != nil &&
			!recursosDecisionVECYDenegacionDecisionCoberturaO404EIguales(
				carga.DecisionVEC,
				*carga.Denegacion,
			)) {
		return nil, errSesionDecisionCoberturaO404EInvalida
	}
	return codificarCargaDecisionCoberturaO404E(carga)
}

func decodificarReciboDecisionCoberturaO404E(
	contenido []byte,
) (cobertura.DatosReciboSesionTCBOperacionDecisionCobertura, error) {
	if len(contenido) == 0 ||
		len(contenido) > maximoBytesReciboDecisionCoberturaO404E {
		return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{},
			errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	var dto reciboDecisionCoberturaO404E
	if _, err := decodificarObjetoJSONExactoDecisionCoberturaO404E(
		contenido,
		clavesReciboDecisionCoberturaO404E,
		&dto,
	); err != nil {
		return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{},
			errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	dto = normalizarReciboDecisionCoberturaPostgreSQL(dto)
	if !validarReciboDecisionCoberturaO404E(dto) {
		return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{},
			errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{
		ReciboRef: dto.ReciboRef, ReservaRef: dto.ReservaRef,
		AuditoriaRef:            dto.AuditoriaRef,
		CorrelacionVECRef:       dto.CorrelacionVECRef,
		DecisionVECRef:          dto.DecisionVECRef,
		DecisionVECHuellaSHA256: dto.DecisionVECHuellaSHA256,
		CodigoProbatorioVEC:     dto.CodigoProbatorioVEC,
		ConcedidaVEC:            dto.ConcedidaVEC, RevisionCercado: dto.RevisionCercado,
		AmbitoIdempotenciaHMAC: dto.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:    dto.HuellaSemanticaHMAC,
		ConfirmadaEn:           dto.ConfirmadaEn, Aplicada: dto.Aplicada,
		DenegadaVEC:             dto.DenegadaVEC,
		DecisionCoberturaRef:    dto.DecisionCoberturaRef,
		DecisionCoberturaHuella: dto.DecisionCoberturaHuella,
		VersionResultante:       dto.VersionResultante, EventoRef: dto.EventoRef,
		ActuacionRef: dto.ActuacionRef,
	}, nil
}

func normalizarReciboDecisionCoberturaPostgreSQL(
	dto reciboDecisionCoberturaO404E,
) reciboDecisionCoberturaO404E {
	dto.ConfirmadaEn = normalizarInstantePostgreSQL(dto.ConfirmadaEn)
	return dto
}

func igualesDecisionCoberturaO404E(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

func limpiarPruebasCanonicasDecisionCoberturaO404E(
	p *puertosct.DatosPruebasCanonicasOrdenConsumoCobertura,
) {
	if p == nil {
		return
	}
	borrarBytes(p.Peticion)
	borrarBytes(p.Resultado)
	borrarBytes(p.Atestacion)
	borrarBytes(p.ConfirmacionTCB)
	borrarBytes(p.Catalogo)
	borrarBytes(p.Verificador)
	borrarBytes(p.Resumen)
	*p = puertosct.DatosPruebasCanonicasOrdenConsumoCobertura{}
}

func limpiarCargaConfirmarDecisionCoberturaO404E(
	c *cargaConfirmarDecisionCoberturaO404E,
) {
	if c == nil {
		return
	}
	borrarBytes(c.DecisionVEC.DecisionCanonica)
	borrarBytes(c.DecisionVEC.MotivoCanonico)
	clear(c.DecisionVEC.Ambitos)
	clear(c.DecisionVEC.Atributos)
	c.DecisionVEC.Ambitos = nil
	c.DecisionVEC.Atributos = nil
	for i := range c.ConsumosC1 {
		limpiarPruebasC1DecisionCoberturaO404E(
			&c.ConsumosC1[i].Pruebas,
		)
		c.ConsumosC1[i] = consumoC1DecisionCoberturaO404E{}
	}
	if c.Denegacion != nil {
		clear(c.Denegacion.Ambitos)
		clear(c.Denegacion.Atributos)
		c.Denegacion.Ambitos = nil
		c.Denegacion.Atributos = nil
	}
	c.ConsumosC1 = nil
	*c = cargaConfirmarDecisionCoberturaO404E{}
}

func limpiarPruebasC1DecisionCoberturaO404E(
	p *pruebasCanonicasC1DecisionCoberturaO404E,
) {
	if p == nil {
		return
	}
	borrarBytes(p.Peticion)
	borrarBytes(p.Resultado)
	borrarBytes(p.Atestacion)
	borrarBytes(p.ConfirmacionTCB)
	borrarBytes(p.Catalogo)
	borrarBytes(p.Verificador)
	borrarBytes(p.Resumen)
	*p = pruebasCanonicasC1DecisionCoberturaO404E{}
}

func limpiarDecisionVECDecisionCoberturaO404E(
	d *decisionVECDecisionCoberturaO404E,
) {
	if d == nil {
		return
	}
	borrarBytes(d.DecisionCanonica)
	borrarBytes(d.MotivoCanonico)
	clear(d.Ambitos)
	clear(d.Atributos)
	d.Ambitos = nil
	d.Atributos = nil
	*d = decisionVECDecisionCoberturaO404E{}
}
