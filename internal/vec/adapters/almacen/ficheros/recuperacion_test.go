package ficheros

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestSubidaLocalFicherosConservaResultadoOriginalYRechazaAlteracion(t *testing.T) {
	a:=nuevoPrueba(t,directorioPrueba(t))
	sol:=escritura(t,"recuperacion","clave:recuperacion:1",ports.ZonaAlmacenCuarentena,
		[]byte("%PDF-1.4\noriginal de prueba\n%%EOF"))
	resultado,err:=a.Escribir(context.Background(),sol)
	if err!=nil { t.Fatal(err) }
	registro,ok,err:=a.cargarIdempotencia(sol.ClaveIdempotencia)
	if err!=nil || !ok || registro.ResultadoOriginal==nil ||
		!reflect.DeepEqual(*registro.ResultadoOriginal,resultado) {
		t.Fatalf("recibo original no conservado: %v",err)
	}
	alterado:=registro
	copia:=*registro.ResultadoOriginal
	copia.Evidencia.ReintentoIdempotente=true
	alterado.ResultadoOriginal=&copia
	if err:=a.guardarIdempotencia(sol.ClaveIdempotencia,alterado); err!=nil { t.Fatal(err) }
	if _,_,err:=a.cargarIdempotencia(sol.ClaveIdempotencia); !errors.Is(err,ports.ErrIntegridadObjetoAlmacen) {
		t.Fatalf("recibo adulterado admitido: %v",err)
	}
}
