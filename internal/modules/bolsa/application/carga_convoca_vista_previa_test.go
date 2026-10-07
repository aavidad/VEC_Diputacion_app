package application

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/adapters/xlsconvoca"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

func ejemploCargaConvoca(t *testing.T) []byte {
	t.Helper()
	contenido, err := os.ReadFile(filepath.Join("testdata", "carga_convoca", "carga_convoca_ejemplo.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}

func TestVistaPreviaCargaConvocaDelEjemploSintetico(t *testing.T) {
	p, err := NuevoPrevisualizadorCargaConvoca(xlsconvoca.NuevoLector())
	if err != nil {
		t.Fatal(err)
	}
	vista, err := p.Previsualizar(context.Background(), "carga_convoca_ejemplo.xlsx", ejemploCargaConvoca(t))
	if err != nil {
		t.Fatal(err)
	}
	if vista.Esquema != string(importacion.EsquemaResumenPersona) || vista.Bloqueo != "" || vista.FilasLeidas != 12 ||
		vista.Aceptadas != 11 || vista.Rechazadas != 1 || vista.ConAvisos != 2 || len(vista.HuellaSHA256) != 64 {
		t.Fatalf("vista inesperada: %+v", vista)
	}
	if len(vista.Filas) != 12 {
		t.Fatalf("filas = %d", len(vista.Filas))
	}
	primera := vista.Filas[0]
	if primera.Posicion != 1 || primera.Nombre != "Antonio" || primera.PrimerApellido != "Reyes" || primera.Total != "15.75" || primera.Documento != "***4521**" {
		t.Fatalf("la primera posición no es la de mayor puntuación: %+v", primera)
	}
	ultima := vista.Filas[len(vista.Filas)-1]
	if ultima.Estado != EstadoFilaCargaRechazada || ultima.Posicion != 0 || ultima.Nombre != "" || len(ultima.Errores) != 1 ||
		ultima.Errores[0].Codigo != "total_incoherente" || ultima.Errores[0].Campo != "Total" {
		t.Fatalf("la fila con error no va al final o no explica el motivo: %+v", ultima)
	}
	avisos := 0
	for _, fila := range vista.Filas {
		if len(fila.Avisos) == 1 && fila.Avisos[0] == AvisoIdentidadAmbigua {
			avisos++
			if fila.PrimerApellido != "García" || fila.Nombre != "María" {
				t.Fatalf("aviso en una fila que no es la repetida: %+v", fila)
			}
		}
	}
	if avisos != 2 {
		t.Fatalf("avisos de identidad ambigua = %d", avisos)
	}
}

func TestVistaPreviaMantieneAvisosYBloqueaIdentidadNoDerivable(t *testing.T) {
	identidad := importacion.IdentidadEnmascarada{
		Documento: "***1234**", PrimerApellido: "García", Nombre: "María",
	}
	filas := []importacion.FilaAceptada{
		{Numero: 2, Identidad: identidad},
		{Numero: 3, Identidad: identidad},
		{Numero: 4, Identidad: importacion.IdentidadEnmascarada{Documento: "***5678**", Nombre: "Sin apellido"}},
	}
	avisos, derivable, err := filasIdentidadAmbigua(filas)
	if err != nil || derivable || !avisos[2] || !avisos[3] || avisos[4] {
		t.Fatalf("identidad inválida debe bloquear sin perder avisos previos: avisos=%v derivable=%t err=%v", avisos, derivable, err)
	}
}

type decodificadorCargaPrueba struct {
	hoja importacion.HojaStaging
	err  error
	leyo bool
}

func (d *decodificadorCargaPrueba) Decodificar(context.Context, io.ReadSeeker) (importacion.HojaStaging, error) {
	d.leyo = true
	return d.hoja, d.err
}

func TestVistaPreviaCargaConvocaLimitesAntesDeLeer(t *testing.T) {
	d := &decodificadorCargaPrueba{}
	p, _ := NuevoPrevisualizadorCargaConvoca(d)
	ctx := context.Background()
	if _, err := p.Previsualizar(ctx, "bolsa.xlsx", make([]byte, MaximoBytesCargaConvoca+1)); !errors.Is(err, ErrFicheroCargaConvocaExcesivo) || d.leyo {
		t.Fatalf("tamaño excesivo: err=%v leyo=%v", err, d.leyo)
	}
	for _, nombre := range []string{"", "bolsa.csv", "../bolsa.xlsx", "a/b.xls", " bolsa.xls", ".xlsx", "bolsa\x00.xls", "bolsa\u202exslx.xls"} {
		if _, err := p.Previsualizar(ctx, nombre, []byte("x")); !errors.Is(err, ErrFicheroCargaConvocaInvalido) || d.leyo {
			t.Fatalf("nombre %q admitido: err=%v", nombre, err)
		}
	}
	if _, err := p.Previsualizar(ctx, "bolsa.xls", nil); !errors.Is(err, ErrFicheroCargaConvocaInvalido) || d.leyo {
		t.Fatalf("fichero vacío admitido: %v", err)
	}
	d.hoja = importacion.HojaStaging{Esquema: importacion.EsquemaResumenPersona, Filas: make([]importacion.FilaStaging, MaximoFilasCargaConvoca+1)}
	if _, err := p.Previsualizar(ctx, "bolsa.xls", []byte("x")); !errors.Is(err, ErrFilasCargaConvocaExcesivas) {
		t.Fatalf("filas excesivas: %v", err)
	}
	d.hoja, d.err = importacion.HojaStaging{}, errors.New("libro roto")
	if _, err := p.Previsualizar(ctx, "bolsa.xls", []byte("x")); !errors.Is(err, ErrFicheroCargaConvocaInvalido) {
		t.Fatalf("libro roto: %v", err)
	}
}

func TestVistaPreviaCargaConvocaBloqueaDetalleYSinAceptadas(t *testing.T) {
	cabeceras := importacion.EsquemaDetalleMerito.Cabeceras()
	d := &decodificadorCargaPrueba{hoja: importacion.HojaStaging{Esquema: importacion.EsquemaDetalleMerito, Cabeceras: cabeceras}}
	p, _ := NuevoPrevisualizadorCargaConvoca(d)
	vista, err := p.Previsualizar(context.Background(), "detalle.xls", []byte("x"))
	if err != nil || vista.Bloqueo != BloqueoEsquemaDetalle {
		t.Fatalf("detalle: vista=%+v err=%v", vista, err)
	}
	d.hoja = importacion.HojaStaging{Esquema: importacion.EsquemaResumenPersona, Cabeceras: importacion.EsquemaResumenPersona.Cabeceras()}
	vista, err = p.Previsualizar(context.Background(), "vacio.xls", []byte("x"))
	if err != nil || vista.Bloqueo != BloqueoSinFilasAceptadas {
		t.Fatalf("sin filas: vista=%+v err=%v", vista, err)
	}
}
