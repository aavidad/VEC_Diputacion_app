package ports

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestEvidenciaUsoDecisionAutorizacionV1BloqueaCodecsAlternativos(t *testing.T) {
	decision, verificadaEn := decisionAutorizacionReforzadaPrueba(t)
	evidencia, err := NuevaEvidenciaUsoDecisionAutorizacion(decision, verificadaEn)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := evidencia.Datos()
	if err != nil {
		t.Fatal(err)
	}
	for nombre, valor := range map[string]any{"evidencia": evidencia, "datos": datos} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := xml.Marshal(valor); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("XML no bloqueado: %v", err)
			}
			var salida bytes.Buffer
			if err := gob.NewEncoder(&salida).Encode(valor); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("Gob no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalBinary() ([]byte, error) }).MarshalBinary(); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("binario no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalCBOR() ([]byte, error) }).MarshalCBOR(); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("CBOR no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalYAML() (any, error) }).MarshalYAML(); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
				t.Fatalf("YAML no bloqueado: %v", err)
			}
		})
	}
	for nombre, destino := range map[string]interface{ UnmarshalYAML(func(any) error) error }{
		"evidencia": &EvidenciaUsoDecisionAutorizacion{},
		"datos":     &DatosEvidenciaUsoDecisionAutorizacion{},
	} {
		if err := destino.UnmarshalYAML(func(any) error { return nil }); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
			t.Fatalf("%s: decodificacion YAML no bloqueada: %v", nombre, err)
		}
	}
	for nombre, destino := range map[string]interface{ UnmarshalCBOR([]byte) error }{
		"evidencia": &EvidenciaUsoDecisionAutorizacion{},
		"datos":     &DatosEvidenciaUsoDecisionAutorizacion{},
	} {
		if err := destino.UnmarshalCBOR([]byte{0xa0}); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
			t.Fatalf("%s: decodificacion CBOR no bloqueada: %v", nombre, err)
		}
	}
}

