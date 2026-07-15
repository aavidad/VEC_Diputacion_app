package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const finalidadFlujoPrueba = "tramitar_solicitud_bolsa"

func catalogoEstadosFlujoPublicadoPrueba(t *testing.T) CatalogoConfigurable {
	t.Helper()
	fecha := time.Date(2026, time.July, 14, 8, 0, 0, 0, time.UTC)
	claves := []string{
		"borrador", "presentada", "en_revision", "admitida", "excluida",
		"subsanacion", "aislado", "espera_a", "espera_b",
	}
	entradas := make([]EntradaCatalogoConfigurable, 0, len(claves))
	for indice, clave := range claves {
		entradas = append(entradas, EntradaCatalogoConfigurable{
			Clave:        clave,
			Etiqueta:     "Estado " + clave,
			Orden:        (indice + 1) * 10,
			VigenteDesde: fecha,
		})
	}
	catalogo := CatalogoConfigurable{
		ID:             "bolsa.estados_solicitud",
		Version:        1,
		Revision:       1,
		ModuloID:       "bolsa",
		Nombre:         "Estados de solicitud",
		Descripcion:    "Estados gobernados del procedimiento de seleccion.",
		FuenteRef:      "bases:bolsa:2026",
		MotivoCreacion: "Configuracion inicial del procedimiento",
		Entradas:       entradas,
		Estado:         EstadoCatalogoBorrador,
		CreadoPor:      "tecnico-catalogos-1",
		CreadoEn:       fecha,
	}
	publicado, err := catalogo.Publicar(
		"responsable-catalogos-2",
		"aprobacion-catalogo-1",
		"Catalogo revisado por Seleccion Externa",
		fecha.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("publicar catalogo de prueba: %v", err)
	}
	return publicado
}

func estadoFlujoPrueba(t *testing.T, catalogo CatalogoConfigurable, clave string, orden int, terminal bool) EstadoFlujoConfigurable {
	t.Helper()
	huella, err := catalogo.HuellaContenidoSHA256()
	if err != nil {
		t.Fatalf("huella de catalogo: %v", err)
	}
	return EstadoFlujoConfigurable{
		Clave: clave,
		Catalogo: ReferenciaEntradaCatalogo{
			CatalogoID:           catalogo.ID,
			CatalogoVersion:      catalogo.Version,
			CatalogoHuellaSHA256: huella,
			EntradaClave:         clave,
		},
		Orden:    orden,
		Terminal: terminal,
		Atributos: map[string]string{
			"publicable": "si",
			"tono":       "informativo",
		},
	}
}

func definicionFlujoBorradorPrueba(t *testing.T, permiteFinalizacion bool) DefinicionFlujo {
	t.Helper()
	catalogo := catalogoEstadosFlujoPublicadoPrueba(t)
	fecha := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	definicion := DefinicionFlujo{
		ID:                              "bolsa.solicitud",
		Version:                         1,
		Revision:                        1,
		ModuloID:                        "bolsa",
		TipoEntidad:                     "solicitud",
		Nombre:                          "Tramitacion de solicitud de bolsa",
		Descripcion:                     "Flujo configurable de una solicitud de participacion.",
		FuenteRef:                       "bases:bolsa:2026",
		MotivoCreacion:                  "Implantar la convocatoria inicial",
		EstadoInicial:                   "borrador",
		AccionInicio:                    "bolsa.solicitud.crear",
		GarantiaInicio:                  AuthAssuranceSubstantial,
		PermiteFinalizacionTrasRetirada: permiteFinalizacion,
		Estados: []EstadoFlujoConfigurable{
			estadoFlujoPrueba(t, catalogo, "borrador", 10, false),
			estadoFlujoPrueba(t, catalogo, "presentada", 20, false),
			estadoFlujoPrueba(t, catalogo, "en_revision", 30, false),
			estadoFlujoPrueba(t, catalogo, "admitida", 40, true),
			estadoFlujoPrueba(t, catalogo, "excluida", 50, true),
		},
		Transiciones: []TransicionFlujoConfigurable{
			{
				Clave:          "presentar",
				Desde:          []string{"borrador"},
				Hacia:          "presentada",
				Accion:         "bolsa.solicitud.presentar",
				ReglaRef:       "regla:bolsa:presentar:v1",
				Prioridad:      10,
				GarantiaMinima: AuthAssuranceSubstantial,
				RequiereMotivo: true,
				Atributos: map[string]string{
					"notificar": "si",
					"visible":   "si",
				},
			},
			{
				Clave:          "iniciar_revision",
				Desde:          []string{"presentada"},
				Hacia:          "en_revision",
				Accion:         "bolsa.solicitud.revisar",
				ReglaRef:       "regla:bolsa:revisar:v1",
				Prioridad:      20,
				GarantiaMinima: AuthAssuranceHigh,
			},
			{
				Clave:              "admitir",
				Desde:              []string{"en_revision"},
				Hacia:              "admitida",
				Accion:             "bolsa.solicitud.admitir",
				ReglaRef:           "regla:bolsa:admitir:v1",
				Prioridad:          30,
				GarantiaMinima:     AuthAssuranceHigh,
				RequiereAprobacion: true,
			},
			{
				Clave:              "excluir",
				Desde:              []string{"en_revision"},
				Hacia:              "excluida",
				Accion:             "bolsa.solicitud.excluir",
				ReglaRef:           "regla:bolsa:excluir:v1",
				Prioridad:          40,
				GarantiaMinima:     AuthAssuranceHigh,
				RequiereAprobacion: true,
			},
		},
		Estado:    EstadoDefinicionFlujoBorrador,
		CreadaPor: "tecnico-flujos-1",
		CreadaEn:  fecha,
	}
	if err := definicion.Validar(); err != nil {
		t.Fatalf("definicion de prueba invalida: %v", err)
	}
	return definicion
}

