package domain

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDecisionAutorizacionLigadaV3SoloNaceDeEvidenciaCompletaYConcede(t *testing.T) {
	solicitud, instantanea, emitida := escenarioDecisionAutorizacionV3Prueba(t)
	evidencia, err := NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, "dec_0123456789abcdef0123456789abcdef", emitida, emitida.Add(4*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := evidencia.ValidarPara(solicitud); err != nil {
		t.Fatalf("evidencia completa invalida: %v", err)
	}
	decision, err := NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	concedida, codigo, err := decision.Resultado()
	if err != nil || !concedida || codigo != "concedida" {
		t.Fatalf("resultado = %t, %q, %v", concedida, codigo, err)
	}
	if err := decision.ValidarPara(solicitud); err != nil {
		t.Fatalf("decision no quedo ligada a la solicitud exacta: %v", err)
	}
	if reflect.TypeOf(decision) == reflect.TypeOf(DecisionAutorizacion{}) ||
		reflect.TypeOf(evidencia) == reflect.TypeOf(DecisionAutorizacion{}) {
		t.Fatal("V3 se hizo asignable a la decision historica")
	}

	desde, hasta, err := decision.VentanaValidez()
	if err != nil || !desde.Equal(emitida) || !hasta.Equal(emitida.Add(90*time.Second)) {
		t.Fatalf("ventana no recortada por politica futura: [%v,%v), %v", desde, hasta, err)
	}
	for _, instante := range []time.Time{
		desde.Add(-time.Microsecond), desde, hasta.Add(-time.Microsecond), hasta,
	} {
		if decision.VigenteEn(instante) {
			t.Fatal("una evaluacion sin confirmacion durable se hizo ejecutable")
		}
	}
}

func TestDecisionAutorizacionLigadaV3NoEjecutaConcesionSinConfirmacionDurable(t *testing.T) {
	decision := decisionAutorizacionLigadaV3Prueba(t)
	concedida, codigo, err := decision.Resultado()
	if err != nil || !concedida || codigo != "concedida" {
		t.Fatalf("resultado evaluado = %t, %q, %v", concedida, codigo, err)
	}
	desde, hasta, err := decision.VentanaValidez()
	if err != nil {
		t.Fatal(err)
	}
	for _, instante := range []time.Time{desde, desde.Add(time.Second), hasta.Add(-time.Microsecond)} {
		if decision.VigenteEn(instante) {
			t.Fatalf("concesion no registrada ejecutable en %v", instante)
		}
	}
}

func TestDecisionAutorizacionLigadaV3DenegacionNuncaEsCapacidad(t *testing.T) {
	solicitud, instantanea, emitida := escenarioDecisionAutorizacionV3Prueba(t)
	instantanea.Politicas[0].Efecto = EfectoPoliticaDenegar
	instantanea.Politicas[0].Restricciones = nil
	instantanea.Politicas[0].RestringeCampos = false
	instantanea.Politicas[0].CamposPermitidos = nil
	instantanea.CatalogoPoliticasHuellaSHA256 = huellaCatalogoDecisionAutorizacionV3Prueba(t, instantanea.Politicas)
	evidencia, err := NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, "dec_denegada_0123456789abcdef", emitida, emitida.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	concedida, codigo, err := decision.Resultado()
	if err != nil || concedida || codigo != "denegada_por_politica" {
		t.Fatalf("denegacion = %t, %q, %v", concedida, codigo, err)
	}
	if decision.VigenteEn(emitida) || decision.VigenteEn(emitida.Add(time.Second)) {
		t.Fatal("una denegacion se convirtio en capacidad vigente")
	}
}

func TestDecisionAutorizacionLigadaV3RechazaCerosNilYTiemposNoCanonicos(t *testing.T) {
	solicitud, instantanea, emitida := escenarioDecisionAutorizacionV3Prueba(t)
	if (EvidenciaEvaluacionAutorizacionV3{}).Validar() == nil ||
		(DecisionAutorizacionLigadaV3{}).Validar() == nil {
		t.Fatal("valor cero aceptado")
	}
	if _, err := NuevaDecisionAutorizacionLigadaV3(solicitud, EvidenciaEvaluacionAutorizacionV3{}); !errors.Is(err, ErrDecisionAutorizacionLigadaV3Invalida) {
		t.Fatalf("evidencia nil aceptada: %v", err)
	}
	casos := []struct {
		nombre string
		desde  time.Time
		hasta  time.Time
	}{
		{"emitida cero", time.Time{}, emitida.Add(time.Minute)},
		{"hasta cero", emitida, time.Time{}},
		{"intervalo vacio", emitida, emitida},
		{"intervalo inverso", emitida, emitida.Add(-time.Microsecond)},
		{"duracion excesiva", emitida, emitida.Add(VigenciaMaximaDecisionAutorizacion + time.Microsecond)},
		{"submicrosegundo", emitida.Add(time.Nanosecond), emitida.Add(time.Minute)},
		{"zona no UTC", emitida.In(time.FixedZone("UTC-falso", 0)), emitida.Add(time.Minute)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevaEvidenciaEvaluacionAutorizacionV3(
				solicitud, instantanea, "dec_0123456789abcdef0123456789abcdef", caso.desde, caso.hasta,
			); !errors.Is(err, ErrEvidenciaEvaluacionAutorizacionV3Invalida) {
				t.Fatalf("tiempo no canonico aceptado: %v", err)
			}
		})
	}
}

