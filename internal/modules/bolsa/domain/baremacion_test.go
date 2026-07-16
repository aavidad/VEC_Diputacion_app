package domain

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

var instanteBasePrueba = time.Date(2026, time.July, 14, 8, 0, 0, 0, time.FixedZone("CEST", 2*60*60))

func TestDecisionInicialFirmadaConPuntuacionFijaYCriterioConfigurable(t *testing.T) {
	if reflect.TypeOf(Puntos(0)).Kind() != reflect.Int64 {
		t.Fatal("los puntos deben almacenarse como entero fijo de 64 bits")
	}
	baremacion := nuevaBaremacionPrueba(t)
	contenido, err := baremacion.PrepararDecisionInicial(propuestaInicial(ResultadoAceptado, 4_000_000))
	if err != nil {
		t.Fatalf("preparar decision inicial: %v", err)
	}
	decision := firmarContenidoPrueba(t, contenido, instanteBasePrueba.Add(2*time.Hour))
	baremacion, err = baremacion.IncorporarDecision(decision)
	if err != nil {
		t.Fatalf("incorporar decision inicial: %v", err)
	}
	if err := baremacion.Validar(); err != nil {
		t.Fatalf("baremacion valida: %v", err)
	}
	if baremacion.Criterio.Clave != "experiencia.entidad_publica.grupo_c1" {
		t.Fatalf("la categoria debe proceder del criterio configurable: %q", baremacion.Criterio.Clave)
	}
	if len(decision.Contenido.ValoracionesEvidencia) != 2 {
		t.Fatalf("un merito conjunto debe conservar sus dos evidencias: %+v", decision.Contenido.ValoracionesEvidencia)
	}
	if decision.Contenido.PuntosDeclarados != 5_750_000 ||
		decision.Contenido.CalculoOficial.PuntosCalculados != 4_250_000 ||
		decision.Contenido.PuntosReconocidos != 4_000_000 {
		t.Fatalf("no se conservaron las tres puntuaciones: %+v", decision.Contenido)
	}
	serializado, err := json.Marshal(decision)
	if err != nil {
		t.Fatalf("serializar: %v", err)
	}
	if !strings.Contains(string(serializado), `"puntos_declarados":5750000`) {
		t.Fatalf("la puntuacion no se serializo como entero fijo: %s", serializado)
	}
}

func TestInspectorRevocaAceptacionCambiandoValoracionDocumental(t *testing.T) {
	baremacion := nuevaBaremacionPrueba(t)
	baremacion, inicial := incorporarInicialPrueba(t, baremacion, ResultadoAceptado, 4_000_000)
	valoraciones := clonarValoracionesPrueba(inicial.Contenido.ValoracionesEvidencia)
	valoraciones[1].Estado = EstadoEvidenciaNoApta
	valoraciones[1].ResultadoSubsanacion = ResultadoSubsanacionNoAplica
	valoraciones[1].MotivoClave = "periodo_no_acreditado"
	valoraciones[1].Motivo = "La certificacion no acredita el periodo completo exigido."

	propuesta := propuestaDesdeDecision(inicial)
	propuesta.ID = "decision:revocacion:002"
	propuesta.Resultado = ResultadoDesestimado
	propuesta.PuntosReconocidos = 0
	propuesta.DecisorRef = "persona-interna:inspectora-03"
	propuesta.PerfilDecisorClave = "inspector_rrhh"
	propuesta.ValoracionesEvidencia = valoraciones
	propuesta.MotivoClave = "revision_inspeccion_desfavorable"
	propuesta.Motivo = "La inspeccion revoca la aceptacion al no quedar acreditado todo el periodo."
	propuesta.AutorizacionRef = "autorizacion:inspeccion:72"
	propuesta.CorrelacionRef = "correlacion:revision:72"
	propuesta.DecididaEn = instanteBasePrueba.Add(3 * time.Hour)
	contenido, err := baremacion.PrepararRevocacion(propuesta)
	if err != nil {
		t.Fatalf("preparar revocacion: %v", err)
	}
	if contenido.Clase != ClaseDecisionRevocacion || contenido.Resultado != ResultadoDesestimado ||
		contenido.PuntosReconocidos != 0 || contenido.Sustituye == nil ||
		*contenido.Sustituye != inicial.Referencia() {
		t.Fatalf("revocacion incorrecta: %+v", contenido)
	}
	revocacion := firmarContenidoPrueba(t, contenido, instanteBasePrueba.Add(4*time.Hour))
	actualizada, err := baremacion.IncorporarDecision(revocacion)
	if err != nil {
		t.Fatalf("incorporar revocacion: %v", err)
	}
	historial, err := actualizada.HistorialDecisiones()
	if err != nil {
		t.Fatalf("historial: %v", err)
	}
	if len(historial) != 2 || historial[0].HuellaSHA256 != inicial.HuellaSHA256 ||
		historial[0].Contenido.Resultado != ResultadoAceptado ||
		historial[1].Contenido.Resultado != ResultadoDesestimado {
		t.Fatalf("la revocacion no preservo el historial: %+v", historial)
	}
	if historial[1].Firma.FirmanteRef != "persona-interna:inspectora-03" ||
		historial[1].Firma.PerfilFirmanteClave != "inspector_rrhh" {
		t.Fatalf("no consta la firma del inspector: %+v", historial[1].Firma)
	}
	historial[0].Contenido.Motivo = "texto manipulado"
	if err := actualizada.Validar(); err != nil {
		t.Fatalf("la copia externa altero el agregado: %v", err)
	}
}

