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

func TestSolicitudAutorizacionLigadaV3EsNominalOpacaYDefensiva(t *testing.T) {
	solicitud := solicitudAutorizacionLigadaV3Prueba(t)
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.Recurso.Ambitos["unidad"] = "alterada"
	datosDeNuevo, err := solicitud.Datos()
	if err != nil || datosDeNuevo.Recurso.Ambitos["unidad"] != "seleccion" {
		t.Fatalf("Datos no devolvio copia defensiva: %+v, %v", datosDeNuevo.Recurso, err)
	}
	if _, err := json.Marshal(solicitud); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if err := json.Unmarshal([]byte(`{}`), &solicitud); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if err := (&solicitud).UnmarshalText(nil); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
		t.Fatalf("UnmarshalText() error = %v", err)
	}
	if _, err := NuevaSolicitudAutorizacionLigadaV3(DatosSolicitudAutorizacionLigadaV3{}); !errors.Is(err, ErrSolicitudAutorizacionLigadaV3Invalida) {
		t.Fatalf("capacidad vacia aceptada: %v", err)
	}
	tipo := reflect.TypeOf(DatosSolicitudAutorizacionLigadaV3{})
	campo, existe := tipo.FieldByName("VinculoAutenticacionActor")
	if !existe || campo.Type != reflect.TypeOf(VinculoAutenticacionActorV2{}) {
		t.Fatalf("el contrato V3 admitiria downgrade: %v", campo.Type)
	}
}

func TestSolicitudAutorizacionLigadaV3CierraCodecsYRedactaFormato(t *testing.T) {
	solicitud := solicitudAutorizacionLigadaV3Prueba(t)
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := datos.Correlacion.ValorCanonico()
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	secretos := []string{
		datos.ReferenciaMotivo.EntradaClave, correlacion, datos.Recurso.Referencia,
		vinculo.PrincipalID, vinculo.SesionRef, vinculo.RegistroContextoRef,
	}
	const redactada = "[SOLICITUD-AUTORIZACION-LIGADA-V3-OPACA]"

	for nombre, valor := range map[string]any{"solicitud": solicitud, "datos": datos} {
		t.Run("codificar "+nombre, func(t *testing.T) {
			if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
				t.Fatalf("JSON no bloqueado: %v", err)
			}
			if _, err := xml.Marshal(valor); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
				t.Fatalf("XML no bloqueado: %v", err)
			}
			var gobBytes bytes.Buffer
			if err := gob.NewEncoder(&gobBytes).Encode(valor); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
				t.Fatalf("Gob no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalText() ([]byte, error) }).MarshalText(); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
				t.Fatalf("Text no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalBinary() ([]byte, error) }).MarshalBinary(); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
				t.Fatalf("Binary no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalCBOR() ([]byte, error) }).MarshalCBOR(); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
				t.Fatalf("CBOR no bloqueado: %v", err)
			}
			if _, err := valor.(interface{ MarshalYAML() (any, error) }).MarshalYAML(); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
				t.Fatalf("YAML no bloqueado: %v", err)
			}

			stringer, okStringer := valor.(fmt.Stringer)
			goStringer, okGoStringer := valor.(interface{ GoString() string })
			logValuer, okLogValuer := valor.(slog.LogValuer)
			if !okStringer || !okGoStringer || !okLogValuer || stringer.String() != redactada ||
				goStringer.GoString() != redactada || logValuer.LogValue().Resolve().String() != redactada {
				t.Fatalf("interfaces de redaccion incompletas para %T", valor)
			}
			formatos := []string{
				fmt.Sprintf("%v", valor), fmt.Sprintf("%+v", valor), fmt.Sprintf("%#v", valor),
				fmt.Sprintf("%s", valor), fmt.Sprintf("%q", valor),
			}
			for _, texto := range formatos {
				if texto != redactada {
					t.Fatalf("Format filtro o altero la marca: %q", texto)
				}
				exigirTextoSolicitudAutorizacionLigadaV3Redactado(t, texto, redactada, secretos)
			}

			var registro bytes.Buffer
			slog.New(slog.NewTextHandler(&registro, nil)).Info("prueba", "solicitud", valor)
			exigirTextoSolicitudAutorizacionLigadaV3Redactado(t, registro.String(), redactada, secretos)
		})
	}

	for nombre, destino := range map[string]any{
		"solicitud": &SolicitudAutorizacionLigadaV3{},
		"datos":     &DatosSolicitudAutorizacionLigadaV3{},
	} {
		t.Run("decodificar "+nombre, func(t *testing.T) {
			if err := json.Unmarshal([]byte(`{}`), destino); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
				t.Fatalf("JSON no bloqueado: %v", err)
			}
			if err := xml.Unmarshal([]byte(`<solicitud/>`), destino); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
				t.Fatalf("XML no bloqueado: %v", err)
			}
			if err := destino.(interface{ UnmarshalText([]byte) error }).UnmarshalText(nil); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
				t.Fatalf("Text no bloqueado: %v", err)
			}
			if err := destino.(interface{ UnmarshalBinary([]byte) error }).UnmarshalBinary(nil); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
				t.Fatalf("Binary no bloqueado: %v", err)
			}
			if err := destino.(interface{ GobDecode([]byte) error }).GobDecode(nil); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
				t.Fatalf("Gob no bloqueado: %v", err)
			}
			if err := destino.(interface{ UnmarshalCBOR([]byte) error }).UnmarshalCBOR([]byte{0xa0}); !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) {
				t.Fatalf("CBOR no bloqueado: %v", err)
			}
			llamado := false
			err := destino.(interface{ UnmarshalYAML(func(any) error) error }).UnmarshalYAML(func(any) error {
				llamado = true
				return nil
			})
			if !errors.Is(err, ErrSerializacionSolicitudAutorizacionLigadaV3Prohibida) || llamado {
				t.Fatalf("YAML no bloqueado antes de decodificar: err=%v llamado=%t", err, llamado)
			}
		})
	}
}

