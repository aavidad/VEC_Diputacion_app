package domain

import (
	"errors"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestDecisionCoberturaGobernadaAltaYRectificacionAppendOnly(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	decidido := adoptarDecisionGobernada(t, base, propuesta)
	anterior := decidido.Clonar()
	nuevaPropuesta := propuestaDecisionParaExpediente(
		t,
		decidido,
		decidido.ActualizadoEn.Add(time.Minute),
	)
	rectificacion := datosRectificacion(decidido)
	rectificacion.ViaElegida = "via_alternativa_configurable"
	acto := actuacionDecision(
		string(AccionRectificarCoberturaGobernada),
		"actor_rrhh_rectificador_02",
		decidido.FaseActual,
		decidido.ActualizadoEn.Add(2*time.Minute),
	)

	rectificado, err := decidido.RectificarDecisionCoberturaGobernada(
		decidido.Version,
		rectificacion,
		nuevaPropuesta,
		acto,
	)
	if err != nil {
		t.Fatalf("rectificar: %v", err)
	}
	if rectificado.Version != decidido.Version+1 ||
		len(rectificado.DecisionesCobertura) != 2 ||
		rectificado.ViaCobertura.ViaClave != rectificacion.ViaElegida ||
		rectificado.DecisionesCobertura[1].PredecesoraRef !=
			rectificado.DecisionesCobertura[0].Referencia {
		t.Fatalf("cadena inesperada: %#v", rectificado.DecisionesCobertura)
	}
	if decidido.Version != anterior.Version ||
		len(decidido.DecisionesCobertura) != 1 ||
		decidido.ViaCobertura.ViaClave != anterior.ViaCobertura.ViaClave {
		t.Fatal("la rectificación reescribió la instantánea anterior")
	}
	if err := rectificado.Validar(); err != nil {
		t.Fatalf("expediente rectificado inválido: %v", err)
	}
}

func TestRectificacionDecisionExigeActorDistintoYPredecesoraExacta(
	t *testing.T,
) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	decidido := adoptarDecisionGobernada(t, base, propuesta)
	nueva := propuestaDecisionParaExpediente(
		t,
		decidido,
		decidido.ActualizadoEn.Add(time.Minute),
	)
	datos := datosRectificacion(decidido)
	datos.ViaElegida = "via_alternativa_configurable"
	acto := actuacionDecision(
		string(AccionRectificarCoberturaGobernada),
		decidido.DecisionesCobertura[0].ActorRef,
		decidido.FaseActual,
		decidido.ActualizadoEn.Add(2*time.Minute),
	)
	if _, err := decidido.RectificarDecisionCoberturaGobernada(
		decidido.Version,
		datos,
		nueva,
		acto,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("mismo actor: %v", err)
	}

	acto.ActorRef = "actor_rrhh_rectificador_02"
	datos.PredecesoraHuellaSHA256 = cadena64("f")
	if _, err := decidido.RectificarDecisionCoberturaGobernada(
		decidido.Version,
		datos,
		nueva,
		acto,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("predecesora falsa: %v", err)
	}
}

func TestRectificacionDecisionRechazaNoOpYConflictoCAS(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	decidido := adoptarDecisionGobernada(t, base, propuesta)
	nueva := propuestaDecisionParaExpediente(
		t,
		decidido,
		decidido.ActualizadoEn.Add(time.Minute),
	)
	datos := datosRectificacion(decidido)
	acto := actuacionDecision(
		string(AccionRectificarCoberturaGobernada),
		"actor_rrhh_rectificador_02",
		decidido.FaseActual,
		decidido.ActualizadoEn.Add(2*time.Minute),
	)
	if _, err := decidido.RectificarDecisionCoberturaGobernada(
		decidido.Version,
		datos,
		nueva,
		acto,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("no-op: %v", err)
	}
	datos.ViaElegida = "via_alternativa_configurable"
	if _, err := decidido.RectificarDecisionCoberturaGobernada(
		decidido.Version+1,
		datos,
		nueva,
		acto,
	); !errors.Is(err, ErrVersionEnConflicto) {
		t.Fatalf("CAS: %v", err)
	}
}

