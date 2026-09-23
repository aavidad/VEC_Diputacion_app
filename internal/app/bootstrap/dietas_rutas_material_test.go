package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/adapters/seguridad/verificacioncose"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type relojMaterialRutasDietasPrueba struct {
	ahora        time.Time
	invocaciones int
	cancelar     context.CancelFunc
}

func (r *relojMaterialRutasDietasPrueba) Ahora() time.Time {
	r.invocaciones++
	if r.cancelar != nil {
		r.cancelar()
	}
	return r.ahora
}

type revalidadorMaterialRutasDietasPrueba struct {
	resultado domain.AutenticacionRevalidadaV1
}

func (r revalidadorMaterialRutasDietasPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type resolutorMaterialRutasDietasPrueba struct {
	resultado domain.ResultadoContextoActorRegistradoV2
}

func (r resolutorMaterialRutasDietasPrueba) ResolverContextoActorRegistradoV2(
	context.Context,
	domain.SolicitudContextoActor,
) (domain.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type generadorCorrelacionMaterialRutasDietasPrueba struct{ valor string }

func (g generadorCorrelacionMaterialRutasDietasPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

func nuevoEscenarioMaterialRutasDietasPrueba(t *testing.T, accion string, instante ...time.Time) escenarioMaterialRutasDietasPrueba {
	t.Helper()
	ahora := time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC)
	if len(instante) > 0 {
		ahora = instante[0]
	}
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl",
		Metodo:    domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh,
	}
	instantaneaActor := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 3,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: 4,
		PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 5,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	actor, err := domain.NuevoContextoActor(
		cuenta,
		instantaneaActor,
		ahora.Add(-2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	representacion, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := domain.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:          "prc_0123456789abcdefghijkl",
		ProcedenciaVersion:      1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad:    domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := domain.ManifiestoProcedenciaContextoActorV1{
		Esquema:           domain.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: domain.ProcedenciaCuentaContextoActorV1{
			CuentaRef: cuenta.CuentaRef, Version: instantaneaActor.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: domain.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantaneaActor.PersonaRef, Version: instantaneaActor.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: domain.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantaneaActor.PerfilActivoRef, Version: instantaneaActor.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: domain.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantaneaActor.VinculoRef, Version: instantaneaActor.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: make([]domain.ProcedenciaVinculoReferenciaContextoActorV1, 0),
	}
	canonManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
		canonManifiesto,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn",
		Contexto:            actor, RepresentacionCanonica: representacion,
		HuellaSHA256:                      huella,
		ManifiestoProcedenciaCanonico:     canonManifiesto,
		ManifiestoProcedenciaHuellaSHA256: huellaManifiesto,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef:               "ase_0123456789abcdefghijkl",
		SesionRef:                 "ses_0123456789abcdefghijkl",
		ControlSesionRef:          "cse_0123456789abcdefghijkl",
		ControlSesionRevision:     2,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:      domain.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
		SesionValidaHasta:            ahora.Add(20 * time.Minute),
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(
		context.Background(),
		revalidadorMaterialRutasDietasPrueba{autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		resolutorMaterialRutasDietasPrueba{resultado},
		domain.SolicitudContextoActor{
			Cuenta: cuenta, PerfilActivoRef: instantaneaActor.PerfilActivoRef,
		},
		&relojMaterialRutasDietasPrueba{ahora: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo := domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64),
		EntradaClave:         "motivo_11111111111111111111111111111111",
	}
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionMaterialRutasDietasPrueba{
			valor: "correlacion_11111111111111111111111111111111",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	tipo, ruta, metodo := "catalogo_rutas_dietas", "/api/vec/dietas/route-catalog", "GET"
	if accion == "dietas.ruta.calculo.solicitar" {
		tipo, ruta, metodo = "calculo_rutas_dietas", "/api/vec/dietas/road-route", "POST"
	}
	correlacionRef, _ := correlacion.ValorCanonico()
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(
		domain.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo,
			Accion: accion,
			Recurso: domain.RecursoAutorizable{
				Referencia: "dietas:rutas:" + tipo, ModuloID: "dietas", Tipo: tipo,
				Ambitos:   map[string]string{"ambito_ref": "granada"},
				Atributos: map[string]string{"version": "grafo-v1", "huella_sha256": strings.Repeat("e", 64), "canal": strings.Repeat("c", 64), "instancia": strings.Repeat("a", 64), "ruta": ruta, "metodo": metodo, "correlacion_ref": correlacionRef},
			},
			Finalidad:   "consultar_itinerario_dietas",
			Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	version := domain.VersionRol{
		RolID: "consulta_rutas", Version: 1, Nombre: "Consulta rutas",
		Estado: domain.EstadoVersionRolPublicada,
		Concesiones: []domain.ConcesionRol{{
			Accion:         accion,
			ModuloID:       "dietas",
			TipoRecurso:    tipo,
			Finalidades:    []string{"consultar_itinerario_dietas"},
			GarantiaMinima: domain.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable-seguridad",
		PublicadaEn:  ahora.Add(-24 * time.Hour),
	}
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := domain.InstantaneaAutorizacion{
		AsignacionPerfil: domain.AsignacionPerfil{
			AsignacionID: "asig-dietas", Version: 1,
			PerfilActivoRef: instantaneaActor.PerfilActivoRef,
			PrincipalID:     instantaneaActor.PersonaRef,
			VersionRolRef:   version.Referencia(),
			Estado:          domain.EstadoAsignacionPerfilActiva,
			Ambitos:         []domain.AmbitoPerfil{{Clave: "ambito_ref", Valores: []string{"granada"}}},
			VigenteDesde:    ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			EmitidaPor: "administrador-identidades",
			EmitidaEn:  ahora.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn,
		},
		RevisionCatalogoPoliticas:     1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud,
		instantanea,
		"dec_0123456789abcdef0123456789abcdef",
		ahora,
		ahora.Add(90*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	if err := decision.Validar(); err != nil {
		t.Fatalf("decision V3 recien creada invalida: %v", err)
	}
	if concedida, codigo, err := decision.Resultado(); err != nil || !concedida {
		t.Fatalf("decision V3 no concedida (%t, %q): %v", concedida, codigo, err)
	}
	datos := datosPrivadosMaterialRutasDietasPrueba(t, ahora)
	material, err := nuevoMaterialAtestacionRutasDietasDesarrollo(datos)
	if err != nil {
		t.Fatal(err)
	}
	defer material.borrarCopiasEfimeras()
	reloj := &relojMaterialRutasDietasPrueba{ahora: ahora}
	proveedor, err := nuevoProveedorMaterialAccesoRutasDietasDesarrollo(material, reloj)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(proveedor.Cerrar)
	t.Cleanup(datos.borrarCopiasEfimeras)
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, motivo, resultado)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := ports.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Background(), registroMaterialRutasDietasPrueba{ahora}, orden)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioMaterialRutasDietasPrueba{ahora: ahora, reloj: reloj, solicitud: solicitud, decision: decision, motivo: motivo, resultado: resultado, confirmacion: confirmacion, datos: datos, proveedor: proveedor}
}

type escenarioMaterialRutasDietasPrueba struct {
	ahora        time.Time
	reloj        *relojMaterialRutasDietasPrueba
	solicitud    domain.SolicitudAutorizacionLigadaV3
	decision     domain.DecisionAutorizacionLigadaV3
	motivo       domain.ReferenciaEntradaCatalogo
	resultado    domain.ResultadoContextoActorRegistradoV2
	confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	datos        datosMaterialRutasDietasDesarrollo
	proveedor    *proveedorMaterialAccesoRutasDietasDesarrollo
}

type registroMaterialRutasDietasPrueba struct{ ahora time.Time }

func (r registroMaterialRutasDietasPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	return r.ahora, nil
}

func datosPrivadosMaterialRutasDietasPrueba(t *testing.T, ahora time.Time) datosMaterialRutasDietasDesarrollo {
	t.Helper()
	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hmac := make([]byte, 32)
	if _, err := rand.Read(hmac); err != nil {
		t.Fatal(err)
	}
	return datosMaterialRutasDietasDesarrollo{
		EstadoRaiz: confianza.EstadoClaveAtestacionAutorizacionV3Activa, EstadoHMAC: confianza.EstadoClaveHMACCapacidadAtestacionV3Emision,
		ClaveID: "clave:dietas:rutas:prueba", ClaveVersion: 1, PrivadaEd25519: privada, PublicaEd25519: publica,
		ValidaDesde: ahora.Add(-time.Hour), ValidaHasta: ahora.Add(time.Hour),
		ConfiguracionReferencia: "confianza:dietas:rutas:prueba", ConfiguracionOrden: 1, PublicadaEn: ahora.Add(-time.Minute), ExpiraEn: ahora.Add(30 * time.Minute),
		ClaveHMACID: "clave:hmac:dietas:rutas:prueba", ClaveHMACVersion: 1, MaterialHMAC: hmac, EmisorID: "emisor:dietas:rutas:prueba",
		HMACValidaDesde: ahora.Add(-time.Hour), HMACValidaHasta: ahora.Add(time.Hour), RevisionGobierno: 1, HuellaGobierno: strings.Repeat("a", 64),
	}
}

func TestMaterialRutasDietasEmiteConFirmaNominalYAislaAudiencias(t *testing.T) {
	for _, accion := range []string{"dietas.ruta.catalogo.consultar", "dietas.ruta.calculo.solicitar"} {
		t.Run(accion, func(t *testing.T) {
			e := nuevoEscenarioMaterialRutasDietasPrueba(t, accion)
			material, err := e.proveedor.ProveerMaterialAccesoRutas(context.Background(), e.solicitud, e.decision, e.confirmacion, e.resultado)
			if err != nil || material.ValidarEstructura() != nil {
				t.Fatalf("emision nominal rechazada: %v", err)
			}
			if material.ResumenCapacidad().AudienciaConsumo() != audienciaConsumoRutasDietasDesarrollo || material.ResumenCapacidad().Operacion() != accion {
				t.Fatal("audiencia u operacion cruzada")
			}
			// Verificador criptográfico independiente del proveedor: COSE separado,
			// clave pública provisionada y AAD de la audiencia nominal.
			verificador, err := verificacioncose.NuevoVerificadorClave([]byte(e.datos.ClaveID), verificacioncose.AlgoritmoEdDSA, ed25519.PublicKey(e.datos.PublicaEd25519))
			if err != nil {
				t.Fatal(err)
			}
			sobre := material.SobreCOSESign1()
			inspeccion, err := verificacioncose.InspeccionarSobreSign1(sobre, len(sobre))
			if err != nil {
				t.Fatal(err)
			}
			aad, err := confianza.AADExternoAtestacionAutorizacionV3(audienciaAtestacionRutasDietasDesarrollo)
			if err != nil || verificador.VerificarPayloadSeparado(inspeccion, material.PayloadVECAD3(), aad) != nil {
				t.Fatal("firma Dietas no verificable")
			}
			ajena, _ := confianza.AADExternoAtestacionAutorizacionV3("vec:desarrollo:contratacion-temporal:atestacion:v3")
			if verificador.VerificarPayloadSeparado(inspeccion, material.PayloadVECAD3(), ajena) == nil {
				t.Fatal("firma valida en otra audiencia")
			}
			e.proveedor.Cerrar()
			if _, err := e.proveedor.ProveerMaterialAccesoRutas(context.Background(), e.solicitud, e.decision, e.confirmacion, e.resultado); err == nil {
				t.Fatal("emision tras cierre")
			}
		})
	}
}

func TestMaterialRutasDietasRechazaCrucesYMaterialVencido(t *testing.T) {
	for _, caso := range []string{"accion_ajena", "confirmacion_cero", "contexto_cambiado", "decision_cruzada", "raiz_revocada", "configuracion_vencida", "clave_no_corresponde", "audiencia_consumo_ajena", "cancelada"} {
		t.Run(caso, func(t *testing.T) {
			accion := "dietas.ruta.catalogo.consultar"
			if caso == "accion_ajena" {
				accion = "otro.modulo.consultar"
			}
			e := nuevoEscenarioMaterialRutasDietasPrueba(t, accion)
			ctx := context.Background()
			switch caso {
			case "decision_cruzada":
				e.decision = nuevoEscenarioMaterialRutasDietasPrueba(t, "dietas.ruta.calculo.solicitar").decision
			case "raiz_revocada":
				e.datos.EstadoRaiz = confianza.EstadoClaveAtestacionAutorizacionV3Revocada
				e.datos.RevocadaEn = e.ahora.Add(-time.Minute)
				m, err := nuevoMaterialAtestacionRutasDietasDesarrollo(e.datos)
				if err != nil {
					t.Fatal(err)
				}
				defer m.borrarCopiasEfimeras()
				e.proveedor.Cerrar()
				e.proveedor, err = nuevoProveedorMaterialAccesoRutasDietasDesarrollo(m, e.reloj)
				if err != nil {
					t.Fatal(err)
				}
				defer e.proveedor.Cerrar()
			case "confirmacion_cero":
				e.confirmacion = ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
			case "contexto_cambiado":
				e.resultado.RegistroContextoRef = "rca_999999999999999999999999"
			case "configuracion_vencida":
				e.reloj.ahora = e.datos.ExpiraEn
			case "clave_no_corresponde":
				_, privada, err := ed25519.GenerateKey(rand.Reader)
				if err != nil {
					t.Fatal(err)
				}
				e.proveedor.firmante.privada = privada
			case "audiencia_consumo_ajena":
				c, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(e.datos.ClaveHMACID, 1, e.datos.MaterialHMAC, e.datos.EmisorID, "vec_otro.acceso.v1", confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, e.datos.HMACValidaDesde, e.datos.HMACValidaHasta, time.Time{}, 1, e.datos.HuellaGobierno)
				if err != nil {
					t.Fatal(err)
				}
				e.proveedor.emisor, err = confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(c, e.reloj)
				if err != nil {
					t.Fatal(err)
				}
			case "cancelada":
				var cancelar context.CancelFunc
				ctx, cancelar = context.WithCancel(ctx)
				cancelar()
			}
			if _, err := e.proveedor.ProveerMaterialAccesoRutas(ctx, e.solicitud, e.decision, e.confirmacion, e.resultado); err == nil {
				t.Fatal("cruce aceptado")
			}
		})
	}
}

func TestMaterialRutasDietasConfiguracionExplicitaYCopias(t *testing.T) {
	ahora := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	datos := datosPrivadosMaterialRutasDietasPrueba(t, ahora)
	defer datos.borrarCopiasEfimeras()
	material, err := nuevoMaterialAtestacionRutasDietasDesarrollo(datos)
	if err != nil {
		t.Fatal(err)
	}
	defer material.borrarCopiasEfimeras()
	proveedor, err := nuevoProveedorMaterialAccesoRutasDietasDesarrollo(material, &relojMaterialRutasDietasPrueba{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	defer proveedor.Cerrar()
	copia := append([]byte(nil), proveedor.firmante.privada...)
	datos.borrarCopiasEfimeras()
	material.borrarCopiasEfimeras()
	if !bytes.Equal(copia, proveedor.firmante.privada) {
		t.Fatal("el proveedor comparte la copia del cargador")
	}
	if _, err := nuevoMaterialAtestacionRutasDietasDesarrollo(datosMaterialRutasDietasDesarrollo{}); err == nil {
		t.Fatal("material ausente aceptado")
	}
	if _, err := nuevoProveedorMaterialAccesoRutasDietasDesarrollo(material, nil); err == nil {
		t.Fatal("reloj ausente aceptado")
	}
	if _, err := json.Marshal(datos); err == nil {
		t.Fatal("serializacion privada permitida")
	}
	if texto := fmt.Sprintf("%+v %#v", datos, proveedor); strings.Contains(texto, datos.ClaveID) || !strings.Contains(texto, "redactado") {
		t.Fatal("material sin redaccion")
	}
}