func TestDecisionAutorizacionLigadaV3CopiaDefensivaYOrdenDeterminista(t *testing.T) {
	solicitud, instantanea, emitida := escenarioDecisionAutorizacionV3Prueba(t)
	evidencia, err := NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, "dec_0123456789abcdef0123456789abcdef", emitida, emitida.Add(4*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := HuellaSHA256DecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}

	// Alterar todas las colecciones de entrada despues de construir no puede
	// cambiar ni la evidencia ni la decision.
	instantanea.Politicas[0].Obligaciones[0] = "alterada"
	instantanea.Politicas[1].Acciones[0] = "alterada"
	instantanea.AsignacionPerfil.Ambitos[0].Valores[0] = "alterada"
	instantanea.VersionRol.Concesiones[0].CamposPermitidos[0] = "alterado"
	if evidencia.Validar() != nil || decision.Validar() != nil {
		t.Fatal("una mutacion de la entrada alcanzo la capacidad")
	}
	otra, _ := HuellaSHA256DecisionAutorizacionV3(decision)
	if otra != huella {
		t.Fatalf("copia no defensiva: %s != %s", otra, huella)
	}

	_, invertida, _ := escenarioDecisionAutorizacionV3Prueba(t)
	invertida.Politicas[0], invertida.Politicas[1] = invertida.Politicas[1], invertida.Politicas[0]
	invertida.CatalogoPoliticasHuellaSHA256 = huellaCatalogoDecisionAutorizacionV3Prueba(t, invertida.Politicas)
	evidenciaInvertida, err := NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, invertida, "dec_0123456789abcdef0123456789abcdef", emitida, emitida.Add(4*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	decisionInvertida, err := NuevaDecisionAutorizacionLigadaV3(solicitud, evidenciaInvertida)
	if err != nil {
		t.Fatal(err)
	}
	huellaInvertida, _ := HuellaSHA256DecisionAutorizacionV3(decisionInvertida)
	if huellaInvertida != huella {
		t.Fatalf("orden fisico de politicas cambio el canon: %s != %s", huellaInvertida, huella)
	}
}

func TestEvidenciaEvaluacionAutorizacionV3DetectaAdulteracionYTrasplante(t *testing.T) {
	solicitud, instantanea, emitida := escenarioDecisionAutorizacionV3Prueba(t)
	evidencia, err := NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, "dec_0123456789abcdef0123456789abcdef", emitida, emitida.Add(4*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("e", 64)
	casos := map[string]func(*datosEvidenciaEvaluacionAutorizacionV3){
		"esquema solicitud": func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.esquemaHuellaSolicitud += "x" },
		"huella solicitud":  func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.solicitudHuellaSHA256 = h },
		"decision ref":      func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.decisionRef += "x" },
		"resultado":         func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.concedida = false },
		"codigo":            func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.codigo = "accion_no_concedida" },
		"contexto":          func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.contextoRecursoHuellaSHA256 = h },
		"asignacion ref":    func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.asignacionRef += "x" },
		"asignacion huella": func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.asignacionHuellaSHA256 = h },
		"rol ref":           func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.versionRolRef += "x" },
		"rol huella":        func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.versionRolHuellaSHA256 = h },
		"control ref":       func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.controlVigenciaVersionRolRef += "x" },
		"control revision":  func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.controlVigenciaVersionRolRevision++ },
		"control huella":    func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.controlVigenciaVersionRolHuellaSHA256 = h },
		"catalogo revision": func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.revisionCatalogoPoliticas++ },
		"catalogo huella":   func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.catalogoPoliticasHuellaSHA256 = h },
		"evaluadas":         func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.politicasEvaluadas[0].huellaSHA256 = h },
		"aplicables":        func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.politicasAplicables[0].huellaSHA256 = h },
		"garantia":          func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.garantiaMinima = AuthAssuranceSubstantial },
		"campos":            func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.camposPermitidos[0] += "x" },
		"obligaciones":      func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.obligaciones[0] += "x" },
		"emitida":           func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.emitidaEn = d.emitidaEn.Add(time.Microsecond) },
		"valida hasta":      func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.validaHasta = d.validaHasta.Add(time.Microsecond) },
		"sello":             func(d *datosEvidenciaEvaluacionAutorizacionV3) { d.selloSHA256 = h },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			adulterada := clonarEvidenciaEvaluacionAutorizacionV3Prueba(t, evidencia)
			mutar(adulterada.datos)
			if adulterada.Validar() == nil {
				t.Fatal("adulteracion de evidencia aceptada")
			}
		})
	}

	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosSolicitud.Accion = "bolsa.merito.validar"
	otraSolicitud, err := NuevaSolicitudAutorizacionLigadaV3(datosSolicitud)
	if err != nil {
		t.Fatal(err)
	}
	if evidencia.ValidarPara(otraSolicitud) == nil {
		t.Fatal("la evidencia se trasplanto a otra solicitud V3")
	}
	if _, err := NuevaDecisionAutorizacionLigadaV3(otraSolicitud, evidencia); !errors.Is(err, ErrDecisionAutorizacionLigadaV3Invalida) {
		t.Fatalf("se construyo decision desde evidencia trasplantada: %v", err)
	}
}

