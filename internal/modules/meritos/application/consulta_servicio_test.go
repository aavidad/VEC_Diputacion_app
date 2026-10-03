package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func escenarioConsulta(t *testing.T) (*ServicioConsultaPropia, SolicitudConsultaPropia, *autoridadConsultaPrueba, *repositorioConsultaPrueba, *auditoriaConsultaPrueba) {
	t.Helper()
	_, original, _, _, _ := escenario(t)
	audit := &auditoriaConsultaPrueba{t: t}
	a, r := &autoridadConsultaPrueba{t: t}, &repositorioConsultaPrueba{}
	s, err := NuevoServicioConsultaPropia(a, r, configuracionAuditConsultaPrueba(audit), relojPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	r.ficha = &ports.FichaHechoPropio{Referencia: original.Hecho.Referencia, Version: 7, Tipo: original.Hecho.Tipo,
		ConceptoRef: original.Hecho.ConceptoRef, Denominacion: original.Hecho.Denominacion,
		Procedencia: original.Hecho.Procedencia, Vigencia: original.Hecho.Vigencia, Estado: domain.Pendiente,
		Evidencias: append([]vec.ReferenciaDocumento{}, original.Hecho.Evidencias...)}
	return s, SolicitudConsultaPropia{Vinculo: original.Vinculo, Contexto: original.Contexto,
		Correlacion: original.Correlacion, Motivo: original.Motivo, HechoRef: original.Hecho.Referencia}, a, r, audit
}

// Estos dobles solo cotejan contratos de aplicación. No verifican criptografía,
// consumo V3 real, SQL, persistencia ni recuperación; no se montan en producto.
type autoridadConsultaPrueba struct {
	t        *testing.T
	llamadas int
	defecto  string
}

func (a *autoridadConsultaPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, solicitud vec.SolicitudAutorizacionLigadaV3, resultado vec.ResultadoContextoActorRegistradoV2) (vec.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	if a.defecto == "no_disponible" {
		return vec.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ports.ErrConsultaNoDisponible
	}
	if a.defecto == "denegada" {
		return vec.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, vec.ErrAutorizacionDenegada
	}
	datos, err := solicitud.Datos()
	if err != nil {
		a.t.Fatal(err)
	}
	d := pruebas.DatosConcesionV3Prueba{Instante: instante, PersonaRef: persona, PerfilRef: perfil, Accion: datos.Accion,
		Recurso: datos.Recurso, Finalidad: datos.Finalidad, Campos: []string{"hecho_actual", "recibo_consulta"}, Obligaciones: []string{"auditar"}, DecisionRef: decisionRef}
	switch a.defecto {
	case "campo_ausente":
		d.Campos = []string{"hecho_actual"}
	case "campo_ajeno":
		d.Campos = append(d.Campos, "persona_ref")
	case "obligacion_ausente":
		d.Obligaciones = nil
	case "obligacion_ajena":
		d.Obligaciones = append(d.Obligaciones, "firmar")
	}
	c, err := pruebas.NuevaConcesionV3Prueba(d)
	if err != nil {
		a.t.Fatal(err)
	}
	dh, _ := vec.HuellaSHA256DecisionAutorizacionV3(c.Decision)
	mh, _ := vec.HuellaSHA256MotivoAutorizacionV2(datos.ReferenciaMotivo)
	rh, _ := datos.Recurso.HuellaContextoAutorizacionSHA256()
	audiencia, expira := AudienciaConsultaPropia, instante.Add(5*time.Second)
	if a.defecto == "audiencia" {
		audiencia = "otra.audiencia.v1"
	}
	if a.defecto == "caducada" {
		expira = instante.Add(1500 * time.Millisecond)
	}
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(decisionRef, dh, mh,
		resultado.RegistroContextoRef, resultado.HuellaSHA256, datos.Accion, datos.Recurso.Referencia, rh,
		audiencia, instante.Add(time.Second), expira)
	if err != nil {
		a.t.Fatal(err)
	}
	dc, _ := vec.RepresentacionCanonicaDecisionAutorizacionV3(c.Decision)
	mc, _ := vec.RepresentacionCanonicaMotivoAutorizacionV2(datos.ReferenciaMotivo)
	privada := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{2}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(privada.Public())
	if err != nil {
		a.t.Fatal(err)
	}
	m, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{'b'}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen,
		dc, mc, resultado.RepresentacionCanonica, resultado.Contexto.Instantanea.PersonaVersion, resultado.Contexto.Instantanea.PerfilVersion,
		[]byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if err != nil {
		a.t.Fatal(err)
	}
	return c.Decision, c.Confirmacion, exportadorPrueba{m}, nil
}

type repositorioConsultaPrueba struct {
	llamadas  int
	ficha     *ports.FichaHechoPropio
	ultima    ports.OrdenConsultaPropia
	resultado ports.ResultadoConsultaPropia
	mutar     func(*ports.ResultadoConsultaPropia)
	err       error
	despues   func()
}

func (r *repositorioConsultaPrueba) ConsultarActual(_ context.Context, o ports.OrdenConsultaPropia) (ports.ResultadoConsultaPropia, error) {
	r.llamadas++
	r.ultima = o
	if r.despues != nil {
		defer r.despues()
	}
	if r.err != nil {
		return ports.ResultadoConsultaPropia{}, r.err
	}
	d, _ := o.Autorizacion.Solicitud.Datos()
	correlacion, _ := d.Correlacion.ValorCanonico()
	out := ports.ResultadoConsultaPropia{Codigo: "obtenida", HechoActual: r.ficha,
		ReciboConsulta: &ports.ReciboConsultaPropia{Referencia: "recibo:consulta:prueba", HechoRef: o.HechoRef,
			DecisionRef: o.Autorizacion.Material.ResumenCapacidad().DecisionRef(), ConsumoHuellaSHA256: strings.Repeat("a", 64),
			AuditoriaRef: "aud_v3_consulta_prueba", CorrelacionRef: correlacion, ConsultadaEn: instante.Add(2 * time.Second)}}
	if r.ficha == nil {
		out.Codigo = "no_encontrada"
	} else {
		out.ReciboConsulta.VersionConsultada = r.ficha.Version
	}
	if r.mutar != nil {
		r.mutar(&out)
	}
	r.resultado = out
	return out, nil
}

func TestConsultaPropiaVersionActualYSelectorAtestado(t *testing.T) {
	s, solicitud, _, r, audit := escenarioConsulta(t)
	out, err := s.ConsultarActual(context.Background(), solicitud)
	if err != nil || out.Codigo != "obtenida" || out.HechoActual.Version != 7 || out.ReciboConsulta.VersionConsultada != 7 ||
		out.ReciboConsulta.AuditoriaRef != "aud_v3_consulta_prueba" {
		t.Fatalf("consulta actual: %#v, %v", out, err)
	}
	esperado := `{"esquema":"vec.meritos.hecho.consulta_propia.v1","hecho_ref":"hecho:prueba","persona_ref":"per_0123456789abcdefghijkl"}`
	if string(r.ultima.SelectorCanonico) != esperado || r.ultima.PersonaRef != solicitud.Contexto.Contexto.PersonaRef || audit.llamadas != 0 {
		t.Fatal("selector no atestado o auditoría de éxito fuera de transacción")
	}
	d, _ := r.ultima.Autorizacion.Solicitud.Datos()
	if len(d.Recurso.Ambitos) != 2 || d.Recurso.Ambitos["huella_consulta_sha256"] != huellaConsulta([]byte(esperado)) {
		t.Fatal("consulta no ligada al selector exacto")
	}
	r.ficha.Version = 8
	out, err = s.ConsultarActual(context.Background(), solicitud)
	if err != nil || out.HechoActual.Version != 8 || out.ReciboConsulta.VersionConsultada != 8 {
		t.Fatal("la consulta debe obtener la versión actual sin CAS de negocio", err)
	}
}

func TestConsultaPropiaAusenciaConRecibo(t *testing.T) {
	s, solicitud, _, r, audit := escenarioConsulta(t)
	r.ficha = nil
	out, err := s.ConsultarActual(context.Background(), solicitud)
	if err != nil || out.Codigo != "no_encontrada" || out.HechoActual != nil || out.ReciboConsulta == nil || out.ReciboConsulta.VersionConsultada != 0 || audit.llamadas != 0 {
		t.Fatalf("ausencia sin recibo nominal: %#v, %v", out, err)
	}
}

func TestConsultaPropiaEvidenciasVaciasSonArray(t *testing.T) {
	s, solicitud, _, r, _ := escenarioConsulta(t)
	r.ficha.Evidencias = nil
	out, err := s.ConsultarActual(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(out)
	if err != nil || !bytes.Contains(b, []byte(`"evidencias":[]`)) {
		t.Fatal("las evidencias vacías deben serializarse como array", err)
	}
}

func TestConsultaPropiaNoConsultaSinConcesionExacta(t *testing.T) {
	for _, defecto := range []string{"denegada", "no_disponible", "campo_ausente", "campo_ajeno", "obligacion_ausente", "obligacion_ajena", "audiencia", "caducada"} {
		t.Run(defecto, func(t *testing.T) {
			s, solicitud, a, r, audit := escenarioConsulta(t)
			a.defecto = defecto
			out, err := s.ConsultarActual(context.Background(), solicitud)
			if err == nil || out.HechoActual != nil || out.ReciboConsulta != nil || r.llamadas != 0 || audit.llamadas != 1 {
				t.Fatal("autoridad no exacta alcanza el repositorio o devuelve datos", err)
			}
		})
	}
}

func TestConsultaPropiaVinculoAjenoNoLlegaAEmisor(t *testing.T) {
	s, solicitud, a, r, audit := escenarioConsulta(t)
	_, ajeno, err := pruebas.NuevoContextoRegistradoYVinculoV2(instante, "per_abcdefghijkl0123456789", perfil, vec.AuthMethodCertificate, vec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	solicitud.Vinculo = ajeno
	out, err := s.ConsultarActual(context.Background(), solicitud)
	if !errors.Is(err, vec.ErrAutorizacionDenegada) || out.HechoActual != nil || out.ReciboConsulta != nil || a.llamadas != 0 || r.llamadas != 0 || audit.llamadas != 0 {
		t.Fatal("vínculo ajeno aceptado o identidad libre auditada", err)
	}
}

func TestConsultaPropiaRechazaResultadoManipulado(t *testing.T) {
	casos := map[string]func(*ports.ResultadoConsultaPropia){
		"hecho_ajeno": func(r *ports.ResultadoConsultaPropia) { r.HechoActual.Referencia = "hecho:ajeno" },
		"version":     func(r *ports.ResultadoConsultaPropia) { r.ReciboConsulta.VersionConsultada++ },
		"decision":    func(r *ports.ResultadoConsultaPropia) { r.ReciboConsulta.DecisionRef = "decision:ajena" },
		"correlacion": func(r *ports.ResultadoConsultaPropia) { r.ReciboConsulta.CorrelacionRef = "correlacion:ajena" },
		"consumo":     func(r *ports.ResultadoConsultaPropia) { r.ReciboConsulta.ConsumoHuellaSHA256 = "a" },
		"sin_recibo":  func(r *ports.ResultadoConsultaPropia) { r.ReciboConsulta = nil },
		"auditoria":   func(r *ports.ResultadoConsultaPropia) { r.ReciboConsulta.AuditoriaRef = "" },
		"estado":      func(r *ports.ResultadoConsultaPropia) { r.HechoActual.Estado = "desconocido" },
		"evidencia_repetida": func(r *ports.ResultadoConsultaPropia) {
			r.HechoActual.Evidencias = append(r.HechoActual.Evidencias, r.HechoActual.Evidencias[0])
		},
		"acreditado_sin_revision": func(r *ports.ResultadoConsultaPropia) { r.HechoActual.Estado = domain.Acreditado },
		"instante_anterior":       func(r *ports.ResultadoConsultaPropia) { r.ReciboConsulta.ConsultadaEn = instante },
		"instante_caducado":       func(r *ports.ResultadoConsultaPropia) { r.ReciboConsulta.ConsultadaEn = instante.Add(time.Minute) },
		"instante_no_utc": func(r *ports.ResultadoConsultaPropia) {
			r.ReciboConsulta.ConsultadaEn = r.ReciboConsulta.ConsultadaEn.In(time.FixedZone("desplazada", 3600))
		},
		"codigo_ajeno":       func(r *ports.ResultadoConsultaPropia) { r.Codigo = "confirmada" },
		"denegada_con_datos": func(r *ports.ResultadoConsultaPropia) { r.Codigo = "denegada" },
		"ausencia_con_ficha": func(r *ports.ResultadoConsultaPropia) { r.Codigo = "no_encontrada" },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			s, solicitud, _, r, audit := escenarioConsulta(t)
			r.mutar = mutar
			out, err := s.ConsultarActual(context.Background(), solicitud)
			if !errors.Is(err, ports.ErrConsultaNoDisponible) || out.HechoActual != nil || out.ReciboConsulta != nil || audit.llamadas != 1 || audit.ultima.Datos.Resultado != vec.ResultadoIntentoAuditoriaError {
				t.Fatal("resultado manipulado aceptado", err)
			}
		})
	}
}

func TestConsultaPropiaMinimizaYCopiaProyeccion(t *testing.T) {
	s, solicitud, _, r, _ := escenarioConsulta(t)
	horas := 20
	r.ficha.Horas = &horas
	r.ficha.Estado = domain.Rechazado
	r.ficha.Revision = &ports.RevisionConsultaPropia{Referencia: "revision:prueba", MotivoRef: "motivo:prueba", Fecha: instante.Format(time.RFC3339Nano)}
	out, err := s.ConsultarActual(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, prohibido := range []string{"persona_ref", "actor_ref", "declarante_ref", "empleado_ref"} {
		if bytes.Contains(b, []byte(prohibido)) {
			t.Fatal("proyección expone identidad", prohibido)
		}
	}
	*out.HechoActual.Horas = 999
	out.HechoActual.Evidencias[0].ID = "documento:otro"
	out.HechoActual.Revision.MotivoRef = "motivo:otro"
	out.ReciboConsulta.Referencia = "recibo:otro"
	if *r.ficha.Horas != 20 || r.ficha.Evidencias[0].ID != "documento:prueba" || r.ficha.Revision.MotivoRef != "motivo:prueba" || r.resultado.ReciboConsulta.Referencia != "recibo:consulta:prueba" {
		t.Fatal("respuesta comparte memoria con resultado confiable")
	}
}

func TestConsultaPropiaDenegacionNominalSinDatos(t *testing.T) {
	s, solicitud, _, r, audit := escenarioConsulta(t)
	r.mutar = func(r *ports.ResultadoConsultaPropia) { *r = ports.ResultadoConsultaPropia{Codigo: "denegada"} }
	out, err := s.ConsultarActual(context.Background(), solicitud)
	if !errors.Is(err, vec.ErrAutorizacionDenegada) || out.Codigo != "" || out.HechoActual != nil || out.ReciboConsulta != nil || audit.llamadas != 1 || audit.ultima.Datos.Resultado != vec.ResultadoIntentoAuditoriaDenegado {
		t.Fatal("denegación no cerrada", err)
	}
}

func TestConsultaPropiaRechazoTrasEmisionNoDevuelveFichaNiRecibo(t *testing.T) {
	s, solicitud, autoridad, repositorio, audit := escenarioConsulta(t)
	repositorio.err = vec.ErrAutorizacionDenegada
	retorno := false
	repositorio.despues = func() { retorno = true }
	audit.observar = func(_ context.Context, _ vecports.DatosOrdenIntentoAuditoria) {
		if !retorno {
			t.Fatal("la auditoría empezó antes de cerrar el repositorio")
		}
	}
	out, err := s.ConsultarActual(context.Background(), solicitud)
	if !errors.Is(err, vec.ErrAutorizacionDenegada) || out.Codigo != "" ||
		out.HechoActual != nil || out.ReciboConsulta != nil ||
		autoridad.llamadas != 1 || repositorio.llamadas != 1 {
		t.Fatal("rechazo posterior a la emisión expone un resultado", err)
	}
	if repositorio.ultima.Autorizacion.Material.ResumenCapacidad().DecisionRef() != decisionRef {
		t.Fatal("el repositorio no recibió la autorización emitida")
	}
	correlacion, _ := solicitud.Correlacion.ValorCanonico()
	entrada := audit.ultima
	vinculo, vinculoErr := entrada.Vinculo.Datos()
	if audit.llamadas != 1 || entrada.Datos.Resultado != vec.ResultadoIntentoAuditoriaDenegado ||
		entrada.ResultadoContexto.Contexto.PersonaRef != persona || entrada.ResultadoContexto.Contexto.PerfilActivoRef != perfil ||
		entrada.Datos.Accion != AccionConsultaPropia || entrada.Datos.ModuloID != "meritos" ||
		entrada.Datos.FinalidadRef != FinalidadConsultaPropia || entrada.Datos.RecursoRef != solicitud.HechoRef ||
		entrada.Datos.CorrelacionRef != correlacion || entrada.Datos.Motivo != s.auditoria.MotivoDenegacion ||
		entrada.ResultadoContexto.HuellaSHA256 != solicitud.Contexto.HuellaSHA256 ||
		vinculoErr != nil || entrada.Vinculo.ValidarPara(solicitud.Contexto) != nil ||
		entrada.Datos.Proceso != "vec-rum04-ensayo" || entrada.Datos.Canal != string(vinculo.Superficie) {
		t.Fatal("falta constancia minimizada del rechazo nominal")
	}
	audit.err = errors.New("diagnóstico privado de auditoría")
	out, err = s.ConsultarActual(context.Background(), solicitud)
	if !errors.Is(err, ports.ErrConsultaNoDisponible) || out.HechoActual != nil || out.ReciboConsulta != nil || audit.llamadas != 2 {
		t.Fatal("el fallo de auditoría no termina indisponible y vacío", err)
	}
}

// El doble recibe la orden nominal real y devuelve un acuse ligado a ella;
// no convierte la petición ni una concesión en una AuditEntry libre.
type auditoriaConsultaPrueba struct {
	t          *testing.T
	llamadas   int
	ultima     vecports.DatosOrdenIntentoAuditoria
	err        error
	observar   func(context.Context, vecports.DatosOrdenIntentoAuditoria)
	mutarAcuse func(*vecports.AcuseIntentoAuditoria)
	errMotivo  error
}

func configuracionAuditConsultaPrueba(a vecports.RegistradorIntentosAuditoria) ConfiguracionAuditoriaConsulta {
	denegacion := vec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_auditoria", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_" + strings.Repeat("1", 32)}
	fallo := denegacion
	fallo.EntradaClave = "motivo_" + strings.Repeat("2", 32)
	return ConfiguracionAuditoriaConsulta{Registrador: a, Proceso: "vec-rum04-ensayo", Plazo: 2 * time.Second,
		ValidadorMotivos: a.(vecports.ValidadorReferenciaMotivoAutorizacionV2), MotivoDenegacion: denegacion,
		MotivoError: fallo, RecursoConsultaRef: "meritos:consulta_propia"}
}

func (a *auditoriaConsultaPrueba) ValidarReferenciaMotivoAutorizacionV2(_ context.Context, motivo vec.ReferenciaEntradaCatalogo, _ time.Time) error {
	if a.errMotivo != nil {
		return a.errMotivo
	}
	return motivo.Validar()
}

func (a *auditoriaConsultaPrueba) AppendIntentoAuditoria(ctx context.Context, o vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	a.llamadas++
	d, err := o.Datos()
	if err != nil {
		a.t.Fatal("orden de auditoría no nominal")
	}
	a.ultima = d
	if a.observar != nil {
		a.observar(ctx, d)
	}
	if a.err != nil {
		return vecports.AcuseIntentoAuditoria{}, a.err
	}
	acuse := vecports.AcuseIntentoAuditoria{AuditoriaRef: "audit_rum04_prueba", Secuencia: 1,
		HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: d.Datos.CorrelacionRef, RegistradaEn: instante}
	if a.mutarAcuse != nil {
		a.mutarAcuse(&acuse)
	}
	return acuse, nil
}

func TestConsultaPropiaCancelacionPosteriorNoBorraElIntento(t *testing.T) {
	s, solicitud, _, repositorio, _ := escenarioConsulta(t)
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	repositorio.err, repositorio.despues = context.Canceled, cancelar
	llamadas := 0
	s.auditoria = configuracionAuditConsultaPrueba(&auditoriaConsultaPrueba{t: t, observar: func(auditCtx context.Context, entrada vecports.DatosOrdenIntentoAuditoria) {
		llamadas++
		limite, limitado := auditCtx.Deadline()
		if ctx.Err() == nil || auditCtx.Err() != nil || !limitado ||
			time.Until(limite) <= 0 || time.Until(limite) > 2*time.Second ||
			entrada.Datos.Resultado != vec.ResultadoIntentoAuditoriaError || repositorio.llamadas != 1 {
			t.Fatal("intento cancelado sin contexto independiente acotado")
		}
	}})
	out, err := s.ConsultarActual(ctx, solicitud)
	if !errors.Is(err, context.Canceled) || out.HechoActual != nil || out.ReciboConsulta != nil || llamadas != 1 {
		t.Fatal("cancelación posterior borra el intento o expone datos", err)
	}
}

func TestConsultaPropiaRechazaOrdenMutadaAntesDeConsumo(t *testing.T) {
	for _, nombre := range []string{"selector", "persona", "huella", "hecho", "motivo", "contexto"} {
		t.Run(nombre, func(t *testing.T) {
			s, solicitud, _, r, _ := escenarioConsulta(t)
			if _, err := s.ConsultarActual(context.Background(), solicitud); err != nil {
				t.Fatal(err)
			}
			o := r.ultima
			switch nombre {
			case "selector":
				o.SelectorCanonico = append(o.SelectorCanonico, ' ')
			case "persona":
				o.PersonaRef = "persona:ajena"
			case "huella":
				o.HuellaConsultaSHA256 = strings.Repeat("0", 64)
			case "hecho":
				o.HechoRef = "hecho:ajeno"
			case "motivo":
				o.Motivo = vec.ReferenciaEntradaCatalogo{}
			case "contexto":
				o.Autorizacion.Contexto = vec.ResultadoContextoActorRegistradoV2{}
			}
			if ValidarOrdenConsultaPropia(o) == nil {
				t.Fatal("orden manipulada aceptada", nombre)
			}
		})
	}
}

func TestConsultaPropiaErroresYCancelacionNoDevuelvenDatos(t *testing.T) {
	s, solicitud, _, r, audit := escenarioConsulta(t)
	r.err = ports.ErrConsultaNoDisponible
	out, err := s.ConsultarActual(context.Background(), solicitud)
	if err == nil || out.HechoActual != nil || out.ReciboConsulta != nil || audit.llamadas != 1 || audit.ultima.Datos.Resultado != vec.ResultadoIntentoAuditoriaError {
		t.Fatal("fallo de repositorio devuelve datos", err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	llamadas := r.llamadas
	if _, err = s.ConsultarActual(ctx, solicitud); !errors.Is(err, context.Canceled) || r.llamadas != llamadas {
		t.Fatal("cancelación no respetada", err)
	}
	if _, err = NuevoServicioConsultaPropia(nil, r, configuracionAuditConsultaPrueba(audit), relojPrueba{}); !errors.Is(err, ports.ErrConsultaNoDisponible) {
		t.Fatal("dependencia ausente aceptada", err)
	}
	if _, err = NuevoServicioConsultaPropia(s.autorizador, r, ConfiguracionAuditoriaConsulta{}, relojPrueba{}); !errors.Is(err, ports.ErrConsultaNoDisponible) {
		t.Fatal("auditoría ausente aceptada", err)
	}
}

func TestConsultaPropiaAcuseNoLigadoCierraSinDatos(t *testing.T) {
	for _, campo := range []string{"referencia", "secuencia", "huella", "correlacion", "fecha"} {
		t.Run(campo, func(t *testing.T) {
			s, solicitud, _, r, audit := escenarioConsulta(t)
			r.err = vec.ErrAutorizacionDenegada
			audit.mutarAcuse = func(a *vecports.AcuseIntentoAuditoria) {
				switch campo {
				case "referencia":
					a.AuditoriaRef = ""
				case "secuencia":
					a.Secuencia = 0
				case "huella":
					a.HuellaSHA256 = "invalida"
				case "correlacion":
					a.CorrelacionRef = "correlacion_ajena"
				case "fecha":
					a.RegistradaEn = time.Time{}
				}
			}
			out, err := s.ConsultarActual(context.Background(), solicitud)
			if !errors.Is(err, ports.ErrConsultaNoDisponible) || out.HechoActual != nil || out.ReciboConsulta != nil || audit.llamadas != 1 {
				t.Fatal("acuse no ligado acepta denegación o expone datos", err)
			}
		})
	}
}

func TestConsultaPropiaAuditaEmisionFallidaSinConcesion(t *testing.T) {
	for _, defecto := range []string{"denegada", "no_disponible"} {
		t.Run(defecto, func(t *testing.T) {
			s, solicitud, a, r, audit := escenarioConsulta(t)
			a.defecto = defecto
			out, err := s.ConsultarActual(context.Background(), solicitud)
			esperado := vec.ResultadoIntentoAuditoriaError
			if defecto == "denegada" {
				esperado = vec.ResultadoIntentoAuditoriaDenegado
			}
			if err == nil || out.HechoActual != nil || out.ReciboConsulta != nil || r.llamadas != 0 ||
				audit.llamadas != 1 || audit.ultima.Datos.Resultado != esperado ||
				audit.ultima.ResultadoContexto.RegistroContextoRef != solicitud.Contexto.RegistroContextoRef ||
				audit.ultima.Vinculo.ValidarPara(solicitud.Contexto) != nil {
				t.Fatal("emisión fallida sin constancia nominal", err)
			}
		})
	}
}

func TestConsultaPropiaMotivoFallidoNoAfirmaAuditoria(t *testing.T) {
	s, solicitud, _, r, audit := escenarioConsulta(t)
	r.err = vec.ErrAutorizacionDenegada
	audit.errMotivo = errors.New("catalogo no disponible")
	out, err := s.ConsultarActual(context.Background(), solicitud)
	if !errors.Is(err, ports.ErrConsultaNoDisponible) || out.HechoActual != nil || out.ReciboConsulta != nil || audit.llamadas != 0 {
		t.Fatal("registro con motivo no resuelto positivamente", err)
	}
}

func TestConsultaPropiaFalloCompuestoEsErrorTecnico(t *testing.T) {
	for _, fallo := range []error{ports.ErrConsultaNoDisponible, context.Canceled, context.DeadlineExceeded, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible} {
		s, solicitud, _, r, audit := escenarioConsulta(t)
		r.err = errors.Join(vec.ErrAutorizacionDenegada, fallo)
		_, err := s.ConsultarActual(context.Background(), solicitud)
		if DenegacionConsultaReal(err) || audit.llamadas != 1 || audit.ultima.Datos.Resultado != vec.ResultadoIntentoAuditoriaError || audit.ultima.Datos.Motivo != s.auditoria.MotivoError {
			t.Fatal("fallo técnico compuesto declarado denegación", err)
		}
	}
}