func TestEvidenciaUsoDecisionAutorizacionValidaEsOpacaYTemporal(t *testing.T) {
	decision, verificadaEn := decisionAutorizacionReforzadaPrueba(t)
	evidencia, err := NuevaEvidenciaUsoDecisionAutorizacion(decision, verificadaEn)
	if err != nil {
		t.Fatalf("crear evidencia valida: %v", err)
	}
	datos, err := evidencia.Datos()
	if err != nil {
		t.Fatalf("obtener proyeccion defensiva: %v", err)
	}
	if datos.EsquemaHuella != EsquemaHuellaDecisionAutorizacionReforzadaV1 ||
		!esHuellaSHA256EvidenciaUsoAutorizacion(datos.HuellaDecisionSHA256) ||
		!datos.VerificadaEn.Equal(verificadaEn) || datos.Decision.DecisionRef != decision.DecisionRef {
		t.Fatalf("proyeccion inesperada: %+v", datos)
	}
	if !sort.StringsAreSorted(datos.Decision.PoliticasEvaluadasRefs) ||
		!sort.StringsAreSorted(datos.Decision.PoliticasRefs) ||
		!sort.StringsAreSorted(datos.Decision.CamposPermitidos) {
		t.Fatalf("la proyeccion no esta canonizada: %+v", datos.Decision)
	}
	representacion, err := datos.RepresentacionCanonica()
	if err != nil {
		t.Fatalf("obtener representacion canonica: %v", err)
	}
	esperada, err := serializarDecisionAutorizacionReforzadaV1(decision)
	if err != nil || !bytes.Equal(representacion, esperada) {
		t.Fatal("la evidencia no entrego los bytes exactos comprometidos")
	}
	representacion[0] ^= 0xff
	datosOtraVez, err := evidencia.Datos()
	if err != nil {
		t.Fatalf("releer proyeccion defensiva: %v", err)
	}
	representacionOtraVez, err := datosOtraVez.RepresentacionCanonica()
	if err != nil || bytes.Equal(representacion, representacionOtraVez) || !bytes.Equal(representacionOtraVez, esperada) {
		t.Fatal("la representacion canonica comparte memoria mutable")
	}
	if err := evidencia.ValidarEn(verificadaEn); err != nil {
		t.Fatalf("la evidencia debe ser valida en su instante de comprobacion: %v", err)
	}
	if err := evidencia.ValidarEn(decision.ValidaHasta.Add(-time.Microsecond)); err != nil {
		t.Fatalf("la evidencia debe seguir vigente antes del limite: %v", err)
	}
	for nombre, instante := range map[string]time.Time{
		"retroceso de reloj": verificadaEn.Add(-time.Microsecond),
		"limite exclusivo":   decision.ValidaHasta,
		"despues del limite": decision.ValidaHasta.Add(time.Microsecond),
		"instante cero":      {},
	} {
		t.Run(nombre, func(t *testing.T) {
			if err := evidencia.ValidarEn(instante); !errors.Is(err, ErrEvidenciaUsoDecisionAutorizacionInvalida) ||
				!errors.Is(err, domain.ErrAutorizacionDenegada) {
				t.Fatalf("se esperaba denegacion uniforme, recibido %v", err)
			}
		})
	}
	// Datos permite verificar una evidencia historica, no reactivar su permiso.
	if _, err := evidencia.Datos(); err != nil {
		t.Fatalf("la proyeccion estructural no debe depender del reloj de pared: %v", err)
	}

	if _, err := json.Marshal(evidencia); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
		t.Fatalf("JSON no fue bloqueado: %v", err)
	}
	if _, err := json.Marshal(datos); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
		t.Fatalf("JSON de la proyeccion no fue bloqueado: %v", err)
	}
	if _, err := evidencia.MarshalText(); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
		t.Fatalf("texto no fue bloqueado: %v", err)
	}
	if _, err := datos.MarshalText(); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
		t.Fatalf("texto de la proyeccion no fue bloqueado: %v", err)
	}
	var destino EvidenciaUsoDecisionAutorizacion
	if err := json.Unmarshal([]byte(`{}`), &destino); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
		t.Fatalf("deserializacion JSON no fue bloqueada: %v", err)
	}
	if err := destino.UnmarshalText([]byte("dato")); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
		t.Fatalf("deserializacion textual no fue bloqueada: %v", err)
	}
	var datosDestino DatosEvidenciaUsoDecisionAutorizacion
	if err := json.Unmarshal([]byte(`{}`), &datosDestino); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
		t.Fatalf("deserializacion JSON de datos no fue bloqueada: %v", err)
	}
	if err := datosDestino.UnmarshalText([]byte("dato")); !errors.Is(err, ErrSerializacionEvidenciaUsoAutorizacionProhibida) {
		t.Fatalf("deserializacion textual de datos no fue bloqueada: %v", err)
	}

	texto := fmt.Sprintf("%v %+v %#v %s %q", evidencia, evidencia, evidencia, evidencia, evidencia)
	if strings.Contains(texto, decision.DecisionRef) || strings.Contains(texto, decision.PrincipalID) ||
		strings.Contains(texto, datos.HuellaDecisionSHA256) || !strings.Contains(texto, "EVIDENCIA-USO-AUTORIZACION-INTERNA") {
		t.Fatalf("el formato expuso datos de autorizacion: %s", texto)
	}
	textoDatos := fmt.Sprintf("%v %+v %#v %s %q", datos, datos, datos, datos, datos)
	if strings.Contains(textoDatos, decision.DecisionRef) || strings.Contains(textoDatos, decision.PrincipalID) ||
		strings.Contains(textoDatos, datos.HuellaDecisionSHA256) || !strings.Contains(textoDatos, "DATOS-EVIDENCIA-USO-AUTORIZACION-INTERNOS") {
		t.Fatalf("el formato de datos expuso la decision: %s", textoDatos)
	}
	var registro bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&registro, nil))
	logger.Info("prueba", "evidencia", evidencia)
	if strings.Contains(registro.String(), decision.DecisionRef) || strings.Contains(registro.String(), decision.PrincipalID) {
		t.Fatalf("LogValue expuso la decision: %s", registro.String())
	}
}

