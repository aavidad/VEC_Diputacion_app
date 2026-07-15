package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSolicitudAutorizacionRechazaContextoNoCanonicoONoAcotado(t *testing.T) {
	base := SolicitudAutorizacion{
		Principal:       Principal{ID: "persona:uno", AuthMethod: AuthMethodCertificate, AuthAssurance: AuthAssuranceHigh},
		PerfilActivoRef: "perfil:uno", Accion: "vec.documentos.leer",
		Recurso:   RecursoAutorizable{Referencia: "documento:uno", ModuloID: "vec", Tipo: "documento"},
		Finalidad: "tramitar_expediente", CorrelacionRef: "correlacion:uno", Motivo: "Consulta necesaria",
	}
	if err := base.Validar(); err != nil {
		t.Fatalf("solicitud valida: %v", err)
	}
	pruebas := []struct {
		nombre string
		muta   func(*SolicitudAutorizacion)
	}{
		{"accion con espacio", func(s *SolicitudAutorizacion) { s.Accion = "vec.documentos leer" }},
		{"accion con comodin parcial", func(s *SolicitudAutorizacion) { s.Accion = "vec.documentos.*" }},
		{"accion con marca bidi", func(s *SolicitudAutorizacion) { s.Accion = "vec.documentos.\u202eleer" }},
		{"referencia con ancho cero", func(s *SolicitudAutorizacion) { s.Recurso.Referencia = "documento:\u200buno" }},
		{"referencia con salto", func(s *SolicitudAutorizacion) { s.Recurso.Referencia = "documento:uno\notro" }},
		{"referencia con comodin parcial", func(s *SolicitudAutorizacion) { s.Recurso.Referencia = "documento:*" }},
		{"atributo enorme", func(s *SolicitudAutorizacion) {
			s.Recurso.Atributos = map[string]string{"dato": strings.Repeat("x", 513)}
		}},
		{"clave de ambito no canonica", func(s *SolicitudAutorizacion) {
			s.Recurso.Ambitos = map[string]string{"proceso actual": "proceso:uno"}
		}},
		{"valor de ambito con comodin", func(s *SolicitudAutorizacion) {
			s.Recurso.Ambitos = map[string]string{"proceso": "proceso:*"}
		}},
		{"motivo de control", func(s *SolicitudAutorizacion) { s.Motivo = "consulta\x00oculta" }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			solicitud := base
			prueba.muta(&solicitud)
			if err := solicitud.Validar(); err == nil {
				t.Fatal("se acepto contexto de autorizacion no canonico")
			}
		})
	}
}

func TestGarantiaAutenticacionMasAltaNuncaRebaja(t *testing.T) {
	garantia, err := GarantiaAutenticacionMasAlta(AuthAssuranceHigh, AuthAssuranceLow)
	if err != nil {
		t.Fatalf("elevar garantia: %v", err)
	}
	if garantia != AuthAssuranceHigh {
		t.Fatalf("la politica rebajo la garantia: %q", garantia)
	}
	if CumpleGarantiaAutenticacion(AuthAssuranceSubstantial, garantia) {
		t.Fatal("una autenticacion sustancial no debe satisfacer garantia alta")
	}
}

func TestAsignacionPerfilExigeAmbitoExplicito(t *testing.T) {
	asignacion := asignacionPerfilValidaPrueba()
	if !asignacion.Cubre(RecursoAutorizable{
		Referencia: "expediente:1",
		ModuloID:   "bolsa",
		Tipo:       "expediente",
		Ambitos:    map[string]string{"unidad": "seleccion"},
	}) {
		t.Fatal("la unidad asignada debe estar cubierta")
	}
	if asignacion.Cubre(RecursoAutorizable{
		Referencia: "expediente:2",
		ModuloID:   "bolsa",
		Tipo:       "expediente",
		Ambitos:    map[string]string{"unidad": "nominas"},
	}) {
		t.Fatal("una unidad distinta no debe quedar cubierta")
	}
	if asignacion.Cubre(RecursoAutorizable{
		Referencia: "expediente:3",
		ModuloID:   "bolsa",
		Tipo:       "expediente",
		Ambitos:    map[string]string{"unidad": " seleccion"},
	}) {
		t.Fatal("un ambito no canonico no debe normalizarse para conceder acceso")
	}
	if asignacion.Cubre(RecursoAutorizable{
		Referencia: "expediente:4", ModuloID: "bolsa", Tipo: "expediente",
		Ambitos: map[string]string{"unidad": "seleccion", "provincia": "granada"},
	}) {
		t.Fatal("una dimension nueva no debe quedar implicitamente sin restriccion")
	}
	if asignacion.Cubre(RecursoAutorizable{
		Referencia: "expediente:5", ModuloID: "bolsa", Tipo: "expediente",
		Ambitos: map[string]string{},
	}) {
		t.Fatal("la ausencia del ambito esperado no debe equivaler a acceso ilimitado")
	}

	global := asignacion
	global.Ambitos = []AmbitoPerfil{{Clave: "global", Valores: []string{"*"}}}
	if global.Cubre(RecursoAutorizable{
		Referencia: "expediente:6", ModuloID: "bolsa", Tipo: "expediente",
		Ambitos: map[string]string{"unidad": "nominas", "provincia": "granada"},
	}) {
		t.Fatal("un comodin global no debe ampliar acceso a recursos presentes o futuros")
	}
}