func publicarFlujoPrueba(t *testing.T, permiteFinalizacion bool) DefinicionFlujo {
	t.Helper()
	borrador := definicionFlujoBorradorPrueba(t, permiteFinalizacion)
	publicada, err := borrador.Publicar(
		"responsable-flujos-2",
		"aprobacion-flujo-1",
		"Flujo revisado funcional y juridicamente",
		borrador.CreadaEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("publicar flujo de prueba: %v", err)
	}
	return publicada
}

func decisionReglaFlujoPrueba(
	t *testing.T,
	definicion DefinicionFlujo,
	instancia InstanciaFlujo,
	transicionClave string,
	concedida bool,
	actorID, correlacionRef string,
	evaluadaEn time.Time,
) DecisionReglaFlujo {
	t.Helper()
	transicion, err := definicion.ObtenerTransicion(transicionClave, instancia.EstadoActual)
	if err != nil {
		t.Fatalf("obtener transicion %q: %v", transicionClave, err)
	}
	huella, err := definicion.HuellaContenidoSHA256()
	if err != nil {
		t.Fatalf("huella de definicion: %v", err)
	}
	codigo := "regla_satisfecha"
	if !concedida {
		codigo = "regla_no_satisfecha"
	}
	return DecisionReglaFlujo{
		DecisionRef:                     "decision-regla-" + transicion.Clave + "-" + evaluadaEn.UTC().Format("150405"),
		Concedida:                       concedida,
		Codigo:                          codigo,
		DefinicionRef:                   definicion.Referencia(),
		DefinicionContenidoHuellaSHA256: huella,
		InstanciaRef:                    instancia.ID,
		InstanciaRevision:               instancia.Revision,
		EstadoOrigen:                    instancia.EstadoActual,
		TransicionClave:                 transicion.Clave,
		ReglaRef:                        transicion.ReglaRef,
		ActorID:                         actorID,
		Finalidad:                       finalidadFlujoPrueba,
		CorrelacionRef:                  correlacionRef,
		EntradaHuellaHMAC:               "hmac-sha256:regla:" + strings.Repeat("a", 64),
		ResultadoHuellaSHA256:           strings.Repeat("b", 64),
		EvaluadaEn:                      evaluadaEn.UTC(),
		ValidaHasta:                     evaluadaEn.UTC().Add(5 * time.Minute),
	}
}