func TestInspectorRehabilitaYAnadeEvidenciaSubsanadaSinBorrarAnteriores(t *testing.T) {
	baremacion := nuevaBaremacionPrueba(t)
	baremacion, inicial := incorporarInicialPrueba(t, baremacion, ResultadoDesestimado, 0)
	valoraciones := clonarValoracionesPrueba(inicial.Contenido.ValoracionesEvidencia)
	original := valoraciones[1].Evidencia.Referencia
	subsanada := EvidenciaMerito{
		Referencia: ReferenciaEvidencia{
			DocumentoRef:      "documento:subsanacion:01J2Y4",
			VersionDocumento:  1,
			RepresentacionRef: "objeto:sha256:subsanacion:01J2Y4",
			HuellaSHA256:      huellaPrueba("c"),
		},
		SubsanacionDe: &original,
	}
	valoraciones = append(valoraciones, ValoracionEvidencia{
		Evidencia:            subsanada,
		Estado:               EstadoEvidenciaApta,
		ResultadoSubsanacion: ResultadoSubsanacionAceptada,
		MotivoClave:          "subsanacion_suficiente",
		Motivo:               "La nueva certificacion completa el periodo que faltaba.",
	})
	propuesta := propuestaDesdeDecision(inicial)
	propuesta.ID = "decision:rehabilitacion:002"
	propuesta.CalculoOficial = calculoOficialPrueba(3_500_000, "rehabilitacion")
	propuesta.CalculoOficial.Evidencias = evidenciasDeValoraciones(valoraciones)
	propuesta.PuntosReconocidos = 3_000_000
	propuesta.Resultado = ResultadoAceptado
	propuesta.DecisorRef = "persona-interna:inspectora-03"
	propuesta.PerfilDecisorClave = "inspector_rrhh"
	propuesta.ValoracionesEvidencia = valoraciones
	propuesta.MotivoClave = "revision_favorable_subsanacion"
	propuesta.Motivo = "La evidencia subsanada permite aceptar el merito."
	propuesta.AutorizacionRef = "autorizacion:inspeccion:73"
	propuesta.CorrelacionRef = "correlacion:revision:73"
	propuesta.DecididaEn = instanteBasePrueba.Add(3 * time.Hour)
	contenido, err := baremacion.PrepararRehabilitacion(propuesta)
	if err != nil {
		t.Fatalf("preparar rehabilitacion: %v", err)
	}
	if contenido.Clase != ClaseDecisionRehabilitacion || contenido.Sustituye == nil ||
		*contenido.Sustituye != inicial.Referencia() || len(contenido.ValoracionesEvidencia) != 3 {
		t.Fatalf("rehabilitacion incorrecta: %+v", contenido)
	}
	rehabilitacion := firmarContenidoPrueba(t, contenido, instanteBasePrueba.Add(4*time.Hour))
	actualizada, err := baremacion.IncorporarDecision(rehabilitacion)
	if err != nil {
		t.Fatalf("incorporar rehabilitacion: %v", err)
	}
	ultima, existe := actualizada.UltimaDecision()
	if !existe || ultima.Contenido.Resultado != ResultadoAceptado ||
		ultima.Contenido.PuntosReconocidos != 3_000_000 || len(ultima.Contenido.ValoracionesEvidencia) != 3 {
		t.Fatalf("resultado vigente incorrecto: %+v", ultima)
	}
	if len(actualizada.Decisiones[0].Contenido.ValoracionesEvidencia) != 2 {
		t.Fatal("la subsanacion modifico retrospectivamente la decision anterior")
	}
}

func TestResultadoPendienteExigeEvidenciaSubsanable(t *testing.T) {
	baremacion := nuevaBaremacionPrueba(t)
	propuesta := propuestaInicial(ResultadoPendienteSubsanacion, 0)
	propuesta.ValoracionesEvidencia[1].Estado = EstadoEvidenciaSubsanable
	propuesta.ValoracionesEvidencia[1].ResultadoSubsanacion = ResultadoSubsanacionPendiente
	propuesta.ValoracionesEvidencia[1].MotivoClave = "documento_subsanable"
	propuesta.ValoracionesEvidencia[1].Motivo = "Falta la fecha final, que puede subsanarse."
	contenido, err := baremacion.PrepararDecisionInicial(propuesta)
	if err != nil {
		t.Fatalf("preparar pendiente de subsanacion: %v", err)
	}
	decision := firmarContenidoPrueba(t, contenido, instanteBasePrueba.Add(2*time.Hour))
	if _, err := baremacion.IncorporarDecision(decision); err != nil {
		t.Fatalf("incorporar pendiente de subsanacion: %v", err)
	}

	propuesta = propuestaInicial(ResultadoPendienteSubsanacion, 0)
	if _, err := baremacion.PrepararDecisionInicial(propuesta); err == nil {
		t.Fatal("se admitio pendiente de subsanacion sin ninguna evidencia subsanable")
	}
}