func exigirTextoSolicitudAutorizacionLigadaV3Redactado(
	t *testing.T,
	texto string,
	marca string,
	secretos []string,
) {
	t.Helper()
	if !strings.Contains(texto, marca) {
		t.Fatalf("falta marca opaca en %q", texto)
	}
	for _, secreto := range secretos {
		if secreto != "" && strings.Contains(texto, secreto) {
			t.Fatalf("contenido sensible filtrado en %q", texto)
		}
	}
}

func TestSolicitudAutorizacionLigadaV3HuellaEsDeterministaYComprometeVinculoV2(t *testing.T) {
	base := solicitudAutorizacionLigadaV3Prueba(t)
	huella, err := HuellaSHA256SolicitudAutorizacionV3(base)
	if err != nil {
		t.Fatal(err)
	}
	datos, _ := base.Datos()
	for nombre, caso := range map[string]struct {
		mutar  func(*DatosVinculoAutenticacionActorV2)
		valido bool
	}{
		"registro": {func(v *DatosVinculoAutenticacionActorV2) {
			v.RegistroContextoRef = "rca_abcdef0123456789abcdef0123456789"
		}, true},
		"esquema contexto": {func(v *DatosVinculoAutenticacionActorV2) { v.ContextoActorEsquema = "otro" }, false},
		"version cuenta":   {func(v *DatosVinculoAutenticacionActorV2) { v.ContextoActorCuentaVersion++ }, true},
		"huella manifiesto": {func(v *DatosVinculoAutenticacionActorV2) {
			v.ManifiestoProcedenciaHuellaSHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		}, true},
		"autoridad": {func(v *DatosVinculoAutenticacionActorV2) { v.AutoridadEfectiva = "otra" }, false},
	} {
		t.Run(nombre, func(t *testing.T) {
			vinculo, _ := datos.VinculoAutenticacionActor.Datos()
			caso.mutar(&vinculo)
			alternativa := datos
			alternativa.VinculoAutenticacionActor = VinculoAutenticacionActorV2{datos: &vinculo}
			solicitud, err := NuevaSolicitudAutorizacionLigadaV3(alternativa)
			if !caso.valido {
				if err == nil {
					t.Fatal("mutacion invalida aceptada")
				}
				return
			}
			otra, err := HuellaSHA256SolicitudAutorizacionV3(solicitud)
			if err != nil || otra == huella {
				t.Fatalf("campo V2 no quedo comprometido: huella=%q err=%v", otra, err)
			}
		})
	}
	// Mapas iguales con orden de insercion diferente deben producir los mismos bytes.
	invertida := datos
	invertida.Recurso.Ambitos = map[string]string{"unidad": "seleccion", "provincia": "granada"}
	invertida.Recurso.Atributos = map[string]string{"estado": "presentado", "fase": "revision"}
	solicitudInvertida, err := NuevaSolicitudAutorizacionLigadaV3(invertida)
	if err != nil {
		t.Fatal(err)
	}
	otra, err := HuellaSHA256SolicitudAutorizacionV3(solicitudInvertida)
	if err != nil || otra != huella {
		t.Fatalf("orden no determinista: %q / %q, %v", huella, otra, err)
	}
}