func TestDefinicionFlujoAdmiteEstadosYTransicionesNuevosSinRecompilar(t *testing.T) {
	borrador := definicionFlujoBorradorPrueba(t, true)
	catalogo := catalogoEstadosFlujoPublicadoPrueba(t)
	borrador.Estados = append(borrador.Estados, estadoFlujoPrueba(t, catalogo, "subsanacion", 25, false))
	borrador.Transiciones = append(borrador.Transiciones,
		TransicionFlujoConfigurable{
			Clave:          "requerir_subsanacion",
			Desde:          []string{"presentada"},
			Hacia:          "subsanacion",
			Accion:         "bolsa.solicitud.requerir_subsanacion",
			ReglaRef:       "regla:bolsa:requerir-subsanacion:v1",
			Prioridad:      25,
			GarantiaMinima: AuthAssuranceHigh,
		},
		TransicionFlujoConfigurable{
			Clave:          "subsanar",
			Desde:          []string{"subsanacion"},
			Hacia:          "en_revision",
			Accion:         "bolsa.solicitud.subsanar",
			ReglaRef:       "regla:bolsa:subsanar:v1",
			Prioridad:      26,
			GarantiaMinima: AuthAssuranceSubstantial,
		},
	)
	publicada, err := borrador.Publicar(
		"responsable-flujos-2",
		"aprobacion-flujo-dinamico",
		"Incorporar subsanacion configurada desde administracion",
		borrador.CreadaEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("publicar flujo ampliado: %v", err)
	}
	transicion, err := publicada.ObtenerTransicion("subsanar", "subsanacion")
	if err != nil || transicion.Hacia != "en_revision" {
		t.Fatalf("transicion configurable = %+v, %v", transicion, err)
	}
}

func TestDefinicionFlujoCanonizaOrdenZonaHorariaYAtributos(t *testing.T) {
	borrador := definicionFlujoBorradorPrueba(t, true)
	huellaPrimera, err := borrador.HuellaSHA256()
	if err != nil {
		t.Fatalf("HuellaSHA256() error = %v", err)
	}

	reordenado := borrador
	reordenado.Estados = append([]EstadoFlujoConfigurable(nil), borrador.Estados...)
	for izquierda, derecha := 0, len(reordenado.Estados)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
		reordenado.Estados[izquierda], reordenado.Estados[derecha] = reordenado.Estados[derecha], reordenado.Estados[izquierda]
	}
	reordenado.Transiciones = append([]TransicionFlujoConfigurable(nil), borrador.Transiciones...)
	for izquierda, derecha := 0, len(reordenado.Transiciones)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
		reordenado.Transiciones[izquierda], reordenado.Transiciones[derecha] = reordenado.Transiciones[derecha], reordenado.Transiciones[izquierda]
	}
	reordenado.CreadaEn = reordenado.CreadaEn.In(time.FixedZone("zona-prueba", 2*60*60))
	reordenado.Estados[0].Atributos = map[string]string{"tono": "informativo", "publicable": "si"}
	reordenado.Transiciones[len(reordenado.Transiciones)-1].Atributos = map[string]string{"visible": "si", "notificar": "si"}

	huellaSegunda, err := reordenado.HuellaSHA256()
	if err != nil || huellaPrimera != huellaSegunda {
		t.Fatalf("huella no canonica: primera=%q segunda=%q error=%v", huellaPrimera, huellaSegunda, err)
	}
	canonico, err := reordenado.ClonarCanonico()
	if err != nil {
		t.Fatalf("ClonarCanonico() error = %v", err)
	}
	canonico.Estados[0].Atributos["tono"] = "alterado"
	if borrador.Estados[0].Atributos["tono"] != "informativo" {
		t.Fatal("el clon canonico comparte atributos con la fuente")
	}
}