func TestAltaDecisionGobernadaExigeCASYReservaAccion(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	acto := actuacionDecision(
		string(AccionDecidirCoberturaGobernada),
		"actor_rrhh_decisor_01",
		"asignacion_unidad",
		propuesta.Publicacion().GeneradaEn.Add(time.Minute),
	)
	datos := DatosAdoptarDecisionCobertura{
		PerfilRef:  "perfil_rrhh_decisor_01",
		ViaElegida: propuesta.ViaPropuesta(),
	}
	if _, err := base.RegistrarDecisionCoberturaGobernada(
		base.Version+1,
		datos,
		propuesta,
		acto,
	); !errors.Is(err, ErrVersionEnConflicto) {
		t.Fatalf("CAS inicial: %v", err)
	}
	if _, err := base.RegistrarViaCobertura(
		base.Version,
		decisionValida(),
		acto,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("acción O4-03 entró por contrato legado: %v", err)
	}
}

func TestRectificacionDecisionProhibidaTrasCualquierEfectoPosterior(
	t *testing.T,
) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	decidido := adoptarDecisionGobernada(t, base, propuesta)
	nueva := propuestaDecisionParaExpediente(
		t,
		decidido,
		decidido.ActualizadoEn.Add(2*time.Minute),
	)
	datos := datosRectificacion(decidido)
	datos.ViaElegida = "via_alternativa_configurable"

	t.Run("actuacion posterior", func(t *testing.T) {
		actoPosterior := actuacionDecision(
			"cobertura.efecto_posterior",
			"actor_efecto_posterior_01",
			decidido.FaseActual,
			decidido.ActualizadoEn.Add(time.Minute),
		)
		conEfecto, err := decidido.confirmarTransicion(actoPosterior)
		if err != nil {
			t.Fatalf("preparar efecto: %v", err)
		}
		acto := actuacionDecision(
			string(AccionRectificarCoberturaGobernada),
			"actor_rrhh_rectificador_02",
			conEfecto.FaseActual,
			conEfecto.ActualizadoEn.Add(2*time.Minute),
		)
		if _, err := conEfecto.RectificarDecisionCoberturaGobernada(
			conEfecto.Version,
			datos,
			nueva,
			acto,
		); !errors.Is(err, ErrTransicionInvalida) {
			t.Fatalf("rectificó tras efecto: %v", err)
		}
	})

	t.Run("asignacion", func(t *testing.T) {
		instante := decidido.ActualizadoEn.Add(time.Minute)
		asignado, err := decidido.RegistrarAsignacion(
			decidido.Version,
			AsignacionUnidad{
				UnidadRef:       "unidad_destino_gobernada_01",
				ResponsableRef:  "responsable_opaco_gobernado_01",
				NotificacionRef: "notificacion_gobernada_01",
				AsignadaEn:      instante,
			},
			actuacionDecision(
				"unidad.asignada",
				"actor_asignacion_01",
				"unidad_gestora",
				instante,
			),
		)
		if err != nil {
			t.Fatalf("asignar: %v", err)
		}
		acto := actuacionDecision(
			string(AccionRectificarCoberturaGobernada),
			"actor_rrhh_rectificador_02",
			asignado.FaseActual,
			asignado.ActualizadoEn.Add(time.Minute),
		)
		if _, err := asignado.RectificarDecisionCoberturaGobernada(
			asignado.Version,
			datos,
			nueva,
			acto,
		); !errors.Is(err, ErrTransicionInvalida) {
			t.Fatalf("rectificó tras asignación: %v", err)
		}
	})
}