func TestEvidenciaUsoDecisionAutorizacionDeniegaValorCeroDenegadaCaducadaEIncompleta(t *testing.T) {
	decision, verificadaEn := decisionAutorizacionReforzadaPrueba(t)
	var cero EvidenciaUsoDecisionAutorizacion
	if _, err := cero.Datos(); !errors.Is(err, ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("el valor cero produjo datos: %v", err)
	}
	if err := cero.ValidarEn(verificadaEn); !errors.Is(err, ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("el valor cero fue valido: %v", err)
	}
	if _, err := (DatosEvidenciaUsoDecisionAutorizacion{}).RepresentacionCanonica(); !errors.Is(err, ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("los datos cero expusieron una representacion: %v", err)
	}

	denegada := decision
	denegada.Concedida = false
	denegada.Codigo = "accion_no_concedida"
	if denegada.ValidarEvidenciaInstantanea() != nil {
		t.Fatal("la precondicion necesita una decision denegada estructuralmente valida")
	}
	comprobarCreacionEvidenciaDenegada(t, denegada, verificadaEn)
	comprobarCreacionEvidenciaDenegada(t, decision, decision.ValidaHasta)
	comprobarCreacionEvidenciaDenegada(t, decision, decision.ValidaHasta.Add(time.Microsecond))
	comprobarCreacionEvidenciaDenegada(t, decision, decision.EmitidaEn.Add(-time.Microsecond))
	comprobarCreacionEvidenciaDenegada(t, decision, time.Time{})
	comprobarCreacionEvidenciaDenegada(t, decision, verificadaEn.In(time.FixedZone("UTC equivalente", 0)))
	comprobarCreacionEvidenciaDenegada(t, decision, verificadaEn.Add(time.Nanosecond))

	conObligacion := decision
	conObligacion.Obligaciones = []string{"doble_control"}
	if conObligacion.ValidarEvidenciaInstantanea() != nil {
		t.Fatal("la precondicion necesita una obligacion estructuralmente valida")
	}
	comprobarCreacionEvidenciaDenegada(t, conObligacion, verificadaEn)

	camposIncompletos := []struct {
		nombre string
		mutar  func(*domain.DecisionAutorizacion)
	}{
		{"referencia decision", func(d *domain.DecisionAutorizacion) { d.DecisionRef = "" }},
		{"codigo", func(d *domain.DecisionAutorizacion) { d.Codigo = "" }},
		{"principal", func(d *domain.DecisionAutorizacion) { d.PrincipalID = "" }},
		{"perfil", func(d *domain.DecisionAutorizacion) { d.PerfilActivoRef = "" }},
		{"accion", func(d *domain.DecisionAutorizacion) { d.Accion = "" }},
		{"recurso", func(d *domain.DecisionAutorizacion) { d.RecursoRef = "" }},
		{"modulo", func(d *domain.DecisionAutorizacion) { d.ModuloID = "" }},
		{"tipo", func(d *domain.DecisionAutorizacion) { d.TipoRecurso = "" }},
		{"contexto", func(d *domain.DecisionAutorizacion) { d.ContextoRecursoHuellaSHA256 = "" }},
		{"finalidad", func(d *domain.DecisionAutorizacion) { d.Finalidad = "" }},
		{"correlacion", func(d *domain.DecisionAutorizacion) { d.CorrelacionRef = "" }},
		{"vinculo autenticacion actor", func(d *domain.DecisionAutorizacion) {
			d.VinculoAutenticacionActor = domain.VinculoAutenticacionActorV1{}
		}},
		{"asignacion", func(d *domain.DecisionAutorizacion) { d.AsignacionRef = "" }},
		{"huella asignacion", func(d *domain.DecisionAutorizacion) { d.AsignacionHuellaSHA256 = "" }},
		{"rol", func(d *domain.DecisionAutorizacion) { d.VersionRolRef = "" }},
		{"huella rol", func(d *domain.DecisionAutorizacion) { d.VersionRolHuellaSHA256 = "" }},
		{"control rol", func(d *domain.DecisionAutorizacion) { d.ControlVigenciaVersionRolRef = "" }},
		{"revision control", func(d *domain.DecisionAutorizacion) { d.ControlVigenciaVersionRolRevision = 0 }},
		{"huella control", func(d *domain.DecisionAutorizacion) { d.ControlVigenciaVersionRolHuellaSHA256 = "" }},
		{"revision catalogo", func(d *domain.DecisionAutorizacion) { d.RevisionCatalogoPoliticas = 0 }},
		{"huella catalogo", func(d *domain.DecisionAutorizacion) { d.CatalogoPoliticasHuellaSHA256 = "" }},
		{"garantia", func(d *domain.DecisionAutorizacion) { d.GarantiaMinima = "" }},
		{"emision", func(d *domain.DecisionAutorizacion) { d.EmitidaEn = time.Time{} }},
		{"caducidad", func(d *domain.DecisionAutorizacion) { d.ValidaHasta = time.Time{} }},
	}
	for _, caso := range camposIncompletos {
		t.Run("tupla incompleta/"+caso.nombre, func(t *testing.T) {
			candidata := clonarDecisionAutorizacionCanonica(decision)
			caso.mutar(&candidata)
			comprobarCreacionEvidenciaDenegada(t, candidata, verificadaEn)
		})
	}
}

