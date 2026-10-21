package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func TestFirmaNominalDescargaDenegadaCreaOrdenConContextoYCorrelacion(t *testing.T) {
	e := nuevaCapsulaContextoFijadoPrueba(t, 0)
	p := e.peticion.Context().Value(clavePeticionNominalFirmaCT{}).(*peticionNominalFirmaCT)
	p.fuente.proceso = "vec-rrhh"
	const recurso = "expediente:prueba"
	intento, err := vp.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre, ruta, accion string
		admite               bool
	}{
		{"descarga", httpinterno.RutaDescargaFirmaR5V2, ports.AccionConsultarFirmasR5V2, true},
		{"accion_ajena", httpinterno.RutaDescargaFirmaR5V2, ports.AccionRecuperarFirmasR5V2, false},
		{"otra_ruta", httpinterno.RutaRegistroFirmaVec, ports.AccionConsultarFirmasR5V2, false},
		{"recuperacion", httpinterno.RutaRecuperacionFirmasR5V2, ports.AccionRecuperarFirmasR5V2, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			p.peticion = httptest.NewRequest(http.MethodPost, caso.ruta, nil)
			ctx, err := vp.ConCorrelacionIncidenciasPeticion(context.WithValue(p.peticion.Context(), clavePeticionNominalFirmaCT{}, p))
			if err != nil {
				t.Fatal(err)
			}
			orden, err := p.fuente.CrearOrdenIntentoFirma(ctx, intento, caso.accion, recurso, core.ResultadoIntentoAuditoriaDenegado)
			if !caso.admite {
				if !errors.Is(err, ports.ErrRegistroFirmaDocumentoNoDisponible) {
					t.Fatalf("ruta o acción ajena admitida: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			datos, err := orden.Datos()
			if err != nil {
				t.Fatal(err)
			}
			correlacion, err := vp.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
			if err != nil {
				t.Fatal(err)
			}
			ref, err := correlacion.ValorCanonico()
			if err != nil {
				t.Fatal(err)
			}
			if datos.IntentoRef != intento || datos.Datos.Accion != caso.accion || datos.Datos.RecursoRef != recurso ||
				datos.Datos.Resultado != core.ResultadoIntentoAuditoriaDenegado || datos.Datos.CorrelacionRef != ref ||
				datos.ResultadoContexto.RegistroContextoRef != e.primera.Resultado.RegistroContextoRef ||
				datos.Vinculo.ValidarPara(datos.ResultadoContexto) != nil {
				t.Fatal("la orden no conserva el intento, el contexto nominal o la correlación")
			}
		})
	}
}
