package bootstrap

import (
	"context"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestAuditoriaNominalDistingueConsultaYRecuperacionR5(t *testing.T) {
	for _, caso := range []struct {
		ruta, accion string
		admite       bool
	}{
		{httpinterno.RutaConsultaFirmasR5V2, ports.AccionConsultarFirmasR5V2, true},
		{httpinterno.RutaRecuperacionFirmasR5V2, ports.AccionRecuperarFirmasR5V2, true},
		{httpinterno.RutaConsultaFirmasR5V2, ports.AccionRecuperarFirmasR5V2, false},
		{httpinterno.RutaRecuperacionFirmasR5V2, ports.AccionConsultarFirmasR5V2, false},
		{httpinterno.RutaRegistroFirmaVec, ports.AccionRecuperarFirmasR5V2, false},
		{httpinterno.RutaRecuperacionFirmasR5V2, ports.AccionRegistrarFirmaVec, false},
	} {
		if accionAuditoriaLecturaFirmaNominalCT(caso.ruta, caso.accion) != caso.admite {
			t.Fatalf("auditoría cruzó ruta y acción: %s %s", caso.ruta, caso.accion)
		}
	}
}

type revalidadorNominalFijoPrueba struct {
	actual core.AutenticacionRevalidadaV1
}

func (r revalidadorNominalFijoPrueba) RevalidarAutenticacionActorV1(
	_ context.Context, _ core.SolicitudRevalidacionAutenticacionActorV1,
) (core.AutenticacionRevalidadaV1, error) {
	return r.actual, nil
}

func TestContextoNominalFirmaPorPeticionConservaReciboYRechazaDeriva(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	base := soporte.contexto
	datos, err := base.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	reloj := &relojSesionConsultaPrueba{ahora: base.Resultado.ResueltoEnAutoritativo.Add(time.Second)}
	resolutor := &resolutorSesionConsultaPrueba{base: base.Resultado, reloj: reloj}
	autenticacion := datos.Autenticacion()
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: datos.CuentaRef,
		Metodo: datos.MetodoObservado, Garantia: datos.GarantiaObservada}
	vinculo, resultado, err := core.CrearVinculoAutenticacionActorV2ConResultado(context.Background(),
		revalidadorNominalFijoPrueba{actual: autenticacion},
		core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: datos.AutenticacionRef, SesionRef: datos.SesionRef},
		resolutor, core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: datos.PerfilActivoRef}, reloj)
	if err != nil || resultado.RegistroContextoRef == base.Resultado.RegistroContextoRef {
		t.Fatalf("no se obtuvo un recibo F1 fresco: %v", err)
	}
	certificado := soporte.certificadoSHA256
	primera := ports.EvidenciaSesionFirmanteV2{Vinculo: base.Vinculo, Resultado: base.Resultado,
		CertificadoCanalSHA256: certificado, CertificadoValidoHasta: reloj.Ahora().Add(time.Minute)}
	fresca := ports.EvidenciaSesionFirmanteV2{Vinculo: vinculo, Resultado: resultado,
		CertificadoCanalSHA256: certificado, CertificadoValidoHasta: primera.CertificadoValidoHasta}
	p := &peticionNominalFirmaCT{contextos: make(map[string]contextoNominalFirmaCT, 4)}
	guardada, err := p.conservarContextoNominalFirmaCT(primera, soporte.instantanea, reloj.Ahora())
	if err != nil || guardada.Resultado.RegistroContextoRef != primera.Resultado.RegistroContextoRef {
		t.Fatalf("el primer contexto no quedó fijado: %v", err)
	}
	const concurrentes = 12
	var wg sync.WaitGroup
	errores := make(chan error, concurrentes)
	for i := 0; i < concurrentes; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.mu.Lock()
			actual, err := p.conservarContextoNominalFirmaCT(fresca, soporte.instantanea, reloj.Ahora())
			p.mu.Unlock()
			if err == nil && actual.Resultado.RegistroContextoRef != primera.Resultado.RegistroContextoRef {
				err = ports.ErrFirmaDocumentoDenegada
			}
			errores <- err
		}()
	}
	wg.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("una revalidación concurrente cambió el recibo fijado: %v", err)
		}
	}
	alterada := soporte.instantanea
	alterada.ControlVigenciaVersionRol.Revision++
	if _, err := p.conservarContextoNominalFirmaCT(fresca, alterada, reloj.Ahora()); err == nil {
		t.Fatal("se aceptó un control de rol diferente dentro de la petición")
	}
	retirada := soporte.instantanea
	retirada.ControlVigenciaVersionRol.Estado = core.EstadoControlVigenciaVersionRolRetirada
	retirada.ControlVigenciaVersionRol.ActoRef = "acto:retirada"
	retirada.ControlVigenciaVersionRol.MotivoCodigo = "retirada"
	if _, err := p.conservarContextoNominalFirmaCT(fresca, retirada, reloj.Ahora()); err == nil {
		t.Fatal("se aceptó un control de rol retirado")
	}
	fresca.CertificadoCanalSHA256 = soporte.certificadoSHA256[:63] + "0"
	if _, err := p.conservarContextoNominalFirmaCT(fresca, soporte.instantanea, reloj.Ahora()); err == nil {
		t.Fatal("se aceptó otro certificado en el contexto ya fijado")
	}
}
