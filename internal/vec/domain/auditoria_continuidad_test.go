package domain

import (
	"strings"
	"testing"
)

func reciboContinuidadPrueba(primera, ultima uint64, anterior, cabeza string) ReciboCheckpointDesarrollo {
	p := PoliticaCheckpoint{Version: 1, PoliticaRef: "politica:sintetica", PoliticaVersion: 1, ClaveRef: "clave:sintetica", ClaveVersion: 1,
		ProveedorKMS: "kms:sintetico", ProveedorKMSVersion: 1, ProveedorTSA: "tsa:sintetica", ProveedorTSAVersion: 1, OperacionTSA: "checkpoint.sintetico", Modo: "DESARROLLO"}
	c := CheckpointDesarrollo{Esquema: EsquemaCheckpointDesarrollo, Politica: p, Cobertura: CoberturaCheckpoint{
		CadenaID: "cadena:sintetica", PrimeraSecuencia: primera, UltimaSecuencia: ultima, AnteriorSHA256: anterior, CabezaSHA256: cabeza}}
	if primera != 0 {
		c.Cobertura.Registros = ultima - primera + 1
	}
	b, _ := c.Canonico()
	return ReciboCheckpointDesarrollo{Checkpoint: c, PinSPKISHA256: strings.Repeat("a", 64), TSA: ReciboTSACheckpoint{
		Referencia: "tsa-desarrollo:hmac-sha256:" + strings.Repeat("b", 64), HuellaPreimagenSHA256: strings.Repeat("c", 64), HuellaCheckpointSHA256: HuellaCheckpoint(b), Autoridad: "no_autoritativo", Esquema: "vec.tsa.desarrollo.v1"}}
}

func TestContinuidadCheckpointCoordenadasYLimites(t *testing.T) {
	z, h := strings.Repeat("0", 64), strings.Repeat("1", 64)
	ancla := reciboContinuidadPrueba(1, 2, z, h)
	siguiente := reciboContinuidadPrueba(3, 4, h, strings.Repeat("2", 64))
	if r := CotejarContinuidadCheckpoint(ancla, []ReciboCheckpointDesarrollo{siguiente}, 4, 1); r.Continuidad != "verificada" || r.RegistrosTotal != 4 || r.RecibosVerificados != 2 || r.PrimeraSecuencia != 1 || r.UltimaSecuencia != 4 {
		t.Fatalf("rango: %+v", r)
	}
	if r := CotejarContinuidadCheckpoint(reciboContinuidadPrueba(0, 0, z, z), []ReciboCheckpointDesarrollo{ancla}, 2, 1); r.Continuidad != "verificada" {
		t.Fatal(r)
	}
	for nombre, tramo := range map[string]ReciboCheckpointDesarrollo{
		"hueco": reciboContinuidadPrueba(4, 5, h, h), "solape": reciboContinuidadPrueba(2, 3, h, h), "repetido": ancla,
		"retroceso": reciboContinuidadPrueba(1, 1, z, h), "vacio": reciboContinuidadPrueba(0, 0, z, z),
		"enlace": reciboContinuidadPrueba(3, 4, strings.Repeat("8", 64), h),
	} {
		t.Run(nombre, func(t *testing.T) {
			if CotejarContinuidadCheckpoint(ancla, []ReciboCheckpointDesarrollo{tramo}, 10, 1).Continuidad != "rechazada" {
				t.Fatal("tramo admitido")
			}
		})
	}
	for _, limite := range []uint64{0, 1, 3} {
		if CotejarContinuidadCheckpoint(ancla, []ReciboCheckpointDesarrollo{siguiente}, limite, 1).Continuidad != "rechazada" {
			t.Fatal("presupuesto admitido")
		}
	}
	for _, max := range []int{0, 257} {
		if CotejarContinuidadCheckpoint(ancla, []ReciboCheckpointDesarrollo{siguiente}, 4, max).Continuidad != "rechazada" {
			t.Fatal("max-recibos admitido")
		}
	}
	for _, cambiar := range []func(*ReciboCheckpointDesarrollo){
		func(r *ReciboCheckpointDesarrollo) { r.PinSPKISHA256 = strings.Repeat("9", 64) },
		func(r *ReciboCheckpointDesarrollo) { r.Checkpoint.Politica.ClaveVersion++ },
		func(r *ReciboCheckpointDesarrollo) { r.Checkpoint.Cobertura.CadenaID = "cadena:otra" },
	} {
		r := siguiente
		cambiar(&r)
		b, _ := r.Checkpoint.Canonico()
		r.TSA.HuellaCheckpointSHA256 = HuellaCheckpoint(b)
		if CotejarContinuidadCheckpoint(ancla, []ReciboCheckpointDesarrollo{r}, 4, 1).Continuidad != "rechazada" {
			t.Fatal("rotación o cadena distinta admitida")
		}
	}
	tope := uint64(9007199254740991)
	final := reciboContinuidadPrueba(tope, tope, h, h)
	if CotejarContinuidadCheckpoint(final, []ReciboCheckpointDesarrollo{siguiente}, 10, 1).Continuidad != "rechazada" {
		t.Fatal("límite de secuencia admitido")
	}
}