func TestEvidenciaUsoDecisionAutorizacionRechazaCualquierComodin(t *testing.T) {
	decision, verificadaEn := decisionAutorizacionReforzadaPrueba(t)
	casos := []struct {
		nombre string
		mutar  func(*domain.DecisionAutorizacion)
	}{
		{"decision", func(d *domain.DecisionAutorizacion) { d.DecisionRef = "decision:*" }},
		{"principal", func(d *domain.DecisionAutorizacion) { d.PrincipalID = "persona:*" }},
		{"perfil", func(d *domain.DecisionAutorizacion) { d.PerfilActivoRef = "perfil:*" }},
		{"accion", func(d *domain.DecisionAutorizacion) { d.Accion = "consultar*" }},
		{"recurso", func(d *domain.DecisionAutorizacion) { d.RecursoRef = "expediente:*" }},
		{"modulo", func(d *domain.DecisionAutorizacion) { d.ModuloID = "bolsa*" }},
		{"tipo", func(d *domain.DecisionAutorizacion) { d.TipoRecurso = "solicitud*" }},
		{"finalidad", func(d *domain.DecisionAutorizacion) { d.Finalidad = "tramitar*" }},
		{"correlacion", func(d *domain.DecisionAutorizacion) { d.CorrelacionRef = "corr:*" }},
		{"asignacion", func(d *domain.DecisionAutorizacion) { d.AsignacionRef = "asignacion:*" }},
		{"campo", func(d *domain.DecisionAutorizacion) { d.CamposPermitidos[0] = "dni*" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			candidata := clonarDecisionAutorizacionCanonica(decision)
			caso.mutar(&candidata)
			comprobarCreacionEvidenciaDenegada(t, candidata, verificadaEn)
		})
	}
	// La barrera local se mantiene aunque el dominio ya rechace los comodines:
	// una relajacion accidental en cualquiera de las dos capas no los convierte
	// en admisibles para un consumo exacto.
	conComodin := decision
	conComodin.DecisionRef = "decision:*"
	if !contieneComodinDecisionAutorizacion(conComodin) {
		t.Fatal("no se detecto el comodin incrustado")
	}
}

