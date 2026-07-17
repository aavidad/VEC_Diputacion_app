package ports

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestConsultaGobernadaRechazaDecisionYAuditoriaCoherentementeDesviadas(t *testing.T) {
	escenario := nuevoEscenarioPuertoAutoridad(t)
	selector := SelectorVersionFuenteAutoridad{FuenteID: escenario.Fuente.ID, Version: 1}
	solicitud, decisionBase, solicitadaEn := solicitudConsultaGobernadaAutoridadPrueba(t, selector)
	auditoriaBase, _ := solicitud.Auditoria()
	motivoCatalogo, _ := solicitud.MotivoCatalogo()
	correlacionEsperada, _ := solicitud.Correlacion()
	recurso, err := RecursoAutorizableConsultaInternaFuenteAutoridad(selector, motivoCatalogo)
	if err != nil {
		t.Fatal(err)
	}

	casos := []struct {
		nombre         string
		mutarDecision  func(*domain.DecisionAutorizacion)
		mutarAuditoria func(*domain.AuditEntry)
	}{
		{
			nombre: "finalidad coherente pero ajena",
			mutarDecision: func(d *domain.DecisionAutorizacion) {
				d.Finalidad = "otra_finalidad_interna"
			},
			mutarAuditoria: func(a *domain.AuditEntry) { a.Purpose = "otra_finalidad_interna" },
		},
		{
			nombre: "correlacion coherente pero distinta de la solicitud opaca",
			mutarDecision: func(d *domain.DecisionAutorizacion) {
				d.CorrelacionRef = correlacionConsultaAutoridadAjenaPrueba
			},
			mutarAuditoria: func(a *domain.AuditEntry) {
				a.CorrelationRef = correlacionConsultaAutoridadAjenaPrueba
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			decision := clonarDecisionAutorizacionCanonica(decisionBase)
			auditoria := clonarAuditoriaFuenteAutoridad(auditoriaBase)
			caso.mutarDecision(&decision)
			caso.mutarAuditoria(&auditoria)
			// Se recalcula deliberadamente para demostrar que no es la huella
			// general de solicitud la que causa este rechazo del puerto.
			ligarDecisionSolicitudConsultaAutoridadPrueba(t, &decision, recurso, motivoCatalogo)
			evidencia, err := NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(decision, solicitadaEn)
			if err != nil {
				t.Fatalf("precondicion de decision coherente: %v", err)
			}
			if _, err := NuevaSolicitudConsultaInternaGobernadaFuenteAutoridad(
				selector, evidencia, auditoria, motivoCatalogo, correlacionEsperada, solicitadaEn,
			); !errors.Is(err, ErrConsultaInternaFuenteAutoridadInvalida) {
				t.Fatalf("desviacion coherente aceptada: %v", err)
			}
		})
	}

	t.Run("motivo valido y auditoria coherente pero ajenos al compromiso", func(t *testing.T) {
		autorizacion, err := solicitud.Autorizacion()
		if err != nil {
			t.Fatal(err)
		}
		motivoCatalogoAjeno := motivoCatalogoConsultaAutoridadPrueba(
			claveMotivoConsultaAutoridadAjenaPrueba,
		)
		auditoria := clonarAuditoriaFuenteAutoridad(auditoriaBase)
		auditoria.Reason = motivoCatalogoAjeno.EntradaClave
		if _, err := NuevaSolicitudConsultaInternaGobernadaFuenteAutoridad(
			selector, autorizacion, auditoria, motivoCatalogoAjeno,
			correlacionEsperada, solicitadaEn,
		); !errors.Is(err, ErrConsultaInternaFuenteAutoridadInvalida) {
			t.Fatalf("motivo ajeno al compromiso aceptado: %v", err)
		}
	})

	t.Run("superficie externa con garantia alta", func(t *testing.T) {
		decision := clonarDecisionAutorizacionCanonica(decisionBase)
		decision.VinculoAutenticacionActor = vinculoConsultaAutoridadSuperficiePrueba(
			t, decision.EmitidaEn, domain.SuperficieAutenticacionExternaPersonalV1,
		)
		ligarDecisionSolicitudConsultaAutoridadPrueba(t, &decision, recurso, motivoCatalogo)
		evidencia, err := NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(decision, solicitadaEn)
		if err != nil {
			t.Fatalf("precondicion de evidencia externa alta: %v", err)
		}
		if _, err := NuevaSolicitudConsultaInternaGobernadaFuenteAutoridad(
			selector, evidencia, auditoriaBase, motivoCatalogo,
			correlacionEsperada, solicitadaEn,
		); !errors.Is(err, ErrConsultaInternaFuenteAutoridadInvalida) {
			t.Fatalf("superficie externa aceptada por el puerto: %v", err)
		}
	})
}