func TestCoincidenciasAutorizacionSonLiterales(t *testing.T) {
	concesion := versionRolValidaPrueba().Concesiones[0]
	if concesion.AdmiteFinalidad(" gestion_bolsa") || concesion.AdmiteFinalidad("gestion_bolsa ") {
		t.Fatal("una finalidad no canonica se normalizo para conceder acceso")
	}

	politica := PoliticaRestrictiva{
		Acciones:     []string{"bolsa.expediente.leer"},
		Modulos:      []string{"bolsa"},
		TiposRecurso: []string{"expediente"},
	}
	solicitud := SolicitudAutorizacion{
		Accion: " bolsa.expediente.leer",
		Recurso: RecursoAutorizable{
			ModuloID: "bolsa",
			Tipo:     "expediente",
		},
	}
	if politica.AplicaA(solicitud) {
		t.Fatal("una accion no canonica se normalizo al aplicar la politica")
	}
}

func TestConfiguracionPublicadaRechazaMetadatosOpcionalesNoCanonicos(t *testing.T) {
	version := versionRolValidaPrueba()
	version.RetiradaPor = " "
	if err := version.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
		t.Fatalf("version publicada con retirada no canonica aceptada: %v", err)
	}

	asignacion := asignacionPerfilValidaPrueba()
	asignacion.RevocacionRef = " "
	if err := asignacion.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
		t.Fatalf("asignacion activa con revocacion no canonica aceptada: %v", err)
	}
}

func TestReferenciasNoCorrigenIdentificadoresInvalidos(t *testing.T) {
	version := versionRolValidaPrueba()
	version.RolID = " tecnico_bolsa"
	if referencia := version.Referencia(); referencia != "rol: tecnico_bolsa:v1" {
		t.Fatalf("la referencia normalizo el identificador: %q", referencia)
	}

	asignacion := asignacionPerfilValidaPrueba()
	asignacion.AsignacionID = "asignacion_invalida "
	if referencia := asignacion.Referencia(); referencia != "asignacion:asignacion_invalida :v1" {
		t.Fatalf("la referencia normalizo el identificador: %q", referencia)
	}
}

func TestConcesionRolRechazaComodinesQueAmpliarianPermisosFuturos(t *testing.T) {
	base := versionRolValidaPrueba().Concesiones[0]
	casos := []struct {
		nombre   string
		preparar func(*ConcesionRol)
	}{
		{nombre: "accion", preparar: func(c *ConcesionRol) { c.Accion = "*" }},
		{nombre: "accion con comodin parcial", preparar: func(c *ConcesionRol) { c.Accion = "bolsa.*" }},
		{nombre: "accion no canonica", preparar: func(c *ConcesionRol) { c.Accion = " bolsa.expediente.leer" }},
		{nombre: "tipo de recurso", preparar: func(c *ConcesionRol) { c.TipoRecurso = "*" }},
		{nombre: "tipo con comodin parcial", preparar: func(c *ConcesionRol) { c.TipoRecurso = "expediente:*" }},
		{nombre: "modulo", preparar: func(c *ConcesionRol) { c.ModuloID = "*" }},
		{nombre: "tipo no canonico", preparar: func(c *ConcesionRol) { c.TipoRecurso = "expediente\n" }},
		{nombre: "finalidad", preparar: func(c *ConcesionRol) { c.Finalidades = []string{"*"} }},
		{nombre: "finalidad con comodin parcial", preparar: func(c *ConcesionRol) { c.Finalidades = []string{"seleccion_*"} }},
		{nombre: "campo", preparar: func(c *ConcesionRol) { c.CamposPermitidos = []string{"*"} }},
		{nombre: "campo con comodin parcial", preparar: func(c *ConcesionRol) { c.CamposPermitidos = []string{"datos.*"} }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			concesion := base
			concesion.Finalidades = append([]string(nil), base.Finalidades...)
			concesion.CamposPermitidos = append([]string(nil), base.CamposPermitidos...)
			caso.preparar(&concesion)
			if err := concesion.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
				t.Fatalf("comodin %s aceptado: %v", caso.nombre, err)
			}
		})
	}
}