func TestHuellaDecisionAutorizacionEsCanonicaDeterministaYCompleta(t *testing.T) {
	decision, _ := decisionAutorizacionReforzadaPrueba(t)
	// Guardia de evolucion: si DecisionAutorizacion incorpora un campo, este
	// test obliga a publicar conscientemente un esquema nuevo o a comprometerlo
	// en V1; no puede quedar fuera de la huella por olvido.
	camposDominioEsperados := []string{
		"DecisionRef", "Concedida", "Codigo", "PrincipalID", "PerfilActivoRef", "Accion",
		"RecursoRef", "ModuloID", "TipoRecurso", "ContextoRecursoHuellaSHA256", "Finalidad",
		"CorrelacionRef", "EsquemaHuellaSolicitud", "SolicitudHuellaSHA256",
		"EsquemaHuellaMotivo", "MotivoHuellaSHA256", "AsignacionRef", "AsignacionHuellaSHA256", "VersionRolRef",
		"VinculoAutenticacionActor",
		"VersionRolHuellaSHA256", "ControlVigenciaVersionRolRef", "ControlVigenciaVersionRolRevision",
		"ControlVigenciaVersionRolHuellaSHA256", "RevisionCatalogoPoliticas",
		"CatalogoPoliticasHuellaSHA256", "PoliticasEvaluadasRefs",
		"PoliticasEvaluadasHuellasSHA256", "PoliticasRefs", "PoliticasHuellasSHA256",
		"GarantiaMinima", "CamposPermitidos", "Obligaciones", "EmitidaEn", "ValidaHasta",
	}
	tipoDecision := reflect.TypeOf(domain.DecisionAutorizacion{})
	camposDominioRecibidos := make([]string, 0, tipoDecision.NumField())
	for indice := 0; indice < tipoDecision.NumField(); indice++ {
		camposDominioRecibidos = append(camposDominioRecibidos, tipoDecision.Field(indice).Name)
	}
	sort.Strings(camposDominioEsperados)
	sort.Strings(camposDominioRecibidos)
	if !reflect.DeepEqual(camposDominioRecibidos, camposDominioEsperados) {
		t.Fatalf("DecisionAutorizacion cambio sin versionar la huella: %v", camposDominioRecibidos)
	}

	invertida := clonarDecisionAutorizacionCanonica(decision)
	invertirCadenas(invertida.PoliticasEvaluadasRefs)
	invertirCadenas(invertida.PoliticasRefs)
	invertirCadenas(invertida.CamposPermitidos)
	invertida.PoliticasEvaluadasHuellasSHA256 = mapaConInsercionInvertida(
		invertida.PoliticasEvaluadasRefs,
		invertida.PoliticasEvaluadasHuellasSHA256,
	)
	invertida.PoliticasHuellasSHA256 = mapaConInsercionInvertida(
		invertida.PoliticasRefs,
		invertida.PoliticasHuellasSHA256,
	)
	huella, err := huellaDecisionAutorizacionReforzadaV1(decision)
	if err != nil {
		t.Fatalf("huella canonica: %v", err)
	}
	huellaInvertida, err := huellaDecisionAutorizacionReforzadaV1(invertida)
	if err != nil {
		t.Fatalf("huella con otro orden fisico: %v", err)
	}
	if huella != huellaInvertida {
		t.Fatalf("el orden fisico cambio la huella: %s != %s", huella, huellaInvertida)
	}
	const huellaDorada = "449da5b5a2b3af01aa42c9049eb0c8a34fd1ec9dc3b797fd7e56aaeded5dda5e"
	if huella != huellaDorada {
		t.Fatalf("cambio incompatible del esquema canonico: recibido %s", huella)
	}

	contenido, err := serializarDecisionAutorizacionReforzadaV1(decision)
	if err != nil {
		t.Fatalf("serializar formato canonico: %v", err)
	}
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &objeto); err != nil {
		t.Fatalf("leer formato canonico: %v", err)
	}
	clavesEsperadas := []string{
		"esquema", "decision_ref", "concedida", "codigo", "principal_id", "perfil_activo_ref",
		"accion", "recurso_ref", "modulo_id", "tipo_recurso", "contexto_recurso_huella_sha256",
		"finalidad", "correlacion_ref", "vinculo_autenticacion_actor", "asignacion_ref", "asignacion_huella_sha256",
		"version_rol_ref", "version_rol_huella_sha256", "control_vigencia_version_rol_ref",
		"control_vigencia_version_rol_revision", "control_vigencia_version_rol_huella_sha256",
		"revision_catalogo_politicas", "catalogo_politicas_huella_sha256", "politicas_evaluadas",
		"politicas_aplicables", "garantia_minima", "campos_permitidos", "obligaciones",
		"emitida_en", "valida_hasta",
	}
	sort.Strings(clavesEsperadas)
	clavesRecibidas := make([]string, 0, len(objeto))
	for clave := range objeto {
		clavesRecibidas = append(clavesRecibidas, clave)
	}
	sort.Strings(clavesRecibidas)
	if !reflect.DeepEqual(clavesRecibidas, clavesEsperadas) {
		t.Fatalf("la representacion no compromete todos los campos: %v", clavesRecibidas)
	}
	if string(objeto["concedida"]) != "true" || string(objeto["codigo"]) != `"concedida"` ||
		string(objeto["esquema"]) != `"`+EsquemaHuellaDecisionAutorizacionReforzadaV1+`"` {
		t.Fatalf("campos de concesion/esquema ausentes: %s", contenido)
	}
	var vinculo map[string]json.RawMessage
	if err := json.Unmarshal(objeto["vinculo_autenticacion_actor"], &vinculo); err != nil {
		t.Fatalf("leer vinculo canonico: %v", err)
	}
	clavesVinculoEsperadas := []string{
		"bloque_version", "autenticacion_ref", "autenticacion_huella_sha256", "asercion_ref",
		"sesion_ref", "control_sesion_ref", "control_sesion_revision", "control_sesion_huella_sha256",
		"cuenta_ref", "cuenta_ordinaria_ref", "principal_id", "perfil_activo_ref",
		"cuenta_privilegiada", "superficie", "metodo_observado",
		"garantia_observada", "politica_garantia_ref", "politica_garantia_huella_sha256",
		"autenticacion_verificada_en", "sesion_emitida_en", "sesion_valida_hasta",
		"sesion_revalidada_en", "contexto_actor_ref", "contexto_actor_version",
		"contexto_actor_huella_sha256",
	}
	clavesVinculoRecibidas := make([]string, 0, len(vinculo))
	for clave := range vinculo {
		clavesVinculoRecibidas = append(clavesVinculoRecibidas, clave)
	}
	sort.Strings(clavesVinculoEsperadas)
	sort.Strings(clavesVinculoRecibidas)
	datosVinculo, _ := decision.VinculoAutenticacionActor.Datos()
	instanteRevalidacionEsperado := `"` + datosVinculo.SesionRevalidadaEn.UTC().Format(formatoInstanteDecisionAutorizacionV1) + `"`
	if !reflect.DeepEqual(clavesVinculoRecibidas, clavesVinculoEsperadas) ||
		string(vinculo["cuenta_privilegiada"]) != "false" ||
		string(vinculo["sesion_revalidada_en"]) != instanteRevalidacionEsperado {
		t.Fatalf("bloque de 25 datos no canonico: %s", objeto["vinculo_autenticacion_actor"])
	}

	huellaBase := huella
	mutacionesValidas := []struct {
		nombre string
		mutar  func(*domain.DecisionAutorizacion)
	}{
		{"decision", func(d *domain.DecisionAutorizacion) { d.DecisionRef = "decision:otra" }},
		{"accion", func(d *domain.DecisionAutorizacion) { d.Accion = "consultar_otro" }},
		{"recurso", func(d *domain.DecisionAutorizacion) { d.RecursoRef = "expediente:otro" }},
		{"modulo", func(d *domain.DecisionAutorizacion) { d.ModuloID = "seleccion" }},
		{"tipo", func(d *domain.DecisionAutorizacion) { d.TipoRecurso = "expediente" }},
		{"contexto", func(d *domain.DecisionAutorizacion) { d.ContextoRecursoHuellaSHA256 = huellaPrueba('9') }},
		{"finalidad", func(d *domain.DecisionAutorizacion) { d.Finalidad = "resolver" }},
		{"correlacion", func(d *domain.DecisionAutorizacion) { d.CorrelacionRef = "corr:otra" }},
		{"vinculo autenticacion actor", func(d *domain.DecisionAutorizacion) {
			d.VinculoAutenticacionActor = vinculoAutenticacionActorPuertoPrueba(t, d.EmitidaEn.Add(time.Microsecond))
		}},
		{"asignacion", func(d *domain.DecisionAutorizacion) { d.AsignacionRef = "asignacion:otra:v1" }},
		{"huella asignacion", func(d *domain.DecisionAutorizacion) { d.AsignacionHuellaSHA256 = huellaPrueba('8') }},
		{"version/control rol", func(d *domain.DecisionAutorizacion) {
			d.VersionRolRef = "rol:tecnico:v2"
			d.ControlVigenciaVersionRolRef = d.VersionRolRef
		}},
		{"huella rol", func(d *domain.DecisionAutorizacion) { d.VersionRolHuellaSHA256 = huellaPrueba('7') }},
		{"revision control", func(d *domain.DecisionAutorizacion) { d.ControlVigenciaVersionRolRevision++ }},
		{"huella control", func(d *domain.DecisionAutorizacion) { d.ControlVigenciaVersionRolHuellaSHA256 = huellaPrueba('b') }},
		{"revision catalogo", func(d *domain.DecisionAutorizacion) { d.RevisionCatalogoPoliticas++ }},
		{"politicas evaluadas/catalogo", mutarCatalogoPoliticasPrueba(t)},
		{"politicas aplicables", func(d *domain.DecisionAutorizacion) {
			d.PoliticasRefs = d.PoliticasRefs[:1]
			d.PoliticasHuellasSHA256 = map[string]string{
				d.PoliticasRefs[0]: d.PoliticasEvaluadasHuellasSHA256[d.PoliticasRefs[0]],
			}
		}},
		{"garantia", func(d *domain.DecisionAutorizacion) { d.GarantiaMinima = domain.AuthAssuranceSubstantial }},
		{"campos", func(d *domain.DecisionAutorizacion) { d.CamposPermitidos[0] = "apellidos" }},
		{"obligaciones", func(d *domain.DecisionAutorizacion) { d.Obligaciones = []string{"doble_control"} }},
		{"emision/caducidad", func(d *domain.DecisionAutorizacion) {
			d.EmitidaEn = d.EmitidaEn.Add(time.Microsecond)
			d.ValidaHasta = d.ValidaHasta.Add(time.Microsecond)
		}},
		{"caducidad", func(d *domain.DecisionAutorizacion) { d.ValidaHasta = d.ValidaHasta.Add(time.Microsecond) }},
		{"concesion/codigo", func(d *domain.DecisionAutorizacion) {
			d.Concedida = false
			d.Codigo = "accion_no_concedida"
		}},
	}
	for _, caso := range mutacionesValidas {
		t.Run("compromete/"+caso.nombre, func(t *testing.T) {
			candidata := clonarDecisionAutorizacionCanonica(decision)
			caso.mutar(&candidata)
			if err := candidata.ValidarEvidenciaInstantanea(); err != nil {
				t.Fatalf("mutacion de prueba no valida: %v", err)
			}
			huellaCandidata, err := huellaDecisionAutorizacionReforzadaV1(candidata)
			if err != nil {
				t.Fatalf("calcular huella: %v", err)
			}
			if huellaCandidata == huellaBase {
				t.Fatal("la mutacion no cambio la huella")
			}
		})
	}
}

