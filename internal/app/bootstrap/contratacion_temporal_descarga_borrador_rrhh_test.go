package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestRutaDetalleSoloAdmiteConsultaODescargaExactas(t *testing.T) {
	s := ports.SolicitudDescargaBorradorRRHH{
		ExpedienteRef: "expediente:ct:" + strings.Repeat("a", 64), VersionExpediente: 7, Tipo: ports.BorradorResolucion,
		Formato: ports.FormatoBorradorRRHHPDF, DocumentoSHA256: strings.Repeat("b", 64), TamanoBytes: 10,
		ConsultaHuellaSHA256: strings.Repeat("c", 64),
	}
	recurso, err := ports.RecursoDescargaBorradorRRHH(s, ports.AlcanceDescargaBorradorRRHH{
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, ClaseAmbito: ports.AmbitoOrganizacionRRHH,
		AmbitoRef: organizacionAltaContratacionTemporalDesarrollo})
	if err != nil {
		t.Fatal(err)
	}
	descarga := dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: ports.AccionDescargarBorradorRRHH,
		Finalidad: ports.FinalidadDescargarBorradorRRHH, Recurso: recurso}
	if !solicitudDetalleODescargaRRHHValida(descarga) {
		t.Fatal("descarga exacta rechazada")
	}
	for nombre, cambiar := range map[string]func(*dominiovec.DatosSolicitudAutorizacionLigadaV3){
		"acción de registro": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Accion = ports.AccionCrearSolicitud },
		"descarga con dominio consulta": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos = map[string]string{"consulta_dominio": ports.DominioHuellaConsultaDetalleRRHH, "consulta_huella_sha256": strings.Repeat("d", 64)}
		},
		"consulta con dominio descarga": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Accion = ports.AccionConsultarDetalleRRHH },
		"otra finalidad": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Finalidad = ports.FinalidadConsultarCuadroRRHH
		},
		"otro tipo de recurso": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.Tipo = ports.TipoRecursoCuadroRRHH },
	} {
		d := descarga
		d.Recurso.Atributos = map[string]string{}
		for k, v := range descarga.Recurso.Atributos {
			d.Recurso.Atributos[k] = v
		}
		cambiar(&d)
		if solicitudDetalleODescargaRRHHValida(d) {
			t.Fatalf("%s admitida en la ruta de detalle", nombre)
		}
	}
}

func TestDescargaBorradorClasificaDenegacionYFallo(t *testing.T) {
	for _, err := range []error{ports.ErrAutorizacionDenegada, dominiovec.ErrAutorizacionDenegada,
		puertosvec.ErrDenegacionExplicitaAutorizacionLigadaV3, application.ErrConsultaRRHHNoObservable,
		fmt.Errorf("envuelto: %w", ports.ErrAutorizacionDenegada)} {
		if !errorDenegacionDescargaBorrador(err) {
			t.Fatalf("denegación no reconocida: %v", err)
		}
	}
	for _, err := range []error{ports.ErrDescargaBorradorRRHHNoDisponible, ports.ErrDescargaBorradorRRHHVersionAusente,
		application.ErrConsultaRRHHNoDisponible, context.Canceled, errors.New("render")} {
		if errorDenegacionDescargaBorrador(err) {
			t.Fatalf("fallo técnico tomado por denegación: %v", err)
		}
	}
	// Sin dependencias no se construye: nunca hay descarga sin consumo.
	if _, err := nuevoRegistradorDescargaBorradorRRHHDesarrollo(nil, nil, nil, dominiovec.ReferenciaEntradaCatalogo{}, nil, ""); err == nil {
		t.Fatal("registrador sin dependencias admitido")
	}
	var r *registradorDescargaBorradorRRHHDesarrollo
	if _, err := r.RegistrarDescarga(context.Background(), ports.SolicitudDescargaBorradorRRHH{}); err == nil {
		t.Fatal("registro nulo admitido")
	}
	if err := r.RegistrarFalloDescarga(context.Background(), "expediente:ct:1", errors.New("x")); !errors.Is(err, ports.ErrDescargaBorradorRRHHSinAnotar) {
		t.Fatalf("fallo sin registrador debería no anotar nada: %v", err)
	}
}

type registradorIntentosDescargaPrueba struct {
	ordenes []puertosvec.DatosOrdenIntentoAuditoria
}