func TestPoliticaRestrictivaSoloAdmiteComodinRestrictivoCompleto(t *testing.T) {
	politica := politicaRestrictivaValidaPrueba("denegacion_global")
	politica.Efecto = EfectoPoliticaDenegar
	politica.Acciones = []string{"*"}
	politica.Modulos = []string{"*"}
	politica.TiposRecurso = []string{"*"}
	if err := politica.Validar(); err != nil {
		t.Fatalf("el comodin completo de una denegacion debe ser valido: %v", err)
	}

	for _, parcial := range []struct {
		nombre string
		muta   func(*PoliticaRestrictiva)
	}{
		{"accion", func(p *PoliticaRestrictiva) { p.Acciones = []string{"bolsa.*"} }},
		{"modulo", func(p *PoliticaRestrictiva) { p.Modulos = []string{"vec*"} }},
		{"tipo", func(p *PoliticaRestrictiva) { p.TiposRecurso = []string{"expediente:*"} }},
		{"finalidad", func(p *PoliticaRestrictiva) { p.FinalidadesPermitidas = []string{"seleccion_*"} }},
	} {
		t.Run(parcial.nombre, func(t *testing.T) {
			copia := politica
			parcial.muta(&copia)
			if err := copia.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
				t.Fatalf("comodin restrictivo parcial aceptado: %v", err)
			}
		})
	}
}

func TestAsignacionPerfilRechazaTodoComodinPositivoYAmbitoGlobalHeredado(t *testing.T) {
	for _, ambito := range []AmbitoPerfil{
		{Clave: "unidad", Valores: []string{"*"}},
		{Clave: "unidad", Valores: []string{"rrhh:*"}},
		{Clave: "unidad:*", Valores: []string{"rrhh"}},
		{Clave: "global", Valores: []string{"seleccion"}},
		{Clave: "global", Valores: []string{"*"}},
		{Clave: "global", Valores: []string{"*", "seleccion"}},
	} {
		if err := ambito.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
			t.Fatalf("ambito con comodin o global heredado aceptado: %+v, %v", ambito, err)
		}
	}
	mezclada := asignacionPerfilValidaPrueba()
	mezclada.Ambitos = []AmbitoPerfil{
		{Clave: "global", Valores: []string{"*"}},
		{Clave: "unidad", Valores: []string{"seleccion"}},
	}
	if err := mezclada.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
		t.Fatalf("ambito global mezclado con dimensiones aceptado: %v", err)
	}
}

func TestDecisionAutorizacionRechazaVigenciaLarga(t *testing.T) {
	ahora := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	decision := DecisionAutorizacion{
		DecisionRef:     "decision:interna",
		Codigo:          "denegada",
		PrincipalID:     "persona-1",
		PerfilActivoRef: "perfil:persona-1:bolsa",
		Accion:          "bolsa.expediente.leer",
		RecursoRef:      "expediente:1",
		Finalidad:       "gestion_bolsa",
		CorrelacionRef:  "corr-1",
		EmitidaEn:       ahora,
		ValidaHasta:     ahora.Add(VigenciaMaximaDecisionAutorizacion + time.Second),
	}
	if err := decision.Validar(); !errors.Is(err, ErrDecisionAutorizacionInvalida) {
		t.Fatalf("se esperaba rechazo de vigencia larga, recibido %v", err)
	}
}