func TestDecisionAlternativaExigeMotivoGobernadoI18n(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	acto := actuacionDecision(
		string(AccionDecidirCoberturaGobernada),
		"actor_rrhh_decisor_01",
		"asignacion_unidad",
		propuesta.Publicacion().GeneradaEn.Add(time.Minute),
	)
	if _, err := base.RegistrarDecisionCoberturaGobernada(
		base.Version,
		DatosAdoptarDecisionCobertura{
			PerfilRef:  "perfil_rrhh_decisor_01",
			ViaElegida: "via_alternativa_configurable",
		},
		propuesta,
		acto,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("desviación sin motivo: %v", err)
	}

	motivo := MotivoGobernadoDecisionCobertura{
		ReferenciaCatalogo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:           "motivos_cobertura",
			CatalogoVersion:      7,
			CatalogoHuellaSHA256: cadena64("8"),
			EntradaClave:         "desviacion_recomendacion",
		},
		ClaveI18n: "cobertura.motivo.desviacion",
	}
	decidido, err := base.RegistrarDecisionCoberturaGobernada(
		base.Version,
		DatosAdoptarDecisionCobertura{
			PerfilRef:  "perfil_rrhh_decisor_01",
			ViaElegida: "via_alternativa_configurable",
			Motivo:     motivo,
		},
		propuesta,
		acto,
	)
	if err != nil {
		t.Fatalf("desviación gobernada: %v", err)
	}
	if decidido.DecisionesCobertura[0].Motivo != motivo {
		t.Fatal("no conservó identidad inmutable e i18n del motivo")
	}
	adulterada := decidido.DecisionesCobertura[0]
	adulterada.Motivo.ReferenciaCatalogo.CatalogoVersion++
	if _, err := RestaurarDecisionCoberturaGobernada(adulterada); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("aceptó motivo adulterado: %v", err)
	}
}

func TestDecisionCoberturaLimitesDePeriodoEInstantes(t *testing.T) {
	base, propuesta := prepararDecisionCoberturaGobernada(t)
	publicacion := propuesta.Publicacion()
	actoLimite := actuacionDecision(
		string(AccionDecidirCoberturaGobernada),
		"actor_rrhh_decisor_01",
		"asignacion_unidad",
		publicacion.ValidaHasta,
	)
	if _, err := base.RegistrarDecisionCoberturaGobernada(
		base.Version,
		DatosAdoptarDecisionCobertura{
			PerfilRef:  "perfil_rrhh_decisor_01",
			ViaElegida: publicacion.ViaPropuesta,
		},
		propuesta,
		actoLimite,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("aceptó límite exclusivo: %v", err)
	}

	datos := datosPropuestaDecisionCoberturaPrueba(t)
	datos.OrganizacionRef = base.OrganizacionRef
	datos.ExpedienteRef = base.Referencia
	datos.VersionExpediente = base.Version
	datos.CategoriaRef = base.Analisis.CategoriaRef
	datos.Periodo = base.Analisis.Periodo
	datos.Periodo.Fin = datos.Periodo.Fin.AddDate(0, 0, 1)
	borrador := borradorPoliticaDecisionCoberturaPrueba(datos.Catalogo)
	borrador.OrganizacionRef = base.OrganizacionRef
	datos.Politica, _ = PublicarPoliticaDecisionCobertura(
		borrador,
		datos.Catalogo,
	)
	otra, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatalf("crear propuesta con otro periodo: %v", err)
	}
	acto := actuacionDecision(
		string(AccionDecidirCoberturaGobernada),
		"actor_rrhh_decisor_01",
		"asignacion_unidad",
		otra.Publicacion().GeneradaEn.Add(time.Minute),
	)
	if _, err := base.RegistrarDecisionCoberturaGobernada(
		base.Version,
		DatosAdoptarDecisionCobertura{
			PerfilRef:  "perfil_rrhh_decisor_01",
			ViaElegida: otra.ViaPropuesta(),
		},
		otra,
		acto,
	); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("aceptó periodo ajeno al análisis: %v", err)
	}
}