func TestDefinicionFlujoExigeDobleControlRevisionOptimistaYOrdenTemporal(t *testing.T) {
	borrador := definicionFlujoBorradorPrueba(t, true)
	if _, err := borrador.Publicar(
		borrador.CreadaPor,
		"aprobacion-invalida",
		"Autopublicacion",
		borrador.CreadaEn.Add(time.Hour),
	); !errors.Is(err, ErrTransicionFlujoInvalida) {
		t.Fatalf("autopublicacion: error = %v", err)
	}

	actualizado, err := borrador.ActualizarBorrador(
		1,
		"tecnico-flujos-2",
		"Aclarar el alcance de la definicion",
		ConfiguracionBorradorFlujo{
			Nombre:                          borrador.Nombre,
			Descripcion:                     "Descripcion revisada por el equipo funcional.",
			FuenteRef:                       borrador.FuenteRef,
			EstadoInicial:                   borrador.EstadoInicial,
			AccionInicio:                    borrador.AccionInicio,
			GarantiaInicio:                  borrador.GarantiaInicio,
			PermiteFinalizacionTrasRetirada: true,
			Estados:                         borrador.Estados,
			Transiciones:                    borrador.Transiciones,
		},
		borrador.CreadaEn.Add(2*time.Hour),
	)
	if err != nil || actualizado.Revision != 2 {
		t.Fatalf("ActualizarBorrador() = rev %d, %v", actualizado.Revision, err)
	}
	if _, err := actualizado.ActualizarBorrador(
		1,
		"tecnico-flujos-3",
		"Revision obsoleta",
		ConfiguracionBorradorFlujo{
			Nombre:                          actualizado.Nombre,
			Descripcion:                     actualizado.Descripcion,
			FuenteRef:                       actualizado.FuenteRef,
			EstadoInicial:                   actualizado.EstadoInicial,
			AccionInicio:                    actualizado.AccionInicio,
			GarantiaInicio:                  actualizado.GarantiaInicio,
			PermiteFinalizacionTrasRetirada: true,
			Estados:                         actualizado.Estados,
			Transiciones:                    actualizado.Transiciones,
		},
		actualizado.UltimaModificacionEn.Add(time.Minute),
	); !errors.Is(err, ErrTransicionFlujoInvalida) {
		t.Fatalf("revision obsoleta: error = %v", err)
	}
	if _, err := actualizado.Publicar(
		actualizado.UltimaModificacionPor,
		"aprobacion-invalida-2",
		"Autopublicacion del modificador",
		actualizado.UltimaModificacionEn.Add(time.Hour),
	); !errors.Is(err, ErrTransicionFlujoInvalida) {
		t.Fatalf("publicacion por ultimo modificador: error = %v", err)
	}
	if _, err := actualizado.Publicar(
		"responsable-flujos-3",
		"aprobacion-temporal-invalida",
		"Publicacion anterior al cambio",
		actualizado.UltimaModificacionEn.Add(-time.Minute),
	); !errors.Is(err, ErrTransicionFlujoInvalida) {
		t.Fatalf("publicacion retroactiva: error = %v", err)
	}

	publicada, err := actualizado.Publicar(
		"responsable-flujos-3",
		"aprobacion-flujo-2",
		"Revision independiente superada",
		actualizado.UltimaModificacionEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("Publicar() error = %v", err)
	}
	nueva, err := publicada.NuevaVersion(
		2,
		"tecnico-flujos-4",
		"bases:bolsa:2027",
		"Preparar la siguiente convocatoria",
		publicada.PublicadaEn.Add(time.Hour),
	)
	if err != nil || nueva.VersionAnteriorRef != publicada.Referencia() ||
		nueva.Estado != EstadoDefinicionFlujoBorrador || !nueva.PermiteFinalizacionTrasRetirada {
		t.Fatalf("NuevaVersion() = %+v, %v", nueva, err)
	}
}