func TestDecisionAutorizacionLigadaV3DetectaAdulteracionCampoACampo(t *testing.T) {
	decision := decisionAutorizacionLigadaV3Prueba(t)
	h := strings.Repeat("f", 64)
	casos := map[string]func(*datosDecisionAutorizacionLigadaV3){
		"bloque":           func(d *datosDecisionAutorizacionLigadaV3) { d.bloqueVersion++ },
		"referencia":       func(d *datosDecisionAutorizacionLigadaV3) { d.decisionRef += "x" },
		"resultado":        func(d *datosDecisionAutorizacionLigadaV3) { d.concedida = false },
		"codigo":           func(d *datosDecisionAutorizacionLigadaV3) { d.codigo = "accion_no_concedida" },
		"principal":        func(d *datosDecisionAutorizacionLigadaV3) { d.principalID += "x" },
		"perfil":           func(d *datosDecisionAutorizacionLigadaV3) { d.perfilActivoRef += "x" },
		"accion":           func(d *datosDecisionAutorizacionLigadaV3) { d.accion += "x" },
		"recurso":          func(d *datosDecisionAutorizacionLigadaV3) { d.recursoRef += "x" },
		"modulo":           func(d *datosDecisionAutorizacionLigadaV3) { d.moduloID += "x" },
		"tipo":             func(d *datosDecisionAutorizacionLigadaV3) { d.tipoRecurso += "x" },
		"contexto recurso": func(d *datosDecisionAutorizacionLigadaV3) { d.contextoRecursoHuellaSHA256 = h },
		"finalidad":        func(d *datosDecisionAutorizacionLigadaV3) { d.finalidad += "x" },
		"correlacion": func(d *datosDecisionAutorizacionLigadaV3) {
			d.correlacionRef = "correlacion_fedcba9876543210fedcba9876543210"
		},
		"esquema solicitud": func(d *datosDecisionAutorizacionLigadaV3) { d.esquemaHuellaSolicitud += "x" },
		"huella solicitud":  func(d *datosDecisionAutorizacionLigadaV3) { d.solicitudHuellaSHA256 = h },
		"esquema motivo":    func(d *datosDecisionAutorizacionLigadaV3) { d.esquemaHuellaMotivo += "x" },
		"huella motivo":     func(d *datosDecisionAutorizacionLigadaV3) { d.motivoHuellaSHA256 = h },
		"vinculo completo": func(d *datosDecisionAutorizacionLigadaV3) {
			v, _ := d.vinculoAutenticacionActor.Datos()
			v.ContextoActorCuentaVersion++
			d.vinculoAutenticacionActor = VinculoAutenticacionActorV2{datos: &v}
		},
		"asignacion ref":       func(d *datosDecisionAutorizacionLigadaV3) { d.asignacionRef += "x" },
		"asignacion huella":    func(d *datosDecisionAutorizacionLigadaV3) { d.asignacionHuellaSHA256 = h },
		"rol ref":              func(d *datosDecisionAutorizacionLigadaV3) { d.versionRolRef += "x" },
		"rol huella":           func(d *datosDecisionAutorizacionLigadaV3) { d.versionRolHuellaSHA256 = h },
		"control ref":          func(d *datosDecisionAutorizacionLigadaV3) { d.controlVigenciaVersionRolRef += "x" },
		"control revision":     func(d *datosDecisionAutorizacionLigadaV3) { d.controlVigenciaVersionRolRevision++ },
		"control huella":       func(d *datosDecisionAutorizacionLigadaV3) { d.controlVigenciaVersionRolHuellaSHA256 = h },
		"catalogo revision":    func(d *datosDecisionAutorizacionLigadaV3) { d.revisionCatalogoPoliticas++ },
		"catalogo huella":      func(d *datosDecisionAutorizacionLigadaV3) { d.catalogoPoliticasHuellaSHA256 = h },
		"politicas evaluadas":  func(d *datosDecisionAutorizacionLigadaV3) { d.politicasEvaluadas[0].huellaSHA256 = h },
		"politicas aplicables": func(d *datosDecisionAutorizacionLigadaV3) { d.politicasAplicables[0].huellaSHA256 = h },
		"garantia":             func(d *datosDecisionAutorizacionLigadaV3) { d.garantiaMinima = AuthAssuranceSubstantial },
		"campos":               func(d *datosDecisionAutorizacionLigadaV3) { d.camposPermitidos[0] += "x" },
		"obligaciones":         func(d *datosDecisionAutorizacionLigadaV3) { d.obligaciones[0] += "x" },
		"emitida":              func(d *datosDecisionAutorizacionLigadaV3) { d.emitidaEn = d.emitidaEn.Add(time.Microsecond) },
		"valida hasta":         func(d *datosDecisionAutorizacionLigadaV3) { d.validaHasta = d.validaHasta.Add(time.Microsecond) },
		"sello":                func(d *datosDecisionAutorizacionLigadaV3) { d.selloSHA256 = h },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			adulterada := clonarDecisionAutorizacionLigadaV3Prueba(t, decision)
			mutar(adulterada.datos)
			if adulterada.Validar() == nil {
				t.Fatal("adulteracion aceptada")
			}
			if _, err := HuellaSHA256DecisionAutorizacionV3(adulterada); !errors.Is(err, ErrDecisionAutorizacionLigadaV3Invalida) {
				t.Fatalf("huella acepto adulteracion: %v", err)
			}
		})
	}
}

