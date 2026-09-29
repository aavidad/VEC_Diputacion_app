package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	seguridadcontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func sellosSubsanacionTransicionPrueba(t *testing.T) *seguridadcontratacion.AutoridadSellosSubsanacionReparoHMAC {
	t.Helper()
	derivador := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	ambito, _, err := configuracionesHMACAltaContratacionTemporalDesarrollo(
		derivador, ports.DominioAmbitoIdempotenciaSubsanacionReparo, true,
	)
	if err != nil {
		t.Fatal(err)
	}
	huella, _, err := configuracionesHMACAltaContratacionTemporalDesarrollo(
		derivador, ports.DominioHuellaPeticionSubsanacionReparo, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	sellos, err := seguridadcontratacion.NuevaAutoridadSellosSubsanacionReparoHMAC(ambito, nil, huella, nil)
	if err != nil {
		t.Fatal(err)
	}
	return sellos
}

// El perfil forma parte de ambos sellos. Una subsanación confirmada con el
// perfil anterior no puede reconciliarse como replay del perfil fijo nuevo.
func TestSubsanacionHistoricaCambiaSellosAlCambiarPerfil(t *testing.T) {
	sellos := sellosSubsanacionTransicionPrueba(t)
	material := ports.MaterialSubsanacionReparo{
		OrganizacionRef: "organizacion:ct:prueba", ExpedienteRef: "expediente:ct:prueba",
		ActorRef: "actor:ct:prueba", PerfilRef: "perfil:ct:anterior",
		VersionEsperada: 6, ClaveIdempotencia: "11111111-2222-4333-8444-555555555555",
		Observaciones: "Corrección sintética.",
	}
	derivar := func(m ports.MaterialSubsanacionReparo) (ports.ColeccionSellosHMAC, ports.ColeccionSellosHMAC) {
		t.Helper()
		ctx := context.Background()
		a, err := sellos.SellarAmbitoSubsanacionReparo(ctx, ports.SolicitudSellarAmbitoIdempotencia{
			ClaveIdempotencia: m.ClaveIdempotencia, OrganizacionRef: m.OrganizacionRef,
			ActorRef: m.ActorRef, PerfilRef: m.PerfilRef,
		})
		if err != nil {
			t.Fatal(err)
		}
		h, err := sellos.DerivarHuellaSubsanacionReparo(ctx, m)
		if err != nil {
			t.Fatal(err)
		}
		return a, h
	}
	anteriorAmbito, anteriorHuella := derivar(material)
	if anteriorAmbito.ValidarDominio(ports.DominioAmbitoIdempotenciaSubsanacionReparo) != nil ||
		anteriorHuella.ValidarDominio(ports.DominioHuellaPeticionSubsanacionReparo) != nil {
		t.Fatal("sellos históricos inválidos")
	}
	perfilNuevo := material
	perfilNuevo.PerfilRef = "perfil:ct:fijo"
	nuevoAmbito, nuevaHuella := derivar(perfilNuevo)
	antesA, err := anteriorAmbito.Datos()
	if err != nil {
		t.Fatal(err)
	}
	antesH, err := anteriorHuella.Datos()
	if err != nil {
		t.Fatal(err)
	}
	despuesA, err := nuevoAmbito.Datos()
	if err != nil {
		t.Fatal(err)
	}
	despuesH, err := nuevaHuella.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if antesA.Activo.Valor == despuesA.Activo.Valor || antesH.Activo.Valor == despuesH.Activo.Valor ||
		ports.ColeccionesHMACContienenPar(nuevoAmbito, ports.DominioAmbitoIdempotenciaSubsanacionReparo,
			nuevaHuella, ports.DominioHuellaPeticionSubsanacionReparo, antesA.Activo.Valor, antesH.Activo.Valor) {
		t.Fatal("el perfil nuevo pudo reconciliar los sellos de la subsanación histórica")
	}
}

type contextoSubsanacionHistoricaPrueba struct {
	valor ports.ContextoAutorizacionAltaV3
}

func (c contextoSubsanacionHistoricaPrueba) ResolverContextoAutorizacionAltaV3(context.Context, ports.SolicitudResolverContextoAutorizacionAltaV3) (ports.ContextoAutorizacionAltaV3, error) {
	return c.valor, nil
}

type preparadorSubsanacionHistoricaPrueba struct {
	ambitoAnterior, huellaAnterior string
	llamadas                       int
}

func (p *preparadorSubsanacionHistoricaPrueba) PrepararSubsanacionReparo(_ context.Context, s ports.SolicitudPrepararSubsanacionReparo) (ports.PreparacionSubsanacionReparo, error) {
	p.llamadas++
	if ports.ColeccionesHMACContienenPar(s.AmbitosHMAC, ports.DominioAmbitoIdempotenciaSubsanacionReparo,
		s.HuellasPeticionHMAC, ports.DominioHuellaPeticionSubsanacionReparo, p.ambitoAnterior, p.huellaAnterior) {
		return ports.PreparacionSubsanacionReparo{}, errors.New("la petición nueva conservó los sellos históricos")
	}
	return ports.PreparacionSubsanacionReparo{}, ports.ErrClaveIdempotenciaUsada
}

type confirmadorSubsanacionHistoricaPrueba struct{ llamadas int }

func (c *confirmadorSubsanacionHistoricaPrueba) ConfirmarSubsanacionReparo(context.Context, ports.OrdenConfirmarSubsanacionReparo) (ports.ReciboSubsanacionReparo, error) {
	c.llamadas++
	return ports.ReciboSubsanacionReparo{}, errors.New("segunda escritura prohibida")
}

// El servicio corta antes de confirmar al recibir el conflicto del preparador
// de solo lectura. La clasificación real de PostgreSQL exige ensayo separado.
func TestSubsanacionHistoricaEnConflictoNoLlamaConfirmador(t *testing.T) {
	soporte, autorizador, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	vinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	sellos := sellosSubsanacionTransicionPrueba(t)
	solicitud := application.SolicitudRegistrarSubsanacionReparo{
		AutenticacionRef: vinculo.AutenticacionRef, SesionRef: vinculo.SesionRef,
		PerfilRef: vinculo.PerfilActivoRef, OrganizacionRef: "organizacion:ct:prueba",
		ExpedienteRef: "expediente:ct:prueba", VersionEsperada: 6,
		ClaveIdempotencia: "11111111-2222-4333-8444-555555555555",
		Observaciones:     "Corrección sintética.",
	}
	anterior := ports.MaterialSubsanacionReparo{
		OrganizacionRef: solicitud.OrganizacionRef, ExpedienteRef: solicitud.ExpedienteRef,
		ActorRef: vinculo.PrincipalID, PerfilRef: "perfil:ct:anterior",
		VersionEsperada: solicitud.VersionEsperada, ClaveIdempotencia: solicitud.ClaveIdempotencia,
		Observaciones: solicitud.Observaciones,
	}
	ambito, err := sellos.SellarAmbitoSubsanacionReparo(context.Background(), ports.SolicitudSellarAmbitoIdempotencia{
		ClaveIdempotencia: anterior.ClaveIdempotencia, OrganizacionRef: anterior.OrganizacionRef,
		ActorRef: anterior.ActorRef, PerfilRef: anterior.PerfilRef,
	})
	if err != nil {
		t.Fatal(err)
	}
	huella, err := sellos.DerivarHuellaSubsanacionReparo(context.Background(), anterior)
	if err != nil {
		t.Fatal(err)
	}
	datosAmbito, err := ambito.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosHuella, err := huella.Datos()
	if err != nil {
		t.Fatal(err)
	}
	preparador := &preparadorSubsanacionHistoricaPrueba{ambitoAnterior: datosAmbito.Activo.Valor, huellaAnterior: datosHuella.Activo.Valor}
	confirmador := &confirmadorSubsanacionHistoricaPrueba{}
	servicio, err := application.NuevoServicioSubsanacionReparos(
		contextoSubsanacionHistoricaPrueba{soporte.contexto}, sellos, sellos,
		preparador, confirmador, fuentePoliticaSubsanacionReparosDesarrollo{},
		seguridadvec.GeneradorReferenciasCriptograficas{}, autorizador.autorizador, soporte.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := servicio.RegistrarSubsanacionReparo(context.Background(), solicitud)
	if !errors.Is(err, ports.ErrClaveIdempotenciaUsada) || recibo != (ports.ReciboSubsanacionReparo{}) ||
		preparador.llamadas != 1 || confirmador.llamadas != 0 {
		t.Fatalf("conflicto generó efecto o recibo: err=%v recibo=%#v preparación=%d confirmación=%d",
			err, recibo, preparador.llamadas, confirmador.llamadas)
	}
}

func TestPoliticaSubsanacionExigeMotivoCompatibleConPublicacion(t *testing.T) {
	politica := configuracionPoliticaSubsanacionReparosDesarrollo{
		DefinicionRef: "politica:subsanacion:sintetica", DefinicionVersion: 1,
		DefinicionHuellaSHA256: strings.Repeat("a", 64),
		MotivoAutorizacion: vecdomain.ReferenciaEntradaCatalogo{
			CatalogoID: "motivos_subsanacion_prueba", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("b", 64),
			EntradaClave:         "motivo_" + strings.Repeat("c", 32),
		},
	}
	for _, caso := range []struct {
		nombre, clave string
		valida        bool
	}{
		{"motivo publicable", politica.MotivoAutorizacion.EntradaClave, true},
		{"clave generica no publicable", "subsanacion_sintetica", false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			candidata := politica
			candidata.MotivoAutorizacion.EntradaClave = caso.clave
			if err := candidata.MotivoAutorizacion.Validar(); err != nil {
				t.Fatal(err)
			}
			contenido, err := json.Marshal(candidata)
			if err != nil {
				t.Fatal(err)
			}
			ruta := filepath.Join(t.TempDir(), "politica.json")
			if err := os.WriteFile(ruta, contenido, 0600); err != nil {
				t.Fatal(err)
			}
			recibida, err := cargarConfiguracionPoliticaSubsanacionReparosDesarrollo(config.Config{ContratacionTemporalSubsanacionPoliticaFile: ruta})
			if (err == nil) != caso.valida {
				t.Fatalf("valida=%v: %v", caso.valida, err)
			}
			if caso.valida && recibida != candidata {
				t.Fatal("politica alterada durante carga")
			}
		})
	}
}