func TestDefinicionFlujoRechazaGrafosAdministrativosInseguros(t *testing.T) {
	catalogo := catalogoEstadosFlujoPublicadoPrueba(t)
	casos := []struct {
		nombre  string
		alterar func(*DefinicionFlujo)
	}{
		{
			nombre: "estado inalcanzable",
			alterar: func(d *DefinicionFlujo) {
				d.Estados = append(d.Estados, estadoFlujoPrueba(t, catalogo, "aislado", 60, true))
			},
		},
		{
			nombre: "terminal con salida",
			alterar: func(d *DefinicionFlujo) {
				d.Transiciones = append(d.Transiciones, TransicionFlujoConfigurable{
					Clave:          "reabrir_admitida",
					Desde:          []string{"admitida"},
					Hacia:          "excluida",
					Accion:         "bolsa.solicitud.reabrir",
					ReglaRef:       "regla:bolsa:reabrir:v1",
					Prioridad:      50,
					GarantiaMinima: AuthAssuranceHigh,
				})
			},
		},
		{
			nombre: "ruta ambigua",
			alterar: func(d *DefinicionFlujo) {
				duplicada := d.Transiciones[0]
				duplicada.Clave = "presentar_alternativa"
				d.Transiciones = append(d.Transiciones, duplicada)
			},
		},
		{
			nombre: "destino inexistente",
			alterar: func(d *DefinicionFlujo) {
				d.Transiciones[0].Hacia = "estado_inexistente"
			},
		},
		{
			nombre: "ciclo sin camino a terminal",
			alterar: func(d *DefinicionFlujo) {
				d.Estados = append(d.Estados,
					estadoFlujoPrueba(t, catalogo, "espera_a", 60, false),
					estadoFlujoPrueba(t, catalogo, "espera_b", 70, false),
				)
				d.Transiciones = append(d.Transiciones,
					TransicionFlujoConfigurable{
						Clave: "entrar_espera", Desde: []string{"presentada"}, Hacia: "espera_a",
						Accion: "bolsa.solicitud.esperar", ReglaRef: "regla:bolsa:esperar:v1",
						Prioridad: 60, GarantiaMinima: AuthAssuranceHigh,
					},
					TransicionFlujoConfigurable{
						Clave: "esperar_b", Desde: []string{"espera_a"}, Hacia: "espera_b",
						Accion: "bolsa.solicitud.esperar_b", ReglaRef: "regla:bolsa:esperar-b:v1",
						Prioridad: 70, GarantiaMinima: AuthAssuranceHigh,
					},
					TransicionFlujoConfigurable{
						Clave: "volver_espera_a", Desde: []string{"espera_b"}, Hacia: "espera_a",
						Accion: "bolsa.solicitud.volver_espera", ReglaRef: "regla:bolsa:volver-espera:v1",
						Prioridad: 80, GarantiaMinima: AuthAssuranceHigh,
					},
				)
			},
		},
	}

	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			borrador := definicionFlujoBorradorPrueba(t, true)
			caso.alterar(&borrador)
			if _, err := borrador.Publicar(
				"responsable-flujos-2",
				"aprobacion-grafo-1",
				"Intento de publicar un grafo invalido",
				borrador.CreadaEn.Add(time.Hour),
			); err == nil {
				t.Fatal("el grafo inseguro fue publicado")
			}
		})
	}
}

func TestHuellaContenidoFlujoPermaneceYHuellaGobiernoCambia(t *testing.T) {
	borrador := definicionFlujoBorradorPrueba(t, true)
	huellaContenidoBorrador, err := borrador.HuellaContenidoSHA256()
	if err != nil {
		t.Fatalf("huella de contenido borrador: %v", err)
	}
	huellaGobiernoBorrador, err := borrador.HuellaSHA256()
	if err != nil {
		t.Fatalf("huella de gobierno borrador: %v", err)
	}
	publicada, err := borrador.Publicar(
		"responsable-flujos-2",
		"aprobacion-flujo-huella",
		"Publicacion para comprobar huellas",
		borrador.CreadaEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("Publicar() error = %v", err)
	}
	huellaContenidoPublicada, _ := publicada.HuellaContenidoSHA256()
	huellaGobiernoPublicada, _ := publicada.HuellaSHA256()
	retirada, err := publicada.Retirar(
		"responsable-flujos-3",
		"aprobacion-retirada-huella",
		"Sustituida por una nueva version",
		publicada.PublicadaEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("Retirar() error = %v", err)
	}
	huellaContenidoRetirada, _ := retirada.HuellaContenidoSHA256()
	huellaGobiernoRetirada, _ := retirada.HuellaSHA256()
	if huellaContenidoBorrador != huellaContenidoPublicada || huellaContenidoPublicada != huellaContenidoRetirada {
		t.Fatalf("la huella semantica cambio con el gobierno: %q %q %q",
			huellaContenidoBorrador, huellaContenidoPublicada, huellaContenidoRetirada)
	}
	if huellaGobiernoBorrador == huellaGobiernoPublicada || huellaGobiernoPublicada == huellaGobiernoRetirada {
		t.Fatalf("la huella de gobierno no cambio: %q %q %q",
			huellaGobiernoBorrador, huellaGobiernoPublicada, huellaGobiernoRetirada)
	}
	alterada := borrador
	alterada.PermiteFinalizacionTrasRetirada = false
	huellaContenidoAlterada, err := alterada.HuellaContenidoSHA256()
	if err != nil || huellaContenidoAlterada == huellaContenidoBorrador {
		t.Fatalf("el cambio semantico no altero la huella: %q, %v", huellaContenidoAlterada, err)
	}
}

