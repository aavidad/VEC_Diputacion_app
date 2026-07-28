package ports

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"fmt"
	"strings"
	"testing"
	"time"
)

var huellasMutacionesMaterialPostgreSQLV3 = [...]string{
	"b65739d07b756dfda24371f87a251ed5c00403f229edd12beafb1e19c7d7c2ba",
	"99043ef80ec527b4656c6ebf17fbe70cad3948aa166486c1604582f2662244c9",
	"430d2521333467e61a5970248388b9ee71442da105dda03e297b83138df41e6a",
	"a5a4554a9e6cf3fb4fa49b16826139c17d00de3b9c8c2245b26042b4fd0ee35d",
	"0833ad4684de627151eaa26efddf552f48c8a4ab56853689f52664efd8f57f41",
	"5137b04fa254136f9fec6c9ee675424dcb48f492e968369f948e1a5d0540da91",
	"8879232c6eefebd7b09d953fbd752c18e8548e274423a9d4c4811473803d7a69",
	"95ae589e23cf92e71e4e4e631bc738d003bcae9f5407299e4d445564812c0cb5",
	"937fdb129acf8f263b4ed89e42d203270d83c597e50b228f4ad5cf432aff4c83",
	"d2913bbec135a2a45eca11f2b4ade042f6913107d912c6c3c2d9239ec9945f0c",
	"bcebf071b0b2eb8ae2a8155e44f512078779485f221b6cf12003cb1dc1365547",
	"31565565070d5a0f9346ee7dd6422f2bc0b58cd044446baaa418e20e0bd4a434",
	"5a309a3b596f43379e6a1240693dfd08808f73e0876ee3563b1bb24bfe77a4f8",
	"feb4468ae7dc96885e420d3796102f95a1362de0cff8add425aaffc0be0284c1",
	"62e8c10da41b6f5c246ddcd02865fb76c23da0fa0f6073c58c1022641845b7e1",
	"9c0ef62650e647a9463760fd20f71a47ca1c9b9a77b66029d4a892a7acc2b120",
	"a31801128d07a61c9da3d8a45476709f617d1b2bf9fc518d54be7b022ff9d74b",
	"04fab017f8b25ccdf5b4e2b6b036fe455deeb32f2a0ec472801a7badc02f5bef",
	"c1e63385149893769df8abf1ec3946e26b6ba6f9922a5a9903e2487c865abe5c",
	"4b9eee752de03f8aa2869b025d81c2551da36c25facedb14457d9ab9aa95bc26",
	"7a72e01afd99529e2ab2c562f11147278828c08ade21b4a01f6c7718232f469a",
}