func TestDecisionAutorizacionExigeResultadoYHuellasCoherentes(t *testing.T) {
	ahora := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	base := DecisionAutorizacion{
		DecisionRef:               "decision:interna",
		Concedida:                 true,
		Codigo:                    "concedida",
		PrincipalID:               "per_0123456789abcdefghijkl",
		PerfilActivoRef:           "prf_0123456789abcdefghijkl",
		Accion:                    "bolsa.expediente.leer",
		RecursoRef:                "expediente:1",
		Finalidad:                 "gestion_bolsa",
		CorrelacionRef:            "corr-1",
		VinculoAutenticacionActor: vinculoAutenticacionActorV1Prueba(t, ahora),
		AsignacionRef:             "asignacion:persona-1:v1",
		AsignacionHuellaSHA256:    strings.Repeat("a", 64),
		VersionRolRef:             "rol:tecnico:v1",
		VersionRolHuellaSHA256:    strings.Repeat("b", 64),
		GarantiaMinima:            AuthAssuranceHigh,
		PoliticasRefs:             []string{"politica:minimizacion:v1"},
		PoliticasHuellasSHA256:    map[string]string{"politica:minimizacion:v1": strings.Repeat("c", 64)},
		EmitidaEn:                 ahora,
		ValidaHasta:               ahora.Add(time.Minute),
	}
	if err := base.Validar(); err != nil {
		t.Fatalf("decision valida: %v", err)
	}
	casos := []struct {
		nombre string
		muta   func(*DecisionAutorizacion)
	}{
		{"codigo de denegacion en concesion", func(d *DecisionAutorizacion) { d.Codigo = "denegada" }},
		{"accion con comodin parcial", func(d *DecisionAutorizacion) { d.Accion = "bolsa.*" }},
		{"recurso con comodin parcial", func(d *DecisionAutorizacion) { d.RecursoRef = "expediente:*" }},
		{"huella de asignacion solo sintactica", func(d *DecisionAutorizacion) { d.AsignacionHuellaSHA256 = "huella" }},
		{"huella de rol en mayusculas", func(d *DecisionAutorizacion) { d.VersionRolHuellaSHA256 = strings.Repeat("A", 64) }},
		{"huella de politica adicional", func(d *DecisionAutorizacion) {
			d.PoliticasHuellasSHA256 = map[string]string{
				"politica:minimizacion:v1": strings.Repeat("c", 64),
				"politica:no_aplicada:v1":  strings.Repeat("d", 64),
			}
		}},
		{"campo no canonico", func(d *DecisionAutorizacion) { d.CamposPermitidos = []string{" estado"} }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			decision := base
			decision.PoliticasRefs = append([]string(nil), base.PoliticasRefs...)
			decision.PoliticasHuellasSHA256 = map[string]string{"politica:minimizacion:v1": strings.Repeat("c", 64)}
			caso.muta(&decision)
			if err := decision.Validar(); !errors.Is(err, ErrDecisionAutorizacionInvalida) {
				t.Fatalf("decision incoherente aceptada: %v", err)
			}
		})
	}

	denegada := base
	denegada.Concedida = false
	denegada.Codigo = "concedida"
	if err := denegada.Validar(); !errors.Is(err, ErrDecisionAutorizacionInvalida) {
		t.Fatalf("denegacion con codigo concedida aceptada: %v", err)
	}
}

func TestHuellaVersionRolFijaLaInstantanea(t *testing.T) {
	version := versionRolValidaPrueba()
	primera, err := version.HuellaSHA256()
	if err != nil {
		t.Fatalf("calcular huella: %v", err)
	}
	version.Concesiones[0].CamposPermitidos = append(version.Concesiones[0].CamposPermitidos, "dni")
	segunda, err := version.HuellaSHA256()
	if err != nil {
		t.Fatalf("calcular segunda huella: %v", err)
	}
	if primera == segunda {
		t.Fatal("la huella debe cambiar al modificar la instantanea")
	}
}