func TestDecisionAutorizacionLigadaV3CanonExhaustivoSinPIINiOperacion(t *testing.T) {
	decision := decisionAutorizacionLigadaV3Prueba(t)
	contenido, err := RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	var documento map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &documento); err != nil {
		t.Fatal(err)
	}
	tipoCanon := reflect.TypeOf(decisionAutorizacionCanonicaV3{})
	if len(documento) != tipoCanon.NumField() {
		t.Fatalf("canon incompleto: JSON=%d tipo=%d: %s", len(documento), tipoCanon.NumField(), contenido)
	}
	for indice := 0; indice < tipoCanon.NumField(); indice++ {
		clave := tipoCanon.Field(indice).Tag.Get("json")
		if _, existe := documento[clave]; !existe {
			t.Fatalf("campo canonico ausente: %s", clave)
		}
	}
	for _, prohibida := range []string{
		"display_name", "email", "roles", "permissions", "attributes", "claims", "operacion_ref",
	} {
		if contieneJSONClavePrueba(contenido, prohibida) {
			t.Fatalf("campo prohibido %q en %s", prohibida, contenido)
		}
	}
	var vinculo map[string]json.RawMessage
	if err := json.Unmarshal(documento["vinculo_autenticacion_actor"], &vinculo); err != nil {
		t.Fatal(err)
	}
	tipoVinculo := reflect.TypeOf(vinculoSolicitudAutorizacionCanonicoV3{})
	tipoDTO := reflect.TypeOf(DatosVinculoAutenticacionActorV2{})
	if len(vinculo) != tipoVinculo.NumField() || tipoVinculo.NumField() != tipoDTO.NumField() {
		t.Fatalf("el vinculo V2 no quedo completo: JSON=%d canon=%d DTO=%d", len(vinculo), tipoVinculo.NumField(), tipoDTO.NumField())
	}
	for indice := 0; indice < tipoDTO.NumField(); indice++ {
		campo := tipoDTO.Field(indice)
		canon, existe := tipoVinculo.FieldByName(campo.Name)
		if !existe || canon.Tag.Get("json") != campo.Tag.Get("json") {
			t.Fatalf("campo V2 no congelado exactamente: %s", campo.Name)
		}
	}
	if strings.Contains(string(contenido), "OperacionRef") || strings.Contains(string(contenido), "operacion") {
		t.Fatalf("OperacionRef filtrada: %s", contenido)
	}
}

