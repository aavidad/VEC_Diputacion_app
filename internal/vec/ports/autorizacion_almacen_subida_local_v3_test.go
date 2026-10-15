package ports

import (
	"testing"
	"time"
)

func TestAlmacenSubidaLocalV3FasesNoPrestanCapacidad(t *testing.T) {
	casos := []struct {
		fase   FaseAlmacenSubidaLocal
		paso   string
		objeto bool
	}{
		{FaseAlmacenSubidaLocalEscribir, AccionAlmacenEscribir, false},
		{FaseAlmacenSubidaLocalAnalizar, AccionAlmacenLeer, true},
		{FaseAlmacenSubidaLocalPromover, AccionAlmacenPromover, true},
	}
	for _, caso := range casos {
		t.Run(string(caso.fase), func(t *testing.T) {
			e := nuevoAlmacenV3Prueba(t, opcionesAlmacenV3Prueba{
				accion: accionAlmacenSubirComprobanteMeritos,
				campos: []string{"contenido", "estado", "recibo"}, requiereObjeto: caso.objeto,
			})
			instante := e.ahora.Add(2 * time.Second)
			ctx, err := NuevoContextoSubidaLocalAlmacenV3(
				e.solicitud, e.decision, e.confirmacion, e.vinculos, caso.fase, instante)
			if err != nil || ctx.ValidarParaEn(caso.paso, instante) != nil {
				t.Fatalf("fase V3 rechazada: %v", err)
			}
			for _, otra := range []string{AccionAlmacenEscribir, AccionAlmacenLeer, AccionAlmacenPromover} {
				if otra != caso.paso && ctx.ValidarParaEn(otra, instante) == nil {
					t.Fatalf("fase %s prestó %s", caso.fase, otra)
				}
			}
			if _, err := ctx.EvidenciaAutorizacion(); err == nil {
				t.Fatal("capacidad V3 proyectada como decisión V1")
			}
			if caso.objeto {
				otro := e.vinculos
				otro.ObjetoVinculado.Version = "otra_version"
				if _, err := NuevoContextoSubidaLocalAlmacenV3(e.solicitud, e.decision,
					e.confirmacion, otro, caso.fase, instante); err == nil {
					t.Fatal("se cambió la versión del objeto")
				}
			}
		})
	}
}

func TestAlmacenSubidaLocalBolsaRRHHSoloPermiteLectura(t *testing.T) {
	e := nuevoAlmacenV3Prueba(t, opcionesAlmacenV3Prueba{
		accion: accionAlmacenDescargarJustificanteRenunciaRRHHBolsa,
		campos: []string{"contenido"}, requiereObjeto: true,
	})
	instante := e.ahora.Add(2 * time.Second)
	ctx, err := NuevoContextoSubidaLocalAlmacenV3(e.solicitud, e.decision,
		e.confirmacion, e.vinculos, FaseAlmacenSubidaLocalDescargar, instante)
	if err != nil || ctx.ValidarParaEn(AccionAlmacenLeer, instante) != nil {
		t.Fatalf("lectura RRHH denegada: %v", err)
	}
	if _, err := NuevoContextoSubidaLocalAlmacenV3(e.solicitud, e.decision,
		e.confirmacion, e.vinculos, FaseAlmacenSubidaLocalEscribir, instante); err == nil {
		t.Fatal("la capacidad RRHH se reutilizó para escribir")
	}
}