func TestNoOpIgnoraNuevaTrazaPeroMotivacionONormaSiSonCambioMaterial(t *testing.T) {
	baremacion := nuevaBaremacionPrueba(t)
	baremacion, inicial := incorporarInicialPrueba(t, baremacion, ResultadoAceptado, 4_000_000)
	propuesta := propuestaDesdeDecision(inicial)
	propuesta.ID = "decision:rectificacion:002"
	propuesta.AutorizacionRef = "autorizacion:distinta:002"
	propuesta.CorrelacionRef = "correlacion:distinta:002"
	propuesta.DecididaEn = instanteBasePrueba.Add(3 * time.Hour)
	if _, err := baremacion.PrepararRectificacion(propuesta); !errors.Is(err, ErrDecisionSinCambios) {
		t.Fatalf("una nueva traza no debe disfrazar un no-op: %v", err)
	}

	propuesta.Motivo = "Se corrige la motivacion para precisar el periodo reconocido."
	contenido, err := baremacion.PrepararRectificacion(propuesta)
	if err != nil {
		t.Fatalf("corregir motivacion debe ser un cambio material: %v", err)
	}
	if contenido.PuntosReconocidos != inicial.Contenido.PuntosReconocidos {
		t.Fatal("corregir la motivacion no debe alterar los puntos")
	}

	propuesta = propuestaDesdeDecision(inicial)
	propuesta.ID = "decision:rectificacion:003"
	propuesta.FuentesNormativasRefs = []string{"norma:baremo:v7", "informe-juridico:2026-19"}
	propuesta.DecididaEn = instanteBasePrueba.Add(3 * time.Hour)
	if _, err := baremacion.PrepararRectificacion(propuesta); err != nil {
		t.Fatalf("corregir fuentes debe ser un cambio material: %v", err)
	}
}

func TestRectificacionNoPuedeEliminarEvidenciaNiAnadirUnaAjena(t *testing.T) {
	baremacion := nuevaBaremacionPrueba(t)
	baremacion, inicial := incorporarInicialPrueba(t, baremacion, ResultadoAceptado, 4_000_000)
	propuesta := propuestaDesdeDecision(inicial)
	propuesta.ID = "decision:rectificacion:002"
	propuesta.ValoracionesEvidencia = propuesta.ValoracionesEvidencia[:1]
	propuesta.Motivo = "Se intenta eliminar una de las evidencias anteriores."
	propuesta.DecididaEn = instanteBasePrueba.Add(3 * time.Hour)
	if _, err := baremacion.PrepararRectificacion(propuesta); !errors.Is(err, ErrTransicionDecisionInvalida) {
		t.Fatalf("se permitio eliminar evidencia previa: %v", err)
	}

	propuesta = propuestaDesdeDecision(inicial)
	propuesta.ID = "decision:rectificacion:003"
	propuesta.ValoracionesEvidencia = append(propuesta.ValoracionesEvidencia, ValoracionEvidencia{
		Evidencia: EvidenciaMerito{Referencia: ReferenciaEvidencia{
			DocumentoRef:      "documento:ajeno:01",
			VersionDocumento:  1,
			RepresentacionRef: "objeto:ajeno:01",
			HuellaSHA256:      huellaPrueba("e"),
		}},
		Estado:               EstadoEvidenciaApta,
		ResultadoSubsanacion: ResultadoSubsanacionNoAplica,
		MotivoClave:          "documento_adicional",
		Motivo:               "Documento añadido sin vínculo de subsanación.",
	})
	propuesta.Motivo = "Se intenta añadir una evidencia sin origen autorizado de subsanación."
	propuesta.DecididaEn = instanteBasePrueba.Add(3 * time.Hour)
	if _, err := baremacion.PrepararRectificacion(propuesta); !errors.Is(err, ErrTransicionDecisionInvalida) {
		t.Fatalf("se permitio añadir evidencia ajena: %v", err)
	}

	propuesta = propuestaDesdeDecision(inicial)
	propuesta.ID = "decision:rectificacion:004"
	referenciaAjena := propuesta.ValoracionesEvidencia[0].Evidencia.Referencia
	propuesta.ValoracionesEvidencia[1].Evidencia.SubsanacionDe = &referenciaAjena
	propuesta.ValoracionesEvidencia[1].ResultadoSubsanacion = ResultadoSubsanacionAceptada
	propuesta.Motivo = "Se intenta cambiar retrospectivamente el origen de una evidencia."
	propuesta.DecididaEn = instanteBasePrueba.Add(3 * time.Hour)
	if _, err := baremacion.PrepararRectificacion(propuesta); !errors.Is(err, ErrTransicionDecisionInvalida) {
		t.Fatalf("se permitio mutar el linaje de una evidencia previa: %v", err)
	}
}

