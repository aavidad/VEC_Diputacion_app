package confianzaatestacion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	usuariosdomain "vec-diputacion-granada/internal/modules/usuarios/domain"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorOrdenUsuariosPrueba struct{}

func (proveedorOrdenUsuariosPrueba) ProveerMaterialPreferencias(context.Context, core.VinculoAutenticacionActorV2, usuariosports.MaterialPreferencias) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, usuariosports.ErrNoDisponible
}

func TestVinculoExteriorUsuariosRechazaRutaYCuentaSustituidas(t *testing.T) {
	base := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	resultado, v := vinculoExternoUsuariosPrueba(t, base)
	actor := resultado.Contexto
	if actor.PersonaRef != base.resultado.Contexto.PersonaRef || actor.PerfilActivoRef == base.resultado.Contexto.PerfilActivoRef || actor.Instantanea.CuentaRef == base.resultado.Contexto.Instantanea.CuentaRef {
		t.Fatal("fixture exterior no separa cuenta/perfil de interna para misma persona")
	}
	if _, err := usuariosports.NuevaOrdenPreferencias(actor, v, core.SuperficieAutenticacionExternaPersonalV1, proveedorOrdenUsuariosPrueba{}); err != nil {
		t.Fatalf("vínculo exterior legítimo: %v", err)
	}
	if _, err := usuariosports.NuevaOrdenPreferencias(actor, v, core.SuperficieAutenticacionInternaCorporativaV1, proveedorOrdenUsuariosPrueba{}); !errors.Is(err, usuariosports.ErrProhibido) {
		t.Fatalf("ruta interna con V2 exterior: %v", err)
	}
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: "cta_otra_cuenta_0123456789abcd", Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}
	snap := actor.Instantanea
	snap.CuentaRef = cuenta.CuentaRef
	ajeno, err := core.NuevoContextoActor(cuenta, snap, actor.ResueltoEn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = usuariosports.NuevaOrdenPreferencias(ajeno, v, core.SuperficieAutenticacionExternaPersonalV1, proveedorOrdenUsuariosPrueba{}); !errors.Is(err, usuariosports.ErrNoAutenticado) {
		t.Fatalf("certificado/cuenta sustituida aceptada: %v", err)
	}
}

// Atraviesa el emisor HMAC real con una decisión y una atestación reales de
// prueba, usando exactamente la preimagen de Usuarios que reconstruye SQL.
func vinculoExternoUsuariosPrueba(t *testing.T, base escenarioConfianzaAtestacionV3Prueba) (core.ResultadoContextoActorRegistradoV2, core.VinculoAutenticacionActorV2) {
	t.Helper()
	d, err := base.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	v, err := d.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	z := strings.Repeat("e", 24)
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}
	snap := base.resultado.Contexto.Instantanea
	snap.VinculoRef = "vca_" + z
	snap.VinculoVersion = 1
	snap.CuentaRef = cuenta.CuentaRef
	snap.CuentaVersion = 1
	snap.PerfilActivoRef = "prf_" + z
	snap.PerfilVersion = 1
	actor, err := core.NuevoContextoActor(cuenta, snap, base.resultado.Contexto.ResueltoEn)
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	ac := core.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + z, ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("e", 64), ProcedenciaAutoridad: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	man := core.ManifiestoProcedenciaContextoActorV1{Esquema: core.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta:   core.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac},
		Persona:  core.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: snap.PersonaVersion, AcreditacionProcedenciaComponenteContextoActorV1: ac},
		Perfil:   core.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac},
		Contexto: core.ProcedenciaVinculoContextoActorV1{VinculoRef: snap.VinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac},
		Vinculos: []core.ProcedenciaVinculoReferenciaContextoActorV1{}}
	bm, err := man.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	hm, err := core.HuellaSHA256ManifiestoProcedenciaContextoActorV1(bm)
	if err != nil {
		t.Fatal(err)
	}
	resultado := core.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_" + z, Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico: bm, ManifiestoProcedenciaHuellaSHA256: hm, AutoridadEfectiva: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: actor.ResueltoEn}
	if err = resultado.Validar(); err != nil {
		t.Fatal(err)
	}
	auth := v.Autenticacion()
	auth.Superficie = core.SuperficieAutenticacionExternaPersonalV1
	auth.AutenticacionRef = "aut_" + z
	auth.AsercionRef = "ase_" + z
	auth.SesionRef = "ses_" + z
	auth.ControlSesionRef = "cse_" + z
	auth.AutenticacionHuellaSHA256 = strings.Repeat("9", 64)
	auth.ControlSesionHuellaSHA256 = strings.Repeat("e", 64)
	auth.CuentaRef = cuenta.CuentaRef
	auth.CuentaOrdinariaRef = cuenta.CuentaRef
	externo, err := core.CrearVinculoAutenticacionActorV2(context.Background(), revalidadorConfianzaAtestacionV3Prueba{resultado: auth},
		core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auth.AutenticacionRef, SesionRef: auth.SesionRef},
		resolutorConfianzaAtestacionV3Prueba{resultado: resultado}, core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef},
		&relojConfianzaAtestacionV3Prueba{ahora: base.ahora})
	if err != nil {
		t.Fatal(err)
	}
	return resultado, externo
}

