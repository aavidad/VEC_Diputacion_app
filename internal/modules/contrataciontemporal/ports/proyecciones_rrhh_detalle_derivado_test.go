package ports_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func entradaDetalleDerivadoPrueba(t *testing.T, conPropuesta ...bool) (ports.EntradaDetalleExpedienteRRHHMinimizada, ports.ReciboLecturaRRHH) {
	t.Helper()
	d := datosDetalleMinimizadoPrueba(3)
	informeEn := d.hitos[3].RealizadaEn.Add(time.Minute)
	d.hitos = append(d.hitos,
		ports.HitoExpedienteRRHH{Secuencia: 5, VersionExpediente: 5,
			AccionClave: domain.AccionEmitirInformeJuridico, RealizadaEn: informeEn,
			FaseOrigen: d.hitos[3].FaseDestino, FaseDestino: domain.FaseInformeJuridico,
			EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso},
		ports.HitoExpedienteRRHH{Secuencia: 6, VersionExpediente: 6,
			AccionClave: domain.AccionRegistrarFiscalizacion, RealizadaEn: informeEn.Add(time.Minute),
			FaseOrigen: domain.FaseInformeJuridico, FaseDestino: domain.FaseFiscalizacion,
			EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso},
	)
	padreHitos := append([]ports.HitoExpedienteRRHH(nil), d.hitos...)
	padreRef, numeroPadre, creadoPadre := d.resumen.ExpedienteRef, d.resumen.NumeroVisible, d.resumen.CreadoEn
	creadoHijo := d.hitos[5].RealizadaEn.Add(time.Minute)
	d.resumen.ExpedienteRef, d.resumen.NumeroVisible = "expediente:rrhh:hijo:01", numeroPadre+"-P1"
	d.resumen.Version, d.resumen.CreadoEn, d.resumen.ActualizadoEn = 6, creadoHijo, creadoHijo
	d.resumen.FaseClave, d.resumen.EstadoClave = domain.FaseFiscalizacion, domain.EstadoEnCurso
	d.hitos = []ports.HitoExpedienteRRHH{{Secuencia: 6, VersionExpediente: 6,
		AccionClave: domain.AccionDerivarNombramiento, RealizadaEn: creadoHijo,
		FaseOrigen: domain.FaseFiscalizacion, FaseDestino: domain.FaseFiscalizacion,
		EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso}}
	var propuesta *ports.PropuestaFormalizacionRRHH
	if len(conPropuesta) == 1 && conPropuesta[0] {
		confirmadaEn := creadoHijo.Add(time.Minute)
		d.hitos = append(d.hitos, ports.HitoExpedienteRRHH{
			Secuencia: 7, VersionExpediente: 7, AccionClave: "registrar_propuesta_formalizacion",
			RealizadaEn: confirmadaEn, FaseOrigen: domain.FaseFiscalizacion,
			FaseDestino: domain.FaseNombramiento, EstadoOrigen: domain.EstadoEnCurso,
			EstadoDestino: domain.EstadoEnCurso,
		})
		d.resumen.Version, d.resumen.ActualizadoEn = 7, confirmadaEn
		d.resumen.FaseClave = domain.FaseNombramiento
		propuesta = &ports.PropuestaFormalizacionRRHH{
			PropuestaRef: "propuesta:ct:01", ReciboRef: "recibo:propuesta:01",
			Version: 7, ConfirmadaEn: confirmadaEn,
		}
	}
	refFiscal, _ := ports.NuevaReferenciaHitoFiscalizacionRRHH(6)
	entrada, err := ports.NuevaEntradaDetalleExpedienteRRHHMinimizadaDerivadaV1(
		d.resumen, d.solicitud, d.analisis, referenciaAnalisisMinimizadaPrueba(t, 2),
		d.cobertura, referenciaCoberturaMinimizadaPrueba(t, 3),
		d.asignacion, referenciaAsignacionMinimizadaPrueba(t, 4),
		&ports.FiscalizacionOperativaRRHH{ResultadoClave: domain.FiscalizacionFavorable,
			RegistradaEn: padreHitos[5].RealizadaEn},
		refFiscal, ports.ReferenciaHitoSubsanacionFiscalizacionRRHH{}, d.hitos,
		ports.InstruccionCompartidaRRHH{
			DerivacionRef: "derivacion:ct:01", PadreRef: padreRef, NumeroVisiblePadre: numeroPadre,
			OfertaRef: "oferta:ct:01", NumeroPlaza: 1, ReciboDerivacionRef: "recibo:ct:01",
			HuellaPadreSHA256: strings.Repeat("a", 64), CreadoEnPadre: creadoPadre,
			ActualizadoEnPadre: padreHitos[5].RealizadaEn, HitosPadre: padreHitos,
		}, propuesta,
	)
	if err != nil {
		t.Fatal(err)
	}
	return entrada, reciboDetalleMinimizadoPrueba(t, d.resumen.ExpedienteRef, d.resumen.Version, d.resumen.ActualizadoEn)
}