func TestHistorialRechazaSustitucionNoExactaAunqueEsteFirmada(t *testing.T) {
	baremacion := nuevaBaremacionPrueba(t)
	baremacion, inicial := incorporarInicialPrueba(t, baremacion, ResultadoAceptado, 4_000_000)
	propuesta := propuestaDesdeDecision(inicial)
	propuesta.ID = "decision:rectificacion:002"
	propuesta.PuntosReconocidos = 3_900_000
	propuesta.Motivo = "Se corrige el computo de dias acreditados."
	propuesta.DecididaEn = instanteBasePrueba.Add(3 * time.Hour)
	contenido, err := baremacion.PrepararRectificacion(propuesta)
	if err != nil {
		t.Fatalf("preparar rectificacion: %v", err)
	}
	contenido.Sustituye.HuellaSHA256 = huellaPrueba("f")
	decision := firmarContenidoPrueba(t, contenido, instanteBasePrueba.Add(4*time.Hour))
	if _, err := baremacion.IncorporarDecision(decision); !errors.Is(err, ErrTransicionDecisionInvalida) {
		t.Fatalf("se admitio una sustitucion que no apunta a la huella vigente: %v", err)
	}
}

func TestDecisionFirmadaDetectaManipulacionYFirmanteDistinto(t *testing.T) {
	baremacion := nuevaBaremacionPrueba(t)
	contenido, err := baremacion.PrepararDecisionInicial(propuestaInicial(ResultadoAceptado, 4_000_000))
	if err != nil {
		t.Fatalf("preparar: %v", err)
	}
	huella, err := contenido.HuellaContenidoSHA256()
	if err != nil {
		t.Fatalf("huella contenido: %v", err)
	}
	firma := firmaPrueba(contenido, huella, instanteBasePrueba.Add(2*time.Hour))
	firma.FirmanteRef = "persona-interna:otra-persona"
	if _, err := ConstituirDecisionFirmada(contenido, firma); !errors.Is(err, ErrFirmaDecisionInvalida) {
		t.Fatalf("se admitio un firmante distinto del decisor: %v", err)
	}
	decision := firmarContenidoPrueba(t, contenido, instanteBasePrueba.Add(2*time.Hour))
	decision.Contenido.PuntosReconocidos = 3_000_000
	if err := decision.Validar(); !errors.Is(err, ErrDecisionTecnicaInvalida) {
		t.Fatalf("no se detecto la manipulacion posterior a la firma: %v", err)
	}

	firma = firmaPrueba(contenido, huella, instanteBasePrueba.Add(2*time.Hour))
	firma.HuellaFirmaSHA256 = ""
	if _, err := ConstituirDecisionFirmada(contenido, firma); !errors.Is(err, ErrFirmaDecisionInvalida) {
		t.Fatalf("se admitio una referencia de firma sin huella exacta: %v", err)
	}
	firma = firmaPrueba(contenido, huella, instanteBasePrueba.Add(2*time.Hour))
	firma.HuellaSelloTiempoSHA256 = ""
	if _, err := ConstituirDecisionFirmada(contenido, firma); !errors.Is(err, ErrFirmaDecisionInvalida) {
		t.Fatalf("se admitio un sello de tiempo sin huella exacta: %v", err)
	}
}

func TestHistorialRechazaNoOpForjadoFueraDeLosPreparadores(t *testing.T) {
	baremacion := nuevaBaremacionPrueba(t)
	baremacion, inicial := incorporarInicialPrueba(t, baremacion, ResultadoAceptado, 4_000_000)
	contenido := inicial.Contenido
	contenido.ID = "decision:forjada:002"
	contenido.Numero = 2
	contenido.Clase = ClaseDecisionRectificacion
	contenido.VersionAnteriorBaremacion = 2
	contenido.VersionBaremacion = 3
	contenido.HuellaEstadoAnteriorSHA256 = huellaPrueba("a")
	contenido.HuellaEstadoResultanteSHA256 = huellaPrueba("b")
	contenido.DecididaEn = instanteBasePrueba.Add(3 * time.Hour)
	contenido.AutorizacionRef = "autorizacion:nueva:002"
	contenido.CorrelacionRef = "correlacion:nueva:002"
	referencia := inicial.Referencia()
	contenido.Sustituye = &referencia
	decision := firmarContenidoPrueba(t, contenido, instanteBasePrueba.Add(4*time.Hour))
	if _, err := baremacion.IncorporarDecision(decision); !errors.Is(err, ErrTransicionDecisionInvalida) {
		t.Fatalf("el agregado admitio un no-op forjado: %v", err)
	}
}

