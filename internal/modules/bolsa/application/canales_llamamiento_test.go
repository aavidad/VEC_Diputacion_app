package application

import (
	"context"
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type canalPrueba struct {
	nombre     string
	disponible bool
}

func (c canalPrueba) Canal() string                   { return c.nombre }
func (c canalPrueba) Disponible(context.Context) bool { return c.disponible }

func catalogoCanalesPrueba(smsActivo bool) dominiobolsa.CatalogoCanalesLlamamiento {
	return dominiobolsa.CatalogoCanalesLlamamiento{Version: "bolsa-canales-llamamiento-v1", Canales: []dominiobolsa.CanalLlamamiento{
		{Canal: "correo", Modo: "automatico", AlEmitir: true, Resultados: []string{"enviado"}, Activo: true},
		{Canal: "telefono", Modo: "manual", Seguimiento: true, Resultados: []string{"contactado", "acepta"}, ResultadosCierre: []string{"acepta"}, Activo: true},
		{Canal: "sms", Modo: "automatico", Seguimiento: true, Resultados: []string{"enviado"}, LimiteCaracteres: 160, Activo: smsActivo},
	}}
}

func TestRegistroCanalesPublicaSoloActivosYDisponibles(t *testing.T) {
	ctx := context.Background()
	r, err := NuevoRegistroCanalesLlamamiento(catalogoCanalesPrueba(true), canalPrueba{"correo", true}, canalPrueba{"telefono", false})
	if err != nil {
		t.Fatal(err)
	}
	activos := r.Activos(ctx)
	if len(activos) != 1 || activos[0].Canal != "correo" {
		t.Fatalf("activos: %+v", activos)
	}
	if faltan := r.CanalesSinProveedor(); len(faltan) != 1 || faltan[0] != "sms" {
		t.Fatalf("sin proveedor: %v", faltan)
	}
	r, _ = NuevoRegistroCanalesLlamamiento(catalogoCanalesPrueba(false), canalPrueba{"correo", true}, canalPrueba{"telefono", true}, canalPrueba{"sms", true})
	activos = r.Activos(ctx)
	if len(activos) != 2 || activos[1].Canal != "telefono" || activos[1].ResultadosCierre[0] != "acepta" {
		t.Fatalf("SMS apagado por catálogo publicado: %+v", activos)
	}
	activos[1].ResultadosCierre[0] = "mutado"
	if r.Activos(ctx)[1].ResultadosCierre[0] != "acepta" {
		t.Fatal("la lista publicada comparte memoria con el catálogo")
	}
	for _, malos := range [][]canalPrueba{{{"correo", true}, {"correo", true}}, {{"telegram", true}}} {
		adaptadores := make([]puertosbolsa.CanalAvisoLlamamiento, 0)
		for _, a := range malos {
			adaptadores = append(adaptadores, a)
		}
		if _, err := NuevoRegistroCanalesLlamamiento(catalogoCanalesPrueba(false), adaptadores...); err == nil {
			t.Fatalf("adaptadores %v admitidos", malos)
		}
	}
}