func TestInstanciaFlujoSoloComienzaConVersionPublicadaYNoRetroactiva(t *testing.T) {
	borrador := definicionFlujoBorradorPrueba(t, true)
	if _, err := IniciarInstanciaFlujo(
		borrador,
		"instancia-solicitud-1",
		"solicitud-1",
		"persona-1",
		borrador.CreadaEn.Add(time.Hour),
	); !errors.Is(err, ErrInstanciaFlujoInvalida) {
		t.Fatalf("inicio sobre borrador: error = %v", err)
	}
	publicada := publicarFlujoPrueba(t, true)
	if _, err := IniciarInstanciaFlujo(
		publicada,
		"instancia-solicitud-1",
		"solicitud-1",
		"persona-1",
		publicada.PublicadaEn.Add(-time.Nanosecond),
	); !errors.Is(err, ErrInstanciaFlujoInvalida) {
		t.Fatalf("inicio retroactivo: error = %v", err)
	}
	instancia, err := IniciarInstanciaFlujo(
		publicada,
		"instancia-solicitud-1",
		"solicitud-1",
		"persona-1",
		publicada.PublicadaEn,
	)
	if err != nil || instancia.EstadoActual != publicada.EstadoInicial || instancia.Revision != 1 {
		t.Fatalf("IniciarInstanciaFlujo() = %+v, %v", instancia, err)
	}
	huellaContenido, _ := publicada.HuellaContenidoSHA256()
	if instancia.DefinicionContenidoHuellaSHA256 != huellaContenido {
		t.Fatalf("huella fijada = %q, quiere %q", instancia.DefinicionContenidoHuellaSHA256, huellaContenido)
	}
}

func TestInstanciaFlujoAplicaTransicionYPreservaEvidencias(t *testing.T) {
	definicion := publicarFlujoPrueba(t, true)
	creadaEn := definicion.PublicadaEn.Add(time.Minute)
	instancia, err := IniciarInstanciaFlujo(
		definicion,
		"instancia-solicitud-2",
		"solicitud-2",
		"persona-2",
		creadaEn,
	)
	if err != nil {
		t.Fatalf("IniciarInstanciaFlujo() error = %v", err)
	}
	evaluadaEn := creadaEn.Add(time.Minute)
	decision := decisionReglaFlujoPrueba(t, definicion, instancia, "presentar", true, "persona-2", "correlacion-presentar-1", evaluadaEn)
	actualizada, cambio, err := instancia.AplicarTransicion(
		definicion,
		"presentar",
		decision,
		"autorizacion-presentar-1",
		"",
		"persona-2",
		finalidadFlujoPrueba,
		"Presentar la solicitud firmada",
		"correlacion-presentar-1",
		evaluadaEn.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("AplicarTransicion() error = %v", err)
	}
	if actualizada.EstadoActual != "presentada" || actualizada.Revision != 2 ||
		actualizada.UltimaDecisionReglaRef != decision.DecisionRef ||
		actualizada.UltimaAutorizacionRef != "autorizacion-presentar-1" {
		t.Fatalf("instancia actualizada inesperada: %+v", actualizada)
	}
	if cambio.EstadoAnterior != "borrador" || cambio.EstadoPosterior != "presentada" ||
		cambio.RevisionAnterior != 1 || cambio.RevisionPosterior != 2 ||
		cambio.HuellaAnterior == cambio.HuellaPosterior {
		t.Fatalf("cambio de estado inesperado: %+v", cambio)
	}
	if instancia.EstadoActual != "borrador" || instancia.Revision != 1 {
		t.Fatal("la transicion modifico la instancia original")
	}
}