func TestCriterioYCalculoOficialQuedanLigadosAlProcesoYReglaExactos(t *testing.T) {
	if _, existe := reflect.TypeOf(AltaMeritoBaremable{}).FieldByName("PuntosCalculados"); existe {
		t.Fatal("el alta no debe aceptar puntos calculados sueltos")
	}
	if _, existe := reflect.TypeOf(PropuestaDecisionTecnica{}).FieldByName("PuntosCalculados"); existe {
		t.Fatal("la propuesta tecnica no debe aceptar puntos calculados sueltos")
	}
	criterio := calculoOficialPrueba(4_250_000, "inicial").Criterio
	if err := criterio.Validar(); err != nil {
		t.Fatalf("criterio gobernado valido: %v", err)
	}
	criterio.ProcesoRef = "proceso-selectivo:otro"
	calculo := calculoOficialPrueba(4_250_000, "inicial")
	calculo.Criterio = criterio
	if err := calculo.Validar(); !errors.Is(err, ErrCalculoOficialInvalido) {
		t.Fatalf("calculo ligado a otro proceso admitido: %v", err)
	}
	calculo = calculoOficialPrueba(4_250_000, "inicial")
	calculo.Regla.Version++
	if err := calculo.Validar(); !errors.Is(err, ErrCalculoOficialInvalido) {
		t.Fatalf("calculo con regla/version distinta admitido: %v", err)
	}
	calculo = calculoOficialPrueba(4_250_000, "inicial")
	calculo.HuellaEntradaSHA256 = huellaPrueba("0")
	if !calculo.CoincideCon(calculo) {
		t.Fatal("un recibo valido debe coincidir consigo mismo")
	}
	if calculo.CoincideCon(calculoOficialPrueba(4_250_000, "inicial")) {
		t.Fatal("dos entradas oficiales distintas se trataron como el mismo calculo")
	}
}

func TestDecisionFirmaIdentidadVersionYHuellasExactasDelEstado(t *testing.T) {
	baremacion := nuevaBaremacionPrueba(t)
	huellaAnterior, err := baremacion.HuellaEstadoAdministrativoSHA256()
	if err != nil {
		t.Fatalf("huella administrativa anterior: %v", err)
	}
	contenido, err := baremacion.PrepararDecisionInicial(propuestaInicial(ResultadoAceptado, 4_000_000))
	if err != nil {
		t.Fatalf("preparar: %v", err)
	}
	huellaResultante, err := baremacion.huellaEstadoAdministrativoCon(contenido)
	if err != nil {
		t.Fatalf("huella administrativa resultante: %v", err)
	}
	if contenido.ProcesoRef != baremacion.ProcesoRef || contenido.SolicitudRef != baremacion.SolicitudRef ||
		contenido.SujetoRef != baremacion.SujetoRef || contenido.BaremacionMeritoRef != baremacion.ID ||
		contenido.VersionAnteriorBaremacion != 1 || contenido.VersionBaremacion != 2 ||
		contenido.HuellaEstadoAnteriorSHA256 != huellaAnterior ||
		contenido.HuellaEstadoResultanteSHA256 != huellaResultante {
		t.Fatalf("enlace de estado incompleto: %+v", contenido)
	}
	decision := firmarContenidoPrueba(t, contenido, instanteBasePrueba.Add(2*time.Hour))
	actualizada, err := baremacion.IncorporarDecision(decision)
	if err != nil {
		t.Fatalf("incorporar: %v", err)
	}
	huellaVigente, err := actualizada.HuellaEstadoAdministrativoSHA256()
	if err != nil || huellaVigente != contenido.HuellaEstadoResultanteSHA256 {
		t.Fatalf("huella resultante no es el estado vigente: %q / %v", huellaVigente, err)
	}

	forjada := contenido
	forjada.HuellaEstadoResultanteSHA256 = huellaPrueba("0")
	decisionForjada := firmarContenidoPrueba(t, forjada, instanteBasePrueba.Add(2*time.Hour))
	if _, err := baremacion.IncorporarDecision(decisionForjada); !errors.Is(err, ErrTransicionDecisionInvalida) {
		t.Fatalf("se admitio una huella nueva no reproducible aunque estuviera firmada: %v", err)
	}
}

func TestPuntosReconocidosNuncaSuperanCalculoOficial(t *testing.T) {
	baremacion := nuevaBaremacionPrueba(t)
	propuesta := propuestaInicial(ResultadoAceptado, 4_500_000)
	if _, err := baremacion.PrepararDecisionInicial(propuesta); err == nil {
		t.Fatal("se reconocieron mas puntos que el calculo oficial")
	}
	propuesta = propuestaInicial(ResultadoAceptado, 4_000_000)
	propuesta.CalculoOficial.HuellaResultadoSHA256 = huellaPrueba("0")
	if _, err := baremacion.PrepararDecisionInicial(propuesta); !errors.Is(err, ErrTransicionDecisionInvalida) {
		t.Fatalf("se admitio otro recibo de calculo para la decision inicial: %v", err)
	}
}