// TestVectoresPostgreSQLMutacionesMaterialConsumoV3 congela en Go la
// salida exacta de cada una de las 21 mutaciones que PostgreSQL reproduce.
func TestVectoresPostgreSQLMutacionesMaterialConsumoV3(t *testing.T) {
	emitida := time.Date(2026, 7, 28, 8, 9, 10, 123456000, time.UTC)
	expira := emitida.Add(4 * time.Second)
	valores := []string{
		"decision:rrhh:vector-pg",
		strings.Repeat("3", 64),
		strings.Repeat("4", 64),
		"contexto:rrhh:vector-pg",
		strings.Repeat("8", 64),
		"contratacion_temporal.consultar_cuadro_rrhh",
		"consulta:rrhh:vector-pg",
		strings.Repeat("9", 64),
		"vec_contratacion_temporal.consultar_cuadro_rrhh.v3",
	}
	sustituciones := []string{
		"decision:rrhh:vector-pg-mutada",
		strings.Repeat("d", 64),
		strings.Repeat("e", 64),
		"contexto:rrhh:vector-pg-mutado",
		strings.Repeat("f", 64),
		"contratacion_temporal.consultar_detalle_rrhh",
		"consulta:rrhh:vector-pg-mutada",
		strings.Repeat("0", 64),
		"vec_contratacion_temporal.consultar_detalle_rrhh.v3",
		"2026-07-28T08:09:10.123455Z",
		"2026-07-28T08:09:14.123455Z",
	}
	privada := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, 32))
	spki, err := x509.MarshalPKIXPublicKey(privada.Public())
	if err != nil {
		t.Fatal(err)
	}
	crearResumen := func(campos []string, desde, hasta time.Time) ResumenCapacidadAtestacionAutorizacionV3 {
		resumen, errResumen := NuevoResumenCapacidadAtestacionAutorizacionV3(
			campos[0], campos[1], campos[2], campos[3], campos[4],
			campos[5], campos[6], campos[7], campos[8], desde, hasta,
		)
		if errResumen != nil {
			t.Fatal(errResumen)
		}
		return resumen
	}
	huella := func(
		capacidad string,
		resumen ResumenCapacidadAtestacionAutorizacionV3,
		decision, motivo, contexto []byte,
		persona, perfil uint64,
		payload, cose, evidencia, raiz []byte,
	) string {
		exportacion, errExportacion :=
			NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
				[]byte(capacidad), resumen, decision, motivo, contexto,
				persona, perfil, payload, cose, evidencia, raiz,
			)
		if errExportacion != nil {
			t.Fatal(errExportacion)
		}
		resultado, errHuella := exportacion.HuellaConjuntoSHA256()
		if errHuella != nil {
			t.Fatal(errHuella)
		}
		return resultado
	}
	resumenBase := crearResumen(valores, emitida, expira)
	calcular := func(
		capacidad string,
		resumen ResumenCapacidadAtestacionAutorizacionV3,
		indice int,
	) string {
		decision := []byte("decision-canonica-vector-pg")
		motivo := []byte("motivo-canonico-vector-pg")
		contexto := []byte("contexto-actor-canonico-vector-pg")
		persona, perfil := uint64(7), uint64(11)
		payload := []byte("payload-vec-ad-3-vector-pg")
		cose := []byte("sobre-cose-sign1-vector-pg")
		evidencia := []byte("evidencia-verificacion-vector-pg")
		raiz := bytes.Clone(spki)
		switch indice {
		case 13:
			decision = []byte{2}
		case 14:
			motivo = []byte{2}
		case 15:
			contexto = []byte{2}
		case 16:
			persona = 8
		case 17:
			perfil = 12
		case 18:
			payload = []byte{2}
		case 19:
			cose = []byte{2}
		case 20:
			evidencia = []byte{2}
		case 21:
			raiz[len(raiz)-1]++
		}
		return huella(
			capacidad, resumen, decision, motivo, contexto, persona, perfil,
			payload, cose, evidencia, raiz,
		)
	}
	capacidadMutada := strings.TrimSuffix(
		capacidadCanonicaPostgreSQLV3, "}",
	) + `,"marca_prueba":"mutada"}`
	obtenidas := []string{calcular(capacidadMutada, resumenBase, 1)}
	claves := []string{
		"decision_ref", "huella_decision_sha256",
		"huella_motivo_sha256", "contexto_ref",
		"huella_contexto_sha256", "operacion", "efecto_ref",
		"huella_efecto_sha256", "audiencia_consumo",
	}
	for indice := 0; indice < 11; indice++ {
		campos := append([]string(nil), valores...)
		desde, hasta := emitida, expira
		var capacidad string
		if indice < 9 {
			campos[indice] = sustituciones[indice]
			anterior := fmt.Sprintf(`"%s":"%s"`, claves[indice], valores[indice])
			nueva := fmt.Sprintf(`"%s":"%s"`, claves[indice], sustituciones[indice])
			capacidad = strings.Replace(capacidadCanonicaPostgreSQLV3, anterior, nueva, 1)
		} else if indice == 9 {
			desde = time.Date(2026, 7, 28, 8, 9, 10, 123455000, time.UTC)
			capacidad = strings.Replace(
				capacidadCanonicaPostgreSQLV3,
				`"emitida_en":"2026-07-28T08:09:10.123456Z"`,
				`"emitida_en":"2026-07-28T08:09:10.123455Z"`, 1,
			)
		} else {
			hasta = time.Date(2026, 7, 28, 8, 9, 14, 123455000, time.UTC)
			capacidad = strings.Replace(
				capacidadCanonicaPostgreSQLV3,
				`"expira_en":"2026-07-28T08:09:14.123456Z"`,
				`"expira_en":"2026-07-28T08:09:14.123455Z"`, 1,
			)
		}
		obtenidas = append(
			obtenidas,
			calcular(capacidad, crearResumen(campos, desde, hasta), indice+2),
		)
	}
	for indice := 13; indice <= 21; indice++ {
		obtenidas = append(
			obtenidas,
			calcular(capacidadCanonicaPostgreSQLV3, resumenBase, indice),
		)
	}
	if len(obtenidas) != len(huellasMutacionesMaterialPostgreSQLV3) {
		t.Fatalf("matriz de mutaciones incompleta: %d", len(obtenidas))
	}
	for indice, obtenida := range obtenidas {
		if obtenida != huellasMutacionesMaterialPostgreSQLV3[indice] {
			t.Fatalf(
				"mutación material %d divergente: %s", indice+1, obtenida,
			)
		}
	}
}