func TestHuellaCatalogoPoliticasFijaConjuntoCompletoSinDependerDelOrden(t *testing.T) {
	primeraPolitica := politicaRestrictivaValidaPrueba("minimizacion")
	segundaPolitica := politicaRestrictivaValidaPrueba("doble_control")

	huellaDirecta, err := HuellaCatalogoPoliticasAutorizacion([]PoliticaRestrictiva{primeraPolitica, segundaPolitica})
	if err != nil {
		t.Fatalf("huella directa: %v", err)
	}
	huellaInvertida, err := HuellaCatalogoPoliticasAutorizacion([]PoliticaRestrictiva{segundaPolitica, primeraPolitica})
	if err != nil {
		t.Fatalf("huella invertida: %v", err)
	}
	if huellaDirecta != huellaInvertida {
		t.Fatalf("el orden fisico cambio el catalogo: %q != %q", huellaDirecta, huellaInvertida)
	}

	segundaPolitica.Obligaciones = []string{"registrar_consulta_reforzada"}
	huellaModificada, err := HuellaCatalogoPoliticasAutorizacion([]PoliticaRestrictiva{primeraPolitica, segundaPolitica})
	if err != nil {
		t.Fatalf("huella modificada: %v", err)
	}
	if huellaModificada == huellaDirecta {
		t.Fatal("la huella no fijo el contenido completo de las politicas")
	}

	duplicada := primeraPolitica
	duplicada.Version++
	if _, err := HuellaCatalogoPoliticasAutorizacion([]PoliticaRestrictiva{primeraPolitica, duplicada}); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
		t.Fatalf("dos versiones actuales de la misma politica fueron aceptadas: %v", err)
	}
}

func TestInstantaneaAutorizacionExigeRolYControlDeCatalogoCoherentes(t *testing.T) {
	version := versionRolValidaPrueba()
	asignacion := asignacionPerfilValidaPrueba()
	politicas := []PoliticaRestrictiva{politicaRestrictivaValidaPrueba("minimizacion")}
	huella, err := HuellaCatalogoPoliticasAutorizacion(politicas)
	if err != nil {
		t.Fatalf("calcular huella: %v", err)
	}
	instantanea := InstantaneaAutorizacion{
		AsignacionPerfil: asignacion,
		VersionRol:       version,
		ControlVigenciaVersionRol: ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn,
		},
		Politicas:                     politicas,
		RevisionCatalogoPoliticas:     3,
		CatalogoPoliticasHuellaSHA256: huella,
	}
	if err := instantanea.Validar(); err != nil {
		t.Fatalf("instantanea valida: %v", err)
	}

	incoherente := instantanea
	incoherente.CatalogoPoliticasHuellaSHA256 = strings.Repeat("f", 64)
	if err := incoherente.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
		t.Fatalf("huella de catalogo incoherente aceptada: %v", err)
	}
	incoherente = instantanea
	incoherente.VersionRol.RolID = "otro_rol"
	if err := incoherente.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
		t.Fatalf("rol ajeno a la asignacion aceptado: %v", err)
	}
}