func TestEvidenciaUsoDecisionAutorizacionHaceCopiasDefensivas(t *testing.T) {
	decision, verificadaEn := decisionAutorizacionReforzadaPrueba(t)
	evidencia, err := NuevaEvidenciaUsoDecisionAutorizacion(decision, verificadaEn)
	if err != nil {
		t.Fatalf("crear evidencia: %v", err)
	}
	base, err := evidencia.Datos()
	if err != nil {
		t.Fatalf("obtener referencia canonica: %v", err)
	}
	referenciaOriginal := base.Decision.PoliticasEvaluadasRefs[0]
	huellaOriginal := base.Decision.PoliticasEvaluadasHuellasSHA256[referenciaOriginal]
	campoOriginal := base.Decision.CamposPermitidos[0]

	decision.PoliticasEvaluadasRefs[0] = "politica:alterada:v1"
	decision.PoliticasRefs[0] = "politica:alterada:v1"
	decision.CamposPermitidos[0] = "alterado"
	decision.PoliticasEvaluadasHuellasSHA256[referenciaOriginal] = huellaPrueba('0')
	decision.PoliticasHuellasSHA256[referenciaOriginal] = huellaPrueba('0')
	if err := evidencia.ValidarEn(verificadaEn); err != nil {
		t.Fatalf("la mutacion del origen altero la evidencia: %v", err)
	}

	primera, err := evidencia.Datos()
	if err != nil {
		t.Fatalf("primera copia: %v", err)
	}
	primera.Decision.PoliticasEvaluadasRefs[0] = "politica:alterada:v2"
	primera.Decision.PoliticasRefs[0] = "politica:alterada:v2"
	primera.Decision.CamposPermitidos[0] = "alterado_de_nuevo"
	primera.Decision.PoliticasEvaluadasHuellasSHA256[referenciaOriginal] = huellaPrueba('f')
	primera.Decision.PoliticasHuellasSHA256[referenciaOriginal] = huellaPrueba('f')
	segunda, err := evidencia.Datos()
	if err != nil {
		t.Fatalf("segunda copia: %v", err)
	}
	if segunda.Decision.PoliticasEvaluadasRefs[0] != referenciaOriginal ||
		segunda.Decision.PoliticasRefs[0] != referenciaOriginal ||
		segunda.Decision.CamposPermitidos[0] != campoOriginal ||
		segunda.Decision.PoliticasEvaluadasHuellasSHA256[referenciaOriginal] != huellaOriginal ||
		segunda.Decision.PoliticasHuellasSHA256[referenciaOriginal] != huellaOriginal {
		t.Fatalf("una proyeccion compartio memoria interna: %+v", segunda.Decision)
	}
}