func TestSolicitudAutorizacionLigadaV3CanonicoEsExhaustivoYNoIncluyePII(t *testing.T) {
	solicitud := solicitudAutorizacionLigadaV3Prueba(t)
	contenido, err := representacionCanonicaSolicitudAutorizacionV3(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	var documento map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &documento); err != nil {
		t.Fatal(err)
	}
	esperados := []string{"esquema", "vinculo_autenticacion_actor", "accion", "recurso", "finalidad", "correlacion_ref", "referencia_motivo"}
	if len(documento) != len(esperados) {
		t.Fatalf("campos canonicos=%d: %s", len(documento), contenido)
	}
	for _, campo := range esperados {
		if _, existe := documento[campo]; !existe {
			t.Fatalf("falta %q en %s", campo, contenido)
		}
	}
	for _, prohibido := range []string{"display_name", "email", "roles", "permissions", "attributes", "claims"} {
		if string(contenido) != "" && contieneJSONClavePrueba(contenido, prohibido) {
			t.Fatalf("PII/claim prohibido %q en %s", prohibido, contenido)
		}
	}
	// La preimagen V3 es privada y explicita: el contrato debe revisarse si el
	// DTO V2 crece o si cualquiera de los dos lados renombra/reordena un campo.
	var vinculo map[string]json.RawMessage
	if err := json.Unmarshal(documento["vinculo_autenticacion_actor"], &vinculo); err != nil {
		t.Fatal(err)
	}
	tipoDTO := reflect.TypeOf(DatosVinculoAutenticacionActorV2{})
	tipoCanon := reflect.TypeOf(vinculoSolicitudAutorizacionCanonicoV3{})
	if len(vinculo) != tipoDTO.NumField() || tipoDTO.NumField() != tipoCanon.NumField() {
		t.Fatalf("campos JSON=%d; DTO V2=%d; canon V3=%d", len(vinculo), tipoDTO.NumField(), tipoCanon.NumField())
	}
	for i := 0; i < tipoDTO.NumField(); i++ {
		campoDTO := tipoDTO.Field(i)
		campoCanon, existeCanon := tipoCanon.FieldByName(campoDTO.Name)
		clave := campoDTO.Tag.Get("json")
		if !existeCanon || campoCanon.Tag.Get("json") != clave {
			t.Fatalf("DTO->canon campo %q/%q sin pareja exacta", campoDTO.Name, clave)
		}
		if _, existe := vinculo[clave]; !existe {
			t.Fatalf("V3 no comprometio %q", clave)
		}
	}
	for i := 0; i < tipoCanon.NumField(); i++ {
		campoCanon := tipoCanon.Field(i)
		campoDTO, existeDTO := tipoDTO.FieldByName(campoCanon.Name)
		if !existeDTO || campoDTO.Tag.Get("json") != campoCanon.Tag.Get("json") {
			t.Fatalf("canon->DTO campo %q/%q sin pareja exacta", campoCanon.Name, campoCanon.Tag.Get("json"))
		}
	}
}