func (r *registradorIntentosDescargaPrueba) AppendIntentoAuditoria(ctx context.Context, orden puertosvec.OrdenIntentoAuditoria) (puertosvec.AcuseIntentoAuditoria, error) {
	if ctx.Err() != nil {
		return puertosvec.AcuseIntentoAuditoria{}, errors.New("registro cancelado")
	}
	d, err := orden.Datos()
	if err != nil {
		return puertosvec.AcuseIntentoAuditoria{}, err
	}
	r.ordenes = append(r.ordenes, d)
	return puertosvec.AcuseIntentoAuditoria{AuditoriaRef: "aud_v3_i_" + strings.Repeat("1", 32), Secuencia: 1,
		HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: d.Datos.CorrelacionRef, RegistradaEn: time.Now().UTC()}, nil
}

type repoDescargaNoUsadoPrueba struct{}

func (repoDescargaNoUsadoPrueba) RegistrarDescargaBorrador(context.Context, ports.AlcanceDescargaBorradorRRHH, ports.SolicitudDescargaBorradorRRHH, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboDescargaBorradorRRHH, error) {
	return ports.ReciboDescargaBorradorRRHH{}, errors.New("no debe llegar al repositorio")
}

// Una petición cancelada o con el plazo agotado no borra el intento: el actor
// sale de la misma petición y el registro no depende de su cancelación.
func TestDescargaBorradorCanceladaQuedaAnotada(t *testing.T) {
	alta, autoridad, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	base, err := puertosvec.ConCorrelacionIncidenciasPeticion(contextoRutaConsultasRRHHDesarrolloPrueba(alta.soporte, principal, httpinterno.RutaConsultaDetalleRRHH))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := autoridad.ResolverContextoConsultaRRHH(base); err != nil {
		t.Fatalf("contexto de la consulta: %v", err)
	}
	registrador := &registradorIntentosDescargaPrueba{}
	r, err := nuevoRegistradorDescargaBorradorRRHHDesarrollo(autoridad, &proveedorMaterialAltaContratacionTemporalDesarrollo{},
		repoDescargaNoUsadoPrueba{}, alta.soporte.motivoDetalleRRHH, registrador, "vec-rrhh")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(base)
	cancelar()
	s := ports.SolicitudDescargaBorradorRRHH{
		ExpedienteRef: "expediente:ct:" + strings.Repeat("a", 64), VersionExpediente: 7, Tipo: ports.BorradorResolucion,
		Formato: ports.FormatoBorradorRRHHPDF, DocumentoSHA256: strings.Repeat("b", 64), TamanoBytes: 10,
		ConsultaHuellaSHA256: strings.Repeat("c", 64),
	}
	_, err = r.RegistrarDescarga(ctx, s)
	var auditado ports.FalloLecturaAuditado
	if !errors.As(err, &auditado) || len(registrador.ordenes) != 1 ||
		registrador.ordenes[0].Datos.Accion != ports.AccionDescargarBorradorRRHH ||
		registrador.ordenes[0].Datos.Resultado != dominiovec.ResultadoIntentoAuditoriaError {
		t.Fatalf("descarga cancelada sin anotar: err=%v ordenes=%+v", err, registrador.ordenes)
	}
	if registrador.ordenes[0].Datos.FinalidadRef != ports.FinalidadDescargarBorradorRRHH {
		t.Fatalf("finalidad del intento: %s", registrador.ordenes[0].Datos.FinalidadRef)
	}
	// Un fallo previo con la petición cancelada también se anota, como error.
	if err := r.RegistrarFalloDescarga(ctx, s.ExpedienteRef, application.ErrConsultaRRHHNoObservable); !errors.As(err, &auditado) ||
		len(registrador.ordenes) != 2 || registrador.ordenes[1].Datos.Resultado != dominiovec.ResultadoIntentoAuditoriaError {
		t.Fatalf("fallo previo cancelado sin anotar: err=%v", err)
	}
	// Sin cancelación, la consulta no observable queda como denegación.
	if err := r.RegistrarFalloDescarga(base, s.ExpedienteRef, application.ErrConsultaRRHHNoObservable); !errors.As(err, &auditado) ||
		len(registrador.ordenes) != 3 || registrador.ordenes[2].Datos.Resultado != dominiovec.ResultadoIntentoAuditoriaDenegado {
		t.Fatalf("denegación sin anotar: err=%v", err)
	}
}