func TestEvidenciaUsoDecisionAutorizacionEsSeguraEnLecturasConcurrentes(t *testing.T) {
	decision, verificadaEn := decisionAutorizacionReforzadaPrueba(t)
	evidencia, err := NuevaEvidenciaUsoDecisionAutorizacion(decision, verificadaEn)
	if err != nil {
		t.Fatalf("crear evidencia: %v", err)
	}
	const lectores = 32
	const iteraciones = 100
	errores := make(chan error, lectores)
	var grupo sync.WaitGroup
	for lector := 0; lector < lectores; lector++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			for iteracion := 0; iteracion < iteraciones; iteracion++ {
				if err := evidencia.ValidarEn(verificadaEn.Add(time.Microsecond)); err != nil {
					errores <- err
					return
				}
				datos, err := evidencia.Datos()
				if err != nil {
					errores <- err
					return
				}
				datos.Decision.CamposPermitidos[0] = "copia_local"
				for referencia := range datos.Decision.PoliticasEvaluadasHuellasSHA256 {
					datos.Decision.PoliticasEvaluadasHuellasSHA256[referencia] = huellaPrueba('0')
				}
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatalf("lectura concurrente: %v", err)
	}
}

func comprobarCreacionEvidenciaDenegada(
	t *testing.T,
	decision domain.DecisionAutorizacion,
	instante time.Time,
) {
	t.Helper()
	evidencia, err := NuevaEvidenciaUsoDecisionAutorizacion(decision, instante)
	if err == nil || !errors.Is(err, ErrEvidenciaUsoDecisionAutorizacionInvalida) ||
		!errors.Is(err, domain.ErrAutorizacionDenegada) || evidencia.datos != nil {
		t.Fatalf("se esperaba denegacion cerrada; evidencia=%+v error=%v", evidencia, err)
	}
}

