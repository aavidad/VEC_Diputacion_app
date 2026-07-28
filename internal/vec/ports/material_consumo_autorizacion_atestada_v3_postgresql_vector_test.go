package ports

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"testing"
	"time"
)

const capacidadCanonicaPostgreSQLV3 = `{"esquema":"vec.autorizacion.capacidad-registro-consumo-atestado.v3","version":3,"clave_id":"clave-capacidad-vector-pg","clave_version":7,"revision_gobierno":9,"huella_gobierno_sha256":"1111111111111111111111111111111111111111111111111111111111111111","emisor_id":"broker-vector-pg","audiencia_consumo":"vec_contratacion_temporal.consultar_cuadro_rrhh.v3","nonce":"2222222222222222222222222222222222222222222222222222222222222222","emitida_en":"2026-07-28T08:09:10.123456Z","expira_en":"2026-07-28T08:09:14.123456Z","decision_ref":"decision:rrhh:vector-pg","huella_decision_sha256":"3333333333333333333333333333333333333333333333333333333333333333","huella_motivo_sha256":"4444444444444444444444444444444444444444444444444444444444444444","huella_payload_vec_ad_3_sha256":"5555555555555555555555555555555555555555555555555555555555555555","huella_sobre_cose_sign1_sha256":"6666666666666666666666666666666666666666666666666666666666666666","huella_prueba_confianza_sha256":"7777777777777777777777777777777777777777777777777777777777777777","contexto_ref":"contexto:rrhh:vector-pg","huella_contexto_sha256":"8888888888888888888888888888888888888888888888888888888888888888","audiencia_despliegue":"vec-diputacion/pruebas/ct000042","operacion":"contratacion_temporal.consultar_cuadro_rrhh","efecto_ref":"consulta:rrhh:vector-pg","huella_efecto_sha256":"9999999999999999999999999999999999999999999999999999999999999999","decision_valida_hasta":"2026-07-28T08:10:00Z","verificada_en":"2026-07-28T08:09:10.123456Z","revision_confianza":"configuracion-vector-pg","configuracion_secuencia":4,"huella_configuracion_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","configuracion_publicada_en":"2026-07-28T08:00:00Z","configuracion_expira_en":"2026-07-28T09:00:00Z","raiz_clave_id":"raiz-vector-pg","raiz_version":3,"huella_raiz_spki_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","raiz_valida_desde":"2026-07-28T07:00:00Z","raiz_valida_hasta":"2026-07-28T09:00:00Z","suite":"VEC-AD-3-COSE-EDDSA-1","mac_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}`

const huellaMaterialPostgreSQLV3 = "7195a7271ec37370688df27f8e3dfddf4ea91864babf4906910b8faf81f6a0a3"

func TestVectorPostgreSQLHuellaMaterialConsumoV3(t *testing.T) {
	t.Parallel()
	emitida := time.Date(2026, 7, 28, 8, 9, 10, 123456000, time.UTC)
	expira := emitida.Add(4 * time.Second)
	resumen, err := NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:rrhh:vector-pg",
		string(bytes.Repeat([]byte{'3'}, 64)),
		string(bytes.Repeat([]byte{'4'}, 64)),
		"contexto:rrhh:vector-pg",
		string(bytes.Repeat([]byte{'8'}, 64)),
		"contratacion_temporal.consultar_cuadro_rrhh",
		"consulta:rrhh:vector-pg",
		string(bytes.Repeat([]byte{'9'}, 64)),
		"vec_contratacion_temporal.consultar_cuadro_rrhh.v3",
		emitida,
		expira,
	)
	if err != nil {
		t.Fatal(err)
	}
	privada := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, 32))
	spki, err := x509.MarshalPKIXPublicKey(privada.Public())
	if err != nil {
		t.Fatal(err)
	}
	exportacion, err := NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		[]byte(capacidadCanonicaPostgreSQLV3),
		resumen,
		[]byte("decision-canonica-vector-pg"),
		[]byte("motivo-canonico-vector-pg"),
		[]byte("contexto-actor-canonico-vector-pg"),
		7,
		11,
		[]byte("payload-vec-ad-3-vector-pg"),
		[]byte("sobre-cose-sign1-vector-pg"),
		[]byte("evidencia-verificacion-vector-pg"),
		spki,
	)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := exportacion.HuellaConjuntoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if huella != huellaMaterialPostgreSQLV3 {
		t.Fatalf("vector material PostgreSQL cambió: %s", huella)
	}
}