func TestEmisorRealCapacidadV3UsuariosPreferencias(t *testing.T) {
	base := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	resultadoExterno, vinculoExterno := vinculoExternoUsuariosPrueba(t, base)
	if resultadoExterno.Contexto.PersonaRef != base.resultado.Contexto.PersonaRef || resultadoExterno.Contexto.PerfilActivoRef == base.resultado.Contexto.PerfilActivoRef ||
		resultadoExterno.Contexto.Instantanea.CuentaRef == base.resultado.Contexto.Instantanea.CuentaRef {
		t.Fatal("superficies sin cuenta/perfil distintos para la misma persona")
	}
	var huellaInternaActualizar, huellaExternaActualizar string
	for _, caso := range []struct {
		accion, audiencia string
		campos            []string
		superficie        core.SuperficieAutenticacionActorV1
		rol               string
	}{
		{usuariosports.AccionConsultarPreferencias, usuariosports.AudienciaConsultarPreferenciasInterna, []string{"catalogo", "valores", "version"}, core.SuperficieAutenticacionInternaCorporativaV1, "titular_preferencias_interno"},
		{usuariosports.AccionActualizarPreferencias, usuariosports.AudienciaActualizarPreferenciasInterna, []string{"valores", "version"}, core.SuperficieAutenticacionInternaCorporativaV1, "titular_preferencias_interno"},
		{usuariosports.AccionConsultarPreferencias, usuariosports.AudienciaConsultarPreferenciasExterna, []string{"catalogo", "valores", "version"}, core.SuperficieAutenticacionExternaPersonalV1, "titular_preferencias_externo"},
		{usuariosports.AccionActualizarPreferencias, usuariosports.AudienciaActualizarPreferenciasExterna, []string{"valores", "version"}, core.SuperficieAutenticacionExternaPersonalV1, "titular_preferencias_externo"},
	} {
		t.Run(string(caso.superficie)+"/"+caso.accion, func(t *testing.T) {
			resultadoCaso := base.resultado
			if caso.superficie == core.SuperficieAutenticacionExternaPersonalV1 {
				resultadoCaso = resultadoExterno
			}
			m := usuariosports.MaterialPreferencias{Superficie: caso.superficie,
				PersonaRef: resultadoCaso.Contexto.PersonaRef, PerfilRef: resultadoCaso.Contexto.PerfilActivoRef,
				Accion: caso.accion, FinalidadRef: usuariosports.FinalidadPreferenciasPropias,
				CatalogoVersionRef: "usuarios-preferencias-v1", VersionEsperada: 0,
				ClaveOperacion: "operacion-1234567890", HuellaPeticion: strings.Repeat("a", 64),
				Valores: usuariosdomain.CatalogoBasePreferencias().Predeterminados,
			}
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			h := sha256.Sum256(b)
			recurso := core.RecursoAutorizable{Referencia: m.PersonaRef, ModuloID: "usuarios", Tipo: "preferencias_persona", Ambitos: map[string]string{"persona_ref": m.PersonaRef}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
			datosBase, err := base.solicitud.Datos()
			if err != nil {
				t.Fatal(err)
			}
			vinculoCaso := datosBase.VinculoAutenticacionActor
			if caso.superficie == core.SuperficieAutenticacionExternaPersonalV1 {
				vinculoCaso = vinculoExterno
			}
			solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
				VinculoAutenticacionActor: vinculoCaso, ReferenciaMotivo: base.motivo, Accion: caso.accion, Recurso: recurso,
				Finalidad: usuariosports.FinalidadPreferenciasPropias, Correlacion: datosBase.Correlacion,
			})
			if err != nil {
				t.Fatal(err)
			}
			vinculo, err := vinculoCaso.Datos()
			if err != nil {
				t.Fatal(err)
			}
			version := core.VersionRol{RolID: caso.rol, Version: 1, Nombre: "Titular preferencias", Estado: core.EstadoVersionRolPublicada,
				Concesiones:  []core.ConcesionRol{{Accion: caso.accion, ModuloID: "usuarios", TipoRecurso: "preferencias_persona", Finalidades: []string{usuariosports.FinalidadPreferenciasPropias}, GarantiaMinima: core.AuthAssuranceSubstantial, CamposPermitidos: caso.campos}},
				PublicadaPor: "responsable-seguridad", PublicadaEn: base.ahora.Add(-24 * time.Hour)}
			huellaCatalogo, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
			if err != nil {
				t.Fatal(err)
			}
			instantanea := core.InstantaneaAutorizacion{AsignacionPerfil: core.AsignacionPerfil{AsignacionID: "asig-usuarios-preferencias", Version: 1, PerfilActivoRef: vinculo.PerfilActivoRef, PrincipalID: vinculo.PrincipalID,
				Ambitos:       []core.AmbitoPerfil{{Clave: "persona_ref", Valores: []string{m.PersonaRef}}},
				VersionRolRef: version.Referencia(), Estado: core.EstadoAsignacionPerfilActiva, VigenteDesde: base.ahora.Add(-time.Hour), VigenteHasta: base.ahora.Add(time.Hour), EmitidaPor: "administrador-identidades", EmitidaEn: base.ahora.Add(-2 * time.Hour)},
				VersionRol: version, ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{VersionRolRef: version.Referencia(), Revision: 1, Estado: core.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn},
				RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo}
			evidencia, err := core.NuevaEvidenciaEvaluacionAutorizacionV3(solicitud, instantanea, "dec_0123456789abcdef0123456789abcdef", base.ahora, base.ahora.Add(90*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			decision, err := core.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
			if err != nil {
				t.Fatal(err)
			}
			cabecera := core.CabeceraAtestacionAutorizacionV3{FormatoVersion: core.VersionFormatoAtestacionAutorizacionV3, Suite: SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: base.raiz.claveID, Audiencia: audienciaConfianzaAtestacionV3Prueba}
			atestacion := atestacionConfianzaAtestacionV3Prueba(t, cabecera, decision, base.motivo, resultadoCaso, base.privada, base.ahora)
			prueba, err := base.servicio.Verificar(context.Background(), solicitud, decision, base.motivo, resultadoCaso, atestacion)
			if err != nil {
				t.Fatal(err)
			}
			clave, err := NuevaClaveHMACCapacidadAtestacionAutorizacionV3("clave:capacidad:usuarios-preferencias:v1", 1, bytes.Repeat([]byte{0x51}, 32), "emisor:usuarios-preferencias:v1", caso.audiencia, EstadoClaveHMACCapacidadAtestacionV3Emision, base.ahora.Add(-time.Hour), base.ahora.Add(time.Hour), time.Time{}, 7, strings.Repeat("7", 64))
			if err != nil {
				t.Fatal(err)
			}
			emisor, err := nuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, &relojConfianzaAtestacionV3Prueba{ahora: base.ahora.Add(time.Microsecond)}, bytes.NewReader(append(bytes.Repeat([]byte{0x91}, 32), bytes.Repeat([]byte{0x92}, 32)...)))
			if err != nil {
				t.Fatal(err)
			}
			capacidad, err := emisor.Emitir(context.Background(), solicitud, decision, base.motivo, resultadoCaso, atestacion, prueba)
			if err != nil {
				t.Fatal(err)
			}
			segunda, err := emisor.Emitir(context.Background(), solicitud, decision, base.motivo, resultadoCaso, atestacion, prueba)
			if err != nil {
				t.Fatal(err)
			}
			primeraBytes, err := capacidad.ExportacionCanonicaParaConsumidor()
			if err != nil {
				t.Fatal(err)
			}
			segundaBytes, err := segunda.ExportacionCanonicaParaConsumidor()
			if err != nil || bytes.Equal(primeraBytes, segundaBytes) {
				t.Fatal("dos emisiones reutilizan la misma capacidad V3")
			}
			resumen, err := capacidad.ResumenParaConsumidor()
			if err != nil {
				t.Fatal(err)
			}
			huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
			if err != nil {
				t.Fatal(err)
			}
			if resumen.AudienciaConsumo() != caso.audiencia || resumen.Operacion() != caso.accion || resumen.EfectoRef() != m.PersonaRef || resumen.EfectoHuellaSHA256() != huellaRecurso {
				t.Fatal("capacidad real no liga acción, persona, audiencia y preimagen")
			}
			if caso.accion == usuariosports.AccionActualizarPreferencias {
				if caso.superficie == core.SuperficieAutenticacionInternaCorporativaV1 {
					huellaInternaActualizar = resumen.EfectoHuellaSHA256()
				} else {
					huellaExternaActualizar = resumen.EfectoHuellaSHA256()
				}
			}
			for indice, mutar := range []func(*usuariosports.MaterialPreferencias){
				func(x *usuariosports.MaterialPreferencias) { x.PersonaRef = "per_otra_persona_0123456789" },
				func(x *usuariosports.MaterialPreferencias) { x.PerfilRef = "prf_otro_perfil_0123456789" },
				func(x *usuariosports.MaterialPreferencias) { x.CatalogoVersionRef = "usuarios-preferencias-v2" },
				func(x *usuariosports.MaterialPreferencias) { x.Valores.Tema = "oscuro" },
				func(x *usuariosports.MaterialPreferencias) {
					if x.Superficie == core.SuperficieAutenticacionInternaCorporativaV1 {
						x.Superficie = core.SuperficieAutenticacionExternaPersonalV1
					} else {
						x.Superficie = core.SuperficieAutenticacionInternaCorporativaV1
					}
				},
			} {
				copia := m
				mutar(&copia)
				bytesMutados, err := json.Marshal(copia)
				if err != nil {
					t.Fatal(err)
				}
				hm := sha256.Sum256(bytesMutados)
				recursoMutado := core.RecursoAutorizable{Referencia: copia.PersonaRef, ModuloID: "usuarios", Tipo: "preferencias_persona", Ambitos: map[string]string{"persona_ref": copia.PersonaRef}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(hm[:])}}
				huellaMutada, err := recursoMutado.HuellaContextoAutorizacionSHA256()
				if err != nil || huellaMutada == resumen.EfectoHuellaSHA256() {
					t.Fatal("capacidad previa serviría para otro material")
				}
				if indice == 0 && instantanea.AsignacionPerfil.Cubre(recursoMutado) {
					t.Fatal("la asignación propia cubriría otra persona")
				}
				solicitudMutada, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
					VinculoAutenticacionActor: vinculoCaso, ReferenciaMotivo: base.motivo,
					Accion: caso.accion, Recurso: recursoMutado, Finalidad: usuariosports.FinalidadPreferenciasPropias,
					Correlacion: datosBase.Correlacion,
				})
				if err != nil || decision.ValidarPara(solicitudMutada) == nil {
					t.Fatal("la decisión firmada admite un ámbito o material alterado")
				}
			}
		})
	}
	if huellaInternaActualizar == "" || huellaExternaActualizar == "" || huellaInternaActualizar == huellaExternaActualizar {
		t.Fatal("mismo comando semántico reutiliza capacidad V3 entre portales")
	}
}