func decisionAutorizacionReforzadaPrueba(t *testing.T) (domain.DecisionAutorizacion, time.Time) {
	t.Helper()
	referencias := []string{"politica:proteccion:v2", "politica:seguridad:v4"}
	huellas := map[string]string{
		referencias[0]: huellaPrueba('1'),
		referencias[1]: huellaPrueba('2'),
	}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(referencias, huellas)
	if err != nil {
		t.Fatalf("crear catalogo de prueba: %v", err)
	}
	emitidaEn := time.Date(2026, time.July, 15, 8, 30, 0, 123_456_000, time.UTC)
	decision := domain.DecisionAutorizacion{
		DecisionRef: "decision:prueba:001", Concedida: true, Codigo: "concedida",
		PrincipalID: personaVinculoPuertoPrueba, PerfilActivoRef: perfilVinculoPuertoPrueba,
		Accion: "consultar_expediente", RecursoRef: "expediente:001", ModuloID: "bolsa",
		TipoRecurso: "expediente_bolsa", ContextoRecursoHuellaSHA256: huellaPrueba('3'),
		Finalidad: "tramitar_expediente", CorrelacionRef: "corr:001",
		VinculoAutenticacionActor: vinculoAutenticacionActorPuertoPrueba(t, emitidaEn),
		AsignacionRef:             "asignacion:rrhh:001:v3", AsignacionHuellaSHA256: huellaPrueba('4'),
		VersionRolRef: "rol:tecnico_rrhh:v7", VersionRolHuellaSHA256: huellaPrueba('5'),
		ControlVigenciaVersionRolRef: "rol:tecnico_rrhh:v7", ControlVigenciaVersionRolRevision: 11,
		ControlVigenciaVersionRolHuellaSHA256: huellaPrueba('6'), RevisionCatalogoPoliticas: 19,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasRefs:        []string{referencias[1], referencias[0]},
		PoliticasEvaluadasHuellasSHA256: map[string]string{
			referencias[1]: huellas[referencias[1]], referencias[0]: huellas[referencias[0]],
		},
		PoliticasRefs: []string{referencias[1], referencias[0]},
		PoliticasHuellasSHA256: map[string]string{
			referencias[0]: huellas[referencias[0]], referencias[1]: huellas[referencias[1]],
		},
		GarantiaMinima:   domain.AuthAssuranceHigh,
		CamposPermitidos: []string{"nombre", "dni"}, Obligaciones: nil,
		EmitidaEn: emitidaEn, ValidaHasta: emitidaEn.Add(2 * time.Minute),
	}
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("decision reforzada de prueba invalida: %v", err)
	}
	return decision, emitidaEn.Add(15 * time.Second)
}

func ligarDecisionAutorizacionReforzadaPrueba(
	t *testing.T,
	decision *domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	motivo domain.ReferenciaEntradaCatalogo,
) {
	t.Helper()
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV2(
		domain.DatosSolicitudAutorizacionLigadaV2{
			VinculoAutenticacionActor: decision.VinculoAutenticacionActor,
			ReferenciaMotivo:          motivo,
			Accion:                    decision.Accion, Recurso: recurso, Finalidad: decision.Finalidad,
			Correlacion: referenciaCorrelacionPuertoPrueba(t, decision.CorrelacionRef),
		},
	)
	if err != nil {
		t.Fatalf("crear solicitud nominal de prueba: %v", err)
	}
	huellaSolicitud, err := domain.HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil {
		t.Fatalf("ligar solicitud de prueba: %v", err)
	}
	huellaMotivo, err := domain.HuellaSHA256MotivoAutorizacionV2(motivo)
	if err != nil {
		t.Fatalf("ligar motivo de prueba: %v", err)
	}
	decision.EsquemaHuellaSolicitud = domain.EsquemaHuellaSolicitudAutorizacionV2
	decision.SolicitudHuellaSHA256 = huellaSolicitud
	decision.EsquemaHuellaMotivo = domain.EsquemaHuellaMotivoAutorizacionV2
	decision.MotivoHuellaSHA256 = huellaMotivo
}

func huellaPrueba(caracter byte) string { return strings.Repeat(string(caracter), 64) }

func invertirCadenas(valores []string) {
	for izquierda, derecha := 0, len(valores)-1; izquierda < derecha; izquierda, derecha = izquierda+1, derecha-1 {
		valores[izquierda], valores[derecha] = valores[derecha], valores[izquierda]
	}
}

func mapaConInsercionInvertida(referencias []string, origen map[string]string) map[string]string {
	resultado := make(map[string]string, len(origen))
	for indice := len(referencias) - 1; indice >= 0; indice-- {
		resultado[referencias[indice]] = origen[referencias[indice]]
	}
	return resultado
}

func mutarCatalogoPoliticasPrueba(t *testing.T) func(*domain.DecisionAutorizacion) {
	t.Helper()
	return func(d *domain.DecisionAutorizacion) {
		referencia := d.PoliticasEvaluadasRefs[0]
		d.PoliticasEvaluadasHuellasSHA256[referencia] = huellaPrueba('a')
		if _, aplicable := d.PoliticasHuellasSHA256[referencia]; aplicable {
			d.PoliticasHuellasSHA256[referencia] = huellaPrueba('a')
		}
		huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(
			d.PoliticasEvaluadasRefs,
			d.PoliticasEvaluadasHuellasSHA256,
		)
		if err != nil {
			t.Fatalf("recalcular catalogo: %v", err)
		}
		d.CatalogoPoliticasHuellaSHA256 = huellaCatalogo
	}
}