func TestDetalleDerivadoConservaDosCronologiasYCanonPropio(t *testing.T) {
	entrada, lectura := entradaDetalleDerivadoPrueba(t)
	detalle, err := ports.NuevoDetalleExpedienteRRHHMinimizado(entrada, lectura)
	if err != nil {
		t.Fatal(err)
	}
	if len(detalle.Hitos) != 1 || detalle.Hitos[0].Secuencia != 6 ||
		detalle.InstruccionCompartida == nil || len(detalle.InstruccionCompartida.HitosPadre) != 6 {
		t.Fatal("se mezclaron los actos propios y los originales")
	}
	canon, err := entrada.ExportarContenidoCanonicoParaSQL(detalle.Resumen.ActualizadoEn.Add(time.Minute))
	if err != nil || !strings.HasPrefix(string(canon.BytesCanonicos()), "VEC-CT-CONTENIDO-DETALLE-RRHH-V5\n") {
		t.Fatalf("canon derivado: %v", err)
	}
	huellaV5 := sha256.Sum256(canon.BytesCanonicos())
	if hex.EncodeToString(huellaV5[:]) != "7a1018948c4d1ebdb1bebf572855964d65994e0f05871e7c04ed89ee9bfa04fc" {
		t.Fatal("el canon V5 previo cambió")
	}
	segundo, err := detalle.ExportarContenidoCanonicoParaSQL(detalle.Resumen.ActualizadoEn.Add(time.Minute))
	if err != nil || segundo.HuellaSHA256() != canon.HuellaSHA256() {
		t.Fatalf("canon no reproducible: %v", err)
	}
}

func TestDetalleDerivadoPropuestaV6YContinuacion(t *testing.T) {
	entrada, lectura := entradaDetalleDerivadoPrueba(t, true)
	detalle, err := ports.NuevoDetalleExpedienteRRHHMinimizado(entrada, lectura)
	if err != nil || detalle.PropuestaFormalizacion == nil {
		t.Fatalf("propuesta confirmada perdida: %v", err)
	}
	canon, err := entrada.ExportarContenidoCanonicoParaSQL(detalle.Resumen.ActualizadoEn.Add(time.Minute))
	if err != nil || !strings.HasPrefix(string(canon.BytesCanonicos()), "VEC-CT-CONTENIDO-DETALLE-RRHH-V6\n") {
		t.Fatalf("canon V6: %v", err)
	}
	desdeDetalle, err := detalle.ExportarContenidoCanonicoParaSQL(detalle.Resumen.ActualizadoEn.Add(time.Minute))
	if err != nil || desdeDetalle.HuellaSHA256() != canon.HuellaSHA256() {
		t.Fatalf("V6 no reproduce su canon: %v", err)
	}
	solicitud, _ := ports.NuevaSolicitudDetalleRRHH(detalle.Resumen.ExpedienteRef, 7)
	casos := map[string]func(*ports.DetalleExpedienteRRHH){
		"referencia": func(d *ports.DetalleExpedienteRRHH) { d.PropuestaFormalizacion.PropuestaRef = "" },
		"recibo":     func(d *ports.DetalleExpedienteRRHH) { d.PropuestaFormalizacion.ReciboRef = "" },
		"versión":    func(d *ports.DetalleExpedienteRRHH) { d.PropuestaFormalizacion.Version = 8 },
		"fecha": func(d *ports.DetalleExpedienteRRHH) {
			d.PropuestaFormalizacion.ConfirmadaEn = d.PropuestaFormalizacion.ConfirmadaEn.Add(time.Second)
		},
		"hito":           func(d *ports.DetalleExpedienteRRHH) { d.Hitos[1].AccionClave = "accion.ajena" },
		"bloque ausente": func(d *ports.DetalleExpedienteRRHH) { d.PropuestaFormalizacion = nil },
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			mutado := detalle.Clonar()
			alterar(&mutado)
			if mutado.ValidarContenidoPublicablePara(solicitud) == nil {
				t.Fatal("se admitió propuesta desvinculada")
			}
		})
	}
	posterior := detalle.Clonar()
	posterior.Hitos = append(posterior.Hitos, ports.HitoExpedienteRRHH{
		Secuencia: 8, VersionExpediente: 8, AccionClave: "registrar_resolucion_formalizacion",
		RealizadaEn: posterior.Resumen.ActualizadoEn.Add(time.Minute),
		FaseOrigen:  domain.FaseNombramiento, FaseDestino: domain.FaseNombramiento,
		EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso,
	})
	posterior.Resumen.Version, posterior.Resumen.ActualizadoEn = 8, posterior.Hitos[2].RealizadaEn
	solicitudV8, _ := ports.NuevaSolicitudDetalleRRHH(posterior.Resumen.ExpedienteRef, 8)
	if posterior.ValidarContenidoPublicablePara(solicitudV8) != nil {
		t.Fatal("la propuesta v7 desapareció al avanzar el hijo a v8")
	}
}

func TestDetalleDerivadoRechazaPadreReciboYCronologiaMezclada(t *testing.T) {
	entrada, lectura := entradaDetalleDerivadoPrueba(t)
	nominal, err := ports.NuevoDetalleExpedienteRRHHMinimizado(entrada, lectura)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(nominal.Resumen.ExpedienteRef, nominal.Resumen.Version)
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string]func(*ports.DetalleExpedienteRRHH){
		"hito padre adulterado": func(d *ports.DetalleExpedienteRRHH) {
			d.InstruccionCompartida.HitosPadre[3].FaseDestino = "otra_fase"
		},
		"recibo ausente": func(d *ports.DetalleExpedienteRRHH) {
			d.InstruccionCompartida.ReciboDerivacionRef = ""
		},
		"hito padre como propio": func(d *ports.DetalleExpedienteRRHH) {
			d.Hitos[0] = d.InstruccionCompartida.HitosPadre[5]
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			d := nominal.Clonar()
			alterar(&d)
			if d.ValidarContenidoPublicablePara(solicitud) == nil {
				t.Fatal("se admitió un detalle adulterado")
			}
		})
	}
}