func TestDecisionAutorizacionLigadaV3VectorCanonicoEstable(t *testing.T) {
	decision := decisionAutorizacionLigadaV3Prueba(t)
	huella, err := HuellaSHA256DecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	const esperada = "8ade15d03cd3c35525e6fd03f4e334b31689bd86b710fe378748b915814229c6"
	if huella != esperada {
		t.Fatalf("vector V3 cambio: obtenido=%s esperado=%s", huella, esperada)
	}
	contenido, err := RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	for _, instante := range []string{
		`"emitida_en":"2026-07-15T10:11:12.123456Z"`,
		`"valida_hasta":"2026-07-15T10:12:42.123456Z"`,
	} {
		if !bytes.Contains(contenido, []byte(instante)) {
			t.Fatalf("timestamp sin UTC/microsegundos fijos: %s", contenido)
		}
	}
}

func TestDecisionAutorizacionLigadaV3BloqueaCodecsYRedacta(t *testing.T) {
	solicitud, instantanea, emitida := escenarioDecisionAutorizacionV3Prueba(t)
	evidencia, err := NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, "dec_0123456789abcdef0123456789abcdef", emitida, emitida.Add(4*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	probarCierreCodecsDecisionAutorizacionV3(
		t, evidencia, &EvidenciaEvaluacionAutorizacionV3{},
		ErrSerializacionEvidenciaEvaluacionAutorizacionV3Prohibida,
		"[EVIDENCIA-EVALUACION-AUTORIZACION-V3-OPACA]",
	)
	probarCierreCodecsDecisionAutorizacionV3(
		t, decision, &DecisionAutorizacionLigadaV3{},
		ErrSerializacionDecisionAutorizacionLigadaV3Prohibida,
		"[DECISION-AUTORIZACION-LIGADA-V3-OPACA]",
	)
}

func probarCierreCodecsDecisionAutorizacionV3(
	t *testing.T,
	valor any,
	destino any,
	errEsperado error,
	marca string,
) {
	t.Helper()
	if _, err := json.Marshal(valor); !errors.Is(err, errEsperado) {
		t.Fatalf("JSON no bloqueado para %T: %v", valor, err)
	}
	if _, err := xml.Marshal(valor); !errors.Is(err, errEsperado) {
		t.Fatalf("XML no bloqueado para %T: %v", valor, err)
	}
	var gobBytes bytes.Buffer
	if err := gob.NewEncoder(&gobBytes).Encode(valor); !errors.Is(err, errEsperado) {
		t.Fatalf("Gob no bloqueado para %T: %v", valor, err)
	}
	if _, err := valor.(interface{ MarshalText() ([]byte, error) }).MarshalText(); !errors.Is(err, errEsperado) {
		t.Fatalf("Text no bloqueado: %v", err)
	}
	if _, err := valor.(interface{ MarshalBinary() ([]byte, error) }).MarshalBinary(); !errors.Is(err, errEsperado) {
		t.Fatalf("Binary no bloqueado: %v", err)
	}
	if _, err := valor.(interface{ MarshalCBOR() ([]byte, error) }).MarshalCBOR(); !errors.Is(err, errEsperado) {
		t.Fatalf("CBOR no bloqueado: %v", err)
	}
	if _, err := valor.(interface{ MarshalYAML() (any, error) }).MarshalYAML(); !errors.Is(err, errEsperado) {
		t.Fatalf("YAML no bloqueado: %v", err)
	}
	if err := json.Unmarshal([]byte(`{}`), destino); !errors.Is(err, errEsperado) {
		t.Fatalf("JSON decode no bloqueado: %v", err)
	}
	if err := xml.Unmarshal([]byte(`<x/>`), destino); !errors.Is(err, errEsperado) {
		t.Fatalf("XML decode no bloqueado: %v", err)
	}
	if err := destino.(interface{ UnmarshalText([]byte) error }).UnmarshalText(nil); !errors.Is(err, errEsperado) {
		t.Fatalf("Text decode no bloqueado: %v", err)
	}
	if err := destino.(interface{ UnmarshalBinary([]byte) error }).UnmarshalBinary(nil); !errors.Is(err, errEsperado) {
		t.Fatalf("Binary decode no bloqueado: %v", err)
	}
	if err := destino.(interface{ GobDecode([]byte) error }).GobDecode(nil); !errors.Is(err, errEsperado) {
		t.Fatalf("Gob decode no bloqueado: %v", err)
	}
	if err := destino.(interface{ UnmarshalCBOR([]byte) error }).UnmarshalCBOR(nil); !errors.Is(err, errEsperado) {
		t.Fatalf("CBOR decode no bloqueado: %v", err)
	}
	llamado := false
	err := destino.(interface{ UnmarshalYAML(func(any) error) error }).UnmarshalYAML(func(any) error {
		llamado = true
		return nil
	})
	if !errors.Is(err, errEsperado) || llamado {
		t.Fatalf("YAML decode no bloqueado antes del callback: %v, %t", err, llamado)
	}
	for _, texto := range []string{
		fmt.Sprintf("%v", valor), fmt.Sprintf("%+v", valor), fmt.Sprintf("%#v", valor),
		fmt.Sprintf("%s", valor), fmt.Sprintf("%q", valor),
	} {
		if texto != marca || strings.Contains(texto, "dec_0123456789") {
			t.Fatalf("formato filtro contenido: %q", texto)
		}
	}
	var registro bytes.Buffer
	slog.New(slog.NewTextHandler(&registro, nil)).Info("prueba", "valor", valor)
	if !strings.Contains(registro.String(), marca) || strings.Contains(registro.String(), "dec_0123456789") {
		t.Fatalf("slog filtro contenido: %s", registro.String())
	}
}

func escenarioDecisionAutorizacionV3Prueba(
	t *testing.T,
) (SolicitudAutorizacionLigadaV3, InstantaneaAutorizacion, time.Time) {
	t.Helper()
	solicitud := solicitudAutorizacionLigadaV3Prueba(t)
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := datosSolicitud.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	emitida := time.Date(2026, 7, 15, 10, 11, 12, 123_456_000, time.UTC)
	version := VersionRol{
		RolID: "revisor_meritos", Version: 7, Nombre: "Revision de meritos", Estado: EstadoVersionRolPublicada,
		Concesiones: []ConcesionRol{{
			Accion: datosSolicitud.Accion, ModuloID: datosSolicitud.Recurso.ModuloID,
			TipoRecurso: datosSolicitud.Recurso.Tipo, Finalidades: []string{datosSolicitud.Finalidad},
			GarantiaMinima:   AuthAssuranceSubstantial,
			CamposPermitidos: []string{"nombre", "estado"}, Obligaciones: []string{"registrar_revision"},
		}},
		PublicadaPor: "usr_seguridad_0123456789", PublicadaEn: emitida.Add(-2 * time.Hour),
	}
	asignacion := AsignacionPerfil{
		AsignacionID: "asig_revision_meritos_0123456789", Version: 4,
		PerfilActivoRef: vinculo.PerfilActivoRef, PrincipalID: vinculo.PrincipalID,
		VersionRolRef: version.Referencia(), Estado: EstadoAsignacionPerfilActiva,
		Ambitos: []AmbitoPerfil{
			{Clave: "unidad", Valores: []string{"seleccion"}},
			{Clave: "provincia", Valores: []string{"granada"}},
		},
		VigenteDesde: emitida.Add(-time.Hour), VigenteHasta: emitida.Add(3 * time.Minute),
		EmitidaPor: "usr_identidades_0123456789", EmitidaEn: emitida.Add(-3 * time.Hour),
	}
	control := ControlVigenciaVersionRol{
		VersionRolRef: version.Referencia(), Revision: 9,
		Estado:         EstadoControlVigenciaVersionRolHabilitada,
		ActualizadoPor: "usr_seguridad_0123456789", ActualizadoEn: emitida.Add(-time.Hour),
	}
	aplicable := PoliticaRestrictiva{
		PoliticaID: "minimizacion_meritos", Version: 2, Nombre: "Minimizacion de meritos",
		Estado: EstadoPoliticaRestrictivaPublicada, Efecto: EfectoPoliticaRestringir,
		Acciones: []string{datosSolicitud.Accion}, Modulos: []string{datosSolicitud.Recurso.ModuloID},
		TiposRecurso: []string{datosSolicitud.Recurso.Tipo}, FinalidadesPermitidas: []string{datosSolicitud.Finalidad},
		GarantiaMinima:  AuthAssuranceHigh,
		Restricciones:   []RestriccionAtributoRecurso{{Clave: "fase", ValoresPermitidos: []string{"revision"}}},
		RestringeCampos: true, CamposPermitidos: []string{"estado"}, Obligaciones: []string{"auditar_acceso"},
		VigenteDesde: emitida.Add(-time.Hour), VigenteHasta: emitida.Add(2 * time.Minute),
		PublicadaPor: "usr_seguridad_0123456789", PublicadaEn: emitida.Add(-2 * time.Hour),
	}
	futura := PoliticaRestrictiva{
		PoliticaID: "control_futuro", Version: 1, Nombre: "Control futuro",
		Estado: EstadoPoliticaRestrictivaPublicada, Efecto: EfectoPoliticaRestringir,
		Acciones: []string{"*"}, Modulos: []string{"*"}, TiposRecurso: []string{"*"},
		VigenteDesde: emitida.Add(90 * time.Second), VigenteHasta: emitida.Add(time.Hour),
		PublicadaPor: "usr_seguridad_0123456789", PublicadaEn: emitida.Add(-time.Hour),
	}
	politicas := []PoliticaRestrictiva{aplicable, futura}
	instantanea := InstantaneaAutorizacion{
		AsignacionPerfil: asignacion, VersionRol: version, ControlVigenciaVersionRol: control,
		Politicas: politicas, RevisionCatalogoPoliticas: 21,
		CatalogoPoliticasHuellaSHA256: huellaCatalogoDecisionAutorizacionV3Prueba(t, politicas),
	}
	if err := instantanea.Validar(); err != nil {
		t.Fatalf("instantanea de prueba invalida: %v", err)
	}
	return solicitud, instantanea, emitida
}

func decisionAutorizacionLigadaV3Prueba(t *testing.T) DecisionAutorizacionLigadaV3 {
	t.Helper()
	solicitud, instantanea, emitida := escenarioDecisionAutorizacionV3Prueba(t)
	evidencia, err := NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, "dec_0123456789abcdef0123456789abcdef", emitida, emitida.Add(4*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	return decision
}

func clonarDecisionAutorizacionLigadaV3Prueba(
	t *testing.T,
	d DecisionAutorizacionLigadaV3,
) DecisionAutorizacionLigadaV3 {
	t.Helper()
	if d.Validar() != nil {
		t.Fatal("decision base invalida")
	}
	copia := *d.datos
	copia.politicasEvaluadas = clonarEvidenciasPoliticaAutorizacionV3(d.datos.politicasEvaluadas)
	copia.politicasAplicables = clonarEvidenciasPoliticaAutorizacionV3(d.datos.politicasAplicables)
	copia.camposPermitidos = append([]string(nil), d.datos.camposPermitidos...)
	copia.obligaciones = append([]string(nil), d.datos.obligaciones...)
	vinculo, err := clonarVinculoAutenticacionActorV2(d.datos.vinculoAutenticacionActor)
	if err != nil {
		t.Fatal(err)
	}
	copia.vinculoAutenticacionActor = vinculo
	return DecisionAutorizacionLigadaV3{datos: &copia}
}

func clonarEvidenciaEvaluacionAutorizacionV3Prueba(
	t *testing.T,
	e EvidenciaEvaluacionAutorizacionV3,
) EvidenciaEvaluacionAutorizacionV3 {
	t.Helper()
	if e.Validar() != nil {
		t.Fatal("evidencia base invalida")
	}
	copia := *e.datos
	copia.politicasEvaluadas = clonarEvidenciasPoliticaAutorizacionV3(e.datos.politicasEvaluadas)
	copia.politicasAplicables = clonarEvidenciasPoliticaAutorizacionV3(e.datos.politicasAplicables)
	copia.camposPermitidos = append([]string(nil), e.datos.camposPermitidos...)
	copia.obligaciones = append([]string(nil), e.datos.obligaciones...)
	return EvidenciaEvaluacionAutorizacionV3{datos: &copia}
}

func huellaCatalogoDecisionAutorizacionV3Prueba(t *testing.T, politicas []PoliticaRestrictiva) string {
	t.Helper()
	huella, err := HuellaCatalogoPoliticasAutorizacion(politicas)
	if err != nil {
		t.Fatal(err)
	}
	return huella
}