func prepararDecisionCoberturaGobernada(
	t *testing.T,
) (Expediente, PropuestaDecisionCobertura) {
	t.Helper()
	expediente := expedienteConAnalisisRehidratado(t, RCValidada)
	return expediente, propuestaDecisionParaExpediente(
		t,
		expediente,
		instantePoliticaCoberturaPrueba.Add(time.Minute),
	)
}

func propuestaDecisionParaExpediente(
	t *testing.T,
	expediente Expediente,
	generadaEn time.Time,
) PropuestaDecisionCobertura {
	t.Helper()
	datos := datosPropuestaDecisionCoberturaPrueba(t)
	datos.OrganizacionRef = expediente.OrganizacionRef
	datos.ExpedienteRef = expediente.Referencia
	datos.VersionExpediente = expediente.Version
	datos.CategoriaRef = expediente.Analisis.CategoriaRef
	datos.Periodo = expediente.Analisis.Periodo
	datos.GeneradaEn = generadaEn
	datos.ValidaHasta = generadaEn.Add(8 * time.Minute)
	borrador := borradorPoliticaDecisionCoberturaPrueba(datos.Catalogo)
	borrador.OrganizacionRef = expediente.OrganizacionRef
	politica, err := PublicarPoliticaDecisionCobertura(
		borrador,
		datos.Catalogo,
	)
	if err != nil {
		t.Fatalf("publicar política: %v", err)
	}
	datos.Politica = politica
	propuesta, err := CrearPropuestaDecisionCobertura(datos)
	if err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}
	return propuesta
}

func adoptarDecisionGobernada(
	t *testing.T,
	expediente Expediente,
	propuesta PropuestaDecisionCobertura,
) Expediente {
	t.Helper()
	acto := actuacionDecision(
		string(AccionDecidirCoberturaGobernada),
		"actor_rrhh_decisor_01",
		"asignacion_unidad",
		propuesta.Publicacion().GeneradaEn.Add(time.Minute),
	)
	decidido, err := expediente.RegistrarDecisionCoberturaGobernada(
		expediente.Version,
		DatosAdoptarDecisionCobertura{
			PerfilRef:  "perfil_rrhh_decisor_01",
			ViaElegida: propuesta.ViaPropuesta(),
		},
		propuesta,
		acto,
	)
	if err != nil {
		t.Fatalf("adoptar decisión: %v", err)
	}
	return decidido
}

func datosRectificacion(e Expediente) DatosRectificarDecisionCobertura {
	anterior := e.DecisionesCobertura[len(e.DecisionesCobertura)-1]
	return DatosRectificarDecisionCobertura{
		PerfilRef:  "perfil_rrhh_rectificador_02",
		ViaElegida: anterior.ViaElegida,
		Motivo: MotivoGobernadoDecisionCobertura{
			ReferenciaCatalogo: dominiovec.ReferenciaEntradaCatalogo{
				CatalogoID:           "motivos_cobertura",
				CatalogoVersion:      7,
				CatalogoHuellaSHA256: cadena64("9"),
				EntradaClave:         "rectificacion_decision",
			},
			ClaveI18n: "cobertura.motivo.rectificacion",
		},
		PredecesoraRef:          anterior.Referencia,
		PredecesoraHuellaSHA256: anterior.HuellaSHA256,
	}
}

func actuacionDecision(
	accion string,
	actor string,
	fase ClaveFase,
	instante time.Time,
) DatosActuacion {
	return DatosActuacion{
		AccionClave: ClaveCatalogo(accion), ActorRef: actor,
		UnidadRef:   "unidad_rrhh_gobernada_01",
		ReciboRef:   "recibo:" + accion,
		RealizadaEn: instante, FaseDestino: fase,
		EstadoDestino: EstadoEnCurso,
	}
}