func TestDecisionAutorizacionSeparaCatalogoEvaluadoDePoliticasAplicadas(t *testing.T) {
	ahora := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	politicaAplicada := politicaRestrictivaValidaPrueba("minimizacion")
	politicaNoAplicada := politicaRestrictivaValidaPrueba("otro_modulo")
	politicaNoAplicada.Modulos = []string{"cronos"}
	politicas := []PoliticaRestrictiva{politicaAplicada, politicaNoAplicada}
	huellaCatalogo, err := HuellaCatalogoPoliticasAutorizacion(politicas)
	if err != nil {
		t.Fatalf("huella de catalogo: %v", err)
	}
	huellaAplicada, _ := politicaAplicada.HuellaSHA256()
	huellaNoAplicada, _ := politicaNoAplicada.HuellaSHA256()
	decision := DecisionAutorizacion{
		DecisionRef:                           "decision:catalogo-completo",
		Concedida:                             true,
		Codigo:                                "concedida",
		PrincipalID:                           "per_0123456789abcdefghijkl",
		PerfilActivoRef:                       "prf_0123456789abcdefghijkl",
		Accion:                                "bolsa.expediente.leer",
		RecursoRef:                            "expediente:1",
		ModuloID:                              "bolsa",
		TipoRecurso:                           "expediente",
		ContextoRecursoHuellaSHA256:           strings.Repeat("e", 64),
		Finalidad:                             "gestion_bolsa",
		CorrelacionRef:                        "corr-catalogo",
		VinculoAutenticacionActor:             vinculoAutenticacionActorV1Prueba(t, ahora),
		AsignacionRef:                         "asignacion:persona-1:v1",
		AsignacionHuellaSHA256:                strings.Repeat("a", 64),
		VersionRolRef:                         "rol:tecnico:v1",
		VersionRolHuellaSHA256:                strings.Repeat("b", 64),
		ControlVigenciaVersionRolRef:          "rol:tecnico:v1",
		ControlVigenciaVersionRolRevision:     1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("d", 64),
		RevisionCatalogoPoliticas:             7,
		CatalogoPoliticasHuellaSHA256:         huellaCatalogo,
		PoliticasEvaluadasRefs:                []string{politicaNoAplicada.Referencia(), politicaAplicada.Referencia()},
		PoliticasEvaluadasHuellasSHA256:       map[string]string{politicaAplicada.Referencia(): huellaAplicada, politicaNoAplicada.Referencia(): huellaNoAplicada},
		PoliticasRefs:                         []string{politicaAplicada.Referencia()},
		PoliticasHuellasSHA256:                map[string]string{politicaAplicada.Referencia(): huellaAplicada},
		GarantiaMinima:                        AuthAssuranceHigh,
		EmitidaEn:                             ahora,
		ValidaHasta:                           ahora.Add(time.Minute),
	}
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("evidencia completa valida: %v", err)
	}

	casos := []struct {
		nombre string
		muta   func(*DecisionAutorizacion)
	}{
		{"revision ausente", func(d *DecisionAutorizacion) { d.RevisionCatalogoPoliticas = 0 }},
		{"politica evaluada omitida", func(d *DecisionAutorizacion) {
			d.PoliticasEvaluadasRefs = d.PoliticasEvaluadasRefs[:1]
			delete(d.PoliticasEvaluadasHuellasSHA256, politicaAplicada.Referencia())
		}},
		{"aplicada no evaluada", func(d *DecisionAutorizacion) {
			d.PoliticasRefs = []string{"politica:inventada:v1"}
			d.PoliticasHuellasSHA256 = map[string]string{"politica:inventada:v1": strings.Repeat("c", 64)}
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			copia := decision
			copia.PoliticasEvaluadasRefs = append([]string(nil), decision.PoliticasEvaluadasRefs...)
			copia.PoliticasEvaluadasHuellasSHA256 = map[string]string{
				politicaAplicada.Referencia():   huellaAplicada,
				politicaNoAplicada.Referencia(): huellaNoAplicada,
			}
			caso.muta(&copia)
			if err := copia.ValidarEvidenciaInstantanea(); !errors.Is(err, ErrDecisionAutorizacionInvalida) {
				t.Fatalf("evidencia incompleta aceptada: %v", err)
			}
		})
	}
}

func TestAutorizacionRechazaTiemposQueCambiarianEnTimestamptz(t *testing.T) {
	version := versionRolValidaPrueba()
	version.PublicadaEn = version.PublicadaEn.Add(time.Nanosecond)
	if err := version.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
		t.Fatalf("version con resto submicrosegundo aceptada: %v", err)
	}

	asignacion := asignacionPerfilValidaPrueba()
	asignacion.VigenteHasta = asignacion.VigenteHasta.Add(time.Nanosecond)
	if err := asignacion.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
		t.Fatalf("asignacion con resto submicrosegundo aceptada: %v", err)
	}

	politica := politicaRestrictivaValidaPrueba("precision")
	politica.PublicadaEn = politica.PublicadaEn.In(time.FixedZone("UTC-no-canonico", 0))
	if err := politica.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
		t.Fatalf("politica fuera de UTC canonico aceptada: %v", err)
	}

	decision := DecisionAutorizacion{
		DecisionRef: "decision:precision", Codigo: "denegada", PrincipalID: "persona-1",
		PerfilActivoRef: "perfil:persona-1:bolsa", Accion: "bolsa.expediente.leer",
		RecursoRef: "expediente:1", Finalidad: "gestion_bolsa", CorrelacionRef: "corr-precision",
		EmitidaEn:   time.Date(2026, 7, 14, 10, 0, 0, 1, time.UTC),
		ValidaHasta: time.Date(2026, 7, 14, 10, 1, 0, 0, time.UTC),
	}
	if err := decision.Validar(); !errors.Is(err, ErrDecisionAutorizacionInvalida) {
		t.Fatalf("decision con resto submicrosegundo aceptada: %v", err)
	}
}

func TestControlVigenciaRolExigeActoYMotivoAlRetirar(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	control := ControlVigenciaVersionRol{
		VersionRolRef: "rol:tecnico_bolsa:v1", Revision: 2,
		Estado:         EstadoControlVigenciaVersionRolRetirada,
		ActualizadoPor: "responsable-seguridad", ActualizadoEn: ahora,
	}
	if err := control.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
		t.Fatalf("retirada sin acto ni motivo aceptada: %v", err)
	}
	control.ActoRef = "acto:retirada-rol:2026-1"
	control.MotivoCodigo = "incidente_seguridad"
	if err := control.Validar(); err != nil {
		t.Fatalf("retirada trazable rechazada: %v", err)
	}
	control.Estado = EstadoControlVigenciaVersionRolHabilitada
	if err := control.Validar(); !errors.Is(err, ErrConfiguracionAccesoInvalida) {
		t.Fatalf("control habilitado con metadatos de retirada aceptado: %v", err)
	}
}