func TestConsultaGobernadaExigeReferenciaCatalogadaExactaSinTextoLibre(t *testing.T) {
	escenario := nuevoEscenarioPuertoAutoridad(t)
	selector := SelectorVersionFuenteAutoridad{FuenteID: escenario.Fuente.ID, Version: 1}
	solicitud, _, solicitadaEn := solicitudConsultaGobernadaAutoridadPrueba(t, selector)
	autorizacion, _ := solicitud.Autorizacion()
	auditoria, _ := solicitud.Auditoria()
	motivoCatalogo, _ := solicitud.MotivoCatalogo()
	correlacion, _ := solicitud.Correlacion()
	recurso, err := RecursoAutorizableConsultaInternaFuenteAutoridad(selector, motivoCatalogo)
	if err != nil || len(recurso.Atributos) != 6 ||
		recurso.Atributos[AtributoMotivoCatalogoIDConsultaAutoridad] != motivoCatalogo.CatalogoID ||
		recurso.Atributos[AtributoMotivoCatalogoVersionConsultaAutoridad] != "3" ||
		recurso.Atributos[AtributoMotivoCatalogoHuellaConsultaAutoridad] != motivoCatalogo.CatalogoHuellaSHA256 ||
		recurso.Atributos[AtributoMotivoEntradaClaveConsultaAutoridad] != motivoCatalogo.EntradaClave {
		t.Fatalf("contexto ABAC de motivo incompleto: %+v, %v", recurso, err)
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	for _, distinta := range []domain.ReferenciaEntradaCatalogo{
		{
			CatalogoID: "motivos_consulta_fuentes_autoridad_alternativo", CatalogoVersion: 3,
			CatalogoHuellaSHA256: motivoCatalogo.CatalogoHuellaSHA256, EntradaClave: motivoCatalogo.EntradaClave,
		},
		{
			CatalogoID: motivoCatalogo.CatalogoID, CatalogoVersion: 4,
			CatalogoHuellaSHA256: motivoCatalogo.CatalogoHuellaSHA256, EntradaClave: motivoCatalogo.EntradaClave,
		},
		{
			CatalogoID: motivoCatalogo.CatalogoID, CatalogoVersion: 3,
			CatalogoHuellaSHA256: strings.Repeat("e", 64), EntradaClave: motivoCatalogo.EntradaClave,
		},
		{
			CatalogoID: motivoCatalogo.CatalogoID, CatalogoVersion: 3,
			CatalogoHuellaSHA256: motivoCatalogo.CatalogoHuellaSHA256,
			EntradaClave:         claveMotivoConsultaAutoridadAjenaPrueba,
		},
	} {
		recursoDistinto, err := RecursoAutorizableConsultaInternaFuenteAutoridad(selector, distinta)
		if err != nil {
			t.Fatalf("referencia alternativa valida rechazada: %v", err)
		}
		huellaDistinta, err := recursoDistinto.HuellaContextoAutorizacionSHA256()
		if err != nil || huellaDistinta == huellaContexto {
			t.Fatalf("coordenada catalogada no comprometida por ABAC: %v", err)
		}
	}
	for _, prohibida := range []struct {
		nombre string
		ref    domain.ReferenciaEntradaCatalogo
	}{
		{"referencia cero", domain.ReferenciaEntradaCatalogo{}},
		{"clave semantica", func() domain.ReferenciaEntradaCatalogo {
			ref := motivoCatalogo
			ref.EntradaClave = "consulta_tecnica"
			return ref
		}()},
		{"prefijo PII canonico antiguo", func() domain.ReferenciaEntradaCatalogo {
			ref := motivoCatalogo
			ref.EntradaClave = "dni_12345678z"
			return ref
		}()},
		{"hexadecimal corto", func() domain.ReferenciaEntradaCatalogo {
			ref := motivoCatalogo
			ref.EntradaClave = "motivo_0123456789abcdef0123456789abcde"
			return ref
		}()},
		{"hexadecimal no canonico", func() domain.ReferenciaEntradaCatalogo {
			ref := motivoCatalogo
			ref.EntradaClave = "motivo_0123456789abcdef0123456789abcdeg"
			return ref
		}()},
		{"version cero", func() domain.ReferenciaEntradaCatalogo {
			ref := motivoCatalogo
			ref.CatalogoVersion = 0
			return ref
		}()},
		{"huella distinta", func() domain.ReferenciaEntradaCatalogo {
			ref := motivoCatalogo
			ref.CatalogoHuellaSHA256 = strings.Repeat("e", 64)
			return ref
		}()},
		{"texto libre y PII", func() domain.ReferenciaEntradaCatalogo {
			ref := motivoCatalogo
			ref.EntradaClave = "DNI 12345678A de la persona"
			return ref
		}()},
	} {
		t.Run(prohibida.nombre, func(t *testing.T) {
			if _, err := NuevaSolicitudConsultaInternaGobernadaFuenteAutoridad(
				selector, autorizacion, auditoria, prohibida.ref, correlacion, solicitadaEn,
			); !errors.Is(err, ErrConsultaInternaFuenteAutoridadInvalida) {
				t.Fatalf("referencia de motivo invalida aceptada: %v", err)
			}
		})
	}
}

func TestReciboConsultaComprometeReferenciaMotivoOpaca(t *testing.T) {
	escenario := nuevoEscenarioPuertoAutoridad(t)
	selector := SelectorVersionFuenteAutoridad{FuenteID: escenario.Fuente.ID, Version: 1}
	solicitudBase, decisionBase, solicitadaEn := solicitudConsultaGobernadaAutoridadPrueba(t, selector)
	estado, err := EstadoExactoFuenteAutoridad(escenario.Fuente)
	if err != nil {
		t.Fatal(err)
	}
	reciboBase, err := NuevoReciboConsultaInternaFuenteAutoridad(
		solicitudBase,
		datosReciboConsultaAutoridadPrueba(t, solicitudBase, ResultadoConsultaFuenteEncontrada, estado),
	)
	if err != nil {
		t.Fatal(err)
	}

	motivoAlternativo := motivoCatalogoConsultaAutoridadPrueba(
		claveMotivoConsultaAutoridadAjenaPrueba,
	)
	recursoAlternativo, err := RecursoAutorizableConsultaInternaFuenteAutoridad(
		selector, motivoAlternativo,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaContexto, err := recursoAlternativo.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	decisionAlternativa := clonarDecisionAutorizacionCanonica(decisionBase)
	decisionAlternativa.ContextoRecursoHuellaSHA256 = huellaContexto
	ligarDecisionSolicitudConsultaAutoridadPrueba(
		t, &decisionAlternativa, recursoAlternativo, motivoAlternativo,
	)
	evidenciaAlternativa, err := NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
		decisionAlternativa, solicitadaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	auditoriaAlternativa, err := solicitudBase.Auditoria()
	if err != nil {
		t.Fatal(err)
	}
	auditoriaAlternativa.Reason = motivoAlternativo.EntradaClave
	auditoriaAlternativa.Metadata[AtributoMotivoEntradaClaveConsultaAutoridad] = motivoAlternativo.EntradaClave
	solicitudAlternativa, err := NuevaSolicitudConsultaInternaGobernadaFuenteAutoridad(
		selector, evidenciaAlternativa, auditoriaAlternativa, motivoAlternativo,
		referenciaCorrelacionPuertoPrueba(t, correlacionConsultaAutoridadPrueba), solicitadaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	reciboAlternativo, err := NuevoReciboConsultaInternaFuenteAutoridad(
		solicitudAlternativa,
		datosReciboConsultaAutoridadPrueba(
			t, solicitudAlternativa, ResultadoConsultaFuenteEncontrada, estado,
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	datosBase, _ := reciboBase.Datos()
	datosAlternativos, _ := reciboAlternativo.Datos()
	if datosBase.AuditoriaHuellaEntradaSHA256 == datosAlternativos.AuditoriaHuellaEntradaSHA256 ||
		datosBase.HuellaCompromisoReciboSHA256 == datosAlternativos.HuellaCompromisoReciboSHA256 ||
		reciboBase.ValidarPara(solicitudAlternativa) == nil {
		t.Fatal("el recibo no separo referencias opacas de motivo distintas")
	}
}

func TestReciboConsultaExigeEntradaAuditoriaConfirmadaYCompromisos(t *testing.T) {
	escenario := nuevoEscenarioPuertoAutoridad(t)
	selector := SelectorVersionFuenteAutoridad{FuenteID: escenario.Fuente.ID, Version: 1}
	solicitud, _, _ := solicitudConsultaGobernadaAutoridadPrueba(t, selector)
	solicitadaEn, _ := solicitud.SolicitadaEn()
	estado, err := EstadoExactoFuenteAutoridad(escenario.Fuente)
	if err != nil {
		t.Fatal(err)
	}
	base := datosReciboConsultaAutoridadPrueba(
		t, solicitud, ResultadoConsultaFuenteEncontrada, estado,
	)
	recibo, err := NuevoReciboConsultaInternaFuenteAutoridad(solicitud, base)
	if err != nil {
		t.Fatalf("recibo valido rechazado: %v", err)
	}
	if _, err := recibo.MarshalCBOR(); !errors.Is(err, ErrSerializacionGobiernoFuenteAutoridad) {
		t.Fatalf("recibo CBOR serializable: %v", err)
	}
	if _, err := base.MarshalYAML(); !errors.Is(err, ErrSerializacionGobiernoFuenteAutoridad) {
		t.Fatalf("datos de recibo YAML serializables: %v", err)
	}
	var reconstruido ReciboConsultaInternaFuenteAutoridad
	if err := reconstruido.UnmarshalCBOR([]byte{0xa0}); !errors.Is(err, ErrSerializacionGobiernoFuenteAutoridad) {
		t.Fatalf("recibo reconstruible desde CBOR: %v", err)
	}
	if err := reconstruido.UnmarshalYAML(func(any) error { return nil }); !errors.Is(err, ErrSerializacionGobiernoFuenteAutoridad) {
		t.Fatalf("recibo reconstruible desde YAML: %v", err)
	}
	datos, err := recibo.Datos()
	if err != nil || datos.AuditoriaSecuencia <= 0 ||
		!huellaSHA256PuertoAutoridadValida(datos.AuditoriaHuellaEntradaSHA256) ||
		!huellaSHA256PuertoAutoridadValida(datos.HuellaCompromisoReciboSHA256) ||
		datos.AuditoriaConfirmada.ID != datos.AuditoriaRef ||
		datos.AuditoriaConfirmada.Result != string(ResultadoConsultaFuenteEncontrada) ||
		datos.AuditoriaConfirmada.AfterHash == "" {
		t.Fatalf("evidencia post-COMMIT incompleta: %+v, %v", datos, err)
	}

	casos := []struct {
		nombre string
		mutar  func(*DatosReciboConsultaInternaFuenteAutoridad)
	}{
		{"entrada nula", func(d *DatosReciboConsultaInternaFuenteAutoridad) {
			d.AuditoriaConfirmada = domain.AuditEntry{}
		}},
		{"referencia ficticia", func(d *DatosReciboConsultaInternaFuenteAutoridad) {
			d.AuditoriaRef = "auditoria:consulta:ficticia"
		}},
		{"secuencia nula", func(d *DatosReciboConsultaInternaFuenteAutoridad) {
			d.AuditoriaSecuencia = 0
			d.AuditoriaConfirmada.Seq = 0
		}},
		{"firma nula", func(d *DatosReciboConsultaInternaFuenteAutoridad) {
			d.AuditoriaFirmaRef = ""
			d.AuditoriaConfirmada.Signature = ""
		}},
		{"encadenado nulo en secuencia posterior", func(d *DatosReciboConsultaInternaFuenteAutoridad) {
			d.AuditoriaEncadenadoAnteriorRef = ""
			d.AuditoriaConfirmada.PrevSignature = ""
		}},
		{"algoritmo ficticio", func(d *DatosReciboConsultaInternaFuenteAutoridad) {
			d.AuditoriaAlgoritmoIntegridad = "SHA256 CON ESPACIO"
			d.AuditoriaConfirmada.IntegrityAlgorithm = "SHA256 CON ESPACIO"
		}},
		{"outcome divergente", func(d *DatosReciboConsultaInternaFuenteAutoridad) {
			d.AuditoriaConfirmada.Result = string(ResultadoConsultaFuenteNoEncontrada)
		}},
		{"snapshot no ligado", func(d *DatosReciboConsultaInternaFuenteAutoridad) {
			d.AuditoriaConfirmada.AfterHash = strings.Repeat("f", 64)
		}},
		{"campo personal ajeno", func(d *DatosReciboConsultaInternaFuenteAutoridad) {
			d.AuditoriaConfirmada.RepresentedSubjectID = "per_sujeto_ajeno_000000000001"
		}},
		{"metadato no minimizado", func(d *DatosReciboConsultaInternaFuenteAutoridad) {
			d.AuditoriaConfirmada.Metadata["dato_personal"] = "token:no:permitido"
		}},
		{"huella declarada por adaptador", func(d *DatosReciboConsultaInternaFuenteAutoridad) {
			d.AuditoriaHuellaEntradaSHA256 = strings.Repeat("a", 64)
		}},
		{"confirmacion no posterior", func(d *DatosReciboConsultaInternaFuenteAutoridad) {
			d.ConfirmadaEn = solicitadaEn
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterados := clonarDatosReciboConsultaAutoridad(base)
			caso.mutar(&alterados)
			if _, err := NuevoReciboConsultaInternaFuenteAutoridad(
				solicitud, alterados,
			); !errors.Is(err, ErrReciboConsultaFuenteAutoridadInvalido) {
				t.Fatalf("evidencia de auditoria invalida aceptada: %v", err)
			}
		})
	}

	t.Run("corrupcion posterior de huella interna", func(t *testing.T) {
		alterado := recibo
		copia := clonarDatosReciboConsultaAutoridad(*recibo.datos)
		copia.AuditoriaHuellaEntradaSHA256 = strings.Repeat("0", 64)
		alterado.datos = &copia
		if _, err := alterado.Datos(); !errors.Is(err, ErrReciboConsultaFuenteAutoridadInvalido) {
			t.Fatalf("corrupcion de huella aceptada: %v", err)
		}
	})
	t.Run("corrupcion posterior de referencia de firma", func(t *testing.T) {
		alterado := recibo
		copia := clonarDatosReciboConsultaAutoridad(*recibo.datos)
		copia.AuditoriaFirmaRef = "firma:auditoria:autoridad:otra"
		alterado.datos = &copia
		if _, err := alterado.Datos(); !errors.Is(err, ErrReciboConsultaFuenteAutoridadInvalido) {
			t.Fatalf("corrupcion de firma aceptada: %v", err)
		}
	})
}

func TestReciboConsultaClonaEntradaAuditoriaConfirmada(t *testing.T) {
	escenario := nuevoEscenarioPuertoAutoridad(t)
	selector := SelectorVersionFuenteAutoridad{FuenteID: escenario.Fuente.ID, Version: 1}
	solicitud, _, _ := solicitudConsultaGobernadaAutoridadPrueba(t, selector)
	estado, _ := EstadoExactoFuenteAutoridad(escenario.Fuente)
	datosEntrada := datosReciboConsultaAutoridadPrueba(
		t, solicitud, ResultadoConsultaFuenteEncontrada, estado,
	)
	recibo, err := NuevoReciboConsultaInternaFuenteAutoridad(solicitud, datosEntrada)
	if err != nil {
		t.Fatal(err)
	}
	datosEntrada.AuditoriaConfirmada.Metadata["fuente_id"] = "mutada_fuera"
	primera, _ := recibo.Datos()
	primera.AuditoriaConfirmada.Metadata["fuente_id"] = "mutada_en_copia"
	segunda, err := recibo.Datos()
	if err != nil || segunda.AuditoriaConfirmada.Metadata["fuente_id"] != selector.FuenteID {
		t.Fatalf("el recibo compartio memoria mutable: %+v, %v", segunda, err)
	}
}

func datosReciboConsultaAutoridadPrueba(
	t *testing.T,
	solicitud SolicitudConsultaInternaGobernadaFuenteAutoridad,
	resultado ResultadoConsultaFuenteAutoridad,
	estado ReferenciaEstadoFuenteAutoridad,
) DatosReciboConsultaInternaFuenteAutoridad {
	t.Helper()
	selector, errSelector := solicitud.Selector()
	autorizacion, errAutorizacion := solicitud.Autorizacion()
	datosAutorizacion, errDatos := autorizacion.Datos()
	solicitadaEn, errInstante := solicitud.SolicitadaEn()
	auditoria, errAuditoria := PrepararAuditoriaResultadoConsultaInternaFuenteAutoridad(
		solicitud, resultado, estado,
	)
	if errSelector != nil || errAutorizacion != nil || errDatos != nil ||
		errInstante != nil || errAuditoria != nil {
		t.Fatalf("preparar recibo: %v", errors.Join(
			errSelector, errAutorizacion, errDatos, errInstante, errAuditoria,
		))
	}
	auditoria.ID = "auditoria:consulta:autoridad:confirmada"
	auditoria.Seq = 23
	auditoria.IntegrityAlgorithm = "hmac-sha256-chain-v1"
	auditoria.PrevSignature = "firma:auditoria:autoridad:previa"
	auditoria.Signature = "firma:auditoria:autoridad:confirmada"
	return DatosReciboConsultaInternaFuenteAutoridad{
		TransaccionRef: "transaccion:consulta:autoridad:confirmada",
		Selector:       selector, Resultado: resultado, Estado: estado,
		DecisionRef:          datosAutorizacion.Decision.DecisionRef,
		HuellaDecisionSHA256: datosAutorizacion.HuellaDecisionSHA256,
		AuditoriaRef:         auditoria.ID, AuditoriaSecuencia: auditoria.Seq,
		AuditoriaAlgoritmoIntegridad:   auditoria.IntegrityAlgorithm,
		AuditoriaEncadenadoAnteriorRef: auditoria.PrevSignature,
		AuditoriaFirmaRef:              auditoria.Signature, AuditoriaConfirmada: auditoria,
		ConfirmadaEn: solicitadaEn.Add(time.Microsecond),
	}
}

func vinculoConsultaAutoridadSuperficiePrueba(
	t *testing.T,
	instante time.Time,
	superficie domain.SuperficieAutenticacionActorV1,
) domain.VinculoAutenticacionActorV1 {
	t.Helper()
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: domain.AuthMethodCertificate,
		Garantia: domain.AuthAssuranceHigh,
	}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 5,
		CuentaRef: cuenta.CuentaRef, PersonaRef: personaVinculoPuertoPrueba, PersonaVersion: 3,
		PerfilActivoRef: perfilVinculoPuertoPrueba, PerfilVersion: 4,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: instante.Add(-time.Hour), VigenteHasta: instante.Add(30 * time.Minute),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, instante.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef:               "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 7,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie: superficie, MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    instante.Add(-5 * time.Minute),
		SesionEmitidaEn:              instante.Add(-4 * time.Minute),
		SesionRevalidadaEn:           instante.Add(-3 * time.Minute),
		SesionValidaHasta:            instante.Add(10 * time.Minute),
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV1(
		context.Background(), revalidadorVinculoPuertoPrueba{resultado: autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
		},
		actor, instante,
	)
	if err != nil {
		t.Fatalf("crear vinculo de superficie: %v", err)
	}
	return vinculo
}