func TestFirmaExigeCustodiaInteractivaValidacionesYPoliticasExactas(t *testing.T) {
	baremacion := nuevaBaremacionPrueba(t)
	contenido, err := baremacion.PrepararDecisionInicial(propuestaInicial(ResultadoAceptado, 4_000_000))
	if err != nil {
		t.Fatal(err)
	}
	huella, err := contenido.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	base := firmaPrueba(contenido, huella, instanteBasePrueba.Add(2*time.Hour))
	casos := []struct {
		nombre string
		mutar  func(*FirmaDecisionTecnica)
	}{
		{"sin_interactividad", func(f *FirmaDecisionTecnica) { f.RequiereFirmaInteractiva = false }},
		{"sin_validacion_servidor", func(f *FirmaDecisionTecnica) { f.RequiereValidacionServidor = false }},
		{"sin_version_politica", func(f *FirmaDecisionTecnica) { f.PoliticaFirmaVersion = 0 }},
		{"sin_huella_politica", func(f *FirmaDecisionTecnica) { f.HuellaPoliticaFirmaSHA256 = "" }},
		{"sin_objeto_custodiado", func(f *FirmaDecisionTecnica) { f.DocumentoFirmableRef = "" }},
		{"sin_version_objeto", func(f *FirmaDecisionTecnica) { f.VersionDocumentoFirmable = "" }},
		{"sin_huella_objeto", func(f *FirmaDecisionTecnica) { f.HuellaDocumentoFirmableSHA256 = "" }},
		{"sin_validacion_inicial", func(f *FirmaDecisionTecnica) { f.ValidacionInicialFirmaRef = "" }},
		{"sin_validacion_final", func(f *FirmaDecisionTecnica) { f.ValidacionFirmaRef = "" }},
		{"sin_politica_sello", func(f *FirmaDecisionTecnica) { f.HuellaPoliticaSelloTiempoSHA256 = "" }},
		{"sin_politica_longevidad", func(f *FirmaDecisionTecnica) { f.PoliticaLongevidadVersion = 0 }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			firma := base
			caso.mutar(&firma)
			if _, err := ConstituirDecisionFirmada(contenido, firma); !errors.Is(err, ErrFirmaDecisionInvalida) {
				t.Fatalf("evidencia incompleta admitida: %v", err)
			}
		})
	}
}

func nuevaBaremacionPrueba(t *testing.T) BaremacionMerito {
	t.Helper()
	baremacion, err := NuevaBaremacionMerito(AltaMeritoBaremable{
		ID:           "merito-baremacion:8d6954b9",
		ProcesoRef:   "proceso-selectivo:2026-017",
		SolicitudRef: "solicitud:5c761d21",
		SujetoRef:    "sujeto:01J2Y39S9X7C6A",
		Criterio: ReferenciaCriterio{
			ProcesoRef:    "proceso-selectivo:2026-017",
			Clave:         "experiencia.entidad_publica.grupo_c1",
			Version:       7,
			HuellaSHA256:  huellaPrueba("a"),
			PuntosMaximos: 10 * UnidadesPorPunto,
			ReglaCalculo: ReferenciaReglaCalculo{
				Clave: "experiencia_publica_dias", Version: 3, HuellaSHA256: huellaPrueba("9"),
			},
		},
		EvidenciasIniciales: evidenciasInicialesPrueba(),
		PuntosDeclarados:    5_750_000,
		CalculoOficial:      calculoOficialPrueba(4_250_000, "inicial"),
		CreadaEn:            instanteBasePrueba,
	})
	if err != nil {
		t.Fatalf("crear baremacion: %v", err)
	}
	return baremacion
}

func evidenciasInicialesPrueba() []EvidenciaMerito {
	return []EvidenciaMerito{
		{Referencia: ReferenciaEvidencia{
			DocumentoRef:      "documento:01J2Y3AP1Q",
			VersionDocumento:  3,
			RepresentacionRef: "objeto:sha256:01J2Y3BB",
			HuellaSHA256:      huellaPrueba("b"),
		}},
		{Referencia: ReferenciaEvidencia{
			DocumentoRef:      "documento:01J2Y3AP2R",
			VersionDocumento:  1,
			RepresentacionRef: "objeto:sha256:01J2Y3CC",
			HuellaSHA256:      huellaPrueba("8"),
		}},
	}
}

func valoracionesInicialesPrueba(resultado ResultadoDecisionTecnica) []ValoracionEvidencia {
	evidencias := evidenciasInicialesPrueba()
	valoraciones := []ValoracionEvidencia{
		{
			Evidencia:            evidencias[0],
			Estado:               EstadoEvidenciaApta,
			ResultadoSubsanacion: ResultadoSubsanacionNoAplica,
			MotivoClave:          "documento_valido",
			Motivo:               "El certificado de servicios es autentico y legible.",
		},
		{
			Evidencia:            evidencias[1],
			Estado:               EstadoEvidenciaApta,
			ResultadoSubsanacion: ResultadoSubsanacionNoAplica,
			MotivoClave:          "documento_valido",
			Motivo:               "La vida laboral confirma las fechas declaradas.",
		},
	}
	if resultado == ResultadoDesestimado {
		valoraciones[1].Estado = EstadoEvidenciaNoApta
		valoraciones[1].MotivoClave = "periodo_no_acreditado"
		valoraciones[1].Motivo = "La segunda evidencia no acredita todo el periodo."
	}
	return valoraciones
}