func TestInstanciaFlujoDistingueReglaDenegadaDeDecisionInvalida(t *testing.T) {
	definicion := publicarFlujoPrueba(t, true)
	creadaEn := definicion.PublicadaEn.Add(time.Minute)
	instancia, err := IniciarInstanciaFlujo(
		definicion,
		"instancia-solicitud-3",
		"solicitud-3",
		"persona-3",
		creadaEn,
	)
	if err != nil {
		t.Fatalf("IniciarInstanciaFlujo() error = %v", err)
	}
	evaluadaEn := creadaEn.Add(time.Minute)
	denegada := decisionReglaFlujoPrueba(t, definicion, instancia, "presentar", false, "persona-3", "correlacion-1", evaluadaEn)
	if _, _, err := instancia.AplicarTransicion(
		definicion, "presentar", denegada, "autorizacion-1", "", "persona-3",
		finalidadFlujoPrueba, "Intento de presentacion", "correlacion-1", evaluadaEn.Add(time.Minute),
	); !errors.Is(err, ErrReglaFlujoDenegada) {
		t.Fatalf("regla denegada: error = %v", err)
	}

	invalida := decisionReglaFlujoPrueba(t, definicion, instancia, "presentar", true, "persona-3", "correlacion-2", evaluadaEn)
	invalida.DefinicionContenidoHuellaSHA256 = strings.Repeat("c", 64)
	if _, _, err := instancia.AplicarTransicion(
		definicion, "presentar", invalida, "autorizacion-2", "", "persona-3",
		finalidadFlujoPrueba, "Decision incompatible", "correlacion-2", evaluadaEn.Add(time.Minute),
	); !errors.Is(err, ErrDecisionReglaInvalida) {
		t.Fatalf("decision incompatible: error = %v", err)
	}

	expirada := decisionReglaFlujoPrueba(t, definicion, instancia, "presentar", true, "persona-3", "correlacion-3", evaluadaEn)
	if _, _, err := instancia.AplicarTransicion(
		definicion, "presentar", expirada, "autorizacion-3", "", "persona-3",
		finalidadFlujoPrueba, "Decision caducada", "correlacion-3", expirada.ValidaHasta,
	); !errors.Is(err, ErrDecisionReglaInvalida) {
		t.Fatalf("decision caducada: error = %v", err)
	}

	codigoExcesivo := decisionReglaFlujoPrueba(t, definicion, instancia, "presentar", true, "persona-3", "correlacion-codigo", evaluadaEn)
	codigoExcesivo.Codigo = strings.Repeat("x", maximoCaracteresEtiqueta+1)
	if err := codigoExcesivo.Validar(); !errors.Is(err, ErrDecisionReglaInvalida) {
		t.Fatalf("codigo excesivo: error = %v", err)
	}
}

func TestInstanciaFlujoExigeAprobacionConfigurada(t *testing.T) {
	definicion := publicarFlujoPrueba(t, true)
	instancia, err := IniciarInstanciaFlujo(
		definicion,
		"instancia-solicitud-4",
		"solicitud-4",
		"persona-4",
		definicion.PublicadaEn.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("IniciarInstanciaFlujo() error = %v", err)
	}
	instante := instancia.CreadaEn.Add(time.Minute)
	decisionPresentar := decisionReglaFlujoPrueba(t, definicion, instancia, "presentar", true, "persona-4", "correlacion-presentar-4", instante)
	instancia, _, err = instancia.AplicarTransicion(
		definicion, "presentar", decisionPresentar, "autorizacion-presentar-4", "", "persona-4",
		finalidadFlujoPrueba, "Presentar solicitud", "correlacion-presentar-4", instante.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("presentar: %v", err)
	}
	instante = instancia.ActualizadaEn.Add(time.Minute)
	decisionRevisar := decisionReglaFlujoPrueba(t, definicion, instancia, "iniciar_revision", true, "tecnico-rrhh-4", "correlacion-revisar-4", instante)
	instancia, _, err = instancia.AplicarTransicion(
		definicion, "iniciar_revision", decisionRevisar, "autorizacion-revisar-4", "", "tecnico-rrhh-4",
		finalidadFlujoPrueba, "Iniciar revision tecnica", "correlacion-revisar-4", instante.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("iniciar revision: %v", err)
	}
	instante = instancia.ActualizadaEn.Add(time.Minute)
	decisionAdmitir := decisionReglaFlujoPrueba(t, definicion, instancia, "admitir", true, "responsable-rrhh-4", "correlacion-admitir-4", instante)
	if _, _, err := instancia.AplicarTransicion(
		definicion, "admitir", decisionAdmitir, "autorizacion-admitir-4", "", "responsable-rrhh-4",
		finalidadFlujoPrueba, "Admitir tras la revision", "correlacion-admitir-4", instante.Add(time.Minute),
	); !errors.Is(err, ErrAprobacionFlujoRequerida) {
		t.Fatalf("admision sin aprobacion: error = %v", err)
	}
	admitida, _, err := instancia.AplicarTransicion(
		definicion, "admitir", decisionAdmitir, "autorizacion-admitir-4", "aprobacion-admitir-4", "responsable-rrhh-4",
		finalidadFlujoPrueba, "Admitir tras la revision", "correlacion-admitir-4", instante.Add(time.Minute),
	)
	if err != nil || admitida.EstadoActual != "admitida" || admitida.UltimaAprobacionRef != "aprobacion-admitir-4" {
		t.Fatalf("admision = %+v, %v", admitida, err)
	}
}

