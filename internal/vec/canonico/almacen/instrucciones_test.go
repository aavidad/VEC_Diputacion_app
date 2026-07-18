package almacen

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func datosInstruccionesPrueba() DatosInstruccionesCargaDirecta {
	emitida := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	return DatosInstruccionesCargaDirecta{
		ConectorID: "almacen_s3_corporativo", SesionRef: "sesion:carga:abcdefghijkl",
		Metodo: MetodoCargaDirectaPUT, Destino: "https://objetos.example.test/carga?firma=opaca",
		Cabeceras: []CabeceraCargaDirecta{{Nombre: "content-type", Valor: "application/pdf"}},
		EmitidaEn: emitida, ExpiraEn: emitida.Add(5 * time.Minute), TamanoMaximo: 4096,
	}
}

func TestInstruccionesOpacasCopianEntradaYSalida(t *testing.T) {
	t.Parallel()

	datos := datosInstruccionesPrueba()
	instrucciones, err := NuevasInstruccionesCargaDirecta(datos)
	if err != nil {
		t.Fatalf("crear instrucciones: %v", err)
	}
	datos.Cabeceras[0].Valor = "text/plain"

	proyectadas, _, err := instrucciones.DatosVerificados()
	if err != nil || proyectadas.Cabeceras[0].Valor != "application/pdf" {
		t.Fatalf("la entrada altero la concesion: %#v, %v", proyectadas.Cabeceras, err)
	}
	proyectadas.Cabeceras[0].Valor = "image/png"
	segunda, _, err := instrucciones.DatosVerificados()
	if err != nil || segunda.Cabeceras[0].Valor != "application/pdf" {
		t.Fatalf("la salida altero la concesion: %#v, %v", segunda.Cabeceras, err)
	}
}

func TestInstruccionesVinculadasExigenSolicitudExacta(t *testing.T) {
	t.Parallel()

	datos := datosInstruccionesPrueba()
	instrucciones, err := NuevasInstruccionesCargaDirecta(datos)
	if err != nil {
		t.Fatal(err)
	}
	capacidades := Capacidades{
		ConectorID: datos.ConectorID, CargaDirectaTemporal: true, TamanoMaximoObjeto: 8192,
		OrigenesCargaDirecta: []string{"https://objetos.example.test"},
	}
	huella := strings.Repeat("a", 64)
	if !errors.Is(
		instrucciones.ValidarPara(datos.TamanoMaximo, datos.ExpiraEn, huella, capacidades),
		ErrInstruccionesCargaDirectaNoValidas,
	) {
		t.Fatal("una concesion sin vinculo acepto una solicitud")
	}
	vinculadas, err := instrucciones.VincularSolicitud(huella)
	if err != nil || vinculadas.ValidarPara(datos.TamanoMaximo, datos.ExpiraEn, huella, capacidades) != nil {
		t.Fatalf("la concesion ligada fue rechazada: %v", err)
	}
	if vinculadas.ValidarPara(datos.TamanoMaximo+1, datos.ExpiraEn, huella, capacidades) == nil ||
		vinculadas.ValidarPara(datos.TamanoMaximo, datos.ExpiraEn, strings.Repeat("b", 64), capacidades) == nil {
		t.Fatal("la concesion acepto tamano o huella distintos")
	}
}

func TestInstruccionesRespetanVigenciaSemiabiertaYOrigen(t *testing.T) {
	t.Parallel()

	datos := datosInstruccionesPrueba()
	instrucciones, err := NuevasInstruccionesCargaDirecta(datos)
	if err != nil {
		t.Fatal(err)
	}
	if !instrucciones.VigenteEn(datos.EmitidaEn) || instrucciones.VigenteEn(datos.ExpiraEn) {
		t.Fatal("la ventana no es [emitida, expira)")
	}
	capacidades := Capacidades{
		ConectorID: datos.ConectorID, CargaDirectaTemporal: true, TamanoMaximoObjeto: datos.TamanoMaximo,
		OrigenesCargaDirecta: []string{"https://otro.example.test"},
	}
	if !errors.Is(instrucciones.ValidarContra(capacidades), ErrInstruccionesCargaDirectaNoValidas) {
		t.Fatal("se acepto un destino fuera de los origenes declarados")
	}
	capacidades.OrigenesCargaDirecta = []string{"https://objetos.example.test"}
	if err := instrucciones.ValidarContra(capacidades); err != nil {
		t.Fatalf("se rechazo el origen exacto: %v", err)
	}
}