func propuestaInicial(resultado ResultadoDecisionTecnica, reconocidos Puntos) PropuestaDecisionTecnica {
	return PropuestaDecisionTecnica{
		ID:                    "decision:inicial:001",
		CalculoOficial:        calculoOficialPrueba(4_250_000, "inicial"),
		PuntosReconocidos:     reconocidos,
		Resultado:             resultado,
		DecisorRef:            "persona-interna:tecnica-17",
		PerfilDecisorClave:    "tecnico_baremacion",
		ValoracionesEvidencia: valoracionesInicialesPrueba(resultado),
		MotivoClave:           "valoracion_inicial",
		Motivo:                "Valoracion inicial conforme al criterio vigente.",
		FuentesNormativasRefs: []string{"norma:baremo:v7"},
		AutorizacionRef:       "autorizacion:expediente:001",
		FinalidadClave:        "baremacion_proceso_selectivo",
		CorrelacionRef:        "correlacion:baremacion:001",
		DecididaEn:            instanteBasePrueba.Add(time.Hour),
	}
}

func propuestaDesdeDecision(decision DecisionTecnica) PropuestaDecisionTecnica {
	c := decision.Contenido
	return PropuestaDecisionTecnica{
		CalculoOficial:        c.CalculoOficial,
		PuntosReconocidos:     c.PuntosReconocidos,
		Resultado:             c.Resultado,
		DecisorRef:            c.DecisorRef,
		PerfilDecisorClave:    c.PerfilDecisorClave,
		ValoracionesEvidencia: clonarValoracionesPrueba(c.ValoracionesEvidencia),
		MotivoClave:           c.MotivoClave,
		Motivo:                c.Motivo,
		FuentesNormativasRefs: append([]string(nil), c.FuentesNormativasRefs...),
		AutorizacionRef:       c.AutorizacionRef,
		FinalidadClave:        c.FinalidadClave,
		CorrelacionRef:        c.CorrelacionRef,
	}
}

func incorporarInicialPrueba(t *testing.T, baremacion BaremacionMerito, resultado ResultadoDecisionTecnica, reconocidos Puntos) (BaremacionMerito, DecisionTecnica) {
	t.Helper()
	contenido, err := baremacion.PrepararDecisionInicial(propuestaInicial(resultado, reconocidos))
	if err != nil {
		t.Fatalf("preparar inicial: %v", err)
	}
	decision := firmarContenidoPrueba(t, contenido, instanteBasePrueba.Add(2*time.Hour))
	actualizada, err := baremacion.IncorporarDecision(decision)
	if err != nil {
		t.Fatalf("incorporar inicial: %v", err)
	}
	return actualizada, decision
}

func firmarContenidoPrueba(t *testing.T, contenido ContenidoDecisionTecnica, instante time.Time) DecisionTecnica {
	t.Helper()
	huella, err := contenido.HuellaContenidoSHA256()
	if err != nil {
		t.Fatalf("huella de contenido: %v", err)
	}
	decision, err := ConstituirDecisionFirmada(contenido, firmaPrueba(contenido, huella, instante))
	if err != nil {
		t.Fatalf("constituir decision firmada: %v", err)
	}
	return decision
}