func TestHuellaContextoRecursoEsCanonicaYNoExponeValoresEnDecision(t *testing.T) {
	primero := RecursoAutorizable{
		Referencia: "expediente:1", ModuloID: "bolsa", Tipo: "expediente",
		Ambitos:   map[string]string{"unidad": "seleccion", "provincia": "granada"},
		Atributos: map[string]string{"clasificacion": "interno", "estado": "abierto"},
	}
	segundo := RecursoAutorizable{
		Referencia: "expediente:1", ModuloID: "bolsa", Tipo: "expediente",
		Ambitos:   map[string]string{"provincia": "granada", "unidad": "seleccion"},
		Atributos: map[string]string{"estado": "abierto", "clasificacion": "interno"},
	}
	huellaPrimera, err := primero.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("huella primera: %v", err)
	}
	huellaSegunda, err := segundo.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("huella segunda: %v", err)
	}
	if huellaPrimera != huellaSegunda {
		t.Fatalf("el orden de mapas cambio la huella: %q != %q", huellaPrimera, huellaSegunda)
	}
	segundo.Atributos["estado"] = "cerrado"
	huellaCambiada, err := segundo.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("huella cambiada: %v", err)
	}
	if huellaCambiada == huellaPrimera {
		t.Fatal("un cambio autorizador del recurso no cambio la huella")
	}
	if strings.Contains(huellaPrimera, "seleccion") || strings.Contains(huellaPrimera, "interno") {
		t.Fatalf("la huella expuso valores del contexto: %q", huellaPrimera)
	}
}

func versionRolValidaPrueba() VersionRol {
	ahora := time.Date(2026, 7, 14, 9, 0, 0, 0, time.UTC)
	return VersionRol{
		RolID:   "tecnico_bolsa",
		Version: 1,
		Nombre:  "Tecnico de bolsa",
		Estado:  EstadoVersionRolPublicada,
		Concesiones: []ConcesionRol{{
			Accion:           "bolsa.expediente.leer",
			ModuloID:         "bolsa",
			TipoRecurso:      "expediente",
			Finalidades:      []string{"gestion_bolsa"},
			GarantiaMinima:   AuthAssuranceSubstantial,
			CamposPermitidos: []string{"estado", "nombre"},
		}},
		PublicadaPor: "responsable-seguridad",
		PublicadaEn:  ahora,
	}
}

func asignacionPerfilValidaPrueba() AsignacionPerfil {
	ahora := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	return AsignacionPerfil{
		AsignacionID:    "asig-persona-1-bolsa",
		Version:         1,
		PerfilActivoRef: "perfil:persona-1:bolsa",
		PrincipalID:     "persona-1",
		VersionRolRef:   versionRolValidaPrueba().Referencia(),
		Estado:          EstadoAsignacionPerfilActiva,
		Ambitos:         []AmbitoPerfil{{Clave: "unidad", Valores: []string{"seleccion"}}},
		VigenteDesde:    ahora.Add(-time.Hour),
		VigenteHasta:    ahora.Add(time.Hour),
		EmitidaPor:      "administrador-identidades",
		EmitidaEn:       ahora.Add(-2 * time.Hour),
	}
}

func politicaRestrictivaValidaPrueba(id string) PoliticaRestrictiva {
	ahora := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	return PoliticaRestrictiva{
		PoliticaID:   id,
		Version:      1,
		Nombre:       id,
		Estado:       EstadoPoliticaRestrictivaPublicada,
		Efecto:       EfectoPoliticaRestringir,
		Acciones:     []string{"bolsa.expediente.leer"},
		Modulos:      []string{"bolsa"},
		TiposRecurso: []string{"expediente"},
		VigenteDesde: ahora.Add(-time.Hour),
		VigenteHasta: ahora.Add(time.Hour),
		PublicadaPor: "responsable-seguridad",
		PublicadaEn:  ahora.Add(-2 * time.Hour),
	}
}