func TestSolicitudAutorizacionLigadaV3VectorCanonicoEstable(t *testing.T) {
	solicitud := solicitudAutorizacionLigadaV3Prueba(t)
	huella, err := HuellaSHA256SolicitudAutorizacionV3(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	const esperada = "5288f38c0a1aacfb5604447d00359f7b51ac9ee8a97014f7e4133b6ba22f1416"
	if huella != esperada {
		t.Fatalf("vector V3 cambio: obtenido=%s esperado=%s", huella, esperada)
	}
}

func TestSolicitudAutorizacionLigadaV3TiemposCanonicosSiempreTienenSeisDecimales(t *testing.T) {
	base := time.Date(2026, 7, 15, 10, 11, 12, 0, time.UTC)
	datos := DatosVinculoAutenticacionActorV2{
		AutenticacionVerificadaEn: base,
		SesionEmitidaEn:           base.Add(123_400 * time.Microsecond),
		SesionValidaHasta:         base.Add(time.Hour),
		SesionRevalidadaEn:        base.Add(time.Second + 123_400*time.Microsecond),
	}
	canon := vinculoSolicitudAutorizacionCanonicoV3Desde(datos)
	if canon.AutenticacionVerificadaEn != "2026-07-15T10:11:12.000000Z" ||
		canon.SesionEmitidaEn != "2026-07-15T10:11:12.123400Z" ||
		canon.SesionValidaHasta != "2026-07-15T11:11:12.000000Z" ||
		canon.SesionRevalidadaEn != "2026-07-15T10:11:13.123400Z" {
		t.Fatalf("precision temporal V3 inestable: %+v", canon)
	}
}

func solicitudAutorizacionLigadaV3Prueba(t *testing.T) SolicitudAutorizacionLigadaV3 {
	t.Helper()
	instante := time.Date(2026, 7, 15, 10, 11, 12, 123_456_000, time.UTC)
	v1 := vinculoAutenticacionActorV1Prueba(t, instante)
	datosV1, err := v1.Datos()
	if err != nil {
		t.Fatal(err)
	}
	vinculo := DatosVinculoAutenticacionActorV2{
		Esquema: EsquemaVinculoAutenticacionActorV2, BloqueVersion: VersionVinculoAutenticacionActorV2,
		AutenticacionRef: datosV1.AutenticacionRef, AutenticacionHuellaSHA256: datosV1.AutenticacionHuellaSHA256,
		AsercionRef: datosV1.AsercionRef, SesionRef: datosV1.SesionRef, ControlSesionRef: datosV1.ControlSesionRef,
		ControlSesionRevision: datosV1.ControlSesionRevision, ControlSesionHuellaSHA256: datosV1.ControlSesionHuellaSHA256,
		CuentaRef: datosV1.CuentaRef, CuentaOrdinariaRef: datosV1.CuentaOrdinariaRef,
		PrincipalID: datosV1.PrincipalID, PerfilActivoRef: datosV1.PerfilActivoRef, CuentaPrivilegiada: datosV1.CuentaPrivilegiada,
		Superficie: datosV1.Superficie, MetodoObservado: datosV1.MetodoObservado, GarantiaObservada: datosV1.GarantiaObservada,
		PoliticaGarantiaRef: datosV1.PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256: datosV1.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn: datosV1.AutenticacionVerificadaEn, SesionEmitidaEn: datosV1.SesionEmitidaEn,
		SesionValidaHasta: datosV1.SesionValidaHasta, SesionRevalidadaEn: datosV1.SesionRevalidadaEn,
		RegistroContextoRef: "rca_0123456789abcdef0123456789abcdef", ContextoActorEsquema: EsquemaRepresentacionContextoActorV2,
		ContextoActorRef: datosV1.ContextoActorRef, ContextoActorVersion: datosV1.ContextoActorVersion,
		ContextoActorCuentaVersion: 1, ContextoActorHuellaSHA256: datosV1.ContextoActorHuellaSHA256,
		ManifiestoProcedenciaHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		AutoridadEfectiva:                 AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	solicitud, err := NuevaSolicitudAutorizacionLigadaV3(DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: VinculoAutenticacionActorV2{datos: &vinculo},
		ReferenciaMotivo:          referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba), Accion: "bolsa.merito.revisar",
		Recurso: RecursoAutorizable{Referencia: "merito:456", ModuloID: "bolsa", Tipo: "merito",
			Ambitos:   map[string]string{"provincia": "granada", "unidad": "seleccion"},
			Atributos: map[string]string{"fase": "revision", "estado": "presentado"}},
		Finalidad: "gestion_bolsa", Correlacion: referenciaCorrelacionAutorizacionV2ParaPrueba(t, referenciaCorrelacionAutorizacionV2Prueba),
	})
	if err != nil {
		t.Fatalf("NuevaSolicitudAutorizacionLigadaV3() error = %v", err)
	}
	return solicitud
}

func contieneJSONClavePrueba(contenido []byte, clave string) bool {
	var documento any
	_ = json.Unmarshal(contenido, &documento)
	return contieneClavePrueba(documento, clave)
}

func contieneClavePrueba(valor any, buscada string) bool {
	switch actual := valor.(type) {
	case map[string]any:
		for clave, hijo := range actual {
			if clave == buscada || contieneClavePrueba(hijo, buscada) {
				return true
			}
		}
	case []any:
		for _, hijo := range actual {
			if contieneClavePrueba(hijo, buscada) {
				return true
			}
		}
	}
	return false
}