func firmaPrueba(contenido ContenidoDecisionTecnica, huella string, instante time.Time) FirmaDecisionTecnica {
	return FirmaDecisionTecnica{
		FirmanteRef:                            contenido.DecisorRef,
		PerfilFirmanteClave:                    contenido.PerfilDecisorClave,
		PoliticaFirmaRef:                       "politica-firma:baremacion",
		PoliticaFirmaVersion:                   4,
		HuellaPoliticaFirmaSHA256:              huellaPrueba("1"),
		PerfilFirmaAlcanzadoClave:              "pades_baseline_lta",
		RequiereFirmaInteractiva:               true,
		RequiereValidacionServidor:             true,
		RequiereSelloTiempo:                    true,
		RequiereAumentoLongevidad:              true,
		SesionFirmaInteractivaRef:              "sesion-firma:" + contenido.ID,
		HuellaEvidenciaFirmaInteractivaSHA256:  huellaPrueba("2"),
		DocumentoFirmableRef:                   "objeto-firmable:" + contenido.ID,
		VersionDocumentoFirmable:               "version-1",
		HuellaDocumentoFirmableSHA256:          huellaPrueba("8"),
		EvidenciaCustodiaRef:                   "evidencia-custodia:" + contenido.ID,
		FirmaRef:                               "firma:" + contenido.ID,
		HuellaFirmaSHA256:                      huellaPrueba("f"),
		DocumentoFirmadoRef:                    "documento-firmado:" + contenido.ID,
		HuellaDocumentoSHA256:                  huellaPrueba("d"),
		DocumentoFirmadoCustodiadoRef:          "objeto-firmado:" + contenido.ID,
		VersionDocumentoFirmadoCustodiado:      "version-firmada-1",
		EvidenciaRecuperacionFirmadoRef:        "evidencia-recuperacion-firmado:" + contenido.ID,
		HuellaEvidenciaRecuperacionSHA256:      huellaPrueba("b"),
		EvidenciaCustodiaDocumentoFirmadoRef:   "evidencia-custodia-firmado:" + contenido.ID,
		EvidenciaRetencionDocumentoFirmadoRef:  "evidencia-retencion-firmado:" + contenido.ID,
		PoliticaRetencionDocumentoFirmadoRef:   "politica-retencion-firmado:v1",
		DocumentoFirmadoRetenidoHasta:          instante.Add(365 * 24 * time.Hour),
		ManifiestoProbatorioRef:                "manifiesto-probatorio:" + contenido.ID,
		HuellaManifiestoProbatorioSHA256:       huellaPrueba("6"),
		SelloManifiestoProbatorioHMACSHA256:    "hmac-sha256:manifiesto_1:" + huellaPrueba("7"),
		HuellaContenidoSHA256:                  huella,
		ValidacionInicialFirmaRef:              "validacion-firma-inicial:" + contenido.ID,
		HuellaValidacionInicialSHA256:          huellaPrueba("9"),
		ValidadaInicialEn:                      instante.Add(30 * time.Second),
		ValidacionFirmaRef:                     "validacion-firma-final:" + contenido.ID,
		HuellaValidacionSHA256:                 huellaPrueba("e"),
		ValidadaEn:                             instante.Add(4 * time.Minute),
		SelloTiempoRef:                         "sello-tiempo:" + contenido.ID,
		HuellaSelloTiempoSHA256:                huellaPrueba("c"),
		PoliticaSelloTiempoRef:                 "politica-sello:tsa",
		PoliticaSelloTiempoVersion:             2,
		HuellaPoliticaSelloTiempoSHA256:        huellaPrueba("3"),
		ValidacionSelloTiempoRef:               "validacion-sello:" + contenido.ID,
		HuellaValidacionSelloTiempoSHA256:      huellaPrueba("4"),
		SelladaEn:                              instante.Add(time.Minute),
		ValidacionDocumentoSelladoRef:          "validacion-documento-sellado:" + contenido.ID,
		HuellaValidacionDocumentoSelladoSHA256: huellaPrueba("a"),
		ValidadoDocumentoSelladoEn:             instante.Add(2 * time.Minute),
		NivelLongevidadClave:                   "pades_lta",
		AumentoLongevidadRef:                   "aumento-longevidad:" + contenido.ID,
		HuellaAumentoLongevidadSHA256:          huellaPrueba("5"),
		PoliticaLongevidadRef:                  "politica-longevidad:lta",
		PoliticaLongevidadVersion:              3,
		HuellaPoliticaLongevidadSHA256:         huellaPrueba("6"),
		ValidacionLongevidadRef:                "validacion-longevidad:" + contenido.ID,
		HuellaValidacionLongevidadSHA256:       huellaPrueba("7"),
		AumentadaEn:                            instante.Add(3 * time.Minute),
		FirmadaEn:                              instante,
	}
}

func calculoOficialPrueba(puntos Puntos, sufijo string) CalculoOficialBaremacion {
	criterio := ReferenciaCriterio{
		ProcesoRef: "proceso-selectivo:2026-017", Clave: "experiencia.entidad_publica.grupo_c1",
		Version: 7, HuellaSHA256: huellaPrueba("a"), PuntosMaximos: 10 * UnidadesPorPunto,
		ReglaCalculo: ReferenciaReglaCalculo{
			Clave: "experiencia_publica_dias", Version: 3, HuellaSHA256: huellaPrueba("9"),
		},
	}
	return CalculoOficialBaremacion{
		CalculoRef: "calculo-oficial:" + sufijo, ProcesoRef: criterio.ProcesoRef,
		SolicitudRef: "solicitud:5c761d21", SujetoRef: "sujeto:01J2Y39S9X7C6A",
		BaremacionMeritoRef: "merito-baremacion:8d6954b9", Criterio: criterio, Regla: criterio.ReglaCalculo,
		Evidencias: evidenciasInicialesPrueba(),
		EntradaRef: "entrada-calculo:" + sufijo, HuellaEntradaSHA256: huellaPrueba("b"),
		PuntosCalculados: puntos, DesgloseRef: "desglose-calculo:" + sufijo,
		HuellaDesgloseSHA256: huellaPrueba("c"), ResultadoRef: "resultado-calculo:" + sufijo,
		HuellaResultadoSHA256: huellaPrueba("d"), MotorCalculoRef: "motor-baremo:oficial",
		VersionMotorCalculo: "motor-v2.1.0", EvidenciaEjecucionRef: "ejecucion-calculo:" + sufijo,
		HuellaEjecucionSHA256: huellaPrueba("e"), CalculadoEn: instanteBasePrueba.Add(-time.Minute),
	}
}

func clonarValoracionesPrueba(valoraciones []ValoracionEvidencia) []ValoracionEvidencia {
	clon := make([]ValoracionEvidencia, len(valoraciones))
	for indice := range valoraciones {
		clon[indice] = valoraciones[indice]
		if valoraciones[indice].Evidencia.SubsanacionDe != nil {
			referencia := *valoraciones[indice].Evidencia.SubsanacionDe
			clon[indice].Evidencia.SubsanacionDe = &referencia
		}
	}
	return clon
}

func huellaPrueba(caracter string) string {
	return strings.Repeat(caracter, 64)
}