func TestDefinicionRetiradaBloqueaAltasYControlaFinalizacionEnCurso(t *testing.T) {
	publicada := publicarFlujoPrueba(t, true)
	instancia, err := IniciarInstanciaFlujo(
		publicada,
		"instancia-solicitud-5",
		"solicitud-5",
		"persona-5",
		publicada.PublicadaEn.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("iniciar instancia: %v", err)
	}
	retirada, err := publicada.Retirar(
		"responsable-flujos-3",
		"aprobacion-retirada-1",
		"Sustituida por la version siguiente",
		publicada.PublicadaEn.Add(10*time.Minute),
	)
	if err != nil {
		t.Fatalf("retirar definicion: %v", err)
	}
	if _, err := IniciarInstanciaFlujo(
		retirada,
		"instancia-solicitud-nueva",
		"solicitud-nueva",
		"persona-nueva",
		retirada.RetiradaEn.Add(time.Minute),
	); !errors.Is(err, ErrInstanciaFlujoInvalida) {
		t.Fatalf("alta sobre version retirada: error = %v", err)
	}
	evaluadaEn := retirada.RetiradaEn.Add(time.Minute)
	decision := decisionReglaFlujoPrueba(t, retirada, instancia, "presentar", true, "persona-5", "correlacion-retirada-1", evaluadaEn)
	continuada, _, err := instancia.AplicarTransicion(
		retirada, "presentar", decision, "autorizacion-retirada-1", "", "persona-5",
		finalidadFlujoPrueba, "Finalizar expediente iniciado", "correlacion-retirada-1", evaluadaEn.Add(time.Minute),
	)
	if err != nil || continuada.EstadoActual != "presentada" {
		t.Fatalf("continuacion permitida = %+v, %v", continuada, err)
	}

	publicadaBloqueante := publicarFlujoPrueba(t, false)
	instanciaBloqueada, err := IniciarInstanciaFlujo(
		publicadaBloqueante,
		"instancia-solicitud-6",
		"solicitud-6",
		"persona-6",
		publicadaBloqueante.PublicadaEn.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("iniciar instancia bloqueable: %v", err)
	}
	retiradaBloqueante, err := publicadaBloqueante.Retirar(
		"responsable-flujos-3",
		"aprobacion-retirada-2",
		"Retirada inmediata por incidencia normativa",
		publicadaBloqueante.PublicadaEn.Add(10*time.Minute),
	)
	if err != nil {
		t.Fatalf("retirar definicion bloqueante: %v", err)
	}
	evaluadaEn = retiradaBloqueante.RetiradaEn.Add(time.Minute)
	decisionBloqueada := decisionReglaFlujoPrueba(t, publicadaBloqueante, instanciaBloqueada, "presentar", true, "persona-6", "correlacion-retirada-2", evaluadaEn)
	if _, _, err := instanciaBloqueada.AplicarTransicion(
		retiradaBloqueante, "presentar", decisionBloqueada, "autorizacion-retirada-2", "", "persona-6",
		finalidadFlujoPrueba, "Intento tras retirada bloqueante", "correlacion-retirada-2", evaluadaEn.Add(time.Minute),
	); !errors.Is(err, ErrDefinicionFlujoNoPublicada) {
		t.Fatalf("continuacion bloqueada: error = %v", err)
	}
}
